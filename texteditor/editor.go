// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package texteditor

//go:generate core generate

import (
	"image"
	"sync"
	"time"

	"cogentcore.org/core/colors"
	"cogentcore.org/core/core"
	"cogentcore.org/core/cursors"
	"cogentcore.org/core/enums"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/paint"
	"cogentcore.org/core/parse/lexer"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/abilities"
	"cogentcore.org/core/styles/states"
	"cogentcore.org/core/styles/units"
	"cogentcore.org/core/texteditor/histyle"
	"cogentcore.org/core/texteditor/textbuf"
	"cogentcore.org/core/tree"
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

// Editor is a widget for editing multiple lines of complicated text (as compared to
// [core.TextField] for a single line of simple text).  The Editor is driven by a [Buffer]
// buffer which contains all the text, and manages all the edits,
// sending update events out to the editors.
//
// Use NeedsRender to drive an render update for any change that does
// not change the line-level layout of the text.
// Use NeedsLayout whenever there are changes across lines that require
// re-layout of the text.  This sets the Widget NeedsRender flag and triggers
// layout during that render.
//
// Multiple editors can be attached to a given buffer.  All updating in the
// Editor should be within a single goroutine, as it would require
// extensive protections throughout code otherwise.
type Editor struct { //core:embedder
	core.Layout

	// Buffer is the text buffer being edited.
	Buffer *Buffer `set:"-" json:"-" xml:"-"`

	// width of cursor -- set from cursor-width property (inherited)
	CursorWidth units.Value `xml:"cursor-width"`

	// the color used for the side bar containing the line numbers;
	// this should be set in Stylers like all other style properties
	LineNumberColor image.Image

	// the color used for the user text selection background color;
	// this should be set in Stylers like all other style properties
	SelectColor image.Image

	// the color used for the text highlight background color
	// (like in find); this should be set in Stylers like all other style properties
	HighlightColor image.Image

	// the color used for the text editor cursor bar;
	// this should be set in Stylers like all other style properties
	CursorColor image.Image

	// number of lines in the view, sync'd with the Buf after edits,
	// but always reflects storage size of Renders etc
	NLines int `set:"-" view:"-" json:"-" xml:"-"`

	// renders of the text lines, with one render per line
	// (each line could visibly wrap-around,
	// so these are logical lines, not display lines)
	Renders []paint.Text `set:"-" json:"-" xml:"-"`

	// starting render offsets for top of each line
	Offsets []float32 `set:"-" view:"-" json:"-" xml:"-"`

	// number of line number digits needed
	LineNumberDigits int `set:"-" view:"-" json:"-" xml:"-"`

	// horizontal offset for start of text after line numbers
	LineNumberOffset float32 `set:"-" view:"-" json:"-" xml:"-"`

	// render for line numbers
	LineNumberRender paint.Text `set:"-" view:"-" json:"-" xml:"-"`

	// current cursor position
	CursorPos lexer.Pos `set:"-" edit:"-" json:"-" xml:"-"`

	// target cursor position for externally set targets: ensures that it is visible
	CursorTarg lexer.Pos `set:"-" edit:"-" json:"-" xml:"-"`

	// desired cursor column, where the cursor was last when moved using
	// left / right arrows.  used when doing up / down to not always go
	//  to short line columns
	CursorCol int `set:"-" edit:"-" json:"-" xml:"-"`

	// current index within PosHistory
	PosHistIndex int `set:"-" edit:"-" json:"-" xml:"-"`

	// starting point for selection, which will either be the
	// start or end of selected region depending on subsequent selection.
	SelectStart lexer.Pos `set:"-" edit:"-" json:"-" xml:"-"`

	// current selection region
	SelectRegion textbuf.Region `set:"-" edit:"-" json:"-" xml:"-"`

	// previous selection region, that was actually rendered -- needed to update render
	PreviousSelectRegion textbuf.Region `set:"-" edit:"-" json:"-" xml:"-"`

	// highlighted regions, e.g., for search results
	Highlights []textbuf.Region `set:"-" edit:"-" json:"-" xml:"-"`

	// highlighted regions, specific to scope markers
	Scopelights []textbuf.Region `set:"-" edit:"-" json:"-" xml:"-"`

	// if true, select text as cursor moves
	SelectMode bool `set:"-" edit:"-" json:"-" xml:"-"`

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
	LinesSize math32.Vector2 `set:"-" edit:"-" json:"-" xml:"-"`

	// the LinesSize plus extra space and line numbers etc
	TotalSize math32.Vector2 `set:"-" edit:"-" json:"-" xml:"-"`

	// the Geom.Size.Actual.Total subtracting
	// extra space and line numbers -- this is what
	// LayoutStdLR sees for laying out each line
	LineLayoutSize math32.Vector2 `set:"-" edit:"-" json:"-" xml:"-"`

	// the last LineLayoutSize used in laying out lines.
	// Used to trigger a new layout only when needed.
	lastlineLayoutSize math32.Vector2 `set:"-" edit:"-" json:"-" xml:"-"`

	// oscillates between on and off for blinking
	BlinkOn bool `set:"-" edit:"-" json:"-" xml:"-"`

	// mutex protecting cursor rendering -- shared between blink and main code
	CursorMu sync.Mutex `set:"-" json:"-" xml:"-" view:"-"`

	// at least one of the renders has links -- determines if we set the cursor for hand movements
	HasLinks bool `set:"-" edit:"-" json:"-" xml:"-"`

	// handles link clicks -- if nil, they are sent to the standard web URL handler
	LinkHandler func(tl *paint.TextLink)

	lastRecenter   int           `set:"-"`
	lastAutoInsert rune          `set:"-"`
	lastFilename   core.Filename `set:"-"`
}

// NewSoloEditor returns a new [Editor] with an associated [Buffer].
// This is appropriate for making a standalone editor in which there
// is there is one editor per buffer.
func NewSoloEditor(parent tree.Node, name ...string) *Editor {
	return NewEditor(parent, name...).SetBuffer(NewBuffer())
}

func (ed *Editor) FlagType() enums.BitFlagSetter {
	return (*EditorFlags)(&ed.Flags)
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
		ed.LineNumberColor = colors.C(colors.Transparent)
		ed.SelectColor = colors.C(colors.Scheme.Select.Container)
		ed.HighlightColor = colors.C(colors.Scheme.Warn.Container)
		ed.CursorColor = colors.C(colors.Scheme.Primary.Base)

		s.VirtualKeyboard = styles.KeyboardMultiLine
		s.Cursor = cursors.Text
		if core.SystemSettings.Editor.WordWrap {
			s.Text.WhiteSpace = styles.WhiteSpacePreWrap
		} else {
			s.Text.WhiteSpace = styles.WhiteSpacePre
		}
		s.SetMono(true)
		s.Grow.Set(1, 1)
		s.Overflow.Set(styles.OverflowAuto) // absorbs all
		s.Border.Radius = styles.BorderRadiusLarge
		s.Margin.Zero()
		s.Padding.Set(units.Em(0.5))
		s.Align.Content = styles.Start
		s.Align.Items = styles.Start
		s.Text.Align = styles.Start
		s.Text.TabSize = core.SystemSettings.Editor.TabSize
		s.Color = colors.C(colors.Scheme.OnSurface)
		s.Min.Set(units.Em(10), units.Em(5)) // TODO: remove after #900 is fixed

		s.MaxBorder.Width.Set(units.Dp(2))
		s.Background = colors.C(colors.Scheme.SurfaceContainerLow)
		// note: a blank background does NOT work for depth color rendering
		if s.Is(states.Focused) {
			s.StateLayer = 0
			s.Border.Width.Set(units.Dp(2))
		}
	})
}

