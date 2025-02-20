// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package textcore

//go:generate core generate

import (
	"image"
	"sync"

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
	"cogentcore.org/core/text/rich"
	"cogentcore.org/core/text/text"
	"cogentcore.org/core/text/textpos"
)

// TODO: move these into an editor settings object
var (
	// Maximum amount of clipboard history to retain
	clipboardHistoryMax = 100 // `default:"100" min:"0" max:"1000" step:"5"`
)

// Base is a widget with basic infrastructure for viewing and editing
// [lines.Lines] of monospaced text, used in [textcore.Editor] and
// terminal. There can be multiple Base widgets for each lines buffer.
//
// Use NeedsRender to drive an render update for any change that does
// not change the line-level layout of the text.
//
// All updating in the Base should be within a single goroutine,
// as it would require extensive protections throughout code otherwise.
type Base struct { //core:embedder
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
	// (excluding line numbers), which can be larger than the visSize.
	linesSize image.Point

	// scrollPos is the position of the scrollbar, in units of lines of text.
	// fractional scrolling is supported.
	scrollPos float32

	// hasLineNumbers indicates that this editor has line numbers
	// (per [Editor] option)
	hasLineNumbers bool

	// lineNumberOffset is the horizontal offset in chars for the start of text
	// after line numbers. This is 0 if no line numbers.
	lineNumberOffset int

	// totalSize is total size of all text, including line numbers,
	// multiplied by charSize.
	totalSize math32.Vector2

	// lineNumberDigits is the number of line number digits needed.
	lineNumberDigits int

	// CursorPos is the current cursor position.
	CursorPos textpos.Pos `set:"-" edit:"-" json:"-" xml:"-"`

	// blinkOn oscillates between on and off for blinking.
	blinkOn bool

	// cursorMu is a mutex protecting cursor rendering, shared between blink and main code.
	cursorMu sync.Mutex

	// cursorTarget is the target cursor position for externally set targets.
	// It ensures that the target position is visible.
	cursorTarget textpos.Pos

	// cursorColumn is the desired cursor column, where the cursor was
	// last when moved using left / right arrows.
	// It is used when doing up / down to not always go to short line columns.
	cursorColumn int

	// posHistoryIndex is the current index within PosHistory.
	posHistoryIndex int

	// selectStart is the starting point for selection, which will either
	// be the start or end of selected region depending on subsequent selection.
	selectStart textpos.Pos

	// SelectRegion is the current selection region.
	SelectRegion textpos.Region `set:"-" edit:"-" json:"-" xml:"-"`

	// previousSelectRegion is the previous selection region that was actually rendered.
	// It is needed to update the render.
	previousSelectRegion textpos.Region

	// Highlights is a slice of regions representing the highlighted
	// regions, e.g., for search results.
	Highlights []textpos.Region `set:"-" edit:"-" json:"-" xml:"-"`

	// scopelights is a slice of regions representing the highlighted
	// regions specific to scope markers.
	scopelights []textpos.Region

	// LinkHandler handles link clicks.
	// If it is nil, they are sent to the standard web URL handler.
	LinkHandler func(tl *rich.Hyperlink)

	// selectMode is a boolean indicating whether to select text as the cursor moves.
	selectMode bool

	// hasLinks is a boolean indicating if at least one of the renders has links.
	// It determines if we set the cursor for hand movements.
	hasLinks bool

	// lastWasTabAI indicates that last key was a Tab auto-indent
	lastWasTabAI bool

	// lastWasUndo indicates that last key was an undo
	lastWasUndo bool

	// targetSet indicates that the CursorTarget is set
	targetSet bool

	lastRecenter   int
	lastAutoInsert rune
	lastFilename   string
}

func (ed *Base) WidgetValue() any { return ed.Lines.Text() }

func (ed *Base) SetWidgetValue(value any) error {
	ed.Lines.SetString(reflectx.ToString(value))
	return nil
}

