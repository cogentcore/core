// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"fmt"
	"image"
	"image/color"
	"log/slog"
	"strings"
	"sync"
	"time"
	"unicode"

	"goki.dev/colors"
	"goki.dev/cursors"
	"goki.dev/gi/v2/keyfun"
	"goki.dev/girl/abilities"
	"goki.dev/girl/paint"
	"goki.dev/girl/states"
	"goki.dev/girl/styles"
	"goki.dev/girl/units"
	"goki.dev/goosi"
	"goki.dev/goosi/events"
	"goki.dev/goosi/events/key"
	"goki.dev/goosi/mimedata"
	"goki.dev/icons"
	"goki.dev/ki/v2"
	"goki.dev/mat32/v2"
	"goki.dev/pi/v2/complete"
	"goki.dev/pi/v2/filecat"
	"goki.dev/pi/v2/lex"
	"golang.org/x/image/draw"
)

const force = true
const dontForce = false

// CursorBlinkTime is time that cursor blinks on
// and off -- set to 0 to disable blinking
var CursorBlinkTime = 500 * time.Millisecond

// TextField is a widget for editing a line of text
type TextField struct { //goki:embedder
	WidgetBase

	// the last saved value of the text string being edited
	Txt string `json:"-" xml:"text" set:"-"`

	// text that is displayed when the field is empty, in a lower-contrast manner
	Placeholder string `json:"-" xml:"placeholder"`

	// functions and data for textfield completion
	Complete *Complete `copy:"-" json:"-" xml:"-"`

	// replace displayed characters with bullets to conceal text
	NoEcho bool

	// if specified, a button will be added at the start of the text field with this icon
	LeadingIcon icons.Icon

	// if specified, a button will be added at the end of the text field with this icon
	TrailingIcon icons.Icon

	// width of cursor -- set from cursor-width property (inherited)
	CursorWidth units.Value `xml:"cursor-width"`

	// the type of the text field
	Type TextFieldTypes

	// the color used for the placeholder text; this should be set in Stylers like all other style properties; it is typically a highlighted version of the normal text color
	PlaceholderColor color.RGBA

	// the color used for the text selection background color on active text fields; this should be set in Stylers like all other style properties
	SelectColor colors.Full

	// the color used for the text field cursor (caret); this should be set in Stylers like all other style properties
	CursorColor colors.Full

	// true if the text has been edited relative to the original
	Edited bool `json:"-" xml:"-" set:"-"`

	// the live text string being edited, with latest modifications -- encoded as runes
	EditTxt []rune `json:"-" xml:"-" set:"-"`

	// maximum width that field will request, in characters, during GetSize process -- if 0 then is 50 -- ensures that large strings don't request super large values -- standard max-width can override
	MaxWidthReq int

	// effective position with any leading icon space added
	EffPos mat32.Vec2 `copy:"-" json:"-" xml:"-" set:"-"`

	// effective size, subtracting any leading and trailing icon space
	EffSize mat32.Vec2 `copy:"-" json:"-" xml:"-" set:"-"`

	// starting display position in the string
	StartPos int `copy:"-" json:"-" xml:"-" set:"-"`

	// ending display position in the string
	EndPos int `copy:"-" json:"-" xml:"-" set:"-"`

	// current cursor position
	CursorPos int `copy:"-" json:"-" xml:"-" set:"-"`

	// approximate number of chars that can be displayed at any time -- computed from font size etc
	CharWidth int `copy:"-" json:"-" xml:"-" set:"-"`

	// starting position of selection in the string
	SelectStart int `copy:"-" json:"-" xml:"-" set:"-"`

	// ending position of selection in the string
	SelectEnd int `copy:"-" json:"-" xml:"-" set:"-"`

	// initial selection position -- where it started
	SelectInit int `copy:"-" json:"-" xml:"-" set:"-"`

	// if true, select text as cursor moves
	SelectMode bool `copy:"-" json:"-" xml:"-"`

	// render version of entire text, for sizing
	RenderAll paint.Text `copy:"-" json:"-" xml:"-" set:"-"`

	// render version of just visible text
	RenderVis paint.Text `copy:"-" json:"-" xml:"-" set:"-"`

	// font height, cached during styling
	FontHeight float32 `copy:"-" json:"-" xml:"-" set:"-"`

	// oscillates between on and off for blinking
	BlinkOn bool `copy:"-" json:"-" xml:"-" set:"-"`

	// mutex for updating cursor between blinker and field
	CursorMu sync.Mutex `copy:"-" json:"-" xml:"-" view:"-" set:"-"`
}

