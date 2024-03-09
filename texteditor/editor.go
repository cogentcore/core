// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package texteditor

//go:generate core generate

import (
	"image"
	"sync"
	"time"

	"cogentcore.org/core/abilities"
	"cogentcore.org/core/colors"
	"cogentcore.org/core/cursors"
	"cogentcore.org/core/enums"
	"cogentcore.org/core/gi"
	"cogentcore.org/core/goosi"
	"cogentcore.org/core/ki"
	"cogentcore.org/core/mat32"
	"cogentcore.org/core/paint"
	"cogentcore.org/core/pi/lex"
	"cogentcore.org/core/states"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/texteditor/histyle"
	"cogentcore.org/core/texteditor/textbuf"
	"cogentcore.org/core/units"
)

var (
	// Maximum amount of clipboard history to retain
	ClipHistMax = 100 // `default:"100" min:"0" max:"1000" step:"5"`

	// maximum number of lines to look for matching scope syntax (parens, brackets)
	MaxScopeLines = 100 // `default:"100" min:"10" step:"10"`

	// text buffer max lines to use diff-based revert to more quickly update e.g., after file has been reformatted
	DiffRevertLines = 10000 // `default:"10000" min:"0" step:"1000"`

	// text buffer max diffs to use diff-based revert to more quickly update e.g., after file has been reformatted -- if too many differences, just revert
	DiffRevertDiffs = 20 // `default:"20" min:"0" step:"1"`

	// amount of time to wait before starting a new background markup process, after text changes within a single line (always does after line insertion / deletion)
	MarkupDelay = 1000 * time.Millisecond // `default:"1000" min:"100" step:"100"`
)

// Editor is a widget for editing multiple lines of text (as compared to
// [gi.TextField] for a single line).  The Editor is driven by a [Buf]
// buffer which contains all the text, and manages all the edits,
// sending update signals out to the views.
//
// Use NeedsRender to drive an render update for any change that does
// not change the line-level layout of the text.
// Use NeedsLayout whenever there are changes across lines that require
// re-layout of the text.  This sets the Widget NeedsRender flag and triggers
// layout during that render.
//
// Multiple views can be attached to a given buffer.  All updating in the
// Editor should be within a single goroutine, as it would require
// extensive protections throughout code otherwise.
type Editor struct { //core:embedder
	gi.Layout

	// the text buffer that we're editing
	Buf *Buf `set:"-" json:"-" xml:"-"`

	// text that is displayed when the field is empty, in a lower-contrast manner
	Placeholder string `json:"-" xml:"placeholder"`

	// width of cursor -- set from cursor-width property (inherited)
	CursorWidth units.Value `xml:"cursor-width"`

	// the color used for the side bar containing the line numbers; this should be set in Stylers like all other style properties
	LineNumberColor image.Image

	// the color used for the user text selection background color; this should be set in Stylers like all other style properties
	SelectColor image.Image

	// the color used for the text highlight background color (like in find); this should be set in Stylers like all other style properties
	HighlightColor image.Image

	// the color used for the text field cursor (caret); this should be set in Stylers like all other style properties
	CursorColor image.Image

	// number of lines in the view -- sync'd with the Buf after edits, but always reflects storage size of Renders etc
	NLines int `set:"-" view:"-" json:"-" xml:"-"`

	// renders of the text lines, with one render per line (each line could visibly wrap-around, so these are logical lines, not display lines)
	Renders []paint.Text `set:"-" json:"-" xml:"-"`

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

	// target cursor position for externally-set targets: ensures that it is visible
	CursorTarg lex.Pos `set:"-" edit:"-" json:"-" xml:"-"`

	// desired cursor column -- where the cursor was last when moved using left / right arrows -- used when doing up / down to not always go to short line columns
	CursorCol int `set:"-" edit:"-" json:"-" xml:"-"`

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

	// font ascent, cached during styling
	FontAscent float32 `set:"-" edit:"-" json:"-" xml:"-"`

	// font descent, cached during styling
	FontDescent float32 `set:"-" edit:"-" json:"-" xml:"-"`

	// height in lines and width in chars of the visible area
	NLinesChars image.Point `set:"-" edit:"-" json:"-" xml:"-"`

	// total size of all lines as rendered
	LinesSize mat32.Vec2 `set:"-" edit:"-" json:"-" xml:"-"`

	// the LinesSize plus extra space and line numbers etc
	TotalSize mat32.Vec2 `set:"-" edit:"-" json:"-" xml:"-"`

	// the Geom.Size.Actual.Total subtracting
	// extra space and line numbers -- this is what
	// LayoutStdLR sees for laying out each line
	LineLayoutSize mat32.Vec2 `set:"-" edit:"-" json:"-" xml:"-"`

	// the last LineLayoutSize used in laying out lines.
	// Used to trigger a new layout only when needed.
	lastlineLayoutSize mat32.Vec2 `set:"-" edit:"-" json:"-" xml:"-"`

	// oscillates between on and off for blinking
	BlinkOn bool `set:"-" edit:"-" json:"-" xml:"-"`

	// mutex protecting cursor rendering -- shared between blink and main code
	CursorMu sync.Mutex `set:"-" json:"-" xml:"-" view:"-"`

	// at least one of the renders has links -- determines if we set the cursor for hand movements
	HasLinks bool `set:"-" edit:"-" json:"-" xml:"-"`

	// handles link clicks -- if nil, they are sent to the standard web URL handler
	LinkHandler func(tl *paint.TextLink)

	lastRecenter   int         `set:"-"`
	lastAutoInsert rune        `set:"-"`
	lastFilename   gi.Filename `set:"-"`
}

