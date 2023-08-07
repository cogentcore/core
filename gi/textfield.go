// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"fmt"
	"image"
	"image/draw"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/goki/gi/girl"
	"github.com/goki/gi/gist"
	"github.com/goki/gi/icons"
	"github.com/goki/gi/oswin"
	"github.com/goki/gi/oswin/cursor"
	"github.com/goki/gi/oswin/key"
	"github.com/goki/gi/oswin/mimedata"
	"github.com/goki/gi/oswin/mouse"
	"github.com/goki/gi/units"
	"github.com/goki/ki/ints"
	"github.com/goki/ki/ki"
	"github.com/goki/ki/kit"
	"github.com/goki/mat32"
	"github.com/goki/pi/complete"
	"github.com/goki/pi/filecat"
)

const force = true
const dontForce = false

// CursorBlinkMSec is number of milliseconds that cursor blinks on
// and off -- set to 0 to disable blinking
var CursorBlinkMSec = 500

////////////////////////////////////////////////////////////////////////////////////////
// TextField

// TextField is a widget for editing a line of text
type TextField struct {
	PartsWidgetBase

	// the last saved value of the text string being edited
	Txt string `json:"-" xml:"text" desc:"the last saved value of the text string being edited"`

	// text that is displayed when the field is empty, in a lower-contrast manner
	Placeholder string `json:"-" xml:"placeholder" desc:"text that is displayed when the field is empty, in a lower-contrast manner"`

	// add a clear action x at right side of edit, set from clear-act property (inherited) -- on by default
	ClearAct bool `xml:"clear-act" desc:"add a clear action x at right side of edit, set from clear-act property (inherited) -- on by default"`

	// width of cursor -- set from cursor-width property (inherited)
	CursorWidth units.Value `xml:"cursor-width" desc:"width of cursor -- set from cursor-width property (inherited)"`

	// the type of the text field
	Type TextFieldTypes `desc:"the type of the text field"`

	// the current state of the text field
	State TextFieldStates `desc:"the current state of the text field"`

	// the color used for the placeholder text; this should be set in StyleFuncs like all other style properties; it is typically a highlighted version of the normal text color
	PlaceholderColor gist.Color `desc:"the color used for the placeholder text; this should be set in StyleFuncs like all other style properties; it is typically a highlighted version of the normal text color"`

	// the color used for the text selection background color on active text fields; this should be set in StyleFuncs like all other style properties
	SelectColor gist.ColorSpec `desc:"the color used for the text selection background color on active text fields; this should be set in StyleFuncs like all other style properties"`

	// true if the text has been edited relative to the original
	Edited bool `json:"-" xml:"-" desc:"true if the text has been edited relative to the original"`

	// the live text string being edited, with latest modifications -- encoded as runes
	EditTxt []rune `json:"-" xml:"-" desc:"the live text string being edited, with latest modifications -- encoded as runes"`

	// maximum width that field will request, in characters, during Size2D process -- if 0 then is 50 -- ensures that large strings don't request super large values -- standard max-width can override
	MaxWidthReq int `desc:"maximum width that field will request, in characters, during Size2D process -- if 0 then is 50 -- ensures that large strings don't request super large values -- standard max-width can override"`

	// effective size, subtracting the close widget
	EffSize mat32.Vec2 `copy:"-" json:"-" xml:"-" desc:"effective size, subtracting the close widget"`

	// starting display position in the string
	StartPos int `copy:"-" json:"-" xml:"-" desc:"starting display position in the string"`

	// ending display position in the string
	EndPos int `copy:"-" json:"-" xml:"-" desc:"ending display position in the string"`

	// current cursor position
	CursorPos int `copy:"-" json:"-" xml:"-" desc:"current cursor position"`

	// approximate number of chars that can be displayed at any time -- computed from font size etc
	CharWidth int `copy:"-" json:"-" xml:"-" desc:"approximate number of chars that can be displayed at any time -- computed from font size etc"`

	// starting position of selection in the string
	SelectStart int `copy:"-" json:"-" xml:"-" desc:"starting position of selection in the string"`

	// ending position of selection in the string
	SelectEnd int `copy:"-" json:"-" xml:"-" desc:"ending position of selection in the string"`

	// initial selection position -- where it started
	SelectInit int `copy:"-" json:"-" xml:"-" desc:"initial selection position -- where it started"`

	// if true, select text as cursor moves
	SelectMode bool `copy:"-" json:"-" xml:"-" desc:"if true, select text as cursor moves"`

	// [view: -] signal for line edit -- see TextFieldSignals for the types
	TextFieldSig ki.Signal `copy:"-" json:"-" xml:"-" view:"-" desc:"signal for line edit -- see TextFieldSignals for the types"`

	// render version of entire text, for sizing
	RenderAll girl.Text `copy:"-" json:"-" xml:"-" desc:"render version of entire text, for sizing"`

	// render version of just visible text
	RenderVis girl.Text `copy:"-" json:"-" xml:"-" desc:"render version of just visible text"`

	// normal style and focus style
	StateStyles [TextFieldStatesN]gist.Style `copy:"-" json:"-" xml:"-" desc:"normal style and focus style"`

	// font height, cached during styling
	FontHeight float32 `copy:"-" json:"-" xml:"-" desc:"font height, cached during styling"`

	// oscillates between on and off for blinking
	BlinkOn bool `copy:"-" json:"-" xml:"-" desc:"oscillates between on and off for blinking"`

	// [view: -] mutex for updating cursor between blinker and field
	CursorMu sync.Mutex `copy:"-" json:"-" xml:"-" view:"-" desc:"mutex for updating cursor between blinker and field"`

	// functions and data for textfield completion
	Complete *Complete `copy:"-" json:"-" xml:"-" desc:"functions and data for textfield completion"`

	// replace displayed characters with bullets to conceal text
	NoEcho bool `copy:"-" json:"-" xml:"-" desc:"replace displayed characters with bullets to conceal text"`
}

var TypeTextField = kit.Types.AddType(&TextField{}, TextFieldProps)

