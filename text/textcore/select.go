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

// HighlightRegion adds a new highlighted region. Use HighlightsReset to
// clear any existing prior to this if only one region desired.
func (ed *Base) HighlightRegion(reg textpos.Region) {
	ed.Highlights = append(ed.Highlights, reg)
	ed.NeedsRender()
}

// HighlightsReset resets the list of all highlighted regions.
func (ed *Base) HighlightsReset() {
	if len(ed.Highlights) == 0 {
		return
	}
	ed.Highlights = ed.Highlights[:0]
	ed.NeedsRender()
}

// scopelightsReset clears the scopelights slice of all regions
func (ed *Base) scopelightsReset() {
	if len(ed.scopelights) == 0 {
		return
	}
	sl := make([]textpos.Region, len(ed.scopelights))
	copy(sl, ed.scopelights)
	ed.scopelights = ed.scopelights[:0]
	ed.NeedsRender()
}

func (ed *Base) addScopelights(st, end textpos.Pos) {
	ed.scopelights = append(ed.scopelights, textpos.NewRegionPos(st, textpos.Pos{st.Line, st.Char + 1}))
	ed.scopelights = append(ed.scopelights, textpos.NewRegionPos(end, textpos.Pos{end.Line, end.Char + 1}))
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

// validateSelection ensures that the selection region is still valid.
func (ed *Base) validateSelection() {
	ed.SelectRegion.Start = ed.Lines.ValidPos(ed.SelectRegion.Start)
	ed.SelectRegion.End = ed.Lines.ValidPos(ed.SelectRegion.End)
}

// Selection returns the currently selected text as a textpos.Edit, which
// captures start, end, and full lines in between -- nil if no selection
func (ed *Base) Selection() *textpos.Edit {
	if ed.HasSelection() {
		ed.validateSelection()
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

// selectWord selects the word (whitespace, punctuation delimited) that the cursor is on.
// returns true if word selected
func (ed *Base) selectWord() bool {
	if ed.Lines == nil {
		return false
	}
	reg := ed.Lines.WordAt(ed.CursorPos)
	ed.SelectRegion = reg
	ed.selectStart = ed.SelectRegion.Start
	return true
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

////////    Undo / Redo

// undo undoes previous action
func (ed *Base) undo() {
	tbes := ed.Lines.Undo()
	if tbes != nil {
		tbe := tbes[len(tbes)-1]
		if tbe.Delete { // now an insert
			ed.SetCursorShow(tbe.Region.End)
		} else {
			ed.SetCursorShow(tbe.Region.Start)
		}
	} else {
		ed.SendInput() // updates status..
		ed.scrollCursorToCenterIfHidden()
	}
	ed.savePosHistory(ed.CursorPos)
	ed.NeedsRender()
}

// redo redoes previously undone action
func (ed *Base) redo() {
	tbes := ed.Lines.Redo()
	if tbes != nil {
		tbe := tbes[len(tbes)-1]
		if tbe.Delete {
			ed.SetCursorShow(tbe.Region.Start)
		} else {
			ed.SetCursorShow(tbe.Region.End)
		}
	} else {
		ed.scrollCursorToCenterIfHidden()
	}
	ed.savePosHistory(ed.CursorPos)
	ed.NeedsRender()
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
	ed.validateSelection()
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
	ed.validateSelection()
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
	cp := ed.validateCursor()
	tbe := ed.Lines.InsertText(cp, []rune(string(txt)))
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
	ed.validateSelection()
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
	ed.validateSelection()
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
