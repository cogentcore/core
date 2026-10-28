// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package vt

import (
	"slices"

	"github.com/charmbracelet/x/ansi"
	"golang.org/x/text/unicode/norm"
)

// screen is a grid of cells: the primary or alternate screen.
type screen struct {
	// lines are the rows of the screen.
	lines []*Line

	// saved is the saved cursor (DECSC).
	saved Cursor
}

// newScreen returns a new blank screen of the given size.
func newScreen(cols, rows int) *screen {
	sc := &screen{lines: make([]*Line, rows)}
	for i := range rows {
		sc.lines[i] = newLine(cols, ColorDefault)
	}
	return sc
}

// clear clears all of the lines.
func (sc *screen) clear() {
	for _, ln := range sc.lines {
		ln.clear(0, len(ln.Cells), ColorDefault)
		ln.Wrapped = false
	}
}

// line returns the line at the cursor.
func (em *Emulator) line() *Line {
	return em.screen.lines[em.Cursor.Row]
}

// bg returns the current background color, used for erasing (BCE).
func (em *Emulator) bg() Color {
	return em.Cursor.Style.Bg
}

// print writes the given character at the cursor.
func (em *Emulator) print(r rune) {
	cs := em.Cursor.charsets[em.Cursor.shift]
	if cs == charsetDECGraphics && r >= 0x5f && r <= 0x7e {
		r = decGraphics[r-0x5f]
	}
	w := ansi.StringWidth(string(r))
	if w == 0 {
		em.combine(r)
		return
	}
	w = min(w, 2)
	cur := &em.Cursor
	if cur.pendingWrap && em.Modes.AutoWrap {
		em.line().Wrapped = true
		cur.Col = 0
		em.index()
	}
	cur.pendingWrap = false
	if w == 2 && cur.Col == em.Cols-1 { // no room for a wide character
		if !em.Modes.AutoWrap || em.Cols < 2 {
			return
		}
		em.setCell(cur.Col, blankCell(em.bg()))
		em.line().Wrapped = true
		cur.Col = 0
		em.index()
	}
	if em.Modes.Insert {
		em.insertCells(w)
	}
	em.setCell(cur.Col, Cell{Rune: r, Width: uint8(w), Style: em.cellStyle()})
	if w == 2 {
		em.setCell(cur.Col+1, Cell{Width: 0, Style: em.cellStyle()})
	}
	em.lastRune = r
	cur.Col += w
	if cur.Col >= em.Cols {
		cur.Col = em.Cols - 1
		if em.Modes.AutoWrap {
			cur.pendingWrap = true
		}
	}
}

// cellStyle returns the style for a new character.
func (em *Emulator) cellStyle() Style {
	st := em.Cursor.Style
	st.Link = em.link
	return st
}

// setCell sets the cell at the given column of the cursor line,
// clearing any wide character that it overwrites part of.
func (em *Emulator) setCell(col int, c Cell) {
	ln := em.line()
	if col < 0 || col >= len(ln.Cells) {
		return
	}
	old := ln.Cells[col]
	if old.Width == 2 && c.Width != 0 && col+1 < len(ln.Cells) {
		ln.Cells[col+1] = blankCell(old.Style.Bg)
	}
	if old.Width == 0 && c.Width != 0 && col > 0 && ln.Cells[col-1].Width == 2 {
		ln.Cells[col-1] = blankCell(ln.Cells[col-1].Style.Bg)
	}
	ln.Cells[col] = c
}

// combine combines a zero-width character, such as an accent, with
// the previous character, if they compose into a single character.
func (em *Emulator) combine(r rune) {
	ln := em.line()
	col := em.Cursor.Col - 1
	if em.Cursor.pendingWrap {
		col = em.Cursor.Col
	}
	for col > 0 && ln.Cells[col].Width == 0 {
		col--
	}
	if col < 0 || ln.Cells[col].Rune == 0 {
		return
	}
	c := &ln.Cells[col]
	rs := []rune(norm.NFC.String(string([]rune{c.Rune, r})))
	if len(rs) == 1 {
		c.Rune = rs[0]
	}
}

// index moves the cursor down one line, scrolling at the bottom
// of the scroll region (IND, LF).
func (em *Emulator) index() {
	cur := &em.Cursor
	cur.pendingWrap = false
	switch {
	case cur.Row == em.bottom:
		em.scrollUp(1)
	case cur.Row < em.Rows-1:
		cur.Row++
	}
}

// reverseIndex moves the cursor up one line, scrolling at the top
// of the scroll region (RI).
func (em *Emulator) reverseIndex() {
	cur := &em.Cursor
	cur.pendingWrap = false
	switch {
	case cur.Row == em.top:
		em.scrollDown(1)
	case cur.Row > 0:
		cur.Row--
	}
}

// scrollUp scrolls the lines of the scroll region up by n lines.
// Lines scrolled off the top of the primary screen go to the scrollback.
func (em *Emulator) scrollUp(n int) {
	em.scrollUpRegion(em.top, n, true)
}

// scrollUpRegion scrolls the lines from the top row to the bottom of
// the scroll region up by n lines. If scrollback is true, lines scrolled
// off the top of the primary screen go to the scrollback.
func (em *Emulator) scrollUpRegion(top, n int, scrollback bool) {
	n = min(n, em.bottom-top+1)
	if n <= 0 {
		return
	}
	lns := em.screen.lines
	if scrollback && top == 0 && em.screen == em.primary {
		for i := range n {
			em.scrolled = append(em.scrolled, lns[i].clone())
		}
		em.numScrolled += n
	}
	reg := lns[top : em.bottom+1]
	removed := slices.Clone(reg[:n])
	copy(reg, reg[n:])
	for i, ln := range removed { // reuse the line memory
		ln.clear(0, len(ln.Cells), em.bg())
		ln.Wrapped = false
		reg[len(reg)-n+i] = ln
	}
}

