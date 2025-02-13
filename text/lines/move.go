// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package lines

import (
	"cogentcore.org/core/core"
	"cogentcore.org/core/text/textpos"
)

// displayPos returns the local display position of rune
// at given source line and char: wrapped line, char.
// returns -1, -1 for an invalid source position.
func (ls *Lines) displayPos(pos textpos.Pos) textpos.Pos {
	if !ls.isValidPos(pos) {
		return textpos.Pos{-1, -1}
	}
	return ls.layout[pos.Line][pos.Char].ToPos()
}

// todo: pass and return cursor column for up / down

// moveForward moves given source position forward given number of steps.
func (ls *Lines) moveForward(pos textpos.Pos, steps int) textpos.Pos {
	if !ls.isValidPos(pos) {
		return pos
	}
	for i := range steps {
		pos.Char++
		llen := len(ls.lines[pos.Ln])
		if pos.Char > llen {
			if pos.Line < len(ls.lines)-1 {
				pos.Char = 0
				pos.Line++
			} else {
				pos.Char = llen
			}
		}
	}
	return pos
}

// moveForwardWord moves the cursor forward by words
func (ls *Lines) moveForwardWord(pos textpos.Pos, steps int) textpos.Pos {
	if !ls.isValidPos(pos) {
		return pos
	}
	for i := 0; i < steps; i++ {
		txt := ed.Buffer.Line(pos.Line)
		sz := len(txt)
		if sz > 0 && pos.Char < sz {
			ch := pos.Ch
			var done = false
			for ch < sz && !done { // if on a wb, go past
				r1 := txt[ch]
				r2 := rune(-1)
				if ch < sz-1 {
					r2 = txt[ch+1]
				}
				if core.IsWordBreak(r1, r2) { // todo: local
					ch++
				} else {
					done = true
				}
			}
			done = false
			for ch < sz && !done {
				r1 := txt[ch]
				r2 := rune(-1)
				if ch < sz-1 {
					r2 = txt[ch+1]
				}
				if !core.IsWordBreak(r1, r2) {
					ch++
				} else {
					done = true
				}
			}
			pos.Char = ch
		} else {
			if pos.Line < ed.NumLines-1 {
				pos.Char = 0
				pos.Line++
			} else {
				pos.Char = ed.Buffer.LineLen(pos.Line)
			}
		}
	}
}

// moveDown moves the cursor down line(s)
func (ls *Lines) moveDown(steps int) {
	if !ls.isValidPos(pos) {
		return pos
	}
	org := pos
	pos := pos
	for i := 0; i < steps; i++ {
		gotwrap := false
		if wln := ed.wrappedLines(pos.Line); wln > 1 {
			si, ri, _ := ed.wrappedLineNumber(pos)
			if si < wln-1 {
				si++
				mxlen := min(len(ed.renders[pos.Line].Spans[si].Text), ed.cursorColumn)
				if ed.cursorColumn < mxlen {
					ri = ed.cursorColumn
				} else {
					ri = mxlen
				}
				nwc, _ := ed.renders[pos.Line].SpanPosToRuneIndex(si, ri)
				pos.Char = nwc
				gotwrap = true

			}
		}
		if !gotwrap {
			pos.Line++
			if pos.Line >= ed.NumLines {
				pos.Line = ed.NumLines - 1
				break
			}
			mxlen := min(ed.Buffer.LineLen(pos.Line), ed.cursorColumn)
			if ed.cursorColumn < mxlen {
				pos.Char = ed.cursorColumn
			} else {
				pos.Char = mxlen
			}
		}
	}
}

// cursorPageDown moves the cursor down page(s), where a page is defined abcdef
// dynamically as just moving the cursor off the screen
func (ls *Lines) movePageDown(steps int) {
	if !ls.isValidPos(pos) {
		return pos
	}
	org := pos
	for i := 0; i < steps; i++ {
		lvln := ed.lastVisibleLine(pos.Line)
		pos.Line = lvln
		if pos.Line >= ed.NumLines {
			pos.Line = ed.NumLines - 1
		}
		pos.Char = min(ed.Buffer.LineLen(pos.Line), ed.cursorColumn)
		ed.scrollCursorToTop()
		ed.renderCursor(true)
	}
}

