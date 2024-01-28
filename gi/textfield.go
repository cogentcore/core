// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"fmt"
	"image"
	"image/color"
	"strings"
	"sync"
	"unicode"

	"cogentcore.org/core/abilities"
	"cogentcore.org/core/colors"
	"cogentcore.org/core/cursors"
	"cogentcore.org/core/events"
	"cogentcore.org/core/events/key"
	"cogentcore.org/core/fi"
	"cogentcore.org/core/goosi"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/keyfun"
	"cogentcore.org/core/ki"
	"cogentcore.org/core/mat32"
	"cogentcore.org/core/mimedata"
	"cogentcore.org/core/paint"
	"cogentcore.org/core/pi/complete"
	"cogentcore.org/core/pi/lex"
	"cogentcore.org/core/states"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/units"
	"golang.org/x/image/draw"
)

// TODO(kai): get rid of these?

const force = true
const dontForce = false

// TextField is a widget for editing a line of text
type TextField struct { //core:embedder
	WidgetBase

	// the last saved value of the text string being edited
	Txt string `json:"-" xml:"text" set:"-"`

	// text that is displayed when the field is empty, in a lower-contrast manner
	Placeholder string `json:"-" xml:"placeholder"`

	// functions and data for textfield completion
	Complete *Complete `copier:"-" json:"-" xml:"-"`

	// replace displayed characters with bullets to conceal text
	NoEcho bool

	// if specified, a button will be added at the start of
	// the text field with this icon
	LeadingIcon icons.Icon `set:"-"`

	// if LeadingIcon is specified, the function to call when
	// the leading icon is clicked; if this is nil, the leading
	// icon will not be interactive.
	LeadingIconOnClick func(e events.Event)

	// if specified, a button will be added at the end of
	// the text field with this icon
	TrailingIcon icons.Icon `set:"-"`

	// if TrailingIcon is specified, the function to call when
	// the trailing icon is clicked; if this is nil, the trailing
	// icon will not be interactive.
	TrailingIconOnClick func(e events.Event)

	// width of cursor -- set from cursor-width property (inherited)
	CursorWidth units.Value `xml:"cursor-width"`

	// the type of the text field
	Type TextFieldTypes

	// the color used for the placeholder text; this should be set in Stylers like all other style properties; it is typically a highlighted version of the normal text color
	PlaceholderColor color.RGBA

	// the color used for the text selection background color on active text fields; this should be set in Stylers like all other style properties
	SelectColor image.Image

	// the color used for the text field cursor (caret); this should be set in Stylers like all other style properties
	CursorColor image.Image

	// true if the text has been edited relative to the original
	Edited bool `json:"-" xml:"-" set:"-"`

	// the live text string being edited, with latest modifications -- encoded as runes
	EditTxt []rune `copier:"-" json:"-" xml:"-" set:"-"`

	// maximum width that field will request, in characters, during GetSize process -- if 0 then is 50 -- ensures that large strings don't request super large values -- standard max-width can override
	MaxWidthReq int

	// effective position with any leading icon space added
	EffPos mat32.Vec2 `copier:"-" json:"-" xml:"-" set:"-"`

	// effective size, subtracting any leading and trailing icon space
	EffSize mat32.Vec2 `copier:"-" json:"-" xml:"-" set:"-"`

	// starting display position in the string
	StartPos int `copier:"-" json:"-" xml:"-" set:"-"`

	// ending display position in the string
	EndPos int `copier:"-" json:"-" xml:"-" set:"-"`

	// current cursor position
	CursorPos int `copier:"-" json:"-" xml:"-" set:"-"`

	// approximate number of chars that can be displayed at any time -- computed from font size etc
	CharWidth int `copier:"-" json:"-" xml:"-" set:"-"`

	// starting position of selection in the string
	SelectStart int `copier:"-" json:"-" xml:"-" set:"-"`

	// ending position of selection in the string
	SelectEnd int `copier:"-" json:"-" xml:"-" set:"-"`

	// initial selection position -- where it started
	SelectInit int `copier:"-" json:"-" xml:"-" set:"-"`

	// if true, select text as cursor moves
	SelectMode bool `copier:"-" json:"-" xml:"-"`

	// render version of entire text, for sizing
	RenderAll paint.Text `copier:"-" json:"-" xml:"-" set:"-"`

	// render version of just visible text
	RenderVis paint.Text `copier:"-" json:"-" xml:"-" set:"-"`

	// font height, cached during styling
	FontHeight float32 `copier:"-" json:"-" xml:"-" set:"-"`

	// oscillates between on and off for blinking
	BlinkOn bool `copier:"-" json:"-" xml:"-" set:"-"`

	// mutex for updating cursor between blinker and field
	CursorMu sync.Mutex `copier:"-" json:"-" xml:"-" view:"-" set:"-"`
}

func (tf *TextField) OnInit() {
	tf.WidgetBase.OnInit()
	tf.HandleEvents()
	tf.SetStyles()
	tf.AddContextMenu(tf.ContextMenu)
}

