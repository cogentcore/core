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
	"cogentcore.org/core/events"
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
	core.Frame

	// Buffer is the text buffer being edited.
	Buffer *Buffer `set:"-" json:"-" xml:"-"`

	// CursorWidth is the width of the cursor.
	CursorWidth units.Value

	// LineNumberColor is the color used for the side bar containing the line numbers.
	// This should be set in Stylers like all other style properties.
	LineNumberColor image.Image

	// SelectColor is the color used for the user text selection background color.
	// This should be set in Stylers like all other style properties.
	SelectColor image.Image

	// HighlightColor is the color used for the text highlight background color (like in find).
	// This should be set in Stylers like all other style properties.
	HighlightColor image.Image

	// CursorColor is the color used for the text editor cursor bar.
	// This should be set in Stylers like all other style properties.
	CursorColor image.Image

	// NLines is the number of lines in the view, synced with the Buf after edits,
	// but always reflects the storage size of Renders etc.
	NLines int `set:"-" display:"-" json:"-" xml:"-"`

	// Renders is a slice of paint.Text representing the renders of the text lines,
	// with one render per line (each line could visibly wrap-around, so these are logical lines, not display lines).
	Renders []paint.Text `set:"-" json:"-" xml:"-"`

	// Offsets is a slice of float32 representing the starting render offsets for the top of each line.
	Offsets []float32 `set:"-" display:"-" json:"-" xml:"-"`

	// LineNumberDigits is the number of line number digits needed.
	LineNumberDigits int `set:"-" display:"-" json:"-" xml:"-"`

	// LineNumberOffset is the horizontal offset for the start of text after line numbers.
	LineNumberOffset float32 `set:"-" display:"-" json:"-" xml:"-"`

	// LineNumberRender is the render for line numbers.
	LineNumberRender paint.Text `set:"-" display:"-" json:"-" xml:"-"`

	// CursorPos is the current cursor position.
	CursorPos lexer.Pos `set:"-" edit:"-" json:"-" xml:"-"`

	// CursorTarget is the target cursor position for externally set targets.
	// It ensures that the target position is visible.
	CursorTarget lexer.Pos `set:"-" edit:"-" json:"-" xml:"-"`

	// CursorCol is the desired cursor column, where the cursor was last when moved using left / right arrows.
	// It is used when doing up / down to not always go to short line columns.
	CursorCol int `set:"-" edit:"-" json:"-" xml:"-"`

	// PosHistIndex is the current index within PosHistory.
	PosHistIndex int `set:"-" edit:"-" json:"-" xml:"-"`

	// SelectStart is the starting point for selection, which will either be the start or end of selected region
	// depending on subsequent selection.
	SelectStart lexer.Pos `set:"-" edit:"-" json:"-" xml:"-"`

	// SelectRegion is the current selection region.
	SelectRegion textbuf.Region `set:"-" edit:"-" json:"-" xml:"-"`

	// PreviousSelectRegion is the previous selection region that was actually rendered.
	// It is needed to update the render.
	PreviousSelectRegion textbuf.Region `set:"-" edit:"-" json:"-" xml:"-"`

	// Highlights is a slice of regions representing the highlighted regions, e.g., for search results.
	Highlights []textbuf.Region `set:"-" edit:"-" json:"-" xml:"-"`

	// Scopelights is a slice of regions representing the highlighted regions specific to scope markers.
	Scopelights []textbuf.Region `set:"-" edit:"-" json:"-" xml:"-"`

	// LinkHandler handles link clicks.
	// If it is nil, they are sent to the standard web URL handler.
	LinkHandler func(tl *paint.TextLink)

	// ISearch is the interactive search data.
	ISearch ISearch `set:"-" edit:"-" json:"-" xml:"-"`

	// QReplace is the query replace data.
	QReplace QReplace `set:"-" edit:"-" json:"-" xml:"-"`

	// selectMode is a boolean indicating whether to select text as the cursor moves.
	selectMode bool

	// fontHeight is the font height, cached during styling.
	fontHeight float32

	// lineHeight is the line height, cached during styling.
	lineHeight float32

	// fontAscent is the font ascent, cached during styling.
	fontAscent float32

	// fontDescent is the font descent, cached during styling.
	fontDescent float32

	// nLinesChars is the height in lines and width in chars of the visible area.
	nLinesChars image.Point

	// linesSize is the total size of all lines as rendered.
	linesSize math32.Vector2

	// totalSize is the LinesSize plus extra space and line numbers etc.
	totalSize math32.Vector2

	// lineLayoutSize is the Geom.Size.Actual.Total subtracting extra space and line numbers.
	// This is what LayoutStdLR sees for laying out each line.
	lineLayoutSize math32.Vector2

	// lastlineLayoutSize is the last LineLayoutSize used in laying out lines.
	// It is used to trigger a new layout only when needed.
	lastlineLayoutSize math32.Vector2

	// blinkOn oscillates between on and off for blinking.
	blinkOn bool

	// cursorMu is a mutex protecting cursor rendering, shared between blink and main code.
	cursorMu sync.Mutex

	// hasLinks is a boolean indicating if at least one of the renders has links.
	// It determines if we set the cursor for hand movements.
	hasLinks bool

	// hasLineNumbers indicates that this editor has line numbers
	// (per [Buffer] option)
	hasLineNumbers bool // TODO: is this really necessary?

	// needsLayout is set by NeedsLayout: Editor does significant
	// internal layout in LayoutAllLines, and its layout is simply based
	// on what it gets allocated, so it does not affect the rest
	// of the Scene.
	needsLayout bool

	// lastWasTabAI indicates that last key was a Tab auto-indent
	lastWasTabAI bool

	// lastWasUndo indicates that last key was an undo
	lastWasUndo bool

	// targetSet indicates that the CursorTarget is set
	targetSet bool

	lastRecenter   int
	lastAutoInsert rune
	lastFilename   core.Filename
}

