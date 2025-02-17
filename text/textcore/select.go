// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package textcore

import (
	"cogentcore.org/core/base/fileinfo"
	"cogentcore.org/core/base/fileinfo/mimedata"
	"cogentcore.org/core/base/strcase"
	"cogentcore.org/core/core"
	"cogentcore.org/core/text/lines"
	"cogentcore.org/core/text/textpos"
)

//////// Regions

// HighlightRegion creates a new highlighted region,
// triggers updating.
func (ed *Base) HighlightRegion(reg textpos.Region) {
	ed.Highlights = []textpos.Region{reg}
	ed.NeedsRender()
}

// ClearHighlights clears the Highlights slice of all regions
func (ed *Base) ClearHighlights() {
	if len(ed.Highlights) == 0 {
		return
	}
	ed.Highlights = ed.Highlights[:0]
	ed.NeedsRender()
}

// clearScopelights clears the scopelights slice of all regions
func (ed *Base) clearScopelights() {
	if len(ed.scopelights) == 0 {
		return
	}
	sl := make([]textpos.Region, len(ed.scopelights))
	copy(sl, ed.scopelights)
	ed.scopelights = ed.scopelights[:0]
	ed.NeedsRender()
}

//////// Selection

// clearSelected resets both the global selected flag and any current selection
func (ed *Base) clearSelected() {
	// ed.WidgetBase.ClearSelected()
	ed.SelectReset()
}

// HasSelection returns whether there is a selected region of text
func (ed *Base) HasSelection() bool {
	return ed.SelectRegion.Start.IsLess(ed.SelectRegion.End)
}

// Selection returns the currently selected text as a textpos.Edit, which
// captures start, end, and full lines in between -- nil if no selection
func (ed *Base) Selection() *textpos.Edit {
	if ed.HasSelection() {
		return ed.Lines.Region(ed.SelectRegion.Start, ed.SelectRegion.End)
	}
	return nil
}

// selectModeToggle toggles the SelectMode, updating selection with cursor movement
func (ed *Base) selectModeToggle() {
	if ed.selectMode {
		ed.selectMode = false
	} else {
		ed.selectMode = true
		ed.selectStart = ed.CursorPos
		ed.selectRegionUpdate(ed.CursorPos)
	}
	ed.savePosHistory(ed.CursorPos)
}

// selectRegionUpdate updates current select region based on given cursor position
// relative to SelectStart position
func (ed *Base) selectRegionUpdate(pos textpos.Pos) {
	if pos.IsLess(ed.selectStart) {
		ed.SelectRegion.Start = pos
		ed.SelectRegion.End = ed.selectStart
	} else {
		ed.SelectRegion.Start = ed.selectStart
		ed.SelectRegion.End = pos
	}
}

// selectAll selects all the text
func (ed *Base) selectAll() {
	ed.SelectRegion.Start = textpos.PosZero
	ed.SelectRegion.End = ed.Lines.EndPos()
	ed.NeedsRender()
}

// isWordEnd returns true if the cursor is just past the last letter of a word
// word is a string of characters none of which are classified as a word break
func (ed *Base) isWordEnd(tp textpos.Pos) bool {
	return false
	// todo
	// txt := ed.Lines.Line(ed.CursorPos.Line)
	// sz := len(txt)
	// if sz == 0 {
	// 	return false
	// }
	// if tp.Char >= len(txt) { // end of line
	// 	r := txt[len(txt)-1]
	// 	return core.IsWordBreak(r, -1)
	// }
	// if tp.Char == 0 { // start of line
	// 	r := txt[0]
	// 	return !core.IsWordBreak(r, -1)
	// }
	// r1 := txt[tp.Ch-1]
	// r2 := txt[tp.Ch]
	// return !core.IsWordBreak(r1, rune(-1)) && core.IsWordBreak(r2, rune(-1))
}

// isWordMiddle - returns true if the cursor is anywhere inside a word,
// i.e. the character before the cursor and the one after the cursor
// are not classified as word break characters
func (ed *Base) isWordMiddle(tp textpos.Pos) bool {
	return false
	// todo:
	// txt := ed.Lines.Line(ed.CursorPos.Line)
	// sz := len(txt)
	// if sz < 2 {
	// 	return false
	// }
	// if tp.Char >= len(txt) { // end of line
	// 	return false
	// }
	// if tp.Char == 0 { // start of line
	// 	return false
	// }
	// r1 := txt[tp.Ch-1]
	// r2 := txt[tp.Ch]
	// return !core.IsWordBreak(r1, rune(-1)) && !core.IsWordBreak(r2, rune(-1))
}

// selectWord selects the word (whitespace, punctuation delimited) that the cursor is on.
// returns true if word selected
func (ed *Base) selectWord() bool {
	// if ed.Lines == nil {
	// 	return false
	// }
	// txt := ed.Lines.Line(ed.CursorPos.Line)
	// sz := len(txt)
	// if sz == 0 {
	// 	return false
	// }
	// reg := ed.wordAt()
	// ed.SelectRegion = reg
	// ed.selectStart = ed.SelectRegion.Start
	return true
}

