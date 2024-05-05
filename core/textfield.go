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
// With the default WhiteSpaceNormal style setting,
// text will wrap onto multiple lines as needed.
// Set to WhiteSpaceNowrap (e.g., Styles.SetTextWrap(false)) to
// force everything to be on a single line.
// With multi-line wrapped text, the text is still treated as a contiguous
// wrapped text.
type TextField struct {
	WidgetBase

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
	LeadingIcon icons.Icon `set:"-"`

	// LeadingIconOnClick, if specified, is the function to call when
	// the LeadingIcon is clicked. If this is nil, the leading icon
	// will not be interactive.
	LeadingIconOnClick func(e events.Event) `json:"-" xml:"-"`

	// TrailingIcon, if specified, indicates to add a button
	// at the end of the text field with this icon.
	TrailingIcon icons.Icon `set:"-"`

	// TrailingIconOnClick, if specified, is the function to call when
	// the TrailingIcon is clicked. If this is nil, the trailing icon
	// will not be interactive.
	TrailingIconOnClick func(e events.Event) `json:"-" xml:"-"`

	// NoEcho is whether replace displayed characters with bullets to conceal text
	// (for example, for a password input).
	NoEcho bool

	// CursorWidth is the width of the text field cursor.
	// It should be set in Style like all other style properties.
	// By default, it is 1dp.
	CursorWidth units.Value

	// CursorColor is the color used for the text field cursor (caret).
	// It should be set in Style like all other style properties.
	// By default, it is [colors.Scheme.Primary.Base].
	CursorColor image.Image

	// PlaceholderColor is the color used for the Placeholder text.
	// It should be set in Style like all other style properties.
	// By default, it is [colors.Scheme.OnSurfaceVariant].
	PlaceholderColor image.Image

	// SelectColor is the color used for the text selection background color.
	// It should be set in Style like all other style properties.
	// By default, it is [colors.Scheme.Select.Container]
	SelectColor image.Image

	// Complete contains functions and data for text field completion.
	// It must be set using [TextField.SetCompleter].
	Complete *Complete `copier:"-" json:"-" xml:"-" set:"-"`

	// Txt is the last saved value of the text string being edited.
	Txt string `json:"-" xml:"-" set:"-"`

	// Edited is whether the text has been edited relative to the original.
	Edited bool `json:"-" xml:"-" set:"-"`

	// EditTxt is the live text string being edited, with the latest modifications.
	EditTxt []rune `copier:"-" json:"-" xml:"-" set:"-"`

	// Error is the current validation error of the text field.
	Error error `json:"-" xml:"-" set:"-"`

	// EffPos is the effective position with any leading icon space added.
	EffPos math32.Vector2 `copier:"-" json:"-" xml:"-" set:"-"`

	// EffSize is the effective size, subtracting any leading and trailing icon space.
	EffSize math32.Vector2 `copier:"-" json:"-" xml:"-" set:"-"`

	// StartPos is the starting display position in the string.
	StartPos int `copier:"-" json:"-" xml:"-" set:"-"`

	// EndPos is the ending display position in the string.
	EndPos int `copier:"-" json:"-" xml:"-" set:"-"`

	// CursorPos is the current cursor position.
	CursorPos int `copier:"-" json:"-" xml:"-" set:"-"`

	// CursorLine is the current cursor line position.
	CursorLine int `copier:"-" json:"-" xml:"-" set:"-"`

	// CharWidth is the approximate number of chars that can be
	// displayed at any time, which is computed from the font size.
	CharWidth int `copier:"-" json:"-" xml:"-" set:"-"`

	// SelectStart is the starting position of selection in the string.
	SelectStart int `copier:"-" json:"-" xml:"-" set:"-"`

	// SelectEnd is the ending position of selection in the string.
	SelectEnd int `copier:"-" json:"-" xml:"-" set:"-"`

	// SelectInit is the initial selection position (where it started).
	SelectInit int `copier:"-" json:"-" xml:"-" set:"-"`

	// SelectMode is whether to select text as the cursor moves.
	SelectMode bool `copier:"-" json:"-" xml:"-"`

	// RenderAll is the render version of entire text, for sizing.
	RenderAll paint.Text `copier:"-" json:"-" xml:"-" set:"-"`

	// RenderVis is the render version of just the visible text.
	RenderVis paint.Text `copier:"-" json:"-" xml:"-" set:"-"`

	// number of lines from last render update, for word-wrap version
	NLines int `copier:"-" json:"-" xml:"-" set:"-"`

	// FontHeight is the font height cached during styling.
	FontHeight float32 `copier:"-" json:"-" xml:"-" set:"-"`

	// BlinkOn oscillates between on and off for blinking.
	BlinkOn bool `copier:"-" json:"-" xml:"-" set:"-"`

	// CursorMu is the mutex for updating the cursor between blinker and field.
	CursorMu sync.Mutex `copier:"-" json:"-" xml:"-" view:"-" set:"-"`

	// Undos is the undo manager for the text field.
	Undos TextFieldUndos `json:"-" xml:"-" set:"-"`
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

func (tf *TextField) OnInit() {
	tf.WidgetBase.OnInit()
	tf.HandleEvents()
	tf.SetStyles()
	tf.AddContextMenu(tf.ContextMenu)
}

func (tf *TextField) OnAdd() {
	tf.WidgetBase.OnAdd()
	tf.OnClose(func(e events.Event) {
		tf.EditDone() // todo: this must be protected against something else, for race detector
	})
}

func (tf *TextField) SetStyles() {
	tf.Style(func(s *styles.Style) {
		s.SetAbilities(true, abilities.Activatable, abilities.Focusable, abilities.Hoverable, abilities.Slideable, abilities.DoubleClickable, abilities.TripleClickable)
		tf.CursorWidth.Dp(1)
		tf.SelectColor = colors.C(colors.Scheme.Select.Container)
		tf.PlaceholderColor = colors.C(colors.Scheme.OnSurfaceVariant)
		tf.CursorColor = colors.C(colors.Scheme.Primary.Base)

		s.VirtualKeyboard = styles.KeyboardSingleLine
		if !tf.IsReadOnly() {
			s.Cursor = cursors.Text
		}
		s.GrowWrap = false // note: doesn't work with Grow
		s.Grow.Set(1, 0)
		s.Min.Y.Em(1.1)
		s.Min.X.Ch(20)
		s.Max.X.Ch(40)
		s.Padding.Set(units.Dp(8), units.Dp(8))
		if tf.LeadingIcon.IsSet() {
			s.Padding.Left.Dp(12)
		}
		if tf.TrailingIcon.IsSet() {
			s.Padding.Right.Dp(12)
		}
		s.Text.Align = styles.Start
		s.Color = colors.C(colors.Scheme.OnSurface)
		switch tf.Type {
		case TextFieldFilled:
			s.Border.Style.Set(styles.BorderNone)
			s.Border.Style.Bottom = styles.BorderSolid
			s.Border.Width.Zero()
			s.Border.Color.Zero()
			s.Border.Radius = styles.BorderRadiusExtraSmallTop
			s.Background = colors.C(colors.Scheme.SurfaceContainer)

			s.MaxBorder = s.Border
			s.MaxBorder.Width.Bottom = units.Dp(2)
			s.MaxBorder.Color.Bottom = colors.C(colors.Scheme.Primary.Base)

			if !tf.IsReadOnly() && s.Is(states.Focused) {
				s.Border = s.MaxBorder
			} else {
				s.Border.Width.Bottom = units.Dp(1)
				s.Border.Color.Bottom = colors.C(colors.Scheme.OnSurfaceVariant)
			}
			if tf.Error != nil {
				s.Border.Color.Bottom = colors.C(colors.Scheme.Error.Base)
			}
		case TextFieldOutlined:
			s.Border.Style.Set(styles.BorderSolid)
			s.Border.Radius = styles.BorderRadiusExtraSmall

			s.MaxBorder = s.Border
			s.MaxBorder.Width.Set(units.Dp(2))
			s.MaxBorder.Color.Set(colors.C(colors.Scheme.Primary.Base))
			if !tf.IsReadOnly() && s.Is(states.Focused) {
				s.Border = s.MaxBorder
			} else {
				s.Border.Width.Set(units.Dp(1))
				s.Border.Color.Set(colors.C(colors.Scheme.Outline))
			}
			if tf.Error != nil {
				s.Border.Color.Set(colors.C(colors.Scheme.Error.Base))
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
			s.Background = colors.C(colors.Scheme.Select.Container)
		}
	})
	tf.StyleFinal(func(s *styles.Style) {
		tf.SetAbilities(!tf.IsReadOnly(), abilities.Focusable)
	})
	tf.OnWidgetAdded(func(w Widget) {
		switch w.PathFrom(tf) {
		case "parts":
			w.Style(func(s *styles.Style) {
				s.Align.Content = styles.Center
				s.Align.Items = styles.Center
				s.Text.AlignV = styles.Center
				s.Gap.Zero()
			})
		case "parts/lead-icon":
			lead := w.(*Button)
			lead.Type = ButtonAction
			lead.Style(func(s *styles.Style) {
				s.Padding.Zero()
				s.Color = colors.C(colors.Scheme.OnSurfaceVariant)
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
			lead.OnClick(func(e events.Event) {
				if tf.LeadingIconOnClick != nil {
					tf.LeadingIconOnClick(e)
				}
			})
		case "parts/trail-icon":
			trail := w.(*Button)
			trail.Type = ButtonAction
			trail.Style(func(s *styles.Style) {
				s.Padding.Zero()
				s.Color = colors.C(colors.Scheme.OnSurfaceVariant)
				if tf.Error != nil {
					s.Color = colors.C(colors.Scheme.Error.Base)
				}
				s.Margin.SetLeft(units.Dp(8))
				if tf.TrailingIconOnClick == nil || tf.Error != nil {
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
			trail.OnClick(func(e events.Event) {
				if tf.TrailingIconOnClick != nil {
					tf.TrailingIconOnClick(e)
				}
			})
		case "parts/error":
			w.Style(func(s *styles.Style) {
				s.Color = colors.C(colors.Scheme.Error.Base)
			})
		}
	})
}

func (tf *TextField) Destroy() {
	tf.StopCursor()
	tf.WidgetBase.Destroy()
}

// Text returns the current text -- applies any unapplied changes first, and
// sends a signal if so -- this is the end-user method to get the current
// value of the field.
func (tf *TextField) Text() string {
	tf.EditDone()
	return tf.Txt
}

// SetText sets the text to be edited and reverts any current edit
// to reflect this new text.
func (tf *TextField) SetText(txt string) *TextField {
	if tf.Txt == txt && !tf.Edited {
		return tf
	}
	tf.Txt = txt
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
	}).Style(func(s *styles.Style) {
		s.VirtualKeyboard = styles.KeyboardPassword
	})
	return tf
}

// EditDone completes editing and copies the active edited text to the text --
// called when the return key is pressed or goes out of focus
func (tf *TextField) EditDone() {
	if tf.Edited {
		tf.Edited = false
		tf.Txt = string(tf.EditTxt)
		tf.SendChange()
		// widget can be killed after SendChange
		if tf.This() == nil {
			return
		}
	}
	tf.ClearSelected()
	tf.ClearCursor()
}

// Revert aborts editing and reverts to last saved text
func (tf *TextField) Revert() {
	tf.EditTxt = []rune(tf.Txt)
	tf.Edited = false
	tf.StartPos = 0
	tf.EndPos = tf.CharWidth
	tf.SelectReset()
	tf.NeedsRender()
}

// Clear clears any existing text
func (tf *TextField) Clear() {
	tf.Edited = true
	tf.EditTxt = tf.EditTxt[:0]
	tf.StartPos = 0
	tf.EndPos = 0
	tf.SelectReset()
	tf.SetFocusEvent() // this is essential for ensuring that the clear applies after focus is lost..
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
		if tf.Error == nil {
			return
		}
		tf.Error = nil
		tf.TrailingIconButton().SetIcon(tf.TrailingIcon).Update()
		return
	}
	tf.Error = err
	if tf.TrailingIconButton() == nil {
		tf.SetTrailingIcon(icons.Blank).Update()
	}
	tf.TrailingIconButton().SetIcon(icons.Error).Update()
	// show the error tooltip immediately
	tf.Send(events.LongHoverStart)
}

func (tf *TextField) WidgetTooltip(pos image.Point) (string, image.Point) {
	if tf.Error == nil {
		return tf.Tooltip, tf.DefaultTooltipPos()
	}
	return tf.Error.Error(), tf.DefaultTooltipPos()
}

//////////////////////////////////////////////////////////////////////////////////////////
//  Cursor Navigation

// CursorForward moves the cursor forward
func (tf *TextField) CursorForward(steps int) {
	tf.CursorPos += steps
	if tf.CursorPos > len(tf.EditTxt) {
		tf.CursorPos = len(tf.EditTxt)
	}
	if tf.CursorPos > tf.EndPos {
		inc := tf.CursorPos - tf.EndPos
		tf.EndPos += inc
	}
	tf.CursorLine, _, _ = tf.RenderAll.RuneSpanPos(tf.CursorPos)
	if tf.SelectMode {
		tf.SelectRegUpdate(tf.CursorPos)
	}
	tf.NeedsRender()
}

// CursorForwardWord moves the cursor forward by words
func (tf *TextField) CursorForwardWord(steps int) {
	for i := 0; i < steps; i++ {
		sz := len(tf.EditTxt)
		if sz > 0 && tf.CursorPos < sz {
			ch := tf.CursorPos
			var done = false
			for ch < sz && !done { // if on a wb, go past
				r1 := tf.EditTxt[ch]
				r2 := rune(-1)
				if ch < sz-1 {
					r2 = tf.EditTxt[ch+1]
				}
				if IsWordBreak(r1, r2) {
					ch++
				} else {
					done = true
				}
			}
			done = false
			for ch < sz && !done {
				r1 := tf.EditTxt[ch]
				r2 := rune(-1)
				if ch < sz-1 {
					r2 = tf.EditTxt[ch+1]
				}
				if !IsWordBreak(r1, r2) {
					ch++
				} else {
					done = true
				}
			}
			tf.CursorPos = ch
		} else {
			tf.CursorPos = sz
		}
	}
	if tf.CursorPos > len(tf.EditTxt) {
		tf.CursorPos = len(tf.EditTxt)
	}
	if tf.CursorPos > tf.EndPos {
		inc := tf.CursorPos - tf.EndPos
		tf.EndPos += inc
	}
	tf.CursorLine, _, _ = tf.RenderAll.RuneSpanPos(tf.CursorPos)
	if tf.SelectMode {
		tf.SelectRegUpdate(tf.CursorPos)
	}
	tf.NeedsRender()
}

// CursorBackward moves the cursor backward
func (tf *TextField) CursorBackward(steps int) {
	tf.CursorPos -= steps
	if tf.CursorPos < 0 {
		tf.CursorPos = 0
	}
	if tf.CursorPos <= tf.StartPos {
		dec := min(tf.StartPos, 8)
		tf.StartPos -= dec
	}
	tf.CursorLine, _, _ = tf.RenderAll.RuneSpanPos(tf.CursorPos)
	if tf.SelectMode {
		tf.SelectRegUpdate(tf.CursorPos)
	}
	tf.NeedsRender()
}

// CursorBackwardWord moves the cursor backward by words
func (tf *TextField) CursorBackwardWord(steps int) {
	for i := 0; i < steps; i++ {
		sz := len(tf.EditTxt)
		if sz > 0 && tf.CursorPos > 0 {
			ch := min(tf.CursorPos, sz-1)
			var done = false
			for ch < sz && !done { // if on a wb, go past
				r1 := tf.EditTxt[ch]
				r2 := rune(-1)
				if ch > 0 {
					r2 = tf.EditTxt[ch-1]
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
				r1 := tf.EditTxt[ch]
				r2 := rune(-1)
				if ch > 0 {
					r2 = tf.EditTxt[ch-1]
				}
				if !IsWordBreak(r1, r2) {
					ch--
				} else {
					done = true
				}
			}
			tf.CursorPos = ch
		} else {
			tf.CursorPos = 0
		}
	}
	if tf.CursorPos < 0 {
		tf.CursorPos = 0
	}
	if tf.CursorPos <= tf.StartPos {
		dec := min(tf.StartPos, 8)
		tf.StartPos -= dec
	}
	tf.CursorLine, _, _ = tf.RenderAll.RuneSpanPos(tf.CursorPos)
	if tf.SelectMode {
		tf.SelectRegUpdate(tf.CursorPos)
	}
	tf.NeedsRender()
}

// CursorDown moves the cursor down
func (tf *TextField) CursorDown(steps int) {
	if tf.NLines <= 1 {
		return
	}
	if tf.CursorLine >= tf.NLines-1 {
		return
	}

	_, ri, _ := tf.RenderAll.RuneSpanPos(tf.CursorPos)
	tf.CursorLine = min(tf.CursorLine+steps, tf.NLines-1)
	tf.CursorPos, _ = tf.RenderAll.SpanPosToRuneIndex(tf.CursorLine, ri)
	if tf.SelectMode {
		tf.SelectRegUpdate(tf.CursorPos)
	}
	tf.NeedsRender()
}

// CursorUp moves the cursor up
func (tf *TextField) CursorUp(steps int) {
	if tf.NLines <= 1 {
		return
	}
	if tf.CursorLine <= 0 {
		return
	}

	_, ri, _ := tf.RenderAll.RuneSpanPos(tf.CursorPos)
	tf.CursorLine = max(tf.CursorLine-steps, 0)
	tf.CursorPos, _ = tf.RenderAll.SpanPosToRuneIndex(tf.CursorLine, ri)
	if tf.SelectMode {
		tf.SelectRegUpdate(tf.CursorPos)
	}
	tf.NeedsRender()
}

// CursorStart moves the cursor to the start of the text, updating selection
// if select mode is active
func (tf *TextField) CursorStart() {
	tf.CursorPos = 0
	tf.StartPos = 0
	tf.EndPos = min(len(tf.EditTxt), tf.StartPos+tf.CharWidth)
	if tf.SelectMode {
		tf.SelectRegUpdate(tf.CursorPos)
	}
	tf.NeedsRender()
}

// CursorEnd moves the cursor to the end of the text
func (tf *TextField) CursorEnd() {
	ed := len(tf.EditTxt)
	tf.CursorPos = ed
	tf.EndPos = len(tf.EditTxt) // try -- display will adjust
	tf.StartPos = max(0, tf.EndPos-tf.CharWidth)
	if tf.SelectMode {
		tf.SelectRegUpdate(tf.CursorPos)
	}
	tf.NeedsRender()
}

// CursorBackspace deletes character(s) immediately before cursor
func (tf *TextField) CursorBackspace(steps int) {
	if tf.HasSelection() {
		tf.DeleteSelection()
		return
	}
	if tf.CursorPos < steps {
		steps = tf.CursorPos
	}
	if steps <= 0 {
		return
	}
	tf.Edited = true
	tf.EditTxt = append(tf.EditTxt[:tf.CursorPos-steps], tf.EditTxt[tf.CursorPos:]...)
	tf.CursorBackward(steps)
	tf.NeedsRender()
}

// CursorDelete deletes character(s) immediately after the cursor
func (tf *TextField) CursorDelete(steps int) {
	if tf.HasSelection() {
		tf.DeleteSelection()
		return
	}
	if tf.CursorPos+steps > len(tf.EditTxt) {
		steps = len(tf.EditTxt) - tf.CursorPos
	}
	if steps <= 0 {
		return
	}
	tf.Edited = true
	tf.EditTxt = append(tf.EditTxt[:tf.CursorPos], tf.EditTxt[tf.CursorPos+steps:]...)
	tf.NeedsRender()
}

// CursorBackspaceWord deletes words(s) immediately before cursor
func (tf *TextField) CursorBackspaceWord(steps int) {
	if tf.HasSelection() {
		tf.DeleteSelection()
		return
	}
	org := tf.CursorPos
	tf.CursorBackwardWord(steps)
	tf.Edited = true
	tf.EditTxt = append(tf.EditTxt[:tf.CursorPos], tf.EditTxt[org:]...)
	tf.NeedsRender()
}

// CursorDeleteWord deletes word(s) immediately after the cursor
func (tf *TextField) CursorDeleteWord(steps int) {
	if tf.HasSelection() {
		tf.DeleteSelection()
		return
	}
	// note: no update b/c signal from buf will drive update
	org := tf.CursorPos
	tf.CursorForwardWord(steps)
	tf.Edited = true
	tf.EditTxt = append(tf.EditTxt[:tf.CursorPos], tf.EditTxt[org:]...)
	tf.NeedsRender()
}

// CursorKill deletes text from cursor to end of text
func (tf *TextField) CursorKill() {
	steps := len(tf.EditTxt) - tf.CursorPos
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
	return tf.SelectStart < tf.SelectEnd
}

// Selection returns the currently selected text
func (tf *TextField) Selection() string {
	if tf.HasSelection() {
		return string(tf.EditTxt[tf.SelectStart:tf.SelectEnd])
	}
	return ""
}

// SelectModeToggle toggles the SelectMode, updating selection with cursor movement
func (tf *TextField) SelectModeToggle() {
	if tf.SelectMode {
		tf.SelectMode = false
	} else {
		tf.SelectMode = true
		tf.SelectInit = tf.CursorPos
		tf.SelectStart = tf.CursorPos
		tf.SelectEnd = tf.SelectStart
	}
}

// ShiftSelect sets the selection start if the shift key is down but wasn't previously.
// If the shift key has been released, the selection info is cleared.
func (tf *TextField) ShiftSelect(e events.Event) {
	hasShift := e.HasAnyModifier(key.Shift)
	if hasShift && !tf.SelectMode {
		tf.SelectModeToggle()
	}
	if !hasShift && tf.SelectMode {
		tf.SelectReset()
	}
}

// SelectRegUpdate updates current select region based on given cursor position
// relative to SelectStart position
func (tf *TextField) SelectRegUpdate(pos int) {
	if pos < tf.SelectInit {
		tf.SelectStart = pos
		tf.SelectEnd = tf.SelectInit
	} else {
		tf.SelectStart = tf.SelectInit
		tf.SelectEnd = pos
	}
	tf.SelectUpdate()
}

// SelectAll selects all the text
func (tf *TextField) SelectAll() {
	tf.SelectStart = 0
	tf.SelectInit = 0
	tf.SelectEnd = len(tf.EditTxt)
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
	sz := len(tf.EditTxt)
	if sz <= 3 {
		tf.SelectAll()
		return
	}
	tf.SelectStart = tf.CursorPos
	if tf.SelectStart >= sz {
		tf.SelectStart = sz - 2
	}
	if !tf.IsWordBreak(tf.EditTxt[tf.SelectStart]) {
		for tf.SelectStart > 0 {
			if tf.IsWordBreak(tf.EditTxt[tf.SelectStart-1]) {
				break
			}
			tf.SelectStart--
		}
		tf.SelectEnd = tf.CursorPos + 1
		for tf.SelectEnd < sz {
			if tf.IsWordBreak(tf.EditTxt[tf.SelectEnd]) {
				break
			}
			tf.SelectEnd++
		}
	} else { // keep the space start -- go to next space..
		tf.SelectEnd = tf.CursorPos + 1
		for tf.SelectEnd < sz {
			if !tf.IsWordBreak(tf.EditTxt[tf.SelectEnd]) {
				break
			}
			tf.SelectEnd++
		}
		for tf.SelectEnd < sz { // include all trailing spaces
			if tf.IsWordBreak(tf.EditTxt[tf.SelectEnd]) {
				break
			}
			tf.SelectEnd++
		}
	}
	tf.SelectInit = tf.SelectStart
	if TheApp.SystemPlatform().IsMobile() {
		tf.Send(events.ContextMenu)
	}
	tf.NeedsRender()
}

// SelectReset resets the selection
func (tf *TextField) SelectReset() {
	tf.SelectMode = false
	if tf.SelectStart == 0 && tf.SelectEnd == 0 {
		return
	}
	tf.SelectStart = 0
	tf.SelectEnd = 0
	tf.NeedsRender()
}

// SelectUpdate updates the select region after any change to the text, to keep it in range
func (tf *TextField) SelectUpdate() {
	if tf.SelectStart < tf.SelectEnd {
		ed := len(tf.EditTxt)
		if tf.SelectStart < 0 {
			tf.SelectStart = 0
		}
		if tf.SelectEnd > ed {
			tf.SelectEnd = ed
		}
	} else {
		tf.SelectReset()
	}
}

// Cut cuts any selected text and adds it to the clipboard
func (tf *TextField) Cut() {
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
	tf.Edited = true
	tf.EditTxt = append(tf.EditTxt[:tf.SelectStart], tf.EditTxt[tf.SelectEnd:]...)
	if tf.CursorPos > tf.SelectStart {
		if tf.CursorPos < tf.SelectEnd {
			tf.CursorPos = tf.SelectStart
		} else {
			tf.CursorPos -= tf.SelectEnd - tf.SelectStart
		}
	}
	tf.SelectReset()
	tf.NeedsRender()
	return cut
}

// Copy copies any selected text to the clipboard.
// Satisfies Clipper interface -- can be extended in subtypes.
// optionally resetting the current selection
func (tf *TextField) Copy(reset bool) {
	if tf.NoEcho {
		return
	}
	tf.SelectUpdate()
	if !tf.HasSelection() {
		return
	}

	md := mimedata.NewText(tf.Text())
	tf.Clipboard().Write(md)
	if reset {
		tf.SelectReset()
	}
}

// Paste inserts text from the clipboard at current cursor position -- if
// cursor is within a current selection, that selection is replaced.
// Satisfies Clipper interface -- can be extended in subtypes.
func (tf *TextField) Paste() {
	data := tf.Clipboard().Read([]string{mimedata.TextPlain})
	if data != nil {
		if tf.CursorPos >= tf.SelectStart && tf.CursorPos < tf.SelectEnd {
			tf.DeleteSelection()
		}
		tf.InsertAtCursor(data.Text(mimedata.TextPlain))
	}
}

// InsertAtCursor inserts given text at current cursor position
func (tf *TextField) InsertAtCursor(str string) {
	if tf.HasSelection() {
		tf.Cut()
	}
	tf.Edited = true
	rs := []rune(str)
	rsl := len(rs)
	nt := append(tf.EditTxt, rs...)                // first append to end
	copy(nt[tf.CursorPos+rsl:], nt[tf.CursorPos:]) // move stuff to end
	copy(nt[tf.CursorPos:], rs)                    // copy into position
	tf.EditTxt = nt
	tf.EndPos += rsl
	tf.CursorForward(rsl)
	tf.NeedsRender()
}

func (tf *TextField) ContextMenu(m *Scene) {
	NewButton(m).SetText("Copy").SetIcon(icons.ContentCopy).SetKey(keymap.Copy).SetState(tf.NoEcho || !tf.HasSelection(), states.Disabled).
		OnClick(func(e events.Event) {
			tf.Copy(true)
		})
	if !tf.IsReadOnly() {
		NewButton(m).SetText("Cut").SetIcon(icons.ContentCut).SetKey(keymap.Cut).SetState(tf.NoEcho || !tf.HasSelection(), states.Disabled).
			OnClick(func(e events.Event) {
				tf.Cut()
			})
		pbt := NewButton(m).SetText("Paste").SetIcon(icons.ContentPaste).SetKey(keymap.Paste).
			OnClick(func(e events.Event) {
				tf.Paste()
			})
		cb := tf.Scene.Events.Clipboard()
		if cb != nil {
			pbt.SetState(cb.IsEmpty(), states.Disabled)
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
	tf.Undos.SaveUndo(tf.EditTxt, tf.CursorPos)
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
	r := tf.Undos.Undo(tf.EditTxt, tf.CursorPos)
	if r != nil {
		tf.EditTxt = r.Text
		tf.CursorPos = r.CursorPos
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
	r := tf.Undos.Redo()
	if r != nil {
		tf.EditTxt = r.Text
		tf.CursorPos = r.CursorPos
		tf.NeedsRender()
	}
}

///////////////////////////////////////////////////////////////////////////////
//    Complete

// SetCompleter sets completion functions so that completions will
// automatically be offered as the user types
func (tf *TextField) SetCompleter(data any, matchFun complete.MatchFunc, editFun complete.EditFunc) {
	if matchFun == nil || editFun == nil {
		tf.Complete = nil
		return
	}
	tf.Complete = NewComplete().SetContext(data).SetMatchFunc(matchFun).SetEditFunc(editFun)
	tf.Complete.OnSelect(func(e events.Event) {
		tf.CompleteText(tf.Complete.Completion)
	})
}

// OfferComplete pops up a menu of possible completions
func (tf *TextField) OfferComplete() {
	if tf.Complete == nil {
		return
	}
	s := string(tf.EditTxt[0:tf.CursorPos])
	cpos := tf.CharRenderPos(tf.CursorPos, true).ToPoint()
	cpos.X += 5
	cpos.Y = tf.Geom.TotalBBox.Max.Y
	tf.Complete.SrcLn = 0
	tf.Complete.SrcCh = tf.CursorPos
	tf.Complete.Show(tf, cpos, s)
}

// CancelComplete cancels any pending completion -- call this when new events
// have moved beyond any prior completion scenario
func (tf *TextField) CancelComplete() {
	if tf.Complete == nil {
		return
	}
	tf.Complete.Cancel()
}

// CompleteText edits the text field using the string chosen from the completion menu
func (tf *TextField) CompleteText(s string) {
	txt := string(tf.EditTxt) // Reminder: do NOT call tf.Text() in an active editing context!!!
	c := tf.Complete.GetCompletion(s)
	ed := tf.Complete.EditFunc(tf.Complete.Context, txt, tf.CursorPos, c, tf.Complete.Seed)
	st := tf.CursorPos - len(tf.Complete.Seed)
	tf.CursorPos = st
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
	if idx <= 0 || len(tf.RenderAll.Spans) == 0 {
		return math32.Vector2{}
	}
	pos, _, _, _ := tf.RenderAll.RuneRelPos(idx)
	pos.Y -= tf.RenderAll.Spans[0].RelPos.Y
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
	pos := tf.EffPos
	if wincoords {
		sc := tf.Scene
		pos = pos.Add(math32.Vector2FromPoint(sc.SceneGeom.Pos))
	}
	cpos := tf.RelCharPos(tf.StartPos, charidx)
	return pos.Add(cpos)
}

// ScrollLayoutToCursor scrolls any scrolling layout above us so that the cursor is in view
func (tf *TextField) ScrollLayoutToCursor() bool {
	ly := tf.ParentScrollLayout()
	if ly == nil {
		return false
	}
	cpos := tf.CharRenderPos(tf.CursorPos, false).ToPointFloor()
	bbsz := image.Point{int(math32.Ceil(tf.CursorWidth.Dots)), int(math32.Ceil(tf.FontHeight))}
	bbox := image.Rectangle{Min: cpos, Max: cpos.Add(bbsz)}
	return ly.ScrollToBox(bbox)
}

var (
	// TextFieldBlinker manages cursor blinking
	TextFieldBlinker = Blinker{}

	// TextFieldSpriteName is the name of the window sprite used for the cursor
	TextFieldSpriteName = "core.TextField.Cursor"
)

func init() {
	TheApp.AddQuitCleanFunc(TextFieldBlinker.QuitClean)
}

// StartCursor starts the cursor blinking and renders it
func (tf *TextField) StartCursor() {
	if tf == nil || tf.This() == nil {
		return
	}
	if !tf.This().(Widget).IsVisible() {
		return
	}
	tf.BlinkOn = true
	tf.RenderCursor(true)
	if SystemSettings.CursorBlinkTime == 0 {
		return
	}
	TextFieldBlinker.Blink(SystemSettings.CursorBlinkTime, func() {
		if !tf.StateIs(states.Focused) || !tf.IsVisible() {
			tf.BlinkOn = false
			tf.RenderCursor(false)
			TextFieldBlinker.Widget = nil
		} else {
			tf.BlinkOn = !tf.BlinkOn
			tf.RenderCursor(tf.BlinkOn)
		}
	})
	TextFieldBlinker.SetWidget(tf.This().(Widget))
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
	if tf == nil || tf.This() == nil {
		return
	}
	TextFieldBlinker.ResetWidget(tf.This().(Widget))
}

// RenderCursor renders the cursor on or off, as a sprite that is either on or off
func (tf *TextField) RenderCursor(on bool) {
	if tf == nil || tf.This() == nil {
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
		spnm := fmt.Sprintf("%v-%v", TextFieldSpriteName, tf.FontHeight)
		ms.Sprites.InactivateSprite(spnm)
		return
	}
	if !tf.This().(Widget).IsVisible() {
		return
	}

	tf.CursorMu.Lock()
	defer tf.CursorMu.Unlock()

	sp := tf.CursorSprite(on)
	if sp == nil {
		return
	}
	sp.Geom.Pos = tf.CharRenderPos(tf.CursorPos, true).ToPointFloor()
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
	spnm := fmt.Sprintf("%v-%v", TextFieldSpriteName, tf.FontHeight)
	sp, ok := ms.Sprites.SpriteByName(spnm)
	// TODO: figure out how to update caret color on color scheme change
	if !ok {
		bbsz := image.Point{int(math32.Ceil(tf.CursorWidth.Dots)), int(math32.Ceil(tf.FontHeight))}
		if bbsz.X < 2 { // at least 2
			bbsz.X = 2
		}
		sp = NewSprite(spnm, bbsz, image.Point{})
		sp.On = on
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
	effst := max(tf.StartPos, tf.SelectStart)
	if effst >= tf.EndPos {
		return
	}
	effed := min(tf.EndPos, tf.SelectEnd)
	if effed < tf.StartPos {
		return
	}
	if effed <= effst {
		return
	}

	spos := tf.CharRenderPos(effst, false)

	pc := &tf.Scene.PaintContext
	tsz := tf.RelCharPos(effst, effed)
	if !tf.HasWordWrap() || tsz.Y == 0 {
		pc.FillBox(spos, math32.Vec2(tsz.X, tf.FontHeight), tf.SelectColor)
		return
	}
	ex := float32(tf.Geom.ContentBBox.Max.X)
	sx := float32(tf.Geom.ContentBBox.Min.X)
	ssi, _, _ := tf.RenderAll.RuneSpanPos(effst)
	esi, _, _ := tf.RenderAll.RuneSpanPos(effed)
	ep := tf.CharRenderPos(effed, false)

	pc.FillBox(spos, math32.Vec2(ex-spos.X, tf.FontHeight), tf.SelectColor)

	spos.X = sx
	spos.Y += tf.RenderAll.Spans[ssi+1].RelPos.Y - tf.RenderAll.Spans[ssi].RelPos.Y
	for si := ssi + 1; si <= esi; si++ {
		if si < esi {
			pc.FillBox(spos, math32.Vec2(ex-spos.X, tf.FontHeight), tf.SelectColor)
		} else {
			pc.FillBox(spos, math32.Vec2(ep.X-spos.X, tf.FontHeight), tf.SelectColor)
		}
		spos.Y += tf.RenderAll.Spans[si].RelPos.Y - tf.RenderAll.Spans[si-1].RelPos.Y
	}
}

// AutoScroll scrolls the starting position to keep the cursor visible
func (tf *TextField) AutoScroll() {
	sz := &tf.Geom.Size
	icsz := tf.IconsSize()
	availSz := sz.Actual.Content.Sub(icsz)
	tf.ConfigTextSize(availSz)
	n := len(tf.EditTxt)
	tf.CursorPos = math32.ClampInt(tf.CursorPos, 0, n)

	if tf.HasWordWrap() { // does not scroll
		tf.StartPos = 0
		tf.EndPos = n
		if len(tf.RenderAll.Spans) != tf.NLines {
			tf.NeedsLayout()
		}
		return
	}
	st := &tf.Styles

	if n == 0 || tf.Geom.Size.Actual.Content.X <= 0 {
		tf.CursorPos = 0
		tf.EndPos = 0
		tf.StartPos = 0
		return
	}
	maxw := tf.EffSize.X
	if maxw < 0 {
		return
	}
	tf.CharWidth = int(maxw / st.UnitContext.Dots(units.UnitCh)) // rough guess in chars
	if tf.CharWidth < 1 {
		tf.CharWidth = 1
	}

	// first rationalize all the values
	if tf.EndPos == 0 || tf.EndPos > n { // not init
		tf.EndPos = n
	}
	if tf.StartPos >= tf.EndPos {
		tf.StartPos = max(0, tf.EndPos-tf.CharWidth)
	}

	inc := int(math32.Ceil(.1 * float32(tf.CharWidth)))
	inc = max(4, inc)

	// keep cursor in view with buffer
	startIsAnchor := true
	if tf.CursorPos < (tf.StartPos + inc) {
		tf.StartPos -= inc
		tf.StartPos = max(tf.StartPos, 0)
		tf.EndPos = tf.StartPos + tf.CharWidth
		tf.EndPos = min(n, tf.EndPos)
	} else if tf.CursorPos > (tf.EndPos - inc) {
		tf.EndPos += inc
		tf.EndPos = min(tf.EndPos, n)
		tf.StartPos = tf.EndPos - tf.CharWidth
		tf.StartPos = max(0, tf.StartPos)
		startIsAnchor = false
	}
	if tf.EndPos < tf.StartPos {
		return
	}

	if startIsAnchor {
		gotWidth := false
		spos := tf.CharPos(tf.StartPos).X
		for {
			w := tf.CharPos(tf.EndPos).X - spos
			if w < maxw {
				if tf.EndPos == n {
					break
				}
				nw := tf.CharPos(tf.EndPos+1).X - spos
				if nw >= maxw {
					gotWidth = true
					break
				}
				tf.EndPos++
			} else {
				tf.EndPos--
			}
		}
		if gotWidth || tf.StartPos == 0 {
			return
		}
		// otherwise, try getting some more chars by moving up start..
	}

	// end is now anchor
	epos := tf.CharPos(tf.EndPos).X
	for {
		w := epos - tf.CharPos(tf.StartPos).X
		if w < maxw {
			if tf.StartPos == 0 {
				break
			}
			nw := epos - tf.CharPos(tf.StartPos-1).X
			if nw >= maxw {
				break
			}
			tf.StartPos--
		} else {
			tf.StartPos++
		}
	}
}

// PixelToCursor finds the cursor position that corresponds to the given pixel location
func (tf *TextField) PixelToCursor(pt image.Point) int {
	ptf := math32.Vector2FromPoint(pt)
	rpt := ptf.Sub(tf.EffPos)
	if rpt.X <= 0 || rpt.Y < 0 {
		return tf.StartPos
	}
	n := len(tf.EditTxt)
	if tf.HasWordWrap() {
		si, ri, ok := tf.RenderAll.PosToRune(rpt)
		if ok {
			ix, _ := tf.RenderAll.SpanPosToRuneIndex(si, ri)
			ix = min(ix, n)
			return ix
		}
		return tf.StartPos
	}
	pr := tf.PointToRelPos(pt)

	px := float32(pr.X)
	st := &tf.Styles
	c := tf.StartPos + int(float64(px/st.UnitContext.Dots(units.UnitCh)))
	c = min(c, n)

	w := tf.RelCharPos(tf.StartPos, c).X
	if w > px {
		for w > px {
			c--
			if c <= tf.StartPos {
				c = tf.StartPos
				break
			}
			w = tf.RelCharPos(tf.StartPos, c).X
		}
	} else if w < px {
		for c < tf.EndPos {
			wn := tf.RelCharPos(tf.StartPos, c+1).X
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
	oldPos := tf.CursorPos
	tf.CursorPos = tf.PixelToCursor(pt)
	if tf.SelectMode || selMode != events.SelectOne {
		if !tf.SelectMode && selMode != events.SelectOne {
			tf.SelectStart = oldPos
			tf.SelectMode = true
		}
		if !tf.StateIs(states.Sliding) && selMode == events.SelectOne { // && tf.CursorPos >= tf.SelectStart && tf.CursorPos < tf.SelectEnd {
			tf.SelectReset()
		} else {
			tf.SelectRegUpdate(tf.CursorPos)
		}
		tf.SelectUpdate()
	} else if tf.HasSelection() {
		tf.SelectReset()
	}
	tf.NeedsRender()
}

///////////////////////////////////////////////////////////////////////////////
//    Event handling

func (tf *TextField) HandleEvents() {
	tf.HandleKeyEvents()
	tf.HandleSelectToggle()
	tf.OnFirst(events.Change, func(e events.Event) {
		tf.Validate()
		if tf.Error != nil {
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
		if !tf.SelectMode {
			tf.SelectModeToggle()
		}
		tf.SetCursorFromPixel(e.Pos(), events.SelectOne)
	})
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
			if tf.NLines > 1 {
				e.SetHandled()
				tf.ShiftSelect(e)
				tf.CursorDown(1)
			}
		case keymap.MoveUp:
			if tf.NLines > 1 {
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
			tf.Copy(true) // reset
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

func (tf *TextField) Config() {
	config := tree.Config{}

	tf.EditTxt = []rune(tf.Txt)
	tf.Edited = false

	lii, tii := -1, -1
	if !tf.IsReadOnly() {
		if tf.LeadingIcon.IsSet() {
			lii = len(config)
			config.Add(ButtonType, "lead-icon")
		}
		if tf.TrailingIcon.IsSet() {
			config.Add(StretchType, "trail-icon-str")
			tii = len(config)
			config.Add(ButtonType, "trail-icon")
		}
	}
	tf.ConfigParts(config, func() {
		if lii >= 0 {
			li := tf.Parts.Child(lii).(*Button)
			li.SetIcon(tf.LeadingIcon)
		}
		if tii >= 0 {
			ti := tf.Parts.Child(tii).(*Button)
			ti.SetIcon(tf.TrailingIcon)
		}
	})
}

////////////////////////////////////////////////////
//  Widget Interface

func (tf *TextField) ApplyStyle() {
	tf.ApplyStyleWidget()
	tf.CursorWidth.ToDots(&tf.Styles.UnitContext)
}

func (tf *TextField) ConfigTextSize(sz math32.Vector2) math32.Vector2 {
	st := &tf.Styles
	txs := &st.Text
	fs := st.FontRender()
	st.Font = paint.OpenFont(fs, &st.UnitContext)
	txt := tf.EditTxt
	if tf.NoEcho {
		txt = ConcealDots(len(tf.EditTxt))
	}
	align, alignV := txs.Align, txs.AlignV
	txs.Align, txs.AlignV = styles.Start, styles.Start // only works with this
	tf.RenderAll.SetRunes(txt, fs, &st.UnitContext, txs, true, 0, 0)
	tf.RenderAll.Layout(txs, fs, &st.UnitContext, sz)
	txs.Align, txs.AlignV = align, alignV
	rsz := tf.RenderAll.BBox.Size().Ceil()
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
	tf.WidgetBase.SizeUp() // sets Actual size based on styles
	tmptxt := tf.EditTxt
	if len(tf.Txt) == 0 && len(tf.Placeholder) > 0 {
		tf.EditTxt = []rune(tf.Placeholder)
	} else {
		tf.EditTxt = []rune(tf.Txt)
	}
	tf.StartPos = 0
	tf.EndPos = len(tf.EditTxt)

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
	sz.FitSizeMax(&sz.Actual.Content, rsz)
	sz.SetTotalFromContent(&sz.Actual)
	tf.FontHeight = tf.Styles.Font.Face.Metrics.Height
	tf.EditTxt = tmptxt
	if DebugSettings.LayoutTrace {
		fmt.Println(tf, "TextField SizeUp:", rsz, "Actual:", sz.Actual.Content)
	}
}

func (tf *TextField) SizeDown(iter int) bool {
	if !tf.HasWordWrap() {
		return tf.SizeDownParts(iter)
	}
	sz := &tf.Geom.Size
	pgrow, _ := tf.GrowToAllocSize(sz.Actual.Content, sz.Alloc.Content) // key to grow
	icsz := tf.IconsSize()
	prevContent := sz.Actual.Content
	availSz := pgrow.Sub(icsz)
	rsz := tf.ConfigTextSize(availSz)
	rsz.SetAdd(icsz)
	// start over so we don't reflect hysteresis of prior guess
	sz.SetInitContentMin(tf.Styles.Min.Dots().Ceil())
	sz.FitSizeMax(&sz.Actual.Content, rsz)
	sz.SetTotalFromContent(&sz.Actual)
	chg := prevContent != sz.Actual.Content
	if chg {
		if DebugSettings.LayoutTrace {
			fmt.Println(tf, "TextField Size Changed:", sz.Actual.Content, "was:", prevContent)
		}
	}
	sdp := tf.SizeDownParts(iter)
	return chg || sdp
}

func (tf *TextField) ScenePos() {
	tf.WidgetBase.ScenePos()
	tf.SetEffPosAndSize()
}

// LeadingIconButton returns the [LeadingIcon] [Button] if present.
func (tf *TextField) LeadingIconButton() *Button {
	if tf.Parts == nil {
		return nil
	}
	bi := tf.Parts.ChildByName("lead-icon", 0)
	if bi == nil {
		return nil
	}
	return bi.(*Button)
}

// TrailingIconButton returns the [TrailingIcon] [Button] if present.
func (tf *TextField) TrailingIconButton() *Button {
	if tf.Parts == nil {
		return nil
	}
	bi := tf.Parts.ChildByName("trail-icon", 1)
	if bi == nil {
		return nil
	}
	return bi.(*Button)
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
	tf.NLines = len(tf.RenderAll.Spans)
	if tf.NLines <= 1 {
		pos.Y += 0.5 * (sz.Y - tf.FontHeight) // center
	}
	tf.EffSize = sz.Ceil()
	tf.EffPos = pos.Ceil()
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
	if tf.StartPos < 0 || tf.EndPos > len(tf.EditTxt) {
		return
	}
	cur := tf.EditTxt[tf.StartPos:tf.EndPos]
	tf.RenderSelect()
	pos := tf.EffPos
	prevColor := st.Color
	if len(tf.EditTxt) == 0 && len(tf.Placeholder) > 0 {
		st.Color = tf.PlaceholderColor
		fs = st.FontRender() // need to update
		cur = []rune(tf.Placeholder)
	} else if tf.NoEcho {
		cur = ConcealDots(len(cur))
	}
	sz := &tf.Geom.Size
	icsz := tf.IconsSize()
	availSz := sz.Actual.Content.Sub(icsz)
	tf.RenderVis.SetRunes(cur, fs, &st.UnitContext, &st.Text, true, 0, 0)
	tf.RenderVis.Layout(txs, fs, &st.UnitContext, availSz)
	tf.RenderVis.Render(pc, pos)
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
