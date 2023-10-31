// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package texteditor

//go:generate goki generate

import (
	"image"
	"sync"

	"goki.dev/colors"
	"goki.dev/cursors"
	"goki.dev/enums"
	"goki.dev/gi/v2/gi"
	"goki.dev/gi/v2/texteditor/histyle"
	"goki.dev/gi/v2/texteditor/textbuf"
	"goki.dev/girl/abilities"
	"goki.dev/girl/paint"
	"goki.dev/girl/states"
	"goki.dev/girl/styles"
	"goki.dev/girl/units"
	"goki.dev/ki/v2"
	"goki.dev/mat32/v2"
	"goki.dev/pi/v2/lex"
)

// todo: make it a Layout and handle all the scrollbar stuff internally!
// could also see about wrapping in a Scene in gide and benchmark that.

// Editor is a widget for editing multiple lines of text (as compared to
// [gi.TextField] for a single line).  The Editor is driven by a Buf buffer which
// contains all the text, and manages all the edits, sending update signals
// out to the views -- multiple views can be attached to a given buffer.  All
// updating in the Editor should be within a single goroutine -- it would
// require extensive protections throughout code otherwise.
type Editor struct { //goki:embedder
	gi.Layout

	// the text buffer that we're editing
	Buf *Buf `set:"-" json:"-" xml:"-"`

	// text that is displayed when the field is empty, in a lower-contrast manner
	Placeholder string `json:"-" xml:"placeholder"`

	// width of cursor -- set from cursor-width property (inherited)
	CursorWidth units.Value `xml:"cursor-width"`

	// the color used for the side bar containing the line numbers; this should be set in Stylers like all other style properties
	LineNumberColor colors.Full

	// the color used for the user text selection background color; this should be set in Stylers like all other style properties
	SelectColor colors.Full

	// the color used for the text highlight background color (like in find); this should be set in Stylers like all other style properties
	HighlightColor colors.Full

	// the color used for the text field cursor (caret); this should be set in Stylers like all other style properties
	CursorColor colors.Full

	// number of lines in the view -- sync'd with the Buf after edits, but always reflects storage size of Renders etc
	NLines int `set:"-" view:"-" json:"-" xml:"-"`

	// renders of the text lines, with one render per line (each line could visibly wrap-around, so these are logical lines, not display lines)
	Renders []paint.Text `set:"-" view:"-" json:"-" xml:"-"`

	// starting render offsets for top of each line
	Offs []float32 `set:"-" view:"-" json:"-" xml:"-"`

	// number of line number digits needed
	LineNoDigs int `set:"-" view:"-" json:"-" xml:"-"`

	// horizontal offset for start of text after line numbers
	LineNoOff float32 `set:"-" view:"-" json:"-" xml:"-"`

	// render for line numbers
	LineNoRender paint.Text `set:"-" view:"-" json:"-" xml:"-"`

	// current cursor position
	CursorPos lex.Pos `set:"-" edit:"-" json:"-" xml:"-"`

	// desired cursor column -- where the cursor was last when moved using left / right arrows -- used when doing up / down to not always go to short line columns
	CursorCol int `set:"-" edit:"-" json:"-" xml:"-"`

	// if true, scroll screen to cursor on next render
	ScrollToCursorOnRender bool `set:"-" edit:"-" json:"-" xml:"-"`

	// cursor position to scroll to
	ScrollToCursorPos lex.Pos `set:"-" edit:"-" json:"-" xml:"-"`

	// current index within PosHistory
	PosHistIdx int `set:"-" edit:"-" json:"-" xml:"-"`

	// starting point for selection -- will either be the start or end of selected region depending on subsequent selection.
	SelectStart lex.Pos `set:"-" edit:"-" json:"-" xml:"-"`

	// current selection region
	SelectReg textbuf.Region `set:"-" edit:"-" json:"-" xml:"-"`

	// previous selection region, that was actually rendered -- needed to update render
	PrevSelectReg textbuf.Region `set:"-" edit:"-" json:"-" xml:"-"`

	// highlighted regions, e.g., for search results
	Highlights []textbuf.Region `set:"-" edit:"-" json:"-" xml:"-"`

	// highlighted regions, specific to scope markers
	Scopelights []textbuf.Region `set:"-" edit:"-" json:"-" xml:"-"`

	// if true, select text as cursor moves
	SelectMode bool `set:"-" edit:"-" json:"-" xml:"-"`

	// if true, complete regardless of any disqualifying reasons
	ForceComplete bool `set:"-" edit:"-" json:"-" xml:"-"`

	// interactive search data
	ISearch ISearch `set:"-" edit:"-" json:"-" xml:"-"`

	// query replace data
	QReplace QReplace `set:"-" edit:"-" json:"-" xml:"-"`

	// font height, cached during styling
	FontHeight float32 `set:"-" edit:"-" json:"-" xml:"-"`

	// line height, cached during styling
	LineHeight float32 `set:"-" edit:"-" json:"-" xml:"-"`

	// height in lines and width in chars of the visible area
	NLinesChars image.Point `set:"-" edit:"-" json:"-" xml:"-"`

	// total size of all lines as rendered
	LinesSize mat32.Vec2 `set:"-" edit:"-" json:"-" xml:"-"`

	// TotalSize = LinesSize plus extra space and line numbers etc
	TotalSize mat32.Vec2 `set:"-" edit:"-" json:"-" xml:"-"`

	// LineLayoutSize is LayState.Alloc.Size subtracting
	// extra space and line numbers -- this is what
	// LayoutStdLR sees for laying out each line
	LineLayoutSize mat32.Vec2 `set:"-" edit:"-" json:"-" xml:"-"`

	// oscillates between on and off for blinking
	BlinkOn bool `set:"-" edit:"-" json:"-" xml:"-"`

	// mutex protecting cursor rendering -- shared between blink and main code
	CursorMu sync.Mutex `set:"-" json:"-" xml:"-" view:"-"`

	// at least one of the renders has links -- determines if we set the cursor for hand movements
	HasLinks bool `set:"-" edit:"-" json:"-" xml:"-"`

	lastRecenter   int
	lastAutoInsert rune
	lastFilename   gi.FileName
}