// AddNewTextField adds a new textfield to given parent node, with given name.
func AddNewTextField(parent ki.Ki, name string) *TextField {
	return parent.AddNewChild(TypeTextField, name).(*TextField)
}

func (tf *TextField) CopyFieldsFrom(frm any) {
	fr := frm.(*TextField)
	tf.PartsWidgetBase.CopyFieldsFrom(&fr.PartsWidgetBase)
	tf.Txt = fr.Txt
	tf.Placeholder = fr.Placeholder
	tf.ClearAct = fr.ClearAct
	tf.CursorWidth = fr.CursorWidth
	tf.Edited = fr.Edited
	tf.MaxWidthReq = fr.MaxWidthReq
}

func (tf *TextField) Disconnect() {
	tf.PartsWidgetBase.Disconnect()
	tf.TextFieldSig.DisconnectAll()
}

var TextFieldProps = ki.Props{
	// ki.EnumTypeFlag:    TypeNodeFlags,
	// "border-width":     units.Px(1),
	// "cursor-width":     units.Px(3),
	// "border-color":     &Prefs.Colors.Border,
	// "padding":          units.Px(4),
	// "margin":           units.Px(1),
	// "text-align":       gist.AlignLeft,
	// "color":            &Prefs.Colors.Font,
	// "background-color": &Prefs.Colors.Control,
	// "clear-act":        true,
	// "#clear": ki.Props{
	// 	"width":          units.Ex(0.5),
	// 	"height":         units.Ex(0.5),
	// 	"margin":         units.Px(0),
	// 	"padding":        units.Px(0),
	// 	"vertical-align": gist.AlignMiddle,
	// },
	// TextFieldSelectors[TextFieldActive]: ki.Props{
	// 	"background-color": "lighter-0",
	// },
	// TextFieldSelectors[TextFieldFocus]: ki.Props{
	// 	"border-width":     units.Px(2),
	// 	"background-color": "samelight-80",
	// },
	// TextFieldSelectors[TextFieldInactive]: ki.Props{
	// 	"background-color": "highlight-10",
	// },
	// TextFieldSelectors[TextFieldSel]: ki.Props{
	// 	"background-color": &Prefs.Colors.Select,
	// },
}

// TextFieldTypes is an enum containing the
// different possible types of text fields
type TextFieldTypes int

const (
	// TextFieldFilled represents a filled
	// TextField with a background color
	// and a bottom border
	TextFieldFilled TextFieldTypes = iota
	// TextFieldOutlined represents an outlined
	// TextField with a border on all sides
	// and no background color
	TextFieldOutlined

	TextFieldTypesN
)

var TypeTextFieldTypes = kit.Enums.AddEnumAltLower(TextFieldTypesN, kit.NotBitFlag, gist.StylePropProps, "TextField")

//go:generate stringer -type=TextFieldTypes

// TextFieldSignals are signals that that textfield can send
type TextFieldSignals int64

const (
	// TextFieldDone is main signal -- return or tab was pressed and the edit was
	// intentionally completed.  data is the text.
	TextFieldDone TextFieldSignals = iota

	// TextFieldDeFocused means that the user has transitioned focus away from
	// the text field due to interactions elsewhere, and any ongoing changes have been
	// applied and the editor is no longer active.  data is the text.
	// If you have a button that performs the same action as pressing enter in a textfield,
	// then pressing that button will trigger a TextFieldDeFocused event, for any active
	// edits.  Otherwise, you probably want to respond to both TextFieldDone and
	// TextFieldDeFocused as "apply" events that trigger actions associated with the field.
	TextFieldDeFocused

	// TextFieldSelected means that some text was selected (for Inactive state,
	// selection is via WidgetSig)
	TextFieldSelected

	// TextFieldCleared means the clear button was clicked
	TextFieldCleared

	// TextFieldInsert is emitted when a character is inserted into the textfield
	TextFieldInsert

	// TextFieldBackspace is emitted when a character before cursor is deleted
	TextFieldBackspace

	// TextFieldDelete is emitted when a character after cursor is deleted
	TextFieldDelete

	TextFieldSignalsN
)

//go:generate stringer -type=TextFieldSignals

// TextFieldStates are mutually-exclusive textfield states -- determines appearance
type TextFieldStates int32

const (
	// normal state -- there but not being interacted with
	TextFieldActive TextFieldStates = iota

	// inactive -- not editable
	TextFieldInactive

	// textfield is the focus -- will respond to keyboard input
	TextFieldFocus

	// selected -- for inactive state, can select entire element
	TextFieldSel

	TextFieldStatesN
)

//go:generate stringer -type=TextFieldStates

// Style selector names for the different states
var TextFieldSelectors = []string{":active", ":focus", ":inactive", ":selected"}

// these extend NodeBase NodeFlags to hold TextField state
const (
	// TextFieldFocusActive indicates that the focus is active in this field
	TextFieldFocusActive NodeFlags = NodeFlagsN + iota
)

// IsFocusActive returns true if we have active focus for keyboard input
func (tf *TextField) IsFocusActive() bool {
	return tf.HasFlag(int(TextFieldFocusActive))
}

// Text returns the current text -- applies any unapplied changes first, and
// sends a signal if so -- this is the end-user method to get the current
// value of the field.
func (tf *TextField) Text() string {
	tf.EditDone()
	return tf.Txt
}

// SetText sets the text to be edited and reverts any current edit to reflect this new text
func (tf *TextField) SetText(txt string) {
	if tf.Txt == txt && !tf.Edited {
		return
	}
	tf.StyMu.RLock()
	needSty := tf.Style.Font.Size.Val == 0
	tf.StyMu.RUnlock()
	if needSty {
		tf.StyleTextField()
	}
	tf.Txt = txt
	tf.Revert()
}

// EditDone completes editing and copies the active edited text to the text --
// called when the return key is pressed or goes out of focus
func (tf *TextField) EditDone() {
	if tf.Edited {
		tf.Edited = false
		tf.Txt = string(tf.EditTxt)
		tf.TextFieldSig.Emit(tf.This(), int64(TextFieldDone), tf.Txt)
	}
	tf.ClearSelected()
	tf.ClearCursor()
	oswin.TheApp.HideVirtualKeyboard()
}