// EditorFlags extend WidgetFlags to hold [Editor] state
type EditorFlags core.WidgetFlags //enums:bitflag -trim-prefix View

const (
	// EditorHasLineNumbers indicates that this editor has line numbers
	// (per [Buffer] option)
	EditorHasLineNumbers EditorFlags = EditorFlags(core.WidgetFlagsN) + iota

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
	if ed.Buffer != nil {
		ed.Buffer.EditDone()
	}
	ed.ClearSelected()
	ed.ClearCursor()
}

// Remarkup triggers a complete re-markup of the entire text --
// can do this when needed if the markup gets off due to multi-line
// formatting issues -- via Recenter key
func (ed *Editor) ReMarkup() {
	if ed.Buffer == nil {
		return
	}
	ed.Buffer.ReMarkup()
}

// IsChanged returns true if buffer was changed (edited) since last EditDone
func (ed *Editor) IsChanged() bool {
	return ed.Buffer != nil && ed.Buffer.IsChanged()
}

// IsNotSaved returns true if buffer was changed (edited) since last Save
func (ed *Editor) IsNotSaved() bool {
	return ed.Buffer != nil && ed.Buffer.IsNotSaved()
}

// HasLineNumbers returns true if view is showing line numbers
// (per [Buffer] option, cached here).
func (ed *Editor) HasLineNumbers() bool {
	return ed.Is(EditorHasLineNumbers)
}

