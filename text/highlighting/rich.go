// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package highlighting

import (
	"fmt"
	"slices"

	"cogentcore.org/core/text/parse/lexer"
	"cogentcore.org/core/text/rich"
	"cogentcore.org/core/text/runes"
	"cogentcore.org/core/text/token"
)

// MarkupLineRich returns the [rich.Text] styled line for each tag.
// Takes both the hi highlighting tags and extra tags.
// The style provides the starting default style properties.
func MarkupLineRich(hs *Style, sty *rich.Style, txt []rune, hitags, tags lexer.Line) rich.Text {
	if len(txt) > maxLineLen { // avoid overflow
		return rich.NewText(sty, txt[:maxLineLen])
	}
	if hs == nil {
		return rich.NewText(sty, txt)
	}
	sz := len(txt)
	if sz == 0 {
		return nil
	}

	ttags := lexer.MergeLines(hitags, tags) // ensures that inner-tags are *after* outer tags
	// fmt.Println(ttags)
	nt := len(ttags)
	if nt == 0 || nt > maxNumTags {
		return rich.NewText(sty, txt)
	}

	stys := []rich.Style{*sty}
	tstack := []int{0} // stack of tags indexes that remain to be completed, sorted soonest at end
	var tx rich.Text
	cp := 0
	if ttags[0].Start > 0 {
		tx = rich.NewText(sty, txt[:ttags[0].Start])
		cp = ttags[0].Start
	}
	// fmt.Println("start")
	for i, tr := range ttags {
		if cp >= sz {
			break
		}
		// pop anyone off the stack who ends before we start
		for si := len(tstack) - 1; si >= 1; si-- {
			ts := ttags[tstack[si]]
			// fmt.Println("pop potential:", si, ts)
			if ts.End <= tr.Start {
				ep := min(sz, ts.End)
				if cp < ep {
					// fmt.Println("adding runes to prior:", cp, ep)
					tx.AddRunes(txt[cp:ep])
					cp = ep
				}
				// fmt.Println("delete style")
				tstack = slices.Delete(tstack, si, si+1)
				stys = slices.Delete(stys, si, si+1)
			}
		}
		if cp >= sz || tr.Start >= sz {
			break
		}
		cst := stys[len(stys)-1]
		if tr.Start > cp+1 { // finish any existing before pushing new
			// fmt.Printf("add: %d - %d: %q\n", cp, tr.Start, string(txt[cp:tr.Start]))
			tx.AddSpan(&cst, txt[cp:tr.Start])
			cp = tr.Start
		}
		nst := cst
		entry := hs.Tag(tr.Token.Token)
		if !entry.IsZero() {
			entry.ToRichStyle(&nst)
		} else {
			if tr.Token.Token == token.TextSpellErr {
				nst.SetDecoration(rich.DottedUnderline)
				// fmt.Println(i, tr)
			}
		}
		tstack = append(tstack, i)
		stys = append(stys, nst)

		ep := tr.End
		if i < nt-1 && ttags[i+1].Start < ep {
			ep = ttags[i+1].Start
		}
		tx.AddSpan(&nst, txt[cp:ep])
		// fmt.Println("added:", cp, ep, string(txt[cp:ep]))
		cp = ep
	}
	if cp < sz {
		esp := len(stys) - 1
		if esp > 0 {
			esp = esp - 1
		}
		tx.AddSpan(&stys[esp], txt[cp:sz])
	}
	return tx
}

// MarkupPathsAsLinks adds hyperlink span styles to given markup of given text,
// for any strings that look like file paths / urls.
// maxFields is the maximum number of fieldsto look for file paths in:
// 2 is a reasonable default, to avoid getting other false-alarm info later.
func MarkupPathsAsLinks(txt []rune, mu rich.Text, maxFields int) rich.Text {
	fl := runes.Fields(txt)
	mx := min(len(fl), maxFields)
	for i := range mx {
		ff := fl[i]
		if !runes.HasPrefix(ff, []rune("./")) && !runes.HasPrefix(ff, []rune("/")) && !runes.HasPrefix(ff, []rune("../")) {
			// todo: use regex instead of this.
			if !runes.Contains(ff, []rune("/")) && !runes.Contains(ff, []rune(":")) {
				continue
			}
		}
		fi := runes.Index(txt, ff)
		fnflds := runes.Split(ff, []rune(":"))
		fn := string(fnflds[0])
		pos := ""
		col := ""
		if len(fnflds) > 1 {
			pos = string(fnflds[1])
			col = ""
			if len(fnflds) > 2 {
				col = string(fnflds[2])
			}
		}
		url := ""
		if col != "" {
			url = fmt.Sprintf(`file:///%v#L%vC%v`, fn, pos, col)
		} else if pos != "" {
			url = fmt.Sprintf(`file:///%v#L%v`, fn, pos)
		} else {
			url = fmt.Sprintf(`file:///%v`, fn)
		}
		si := mu.SplitSpan(fi)
		efi := fi + len(ff)
		esi := mu.SplitSpan(efi)
		sty, _ := mu.Span(si)
		sty.SetLink(url)
		mu.SetSpanStyle(si, sty)
		if esi > 0 {
			mu.InsertEndSpecial(esi)
		} else {
			mu.EndSpecial()
		}
	}
	if string(mu.Join()) != string(txt) {
		panic("markup is not the same: " + string(txt) + " mu: " + string(mu.Join()))
	}
	return mu
}
