// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package lines

import (
	"unicode"

	"cogentcore.org/core/base/runes"
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

func (ls *Lines) layoutLine(txt []rune) ([]rune, []textpos.Pos16) {
	spc := []rune(" ")
	n := len(txt)
	lt := make([]rune, 0, n)
	lay := make([]textpos.Pos16, n)
	var cp textpos.Pos16
	start := true
	for i, r := range txt {
		lay[i] = cp
		switch {
		case start && r == '\t':
			cp.Char += int16(ls.Settings.TabSize)
			lt = append(lt, runes.Repeat(spc, ls.Settings.TabSize)...)
		case r == '\t':
			tp := (cp.Char + 1) / 8
			tp *= 8
			cp.Char = tp
			lt = append(lt, runes.Repeat(spc, int(tp-cp.Char))...)
		case unicode.IsSpace(r):
			start = false
			lt = append(lt, r)
			cp.Char++
		default:
			start = false
			ns := NextSpace(txt, i)
			if cp.Char+int16(ns) > int16(ls.width) {
				cp.Char = 0
				cp.Line++
			}
			for j := i; j < ns; j++ {
				if cp.Char >= int16(ls.width) {
					cp.Char = 0
					cp.Line++
				}
				lay[i] = cp
				lt = append(lt, txt[j])
				cp.Char++
			}
		}
	}
	return lt, lay
}