func (tf *TextField) OnAdd() {
	tf.WidgetBase.OnAdd()
	tf.OnClose(func(e events.Event) {
		tf.EditDone()
	})
}

func (tf *TextField) SetStyles() {
	// TOOD: figure out how to have primary cursor color
	tf.Style(func(s *styles.Style) {
		s.SetAbilities(true, abilities.Activatable, abilities.Focusable, abilities.Hoverable, abilities.Slideable, abilities.DoubleClickable)
		tf.CursorWidth.Dp(1)
		tf.SelectColor = colors.C(colors.Scheme.Select.Container)
		tf.PlaceholderColor = colors.Scheme.OnSurfaceVariant
		tf.CursorColor = colors.C(colors.Scheme.Primary.Base)

		if !tf.IsReadOnly() {
			s.Cursor = cursors.Text
		}
		s.Grow.Set(0, 0)
		s.Min.Y.Em(1.1)
		s.Min.X.Ch(20)
		s.Max.X.Ch(60)
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
			s.Background = colors.C(colors.Scheme.SurfaceContainer)

			s.MaxBorder = s.Border
			s.MaxBorder.Width.Bottom = units.Dp(2)
			s.MaxBorder.Color.Bottom = colors.Scheme.Primary.Base
			if !tf.IsReadOnly() && s.Is(states.Focused) {
				s.Border = s.MaxBorder
			} else {
				s.Border.Width.Bottom = units.Dp(1)
				s.Border.Color.Bottom = colors.Scheme.OnSurfaceVariant
			}
		case TextFieldOutlined:
			s.Border.Style.Set(styles.BorderSolid)
			s.Border.Radius = styles.BorderRadiusExtraSmall

			s.MaxBorder = s.Border
			s.MaxBorder.Width.Set(units.Dp(2))
			s.MaxBorder.Color.Set(colors.Scheme.Primary.Base)
			if !tf.IsReadOnly() && s.Is(states.Focused) {
				s.Border = s.MaxBorder
			} else {
				s.Border.Width.Set(units.Dp(1))
				s.Border.Color.Set(colors.Scheme.Outline)
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
				s.Min.Y.Em(1)
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
				s.Min.Y.Em(1)
				s.Color = colors.Scheme.OnSurfaceVariant
				s.Margin.SetLeft(units.Dp(8))
				if tf.TrailingIconOnClick == nil {
					s.SetAbilities(false, abilities.Activatable, abilities.Focusable, abilities.Hoverable)
					s.Cursor = cursors.None
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
		}
	})
}

func (tf *TextField) Destroy() {
	tf.StopCursor()
	tf.WidgetBase.Destroy()
}

// TextFieldTypes is an enum containing the
// different possible types of text fields
type TextFieldTypes int32 //enums:enum

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

// SetTextUpdate sets the text to be edited and reverts any current edit
// to reflect this new text, and triggers a Render update
func (tf *TextField) SetTextUpdate(txt string) *TextField {
	return tf.SetText(txt) // already does everything
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

// SetLeadingIconUpdate sets the leading icon of the text field to the given icon,
// and ensures that it will render with new icon, for already displayed case.
// If an on click function is specified, it also sets the leading icon on click
// function to that function. If no function is specified, it does not
// override any already set function.
func (tf *TextField) SetLeadingIconUpdate(icon icons.Icon, onClick ...func(e events.Event)) *TextField {
	updt := tf.UpdateStart()
	defer tf.UpdateEndRender(updt)

	tf.SetLeadingIcon(icon, onClick...)
	if lead, ok := tf.LeadingIconButton(); ok {
		lead.SetIconUpdate(icon)
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

// SetTrailingIconUpdate sets the trailing icon of the text field to the given icon,
// and ensures that it will render with new icon, for already displayed case.
// If an on click function is specified, it also sets the leading icon on click
// function to that function. If no function is specified, it does not
// override any already set function.
func (tf *TextField) SetTrailingIconUpdate(icon icons.Icon, onClick ...func(e events.Event)) *TextField {
	updt := tf.UpdateStart()
	defer tf.UpdateEndRender(updt)

	tf.SetTrailingIcon(icon, onClick...)
	if trail, ok := tf.TrailingIconButton(); ok {
		trail.SetIconUpdate(icon)
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
// icon button at the end of the textfield that toggles [TextField.NoEcho]
func (tf *TextField) SetTypePassword() *TextField {
	return tf.SetNoEcho(true).
		SetTrailingIcon(icons.Visibility, func(e events.Event) {
			tf.NoEcho = !tf.NoEcho
			if tf.NoEcho {
				tf.TrailingIcon = icons.Visibility
			} else {
				tf.TrailingIcon = icons.VisibilityOff
			}
			if icon, ok := tf.Parts.ChildByName("trail-icon", 1).(*Button); ok {
				icon.SetIcon(tf.TrailingIcon)
			}
		})
}

// EditDone completes editing and copies the active edited text to the text --
// called when the return key is pressed or goes out of focus
func (tf *TextField) EditDone() {
	if tf.Edited {
		tf.Edited = false
		tf.Txt = string(tf.EditTxt)
		tf.SendChange()
		// widget can be killed after SendChange
		if tf.This() == nil || tf.Is(ki.Deleted) {
			return
		}
	}
	tf.ClearSelected()
	tf.ClearCursor()
	goosi.TheApp.HideVirtualKeyboard()
}

// Revert aborts editing and reverts to last saved text
func (tf *TextField) Revert() {
	updt := tf.UpdateStart()
	defer tf.UpdateEndRender(updt)
	tf.EditTxt = []rune(tf.Txt)
	tf.Edited = false
	tf.StartPos = 0
	tf.EndPos = tf.CharWidth
	tf.SelectReset()
}

// Clear clears any existing text
func (tf *TextField) Clear() {
	updt := tf.UpdateStart()
	defer tf.UpdateEndRender(updt)
	tf.Edited = true
	tf.EditTxt = tf.EditTxt[:0]
	tf.StartPos = 0
	tf.EndPos = 0
	tf.SelectReset()
	tf.SetFocusEvent() // this is essential for ensuring that the clear applies after focus is lost..
}

//////////////////////////////////////////////////////////////////////////////////////////
//  Cursor Navigation

// CursorForward moves the cursor forward
func (tf *TextField) CursorForward(steps int) {
	updt := tf.UpdateStart()
	defer tf.UpdateEndRender(updt)
	tf.CursorPos += steps
	if tf.CursorPos > len(tf.EditTxt) {
		tf.CursorPos = len(tf.EditTxt)
	}
	if tf.CursorPos > tf.EndPos {
		inc := tf.CursorPos - tf.EndPos
		tf.EndPos += inc
	}
	if tf.SelectMode {
		tf.SelectRegUpdate(tf.CursorPos)
	}
}

// CursorForwardWord moves the cursor forward by words
func (tf *TextField) CursorForwardWord(steps int) {
	updt := tf.UpdateStart()
	defer tf.UpdateEndRender(updt)
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
				if lex.IsWordBreak(r1, r2) {
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
				if !lex.IsWordBreak(r1, r2) {
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
	if tf.SelectMode {
		tf.SelectRegUpdate(tf.CursorPos)
	}
}

// CursorBackward moves the cursor backward
func (tf *TextField) CursorBackward(steps int) {
	updt := tf.UpdateStart()
	defer tf.UpdateEndRender(updt)
	tf.CursorPos -= steps
	if tf.CursorPos < 0 {
		tf.CursorPos = 0
	}
	if tf.CursorPos <= tf.StartPos {
		dec := min(tf.StartPos, 8)
		tf.StartPos -= dec
	}
	if tf.SelectMode {
		tf.SelectRegUpdate(tf.CursorPos)
	}
}

// CursorBackwardWord moves the cursor backward by words
func (tf *TextField) CursorBackwardWord(steps int) {
	updt := tf.UpdateStart()
	defer tf.UpdateEndRender(updt)
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
				if lex.IsWordBreak(r1, r2) {
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
				if !lex.IsWordBreak(r1, r2) {
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
	if tf.SelectMode {
		tf.SelectRegUpdate(tf.CursorPos)
	}
}

// CursorStart moves the cursor to the start of the text, updating selection
// if select mode is active
func (tf *TextField) CursorStart() {
	updt := tf.UpdateStart()
	defer tf.UpdateEndRender(updt)
	tf.CursorPos = 0
	tf.StartPos = 0
	tf.EndPos = min(len(tf.EditTxt), tf.StartPos+tf.CharWidth)
	if tf.SelectMode {
		tf.SelectRegUpdate(tf.CursorPos)
	}
}

// CursorEnd moves the cursor to the end of the text
func (tf *TextField) CursorEnd() {
	updt := tf.UpdateStart()
	defer tf.UpdateEndRender(updt)
	ed := len(tf.EditTxt)
	tf.CursorPos = ed
	tf.EndPos = len(tf.EditTxt) // try -- display will adjust
	tf.StartPos = max(0, tf.EndPos-tf.CharWidth)
	if tf.SelectMode {
		tf.SelectRegUpdate(tf.CursorPos)
	}
}

// todo: ctrl+backspace = delete word
// shift+arrow = select
// uparrow = start / down = end

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
	updt := tf.UpdateStart()
	defer tf.UpdateEndRender(updt)
	tf.Edited = true
	tf.EditTxt = append(tf.EditTxt[:tf.CursorPos-steps], tf.EditTxt[tf.CursorPos:]...)
	tf.CursorBackward(steps)
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
	updt := tf.UpdateStart()
	defer tf.UpdateEndRender(updt)
	tf.Edited = true
	tf.EditTxt = append(tf.EditTxt[:tf.CursorPos], tf.EditTxt[tf.CursorPos+steps:]...)
}

// CursorBackspaceWord deletes words(s) immediately before cursor
func (tf *TextField) CursorBackspaceWord(steps int) {
	updt := tf.UpdateStart()
	defer tf.UpdateEndRender(updt)
	if tf.HasSelection() {
		tf.DeleteSelection()
		return
	}
	org := tf.CursorPos
	tf.CursorBackwardWord(steps)
	tf.Edited = true
	tf.EditTxt = append(tf.EditTxt[:tf.CursorPos], tf.EditTxt[org:]...)
}

// CursorDeleteWord deletes word(s) immediately after the cursor
func (tf *TextField) CursorDeleteWord(steps int) {
	updt := tf.UpdateStart()
	defer tf.UpdateEndRender(updt)
	if tf.HasSelection() {
		tf.DeleteSelection()
		return
	}
	// note: no update b/c signal from buf will drive update
	org := tf.CursorPos
	tf.CursorForwardWord(steps)
	tf.Edited = true
	tf.EditTxt = append(tf.EditTxt[:tf.CursorPos], tf.EditTxt[org:]...)
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
	updt := tf.UpdateStart()
	tf.SelectStart = 0
	tf.SelectInit = 0
	tf.SelectEnd = len(tf.EditTxt)
	tf.UpdateEndRender(updt)
}

// IsWordBreak defines what counts as a word break for the purposes of selecting words
func (tf *TextField) IsWordBreak(r rune) bool {
	return unicode.IsSpace(r) || unicode.IsSymbol(r) || unicode.IsPunct(r)
}

// SelectWord selects the word (whitespace delimited) that the cursor is on
func (tf *TextField) SelectWord() {
	updt := tf.UpdateStart()
	defer tf.UpdateEndRender(updt)
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
}

// SelectReset resets the selection
func (tf *TextField) SelectReset() {
	tf.SelectMode = false
	if tf.SelectStart == 0 && tf.SelectEnd == 0 {
		return
	}
	updt := tf.UpdateStart()
	tf.SelectStart = 0
	tf.SelectEnd = 0
	tf.UpdateEndRender(updt)
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
		em := tf.EventMgr()
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
	updt := tf.UpdateStart()
	defer tf.UpdateEndRender(updt)
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
	data := tf.Clipboard().Read([]string{fi.TextPlain})
	if data != nil {
		if tf.CursorPos >= tf.SelectStart && tf.CursorPos < tf.SelectEnd {
			tf.DeleteSelection()
		}
		tf.InsertAtCursor(data.Text(fi.TextPlain))
	}
}

// InsertAtCursor inserts given text at current cursor position
func (tf *TextField) InsertAtCursor(str string) {
	updt := tf.UpdateStart()
	defer tf.UpdateEndRender(updt)
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
}

func (tf *TextField) ContextMenu(m *Scene) {
	NewButton(m).SetText("Copy").SetIcon(icons.ContentCopy).SetKey(keyfun.Copy).SetState(tf.NoEcho || !tf.HasSelection(), states.Disabled).
		OnClick(func(e events.Event) {
			tf.Copy(true)
		})
	if !tf.IsReadOnly() {
		NewButton(m).SetText("Cut").SetIcon(icons.ContentCut).SetKey(keyfun.Cut).SetState(tf.NoEcho || !tf.HasSelection(), states.Disabled).
			OnClick(func(e events.Event) {
				tf.Cut()
			})
		pbt := NewButton(m).SetText("Paste").SetIcon(icons.ContentPaste).SetKey(keyfun.Paste).
			OnClick(func(e events.Event) {
				tf.Paste()
			})
		cb := tf.Scene.EventMgr.Clipboard()
		if cb != nil {
			pbt.SetState(cb.IsEmpty(), states.Disabled)
		}
	}
}

///////////////////////////////////////////////////////////////////////////////
//    Complete

// SetCompleter sets completion functions so that completions will
// automatically be offered as the user types
func (tf *TextField) SetCompleter(data any, matchFun complete.MatchFunc, editFun complete.EditFunc) {
	if matchFun == nil || editFun == nil {
		if tf.Complete != nil {
			// tf.Complete.Destroy()
		}
		tf.Complete = nil
		return
	}
	tf.Complete = NewComplete().SetContext(data).SetMatchFunc(matchFun).SetEditFunc(editFun)
	tf.Complete.OnSelect(func(e events.Event) {
		tf.CompleteText(tf.Complete.Completion)
	})
	// TODO(kai/complete): clean this up and figure out what to do about Extend and only connecting once
	// note: only need to connect once..
	// todo:
	// tf.Complete.CompleteSig.ConnectOnly(tf.This(), func(recv, send ki.Ki, sig int64, data any) {
	// 	tff := AsTextField(recv)
	// 	if sig == int64(CompleteSelect) {
	// 		tff.CompleteText(data.(string)) // always use data
	// 	} else if sig == int64(CompleteExtend) {
	// 		tff.CompleteExtend(data.(string)) // always use data
	// 	}
	// })
}

// OfferComplete pops up a menu of possible completions
func (tf *TextField) OfferComplete(forceComplete bool) {
	if tf.Complete == nil {
		return
	}
	s := string(tf.EditTxt[0:tf.CursorPos])
	cpos := tf.CharStartPos(tf.CursorPos, true).ToPoint()
	cpos.X += 5
	cpos.Y = tf.Geom.TotalBBox.Max.Y
	tf.Complete.SrcLn = 0
	tf.Complete.SrcCh = tf.CursorPos
	tf.Complete.Show(tf, cpos, s, forceComplete)
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
}

// CompleteExtend inserts the extended seed at the current cursor position
func (tf *TextField) CompleteExtend(s string) {
	if s == "" {
		return
	}
	addon := strings.TrimPrefix(s, tf.Complete.Seed)
	tf.InsertAtCursor(addon)
	tf.OfferComplete(dontForce)
}

///////////////////////////////////////////////////////////////////////////////
//    Rendering

// TextWidth returns the text width in dots between the two text string
// positions (ed is exclusive -- +1 beyond actual char)
func (tf *TextField) TextWidth(st, ed int) float32 {
	return tf.StartCharPos(ed) - tf.StartCharPos(st)
}

// StartCharPos returns the starting position of the given rune
func (tf *TextField) StartCharPos(idx int) float32 {
	if idx <= 0 || len(tf.RenderAll.Spans) != 1 {
		return 0.0
	}
	sr := &(tf.RenderAll.Spans[0])
	sz := len(sr.Render)
	if sz == 0 {
		return 0.0
	}
	if idx >= sz {
		return sr.LastPos.X
	}
	return sr.Render[idx].RelPos.X
}

// CharStartPos returns the starting render coords for the given character
// position in string -- makes no attempt to rationalize that pos (i.e., if
// not in visible range, position will be out of range too).
// if wincoords is true, then adds window box offset -- for cursor, popups
func (tf *TextField) CharStartPos(charidx int, wincoords bool) mat32.Vec2 {
	pos := tf.EffPos
	if wincoords {
		sc := tf.Scene
		pos = pos.Add(mat32.V2FromPoint(sc.SceneGeom.Pos))
	}
	cpos := tf.TextWidth(tf.StartPos, charidx)
	return mat32.V2(pos.X+cpos, pos.Y)
}

// ScrollLayoutToCursor scrolls any scrolling layout above us so that the cursor is in view
func (tf *TextField) ScrollLayoutToCursor() bool {
	ly := tf.ParentScrollLayout()
	if ly == nil {
		return false
	}
	cpos := tf.CharStartPos(tf.CursorPos, false).ToPointFloor()
	bbsz := image.Point{int(mat32.Ceil(tf.CursorWidth.Dots)), int(mat32.Ceil(tf.FontHeight))}
	bbox := image.Rectangle{Min: cpos, Max: cpos.Add(bbsz)}
	return ly.ScrollToBox(bbox)
}

var (
	// TextFieldBlinker manages cursor blinking
	TextFieldBlinker = Blinker{}

	// TextFieldSpriteName is the name of the window sprite used for the cursor
	TextFieldSpriteName = "gi.TextField.Cursor"
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
	TextFieldBlinker.Blink(SystemSettings.CursorBlinkTime, func(w Widget) {
		ttf := AsTextField(w)
		if !ttf.StateIs(states.Focused) || !w.IsVisible() {
			ttf.BlinkOn = false
			ttf.RenderCursor(false)
			TextFieldBlinker.Widget = nil
		} else {
			ttf.BlinkOn = !ttf.BlinkOn
			ttf.RenderCursor(ttf.BlinkOn)
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
	sp.Geom.Pos = tf.CharStartPos(tf.CursorPos, true).ToPointFloor()
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
		bbsz := image.Point{int(mat32.Ceil(tf.CursorWidth.Dots)), int(mat32.Ceil(tf.FontHeight))}
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

	spos := tf.CharStartPos(effst, false)

	pc := &tf.Scene.PaintContext
	tsz := tf.TextWidth(effst, effed)
	pc.FillBox(spos, mat32.V2(tsz, tf.FontHeight), tf.SelectColor)
}

// AutoScroll scrolls the starting position to keep the cursor visible
func (tf *TextField) AutoScroll() {
	st := &tf.Styles

	tf.UpdateRenderAll()

	sz := len(tf.EditTxt)

	if sz == 0 || tf.Geom.Size.Actual.Content.X <= 0 {
		tf.CursorPos = 0
		tf.EndPos = 0
		tf.StartPos = 0
		return
	}
	maxw := tf.EffSize.X
	if maxw < 0 {
		return
	}
	tf.CharWidth = int(maxw / st.UnContext.Dots(units.UnitCh)) // rough guess in chars
	if tf.CharWidth < 1 {
		tf.CharWidth = 1
	}

	// first rationalize all the values
	if tf.EndPos == 0 || tf.EndPos > sz { // not init
		tf.EndPos = sz
	}
	if tf.StartPos >= tf.EndPos {
		tf.StartPos = max(0, tf.EndPos-tf.CharWidth)
	}
	tf.CursorPos = mat32.ClampInt(tf.CursorPos, 0, sz)

	inc := int(mat32.Ceil(.1 * float32(tf.CharWidth)))
	inc = max(4, inc)

	// keep cursor in view with buffer
	startIsAnchor := true
	if tf.CursorPos < (tf.StartPos + inc) {
		tf.StartPos -= inc
		tf.StartPos = max(tf.StartPos, 0)
		tf.EndPos = tf.StartPos + tf.CharWidth
		tf.EndPos = min(sz, tf.EndPos)
	} else if tf.CursorPos > (tf.EndPos - inc) {
		tf.EndPos += inc
		tf.EndPos = min(tf.EndPos, sz)
		tf.StartPos = tf.EndPos - tf.CharWidth
		tf.StartPos = max(0, tf.StartPos)
		startIsAnchor = false
	}
	if tf.EndPos < tf.StartPos {
		return
	}

	if startIsAnchor {
		gotWidth := false
		spos := tf.StartCharPos(tf.StartPos)
		for {
			w := tf.StartCharPos(tf.EndPos) - spos
			if w < maxw {
				if tf.EndPos == sz {
					break
				}
				nw := tf.StartCharPos(tf.EndPos+1) - spos
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
	epos := tf.StartCharPos(tf.EndPos)
	for {
		w := epos - tf.StartCharPos(tf.StartPos)
		if w < maxw {
			if tf.StartPos == 0 {
				break
			}
			nw := epos - tf.StartCharPos(tf.StartPos-1)
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
func (tf *TextField) PixelToCursor(pixOff float32) int {
	st := &tf.Styles
	px := pixOff
	if px <= 0 {
		return tf.StartPos
	}

	// for selection to work correctly, we need this to be deterministic

	sz := len(tf.EditTxt)
	c := tf.StartPos + int(float64(px/st.UnContext.Dots(units.UnitCh)))
	c = min(c, sz)

	w := tf.TextWidth(tf.StartPos, c)
	if w > px {
		for w > px {
			c--
			if c <= tf.StartPos {
				c = tf.StartPos
				break
			}
			w = tf.TextWidth(tf.StartPos, c)
		}
	} else if w < px {
		for c < tf.EndPos {
			wn := tf.TextWidth(tf.StartPos, c+1)
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

// SetCursorFromPixel finds cursor location from pixel offset relative to
// WinBBox of text field, and sets current cursor to it, updating selection too.
func (tf *TextField) SetCursorFromPixel(pixOff float32, selMode events.SelectModes) {
	updt := tf.UpdateStart()
	defer tf.UpdateEndRender(updt)

	oldPos := tf.CursorPos
	tf.CursorPos = tf.PixelToCursor(pixOff)
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
}

///////////////////////////////////////////////////////////////////////////////
//    Event handling

func (tf *TextField) HandleEvents() {
	tf.HandleSelectToggle()
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
			pt := tf.PointToRelPos(e.Pos())
			tf.SetCursorFromPixel(float32(pt.X), e.SelectMode())
		case events.Middle:
			e.SetHandled()
			pt := tf.PointToRelPos(e.Pos())
			tf.SetCursorFromPixel(float32(pt.X), e.SelectMode())
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
		pt := tf.PointToRelPos(e.Pos())
		tf.SetCursorFromPixel(float32(pt.X), events.SelectOne)
	})
	tf.OnKeyChord(func(e events.Event) {
		if DebugSettings.KeyEventTrace {
			fmt.Printf("TextField KeyInput: %v\n", tf.Path())
		}
		kf := keyfun.Of(e.KeyChord())
		if !tf.StateIs(states.Focused) && kf == keyfun.Abort {
			return
		}

		// first all the keys that work for both inactive and active
		switch kf {
		case keyfun.MoveRight:
			e.SetHandled()
			tf.ShiftSelect(e)
			tf.CursorForward(1)
			tf.OfferComplete(dontForce)
		case keyfun.WordRight:
			e.SetHandled()
			tf.ShiftSelect(e)
			tf.CursorForwardWord(1)
			tf.OfferComplete(dontForce)
		case keyfun.MoveLeft:
			e.SetHandled()
			tf.ShiftSelect(e)
			tf.CursorBackward(1)
			tf.OfferComplete(dontForce)
		case keyfun.WordLeft:
			e.SetHandled()
			tf.ShiftSelect(e)
			tf.CursorBackwardWord(1)
			tf.OfferComplete(dontForce)
		case keyfun.Home:
			e.SetHandled()
			tf.ShiftSelect(e)
			tf.CancelComplete()
			tf.CursorStart()
		case keyfun.End:
			e.SetHandled()
			tf.ShiftSelect(e)
			tf.CancelComplete()
			tf.CursorEnd()
		case keyfun.SelectMode:
			e.SetHandled()
			tf.CancelComplete()
			tf.SelectModeToggle()
		case keyfun.CancelSelect:
			e.SetHandled()
			tf.CancelComplete()
			tf.SelectReset()
		case keyfun.SelectAll:
			e.SetHandled()
			tf.CancelComplete()
			tf.SelectAll()
		case keyfun.Copy:
			e.SetHandled()
			tf.CancelComplete()
			tf.Copy(true) // reset
		}
		if tf.IsReadOnly() || e.IsHandled() {
			return
		}
		switch kf {
		case keyfun.Enter:
			fallthrough
		case keyfun.FocusNext: // we process tab to make it EditDone as opposed to other ways of losing focus
			e.SetHandled()
			tf.CancelComplete()
			tf.EditDone()
			tf.FocusNext()
		case keyfun.Accept: // ctrl+enter
			e.SetHandled()
			tf.CancelComplete()
			tf.EditDone()
		case keyfun.FocusPrev:
			e.SetHandled()
			tf.CancelComplete()
			tf.EditDone()
			tf.FocusPrev()
		case keyfun.Abort: // esc
			e.SetHandled()
			tf.CancelComplete()
			tf.Revert()
			// tf.FocusChanged(FocusInactive)
		case keyfun.Backspace:
			e.SetHandled()
			tf.CursorBackspace(1)
			tf.OfferComplete(dontForce)
		case keyfun.Kill:
			e.SetHandled()
			tf.CancelComplete()
			tf.CursorKill()
		case keyfun.Delete:
			e.SetHandled()
			tf.CursorDelete(1)
			tf.OfferComplete(dontForce)
		case keyfun.BackspaceWord:
			e.SetHandled()
			tf.CursorBackspaceWord(1)
			tf.OfferComplete(dontForce)
		case keyfun.DeleteWord:
			e.SetHandled()
			tf.CursorDeleteWord(1)
			tf.OfferComplete(dontForce)
		case keyfun.Cut:
			e.SetHandled()
			tf.CancelComplete()
			tf.Cut()
		case keyfun.Paste:
			e.SetHandled()
			tf.CancelComplete()
			tf.Paste()
		case keyfun.Complete:
			e.SetHandled()
			tf.OfferComplete(force)
		case keyfun.Nil:
			if unicode.IsPrint(e.KeyRune()) {
				if !e.HasAnyModifier(key.Control, key.Meta) {
					e.SetHandled()
					tf.InsertAtCursor(string(e.KeyRune()))
					if e.KeyRune() == ' ' {
						tf.CancelComplete()
					} else {
						tf.OfferComplete(dontForce)
					}
					tf.Send(events.Input)
				}
			}
		}
	})
	tf.OnFocus(func(e events.Event) {
		if tf.IsReadOnly() {
			return
		}
		if tf.AbilityIs(abilities.Focusable) {
			tf.ScrollToMe()
			if _, ok := tf.Parent().Parent().(*Spinner); ok {
				goosi.TheApp.ShowVirtualKeyboard(goosi.NumberKeyboard)
			} else {
				goosi.TheApp.ShowVirtualKeyboard(goosi.SingleLineKeyboard)
			}
			tf.SetState(true, states.Focused)
		}
	})
	tf.OnFocusLost(func(e events.Event) {
		if tf.IsReadOnly() {
			return
		}
		if tf.AbilityIs(abilities.Focusable) {
			tf.EditDone()
			tf.SetState(false, states.Focused)
		}
	})
}

func (tf *TextField) ConfigWidget() {
	config := ki.Config{}

	tf.EditTxt = []rune(tf.Txt)
	tf.Edited = false

	lii, tii := -1, -1
	if !tf.IsReadOnly() {
		if tf.LeadingIcon.IsSet() {
			config.Add(ButtonType, "lead-icon")
			lii = 0
		}
		if tf.TrailingIcon.IsSet() {
			config.Add(StretchType, "trail-icon-str")
			config.Add(ButtonType, "trail-icon")
			if lii == -1 {
				tii = 1
			} else {
				tii = 2
			}
		}
	}
	tf.ConfigParts(config, func() {
		if lii >= 0 {
			li := tf.Parts.Child(lii).(*Button)
			li.SetIcon(tf.LeadingIcon)
		}
		if tii != -1 {
			ti := tf.Parts.Child(tii).(*Button)
			ti.SetIcon(tf.TrailingIcon)
		}
	})
}

////////////////////////////////////////////////////
//  Widget Interface

// StyleTextField does text field styling -- sets StyMu Lock
func (tf *TextField) StyleTextField() {
	tf.StyMu.Lock()
	tf.SetAbilities(!tf.IsReadOnly(), abilities.Focusable)
	tf.ApplyStyleWidget()
	tf.CursorWidth.ToDots(&tf.Styles.UnContext)
	tf.StyMu.Unlock()
}

func (tf *TextField) ApplyStyle() {
	tf.StyleTextField()
}

func (tf *TextField) UpdateRenderAll() bool {
	st := &tf.Styles
	st.Font = paint.OpenFont(st.FontRender(), &st.UnContext)
	txt := tf.EditTxt
	if tf.NoEcho {
		txt = concealDots(len(tf.EditTxt))
	}
	tf.RenderAll.SetRunes(txt, st.FontRender(), &st.UnContext, &st.Text, true, 0, 0)
	return true
}

func (tf *TextField) SizeUp() {
	tf.WidgetBase.SizeUp()
	tmptxt := tf.EditTxt
	if len(tf.Txt) == 0 && len(tf.Placeholder) > 0 {
		tf.EditTxt = []rune(tf.Placeholder)
	} else {
		tf.EditTxt = []rune(tf.Txt)
	}
	tf.Edited = false
	tf.StartPos = 0
	maxlen := tf.MaxWidthReq
	if maxlen <= 0 {
		maxlen = 50
	}
	tf.EndPos = min(len(tf.EditTxt), maxlen)
	tf.UpdateRenderAll()
	tf.FontHeight = tf.RenderAll.Size.Y
	w := tf.TextWidth(tf.StartPos, tf.EndPos)
	// w += 2.0 // give some extra buffer
	nsz := mat32.V2(w, tf.FontHeight)
	sz := &tf.Geom.Size
	sz.FitSizeMax(&sz.Actual.Content, nsz)
	sz.SetTotalFromContent(&sz.Actual)
	tf.EditTxt = tmptxt
}

func (tf *TextField) ScenePos() {
	tf.WidgetBase.ScenePos()
	tf.SetEffPosAndSize()
}

// LeadingIconButton returns the [LeadingIcon] [Button] if present, else false
func (tf *TextField) LeadingIconButton() (*Button, bool) {
	if tf.Parts == nil {
		return nil, false
	}
	bi := tf.Parts.ChildByName("lead-icon", 0)
	if bi == nil {
		return nil, false
	}
	return bi.(*Button), true
}

// TrailingIconButton returns the [TrailingIcon] [Button] if present, else false
func (tf *TextField) TrailingIconButton() (*Button, bool) {
	if tf.Parts == nil {
		return nil, false
	}
	bi := tf.Parts.ChildByName("trail-icon", 1)
	if bi == nil {
		return nil, false
	}
	return bi.(*Button), true
}

// SetEffPosAndSize sets the effective position and size of
// the textfield based on its base position and size
// and its icons or lack thereof
func (tf *TextField) SetEffPosAndSize() {
	// if tf.Parts == nil {
	// 	fmt.Println("nil parts sepas")
	// 	tf.ConfigParts(tf.Sc)
	// }
	sz := tf.Geom.Size.Actual.Content
	pos := tf.Geom.Pos.Content
	if lead, ok := tf.LeadingIconButton(); ok {
		pos.X += lead.Geom.Size.Actual.Total.X
		sz.X -= lead.Geom.Size.Actual.Total.X
	}
	if trail, ok := tf.TrailingIconButton(); ok {
		sz.X -= trail.Geom.Size.Actual.Total.X
	}
	pos.Y += 0.5 * (sz.Y - tf.FontHeight) // center
	tf.EffSize = sz.Ceil()
	tf.EffPos = pos.Ceil()
}

func (tf *TextField) RenderTextField() {
	pc, _ := tf.RenderLock()
	defer tf.RenderUnlock()

	tf.AutoScroll() // inits paint with our style
	st := &tf.Styles
	st.Font = paint.OpenFont(st.FontRender(), &st.UnContext)
	tf.RenderStdBox(st)
	if len(tf.EditTxt) == 0 {
		return
	}
	cur := tf.EditTxt[tf.StartPos:tf.EndPos]
	tf.RenderSelect()
	pos := tf.EffPos
	if len(tf.EditTxt) == 0 && len(tf.Placeholder) > 0 {
		prevColor := st.Color
		st.Color = tf.PlaceholderColor
		tf.RenderVis.SetString(tf.Placeholder, st.FontRender(), &st.UnContext, &st.Text, true, 0, 0)
		tf.RenderVis.RenderTopPos(pc, pos)
		st.Color = prevColor
	} else {
		if tf.NoEcho {
			cur = concealDots(len(cur))
		}
		tf.RenderVis.SetRunes(cur, st.FontRender(), &st.UnContext, &st.Text, true, 0, 0)
		tf.RenderVis.RenderTopPos(pc, pos)
	}
}

func (tf *TextField) Render() {
	// todo: this is probably not a great idea
	// if tf.StateIs(states.Focused) && TextFieldBlinker.Widget == tf.This().(Widget) {
	// 	tf.ScrollLayoutToCursor()
	// }
	if tf.PushBounds() {
		tf.RenderTextField()
		if !tf.IsReadOnly() {
			if tf.StateIs(states.Focused) {
				tf.StartCursor()
			} else {
				tf.StopCursor()
			}
		}
		tf.RenderParts()
		tf.RenderChildren()
		tf.PopBounds()
	}
}

// concealDots creates an n-length []rune of bullet characters.
func concealDots(n int) []rune {
	dots := make([]rune, n)
	for i := range dots {
		dots[i] = 0x2022 // bullet character •
	}
	return dots
}
