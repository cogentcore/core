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

// CursorMovedSig sends the signal that cursor has moved
func (ed *Editor) CursorMovedSig() {
	ed.Send(events.Input, nil)
}

// ValidateCursor sets current cursor to a valid cursor position
func (ed *Editor) ValidateCursor() {
	if ed.Buffer != nil {
		ed.CursorPos = ed.Buffer.ValidPos(ed.CursorPos)
	} else {
		ed.CursorPos = lexer.PosZero
	}
}

// WrappedLines returns the number of wrapped lines (spans) for given line number
func (ed *Editor) WrappedLines(ln int) int {
	if ln >= len(ed.Renders) {
		return 0
	}
	return len(ed.Renders[ln].Spans)
}

// WrappedLineNumber returns the wrapped line number (span index) and rune index
// within that span of the given character position within line in position,
// and false if out of range (last valid position returned in that case -- still usable).
func (ed *Editor) WrappedLineNumber(pos lexer.Pos) (si, ri int, ok bool) {
	if pos.Ln >= len(ed.Renders) {
		return 0, 0, false
	}
	return ed.Renders[pos.Ln].RuneSpanPos(pos.Ch)
}

// SetCursor sets a new cursor position, enforcing it in range.
// This is the main final pathway for all cursor movement.
func (ed *Editor) SetCursor(pos lexer.Pos) {
	if ed.NLines == 0 || ed.Buffer == nil {
		ed.CursorPos = lexer.PosZero
		return
	}

	ed.ClearScopelights()
	ed.CursorPos = ed.Buffer.ValidPos(pos)
	ed.CursorMovedSig()
	txt := ed.Buffer.Line(ed.CursorPos.Ln)
	ch := ed.CursorPos.Ch
	if ch < len(txt) {
		r := txt[ch]
		if r == '{' || r == '}' || r == '(' || r == ')' || r == '[' || r == ']' {
			tp, found := ed.Buffer.BraceMatch(txt[ch], ed.CursorPos)
			if found {
				ed.Scopelights = append(ed.Scopelights, textbuf.NewRegionPos(ed.CursorPos, lexer.Pos{ed.CursorPos.Ln, ed.CursorPos.Ch + 1}))
				ed.Scopelights = append(ed.Scopelights, textbuf.NewRegionPos(tp, lexer.Pos{tp.Ln, tp.Ch + 1}))
			}
		}
	}
	ed.NeedsRender()
}

// SetCursorShow sets a new cursor position, enforcing it in range, and shows
// the cursor (scroll to if hidden, render)
func (ed *Editor) SetCursorShow(pos lexer.Pos) {
	ed.SetCursor(pos)
	ed.ScrollCursorToCenterIfHidden()
	ed.RenderCursor(true)
}

// SetCursorTarget sets a new cursor target position, ensures that it is visible
func (ed *Editor) SetCursorTarget(pos lexer.Pos) {
	ed.SetFlag(true, EditorTargetSet)
	ed.CursorTarg = pos
	ed.SetCursorShow(pos)
	ed.NeedsRender()
	// fmt.Println(ed, "set target:", ed.CursorTarg)
}

// SetCursorCol sets the current target cursor column (CursorCol) to that
// of the given position
func (ed *Editor) SetCursorCol(pos lexer.Pos) {
	if wln := ed.WrappedLines(pos.Ln); wln > 1 {
		si, ri, ok := ed.WrappedLineNumber(pos)
		if ok && si > 0 {
			ed.CursorCol = ri
		} else {
			ed.CursorCol = pos.Ch
		}
	} else {
		ed.CursorCol = pos.Ch
	}
}

// SavePosHistory saves the cursor position in history stack of cursor positions
func (ed *Editor) SavePosHistory(pos lexer.Pos) {
	if ed.Buffer == nil {
		return
	}
	ed.Buffer.SavePosHistory(pos)
	ed.PosHistIndex = len(ed.Buffer.PosHistory) - 1
}

