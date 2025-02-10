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
func MarkupLineRich(sty *rich.Style, txt []rune, hitags, tags lexer.Line) rich.Text {
	if len(txt) > maxLineLen { // avoid overflow
		return txt[:maxLineLen]
	}
	sz := len(txt)
	if sz == 0 {
		return nil
	}

	ttags := lexer.MergeLines(hitags, tags) // ensures that inner-tags are *after* outer tags
	nt := len(ttags)
	if nt == 0 || nt > maxNumTags {
		return rich.NewText(sty, txt)
	}

	csty := *sty // todo: need to keep updating the current style based on stack.
	stys := []rich.Style{*sty}
	tstack := []int{0} // stack of tags indexes that remain to be completed, sorted soonest at end

	tx := rich.NewText(sty)
	cp := 0
	for i, tr := range ttags {
		if cp >= sz {
			break
		}
		for si := len(tstack) - 1; si >= 0; si-- {
			ts := ttags[tstack[si]]
			if ts.End <= tr.Start {
				ep := min(sz, ts.End)
				if cp < ep {
					tx.AddRunes(txt[cp:ep])
					cp = ep
				}
				tstack = slices.Delete(tstack, si)
				stys = slices.Delete(stys, si)
			}
		}
		if cp >= sz || tr.Start >= sz {
			break
		}
		if tr.Start > cp {
			tx.AddRunes(txt[cp:tr.Start])
		}
		// todo: get style
		nst := *sty
		//	clsnm := tr.Token.Token.StyleName()
		ep := tr.End
		if i < nt-1 {
			if ttags[i+1].Start < tr.End { // next one starts before we end, add to stack
				ep = ttags[i+1].Start
				if len(tstack) == 0 {
					tstack = append(tstack, i)
					stys = append(stys, nst)
				} else {
					for si := len(tstack) - 1; si >= 0; si-- {
						ts := ttags[tstack[si]]
						if tr.End <= ts.End {
							ni := si // + 1 // new index in stack -- right *before* current
							tstack = slices.Insert(tstack, ni, i)
							stys = slices.Insert(stys, ni, nst)
						}
					}
				}
			}
		}
		ep = min(len(txt), ep)
		if tr.Start < ep {
			tx.AddSpan(&nst, txt[tr.Start:ep])
		}
		cp = ep
	}
	if sz > cp {
		tx.AddRunes(sty, txt[cp:sz])
	}
	return tx
}