func (ed *Editor) FlagType() enums.BitFlag {
	return EditorFlags(ed.Flags)
}

// NewViewLayout adds a new layout with text editor
// to given parent node, with given name.  Layout adds "-lay" suffix.
// Texediew should always have a parent Layout to manage
// the scrollbars.
func NewViewLayout(parent ki.Ki, name string) (*Editor, *gi.Layout) {
	ly := parent.NewChild(gi.LayoutType, name+"-lay").(*gi.Layout)
	ed := NewEditor(ly, name)
	return ed, ly
}

func (ed *Editor) OnInit() {
	ed.HandleTextViewEvents()
	ed.ViewStyles()
}

func (ed *Editor) ViewStyles() {
	ed.Style(func(s *styles.Style) {
		s.SetAbilities(true, abilities.Activatable, abilities.Focusable, abilities.Hoverable, abilities.Slideable)
		ed.CursorWidth.Dp(1)
		ed.LineNumberColor.SetSolid(colors.Scheme.SurfaceContainer)
		ed.SelectColor.SetSolid(colors.Scheme.Select.Container)
		ed.HighlightColor.SetSolid(colors.Orange)
		ed.CursorColor.SetSolid(colors.Scheme.Primary.Base)

		s.Cursor = cursors.Text
		if gi.Prefs.Editor.WordWrap {
			s.Text.WhiteSpace = styles.WhiteSpacePreWrap
		} else {
			s.Text.WhiteSpace = styles.WhiteSpacePre
		}
		s.Border.Style.Set(styles.BorderNone) // don't render our own border
		s.Border.Radius = styles.BorderRadiusLarge
		s.Margin.Set()
		s.Padding.Set(units.Dp(4))
		s.AlignV = styles.AlignTop
		s.Text.Align = styles.AlignLeft
		s.Text.TabSize = 4
		s.Color = colors.Scheme.OnSurface

		if s.State.Is(states.Focused) {
			s.BackgroundColor.SetSolid(colors.Scheme.Surface)
		} else {
			s.BackgroundColor.SetSolid(colors.Scheme.SurfaceContainerHigh)
		}
	})
}

// EditorFlags extend WidgetFlags to hold [Editor] state
type EditorFlags gi.WidgetFlags //enums:bitflag -trim-prefix View

const (
	// EditorHasLineNos indicates that this editor has line numbers (per Buf option)
	EditorHasLineNos EditorFlags = EditorFlags(gi.WidgetFlagsN) + iota

	// EditorLastWasTabAI indicates that last key was a Tab auto-indent
	EditorLastWasTabAI

	// EditorLastWasUndo indicates that last key was an undo
	EditorLastWasUndo
)