// scrollDown scrolls the lines of the scroll region down by n lines.
func (em *Emulator) scrollDown(n int) {
	em.scrollDownRegion(em.top, n)
}

// scrollDownRegion scrolls the lines from the top row to the bottom
// of the scroll region down by n lines.
func (em *Emulator) scrollDownRegion(top, n int) {
	n = min(n, em.bottom-top+1)
	if n <= 0 {
		return
	}
	reg := em.screen.lines[top : em.bottom+1]
	removed := slices.Clone(reg[len(reg)-n:])
	copy(reg[n:], reg)
	for i, ln := range removed {
		ln.clear(0, len(ln.Cells), em.bg())
		ln.Wrapped = false
		reg[i] = ln
	}
}

// insertLines inserts n blank lines at the cursor, within the scroll region (IL).
func (em *Emulator) insertLines(n int) {
	row := em.Cursor.Row
	if row < em.top || row > em.bottom {
		return
	}
	em.scrollDownRegion(row, n)
	em.Cursor.Col = 0
	em.Cursor.pendingWrap = false
}

// deleteLines deletes n lines at the cursor, within the scroll region (DL).
func (em *Emulator) deleteLines(n int) {
	row := em.Cursor.Row
	if row < em.top || row > em.bottom {
		return
	}
	em.scrollUpRegion(row, n, false)
	em.Cursor.Col = 0
	em.Cursor.pendingWrap = false
}

// insertCells inserts n blank cells at the cursor, shifting the rest
// of the line right (ICH).
func (em *Emulator) insertCells(n int) {
	ln := em.line()
	col := em.Cursor.Col
	n = min(n, em.Cols-col)
	if n <= 0 {
		return
	}
	cs := ln.Cells
	copy(cs[col+n:], cs[col:])
	ln.clear(col, col+n, em.bg())
	if last := &cs[em.Cols-1]; last.Width == 2 {
		*last = blankCell(last.Style.Bg)
	}
	em.Cursor.pendingWrap = false
}

// deleteCells deletes n cells at the cursor, shifting the rest
// of the line left (DCH).
func (em *Emulator) deleteCells(n int) {
	ln := em.line()
	col := em.Cursor.Col
	n = min(n, em.Cols-col)
	if n <= 0 {
		return
	}
	cs := ln.Cells
	copy(cs[col:], cs[col+n:])
	ln.clear(em.Cols-n, em.Cols, em.bg())
	em.Cursor.pendingWrap = false
}

// eraseCells erases n cells at the cursor (ECH).
func (em *Emulator) eraseCells(n int) {
	col := em.Cursor.Col
	em.line().clear(col, min(col+n, em.Cols), em.bg())
	em.Cursor.pendingWrap = false
}

// eraseLine erases part of the cursor line (EL):
// 0 to the end, 1 from the start, 2 all.
func (em *Emulator) eraseLine(mode int) {
	ln := em.line()
	col := em.Cursor.Col
	switch mode {
	case 0:
		ln.clear(col, em.Cols, em.bg())
		ln.Wrapped = false
	case 1:
		ln.clear(0, min(col+1, em.Cols), em.bg())
	case 2:
		ln.clear(0, em.Cols, em.bg())
		ln.Wrapped = false
	}
	em.Cursor.pendingWrap = false
}

// eraseDisplay erases part of the screen (ED): 0 from the cursor to
// the end, 1 from the start to the cursor, 2 all, 3 the scrollback.
// Erasing all of the primary screen first moves its text into the
// scrollback, so that it is not lost (as in most modern terminals).
func (em *Emulator) eraseDisplay(mode int) {
	lns := em.screen.lines
	row := em.Cursor.Row
	switch mode {
	case 0:
		em.eraseLine(0)
		for _, ln := range lns[row+1:] {
			ln.clear(0, em.Cols, em.bg())
			ln.Wrapped = false
		}
		if row > 0 {
			lns[row-1].Wrapped = lns[row-1].Wrapped && lns[row].Len() > 0
		}
	case 1:
		for _, ln := range lns[:row] {
			ln.clear(0, em.Cols, em.bg())
			ln.Wrapped = false
		}
		em.eraseLine(1)
	case 2:
		if em.screen == em.primary {
			em.saveScreenToScrollback()
		}
		for _, ln := range lns {
			ln.clear(0, em.Cols, em.bg())
			ln.Wrapped = false
		}
	case 3:
		em.scrolled = nil
		em.scrollbackCleared = true
	}
	em.Cursor.pendingWrap = false
}

// saveScreenToScrollback adds the lines of the primary screen, up to the
// last one with text, to the scrollback.
func (em *Emulator) saveScreenToScrollback() {
	lns := em.primary.lines
	n := len(lns)
	for n > 0 && lns[n-1].Len() == 0 {
		n--
	}
	for _, ln := range lns[:n] {
		em.scrolled = append(em.scrolled, ln.clone())
	}
	em.numScrolled += n
}

// decGraphics are the DEC special graphics characters for 0x5f-0x7e,
// used for drawing lines and boxes.
var decGraphics = [...]rune{
	' ', '◆', '▒', '␉', '␌', '␍', '␊', '°', '±', '␤', '␋', '┘', '┐', '┌', '└', '┼',
	'⎺', '⎻', '─', '⎼', '⎽', '├', '┤', '┴', '┬', '│', '≤', '≥', 'π', '≠', '£', '·',
}
