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
//
//goki:embedder
type Editor struct {
	gi.Layout

	// the text buffer that we're editing
	Buf *Buf `json:"-" xml:"-"`

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
	NLines int `json:"-" xml:"-"`

	// renders of the text lines, with one render per line (each line could visibly wrap-around, so these are logical lines, not display lines)
	Renders []paint.Text `json:"-" xml:"-"`

	// starting render offsets for top of each line
	Offs []float32 `json:"-" xml:"-"`

	// number of line number digits needed
	LineNoDigs int `json:"-" xml:"-"`

	// horizontal offset for start of text after line numbers
	LineNoOff float32 `json:"-" xml:"-"`

	// render for line numbers
	LineNoRender paint.Text `json:"-" xml:"-"`

	// current cursor position
	CursorPos lex.Pos `json:"-" xml:"-"`

	// desired cursor column -- where the cursor was last when moved using left / right arrows -- used when doing up / down to not always go to short line columns
	CursorCol int `json:"-" xml:"-"`

	// if true, scroll screen to cursor on next render
	ScrollToCursorOnRender bool `json:"-" xml:"-"`

	// cursor position to scroll to
	ScrollToCursorPos lex.Pos `json:"-" xml:"-"`

	// current index within PosHistory
	PosHistIdx int `json:"-" xml:"-"`

	// starting point for selection -- will either be the start or end of selected region depending on subsequent selection.
	SelectStart lex.Pos `json:"-" xml:"-"`

	// current selection region
	SelectReg textbuf.Region `json:"-" xml:"-"`

	// previous selection region, that was actually rendered -- needed to update render
	PrevSelectReg textbuf.Region `json:"-" xml:"-"`

	// highlighted regions, e.g., for search results
	Highlights []textbuf.Region `json:"-" xml:"-"`

	// highlighted regions, specific to scope markers
	Scopelights []textbuf.Region `json:"-" xml:"-"`

	// if true, select text as cursor moves
	SelectMode bool `json:"-" xml:"-"`

	// if true, complete regardless of any disqualifying reasons
	ForceComplete bool `json:"-" xml:"-"`

	// interactive search data
	ISearch ISearch `json:"-" xml:"-"`

	// query replace data
	QReplace QReplace `json:"-" xml:"-"`

	// font height, cached during styling
	FontHeight float32 `json:"-" xml:"-"`

	// line height, cached during styling
	LineHeight float32 `json:"-" xml:"-"`

	// height in lines and width in chars of the visible area
	NLinesChars image.Point `json:"-" xml:"-"`

	// total size of all lines as rendered
	LinesSize mat32.Vec2 `json:"-" xml:"-"`

	// TotalSize = LinesSize plus extra space and line numbers etc
	TotalSize mat32.Vec2 `json:"-" xml:"-"`

	// LineLayoutSize is LayState.Alloc.Size subtracting
	// extra space and line numbers -- this is what
	// LayoutStdLR sees for laying out each line
	LineLayoutSize mat32.Vec2 `json:"-" xml:"-"`

	// oscillates between on and off for blinking
	BlinkOn bool `json:"-" xml:"-"`

	// mutex protecting cursor rendering -- shared between blink and main code
	CursorMu sync.Mutex `json:"-" xml:"-" view:"-"`

	// at least one of the renders has links -- determines if we set the cursor for hand movements
	HasLinks bool `json:"-" xml:"-"`

	lastRecenter   int
	lastAutoInsert rune
	lastFilename   gi.FileName
}

// NewViewLayout adds a new layout with textview
// to given parent node, with given name.  Layout adds "-lay" suffix.
// Textview should always have a parent Layout to manage
// the scrollbars.
func NewViewLayout(parent ki.Ki, name string) (*Editor, *gi.Layout) {
	ly := parent.NewChild(gi.LayoutType, name+"-lay").(*gi.Layout)
	tv := NewView(ly, name)
	return tv, ly
}

func (tv *Editor) OnInit() {
	tv.HandleTextViewEvents()
	tv.ViewStyles()
}

