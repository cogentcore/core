// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package texteditor

import (
	"image"

	"cogentcore.org/core/base/reflectx"
	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/parse/lexer"
	"cogentcore.org/core/texteditor/textbuf"
)

///////////////////////////////////////////////////////////////////////////////
//  Cursor Navigation

// cursorMovedEvent sends the event that cursor has moved
func (ed *Editor) cursorMovedEvent() {
	ed.Send(events.Input, nil)
}

// validateCursor sets current cursor to a valid cursor position
func (ed *Editor) validateCursor() {
	if ed.Buffer != nil {
		ed.CursorPos = ed.Buffer.validPos(ed.CursorPos)
	} else {
		ed.CursorPos = lexer.PosZero
	}
}

// wrappedLines returns the number of wrapped lines (spans) for given line number
func (ed *Editor) wrappedLines(ln int) int {
	if ln >= len(ed.renders) {
		return 0
	}
	return len(ed.renders[ln].Spans)
}

// wrappedLineNumber returns the wrapped line number (span index) and rune index
// within that span of the given character position within line in position,
// and false if out of range (last valid position returned in that case -- still usable).
func (ed *Editor) wrappedLineNumber(pos lexer.Pos) (si, ri int, ok bool) {
	if pos.Ln >= len(ed.renders) {
		return 0, 0, false
	}
	return ed.renders[pos.Ln].RuneSpanPos(pos.Ch)
}

// setCursor sets a new cursor position, enforcing it in range.
// This is the main final pathway for all cursor movement.
func (ed *Editor) setCursor(pos lexer.Pos) {
	if ed.NumLines == 0 || ed.Buffer == nil {
		ed.CursorPos = lexer.PosZero
		return
	}

	ed.ClearScopelights()
	ed.CursorPos = ed.Buffer.validPos(pos)
	ed.cursorMovedEvent()
	txt := ed.Buffer.line(ed.CursorPos.Ln)
	ch := ed.CursorPos.Ch
	if ch < len(txt) {
		r := txt[ch]
		if r == '{' || r == '}' || r == '(' || r == ')' || r == '[' || r == ']' {
			tp, found := ed.Buffer.braceMatch(txt[ch], ed.CursorPos)
			if found {
				ed.scopelights = append(ed.scopelights, textbuf.NewRegionPos(ed.CursorPos, lexer.Pos{ed.CursorPos.Ln, ed.CursorPos.Ch + 1}))
				ed.scopelights = append(ed.scopelights, textbuf.NewRegionPos(tp, lexer.Pos{tp.Ln, tp.Ch + 1}))
			}
		}
	}
	ed.NeedsRender()
}

// SetCursorShow sets a new cursor position, enforcing it in range, and shows
// the cursor (scroll to if hidden, render)
func (ed *Editor) SetCursorShow(pos lexer.Pos) {
	ed.setCursor(pos)
	ed.scrollCursorToCenterIfHidden()
	ed.renderCursor(true)
}

// SetCursorTarget sets a new cursor target position, ensures that it is visible
func (ed *Editor) SetCursorTarget(pos lexer.Pos) {
	ed.targetSet = true
	ed.cursorTarget = pos
	ed.SetCursorShow(pos)
	ed.NeedsRender()
	// fmt.Println(ed, "set target:", ed.CursorTarg)
}

// setCursorColumn sets the current target cursor column (cursorColumn) to that
// of the given position
func (ed *Editor) setCursorColumn(pos lexer.Pos) {
	if wln := ed.wrappedLines(pos.Ln); wln > 1 {
		si, ri, ok := ed.wrappedLineNumber(pos)
		if ok && si > 0 {
			ed.cursorColumn = ri
		} else {
			ed.cursorColumn = pos.Ch
		}
	} else {
		ed.cursorColumn = pos.Ch
	}
}

// savePosHistory saves the cursor position in history stack of cursor positions
func (ed *Editor) savePosHistory(pos lexer.Pos) {
	if ed.Buffer == nil {
		return
	}
	ed.Buffer.savePosHistory(pos)
	ed.posHistoryIndex = len(ed.Buffer.posHistory) - 1
}

