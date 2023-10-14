// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package textview

import (
	"image"

	"goki.dev/gi/v2/gi"
	"goki.dev/gi/v2/textview/textbuf"
	"goki.dev/girl/states"
	"goki.dev/goosi/mimedata"
	"goki.dev/pi/v2/filecat"
	"goki.dev/pi/v2/lex"
)

//////////////////////////////////////////////////////////
// 	Regions

// HighlightRegion creates a new highlighted region,
// triggers updating.
func (tv *View) HighlightRegion(reg textbuf.Region) {
	tv.Highlights = []textbuf.Region{reg}
	tv.SetNeedsRender()
}

// ClearHighlights clears the Highlights slice of all regions
func (tv *View) ClearHighlights() {
	if len(tv.Highlights) == 0 {
		return
	}
	tv.Highlights = tv.Highlights[:0]
	tv.SetNeedsRender()
}

// ClearScopelights clears the Highlights slice of all regions
func (tv *View) ClearScopelights() {
	if len(tv.Scopelights) == 0 {
		return
	}
	updt := tv.UpdateStart()
	defer tv.UpdateEndRender(updt)
	sl := make([]textbuf.Region, len(tv.Scopelights))
	copy(sl, tv.Scopelights)
	tv.Scopelights = tv.Scopelights[:0]
}

//////////////////////////////////////////////////////////
// 	Selection

// ClearSelected resets both the global selected flag and any current selection
func (tv *View) ClearSelected() {
	// tv.WidgetBase.ClearSelected()
	tv.SelectReset()
}

// HasSelection returns whether there is a selected region of text
func (tv *View) HasSelection() bool {
	if tv.SelectReg.Start.IsLess(tv.SelectReg.End) {
		return true
	}
	return false
}

// Selection returns the currently selected text as a textbuf.Edit, which
// captures start, end, and full lines in between -- nil if no selection
func (tv *View) Selection() *textbuf.Edit {
	if tv.HasSelection() {
		return tv.Buf.Region(tv.SelectReg.Start, tv.SelectReg.End)
	}
	return nil
}

// SelectModeToggle toggles the SelectMode, updating selection with cursor movement
func (tv *View) SelectModeToggle() {
	if tv.SelectMode {
		tv.SelectMode = false
	} else {
		tv.SelectMode = true
		tv.SelectStart = tv.CursorPos
		tv.SelectRegUpdate(tv.CursorPos)
	}
	tv.SavePosHistory(tv.CursorPos)
}

// SelectAll selects all the text
func (tv *View) SelectAll() {
	updt := tv.UpdateStart()
	defer tv.UpdateEndRender(updt)
	tv.SelectReg.Start = lex.PosZero
	tv.SelectReg.End = tv.Buf.EndPos()
}

// WordBefore returns the word before the lex.Pos
// uses IsWordBreak to determine the bounds of the word
func (tv *View) WordBefore(tp lex.Pos) *textbuf.Edit {
	txt := tv.Buf.Line(tp.Ln)
	ch := tp.Ch
	ch = min(ch, len(txt))
	st := ch
	for i := ch - 1; i >= 0; i-- {
		if i == 0 { // start of line
			st = 0
			break
		}
		r1 := txt[i]
		r2 := txt[i-1]
		if lex.IsWordBreak(r1, r2) {
			st = i + 1
			break
		}
	}
	if st != ch {
		return tv.Buf.Region(lex.Pos{Ln: tp.Ln, Ch: st}, tp)
	}
	return nil
}

// IsWordStart returns true if the cursor is just before the start of a word
// word is a string of characters none of which are classified as a word break
func (tv *View) IsWordStart(tp lex.Pos) bool {
	txt := tv.Buf.Line(tv.CursorPos.Ln)
	sz := len(txt)
	if sz == 0 {
		return false
	}
	if tp.Ch >= len(txt) { // end of line
		return false
	}
	if tp.Ch == 0 { // start of line
		r := txt[0]
		if lex.IsWordBreak(r, rune(-1)) {
			return false
		}
		return true
	}
	r1 := txt[tp.Ch-1]
	r2 := txt[tp.Ch]
	if lex.IsWordBreak(r1, rune(-1)) && !lex.IsWordBreak(r2, rune(-1)) {
		return true
	}
	return false
}

