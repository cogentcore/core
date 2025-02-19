// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package textcore

import (
	"cogentcore.org/core/text/textpos"
)

// validateCursor sets current cursor to a valid cursor position
func (ed *Base) validateCursor() textpos.Pos {
	if ed.Lines != nil {
		ed.CursorPos = ed.Lines.ValidPos(ed.CursorPos)
	} else {
		ed.CursorPos = textpos.Pos{}
	}
	return ed.CursorPos
}

// setCursor sets a new cursor position, enforcing it in range.
// This is the main final pathway for all cursor movement.
func (ed *Base) setCursor(pos textpos.Pos) {
	if ed.Lines == nil {
		ed.CursorPos = textpos.PosZero
		return
	}

	ed.clearScopelights()
	// ed.CursorPos = ed.Lines.ValidPos(pos) // todo
	ed.CursorPos = pos
	ed.SendInput()
	// todo:
	// txt := ed.Lines.Line(ed.CursorPos.Line)
	// ch := ed.CursorPos.Char
	// if ch < len(txt) {
	// 	r := txt[ch]
	// 	if r == '{' || r == '}' || r == '(' || r == ')' || r == '[' || r == ']' {
	// 		tp, found := ed.Lines.BraceMatch(txt[ch], ed.CursorPos)
	// 		if found {
	// 			ed.scopelights = append(ed.scopelights, lines.NewRegionPos(ed.CursorPos, textpos.Pos{ed.CursorPos.Line, ed.CursorPos.Char + 1}))
	// 			ed.scopelights = append(ed.scopelights, lines.NewRegionPos(tp, textpos.Pos{tp.Line, tp.Char + 1}))
	// 		}
	// 	}
	// }
	ed.NeedsRender()
}

// SetCursorShow sets a new cursor position, enforcing it in range, and shows
// the cursor (scroll to if hidden, render)
func (ed *Base) SetCursorShow(pos textpos.Pos) {
	ed.setCursor(pos)
	ed.scrollCursorToCenterIfHidden()
	ed.renderCursor(true)
}

// savePosHistory saves the cursor position in history stack of cursor positions.
// Tracks across views. Returns false if position was on same line as last one saved.
func (ed *Base) savePosHistory(pos textpos.Pos) bool {
	if ed.Lines == nil {
		return false
	}
	return ed.Lines.PosHistorySave(pos)
	ed.posHistoryIndex = ed.Lines.PosHistoryLen() - 1
	return true
}

// CursorToHistoryPrev moves cursor to previous position on history list.
// returns true if moved
func (ed *Base) CursorToHistoryPrev() bool {
	if ed.Lines == nil {
		ed.CursorPos = textpos.Pos{}
		return false
	}
	sz := ed.Lines.PosHistoryLen()
	if sz == 0 {
		return false
	}
	ed.posHistoryIndex--
	if ed.posHistoryIndex < 0 {
		ed.posHistoryIndex = 0
		return false
	}
	ed.posHistoryIndex = min(sz-1, ed.posHistoryIndex)
	pos, _ := ed.Lines.PosHistoryAt(ed.posHistoryIndex)
	ed.CursorPos = ed.Lines.ValidPos(pos)
	ed.SendInput()
	ed.scrollCursorToCenterIfHidden()
	ed.renderCursor(true)
	return true
}

// CursorToHistoryNext moves cursor to previous position on history list --
// returns true if moved
func (ed *Base) CursorToHistoryNext() bool {
	if ed.Lines == nil {
		ed.CursorPos = textpos.Pos{}
		return false
	}
	sz := ed.Lines.PosHistoryLen()
	if sz == 0 {
		return false
	}
	ed.posHistoryIndex++
	if ed.posHistoryIndex >= sz-1 {
		ed.posHistoryIndex = sz - 1
		return false
	}
	pos, _ := ed.Lines.PosHistoryAt(ed.posHistoryIndex)
	ed.CursorPos = ed.Lines.ValidPos(pos)
	ed.SendInput()
	ed.scrollCursorToCenterIfHidden()
	ed.renderCursor(true)
	return true
}

// setCursorColumn sets the current target cursor column (cursorColumn) to that
// of the given position
func (ed *Base) setCursorColumn(pos textpos.Pos) {
	if ed.Lines == nil {
		return
	}
	vpos := ed.Lines.PosToView(ed.viewId, pos)
	ed.cursorColumn = vpos.Char
}

////////  cursor moving

// cursorSelect updates selection based on cursor movements, given starting
// cursor position and ed.CursorPos is current
func (ed *Base) cursorSelect(org textpos.Pos) {
	if !ed.selectMode {
		return
	}
	ed.selectRegionUpdate(ed.CursorPos)
}