// CursorToHistPrev moves cursor to previous position on history list --
// returns true if moved
func (ed *Editor) CursorToHistPrev() bool {
	if ed.NLines == 0 || ed.Buffer == nil {
		ed.CursorPos = lexer.PosZero
		return false
	}
	sz := len(ed.Buffer.PosHistory)
	if sz == 0 {
		return false
	}
	ed.PosHistIndex--
	if ed.PosHistIndex < 0 {
		ed.PosHistIndex = 0
		return false
	}
	ed.PosHistIndex = min(sz-1, ed.PosHistIndex)
	pos := ed.Buffer.PosHistory[ed.PosHistIndex]
	ed.CursorPos = ed.Buffer.ValidPos(pos)
	ed.CursorMovedSig()
	ed.ScrollCursorToCenterIfHidden()
	ed.RenderCursor(true)
	return true
}

// CursorToHistNext moves cursor to previous position on history list --
// returns true if moved
func (ed *Editor) CursorToHistNext() bool {
	if ed.NLines == 0 || ed.Buffer == nil {
		ed.CursorPos = lexer.PosZero
		return false
	}
	sz := len(ed.Buffer.PosHistory)
	if sz == 0 {
		return false
	}
	ed.PosHistIndex++
	if ed.PosHistIndex >= sz-1 {
		ed.PosHistIndex = sz - 1
		return false
	}
	pos := ed.Buffer.PosHistory[ed.PosHistIndex]
	ed.CursorPos = ed.Buffer.ValidPos(pos)
	ed.CursorMovedSig()
	ed.ScrollCursorToCenterIfHidden()
	ed.RenderCursor(true)
	return true
}

// SelectRegUpdate updates current select region based on given cursor position
// relative to SelectStart position
func (ed *Editor) SelectRegUpdate(pos lexer.Pos) {
	if pos.IsLess(ed.SelectStart) {
		ed.SelectRegion.Start = pos
		ed.SelectRegion.End = ed.SelectStart
	} else {
		ed.SelectRegion.Start = ed.SelectStart
		ed.SelectRegion.End = pos
	}
}

// CursorSelect updates selection based on cursor movements, given starting
// cursor position and ed.CursorPos is current
func (ed *Editor) CursorSelect(org lexer.Pos) {
	if !ed.SelectMode {
		return
	}
	ed.SelectRegUpdate(ed.CursorPos)
}

// CursorForward moves the cursor forward
func (ed *Editor) CursorForward(steps int) {
	ed.ValidateCursor()
	org := ed.CursorPos
	for i := 0; i < steps; i++ {
		ed.CursorPos.Ch++
		if ed.CursorPos.Ch > ed.Buffer.LineLen(ed.CursorPos.Ln) {
			if ed.CursorPos.Ln < ed.NLines-1 {
				ed.CursorPos.Ch = 0
				ed.CursorPos.Ln++
			} else {
				ed.CursorPos.Ch = ed.Buffer.LineLen(ed.CursorPos.Ln)
			}
		}
	}
	ed.SetCursorCol(ed.CursorPos)
	ed.SetCursorShow(ed.CursorPos)
	ed.CursorSelect(org)
	ed.NeedsRender()
}

// CursorForwardWord moves the cursor forward by words
func (ed *Editor) CursorForwardWord(steps int) {
	ed.ValidateCursor()
	org := ed.CursorPos
	for i := 0; i < steps; i++ {
		txt := ed.Buffer.Line(ed.CursorPos.Ln)
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
			if ed.CursorPos.Ln < ed.NLines-1 {
				ed.CursorPos.Ch = 0
				ed.CursorPos.Ln++
			} else {
				ed.CursorPos.Ch = ed.Buffer.LineLen(ed.CursorPos.Ln)
			}
		}
	}
	ed.SetCursorCol(ed.CursorPos)
	ed.SetCursorShow(ed.CursorPos)
	ed.CursorSelect(org)
	ed.NeedsRender()
}

// CursorDown moves the cursor down line(s)
func (ed *Editor) CursorDown(steps int) {
	ed.ValidateCursor()
	org := ed.CursorPos
	pos := ed.CursorPos
	for i := 0; i < steps; i++ {
		gotwrap := false
		if wln := ed.WrappedLines(pos.Ln); wln > 1 {
			si, ri, _ := ed.WrappedLineNumber(pos)
			if si < wln-1 {
				si++
				mxlen := min(len(ed.Renders[pos.Ln].Spans[si].Text), ed.CursorCol)
				if ed.CursorCol < mxlen {
					ri = ed.CursorCol
				} else {
					ri = mxlen
				}
				nwc, _ := ed.Renders[pos.Ln].SpanPosToRuneIndex(si, ri)
				pos.Ch = nwc
				gotwrap = true

			}
		}
		if !gotwrap {
			pos.Ln++
			if pos.Ln >= ed.NLines {
				pos.Ln = ed.NLines - 1
				break
			}
			mxlen := min(ed.Buffer.LineLen(pos.Ln), ed.CursorCol)
			if ed.CursorCol < mxlen {
				pos.Ch = ed.CursorCol
			} else {
				pos.Ch = mxlen
			}
		}
	}
	ed.SetCursorShow(pos)
	ed.CursorSelect(org)
	ed.NeedsRender()
}

