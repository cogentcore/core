// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"fmt"
	"image"
	"log/slog"
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
	undos TextFieldUndos
}

// TextFieldTypes is an enum containing the
// different possible types of text fields
type TextFieldTypes int32 //enums:enum -trim-prefix TextField

const (
	// TextFieldFilled represents a filled
	// TextField with a background color
	// and a bottom border
	TextFieldFilled TextFieldTypes = iota
	// TextFieldOutlined represents an outlined
	// TextField with a border on all sides
	// and no background color
	TextFieldOutlined
)

// Validator is an interface for types to provide a Validate method
// that is used to validate string [Value]s using [TextField.Validator].
type Validator interface {
	// Validate returns an error if the value is invalid.
	Validate() error
}

func (tf *TextField) WidgetValue() any { return &tf.text }

func (tf *TextField) OnBind(value any) {
	if vd, ok := value.(Validator); ok {
		tf.Validator = vd.Validate
	}
}

func (tf *TextField) Init() {
	tf.Frame.Init()
	tf.AddContextMenu(tf.ContextMenu)

	tf.Styler(func(s *styles.Style) {
		s.SetAbilities(true, abilities.Activatable, abilities.Focusable, abilities.Hoverable, abilities.Slideable, abilities.DoubleClickable, abilities.TripleClickable)
		tf.CursorWidth.Dp(1)
		tf.SelectColor = colors.Scheme.Select.Container
		tf.PlaceholderColor = colors.Scheme.OnSurfaceVariant
		tf.CursorColor = colors.Scheme.Primary.Base

		s.VirtualKeyboard = styles.KeyboardSingleLine
		if !tf.IsReadOnly() {
			s.Cursor = cursors.Text
		}
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
			s.Border.Color.Set(colors.Scheme.Outline)
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

	tf.HandleKeyEvents()
	tf.HandleSelectToggle()
	tf.OnFirst(events.Change, func(e events.Event) {
		tf.Validate()
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
			tf.SetCursorFromPixel(e.Pos(), e.SelectMode())
		case events.Middle:
			e.SetHandled()
			tf.SetCursorFromPixel(e.Pos(), e.SelectMode())
			tf.Paste()
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
		tf.SelectWord()
	})
	tf.On(events.TripleClick, func(e events.Event) {
		if tf.IsReadOnly() {
			return
		}
		if !tf.IsReadOnly() && !tf.StateIs(states.Focused) {
			tf.SetFocusEvent()
		}
		e.SetHandled()
		tf.SelectAll()
	})
	tf.On(events.SlideMove, func(e events.Event) {
		if tf.IsReadOnly() {
			return
		}
		e.SetHandled()
		if !tf.selectMode {
			tf.SelectModeToggle()
		}
		tf.SetCursorFromPixel(e.Pos(), events.SelectOne)
	})
	tf.OnClose(func(e events.Event) {
		tf.EditDone() // todo: this must be protected against something else, for race detector
	})

	tf.Maker(func(p *tree.Plan) {
		tf.editText = []rune(tf.text)
		tf.edited = false

		if tf.IsReadOnly() {
			return
		}
		if tf.LeadingIcon.IsSet() {
			tree.AddAt(p, "lead-icon", func(w *Button) {
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
		}
		if tf.TrailingIcon.IsSet() {
			tree.AddAt(p, "trail-icon-stretch", func(w *Stretch) {
				w.Styler(func(s *styles.Style) {
					s.Grow.Set(1, 0)
				})
			})
			tree.AddAt(p, "trail-icon", func(w *Button) {
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
				})
			})
		}
	})
}

func (tf *TextField) Destroy() {
	tf.StopCursor()
	tf.Frame.Destroy()
}

// Text returns the current text -- applies any unapplied changes first, and
// sends a signal if so -- this is the end-user method to get the current
// value of the field.
func (tf *TextField) Text() string {
	tf.EditDone()
	return tf.text
}

// SetText sets the text to be edited and reverts any current edit
// to reflect this new text.
func (tf *TextField) SetText(txt string) *TextField {
	if tf.text == txt && !tf.edited {
		return tf
	}
	tf.text = txt
	tf.Revert()
	return tf
}

// SetLeadingIcon sets the leading icon of the text field to the given icon.
// If an on click function is specified, it also sets the leading icon on click
// function to that function. If no function is specified, it does not
// override any already set function.
func (tf *TextField) SetLeadingIcon(icon icons.Icon, onClick ...func(e events.Event)) *TextField {
	tf.LeadingIcon = icon
	if len(onClick) > 0 {
		tf.LeadingIconOnClick = onClick[0]
	}
	return tf
}

// SetTrailingIcon sets the trailing icon of the text field to the given icon.
// If an on click function is specified, it also sets the trailing icon on click
// function to that function. If no function is specified, it does not
// override any already set function.
func (tf *TextField) SetTrailingIcon(icon icons.Icon, onClick ...func(e events.Event)) *TextField {
	tf.TrailingIcon = icon
	if len(onClick) > 0 {
		tf.TrailingIconOnClick = onClick[0]
	}
	return tf
}

// AddClearButton adds a trailing icon button at the end
// of the textfield that clears the text in the textfield when pressed
func (tf *TextField) AddClearButton() *TextField {
	return tf.SetTrailingIcon(icons.Close, func(e events.Event) {
		tf.Clear()
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
		if icon := tf.TrailingIconButton(); icon != nil {
			icon.SetIcon(tf.TrailingIcon).Update()
		}
	}).Styler(func(s *styles.Style) {
		s.VirtualKeyboard = styles.KeyboardPassword
	})
	return tf
}

// EditDone completes editing and copies the active edited text to the text --
// called when the return key is pressed or goes out of focus
func (tf *TextField) EditDone() {
	if tf.edited {
		tf.edited = false
		tf.text = string(tf.editText)
		tf.SendChange()
		// widget can be killed after SendChange
		if tf.This == nil {
			return
		}
	}
	tf.ClearSelected()
	tf.ClearCursor()
}

// Revert aborts editing and reverts to last saved text
func (tf *TextField) Revert() {
	tf.editText = []rune(tf.text)
	tf.edited = false
	tf.startPos = 0
	tf.endPos = tf.charWidth
	tf.SelectReset()
	tf.NeedsRender()
}

// Clear clears any existing text
func (tf *TextField) Clear() {
	tf.edited = true
	tf.editText = tf.editText[:0]
	tf.startPos = 0
	tf.endPos = 0
	tf.SelectReset()
	tf.SetFocusEvent() // this is essential for ensuring that the clear applies after focus is lost..
	tf.NeedsRender()
}

// ClearError clears any existing validation error
func (tf *TextField) ClearError() {
	tf.error = nil
	tf.NeedsRender()
}

// Validate runs [TextField.Validator] and takes any necessary actions
// as a result of that.
func (tf *TextField) Validate() {
	if tf.Validator == nil {
		return
	}
	err := tf.Validator()
	if err == nil {
		if tf.error == nil {
			return
		}
		tf.error = nil
		tf.TrailingIconButton().SetIcon(tf.TrailingIcon).Update()
		return
	}
	tf.error = err
	if tf.TrailingIconButton() == nil {
		tf.SetTrailingIcon(icons.Blank).Update()
	}
	tf.TrailingIconButton().SetIcon(icons.Error).Update()
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

// CursorForward moves the cursor forward
func (tf *TextField) CursorForward(steps int) {
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
		tf.SelectRegUpdate(tf.cursorPos)
	}
	tf.NeedsRender()
}

// CursorForwardWord moves the cursor forward by words
func (tf *TextField) CursorForwardWord(steps int) {
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
		tf.SelectRegUpdate(tf.cursorPos)
	}
	tf.NeedsRender()
}

// CursorBackward moves the cursor backward
func (tf *TextField) CursorBackward(steps int) {
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
		tf.SelectRegUpdate(tf.cursorPos)
	}
	tf.NeedsRender()
}

