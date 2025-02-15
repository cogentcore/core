// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package textcore

import (
	"image"

	"cogentcore.org/core/base/reflectx"
	"cogentcore.org/core/colors"
	"cogentcore.org/core/core"
	"cogentcore.org/core/cursors"
	"cogentcore.org/core/events"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/abilities"
	"cogentcore.org/core/styles/states"
	"cogentcore.org/core/styles/units"
	"cogentcore.org/core/text/highlighting"
	"cogentcore.org/core/text/lines"
	"cogentcore.org/core/text/shaped"
	"cogentcore.org/core/text/textpos"
)

// TODO: move these into an editor settings object
var (
	// Maximum amount of clipboard history to retain
	clipboardHistoryMax = 100 // `default:"100" min:"0" max:"1000" step:"5"`
)

// Editor is a widget with basic infrastructure for viewing and editing
// [lines.Lines] of monospaced text, used in [texteditor.Editor] and
// terminal. There can be multiple Editor widgets for each lines buffer.
//
// Use NeedsRender to drive an render update for any change that does
// not change the line-level layout of the text.
//
// All updating in the Editor should be within a single goroutine,
// as it would require extensive protections throughout code otherwise.
type Editor struct { //core:embedder
	core.Frame

	// Lines is the text lines content for this editor.
	Lines *lines.Lines `set:"-" json:"-" xml:"-"`

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

	// renders is a slice of shaped.Lines representing the renders of the
	// visible text lines, with one render per line (each line could visibly
	// wrap-around, so these are logical lines, not display lines).
	renders []*shaped.Lines

	// viewId is the unique id of the Lines view.
	viewId int

	// charSize is the render size of one character (rune).
	// Y = line height, X = total glyph advance.
	charSize math32.Vector2

	// visSizeAlloc is the Geom.Size.Alloc.Total subtracting extra space,
	// available for rendering text lines and line numbers.
	visSizeAlloc math32.Vector2

	// lastVisSizeAlloc is the last visSizeAlloc used in laying out lines.
	// It is used to trigger a new layout only when needed.
	lastVisSizeAlloc math32.Vector2

	// visSize is the height in lines and width in chars of the visible area.
	visSize image.Point

	// linesSize is the height in lines and width in chars of the Lines text area,
	// (including line numbers), which can be larger than the visSize.
	linesSize image.Point

	// lineNumberOffset is the horizontal offset in chars for the start of text
	// after line numbers. This is 0 if no line numbers.
	lineNumberOffset int

	// totalSize is total size of all text, including line numbers,
	// multiplied by charSize.
	totalSize math32.Vector2

	// lineNumberDigits is the number of line number digits needed.
	lineNumberDigits int

	// lineNumberRenders are the renderers for line numbers, per visible line.
	lineNumberRenders []*shaped.Lines

	/*
		// CursorPos is the current cursor position.
		CursorPos textpos.Pos `set:"-" edit:"-" json:"-" xml:"-"`

		// cursorTarget is the target cursor position for externally set targets.
		// It ensures that the target position is visible.
		cursorTarget textpos.Pos

		// cursorColumn is the desired cursor column, where the cursor was last when moved using left / right arrows.
		// It is used when doing up / down to not always go to short line columns.
		cursorColumn int

		// posHistoryIndex is the current index within PosHistory.
		posHistoryIndex int

		// selectStart is the starting point for selection, which will either be the start or end of selected region
		// depending on subsequent selection.
		selectStart textpos.Pos

		// SelectRegion is the current selection region.
		SelectRegion lines.Region `set:"-" edit:"-" json:"-" xml:"-"`

		// previousSelectRegion is the previous selection region that was actually rendered.
		// It is needed to update the render.
		previousSelectRegion lines.Region

		// Highlights is a slice of regions representing the highlighted regions, e.g., for search results.
		Highlights []lines.Region `set:"-" edit:"-" json:"-" xml:"-"`

		// scopelights is a slice of regions representing the highlighted regions specific to scope markers.
		scopelights []lines.Region

		// LinkHandler handles link clicks.
		// If it is nil, they are sent to the standard web URL handler.
		LinkHandler func(tl *rich.Link)

		// ISearch is the interactive search data.
		ISearch ISearch `set:"-" edit:"-" json:"-" xml:"-"`

		// QReplace is the query replace data.
		QReplace QReplace `set:"-" edit:"-" json:"-" xml:"-"`

		// selectMode is a boolean indicating whether to select text as the cursor moves.
		selectMode bool

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
	*/
}

func (ed *Editor) WidgetValue() any { return ed.Lines.Text() }

func (ed *Editor) SetWidgetValue(value any) error {
	ed.Lines.SetString(reflectx.ToString(value))
	return nil
}

