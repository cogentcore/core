// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"fmt"
	"image"
	"log/slog"
	"reflect"
	"slices"
	"sync"
	"time"
	"unicode"

	"cogentcore.org/core/base/fileinfo/mimedata"
	"cogentcore.org/core/colors"
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
	"cogentcore.org/core/text/parse/complete"
	"cogentcore.org/core/text/rich"
	"cogentcore.org/core/text/shaped"
	"cogentcore.org/core/text/text"
	"cogentcore.org/core/text/textpos"
	"cogentcore.org/core/tree"
)

// TextField is a widget for editing a line of text.
//
// With the default [styles.WhiteSpaceNormal] setting,
// text will wrap onto multiple lines as needed. You can
// call [styles.Style.SetTextWrap](false) to force everything
// to be rendered on a single line. With multi-line wrapped text,
// the text is still treated as a single contiguous line of wrapped text.
type TextField struct { //core:embedder
	Frame

	// Type is the styling type of the text field.
	Type TextFieldTypes

	// Placeholder is the text that is displayed
	// when the text field is empty.
	Placeholder string

	// Validator is a function used to validate the input
	// of the text field. If it returns a non-nil error,
	// then an error color, icon, and tooltip will be displayed.
	Validator func() error `json:"-" xml:"-"`

	// LeadingIcon, if specified, indicates to add a button
	// at the start of the text field with this icon.
	// See [TextField.SetLeadingIcon].
	LeadingIcon icons.Icon `set:"-"`

	// LeadingIconOnClick, if specified, is the function to call when
	// the LeadingIcon is clicked. If this is nil, the leading icon
	// will not be interactive. See [TextField.SetLeadingIcon].
	LeadingIconOnClick func(e events.Event) `json:"-" xml:"-"`

	// TrailingIcon, if specified, indicates to add a button
	// at the end of the text field with this icon.
	// See [TextField.SetTrailingIcon].
	TrailingIcon icons.Icon `set:"-"`

	// TrailingIconOnClick, if specified, is the function to call when
	// the TrailingIcon is clicked. If this is nil, the trailing icon
	// will not be interactive. See [TextField.SetTrailingIcon].
	TrailingIconOnClick func(e events.Event) `json:"-" xml:"-"`

	// NoEcho is whether replace displayed characters with bullets
	// to conceal text (for example, for a password input). Also
	// see [TextField.SetTypePassword].
	NoEcho bool

	// CursorWidth is the width of the text field cursor.
	// It should be set in a Styler like all other style properties.
	// By default, it is 1dp.
	CursorWidth units.Value

	// CursorColor is the color used for the text field cursor (caret).
	// It should be set in a Styler like all other style properties.
	// By default, it is [colors.Scheme.Primary.Base].
	CursorColor image.Image

	// PlaceholderColor is the color used for the [TextField.Placeholder] text.
	// It should be set in a Styler like all other style properties.
	// By default, it is [colors.Scheme.OnSurfaceVariant].
	PlaceholderColor image.Image

	// complete contains functions and data for text field completion.
	// It must be set using [TextField.SetCompleter].
	complete *Complete

	// text is the last saved value of the text string being edited.
	text string

	// edited is whether the text has been edited relative to the original.
	edited bool

	// editText is the live text string being edited, with the latest modifications.
	editText []rune

	// error is the current validation error of the text field.
	error error

	// effPos is the effective position with any leading icon space added.
	effPos math32.Vector2

	// effSize is the effective size, subtracting any leading and trailing icon space.
	effSize math32.Vector2

	// dispRange is the range of visible text, for scrolling text case (non-wordwrap).
	dispRange textpos.Range

	// cursorPos is the current cursor position as rune index into string.
	cursorPos int

	// cursorLine is the current cursor line position, for word wrap case.
	cursorLine int

	// charWidth is the approximate number of chars that can be
	// displayed at any time, which is computed from the font size.
	charWidth int

	// selectRange is the selected range.
	selectRange textpos.Range

	// selectInit is the initial selection position (where it started).
	selectInit int

	// selectMode is whether to select text as the cursor moves.
	selectMode bool

	// selectModeShift is whether selectmode was turned on because of the shift key.
	selectModeShift bool

	// renderAll is the render version of entire text, for sizing.
	renderAll *shaped.Lines

	// renderVisible is the render version of just the visible text in dispRange.
	renderVisible *shaped.Lines

	// renderedRange is the dispRange last rendered.
	renderedRange textpos.Range

	// number of lines from last render update, for word-wrap version
	numLines int

	// lineHeight is the line height cached during styling.
	lineHeight float32

	// cursorMu is the mutex for updating the cursor between blinker and field.
	cursorMu sync.Mutex

	// undos is the undo manager for the text field.
	undos textFieldUndos

	leadingIconButton, trailingIconButton *Button
}

// TextFieldTypes is an enum containing the
// different possible types of text fields.
type TextFieldTypes int32 //enums:enum -trim-prefix TextField

const (
	// TextFieldFilled represents a filled
	// [TextField] with a background color
	// and a bottom border.
	TextFieldFilled TextFieldTypes = iota

	// TextFieldOutlined represents an outlined
	// [TextField] with a border on all sides
	// and no background color.
	TextFieldOutlined
)

// Validator is an interface for types to provide a Validate method
// that is used to validate string [Value]s using [TextField.Validator].
type Validator interface {

	// Validate returns an error if the value is invalid.
	Validate() error
}

func (tf *TextField) WidgetValue() any { return &tf.text }

func (tf *TextField) OnBind(value any, tags reflect.StructTag) {
	if vd, ok := value.(Validator); ok {
		tf.Validator = vd.Validate
	}
}

