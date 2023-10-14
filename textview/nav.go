// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package textview

import (
	"image"

	"goki.dev/gi/v2/gi"
	"goki.dev/gi/v2/textview/textbuf"
	"goki.dev/laser"
	"goki.dev/mat32/v2"
	"goki.dev/pi/v2/lex"
)

///////////////////////////////////////////////////////////////////////////////
//  Cursor Navigation

// CursorMovedSig sends the signal that cursor has moved
func (tv *View) CursorMovedSig() {
	// tv.ViewSig.Emit(tv.This(), int64(ViewCursorMoved), tv.CursorPos)
}

// ValidateCursor sets current cursor to a valid cursor position
func (tv *View) ValidateCursor() {
	if tv.Buf != nil {
		tv.CursorPos = tv.Buf.ValidPos(tv.CursorPos)
	} else {
		tv.CursorPos = lex.PosZero
	}
}

// WrappedLines returns the number of wrapped lines (spans) for given line number
func (tv *View) WrappedLines(ln int) int {
	if ln >= len(tv.Renders) {
		return 0
	}
	return len(tv.Renders[ln].Spans)
}

// WrappedLineNo returns the wrapped line number (span index) and rune index
// within that span of the given character position within line in position,
// and false if out of range (last valid position returned in that case -- still usable).
func (tv *View) WrappedLineNo(pos lex.Pos) (si, ri int, ok bool) {
	if pos.Ln >= len(tv.Renders) {
		return 0, 0, false
	}
	return tv.Renders[pos.Ln].RuneSpanPos(pos.Ch)
}

// SetCursor sets a new cursor position, enforcing it in range
func (tv *View) SetCursor(pos lex.Pos) {
	if tv.NLines == 0 || tv.Buf == nil {
		tv.CursorPos = lex.PosZero
		return
	}
	updt := tv.UpdateStart()
	defer tv.UpdateEndRender(updt)

	tv.ClearScopelights()
	tv.CursorPos = tv.Buf.ValidPos(pos)
	tv.Buf.MarkupLine(tv.CursorPos.Ln)
	tv.CursorMovedSig()
	txt := tv.Buf.Line(tv.CursorPos.Ln)
	ch := tv.CursorPos.Ch
	if ch < len(txt) {
		r := txt[ch]
		if r == '{' || r == '}' || r == '(' || r == ')' || r == '[' || r == ']' {
			tp, found := tv.Buf.BraceMatch(txt[ch], tv.CursorPos)
			if found {
				tv.Scopelights = append(tv.Scopelights, textbuf.NewRegionPos(tv.CursorPos, lex.Pos{tv.CursorPos.Ln, tv.CursorPos.Ch + 1}))
				tv.Scopelights = append(tv.Scopelights, textbuf.NewRegionPos(tp, lex.Pos{tp.Ln, tp.Ch + 1}))
			}
		}
	}
}

// SetCursorShow sets a new cursor position, enforcing it in range, and shows
// the cursor (scroll to if hidden, render)
func (tv *View) SetCursorShow(pos lex.Pos) {
	tv.SetCursor(pos)
	tv.ScrollCursorToCenterIfHidden()
	tv.RenderCursor(true)
}

// SetCursorCol sets the current target cursor column (CursorCol) to that
// of the given position
func (tv *View) SetCursorCol(pos lex.Pos) {
	if wln := tv.WrappedLines(pos.Ln); wln > 1 {
		si, ri, ok := tv.WrappedLineNo(pos)
		if ok && si > 0 {
			tv.CursorCol = ri
		} else {
			tv.CursorCol = pos.Ch
		}
	} else {
		tv.CursorCol = pos.Ch
	}
}

// SavePosHistory saves the cursor position in history stack of cursor positions
func (tv *View) SavePosHistory(pos lex.Pos) {
	if tv.Buf == nil {
		return
	}
	tv.Buf.SavePosHistory(pos)
	tv.PosHistIdx = len(tv.Buf.PosHistory) - 1
}