// EditDeFocused completes editing and copies the active edited text to the text --
// called when field is made inactive due to interactions elsewhere.
func (tf *TextField) EditDeFocused() {
	if tf.Edited {
		tf.Edited = false
		tf.Txt = string(tf.EditTxt)
		tf.TextFieldSig.Emit(tf.This(), int64(TextFieldDeFocused), tf.Txt)
	}
	tf.ClearSelected()
	tf.ClearCursor()
}

// Revert aborts editing and reverts to last saved text
func (tf *TextField) Revert() {
	updt := tf.UpdateStart()
	defer tf.UpdateEnd(updt)
	tf.EditTxt = []rune(tf.Txt)
	tf.Edited = false
	tf.StartPos = 0
	tf.EndPos = tf.CharWidth
	tf.SelectReset()
}

// Clear clears any existing text
func (tf *TextField) Clear() {
	updt := tf.UpdateStart()
	defer tf.UpdateEnd(updt)
	tf.Edited = true
	tf.EditTxt = tf.EditTxt[:0]
	tf.StartPos = 0
	tf.EndPos = 0
	tf.SelectReset()
	tf.GrabFocus() // this is essential for ensuring that the clear applies after focus is lost..
	tf.TextFieldSig.Emit(tf.This(), int64(TextFieldCleared), tf.Txt)
}

//////////////////////////////////////////////////////////////////////////////////////////
//  Cursor Navigation

