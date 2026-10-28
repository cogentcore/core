// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package vt

import (
	"fmt"

	"github.com/charmbracelet/x/ansi"
)

// execute handles a C0 control character.
func (em *Emulator) execute(b byte) {
	cur := &em.Cursor
	switch b {
	case ansi.BEL:
		if em.Bell != nil {
			em.Bell()
		}
	case ansi.BS:
		if cur.pendingWrap {
			cur.pendingWrap = false
		} else if cur.Col > 0 {
			cur.Col--
		}
	case ansi.HT:
		em.tabForward(1)
	case ansi.LF, ansi.VT, ansi.FF:
		em.index()
		if em.Modes.NewLine {
			cur.Col = 0
		}
	case ansi.CR:
		cur.Col = 0
		cur.pendingWrap = false
	case ansi.SO:
		cur.shift = 1
	case ansi.SI:
		cur.shift = 0
	}
}

// tabForward moves the cursor to the n-th next tab stop.
func (em *Emulator) tabForward(n int) {
	cur := &em.Cursor
	for ; n > 0 && cur.Col < em.Cols-1; n-- {
		cur.Col++
		for cur.Col < em.Cols-1 && !em.tabStops[cur.Col] {
			cur.Col++
		}
	}
	cur.pendingWrap = false
}

// tabBackward moves the cursor to the n-th previous tab stop.
func (em *Emulator) tabBackward(n int) {
	cur := &em.Cursor
	for ; n > 0 && cur.Col > 0; n-- {
		cur.Col--
		for cur.Col > 0 && !em.tabStops[cur.Col] {
			cur.Col--
		}
	}
	cur.pendingWrap = false
}

// moveTo moves the cursor to the given position, which is relative
// to the scroll region in origin mode.
func (em *Emulator) moveTo(row, col int) {
	cur := &em.Cursor
	minRow, maxRow := 0, em.Rows-1
	if cur.originMode {
		row += em.top
		minRow, maxRow = em.top, em.bottom
	}
	cur.Row = max(minRow, min(row, maxRow))
	cur.Col = max(0, min(col, em.Cols-1))
	cur.pendingWrap = false
}

// moveRel moves the cursor by the given amounts, stopping at the
// scroll region margins if it starts within them.
func (em *Emulator) moveRel(drow, dcol int) {
	cur := &em.Cursor
	row := cur.Row + drow
	minRow, maxRow := 0, em.Rows-1
	if cur.Row >= em.top && cur.Row <= em.bottom {
		minRow, maxRow = em.top, em.bottom
	}
	cur.Row = max(minRow, min(row, maxRow))
	cur.Col = max(0, min(cur.Col+dcol, em.Cols-1))
	cur.pendingWrap = false
}