// CursorToHistPrev moves cursor to previous position on history list --
// returns true if moved
func (tv *View) CursorToHistPrev() bool {
	if tv.NLines == 0 || tv.Buf == nil {
		tv.CursorPos = lex.PosZero
		return false
	}
	sz := len(tv.Buf.PosHistory)
	if sz == 0 {
		return false
	}
	tv.PosHistIdx--
	if tv.PosHistIdx < 0 {
		tv.PosHistIdx = 0
		return false
	}
	tv.PosHistIdx = min(sz-1, tv.PosHistIdx)
	pos := tv.Buf.PosHistory[tv.PosHistIdx]
	tv.CursorPos = tv.Buf.ValidPos(pos)
	tv.CursorMovedSig()
	tv.ScrollCursorToCenterIfHidden()
	tv.RenderCursor(true)
	return true
}

// CursorToHistNext moves cursor to previous position on history list --
// returns true if moved
func (tv *View) CursorToHistNext() bool {
	if tv.NLines == 0 || tv.Buf == nil {
		tv.CursorPos = lex.PosZero
		return false
	}
	sz := len(tv.Buf.PosHistory)
	if sz == 0 {
		return false
	}
	tv.PosHistIdx++
	if tv.PosHistIdx >= sz-1 {
		tv.PosHistIdx = sz - 1
		return false
	}
	pos := tv.Buf.PosHistory[tv.PosHistIdx]
	tv.CursorPos = tv.Buf.ValidPos(pos)
	tv.CursorMovedSig()
	tv.ScrollCursorToCenterIfHidden()
	tv.RenderCursor(true)
	return true
}

// SelectRegUpdate updates current select region based on given cursor position
// relative to SelectStart position
func (tv *View) SelectRegUpdate(pos lex.Pos) {
	if pos.IsLess(tv.SelectStart) {
		tv.SelectReg.Start = pos
		tv.SelectReg.End = tv.SelectStart
	} else {
		tv.SelectReg.Start = tv.SelectStart
		tv.SelectReg.End = pos
	}
}

// CursorSelect updates selection based on cursor movements, given starting
// cursor position and tv.CursorPos is current
func (tv *View) CursorSelect(org lex.Pos) {
	if !tv.SelectMode {
		return
	}
	tv.SelectRegUpdate(tv.CursorPos)
}

// CursorForward moves the cursor forward
func (tv *View) CursorForward(steps int) {
	updt := tv.UpdateStart()
	defer tv.UpdateEndRender(updt)
	tv.ValidateCursor()
	org := tv.CursorPos
	for i := 0; i < steps; i++ {
		tv.CursorPos.Ch++
		if tv.CursorPos.Ch > tv.Buf.LineLen(tv.CursorPos.Ln) {
			if tv.CursorPos.Ln < tv.NLines-1 {
				tv.CursorPos.Ch = 0
				tv.CursorPos.Ln++
			} else {
				tv.CursorPos.Ch = tv.Buf.LineLen(tv.CursorPos.Ln)
			}
		}
	}
	tv.SetCursorCol(tv.CursorPos)
	tv.SetCursorShow(tv.CursorPos)
	tv.CursorSelect(org)
}