// CursorToHistoryPrev moves cursor to previous position on history list --
// returns true if moved
func (ed *Editor) CursorToHistoryPrev() bool {
	if ed.NumLines == 0 || ed.Buffer == nil {
		ed.CursorPos = lexer.PosZero
		return false
	}
	sz := len(ed.Buffer.posHistory)
	if sz == 0 {
		return false
	}
	ed.posHistoryIndex--
	if ed.posHistoryIndex < 0 {
		ed.posHistoryIndex = 0
		return false
	}
	ed.posHistoryIndex = min(sz-1, ed.posHistoryIndex)
	pos := ed.Buffer.posHistory[ed.posHistoryIndex]
	ed.CursorPos = ed.Buffer.validPos(pos)
	ed.cursorMovedEvent()
	ed.scrollCursorToCenterIfHidden()
	ed.renderCursor(true)
	return true
}

// CursorToHistoryNext moves cursor to previous position on history list --
// returns true if moved
func (ed *Editor) CursorToHistoryNext() bool {
	if ed.NumLines == 0 || ed.Buffer == nil {
		ed.CursorPos = lexer.PosZero
		return false
	}
	sz := len(ed.Buffer.posHistory)
	if sz == 0 {
		return false
	}
	ed.posHistoryIndex++
	if ed.posHistoryIndex >= sz-1 {
		ed.posHistoryIndex = sz - 1
		return false
	}
	pos := ed.Buffer.posHistory[ed.posHistoryIndex]
	ed.CursorPos = ed.Buffer.validPos(pos)
	ed.cursorMovedEvent()
	ed.scrollCursorToCenterIfHidden()
	ed.renderCursor(true)
	return true
}

// selectRegionUpdate updates current select region based on given cursor position
// relative to SelectStart position
func (ed *Editor) selectRegionUpdate(pos lexer.Pos) {
	if pos.IsLess(ed.selectStart) {
		ed.SelectRegion.Start = pos
		ed.SelectRegion.End = ed.selectStart
	} else {
		ed.SelectRegion.Start = ed.selectStart
		ed.SelectRegion.End = pos
	}
}

// cursorSelect updates selection based on cursor movements, given starting
// cursor position and ed.CursorPos is current
func (ed *Editor) cursorSelect(org lexer.Pos) {
	if !ed.selectMode {
		return
	}
	ed.selectRegionUpdate(ed.CursorPos)
}

// cursorForward moves the cursor forward
func (ed *Editor) cursorForward(steps int) {
	ed.validateCursor()
	org := ed.CursorPos
	for i := 0; i < steps; i++ {
		ed.CursorPos.Ch++
		if ed.CursorPos.Ch > ed.Buffer.lineLen(ed.CursorPos.Ln) {
			if ed.CursorPos.Ln < ed.NumLines-1 {
				ed.CursorPos.Ch = 0
				ed.CursorPos.Ln++
			} else {
				ed.CursorPos.Ch = ed.Buffer.lineLen(ed.CursorPos.Ln)
			}
		}
	}
	ed.setCursorColumn(ed.CursorPos)
	ed.SetCursorShow(ed.CursorPos)
	ed.cursorSelect(org)
	ed.NeedsRender()
}