// NewSoloEditor returns a new [Editor] with an associated [Buffer].
// This is appropriate for making a standalone editor in which there
// is there is one editor per buffer.
func NewSoloEditor(parent ...tree.Node) *Editor {
	return NewEditor(parent...).SetBuffer(NewBuffer())
}

func (ed *Editor) WidgetValue() any { return &ed.Buffer.Txt }

func (ed *Editor) Init() {
	ed.Frame.Init()
	ed.AddContextMenu(ed.ContextMenu)
	ed.Styler(func(s *styles.Style) {
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

	ed.HandleKeyChord()
	ed.HandleMouse()
	ed.HandleLinkCursor()
	ed.HandleFocus()
	ed.OnClose(func(e events.Event) {
		ed.EditDone()
	})

	ed.Updater(ed.NeedsLayout)
}

func (ed *Editor) Destroy() {
	ed.StopCursor()
	ed.Frame.Destroy()
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

// ReMarkup triggers a complete re-markup of the entire text --
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
	return ed.Buffer != nil && ed.Buffer.Changed
}

// IsNotSaved returns true if buffer was changed (edited) since last Save
func (ed *Editor) IsNotSaved() bool {
	return ed.Buffer != nil && ed.Buffer.NotSaved
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
		if ed == nil || ed.This == nil || !ed.This.(core.Widget).IsVisible() {
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
		if ed == nil || ed.This == nil || !ed.This.(core.Widget).IsVisible() {
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

// StyleView sets the style of widget
func (ed *Editor) StyleView() {
	if ed.NeedsRebuild() {
		if ed.Buffer != nil {
			ed.Buffer.SetHiStyle(histyle.StyleDefault)
		}
	}
	ed.WidgetBase.Style()
	ed.CursorWidth.ToDots(&ed.Styles.UnitContext)
}

// Style calls StyleView and sets the style
func (ed *Editor) Style() {
	ed.StyleView()
	ed.StyleSizes()
}