// IsWordEnd returns true if the cursor is just past the last letter of a word
// word is a string of characters none of which are classified as a word break
func (tv *View) IsWordEnd(tp lex.Pos) bool {
	txt := tv.Buf.Line(tv.CursorPos.Ln)
	sz := len(txt)
	if sz == 0 {
		return false
	}
	if tp.Ch >= len(txt) { // end of line
		r := txt[len(txt)-1]
		if lex.IsWordBreak(r, rune(-1)) {
			return true
		}
		return false
	}
	if tp.Ch == 0 { // start of line
		r := txt[0]
		if lex.IsWordBreak(r, rune(-1)) {
			return false
		}
		return true
	}
	r1 := txt[tp.Ch-1]
	r2 := txt[tp.Ch]
	if !lex.IsWordBreak(r1, rune(-1)) && lex.IsWordBreak(r2, rune(-1)) {
		return true
	}
	return false
}

// IsWordMiddle - returns true if the cursor is anywhere inside a word,
// i.e. the character before the cursor and the one after the cursor
// are not classified as word break characters
func (tv *View) IsWordMiddle(tp lex.Pos) bool {
	txt := tv.Buf.Line(tv.CursorPos.Ln)
	sz := len(txt)
	if sz < 2 {
		return false
	}
	if tp.Ch >= len(txt) { // end of line
		return false
	}
	if tp.Ch == 0 { // start of line
		return false
	}
	r1 := txt[tp.Ch-1]
	r2 := txt[tp.Ch]
	if !lex.IsWordBreak(r1, rune(-1)) && !lex.IsWordBreak(r2, rune(-1)) {
		return true
	}
	return false
}

// SelectWord selects the word (whitespace, punctuation delimited) that the cursor is on
// returns true if word selected
func (tv *View) SelectWord() bool {
	txt := tv.Buf.Line(tv.CursorPos.Ln)
	sz := len(txt)
	if sz == 0 {
		return false
	}
	reg := tv.WordAt()
	tv.SelectReg = reg
	tv.SelectStart = tv.SelectReg.Start
	return true
}

// WordAt finds the region of the word at the current cursor position
func (tv *View) WordAt() (reg textbuf.Region) {
	reg.Start = tv.CursorPos
	reg.End = tv.CursorPos
	txt := tv.Buf.Line(tv.CursorPos.Ln)
	sz := len(txt)
	if sz == 0 {
		return reg
	}
	sch := min(tv.CursorPos.Ch, sz-1)
	if !lex.IsWordBreak(txt[sch], rune(-1)) {
		for sch > 0 {
			r2 := rune(-1)
			if sch-2 >= 0 {
				r2 = txt[sch-2]
			}
			if lex.IsWordBreak(txt[sch-1], r2) {
				break
			}
			sch--
		}
		reg.Start.Ch = sch
		ech := tv.CursorPos.Ch + 1
		for ech < sz {
			r2 := rune(-1)
			if ech < sz-1 {
				r2 = rune(txt[ech+1])
			}
			if lex.IsWordBreak(txt[ech], r2) {
				break
			}
			ech++
		}
		reg.End.Ch = ech
	} else { // keep the space start -- go to next space..
		ech := tv.CursorPos.Ch + 1
		for ech < sz {
			if !lex.IsWordBreak(txt[ech], rune(-1)) {
				break
			}
			ech++
		}
		for ech < sz {
			r2 := rune(-1)
			if ech < sz-1 {
				r2 = rune(txt[ech+1])
			}
			if lex.IsWordBreak(txt[ech], r2) {
				break
			}
			ech++
		}
		reg.End.Ch = ech
	}
	return reg
}

// SelectReset resets the selection
func (tv *View) SelectReset() {
	tv.SelectMode = false
	if !tv.HasSelection() {
		return
	}
	tv.SelectReg = textbuf.RegionNil
	tv.PrevSelectReg = textbuf.RegionNil
}

// RenderSelectLines renders the lines within the current selection region
func (tv *View) RenderSelectLines() {
	tv.PrevSelectReg = tv.SelectReg
}

///////////////////////////////////////////////////////////////////////////////
//    Cut / Copy / Paste

// ViewClipHistory is the text view clipboard history -- everything that has been copied
var ViewClipHistory [][]byte

// Maximum amount of clipboard history to retain
var ViewClipHistMax = 100

// ViewClipHistAdd adds the given clipboard bytes to top of history stack
func ViewClipHistAdd(clip []byte) {
	max := ViewClipHistMax
	if ViewClipHistory == nil {
		ViewClipHistory = make([][]byte, 0, max)
	}

	ch := &ViewClipHistory

	sz := len(*ch)
	if sz > max {
		*ch = (*ch)[:max]
	}
	if sz >= max {
		copy((*ch)[1:max], (*ch)[0:max-1])
		(*ch)[0] = clip
	} else {
		*ch = append(*ch, nil)
		if sz > 0 {
			copy((*ch)[1:], (*ch)[0:sz])
		}
		(*ch)[0] = clip
	}
}