// cursorForwardWord moves the cursor forward by words
func (ed *Editor) cursorForwardWord(steps int) {
	ed.validateCursor()
	org := ed.CursorPos
	for i := 0; i < steps; i++ {
		txt := ed.Buffer.line(ed.CursorPos.Ln)
		sz := len(txt)
		if sz > 0 && ed.CursorPos.Ch < sz {
			ch := ed.CursorPos.Ch
			var done = false
			for ch < sz && !done { // if on a wb, go past
				r1 := txt[ch]
				r2 := rune(-1)
				if ch < sz-1 {
					r2 = txt[ch+1]
				}
				if core.IsWordBreak(r1, r2) {
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
			ed.CursorPos.Ch = ch
		} else {
			if ed.CursorPos.Ln < ed.NumLines-1 {
				ed.CursorPos.Ch = 0
				ed.CursorPos.Ln++
			} else {
				ed.CursorPos.Ch = ed.Buffer.lineLen(ed.CursorPos.Ln)
			}
		}
	}
	ed.setCursorColumn(ed.CursorPos)
	ed.SetCursorShow(ed.CursorPos)
	ed.cursorSelect(org)
	ed.NeedsRender()
}

// cursorDown moves the cursor down line(s)
func (ed *Editor) cursorDown(steps int) {
	ed.validateCursor()
	org := ed.CursorPos
	pos := ed.CursorPos
	for i := 0; i < steps; i++ {
		gotwrap := false
		if wln := ed.wrappedLines(pos.Ln); wln > 1 {
			si, ri, _ := ed.wrappedLineNumber(pos)
			if si < wln-1 {
				si++
				mxlen := min(len(ed.renders[pos.Ln].Spans[si].Text), ed.cursorColumn)
				if ed.cursorColumn < mxlen {
					ri = ed.cursorColumn
				} else {
					ri = mxlen
				}
				nwc, _ := ed.renders[pos.Ln].SpanPosToRuneIndex(si, ri)
				pos.Ch = nwc
				gotwrap = true

			}
		}
		if !gotwrap {
			pos.Ln++
			if pos.Ln >= ed.NumLines {
				pos.Ln = ed.NumLines - 1
				break
			}
			mxlen := min(ed.Buffer.lineLen(pos.Ln), ed.cursorColumn)
			if ed.cursorColumn < mxlen {
				pos.Ch = ed.cursorColumn
			} else {
				pos.Ch = mxlen
			}
		}
	}
	ed.SetCursorShow(pos)
	ed.cursorSelect(org)
	ed.NeedsRender()
}

// cursorPageDown moves the cursor down page(s), where a page is defined abcdef
// dynamically as just moving the cursor off the screen
func (ed *Editor) cursorPageDown(steps int) {
	ed.validateCursor()
	org := ed.CursorPos
	for i := 0; i < steps; i++ {
		lvln := ed.lastVisibleLine(ed.CursorPos.Ln)
		ed.CursorPos.Ln = lvln
		if ed.CursorPos.Ln >= ed.NumLines {
			ed.CursorPos.Ln = ed.NumLines - 1
		}
		ed.CursorPos.Ch = min(ed.Buffer.lineLen(ed.CursorPos.Ln), ed.cursorColumn)
		ed.scrollCursorToTop()
		ed.renderCursor(true)
	}
	ed.setCursor(ed.CursorPos)
	ed.cursorSelect(org)
	ed.NeedsRender()
}

// cursorBackward moves the cursor backward
func (ed *Editor) cursorBackward(steps int) {
	ed.validateCursor()
	org := ed.CursorPos
	for i := 0; i < steps; i++ {
		ed.CursorPos.Ch--
		if ed.CursorPos.Ch < 0 {
			if ed.CursorPos.Ln > 0 {
				ed.CursorPos.Ln--
				ed.CursorPos.Ch = ed.Buffer.lineLen(ed.CursorPos.Ln)
			} else {
				ed.CursorPos.Ch = 0
			}
		}
	}
	ed.setCursorColumn(ed.CursorPos)
	ed.SetCursorShow(ed.CursorPos)
	ed.cursorSelect(org)
	ed.NeedsRender()
}

// cursorBackwardWord moves the cursor backward by words
func (ed *Editor) cursorBackwardWord(steps int) {
	ed.validateCursor()
	org := ed.CursorPos
	for i := 0; i < steps; i++ {
		txt := ed.Buffer.line(ed.CursorPos.Ln)
		sz := len(txt)
		if sz > 0 && ed.CursorPos.Ch > 0 {
			ch := min(ed.CursorPos.Ch, sz-1)
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
			ed.CursorPos.Ch = ch
		} else {
			if ed.CursorPos.Ln > 0 {
				ed.CursorPos.Ln--
				ed.CursorPos.Ch = ed.Buffer.lineLen(ed.CursorPos.Ln)
			} else {
				ed.CursorPos.Ch = 0
			}
		}
	}
	ed.setCursorColumn(ed.CursorPos)
	ed.SetCursorShow(ed.CursorPos)
	ed.cursorSelect(org)
	ed.NeedsRender()
}

// cursorUp moves the cursor up line(s)
func (ed *Editor) cursorUp(steps int) {
	ed.validateCursor()
	org := ed.CursorPos
	pos := ed.CursorPos
	for i := 0; i < steps; i++ {
		gotwrap := false
		if wln := ed.wrappedLines(pos.Ln); wln > 1 {
			si, ri, _ := ed.wrappedLineNumber(pos)
			if si > 0 {
				ri = ed.cursorColumn
				nwc, _ := ed.renders[pos.Ln].SpanPosToRuneIndex(si-1, ri)
				if nwc == pos.Ch {
					ed.cursorColumn = 0
					ri = 0
					nwc, _ = ed.renders[pos.Ln].SpanPosToRuneIndex(si-1, ri)
				}
				pos.Ch = nwc
				gotwrap = true
			}
		}
		if !gotwrap {
			pos.Ln--
			if pos.Ln < 0 {
				pos.Ln = 0
				break
			}
			if wln := ed.wrappedLines(pos.Ln); wln > 1 { // just entered end of wrapped line
				si := wln - 1
				ri := ed.cursorColumn
				nwc, _ := ed.renders[pos.Ln].SpanPosToRuneIndex(si, ri)
				pos.Ch = nwc
			} else {
				mxlen := min(ed.Buffer.lineLen(pos.Ln), ed.cursorColumn)
				if ed.cursorColumn < mxlen {
					pos.Ch = ed.cursorColumn
				} else {
					pos.Ch = mxlen
				}
			}
		}
	}
	ed.SetCursorShow(pos)
	ed.cursorSelect(org)
	ed.NeedsRender()
}

// cursorPageUp moves the cursor up page(s), where a page is defined
// dynamically as just moving the cursor off the screen
func (ed *Editor) cursorPageUp(steps int) {
	ed.validateCursor()
	org := ed.CursorPos
	for i := 0; i < steps; i++ {
		lvln := ed.firstVisibleLine(ed.CursorPos.Ln)
		ed.CursorPos.Ln = lvln
		if ed.CursorPos.Ln <= 0 {
			ed.CursorPos.Ln = 0
		}
		ed.CursorPos.Ch = min(ed.Buffer.lineLen(ed.CursorPos.Ln), ed.cursorColumn)
		ed.scrollCursorToBottom()
		ed.renderCursor(true)
	}
	ed.setCursor(ed.CursorPos)
	ed.cursorSelect(org)
	ed.NeedsRender()
}

// cursorRecenter re-centers the view around the cursor position, toggling
// between putting cursor in middle, top, and bottom of view
func (ed *Editor) cursorRecenter() {
	ed.validateCursor()
	ed.savePosHistory(ed.CursorPos)
	cur := (ed.lastRecenter + 1) % 3
	switch cur {
	case 0:
		ed.scrollCursorToBottom()
	case 1:
		ed.scrollCursorToVerticalCenter()
	case 2:
		ed.scrollCursorToTop()
	}
	ed.lastRecenter = cur
}

// cursorStartLine moves the cursor to the start of the line, updating selection
// if select mode is active
func (ed *Editor) cursorStartLine() {
	ed.validateCursor()
	org := ed.CursorPos
	pos := ed.CursorPos

	gotwrap := false
	if wln := ed.wrappedLines(pos.Ln); wln > 1 {
		si, ri, _ := ed.wrappedLineNumber(pos)
		if si > 0 {
			ri = 0
			nwc, _ := ed.renders[pos.Ln].SpanPosToRuneIndex(si, ri)
			pos.Ch = nwc
			ed.CursorPos = pos
			ed.cursorColumn = ri
			gotwrap = true
		}
	}
	if !gotwrap {
		ed.CursorPos.Ch = 0
		ed.cursorColumn = ed.CursorPos.Ch
	}
	// fmt.Printf("sol cursorcol: %v\n", ed.CursorCol)
	ed.setCursor(ed.CursorPos)
	ed.scrollCursorToRight()
	ed.renderCursor(true)
	ed.cursorSelect(org)
	ed.NeedsRender()
}

// CursorStartDoc moves the cursor to the start of the text, updating selection
// if select mode is active
func (ed *Editor) CursorStartDoc() {
	ed.validateCursor()
	org := ed.CursorPos
	ed.CursorPos.Ln = 0
	ed.CursorPos.Ch = 0
	ed.cursorColumn = ed.CursorPos.Ch
	ed.setCursor(ed.CursorPos)
	ed.scrollCursorToTop()
	ed.renderCursor(true)
	ed.cursorSelect(org)
	ed.NeedsRender()
}

// cursorEndLine moves the cursor to the end of the text
func (ed *Editor) cursorEndLine() {
	ed.validateCursor()
	org := ed.CursorPos
	pos := ed.CursorPos

	gotwrap := false
	if wln := ed.wrappedLines(pos.Ln); wln > 1 {
		si, ri, _ := ed.wrappedLineNumber(pos)
		ri = len(ed.renders[pos.Ln].Spans[si].Text) - 1
		nwc, _ := ed.renders[pos.Ln].SpanPosToRuneIndex(si, ri)
		if si == len(ed.renders[pos.Ln].Spans)-1 { // last span
			ri++
			nwc++
		}
		ed.cursorColumn = ri
		pos.Ch = nwc
		ed.CursorPos = pos
		gotwrap = true
	}
	if !gotwrap {
		ed.CursorPos.Ch = ed.Buffer.lineLen(ed.CursorPos.Ln)
		ed.cursorColumn = ed.CursorPos.Ch
	}
	ed.setCursor(ed.CursorPos)
	ed.scrollCursorToRight()
	ed.renderCursor(true)
	ed.cursorSelect(org)
	ed.NeedsRender()
}

// cursorEndDoc moves the cursor to the end of the text, updating selection if
// select mode is active
func (ed *Editor) cursorEndDoc() {
	ed.validateCursor()
	org := ed.CursorPos
	ed.CursorPos.Ln = max(ed.NumLines-1, 0)
	ed.CursorPos.Ch = ed.Buffer.lineLen(ed.CursorPos.Ln)
	ed.cursorColumn = ed.CursorPos.Ch
	ed.setCursor(ed.CursorPos)
	ed.scrollCursorToBottom()
	ed.renderCursor(true)
	ed.cursorSelect(org)
	ed.NeedsRender()
}

// todo: ctrl+backspace = delete word
// shift+arrow = select
// uparrow = start / down = end

// cursorBackspace deletes character(s) immediately before cursor
func (ed *Editor) cursorBackspace(steps int) {
	ed.validateCursor()
	org := ed.CursorPos
	if ed.HasSelection() {
		org = ed.SelectRegion.Start
		ed.DeleteSelection()
		ed.SetCursorShow(org)
		return
	}
	// note: no update b/c signal from buf will drive update
	ed.cursorBackward(steps)
	ed.scrollCursorToCenterIfHidden()
	ed.renderCursor(true)
	ed.Buffer.DeleteText(ed.CursorPos, org, EditSignal)
	ed.NeedsRender()
}

// cursorDelete deletes character(s) immediately after the cursor
func (ed *Editor) cursorDelete(steps int) {
	ed.validateCursor()
	if ed.HasSelection() {
		ed.DeleteSelection()
		return
	}
	// note: no update b/c signal from buf will drive update
	org := ed.CursorPos
	ed.cursorForward(steps)
	ed.Buffer.DeleteText(org, ed.CursorPos, EditSignal)
	ed.SetCursorShow(org)
	ed.NeedsRender()
}

// cursorBackspaceWord deletes words(s) immediately before cursor
func (ed *Editor) cursorBackspaceWord(steps int) {
	ed.validateCursor()
	org := ed.CursorPos
	if ed.HasSelection() {
		ed.DeleteSelection()
		ed.SetCursorShow(org)
		return
	}
	// note: no update b/c signal from buf will drive update
	ed.cursorBackwardWord(steps)
	ed.scrollCursorToCenterIfHidden()
	ed.renderCursor(true)
	ed.Buffer.DeleteText(ed.CursorPos, org, EditSignal)
	ed.NeedsRender()
}

// cursorDeleteWord deletes word(s) immediately after the cursor
func (ed *Editor) cursorDeleteWord(steps int) {
	ed.validateCursor()
	if ed.HasSelection() {
		ed.DeleteSelection()
		return
	}
	// note: no update b/c signal from buf will drive update
	org := ed.CursorPos
	ed.cursorForwardWord(steps)
	ed.Buffer.DeleteText(org, ed.CursorPos, EditSignal)
	ed.SetCursorShow(org)
	ed.NeedsRender()
}

// cursorKill deletes text from cursor to end of text
func (ed *Editor) cursorKill() {
	ed.validateCursor()
	org := ed.CursorPos
	pos := ed.CursorPos

	atEnd := false
	if wln := ed.wrappedLines(pos.Ln); wln > 1 {
		si, ri, _ := ed.wrappedLineNumber(pos)
		llen := len(ed.renders[pos.Ln].Spans[si].Text)
		if si == wln-1 {
			llen--
		}
		atEnd = (ri == llen)
	} else {
		llen := ed.Buffer.lineLen(pos.Ln)
		atEnd = (ed.CursorPos.Ch == llen)
	}
	if atEnd {
		ed.cursorForward(1)
	} else {
		ed.cursorEndLine()
	}
	ed.Buffer.DeleteText(org, ed.CursorPos, EditSignal)
	ed.SetCursorShow(org)
	ed.NeedsRender()
}

// cursorTranspose swaps the character at the cursor with the one before it
func (ed *Editor) cursorTranspose() {
	ed.validateCursor()
	pos := ed.CursorPos
	if pos.Ch == 0 {
		return
	}
	ppos := pos
	ppos.Ch--
	ed.Buffer.LinesMu.Lock()
	lln := len(ed.Buffer.Lines[pos.Ln])
	end := false
	if pos.Ch >= lln {
		end = true
		pos.Ch = lln - 1
		ppos.Ch = lln - 2
	}
	chr := ed.Buffer.Lines[pos.Ln][pos.Ch]
	pchr := ed.Buffer.Lines[pos.Ln][ppos.Ch]
	ed.Buffer.LinesMu.Unlock()
	repl := string([]rune{chr, pchr})
	pos.Ch++
	ed.Buffer.ReplaceText(ppos, pos, ppos, repl, EditSignal, ReplaceMatchCase)
	if !end {
		ed.SetCursorShow(pos)
	}
	ed.NeedsRender()
}

// CursorTranspose swaps the word at the cursor with the one before it
func (ed *Editor) cursorTransposeWord() {
}

// JumpToLinePrompt jumps to given line number (minus 1) from prompt
func (ed *Editor) JumpToLinePrompt() {
	val := ""
	d := core.NewBody().AddTitle("Jump to line").AddText("Line number to jump to")
	tf := core.NewTextField(d).SetPlaceholder("Line number")
	tf.OnChange(func(e events.Event) {
		val = tf.Text()
	})
	d.AddBottomBar(func(parent core.Widget) {
		d.AddCancel(parent)
		d.AddOK(parent).SetText("Jump").OnClick(func(e events.Event) {
			val = tf.Text()
			ln, err := reflectx.ToInt(val)
			if err == nil {
				ed.jumpToLine(int(ln))
			}
		})
	})
	d.RunDialog(ed)
}

// jumpToLine jumps to given line number (minus 1)
func (ed *Editor) jumpToLine(ln int) {
	ed.SetCursorShow(lexer.Pos{Ln: ln - 1})
	ed.savePosHistory(ed.CursorPos)
	ed.NeedsLayout()
}

// findNextLink finds next link after given position, returns false if no such links
func (ed *Editor) findNextLink(pos lexer.Pos) (lexer.Pos, textbuf.Region, bool) {
	for ln := pos.Ln; ln < ed.NumLines; ln++ {
		if len(ed.renders[ln].Links) == 0 {
			pos.Ch = 0
			pos.Ln = ln + 1
			continue
		}
		rend := &ed.renders[ln]
		si, ri, _ := rend.RuneSpanPos(pos.Ch)
		for ti := range rend.Links {
			tl := &rend.Links[ti]
			if tl.StartSpan >= si && tl.StartIndex >= ri {
				st, _ := rend.SpanPosToRuneIndex(tl.StartSpan, tl.StartIndex)
				ed, _ := rend.SpanPosToRuneIndex(tl.EndSpan, tl.EndIndex)
				reg := textbuf.NewRegion(ln, st, ln, ed)
				pos.Ch = st + 1 // get into it so next one will go after..
				return pos, reg, true
			}
		}
		pos.Ln = ln + 1
		pos.Ch = 0
	}
	return pos, textbuf.RegionNil, false
}

// findPrevLink finds previous link before given position, returns false if no such links
func (ed *Editor) findPrevLink(pos lexer.Pos) (lexer.Pos, textbuf.Region, bool) {
	for ln := pos.Ln - 1; ln >= 0; ln-- {
		if len(ed.renders[ln].Links) == 0 {
			if ln-1 >= 0 {
				pos.Ch = ed.Buffer.lineLen(ln-1) - 2
			} else {
				ln = ed.NumLines
				pos.Ch = ed.Buffer.lineLen(ln - 2)
			}
			continue
		}
		rend := &ed.renders[ln]
		si, ri, _ := rend.RuneSpanPos(pos.Ch)
		nl := len(rend.Links)
		for ti := nl - 1; ti >= 0; ti-- {
			tl := &rend.Links[ti]
			if tl.StartSpan <= si && tl.StartIndex < ri {
				st, _ := rend.SpanPosToRuneIndex(tl.StartSpan, tl.StartIndex)
				ed, _ := rend.SpanPosToRuneIndex(tl.EndSpan, tl.EndIndex)
				reg := textbuf.NewRegion(ln, st, ln, ed)
				pos.Ln = ln
				pos.Ch = st + 1
				return pos, reg, true
			}
		}
	}
	return pos, textbuf.RegionNil, false
}

// CursorNextLink moves cursor to next link. wraparound wraps around to top of
// buffer if none found -- returns true if found
func (ed *Editor) CursorNextLink(wraparound bool) bool {
	if ed.NumLines == 0 {
		return false
	}
	ed.validateCursor()
	npos, reg, has := ed.findNextLink(ed.CursorPos)
	if !has {
		if !wraparound {
			return false
		}
		npos, reg, has = ed.findNextLink(lexer.Pos{}) // wraparound
		if !has {
			return false
		}
	}
	ed.HighlightRegion(reg)
	ed.SetCursorShow(npos)
	ed.savePosHistory(ed.CursorPos)
	ed.NeedsRender()
	return true
}

// CursorPrevLink moves cursor to previous link. wraparound wraps around to
// bottom of buffer if none found. returns true if found
func (ed *Editor) CursorPrevLink(wraparound bool) bool {
	if ed.NumLines == 0 {
		return false
	}
	ed.validateCursor()
	npos, reg, has := ed.findPrevLink(ed.CursorPos)
	if !has {
		if !wraparound {
			return false
		}
		npos, reg, has = ed.findPrevLink(lexer.Pos{}) // wraparound
		if !has {
			return false
		}
	}

	ed.HighlightRegion(reg)
	ed.SetCursorShow(npos)
	ed.savePosHistory(ed.CursorPos)
	ed.NeedsRender()
	return true
}

///////////////////////////////////////////////////////////////////////////////
//    Scrolling

// scrollInView tells any parent scroll layout to scroll to get given box
// (e.g., cursor BBox) in view -- returns true if scrolled
func (ed *Editor) scrollInView(bbox image.Rectangle) bool {
	return ed.ScrollToBox(bbox)
}

// scrollCursorToCenterIfHidden checks if the cursor is not visible, and if
// so, scrolls to the center, along both dimensions.
func (ed *Editor) scrollCursorToCenterIfHidden() bool {
	curBBox := ed.cursorBBox(ed.CursorPos)
	did := false
	lht := int(ed.lineHeight)
	bb := ed.renderBBox()
	if bb.Size().Y <= lht {
		return false
	}
	if (curBBox.Min.Y-lht) < bb.Min.Y || (curBBox.Max.Y+lht) > bb.Max.Y {
		did = ed.scrollCursorToVerticalCenter()
		// fmt.Println("v min:", curBBox.Min.Y, bb.Min.Y, "max:", curBBox.Max.Y+lht, bb.Max.Y, did)
	}
	if curBBox.Max.X < bb.Min.X+int(ed.LineNumberOffset) {
		did2 := ed.scrollCursorToRight()
		// fmt.Println("h max", curBBox.Max.X, bb.Min.X+int(ed.LineNumberOffset), did2)
		did = did || did2
	} else if curBBox.Min.X > bb.Max.X {
		did2 := ed.scrollCursorToRight()
		// fmt.Println("h min", curBBox.Min.X, bb.Max.X, did2)
		did = did || did2
	}
	if did {
		// fmt.Println("scroll to center", did)
	}
	return did
}

///////////////////////////////////////////////////////////////////////////////
//    Scrolling -- Vertical

// scrollToTop tells any parent scroll layout to scroll to get given vertical
// coordinate at top of view to extent possible -- returns true if scrolled
func (ed *Editor) scrollToTop(pos int) bool {
	ed.NeedsRender()
	return ed.ScrollDimToStart(math32.Y, pos)
}

// scrollCursorToTop tells any parent scroll layout to scroll to get cursor
// at top of view to extent possible -- returns true if scrolled.
func (ed *Editor) scrollCursorToTop() bool {
	curBBox := ed.cursorBBox(ed.CursorPos)
	return ed.scrollToTop(curBBox.Min.Y)
}

// scrollToBottom tells any parent scroll layout to scroll to get given
// vertical coordinate at bottom of view to extent possible -- returns true if
// scrolled
func (ed *Editor) scrollToBottom(pos int) bool {
	ed.NeedsRender()
	return ed.ScrollDimToEnd(math32.Y, pos)
}

// scrollCursorToBottom tells any parent scroll layout to scroll to get cursor
// at bottom of view to extent possible -- returns true if scrolled.
func (ed *Editor) scrollCursorToBottom() bool {
	curBBox := ed.cursorBBox(ed.CursorPos)
	return ed.scrollToBottom(curBBox.Max.Y)
}

// scrollToVerticalCenter tells any parent scroll layout to scroll to get given
// vertical coordinate to center of view to extent possible -- returns true if
// scrolled
func (ed *Editor) scrollToVerticalCenter(pos int) bool {
	ed.NeedsRender()
	return ed.ScrollDimToCenter(math32.Y, pos)
}

// scrollCursorToVerticalCenter tells any parent scroll layout to scroll to get
// cursor at vert center of view to extent possible -- returns true if
// scrolled.
func (ed *Editor) scrollCursorToVerticalCenter() bool {
	curBBox := ed.cursorBBox(ed.CursorPos)
	mid := (curBBox.Min.Y + curBBox.Max.Y) / 2
	return ed.scrollToVerticalCenter(mid)
}

func (ed *Editor) scrollCursorToTarget() {
	// fmt.Println(ed, "to target:", ed.CursorTarg)
	ed.CursorPos = ed.cursorTarget
	ed.scrollCursorToVerticalCenter()
	ed.targetSet = false
}

///////////////////////////////////////////////////////////////////////////////
//    Scrolling -- Horizontal

// scrollToRight tells any parent scroll layout to scroll to get given
// horizontal coordinate at right of view to extent possible -- returns true
// if scrolled
func (ed *Editor) scrollToRight(pos int) bool {
	return ed.ScrollDimToEnd(math32.X, pos)
}

// scrollCursorToRight tells any parent scroll layout to scroll to get cursor
// at right of view to extent possible -- returns true if scrolled.
func (ed *Editor) scrollCursorToRight() bool {
	curBBox := ed.cursorBBox(ed.CursorPos)
	return ed.scrollToRight(curBBox.Max.X)
}