func (ed *Editor) FlagType() enums.BitFlagSetter {
	return (*EditorFlags)(&ed.Flags)
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
	ed.WidgetBase.OnInit()
	ed.HandleEvents()
	ed.SetStyles()
}

func (ed *Editor) SetStyles() {
	ed.Style(func(s *styles.Style) {
		s.SetAbilities(true, abilities.Activatable, abilities.Focusable, abilities.Hoverable, abilities.Slideable, abilities.DoubleClickable, abilities.TripleClickable)
		ed.CursorWidth.Dp(2)
		ed.LineNumberColor = colors.C(colors.Scheme.SurfaceContainer)
		ed.SelectColor = colors.C(colors.Scheme.Select.Container)
		ed.HighlightColor = colors.C(colors.Scheme.Warn.Container)
		ed.CursorColor = colors.C(colors.Scheme.Primary.Base)

		s.VirtualKeyboard = styles.KeyboardMultiLine
		s.Cursor = cursors.Text
		if gi.SystemSettings.Editor.WordWrap {
			s.Text.WhiteSpace = styles.WhiteSpacePreWrap
		} else {
			s.Text.WhiteSpace = styles.WhiteSpacePre
		}
		s.Font.Family = string(gi.AppearanceSettings.MonoFont)
		s.Grow.Set(1, 1)
		s.Overflow.Set(styles.OverflowAuto)   // absorbs all
		s.Border.Style.Set(styles.BorderNone) // don't render our own border
		s.Border.Radius = styles.BorderRadiusLarge
		s.Margin.Zero()
		s.Padding.Set(units.Dp(4))
		s.Align.Content = styles.Start
		s.Align.Items = styles.Start
		s.Text.Align = styles.Start
		s.Text.TabSize = gi.SystemSettings.Editor.TabSize
		s.Color = colors.C(colors.Scheme.OnSurface)

		if s.State.Is(states.Focused) {
			s.Background = colors.C(colors.Scheme.Surface)
		} else {
			s.Background = colors.C(colors.Scheme.SurfaceContainerHigh)
		}
	})
}

// EditorFlags extend WidgetFlags to hold [Editor] state
type EditorFlags gi.WidgetFlags //enums:bitflag -trim-prefix View

const (
	// EditorHasLineNos indicates that this editor has line numbers (per Buf option)
	EditorHasLineNos EditorFlags = EditorFlags(gi.WidgetFlagsN) + iota

	// EditorNeedsLayout is set by NeedsLayout: Editor does significant
	// internal layout in LayoutAllLines, and its layout is simply based
	// on what it gets allocated, so it does not affect the rest
	// of the Scene.
	EditorNeedsLayout

	// EditorLastWasTabAI indicates that last key was a Tab auto-indent
	EditorLastWasTabAI

	// EditorLastWasUndo indicates that last key was an undo
	EditorLastWasUndo

	// EditorTargetSet indicates that the CursorTarget is set
	EditorTargetSet
)

func (ed *Editor) Destroy() {
	ed.StopCursor()
	ed.Layout.Destroy()
}

