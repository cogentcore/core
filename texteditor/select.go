// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package texteditor

import (
	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/fileinfo"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/keymap"
	"cogentcore.org/core/mimedata"
	"cogentcore.org/core/pi/lex"
	"cogentcore.org/core/states"
	"cogentcore.org/core/strcase"
	"cogentcore.org/core/texteditor/textbuf"
)

//////////////////////////////////////////////////////////
// 	Regions

// HighlightRegion creates a new highlighted region,
// triggers updating.
func (ed *Editor) HighlightRegion(reg textbuf.Region) {
	ed.Highlights = []textbuf.Region{reg}
	ed.NeedsRender()
}

// ClearHighlights clears the Highlights slice of all regions
func (ed *Editor) ClearHighlights() {
	if len(ed.Highlights) == 0 {
		return
	}
	ed.Highlights = ed.Highlights[:0]
	ed.NeedsRender()
}

// ClearScopelights clears the Highlights slice of all regions
func (ed *Editor) ClearScopelights() {
	if len(ed.Scopelights) == 0 {
		return
	}
	sl := make([]textbuf.Region, len(ed.Scopelights))
	copy(sl, ed.Scopelights)
	ed.Scopelights = ed.Scopelights[:0]
	ed.NeedsRender()
}

//////////////////////////////////////////////////////////
// 	Selection

// ClearSelected resets both the global selected flag and any current selection
func (ed *Editor) ClearSelected() {
	// ed.WidgetBase.ClearSelected()
	ed.SelectReset()
}

// HasSelection returns whether there is a selected region of text
func (ed *Editor) HasSelection() bool {
	return ed.SelectRegion.Start.IsLess(ed.SelectRegion.End)
}

// Selection returns the currently selected text as a textbuf.Edit, which
// captures start, end, and full lines in between -- nil if no selection
func (ed *Editor) Selection() *textbuf.Edit {
	if ed.HasSelection() {
		return ed.Buffer.Region(ed.SelectRegion.Start, ed.SelectRegion.End)
	}
	return nil
}

// SelectModeToggle toggles the SelectMode, updating selection with cursor movement
func (ed *Editor) SelectModeToggle() {
	if ed.SelectMode {
		ed.SelectMode = false
	} else {
		ed.SelectMode = true
		ed.SelectStart = ed.CursorPos
		ed.SelectRegUpdate(ed.CursorPos)
	}
	ed.SavePosHistory(ed.CursorPos)
}

// SelectAll selects all the text
func (ed *Editor) SelectAll() {
	ed.SelectRegion.Start = lex.PosZero
	ed.SelectRegion.End = ed.Buffer.EndPos()
	ed.NeedsRender()
}

// WordBefore returns the word before the lex.Pos
// uses IsWordBreak to determine the bounds of the word
func (ed *Editor) WordBefore(tp lex.Pos) *textbuf.Edit {
	txt := ed.Buffer.Line(tp.Ln)
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
		if core.IsWordBreak(r1, r2) {
			st = i + 1
			break
		}
	}
	if st != ch {
		return ed.Buffer.Region(lex.Pos{Ln: tp.Ln, Ch: st}, tp)
	}
	return nil
}

// IsWordStart returns true if the cursor is just before the start of a word
// word is a string of characters none of which are classified as a word break
func (ed *Editor) IsWordStart(tp lex.Pos) bool {
	txt := ed.Buffer.Line(ed.CursorPos.Ln)
	sz := len(txt)
	if sz == 0 {
		return false
	}
	if tp.Ch >= len(txt) { // end of line
		return false
	}
	if tp.Ch == 0 { // start of line
		r := txt[0]
		if core.IsWordBreak(r, rune(-1)) {
			return false
		}
		return true
	}
	r1 := txt[tp.Ch-1]
	r2 := txt[tp.Ch]
	return core.IsWordBreak(r1, rune(-1)) && !core.IsWordBreak(r2, rune(-1))
}