// CursorForwardWord moves the cursor forward by words
func (tv *View) CursorForwardWord(steps int) {
	updt := tv.UpdateStart()
	defer tv.UpdateEndRender(updt)
	tv.ValidateCursor()
	org := tv.CursorPos
	for i := 0; i < steps; i++ {
		txt := tv.Buf.Line(tv.CursorPos.Ln)
		sz := len(txt)
		if sz > 0 && tv.CursorPos.Ch < sz {
			ch := tv.CursorPos.Ch
			var done = false
			for ch < sz && !done { // if on a wb, go past
				r1 := txt[ch]
				r2 := rune(-1)
				if ch < sz-1 {
					r2 = txt[ch+1]
				}
				if lex.IsWordBreak(r1, r2) {
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
				if !lex.IsWordBreak(r1, r2) {
					ch++
				} else {
					done = true
				}
			}
			tv.CursorPos.Ch = ch
		} else {
			if tv.CursorPos.Ln < tv.NLines-1 {
				tv.CursorPos.Ch = 0
				tv.CursorPos.Ln++
			} else {
				tv.CursorPos.Ch = tv.Buf.LineLen(tv.CursorPos.Ln)
			}
		}
	}
	tv.SetCursorCol(tv.CursorPos)
	tv.SetCursorShow(tv.CursorPos)
	tv.CursorSelect(org)
}

// CursorDown moves the cursor down line(s)
func (tv *View) CursorDown(steps int) {
	updt := tv.UpdateStart()
	defer tv.UpdateEndRender(updt)
	tv.ValidateCursor()
	org := tv.CursorPos
	pos := tv.CursorPos
	for i := 0; i < steps; i++ {
		gotwrap := false
		if wln := tv.WrappedLines(pos.Ln); wln > 1 {
			si, ri, _ := tv.WrappedLineNo(pos)
			if si < wln-1 {
				si++
				mxlen := min(len(tv.Renders[pos.Ln].Spans[si].Text), tv.CursorCol)
				if tv.CursorCol < mxlen {
					ri = tv.CursorCol
				} else {
					ri = mxlen
				}
				nwc, _ := tv.Renders[pos.Ln].SpanPosToRuneIdx(si, ri)
				if si == wln-1 && ri == mxlen {
					nwc++
				}
				pos.Ch = nwc
				gotwrap = true

			}
		}
		if !gotwrap {
			pos.Ln++
			if pos.Ln >= tv.NLines {
				pos.Ln = tv.NLines - 1
				break
			}
			mxlen := min(tv.Buf.LineLen(pos.Ln), tv.CursorCol)
			if tv.CursorCol < mxlen {
				pos.Ch = tv.CursorCol
			} else {
				pos.Ch = mxlen
			}
		}
	}
	tv.SetCursorShow(pos)
	tv.CursorSelect(org)
}

// CursorPageDown moves the cursor down page(s), where a page is defined abcdef
// dynamically as just moving the cursor off the screen
func (tv *View) CursorPageDown(steps int) {
	updt := tv.UpdateStart()
	defer tv.UpdateEndRender(updt)
	tv.ValidateCursor()
	org := tv.CursorPos
	for i := 0; i < steps; i++ {
		lvln := tv.LastVisibleLine(tv.CursorPos.Ln)
		tv.CursorPos.Ln = lvln
		if tv.CursorPos.Ln >= tv.NLines {
			tv.CursorPos.Ln = tv.NLines - 1
		}
		tv.CursorPos.Ch = min(tv.Buf.LineLen(tv.CursorPos.Ln), tv.CursorCol)
		tv.ScrollCursorToTop()
		tv.RenderCursor(true)
	}
	tv.SetCursor(tv.CursorPos)
	tv.CursorSelect(org)
}

// CursorBackward moves the cursor backward
func (tv *View) CursorBackward(steps int) {
	updt := tv.UpdateStart()
	defer tv.UpdateEndRender(updt)
	tv.ValidateCursor()
	org := tv.CursorPos
	for i := 0; i < steps; i++ {
		tv.CursorPos.Ch--
		if tv.CursorPos.Ch < 0 {
			if tv.CursorPos.Ln > 0 {
				tv.CursorPos.Ln--
				tv.CursorPos.Ch = tv.Buf.LineLen(tv.CursorPos.Ln)
			} else {
				tv.CursorPos.Ch = 0
			}
		}
	}
	tv.SetCursorCol(tv.CursorPos)
	tv.SetCursorShow(tv.CursorPos)
	tv.CursorSelect(org)
}

// CursorBackwardWord moves the cursor backward by words
func (tv *View) CursorBackwardWord(steps int) {
	updt := tv.UpdateStart()
	defer tv.UpdateEndRender(updt)
	tv.ValidateCursor()
	org := tv.CursorPos
	for i := 0; i < steps; i++ {
		txt := tv.Buf.Line(tv.CursorPos.Ln)
		sz := len(txt)
		if sz > 0 && tv.CursorPos.Ch > 0 {
			ch := min(tv.CursorPos.Ch, sz-1)
			var done = false
			for ch < sz && !done { // if on a wb, go past
				r1 := txt[ch]
				r2 := rune(-1)
				if ch > 0 {
					r2 = txt[ch-1]
				}
				if lex.IsWordBreak(r1, r2) {
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
				if !lex.IsWordBreak(r1, r2) {
					ch--
				} else {
					done = true
				}
			}
			tv.CursorPos.Ch = ch
		} else {
			if tv.CursorPos.Ln > 0 {
				tv.CursorPos.Ln--
				tv.CursorPos.Ch = tv.Buf.LineLen(tv.CursorPos.Ln)
			} else {
				tv.CursorPos.Ch = 0
			}
		}
	}
	tv.SetCursorCol(tv.CursorPos)
	tv.SetCursorShow(tv.CursorPos)
	tv.CursorSelect(org)
}

// CursorUp moves the cursor up line(s)
func (tv *View) CursorUp(steps int) {
	updt := tv.UpdateStart()
	defer tv.UpdateEndRender(updt)
	tv.ValidateCursor()
	org := tv.CursorPos
	pos := tv.CursorPos
	for i := 0; i < steps; i++ {
		gotwrap := false
		if wln := tv.WrappedLines(pos.Ln); wln > 1 {
			si, ri, _ := tv.WrappedLineNo(pos)
			if si > 0 {
				ri = tv.CursorCol
				// fmt.Printf("up cursorcol: %v\n", tv.CursorCol)
				nwc, _ := tv.Renders[pos.Ln].SpanPosToRuneIdx(si-1, ri)
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
			if wln := tv.WrappedLines(pos.Ln); wln > 1 { // just entered end of wrapped line
				si := wln - 1
				ri := tv.CursorCol
				nwc, _ := tv.Renders[pos.Ln].SpanPosToRuneIdx(si, ri)
				pos.Ch = nwc
			} else {
				mxlen := min(tv.Buf.LineLen(pos.Ln), tv.CursorCol)
				if tv.CursorCol < mxlen {
					pos.Ch = tv.CursorCol
				} else {
					pos.Ch = mxlen
				}
			}
		}
	}
	tv.SetCursorShow(pos)
	tv.CursorSelect(org)
}

// CursorPageUp moves the cursor up page(s), where a page is defined
// dynamically as just moving the cursor off the screen
func (tv *View) CursorPageUp(steps int) {
	updt := tv.UpdateStart()
	defer tv.UpdateEndRender(updt)
	tv.ValidateCursor()
	org := tv.CursorPos
	for i := 0; i < steps; i++ {
		lvln := tv.FirstVisibleLine(tv.CursorPos.Ln)
		tv.CursorPos.Ln = lvln
		if tv.CursorPos.Ln <= 0 {
			tv.CursorPos.Ln = 0
		}
		tv.CursorPos.Ch = min(tv.Buf.LineLen(tv.CursorPos.Ln), tv.CursorCol)
		tv.ScrollCursorToBottom()
		tv.RenderCursor(true)
	}
	tv.SetCursor(tv.CursorPos)
	tv.CursorSelect(org)
}

// CursorRecenter re-centers the view around the cursor position, toggling
// between putting cursor in middle, top, and bottom of view
func (tv *View) CursorRecenter() {
	tv.ValidateCursor()
	tv.SavePosHistory(tv.CursorPos)
	cur := (tv.lastRecenter + 1) % 3
	switch cur {
	case 0:
		tv.ScrollCursorToBottom()
	case 1:
		tv.ScrollCursorToVertCenter()
	case 2:
		tv.ScrollCursorToTop()
	}
	tv.lastRecenter = cur
}

// CursorStartLine moves the cursor to the start of the line, updating selection
// if select mode is active
func (tv *View) CursorStartLine() {
	updt := tv.UpdateStart()
	defer tv.UpdateEndRender(updt)
	tv.ValidateCursor()
	org := tv.CursorPos
	pos := tv.CursorPos

	gotwrap := false
	if wln := tv.WrappedLines(pos.Ln); wln > 1 {
		si, ri, _ := tv.WrappedLineNo(pos)
		if si > 0 {
			ri = 0
			nwc, _ := tv.Renders[pos.Ln].SpanPosToRuneIdx(si, ri)
			pos.Ch = nwc
			tv.CursorPos = pos
			tv.CursorCol = ri
			gotwrap = true
		}
	}
	if !gotwrap {
		tv.CursorPos.Ch = 0
		tv.CursorCol = tv.CursorPos.Ch
	}
	// fmt.Printf("sol cursorcol: %v\n", tv.CursorCol)
	tv.SetCursor(tv.CursorPos)
	tv.ScrollCursorToLeft()
	tv.RenderCursor(true)
	tv.CursorSelect(org)
}

// CursorStartDoc moves the cursor to the start of the text, updating selection
// if select mode is active
func (tv *View) CursorStartDoc() {
	updt := tv.UpdateStart()
	defer tv.UpdateEndRender(updt)
	tv.ValidateCursor()
	org := tv.CursorPos
	tv.CursorPos.Ln = 0
	tv.CursorPos.Ch = 0
	tv.CursorCol = tv.CursorPos.Ch
	tv.SetCursor(tv.CursorPos)
	tv.ScrollCursorToTop()
	tv.RenderCursor(true)
	tv.CursorSelect(org)
}

// CursorEndLine moves the cursor to the end of the text
func (tv *View) CursorEndLine() {
	updt := tv.UpdateStart()
	defer tv.UpdateEndRender(updt)
	tv.ValidateCursor()
	org := tv.CursorPos
	pos := tv.CursorPos

	gotwrap := false
	if wln := tv.WrappedLines(pos.Ln); wln > 1 {
		si, ri, _ := tv.WrappedLineNo(pos)
		ri = len(tv.Renders[pos.Ln].Spans[si].Text) - 1
		nwc, _ := tv.Renders[pos.Ln].SpanPosToRuneIdx(si, ri)
		if si == len(tv.Renders[pos.Ln].Spans)-1 { // last span
			ri++
			nwc++
		}
		tv.CursorCol = ri
		pos.Ch = nwc
		tv.CursorPos = pos
		gotwrap = true
	}
	if !gotwrap {
		tv.CursorPos.Ch = tv.Buf.LineLen(tv.CursorPos.Ln)
		tv.CursorCol = tv.CursorPos.Ch
	}
	tv.SetCursor(tv.CursorPos)
	tv.ScrollCursorToRight()
	tv.RenderCursor(true)
	tv.CursorSelect(org)
}

// CursorEndDoc moves the cursor to the end of the text, updating selection if
// select mode is active
func (tv *View) CursorEndDoc() {
	updt := tv.UpdateStart()
	defer tv.UpdateEndRender(updt)
	tv.ValidateCursor()
	org := tv.CursorPos
	tv.CursorPos.Ln = max(tv.NLines-1, 0)
	tv.CursorPos.Ch = tv.Buf.LineLen(tv.CursorPos.Ln)
	tv.CursorCol = tv.CursorPos.Ch
	tv.SetCursor(tv.CursorPos)
	tv.ScrollCursorToBottom()
	tv.RenderCursor(true)
	tv.CursorSelect(org)
}

// todo: ctrl+backspace = delete word
// shift+arrow = select
// uparrow = start / down = end

// CursorBackspace deletes character(s) immediately before cursor
func (tv *View) CursorBackspace(steps int) {
	updt := tv.UpdateStart()
	defer tv.UpdateEndRender(updt)
	tv.ValidateCursor()
	org := tv.CursorPos
	if tv.HasSelection() {
		org = tv.SelectReg.Start
		tv.DeleteSelection()
		tv.SetCursorShow(org)
		return
	}
	// note: no update b/c signal from buf will drive update
	tv.CursorBackward(steps)
	tv.ScrollCursorToCenterIfHidden()
	tv.RenderCursor(true)
	tv.Buf.DeleteText(tv.CursorPos, org, EditSignal)
}

// CursorDelete deletes character(s) immediately after the cursor
func (tv *View) CursorDelete(steps int) {
	updt := tv.UpdateStart()
	defer tv.UpdateEndRender(updt)
	tv.ValidateCursor()
	if tv.HasSelection() {
		tv.DeleteSelection()
		return
	}
	// note: no update b/c signal from buf will drive update
	org := tv.CursorPos
	tv.CursorForward(steps)
	tv.Buf.DeleteText(org, tv.CursorPos, EditSignal)
	tv.SetCursorShow(org)
}

// CursorBackspaceWord deletes words(s) immediately before cursor
func (tv *View) CursorBackspaceWord(steps int) {
	updt := tv.UpdateStart()
	defer tv.UpdateEndRender(updt)
	tv.ValidateCursor()
	org := tv.CursorPos
	if tv.HasSelection() {
		tv.DeleteSelection()
		tv.SetCursorShow(org)
		return
	}
	// note: no update b/c signal from buf will drive update
	tv.CursorBackwardWord(steps)
	tv.ScrollCursorToCenterIfHidden()
	tv.RenderCursor(true)
	tv.Buf.DeleteText(tv.CursorPos, org, EditSignal)
}

// CursorDeleteWord deletes word(s) immediately after the cursor
func (tv *View) CursorDeleteWord(steps int) {
	updt := tv.UpdateStart()
	defer tv.UpdateEndRender(updt)
	tv.ValidateCursor()
	if tv.HasSelection() {
		tv.DeleteSelection()
		return
	}
	// note: no update b/c signal from buf will drive update
	org := tv.CursorPos
	tv.CursorForwardWord(steps)
	tv.Buf.DeleteText(org, tv.CursorPos, EditSignal)
	tv.SetCursorShow(org)
}

// CursorKill deletes text from cursor to end of text
func (tv *View) CursorKill() {
	updt := tv.UpdateStart()
	defer tv.UpdateEndRender(updt)
	tv.ValidateCursor()
	org := tv.CursorPos
	pos := tv.CursorPos

	atEnd := false
	if wln := tv.WrappedLines(pos.Ln); wln > 1 {
		si, ri, _ := tv.WrappedLineNo(pos)
		llen := len(tv.Renders[pos.Ln].Spans[si].Text)
		if si == wln-1 {
			llen--
		}
		atEnd = (ri == llen)
	} else {
		llen := tv.Buf.LineLen(pos.Ln)
		atEnd = (tv.CursorPos.Ch == llen)
	}
	if atEnd {
		tv.CursorForward(1)
	} else {
		tv.CursorEndLine()
	}
	tv.Buf.DeleteText(org, tv.CursorPos, EditSignal)
	tv.SetCursorShow(org)
}

// CursorTranspose swaps the character at the cursor with the one before it
func (tv *View) CursorTranspose() {
	updt := tv.UpdateStart()
	defer tv.UpdateEndRender(updt)
	tv.ValidateCursor()
	pos := tv.CursorPos
	if pos.Ch == 0 {
		return
	}
	ppos := pos
	ppos.Ch--
	tv.Buf.LinesMu.Lock()
	lln := len(tv.Buf.Lines[pos.Ln])
	end := false
	if pos.Ch >= lln {
		end = true
		pos.Ch = lln - 1
		ppos.Ch = lln - 2
	}
	chr := tv.Buf.Lines[pos.Ln][pos.Ch]
	pchr := tv.Buf.Lines[pos.Ln][ppos.Ch]
	tv.Buf.LinesMu.Unlock()
	repl := string([]rune{chr, pchr})
	pos.Ch++
	tv.Buf.ReplaceText(ppos, pos, ppos, repl, EditSignal, ReplaceMatchCase)
	if !end {
		tv.SetCursorShow(pos)
	}
}

// CursorTranspose swaps the word at the cursor with the one before it
func (tv *View) CursorTransposeWord() {
}

// JumpToLinePrompt jumps to given line number (minus 1) from prompt
func (tv *View) JumpToLinePrompt() {
	gi.StringPromptDialog(tv, gi.DlgOpts{Title: "Jump To Line", Prompt: "Line Number to jump to"},
		"", "Line no..", func(dlg *gi.Dialog) {
			if dlg.Accepted {
				val := dlg.Data.(string)
				ln, err := laser.ToInt(val)
				if err == nil {
					tv.JumpToLine(int(ln))
				}
			}
		})

}

// JumpToLine jumps to given line number (minus 1)
func (tv *View) JumpToLine(ln int) {
	updt := tv.UpdateStart()
	tv.SetCursorShow(lex.Pos{Ln: ln - 1})
	tv.SavePosHistory(tv.CursorPos)
	tv.UpdateEndRender(updt)
}

// FindNextLink finds next link after given position, returns false if no such links
func (tv *View) FindNextLink(pos lex.Pos) (lex.Pos, textbuf.Region, bool) {
	for ln := pos.Ln; ln < tv.NLines; ln++ {
		if len(tv.Renders[ln].Links) == 0 {
			pos.Ch = 0
			pos.Ln = ln + 1
			continue
		}
		rend := &tv.Renders[ln]
		si, ri, _ := rend.RuneSpanPos(pos.Ch)
		for ti := range rend.Links {
			tl := &rend.Links[ti]
			if tl.StartSpan >= si && tl.StartIdx >= ri {
				st, _ := rend.SpanPosToRuneIdx(tl.StartSpan, tl.StartIdx)
				ed, _ := rend.SpanPosToRuneIdx(tl.EndSpan, tl.EndIdx)
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
func (tv *View) FindPrevLink(pos lex.Pos) (lex.Pos, textbuf.Region, bool) {
	for ln := pos.Ln - 1; ln >= 0; ln-- {
		if len(tv.Renders[ln].Links) == 0 {
			if ln-1 >= 0 {
				pos.Ch = tv.Buf.LineLen(ln-1) - 2
			} else {
				ln = tv.NLines
				pos.Ch = tv.Buf.LineLen(ln - 2)
			}
			continue
		}
		rend := &tv.Renders[ln]
		si, ri, _ := rend.RuneSpanPos(pos.Ch)
		nl := len(rend.Links)
		for ti := nl - 1; ti >= 0; ti-- {
			tl := &rend.Links[ti]
			if tl.StartSpan <= si && tl.StartIdx < ri {
				st, _ := rend.SpanPosToRuneIdx(tl.StartSpan, tl.StartIdx)
				ed, _ := rend.SpanPosToRuneIdx(tl.EndSpan, tl.EndIdx)
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
func (tv *View) CursorNextLink(wraparound bool) bool {
	if tv.NLines == 0 {
		return false
	}
	tv.ValidateCursor()
	npos, reg, has := tv.FindNextLink(tv.CursorPos)
	if !has {
		if !wraparound {
			return false
		}
		npos, reg, has = tv.FindNextLink(lex.Pos{}) // wraparound
		if !has {
			return false
		}
	}
	updt := tv.UpdateStart()
	defer tv.UpdateEndRender(updt)
	tv.HighlightRegion(reg)
	tv.SetCursorShow(npos)
	tv.SavePosHistory(tv.CursorPos)
	return true
}

// CursorPrevLink moves cursor to previous link. wraparound wraps around to
// bottom of buffer if none found. returns true if found
func (tv *View) CursorPrevLink(wraparound bool) bool {
	if tv.NLines == 0 {
		return false
	}
	tv.ValidateCursor()
	npos, reg, has := tv.FindPrevLink(tv.CursorPos)
	if !has {
		if !wraparound {
			return false
		}
		npos, reg, has = tv.FindPrevLink(lex.Pos{}) // wraparound
		if !has {
			return false
		}
	}
	updt := tv.UpdateStart()
	defer tv.UpdateEndRender(updt)

	tv.HighlightRegion(reg)
	tv.SetCursorShow(npos)
	tv.SavePosHistory(tv.CursorPos)
	return true
}

///////////////////////////////////////////////////////////////////////////////
//    Scrolling

// ScrollInView tells any parent scroll layout to scroll to get given box
// (e.g., cursor BBox) in view -- returns true if scrolled
func (tv *View) ScrollInView(bbox image.Rectangle) bool {
	return tv.ScrollToBox(bbox)
}

// ScrollCursorInView tells any parent scroll layout to scroll to get cursor
// in view -- returns true if scrolled
func (tv *View) ScrollCursorInView() bool {
	if tv == nil || tv.This() == nil {
		return false
	}
	if tv.This().(gi.Widget).IsVisible() {
		curBBox := tv.CursorBBox(tv.CursorPos)
		return tv.ScrollInView(curBBox)
	}
	return false
}

// TODO: do we need something like this? this stack overflows
// // AutoScroll tells any parent scroll layout to scroll to do its autoscroll
// // based on given location -- for dragging
// func (tv *View) AutoScroll(pos image.Point) bool {
// 	return tv.AutoScroll(pos)
// }

// ScrollCursorToCenterIfHidden checks if the cursor is not visible, and if
// so, scrolls to the center, along both dimensions.
func (tv *View) ScrollCursorToCenterIfHidden() bool {
	curBBox := tv.CursorBBox(tv.CursorPos)
	did := false
	if (curBBox.Min.Y-int(tv.LineHeight)) < tv.ScBBox.Min.Y || (curBBox.Max.Y+int(tv.LineHeight)) > tv.ScBBox.Max.Y {
		did = tv.ScrollCursorToVertCenter()
	}
	if curBBox.Max.X < tv.ScBBox.Min.X || curBBox.Min.X > tv.ScBBox.Max.X {
		did = did || tv.ScrollCursorToHorizCenter()
	}
	return did
}

///////////////////////////////////////////////////////////////////////////////
//    Scrolling -- Vertical

// ScrollToTop tells any parent scroll layout to scroll to get given vertical
// coordinate at top of view to extent possible -- returns true if scrolled
func (tv *View) ScrollToTop(pos int) bool {
	return tv.ScrollDimToStart(mat32.Y, pos)
}

// ScrollCursorToTop tells any parent scroll layout to scroll to get cursor
// at top of view to extent possible -- returns true if scrolled.
func (tv *View) ScrollCursorToTop() bool {
	curBBox := tv.CursorBBox(tv.CursorPos)
	return tv.ScrollToTop(curBBox.Min.Y)
}

// ScrollToBottom tells any parent scroll layout to scroll to get given
// vertical coordinate at bottom of view to extent possible -- returns true if
// scrolled
func (tv *View) ScrollToBottom(pos int) bool {
	return tv.ScrollDimToEnd(mat32.Y, pos)
}

// ScrollCursorToBottom tells any parent scroll layout to scroll to get cursor
// at bottom of view to extent possible -- returns true if scrolled.
func (tv *View) ScrollCursorToBottom() bool {
	curBBox := tv.CursorBBox(tv.CursorPos)
	return tv.ScrollToBottom(curBBox.Max.Y)
}

// ScrollToVertCenter tells any parent scroll layout to scroll to get given
// vertical coordinate to center of view to extent possible -- returns true if
// scrolled
func (tv *View) ScrollToVertCenter(pos int) bool {
	return tv.ScrollDimToCenter(mat32.Y, pos)
}

// ScrollCursorToVertCenter tells any parent scroll layout to scroll to get
// cursor at vert center of view to extent possible -- returns true if
// scrolled.
func (tv *View) ScrollCursorToVertCenter() bool {
	curBBox := tv.CursorBBox(tv.CursorPos)
	mid := (curBBox.Min.Y + curBBox.Max.Y) / 2
	return tv.ScrollToVertCenter(mid)
}

///////////////////////////////////////////////////////////////////////////////
//    Scrolling -- Horizontal

// ScrollToLeft tells any parent scroll layout to scroll to get given
// horizontal coordinate at left of view to extent possible -- returns true if
// scrolled
func (tv *View) ScrollToLeft(pos int) bool {
	return tv.ScrollDimToStart(mat32.X, pos)
}

// ScrollCursorToLeft tells any parent scroll layout to scroll to get cursor
// at left of view to extent possible -- returns true if scrolled.
func (tv *View) ScrollCursorToLeft() bool {
	_, ri, _ := tv.WrappedLineNo(tv.CursorPos)
	if ri <= 0 {
		return tv.ScrollToLeft(tv.ObjBBox.Min.X - int(tv.Style.BoxSpace().Left) - 2)
	}
	curBBox := tv.CursorBBox(tv.CursorPos)
	return tv.ScrollToLeft(curBBox.Min.X)
}

// ScrollToRight tells any parent scroll layout to scroll to get given
// horizontal coordinate at right of view to extent possible -- returns true
// if scrolled
func (tv *View) ScrollToRight(pos int) bool {
	return tv.ScrollDimToEnd(mat32.X, pos)
}

// ScrollCursorToRight tells any parent scroll layout to scroll to get cursor
// at right of view to extent possible -- returns true if scrolled.
func (tv *View) ScrollCursorToRight() bool {
	curBBox := tv.CursorBBox(tv.CursorPos)
	return tv.ScrollToRight(curBBox.Max.X)
}

// ScrollToHorizCenter tells any parent scroll layout to scroll to get given
// horizontal coordinate to center of view to extent possible -- returns true if
// scrolled
func (tv *View) ScrollToHorizCenter(pos int) bool {
	return tv.ScrollDimToCenter(mat32.X, pos)
}

// ScrollCursorToHorizCenter tells any parent scroll layout to scroll to get
// cursor at horiz center of view to extent possible -- returns true if
// scrolled.
func (tv *View) ScrollCursorToHorizCenter() bool {
	curBBox := tv.CursorBBox(tv.CursorPos)
	mid := (curBBox.Min.X + curBBox.Max.X) / 2
	return tv.ScrollToHorizCenter(mid)
}
