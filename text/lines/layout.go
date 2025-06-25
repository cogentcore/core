// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package lines

import (
	"fmt"
	"unicode"

	"cogentcore.org/core/base/slicesx"
	"cogentcore.org/core/text/rich"
	"cogentcore.org/core/text/textpos"
)

// layoutViewLines performs view-specific layout of all lines of current lines markup.
// This manages its own memory allocation, so it can be called on a new view.
// It must be called after any update to the source text or view layout parameters.
func (ls *Lines) layoutViewLines(vw *view) {
	n := len(ls.markup)
	if n == 0 {
		return
	}
	vw.markup = vw.markup[:0]
	vw.vlineStarts = vw.vlineStarts[:0]
	vw.lineToVline = slicesx.SetLength(vw.lineToVline, n)
	nln := 0
	for ln, mu := range ls.markup {
		muls, vst := ls.layoutViewLine(ln, vw.width, ls.lines[ln], mu)
		vw.lineToVline[ln] = len(vw.vlineStarts)
		vw.markup = append(vw.markup, muls...)
		vw.vlineStarts = append(vw.vlineStarts, vst...)
		nln += len(vst)
	}
	vw.viewLines = nln
}

// layoutViewLine performs layout and line wrapping on the given text,
// for given view text, with the given markup rich.Text.
// The layout is implemented in the markup that is returned.
// This clones and then modifies the given markup rich text.
func (ls *Lines) layoutViewLine(ln, width int, txt []rune, mu rich.Text) ([]rich.Text, []textpos.Pos) {
	lt := mu.Clone()
	n := len(txt)
	sp := textpos.Pos{Line: ln, Char: 0} // source startinng position
	vst := []textpos.Pos{sp}             // start with this line
	breaks := []int{}                    // line break indexes into lt spans
	clen := 0                            // current line length so far
	start := true
	prevWasTab := false
	i := 0
	for i < n {
		r := txt[i]
		si, sn, ri := lt.Index(i)
		startOfSpan := sn == ri
		// fmt.Printf("\n####\n%d\tclen:%d\tsi:%dsn:%d\tri:%d\t%v %v, sisrc: %q txt: %q\n", i, clen, si, sn, ri, startOfSpan, prevWasTab, string(lt[si][ri:]), string(txt[i:min(i+5, n)]))
		switch {
		case start && r == '\t':
			clen += ls.Settings.TabSize
			if !startOfSpan {
				lt.SplitSpan(i) // each tab gets its own
			}
			prevWasTab = true
			i++
		case r == '\t':
			tp := (clen / 8) + 1
			tp *= 8
			clen = tp
			if !startOfSpan {
				lt.SplitSpan(i)
			}
			prevWasTab = true
			i++
		case unicode.IsSpace(r):
			start = false
			clen++
			if prevWasTab && !startOfSpan {
				lt.SplitSpan(i)
			}
			prevWasTab = false
			i++
		default:
			start = false
			didSplit := false
			if prevWasTab && !startOfSpan {
				lt.SplitSpan(i)
				didSplit = true
				si++
			}
			prevWasTab = false
			ns := NextSpace(txt, i)
			wlen := ns - i // length of word
			// fmt.Println("word at:", i, "ns:", ns, string(txt[i:ns]))
			if clen+wlen > width { // need to wrap
				clen = 0
				sp.Char = i
				vst = append(vst, sp)
				if !startOfSpan && !didSplit {
					lt.SplitSpan(i)
					si++
				}
				if si < 0 {
					fmt.Println("lines.layout si < 0")
				} else {
					breaks = append(breaks, si)
				}
				if wlen > width {
					nb := wlen / width
					if nb*width == wlen {
						nb--
					}
					bp := i + width
					for range nb {
						si, sn, ri := lt.Index(bp)
						if sn != ri { // not start of span already
							lt.SplitSpan(bp)
							si++
						}
						if si < 0 {
							fmt.Println("lines.layout si < 0")
						} else {
							breaks = append(breaks, si)
						}
						sp.Char = bp
						vst = append(vst, sp)
						bp += width
					}
					clen = wlen - (nb * width)
				}
			}
			clen += wlen
			i = ns
		}
	}
	nb := len(breaks)
	if nb == 0 {
		return []rich.Text{lt}, vst
	}
	muls := make([]rich.Text, 0, nb+1)
	last := 0
	for _, si := range breaks {
		muls = append(muls, lt[last:si])
		last = si
	}
	muls = append(muls, lt[last:])
	return muls, vst
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