// IsWordEnd returns true if the cursor is just past the last letter of a word
// word is a string of characters none of which are classified as a word break
func (ed *Editor) IsWordEnd(tp lex.Pos) bool {
	txt := ed.Buffer.Line(ed.CursorPos.Ln)
	sz := len(txt)
	if sz == 0 {
		return false
	}
	if tp.Ch >= len(txt) { // end of line
		r := txt[len(txt)-1]
		if core.IsWordBreak(r, rune(-1)) {
			return true
		}
		return false
	}
	if tp.Ch == 0 { // start of line
		r := txt[0]
		if core.IsWordBreak(r, rune(-1)) {
			return false
		}
		return true
	}
	r1 := txt[tp.Ch-1]
	r2 := txt[tp.Ch]
	return !core.IsWordBreak(r1, rune(-1)) && core.IsWordBreak(r2, rune(-1))
}

// IsWordMiddle - returns true if the cursor is anywhere inside a word,
// i.e. the character before the cursor and the one after the cursor
// are not classified as word break characters
func (ed *Editor) IsWordMiddle(tp lex.Pos) bool {
	txt := ed.Buffer.Line(ed.CursorPos.Ln)
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
	return !core.IsWordBreak(r1, rune(-1)) && !core.IsWordBreak(r2, rune(-1))
}

// SelectWord selects the word (whitespace, punctuation delimited) that the cursor is on
// returns true if word selected
func (ed *Editor) SelectWord() bool {
	if ed.Buffer == nil {
		return false
	}
	txt := ed.Buffer.Line(ed.CursorPos.Ln)
	sz := len(txt)
	if sz == 0 {
		return false
	}
	reg := ed.WordAt()
	ed.SelectRegion = reg
	ed.SelectStart = ed.SelectRegion.Start
	return true
}

// WordAt finds the region of the word at the current cursor position
func (ed *Editor) WordAt() (reg textbuf.Region) {
	reg.Start = ed.CursorPos
	reg.End = ed.CursorPos
	txt := ed.Buffer.Line(ed.CursorPos.Ln)
	sz := len(txt)
	if sz == 0 {
		return reg
	}
	sch := min(ed.CursorPos.Ch, sz-1)
	if !core.IsWordBreak(txt[sch], rune(-1)) {
		for sch > 0 {
			r2 := rune(-1)
			if sch-2 >= 0 {
				r2 = txt[sch-2]
			}
			if core.IsWordBreak(txt[sch-1], r2) {
				break
			}
			sch--
		}
		reg.Start.Ch = sch
		ech := ed.CursorPos.Ch + 1
		for ech < sz {
			r2 := rune(-1)
			if ech < sz-1 {
				r2 = rune(txt[ech+1])
			}
			if core.IsWordBreak(txt[ech], r2) {
				break
			}
			ech++
		}
		reg.End.Ch = ech
	} else { // keep the space start -- go to next space..
		ech := ed.CursorPos.Ch + 1
		for ech < sz {
			if !core.IsWordBreak(txt[ech], rune(-1)) {
				break
			}
			ech++
		}
		for ech < sz {
			r2 := rune(-1)
			if ech < sz-1 {
				r2 = rune(txt[ech+1])
			}
			if core.IsWordBreak(txt[ech], r2) {
				break
			}
			ech++
		}
		reg.End.Ch = ech
	}
	return reg
}

// SelectReset resets the selection
func (ed *Editor) SelectReset() {
	ed.SelectMode = false
	if !ed.HasSelection() {
		return
	}
	ed.SelectRegion = textbuf.RegionNil
	ed.PreviousSelectRegion = textbuf.RegionNil
}

// RenderSelectLines renders the lines within the current selection region
func (ed *Editor) RenderSelectLines() {
	ed.PreviousSelectRegion = ed.SelectRegion
}

///////////////////////////////////////////////////////////////////////////////
//    Cut / Copy / Paste

// ViewClipHistory is the text view clipboard history -- everything that has been copied
var ViewClipHistory [][]byte