// Clear resets all the text in the buffer for this view
func (ed *Editor) Clear() {
	if ed.Buffer == nil {
		return
	}
	ed.Buffer.NewBuffer(0)
}

///////////////////////////////////////////////////////////////////////////////
//  Buffer communication

// ResetState resets all the random state variables, when opening a new buffer etc
func (ed *Editor) ResetState() {
	ed.SelectReset()
	ed.Highlights = nil
	ed.ISearch.On = false
	ed.QReplace.On = false
	if ed.Buffer == nil || ed.lastFilename != ed.Buffer.Filename { // don't reset if reopening..
		ed.CursorPos = lexer.Pos{}
	}
	if ed.Buffer != nil {
		ed.Buffer.SetReadOnly(ed.IsReadOnly())
	}
}

// SetBuffer sets the [Buffer] that this is a view of, and interconnects their events.
func (ed *Editor) SetBuffer(buf *Buffer) *Editor {
	if buf != nil && ed.Buffer == buf {
		return ed
	}
	// had := false
	if ed.Buffer != nil {
		// had = true
		ed.Buffer.DeleteView(ed)
	}
	ed.Buffer = buf
	ed.ResetState()
	if buf != nil {
		buf.AddView(ed)
		bhl := len(buf.PosHistory)
		if bhl > 0 {
			cp := buf.PosHistory[bhl-1]
			ed.PosHistIndex = bhl - 1
			ed.SetCursorShow(cp)
		} else {
			ed.SetCursorShow(lexer.Pos{})
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
	if stln < len(ed.Offsets) {
		ov = ed.Offsets[stln]
	} else {
		ov = ed.Offsets[len(ed.Offsets)-1]
	}
	for i := range tmpof {
		tmpof[i] = ov
	}
	nof := append(ed.Offsets, tmpof...)
	copy(nof[stln+nsz:], nof[stln:])
	copy(nof[stln:], tmpof)
	ed.Offsets = nof

	ed.NLines += nsz
	ed.NeedsLayout()
}

// LinesDeleted deletes lines of text and reformats remaining one
func (ed *Editor) LinesDeleted(tbe *textbuf.Edit) {
	stln := tbe.Reg.Start.Ln
	edln := tbe.Reg.End.Ln
	dsz := edln - stln

	ed.Renders = append(ed.Renders[:stln], ed.Renders[edln:]...)
	ed.Offsets = append(ed.Offsets[:stln], ed.Offsets[edln:]...)

	ed.NLines -= dsz
	ed.NeedsLayout()
}

// BufferSignal receives a signal from the Buffer when the underlying text
// is changed.
func (ed *Editor) BufferSignal(sig BufferSignals, tbe *textbuf.Edit) {
	switch sig {
	case BufferDone:
	case BufferNew:
		ed.ResetState()
		ed.SetCursorShow(ed.CursorPos)
		ed.NeedsLayout()
	case BufferMods:
		ed.NeedsLayout()
	case BufferInsert:
		if ed == nil || ed.This() == nil || !ed.This().(core.Widget).IsVisible() {
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
	case BufferDelete:
		if ed == nil || ed.This() == nil || !ed.This().(core.Widget).IsVisible() {
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
	case BufferMarkupUpdated:
		ed.NeedsLayout() // comes from another goroutine
	case BufferClosed:
		ed.SetBuffer(nil)
	}
}

///////////////////////////////////////////////////////////////////////////////
//    Undo / Redo

// Undo undoes previous action
func (ed *Editor) Undo() {
	tbe := ed.Buffer.Undo()
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
	tbe := ed.Buffer.Redo()
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
		if ed.Buffer != nil {
			ed.Buffer.SetHiStyle(histyle.StyleDefault)
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