// ViewClipHistChooseLen is the max length of clip history to show in chooser
var ViewClipHistChooseLen = 40

// ViewClipHistChooseList returns a string slice of length-limited clip history, for chooser
func ViewClipHistChooseList() []string {
	cl := make([]string, len(ViewClipHistory))
	for i, hc := range ViewClipHistory {
		szl := len(hc)
		if szl > ViewClipHistChooseLen {
			cl[i] = string(hc[:ViewClipHistChooseLen])
		} else {
			cl[i] = string(hc)
		}
	}
	return cl
}

// PasteHist presents a chooser of clip history items, pastes into text if selected
func (tv *View) PasteHist() {
	if ViewClipHistory == nil {
		return
	}
	cl := ViewClipHistChooseList()
	gi.StringsChooserPopup(cl, "", tv, func(ac *gi.Button) {
		idx := ac.Data.(int)
		clip := ViewClipHistory[idx]
		if clip != nil {
			updt := tv.UpdateStart()
			defer tv.UpdateEndRender(updt)
			tv.EventMgr().ClipBoard().Write(mimedata.NewTextBytes(clip))
			tv.InsertAtCursor(clip)
			tv.SavePosHistory(tv.CursorPos)
		}
	})
}

// Cut cuts any selected text and adds it to the clipboard, also returns cut text
func (tv *View) Cut() *textbuf.Edit {
	if !tv.HasSelection() {
		return nil
	}
	updt := tv.UpdateStart()
	defer tv.UpdateEndRender(updt)
	org := tv.SelectReg.Start
	cut := tv.DeleteSelection()
	if cut != nil {
		cb := cut.ToBytes()
		tv.EventMgr().ClipBoard().Write(mimedata.NewTextBytes(cb))
		ViewClipHistAdd(cb)
	}
	tv.SetCursorShow(org)
	tv.SavePosHistory(tv.CursorPos)
	return cut
}

// DeleteSelection deletes any selected text, without adding to clipboard --
// returns text deleted as textbuf.Edit (nil if none)
func (tv *View) DeleteSelection() *textbuf.Edit {
	tbe := tv.Buf.DeleteText(tv.SelectReg.Start, tv.SelectReg.End, EditSignal)
	tv.SelectReset()
	return tbe
}

// Copy copies any selected text to the clipboard, and returns that text,
// optionally resetting the current selection
func (tv *View) Copy(reset bool) *textbuf.Edit {
	tbe := tv.Selection()
	if tbe == nil {
		return nil
	}
	updt := tv.UpdateStart()
	defer tv.UpdateEndRender(updt)
	cb := tbe.ToBytes()
	ViewClipHistAdd(cb)
	tv.EventMgr().ClipBoard().Write(mimedata.NewTextBytes(cb))
	if reset {
		tv.SelectReset()
	}
	tv.SavePosHistory(tv.CursorPos)
	return tbe
}

// Paste inserts text from the clipboard at current cursor position
func (tv *View) Paste() {
	updt := tv.UpdateStart()
	defer tv.UpdateEndRender(updt)
	data := tv.EventMgr().ClipBoard().Read([]string{filecat.TextPlain})
	if data != nil {
		tv.InsertAtCursor(data.TypeData(filecat.TextPlain))
		tv.SavePosHistory(tv.CursorPos)
	}
}

// InsertAtCursor inserts given text at current cursor position
func (tv *View) InsertAtCursor(txt []byte) {
	updt := tv.UpdateStart()
	defer tv.UpdateEndRender(updt)

	if tv.HasSelection() {
		tbe := tv.DeleteSelection()
		tv.CursorPos = tbe.AdjustPos(tv.CursorPos, textbuf.AdjustPosDelStart) // move to start if in reg
	}
	tbe := tv.Buf.InsertText(tv.CursorPos, txt, EditSignal)
	if tbe == nil {
		return
	}
	pos := tbe.Reg.End
	if len(txt) == 1 && txt[0] == '\n' {
		pos.Ch = 0 // sometimes it doesn't go to the start..
	}
	tv.SetCursorShow(pos)
	tv.SetCursorCol(tv.CursorPos)
}

///////////////////////////////////////////////////////////
//  Rectangular regions

// ViewClipRect is the internal clipboard for Rect rectangle-based
// regions -- the raw text is posted on the system clipboard but the
// rect information is in a special format.
var ViewClipRect *textbuf.Edit