// handleCSI handles a control sequence.
func (em *Emulator) handleCSI(cmd ansi.Cmd, params ansi.Params) {
	cur := &em.Cursor
	// param returns parameter i, with 0 and missing values as def.
	param := func(i, def int) int {
		p, _, _ := params.Param(i, def)
		if p == 0 {
			return def
		}
		return p
	}
	// param0 returns parameter i, with missing values as 0.
	param0 := func(i int) int {
		p, _, _ := params.Param(i, 0)
		return p
	}
	prefix, inter := cmd.Prefix(), cmd.Intermediate()
	switch {
	case prefix == '?':
		em.handleCSIPrivate(cmd, params)
		return
	case prefix == '>':
		if cmd.Final() == 'c' { // secondary device attributes
			em.reply("\x1b[>0;10;1c")
		}
		return
	case prefix != 0:
		return
	case inter == ' ':
		if cmd.Final() == 'q' { // DECSCUSR cursor shape
			em.Modes.CursorShape = param0(0)
		}
		return
	case inter == '!':
		if cmd.Final() == 'p' { // DECSTR soft reset
			em.softReset()
		}
		return
	case inter != 0:
		return
	}
	switch cmd.Final() {
	case '@': // ICH
		em.insertCells(param(0, 1))
	case 'A': // CUU
		em.moveRel(-param(0, 1), 0)
	case 'B', 'e': // CUD, VPR
		em.moveRel(param(0, 1), 0)
	case 'C', 'a': // CUF, HPR
		em.moveRel(0, param(0, 1))
	case 'D': // CUB
		em.moveRel(0, -param(0, 1))
	case 'E': // CNL
		em.moveRel(param(0, 1), 0)
		cur.Col = 0
	case 'F': // CPL
		em.moveRel(-param(0, 1), 0)
		cur.Col = 0
	case 'G', '`': // CHA, HPA
		cur.Col = max(0, min(param(0, 1)-1, em.Cols-1))
		cur.pendingWrap = false
	case 'H', 'f': // CUP, HVP
		em.moveTo(param(0, 1)-1, param(1, 1)-1)
	case 'I': // CHT
		em.tabForward(param(0, 1))
	case 'J': // ED
		em.eraseDisplay(param0(0))
	case 'K': // EL
		em.eraseLine(param0(0))
	case 'L': // IL
		em.insertLines(param(0, 1))
	case 'M': // DL
		em.deleteLines(param(0, 1))
	case 'P': // DCH
		em.deleteCells(param(0, 1))
	case 'S': // SU
		em.scrollUp(param(0, 1))
	case 'T': // SD
		if len(params) <= 1 {
			em.scrollDown(param(0, 1))
		}
	case 'X': // ECH
		em.eraseCells(param(0, 1))
	case 'Z': // CBT
		em.tabBackward(param(0, 1))
	case 'b': // REP
		if em.lastRune != 0 {
			for range min(param(0, 1), em.Cols*em.Rows) {
				em.print(em.lastRune)
			}
		}
	case 'c': // primary device attributes: VT220 with ANSI color
		if param0(0) == 0 {
			em.reply("\x1b[?62;22c")
		}
	case 'd': // VPA
		em.moveTo(param(0, 1)-1, cur.Col)
		if cur.originMode {
			cur.Row = max(em.top, min(param(0, 1)-1+em.top, em.bottom))
		}
	case 'g': // TBC
		switch param0(0) {
		case 0:
			em.tabStops[cur.Col] = false
		case 3:
			clear(em.tabStops)
		}
	case 'h', 'l': // SM, RM
		on := cmd.Final() == 'h'
		for i := range params {
			switch param0(i) {
			case 4:
				em.Modes.Insert = on
			case 20:
				em.Modes.NewLine = on
			}
		}
	case 'm': // SGR
		em.handleSGR(params)
	case 'n': // DSR
		switch param0(0) {
		case 5:
			em.reply("\x1b[0n")
		case 6:
			row := cur.Row
			if cur.originMode {
				row -= em.top
			}
			em.reply(fmt.Sprintf("\x1b[%d;%dR", row+1, cur.Col+1))
		}
	case 'r': // DECSTBM
		top, bottom := param(0, 1)-1, param(1, em.Rows)-1
		bottom = min(bottom, em.Rows-1)
		if top < bottom {
			em.top, em.bottom = top, bottom
			em.moveTo(0, 0)
		}
	case 's': // SCOSC
		em.saveCursor()
	case 'u': // SCORC
		em.restoreCursor()
	case 't': // window operations
		switch param0(0) {
		case 18: // text area size in characters
			em.reply(fmt.Sprintf("\x1b[8;%d;%dt", em.Rows, em.Cols))
		}
	}
}

// handleCSIPrivate handles a DEC private control sequence (CSI ? ...).
func (em *Emulator) handleCSIPrivate(cmd ansi.Cmd, params ansi.Params) {
	switch cmd.Final() {
	case 'h', 'l':
		on := cmd.Final() == 'h'
		for i := range params {
			p, _, _ := params.Param(i, 0)
			em.setPrivateMode(p, on)
		}
	case 'J': // DECSED
		p, _, _ := params.Param(0, 0)
		em.eraseDisplay(p)
	case 'K': // DECSEL
		p, _, _ := params.Param(0, 0)
		em.eraseLine(p)
	case 'n': // DECXCPR
		if p, _, _ := params.Param(0, 0); p == 6 {
			em.reply(fmt.Sprintf("\x1b[?%d;%dR", em.Cursor.Row+1, em.Cursor.Col+1))
		}
	}
}

