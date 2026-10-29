// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package terminal

import (
	"image"
	"strings"

	"cogentcore.org/core/base/fileinfo/mimedata"
	"cogentcore.org/core/colors"
	"cogentcore.org/core/core"
	"cogentcore.org/core/cursors"
	"cogentcore.org/core/events"
	"cogentcore.org/core/events/key"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/keymap"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/abilities"
	"cogentcore.org/core/styles/states"
	"cogentcore.org/core/styles/units"
	"cogentcore.org/core/text/terminal/vt"
	"cogentcore.org/core/text/textcore"
	"cogentcore.org/core/text/textpos"
)

// View is a view of the terminal text, which is either the lines view,
// with the scrollback and the primary screen as lines of text that are
// wrapped to the view width, or the grid view of the alternate screen
// that full-screen programs use. All key events go to the program,
// except for the keys for copying and pasting, and for scrolling the
// lines view (Shift with PageUp, PageDown, Home, and End). The text can
// be selected with the mouse, unless the program uses the mouse, in
// which case the Shift key can be held to select.
type View struct {
	textcore.Editor

	// Terminal is the terminal of this view.
	Terminal *Terminal `set:"-" display:"-" json:"-" xml:"-"`

	// isGrid is whether this is the grid view.
	isGrid bool

	// lastKeyDown is the code of the last key down event,
	// for Alt with a character.
	lastKeyDown key.Codes

	// mouseDown is the button that is down, reported to the program.
	mouseDown int

	// scrollDelta accumulates mouse wheel scrolling, in lines.
	scrollDelta float32
}

func (vw *View) Init() {
	vw.Editor.Init()
	vw.mouseDown = mouseNoButton
	vw.SetReadOnly(true)
	vw.Styler(func(s *styles.Style) {
		s.SetAbilities(true, abilities.Focusable)
		s.Cursor = cursors.Text
		s.Text.LineHeight = 1 // the backgrounds of lines touch
		s.Border.Radius = styles.BorderRadiusLarge // as in the editor
		s.Border.Width.Zero()
		s.MaxBorder.Width.Set(units.Dp(2)) // the focus border, as in the editor
		s.Padding.Set(units.Em(0.5)) // room for the radius, as in the editor
		s.Background = colors.Scheme.Surface
		s.Grow.Set(1, 1)
		s.Min.X.Ch(20)
		s.Min.Y.Em(4)
		if !vw.isGrid {
			s.Overflow.Y = styles.OverflowScroll // stable width
		}
	})
	vw.On(events.KeyDown, func(e events.Event) {
		vw.lastKeyDown = e.KeyCode()
	})
	vw.OnKeyChord(func(e events.Event) {
		vw.keyChord(e)
	})
	vw.OnFocus(func(e events.Event) {
		vw.Terminal.focusChanged(true)
	})
	vw.OnFocusLost(func(e events.Event) {
		vw.Terminal.focusChanged(false)
	})
	vw.handleMouse()
	vw.ContextMenus = nil // not those of the editor
	vw.AddContextMenu(vw.contextMenu)
	vw.AddContextMenu(func(m *core.Scene, pos image.Point) {
		// those of the terminal, first
		cms := vw.Terminal.ContextMenus
		for i := len(cms) - 1; i >= 0; i-- {
			if m.NumChildren() > 0 {
				core.NewSeparator(m)
			}
			cms[i](m, pos)
		}
	})
}

// contextMenu adds the items of the terminal to the context menu.
func (vw *View) contextMenu(m *core.Scene, pos image.Point) {
	tm := vw.Terminal
	if m.NumChildren() > 0 {
		core.NewSeparator(m)
	}
	core.NewButton(m).SetText("Copy").SetIcon(icons.ContentCopy).
		SetShortcut(ShortcutChord(keymap.Copy)).SetState(!vw.HasSelection(), states.Disabled).
		OnClick(func(e events.Event) {
			vw.copy()
		})
	core.NewButton(m).SetText("Paste").SetIcon(icons.ContentPaste).
		SetShortcut(ShortcutChord(keymap.Paste)).SetState(vw.Clipboard().IsEmpty(), states.Disabled).
		OnClick(func(e events.Event) {
			tm.PasteClipboard()
		})
	core.NewButton(m).SetText("Clear scrollback").SetIcon(icons.ClearAll).
		SetShortcut(ShortcutChord(keymap.Kill)).
		OnClick(func(e events.Event) {
			tm.ClearScrollback()
		})
}

// SizeFinal resizes the terminal to fit the lines view.
func (vw *View) SizeFinal() {
	vw.Editor.SizeFinal()
	if !vw.isGrid && vw.Terminal != nil {
		vw.Terminal.viewSized(vw.VisibleSize())
	}
}

// keyChord handles a key chord event.
func (vw *View) keyChord(e events.Event) {
	tm := vw.Terminal
	if tm == nil {
		return
	}
	if vw.shortcut(e) {
		e.SetHandled()
		return
	}
	mods := e.Modifiers()
	if mods.HasFlag(key.Meta) { // app shortcuts
		return
	}
	code := e.KeyCode()
	if code == key.CodeUnknown && mods.HasFlag(key.Alt) {
		code = vw.lastKeyDown
	}
	md := tm.modes()
	b := keyBytes(e.KeyRune(), code, mods, &md, tm.AltAsMeta)
	if b == nil {
		return
	}
	e.SetHandled()
	if tm.restartOnKey() {
		return
	}
	vw.SelectReset()
	tm.send(b, true)
}