func (tf *TextField) Init() {
	tf.Frame.Init()
	tf.AddContextMenu(tf.contextMenu)

	tf.Styler(func(s *styles.Style) {
		s.SetAbilities(true, abilities.Activatable, abilities.Focusable, abilities.Hoverable, abilities.Slideable, abilities.DoubleClickable, abilities.TripleClickable)
		s.SetAbilities(false, abilities.ScrollableUnattended)
		tf.CursorWidth.Dp(1)
		tf.PlaceholderColor = colors.Scheme.OnSurfaceVariant
		tf.CursorColor = colors.Scheme.Primary.Base
		s.Cursor = cursors.Text
		s.VirtualKeyboard = styles.KeyboardSingleLine
		s.GrowWrap = false // note: doesn't work with Grow
		s.Grow.Set(1, 0)
		s.Min.Y.Em(1.1)
		s.Min.X.Ch(20)
		s.Max.X.Ch(40)
		s.Gap.Zero()
		s.Padding.Set(units.Dp(8), units.Dp(8))
		if tf.LeadingIcon.IsSet() {
			s.Padding.Left.Dp(12)
		}
		if tf.TrailingIcon.IsSet() {
			s.Padding.Right.Dp(12)
		}
		s.Text.LineHeight = 1.4
		s.Text.Align = text.Start
		s.Align.Items = styles.Center
		s.Color = colors.Scheme.OnSurface
		s.IconSize.Set(units.Em(18.0 / 16))
		switch tf.Type {
		case TextFieldFilled:
			s.Border.Style.Set(styles.BorderNone)
			s.Border.Style.Bottom = styles.BorderSolid
			s.Border.Width.Zero()
			s.Border.Color.Zero()
			s.Border.Radius = styles.BorderRadiusExtraSmallTop
			s.Background = colors.Scheme.SurfaceContainer

			s.MaxBorder = s.Border
			s.MaxBorder.Width.Bottom = units.Dp(2)
			s.MaxBorder.Color.Bottom = colors.Scheme.Primary.Base

			s.Border.Width.Bottom = units.Dp(1)
			s.Border.Color.Bottom = colors.Scheme.OnSurfaceVariant
			if tf.error != nil {
				s.Border.Color.Bottom = colors.Scheme.Error.Base
			}
		case TextFieldOutlined:
			s.Border.Style.Set(styles.BorderSolid)
			s.Border.Radius = styles.BorderRadiusExtraSmall

			s.MaxBorder = s.Border
			s.MaxBorder.Width.Set(units.Dp(2))
			s.MaxBorder.Color.Set(colors.Scheme.Primary.Base)
			s.Border.Width.Set(units.Dp(1))
			if tf.error != nil {
				s.Border.Color.Set(colors.Scheme.Error.Base)
			}
		}
		if tf.IsReadOnly() {
			s.Border.Color.Zero()
			s.Border.Width.Zero()
			s.Border.Radius.Zero()
			s.MaxBorder = s.Border
			s.Background = nil
		}
		if s.Is(states.Selected) {
			s.Background = colors.Scheme.Select.Container
		}
	})
	tf.FinalStyler(func(s *styles.Style) {
		s.SetAbilities(!tf.IsReadOnly(), abilities.Focusable)
	})

	tf.handleKeyEvents()
	tf.OnFirst(events.Change, func(e events.Event) {
		tf.validate()
		if tf.error != nil {
			e.SetHandled()
		}
	})
	tf.OnClick(func(e events.Event) {
		if !tf.IsReadOnly() {
			tf.SetFocus()
		}
		switch e.MouseButton() {
		case events.Left:
			tf.setCursorFromPixel(e.Pos(), e.SelectMode())
		case events.Middle:
			if !tf.IsReadOnly() {
				tf.paste()
			}
		}
		tf.startCursor()
	})
	tf.On(events.DoubleClick, func(e events.Event) {
		if tf.IsReadOnly() {
			return
		}
		if !tf.IsReadOnly() && !tf.StateIs(states.Focused) {
			tf.SetFocus()
		}
		e.SetHandled()
		tf.selectWord()
		tf.startCursor()
	})
	tf.On(events.TripleClick, func(e events.Event) {
		if tf.IsReadOnly() {
			return
		}
		if !tf.IsReadOnly() && !tf.StateIs(states.Focused) {
			tf.SetFocus()
		}
		e.SetHandled()
		tf.selectAll()
		tf.startCursor()
	})
	tf.On(events.SlideStart, func(e events.Event) {
		e.SetHandled()
		tf.SetState(true, states.Sliding)
		if tf.selectMode || e.SelectMode() != events.SelectOne { // extend existing select
			tf.setCursorFromPixel(e.Pos(), e.SelectMode())
		} else {
			tf.cursorPos = tf.pixelToCursor(e.Pos())
			if !tf.selectMode {
				tf.selectModeToggle()
			}
		}
		tf.startCursor()
	})
	tf.On(events.SlideMove, func(e events.Event) {
		e.SetHandled()
		tf.selectMode = true // always
		tf.setCursorFromPixel(e.Pos(), events.SelectOne)
		tf.startCursor()
	})
	tf.OnClose(func(e events.Event) {
		tf.editDone() // todo: this must be protected against something else, for race detector
	})

	tf.Maker(func(p *tree.Plan) {
		if !tf.edited { // if in edit, don't overwrite
			tf.editText = []rune(tf.text)
		}
		if tf.IsReadOnly() {
			return
		}
		if tf.LeadingIcon.IsSet() {
			tree.AddAt(p, "lead-icon", func(w *Button) {
				tf.leadingIconButton = w
				w.SetType(ButtonAction)
				w.Styler(func(s *styles.Style) {
					s.Padding.Zero()
					s.Color = colors.Scheme.OnSurfaceVariant
					s.Margin.SetRight(units.Dp(8))
					s.Font.Size = tf.Styles.Font.Size
					s.IconSize = tf.Styles.IconSize
					if tf.LeadingIconOnClick == nil {
						s.SetAbilities(false, abilities.Activatable, abilities.Focusable, abilities.Hoverable)
						s.Cursor = cursors.None
					}
					// If we are responsible for a positive (non-disabled) state layer
					// (instead of our parent), then we amplify it so that it is clear
					// that we ourself are receiving a state layer amplifying event.
					// Otherwise, we set our state color to that of our parent
					// so that it does not appear as if we are getting interaction ourself;
					// instead, we are a part of our parent and render a background color no
					// different than them.
					if s.Is(states.Hovered) || s.Is(states.Focused) || s.Is(states.Active) {
						s.StateLayer *= 3
					} else {
						s.StateColor = tf.Styles.Color
					}
				})
				w.OnClick(func(e events.Event) {
					if tf.LeadingIconOnClick != nil {
						tf.LeadingIconOnClick(e)
					}
				})
				w.Updater(func() {
					w.SetIcon(tf.LeadingIcon)
				})
			})
		} else {
			tf.leadingIconButton = nil
		}
		if tf.TrailingIcon.IsSet() || tf.error != nil {
			tree.AddAt(p, "trail-icon-stretch", func(w *Stretch) {
				w.Styler(func(s *styles.Style) {
					s.Grow.Set(1, 0)
				})
			})
			tree.AddAt(p, "trail-icon", func(w *Button) {
				tf.trailingIconButton = w
				w.SetType(ButtonAction)
				w.Styler(func(s *styles.Style) {
					s.Padding.Zero()
					s.Color = colors.Scheme.OnSurfaceVariant
					if tf.error != nil {
						s.Color = colors.Scheme.Error.Base
					}
					s.Margin.SetLeft(units.Dp(8))
					s.Font.Size = tf.Styles.Font.Size
					s.IconSize = tf.Styles.IconSize
					if tf.TrailingIconOnClick == nil || tf.error != nil {
						s.SetAbilities(false, abilities.Activatable, abilities.Focusable, abilities.Hoverable)
						s.Cursor = cursors.None
						// need to clear state in case it was set when there
						// was no error
						s.State = 0
					}
					// same reasoning as for leading icon
					if s.Is(states.Hovered) || s.Is(states.Focused) || s.Is(states.Active) {
						s.StateLayer *= 3
					} else {
						s.StateColor = tf.Styles.Color
					}
				})
				w.OnClick(func(e events.Event) {
					if tf.TrailingIconOnClick != nil {
						tf.TrailingIconOnClick(e)
					}
				})
				w.Updater(func() {
					w.SetIcon(tf.TrailingIcon)
					if tf.error != nil {
						w.SetIcon(icons.Error)
					}
				})
			})
		} else {
			tf.trailingIconButton = nil
		}
	})
	tf.Updater(func() {
		tf.renderVisible = nil // ensures re-render
	})
}

func (tf *TextField) Destroy() {
	tf.stopCursor()
	tf.Frame.Destroy()
}