// setPrivateMode sets the given DEC private mode.
func (em *Emulator) setPrivateMode(mode int, on bool) {
	md := &em.Modes
	switch mode {
	case 1:
		md.AppCursorKeys = on
	case 3: // DECCOLM: 132 columns, which we do not do, but it clears
		em.eraseDisplay(2)
		em.moveTo(0, 0)
	case 5:
		md.ReverseVideo = on
	case 6:
		em.Cursor.originMode = on
		em.moveTo(0, 0)
	case 7:
		md.AutoWrap = on
	case 9:
		md.Mouse = mouseMode(on, MouseX10)
	case 25:
		md.CursorHidden = !on
	case 47, 1047:
		em.setAltScreen(on, false)
	case 1000:
		md.Mouse = mouseMode(on, MouseNormal)
	case 1002:
		md.Mouse = mouseMode(on, MouseButton)
	case 1003:
		md.Mouse = mouseMode(on, MouseAny)
	case 1004:
		md.FocusEvents = on
	case 1006:
		md.MouseSGR = on
	case 1007:
		md.AltScroll = on
	case 1048:
		if on {
			em.saveCursor()
		} else {
			em.restoreCursor()
		}
	case 1049:
		em.setAltScreen(on, true)
	case 2004:
		md.BracketedPaste = on
	}
}

// mouseMode returns the mouse mode for setting the given mode on or off.
func mouseMode(on bool, mode MouseModes) MouseModes {
	if on {
		return mode
	}
	return MouseNone
}

// setAltScreen switches to or from the alternate screen. If
// saveCursor is set, the cursor is saved before switching to it and
// restored after switching back, and it is cleared (mode 1049).
func (em *Emulator) setAltScreen(on, saveCursor bool) {
	if on == em.IsAltScreen() {
		return
	}
	if on {
		if saveCursor {
			em.saveCursor()
		}
		em.screen = em.alt
		if saveCursor {
			em.alt.clear()
		}
	} else {
		if em.screen == em.alt && saveCursor {
			em.alt.clear()
		}
		em.screen = em.primary
		if saveCursor {
			em.restoreCursor()
		}
	}
	em.Cursor.pendingWrap = false
}

// saveCursor saves the cursor state (DECSC).
func (em *Emulator) saveCursor() {
	em.screen.saved = em.Cursor
}

// restoreCursor restores the saved cursor state (DECRC).
func (em *Emulator) restoreCursor() {
	em.Cursor = em.screen.saved
	em.Cursor.Row = min(em.Cursor.Row, em.Rows-1)
	em.Cursor.Col = min(em.Cursor.Col, em.Cols-1)
}

// softReset does a soft terminal reset (DECSTR).
func (em *Emulator) softReset() {
	em.Modes.CursorHidden = false
	em.Modes.Insert = false
	em.Modes.AutoWrap = true
	em.Modes.AppCursorKeys = false
	em.Modes.AppKeypad = false
	em.top, em.bottom = 0, em.Rows-1
	em.Cursor.Style = Style{}
	em.Cursor.originMode = false
	em.Cursor.charsets = [2]charset{}
	em.Cursor.shift = 0
	em.screen.saved = Cursor{}
}

// handleESC handles an escape sequence.
func (em *Emulator) handleESC(cmd ansi.Cmd) {
	cur := &em.Cursor
	inter, final := cmd.Intermediate(), cmd.Final()
	switch inter {
	case '(', ')': // designate G0 or G1 character set
		g := 0
		if inter == ')' {
			g = 1
		}
		cs := charsetASCII
		if final == '0' {
			cs = charsetDECGraphics
		}
		cur.charsets[g] = cs
		return
	case '#':
		if final == '8' { // DECALN: fill the screen with E
			for _, ln := range em.screen.lines {
				for i := range ln.Cells {
					ln.Cells[i] = Cell{Rune: 'E', Width: 1}
				}
				ln.Wrapped = false
			}
		}
		return
	case 0:
	default:
		return
	}
	switch final {
	case '7': // DECSC
		em.saveCursor()
	case '8': // DECRC
		em.restoreCursor()
	case 'D': // IND
		em.index()
	case 'E': // NEL
		em.index()
		cur.Col = 0
	case 'H': // HTS
		em.tabStops[cur.Col] = true
	case 'M': // RI
		em.reverseIndex()
	case 'c': // RIS
		em.reset()
		em.eraseDisplay(3)
	case '=': // DECKPAM
		em.Modes.AppKeypad = true
	case '>': // DECKPNM
		em.Modes.AppKeypad = false
	}
}