// CursorPageDown moves the cursor down page(s), where a page is defined abcdef
// dynamically as just moving the cursor off the screen
func (ed *Editor) CursorPageDown(steps int) {
	ed.ValidateCursor()
	org := ed.CursorPos
	for i := 0; i < steps; i++ {
		lvln := ed.LastVisibleLine(ed.CursorPos.Ln)
		ed.CursorPos.Ln = lvln
		if ed.CursorPos.Ln >= ed.NLines {
			ed.CursorPos.Ln = ed.NLines - 1
		}
		ed.CursorPos.Ch = min(ed.Buffer.LineLen(ed.CursorPos.Ln), ed.CursorCol)
		ed.ScrollCursorToTop()
		ed.RenderCursor(true)
	}
	ed.SetCursor(ed.CursorPos)
	ed.CursorSelect(org)
	ed.NeedsRender()
}

// CursorBackward moves the cursor backward
func (ed *Editor) CursorBackward(steps int) {
	ed.ValidateCursor()
	org := ed.CursorPos
	for i := 0; i < steps; i++ {
		ed.CursorPos.Ch--
		if ed.CursorPos.Ch < 0 {
			if ed.CursorPos.Ln > 0 {
				ed.CursorPos.Ln--
				ed.CursorPos.Ch = ed.Buffer.LineLen(ed.CursorPos.Ln)
			} else {
				ed.CursorPos.Ch = 0
			}
		}
	}
	ed.SetCursorCol(ed.CursorPos)
	ed.SetCursorShow(ed.CursorPos)
	ed.CursorSelect(org)
	ed.NeedsRender()
}

// CursorBackwardWord moves the cursor backward by words
func (ed *Editor) CursorBackwardWord(steps int) {
	ed.ValidateCursor()
	org := ed.CursorPos
	for i := 0; i < steps; i++ {
		txt := ed.Buffer.Line(ed.CursorPos.Ln)
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
				ed.CursorPos.Ch = ed.Buffer.LineLen(ed.CursorPos.Ln)
			} else {
				ed.CursorPos.Ch = 0
			}
		}
	}
	ed.SetCursorCol(ed.CursorPos)
	ed.SetCursorShow(ed.CursorPos)
	ed.CursorSelect(org)
	ed.NeedsRender()
}