// CursorForward moves the cursor forward
func (tf *TextField) CursorForward(steps int) {
	updt := tf.UpdateStart()
	defer tf.UpdateEnd(updt)
	wupdt := tf.TopUpdateStart()
	defer tf.TopUpdateEnd(wupdt)
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

// CursorBackward moves the cursor backward
func (tf *TextField) CursorBackward(steps int) {
	updt := tf.UpdateStart()
	defer tf.UpdateEnd(updt)
	wupdt := tf.TopUpdateStart()
	defer tf.TopUpdateEnd(wupdt)
	tf.CursorPos -= steps
	if tf.CursorPos < 0 {
		tf.CursorPos = 0
	}
	if tf.CursorPos <= tf.StartPos {
		dec := ints.MinInt(tf.StartPos, 8)
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
	defer tf.UpdateEnd(updt)
	wupdt := tf.TopUpdateStart()
	defer tf.TopUpdateEnd(wupdt)
	tf.CursorPos = 0
	tf.StartPos = 0
	tf.EndPos = ints.MinInt(len(tf.EditTxt), tf.StartPos+tf.CharWidth)
	if tf.SelectMode {
		tf.SelectRegUpdate(tf.CursorPos)
	}
}

// CursorEnd moves the cursor to the end of the text
func (tf *TextField) CursorEnd() {
	updt := tf.UpdateStart()
	defer tf.UpdateEnd(updt)
	wupdt := tf.TopUpdateStart()
	defer tf.TopUpdateEnd(wupdt)
	ed := len(tf.EditTxt)
	tf.CursorPos = ed
	tf.EndPos = len(tf.EditTxt) // try -- display will adjust
	tf.StartPos = ints.MaxInt(0, tf.EndPos-tf.CharWidth)
	if tf.SelectMode {
		tf.SelectRegUpdate(tf.CursorPos)
	}
}

// todo: ctrl+backspace = delete word
// shift+arrow = select
// uparrow = start / down = end

// CursorBackspace deletes character(s) immediately before cursor
func (tf *TextField) CursorBackspace(steps int) {
	wupdt := tf.TopUpdateStart()
	defer tf.TopUpdateEnd(wupdt)
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
	defer tf.UpdateEnd(updt)
	tf.Edited = true
	tf.EditTxt = append(tf.EditTxt[:tf.CursorPos-steps], tf.EditTxt[tf.CursorPos:]...)
	tf.CursorBackward(steps)
	tf.TextFieldSig.Emit(tf.This(), int64(TextFieldBackspace), tf.Txt)
}

// CursorDelete deletes character(s) immediately after the cursor
func (tf *TextField) CursorDelete(steps int) {
	wupdt := tf.TopUpdateStart()
	defer tf.TopUpdateEnd(wupdt)
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
	defer tf.UpdateEnd(updt)
	tf.Edited = true
	tf.EditTxt = append(tf.EditTxt[:tf.CursorPos], tf.EditTxt[tf.CursorPos+steps:]...)
	tf.TextFieldSig.Emit(tf.This(), int64(TextFieldDelete), tf.Txt)
}

// CursorKill deletes text from cursor to end of text
func (tf *TextField) CursorKill() {
	wupdt := tf.TopUpdateStart()
	defer tf.TopUpdateEnd(wupdt)
	steps := len(tf.EditTxt) - tf.CursorPos
	tf.CursorDelete(steps)
}

///////////////////////////////////////////////////////////////////////////////
//    Selection

// ClearSelected resets both the global selected flag and any current selection
func (tf *TextField) ClearSelected() {
	tf.WidgetBase.ClearSelected()
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
	tf.UpdateEnd(updt)
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
	defer tf.UpdateEnd(updt)
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
	tf.UpdateEnd(updt)
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
	wupdt := tf.TopUpdateStart()
	defer tf.TopUpdateEnd(wupdt)
	cut := tf.DeleteSelection()
	if cut != "" {
		oswin.TheApp.ClipBoard(tf.ParentWindow().OSWin).Write(mimedata.NewText(cut))
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
	defer tf.UpdateEnd(updt)
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
	wupdt := tf.TopUpdateStart()
	defer tf.TopUpdateEnd(wupdt)
	tf.SelectUpdate()
	if !tf.HasSelection() {
		return
	}
	md := mimedata.NewMimes(0, 1)
	tf.This().(Clipper).MimeData(&md)
	oswin.TheApp.ClipBoard(tf.ParentWindow().OSWin).Write(md)
	if reset {
		tf.SelectReset()
	}
}

// Paste inserts text from the clipboard at current cursor position -- if
// cursor is within a current selection, that selection is replaced.
// Satisfies Clipper interface -- can be extended in subtypes.
func (tf *TextField) Paste() {
	wupdt := tf.TopUpdateStart()
	defer tf.TopUpdateEnd(wupdt)
	data := oswin.TheApp.ClipBoard(tf.ParentWindow().OSWin).Read([]string{filecat.TextPlain})
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
	defer tf.UpdateEnd(updt)
	wupdt := tf.TopUpdateStart()
	defer tf.TopUpdateEnd(wupdt)
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
	tf.TextFieldSig.Emit(tf.This(), int64(TextFieldInsert), tf.EditTxt)
}

func (tf *TextField) MakeContextMenu(m *Menu) {
	cpsc := ActiveKeyMap.ChordForFun(KeyFunCopy)
	ac := m.AddAction(ActOpts{Label: "Copy", Shortcut: cpsc},
		tf.This(), func(recv, send ki.Ki, sig int64, data any) {
			tff := recv.Embed(TypeTextField).(*TextField)
			tff.This().(Clipper).Copy(true)
		})
	ac.SetActiveState(tf.HasSelection())
	if !tf.IsInactive() {
		ctsc := ActiveKeyMap.ChordForFun(KeyFunCut)
		ptsc := ActiveKeyMap.ChordForFun(KeyFunPaste)
		ac = m.AddAction(ActOpts{Label: "Cut", Shortcut: ctsc},
			tf.This(), func(recv, send ki.Ki, sig int64, data any) {
				tff := recv.Embed(TypeTextField).(*TextField)
				tff.This().(Clipper).Cut()
			})
		ac.SetActiveState(tf.HasSelection())
		ac = m.AddAction(ActOpts{Label: "Paste", Shortcut: ptsc},
			tf.This(), func(recv, send ki.Ki, sig int64, data any) {
				tff := recv.Embed(TypeTextField).(*TextField)
				tff.This().(Clipper).Paste()
			})
		ac.SetInactiveState(oswin.TheApp.ClipBoard(tf.ParentWindow().OSWin).IsEmpty())
	}
}

///////////////////////////////////////////////////////////////////////////////
//    Complete

// SetCompleter sets completion functions so that completions will
// automatically be offered as the user types
func (tf *TextField) SetCompleter(data any, matchFun complete.MatchFunc, editFun complete.EditFunc) {
	if matchFun == nil || editFun == nil {
		if tf.Complete != nil {
			tf.Complete.CompleteSig.Disconnect(tf.This())
			tf.Complete.Destroy()
		}
		tf.Complete = nil
		return
	}
	tf.Complete = &Complete{}
	tf.Complete.InitName(tf.Complete, "tf-completion") // needed for standalone Ki's
	tf.Complete.Context = data
	tf.Complete.MatchFunc = matchFun
	tf.Complete.EditFunc = editFun
	// note: only need to connect once..
	tf.Complete.CompleteSig.ConnectOnly(tf.This(), func(recv, send ki.Ki, sig int64, data any) {
		tff, _ := recv.Embed(TypeTextField).(*TextField)
		if sig == int64(CompleteSelect) {
			tff.CompleteText(data.(string)) // always use data
		} else if sig == int64(CompleteExtend) {
			tff.CompleteExtend(data.(string)) // always use data
		}
	})
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
	tf.Complete.Show(s, 0, tf.CursorPos, tf.ViewportSafe(), cpos, forceComplete)
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
	st := &tf.Style
	spc := st.BoxSpace()
	pos := tf.LayState.Alloc.Pos.Add(spc.Pos())
	if wincoords {
		mvp := tf.ViewportSafe()
		mvp.BBoxMu.RLock()
		pos = pos.Add(mat32.NewVec2FmPoint(mvp.WinBBox.Min))
		mvp.BBoxMu.RUnlock()
	}
	cpos := tf.TextWidth(tf.StartPos, charidx)
	return mat32.Vec2{pos.X + cpos, pos.Y}
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
		if BlinkingTextField == nil || BlinkingTextField.This() == nil {
			TextFieldBlinkMu.Unlock()
			continue
		}
		if BlinkingTextField.IsDestroyed() || BlinkingTextField.IsDeleted() {
			BlinkingTextField = nil
			TextFieldBlinkMu.Unlock()
			continue
		}
		tf := BlinkingTextField
		if tf.Viewport == nil || !tf.HasFocus() || !tf.IsFocusActive() || !tf.This().(Node2D).IsVisible() {
			BlinkingTextField = nil
			TextFieldBlinkMu.Unlock()
			continue
		}
		win := tf.ParentWindow()
		if win == nil || win.IsResizing() || win.IsClosed() || !win.IsWindowInFocus() {
			TextFieldBlinkMu.Unlock()
			continue
		}
		if win.IsUpdating() {
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
	if !tf.This().(Node2D).IsVisible() {
		return
	}
	tf.BlinkOn = true
	if CursorBlinkMSec == 0 {
		tf.RenderCursor(true)
		return
	}
	TextFieldBlinkMu.Lock()
	if TextFieldBlinker == nil {
		TextFieldBlinker = time.NewTicker(time.Duration(CursorBlinkMSec) * time.Millisecond)
		go TextFieldBlink()
	}
	tf.BlinkOn = true
	win := tf.ParentWindow()
	if win != nil && !win.IsResizing() {
		tf.RenderCursor(true)
	}
	BlinkingTextField = tf
	TextFieldBlinkMu.Unlock()
}

// ClearCursor turns off cursor and stops it from blinking
func (tf *TextField) ClearCursor() {
	if tf.IsInactive() {
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
	if !tf.This().(Node2D).IsVisible() {
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
	if !tf.This().(Node2D).IsVisible() {
		return
	}

	tf.CursorMu.Lock()
	defer tf.CursorMu.Unlock()

	win := tf.ParentWindow()
	sp := tf.CursorSprite()
	if on {
		win.ActivateSprite(sp.Name)
	} else {
		win.InactivateSprite(sp.Name)
	}
	sp.Geom.Pos = tf.CharStartPos(tf.CursorPos, true).ToPointFloor()
	win.UpdateSig()
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

// CursorSprite returns the Sprite for the cursor (which is
// only rendered once with a vertical bar, and just activated and inactivated
// depending on render status)
func (tf *TextField) CursorSprite() *Sprite {
	win := tf.ParentWindow()
	if win == nil {
		return nil
	}
	sty := &tf.Style
	spnm := fmt.Sprintf("%v-%v", TextFieldSpriteName, tf.FontHeight)
	sp, ok := win.SpriteByName(spnm)
	// TODO: figure out how to update caret color on color scheme change
	if !ok {
		bbsz := image.Point{int(mat32.Ceil(tf.CursorWidth.Dots)), int(mat32.Ceil(tf.FontHeight))}
		if bbsz.X < 2 { // at least 2
			bbsz.X = 2
		}
		// bbsz.X += 2 // inverse border
		sp = NewSprite(spnm, bbsz, image.Point{})
		ibox := sp.Pixels.Bounds()
		// draw.Draw(sp.Pixels, ibox, &image.Uniform{sty.Color.Inverse()}, image.Point{}, draw.Src)
		// ibox.Min.X++ // 1 pixel boundary
		// ibox.Max.X--
		draw.Draw(sp.Pixels, ibox, &image.Uniform{sty.Color}, image.Point{}, draw.Src)
		win.AddSprite(sp)
	}
	return sp
}

// RenderSelect renders the selected region, if any, underneath the text
func (tf *TextField) RenderSelect() {
	if !tf.HasSelection() {
		return
	}
	effst := ints.MaxInt(tf.StartPos, tf.SelectStart)
	if effst >= tf.EndPos {
		return
	}
	effed := ints.MinInt(tf.EndPos, tf.SelectEnd)
	if effed < tf.StartPos {
		return
	}
	if effed <= effst {
		return
	}

	spos := tf.CharStartPos(effst, false)

	rs := &tf.Viewport.Render
	pc := &rs.Paint
	// st := &tf.StateStyles[TextFieldSel]
	// tf.State = TextFieldSel
	// tf.RunStyleFuncs()
	tsz := tf.TextWidth(effst, effed)
	pc.FillBox(rs, spos, mat32.NewVec2(tsz, tf.FontHeight), &tf.SelectColor)
}

// AutoScroll scrolls the starting position to keep the cursor visible
func (tf *TextField) AutoScroll() {
	st := &tf.Style

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
	tf.CharWidth = int(maxw / st.UnContext.ToDotsFactor(units.UnitCh)) // rough guess in chars

	// first rationalize all the values
	if tf.EndPos == 0 || tf.EndPos > sz { // not init
		tf.EndPos = sz
	}
	if tf.StartPos >= tf.EndPos {
		tf.StartPos = ints.MaxInt(0, tf.EndPos-tf.CharWidth)
	}
	tf.CursorPos = mat32.ClampInt(tf.CursorPos, 0, sz)

	inc := int(mat32.Ceil(.1 * float32(tf.CharWidth)))
	inc = ints.MaxInt(4, inc)

	// keep cursor in view with buffer
	startIsAnchor := true
	if tf.CursorPos < (tf.StartPos + inc) {
		tf.StartPos -= inc
		tf.StartPos = ints.MaxInt(tf.StartPos, 0)
		tf.EndPos = tf.StartPos + tf.CharWidth
		tf.EndPos = ints.MinInt(sz, tf.EndPos)
	} else if tf.CursorPos > (tf.EndPos - inc) {
		tf.EndPos += inc
		tf.EndPos = ints.MinInt(tf.EndPos, sz)
		tf.StartPos = tf.EndPos - tf.CharWidth
		tf.StartPos = ints.MaxInt(0, tf.StartPos)
		startIsAnchor = false
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
	st := &tf.Style

	spc := st.BoxSpace()
	px := pixOff - spc.Pos().X

	if px <= 0 {
		return tf.StartPos
	}

	// for selection to work correctly, we need this to be deterministic

	sz := len(tf.EditTxt)
	c := tf.StartPos + int(float64(px/st.UnContext.ToDotsFactor(units.UnitCh)))
	c = ints.MinInt(c, sz)

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
// WinBBox of text field, and sets current cursor to it, updating selection as
// well
func (tf *TextField) SetCursorFromPixel(pixOff float32, selMode mouse.SelectModes) {
	if tf.ParentWindow() == nil {
		return
	}
	updt := tf.UpdateStart()
	defer tf.UpdateEnd(updt)
	wupdt := tf.TopUpdateStart()
	defer tf.TopUpdateEnd(wupdt)
	oldPos := tf.CursorPos
	tf.CursorPos = tf.PixelToCursor(pixOff)
	if tf.SelectMode || selMode != mouse.SelectOne {
		if !tf.SelectMode && selMode != mouse.SelectOne {
			tf.SelectStart = oldPos
			tf.SelectMode = true
		}
		if !tf.IsDragging() && selMode == mouse.SelectOne { // && tf.CursorPos >= tf.SelectStart && tf.CursorPos < tf.SelectEnd {
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
//    KeyInput handling

// KeyInput handles keyboard input into the text field and from the completion menu
func (tf *TextField) KeyInput(kt *key.ChordEvent) {
	if KeyEventTrace {
		fmt.Printf("TextField KeyInput: %v\n", tf.Path())
	}
	kf := KeyFun(kt.Chord())
	win := tf.ParentWindow()

	if tf.Complete != nil {
		cpop := win.CurPopup()
		if PopupIsCompleter(cpop) {
			tf.Complete.KeyInput(kf)
		}
	}

	if !tf.IsFocusActive() && kf == KeyFunAbort {
		return
	}

	// first all the keys that work for both inactive and active
	switch kf {
	case KeyFunMoveRight:
		kt.SetProcessed()
		tf.CursorForward(1)
		tf.OfferComplete(dontForce)
	case KeyFunMoveLeft:
		kt.SetProcessed()
		tf.CursorBackward(1)
		tf.OfferComplete(dontForce)
	case KeyFunHome:
		kt.SetProcessed()
		tf.CancelComplete()
		tf.CursorStart()
	case KeyFunEnd:
		kt.SetProcessed()
		tf.CancelComplete()
		tf.CursorEnd()
	case KeyFunSelectMode:
		kt.SetProcessed()
		tf.CancelComplete()
		tf.SelectModeToggle()
	case KeyFunCancelSelect:
		kt.SetProcessed()
		tf.CancelComplete()
		tf.SelectReset()
	case KeyFunSelectAll:
		kt.SetProcessed()
		tf.CancelComplete()
		tf.SelectAll()
	case KeyFunCopy:
		kt.SetProcessed()
		tf.CancelComplete()
		tf.This().(Clipper).Copy(true) // reset
	}
	if tf.IsInactive() || kt.IsProcessed() {
		return
	}
	switch kf {
	case KeyFunEnter:
		fallthrough
	case KeyFunFocusNext: // we process tab to make it EditDone as opposed to other ways of losing focus
		fallthrough
	case KeyFunAccept: // ctrl+enter
		kt.SetProcessed()
		tf.CancelComplete()
		tf.EditDone()
		tf.FocusNext()
	case KeyFunFocusPrev:
		kt.SetProcessed()
		tf.CancelComplete()
		tf.EditDone()
		tf.FocusPrev()
	case KeyFunAbort: // esc
		kt.SetProcessed()
		tf.CancelComplete()
		tf.Revert()
		tf.FocusChanged2D(FocusInactive)
	case KeyFunBackspace:
		kt.SetProcessed()
		tf.CursorBackspace(1)
		tf.OfferComplete(dontForce)
	case KeyFunKill:
		kt.SetProcessed()
		tf.CancelComplete()
		tf.CursorKill()
	case KeyFunDelete:
		kt.SetProcessed()
		tf.CursorDelete(1)
	case KeyFunCut:
		kt.SetProcessed()
		tf.CancelComplete()
		tf.This().(Clipper).Cut()
	case KeyFunPaste:
		kt.SetProcessed()
		tf.CancelComplete()
		tf.This().(Clipper).Paste()
	case KeyFunComplete:
		kt.SetProcessed()
		tf.OfferComplete(force)
	case KeyFunNil:
		if unicode.IsPrint(kt.Rune) {
			if !kt.HasAnyModifier(key.Control, key.Meta) {
				kt.SetProcessed()
				tf.InsertAtCursor(string(kt.Rune))
				if kt.Rune == ' ' {
					tf.CancelComplete()
				} else {
					tf.OfferComplete(dontForce)
				}
			}
		}
	}
}

// HandleMouseEvent handles the mouse.Event
func (tf *TextField) HandleMouseEvent(me *mouse.Event) {
	if tf.ParentWindow() == nil {
		return
	}
	if !tf.IsInactive() && !tf.HasFocus() {
		tf.GrabFocus()
	}
	me.SetProcessed()
	switch me.Button {
	case mouse.Left:
		if me.Action == mouse.Press {
			if tf.IsInactive() {
				tf.SetSelectedState(!tf.IsSelected())
				tf.EmitSelectedSignal()
				tf.UpdateSig()
			} else {
				pt := tf.PointToRelPos(me.Pos())
				tf.SetCursorFromPixel(float32(pt.X), me.SelectMode())
			}
		} else if me.Action == mouse.DoubleClick {
			me.SetProcessed()
			if tf.HasSelection() {
				if tf.SelectStart == 0 && tf.SelectEnd == len(tf.EditTxt) {
					tf.SelectReset()
				} else {
					tf.SelectAll()
				}
			} else {
				tf.SelectWord()
			}
		}
	case mouse.Middle:
		if !tf.IsInactive() && me.Action == mouse.Press {
			me.SetProcessed()
			pt := tf.PointToRelPos(me.Pos())
			tf.SetCursorFromPixel(float32(pt.X), me.SelectMode())
			tf.Paste()
		}
	case mouse.Right:
		if me.Action == mouse.Press {
			me.SetProcessed()
			tf.EmitContextMenuSignal()
			tf.This().(Node2D).ContextMenu()
		}
	}
}

func (tf *TextField) MouseDragEvent() {
	tf.ConnectEvent(oswin.MouseDragEvent, RegPri, func(recv, send ki.Ki, sig int64, d any) {
		me := d.(*mouse.DragEvent)
		me.SetProcessed()
		tff := recv.Embed(TypeTextField).(*TextField)
		if !tff.SelectMode {
			tff.SelectModeToggle()
		}
		pt := tff.PointToRelPos(me.Pos())
		tff.SetCursorFromPixel(float32(pt.X), mouse.SelectOne)
	})
}

func (tf *TextField) MouseEvent() {
	tf.ConnectEvent(oswin.MouseEvent, RegPri, func(recv, send ki.Ki, sig int64, d any) {
		tff := recv.Embed(TypeTextField).(*TextField)
		me := d.(*mouse.Event)
		tff.HandleMouseEvent(me)
	})
}

func (tf *TextField) MouseFocusEvent() {
	tf.ConnectEvent(oswin.MouseFocusEvent, RegPri, func(recv, send ki.Ki, sig int64, d any) {
		tff := recv.Embed(TypeTextField).(*TextField)
		if tff.IsInactive() {
			return
		}
		me := d.(*mouse.FocusEvent)
		me.SetProcessed()
		if me.Action == mouse.Enter {
			oswin.TheApp.Cursor(tf.ParentWindow().OSWin).PushIfNot(cursor.IBeam)
		} else {
			oswin.TheApp.Cursor(tf.ParentWindow().OSWin).PopIf(cursor.IBeam)
		}
	})
}

func (tf *TextField) KeyChordEvent() {
	tf.ConnectEvent(oswin.KeyChordEvent, RegPri, func(recv, send ki.Ki, sig int64, d any) {
		tff := recv.Embed(TypeTextField).(*TextField)
		kt := d.(*key.ChordEvent)
		tff.KeyInput(kt)
	})
	if dlg, ok := tf.Viewport.This().(*Dialog); ok {
		dlg.DialogSig.Connect(tf.This(), func(recv, send ki.Ki, sig int64, data any) {
			tff, _ := recv.Embed(TypeTextField).(*TextField)
			if sig == int64(DialogAccepted) {
				tff.EditDone()
			}
		})
	}
}

func (tf *TextField) TextFieldEvents() {
	tf.HoverTooltipEvent()
	tf.MouseDragEvent()
	tf.MouseEvent()
	tf.MouseFocusEvent()
	tf.KeyChordEvent()
}

func (tf *TextField) ConfigParts() {
	// fmt.Println("tf config parts")
	tf.Parts.Lay = LayoutHoriz
	if !tf.ClearAct || tf.IsInactive() {
		tf.Parts.DeleteChildren(ki.DestroyKids)
		return
	}
	config := kit.TypeAndNameList{}
	config.Add(TypeStretch, "clr-str")
	config.Add(TypeAction, "clear")
	mods, updt := tf.Parts.ConfigChildren(config)
	// fmt.Println("configured children")
	if mods || gist.RebuildDefaultStyles {
		clr := tf.Parts.Child(1).(*Action)
		tf.StylePart(Node2D(clr))
		clr.SetIcon(icons.Close)
		clr.SetProp("no-focus", true)
		clr.ActionSig.ConnectOnly(tf.This(), func(recv, send ki.Ki, sig int64, data any) {
			tff := recv.Embed(TypeTextField).(*TextField)
			if tff != nil {
				tff.Clear()
			}
		})
		tf.UpdateEnd(updt)
	}
}

////////////////////////////////////////////////////
//  Node2D Interface

func (tf *TextField) Init2D() {
	tf.Init2DWidget()
	tf.EditTxt = []rune(tf.Txt)
	tf.Edited = false
	tf.ClearAct = true
	tf.ConfigParts()
	tf.ConfigStyles()
}

// StyleTextField does text field styling -- sets StyMu Lock
func (tf *TextField) StyleTextField() {
	// fmt.Println("tf style")
	tf.StyMu.Lock()

	tf.SetCanFocusIfActive()
	hasTempl, saveTempl := tf.Style.FromTemplate()
	if !hasTempl || saveTempl {
		tf.Style2DWidget()
	}
	if hasTempl && saveTempl {
		tf.Style.SaveTemplate()
	}
	pst := &(tf.Par.(Node2D).AsWidget().Style)
	if hasTempl && !saveTempl {
		for i := 0; i < int(TextFieldStatesN); i++ {
			tf.StateStyles[i].Template = tf.Style.Template + TextFieldSelectors[i]
			tf.StateStyles[i].FromTemplate()
		}
	} else {
		for i := 0; i < int(TextFieldStatesN); i++ {
			tf.StateStyles[i].CopyFrom(&tf.Style)
			tf.StateStyles[i].SetStyleProps(pst, tf.StyleProps(TextFieldSelectors[i]), tf.Viewport)
			StyleCSS(tf.This().(Node2D), tf.Viewport, &tf.StateStyles[i], tf.CSSAgg, TextFieldSelectors[i])
			tf.StateStyles[i].CopyUnitContext(&tf.Style.UnContext)
		}
	}
	if hasTempl && saveTempl {
		for i := 0; i < int(TextFieldStatesN); i++ {
			tf.StateStyles[i].Template = tf.Style.Template + TextFieldSelectors[i]
			tf.StateStyles[i].SaveTemplate()
		}
	}
	tf.CursorWidth.SetFmInheritProp("cursor-width", tf.This(), ki.Inherit, ki.TypeProps) // get type defaults
	tf.CursorWidth.ToDots(&tf.Style.UnContext)
	if pv, ok := tf.PropInherit("clear-act", ki.Inherit, ki.TypeProps); ok {
		tf.ClearAct, _ = kit.ToBool(pv)
	}
	tf.StyMu.Unlock()
	tf.ConfigParts()
}

func (tf *TextField) Style2D() {
	tf.StyleTextField()
	tf.StyMu.Lock()
	tf.LayState.SetFromStyle(&tf.Style) // also does reset
	tf.StyMu.Unlock()
}

func (tf *TextField) UpdateRenderAll() bool {
	st := &tf.Style
	st.Font = girl.OpenFont(st.FontRender(), &st.UnContext)
	txt := tf.EditTxt
	if tf.NoEcho {
		txt = concealDots(len(tf.EditTxt))
	}
	tf.RenderAll.SetRunes(txt, st.FontRender(), &st.UnContext, &st.Text, true, 0, 0)
	return true
}

func (tf *TextField) Size2D(iter int) {
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
	tf.EndPos = ints.MinInt(len(tf.EditTxt), maxlen)
	tf.UpdateRenderAll()
	tf.FontHeight = tf.RenderAll.Size.Y
	w := tf.TextWidth(tf.StartPos, tf.EndPos)
	w += 2.0 // give some extra buffer
	// fmt.Printf("fontheight: %v width: %v\n", tf.FontHeight, w)
	tf.Size2DFromWH(w, tf.FontHeight)
	tf.EditTxt = tmptxt
}

func (tf *TextField) Layout2D(parBBox image.Rectangle, iter int) bool {
	tf.Layout2DBase(parBBox, true, iter) // init style
	tf.Layout2DParts(parBBox, iter)
	for i := 0; i < int(TextFieldStatesN); i++ {
		tf.StateStyles[i].CopyUnitContext(&tf.Style.UnContext)
	}
	redo := tf.Layout2DChildren(iter)
	sz := tf.LayState.Alloc.Size
	if tf.ClearAct && len(*tf.Parts.Children()) == 2 {
		clr := tf.Parts.Child(1).(*Action)
		sz.X -= clr.LayState.Alloc.Size.X
	}
	tf.EffSize = sz
	return redo
}

func (tf *TextField) RenderTextField() {
	rs, _, _ := tf.RenderLock()
	defer tf.RenderUnlock(rs)

	tf.AutoScroll() // inits paint with our style
	prevState := tf.State
	if tf.IsInactive() {
		if tf.IsSelected() {
			tf.State = TextFieldSel
		} else {
			tf.State = TextFieldInactive
		}
	} else if tf.HasFocus() {
		if tf.IsFocusActive() {
			tf.State = TextFieldFocus
		} else {
			tf.State = TextFieldActive
		}
	} else if tf.IsSelected() {
		tf.State = TextFieldSel
	} else {
		tf.State = TextFieldActive
	}
	if tf.State != prevState {
		tf.Style = tf.StateStyles[tf.State]
		tf.Style2DWidget()
	}
	st := &tf.Style
	st.Font = girl.OpenFont(st.FontRender(), &st.UnContext)
	tf.RenderStdBox(st)
	cur := tf.EditTxt[tf.StartPos:tf.EndPos]
	tf.RenderSelect()
	pos := tf.LayState.Alloc.Pos.Add(st.BoxSpace().Pos())
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

func (tf *TextField) Render2D() {
	if tf.HasFocus() && tf.IsFocusActive() && BlinkingTextField == tf {
		tf.ScrollLayoutToCursor()
	}
	if tf.FullReRenderIfNeeded() {
		return
	}
	if tf.PushBounds() {
		tf.This().(Node2D).ConnectEvents2D()
		tf.RenderTextField()
		if tf.IsActive() {
			if tf.HasFocus() && tf.IsFocusActive() {
				tf.StartCursor()
			} else {
				tf.StopCursor()
			}
		}
		tf.Render2DParts()
		tf.Render2DChildren()
		tf.PopBounds()
	} else {
		tf.DisconnectAllEvents(RegPri)
	}
}

func (tf *TextField) ConnectEvents2D() {
	tf.TextFieldEvents()
}

func (tf *TextField) FocusChanged2D(change FocusChanges) {
	switch change {
	case FocusLost:
		tf.ClearFlag(int(TextFieldFocusActive))
		tf.EditDone()
		tf.UpdateSig()
	case FocusGot:
		tf.SetFlag(int(TextFieldFocusActive))
		tf.ScrollToMe()
		// tf.CursorEnd()
		tf.EmitFocusedSignal()
		tf.UpdateSig()
		if _, ok := tf.Parent().Parent().(*SpinBox); ok {
			oswin.TheApp.ShowVirtualKeyboard(oswin.NumberKeyboard)
		} else {
			oswin.TheApp.ShowVirtualKeyboard(oswin.SingleLineKeyboard)
		}
	case FocusInactive:
		tf.ClearFlag(int(TextFieldFocusActive))
		tf.EditDeFocused()
		tf.UpdateSig()
		oswin.TheApp.HideVirtualKeyboard()
	case FocusActive:
		tf.SetFlag(int(TextFieldFocusActive))
		tf.ScrollToMe()
		if _, ok := tf.Parent().Parent().(*SpinBox); ok {
			oswin.TheApp.ShowVirtualKeyboard(oswin.NumberKeyboard)
		} else {
			oswin.TheApp.ShowVirtualKeyboard(oswin.SingleLineKeyboard)
		}
		// tf.UpdateSig()
		// todo: see about cursor
	}
}

func (tf *TextField) ConfigStyles() {
	tf.AddStyleFunc(StyleFuncDefault, func() {
		tf.Style.MinWidth.SetEm(20)
		tf.Style.Border.Radius.Set(units.Px(4))
		tf.CursorWidth.SetPx(1)
		tf.Style.Margin.Set(units.Px(1 * Prefs.DensityMul()))
		tf.Style.Padding.Set(units.Px(4 * Prefs.DensityMul()))
		tf.Style.Text.Align = gist.AlignLeft
		tf.Style.Color = ColorScheme.OnBackground
		tf.SelectColor.SetColor(ColorScheme.Tertiary)
		tf.ClearAct = true
		switch tf.Type {
		case TextFieldFilled:
			tf.Style.Border.Style.Set(gist.BorderNone)
			tf.Style.BackgroundColor.SetColor(ColorScheme.Background.Highlight(10))
		case TextFieldOutlined:
			tf.Style.Border.Style.Set(gist.BorderSolid)
			tf.Style.Border.Width.Set(units.Px(1))
			tf.Style.Border.Color.Set(ColorScheme.OnBackground)
			tf.Style.BackgroundColor.SetColor(ColorScheme.Background)
		}
		switch tf.State {
		case TextFieldActive:
			// use background as already specified above
		case TextFieldInactive:
			tf.Style.BackgroundColor.SetColor(tf.Style.BackgroundColor.Color.Highlight(20))
			tf.Style.Color = ColorScheme.OnBackground.Highlight(20)
		case TextFieldFocus:
			tf.Style.BackgroundColor.SetColor(tf.Style.BackgroundColor.Color.Highlight(10))
		case TextFieldSel:
			tf.Style.BackgroundColor.SetColor(ColorScheme.Tertiary)
		}
		tf.PlaceholderColor = tf.Style.Color.Highlight(40)
	})
	tf.Parts.AddChildStyleFunc("clear", 1, StyleFuncParts(tf), func(clr *WidgetBase) {
		clr.Style.Width.SetEx(0.5)
		clr.Style.Height.SetEx(0.5)
		clr.Style.Margin.Set()
		clr.Style.Padding.Set()
		clr.Style.AlignV = gist.AlignMiddle
		clr.Style.BackgroundColor = tf.Style.BackgroundColor
	})
}

// concealDots creates an n-length []rune of bullet characters.
func concealDots(n int) []rune {
	dots := make([]rune, n)
	for i := range dots {
		dots[i] = 0x2022 // bullet character •
	}
	return dots
}