// CursorBackwardWord moves the cursor backward by words
func (tf *TextField) CursorBackwardWord(steps int) {
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
		tf.SelectRegUpdate(tf.cursorPos)
	}
	tf.NeedsRender()
}

// CursorDown moves the cursor down
func (tf *TextField) CursorDown(steps int) {
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
		tf.SelectRegUpdate(tf.cursorPos)
	}
	tf.NeedsRender()
}

// CursorUp moves the cursor up
func (tf *TextField) CursorUp(steps int) {
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
		tf.SelectRegUpdate(tf.cursorPos)
	}
	tf.NeedsRender()
}

// CursorStart moves the cursor to the start of the text, updating selection
// if select mode is active
func (tf *TextField) CursorStart() {
	tf.cursorPos = 0
	tf.startPos = 0
	tf.endPos = min(len(tf.editText), tf.startPos+tf.charWidth)
	if tf.selectMode {
		tf.SelectRegUpdate(tf.cursorPos)
	}
	tf.NeedsRender()
}

// CursorEnd moves the cursor to the end of the text
func (tf *TextField) CursorEnd() {
	ed := len(tf.editText)
	tf.cursorPos = ed
	tf.endPos = len(tf.editText) // try -- display will adjust
	tf.startPos = max(0, tf.endPos-tf.charWidth)
	if tf.selectMode {
		tf.SelectRegUpdate(tf.cursorPos)
	}
	tf.NeedsRender()
}