// Text returns the current text of the text field. It applies any unapplied changes
// first, and sends an [events.Change] event if applicable. This is the main end-user
// method to get the current value of the text field.
func (tf *TextField) Text() string {
	tf.editDone()
	return tf.text
}

// SetText sets the text of the text field and reverts any current edits
// to reflect this new text.
func (tf *TextField) SetText(text string) *TextField {
	if tf.text == text && !tf.edited {
		return tf
	}
	tf.text = text
	tf.revert()
	return tf
}

// SetLeadingIcon sets the [TextField.LeadingIcon] to the given icon. If an
// on click function is specified, it also sets the [TextField.LeadingIconOnClick]
// to that function. If no function is specified, it does not override any already
// set function.
func (tf *TextField) SetLeadingIcon(icon icons.Icon, onClick ...func(e events.Event)) *TextField {
	tf.LeadingIcon = icon
	if len(onClick) > 0 {
		tf.LeadingIconOnClick = onClick[0]
	}
	return tf
}

// SetTrailingIcon sets the [TextField.TrailingIcon] to the given icon. If an
// on click function is specified, it also sets the [TextField.TrailingIconOnClick]
// to that function. If no function is specified, it does not override any already
// set function.
func (tf *TextField) SetTrailingIcon(icon icons.Icon, onClick ...func(e events.Event)) *TextField {
	tf.TrailingIcon = icon
	if len(onClick) > 0 {
		tf.TrailingIconOnClick = onClick[0]
	}
	return tf
}

// AddClearButton adds a trailing icon button at the end
// of the text field that clears the text in the text field
// when it is clicked.
func (tf *TextField) AddClearButton() *TextField {
	return tf.SetTrailingIcon(icons.Close, func(e events.Event) {
		tf.clear()
	})
}

// SetTypePassword enables [TextField.NoEcho] and adds a trailing
// icon button at the end of the textfield that toggles [TextField.NoEcho].
// It also sets [styles.Style.VirtualKeyboard] to [styles.KeyboardPassword].
func (tf *TextField) SetTypePassword() *TextField {
	tf.SetNoEcho(true).SetTrailingIcon(icons.Visibility, func(e events.Event) {
		tf.NoEcho = !tf.NoEcho
		if tf.NoEcho {
			tf.TrailingIcon = icons.Visibility
		} else {
			tf.TrailingIcon = icons.VisibilityOff
		}
		if icon := tf.trailingIconButton; icon != nil {
			icon.SetIcon(tf.TrailingIcon).Update()
		}
	}).Styler(func(s *styles.Style) {
		s.VirtualKeyboard = styles.KeyboardPassword
	})
	return tf
}

// textEdited must be called whenever the text is edited.
// it sets the edited flag and ensures a new render of current text.
func (tf *TextField) textEdited() {
	tf.edited = true
	tf.renderVisible = nil
	tf.NeedsRender()
}

// editDone completes editing and copies the active edited text to the [TextField.text].
// It is called when the return key is pressed or the text field goes out of focus.
func (tf *TextField) editDone() {
	if tf.edited {
		tf.edited = false
		tf.text = string(tf.editText)
		tf.SendChange()
		// widget can be killed after SendChange
		if tf.This == nil {
			return
		}
	}
	tf.clearSelected()
	tf.stopCursor()
}

// revert aborts editing and reverts to the last saved text.
func (tf *TextField) revert() {
	tf.renderVisible = nil
	tf.editText = []rune(tf.text)
	tf.edited = false
	tf.dispRange.Start = 0
	tf.dispRange.End = tf.charWidth
	tf.selectReset()
	tf.NeedsRender()
}

// clear clears any existing text.
func (tf *TextField) clear() {
	tf.renderVisible = nil
	tf.edited = true
	tf.editText = tf.editText[:0]
	tf.dispRange.Start = 0
	tf.dispRange.End = 0
	tf.selectReset()
	tf.SetFocus() // this is essential for ensuring that the clear applies after focus is lost..
	tf.NeedsRender()
}

// clearError clears any existing validation error.
func (tf *TextField) clearError() {
	if tf.error == nil {
		return
	}
	tf.error = nil
	tf.Update()
	tf.Send(events.LongHoverEnd) // get rid of any validation tooltip
}

// validate runs [TextField.Validator] and takes any necessary actions
// as a result of that.
func (tf *TextField) validate() {
	if tf.Validator == nil {
		return
	}
	err := tf.Validator()
	if err == nil {
		tf.clearError()
		return
	}
	tf.error = err
	tf.Update()
	// show the error tooltip immediately
	tf.Send(events.LongHoverStart)
}

func (tf *TextField) WidgetTooltip(pos image.Point) (string, image.Point) {
	if tf.error == nil {
		return tf.Tooltip, tf.DefaultTooltipPos()
	}
	return tf.error.Error(), tf.DefaultTooltipPos()
}

////////  Cursor Navigation

func (tf *TextField) updateLinePos() {
	if tf.renderAll == nil {
		return
	}
	tf.cursorLine = tf.renderAll.RuneToLinePos(tf.cursorPos).Line
}

// cursorForward moves the cursor forward
func (tf *TextField) cursorForward(steps int) {
	tf.cursorPos += steps
	if tf.cursorPos > len(tf.editText) {
		tf.cursorPos = len(tf.editText)
	}
	if tf.cursorPos > tf.dispRange.End {
		inc := tf.cursorPos - tf.dispRange.End
		tf.dispRange.End += inc
	}
	tf.updateLinePos()
	if tf.selectMode {
		tf.selectRegionUpdate(tf.cursorPos)
	}
	tf.NeedsRender()
}

// cursorForwardWord moves the cursor forward by words
func (tf *TextField) cursorForwardWord(steps int) {
	tf.cursorPos, _ = textpos.ForwardWord(tf.editText, tf.cursorPos, steps)
	if tf.cursorPos > tf.dispRange.End {
		inc := tf.cursorPos - tf.dispRange.End
		tf.dispRange.End += inc
	}
	tf.updateLinePos()
	if tf.selectMode {
		tf.selectRegionUpdate(tf.cursorPos)
	}
	tf.NeedsRender()
}

// cursorBackward moves the cursor backward
func (tf *TextField) cursorBackward(steps int) {
	tf.cursorPos -= steps
	if tf.cursorPos < 0 {
		tf.cursorPos = 0
	}
	if tf.cursorPos <= tf.dispRange.Start {
		dec := min(tf.dispRange.Start, 8)
		tf.dispRange.Start -= dec
	}
	tf.updateLinePos()
	if tf.selectMode {
		tf.selectRegionUpdate(tf.cursorPos)
	}
	tf.NeedsRender()
}

// cursorBackwardWord moves the cursor backward by words
func (tf *TextField) cursorBackwardWord(steps int) {
	tf.cursorPos, _ = textpos.BackwardWord(tf.editText, tf.cursorPos, steps)
	if tf.cursorPos <= tf.dispRange.Start {
		dec := min(tf.dispRange.Start, 8)
		tf.dispRange.Start -= dec
	}
	tf.updateLinePos()
	if tf.selectMode {
		tf.selectRegionUpdate(tf.cursorPos)
	}
	tf.NeedsRender()
}