func (ed *Editor) Init() {
	ed.Frame.Init()
	ed.AddContextMenu(ed.contextMenu)
	ed.SetLines(lines.NewLines(80))
	ed.Styler(func(s *styles.Style) {
		s.SetAbilities(true, abilities.Activatable, abilities.Focusable, abilities.Hoverable, abilities.Slideable, abilities.DoubleClickable, abilities.TripleClickable)
		s.SetAbilities(false, abilities.ScrollableUnfocused)
		ed.CursorWidth.Dp(1)
		ed.LineNumberColor = colors.Uniform(colors.Transparent)
		ed.SelectColor = colors.Scheme.Select.Container
		ed.HighlightColor = colors.Scheme.Warn.Container
		ed.CursorColor = colors.Scheme.Primary.Base

		s.Cursor = cursors.Text
		s.VirtualKeyboard = styles.KeyboardMultiLine
		// if core.SystemSettings.Editor.WordWrap {
		// 	s.Text.WhiteSpace = styles.WhiteSpacePreWrap
		// } else {
		// 	s.Text.WhiteSpace = styles.WhiteSpacePre
		// }
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
		if s.IsReadOnly() {
			s.Background = colors.Scheme.SurfaceContainer
		}
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
	if ed.Lines != nil {
		ed.Lines.EditDone()
	}
	ed.clearSelected()
	ed.clearCursor()
	ed.SendChange()
}

// reMarkup triggers a complete re-markup of the entire text --
// can do this when needed if the markup gets off due to multi-line
// formatting issues -- via Recenter key
func (ed *Editor) reMarkup() {
	if ed.Lines == nil {
		return
	}
	ed.Lines.ReMarkup()
}

// IsNotSaved returns true if buffer was changed (edited) since last Save.
func (ed *Editor) IsNotSaved() bool {
	return ed.Lines != nil && ed.Lines.IsNotSaved()
}

// Clear resets all the text in the buffer for this editor.
func (ed *Editor) Clear() {
	if ed.Lines == nil {
		return
	}
	ed.Lines.SetText([]byte{})
}

// resetState resets all the random state variables, when opening a new buffer etc
func (ed *Editor) resetState() {
	ed.SelectReset()
	ed.Highlights = nil
	ed.ISearch.On = false
	ed.QReplace.On = false
	if ed.Lines == nil || ed.lastFilename != ed.Lines.Filename { // don't reset if reopening..
		ed.CursorPos = textpos.Pos{}
	}
}

// SetLines sets the [lines.Lines] that this is an editor of,
// creating a new view for this editor and connecting to events.
func (ed *Editor) SetLines(buf *lines.Lines) *Editor {
	oldbuf := ed.Lines
	if ed == nil || (buf != nil && oldbuf == buf) {
		return ed
	}
	if oldbuf != nil {
		oldbuf.DeleteView(ed.viewId)
	}
	ed.Lines = buf
	ed.resetState()
	if buf != nil {
		ed.viewId = buf.NewView()
		// bhl := len(buf.posHistory)
		// if bhl > 0 {
		// 	cp := buf.posHistory[bhl-1]
		// 	ed.posHistoryIndex = bhl - 1
		// 	buf.Unlock()
		// 	ed.SetCursorShow(cp)
		// } else {
		// 	ed.SetCursorShow(textpos.Pos{})
		// }
	}
	ed.layoutAllLines() // relocks
	ed.NeedsRender()
	return ed
}

// bufferSignal receives a signal from the Buffer when the underlying text
// is changed.
func (ed *Editor) bufferSignal(sig bufferSignals, tbe *lines.Edit) {
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
		if tbe.Reg.Start.Line != tbe.Reg.End.Line {
			// fmt.Printf("ed %v lines insert %v - %v\n", ed.Nm, tbe.Reg.Start, tbe.Reg.End)
			ed.linesInserted(tbe) // triggers full layout
		} else {
			ed.layoutLine(tbe.Reg.Start.Line) // triggers layout if line width exceeds
		}
		if ndup {
			ed.Update()
		}
	case bufferDelete:
		if ed == nil || ed.This == nil || !ed.IsVisible() {
			return
		}
		ndup := ed.renders == nil
		if tbe.Reg.Start.Line != tbe.Reg.End.Line {
			ed.linesDeleted(tbe) // triggers full layout
		} else {
			ed.layoutLine(tbe.Reg.Start.Line)
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
	tbes := ed.Lines.undo()
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
	tbes := ed.Lines.redo()
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
		if ed.Lines != nil {
			ed.Lines.SetHighlighting(highlighting.StyleDefault)
		}
	}
	ed.Frame.Style()
	ed.CursorWidth.ToDots(&ed.Styles.UnitContext)
}

func (ed *Editor) Style() {
	ed.styleEditor()
	ed.styleSizes()
}
