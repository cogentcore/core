// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package highlighting

import (
	"fmt"

	"cogentcore.org/core/text/parse/lexer"
	"cogentcore.org/core/text/rich"
	"cogentcore.org/core/text/runes"
	"cogentcore.org/core/text/textpos"
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

	// first ensure text has spans for each tag region.
	ln := len(txt)
	var tx rich.Text
	cp := 0
	for _, tr := range ttags {
		st := min(tr.Start, ln)
		if st > cp {
			tx.AddSpan(sty, txt[cp:st])
			cp = st
		} else if st < cp {
			tx.SplitSpan(st)
		}
		ed := min(tr.End, ln)
		if ed > cp {
			tx.AddSpan(sty, txt[cp:ed])
			cp = ed
		} else {
			tx.SplitSpan(ed)
		}
	}
	if cp < ln {
		tx.AddSpan(sty, txt[cp:])
	}

	// next, accumulate styles for each span
	for si := range tx {
		s, e := tx.Range(si)
		srng := textpos.Range{Start: s, End: e}
		cst := *sty
		for _, tr := range ttags {
			trng := textpos.Range{Start: tr.Start, End: tr.End}
			if srng.Intersect(trng).Len() <= 0 {
				continue
			}
			entry := hs.Tag(tr.Token.Token)
			if !entry.IsZero() {
				entry.ToRichStyle(&cst)
			}
			if tp, ok := Properties[tr.Token.Token]; ok {
				applyPropertyDecorations(&cst, tp)
			}
		}
		tx.SetSpanStyle(si, &cst)
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
	// if string(mu.Join()) != string(txt) {
	// 	panic("markup is not the same: " + string(txt) + " mu: " + string(mu.Join()))
	// }
	return mu
}

// applyPropertyDecorations applies the text-decoration flags from the
// custom tag fallback [Properties] (for tags without explicit style
// entries, such as TextSpellErr and Error) to the given rich style.
func applyPropertyDecorations(cst *rich.Style, pr map[string]any) {
	var bits uint32
	switch dec := pr["text-decoration"].(type) {
	case uint32:
		bits = dec
	case int:
		bits = uint32(dec)
	default:
		return
	}
	for d := rich.Underline; d <= rich.ParagraphStart; d++ {
		if bits&(1<<uint32(d)) != 0 {
			cst.SetDecoration(d)
		}
	}
}