// EditDone completes editing and copies the active edited text to the text --
// called when the return key is pressed or goes out of focus
func (ed *Editor) EditDone() {
	if ed.Buf != nil {
		ed.Buf.EditDone()
	}
	ed.ClearSelected()
	ed.ClearCursor()
	goosi.TheApp.HideVirtualKeyboard()
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

// IsChanged returns true if buffer was changed (edited) since last EditDone
func (ed *Editor) IsChanged() bool {
	return ed.Buf != nil && ed.Buf.IsChanged()
}

// IsNotSaved returns true if buffer was changed (edited) since last Save
func (ed *Editor) IsNotSaved() bool {
	return ed.Buf != nil && ed.Buf.IsNotSaved()
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
			cp := buf.PosHistory[bhl-1]
			ed.PosHistIdx = bhl - 1
			ed.SetCursorShow(cp)
		} else {
			ed.SetCursorShow(lex.Pos{})
		}
	}
	ed.LayoutAllLines()
	ed.NeedsLayout()
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
	ov := float32(0)
	if stln < len(ed.Offs) {
		ov = ed.Offs[stln]
	} else {
		ov = ed.Offs[len(ed.Offs)-1]
	}
	for i := range tmpof {
		tmpof[i] = ov
	}
	nof := append(ed.Offs, tmpof...)
	copy(nof[stln+nsz:], nof[stln:])
	copy(nof[stln:], tmpof)
	ed.Offs = nof

	ed.NLines += nsz
	ed.NeedsLayout()
}

// LinesDeleted deletes lines of text and reformats remaining one
func (ed *Editor) LinesDeleted(tbe *textbuf.Edit) {
	stln := tbe.Reg.Start.Ln
	edln := tbe.Reg.End.Ln
	dsz := edln - stln

	ed.Renders = append(ed.Renders[:stln], ed.Renders[edln:]...)
	ed.Offs = append(ed.Offs[:stln], ed.Offs[edln:]...)

	ed.NLines -= dsz
	ed.NeedsLayout()
}

// BufSignal receives a signal from the Buf when underlying text
// is changed.
func (ed *Editor) BufSignal(sig BufSignals, tbe *textbuf.Edit) {
	switch sig {
	case BufDone:
	case BufNew:
		ed.ResetState()
		ed.SetCursorShow(ed.CursorPos)
		ed.NeedsLayout()
	case BufMods:
		ed.NeedsLayout()
	case BufInsert:
		if ed == nil || ed.This() == nil || !ed.This().(gi.Widget).IsVisible() {
			return
		}
		ndup := ed.Renders == nil
		// fmt.Printf("ed %v got %v\n", ed.Nm, tbe.Reg.Start)
		if tbe.Reg.Start.Ln != tbe.Reg.End.Ln {
			// fmt.Printf("ed %v lines insert %v - %v\n", ed.Nm, tbe.Reg.Start, tbe.Reg.End)
			ed.LinesInserted(tbe) // triggers full layout
		} else {
			ed.LayoutLine(tbe.Reg.Start.Ln) // triggers layout if line width exceeds
		}
		if ndup {
			ed.Update()
		}
	case BufDelete:
		if ed == nil || ed.This() == nil || !ed.This().(gi.Widget).IsVisible() {
			return
		}
		ndup := ed.Renders == nil
		if tbe.Reg.Start.Ln != tbe.Reg.End.Ln {
			ed.LinesDeleted(tbe) // triggers full layout
		} else {
			ed.LayoutLine(tbe.Reg.Start.Ln)
		}
		if ndup {
			ed.Update()
		}
	case BufMarkUpdt:
		ed.NeedsLayout() // comes from another goroutine
	case BufClosed:
		ed.SetBuf(nil)
	}
}

///////////////////////////////////////////////////////////////////////////////
//    Undo / Redo

// Undo undoes previous action
func (ed *Editor) Undo() {
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
	ed.NeedsRender()
}

// Redo redoes previously undone action
func (ed *Editor) Redo() {
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
	ed.NeedsRender()
}

////////////////////////////////////////////////////
//  Widget Interface

func (ed *Editor) Config() {
	ed.NeedsLayout()
}

// StyleView sets the style of widget
func (ed *Editor) StyleView() {
	if ed.NeedsRebuild() {
		if ed.Buf != nil {
			ed.Buf.SetHiStyle(histyle.StyleDefault)
		}
	}
	ed.ApplyStyleWidget()
	ed.CursorWidth.ToDots(&ed.Styles.UnitContext)
}

// ApplyStyle calls StyleView and sets the style
func (ed *Editor) ApplyStyle() {
	ed.StyleView()
	ed.StyleSizes()
}
