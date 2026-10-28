// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package vt

// Resize changes the size of the terminal. The text of the primary
// screen is rewrapped to the new number of columns, and lines that
// no longer fit above the cursor go to the scrollback. The alternate
// screen is cut or extended, as full-screen programs redraw it.
func (em *Emulator) Resize(cols, rows int) {
	cols, rows = max(cols, 1), max(rows, 1)
	if cols == em.Cols && rows == em.Rows {
		return
	}
	alt := em.IsAltScreen()
	pcur, acur := em.Cursor, em.alt.saved
	if alt {
		pcur, acur = em.primary.saved, em.Cursor
	}
	oldCols := em.Cols
	em.Cols, em.Rows = cols, rows

	// the primary screen
	if cols != oldCols {
		em.reflow(&pcur)
	} else {
		em.resizeRows(em.primary, &pcur, true)
	}
	// the alternate screen
	for _, ln := range em.alt.lines {
		ln.resize(cols)
	}
	em.resizeRows(em.alt, &acur, false)
	if alt {
		em.primary.saved, em.Cursor = pcur, acur
	} else {
		em.Cursor, em.alt.saved = pcur, acur
	}
	clampCursor(&em.primary.saved, cols, rows)
	clampCursor(&em.Cursor, cols, rows)
	em.top, em.bottom = 0, rows-1
	em.resetTabStops()
}

// clampCursor keeps the cursor within the given size.
func clampCursor(cur *Cursor, cols, rows int) {
	cur.Row = max(0, min(cur.Row, rows-1))
	cur.Col = max(0, min(cur.Col, cols-1))
	cur.pendingWrap = false
}

// resizeRows changes the number of lines of the given screen to
// em.Rows, which must already have em.Cols cells, keeping the cursor
// line. Lines removed from the top go to the scrollback if scrollback is set.
func (em *Emulator) resizeRows(sc *screen, cur *Cursor, scrollback bool) {
	lns := sc.lines
	rows := em.Rows
	if len(lns) > rows {
		// remove blank lines below the cursor first
		n := len(lns)
		for n > rows && n-1 > cur.Row && lns[n-1].Len() == 0 {
			n--
		}
		lns = lns[:n]
		if over := len(lns) - rows; over > 0 {
			if over > cur.Row { // keep the cursor line: cut the rest from the bottom
				lns = lns[:len(lns)-(over-cur.Row)]
				over = cur.Row
			}
			if scrollback {
				em.scrolled = append(em.scrolled, lns[:over]...)
				em.numScrolled += over
			}
			lns = lns[over:]
			cur.Row -= over
		}
	}
	for len(lns) < rows {
		lns = append(lns, newLine(em.Cols, ColorDefault))
	}
	sc.lines = lns
}

// reflow rewraps the lines of the primary screen to em.Cols columns,
// keeping the cursor at the same place in the text.
func (em *Emulator) reflow(cur *Cursor) {
	lns := em.primary.lines
	// logical lines: rows joined at soft wraps
	type logical struct {
		cells     []Cell
		cursorOff int // offset of the cursor in the cells, or -1
	}
	var lls []logical
	var ll logical
	ll.cursorOff = -1
	last := 0 // last row with text, or the cursor
	for r, ln := range lns {
		if r == cur.Row {
			ll.cursorOff = len(ll.cells) + cur.Col
		}
		ll.cells = append(ll.cells, ln.Cells...)
		if ln.Len() > 0 || r == cur.Row {
			last = len(lls)
		}
		if !ln.Wrapped || r == len(lns)-1 {
			lls = append(lls, ll)
			ll = logical{cursorOff: -1}
		}
	}
	lls = lls[:last+1]

	var out []*Line
	crow, ccol := 0, 0
	cols := em.Cols
	for _, ll := range lls {
		n := len(ll.cells)
		for n > 0 && ll.cells[n-1].IsBlank() {
			n--
		}
		if ll.cursorOff >= 0 {
			n = max(n, min(ll.cursorOff, len(ll.cells)))
		}
		cells := ll.cells[:n]
		ln := newLine(cols, ColorDefault)
		col := 0
		place := func(i int) { // the cursor at cell i of the logical line
			if i == ll.cursorOff {
				crow, ccol = len(out), col
			}
		}
		for i := range len(cells) {
			c := cells[i]
			if c.Width == 0 { // the cell after a wide character, added with it
				place(i)
				continue
			}
			w := int(c.Width)
			if col+w > cols {
				ln.Wrapped = true
				out = append(out, ln)
				ln = newLine(cols, ColorDefault)
				col = 0
			}
			place(i)
			ln.Cells[col] = c
			if w == 2 && col+1 < cols {
				ln.Cells[col+1] = Cell{Style: c.Style}
			}
			col += w
		}
		if ll.cursorOff >= len(cells) {
			if col >= cols {
				crow, ccol = len(out), cols-1
			} else {
				crow, ccol = len(out), col
			}
		}
		out = append(out, ln)
	}
	cur.Row, cur.Col = crow, ccol
	em.primary.lines = out
	em.resizeRows(em.primary, cur, true)
}
