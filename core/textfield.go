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
	"cogentcore.org/core/paint"
	"cogentcore.org/core/parse/complete"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/abilities"
	"cogentcore.org/core/styles/states"
	"cogentcore.org/core/styles/units"
	"cogentcore.org/core/tree"
	"golang.org/x/image/draw"
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

	// SelectColor is the color used for the text selection background color.
	// It should be set in a Styler like all other style properties.
	// By default, it is [colors.Scheme.Select.Container].
	SelectColor image.Image

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

	// startPos is the starting display position in the string.
	startPos int

	// endPos is the ending display position in the string.
	endPos int

	// cursorPos is the current cursor position.
	cursorPos int

	// cursorLine is the current cursor line position.
	cursorLine int

	// charWidth is the approximate number of chars that can be
	// displayed at any time, which is computed from the font size.
	charWidth int

	// selectStart is the starting position of selection in the string.
	selectStart int

	// selectEnd is the ending position of selection in the string.
	selectEnd int

	// selectInit is the initial selection position (where it started).
	selectInit int

	// selectMode is whether to select text as the cursor moves.
	selectMode bool

	// renderAll is the render version of entire text, for sizing.
	renderAll paint.Text

	// renderVisible is the render version of just the visible text.
	renderVisible paint.Text

	// number of lines from last render update, for word-wrap version
	numLines int

	// fontHeight is the font height cached during styling.
	fontHeight float32

	// blinkOn oscillates between on and off for blinking.
	blinkOn bool

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
		tf.CursorWidth.Dp(1)
		tf.SelectColor = colors.Scheme.Select.Container
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
		s.Text.Align = styles.Start
		s.Align.Items = styles.Center
		s.Color = colors.Scheme.OnSurface
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
	tf.On(events.MouseDown, func(e events.Event) {
		if !tf.StateIs(states.Focused) {
			tf.SetFocusEvent() // always grab, even if read only..
		}
		if tf.IsReadOnly() {
			return
		}
		e.SetHandled()
		switch e.MouseButton() {
		case events.Left:
			tf.setCursorFromPixel(e.Pos(), e.SelectMode())
		case events.Middle:
			e.SetHandled()
			tf.setCursorFromPixel(e.Pos(), e.SelectMode())
			tf.paste()
		}
	})
	tf.OnClick(func(e events.Event) {
		if tf.IsReadOnly() {
			return
		}
		tf.SetFocusEvent()
	})
	tf.On(events.DoubleClick, func(e events.Event) {
		if tf.IsReadOnly() {
			return
		}
		if !tf.IsReadOnly() && !tf.StateIs(states.Focused) {
			tf.SetFocusEvent()
		}
		e.SetHandled()
		tf.selectWord()
	})
	tf.On(events.TripleClick, func(e events.Event) {
		if tf.IsReadOnly() {
			return
		}
		if !tf.IsReadOnly() && !tf.StateIs(states.Focused) {
			tf.SetFocusEvent()
		}
		e.SetHandled()
		tf.selectAll()
	})
	tf.On(events.SlideMove, func(e events.Event) {
		if tf.IsReadOnly() {
			return
		}
		e.SetHandled()
		if !tf.selectMode {
			tf.selectModeToggle()
		}
		tf.setCursorFromPixel(e.Pos(), events.SelectOne)
	})
	tf.OnClose(func(e events.Event) {
		tf.editDone() // todo: this must be protected against something else, for race detector
	})

	tf.Maker(func(p *tree.Plan) {
		tf.editText = []rune(tf.text)
		tf.edited = false

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
	tf.clearCursor()
}

// revert aborts editing and reverts to the last saved text.
func (tf *TextField) revert() {
	tf.editText = []rune(tf.text)
	tf.edited = false
	tf.startPos = 0
	tf.endPos = tf.charWidth
	tf.selectReset()
	tf.NeedsRender()
}

// clear clears any existing text.
func (tf *TextField) clear() {
	tf.edited = true
	tf.editText = tf.editText[:0]
	tf.startPos = 0
	tf.endPos = 0
	tf.selectReset()
	tf.SetFocusEvent() // this is essential for ensuring that the clear applies after focus is lost..
	tf.NeedsRender()
}

// clearError clears any existing validation error.
func (tf *TextField) clearError() {
	if tf.error == nil {
		return
	}
	tf.error = nil
	tf.Update()
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

//////////////////////////////////////////////////////////////////////////////////////////
//  Cursor Navigation

// cursorForward moves the cursor forward
func (tf *TextField) cursorForward(steps int) {
	tf.cursorPos += steps
	if tf.cursorPos > len(tf.editText) {
		tf.cursorPos = len(tf.editText)
	}
	if tf.cursorPos > tf.endPos {
		inc := tf.cursorPos - tf.endPos
		tf.endPos += inc
	}
	tf.cursorLine, _, _ = tf.renderAll.RuneSpanPos(tf.cursorPos)
	if tf.selectMode {
		tf.selectRegionUpdate(tf.cursorPos)
	}
	tf.NeedsRender()
}

// cursorForwardWord moves the cursor forward by words
func (tf *TextField) cursorForwardWord(steps int) {
	for i := 0; i < steps; i++ {
		sz := len(tf.editText)
		if sz > 0 && tf.cursorPos < sz {
			ch := tf.cursorPos
			var done = false
			for ch < sz && !done { // if on a wb, go past
				r1 := tf.editText[ch]
				r2 := rune(-1)
				if ch < sz-1 {
					r2 = tf.editText[ch+1]
				}
				if IsWordBreak(r1, r2) {
					ch++
				} else {
					done = true
				}
			}
			done = false
			for ch < sz && !done {
				r1 := tf.editText[ch]
				r2 := rune(-1)
				if ch < sz-1 {
					r2 = tf.editText[ch+1]
				}
				if !IsWordBreak(r1, r2) {
					ch++
				} else {
					done = true
				}
			}
			tf.cursorPos = ch
		} else {
			tf.cursorPos = sz
		}
	}
	if tf.cursorPos > len(tf.editText) {
		tf.cursorPos = len(tf.editText)
	}
	if tf.cursorPos > tf.endPos {
		inc := tf.cursorPos - tf.endPos
		tf.endPos += inc
	}
	tf.cursorLine, _, _ = tf.renderAll.RuneSpanPos(tf.cursorPos)
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
	if tf.cursorPos <= tf.startPos {
		dec := min(tf.startPos, 8)
		tf.startPos -= dec
	}
	tf.cursorLine, _, _ = tf.renderAll.RuneSpanPos(tf.cursorPos)
	if tf.selectMode {
		tf.selectRegionUpdate(tf.cursorPos)
	}
	tf.NeedsRender()
}

// cursorBackwardWord moves the cursor backward by words
func (tf *TextField) cursorBackwardWord(steps int) {
	for i := 0; i < steps; i++ {
		sz := len(tf.editText)
		if sz > 0 && tf.cursorPos > 0 {
			ch := min(tf.cursorPos, sz-1)
			var done = false
			for ch < sz && !done { // if on a wb, go past
				r1 := tf.editText[ch]
				r2 := rune(-1)
				if ch > 0 {
					r2 = tf.editText[ch-1]
				}
				if IsWordBreak(r1, r2) {
					ch--
					if ch == -1 {
						done = true
					}
				} else {
					done = true
				}
			}
			done = false
			for ch < sz && ch >= 0 && !done {
				r1 := tf.editText[ch]
				r2 := rune(-1)
				if ch > 0 {
					r2 = tf.editText[ch-1]
				}
				if !IsWordBreak(r1, r2) {
					ch--
				} else {
					done = true
				}
			}
			tf.cursorPos = ch
		} else {
			tf.cursorPos = 0
		}
	}
	if tf.cursorPos < 0 {
		tf.cursorPos = 0
	}
	if tf.cursorPos <= tf.startPos {
		dec := min(tf.startPos, 8)
		tf.startPos -= dec
	}
	tf.cursorLine, _, _ = tf.renderAll.RuneSpanPos(tf.cursorPos)
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

	_, ri, _ := tf.renderAll.RuneSpanPos(tf.cursorPos)
	tf.cursorLine = min(tf.cursorLine+steps, tf.numLines-1)
	tf.cursorPos, _ = tf.renderAll.SpanPosToRuneIndex(tf.cursorLine, ri)
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

	_, ri, _ := tf.renderAll.RuneSpanPos(tf.cursorPos)
	tf.cursorLine = max(tf.cursorLine-steps, 0)
	tf.cursorPos, _ = tf.renderAll.SpanPosToRuneIndex(tf.cursorLine, ri)
	if tf.selectMode {
		tf.selectRegionUpdate(tf.cursorPos)
	}
	tf.NeedsRender()
}

// cursorStart moves the cursor to the start of the text, updating selection
// if select mode is active.
func (tf *TextField) cursorStart() {
	tf.cursorPos = 0
	tf.startPos = 0
	tf.endPos = min(len(tf.editText), tf.startPos+tf.charWidth)
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
	tf.endPos = len(tf.editText) // try -- display will adjust
	tf.startPos = max(0, tf.endPos-tf.charWidth)
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
	tf.edited = true
	tf.editText = append(tf.editText[:tf.cursorPos-steps], tf.editText[tf.cursorPos:]...)
	tf.cursorBackward(steps)
	tf.NeedsRender()
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
	tf.edited = true
	tf.editText = append(tf.editText[:tf.cursorPos], tf.editText[tf.cursorPos+steps:]...)
	tf.NeedsRender()
}

// cursorBackspaceWord deletes words(s) immediately before cursor
func (tf *TextField) cursorBackspaceWord(steps int) {
	if tf.hasSelection() {
		tf.deleteSelection()
		return
	}
	org := tf.cursorPos
	tf.cursorBackwardWord(steps)
	tf.edited = true
	tf.editText = append(tf.editText[:tf.cursorPos], tf.editText[org:]...)
	tf.NeedsRender()
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
	tf.edited = true
	tf.editText = append(tf.editText[:tf.cursorPos], tf.editText[org:]...)
	tf.NeedsRender()
}

// cursorKill deletes text from cursor to end of text
func (tf *TextField) cursorKill() {
	steps := len(tf.editText) - tf.cursorPos
	tf.cursorDelete(steps)
}

///////////////////////////////////////////////////////////////////////////////
//    Selection

// clearSelected resets both the global selected flag and any current selection
func (tf *TextField) clearSelected() {
	tf.SetState(false, states.Selected)
	tf.selectReset()
}

// hasSelection returns whether there is a selected region of text
func (tf *TextField) hasSelection() bool {
	tf.selectUpdate()
	return tf.selectStart < tf.selectEnd
}

// selection returns the currently selected text
func (tf *TextField) selection() string {
	if tf.hasSelection() {
		return string(tf.editText[tf.selectStart:tf.selectEnd])
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
		tf.selectStart = tf.cursorPos
		tf.selectEnd = tf.selectStart
	}
}

// shiftSelect sets the selection start if the shift key is down but wasn't previously.
// If the shift key has been released, the selection info is cleared.
func (tf *TextField) shiftSelect(e events.Event) {
	hasShift := e.HasAnyModifier(key.Shift)
	if hasShift && !tf.selectMode {
		tf.selectModeToggle()
	}
	if !hasShift && tf.selectMode {
		tf.selectReset()
	}
}

// selectRegionUpdate updates current select region based on given cursor position
// relative to SelectStart position
func (tf *TextField) selectRegionUpdate(pos int) {
	if pos < tf.selectInit {
		tf.selectStart = pos
		tf.selectEnd = tf.selectInit
	} else {
		tf.selectStart = tf.selectInit
		tf.selectEnd = pos
	}
	tf.selectUpdate()
}

// selectAll selects all the text
func (tf *TextField) selectAll() {
	tf.selectStart = 0
	tf.selectInit = 0
	tf.selectEnd = len(tf.editText)
	if TheApp.SystemPlatform().IsMobile() {
		tf.Send(events.ContextMenu)
	}
	tf.NeedsRender()
}

// isWordBreak defines what counts as a word break for the purposes of selecting words
func (tf *TextField) isWordBreak(r rune) bool {
	return unicode.IsSpace(r) || unicode.IsSymbol(r) || unicode.IsPunct(r)
}

// selectWord selects the word (whitespace delimited) that the cursor is on
func (tf *TextField) selectWord() {
	sz := len(tf.editText)
	if sz <= 3 {
		tf.selectAll()
		return
	}
	tf.selectStart = tf.cursorPos
	if tf.selectStart >= sz {
		tf.selectStart = sz - 2
	}
	if !tf.isWordBreak(tf.editText[tf.selectStart]) {
		for tf.selectStart > 0 {
			if tf.isWordBreak(tf.editText[tf.selectStart-1]) {
				break
			}
			tf.selectStart--
		}
		tf.selectEnd = tf.cursorPos + 1
		for tf.selectEnd < sz {
			if tf.isWordBreak(tf.editText[tf.selectEnd]) {
				break
			}
			tf.selectEnd++
		}
	} else { // keep the space start -- go to next space..
		tf.selectEnd = tf.cursorPos + 1
		for tf.selectEnd < sz {
			if !tf.isWordBreak(tf.editText[tf.selectEnd]) {
				break
			}
			tf.selectEnd++
		}
		for tf.selectEnd < sz { // include all trailing spaces
			if tf.isWordBreak(tf.editText[tf.selectEnd]) {
				break
			}
			tf.selectEnd++
		}
	}
	tf.selectInit = tf.selectStart
	if TheApp.SystemPlatform().IsMobile() {
		tf.Send(events.ContextMenu)
	}
	tf.NeedsRender()
}

// selectReset resets the selection
func (tf *TextField) selectReset() {
	tf.selectMode = false
	if tf.selectStart == 0 && tf.selectEnd == 0 {
		return
	}
	tf.selectStart = 0
	tf.selectEnd = 0
	tf.NeedsRender()
}

// selectUpdate updates the select region after any change to the text, to keep it in range
func (tf *TextField) selectUpdate() {
	if tf.selectStart < tf.selectEnd {
		ed := len(tf.editText)
		if tf.selectStart < 0 {
			tf.selectStart = 0
		}
		if tf.selectEnd > ed {
			tf.selectEnd = ed
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
	tf.edited = true
	tf.editText = append(tf.editText[:tf.selectStart], tf.editText[tf.selectEnd:]...)
	if tf.cursorPos > tf.selectStart {
		if tf.cursorPos < tf.selectEnd {
			tf.cursorPos = tf.selectStart
		} else {
			tf.cursorPos -= tf.selectEnd - tf.selectStart
		}
	}
	tf.selectReset()
	tf.NeedsRender()
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

	md := mimedata.NewText(tf.Text())
	tf.Clipboard().Write(md)
}

// paste inserts text from the clipboard at current cursor position; if
// cursor is within a current selection, that selection is replaced.
func (tf *TextField) paste() { //types:add
	data := tf.Clipboard().Read([]string{mimedata.TextPlain})
	if data != nil {
		if tf.cursorPos >= tf.selectStart && tf.cursorPos < tf.selectEnd {
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
	tf.edited = true
	rs := []rune(str)
	rsl := len(rs)
	nt := append(tf.editText, rs...)               // first append to end
	copy(nt[tf.cursorPos+rsl:], nt[tf.cursorPos:]) // move stuff to end
	copy(nt[tf.cursorPos:], rs)                    // copy into position
	tf.editText = nt
	tf.endPos += rsl
	tf.cursorForward(rsl)
	tf.NeedsRender()
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

///////////////////////////////////////////////////////////////////////////////
//    Undo

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
		tf.NeedsRender()
	}
}

///////////////////////////////////////////////////////////////////////////////
//    Complete

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
	cpos := tf.charRenderPos(tf.cursorPos, true).ToPoint()
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

///////////////////////////////////////////////////////////////////////////////
//    Rendering

// hasWordWrap returns true if the layout is multi-line word wrapping
func (tf *TextField) hasWordWrap() bool {
	return tf.Styles.Text.HasWordWrap()
}

// charPos returns the relative starting position of the given rune,
// in the overall RenderAll of all the text.
// These positions can be out of visible range: see CharRenderPos
func (tf *TextField) charPos(idx int) math32.Vector2 {
	if idx <= 0 || len(tf.renderAll.Spans) == 0 {
		return math32.Vector2{}
	}
	pos, _, _, _ := tf.renderAll.RuneRelPos(idx)
	pos.Y -= tf.renderAll.Spans[0].RelPos.Y
	return pos
}

// relCharPos returns the text width in dots between the two text string
// positions (ed is exclusive -- +1 beyond actual char).
func (tf *TextField) relCharPos(st, ed int) math32.Vector2 {
	return tf.charPos(ed).Sub(tf.charPos(st))
}

// charRenderPos returns the starting render coords for the given character
// position in string -- makes no attempt to rationalize that pos (i.e., if
// not in visible range, position will be out of range too).
// if wincoords is true, then adds window box offset -- for cursor, popups
func (tf *TextField) charRenderPos(charidx int, wincoords bool) math32.Vector2 {
	pos := tf.effPos
	if wincoords {
		sc := tf.Scene
		pos = pos.Add(math32.Vector2FromPoint(sc.sceneGeom.Pos))
	}
	cpos := tf.relCharPos(tf.startPos, charidx)
	return pos.Add(cpos)
}

var (
	// textFieldBlinker manages cursor blinking
	textFieldBlinker = Blinker{}

	// textFieldSpriteName is the name of the window sprite used for the cursor
	textFieldSpriteName = "TextField.Cursor"
)

func init() {
	TheApp.AddQuitCleanFunc(textFieldBlinker.QuitClean)
	textFieldBlinker.Func = func() {
		w := textFieldBlinker.Widget
		textFieldBlinker.Unlock() // comes in locked
		if w == nil {
			return
		}
		tf := AsTextField(w)
		tf.AsyncLock()
		if !w.AsWidget().StateIs(states.Focused) || !w.AsWidget().IsVisible() {
			tf.blinkOn = false
			tf.renderCursor(false)
		} else {
			tf.blinkOn = !tf.blinkOn
			tf.renderCursor(tf.blinkOn)
		}
		tf.AsyncUnlock()
	}
}

// startCursor starts the cursor blinking and renders it
func (tf *TextField) startCursor() {
	if tf == nil || tf.This == nil {
		return
	}
	if !tf.IsVisible() {
		return
	}
	tf.blinkOn = true
	tf.renderCursor(true)
	if SystemSettings.CursorBlinkTime == 0 {
		return
	}
	textFieldBlinker.SetWidget(tf.This.(Widget))
	textFieldBlinker.Blink(SystemSettings.CursorBlinkTime)
}

// clearCursor turns off cursor and stops it from blinking
func (tf *TextField) clearCursor() {
	if tf.IsReadOnly() {
		return
	}
	tf.stopCursor()
	tf.renderCursor(false)
}

// stopCursor stops the cursor from blinking
func (tf *TextField) stopCursor() {
	if tf == nil || tf.This == nil {
		return
	}
	textFieldBlinker.ResetWidget(tf.This.(Widget))
}

// renderCursor renders the cursor on or off, as a sprite that is either on or off
func (tf *TextField) renderCursor(on bool) {
	if tf == nil || tf.This == nil {
		return
	}
	if !on {
		if tf.Scene == nil {
			return
		}
		ms := tf.Scene.Stage.Main
		if ms == nil {
			return
		}
		spnm := fmt.Sprintf("%v-%v", textFieldSpriteName, tf.fontHeight)
		ms.Sprites.InactivateSprite(spnm)
		return
	}
	if !tf.IsVisible() {
		return
	}

	tf.cursorMu.Lock()
	defer tf.cursorMu.Unlock()

	sp := tf.cursorSprite(on)
	if sp == nil {
		return
	}
	sp.Geom.Pos = tf.charRenderPos(tf.cursorPos, true).ToPointFloor()
}

// cursorSprite returns the Sprite for the cursor (which is
// only rendered once with a vertical bar, and just activated and inactivated
// depending on render status).  On sets the On status of the cursor.
func (tf *TextField) cursorSprite(on bool) *Sprite {
	sc := tf.Scene
	if sc == nil {
		return nil
	}
	ms := sc.Stage.Main
	if ms == nil {
		return nil // only MainStage has sprites
	}
	spnm := fmt.Sprintf("%v-%v", textFieldSpriteName, tf.fontHeight)
	sp, ok := ms.Sprites.SpriteByName(spnm)
	// TODO: figure out how to update caret color on color scheme change
	if !ok {
		bbsz := image.Point{int(math32.Ceil(tf.CursorWidth.Dots)), int(math32.Ceil(tf.fontHeight))}
		if bbsz.X < 2 { // at least 2
			bbsz.X = 2
		}
		sp = NewSprite(spnm, bbsz, image.Point{})
		sp.Active = on
		ibox := sp.Pixels.Bounds()
		draw.Draw(sp.Pixels, ibox, tf.CursorColor, image.Point{}, draw.Src)
		ms.Sprites.Add(sp)
	}
	if on {
		ms.Sprites.ActivateSprite(sp.Name)
	} else {
		ms.Sprites.InactivateSprite(sp.Name)
	}
	return sp
}

// renderSelect renders the selected region, if any, underneath the text
func (tf *TextField) renderSelect() {
	if !tf.hasSelection() {
		return
	}
	effst := max(tf.startPos, tf.selectStart)
	if effst >= tf.endPos {
		return
	}
	effed := min(tf.endPos, tf.selectEnd)
	if effed < tf.startPos {
		return
	}
	if effed <= effst {
		return
	}

	spos := tf.charRenderPos(effst, false)

	pc := &tf.Scene.PaintContext
	tsz := tf.relCharPos(effst, effed)
	if !tf.hasWordWrap() || tsz.Y == 0 {
		pc.FillBox(spos, math32.Vec2(tsz.X, tf.fontHeight), tf.SelectColor)
		return
	}
	ex := float32(tf.Geom.ContentBBox.Max.X)
	sx := float32(tf.Geom.ContentBBox.Min.X)
	ssi, _, _ := tf.renderAll.RuneSpanPos(effst)
	esi, _, _ := tf.renderAll.RuneSpanPos(effed)
	ep := tf.charRenderPos(effed, false)

	pc.FillBox(spos, math32.Vec2(ex-spos.X, tf.fontHeight), tf.SelectColor)

	spos.X = sx
	spos.Y += tf.renderAll.Spans[ssi+1].RelPos.Y - tf.renderAll.Spans[ssi].RelPos.Y
	for si := ssi + 1; si <= esi; si++ {
		if si < esi {
			pc.FillBox(spos, math32.Vec2(ex-spos.X, tf.fontHeight), tf.SelectColor)
		} else {
			pc.FillBox(spos, math32.Vec2(ep.X-spos.X, tf.fontHeight), tf.SelectColor)
		}
		spos.Y += tf.renderAll.Spans[si].RelPos.Y - tf.renderAll.Spans[si-1].RelPos.Y
	}
}

// autoScroll scrolls the starting position to keep the cursor visible
func (tf *TextField) autoScroll() {
	sz := &tf.Geom.Size
	icsz := tf.iconsSize()
	availSz := sz.Actual.Content.Sub(icsz)
	tf.configTextSize(availSz)
	n := len(tf.editText)
	tf.cursorPos = math32.ClampInt(tf.cursorPos, 0, n)

	if tf.hasWordWrap() { // does not scroll
		tf.startPos = 0
		tf.endPos = n
		if len(tf.renderAll.Spans) != tf.numLines {
			tf.NeedsLayout()
		}
		return
	}
	st := &tf.Styles

	if n == 0 || tf.Geom.Size.Actual.Content.X <= 0 {
		tf.cursorPos = 0
		tf.endPos = 0
		tf.startPos = 0
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
	if tf.endPos == 0 || tf.endPos > n { // not init
		tf.endPos = n
	}
	if tf.startPos >= tf.endPos {
		tf.startPos = max(0, tf.endPos-tf.charWidth)
	}

	inc := int(math32.Ceil(.1 * float32(tf.charWidth)))
	inc = max(4, inc)

	// keep cursor in view with buffer
	startIsAnchor := true
	if tf.cursorPos < (tf.startPos + inc) {
		tf.startPos -= inc
		tf.startPos = max(tf.startPos, 0)
		tf.endPos = tf.startPos + tf.charWidth
		tf.endPos = min(n, tf.endPos)
	} else if tf.cursorPos > (tf.endPos - inc) {
		tf.endPos += inc
		tf.endPos = min(tf.endPos, n)
		tf.startPos = tf.endPos - tf.charWidth
		tf.startPos = max(0, tf.startPos)
		startIsAnchor = false
	}
	if tf.endPos < tf.startPos {
		return
	}

	if startIsAnchor {
		gotWidth := false
		spos := tf.charPos(tf.startPos).X
		for {
			w := tf.charPos(tf.endPos).X - spos
			if w < maxw {
				if tf.endPos == n {
					break
				}
				nw := tf.charPos(tf.endPos+1).X - spos
				if nw >= maxw {
					gotWidth = true
					break
				}
				tf.endPos++
			} else {
				tf.endPos--
			}
		}
		if gotWidth || tf.startPos == 0 {
			return
		}
		// otherwise, try getting some more chars by moving up start..
	}

	// end is now anchor
	epos := tf.charPos(tf.endPos).X
	for {
		w := epos - tf.charPos(tf.startPos).X
		if w < maxw {
			if tf.startPos == 0 {
				break
			}
			nw := epos - tf.charPos(tf.startPos-1).X
			if nw >= maxw {
				break
			}
			tf.startPos--
		} else {
			tf.startPos++
		}
	}
}

// pixelToCursor finds the cursor position that corresponds to the given pixel location
func (tf *TextField) pixelToCursor(pt image.Point) int {
	ptf := math32.Vector2FromPoint(pt)
	rpt := ptf.Sub(tf.effPos)
	if rpt.X <= 0 || rpt.Y < 0 {
		return tf.startPos
	}
	n := len(tf.editText)
	if tf.hasWordWrap() {
		si, ri, ok := tf.renderAll.PosToRune(rpt)
		if ok {
			ix, _ := tf.renderAll.SpanPosToRuneIndex(si, ri)
			ix = min(ix, n)
			return ix
		}
		return tf.startPos
	}
	pr := tf.PointToRelPos(pt)

	px := float32(pr.X)
	st := &tf.Styles
	c := tf.startPos + int(float64(px/st.UnitContext.Dots(units.UnitCh)))
	c = min(c, n)

	w := tf.relCharPos(tf.startPos, c).X
	if w > px {
		for w > px {
			c--
			if c <= tf.startPos {
				c = tf.startPos
				break
			}
			w = tf.relCharPos(tf.startPos, c).X
		}
	} else if w < px {
		for c < tf.endPos {
			wn := tf.relCharPos(tf.startPos, c+1).X
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
			tf.selectStart = oldPos
			tf.selectMode = true
		}
		if !tf.StateIs(states.Sliding) && selMode == events.SelectOne { // && tf.CursorPos >= tf.SelectStart && tf.CursorPos < tf.SelectEnd {
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
	st := &tf.Styles
	txs := &st.Text
	fs := st.FontRender()
	st.Font = paint.OpenFont(fs, &st.UnitContext)
	txt := tf.editText
	if tf.NoEcho {
		txt = concealDots(len(tf.editText))
	}
	align, alignV := txs.Align, txs.AlignV
	txs.Align, txs.AlignV = styles.Start, styles.Start // only works with this
	tf.renderAll.SetRunes(txt, fs, &st.UnitContext, txs, true, 0, 0)
	tf.renderAll.Layout(txs, fs, &st.UnitContext, sz)
	txs.Align, txs.AlignV = align, alignV
	rsz := tf.renderAll.BBox.Size().Ceil()
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
	tf.Frame.SizeUp()
	tmptxt := tf.editText
	if len(tf.text) == 0 && len(tf.Placeholder) > 0 {
		tf.editText = []rune(tf.Placeholder)
	} else {
		tf.editText = []rune(tf.text)
	}
	tf.startPos = 0
	tf.endPos = len(tf.editText)

	sz := &tf.Geom.Size
	icsz := tf.iconsSize()
	availSz := sz.Actual.Content.Sub(icsz)

	var rsz math32.Vector2
	if tf.hasWordWrap() {
		rsz = tf.configTextSize(availSz) // TextWrapSizeEstimate(availSz, len(tf.EditTxt), &tf.Styles.Font))
	} else {
		rsz = tf.configTextSize(availSz)
	}
	rsz.SetAdd(icsz)
	sz.fitSizeMax(&sz.Actual.Content, rsz)
	sz.setTotalFromContent(&sz.Actual)
	tf.fontHeight = tf.Styles.Font.Face.Metrics.Height
	tf.editText = tmptxt
	if DebugSettings.LayoutTrace {
		fmt.Println(tf, "TextField SizeUp:", rsz, "Actual:", sz.Actual.Content)
	}
}

func (tf *TextField) SizeDown(iter int) bool {
	if !tf.hasWordWrap() {
		return tf.Frame.SizeDown(iter)
	}
	sz := &tf.Geom.Size
	pgrow, _ := tf.growToAllocSize(sz.Actual.Content, sz.Alloc.Content) // key to grow
	icsz := tf.iconsSize()
	prevContent := sz.Actual.Content
	availSz := pgrow.Sub(icsz)
	rsz := tf.configTextSize(availSz)
	rsz.SetAdd(icsz)
	// start over so we don't reflect hysteresis of prior guess
	sz.setInitContentMin(tf.Styles.Min.Dots().Ceil())
	sz.fitSizeMax(&sz.Actual.Content, rsz)
	sz.setTotalFromContent(&sz.Actual)
	chg := prevContent != sz.Actual.Content
	if chg {
		if DebugSettings.LayoutTrace {
			fmt.Println(tf, "TextField Size Changed:", sz.Actual.Content, "was:", prevContent)
		}
	}
	sdp := tf.Frame.SizeDown(iter)
	return chg || sdp
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
	tf.numLines = len(tf.renderAll.Spans)
	if tf.numLines <= 1 {
		pos.Y += 0.5 * (sz.Y - tf.fontHeight) // center
	}
	tf.effSize = sz.Ceil()
	tf.effPos = pos.Ceil()
}

func (tf *TextField) Render() {
	defer func() {
		if tf.IsReadOnly() {
			return
		}
		if tf.StateIs(states.Focused) {
			tf.startCursor()
		} else {
			tf.stopCursor()
		}
	}()

	pc := &tf.Scene.PaintContext
	st := &tf.Styles

	tf.autoScroll() // inits paint with our style
	fs := st.FontRender()
	txs := &st.Text
	st.Font = paint.OpenFont(fs, &st.UnitContext)
	tf.RenderStandardBox()
	if tf.startPos < 0 || tf.endPos > len(tf.editText) {
		return
	}
	cur := tf.editText[tf.startPos:tf.endPos]
	tf.renderSelect()
	pos := tf.effPos
	prevColor := st.Color
	if len(tf.editText) == 0 && len(tf.Placeholder) > 0 {
		st.Color = tf.PlaceholderColor
		fs = st.FontRender() // need to update
		cur = []rune(tf.Placeholder)
	} else if tf.NoEcho {
		cur = concealDots(len(cur))
	}
	sz := &tf.Geom.Size
	icsz := tf.iconsSize()
	availSz := sz.Actual.Content.Sub(icsz)
	tf.renderVisible.SetRunes(cur, fs, &st.UnitContext, &st.Text, true, 0, 0)
	tf.renderVisible.Layout(txs, fs, &st.UnitContext, availSz)
	tf.renderVisible.Render(pc, pos)
	st.Color = prevColor
}

// IsWordBreak defines what counts as a word break for the purposes of selecting words.
// r1 is the rune in question, r2 is the rune past r1 in the direction you are moving.
// Pass -1 for r2 if there is no rune past r1.
func IsWordBreak(r1, r2 rune) bool {
	if r2 == -1 {
		if unicode.IsSpace(r1) || unicode.IsSymbol(r1) || unicode.IsPunct(r1) {
			return true
		}
		return false
	}
	if unicode.IsSpace(r1) || unicode.IsSymbol(r1) {
		return true
	}
	if unicode.IsPunct(r1) && r1 != rune('\'') {
		return true
	}
	if unicode.IsPunct(r1) && r1 == rune('\'') {
		if unicode.IsSpace(r2) || unicode.IsSymbol(r2) || unicode.IsPunct(r2) {
			return true
		}
		return false
	}
	return false
}

// concealDots creates an n-length []rune of bullet characters.
func concealDots(n int) []rune {
	dots := make([]rune, n)
	for i := range dots {
		dots[i] = '•'
	}
	return dots
}
