// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package lines

import (
	"cogentcore.org/core/text/rich"
	"cogentcore.org/core/text/textpos"
	"github.com/clipperhouse/displaywidth"
)

// Lines are displayed in a grid of monospaced cells, where most
// characters take one cell, and wide characters, such as CJK
// characters, take two cells, as in a terminal. A view line is
// wrapped at its width in cells, and the cursor column used in
// moving up and down is in cells, so that it stays in the same
// visual column across lines with wide characters. Positions in the
// text are still in characters (runes): cells only affect the display.

// RuneWidth returns the number of cells used to display the given rune:
// 2 for wide characters (East Asian Wide and Fullwidth characters, and
// emoji), and 1 otherwise, including zero width characters. Tabs are
// handled separately, by [LineCells].
func RuneWidth(r rune) int {
	if r < 0x1100 { // no wide characters before the Hangul Jamo
		return 1
	}
	if displaywidth.Rune(r) == 2 {
		return 2
	}
	return 1
}

// textCells returns the total width in cells of the given text,
// which has no tabs.
func textCells(txt []rune) int {
	n := 0
	for _, r := range txt {
		n += RuneWidth(r)
	}
	return n
}

// LineCells returns the cell column at the start of each character
// in the given view line markup, as returned by [Lines.ViewMarkupLine],
// followed by the column at its end. The tabs at the start of the line,
// each of which is in its own span, take tabSize cells, other tabs
// go to the next multiple of 8 cells, and wide characters take 2 cells
// (see [RuneWidth]).
func LineCells(tx rich.Text, tabSize int) []int {
	cells := make([]int, 0, tx.Len()+1)
	cc := 0
	start := true
	for si := range tx {
		sn, rn := rich.SpanLen(tx[si])
		if rn == 0 {
			continue
		}
		rs := tx[si][sn:]
		if start && rn == 1 && rs[0] == '\t' {
			cells = append(cells, cc)
			cc += tabSize
			continue
		}
		start = false
		for _, r := range rs {
			cells = append(cells, cc)
			if r == '\t' {
				cc = ((cc / 8) + 1) * 8
			} else {
				cc += RuneWidth(r)
			}
		}
	}
	return append(cells, cc)
}

// cellChar returns the character index in the line with the given
// cells (see [LineCells]) that is closest to the given cell column:
// a column in the second half of a wide character is after it.
func cellChar(cells []int, col int) int {
	n := len(cells) - 1
	for i := range n {
		if 2*col < cells[i]+cells[i+1] {
			return i
		}
	}
	return n
}

// viewLineCells returns the [LineCells] of the given view line.
func (ls *Lines) viewLineCells(vw *view, vl int) []int {
	if vl < 0 || vl >= len(vw.markup) {
		return []int{0}
	}
	return LineCells(vw.markup[vl], ls.Settings.TabSize)
}

// viewCell returns the cell column of the given view position.
// Positions beyond the end of the line are one cell per character.
func (ls *Lines) viewCell(vw *view, vpos textpos.Pos) int {
	cells := ls.viewLineCells(vw, vpos.Line)
	n := len(cells) - 1
	if vpos.Char <= n {
		return cells[max(vpos.Char, 0)]
	}
	return cells[n] + vpos.Char - n
}