// cursorDown moves the cursor down
func (tf *TextField) cursorDown(steps int) {
	if tf.numLines <= 1 {
		return
	}
	if tf.cursorLine >= tf.numLines-1 {
		return
	}
	tf.cursorPos = tf.renderVisible.RuneAtLineDelta(tf.cursorPos, steps)
	tf.updateLinePos()
	if tf.selectMode {
		tf.selectRegionUpdate(tf.cursorPos)
	}
	tf.NeedsRender()
}

// cursorUp moves the cursor up
func (tf *TextField) cursorUp(steps int) {
	if tf.numLines <= 1 {
		return
	}
	if tf.cursorLine <= 0 {
		return
	}
	tf.cursorPos = tf.renderVisible.RuneAtLineDelta(tf.cursorPos, -steps)
	tf.updateLinePos()
	if tf.selectMode {
		tf.selectRegionUpdate(tf.cursorPos)
	}
	tf.NeedsRender()
}

// cursorStart moves the cursor to the start of the text, updating selection
// if select mode is active.
func (tf *TextField) cursorStart() {
	tf.cursorPos = 0
	tf.dispRange.Start = 0
	tf.dispRange.End = min(len(tf.editText), tf.dispRange.Start+tf.charWidth)
	if tf.selectMode {
		tf.selectRegionUpdate(tf.cursorPos)
	}
	tf.NeedsRender()
}

// cursorEnd moves the cursor to the end of the text, updating selection
// if select mode is active.
func (tf *TextField) cursorEnd() {
	ed := len(tf.editText)
	tf.cursorPos = ed
	tf.dispRange.End = len(tf.editText) // try -- display will adjust
	tf.dispRange.Start = max(0, tf.dispRange.End-tf.charWidth)
	if tf.selectMode {
		tf.selectRegionUpdate(tf.cursorPos)
	}
	tf.NeedsRender()
}

// cursorBackspace deletes character(s) immediately before cursor
func (tf *TextField) cursorBackspace(steps int) {
	if tf.hasSelection() {
		tf.deleteSelection()
		return
	}
	if tf.cursorPos < steps {
		steps = tf.cursorPos
	}
	if steps <= 0 {
		return
	}
	tf.editText = append(tf.editText[:tf.cursorPos-steps], tf.editText[tf.cursorPos:]...)
	tf.textEdited()
	tf.cursorBackward(steps)
}

// cursorDelete deletes character(s) immediately after the cursor
func (tf *TextField) cursorDelete(steps int) {
	if tf.hasSelection() {
		tf.deleteSelection()
		return
	}
	if tf.cursorPos+steps > len(tf.editText) {
		steps = len(tf.editText) - tf.cursorPos
	}
	if steps <= 0 {
		return
	}
	tf.editText = append(tf.editText[:tf.cursorPos], tf.editText[tf.cursorPos+steps:]...)
	tf.textEdited()
}

// cursorBackspaceWord deletes words(s) immediately before cursor
func (tf *TextField) cursorBackspaceWord(steps int) {
	if tf.hasSelection() {
		tf.deleteSelection()
		return
	}
	org := tf.cursorPos
	tf.cursorBackwardWord(steps)
	tf.editText = append(tf.editText[:tf.cursorPos], tf.editText[org:]...)
	tf.textEdited()
}

// cursorDeleteWord deletes word(s) immediately after the cursor
func (tf *TextField) cursorDeleteWord(steps int) {
	if tf.hasSelection() {
		tf.deleteSelection()
		return
	}
	// note: no update b/c signal from buf will drive update
	org := tf.cursorPos
	tf.cursorForwardWord(steps)
	tf.editText = append(tf.editText[:tf.cursorPos], tf.editText[org:]...)
	tf.textEdited()
}

// cursorKill deletes text from cursor to end of text
func (tf *TextField) cursorKill() {
	steps := len(tf.editText) - tf.cursorPos
	tf.cursorDelete(steps)
}

////////  Selection

// clearSelected resets both the global selected flag and any current selection
func (tf *TextField) clearSelected() {
	tf.SetState(false, states.Selected)
	tf.selectReset()
}

// hasSelection returns whether there is a selected region of text
func (tf *TextField) hasSelection() bool {
	tf.selectUpdate()
	return tf.selectRange.Start < tf.selectRange.End
}

// selection returns the currently selected text
func (tf *TextField) selection() string {
	if tf.hasSelection() {
		return string(tf.editText[tf.selectRange.Start:tf.selectRange.End])
	}
	return ""
}

// selectModeToggle toggles the SelectMode, updating selection with cursor movement
func (tf *TextField) selectModeToggle() {
	if tf.selectMode {
		tf.selectMode = false
	} else {
		tf.selectMode = true
		tf.selectInit = tf.cursorPos
		tf.selectRange.Start = tf.cursorPos
		tf.selectRange.End = tf.selectRange.Start
	}
}

// shiftSelect sets the selection start if the shift key is down but wasn't previously.
// If the shift key has been released, the selection info is cleared.
func (tf *TextField) shiftSelect(e events.Event) {
	hasShift := e.HasAnyModifier(key.Shift)
	if hasShift && !tf.selectMode {
		tf.selectModeToggle()
		tf.selectModeShift = true
	}
	if !hasShift && tf.selectMode && tf.selectModeShift {
		tf.selectReset()
		tf.selectModeShift = false
	}
}

// selectRegionUpdate updates current select region based on given cursor position
// relative to SelectStart position
func (tf *TextField) selectRegionUpdate(pos int) {
	if pos < tf.selectInit {
		tf.selectRange.Start = pos
		tf.selectRange.End = tf.selectInit
	} else {
		tf.selectRange.Start = tf.selectInit
		tf.selectRange.End = pos
	}
	tf.selectUpdate()
}

// selectAll selects all the text
func (tf *TextField) selectAll() {
	tf.selectRange.Start = 0
	tf.selectInit = 0
	tf.selectRange.End = len(tf.editText)
	if TheApp.SystemPlatform().IsMobile() {
		tf.Send(events.ContextMenu)
	}
	tf.NeedsRender()
}

// selectWord selects the word (whitespace delimited) that the cursor is on
func (tf *TextField) selectWord() {
	sz := len(tf.editText)
	if sz <= 3 {
		tf.selectAll()
		return
	}
	tf.selectRange = textpos.WordAt(tf.editText, tf.cursorPos)
	tf.selectInit = tf.selectRange.Start
	if TheApp.SystemPlatform().IsMobile() {
		tf.Send(events.ContextMenu)
	}
	tf.NeedsRender()
}

// selectReset resets the selection
func (tf *TextField) selectReset() {
	tf.selectMode = false
	if tf.selectRange.Start == 0 && tf.selectRange.End == 0 {
		return
	}
	tf.selectRange.Start = 0
	tf.selectRange.End = 0
	tf.NeedsRender()
}

// selectUpdate updates the select region after any change to the text, to keep it in range
func (tf *TextField) selectUpdate() {
	if tf.selectRange.Start < tf.selectRange.End {
		ed := len(tf.editText)
		if tf.selectRange.Start < 0 {
			tf.selectRange.Start = 0
		}
		if tf.selectRange.End > ed {
			tf.selectRange.End = ed
		}
	} else {
		tf.selectReset()
	}
}