// moveBackward moves the cursor backward
func (ls *Lines) moveBackward(steps int) {
	if !ls.isValidPos(pos) {
		return pos
	}
	org := pos
	for i := 0; i < steps; i++ {
		pos.Ch--
		if pos.Char < 0 {
			if pos.Line > 0 {
				pos.Line--
				pos.Char = ed.Buffer.LineLen(pos.Line)
			} else {
				pos.Char = 0
			}
		}
	}
}

// moveBackwardWord moves the cursor backward by words
func (ls *Lines) moveBackwardWord(steps int) {
	if !ls.isValidPos(pos) {
		return pos
	}
	org := pos
	for i := 0; i < steps; i++ {
		txt := ed.Buffer.Line(pos.Line)
		sz := len(txt)
		if sz > 0 && pos.Char > 0 {
			ch := min(pos.Ch, sz-1)
			var done = false
			for ch < sz && !done { // if on a wb, go past
				r1 := txt[ch]
				r2 := rune(-1)
				if ch > 0 {
					r2 = txt[ch-1]
				}
				if core.IsWordBreak(r1, r2) {
					ch--
					if ch == -1 {
						done = true
					}
				} else {
					done = true
				}
			}
			done = false
			for ch < sz && ch >= 0 && !done {
				r1 := txt[ch]
				r2 := rune(-1)
				if ch > 0 {
					r2 = txt[ch-1]
				}
				if !core.IsWordBreak(r1, r2) {
					ch--
				} else {
					done = true
				}
			}
			pos.Char = ch
		} else {
			if pos.Line > 0 {
				pos.Line--
				pos.Char = ed.Buffer.LineLen(pos.Line)
			} else {
				pos.Char = 0
			}
		}
	}
}

// moveUp moves the cursor up line(s)
func (ls *Lines) moveUp(steps int) {
	if !ls.isValidPos(pos) {
		return pos
	}
	org := pos
	pos := pos
	for i := 0; i < steps; i++ {
		gotwrap := false
		if wln := ed.wrappedLines(pos.Line); wln > 1 {
			si, ri, _ := ed.wrappedLineNumber(pos)
			if si > 0 {
				ri = ed.cursorColumn
				nwc, _ := ed.renders[pos.Line].SpanPosToRuneIndex(si-1, ri)
				if nwc == pos.Char {
					ed.cursorColumn = 0
					ri = 0
					nwc, _ = ed.renders[pos.Line].SpanPosToRuneIndex(si-1, ri)
				}
				pos.Char = nwc
				gotwrap = true
			}
		}
		if !gotwrap {
			pos.Line--
			if pos.Line < 0 {
				pos.Line = 0
				break
			}
			if wln := ed.wrappedLines(pos.Line); wln > 1 { // just entered end of wrapped line
				si := wln - 1
				ri := ed.cursorColumn
				nwc, _ := ed.renders[pos.Line].SpanPosToRuneIndex(si, ri)
				pos.Char = nwc
			} else {
				mxlen := min(ed.Buffer.LineLen(pos.Line), ed.cursorColumn)
				if ed.cursorColumn < mxlen {
					pos.Char = ed.cursorColumn
				} else {
					pos.Char = mxlen
				}
			}
		}
	}
}

// movePageUp moves the cursor up page(s), where a page is defined
// dynamically as just moving the cursor off the screen
func (ls *Lines) movePageUp(steps int) {
	if !ls.isValidPos(pos) {
		return pos
	}
	org := pos
	for i := 0; i < steps; i++ {
		lvln := ed.firstVisibleLine(pos.Line)
		pos.Line = lvln
		if pos.Line <= 0 {
			pos.Line = 0
		}
		pos.Char = min(ed.Buffer.LineLen(pos.Line), ed.cursorColumn)
		ed.scrollCursorToBottom()
		ed.renderCursor(true)
	}
}