// wordAt finds the region of the word at the current cursor position
func (ed *Base) wordAt() (reg textpos.Region) {
	return textpos.Region{}
	// reg.Start = ed.CursorPos
	// reg.End = ed.CursorPos
	// txt := ed.Lines.Line(ed.CursorPos.Line)
	// sz := len(txt)
	// if sz == 0 {
	// 	return reg
	// }
	// sch := min(ed.CursorPos.Ch, sz-1)
	// if !core.IsWordBreak(txt[sch], rune(-1)) {
	// 	for sch > 0 {
	// 		r2 := rune(-1)
	// 		if sch-2 >= 0 {
	// 			r2 = txt[sch-2]
	// 		}
	// 		if core.IsWordBreak(txt[sch-1], r2) {
	// 			break
	// 		}
	// 		sch--
	// 	}
	// 	reg.Start.Char = sch
	// 	ech := ed.CursorPos.Char + 1
	// 	for ech < sz {
	// 		r2 := rune(-1)
	// 		if ech < sz-1 {
	// 			r2 = rune(txt[ech+1])
	// 		}
	// 		if core.IsWordBreak(txt[ech], r2) {
	// 			break
	// 		}
	// 		ech++
	// 	}
	// 	reg.End.Char = ech
	// } else { // keep the space start -- go to next space..
	// 	ech := ed.CursorPos.Char + 1
	// 	for ech < sz {
	// 		if !core.IsWordBreak(txt[ech], rune(-1)) {
	// 			break
	// 		}
	// 		ech++
	// 	}
	// 	for ech < sz {
	// 		r2 := rune(-1)
	// 		if ech < sz-1 {
	// 			r2 = rune(txt[ech+1])
	// 		}
	// 		if core.IsWordBreak(txt[ech], r2) {
	// 			break
	// 		}
	// 		ech++
	// 	}
	// 	reg.End.Char = ech
	// }
	// return reg
}

// SelectReset resets the selection
func (ed *Base) SelectReset() {
	ed.selectMode = false
	if !ed.HasSelection() {
		return
	}
	ed.SelectRegion = textpos.Region{}
	ed.previousSelectRegion = textpos.Region{}
}

////////    Cut / Copy / Paste

// editorClipboardHistory is the [Base] clipboard history; everything that has been copied
var editorClipboardHistory [][]byte