// cut cuts any selected text and adds it to the clipboard.
func (tf *TextField) cut() { //types:add
	if tf.NoEcho {
		return
	}
	cut := tf.deleteSelection()
	if cut != "" {
		em := tf.Events()
		if em != nil {
			em.Clipboard().Write(mimedata.NewText(cut))
		}
	}
}

// deleteSelection deletes any selected text, without adding to clipboard --
// returns text deleted
func (tf *TextField) deleteSelection() string {
	tf.selectUpdate()
	if !tf.hasSelection() {
		return ""
	}
	cut := tf.selection()
	tf.editText = append(tf.editText[:tf.selectRange.Start], tf.editText[tf.selectRange.End:]...)
	if tf.cursorPos > tf.selectRange.Start {
		if tf.cursorPos < tf.selectRange.End {
			tf.cursorPos = tf.selectRange.Start
		} else {
			tf.cursorPos -= tf.selectRange.End - tf.selectRange.Start
		}
	}
	tf.textEdited()
	tf.selectReset()
	return cut
}

// copy copies any selected text to the clipboard.
func (tf *TextField) copy() { //types:add
	if tf.NoEcho {
		return
	}
	tf.selectUpdate()
	if !tf.hasSelection() {
		return
	}

	md := mimedata.NewText(tf.selection())
	tf.Clipboard().Write(md)
}

// paste inserts text from the clipboard at current cursor position; if
// cursor is within a current selection, that selection is replaced.
func (tf *TextField) paste() { //types:add
	data := tf.Clipboard().Read([]string{mimedata.TextPlain})
	if data != nil {
		if tf.cursorPos >= tf.selectRange.Start && tf.cursorPos < tf.selectRange.End {
			tf.deleteSelection()
		}
		tf.insertAtCursor(data.Text(mimedata.TextPlain))
	}
}

// insertAtCursor inserts the given text at current cursor position.
func (tf *TextField) insertAtCursor(str string) {
	if tf.hasSelection() {
		tf.cut()
	}
	rs := []rune(str)
	rsl := len(rs)
	nt := append(tf.editText, rs...)               // first append to end
	copy(nt[tf.cursorPos+rsl:], nt[tf.cursorPos:]) // move stuff to end
	copy(nt[tf.cursorPos:], rs)                    // copy into position
	tf.editText = nt
	tf.dispRange.End += rsl
	tf.textEdited()
	tf.cursorForward(rsl)
}

func (tf *TextField) contextMenu(m *Scene) {
	NewFuncButton(m).SetFunc(tf.copy).SetIcon(icons.Copy).SetKey(keymap.Copy).SetState(tf.NoEcho || !tf.hasSelection(), states.Disabled)
	if !tf.IsReadOnly() {
		NewFuncButton(m).SetFunc(tf.cut).SetIcon(icons.Cut).SetKey(keymap.Cut).SetState(tf.NoEcho || !tf.hasSelection(), states.Disabled)
		paste := NewFuncButton(m).SetFunc(tf.paste).SetIcon(icons.Paste).SetKey(keymap.Paste)
		cb := tf.Scene.Events.Clipboard()
		if cb != nil {
			paste.SetState(cb.IsEmpty(), states.Disabled)
		}
	}
}

////////  Undo

// textFieldUndoRecord holds one undo record
type textFieldUndoRecord struct {
	text      []rune
	cursorPos int
}

func (ur *textFieldUndoRecord) set(txt []rune, curpos int) {
	ur.text = slices.Clone(txt)
	ur.cursorPos = curpos
}

// textFieldUndos manages everything about the undo process for a [TextField].
type textFieldUndos struct {

	// stack of undo records
	stack []textFieldUndoRecord

	// position within the undo stack
	pos int

	// last time undo was saved, for grouping
	lastSave time.Time
}

func (us *textFieldUndos) saveUndo(txt []rune, curpos int) {
	n := len(us.stack)
	now := time.Now()
	ts := now.Sub(us.lastSave)
	if n > 0 && ts < 250*time.Millisecond {
		r := us.stack[n-1]
		r.set(txt, curpos)
		us.stack[n-1] = r
		return
	}
	r := textFieldUndoRecord{}
	r.set(txt, curpos)
	us.stack = append(us.stack, r)
	us.pos = len(us.stack)
	us.lastSave = now
}

func (tf *TextField) saveUndo() {
	tf.undos.saveUndo(tf.editText, tf.cursorPos)
}

func (us *textFieldUndos) undo(txt []rune, curpos int) *textFieldUndoRecord {
	n := len(us.stack)
	if us.pos <= 0 || n == 0 {
		return &textFieldUndoRecord{}
	}
	if us.pos == n {
		us.lastSave = time.Time{}
		us.saveUndo(txt, curpos)
		us.pos--
	}
	us.pos--
	us.lastSave = time.Time{} // prevent any merging
	r := &us.stack[us.pos]
	return r
}

func (tf *TextField) undo() {
	r := tf.undos.undo(tf.editText, tf.cursorPos)
	if r != nil {
		tf.editText = r.text
		tf.cursorPos = r.cursorPos
		tf.renderVisible = nil
		tf.NeedsRender()
	}
}

func (us *textFieldUndos) redo() *textFieldUndoRecord {
	n := len(us.stack)
	if us.pos >= n-1 {
		return nil
	}
	us.lastSave = time.Time{} // prevent any merging
	us.pos++
	return &us.stack[us.pos]
}

func (tf *TextField) redo() {
	r := tf.undos.redo()
	if r != nil {
		tf.editText = r.text
		tf.cursorPos = r.cursorPos
		tf.renderVisible = nil
		tf.NeedsRender()
	}
}

////////  Complete

// SetCompleter sets completion functions so that completions will
// automatically be offered as the user types.
func (tf *TextField) SetCompleter(data any, matchFun complete.MatchFunc, editFun complete.EditFunc) {
	if matchFun == nil || editFun == nil {
		tf.complete = nil
		return
	}
	tf.complete = NewComplete().SetContext(data).SetMatchFunc(matchFun).SetEditFunc(editFun)
	tf.complete.OnSelect(func(e events.Event) {
		tf.completeText(tf.complete.Completion)
	})
}

// offerComplete pops up a menu of possible completions
func (tf *TextField) offerComplete() {
	if tf.complete == nil {
		return
	}
	s := string(tf.editText[0:tf.cursorPos])
	cpos := tf.charRenderPos(tf.cursorPos).ToPoint()
	cpos.X += 5
	cpos.Y = tf.Geom.TotalBBox.Max.Y
	tf.complete.SrcLn = 0
	tf.complete.SrcCh = tf.cursorPos
	tf.complete.Show(tf, cpos, s)
}

// cancelComplete cancels any pending completion -- call this when new events
// have moved beyond any prior completion scenario
func (tf *TextField) cancelComplete() {
	if tf.complete == nil {
		return
	}
	tf.complete.Cancel()
}

