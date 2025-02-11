// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package highlighting

import (
	"slices"

	"cogentcore.org/core/text/parse/lexer"
	"cogentcore.org/core/text/rich"
)

// MarkupLineRich returns the [rich.Text] styled line for each tag.
// Takes both the hi highlighting tags and extra tags.
// The style provides the starting default style properties.
func MarkupLineRich(hs *Style, sty *rich.Style, txt []rune, hitags, tags lexer.Line) rich.Text {
	if len(txt) > maxLineLen { // avoid overflow
		return rich.NewText(sty, txt[:maxLineLen])
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

	tx := rich.NewText(sty, nil)
	cp := 0
	for i, tr := range ttags {
		if cp >= sz {
			break
		}
		// pop anyone off the stack who ends before we start
		for si := len(tstack) - 1; si >= 1; si-- {
			ts := ttags[tstack[si]]
			if ts.End <= tr.Start {
				ep := min(sz, ts.End)
				if cp < ep {
					tx.AddRunes(txt[cp:ep])
					cp = ep
				}
				tstack = slices.Delete(tstack, si, si+1)
				stys = slices.Delete(stys, si, si+1)
			}
		}
		if cp >= sz || tr.Start >= sz {
			break
		}
		if tr.Start > cp { // finish any existing before pushing new
			tx.AddRunes(txt[cp:tr.Start])
		}
		cst := stys[len(stys)-1]
		nst := cst
		entry := hs.Tag(tr.Token.Token)
		if !entry.IsZero() {
			entry.ToRichStyle(&nst)
		}
		tstack = append(tstack, i)
		stys = append(stys, nst)

		ep := tr.End
		if i < nt-1 && ttags[i+1].Start < ep {
			ep = ttags[i+1].Start
		}
		tx.AddSpan(&nst, txt[cp:ep])
		cp = ep
	}
	if cp < sz {
		tx.AddSpan(&stys[len(stys)-1], txt[cp:sz])
	}
	return tx
}