// addBaseClipboardHistory adds the given clipboard bytes to top of history stack
func addBaseClipboardHistory(clip []byte) {
	max := clipboardHistoryMax
	if editorClipboardHistory == nil {
		editorClipboardHistory = make([][]byte, 0, max)
	}

	ch := &editorClipboardHistory

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

// editorClipHistoryChooserLength is the max length of clip history to show in chooser
var editorClipHistoryChooserLength = 40

// editorClipHistoryChooserList returns a string slice of length-limited clip history, for chooser
func editorClipHistoryChooserList() []string {
	cl := make([]string, len(editorClipboardHistory))
	for i, hc := range editorClipboardHistory {
		szl := len(hc)
		if szl > editorClipHistoryChooserLength {
			cl[i] = string(hc[:editorClipHistoryChooserLength])
		} else {
			cl[i] = string(hc)
		}
	}
	return cl
}

// pasteHistory presents a chooser of clip history items, pastes into text if selected
func (ed *Base) pasteHistory() {
	if editorClipboardHistory == nil {
		return
	}
	cl := editorClipHistoryChooserList()
	m := core.NewMenuFromStrings(cl, "", func(idx int) {
		clip := editorClipboardHistory[idx]
		if clip != nil {
			ed.Clipboard().Write(mimedata.NewTextBytes(clip))
			ed.InsertAtCursor(clip)
			ed.savePosHistory(ed.CursorPos)
			ed.NeedsRender()
		}
	})
	core.NewMenuStage(m, ed, ed.cursorBBox(ed.CursorPos).Min).Run()
}

// Cut cuts any selected text and adds it to the clipboard, also returns cut text
func (ed *Base) Cut() *textpos.Edit {
	if !ed.HasSelection() {
		return nil
	}
	org := ed.SelectRegion.Start
	cut := ed.deleteSelection()
	if cut != nil {
		cb := cut.ToBytes()
		ed.Clipboard().Write(mimedata.NewTextBytes(cb))
		addBaseClipboardHistory(cb)
	}
	ed.SetCursorShow(org)
	ed.savePosHistory(ed.CursorPos)
	ed.NeedsRender()
	return cut
}

// deleteSelection deletes any selected text, without adding to clipboard --
// returns text deleted as textpos.Edit (nil if none)
func (ed *Base) deleteSelection() *textpos.Edit {
	tbe := ed.Lines.DeleteText(ed.SelectRegion.Start, ed.SelectRegion.End)
	ed.SelectReset()
	return tbe
}

// Copy copies any selected text to the clipboard, and returns that text,
// optionally resetting the current selection
func (ed *Base) Copy(reset bool) *textpos.Edit {
	tbe := ed.Selection()
	if tbe == nil {
		return nil
	}
	cb := tbe.ToBytes()
	addBaseClipboardHistory(cb)
	ed.Clipboard().Write(mimedata.NewTextBytes(cb))
	if reset {
		ed.SelectReset()
	}
	ed.savePosHistory(ed.CursorPos)
	ed.NeedsRender()
	return tbe
}

// Paste inserts text from the clipboard at current cursor position
func (ed *Base) Paste() {
	data := ed.Clipboard().Read([]string{fileinfo.TextPlain})
	if data != nil {
		ed.InsertAtCursor(data.TypeData(fileinfo.TextPlain))
		ed.savePosHistory(ed.CursorPos)
	}
	ed.NeedsRender()
}

// InsertAtCursor inserts given text at current cursor position
func (ed *Base) InsertAtCursor(txt []byte) {
	if ed.HasSelection() {
		tbe := ed.deleteSelection()
		ed.CursorPos = tbe.AdjustPos(ed.CursorPos, textpos.AdjustPosDelStart) // move to start if in reg
	}
	tbe := ed.Lines.InsertText(ed.CursorPos, []rune(string(txt)))
	if tbe == nil {
		return
	}
	pos := tbe.Region.End
	if len(txt) == 1 && txt[0] == '\n' {
		pos.Char = 0 // sometimes it doesn't go to the start..
	}
	ed.SetCursorShow(pos)
	ed.setCursorColumn(ed.CursorPos)
	ed.NeedsRender()
}

////////  Rectangular regions

// editorClipboardRect is the internal clipboard for Rect rectangle-based
// regions -- the raw text is posted on the system clipboard but the
// rect information is in a special format.
var editorClipboardRect *textpos.Edit

// CutRect cuts rectangle defined by selected text (upper left to lower right)
// and adds it to the clipboard, also returns cut lines.
func (ed *Base) CutRect() *textpos.Edit {
	if !ed.HasSelection() {
		return nil
	}
	npos := textpos.Pos{Line: ed.SelectRegion.End.Line, Char: ed.SelectRegion.Start.Char}
	cut := ed.Lines.DeleteTextRect(ed.SelectRegion.Start, ed.SelectRegion.End)
	if cut != nil {
		cb := cut.ToBytes()
		ed.Clipboard().Write(mimedata.NewTextBytes(cb))
		editorClipboardRect = cut
	}
	ed.SetCursorShow(npos)
	ed.savePosHistory(ed.CursorPos)
	ed.NeedsRender()
	return cut
}

// CopyRect copies any selected text to the clipboard, and returns that text,
// optionally resetting the current selection
func (ed *Base) CopyRect(reset bool) *textpos.Edit {
	tbe := ed.Lines.RegionRect(ed.SelectRegion.Start, ed.SelectRegion.End)
	if tbe == nil {
		return nil
	}
	cb := tbe.ToBytes()
	ed.Clipboard().Write(mimedata.NewTextBytes(cb))
	editorClipboardRect = tbe
	if reset {
		ed.SelectReset()
	}
	ed.savePosHistory(ed.CursorPos)
	ed.NeedsRender()
	return tbe
}

// PasteRect inserts text from the clipboard at current cursor position
func (ed *Base) PasteRect() {
	if editorClipboardRect == nil {
		return
	}
	ce := editorClipboardRect.Clone()
	nl := ce.Region.End.Line - ce.Region.Start.Line
	nch := ce.Region.End.Char - ce.Region.Start.Char
	ce.Region.Start.Line = ed.CursorPos.Line
	ce.Region.End.Line = ed.CursorPos.Line + nl
	ce.Region.Start.Char = ed.CursorPos.Char
	ce.Region.End.Char = ed.CursorPos.Char + nch
	tbe := ed.Lines.InsertTextRect(ce)

	pos := tbe.Region.End
	ed.SetCursorShow(pos)
	ed.setCursorColumn(ed.CursorPos)
	ed.savePosHistory(ed.CursorPos)
	ed.NeedsRender()
}

// ReCaseSelection changes the case of the currently selected lines.
// Returns the new text; empty if nothing selected.
func (ed *Base) ReCaseSelection(c strcase.Cases) string {
	if !ed.HasSelection() {
		return ""
	}
	sel := ed.Selection()
	nstr := strcase.To(string(sel.ToBytes()), c)
	ed.Lines.ReplaceText(sel.Region.Start, sel.Region.End, sel.Region.Start, nstr, lines.ReplaceNoMatchCase)
	return nstr
}