// completeText edits the text field using the string chosen from the completion menu
func (tf *TextField) completeText(s string) {
	txt := string(tf.editText) // Reminder: do NOT call tf.Text() in an active editing context!
	c := tf.complete.GetCompletion(s)
	ed := tf.complete.EditFunc(tf.complete.Context, txt, tf.cursorPos, c, tf.complete.Seed)
	st := tf.cursorPos - len(tf.complete.Seed)
	tf.cursorPos = st
	tf.cursorDelete(ed.ForwardDelete)
	tf.insertAtCursor(ed.NewText)
	tf.editDone()
}

////////  Rendering

// hasWordWrap returns true if the layout is multi-line word wrapping
func (tf *TextField) hasWordWrap() bool {
	return tf.Styles.Text.WhiteSpace.HasWordWrap()
}

// charPos returns the relative starting position of the given rune,
// in the overall RenderAll of all the text.
// These positions can be out of visible range: see CharRenderPos
func (tf *TextField) charPos(idx int) math32.Vector2 {
	if idx <= 0 || len(tf.renderAll.Lines) == 0 {
		return math32.Vector2{}
	}
	bb := tf.renderAll.RuneBounds(idx)
	if idx >= len(tf.editText) {
		if tf.numLines > 1 && tf.editText[len(tf.editText)-1] == ' ' {
			bb.Max.X += tf.lineHeight * 0.2
			return bb.Max
		}
		return bb.Max
	}
	return bb.Min
}

// relCharPos returns the text width in dots between the two text string
// positions (ed is exclusive -- +1 beyond actual char).
func (tf *TextField) relCharPos(st, ed int) math32.Vector2 {
	return tf.charPos(ed).Sub(tf.charPos(st))
}

// charRenderPos returns the starting render coords for the given character
// position in string -- makes no attempt to rationalize that pos (i.e., if
// not in visible range, position will be out of range too).
func (tf *TextField) charRenderPos(charidx int) math32.Vector2 {
	pos := tf.effPos
	sc := tf.Scene
	pos = pos.Add(math32.FromPoint(sc.SceneGeom.Pos))
	cpos := tf.relCharPos(tf.dispRange.Start, charidx)
	return pos.Add(cpos)
}

var (
	// textFieldSpriteName is the name of the window sprite used for the cursor.
	textFieldSpriteName = "TextField.Cursor"

	// textFieldCursor is the TextField that last created a new cursor sprite.
	textFieldCursor tree.Node
)

// startCursor starts the cursor blinking and renders it
func (tf *TextField) startCursor() {
	if tf == nil || tf.This == nil || !tf.IsVisible() {
		return
	}
	if tf.IsReadOnly() || !tf.AbilityIs(abilities.Focusable) {
		return
	}
	tf.toggleCursor(true)
}

// stopCursor stops the cursor from blinking
func (tf *TextField) stopCursor() {
	tf.toggleCursor(false)
}

// toggleSprite turns on or off the cursor sprite.
func (tf *TextField) toggleCursor(on bool) {
	TextCursor(on, tf.AsWidget(), &textFieldCursor, textFieldSpriteName, tf.CursorWidth.Dots, tf.lineHeight, tf.CursorColor, func() image.Point {
		return tf.charRenderPos(tf.cursorPos).ToPointFloor()
	})
}

// updateCursorPosition updates the position of the cursor.
func (tf *TextField) updateCursorPosition() {
	if tf.IsReadOnly() || !tf.StateIs(states.Focused) {
		return
	}
	sc := tf.Scene
	if sc == nil || sc.Stage == nil || sc.Stage.Main == nil {
		return
	}
	ms := sc.Stage.Main
	ms.Sprites.Lock()
	defer ms.Sprites.Unlock()
	if sp, ok := ms.Sprites.SpriteByNameNoLock(textFieldSpriteName); ok {
		sp.EventBBox.Min = tf.charRenderPos(tf.cursorPos).ToPointFloor()
	}
}

// renderSelect renders the selected region, if any, underneath the text
func (tf *TextField) renderSelect() {
	tf.renderVisible.SelectReset()
	if !tf.hasSelection() {
		return
	}
	dn := tf.dispRange.Len()
	effst := max(0, tf.selectRange.Start-tf.dispRange.Start)
	effed := min(dn, tf.selectRange.End-tf.dispRange.Start)
	if effst == effed {
		return
	}
	// fmt.Println("sel range:", effst, effed)
	tf.renderVisible.SelectRegion(textpos.Range{effst, effed})
}

// autoScroll scrolls the starting position to keep the cursor visible,
// and does various other state-updating steps to ensure everything is updated.
// This is called during Render().
func (tf *TextField) autoScroll() {
	sz := &tf.Geom.Size
	icsz := tf.iconsSize()
	availSz := sz.Actual.Content.Sub(icsz)
	if tf.renderAll != nil {
		availSz.Y += tf.renderAll.LineHeight * 2 // allow it to add a line
	}
	tf.configTextSize(availSz)
	n := len(tf.editText)
	tf.cursorPos = math32.Clamp(tf.cursorPos, 0, n)

	if tf.hasWordWrap() { // does not scroll
		tf.dispRange.Start = 0
		tf.dispRange.End = n
		if len(tf.renderAll.Lines) != tf.numLines {
			tf.renderVisible = nil
			tf.NeedsLayout()
		}
		return
	}
	st := &tf.Styles

	if n == 0 || tf.Geom.Size.Actual.Content.X <= 0 {
		tf.cursorPos = 0
		tf.dispRange.End = 0
		tf.dispRange.Start = 0
		return
	}
	maxw := tf.effSize.X
	if maxw < 0 {
		return
	}
	tf.charWidth = int(maxw / st.UnitContext.Dots(units.UnitCh)) // rough guess in chars
	if tf.charWidth < 1 {
		tf.charWidth = 1
	}

	// first rationalize all the values
	if tf.dispRange.End == 0 || tf.dispRange.End > n { // not init
		tf.dispRange.End = n
	}
	if tf.dispRange.Start >= tf.dispRange.End {
		tf.dispRange.Start = max(0, tf.dispRange.End-tf.charWidth)
	}

	inc := int(math32.Ceil(.1 * float32(tf.charWidth)))
	inc = max(4, inc)

	// keep cursor in view with buffer
	startIsAnchor := true
	if tf.cursorPos < (tf.dispRange.Start + inc) {
		tf.dispRange.Start -= inc
		tf.dispRange.Start = max(tf.dispRange.Start, 0)
		tf.dispRange.End = tf.dispRange.Start + tf.charWidth
		tf.dispRange.End = min(n, tf.dispRange.End)
	} else if tf.cursorPos > (tf.dispRange.End - inc) {
		tf.dispRange.End += inc
		tf.dispRange.End = min(tf.dispRange.End, n)
		tf.dispRange.Start = tf.dispRange.End - tf.charWidth
		tf.dispRange.Start = max(0, tf.dispRange.Start)
		startIsAnchor = false
	}
	if tf.dispRange.End < tf.dispRange.Start {
		return
	}

	if startIsAnchor {
		gotWidth := false
		spos := tf.charPos(tf.dispRange.Start).X
		for {
			w := tf.charPos(tf.dispRange.End).X - spos
			if w < maxw {
				if tf.dispRange.End == n {
					break
				}
				nw := tf.charPos(tf.dispRange.End+1).X - spos
				if nw >= maxw {
					gotWidth = true
					break
				}
				tf.dispRange.End++
			} else {
				tf.dispRange.End--
			}
		}
		if gotWidth || tf.dispRange.Start == 0 {
			return
		}
		// otherwise, try getting some more chars by moving up start..
	}

	// end is now anchor
	epos := tf.charPos(tf.dispRange.End).X
	for {
		w := epos - tf.charPos(tf.dispRange.Start).X
		if w < maxw {
			if tf.dispRange.Start == 0 {
				break
			}
			nw := epos - tf.charPos(tf.dispRange.Start-1).X
			if nw >= maxw {
				break
			}
			tf.dispRange.Start--
		} else {
			tf.dispRange.Start++
		}
	}
}