// cursorSelectShow does SetCursorShow, cursorSelect, and NeedsRender.
// This is typically called for move actions.
func (ed *Base) cursorSelectShow(org textpos.Pos) {
	ed.SetCursorShow(ed.CursorPos)
	ed.cursorSelect(org)
	ed.NeedsRender()
}

// cursorForward moves the cursor forward
func (ed *Base) cursorForward(steps int) {
	org := ed.validateCursor()
	ed.CursorPos = ed.Lines.MoveForward(org, steps)
	ed.setCursorColumn(ed.CursorPos)
	ed.cursorSelectShow(org)
}

// cursorForwardWord moves the cursor forward by words
func (ed *Base) cursorForwardWord(steps int) {
	org := ed.validateCursor()
	ed.CursorPos = ed.Lines.MoveForwardWord(org, steps)
	ed.setCursorColumn(ed.CursorPos)
	ed.cursorSelectShow(org)
}

// cursorBackward moves the cursor backward
func (ed *Base) cursorBackward(steps int) {
	org := ed.validateCursor()
	ed.CursorPos = ed.Lines.MoveBackward(org, steps)
	ed.setCursorColumn(ed.CursorPos)
	ed.cursorSelectShow(org)
}

// cursorBackwardWord moves the cursor backward by words
func (ed *Base) cursorBackwardWord(steps int) {
	org := ed.validateCursor()
	ed.CursorPos = ed.Lines.MoveBackwardWord(org, steps)
	ed.setCursorColumn(ed.CursorPos)
	ed.cursorSelectShow(org)
}

// cursorDown moves the cursor down line(s)
func (ed *Base) cursorDown(steps int) {
	org := ed.validateCursor()
	ed.CursorPos = ed.Lines.MoveDown(ed.viewId, org, steps, ed.cursorColumn)
	ed.cursorSelectShow(org)
}

// cursorPageDown moves the cursor down page(s), where a page is defined
// dynamically as just moving the cursor off the screen
func (ed *Base) cursorPageDown(steps int) {
	org := ed.validateCursor()
	vp := ed.Lines.PosToView(ed.viewId, ed.CursorPos)
	cpr := max(0, vp.Line-int(ed.scrollPos))
	nln := max(1, ed.visSize.Y-cpr)
	for range steps {
		ed.CursorPos = ed.Lines.MoveDown(ed.viewId, ed.CursorPos, nln, ed.cursorColumn)
	}
	ed.setCursor(ed.CursorPos)
	ed.scrollCursorToTop()
	ed.renderCursor(true)
	ed.cursorSelect(org)
	ed.NeedsRender()
}

// cursorUp moves the cursor up line(s)
func (ed *Base) cursorUp(steps int) {
	org := ed.validateCursor()
	ed.CursorPos = ed.Lines.MoveUp(ed.viewId, org, steps, ed.cursorColumn)
	ed.cursorSelectShow(org)
}

// cursorPageUp moves the cursor up page(s), where a page is defined
// dynamically as just moving the cursor off the screen
func (ed *Base) cursorPageUp(steps int) {
	org := ed.validateCursor()
	vp := ed.Lines.PosToView(ed.viewId, ed.CursorPos)
	cpr := max(0, vp.Line-int(ed.scrollPos))
	nln := max(1, cpr)
	for range steps {
		ed.CursorPos = ed.Lines.MoveUp(ed.viewId, ed.CursorPos, nln, ed.cursorColumn)
	}
	ed.setCursor(ed.CursorPos)
	ed.scrollCursorToBottom()
	ed.renderCursor(true)
	ed.cursorSelect(org)
	ed.NeedsRender()
}

// cursorRecenter re-centers the view around the cursor position, toggling
// between putting cursor in middle, top, and bottom of view
func (ed *Base) cursorRecenter() {
	ed.validateCursor()
	ed.savePosHistory(ed.CursorPos)
	cur := (ed.lastRecenter + 1) % 3
	switch cur {
	case 0:
		ed.scrollCursorToBottom()
	case 1:
		ed.scrollCursorToCenter()
	case 2:
		ed.scrollCursorToTop()
	}
	ed.lastRecenter = cur
}

// cursorLineStart moves the cursor to the start of the line, updating selection
// if select mode is active
func (ed *Base) cursorLineStart() {
	org := ed.validateCursor()
	ed.CursorPos = ed.Lines.MoveLineStart(ed.viewId, org)
	ed.scrollCursorToRight()
	ed.cursorSelectShow(org)
}