func (ed *Base) Init() {
	ed.Frame.Init()
	ed.Styles.Font.Family = rich.Monospace // critical
	ed.SetLines(lines.NewLines())
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
		// if core.SystemSettings.Base.WordWrap {
		// 	s.Text.WhiteSpace = styles.WhiteSpacePreWrap
		// } else {
		// 	s.Text.WhiteSpace = styles.WhiteSpacePre
		// }
		s.Text.WhiteSpace = text.WrapNever
		s.Font.Family = rich.Monospace
		s.Grow.Set(1, 0)
		s.Overflow.Set(styles.OverflowAuto) // absorbs all
		s.Border.Radius = styles.BorderRadiusLarge
		s.Margin.Zero()
		s.Padding.Set(units.Em(0.5))
		s.Align.Content = styles.Start
		s.Align.Items = styles.Start
		s.Text.Align = text.Start
		s.Text.AlignV = text.Start
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

	ed.OnClose(func(e events.Event) {
		ed.editDone()
	})

	// ed.Updater(ed.NeedsRender) // todo: delete me
}

func (ed *Base) Destroy() {
	ed.stopCursor()
	ed.Frame.Destroy()
}

func (ed *Base) NumLines() int {
	if ed.Lines != nil {
		return ed.Lines.NumLines()
	}
	return 0
}

// editDone completes editing and copies the active edited text to the text;
// called when the return key is pressed or goes out of focus
func (ed *Base) editDone() {
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
func (ed *Base) reMarkup() {
	if ed.Lines == nil {
		return
	}
	ed.Lines.ReMarkup()
}

// IsNotSaved returns true if buffer was changed (edited) since last Save.
func (ed *Base) IsNotSaved() bool {
	return ed.Lines != nil && ed.Lines.IsNotSaved()
}

// Clear resets all the text in the buffer for this editor.
func (ed *Base) Clear() {
	if ed.Lines == nil {
		return
	}
	ed.Lines.SetText([]byte{})
}

// resetState resets all the random state variables, when opening a new buffer etc
func (ed *Base) resetState() {
	ed.SelectReset()
	ed.Highlights = nil
	if ed.Lines == nil || ed.lastFilename != ed.Lines.Filename() { // don't reset if reopening..
		ed.CursorPos = textpos.Pos{}
	}
}

// SendInput sends the [events.Input] event, for fine-grained updates.
func (ed *Base) SendInput() {
	ed.Send(events.Input, nil)
}

// SendChange sends the [events.Change] event, for big changes.
// func (ed *Base) SendChange() {
// 	ed.Send(events.Change, nil)
// }

// SendClose sends the [events.Close] event, when lines buffer is closed.
func (ed *Base) SendClose() {
	ed.Send(events.Close, nil)
}

// SetLines sets the [lines.Lines] that this is an editor of,
// creating a new view for this editor and connecting to events.
func (ed *Base) SetLines(buf *lines.Lines) *Base {
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
		buf.Settings.EditorSettings = core.SystemSettings.Editor
		wd := ed.linesSize.X
		if wd == 0 {
			wd = 80
		}
		ed.viewId = buf.NewView(wd)
		buf.OnChange(ed.viewId, func(e events.Event) {
			ed.NeedsRender()
			ed.SendChange()
		})
		buf.OnInput(ed.viewId, func(e events.Event) {
			ed.NeedsRender()
			ed.SendInput()
		})
		buf.OnClose(ed.viewId, func(e events.Event) {
			ed.SetLines(nil)
			ed.SendClose()
		})
		phl := buf.PosHistoryLen()
		if phl > 0 {
			cp, _ := buf.PosHistoryAt(phl - 1)
			ed.posHistoryIndex = phl - 1
			ed.SetCursorShow(cp)
		} else {
			ed.SetCursorShow(textpos.Pos{})
		}
	} else {
		ed.viewId = -1
	}
	ed.NeedsRender()
	return ed
}

// styleBase applies the editor styles.
func (ed *Base) styleBase() {
	if ed.NeedsRebuild() {
		highlighting.UpdateFromTheme()
		if ed.Lines != nil {
			ed.Lines.SetHighlighting(highlighting.StyleDefault)
		}
	}
	ed.Frame.Style()
	ed.CursorWidth.ToDots(&ed.Styles.UnitContext)
}

func (ed *Base) Style() {
	ed.styleBase()
	ed.styleSizes()
}