// pixelToCursor finds the cursor position that corresponds to the given pixel location
func (tf *TextField) pixelToCursor(pt image.Point) int {
	ptf := math32.FromPoint(pt)
	rpt := ptf.Sub(tf.effPos)
	if rpt.X <= 0 || rpt.Y < 0 {
		return tf.dispRange.Start
	}
	n := len(tf.editText)
	if tf.hasWordWrap() {
		ix := tf.renderAll.RuneAtPoint(ptf, tf.effPos)
		if ix >= 0 {
			return ix
		}
		return tf.dispRange.Start
	}
	pr := tf.PointToRelPos(pt)

	px := float32(pr.X)
	st := &tf.Styles
	c := tf.dispRange.Start + int(float64(px/st.UnitContext.Dots(units.UnitCh)))
	c = min(c, n)

	w := tf.relCharPos(tf.dispRange.Start, c).X
	if w > px {
		for w > px {
			c--
			if c <= tf.dispRange.Start {
				c = tf.dispRange.Start
				break
			}
			w = tf.relCharPos(tf.dispRange.Start, c).X
		}
	} else if w < px {
		for c < tf.dispRange.End {
			wn := tf.relCharPos(tf.dispRange.Start, c+1).X
			if wn > px {
				break
			} else if wn == px {
				c++
				break
			}
			c++
		}
	}
	return c
}

// setCursorFromPixel finds cursor location from given scene-relative
// pixel location, and sets current cursor to it, updating selection too.
func (tf *TextField) setCursorFromPixel(pt image.Point, selMode events.SelectModes) {
	oldPos := tf.cursorPos
	tf.cursorPos = tf.pixelToCursor(pt)
	if tf.selectMode || selMode != events.SelectOne {
		if !tf.selectMode && selMode != events.SelectOne {
			tf.selectRange.Start = oldPos
			tf.selectMode = true
		}
		if !tf.StateIs(states.Sliding) && selMode == events.SelectOne {
			tf.selectReset()
		} else {
			tf.selectRegionUpdate(tf.cursorPos)
		}
		tf.selectUpdate()
	} else if tf.hasSelection() {
		tf.selectReset()
	}
	tf.NeedsRender()
}

func (tf *TextField) handleKeyEvents() {
	tf.OnKeyChord(func(e events.Event) {
		kf := keymap.Of(e.KeyChord())
		if DebugSettings.KeyEventTrace {
			slog.Info("TextField KeyInput", "widget", tf, "keyFunction", kf)
		}
		if !tf.StateIs(states.Focused) && kf == keymap.Abort {
			return
		}
		tf.startCursor()

		// first all the keys that work for both inactive and active
		switch kf {
		case keymap.MoveRight:
			e.SetHandled()
			tf.shiftSelect(e)
			tf.cursorForward(1)
			tf.offerComplete()
		case keymap.WordRight:
			e.SetHandled()
			tf.shiftSelect(e)
			tf.cursorForwardWord(1)
			tf.offerComplete()
		case keymap.MoveLeft:
			e.SetHandled()
			tf.shiftSelect(e)
			tf.cursorBackward(1)
			tf.offerComplete()
		case keymap.WordLeft:
			e.SetHandled()
			tf.shiftSelect(e)
			tf.cursorBackwardWord(1)
			tf.offerComplete()
		case keymap.MoveDown:
			if tf.numLines > 1 {
				e.SetHandled()
				tf.shiftSelect(e)
				tf.cursorDown(1)
			}
		case keymap.MoveUp:
			if tf.numLines > 1 {
				e.SetHandled()
				tf.shiftSelect(e)
				tf.cursorUp(1)
			}
		case keymap.Home:
			e.SetHandled()
			tf.shiftSelect(e)
			tf.cancelComplete()
			tf.cursorStart()
		case keymap.End:
			e.SetHandled()
			tf.shiftSelect(e)
			tf.cancelComplete()
			tf.cursorEnd()
		case keymap.SelectMode:
			e.SetHandled()
			tf.cancelComplete()
			tf.selectModeToggle()
		case keymap.CancelSelect:
			e.SetHandled()
			tf.cancelComplete()
			tf.selectReset()
		case keymap.SelectAll:
			e.SetHandled()
			tf.cancelComplete()
			tf.selectAll()
		case keymap.Copy:
			e.SetHandled()
			tf.cancelComplete()
			tf.copy()
		}
		if tf.IsReadOnly() || e.IsHandled() {
			return
		}
		switch kf {
		case keymap.Enter:
			fallthrough
		case keymap.FocusNext: // we process tab to make it EditDone as opposed to other ways of losing focus
			e.SetHandled()
			tf.cancelComplete()
			tf.editDone()
			tf.focusNext()
		case keymap.Accept: // ctrl+enter
			e.SetHandled()
			tf.cancelComplete()
			tf.editDone()
		case keymap.FocusPrev:
			e.SetHandled()
			tf.cancelComplete()
			tf.editDone()
			tf.focusPrev()
		case keymap.Abort: // esc
			e.SetHandled()
			tf.cancelComplete()
			tf.revert()
			// tf.FocusChanged(FocusInactive)
		case keymap.Backspace:
			e.SetHandled()
			tf.saveUndo()
			tf.cursorBackspace(1)
			tf.offerComplete()
			tf.Send(events.Input, e)
		case keymap.Kill:
			e.SetHandled()
			tf.cancelComplete()
			tf.cursorKill()
			tf.Send(events.Input, e)
		case keymap.Delete:
			e.SetHandled()
			tf.saveUndo()
			tf.cursorDelete(1)
			tf.offerComplete()
			tf.Send(events.Input, e)
		case keymap.BackspaceWord:
			e.SetHandled()
			tf.saveUndo()
			tf.cursorBackspaceWord(1)
			tf.offerComplete()
			tf.Send(events.Input, e)
		case keymap.DeleteWord:
			e.SetHandled()
			tf.saveUndo()
			tf.cursorDeleteWord(1)
			tf.offerComplete()
			tf.Send(events.Input, e)
		case keymap.Cut:
			e.SetHandled()
			tf.saveUndo()
			tf.cancelComplete()
			tf.cut()
			tf.Send(events.Input, e)
		case keymap.Paste:
			e.SetHandled()
			tf.saveUndo()
			tf.cancelComplete()
			tf.paste()
			tf.Send(events.Input, e)
		case keymap.Undo:
			e.SetHandled()
			tf.undo()
		case keymap.Redo:
			e.SetHandled()
			tf.redo()
		case keymap.Complete:
			e.SetHandled()
			tf.offerComplete()
		case keymap.None:
			if unicode.IsPrint(e.KeyRune()) {
				if !e.HasAnyModifier(key.Control, key.Meta) {
					e.SetHandled()
					tf.saveUndo()
					tf.insertAtCursor(string(e.KeyRune()))
					if e.KeyRune() == ' ' {
						tf.cancelComplete()
					} else {
						tf.offerComplete()
					}
					tf.Send(events.Input, e)
				}
			}
		}
	})
	tf.OnFocus(func(e events.Event) {
		if tf.IsReadOnly() {
			e.SetHandled()
		} else {
			tf.startCursor()
		}
	})
	tf.OnFocusLost(func(e events.Event) {
		if tf.IsReadOnly() {
			e.SetHandled()
			return
		}
		tf.editDone()
	})
}