// ViewClipHistAdd adds the given clipboard bytes to top of history stack
func ViewClipHistAdd(clip []byte) {
	max := ClipHistMax
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
func (ed *Editor) PasteHist() {
	if ViewClipHistory == nil {
		return
	}
	cl := ViewClipHistChooseList()
	m := core.NewMenuFromStrings(cl, "", func(idx int) {
		clip := ViewClipHistory[idx]
		if clip != nil {
			ed.Clipboard().Write(mimedata.NewTextBytes(clip))
			ed.InsertAtCursor(clip)
			ed.SavePosHistory(ed.CursorPos)
			ed.NeedsRender()
		}
	})
	core.NewMenuStage(m, ed, ed.CursorBBox(ed.CursorPos).Min).Run()
}

// Cut cuts any selected text and adds it to the clipboard, also returns cut text
func (ed *Editor) Cut() *textbuf.Edit {
	if !ed.HasSelection() {
		return nil
	}
	org := ed.SelectRegion.Start
	cut := ed.DeleteSelection()
	if cut != nil {
		cb := cut.ToBytes()
		ed.Clipboard().Write(mimedata.NewTextBytes(cb))
		ViewClipHistAdd(cb)
	}
	ed.SetCursorShow(org)
	ed.SavePosHistory(ed.CursorPos)
	ed.NeedsRender()
	return cut
}

// DeleteSelection deletes any selected text, without adding to clipboard --
// returns text deleted as textbuf.Edit (nil if none)
func (ed *Editor) DeleteSelection() *textbuf.Edit {
	tbe := ed.Buffer.DeleteText(ed.SelectRegion.Start, ed.SelectRegion.End, EditSignal)
	ed.SelectReset()
	return tbe
}

// Copy copies any selected text to the clipboard, and returns that text,
// optionally resetting the current selection
func (ed *Editor) Copy(reset bool) *textbuf.Edit {
	tbe := ed.Selection()
	if tbe == nil {
		return nil
	}
	cb := tbe.ToBytes()
	ViewClipHistAdd(cb)
	ed.Clipboard().Write(mimedata.NewTextBytes(cb))
	if reset {
		ed.SelectReset()
	}
	ed.SavePosHistory(ed.CursorPos)
	ed.NeedsRender()
	return tbe
}

// Paste inserts text from the clipboard at current cursor position
func (ed *Editor) Paste() {
	data := ed.Clipboard().Read([]string{fileinfo.TextPlain})
	if data != nil {
		ed.InsertAtCursor(data.TypeData(fileinfo.TextPlain))
		ed.SavePosHistory(ed.CursorPos)
	}
	ed.NeedsRender()
}

// InsertAtCursor inserts given text at current cursor position
func (ed *Editor) InsertAtCursor(txt []byte) {
	if ed.HasSelection() {
		tbe := ed.DeleteSelection()
		ed.CursorPos = tbe.AdjustPos(ed.CursorPos, textbuf.AdjustPosDelStart) // move to start if in reg
	}
	tbe := ed.Buffer.InsertText(ed.CursorPos, txt, EditSignal)
	if tbe == nil {
		return
	}
	pos := tbe.Reg.End
	if len(txt) == 1 && txt[0] == '\n' {
		pos.Ch = 0 // sometimes it doesn't go to the start..
	}
	ed.SetCursorShow(pos)
	ed.SetCursorCol(ed.CursorPos)
	ed.NeedsRender()
}

///////////////////////////////////////////////////////////
//  Rectangular regions

// ViewClipRect is the internal clipboard for Rect rectangle-based
// regions -- the raw text is posted on the system clipboard but the
// rect information is in a special format.
var ViewClipRect *textbuf.Edit

// CutRect cuts rectangle defined by selected text (upper left to lower right)
// and adds it to the clipboard, also returns cut text.
func (ed *Editor) CutRect() *textbuf.Edit {
	if !ed.HasSelection() {
		return nil
	}
	npos := lex.Pos{Ln: ed.SelectRegion.End.Ln, Ch: ed.SelectRegion.Start.Ch}
	cut := ed.Buffer.DeleteTextRect(ed.SelectRegion.Start, ed.SelectRegion.End, EditSignal)
	if cut != nil {
		cb := cut.ToBytes()
		ed.Clipboard().Write(mimedata.NewTextBytes(cb))
		ViewClipRect = cut
	}
	ed.SetCursorShow(npos)
	ed.SavePosHistory(ed.CursorPos)
	ed.NeedsRender()
	return cut
}

// CopyRect copies any selected text to the clipboard, and returns that text,
// optionally resetting the current selection
func (ed *Editor) CopyRect(reset bool) *textbuf.Edit {
	tbe := ed.Buffer.RegionRect(ed.SelectRegion.Start, ed.SelectRegion.End)
	if tbe == nil {
		return nil
	}
	cb := tbe.ToBytes()
	ed.Clipboard().Write(mimedata.NewTextBytes(cb))
	ViewClipRect = tbe
	if reset {
		ed.SelectReset()
	}
	ed.SavePosHistory(ed.CursorPos)
	ed.NeedsRender()
	return tbe
}

// PasteRect inserts text from the clipboard at current cursor position
func (ed *Editor) PasteRect() {
	if ViewClipRect == nil {
		return
	}
	ce := ViewClipRect.Clone()
	nl := ce.Reg.End.Ln - ce.Reg.Start.Ln
	nch := ce.Reg.End.Ch - ce.Reg.Start.Ch
	ce.Reg.Start.Ln = ed.CursorPos.Ln
	ce.Reg.End.Ln = ed.CursorPos.Ln + nl
	ce.Reg.Start.Ch = ed.CursorPos.Ch
	ce.Reg.End.Ch = ed.CursorPos.Ch + nch
	tbe := ed.Buffer.InsertTextRect(ce, EditSignal)

	pos := tbe.Reg.End
	ed.SetCursorShow(pos)
	ed.SetCursorCol(ed.CursorPos)
	ed.SavePosHistory(ed.CursorPos)
	ed.NeedsRender()
}

// ReCaseSelection changes the case of the currently-selected text.
// Returns the new text -- empty if nothing selected.
func (ed *Editor) ReCaseSelection(c strcase.Cases) string {
	if !ed.HasSelection() {
		return ""
	}
	sel := ed.Selection()
	nstr := strcase.To(string(sel.ToBytes()), c)
	ed.Buffer.ReplaceText(sel.Reg.Start, sel.Reg.End, sel.Reg.Start, nstr, EditSignal, ReplaceNoMatchCase)
	return nstr
}

///////////////////////////////////////////////////////////
//  Context Menu

// ShowContextMenu displays the context menu with options dependent on situation
func (ed *Editor) ShowContextMenu(e events.Event) {
	if ed.Buffer.Spell != nil && !ed.HasSelection() && ed.Buffer.IsSpellEnabled(ed.CursorPos) {
		if ed.Buffer.Spell != nil {
			if ed.OfferCorrect() {
				return
			}
		}
	}
	ed.WidgetBase.ShowContextMenu(e)
}

// ContextMenu builds the text editor context menu
func (ed *Editor) ContextMenu(m *core.Scene) {
	core.NewButton(m).SetText("Copy").SetIcon(icons.ContentCopy).
		SetKey(keymap.Copy).SetState(!ed.HasSelection(), states.Disabled).
		OnClick(func(e events.Event) {
			ed.Copy(true)
		})
	if !ed.IsReadOnly() {
		core.NewButton(m).SetText("Cut").SetIcon(icons.ContentCopy).
			SetKey(keymap.Cut).SetState(!ed.HasSelection(), states.Disabled).
			OnClick(func(e events.Event) {
				ed.Cut()
			})
		core.NewButton(m).SetText("Paste").SetIcon(icons.ContentPaste).
			SetKey(keymap.Paste).SetState(ed.Clipboard().IsEmpty(), states.Disabled).
			OnClick(func(e events.Event) {
				ed.Paste()
			})
	} else {
		core.NewButton(m).SetText("Clear").SetIcon(icons.ClearAll).
			OnClick(func(e events.Event) {
				ed.Clear()
			})
	}
}
