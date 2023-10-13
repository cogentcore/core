// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package textview

//go:generate goki generate

import (
	"image"
	"sync"

	"goki.dev/colors"
	"goki.dev/cursors"
	"goki.dev/gi/v2/gi"
	"goki.dev/gi/v2/textview/histyle"
	"goki.dev/gi/v2/textview/textbuf"
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

// View is a widget for editing multiple lines of text (as compared to
// TextField for a single line).  The View is driven by a Buf buffer which
// contains all the text, and manages all the edits, sending update signals
// out to the views -- multiple views can be attached to a given buffer.  All
// updating in the View should be within a single goroutine -- it would
// require extensive protections throughout code otherwise.
//
//goki:embedder
type View struct {
	gi.WidgetBase

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

	// starting offsets for top of each line
	Offs []float32 `json:"-" xml:"-"`

	// number of line number digits needed
	LineNoDigs int `json:"-" xml:"-"`

	// horizontal offset for start of text after line numbers
	LineNoOff float32 `json:"-" xml:"-"`

	// render for line numbers
	LineNoRender paint.Text `json:"-" xml:"-"`

	// total size of all lines as rendered
	LinesSize image.Point `json:"-" xml:"-"`

	// size params to use in render call
	RenderSz mat32.Vec2 `json:"-" xml:"-"`

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
	VisSize image.Point `json:"-" xml:"-"`

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
func NewViewLayout(parent ki.Ki, name string) (*View, *gi.Layout) {
	ly := parent.NewChild(gi.LayoutType, name+"-lay").(*gi.Layout)
	tv := NewView(ly, name)
	return tv, ly
}

func (tv *View) OnInit() {
	tv.HandleViewEvents()
	tv.ViewStyles()
}

func (tv *View) ViewStyles() {
	tv.AddStyles(func(s *styles.Style) {
		s.SetAbilities(true, abilities.Activatable, abilities.Focusable, abilities.Hoverable, abilities.Slideable)
		tv.CursorWidth.SetDp(2)
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

// ViewFlags extend NodeBase NodeFlags to hold View state
type ViewFlags int64 //enums:bitflag

const (
	// ViewNeedsRefresh indicates when refresh is required
	ViewNeedsRefresh ViewFlags = ViewFlags(gi.WidgetFlagsN) + iota

	// ViewInReLayout indicates that we are currently resizing ourselves via parent layout
	ViewInReLayout

	// ViewRenderScrolls indicates that parent layout scrollbars need to be re-rendered at next rerender
	ViewRenderScrolls

	// ViewHasLineNos indicates that this view has line numbers (per Buf option)
	ViewHasLineNos

	// ViewLastWasTabAI indicates that last key was a Tab auto-indent
	ViewLastWasTabAI

	// ViewLastWasUndo indicates that last key was an undo
	ViewLastWasUndo
)

// EditDone completes editing and copies the active edited text to the text --
// called when the return key is pressed or goes out of focus
func (tv *View) EditDone() {
	if tv.Buf != nil {
		tv.Buf.EditDone()
	}
	tv.ClearSelected()
}

// Refresh re-displays everything anew from the buffer
func (tv *View) Refresh() {
	if tv == nil || tv.This() == nil {
		return
	}
	if !tv.This().(gi.Widget).IsVisible() {
		return
	}
	tv.LayoutAllLines(false)
	tv.UpdateSig()
	tv.ClearNeedsRefresh()
}

// Remarkup triggers a complete re-markup of the entire text --
// can do this when needed if the markup gets off due to multi-line
// formatting issues -- via Recenter key
func (tv *View) ReMarkup() {
	if tv.Buf == nil {
		return
	}
	tv.Buf.ReMarkup()
}

// NeedsRefresh checks if a refresh is required -- atomically safe for other
// routines to set the NeedsRefresh flag
func (tv *View) NeedsRefresh() bool {
	return tv.Is(ViewNeedsRefresh)
}

// SetNeedsRefresh flags that a refresh is required -- atomically safe for
// other routines to call this
func (tv *View) SetNeedsRefresh() {
	tv.SetFlag(true, ViewNeedsRefresh)
}

// ClearNeedsRefresh clears needs refresh flag -- atomically safe
func (tv *View) ClearNeedsRefresh() {
	tv.SetFlag(false, ViewNeedsRefresh)
}

// RefreshIfNeeded re-displays everything if SetNeedsRefresh was called --
// returns true if refreshed
func (tv *View) RefreshIfNeeded() bool {
	if tv.NeedsRefresh() {
		tv.Refresh()
		return true
	}
	return false
}

// IsChanged returns true if buffer was changed (edited)
func (tv *View) IsChanged() bool {
	if tv.Buf != nil && tv.Buf.IsChanged() {
		return true
	}
	return false
}

// HasLineNos returns true if view is showing line numbers (per textbuf option, cached here)
func (tv *View) HasLineNos() bool {
	return tv.Is(ViewHasLineNos)
}

// Clear resets all the text in the buffer for this view
func (tv *View) Clear() {
	if tv.Buf == nil {
		return
	}
	tv.Buf.NewBuf(0)
}

///////////////////////////////////////////////////////////////////////////////
//  Buffer communication

// ResetState resets all the random state variables, when opening a new buffer etc
func (tv *View) ResetState() {
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
func (tv *View) SetBuf(buf *Buf) {
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
	tv.LayoutAllLines(false)
	tv.UpdateSig()
	tv.SetCursorShow(tv.CursorPos)
}

// LinesInserted inserts new lines of text and reformats them
func (tv *View) LinesInserted(tbe *textbuf.Edit) {
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

	tv.LayoutLines(tbe.Reg.Start.Ln, tbe.Reg.End.Ln, false)
	tv.UpdateSig()
}

// LinesDeleted deletes lines of text and reformats remaining one
func (tv *View) LinesDeleted(tbe *textbuf.Edit) {
	stln := tbe.Reg.Start.Ln
	edln := tbe.Reg.End.Ln
	dsz := edln - stln

	tv.Renders = append(tv.Renders[:stln], tv.Renders[edln:]...)
	tv.Offs = append(tv.Offs[:stln], tv.Offs[edln:]...)

	tv.NLines -= dsz

	tv.LayoutLines(tbe.Reg.Start.Ln, tbe.Reg.Start.Ln, true)
	tv.UpdateSig()
}

// BufSignal receives a signal from the Buf when underlying text
// is changed.
func (tv *View) BufSignal(sig BufSignals, tbe *textbuf.Edit) {
	switch sig {
	case BufDone:
	case BufNew:
		tv.ResetState()
		tv.SetNeedsRefresh() // in case not visible
		tv.Refresh()
		tv.SetCursorShow(tv.CursorPos)
	case BufInsert:
		if tv.Renders == nil || !tv.This().(gi.Widget).IsVisible() {
			return
		}
		// fmt.Printf("tv %v got %v\n", tv.Nm, tbe.Reg.Start)
		if tbe.Reg.Start.Ln != tbe.Reg.End.Ln {
			// fmt.Printf("tv %v lines insert %v - %v\n", tv.Nm, tbe.Reg.Start, tbe.Reg.End)
			tv.LinesInserted(tbe)
		} else {
			rerend := tv.LayoutLines(tbe.Reg.Start.Ln, tbe.Reg.End.Ln, false)
			if rerend {
				// fmt.Printf("tv %v line insert rerend %v - %v\n", tv.Nm, tbe.Reg.Start, tbe.Reg.End)
				tv.RenderAllLines()
			} else {
				// fmt.Printf("tv %v line insert no rerend %v - %v.  markup: %v\n", tv.Nm, tbe.Reg.Start, tbe.Reg.End, len(tv.Buf.HiTags[tbe.Reg.Start.Ln]))
				tv.RenderLines(tbe.Reg.Start.Ln, tbe.Reg.End.Ln)
			}
		}
	case BufDelete:
		if tv.Renders == nil || !tv.This().(gi.Widget).IsVisible() {
			return
		}
		if tbe.Reg.Start.Ln != tbe.Reg.End.Ln {
			tv.LinesDeleted(tbe)
		} else {
			rerend := tv.LayoutLines(tbe.Reg.Start.Ln, tbe.Reg.End.Ln, true)
			if rerend {
				tv.RenderAllLines()
			} else {
				tv.RenderLines(tbe.Reg.Start.Ln, tbe.Reg.End.Ln)
			}
		}
	case BufMarkUpdt:
		tv.SetNeedsRefresh() // comes from another goroutine
	case BufClosed:
		tv.SetBuf(nil)
	}
}

///////////////////////////////////////////////////////////////////////////////
//    Undo / Redo

// Undo undoes previous action
func (tv *View) Undo() {
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
func (tv *View) Redo() {
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
func (tv *View) StyleView(sc *gi.Scene) {
	tv.StyMu.Lock()
	defer tv.StyMu.Unlock()

	if tv.NeedsRebuild() {
		if tv.Buf != nil {
			tv.Buf.SetHiStyle(histyle.StyleDefault)
		}
		// win := tv.ParentRenderWin()
		// if win != nil {
		// 	spnm := tv.CursorSpriteName()
		// 	win.DeleteSprite(spnm)
		// }
	}
	tv.ApplyStyleWidget(sc)
	tv.CursorWidth.ToDots(&tv.Style.UnContext)
}

// ApplyStyle calls StyleView and sets the style
func (tv *View) ApplyStyle(sc *gi.Scene) {
	// tv.SetFlag(true, gi.CanFocus) // always focusable
	tv.StyleView(sc)
}

// GetSize
func (tv *View) GetSize(sc *gi.Scene, iter int) {
	if iter > 0 {
		return
	}
	tv.InitLayout(sc)
	if tv.LinesSize == (image.Point{}) {
		tv.LayoutAllLines(true)
	} else {
		tv.SetSize()
	}
}

// DoLayoutn
func (tv *View) DoLayout(sc *gi.Scene, parBBox image.Rectangle, iter int) bool {
	tv.DoLayoutBase(sc, parBBox, iter)
	tv.DoLayoutChildren(sc, iter)
	if tv.LinesSize == image.ZP || tv.NeedsRebuild() || tv.NeedsRefresh() || tv.NLines != tv.Buf.NumLines() {
		redo := tv.LayoutAllLines(true) // is our size now different?  if so iterate..
		return redo
	}
	tv.SetSize()
	return false
}

// FocusChanged appropriate actions for various types of focus changes
// func (tv *View) FocusChanged(change gi.FocusChanges) {
// 	switch change {
// 	case gi.FocusLost:
// 		tv.SetFlag(false, ViewFocusActive))
// 		// tv.EditDone()
// 		tv.StopCursor() // make sure no cursor
// 		tv.UpdateSig()
// 		goosi.TheApp.HideVirtualKeyboard()
// 		// fmt.Printf("lost focus: %v\n", tv.Nm)
// 	case gi.FocusGot:
// 		tv.SetFlag(true, ViewFocusActive))
// 		tv.EmitFocusedSignal()
// 		tv.UpdateSig()
// 		goosi.TheApp.ShowVirtualKeyboard(goosi.DefaultKeyboard)
// 		// fmt.Printf("got focus: %v\n", tv.Nm)
// 	case gi.FocusInactive:
// 		tv.SetFlag(false, ViewFocusActive))
// 		tv.StopCursor()
// 		// tv.EditDone()
// 		// tv.UpdateSig()
// 		goosi.TheApp.HideVirtualKeyboard()
// 		// fmt.Printf("focus inactive: %v\n", tv.Nm)
// 	case gi.FocusActive:
// 		// fmt.Printf("focus active: %v\n", tv.Nm)
// 		tv.SetFlag(true, ViewFocusActive))
// 		// tv.UpdateSig()
// 		// todo: see about cursor
// 		tv.StartCursor()
// 		goosi.TheApp.ShowVirtualKeyboard(goosi.DefaultKeyboard)
// 	}
// }