func (tf *TextField) Style() {
	tf.WidgetBase.Style()
	tf.CursorWidth.ToDots(&tf.Styles.UnitContext)
}

func (tf *TextField) configTextSize(sz math32.Vector2) math32.Vector2 {
	txt := tf.editText
	if len(txt) == 0 && len(tf.Placeholder) > 0 {
		txt = []rune(tf.Placeholder)
	}
	if tf.NoEcho {
		txt = concealDots(len(tf.editText))
	}
	sty, tsty := tf.Styles.NewRichText()
	etxs := *tsty
	etxs.Align, etxs.AlignV = text.Start, text.Start // only works with this
	tx := rich.NewText(sty, txt)
	tf.renderAll = tf.Scene.TextShaper().WrapLines(tx, sty, &etxs, &AppearanceSettings.Text, sz)
	rsz := tf.renderAll.Bounds.Size().Ceil()
	return rsz
}

func (tf *TextField) iconsSize() math32.Vector2 {
	var sz math32.Vector2
	if lead := tf.leadingIconButton; lead != nil {
		sz.X += lead.Geom.Size.Actual.Total.X
	}
	if trail := tf.trailingIconButton; trail != nil {
		sz.X += trail.Geom.Size.Actual.Total.X
	}
	return sz
}

func (tf *TextField) SizeUp() {
	tf.renderVisible = nil
	tf.Frame.SizeUp()
	txt := tf.editText
	if len(txt) == 0 && len(tf.Placeholder) > 0 {
		txt = []rune(tf.Placeholder)
	}
	tf.dispRange.Start = 0
	tf.dispRange.End = len(txt)

	sz := &tf.Geom.Size
	icsz := tf.iconsSize()
	availSz := sz.Actual.Content.Sub(icsz)

	rsz := tf.configTextSize(availSz)
	rsz.SetAdd(icsz)
	sz.FitSizeMax(&sz.Actual.Content, rsz)
	sz.setTotalFromContent(&sz.Actual)
	tf.lineHeight = tf.Styles.LineHeightDots()
	if DebugSettings.LayoutTrace {
		fmt.Println(tf, "TextField SizeUp:", rsz, "Actual:", sz.Actual.Content)
	}
}

func (tf *TextField) SizeDown(iter int) bool {
	sz := &tf.Geom.Size
	prevContent := sz.Actual.Content
	sz.setInitContentMin(tf.Styles.Min.Dots().Ceil())
	pgrow, _ := tf.growToAllocSize(sz.Actual.Content, sz.Alloc.Content) // get before update
	icsz := tf.iconsSize()
	availSz := pgrow.Sub(icsz)
	rsz := tf.configTextSize(availSz)
	rsz.SetAdd(icsz)
	// start over so we don't reflect hysteresis of prior guess
	chg := prevContent != sz.Actual.Content
	if chg {
		if DebugSettings.LayoutTrace {
			fmt.Println(tf, "TextField Size Changed:", sz.Actual.Content, "was:", prevContent)
		}
	}
	if tf.Styles.Grow.X > 0 {
		rsz.X = max(pgrow.X, rsz.X)
	}
	if tf.Styles.Grow.Y > 0 {
		rsz.Y = max(pgrow.Y, rsz.Y)
	}
	sz.FitSizeMax(&sz.Actual.Content, rsz)
	sz.setTotalFromContent(&sz.Actual)
	sz.Alloc = sz.Actual // this is important for constraining our children layout:
	redo := tf.Frame.SizeDown(iter)
	return chg || redo
}

func (tf *TextField) SizeFinal() {
	tf.Geom.RelPos.SetZero()
	// tf.sizeFromChildrenFit(0, SizeFinalPass) // key to omit
	tf.growToAlloc()
	tf.sizeFinalChildren()
	tf.styleSizeUpdate() // now that sizes are stable, ensure styling based on size is updated
	tf.sizeFinalParts()
}

func (tf *TextField) ApplyScenePos() {
	tf.Frame.ApplyScenePos()
	tf.setEffPosAndSize()
}

// setEffPosAndSize sets the effective position and size of
// the textfield based on its base position and size
// and its icons or lack thereof
func (tf *TextField) setEffPosAndSize() {
	sz := tf.Geom.Size.Actual.Content
	pos := tf.Geom.Pos.Content
	if lead := tf.leadingIconButton; lead != nil {
		pos.X += lead.Geom.Size.Actual.Total.X
		sz.X -= lead.Geom.Size.Actual.Total.X
	}
	if trail := tf.trailingIconButton; trail != nil {
		sz.X -= trail.Geom.Size.Actual.Total.X
	}
	if tf.renderAll == nil {
		tf.numLines = 0
	} else {
		tf.numLines = len(tf.renderAll.Lines)
	}
	if tf.numLines <= 1 {
		pos.Y += 0.5 * (sz.Y - tf.lineHeight) // center
	}
	tf.effSize = sz.Ceil()
	tf.effPos = pos.Ceil()
}

func (tf *TextField) layoutCurrent() {
	cur := tf.editText[tf.dispRange.Start:tf.dispRange.End]
	clr := tf.Styles.Color
	if len(tf.editText) == 0 && len(tf.Placeholder) > 0 {
		clr = tf.PlaceholderColor
		cur = []rune(tf.Placeholder)
	} else if tf.NoEcho {
		cur = concealDots(len(cur))
	}
	sz := &tf.Geom.Size
	icsz := tf.iconsSize()
	availSz := sz.Actual.Content.Sub(icsz)
	sty, tsty := tf.Styles.NewRichText()
	tsty.Color = colors.ToUniform(clr)
	tx := rich.NewText(sty, cur)
	tf.renderVisible = tf.Scene.TextShaper().WrapLines(tx, sty, tsty, &AppearanceSettings.Text, availSz)
	tf.renderedRange = tf.dispRange
}

func (tf *TextField) Render() {
	tf.autoScroll() // does all update checking, inits paint with our style
	tf.RenderAllocBox()
	if tf.dispRange.Start < 0 || tf.dispRange.End > len(tf.editText) {
		return
	}
	if tf.renderVisible == nil || tf.dispRange != tf.renderedRange {
		tf.layoutCurrent()
	}
	tf.updateCursorPosition()
	tf.renderSelect()
	tf.Scene.Painter.DrawText(tf.renderVisible, tf.effPos)
}

// concealDots creates an n-length []rune of bullet characters.
func concealDots(n int) []rune {
	dots := make([]rune, n)
	for i := range dots {
		dots[i] = '•'
	}
	return dots
}