// CursorUp moves the cursor up line(s)
func (ed *Editor) CursorUp(steps int) {
	ed.ValidateCursor()
	org := ed.CursorPos
	pos := ed.CursorPos
	for i := 0; i < steps; i++ {
		gotwrap := false
		if wln := ed.WrappedLines(pos.Ln); wln > 1 {
			si, ri, _ := ed.WrappedLineNumber(pos)
			if si > 0 {
				ri = ed.CursorCol
				nwc, _ := ed.Renders[pos.Ln].SpanPosToRuneIndex(si-1, ri)
				if nwc == pos.Ch {
					ed.CursorCol = 0
					ri = 0
					nwc, _ = ed.Renders[pos.Ln].SpanPosToRuneIndex(si-1, ri)
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
			if wln := ed.WrappedLines(pos.Ln); wln > 1 { // just entered end of wrapped line
				si := wln - 1
				ri := ed.CursorCol
				nwc, _ := ed.Renders[pos.Ln].SpanPosToRuneIndex(si, ri)
				pos.Ch = nwc
			} else {
				mxlen := min(ed.Buffer.LineLen(pos.Ln), ed.CursorCol)
				if ed.CursorCol < mxlen {
					pos.Ch = ed.CursorCol
				} else {
					pos.Ch = mxlen
				}
			}
		}
	}
	ed.SetCursorShow(pos)
	ed.CursorSelect(org)
	ed.NeedsRender()
}

// CursorPageUp moves the cursor up page(s), where a page is defined
// dynamically as just moving the cursor off the screen
func (ed *Editor) CursorPageUp(steps int) {
	ed.ValidateCursor()
	org := ed.CursorPos
	for i := 0; i < steps; i++ {
		lvln := ed.FirstVisibleLine(ed.CursorPos.Ln)
		ed.CursorPos.Ln = lvln
		if ed.CursorPos.Ln <= 0 {
			ed.CursorPos.Ln = 0
		}
		ed.CursorPos.Ch = min(ed.Buffer.LineLen(ed.CursorPos.Ln), ed.CursorCol)
		ed.ScrollCursorToBottom()
		ed.RenderCursor(true)
	}
	ed.SetCursor(ed.CursorPos)
	ed.CursorSelect(org)
	ed.NeedsRender()
}

// CursorRecenter re-centers the view around the cursor position, toggling
// between putting cursor in middle, top, and bottom of view
func (ed *Editor) CursorRecenter() {
	ed.ValidateCursor()
	ed.SavePosHistory(ed.CursorPos)
	cur := (ed.lastRecenter + 1) % 3
	switch cur {
	case 0:
		ed.ScrollCursorToBottom()
	case 1:
		ed.ScrollCursorToVertCenter()
	case 2:
		ed.ScrollCursorToTop()
	}
	ed.lastRecenter = cur
}

// CursorStartLine moves the cursor to the start of the line, updating selection
// if select mode is active
func (ed *Editor) CursorStartLine() {
	ed.ValidateCursor()
	org := ed.CursorPos
	pos := ed.CursorPos

	gotwrap := false
	if wln := ed.WrappedLines(pos.Ln); wln > 1 {
		si, ri, _ := ed.WrappedLineNumber(pos)
		if si > 0 {
			ri = 0
			nwc, _ := ed.Renders[pos.Ln].SpanPosToRuneIndex(si, ri)
			pos.Ch = nwc
			ed.CursorPos = pos
			ed.CursorCol = ri
			gotwrap = true
		}
	}
	if !gotwrap {
		ed.CursorPos.Ch = 0
		ed.CursorCol = ed.CursorPos.Ch
	}
	// fmt.Printf("sol cursorcol: %v\n", ed.CursorCol)
	ed.SetCursor(ed.CursorPos)
	ed.ScrollCursorToRight()
	ed.RenderCursor(true)
	ed.CursorSelect(org)
	ed.NeedsRender()
}

// CursorStartDoc moves the cursor to the start of the text, updating selection
// if select mode is active
func (ed *Editor) CursorStartDoc() {
	ed.ValidateCursor()
	org := ed.CursorPos
	ed.CursorPos.Ln = 0
	ed.CursorPos.Ch = 0
	ed.CursorCol = ed.CursorPos.Ch
	ed.SetCursor(ed.CursorPos)
	ed.ScrollCursorToTop()
	ed.RenderCursor(true)
	ed.CursorSelect(org)
	ed.NeedsRender()
}

// CursorEndLine moves the cursor to the end of the text
func (ed *Editor) CursorEndLine() {
	ed.ValidateCursor()
	org := ed.CursorPos
	pos := ed.CursorPos

	gotwrap := false
	if wln := ed.WrappedLines(pos.Ln); wln > 1 {
		si, ri, _ := ed.WrappedLineNumber(pos)
		ri = len(ed.Renders[pos.Ln].Spans[si].Text) - 1
		nwc, _ := ed.Renders[pos.Ln].SpanPosToRuneIndex(si, ri)
		if si == len(ed.Renders[pos.Ln].Spans)-1 { // last span
			ri++
			nwc++
		}
		ed.CursorCol = ri
		pos.Ch = nwc
		ed.CursorPos = pos
		gotwrap = true
	}
	if !gotwrap {
		ed.CursorPos.Ch = ed.Buffer.LineLen(ed.CursorPos.Ln)
		ed.CursorCol = ed.CursorPos.Ch
	}
	ed.SetCursor(ed.CursorPos)
	ed.ScrollCursorToRight()
	ed.RenderCursor(true)
	ed.CursorSelect(org)
	ed.NeedsRender()
}

// CursorEndDoc moves the cursor to the end of the text, updating selection if
// select mode is active
func (ed *Editor) CursorEndDoc() {
	ed.ValidateCursor()
	org := ed.CursorPos
	ed.CursorPos.Ln = max(ed.NLines-1, 0)
	ed.CursorPos.Ch = ed.Buffer.LineLen(ed.CursorPos.Ln)
	ed.CursorCol = ed.CursorPos.Ch
	ed.SetCursor(ed.CursorPos)
	ed.ScrollCursorToBottom()
	ed.RenderCursor(true)
	ed.CursorSelect(org)
	ed.NeedsRender()
}

// todo: ctrl+backspace = delete word
// shift+arrow = select
// uparrow = start / down = end

// CursorBackspace deletes character(s) immediately before cursor
func (ed *Editor) CursorBackspace(steps int) {
	ed.ValidateCursor()
	org := ed.CursorPos
	if ed.HasSelection() {
		org = ed.SelectRegion.Start
		ed.DeleteSelection()
		ed.SetCursorShow(org)
		return
	}
	// note: no update b/c signal from buf will drive update
	ed.CursorBackward(steps)
	ed.ScrollCursorToCenterIfHidden()
	ed.RenderCursor(true)
	ed.Buffer.DeleteText(ed.CursorPos, org, EditSignal)
	ed.NeedsRender()
}

// CursorDelete deletes character(s) immediately after the cursor
func (ed *Editor) CursorDelete(steps int) {
	ed.ValidateCursor()
	if ed.HasSelection() {
		ed.DeleteSelection()
		return
	}
	// note: no update b/c signal from buf will drive update
	org := ed.CursorPos
	ed.CursorForward(steps)
	ed.Buffer.DeleteText(org, ed.CursorPos, EditSignal)
	ed.SetCursorShow(org)
	ed.NeedsRender()
}

// CursorBackspaceWord deletes words(s) immediately before cursor
func (ed *Editor) CursorBackspaceWord(steps int) {
	ed.ValidateCursor()
	org := ed.CursorPos
	if ed.HasSelection() {
		ed.DeleteSelection()
		ed.SetCursorShow(org)
		return
	}
	// note: no update b/c signal from buf will drive update
	ed.CursorBackwardWord(steps)
	ed.ScrollCursorToCenterIfHidden()
	ed.RenderCursor(true)
	ed.Buffer.DeleteText(ed.CursorPos, org, EditSignal)
	ed.NeedsRender()
}

// CursorDeleteWord deletes word(s) immediately after the cursor
func (ed *Editor) CursorDeleteWord(steps int) {
	ed.ValidateCursor()
	if ed.HasSelection() {
		ed.DeleteSelection()
		return
	}
	// note: no update b/c signal from buf will drive update
	org := ed.CursorPos
	ed.CursorForwardWord(steps)
	ed.Buffer.DeleteText(org, ed.CursorPos, EditSignal)
	ed.SetCursorShow(org)
	ed.NeedsRender()
}

// CursorKill deletes text from cursor to end of text
func (ed *Editor) CursorKill() {
	ed.ValidateCursor()
	org := ed.CursorPos
	pos := ed.CursorPos

	atEnd := false
	if wln := ed.WrappedLines(pos.Ln); wln > 1 {
		si, ri, _ := ed.WrappedLineNumber(pos)
		llen := len(ed.Renders[pos.Ln].Spans[si].Text)
		if si == wln-1 {
			llen--
		}
		atEnd = (ri == llen)
	} else {
		llen := ed.Buffer.LineLen(pos.Ln)
		atEnd = (ed.CursorPos.Ch == llen)
	}
	if atEnd {
		ed.CursorForward(1)
	} else {
		ed.CursorEndLine()
	}
	ed.Buffer.DeleteText(org, ed.CursorPos, EditSignal)
	ed.SetCursorShow(org)
	ed.NeedsRender()
}

// CursorTranspose swaps the character at the cursor with the one before it
func (ed *Editor) CursorTranspose() {
	ed.ValidateCursor()
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
func (ed *Editor) CursorTransposeWord() {
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
				ed.JumpToLine(int(ln))
			}
		})
	})
	d.RunDialog(ed)
}