// CutRect cuts rectangle defined by selected text (upper left to lower right)
// and adds it to the clipboard, also returns cut text.
func (tv *View) CutRect() *textbuf.Edit {
	if !tv.HasSelection() {
		return nil
	}
	updt := tv.UpdateStart()
	defer tv.UpdateEndRender(updt)
	npos := lex.Pos{Ln: tv.SelectReg.End.Ln, Ch: tv.SelectReg.Start.Ch}
	cut := tv.Buf.DeleteTextRect(tv.SelectReg.Start, tv.SelectReg.End, EditSignal)
	if cut != nil {
		cb := cut.ToBytes()
		tv.EventMgr().ClipBoard().Write(mimedata.NewTextBytes(cb))
		ViewClipRect = cut
	}
	tv.SetCursorShow(npos)
	tv.SavePosHistory(tv.CursorPos)
	return cut
}

// CopyRect copies any selected text to the clipboard, and returns that text,
// optionally resetting the current selection
func (tv *View) CopyRect(reset bool) *textbuf.Edit {
	tbe := tv.Buf.RegionRect(tv.SelectReg.Start, tv.SelectReg.End)
	if tbe == nil {
		return nil
	}
	updt := tv.UpdateStart()
	defer tv.UpdateEndRender(updt)
	cb := tbe.ToBytes()
	tv.EventMgr().ClipBoard().Write(mimedata.NewTextBytes(cb))
	ViewClipRect = tbe
	if reset {
		tv.SelectReset()
	}
	tv.SavePosHistory(tv.CursorPos)
	return tbe
}

// PasteRect inserts text from the clipboard at current cursor position
func (tv *View) PasteRect() {
	if ViewClipRect == nil {
		return
	}
	updt := tv.UpdateStart()
	defer tv.UpdateEndRender(updt)
	ce := ViewClipRect.Clone()
	nl := ce.Reg.End.Ln - ce.Reg.Start.Ln
	nch := ce.Reg.End.Ch - ce.Reg.Start.Ch
	ce.Reg.Start.Ln = tv.CursorPos.Ln
	ce.Reg.End.Ln = tv.CursorPos.Ln + nl
	ce.Reg.Start.Ch = tv.CursorPos.Ch
	ce.Reg.End.Ch = tv.CursorPos.Ch + nch
	tbe := tv.Buf.InsertTextRect(ce, EditSignal)

	pos := tbe.Reg.End
	tv.SetCursorShow(pos)
	tv.SetCursorCol(tv.CursorPos)
	tv.SavePosHistory(tv.CursorPos)
}

// ReCaseSelection changes the case of the currently-selected text.
// Returns the new text -- empty if nothing selected.
func (tv *View) ReCaseSelection(c textbuf.Cases) string {
	if !tv.HasSelection() {
		return ""
	}
	sel := tv.Selection()
	nstr := textbuf.ReCaseString(string(sel.ToBytes()), c)
	tv.Buf.ReplaceText(sel.Reg.Start, sel.Reg.End, sel.Reg.Start, nstr, EditSignal, ReplaceNoMatchCase)
	return nstr
}

///////////////////////////////////////////////////////////
//  Context Menu

// ContextMenu displays the context menu with options dependent on situation
func (tv *View) ContextMenu() {
	if !tv.HasSelection() && tv.Buf.IsSpellEnabled(tv.CursorPos) {
		if tv.Buf.Spell != nil {
			if tv.OfferCorrect() {
				return
			}
		}
	}
	tv.WidgetBase.ContextMenu()
}

// ContextMenuPos returns the position of the context menu
func (tv *View) ContextMenuPos() (pos image.Point) {
	em := tv.EventMgr()
	_ = em
	// if em != nil {
	// 	return em.LastMousePos
	// }
	return image.Point{100, 100}
}

// MakeContextMenu builds the textview context menu
func (tv *View) MakeContextMenu(m *gi.Menu) {
	ac := m.AddButton(gi.ActOpts{Label: "Copy", ShortcutKey: gi.KeyFunCopy}, func(act *gi.Button) {
		tv.Copy(true)
	})
	ac.SetEnabledState(tv.HasSelection())
	if !tv.IsDisabled() {
		ac = m.AddButton(gi.ActOpts{Label: "Cut", ShortcutKey: gi.KeyFunCut}, func(act *gi.Button) {
			tv.Cut()
		})
		ac.SetEnabledState(tv.HasSelection())
		ac = m.AddButton(gi.ActOpts{Label: "Paste", ShortcutKey: gi.KeyFunPaste}, func(act *gi.Button) {
			tv.Paste()
		})
		ac.SetState(tv.EventMgr().ClipBoard().IsEmpty(), states.Disabled)
	} else {
		ac = m.AddButton(gi.ActOpts{Label: "Clear"}, func(act *gi.Button) {
			tv.Clear()
		})
	}
}
