// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package texteditor

//go:generate core generate

import (
	"image"
	"slices"
	"sync"

	"cogentcore.org/core/base/reflectx"
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
	"cogentcore.org/core/texteditor/highlighting"
	"cogentcore.org/core/texteditor/text"
)

// TODO: move these into an editor settings object
var (
	// Maximum amount of clipboard history to retain
	clipboardHistoryMax = 100 // `default:"100" min:"0" max:"1000" step:"5"`

	// text buffer max lines to use diff-based revert to more quickly update e.g., after file has been reformatted
	diffRevertLines = 10000 // `default:"10000" min:"0" step:"1000"`

	// text buffer max diffs to use diff-based revert to more quickly update e.g., after file has been reformatted -- if too many differences, just revert
	diffRevertDiffs = 20 // `default:"20" min:"0" step:"1"`
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
	// This should be set in Stylers like all other style properties.
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

	// NumLines is the number of lines in the view, synced with the [Buffer] after edits,
	// but always reflects the storage size of renders etc.
	NumLines int `set:"-" display:"-" json:"-" xml:"-"`

	// renders is a slice of paint.Text representing the renders of the text lines,
	// with one render per line (each line could visibly wrap-around, so these are logical lines, not display lines).
	renders []paint.Text

	// offsets is a slice of float32 representing the starting render offsets for the top of each line.
	offsets []float32

	// lineNumberDigits is the number of line number digits needed.
	lineNumberDigits int

	// LineNumberOffset is the horizontal offset for the start of text after line numbers.
	LineNumberOffset float32 `set:"-" display:"-" json:"-" xml:"-"`

	// lineNumberRender is the render for line numbers.
	lineNumberRender paint.Text

	// CursorPos is the current cursor position.
	CursorPos lexer.Pos `set:"-" edit:"-" json:"-" xml:"-"`

	// cursorTarget is the target cursor position for externally set targets.
	// It ensures that the target position is visible.
	cursorTarget lexer.Pos

	// cursorColumn is the desired cursor column, where the cursor was last when moved using left / right arrows.
	// It is used when doing up / down to not always go to short line columns.
	cursorColumn int

	// posHistoryIndex is the current index within PosHistory.
	posHistoryIndex int

	// selectStart is the starting point for selection, which will either be the start or end of selected region
	// depending on subsequent selection.
	selectStart lexer.Pos

	// SelectRegion is the current selection region.
	SelectRegion text.Region `set:"-" edit:"-" json:"-" xml:"-"`

	// previousSelectRegion is the previous selection region that was actually rendered.
	// It is needed to update the render.
	previousSelectRegion text.Region

	// Highlights is a slice of regions representing the highlighted regions, e.g., for search results.
	Highlights []text.Region `set:"-" edit:"-" json:"-" xml:"-"`

	// scopelights is a slice of regions representing the highlighted regions specific to scope markers.
	scopelights []text.Region

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

func (ed *Editor) WidgetValue() any { return ed.Buffer.Text() }

func (ed *Editor) SetWidgetValue(value any) error {
	ed.Buffer.SetString(reflectx.ToString(value))
	return nil
}

func (ed *Editor) Init() {
	ed.Frame.Init()
	ed.AddContextMenu(ed.contextMenu)
	ed.SetBuffer(NewBuffer())
	ed.Styler(func(s *styles.Style) {
		s.SetAbilities(true, abilities.Activatable, abilities.Focusable, abilities.Hoverable, abilities.Slideable, abilities.DoubleClickable, abilities.TripleClickable)
		ed.CursorWidth.Dp(2)
		ed.LineNumberColor = colors.Uniform(colors.Transparent)
		ed.SelectColor = colors.Scheme.Select.Container
		ed.HighlightColor = colors.Scheme.Warn.Container
		ed.CursorColor = colors.Scheme.Primary.Base

		s.Cursor = cursors.Text
		s.VirtualKeyboard = styles.KeyboardMultiLine
		if core.SystemSettings.Editor.WordWrap {
			s.Text.WhiteSpace = styles.WhiteSpacePreWrap
		} else {
			s.Text.WhiteSpace = styles.WhiteSpacePre
		}
		s.SetMono(true)
		s.Grow.Set(1, 0)
		s.Overflow.Set(styles.OverflowAuto) // absorbs all
		s.Border.Radius = styles.BorderRadiusLarge
		s.Margin.Zero()
		s.Padding.Set(units.Em(0.5))
		s.Align.Content = styles.Start
		s.Align.Items = styles.Start
		s.Text.Align = styles.Start
		s.Text.AlignV = styles.Start
		s.Text.TabSize = core.SystemSettings.Editor.TabSize
		s.Color = colors.Scheme.OnSurface
		s.Min.X.Em(10)

		s.MaxBorder.Width.Set(units.Dp(2))
		s.Background = colors.Scheme.SurfaceContainerLow
		// note: a blank background does NOT work for depth color rendering
		if s.Is(states.Focused) {
			s.StateLayer = 0
			s.Border.Width.Set(units.Dp(2))
		}
	})

	ed.handleKeyChord()
	ed.handleMouse()
	ed.handleLinkCursor()
	ed.handleFocus()
	ed.OnClose(func(e events.Event) {
		ed.editDone()
	})

	ed.Updater(ed.NeedsLayout)
}

func (ed *Editor) Destroy() {
	ed.stopCursor()
	ed.Frame.Destroy()
}

// editDone completes editing and copies the active edited text to the text;
// called when the return key is pressed or goes out of focus
func (ed *Editor) editDone() {
	if ed.Buffer != nil {
		ed.Buffer.editDone()
	}
	ed.clearSelected()
	ed.clearCursor()
	ed.SendChange()
}

// reMarkup triggers a complete re-markup of the entire text --
// can do this when needed if the markup gets off due to multi-line
// formatting issues -- via Recenter key
func (ed *Editor) reMarkup() {
	if ed.Buffer == nil {
		return
	}
	ed.Buffer.ReMarkup()
}

// IsNotSaved returns true if buffer was changed (edited) since last Save.
func (ed *Editor) IsNotSaved() bool {
	return ed.Buffer != nil && ed.Buffer.IsNotSaved()
}

// Clear resets all the text in the buffer for this editor.
func (ed *Editor) Clear() {
	if ed.Buffer == nil {
		return
	}
	ed.Buffer.SetText([]byte{})
}

///////////////////////////////////////////////////////////////////////////////
//  Buffer communication

// resetState resets all the random state variables, when opening a new buffer etc
func (ed *Editor) resetState() {
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

// SetBuffer sets the [Buffer] that this is an editor of, and interconnects their events.
func (ed *Editor) SetBuffer(buf *Buffer) *Editor {
	if ed == nil || buf != nil && ed.Buffer == buf {
		return ed
	}
	// had := false
	if ed.Buffer != nil {
		// had = true
		ed.Buffer.deleteEditor(ed)
	}
	ed.Buffer = buf
	ed.resetState()
	if buf != nil {
		buf.addEditor(ed)
		bhl := len(buf.posHistory)
		if bhl > 0 {
			cp := buf.posHistory[bhl-1]
			ed.posHistoryIndex = bhl - 1
			ed.SetCursorShow(cp)
		} else {
			ed.SetCursorShow(lexer.Pos{})
		}
	}
	ed.layoutAllLines()
	ed.NeedsLayout()
	return ed
}

// linesInserted inserts new lines of text and reformats them
func (ed *Editor) linesInserted(tbe *text.Edit) {
	stln := tbe.Reg.Start.Ln + 1
	nsz := (tbe.Reg.End.Ln - tbe.Reg.Start.Ln)
	if stln > len(ed.renders) { // invalid
		return
	}
	ed.renders = slices.Insert(ed.renders, stln, make([]paint.Text, nsz)...)

	// Offs
	tmpof := make([]float32, nsz)
	ov := float32(0)
	if stln < len(ed.offsets) {
		ov = ed.offsets[stln]
	} else {
		ov = ed.offsets[len(ed.offsets)-1]
	}
	for i := range tmpof {
		tmpof[i] = ov
	}
	ed.offsets = slices.Insert(ed.offsets, stln, tmpof...)

	ed.NumLines += nsz
	ed.NeedsLayout()
}

// linesDeleted deletes lines of text and reformats remaining one
func (ed *Editor) linesDeleted(tbe *text.Edit) {
	stln := tbe.Reg.Start.Ln
	edln := tbe.Reg.End.Ln
	dsz := edln - stln

	ed.renders = append(ed.renders[:stln], ed.renders[edln:]...)
	ed.offsets = append(ed.offsets[:stln], ed.offsets[edln:]...)

	ed.NumLines -= dsz
	ed.NeedsLayout()
}

// bufferSignal receives a signal from the Buffer when the underlying text
// is changed.
func (ed *Editor) bufferSignal(sig bufferSignals, tbe *text.Edit) {
	switch sig {
	case bufferDone:
	case bufferNew:
		ed.resetState()
		ed.SetCursorShow(ed.CursorPos)
		ed.NeedsLayout()
	case bufferMods:
		ed.NeedsLayout()
	case bufferInsert:
		if ed == nil || ed.This == nil || !ed.IsVisible() {
			return
		}
		ndup := ed.renders == nil
		// fmt.Printf("ed %v got %v\n", ed.Nm, tbe.Reg.Start)
		if tbe.Reg.Start.Ln != tbe.Reg.End.Ln {
			// fmt.Printf("ed %v lines insert %v - %v\n", ed.Nm, tbe.Reg.Start, tbe.Reg.End)
			ed.linesInserted(tbe) // triggers full layout
		} else {
			ed.layoutLine(tbe.Reg.Start.Ln) // triggers layout if line width exceeds
		}
		if ndup {
			ed.Update()
		}
	case bufferDelete:
		if ed == nil || ed.This == nil || !ed.IsVisible() {
			return
		}
		ndup := ed.renders == nil
		if tbe.Reg.Start.Ln != tbe.Reg.End.Ln {
			ed.linesDeleted(tbe) // triggers full layout
		} else {
			ed.layoutLine(tbe.Reg.Start.Ln)
		}
		if ndup {
			ed.Update()
		}
	case bufferMarkupUpdated:
		ed.NeedsLayout() // comes from another goroutine
	case bufferClosed:
		ed.SetBuffer(nil)
	}
}

///////////////////////////////////////////////////////////////////////////////
//    Undo / Redo

// undo undoes previous action
func (ed *Editor) undo() {
	tbes := ed.Buffer.undo()
	if tbes != nil {
		tbe := tbes[len(tbes)-1]
		if tbe.Delete { // now an insert
			ed.SetCursorShow(tbe.Reg.End)
		} else {
			ed.SetCursorShow(tbe.Reg.Start)
		}
	} else {
		ed.cursorMovedEvent() // updates status..
		ed.scrollCursorToCenterIfHidden()
	}
	ed.savePosHistory(ed.CursorPos)
	ed.NeedsRender()
}

// redo redoes previously undone action
func (ed *Editor) redo() {
	tbes := ed.Buffer.redo()
	if tbes != nil {
		tbe := tbes[len(tbes)-1]
		if tbe.Delete {
			ed.SetCursorShow(tbe.Reg.Start)
		} else {
			ed.SetCursorShow(tbe.Reg.End)
		}
	} else {
		ed.scrollCursorToCenterIfHidden()
	}
	ed.savePosHistory(ed.CursorPos)
	ed.NeedsRender()
}

// styleEditor applies the editor styles.
func (ed *Editor) styleEditor() {
	if ed.NeedsRebuild() {
		highlighting.UpdateFromTheme()
		if ed.Buffer != nil {
			ed.Buffer.SetHighlighting(highlighting.StyleDefault)
		}
	}
	ed.WidgetBase.Style()
	ed.CursorWidth.ToDots(&ed.Styles.UnitContext)
}

func (ed *Editor) Style() {
	ed.styleEditor()
	ed.styleSizes()
}
