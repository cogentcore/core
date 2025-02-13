// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package lines

import (
	"slices"
	"unicode"

	"cogentcore.org/core/base/runes"
	"cogentcore.org/core/text/rich"
	"cogentcore.org/core/text/textpos"
)

func NextSpace(txt []rune, pos int) int {
	n := len(txt)
	for i := pos; i < n; i++ {
		r := txt[i]
		if unicode.IsSpace(r) {
			return i
		}
	}
	return n
}

// layoutLine performs layout and line wrapping on the given text, with the
// given markup rich.Text, with the layout implemented in the markup that is returned.
func (ls *Lines) layoutLine(txt []rune, mu rich.Text) (rich.Text, []textpos.Pos16, int) {
	spc := []rune{' '}
	n := len(txt)
	lt := mu
	nbreak := 0
	lay := make([]textpos.Pos16, n)
	var cp textpos.Pos16
	start := true
	i := 0
	for i < n {
		r := txt[i]
		lay[i] = cp
		si, sn, rn := mu.Index(i)
		switch {
		case start && r == '\t':
			cp.Char += int16(ls.Settings.TabSize) - 1
			mu[si] = slices.Delete(mu[si], rn, rn+1) // remove tab
			mu[si] = slices.Insert(mu[si], rn, runes.Repeat(spc, ls.Settings.TabSize)...)
			i++
		case r == '\t':
			tp := (cp.Char + 1) / 8
			tp *= 8
			cp.Char = tp
			mu[si] = slices.Delete(mu[si], rn, rn+1) // remove tab
			mu[si] = slices.Insert(mu[si], rn, runes.Repeat(spc, int(tp-cp.Char))...)
			i++
		case unicode.IsSpace(r):
			start = false
			cp.Char++
			i++
		default:
			start = false
			ns := NextSpace(txt, i)
			// fmt.Println("word at:", i, "ns:", ns, string(txt[i:ns]))
			if cp.Char+int16(ns-i) > int16(ls.width) { // need to wrap
				cp.Char = 0
				cp.Line++
				nbreak++
				if rn == sn+1 { // word is at start of span, insert \n in prior
					if si > 0 {
						mu[si-1] = append(mu[si-1], '\n')
					}
				} else { // split current span at word
					sty, _ := mu.Span(si)
					rtx := mu[si][rn:]
					mu[si] = append(mu[si][:rn], '\n')
					mu.InsertSpan(si+1, sty, rtx)
				}
			}
			for j := i; j < ns; j++ {
				if cp.Char >= int16(ls.width) {
					cp.Char = 0
					cp.Line++
					nbreak++
					// todo: split long
				}
				lay[i] = cp
				cp.Char++
			}
			i = ns
		}
	}
	return lt, lay, nbreak
}