// CursorStartDoc moves the cursor to the start of the text, updating selection
// if select mode is active
func (ed *Base) CursorStartDoc() {
	org := ed.validateCursor()
	ed.CursorPos.Line = 0
	ed.CursorPos.Char = 0
	ed.cursorColumn = ed.CursorPos.Char
	ed.scrollCursorToTop()
	ed.cursorSelectShow(org)
}

// cursorLineEnd moves the cursor to the end of the text
func (ed *Base) cursorLineEnd() {
	org := ed.validateCursor()
	ed.CursorPos = ed.Lines.MoveLineEnd(ed.viewId, org)
	ed.cursorColumn = ed.CursorPos.Char
	ed.scrollCursorToRight()
	ed.cursorSelectShow(org)
}

// cursorEndDoc moves the cursor to the end of the text, updating selection if
// select mode is active
func (ed *Base) cursorEndDoc() {
	org := ed.validateCursor()
	ed.CursorPos.Line = max(ed.NumLines()-1, 0)
	ed.CursorPos.Char = ed.Lines.LineLen(ed.CursorPos.Line)
	ed.cursorColumn = ed.CursorPos.Char
	ed.scrollCursorToBottom()
	ed.cursorSelectShow(org)
}

// todo: ctrl+backspace = delete word
// shift+arrow = select
// uparrow = start / down = end

// cursorBackspace deletes character(s) immediately before cursor
func (ed *Base) cursorBackspace(steps int) {
	org := ed.validateCursor()
	if ed.HasSelection() {
		org = ed.SelectRegion.Start
		ed.deleteSelection()
		ed.SetCursorShow(org)
		return
	}
	// note: no update b/c signal from buf will drive update
	ed.cursorBackward(steps)
	ed.scrollCursorToCenterIfHidden()
	ed.renderCursor(true)
	ed.Lines.DeleteText(ed.CursorPos, org)
	ed.NeedsRender()
}

// cursorDelete deletes character(s) immediately after the cursor
func (ed *Base) cursorDelete(steps int) {
	org := ed.validateCursor()
	if ed.HasSelection() {
		ed.deleteSelection()
		return
	}
	// note: no update b/c signal from buf will drive update
	ed.cursorForward(steps)
	ed.Lines.DeleteText(org, ed.CursorPos)
	ed.SetCursorShow(org)
	ed.NeedsRender()
}

// cursorBackspaceWord deletes words(s) immediately before cursor
func (ed *Base) cursorBackspaceWord(steps int) {
	org := ed.validateCursor()
	if ed.HasSelection() {
		ed.deleteSelection()
		ed.SetCursorShow(org)
		return
	}
	ed.cursorBackwardWord(steps)
	ed.scrollCursorToCenterIfHidden()
	ed.renderCursor(true)
	ed.Lines.DeleteText(ed.CursorPos, org)
	ed.NeedsRender()
}

// cursorDeleteWord deletes word(s) immediately after the cursor
func (ed *Base) cursorDeleteWord(steps int) {
	org := ed.validateCursor()
	if ed.HasSelection() {
		ed.deleteSelection()
		return
	}
	ed.cursorForwardWord(steps)
	ed.Lines.DeleteText(org, ed.CursorPos)
	ed.SetCursorShow(org)
	ed.NeedsRender()
}

// cursorKill deletes text from cursor to end of text
func (ed *Base) cursorKill() {
	org := ed.validateCursor()
	ed.cursorLineEnd()
	ed.Lines.DeleteText(org, ed.CursorPos)
	ed.SetCursorShow(org)
	ed.NeedsRender()
}

// cursorTranspose swaps the character at the cursor with the one before it
func (ed *Base) cursorTranspose() {
	ed.validateCursor()
	pos := ed.CursorPos
	if pos.Char == 0 {
		return
	}
	// todo:
	// ppos := pos
	// ppos.Ch--
	// lln := ed.Lines.LineLen(pos.Line)
	// end := false
	// if pos.Char >= lln {
	// 	end = true
	// 	pos.Char = lln - 1
	// 	ppos.Char = lln - 2
	// }
	// chr := ed.Lines.LineChar(pos.Line, pos.Ch)
	// pchr := ed.Lines.LineChar(pos.Line, ppos.Ch)
	// repl := string([]rune{chr, pchr})
	// pos.Ch++
	// ed.Lines.ReplaceText(ppos, pos, ppos, repl, EditSignal, ReplaceMatchCase)
	// if !end {
	// 	ed.SetCursorShow(pos)
	// }
	ed.NeedsRender()
}

// cursorTranspose swaps the character at the cursor with the one before it
func (ed *Base) cursorTransposeWord() {
	// todo:
}
