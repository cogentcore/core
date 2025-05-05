// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package textcore

import (
	"image"

	"cogentcore.org/core/events"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/styles/states"
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
	oldPos := ed.CursorPos
	ed.CursorPos = ed.Lines.ValidPos(pos)
	if ed.CursorPos == oldPos {
		return
	}
	ed.scopelightsReset()
	bm, has := ed.Lines.BraceMatch(pos)
	if has {
		ed.addScopelights(pos, bm)
		ed.NeedsRender()
	}
}

// SetCursorShow sets a new cursor position, enforcing it in range, and shows
// the cursor (scroll to if hidden, render)
func (ed *Base) SetCursorShow(pos textpos.Pos) {
	ed.setCursor(pos)
	ed.scrollCursorToCenterIfHidden()
	ed.renderCursor(true)
}

// SetCursorTarget sets a new cursor target position, ensures that it is visible.
// Setting the textpos.PosErr value causes it to go the end of doc, the position
// of which may not be known at the time the target is set.
func (ed *Base) SetCursorTarget(pos textpos.Pos) {
	ed.isScrolling = false
	ed.targetSet = true
	ed.cursorTarget = pos
	if pos == textpos.PosErr {
		ed.cursorEndDoc()
		return
	}
	ed.SetCursorShow(pos)
	// fmt.Println(ed, "set target:", ed.CursorTarg)
}

// savePosHistory saves the cursor position in history stack of cursor positions.
// Tracks across views. Returns false if position was on same line as last one saved.
func (ed *Base) savePosHistory(pos textpos.Pos) bool {
	if ed.Lines == nil {
		return false
	}
	did := ed.Lines.PosHistorySave(pos)
	if did {
		ed.posHistoryIndex = ed.Lines.PosHistoryLen() - 1
	}
	return did
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
	if ed.posHistoryIndex < 0 {
		ed.posHistoryIndex = 0
		return false
	}
	ed.posHistoryIndex = min(sz-1, ed.posHistoryIndex)
	pos, _ := ed.Lines.PosHistoryAt(ed.posHistoryIndex)
	ed.CursorPos = ed.Lines.ValidPos(pos)
	if ed.posHistoryIndex > 0 {
		ed.posHistoryIndex--
	}
	ed.scrollCursorToCenterIfHidden()
	ed.renderCursor(true)
	ed.SendInput()
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
	}
	pos, _ := ed.Lines.PosHistoryAt(ed.posHistoryIndex)
	ed.CursorPos = ed.Lines.ValidPos(pos)
	ed.scrollCursorToCenterIfHidden()
	ed.renderCursor(true)
	ed.SendInput()
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
	ed.SendInput()
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
	ed.SendInput()
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
	ed.SendInput()
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
	ed.cursorColumn = 0
	ed.scrollCursorToRight()
	ed.cursorSelectShow(org)
}

// CursorStartDoc moves the cursor to the start of the text, updating selection
// if select mode is active
func (ed *Base) CursorStartDoc() {
	org := ed.validateCursor()
	ed.CursorPos.Line = 0
	ed.CursorPos.Char = 0
	ed.cursorColumn = 0
	ed.scrollCursorToTop()
	ed.cursorSelectShow(org)
}

// cursorLineEnd moves the cursor to the end of the text
func (ed *Base) cursorLineEnd() {
	org := ed.validateCursor()
	ed.CursorPos = ed.Lines.MoveLineEnd(ed.viewId, org)
	ed.setCursorColumn(ed.CursorPos)
	ed.scrollCursorToRight()
	ed.cursorSelectShow(org)
}

// cursorEndDoc moves the cursor to the end of the text, updating selection if
// select mode is active
func (ed *Base) cursorEndDoc() {
	org := ed.validateCursor()
	ed.CursorPos = ed.Lines.EndPos()
	ed.setCursorColumn(ed.CursorPos)
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

// cursorKill deletes text from cursor to end of text.
// if line is empty, deletes the line.
func (ed *Base) cursorKill() {
	org := ed.validateCursor()
	llen := ed.Lines.LineLen(ed.CursorPos.Line)
	if ed.CursorPos.Char == llen { // at end
		ed.cursorForward(1)
	} else {
		ed.cursorLineEnd()
	}
	ed.Lines.DeleteText(org, ed.CursorPos)
	ed.SetCursorShow(org)
	ed.NeedsRender()
}

// cursorTranspose swaps the character at the cursor with the one before it.
func (ed *Base) cursorTranspose() {
	ed.validateCursor()
	pos := ed.CursorPos
	if pos.Char == 0 {
		return
	}
	ed.Lines.TransposeChar(ed.viewId, pos)
	// ed.SetCursorShow(pos)
	ed.NeedsRender()
}

// cursorTranspose swaps the character at the cursor with the one before it
func (ed *Base) cursorTransposeWord() {
	// todo:
}

// setCursorFromMouse sets cursor position from mouse mouse action -- handles
// the selection updating etc.
func (ed *Base) setCursorFromMouse(pt image.Point, newPos textpos.Pos, selMode events.SelectModes) {
	oldPos := ed.CursorPos
	if newPos == oldPos || newPos == textpos.PosErr {
		return
	}
	//	fmt.Printf("set cursor fm mouse: %v\n", newPos)
	defer ed.NeedsRender()

	if !ed.selectMode && selMode == events.ExtendContinuous {
		if ed.SelectRegion == (textpos.Region{}) {
			ed.selectStart = ed.CursorPos
		}
		ed.setCursor(newPos)
		ed.selectRegionUpdate(ed.CursorPos)
		ed.renderCursor(true)
		return
	}

	ed.setCursor(newPos)
	if ed.selectMode || selMode != events.SelectOne {
		if !ed.selectMode && selMode != events.SelectOne {
			ed.selectMode = true
			ed.selectStart = newPos
			ed.selectRegionUpdate(ed.CursorPos)
		}
		if !ed.StateIs(states.Sliding) && selMode == events.SelectOne {
			ln := ed.CursorPos.Line
			ch := ed.CursorPos.Char
			if ln != ed.SelectRegion.Start.Line || ch < ed.SelectRegion.Start.Char || ch > ed.SelectRegion.End.Char {
				ed.SelectReset()
			}
		} else {
			ed.selectRegionUpdate(ed.CursorPos)
		}
		if ed.StateIs(states.Sliding) {
			scPos := math32.FromPoint(pt) // already relative to editor
			ed.AutoScroll(scPos)
		} else {
			ed.scrollCursorToCenterIfHidden()
		}
	} else if ed.HasSelection() {
		ln := ed.CursorPos.Line
		ch := ed.CursorPos.Char
		if ln != ed.SelectRegion.Start.Line || ch < ed.SelectRegion.Start.Char || ch > ed.SelectRegion.End.Char {
			ed.SelectReset()
		}
	}
}