// EditDone completes editing and copies the active edited text to the text --
// called when the return key is pressed or goes out of focus
func (ed *Editor) EditDone() {
	if ed.Buf != nil {
		ed.Buf.EditDone()
	}
	ed.ClearSelected()
}

// Remarkup triggers a complete re-markup of the entire text --
// can do this when needed if the markup gets off due to multi-line
// formatting issues -- via Recenter key
func (ed *Editor) ReMarkup() {
	if ed.Buf == nil {
		return
	}
	ed.Buf.ReMarkup()
}

// IsChanged returns true if buffer was changed (edited)
func (ed *Editor) IsChanged() bool {
	if ed.Buf != nil && ed.Buf.IsChanged() {
		return true
	}
	return false
}

// HasLineNos returns true if view is showing line numbers (per textbuf option, cached here)
func (ed *Editor) HasLineNos() bool {
	return ed.Is(EditorHasLineNos)
}

// Clear resets all the text in the buffer for this view
func (ed *Editor) Clear() {
	if ed.Buf == nil {
		return
	}
	ed.Buf.NewBuf(0)
}

///////////////////////////////////////////////////////////////////////////////
//  Buffer communication

// ResetState resets all the random state variables, when opening a new buffer etc
func (ed *Editor) ResetState() {
	ed.SelectReset()
	ed.Highlights = nil
	ed.ISearch.On = false
	ed.QReplace.On = false
	if ed.Buf == nil || ed.lastFilename != ed.Buf.Filename { // don't reset if reopening..
		ed.CursorPos = lex.Pos{}
	}
	if ed.Buf != nil {
		ed.Buf.SetReadOnly(ed.IsReadOnly())
	}
}

// SetBuf sets the Buf that this is a view of, and interconnects their signals
func (ed *Editor) SetBuf(buf *Buf) *Editor {
	if buf != nil && ed.Buf == buf {
		return ed
	}
	// had := false
	if ed.Buf != nil {
		// had = true
		ed.Buf.DeleteView(ed)
	}
	ed.Buf = buf
	ed.ResetState()
	if buf != nil {
		buf.AddView(ed)
		bhl := len(buf.PosHistory)
		if bhl > 0 {
			ed.CursorPos = buf.PosHistory[bhl-1]
			ed.PosHistIdx = bhl - 1
		}
	}
	ed.SetNeedsLayout()
	return ed
}

// LinesInserted inserts new lines of text and reformats them
func (ed *Editor) LinesInserted(tbe *textbuf.Edit) {
	stln := tbe.Reg.Start.Ln + 1
	nsz := (tbe.Reg.End.Ln - tbe.Reg.Start.Ln)
	if stln > len(ed.Renders) { // invalid
		return
	}

	// Renders
	tmprn := make([]paint.Text, nsz)
	nrn := append(ed.Renders, tmprn...)
	copy(nrn[stln+nsz:], nrn[stln:])
	copy(nrn[stln:], tmprn)
	ed.Renders = nrn

	// Offs
	tmpof := make([]float32, nsz)
	nof := append(ed.Offs, tmpof...)
	copy(nof[stln+nsz:], nof[stln:])
	copy(nof[stln:], tmpof)
	ed.Offs = nof

	ed.NLines += nsz
	ed.SetNeedsLayout()
}

// LinesDeleted deletes lines of text and reformats remaining one
func (ed *Editor) LinesDeleted(tbe *textbuf.Edit) {
	stln := tbe.Reg.Start.Ln
	edln := tbe.Reg.End.Ln
	dsz := edln - stln

	ed.Renders = append(ed.Renders[:stln], ed.Renders[edln:]...)
	ed.Offs = append(ed.Offs[:stln], ed.Offs[edln:]...)

	ed.NLines -= dsz
	ed.SetNeedsLayout()
}

// BufSignal receives a signal from the Buf when underlying text
// is changed.
func (ed *Editor) BufSignal(sig BufSignals, tbe *textbuf.Edit) {
	switch sig {
	case BufDone:
	case BufNew:
		ed.ResetState()
		ed.SetNeedsLayout()
		ed.SetCursorShow(ed.CursorPos)
	case BufMods:
		ed.SetNeedsLayout()
	case BufInsert:
		if ed.Renders == nil || !ed.This().(gi.Widget).IsVisible() {
			return
		}
		// fmt.Printf("ed %v got %v\n", ed.Nm, tbe.Reg.Start)
		if tbe.Reg.Start.Ln != tbe.Reg.End.Ln {
			// fmt.Printf("ed %v lines insert %v - %v\n", ed.Nm, tbe.Reg.Start, tbe.Reg.End)
			ed.LinesInserted(tbe) // triggers full layout
		} else {
			ed.LayoutLine(tbe.Reg.Start.Ln) // triggers layout if line width exceeds
		}
	case BufDelete:
		if ed.Renders == nil || !ed.This().(gi.Widget).IsVisible() {
			return
		}
		if tbe.Reg.Start.Ln != tbe.Reg.End.Ln {
			ed.LinesDeleted(tbe) // triggers full layout
		} else {
			ed.LayoutLine(tbe.Reg.Start.Ln)
		}
	case BufMarkUpdt:
		ed.SetNeedsLayout() // comes from another goroutine
	case BufClosed:
		ed.SetBuf(nil)
	}
}