// shortcut handles the shortcut keys of the terminal GUI,
// returning true if the key is one.
func (vw *View) shortcut(e events.Event) bool {
	tm := vw.Terminal
	mods := e.Modifiers()
	switch KeyFunction(e) {
	case keymap.Copy:
		vw.copy()
		return true
	case keymap.Paste:
		tm.PasteClipboard()
		return true
	case keymap.Kill:
		tm.ClearScrollback()
		return true
	}
	// the Copy key copies when there is a selection, even if it goes to
	// the program otherwise (Control+C), as in Windows Terminal
	if keymap.Of(e.KeyChord()) == keymap.Copy && vw.HasSelection() {
		vw.copy()
		return true
	}
	if !mods.HasFlag(key.Shift) || vw.isGrid {
		return false
	}
	lv := tm.linesView
	switch e.KeyCode() {
	case key.CodePageUp:
		lv.SetScrollPos(lv.ScrollPos() - float32(max(1, lv.VisibleSize().Y-1)))
	case key.CodePageDown:
		lv.SetScrollPos(lv.ScrollPos() + float32(max(1, lv.VisibleSize().Y-1)))
		if lv.IsScrolledToEnd() {
			lv.SetCursorTarget(textpos.PosErr)
		}
	case key.CodeHome:
		lv.SetScrollPos(0)
	case key.CodeEnd:
		lv.SetCursorTarget(textpos.PosErr)
	default:
		return false
	}
	return true
}

// copy copies the selected text to the clipboard.
func (vw *View) copy() {
	tbe := vw.Selection()
	if tbe == nil {
		return
	}
	s := strings.ReplaceAll(string(tbe.ToBytes()), string(wideFiller), "")
	vw.Clipboard().Write(mimedata.NewText(s))
}

// mouseReporting returns whether mouse events are reported to the
// program, which is only done in the grid view when the program has
// asked for them, and the Shift key is not down.
func (vw *View) mouseReporting(e events.Event) bool {
	if !vw.isGrid || e.HasAnyModifier(key.Shift) {
		return false
	}
	md := vw.Terminal.modes()
	return md.Mouse != vt.MouseNone
}

// cellAt returns the terminal row and column at the given event position,
// for the grid view.
func (vw *View) cellAt(e events.Event) (row, col int) {
	pt := e.Pos()
	pos := vw.PixelToCursor(vw.PointToRelPos(pt))
	row = max(0, pos.Line)
	cs := vw.CharSize()
	if cs.X <= 0 {
		return row, 0
	}
	st := vw.CharStartPos(textpos.Pos{Line: row})
	col = max(0, int(math32.Floor((float32(pt.X)-st.X)/cs.X)))
	return row, col
}

// reportMouse reports the given mouse event to the program,
// returning true if it was reported.
func (vw *View) reportMouse(e events.Event, action mouseAction, button int) bool {
	if !vw.mouseReporting(e) {
		return false
	}
	md := vw.Terminal.modes()
	row, col := vw.cellAt(e)
	if b := mouseBytes(&md, action, button, row, col, e.Modifiers()); b != nil {
		vw.Terminal.send(b, false)
	}
	return true
}

// mouseButton returns the terminal mouse button for the event.
func mouseButton(e events.Event) int {
	switch e.MouseButton() {
	case events.Middle:
		return mouseMiddle
	case events.Right:
		return mouseRight
	}
	return mouseLeft
}

// handleMouse handles the mouse events that are reported to the
// program, and the mouse wheel in the grid view.
func (vw *View) handleMouse() {
	vw.On(events.MouseDown, func(e events.Event) {
		b := mouseButton(e)
		if vw.reportMouse(e, mousePress, b) {
			vw.mouseDown = b
			vw.SetFocus()
			e.SetHandled()
		}
	})
	vw.On(events.MouseUp, func(e events.Event) {
		if vw.mouseDown != mouseNoButton && vw.reportMouse(e, mouseRelease, vw.mouseDown) {
			e.SetHandled()
		}
		vw.mouseDown = mouseNoButton
	})
	vw.On(events.MouseMove, func(e events.Event) {
		if vw.reportMouse(e, mouseMotion, vw.mouseDown) {
			e.SetHandled()
		}
	})
	vw.On(events.MouseDrag, func(e events.Event) {
		if vw.reportMouse(e, mouseMotion, vw.mouseDown) {
			e.SetHandled()
		}
	})
	// the events for selection are not used when reporting
	for _, typ := range []events.Types{events.Click, events.DoubleClick, events.TripleClick, events.SlideStart, events.SlideMove, events.SlideStop, events.ContextMenu} {
		vw.On(typ, func(e events.Event) {
			if vw.mouseReporting(e) {
				e.SetHandled()
			}
		})
	}
	vw.On(events.Scroll, func(e events.Event) {
		if !vw.isGrid {
			return
		}
		e.SetHandled()
		se := e.(*events.MouseScroll)
		cs := vw.CharSize()
		if cs.Y <= 0 {
			return
		}
		vw.scrollDelta += se.Delta.Y / cs.Y
		n := int(vw.scrollDelta)
		if n == 0 {
			return
		}
		vw.scrollDelta -= float32(n)
		md := vw.Terminal.modes()
		button, keyCode := mouseWheelDown, key.CodeDownArrow
		if n < 0 {
			n = -n
			button, keyCode = mouseWheelUp, key.CodeUpArrow
		}
		for range n {
			if md.Mouse != vt.MouseNone {
				row, col := vw.cellAt(e)
				vw.Terminal.send(mouseBytes(&md, mousePress, button, row, col, e.Modifiers()), false)
			} else if md.AltScroll {
				vw.Terminal.send(keyBytes(0, keyCode, 0, &md, false), false)
			}
		}
	})
}
