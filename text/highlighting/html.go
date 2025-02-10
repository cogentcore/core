// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package highlighting

import (
	"html"

	"cogentcore.org/core/text/parse/lexer"
)

// maxLineLen prevents overflow in allocating line length
const (
	maxLineLen   = 64 * 1024
	maxNumTags   = 1024
	EscapeHTML   = true
	NoEscapeHTML = false
)

// MarkupLineHTML returns the line with html class tags added for each tag
// takes both the hi tags and extra tags.  Only fully nested tags are supported,
// with any dangling ends truncated.
func MarkupLineHTML(txt []rune, hitags, tags lexer.Line, escapeHTML bool) []byte {
	if len(txt) > maxLineLen { // avoid overflow
		return nil
	}
	sz := len(txt)
	if sz == 0 {
		return nil
	}
	var escf func([]rune) []byte
	if escapeHTML {
		escf = HTMLEscapeRunes
	} else {
		escf = func(r []rune) []byte {
			return []byte(string(r))
		}
	}

	ttags := lexer.MergeLines(hitags, tags) // ensures that inner-tags are *after* outer tags
	nt := len(ttags)
	if nt == 0 || nt > maxNumTags {
		return escf(txt)
	}
	sps := []byte(`<span class="`)
	sps2 := []byte(`">`)
	spe := []byte(`</span>`)
	taglen := len(sps) + len(sps2) + len(spe) + 2

	musz := sz + nt*taglen
	mu := make([]byte, 0, musz)

	cp := 0
	var tstack []int // stack of tags indexes that remain to be completed, sorted soonest at end
	for i, tr := range ttags {
		if cp >= sz {
			break
		}
		for si := len(tstack) - 1; si >= 0; si-- {
			ts := ttags[tstack[si]]
			if ts.End <= tr.Start {
				ep := min(sz, ts.End)
				if cp < ep {
					mu = append(mu, escf(txt[cp:ep])...)
					cp = ep
				}
				mu = append(mu, spe...)
				tstack = append(tstack[:si], tstack[si+1:]...)
			}
		}
		if cp >= sz || tr.Start >= sz {
			break
		}
		if tr.Start > cp {
			mu = append(mu, escf(txt[cp:tr.Start])...)
		}
		mu = append(mu, sps...)
		clsnm := tr.Token.Token.StyleName()
		mu = append(mu, []byte(clsnm)...)
		mu = append(mu, sps2...)
		ep := tr.End
		addEnd := true
		if i < nt-1 {
			if ttags[i+1].Start < tr.End { // next one starts before we end, add to stack
				addEnd = false
				ep = ttags[i+1].Start
				if len(tstack) == 0 {
					tstack = append(tstack, i)
				} else {
					for si := len(tstack) - 1; si >= 0; si-- {
						ts := ttags[tstack[si]]
						if tr.End <= ts.End {
							ni := si // + 1 // new index in stack -- right *before* current
							tstack = append(tstack, i)
							copy(tstack[ni+1:], tstack[ni:])
							tstack[ni] = i
						}
					}
				}
			}
		}
		ep = min(len(txt), ep)
		if tr.Start < ep {
			mu = append(mu, escf(txt[tr.Start:ep])...)
		}
		if addEnd {
			mu = append(mu, spe...)
		}
		cp = ep
	}
	if sz > cp {
		mu = append(mu, escf(txt[cp:sz])...)
	}
	// pop any left on stack..
	for si := len(tstack) - 1; si >= 0; si-- {
		mu = append(mu, spe...)
	}
	return mu
}

// HTMLEscapeBytes escapes special characters like "<" to become "&lt;". It
// escapes only five such characters: <, >, &, ' and ".
// It operates on a *copy* of the byte string and does not modify the input!
// otherwise it causes major problems..
func HTMLEscapeBytes(b []byte) []byte {
	return []byte(html.EscapeString(string(b)))
}

// HTMLEscapeRunes escapes special characters like "<" to become "&lt;". It
// escapes only five such characters: <, >, &, ' and ".
// It operates on a *copy* of the byte string and does not modify the input!
// otherwise it causes major problems..
func HTMLEscapeRunes(r []rune) []byte {
	return []byte(html.EscapeString(string(r)))
}
