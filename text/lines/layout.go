// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package lines

import (
	"slices"
	"unicode"

	"cogentcore.org/core/base/runes"
	"cogentcore.org/core/base/slicesx"
	"cogentcore.org/core/text/rich"
	"cogentcore.org/core/text/textpos"
)

// layoutLines performs view-specific layout of given lines of current markup.
// the view must already have allocated space for these lines.
// it updates the current number of total lines based on any changes from
// the current number of lines withing given range.
func (ls *Lines) layoutLines(vw *view, st, ed int) {
	inln := 0
	for ln := st; ln <= ed; ln++ {
		inln += 1 + vw.nbreaks[ln]
	}
	nln := 0
	for ln := st; ln <= ed; ln++ {
		ltxt := ls.lines[ln]
		lmu, lay, nbreaks := ls.layoutLine(vw.width, ltxt, ls.markup[ln])
		vw.markup[ln] = lmu
		vw.layout[ln] = lay
		vw.nbreaks[ln] = nbreaks
		nln += 1 + nbreaks
	}
	vw.totalLines += nln - inln
}

// layoutAll performs view-specific layout of all lines of current markup.
// ensures that view has capacity to hold all lines, so it can be called on a
// new view.
func (ls *Lines) layoutAll(vw *view) {
	n := len(vw.markup)
	if n == 0 {
		return
	}
	vw.markup = slicesx.SetLength(vw.markup, n)
	vw.layout = slicesx.SetLength(vw.layout, n)
	vw.nbreaks = slicesx.SetLength(vw.nbreaks, n)
	nln := 0
	for ln, mu := range ls.markup {
		lmu, lay, nbreaks := ls.layoutLine(vw.width, ls.lines[ln], mu)
		// fmt.Println("\nlayout:\n", lmu)
		vw.markup[ln] = lmu
		vw.layout[ln] = lay
		vw.nbreaks[ln] = nbreaks
		nln += 1 + nbreaks
	}
	vw.totalLines = nln
}

// layoutLine performs layout and line wrapping on the given text, with the
// given markup rich.Text, with the layout implemented in the markup that is returned.
// This clones and then modifies the given markup rich text.
func (ls *Lines) layoutLine(width int, txt []rune, mu rich.Text) (rich.Text, []textpos.Pos16, int) {
	mu = mu.Clone()
	spc := []rune{' '}
	n := len(txt)
	nbreak := 0
	lay := make([]textpos.Pos16, n)
	var cp textpos.Pos16
	start := true
	i := 0
	for i < n {
		r := txt[i]
		lay[i] = cp
		si, sn, rn := mu.Index(i + nbreak) // extra char for each break
		// fmt.Println("\n####\n", i, cp, si, sn, rn, string(mu[si][rn:]))
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
			wlen := ns - i // length of word
			// fmt.Println("word at:", i, "ns:", ns, string(txt[i:ns]))
			if cp.Char+int16(wlen) > int16(width) { // need to wrap
				// fmt.Println("\n****\nline wrap width:", cp.Char+int16(wlen))
				cp.Char = 0
				cp.Line++
				nbreak++
				if rn == sn+1 { // word is at start of span, insert \n in prior
					if si > 0 {
						mu[si-1] = append(mu[si-1], '\n')
						// _, ps := mu.Span(si - 1)
						// fmt.Printf("break prior span: %q", string(ps))
					}
				} else { // split current span at word, rn is start of word at idx i in span si
					sty, _ := mu.Span(si)
					rtx := mu[si][rn:] // skip past the one space we replace with \n
					mu.InsertSpan(si+1, sty, rtx)
					mu[si] = append(mu[si][:rn], '\n')
					// _, cs := mu.Span(si)
					// _, ns := mu.Span(si + 1)
					// fmt.Printf("insert span break:\n%q\n%q", string(cs), string(ns))
				}
			}
			for j := i; j < ns; j++ {
				// if cp.Char >= int16(width) {
				// 	cp.Char = 0
				// 	cp.Line++
				// 	nbreak++
				// 	// todo: split long
				// }
				lay[j] = cp
				cp.Char++
			}
			i = ns
		}
	}
	return mu, lay, nbreak
}

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