func (tf *TextField) CopyFieldsFrom(frm any) {
	fr := frm.(*TextField)
	tf.WidgetBase.CopyFieldsFrom(&fr.WidgetBase)
	tf.Txt = fr.Txt
	tf.Placeholder = fr.Placeholder
	tf.NoEcho = fr.NoEcho
	tf.LeadingIcon = fr.LeadingIcon
	tf.TrailingIcon = fr.TrailingIcon
	tf.CursorWidth = fr.CursorWidth
	tf.Edited = fr.Edited
	tf.MaxWidthReq = fr.MaxWidthReq
}

func (tf *TextField) OnInit() {
	tf.HandleTextFieldEvents()
	tf.TextFieldStyles()
}

func (tf *TextField) TextFieldStyles() {
	// TOOD: figure out how to have primary cursor color
	tf.Style(func(s *styles.Style) {
		s.SetAbilities(true, abilities.Activatable, abilities.Focusable, abilities.Hoverable, abilities.Slideable)
		tf.CursorWidth.Dp(1)
		tf.SelectColor.SetSolid(colors.Scheme.Select.Container)
		tf.PlaceholderColor = colors.Scheme.OnSurfaceVariant
		tf.CursorColor.SetSolid(colors.Scheme.Primary.Base)

		if !tf.IsReadOnly() {
			s.Cursor = cursors.Text
		}
		s.SetMinPrefWidth(units.Em(10))
		s.Padding.Set(units.Dp(8), units.Dp(16))
		if !tf.LeadingIcon.IsNil() {
			s.Padding.Left.Dp(12)
		}
		if !tf.TrailingIcon.IsNil() {
			s.Padding.Right.Dp(12)
		}
		s.Text.Align = styles.AlignLeft
		s.Color = colors.Scheme.OnSurface
		switch tf.Type {
		case TextFieldFilled:
			s.Border.Style.Set(styles.BorderNone)
			s.Border.Style.Bottom = styles.BorderSolid
			s.Border.Width.Set()
			s.Border.Color.Set()
			s.Border.Radius = styles.BorderRadiusExtraSmallTop
			s.BackgroundColor.SetSolid(colors.Scheme.SurfaceContainer)

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
		if s.Is(states.Selected) {
			s.BackgroundColor.SetSolid(colors.Scheme.Select.Container)
		}
	})
	tf.OnWidgetAdded(func(w Widget) {
		switch w.PathFrom(tf) {
		case "parts/lead-icon":
			lead := w.(*Button)
			lead.Type = ButtonAction
			lead.Style(func(s *styles.Style) {
				s.Font.Size.Dp(20)
				s.Margin.Right.Dp(16)
				s.Color = colors.Scheme.OnSurfaceVariant
				s.AlignV = styles.AlignMiddle
			})
		case "parts/trail-icon":
			trail := w.(*Button)
			trail.Type = ButtonAction
			trail.Style(func(s *styles.Style) {
				s.Font.Size.Dp(20)
				s.Margin.Left.Dp(16)
				s.Color = colors.Scheme.OnSurfaceVariant
				s.AlignV = styles.AlignMiddle
			})
			switch tf.TrailingIcon {
			case icons.Close:
				trail.OnClick(func(e events.Event) {
					tf.Clear()
				})
			case icons.Visibility, icons.VisibilityOff:
				trail.OnClick(func(e events.Event) {
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
		}
	})
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

// SetText sets the text to be edited and reverts any current edit to reflect this new text
func (tf *TextField) SetText(txt string) *TextField {
	if tf.Txt == txt && !tf.Edited {
		return tf
	}
	tf.Txt = txt
	tf.Revert()
	return tf
}

// AddClearButton adds a trailing icon button at the end
// of the textfield that clears the text in the textfield when pressed
func (tf *TextField) AddClearButton() *TextField {
	tf.TrailingIcon = icons.Close
	return tf
}

// SetTypePassword enables [TextField.NoEcho] and adds a trailing
// icon button at the end of the textfield that toggles [TextField.NoEcho]
func (tf *TextField) SetTypePassword() *TextField {
	tf.NoEcho = true
	tf.TrailingIcon = icons.Visibility
	return tf
}

// EditDone completes editing and copies the active edited text to the text --
// called when the return key is pressed or goes out of focus
func (tf *TextField) EditDone() {
	if tf.Edited {
		tf.Edited = false
		tf.Txt = string(tf.EditTxt)
		tf.SendChange()
		// widget can be killed after sendchange
		if tf == nil || tf.This() == nil || tf.Is(ki.Deleted) {
			return
		}
	}
	tf.ClearSelected()
	tf.ClearCursor()
	goosi.TheApp.HideVirtualKeyboard()
}

// EditDeFocused completes editing and copies the active edited text to the text --
// called when field is made inactive due to interactions elsewhere.
func (tf *TextField) EditDeFocused() {
	if tf.Edited {
		tf.Edited = false
		tf.Txt = string(tf.EditTxt)
		// todo: focus lost?
		// tf.TextFieldSig.Emit(tf.This(), int64(TextFieldDeFocused), tf.Txt)
	}
	tf.ClearSelected()
	tf.ClearCursor()
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
	tf.GrabFocus() // this is essential for ensuring that the clear applies after focus is lost..
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
	if unicode.IsSpace(r) || unicode.IsSymbol(r) || unicode.IsPunct(r) {
		return true
	}
	return false
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
			em.ClipBoard().Write(mimedata.NewText(cut))
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

// MimeData adds selection to mimedata.
// Satisfies Clipper interface -- can be extended in subtypes.
func (tf *TextField) MimeData(md *mimedata.Mimes) {
	cpy := tf.Selection()
	*md = append(*md, mimedata.NewTextData(cpy))
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
	md := mimedata.NewMimes(0, 1)
	tf.This().(Clipper).MimeData(&md)
	em := tf.EventMgr()
	if em != nil {
		em.ClipBoard().Write(md)
	}
	if reset {
		tf.SelectReset()
	}
}

// Paste inserts text from the clipboard at current cursor position -- if
// cursor is within a current selection, that selection is replaced.
// Satisfies Clipper interface -- can be extended in subtypes.
func (tf *TextField) Paste() {
	em := tf.EventMgr()
	if em == nil {
		return
	}
	data := em.ClipBoard().Read([]string{filecat.TextPlain})
	if data != nil {
		if tf.CursorPos >= tf.SelectStart && tf.CursorPos < tf.SelectEnd {
			tf.DeleteSelection()
		}
		tf.InsertAtCursor(data.Text(filecat.TextPlain))
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
			tf.This().(Clipper).Copy(true)
		})
	if !tf.IsReadOnly() {
		NewButton(m).SetText("Cut").SetIcon(icons.ContentCut).SetKey(keyfun.Cut).SetState(tf.NoEcho || !tf.HasSelection(), states.Disabled).
			OnClick(func(e events.Event) {
				tf.This().(Clipper).Cut()
			})
		pbt := NewButton(m).SetText("Paste").SetIcon(icons.ContentPaste).SetKey(keyfun.Paste).
			OnClick(func(e events.Event) {
				tf.This().(Clipper).Paste()
			})
		cb := tf.Sc.EventMgr.ClipBoard()
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
	cpos.Y += 10
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
	st := &tf.Styles
	spc := st.BoxSpace()
	pos := tf.EffPos.Add(spc.Pos())
	if wincoords {
		mvp := tf.Sc
		pos = pos.Add(mat32.NewVec2FmPoint(mvp.Geom.Pos))
	}
	cpos := tf.TextWidth(tf.StartPos, charidx)
	return mat32.Vec2{pos.X + cpos, pos.Y}
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

// TextFieldBlinkMu is mutex protecting TextFieldBlink updating and access
var TextFieldBlinkMu sync.Mutex

// TextFieldBlinker is the time.Ticker for blinking cursors for text fields,
// only one of which can be active at at a time
var TextFieldBlinker *time.Ticker

// BlinkingTextField is the text field that is blinking
var BlinkingTextField *TextField

// TextFieldSpriteName is the name of the window sprite used for the cursor
var TextFieldSpriteName = "gi.TextField.Cursor"

// TextFieldBlink is function that blinks text field cursor
func TextFieldBlink() {
	for {
		TextFieldBlinkMu.Lock()
		if TextFieldBlinker == nil {
			TextFieldBlinkMu.Unlock()
			return // shutdown..
		}
		TextFieldBlinkMu.Unlock()
		<-TextFieldBlinker.C
		TextFieldBlinkMu.Lock()
		if BlinkingTextField == nil || BlinkingTextField.This() == nil || BlinkingTextField.Is(ki.Deleted) {
			TextFieldBlinkMu.Unlock()
			continue
		}
		tf := BlinkingTextField
		if tf.Sc == nil || tf.Sc.MainStage() == nil || !tf.StateIs(states.Focused) || !tf.This().(Widget).IsVisible() {
			BlinkingTextField = nil
			TextFieldBlinkMu.Unlock()
			continue
		}
		tf.BlinkOn = !tf.BlinkOn
		tf.RenderCursor(tf.BlinkOn)
		TextFieldBlinkMu.Unlock()
	}
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
	if CursorBlinkTime == 0 {
		tf.RenderCursor(true)
		return
	}
	TextFieldBlinkMu.Lock()
	if TextFieldBlinker == nil {
		TextFieldBlinker = time.NewTicker(CursorBlinkTime)
		go TextFieldBlink()
	}
	tf.BlinkOn = true
	tf.RenderCursor(true)
	BlinkingTextField = tf
	TextFieldBlinkMu.Unlock()
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
	if !tf.This().(Widget).IsVisible() {
		return
	}
	TextFieldBlinkMu.Lock()
	if BlinkingTextField == tf {
		BlinkingTextField = nil
	}
	TextFieldBlinkMu.Unlock()
}

// RenderCursor renders the cursor on or off, as a sprite that is either on or off
func (tf *TextField) RenderCursor(on bool) {
	if tf == nil || tf.This() == nil {
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
	sc := tf.Sc
	if sc == nil {
		return nil
	}
	ms := sc.MainStage()
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
		draw.Draw(sp.Pixels, ibox, &image.Uniform{tf.CursorColor.Solid}, image.Point{}, draw.Src)
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
func (tf *TextField) RenderSelect(sc *Scene) {
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

	rs := &sc.RenderState
	pc := &rs.Paint
	tsz := tf.TextWidth(effst, effed)
	pc.FillBox(rs, spos, mat32.NewVec2(tsz, tf.FontHeight), &tf.SelectColor)
}

// AutoScroll scrolls the starting position to keep the cursor visible
func (tf *TextField) AutoScroll() {
	st := &tf.Styles

	tf.UpdateRenderAll()

	sz := len(tf.EditTxt)

	if sz == 0 || tf.LayState.Alloc.Size.X <= 0 {
		tf.CursorPos = 0
		tf.EndPos = 0
		tf.StartPos = 0
		return
	}
	spc := st.BoxSpace()
	maxw := tf.EffSize.X - spc.Size().X
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

	spc := st.BoxSpace()
	px := pixOff - spc.Pos().X

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

// func (tf *TextField) HandleEvent(ev events.Event) {
// 	// if tf.Sc.Type == ScDialog {
// 	// todo: need dialogsig!
// 	// dlg.DialogSig.Connect(tf.This(), func(recv, send ki.Ki, sig int64, data any) {
// 	// 	tff := AsTextField(recv)
// 	// 	if sig == int64(DialogAccepted) {
// 	// 		tff.EditDone()
// 	// 	}
// 	// })
// 	// }
// }

func (tf *TextField) HandleTextFieldMouse() {
	tf.On(events.MouseDown, func(e events.Event) {
		if !tf.StateIs(states.Focused) {
			tf.GrabFocus() // always grab, even if read only..
		}
		if tf.IsReadOnly() {
			return
		}
		e.SetHandled()
		switch e.MouseButton() {
		case events.Left:
			pt := tf.PointToRelPos(e.LocalPos())
			tf.SetCursorFromPixel(float32(pt.X), e.SelectMode())
		case events.Middle:
			e.SetHandled()
			pt := tf.PointToRelPos(e.LocalPos())
			tf.SetCursorFromPixel(float32(pt.X), e.SelectMode())
			tf.Paste()
		}
	})
	tf.OnClick(func(e events.Event) {
		if tf.IsReadOnly() {
			return
		}
		tf.GrabFocus()
		tf.Send(events.Focus, e) // sets focused flag
	})
	tf.On(events.DoubleClick, func(e events.Event) {
		if tf.IsReadOnly() {
			return
		}
		if !tf.IsReadOnly() && !tf.StateIs(states.Focused) {
			tf.GrabFocus()
			tf.Send(events.Focus, e) // sets focused flag
		}
		e.SetHandled()
		if tf.HasSelection() {
			if tf.SelectStart == 0 && tf.SelectEnd == len(tf.EditTxt) {
				tf.SelectReset()
			} else {
				tf.SelectAll()
			}
		} else {
			tf.SelectWord()
		}
	})
	tf.On(events.SlideMove, func(e events.Event) {
		if tf.IsReadOnly() {
			return
		}
		e.SetHandled()
		if !tf.SelectMode {
			tf.SelectModeToggle()
		}
		pt := tf.PointToRelPos(e.LocalPos())
		tf.SetCursorFromPixel(float32(pt.X), events.SelectOne)
	})
}

func (tf *TextField) HandleTextFieldKeys() {
	tf.OnKeyChord(func(e events.Event) {
		if KeyEventTrace {
			fmt.Printf("TextField KeyInput: %v\n", tf.Path())
		}
		kf := keyfun.Of(e.KeyChord())
		// todo:
		// win := tf.ParentRenderWin()
		// if tf.Complete != nil {
		// 	cpop := win.CurPopup()
		// 	if PopupIsCompleter(cpop) {
		// 		tf.Complete.KeyInput(kf)
		// 	}
		// }

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
			tf.This().(Clipper).Copy(true) // reset
		}
		if tf.IsReadOnly() || e.IsHandled() {
			return
		}
		switch kf {
		case keyfun.Enter:
			fallthrough
		case keyfun.FocusNext: // we process tab to make it EditDone as opposed to other ways of losing focus
			fallthrough
		case keyfun.Accept: // ctrl+enter
			e.SetHandled()
			tf.CancelComplete()
			tf.EditDone()
			tf.FocusNext()
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
			tf.This().(Clipper).Cut()
		case keyfun.Paste:
			e.SetHandled()
			tf.CancelComplete()
			tf.This().(Clipper).Paste()
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
				}
			}
		}
	})
}

/*
func (tf *TextField) FocusChanged(change FocusChanges) {
	switch change {
	case FocusLost:
		tf.SetState(false, states.Focused)
		tf.EditDone()
		tf.ApplyStyleUpdate(tf.Sc)
	case FocusGot:
		tf.SetState(true, states.Focused)
		tf.ScrollToMe()
		// tf.CursorEnd()
		// tf.EmitFocusedSignal()
		tf.ApplyStyleUpdate(tf.Sc)
		if _, ok := tf.Parent().Parent().(*Spinner); ok {
			goosi.TheApp.ShowVirtualKeyboard(goosi.NumberKeyboard)
		} else {
			goosi.TheApp.ShowVirtualKeyboard(goosi.SingleLineKeyboard)
		}
	case FocusInactive:
		tf.SetState(false, states.Focused)
		tf.EditDeFocused()
		tf.ApplyStyleUpdate(tf.Sc)
		goosi.TheApp.HideVirtualKeyboard()
	case FocusActive:
		tf.SetState(true, states.Focused)
		tf.ScrollToMe()
		tf.ApplyStyleUpdate(tf.Sc)
		if _, ok := tf.Parent().Parent().(*Spinner); ok {
			goosi.TheApp.ShowVirtualKeyboard(goosi.NumberKeyboard)
		} else {
			goosi.TheApp.ShowVirtualKeyboard(goosi.SingleLineKeyboard)
		}
		// todo: see about cursor
	}
}
*/

func (tf *TextField) HandleTextFieldStateFromFocus() {
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

func (tf *TextField) HandleTextFieldEvents() {
	tf.HandleWidgetEvents()
	tf.HandleSelectToggle()
	tf.HandleTextFieldMouse()
	tf.HandleTextFieldStateFromFocus()
	tf.HandleTextFieldKeys()
}

func (tf *TextField) ConfigParts(sc *Scene) {
	parts := tf.NewParts(LayoutHoriz)
	if tf.IsReadOnly() || (tf.LeadingIcon.IsNil() && tf.TrailingIcon.IsNil()) {
		parts.DeleteChildren(ki.DestroyKids)
		return
	}
	config := ki.Config{}
	leadIconIdx, trailIconIdx := -1, -1
	if !tf.LeadingIcon.IsNil() {
		// config.Add(StretchType, "lead-icon-str")
		config.Add(ButtonType, "lead-icon")
		leadIconIdx = 0
	}
	if !tf.TrailingIcon.IsNil() {
		config.Add(SpaceType, "trail-icon-str")
		config.Add(ButtonType, "trail-icon")
		if leadIconIdx == -1 {
			trailIconIdx = 1
		} else {
			trailIconIdx = 2
		}
	}

	mods, updt := parts.ConfigChildren(config)
	if mods || tf.NeedsRebuild() {
		if leadIconIdx != -1 {
			leadIcon := parts.Child(leadIconIdx).(*Button)
			leadIcon.SetIcon(tf.LeadingIcon)
		}
		if trailIconIdx != -1 {
			trailIcon := parts.Child(trailIconIdx).(*Button)
			trailIcon.SetIcon(tf.TrailingIcon)
		}
		parts.UpdateEnd(updt)
		tf.SetNeedsLayoutUpdate(sc, updt)
	}
}

////////////////////////////////////////////////////
//  Widget Interface

func (tf *TextField) ConfigWidget(sc *Scene) {
	tf.EditTxt = []rune(tf.Txt)
	tf.Edited = false
	tf.ConfigParts(sc)
}

// StyleTextField does text field styling -- sets StyMu Lock
func (tf *TextField) StyleTextField(sc *Scene) {
	tf.StyMu.Lock()
	tf.SetCanFocusIfActive()
	tf.ApplyStyleWidget(sc)
	tf.CursorWidth.ToDots(&tf.Styles.UnContext)
	tf.StyMu.Unlock()
}

func (tf *TextField) ApplyStyle(sc *Scene) {
	tf.StyleTextField(sc)
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

func (tf *TextField) GetSize(sc *Scene, iter int) {
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
	w += 2.0 // give some extra buffer
	// fmt.Printf("fontheight: %v width: %v\n", tf.FontHeight, w)
	tf.GetSizeFromWH(w, tf.FontHeight)
	tf.EditTxt = tmptxt
}

func (tf *TextField) DoLayout(sc *Scene, parBBox image.Rectangle, iter int) bool {
	if !tf.TrailingIcon.IsNil() {
		slog.Info("tf dl", "icon", tf.TrailingIcon, "parts", tf.Parts.Kids)
	}
	tf.DoLayoutBase(sc, parBBox, iter)
	tf.DoLayoutParts(sc, parBBox, iter)
	redo := tf.DoLayoutChildren(sc, iter)
	tf.SetEffPosAndSize()
	return redo
}

// SetEffPosAndSize sets the effective position and size of
// the textfield based on its base position and size
// and its icons or lack thereof
func (tf *TextField) SetEffPosAndSize() {
	if tf.Parts == nil {
		fmt.Println("nil parts sepas")
		tf.ConfigParts(tf.Sc)
	}
	sz := tf.LayState.Alloc.Size
	pos := tf.LayState.Alloc.Pos
	if lead, ok := tf.Parts.ChildByName("lead-icon", 0).(*Button); ok {
		pos.X += lead.LayState.Alloc.Size.X
		sz.X -= lead.LayState.Alloc.Size.X
	}
	if trail, ok := tf.Parts.ChildByName("trail-icon", 1).(*Button); ok {
		sz.X -= trail.LayState.Alloc.Size.X
	}
	tf.EffSize = sz
	tf.EffPos = pos
}

func (tf *TextField) RenderTextField(sc *Scene) {
	rs, _, _ := tf.RenderLock(sc)
	defer tf.RenderUnlock(rs)

	tf.SetEffPosAndSize()

	tf.AutoScroll() // inits paint with our style
	st := &tf.Styles
	st.Font = paint.OpenFont(st.FontRender(), &st.UnContext)
	tf.RenderStdBox(sc, st)
	cur := tf.EditTxt[tf.StartPos:tf.EndPos]
	tf.RenderSelect(sc)
	pos := tf.EffPos.Add(st.BoxSpace().Pos())
	if len(tf.EditTxt) == 0 && len(tf.Placeholder) > 0 {
		prevColor := st.Color
		st.Color = tf.PlaceholderColor
		tf.RenderVis.SetString(tf.Placeholder, st.FontRender(), &st.UnContext, &st.Text, true, 0, 0)
		tf.RenderVis.RenderTopPos(rs, pos)
		st.Color = prevColor
	} else {
		if tf.NoEcho {
			cur = concealDots(len(cur))
		}
		tf.RenderVis.SetRunes(cur, st.FontRender(), &st.UnContext, &st.Text, true, 0, 0)
		tf.RenderVis.RenderTopPos(rs, pos)
	}
}

func (tf *TextField) Render(sc *Scene) {
	if tf.StateIs(states.Focused) && BlinkingTextField == tf {
		tf.ScrollLayoutToCursor()
	}
	if tf.PushBounds(sc) {
		tf.RenderTextField(sc)
		if !tf.IsReadOnly() {
			if tf.StateIs(states.Focused) {
				tf.StartCursor()
			} else {
				tf.StopCursor()
			}
		}
		tf.RenderParts(sc)
		tf.RenderChildren(sc)
		tf.PopBounds(sc)
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