// JumpToLine jumps to given line number (minus 1)
func (ed *Editor) JumpToLine(ln int) {
	ed.SetCursorShow(lexer.Pos{Ln: ln - 1})
	ed.SavePosHistory(ed.CursorPos)
	ed.NeedsLayout()
}

// FindNextLink finds next link after given position, returns false if no such links
func (ed *Editor) FindNextLink(pos lexer.Pos) (lexer.Pos, textbuf.Region, bool) {
	for ln := pos.Ln; ln < ed.NLines; ln++ {
		if len(ed.Renders[ln].Links) == 0 {
			pos.Ch = 0
			pos.Ln = ln + 1
			continue
		}
		rend := &ed.Renders[ln]
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

// FindPrevLink finds previous link before given position, returns false if no such links
func (ed *Editor) FindPrevLink(pos lexer.Pos) (lexer.Pos, textbuf.Region, bool) {
	for ln := pos.Ln - 1; ln >= 0; ln-- {
		if len(ed.Renders[ln].Links) == 0 {
			if ln-1 >= 0 {
				pos.Ch = ed.Buffer.LineLen(ln-1) - 2
			} else {
				ln = ed.NLines
				pos.Ch = ed.Buffer.LineLen(ln - 2)
			}
			continue
		}
		rend := &ed.Renders[ln]
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
	if ed.NLines == 0 {
		return false
	}
	ed.ValidateCursor()
	npos, reg, has := ed.FindNextLink(ed.CursorPos)
	if !has {
		if !wraparound {
			return false
		}
		npos, reg, has = ed.FindNextLink(lexer.Pos{}) // wraparound
		if !has {
			return false
		}
	}
	ed.HighlightRegion(reg)
	ed.SetCursorShow(npos)
	ed.SavePosHistory(ed.CursorPos)
	ed.NeedsRender()
	return true
}

// CursorPrevLink moves cursor to previous link. wraparound wraps around to
// bottom of buffer if none found. returns true if found
func (ed *Editor) CursorPrevLink(wraparound bool) bool {
	if ed.NLines == 0 {
		return false
	}
	ed.ValidateCursor()
	npos, reg, has := ed.FindPrevLink(ed.CursorPos)
	if !has {
		if !wraparound {
			return false
		}
		npos, reg, has = ed.FindPrevLink(lexer.Pos{}) // wraparound
		if !has {
			return false
		}
	}

	ed.HighlightRegion(reg)
	ed.SetCursorShow(npos)
	ed.SavePosHistory(ed.CursorPos)
	ed.NeedsRender()
	return true
}

///////////////////////////////////////////////////////////////////////////////
//    Scrolling

// ScrollInView tells any parent scroll layout to scroll to get given box
// (e.g., cursor BBox) in view -- returns true if scrolled
func (ed *Editor) ScrollInView(bbox image.Rectangle) bool {
	return ed.ScrollToBox(bbox)
}

// ScrollCursorInView tells any parent scroll layout to scroll to get cursor
// in view -- returns true if scrolled
func (ed *Editor) ScrollCursorInView() bool {
	if ed == nil || ed.This() == nil {
		return false
	}
	if ed.This().(core.Widget).IsVisible() {
		curBBox := ed.CursorBBox(ed.CursorPos)
		return ed.ScrollInView(curBBox)
	}
	return false
}

// ScrollCursorToCenterIfHidden checks if the cursor is not visible, and if
// so, scrolls to the center, along both dimensions.
func (ed *Editor) ScrollCursorToCenterIfHidden() bool {
	curBBox := ed.CursorBBox(ed.CursorPos)
	did := false
	lht := int(ed.LineHeight)
	bb := ed.RenderBBox()
	if bb.Size().Y <= lht {
		return false
	}
	if (curBBox.Min.Y-lht) < bb.Min.Y || (curBBox.Max.Y+lht) > bb.Max.Y {
		did = ed.ScrollCursorToVertCenter()
		// fmt.Println("v min:", curBBox.Min.Y, bb.Min.Y, "max:", curBBox.Max.Y+lht, bb.Max.Y, did)
	}
	if curBBox.Max.X < bb.Min.X+int(ed.LineNumberOffset) {
		did2 := ed.ScrollCursorToRight()
		// fmt.Println("h max", curBBox.Max.X, bb.Min.X+int(ed.LineNumberOffset), did2)
		did = did || did2
	} else if curBBox.Min.X > bb.Max.X {
		did2 := ed.ScrollCursorToRight()
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

// ScrollToTop tells any parent scroll layout to scroll to get given vertical
// coordinate at top of view to extent possible -- returns true if scrolled
func (ed *Editor) ScrollToTop(pos int) bool {
	ed.NeedsRender()
	return ed.ScrollDimToStart(math32.Y, pos)
}

// ScrollCursorToTop tells any parent scroll layout to scroll to get cursor
// at top of view to extent possible -- returns true if scrolled.
func (ed *Editor) ScrollCursorToTop() bool {
	curBBox := ed.CursorBBox(ed.CursorPos)
	return ed.ScrollToTop(curBBox.Min.Y)
}

// ScrollToBottom tells any parent scroll layout to scroll to get given
// vertical coordinate at bottom of view to extent possible -- returns true if
// scrolled
func (ed *Editor) ScrollToBottom(pos int) bool {
	ed.NeedsRender()
	return ed.ScrollDimToEnd(math32.Y, pos)
}

// ScrollCursorToBottom tells any parent scroll layout to scroll to get cursor
// at bottom of view to extent possible -- returns true if scrolled.
func (ed *Editor) ScrollCursorToBottom() bool {
	curBBox := ed.CursorBBox(ed.CursorPos)
	return ed.ScrollToBottom(curBBox.Max.Y)
}

// ScrollToVertCenter tells any parent scroll layout to scroll to get given
// vertical coordinate to center of view to extent possible -- returns true if
// scrolled
func (ed *Editor) ScrollToVertCenter(pos int) bool {
	ed.NeedsRender()
	return ed.ScrollDimToCenter(math32.Y, pos)
}

// ScrollCursorToVertCenter tells any parent scroll layout to scroll to get
// cursor at vert center of view to extent possible -- returns true if
// scrolled.
func (ed *Editor) ScrollCursorToVertCenter() bool {
	curBBox := ed.CursorBBox(ed.CursorPos)
	mid := (curBBox.Min.Y + curBBox.Max.Y) / 2
	return ed.ScrollToVertCenter(mid)
}

func (ed *Editor) ScrollCursorToTarget() {
	// fmt.Println(ed, "to target:", ed.CursorTarg)
	ed.CursorPos = ed.CursorTarg
	ed.ScrollCursorToVertCenter()
	ed.SetFlag(false, EditorTargetSet)
}

///////////////////////////////////////////////////////////////////////////////
//    Scrolling -- Horizontal

// ScrollToLeft tells any parent scroll layout to scroll to get given
// horizontal coordinate at left of view to extent possible -- returns true if
// scrolled
func (ed *Editor) ScrollToLeft(pos int) bool {
	return ed.ScrollDimToStart(math32.X, pos)
}

// ScrollCursorToLeft tells any parent scroll layout to scroll to get cursor
// at left of view to extent possible -- returns true if scrolled.
func (ed *Editor) ScrollCursorToLeft() bool {
	_, ri, _ := ed.WrappedLineNumber(ed.CursorPos)
	if ri <= 0 {
		// todo: what is right thing here?
		// return ed.ScrollToLeft(ed.ObjBBox.Min.X - int(ed.Styles.BoxSpace().Left) - 2)
	}
	curBBox := ed.CursorBBox(ed.CursorPos)
	return ed.ScrollToLeft(curBBox.Min.X)
}

// ScrollToRight tells any parent scroll layout to scroll to get given
// horizontal coordinate at right of view to extent possible -- returns true
// if scrolled
func (ed *Editor) ScrollToRight(pos int) bool {
	return ed.ScrollDimToEnd(math32.X, pos)
}

// ScrollCursorToRight tells any parent scroll layout to scroll to get cursor
// at right of view to extent possible -- returns true if scrolled.
func (ed *Editor) ScrollCursorToRight() bool {
	curBBox := ed.CursorBBox(ed.CursorPos)
	return ed.ScrollToRight(curBBox.Max.X)
}

// ScrollToHorizCenter tells any parent scroll layout to scroll to get given
// horizontal coordinate to center of view to extent possible -- returns true if
// scrolled
func (ed *Editor) ScrollToHorizCenter(pos int) bool {
	return ed.ScrollDimToCenter(math32.X, pos)
}

// ScrollCursorToHorizCenter tells any parent scroll layout to scroll to get
// cursor at horiz center of view to extent possible -- returns true if
// scrolled.
func (ed *Editor) ScrollCursorToHorizCenter() bool {
	curBBox := ed.CursorBBox(ed.CursorPos)
	mn := int(math32.Ceil(float32(curBBox.Min.X) + ed.LineNumberOffset))
	mid := (mn + curBBox.Max.X) / 2
	return ed.ScrollToHorizCenter(mid)
}