///////////////////////////////////////////////////////////////////////////////
//    Undo / Redo

// Undo undoes previous action
func (ed *Editor) Undo() {
	updt := ed.UpdateStart()
	defer ed.UpdateEndRender(updt)

	tbe := ed.Buf.Undo()
	if tbe != nil {
		if tbe.Delete { // now an insert
			ed.SetCursorShow(tbe.Reg.End)
		} else {
			ed.SetCursorShow(tbe.Reg.Start)
		}
	} else {
		ed.CursorMovedSig() // updates status..
		ed.ScrollCursorToCenterIfHidden()
	}
	ed.SavePosHistory(ed.CursorPos)
}

// Redo redoes previously undone action
func (ed *Editor) Redo() {
	updt := ed.UpdateStart()
	defer ed.UpdateEndRender(updt)

	tbe := ed.Buf.Redo()
	if tbe != nil {
		if tbe.Delete {
			ed.SetCursorShow(tbe.Reg.Start)
		} else {
			ed.SetCursorShow(tbe.Reg.End)
		}
	} else {
		ed.ScrollCursorToCenterIfHidden()
	}
	ed.SavePosHistory(ed.CursorPos)
}

////////////////////////////////////////////////////
//  Widget Interface

// Config calls Init on widget
// func (ed *View) ConfigWidget(vp *gi.Scene) {
//
// }

// StyleView sets the style of widget
func (ed *Editor) StyleView(sc *gi.Scene) {
	ed.StyMu.Lock()
	defer ed.StyMu.Unlock()

	if ed.NeedsRebuild() {
		if ed.Buf != nil {
			ed.Buf.SetHiStyle(histyle.StyleDefault)
		}
	}
	ed.ApplyStyleWidget(sc)
	ed.CursorWidth.ToDots(&ed.Styles.UnContext)
}

// ApplyStyle calls StyleView and sets the style
func (ed *Editor) ApplyStyle(sc *gi.Scene) {
	// ed.SetFlag(true, gi.CanFocus) // always focusable
	ed.StyleView(sc)
	ed.StyleSizes()
}

// todo: virtual keyboard stuff

// FocusChanged appropriate actions for various types of focus changes
// func (ed *View) FocusChanged(change gi.FocusChanges) {
// 	switch change {
// 	case gi.FocusLost:
// 		ed.SetFlag(false, ViewFocusActive))
// 		// ed.EditDone()
// 		ed.StopCursor() // make sure no cursor
// 		ed.SetNeedsRender()
// 		goosi.TheApp.HideVirtualKeyboard()
// 		// fmt.Printf("lost focus: %v\n", ed.Nm)
// 	case gi.FocusGot:
// 		ed.SetFlag(true, ViewFocusActive))
// 		ed.EmitFocusedSignal()
// 		ed.SetNeedsRender()
// 		goosi.TheApp.ShowVirtualKeyboard(goosi.DefaultKeyboard)
// 		// fmt.Printf("got focus: %v\n", ed.Nm)
// 	case gi.FocusInactive:
// 		ed.SetFlag(false, ViewFocusActive))
// 		ed.StopCursor()
// 		// ed.EditDone()
// 		// ed.SetNeedsRender()
// 		goosi.TheApp.HideVirtualKeyboard()
// 		// fmt.Printf("focus inactive: %v\n", ed.Nm)
// 	case gi.FocusActive:
// 		// fmt.Printf("focus active: %v\n", ed.Nm)
// 		ed.SetFlag(true, ViewFocusActive))
// 		// ed.SetNeedsRender()
// 		// todo: see about cursor
// 		ed.StartCursor()
// 		goosi.TheApp.ShowVirtualKeyboard(goosi.DefaultKeyboard)
// 	}
// }