func (tv *Editor) ViewStyles() {
	tv.Style(func(s *styles.Style) {
		s.SetAbilities(true, abilities.Activatable, abilities.Focusable, abilities.Hoverable, abilities.Slideable)
		tv.CursorWidth.SetDp(1)
		tv.LineNumberColor.SetSolid(colors.Scheme.SurfaceContainer)
		tv.SelectColor.SetSolid(colors.Scheme.Select.Container)
		tv.HighlightColor.SetSolid(colors.Orange)
		tv.CursorColor.SetSolid(colors.Scheme.Primary.Base)

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

// ViewFlags extend WidgetFlags to hold textview.View state
type ViewFlags int64 //enums:bitflag

const (
	// ViewHasLineNos indicates that this view has line numbers (per Buf option)
	ViewHasLineNos ViewFlags = ViewFlags(gi.WidgetFlagsN) + iota

	// ViewLastWasTabAI indicates that last key was a Tab auto-indent
	ViewLastWasTabAI

	// ViewLastWasUndo indicates that last key was an undo
	ViewLastWasUndo
)

// EditDone completes editing and copies the active edited text to the text --
// called when the return key is pressed or goes out of focus
func (tv *Editor) EditDone() {
	if tv.Buf != nil {
		tv.Buf.EditDone()
	}
	tv.ClearSelected()
}

// Remarkup triggers a complete re-markup of the entire text --
// can do this when needed if the markup gets off due to multi-line
// formatting issues -- via Recenter key
func (tv *Editor) ReMarkup() {
	if tv.Buf == nil {
		return
	}
	tv.Buf.ReMarkup()
}

// IsChanged returns true if buffer was changed (edited)
func (tv *Editor) IsChanged() bool {
	if tv.Buf != nil && tv.Buf.IsChanged() {
		return true
	}
	return false
}

// HasLineNos returns true if view is showing line numbers (per textbuf option, cached here)
func (tv *Editor) HasLineNos() bool {
	return tv.Is(ViewHasLineNos)
}

// Clear resets all the text in the buffer for this view
func (tv *Editor) Clear() {
	if tv.Buf == nil {
		return
	}
	tv.Buf.NewBuf(0)
}

///////////////////////////////////////////////////////////////////////////////
//  Buffer communication

// ResetState resets all the random state variables, when opening a new buffer etc
func (tv *Editor) ResetState() {
	tv.SelectReset()
	tv.Highlights = nil
	tv.ISearch.On = false
	tv.QReplace.On = false
	if tv.Buf == nil || tv.lastFilename != tv.Buf.Filename { // don't reset if reopening..
		tv.CursorPos = lex.Pos{}
	}
	if tv.Buf != nil {
		tv.Buf.SetInactive(tv.IsDisabled())
	}
}

// SetBuf sets the Buf that this is a view of, and interconnects their signals
func (tv *Editor) SetBuf(buf *Buf) {
	if buf != nil && tv.Buf == buf {
		return
	}
	// had := false
	if tv.Buf != nil {
		// had = true
		tv.Buf.DeleteView(tv)
	}
	tv.Buf = buf
	tv.ResetState()
	if buf != nil {
		buf.AddView(tv)
		bhl := len(buf.PosHistory)
		if bhl > 0 {
			tv.CursorPos = buf.PosHistory[bhl-1]
			tv.PosHistIdx = bhl - 1
		}
	}
	tv.SetNeedsLayout()
}

// LinesInserted inserts new lines of text and reformats them
func (tv *Editor) LinesInserted(tbe *textbuf.Edit) {
	stln := tbe.Reg.Start.Ln + 1
	nsz := (tbe.Reg.End.Ln - tbe.Reg.Start.Ln)
	if stln > len(tv.Renders) { // invalid
		return
	}

	// Renders
	tmprn := make([]paint.Text, nsz)
	nrn := append(tv.Renders, tmprn...)
	copy(nrn[stln+nsz:], nrn[stln:])
	copy(nrn[stln:], tmprn)
	tv.Renders = nrn

	// Offs
	tmpof := make([]float32, nsz)
	nof := append(tv.Offs, tmpof...)
	copy(nof[stln+nsz:], nof[stln:])
	copy(nof[stln:], tmpof)
	tv.Offs = nof

	tv.NLines += nsz
	tv.SetNeedsLayout()
}

// LinesDeleted deletes lines of text and reformats remaining one
func (tv *Editor) LinesDeleted(tbe *textbuf.Edit) {
	stln := tbe.Reg.Start.Ln
	edln := tbe.Reg.End.Ln
	dsz := edln - stln

	tv.Renders = append(tv.Renders[:stln], tv.Renders[edln:]...)
	tv.Offs = append(tv.Offs[:stln], tv.Offs[edln:]...)

	tv.NLines -= dsz
	tv.SetNeedsLayout()
}

// BufSignal receives a signal from the Buf when underlying text
// is changed.
func (tv *Editor) BufSignal(sig BufSignals, tbe *textbuf.Edit) {
	switch sig {
	case BufDone:
	case BufNew:
		tv.ResetState()
		tv.SetNeedsLayout()
		tv.SetCursorShow(tv.CursorPos)
	case BufMods:
		tv.SetNeedsLayout()
	case BufInsert:
		if tv.Renders == nil || !tv.This().(gi.Widget).IsVisible() {
			return
		}
		// fmt.Printf("tv %v got %v\n", tv.Nm, tbe.Reg.Start)
		if tbe.Reg.Start.Ln != tbe.Reg.End.Ln {
			// fmt.Printf("tv %v lines insert %v - %v\n", tv.Nm, tbe.Reg.Start, tbe.Reg.End)
			tv.LinesInserted(tbe) // triggers full layout
		} else {
			tv.LayoutLine(tbe.Reg.Start.Ln) // triggers layout if line width exceeds
		}
	case BufDelete:
		if tv.Renders == nil || !tv.This().(gi.Widget).IsVisible() {
			return
		}
		if tbe.Reg.Start.Ln != tbe.Reg.End.Ln {
			tv.LinesDeleted(tbe) // triggers full layout
		} else {
			tv.LayoutLine(tbe.Reg.Start.Ln)
		}
	case BufMarkUpdt:
		tv.SetNeedsLayout() // comes from another goroutine
	case BufClosed:
		tv.SetBuf(nil)
	}
}

///////////////////////////////////////////////////////////////////////////////
//    Undo / Redo

// Undo undoes previous action
func (tv *Editor) Undo() {
	updt := tv.UpdateStart()
	defer tv.UpdateEndRender(updt)

	tbe := tv.Buf.Undo()
	if tbe != nil {
		if tbe.Delete { // now an insert
			tv.SetCursorShow(tbe.Reg.End)
		} else {
			tv.SetCursorShow(tbe.Reg.Start)
		}
	} else {
		tv.CursorMovedSig() // updates status..
		tv.ScrollCursorToCenterIfHidden()
	}
	tv.SavePosHistory(tv.CursorPos)
}

// Redo redoes previously undone action
func (tv *Editor) Redo() {
	updt := tv.UpdateStart()
	defer tv.UpdateEndRender(updt)

	tbe := tv.Buf.Redo()
	if tbe != nil {
		if tbe.Delete {
			tv.SetCursorShow(tbe.Reg.Start)
		} else {
			tv.SetCursorShow(tbe.Reg.End)
		}
	} else {
		tv.ScrollCursorToCenterIfHidden()
	}
	tv.SavePosHistory(tv.CursorPos)
}

////////////////////////////////////////////////////
//  Widget Interface

// Config calls Init on widget
// func (tv *View) ConfigWidget(vp *gi.Scene) {
//
// }

// StyleView sets the style of widget
func (tv *Editor) StyleView(sc *gi.Scene) {
	tv.StyMu.Lock()
	defer tv.StyMu.Unlock()

	if tv.NeedsRebuild() {
		if tv.Buf != nil {
			tv.Buf.SetHiStyle(histyle.StyleDefault)
		}
	}
	tv.ApplyStyleWidget(sc)
	tv.CursorWidth.ToDots(&tv.Styles.UnContext)
}

// ApplyStyle calls StyleView and sets the style
func (tv *Editor) ApplyStyle(sc *gi.Scene) {
	// tv.SetFlag(true, gi.CanFocus) // always focusable
	tv.StyleView(sc)
	tv.StyleSizes()
}

// todo: virtual keyboard stuff

// FocusChanged appropriate actions for various types of focus changes
// func (tv *View) FocusChanged(change gi.FocusChanges) {
// 	switch change {
// 	case gi.FocusLost:
// 		tv.SetFlag(false, ViewFocusActive))
// 		// tv.EditDone()
// 		tv.StopCursor() // make sure no cursor
// 		tv.SetNeedsRender()
// 		goosi.TheApp.HideVirtualKeyboard()
// 		// fmt.Printf("lost focus: %v\n", tv.Nm)
// 	case gi.FocusGot:
// 		tv.SetFlag(true, ViewFocusActive))
// 		tv.EmitFocusedSignal()
// 		tv.SetNeedsRender()
// 		goosi.TheApp.ShowVirtualKeyboard(goosi.DefaultKeyboard)
// 		// fmt.Printf("got focus: %v\n", tv.Nm)
// 	case gi.FocusInactive:
// 		tv.SetFlag(false, ViewFocusActive))
// 		tv.StopCursor()
// 		// tv.EditDone()
// 		// tv.SetNeedsRender()
// 		goosi.TheApp.HideVirtualKeyboard()
// 		// fmt.Printf("focus inactive: %v\n", tv.Nm)
// 	case gi.FocusActive:
// 		// fmt.Printf("focus active: %v\n", tv.Nm)
// 		tv.SetFlag(true, ViewFocusActive))
// 		// tv.SetNeedsRender()
// 		// todo: see about cursor
// 		tv.StartCursor()
// 		goosi.TheApp.ShowVirtualKeyboard(goosi.DefaultKeyboard)
// 	}
// }