// CursorBackspace deletes character(s) immediately before cursor
func (tf *TextField) CursorBackspace(steps int) {
	if tf.HasSelection() {
		tf.DeleteSelection()
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
	tf.CursorBackward(steps)
	tf.NeedsRender()
}

// CursorDelete deletes character(s) immediately after the cursor
func (tf *TextField) CursorDelete(steps int) {
	if tf.HasSelection() {
		tf.DeleteSelection()
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

// CursorBackspaceWord deletes words(s) immediately before cursor
func (tf *TextField) CursorBackspaceWord(steps int) {
	if tf.HasSelection() {
		tf.DeleteSelection()
		return
	}
	org := tf.cursorPos
	tf.CursorBackwardWord(steps)
	tf.edited = true
	tf.editText = append(tf.editText[:tf.cursorPos], tf.editText[org:]...)
	tf.NeedsRender()
}

// CursorDeleteWord deletes word(s) immediately after the cursor
func (tf *TextField) CursorDeleteWord(steps int) {
	if tf.HasSelection() {
		tf.DeleteSelection()
		return
	}
	// note: no update b/c signal from buf will drive update
	org := tf.cursorPos
	tf.CursorForwardWord(steps)
	tf.edited = true
	tf.editText = append(tf.editText[:tf.cursorPos], tf.editText[org:]...)
	tf.NeedsRender()
}

// CursorKill deletes text from cursor to end of text
func (tf *TextField) CursorKill() {
	steps := len(tf.editText) - tf.cursorPos
	tf.CursorDelete(steps)
}

///////////////////////////////////////////////////////////////////////////////
//    Selection

// ClearSelected resets both the global selected flag and any current selection
func (tf *TextField) ClearSelected() {
	tf.SetState(false, states.Selected)
	tf.SelectReset()
}

// HasSelection returns whether there is a selected region of text
func (tf *TextField) HasSelection() bool {
	tf.SelectUpdate()
	return tf.selectStart < tf.selectEnd
}

// Selection returns the currently selected text
func (tf *TextField) Selection() string {
	if tf.HasSelection() {
		return string(tf.editText[tf.selectStart:tf.selectEnd])
	}
	return ""
}

// SelectModeToggle toggles the SelectMode, updating selection with cursor movement
func (tf *TextField) SelectModeToggle() {
	if tf.selectMode {
		tf.selectMode = false
	} else {
		tf.selectMode = true
		tf.selectInit = tf.cursorPos
		tf.selectStart = tf.cursorPos
		tf.selectEnd = tf.selectStart
	}
}

// ShiftSelect sets the selection start if the shift key is down but wasn't previously.
// If the shift key has been released, the selection info is cleared.
func (tf *TextField) ShiftSelect(e events.Event) {
	hasShift := e.HasAnyModifier(key.Shift)
	if hasShift && !tf.selectMode {
		tf.SelectModeToggle()
	}
	if !hasShift && tf.selectMode {
		tf.SelectReset()
	}
}

// SelectRegUpdate updates current select region based on given cursor position
// relative to SelectStart position
func (tf *TextField) SelectRegUpdate(pos int) {
	if pos < tf.selectInit {
		tf.selectStart = pos
		tf.selectEnd = tf.selectInit
	} else {
		tf.selectStart = tf.selectInit
		tf.selectEnd = pos
	}
	tf.SelectUpdate()
}

// SelectAll selects all the text
func (tf *TextField) SelectAll() {
	tf.selectStart = 0
	tf.selectInit = 0
	tf.selectEnd = len(tf.editText)
	if TheApp.SystemPlatform().IsMobile() {
		tf.Send(events.ContextMenu)
	}
	tf.NeedsRender()
}

// IsWordBreak defines what counts as a word break for the purposes of selecting words
func (tf *TextField) IsWordBreak(r rune) bool {
	return unicode.IsSpace(r) || unicode.IsSymbol(r) || unicode.IsPunct(r)
}

// SelectWord selects the word (whitespace delimited) that the cursor is on
func (tf *TextField) SelectWord() {
	sz := len(tf.editText)
	if sz <= 3 {
		tf.SelectAll()
		return
	}
	tf.selectStart = tf.cursorPos
	if tf.selectStart >= sz {
		tf.selectStart = sz - 2
	}
	if !tf.IsWordBreak(tf.editText[tf.selectStart]) {
		for tf.selectStart > 0 {
			if tf.IsWordBreak(tf.editText[tf.selectStart-1]) {
				break
			}
			tf.selectStart--
		}
		tf.selectEnd = tf.cursorPos + 1
		for tf.selectEnd < sz {
			if tf.IsWordBreak(tf.editText[tf.selectEnd]) {
				break
			}
			tf.selectEnd++
		}
	} else { // keep the space start -- go to next space..
		tf.selectEnd = tf.cursorPos + 1
		for tf.selectEnd < sz {
			if !tf.IsWordBreak(tf.editText[tf.selectEnd]) {
				break
			}
			tf.selectEnd++
		}
		for tf.selectEnd < sz { // include all trailing spaces
			if tf.IsWordBreak(tf.editText[tf.selectEnd]) {
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

// SelectReset resets the selection
func (tf *TextField) SelectReset() {
	tf.selectMode = false
	if tf.selectStart == 0 && tf.selectEnd == 0 {
		return
	}
	tf.selectStart = 0
	tf.selectEnd = 0
	tf.NeedsRender()
}

// SelectUpdate updates the select region after any change to the text, to keep it in range
func (tf *TextField) SelectUpdate() {
	if tf.selectStart < tf.selectEnd {
		ed := len(tf.editText)
		if tf.selectStart < 0 {
			tf.selectStart = 0
		}
		if tf.selectEnd > ed {
			tf.selectEnd = ed
		}
	} else {
		tf.SelectReset()
	}
}

// Cut cuts any selected text and adds it to the clipboard.
func (tf *TextField) Cut() { //types:add
	if tf.NoEcho {
		return
	}
	cut := tf.DeleteSelection()
	if cut != "" {
		em := tf.Events()
		if em != nil {
			em.Clipboard().Write(mimedata.NewText(cut))
		}
	}
}

// DeleteSelection deletes any selected text, without adding to clipboard --
// returns text deleted
func (tf *TextField) DeleteSelection() string {
	tf.SelectUpdate()
	if !tf.HasSelection() {
		return ""
	}
	cut := tf.Selection()
	tf.edited = true
	tf.editText = append(tf.editText[:tf.selectStart], tf.editText[tf.selectEnd:]...)
	if tf.cursorPos > tf.selectStart {
		if tf.cursorPos < tf.selectEnd {
			tf.cursorPos = tf.selectStart
		} else {
			tf.cursorPos -= tf.selectEnd - tf.selectStart
		}
	}
	tf.SelectReset()
	tf.NeedsRender()
	return cut
}

// Copy copies any selected text to the clipboard.
func (tf *TextField) Copy() { //types:add
	if tf.NoEcho {
		return
	}
	tf.SelectUpdate()
	if !tf.HasSelection() {
		return
	}

	md := mimedata.NewText(tf.Text())
	tf.Clipboard().Write(md)
}

// Paste inserts text from the clipboard at current cursor position; if
// cursor is within a current selection, that selection is replaced.
func (tf *TextField) Paste() { //types:add
	data := tf.Clipboard().Read([]string{mimedata.TextPlain})
	if data != nil {
		if tf.cursorPos >= tf.selectStart && tf.cursorPos < tf.selectEnd {
			tf.DeleteSelection()
		}
		tf.InsertAtCursor(data.Text(mimedata.TextPlain))
	}
}

// InsertAtCursor inserts the given text at current cursor position.
func (tf *TextField) InsertAtCursor(str string) {
	if tf.HasSelection() {
		tf.Cut()
	}
	tf.edited = true
	rs := []rune(str)
	rsl := len(rs)
	nt := append(tf.editText, rs...)               // first append to end
	copy(nt[tf.cursorPos+rsl:], nt[tf.cursorPos:]) // move stuff to end
	copy(nt[tf.cursorPos:], rs)                    // copy into position
	tf.editText = nt
	tf.endPos += rsl
	tf.CursorForward(rsl)
	tf.NeedsRender()
}

func (tf *TextField) ContextMenu(m *Scene) {
	NewFuncButton(m).SetFunc(tf.Copy).SetIcon(icons.Copy).SetKey(keymap.Copy).SetState(tf.NoEcho || !tf.HasSelection(), states.Disabled)
	if !tf.IsReadOnly() {
		NewFuncButton(m).SetFunc(tf.Cut).SetIcon(icons.Cut).SetKey(keymap.Cut).SetState(tf.NoEcho || !tf.HasSelection(), states.Disabled)
		paste := NewFuncButton(m).SetFunc(tf.Paste).SetIcon(icons.Paste).SetKey(keymap.Paste)
		cb := tf.Scene.Events.Clipboard()
		if cb != nil {
			paste.SetState(cb.IsEmpty(), states.Disabled)
		}
	}
}

///////////////////////////////////////////////////////////////////////////////
//    Undo

// TextFieldUndoRecord holds one undo record
type TextFieldUndoRecord struct {
	Text      []rune
	CursorPos int
}

func (ur *TextFieldUndoRecord) Set(txt []rune, curpos int) {
	ur.Text = slices.Clone(txt)
	ur.CursorPos = curpos
}

// TextFieldUndos manages everything about the undo process for a [TextField].
type TextFieldUndos struct {
	// stack of undo records
	Stack []TextFieldUndoRecord

	// position within the undo stack
	Pos int

	// last time undo was saved, for grouping
	LastSave time.Time
}

func (us *TextFieldUndos) SaveUndo(txt []rune, curpos int) {
	n := len(us.Stack)
	now := time.Now()
	ts := now.Sub(us.LastSave)
	if n > 0 && ts < 250*time.Millisecond {
		r := us.Stack[n-1]
		r.Set(txt, curpos)
		us.Stack[n-1] = r
		return
	}
	r := TextFieldUndoRecord{}
	r.Set(txt, curpos)
	us.Stack = append(us.Stack, r)
	us.Pos = len(us.Stack)
	us.LastSave = now
}

func (tf *TextField) SaveUndo() {
	tf.undos.SaveUndo(tf.editText, tf.cursorPos)
}

func (us *TextFieldUndos) Undo(txt []rune, curpos int) *TextFieldUndoRecord {
	n := len(us.Stack)
	if us.Pos <= 0 || n == 0 {
		return &TextFieldUndoRecord{}
	}
	if us.Pos == n {
		us.LastSave = time.Time{}
		us.SaveUndo(txt, curpos)
		us.Pos--
	}
	us.Pos--
	us.LastSave = time.Time{} // prevent any merging
	r := &us.Stack[us.Pos]
	return r
}

func (tf *TextField) Undo() {
	r := tf.undos.Undo(tf.editText, tf.cursorPos)
	if r != nil {
		tf.editText = r.Text
		tf.cursorPos = r.CursorPos
		tf.NeedsRender()
	}
}

func (us *TextFieldUndos) Redo() *TextFieldUndoRecord {
	n := len(us.Stack)
	if us.Pos >= n-1 {
		return nil
	}
	us.LastSave = time.Time{} // prevent any merging
	us.Pos++
	return &us.Stack[us.Pos]
}

func (tf *TextField) Redo() {
	r := tf.undos.Redo()
	if r != nil {
		tf.editText = r.Text
		tf.cursorPos = r.CursorPos
		tf.NeedsRender()
	}
}

///////////////////////////////////////////////////////////////////////////////
//    Complete

// SetCompleter sets completion functions so that completions will
// automatically be offered as the user types
func (tf *TextField) SetCompleter(data any, matchFun complete.MatchFunc, editFun complete.EditFunc) {
	if matchFun == nil || editFun == nil {
		tf.complete = nil
		return
	}
	tf.complete = NewComplete().SetContext(data).SetMatchFunc(matchFun).SetEditFunc(editFun)
	tf.complete.OnSelect(func(e events.Event) {
		tf.CompleteText(tf.complete.Completion)
	})
}

// OfferComplete pops up a menu of possible completions
func (tf *TextField) OfferComplete() {
	if tf.complete == nil {
		return
	}
	s := string(tf.editText[0:tf.cursorPos])
	cpos := tf.CharRenderPos(tf.cursorPos, true).ToPoint()
	cpos.X += 5
	cpos.Y = tf.Geom.TotalBBox.Max.Y
	tf.complete.SrcLn = 0
	tf.complete.SrcCh = tf.cursorPos
	tf.complete.Show(tf, cpos, s)
}

// CancelComplete cancels any pending completion -- call this when new events
// have moved beyond any prior completion scenario
func (tf *TextField) CancelComplete() {
	if tf.complete == nil {
		return
	}
	tf.complete.Cancel()
}

// CompleteText edits the text field using the string chosen from the completion menu
func (tf *TextField) CompleteText(s string) {
	txt := string(tf.editText) // Reminder: do NOT call tf.Text() in an active editing context!
	c := tf.complete.GetCompletion(s)
	ed := tf.complete.EditFunc(tf.complete.Context, txt, tf.cursorPos, c, tf.complete.Seed)
	st := tf.cursorPos - len(tf.complete.Seed)
	tf.cursorPos = st
	tf.CursorDelete(ed.ForwardDelete)
	tf.InsertAtCursor(ed.NewText)
	tf.EditDone()
}

///////////////////////////////////////////////////////////////////////////////
//    Rendering

// HasWordWrap returns true if the layout is multi-line word wrapping
func (tf *TextField) HasWordWrap() bool {
	return tf.Styles.Text.HasWordWrap()
}

// CharPos returns the relative starting position of the given rune,
// in the overall RenderAll of all the text.
// These positions can be out of visible range: see CharRenderPos
func (tf *TextField) CharPos(idx int) math32.Vector2 {
	if idx <= 0 || len(tf.renderAll.Spans) == 0 {
		return math32.Vector2{}
	}
	pos, _, _, _ := tf.renderAll.RuneRelPos(idx)
	pos.Y -= tf.renderAll.Spans[0].RelPos.Y
	return pos
}

// RelCharPos returns the text width in dots between the two text string
// positions (ed is exclusive -- +1 beyond actual char).
func (tf *TextField) RelCharPos(st, ed int) math32.Vector2 {
	return tf.CharPos(ed).Sub(tf.CharPos(st))
}

// CharRenderPos returns the starting render coords for the given character
// position in string -- makes no attempt to rationalize that pos (i.e., if
// not in visible range, position will be out of range too).
// if wincoords is true, then adds window box offset -- for cursor, popups
func (tf *TextField) CharRenderPos(charidx int, wincoords bool) math32.Vector2 {
	pos := tf.effPos
	if wincoords {
		sc := tf.Scene
		pos = pos.Add(math32.Vector2FromPoint(sc.sceneGeom.Pos))
	}
	cpos := tf.RelCharPos(tf.startPos, charidx)
	return pos.Add(cpos)
}

var (
	// TextFieldBlinker manages cursor blinking
	TextFieldBlinker = Blinker{}

	// TextFieldSpriteName is the name of the window sprite used for the cursor
	TextFieldSpriteName = "TextField.Cursor"
)

func init() {
	TheApp.AddQuitCleanFunc(TextFieldBlinker.QuitClean)
	TextFieldBlinker.Func = func() {
		w := TextFieldBlinker.Widget
		if w == nil {
			return
		}
		tf := AsTextField(w)
		if !w.AsWidget().StateIs(states.Focused) || !w.IsVisible() {
			tf.blinkOn = false
			tf.RenderCursor(false)
		} else {
			tf.blinkOn = !tf.blinkOn
			tf.RenderCursor(tf.blinkOn)
		}
	}
}

// StartCursor starts the cursor blinking and renders it
func (tf *TextField) StartCursor() {
	if tf == nil || tf.This == nil {
		return
	}
	if !tf.This.(Widget).IsVisible() {
		return
	}
	tf.blinkOn = true
	tf.RenderCursor(true)
	if SystemSettings.CursorBlinkTime == 0 {
		return
	}
	TextFieldBlinker.SetWidget(tf.This.(Widget))
	TextFieldBlinker.Blink(SystemSettings.CursorBlinkTime)
}

// ClearCursor turns off cursor and stops it from blinking
func (tf *TextField) ClearCursor() {
	if tf.IsReadOnly() {
		return
	}
	tf.StopCursor()
	tf.RenderCursor(false)
}

// StopCursor stops the cursor from blinking
func (tf *TextField) StopCursor() {
	if tf == nil || tf.This == nil {
		return
	}
	TextFieldBlinker.ResetWidget(tf.This.(Widget))
}

// RenderCursor renders the cursor on or off, as a sprite that is either on or off
func (tf *TextField) RenderCursor(on bool) {
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
		spnm := fmt.Sprintf("%v-%v", TextFieldSpriteName, tf.fontHeight)
		ms.Sprites.InactivateSprite(spnm)
		return
	}
	if !tf.This.(Widget).IsVisible() {
		return
	}

	tf.cursorMu.Lock()
	defer tf.cursorMu.Unlock()

	sp := tf.CursorSprite(on)
	if sp == nil {
		return
	}
	sp.Geom.Pos = tf.CharRenderPos(tf.cursorPos, true).ToPointFloor()
}

// CursorSprite returns the Sprite for the cursor (which is
// only rendered once with a vertical bar, and just activated and inactivated
// depending on render status).  On sets the On status of the cursor.
func (tf *TextField) CursorSprite(on bool) *Sprite {
	sc := tf.Scene
	if sc == nil {
		return nil
	}
	ms := sc.Stage.Main
	if ms == nil {
		return nil // only MainStage has sprites
	}
	spnm := fmt.Sprintf("%v-%v", TextFieldSpriteName, tf.fontHeight)
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

// RenderSelect renders the selected region, if any, underneath the text
func (tf *TextField) RenderSelect() {
	if !tf.HasSelection() {
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

	spos := tf.CharRenderPos(effst, false)

	pc := &tf.Scene.PaintContext
	tsz := tf.RelCharPos(effst, effed)
	if !tf.HasWordWrap() || tsz.Y == 0 {
		pc.FillBox(spos, math32.Vec2(tsz.X, tf.fontHeight), tf.SelectColor)
		return
	}
	ex := float32(tf.Geom.ContentBBox.Max.X)
	sx := float32(tf.Geom.ContentBBox.Min.X)
	ssi, _, _ := tf.renderAll.RuneSpanPos(effst)
	esi, _, _ := tf.renderAll.RuneSpanPos(effed)
	ep := tf.CharRenderPos(effed, false)

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

// AutoScroll scrolls the starting position to keep the cursor visible
func (tf *TextField) AutoScroll() {
	sz := &tf.Geom.Size
	icsz := tf.IconsSize()
	availSz := sz.Actual.Content.Sub(icsz)
	tf.ConfigTextSize(availSz)
	n := len(tf.editText)
	tf.cursorPos = math32.ClampInt(tf.cursorPos, 0, n)

	if tf.HasWordWrap() { // does not scroll
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
		spos := tf.CharPos(tf.startPos).X
		for {
			w := tf.CharPos(tf.endPos).X - spos
			if w < maxw {
				if tf.endPos == n {
					break
				}
				nw := tf.CharPos(tf.endPos+1).X - spos
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
	epos := tf.CharPos(tf.endPos).X
	for {
		w := epos - tf.CharPos(tf.startPos).X
		if w < maxw {
			if tf.startPos == 0 {
				break
			}
			nw := epos - tf.CharPos(tf.startPos-1).X
			if nw >= maxw {
				break
			}
			tf.startPos--
		} else {
			tf.startPos++
		}
	}
}

// PixelToCursor finds the cursor position that corresponds to the given pixel location
func (tf *TextField) PixelToCursor(pt image.Point) int {
	ptf := math32.Vector2FromPoint(pt)
	rpt := ptf.Sub(tf.effPos)
	if rpt.X <= 0 || rpt.Y < 0 {
		return tf.startPos
	}
	n := len(tf.editText)
	if tf.HasWordWrap() {
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

	w := tf.RelCharPos(tf.startPos, c).X
	if w > px {
		for w > px {
			c--
			if c <= tf.startPos {
				c = tf.startPos
				break
			}
			w = tf.RelCharPos(tf.startPos, c).X
		}
	} else if w < px {
		for c < tf.endPos {
			wn := tf.RelCharPos(tf.startPos, c+1).X
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

// SetCursorFromPixel finds cursor location from given scene-relative
// pixel location, and sets current cursor to it, updating selection too.
func (tf *TextField) SetCursorFromPixel(pt image.Point, selMode events.SelectModes) {
	oldPos := tf.cursorPos
	tf.cursorPos = tf.PixelToCursor(pt)
	if tf.selectMode || selMode != events.SelectOne {
		if !tf.selectMode && selMode != events.SelectOne {
			tf.selectStart = oldPos
			tf.selectMode = true
		}
		if !tf.StateIs(states.Sliding) && selMode == events.SelectOne { // && tf.CursorPos >= tf.SelectStart && tf.CursorPos < tf.SelectEnd {
			tf.SelectReset()
		} else {
			tf.SelectRegUpdate(tf.cursorPos)
		}
		tf.SelectUpdate()
	} else if tf.HasSelection() {
		tf.SelectReset()
	}
	tf.NeedsRender()
}

func (tf *TextField) HandleKeyEvents() {
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
			tf.ShiftSelect(e)
			tf.CursorForward(1)
			tf.OfferComplete()
		case keymap.WordRight:
			e.SetHandled()
			tf.ShiftSelect(e)
			tf.CursorForwardWord(1)
			tf.OfferComplete()
		case keymap.MoveLeft:
			e.SetHandled()
			tf.ShiftSelect(e)
			tf.CursorBackward(1)
			tf.OfferComplete()
		case keymap.WordLeft:
			e.SetHandled()
			tf.ShiftSelect(e)
			tf.CursorBackwardWord(1)
			tf.OfferComplete()
		case keymap.MoveDown:
			if tf.numLines > 1 {
				e.SetHandled()
				tf.ShiftSelect(e)
				tf.CursorDown(1)
			}
		case keymap.MoveUp:
			if tf.numLines > 1 {
				e.SetHandled()
				tf.ShiftSelect(e)
				tf.CursorUp(1)
			}
		case keymap.Home:
			e.SetHandled()
			tf.ShiftSelect(e)
			tf.CancelComplete()
			tf.CursorStart()
		case keymap.End:
			e.SetHandled()
			tf.ShiftSelect(e)
			tf.CancelComplete()
			tf.CursorEnd()
		case keymap.SelectMode:
			e.SetHandled()
			tf.CancelComplete()
			tf.SelectModeToggle()
		case keymap.CancelSelect:
			e.SetHandled()
			tf.CancelComplete()
			tf.SelectReset()
		case keymap.SelectAll:
			e.SetHandled()
			tf.CancelComplete()
			tf.SelectAll()
		case keymap.Copy:
			e.SetHandled()
			tf.CancelComplete()
			tf.Copy()
		}
		if tf.IsReadOnly() || e.IsHandled() {
			return
		}
		switch kf {
		case keymap.Enter:
			fallthrough
		case keymap.FocusNext: // we process tab to make it EditDone as opposed to other ways of losing focus
			e.SetHandled()
			tf.CancelComplete()
			tf.EditDone()
			tf.FocusNext()
		case keymap.Accept: // ctrl+enter
			e.SetHandled()
			tf.CancelComplete()
			tf.EditDone()
		case keymap.FocusPrev:
			e.SetHandled()
			tf.CancelComplete()
			tf.EditDone()
			tf.FocusPrev()
		case keymap.Abort: // esc
			e.SetHandled()
			tf.CancelComplete()
			tf.Revert()
			// tf.FocusChanged(FocusInactive)
		case keymap.Backspace:
			e.SetHandled()
			tf.SaveUndo()
			tf.CursorBackspace(1)
			tf.OfferComplete()
			tf.Send(events.Input, e)
		case keymap.Kill:
			e.SetHandled()
			tf.CancelComplete()
			tf.CursorKill()
			tf.Send(events.Input, e)
		case keymap.Delete:
			e.SetHandled()
			tf.SaveUndo()
			tf.CursorDelete(1)
			tf.OfferComplete()
			tf.Send(events.Input, e)
		case keymap.BackspaceWord:
			e.SetHandled()
			tf.SaveUndo()
			tf.CursorBackspaceWord(1)
			tf.OfferComplete()
			tf.Send(events.Input, e)
		case keymap.DeleteWord:
			e.SetHandled()
			tf.SaveUndo()
			tf.CursorDeleteWord(1)
			tf.OfferComplete()
			tf.Send(events.Input, e)
		case keymap.Cut:
			e.SetHandled()
			tf.SaveUndo()
			tf.CancelComplete()
			tf.Cut()
			tf.Send(events.Input, e)
		case keymap.Paste:
			e.SetHandled()
			tf.SaveUndo()
			tf.CancelComplete()
			tf.Paste()
			tf.Send(events.Input, e)
		case keymap.Undo:
			e.SetHandled()
			tf.Undo()
		case keymap.Redo:
			e.SetHandled()
			tf.Redo()
		case keymap.Complete:
			e.SetHandled()
			tf.OfferComplete()
		case keymap.None:
			if unicode.IsPrint(e.KeyRune()) {
				if !e.HasAnyModifier(key.Control, key.Meta) {
					e.SetHandled()
					tf.SaveUndo()
					tf.InsertAtCursor(string(e.KeyRune()))
					if e.KeyRune() == ' ' {
						tf.CancelComplete()
					} else {
						tf.OfferComplete()
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
		tf.EditDone()
	})
}

////////////////////////////////////////////////////
//  Widget Interface

func (tf *TextField) Style() {
	tf.WidgetBase.Style()
	tf.CursorWidth.ToDots(&tf.Styles.UnitContext)
}

func (tf *TextField) ConfigTextSize(sz math32.Vector2) math32.Vector2 {
	st := &tf.Styles
	txs := &st.Text
	fs := st.FontRender()
	st.Font = paint.OpenFont(fs, &st.UnitContext)
	txt := tf.editText
	if tf.NoEcho {
		txt = ConcealDots(len(tf.editText))
	}
	align, alignV := txs.Align, txs.AlignV
	txs.Align, txs.AlignV = styles.Start, styles.Start // only works with this
	tf.renderAll.SetRunes(txt, fs, &st.UnitContext, txs, true, 0, 0)
	tf.renderAll.Layout(txs, fs, &st.UnitContext, sz)
	txs.Align, txs.AlignV = align, alignV
	rsz := tf.renderAll.BBox.Size().Ceil()
	return rsz
}

func (tf *TextField) IconsSize() math32.Vector2 {
	var sz math32.Vector2
	if lead := tf.LeadingIconButton(); lead != nil {
		sz.X += lead.Geom.Size.Actual.Total.X
	}
	if trail := tf.TrailingIconButton(); trail != nil {
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
	icsz := tf.IconsSize()
	availSz := sz.Actual.Content.Sub(icsz)

	var rsz math32.Vector2
	if tf.HasWordWrap() {
		rsz = tf.ConfigTextSize(availSz) // TextWrapSizeEstimate(availSz, len(tf.EditTxt), &tf.Styles.Font))
	} else {
		rsz = tf.ConfigTextSize(availSz)
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
	if !tf.HasWordWrap() {
		return tf.Frame.SizeDown(iter)
	}
	sz := &tf.Geom.Size
	pgrow, _ := tf.growToAllocSize(sz.Actual.Content, sz.Alloc.Content) // key to grow
	icsz := tf.IconsSize()
	prevContent := sz.Actual.Content
	availSz := pgrow.Sub(icsz)
	rsz := tf.ConfigTextSize(availSz)
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
	tf.SetEffPosAndSize()
}

// LeadingIconButton returns the [LeadingIcon] [Button] if present.
func (tf *TextField) LeadingIconButton() *Button {
	bt, _ := tf.ChildByName("lead-icon", 0).(*Button)
	return bt
}

// TrailingIconButton returns the [TrailingIcon] [Button] if present.
func (tf *TextField) TrailingIconButton() *Button {
	bt, _ := tf.ChildByName("trail-icon", 1).(*Button)
	return bt
}

// SetEffPosAndSize sets the effective position and size of
// the textfield based on its base position and size
// and its icons or lack thereof
func (tf *TextField) SetEffPosAndSize() {
	sz := tf.Geom.Size.Actual.Content
	pos := tf.Geom.Pos.Content
	if lead := tf.LeadingIconButton(); lead != nil {
		pos.X += lead.Geom.Size.Actual.Total.X
		sz.X -= lead.Geom.Size.Actual.Total.X
	}
	if trail := tf.TrailingIconButton(); trail != nil {
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
			tf.StartCursor()
		} else {
			tf.StopCursor()
		}
	}()

	pc := &tf.Scene.PaintContext
	st := &tf.Styles

	tf.AutoScroll() // inits paint with our style
	fs := st.FontRender()
	txs := &st.Text
	st.Font = paint.OpenFont(fs, &st.UnitContext)
	tf.RenderStandardBox()
	if tf.startPos < 0 || tf.endPos > len(tf.editText) {
		return
	}
	cur := tf.editText[tf.startPos:tf.endPos]
	tf.RenderSelect()
	pos := tf.effPos
	prevColor := st.Color
	if len(tf.editText) == 0 && len(tf.Placeholder) > 0 {
		st.Color = tf.PlaceholderColor
		fs = st.FontRender() // need to update
		cur = []rune(tf.Placeholder)
	} else if tf.NoEcho {
		cur = ConcealDots(len(cur))
	}
	sz := &tf.Geom.Size
	icsz := tf.IconsSize()
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

// ConcealDots creates an n-length []rune of bullet characters.
func ConcealDots(n int) []rune {
	dots := make([]rune, n)
	for i := range dots {
		dots[i] = 0x2022 // bullet character •
	}
	return dots
}
