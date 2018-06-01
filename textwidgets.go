// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"fmt"
	"image"
	"image/color"
	"log"
	"reflect"
	"sort"
	"strconv"
	"unicode"
	"unicode/utf8"

	"github.com/chewxy/math32"
	"github.com/goki/gi/oswin"
	"github.com/goki/gi/oswin/key"
	"github.com/goki/gi/oswin/mouse"
	"github.com/goki/gi/units"
	"github.com/goki/ki"
	"github.com/goki/ki/bitflag"
	"github.com/goki/ki/kit"
)

////////////////////////////////////////////////////////////////////////////////////////
// Labeler Interface and ToLabel method

// the labeler interface provides a GUI-appropriate label (todo: rich text
// html tags!?) for an item -- use ToLabel converter to attempt to use this
// interface and then fall back on Stringer via kit.ToString conversion
// function
type Labeler interface {
	Label() string
}

// ToLabel returns the gui-appropriate label for an item, using the Labeler
// interface if it is defined, and falling back on kit.ToString converter
// otherwise -- also contains label impls for basic interface types for which
// we cannot easily define the Labeler interface
func ToLabel(it interface{}) string {
	lbler, ok := it.(Labeler)
	if !ok {
		// typ := reflect.TypeOf(it)
		// if kit.EmbeddedTypeImplements(typ, reflect.TypeOf((*reflect.Type)(nil)).Elem()) {
		// 	to, ok :=
		// }
		switch v := it.(type) {
		case reflect.Type:
			return v.Name()
		}
		return kit.ToString(it)
	}
	return lbler.Label()
}

////////////////////////////////////////////////////////////////////////////////////////
// Label

// Label is a widget for rendering text labels -- supports full widget model
// including box rendering
type Label struct {
	WidgetBase
	Text string `xml:"text" desc:"label to display"`
}

var KiT_Label = kit.Types.AddType(&Label{}, LabelProps)

var LabelProps = ki.Props{
	"padding":          units.NewValue(2, units.Px),
	"margin":           units.NewValue(2, units.Px),
	"vertical-align":   AlignTop,
	"background-color": color.Transparent,
}

func (g *Label) Style2D() {
	g.Style2DWidget()
}

func (g *Label) Size2D() {
	g.InitLayout2D()
	g.Size2DFromText(g.Text)
}

func (g *Label) Render2D() {
	if g.PushBounds() {
		st := &g.Style
		g.RenderStdBox(st)
		g.Render2DText(g.Text)
		g.Render2DChildren()
		g.PopBounds()
	}
}

////////////////////////////////////////////////////////////////////////////////////////
// TextField

// signals that buttons can send
type TextFieldSignals int64

const (
	// main signal -- return was pressed and an edit was completed -- data is the text
	TextFieldDone TextFieldSignals = iota

	// some text was selected or for Inactive state, entire field was selected
	TextFieldSelected

	TextFieldSignalsN
)

//go:generate stringer -type=TextFieldSignals

// mutually-exclusive textfield states -- determines appearance
type TextFieldStates int32

const (
	// normal state -- there but not being interacted with
	TextFieldActive TextFieldStates = iota

	// textfield is the focus -- will respond to keyboard input
	TextFieldFocus

	// inactive -- not editable
	TextFieldInactive

	// selected -- for inactive state, can select entire element
	TextFieldSelect

	TextFieldStatesN
)

//go:generate stringer -type=TextFieldStates

// Style selector names for the different states
var TextFieldSelectors = []string{":active", ":focus", ":inactive", ":selected"}

// TextField is a widget for editing a line of text
type TextField struct {
	WidgetBase
	Text          string                  `json:"-" xml:"text" desc:"the last saved value of the text string being edited"`
	Edited        bool                    `json:"-" xml:"-" desc:"true if the text has been edited relative to the original"`
	EditText      []rune                  `json:"-" xml:"-" desc:"the live text string being edited, with latest modifications -- encoded as runes"`
	StartPos      int                     `xml:"-" desc:"starting display position in the string"`
	EndPos        int                     `xml:"-" desc:"ending display position in the string"`
	CursorPos     int                     `xml:"-" desc:"current cursor position"`
	CharWidth     int                     `xml:"-" desc:"approximate number of chars that can be displayed at any time -- computed from font size etc"`
	SelectStart   int                     `xml:"-" desc:"starting position of selection in the string"`
	SelectEnd     int                     `xml:"-" desc:"ending position of selection in the string"`
	SelectMode    bool                    `xml:"-" desc:"if true, select text as cursor moves"`
	Selected      bool                    `xml:"-" desc:"entire field is selected, for Inactive mode"`
	TextFieldSig  ki.Signal               `json:"-" xml:"-" desc:"signal for line edit -- see TextFieldSignals for the types"`
	StateStyles   [TextFieldStatesN]Style `json:"-" xml:"-" desc:"normal style and focus style"`
	CharPos       []float32               `json:"-" xml:"-" desc:"character positions, for point just AFTER the given character -- todo there are likely issues with runes here -- need to test.."`
	lastSizedText []rune                  `json:"-" xml:"-" the last text string we got charpos for"`
}

var KiT_TextField = kit.Types.AddType(&TextField{}, TextFieldProps)

var TextFieldProps = ki.Props{
	"border-width":                      units.NewValue(1, units.Px),
	"border-color":                      &Prefs.BorderColor,
	"border-style":                      BorderSolid,
	"padding":                           units.NewValue(4, units.Px),
	"margin":                            units.NewValue(1, units.Px),
	"text-align":                        AlignLeft,
	"vertical-align":                    AlignTop,
	"background-color":                  &Prefs.ControlColor,
	TextFieldSelectors[TextFieldActive]: ki.Props{},
	TextFieldSelectors[TextFieldFocus]: ki.Props{
		"border-width":     units.NewValue(2, units.Px),
		"background-color": "lighter-80",
	},
	TextFieldSelectors[TextFieldInactive]: ki.Props{
		"background-color": "darker-20",
	},
	TextFieldSelectors[TextFieldSelect]: ki.Props{
		"background-color": &Prefs.SelectColor,
	},
}

// SetText sets the text to be edited and reverts any current edit to reflect this new text
func (tf *TextField) SetText(txt string) {
	if tf.Text == txt && !tf.Edited {
		return
	}
	tf.Text = txt
	tf.RevertEdit()
}

// EditDone completes editing and copies the active edited text to the text --
// called when the return key is pressed or goes out of focus
func (tf *TextField) EditDone() {
	if tf.Edited {
		tf.Text = string(tf.EditText)
		tf.TextFieldSig.Emit(tf.This, int64(TextFieldDone), tf.Text)
		tf.Edited = false
	}
}

// RevertEdit aborts editing and reverts to last saved text
func (tf *TextField) RevertEdit() {
	updt := tf.UpdateStart()
	tf.EditText = []rune(tf.Text)
	tf.Edited = false
	tf.StartPos = 0
	tf.EndPos = tf.CharWidth
	tf.SelectReset()
	tf.UpdateEnd(updt)
}

// CursorForward moves the cursor forward
func (tf *TextField) CursorForward(steps int) {
	updt := tf.UpdateStart()
	tf.CursorPos += steps
	if tf.CursorPos > len(tf.EditText) {
		tf.CursorPos = len(tf.EditText)
	}
	if tf.CursorPos > tf.EndPos {
		inc := tf.CursorPos - tf.EndPos
		tf.EndPos += inc
	}
	if tf.SelectMode {
		if tf.CursorPos-steps < tf.SelectStart {
			tf.SelectStart = tf.CursorPos
		} else if tf.CursorPos > tf.SelectStart {
			tf.SelectEnd = tf.CursorPos
		} else {
			tf.SelectStart = tf.CursorPos
		}
		tf.SelectUpdate()
	}
	tf.UpdateEnd(updt)
}

// CursorForward moves the cursor backward
func (tf *TextField) CursorBackward(steps int) {
	updt := tf.UpdateStart()
	tf.CursorPos -= steps
	if tf.CursorPos < 0 {
		tf.CursorPos = 0
	}
	if tf.CursorPos <= tf.StartPos {
		dec := kit.MinInt(tf.StartPos, 8)
		tf.StartPos -= dec
	}
	if tf.SelectMode {
		if tf.CursorPos+steps < tf.SelectStart {
			tf.SelectStart = tf.CursorPos
		} else if tf.CursorPos > tf.SelectStart {
			tf.SelectEnd = tf.CursorPos
		} else {
			tf.SelectStart = tf.CursorPos
		}
		tf.SelectUpdate()
	}
	tf.UpdateEnd(updt)
}

// CursorStart moves the cursor to the start of the text, updating selection
// if select mode is active
func (tf *TextField) CursorStart() {
	updt := tf.UpdateStart()
	tf.CursorPos = 0
	tf.StartPos = 0
	tf.EndPos = kit.MinInt(len(tf.EditText), tf.StartPos+tf.CharWidth)
	if tf.SelectMode {
		tf.SelectStart = 0
		tf.SelectUpdate()
	}
	tf.UpdateEnd(updt)
}

// CursorEnd moves the cursor to the end of the text
func (tf *TextField) CursorEnd() {
	updt := tf.UpdateStart()
	ed := len(tf.EditText)
	tf.CursorPos = ed
	tf.EndPos = len(tf.EditText) // try -- display will adjust
	tf.StartPos = kit.MaxInt(0, tf.EndPos-tf.CharWidth)
	if tf.SelectMode {
		tf.SelectEnd = ed
		tf.SelectUpdate()
	}
	tf.UpdateEnd(updt)
}

// CursorBackspace deletes character(s) immediately before cursor
func (tf *TextField) CursorBackspace(steps int) {
	if tf.CursorPos < steps {
		steps = tf.CursorPos
	}
	if steps <= 0 {
		return
	}
	updt := tf.UpdateStart()
	tf.Edited = true
	tf.EditText = append(tf.EditText[:tf.CursorPos-steps], tf.EditText[tf.CursorPos:]...)
	tf.CursorBackward(steps)
	if tf.CursorPos > tf.SelectStart && tf.CursorPos <= tf.SelectEnd {
		tf.SelectEnd -= steps
	} else if tf.CursorPos < tf.SelectStart {
		tf.SelectStart -= steps
		tf.SelectEnd -= steps
	}
	tf.SelectUpdate()
	tf.UpdateEnd(updt)
}

// CursorDelete deletes character(s) immediately after the cursor
func (tf *TextField) CursorDelete(steps int) {
	if tf.CursorPos+steps > len(tf.EditText) {
		steps = len(tf.EditText) - tf.CursorPos
	}
	if steps <= 0 {
		return
	}
	updt := tf.UpdateStart()
	tf.Edited = true
	tf.EditText = append(tf.EditText[:tf.CursorPos], tf.EditText[tf.CursorPos+steps:]...)
	if tf.CursorPos > tf.SelectStart && tf.CursorPos <= tf.SelectEnd {
		tf.SelectEnd -= steps
	} else if tf.CursorPos < tf.SelectStart {
		tf.SelectStart -= steps
		tf.SelectEnd -= steps
	}
	tf.SelectUpdate()
	tf.UpdateEnd(updt)
}

// CursorKill deletes text from cursor to end of text
func (tf *TextField) CursorKill() {
	steps := len(tf.EditText) - tf.CursorPos
	tf.CursorDelete(steps)
}

// HasSelection returns whether there is a selected region of text
func (tf *TextField) HasSelection() bool {
	tf.SelectUpdate()
	if tf.SelectStart < tf.SelectEnd {
		return true
	}
	return false
}

// Selection returns the currently selected text
func (tf *TextField) Selection() string {
	if tf.HasSelection() {
		return string(tf.EditText[tf.SelectStart:tf.SelectEnd])
	}
	return ""
}

// SelectModeToggle toggles the SelectMode, updating selection with cursor movement
func (tf *TextField) SelectModeToggle() {
	if tf.SelectMode {
		tf.SelectMode = false
	} else {
		tf.SelectMode = true
		tf.SelectStart = tf.CursorPos
		tf.SelectEnd = tf.SelectStart
	}
}

// SelectAll selects all the text
func (tf *TextField) SelectAll() {
	updt := tf.UpdateStart()
	tf.SelectStart = 0
	tf.SelectEnd = len(tf.EditText)
	tf.UpdateEnd(updt)
}

// SelectWord selects the word (whitespace delimited) that the cursor is on
func (tf *TextField) SelectWord() {
	updt := tf.UpdateStart()
	sz := len(tf.EditText)
	if sz <= 3 {
		tf.SelectAll()
		return
	}
	tf.SelectStart = tf.CursorPos
	if tf.SelectStart >= sz {
		tf.SelectStart = sz - 2
	}
	if !unicode.IsSpace(tf.EditText[tf.SelectStart]) {
		for tf.SelectStart > 0 {
			if unicode.IsSpace(tf.EditText[tf.SelectStart-1]) {
				break
			}
			tf.SelectStart--
		}
		tf.SelectEnd = tf.CursorPos + 1
		for tf.SelectEnd < sz {
			if unicode.IsSpace(tf.EditText[tf.SelectEnd]) {
				break
			}
			tf.SelectEnd++
		}
	} else { // keep the space start -- go to next space..
		tf.SelectEnd = tf.CursorPos + 1
		for tf.SelectEnd < sz {
			if !unicode.IsSpace(tf.EditText[tf.SelectEnd]) {
				break
			}
			tf.SelectEnd++
		}
		for tf.SelectEnd < sz {
			if unicode.IsSpace(tf.EditText[tf.SelectEnd]) {
				break
			}
			tf.SelectEnd++
		}
	}
	tf.UpdateEnd(updt)
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
		ed := len(tf.EditText)
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

// Cut cuts any selected text and adds it to the clipboard, also returns cut text
func (tf *TextField) Cut() string {
	tf.SelectUpdate()
	if !tf.HasSelection() {
		return ""
	}
	updt := tf.UpdateStart()
	cut := tf.Selection()
	tf.Edited = true
	tf.EditText = append(tf.EditText[:tf.SelectStart], tf.EditText[tf.SelectEnd:]...)
	if tf.CursorPos > tf.SelectStart {
		if tf.CursorPos < tf.SelectEnd {
			tf.CursorPos = tf.SelectStart
		} else {
			tf.CursorPos -= tf.SelectEnd - tf.SelectStart
		}
	}
	tf.SelectReset()
	oswin.TheApp.ClipBoard().Write(([]byte)(cut), "text/plain")
	tf.UpdateEnd(updt)
	return cut
}

// Copy copies any selected text to the clipboard, and returns that text,
// optionaly resetting the current selection
func (tf *TextField) Copy(reset bool) string {
	tf.SelectUpdate()
	if !tf.HasSelection() {
		return ""
	}
	cpy := tf.Selection()
	oswin.TheApp.ClipBoard().Write(([]byte)(cpy), "text/plain")
	if reset {
		tf.SelectReset()
	}
	return cpy
}

// Paste inserts text from the clipboard at current cursor position
func (tf *TextField) Paste() {
	data, _, err := oswin.TheApp.ClipBoard().Read()
	if data != nil && err == nil {
		tf.InsertAtCursor(string(data))
	}
}

// InsertAtCursor inserts given text at current cursor position
func (tf *TextField) InsertAtCursor(str string) {
	updt := tf.UpdateStart()
	tf.Edited = true
	rs := []rune(str)
	rsl := len(rs)
	nt := make([]rune, 0, cap(tf.EditText)+cap(rs))
	nt = append(nt, tf.EditText[:tf.CursorPos]...)
	nt = append(nt, rs...)
	nt = append(nt, tf.EditText[tf.CursorPos:]...)
	tf.EditText = nt
	tf.EndPos += rsl
	tf.CursorForward(rsl)
	tf.UpdateEnd(updt)
}

func (tf *TextField) KeyInput(kt *key.ChordEvent) {
	kf := KeyFun(kt.ChordString())
	switch kf {
	case KeyFunSelectItem:
		tf.EditDone() // not processed, others could consume
	case KeyFunAccept:
		tf.EditDone() // not processed, others could consume
	case KeyFunAbort:
		tf.RevertEdit() // not processed, others could consume
	case KeyFunMoveRight:
		tf.CursorForward(1)
		kt.SetProcessed()
	case KeyFunMoveLeft:
		tf.CursorBackward(1)
		kt.SetProcessed()
	case KeyFunHome:
		tf.CursorStart()
		kt.SetProcessed()
	case KeyFunEnd:
		tf.CursorEnd()
		kt.SetProcessed()
	case KeyFunSelectText:
		tf.SelectModeToggle()
		kt.SetProcessed()
	case KeyFunCancelSelect:
		tf.SelectReset()
		kt.SetProcessed()
	case KeyFunSelectAll:
		tf.SelectAll()
		kt.SetProcessed()
	case KeyFunBackspace:
		tf.CursorBackspace(1)
		kt.SetProcessed()
	case KeyFunKill:
		tf.CursorKill()
		kt.SetProcessed()
	case KeyFunDelete:
		tf.CursorDelete(1)
		kt.SetProcessed()
	case KeyFunCopy:
		tf.Copy(true) // reset
		kt.SetProcessed()
	case KeyFunCut:
		tf.Cut()
		kt.SetProcessed()
	case KeyFunPaste:
		tf.Paste()
		kt.SetProcessed()
	case KeyFunNil:
		if unicode.IsPrint(kt.Rune) {
			tf.InsertAtCursor(string(kt.Rune))
		}
	}
}

// PixelToCursor finds the cursor position that corresponds to the given pixel location
func (tf *TextField) PixelToCursor(pixOff float32) int {
	st := &tf.Style

	spc := st.BoxSpace()
	px := pixOff - spc

	if px <= 0 {
		return tf.StartPos
	}

	// for selection to work correctly, we need this to be deterministic

	sz := len(tf.EditText)
	c := tf.StartPos + int(float64(px/st.UnContext.ToDotsFactor(units.Ch)))
	c = kit.MinInt(c, sz)

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

func (tf *TextField) SetCursorFromPixel(pixOff float32) {
	updt := tf.UpdateStart()
	tf.CursorPos = tf.PixelToCursor(pixOff)
	if tf.SelectMode {
		if !tf.IsDragging() && tf.CursorPos >= tf.SelectStart && tf.CursorPos < tf.SelectEnd {
			tf.SelectReset()
		} else if tf.CursorPos > tf.SelectStart {
			tf.SelectEnd = tf.CursorPos
		} else {
			tf.SelectStart = tf.CursorPos
		}
		tf.SelectUpdate()
	} else if tf.HasSelection() {
		tf.SelectReset()
	}
	tf.UpdateEnd(updt)
}

////////////////////////////////////////////////////
//  Node2D Interface

func (tf *TextField) Init2D() {
	tf.Init2DWidget()
	tf.EditText = []rune(tf.Text)
	tf.Edited = false
	bitflag.Set(&tf.Flag, int(InactiveEvents))
	tf.ReceiveEventType(oswin.MouseDragEvent, func(recv, send ki.Ki, sig int64, d interface{}) {
		me := d.(*mouse.DragEvent)
		me.SetProcessed()
		tf := recv.EmbeddedStruct(KiT_TextField).(*TextField)
		if tf.IsDragging() {
			if !tf.SelectMode {
				tf.SelectModeToggle()
			}
			pt := tf.PointToRelPos(me.Pos())
			tf.SetCursorFromPixel(float32(pt.X))
		}
	})
	tf.ReceiveEventType(oswin.MouseEvent, func(recv, send ki.Ki, sig int64, d interface{}) {
		tff := recv.EmbeddedStruct(KiT_TextField).(*TextField)
		me := d.(*mouse.Event)
		me.SetProcessed()
		if tff.IsInactive() {
			if me.Action == mouse.Press {
				tff.Selected = !tff.Selected
				if tff.Selected {
					tff.TextFieldSig.Emit(tff.This, int64(TextFieldSelected), tff.Text)
				}
				tff.UpdateSig()
			}
			return
		}
		if !tff.HasFocus() {
			tff.GrabFocus()
		}
		if me.Action == mouse.Press {
			pt := tff.PointToRelPos(me.Pos())
			tff.SetCursorFromPixel(float32(pt.X))
		} else if me.Action == mouse.DoubleClick {
			if tff.HasSelection() {
				if tff.SelectStart == 0 && tff.SelectEnd == len(tff.EditText) {
					tff.SelectReset()
				} else {
					tff.SelectAll()
				}
			} else {
				tff.SelectWord()
			}
		}
	})
	tf.ReceiveEventType(oswin.KeyChordEvent, func(recv, send ki.Ki, sig int64, d interface{}) {
		tff := recv.EmbeddedStruct(KiT_TextField).(*TextField)
		if tff.IsInactive() {
			return
		}
		kt := d.(*key.ChordEvent)
		tff.KeyInput(kt)
	})
	if dlg, ok := tf.Viewport.This.(*Dialog); ok {
		dlg.DialogSig.Connect(tf.This, func(recv, send ki.Ki, sig int64, data interface{}) {
			tff, _ := recv.EmbeddedStruct(KiT_TextField).(*TextField)
			if sig == int64(DialogAccepted) {
				tff.EditDone()
			}
		})
	}
}

func (tf *TextField) Style2D() {
	tf.SetCanFocusIfActive()
	tf.Style2DWidget()
	var pst *Style
	_, pg := KiToNode2D(tf.Par)
	if pg != nil {
		pst = &pg.Style
	}
	for i := 0; i < int(TextFieldStatesN); i++ {
		tf.StateStyles[i].CopyFrom(&tf.Style)
		tf.StateStyles[i].SetStyle(pst, tf.StyleProps(TextFieldSelectors[i]))
		tf.StateStyles[i].CopyUnitContext(&tf.Style.UnContext)
	}
}

func (tf *TextField) UpdateCharPos() bool {
	tf.CharPos = tf.Paint.MeasureChars(tf.EditText)
	tf.lastSizedText = tf.EditText
	return true
}

func (tf *TextField) Size2D() {
	tf.EditText = []rune(tf.Text)
	tf.Edited = false
	tf.StartPos = 0
	tf.EndPos = len(tf.EditText)
	tf.UpdateCharPos()
	h := tf.Paint.FontHeight()
	w := float32(10.0)
	sz := len(tf.CharPos)
	if sz > 0 {
		w = tf.CharPos[sz-1]
	}
	w += 2.0 // give some extra buffer
	tf.Size2DFromWH(w, h)
}

func (tf *TextField) Layout2D(parBBox image.Rectangle) {
	tf.Layout2DWidget(parBBox)
	for i := 0; i < int(TextFieldStatesN); i++ {
		tf.StateStyles[i].CopyUnitContext(&tf.Style.UnContext)
	}
	tf.Layout2DChildren()
}

// StartCharPos returns the starting position of the given character -- CharPos contains the ending positions
func (tf *TextField) StartCharPos(idx int) float32 {
	if idx <= 0 {
		return 0.0
	}
	sz := len(tf.CharPos)
	if sz == 0 {
		return 0.0
	}
	if idx > sz {
		return tf.CharPos[sz-1]
	}
	return tf.CharPos[idx-1]
}

// TextWidth returns the text width in dots between the two text string
// positions (ed is exclusive -- +1 beyond actual char)
func (tf *TextField) TextWidth(st, ed int) float32 {
	return tf.StartCharPos(ed) - tf.StartCharPos(st)
}

func (tf *TextField) RenderCursor() {
	pc := &tf.Paint
	rs := &tf.Viewport.Render
	st := &tf.Style
	pc.FontStyle = st.Font
	pc.TextStyle = st.Text
	spc := st.BoxSpace()
	pos := tf.LayData.AllocPos.AddVal(spc)

	cpos := tf.TextWidth(tf.StartPos, tf.CursorPos)

	h := pc.FontHeight()
	pc.DrawLine(rs, pos.X+cpos, pos.Y, pos.X+cpos, pos.Y+h)
	pc.Stroke(rs)
}

func (tf *TextField) RenderSelect() {
	if tf.SelectEnd <= tf.SelectStart {
		return
	}
	effst := kit.MaxInt(tf.StartPos, tf.SelectStart)
	if effst >= tf.EndPos {
		return
	}
	effed := kit.MinInt(tf.EndPos, tf.SelectEnd)
	if effed < tf.StartPos {
		return
	}
	if effed <= effst {
		return
	}

	pc := &tf.Paint
	rs := &tf.Viewport.Render
	st := &tf.StateStyles[TextFieldSelect]
	pc.FontStyle = st.Font
	pc.TextStyle = st.Text
	spc := st.BoxSpace()
	pos := tf.LayData.AllocPos.AddVal(spc)

	spos := tf.TextWidth(tf.StartPos, effst)
	tsz := tf.TextWidth(effst, effed)
	h := pc.FontHeight()

	pc.FillBox(rs, Vec2D{pos.X + spos, pos.Y}, Vec2D{tsz, h}, &st.Background.Color)
}

// AutoScroll scrolls the starting position to keep the cursor visible
func (tf *TextField) AutoScroll() {
	st := &tf.Style

	tf.UpdateCharPos()

	sz := len(tf.EditText)

	if sz == 0 {
		tf.CursorPos = 0
		tf.EndPos = 0
		tf.StartPos = 0
		return
	}
	spc := st.BoxSpace()
	maxw := tf.LayData.AllocSize.X - 2.0*spc
	tf.CharWidth = int(maxw / st.UnContext.ToDotsFactor(units.Ch)) // rough guess in chars

	// first rationalize all the values
	if tf.EndPos == 0 || tf.EndPos > sz { // not init
		tf.EndPos = sz
	}
	if tf.StartPos >= tf.EndPos {
		tf.StartPos = kit.MaxInt(0, tf.EndPos-tf.CharWidth)
	}
	tf.CursorPos = InRangeInt(tf.CursorPos, 0, sz)

	inc := int(math32.Ceil(.1 * float32(tf.CharWidth)))
	inc = kit.MaxInt(4, inc)

	// keep cursor in view with buffer
	startIsAnchor := true
	if tf.CursorPos < (tf.StartPos + inc) {
		tf.StartPos -= inc
		tf.StartPos = kit.MaxInt(tf.StartPos, 0)
		tf.EndPos = tf.StartPos + tf.CharWidth
		tf.EndPos = kit.MinInt(sz, tf.EndPos)
	} else if tf.CursorPos > (tf.EndPos - inc) {
		tf.EndPos += inc
		tf.EndPos = kit.MinInt(tf.EndPos, sz)
		tf.StartPos = tf.EndPos - tf.CharWidth
		tf.StartPos = kit.MaxInt(0, tf.StartPos)
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

func (tf *TextField) Render2D() {
	if tf.PushBounds() {
		tf.AutoScroll()
		if tf.IsInactive() {
			if tf.Selected {
				tf.Style = tf.StateStyles[TextFieldSelect]
			} else {
				tf.Style = tf.StateStyles[TextFieldInactive]
			}
		} else if tf.HasFocus() {
			tf.Style = tf.StateStyles[TextFieldFocus]
		} else {
			tf.Style = tf.StateStyles[TextFieldActive]
		}
		tf.RenderStdBox(&tf.Style)
		cur := tf.EditText[tf.StartPos:tf.EndPos]
		tf.RenderSelect()
		tf.Render2DText(string(cur))
		if tf.HasFocus() {
			tf.RenderCursor()
		}
		tf.Render2DChildren()
		tf.PopBounds()
	}
}

func (tf *TextField) FocusChanged2D(gotFocus bool) {
	if !gotFocus && !tf.IsInactive() {
		tf.EditDone() // lose focus
	}
	tf.UpdateSig()
}

////////////////////////////////////////////////////////////////////////////////////////
// SpinBox

//go:generate stringer -type=TextFieldSignals

// SpinBox combines a TextField with up / down buttons for incrementing / decrementing values -- all configured within the Parts of the widget
type SpinBox struct {
	WidgetBase
	Value      float32   `xml:"value" desc:"current value"`
	HasMin     bool      `xml:"has-min" desc:"is there a minimum value to enforce"`
	Min        float32   `xml:"min" desc:"minimum value in range"`
	HasMax     bool      `xml:"has-max" desc:"is there a maximumvalue to enforce"`
	Max        float32   `xml:"max" desc:"maximum value in range"`
	Step       float32   `xml:"step" desc:"smallest step size to increment"`
	PageStep   float32   `xml:"pagestep" desc:"larger PageUp / Dn step size"`
	Prec       int       `desc:"specifies the precision of decimal places (total, not after the decimal point) to use in representing the number -- this helps to truncate small weird floating point values in the nether regions"`
	UpIcon     *Icon     `json:"-" xml:"-" desc:"icon to use for up button -- defaults to widget-wedge-up"`
	DownIcon   *Icon     `json:"-" xml:"-" desc:"icon to use for down button -- defaults to widget-wedge-down"`
	SpinBoxSig ki.Signal `json:"-" xml:"-" desc:"signal for spin box -- has no signal types, just emitted when the value changes"`
}

var KiT_SpinBox = kit.Types.AddType(&SpinBox{}, SpinBoxProps)

var SpinBoxProps = ki.Props{
	"#buttons": ki.Props{
		"vert-align": AlignMiddle,
	},
	"#up": ki.Props{
		"max-width":  units.NewValue(1.5, units.Ex),
		"max-height": units.NewValue(1.5, units.Ex),
		"margin":     units.NewValue(1, units.Px),
		"padding":    units.NewValue(0, units.Px),
		"fill":       &Prefs.IconColor,
		"stroke":     &Prefs.FontColor,
	},
	"#down": ki.Props{
		"max-width":  units.NewValue(1.5, units.Ex),
		"max-height": units.NewValue(1.5, units.Ex),
		"margin":     units.NewValue(1, units.Px),
		"padding":    units.NewValue(0, units.Px),
		"fill":       &Prefs.IconColor,
		"stroke":     &Prefs.FontColor,
	},
	"#space": ki.Props{
		"width": units.NewValue(.1, units.Ex),
	},
	"#text-field": ki.Props{
		"min-width": units.NewValue(4, units.Ex),
		"width":     units.NewValue(8, units.Ex),
		"margin":    units.NewValue(2, units.Px),
		"padding":   units.NewValue(2, units.Px),
	},
}

func (g *SpinBox) Defaults() { // todo: should just get these from props
	g.Step = 0.1
	g.PageStep = 0.2
	g.Max = 1.0
	g.Prec = 6
}

// SetMin sets the min limits on the value
func (g *SpinBox) SetMin(min float32) {
	g.HasMin = true
	g.Min = min
}

// SetMax sets the max limits on the value
func (g *SpinBox) SetMax(max float32) {
	g.HasMax = true
	g.Max = max
}

// SetMinMax sets the min and max limits on the value
func (g *SpinBox) SetMinMax(hasMin bool, min float32, hasMax bool, max float32) {
	g.HasMin = hasMin
	g.Min = min
	g.HasMax = hasMax
	g.Max = max
	if g.Max < g.Min {
		log.Printf("gi.SpinBox SetMinMax: max was less than min -- disabling limits\n")
		g.HasMax = false
		g.HasMin = false
	}
}

// SetValue sets the value, enforcing any limits, and updates the display
func (g *SpinBox) SetValue(val float32) {
	updt := g.UpdateStart()
	g.Value = val
	if g.HasMax {
		g.Value = Min32(g.Value, g.Max)
	}
	if g.HasMin {
		g.Value = Max32(g.Value, g.Min)
	}
	g.Value = Truncate32(g.Value, g.Prec)
	g.UpdateEnd(updt)
}

// SetValueAction calls SetValue and also emits the signal
func (g *SpinBox) SetValueAction(val float32) {
	g.SetValue(val)
	g.SpinBoxSig.Emit(g.This, 0, g.Value)
}

// IncrValue increments the value by given number of steps (+ or -), and enforces it to be an even multiple of the step size (snap-to-value), and emits the signal
func (g *SpinBox) IncrValue(steps float32) {
	val := g.Value + steps*g.Step
	val = FloatMod32(val, g.Step)
	g.SetValueAction(val)
}

// internal indexes for accessing elements of the widget
const (
	sbTextFieldIdx = iota
	sbSpaceIdx
	sbButtonsIdx
)

func (g *SpinBox) ConfigParts() {
	if g.UpIcon == nil {
		g.UpIcon = IconByName("widget-wedge-up")
	}
	if g.DownIcon == nil {
		g.DownIcon = IconByName("widget-wedge-down")
	}
	g.Parts.Lay = LayoutRow
	g.Parts.SetProp("vert-align", AlignMiddle) // todo: style..
	config := kit.TypeAndNameList{}
	config.Add(KiT_TextField, "text-field")
	config.Add(KiT_Space, "space")
	config.Add(KiT_Layout, "buttons")
	mods, updt := g.Parts.ConfigChildren(config, false) // not unique names
	if mods {
		buts := g.Parts.Child(sbButtonsIdx).(*Layout)
		buts.Lay = LayoutCol
		g.StylePart(buts.This)
		buts.SetNChildren(2, KiT_Action, "but")
		// up
		up := buts.Child(0).(*Action)
		up.SetName("up")
		bitflag.SetState(up.Flags(), g.IsInactive(), int(Inactive))
		up.Icon = g.UpIcon
		g.StylePart(up.This)
		if !g.IsInactive() {
			up.ActionSig.ConnectOnly(g.This, func(recv, send ki.Ki, sig int64, data interface{}) {
				sb := recv.EmbeddedStruct(KiT_SpinBox).(*SpinBox)
				sb.IncrValue(1.0)
			})
		}
		// dn
		dn := buts.Child(1).(*Action)
		bitflag.SetState(dn.Flags(), g.IsInactive(), int(Inactive))
		dn.SetName("down")
		dn.Icon = g.DownIcon
		g.StylePart(dn.This)
		if !g.IsInactive() {
			dn.ActionSig.ConnectOnly(g.This, func(recv, send ki.Ki, sig int64, data interface{}) {
				sb := recv.EmbeddedStruct(KiT_SpinBox).(*SpinBox)
				sb.IncrValue(-1.0)
			})
		}
		// space
		g.StylePart(g.Parts.Child(sbSpaceIdx)) // also get the space
		// text-field
		tf := g.Parts.Child(sbTextFieldIdx).(*TextField)
		bitflag.SetState(tf.Flags(), g.IsInactive(), int(Inactive))
		g.StylePart(tf.This)
		tf.Text = fmt.Sprintf("%g", g.Value)
		if !g.IsInactive() {
			tf.TextFieldSig.ConnectOnly(g.This, func(recv, send ki.Ki, sig int64, data interface{}) {
				if sig == int64(TextFieldDone) {
					sb := recv.EmbeddedStruct(KiT_SpinBox).(*SpinBox)
					tf := send.(*TextField)
					vl, err := strconv.ParseFloat(tf.Text, 32)
					if err == nil {
						sb.SetValueAction(float32(vl))
					}
				}
			})
		}
		g.UpdateEnd(updt)
	}
}

func (g *SpinBox) ConfigPartsIfNeeded() {
	if !g.Parts.HasChildren() {
		g.ConfigParts()
	}
	tf := g.Parts.Child(sbTextFieldIdx).(*TextField)
	txt := fmt.Sprintf("%g", g.Value)
	if tf.Text != txt {
		tf.SetText(txt)
	}
}

func (g *SpinBox) Init2D() {
	g.Init2DWidget()
	g.ConfigParts()
	g.ReceiveEventType(oswin.MouseScrollEvent, func(recv, send ki.Ki, sig int64, d interface{}) {
		me := d.(*mouse.ScrollEvent)
		sb := recv.EmbeddedStruct(KiT_SpinBox).(*SpinBox)
		sb.IncrValue(float32(me.NonZeroDelta(false)))
		me.SetProcessed()
	})
}

func (g *SpinBox) Style2D() {
	if g.Step == 0 {
		g.Defaults()
	}
	g.Style2DWidget()
	g.ConfigParts()
}

func (g *SpinBox) Size2D() {
	g.Size2DWidget()
	g.ConfigParts()
}

func (g *SpinBox) Layout2D(parBBox image.Rectangle) {
	g.ConfigPartsIfNeeded()
	g.Layout2DWidget(parBBox)
	g.Layout2DChildren()
}

func (g *SpinBox) Render2D() {
	if g.PushBounds() {
		// g.Style = g.StateStyles[g.State] // get current styles
		g.ConfigPartsIfNeeded()
		g.Render2DChildren()
		g.Render2DParts()
		g.PopBounds()
	}
}

////////////////////////////////////////////////////////////////////////////////////////
// ComboBox for selecting items from a list

type ComboBox struct {
	ButtonBase
	Editable  bool          `desc:"provide a text field for editing the value, or just a button for selecting items?"`
	CurVal    interface{}   `json:"-" xml:"-" desc:"current selected value"`
	CurIndex  int           `json:"-" xml:"-" desc:"current index in list of possible items"`
	Items     []interface{} `json:"-" xml:"-" desc:"items available for selection"`
	ItemsMenu ki.Slice      `json:"-" xml:"-" desc:"the menu of actions for selecting items -- automatically generated from Items"`
	ComboSig  ki.Signal     `json:"-" xml:"-" desc:"signal for combo box, when a new value has been selected -- the signal type is the index of the selected item, and the data is the value"`
	MaxLength int           `desc:"maximum label length (in runes)"`
}

var KiT_ComboBox = kit.Types.AddType(&ComboBox{}, ComboBoxProps)

var ComboBoxProps = ki.Props{
	"border-width":     units.NewValue(1, units.Px),
	"border-radius":    units.NewValue(4, units.Px),
	"border-color":     &Prefs.BorderColor,
	"border-style":     BorderSolid,
	"padding":          units.NewValue(4, units.Px),
	"margin":           units.NewValue(4, units.Px),
	"text-align":       AlignCenter,
	"vertical-align":   AlignMiddle,
	"background-color": &Prefs.ControlColor,
	"#icon": ki.Props{
		"width":   units.NewValue(1, units.Em),
		"height":  units.NewValue(1, units.Em),
		"margin":  units.NewValue(0, units.Px),
		"padding": units.NewValue(0, units.Px),
		"fill":    &Prefs.IconColor,
		"stroke":  &Prefs.FontColor,
	},
	"#label": ki.Props{
		"margin":  units.NewValue(0, units.Px),
		"padding": units.NewValue(0, units.Px),
	},
	"#indicator": ki.Props{
		"width":          units.NewValue(1.5, units.Ex),
		"height":         units.NewValue(1.5, units.Ex),
		"margin":         units.NewValue(0, units.Px),
		"padding":        units.NewValue(0, units.Px),
		"vertical-align": AlignBottom,
		"fill":           &Prefs.IconColor,
		"stroke":         &Prefs.FontColor,
	},
	ButtonSelectors[ButtonActive]: ki.Props{},
	ButtonSelectors[ButtonInactive]: ki.Props{
		"border-color": "lighter-50",
		"color":        "lighter-50",
	},
	ButtonSelectors[ButtonHover]: ki.Props{
		"background-color": "darker-10",
	},
	ButtonSelectors[ButtonFocus]: ki.Props{
		"border-width":     units.NewValue(2, units.Px),
		"background-color": "lighter-20",
	},
	ButtonSelectors[ButtonDown]: ki.Props{
		"color":            "lighter-90",
		"background-color": "darker-30",
	},
	ButtonSelectors[ButtonSelected]: ki.Props{
		"background-color": "darker-25",
		"color":            "lighter-90",
	},
}

// ButtonWidget interface

func (g *ComboBox) ButtonAsBase() *ButtonBase {
	return &(g.ButtonBase)
}

func (g *ComboBox) ButtonRelease() {
	wasPressed := (g.State == ButtonDown)
	updt := g.UpdateStart()
	g.MakeItemsMenu()
	g.SetButtonState(ButtonActive)
	g.ButtonSig.Emit(g.This, int64(ButtonReleased), nil)
	if wasPressed {
		g.ButtonSig.Emit(g.This, int64(ButtonClicked), nil)
	}
	g.UpdateEnd(updt)
	pos := g.WinBBox.Max
	_, indic := KiToNode2D(g.Parts.ChildByName("indicator", 3))
	if indic != nil {
		pos = indic.WinBBox.Min
	} else {
		pos.Y -= 10
		pos.X -= 10
	}
	PopupMenu(g.ItemsMenu, pos.X, pos.Y, g.Viewport, g.Text)
}

// MakeItems makes sure the Items list is made, and if not, or reset is true, creates one with the given capacity
func (g *ComboBox) MakeItems(reset bool, capacity int) {
	if g.Items == nil || reset {
		g.Items = make([]interface{}, 0, capacity)
	}
}

// SortItems sorts the items according to their labels
func (g *ComboBox) SortItems(ascending bool) {
	sort.Slice(g.Items, func(i, j int) bool {
		if ascending {
			return ToLabel(g.Items[i]) < ToLabel(g.Items[j])
		} else {
			return ToLabel(g.Items[i]) > ToLabel(g.Items[j])
		}
	})
}

// SetToMaxLength gets the maximum label length so that the width of the button label is automatically set according to the max length of all items in the list -- if maxLen > 0 then it is used as an upper do-not-exceed length
func (g *ComboBox) SetToMaxLength(maxLen int) {
	ml := 0
	for _, it := range g.Items {
		ml = kit.MaxInt(ml, utf8.RuneCountInString(ToLabel(it)))
	}
	if maxLen > 0 {
		ml = kit.MinInt(ml, maxLen)
	}
	g.MaxLength = ml
}

// ItemsFromTypes sets the Items list from a list of types -- see e.g., AllImplementersOf or AllEmbedsOf in kit.TypeRegistry -- if setFirst then set current item to the first item in the list, sort sorts the list in ascending order, and maxLen if > 0 auto-sets the width of the button to the contents, with the given upper limit
func (g *ComboBox) ItemsFromTypes(tl []reflect.Type, setFirst, sort bool, maxLen int) {
	sz := len(tl)
	g.Items = make([]interface{}, sz)
	for i, typ := range tl {
		g.Items[i] = typ
	}
	if sort {
		g.SortItems(true)
	}
	if maxLen > 0 {
		g.SetToMaxLength(maxLen)
	}
	if setFirst {
		g.SetCurIndex(0)
	}
}

// ItemsFromEnumList sets the Items list from a list of enum values (see kit.EnumRegistry) -- if setFirst then set current item to the first item in the list, and maxLen if > 0 auto-sets the width of the button to the contents, with the given upper limit
func (g *ComboBox) ItemsFromEnumList(el []kit.EnumValue, setFirst bool, maxLen int) {
	sz := len(el)
	g.Items = make([]interface{}, sz)
	for i, enum := range el {
		g.Items[i] = enum
	}
	if maxLen > 0 {
		g.SetToMaxLength(maxLen)
	}
	if setFirst {
		g.SetCurIndex(0)
	}
}

// ItemsFromEnum sets the Items list from an enum type, which must be registered on kit.EnumRegistry -- if setFirst then set current item to the first item in the list, and maxLen if > 0 auto-sets the width of the button to the contents, with the given upper limit -- see kit.EnumRegistry, and maxLen if > 0 auto-sets the width of the button to the contents, with the given upper limit
func (g *ComboBox) ItemsFromEnum(enumtyp reflect.Type, setFirst bool, maxLen int) {
	g.ItemsFromEnumList(kit.Enums.TypeValues(enumtyp, true), setFirst, maxLen)
}

// FindItem finds an item on list of items and returns its index
func (g *ComboBox) FindItem(it interface{}) int {
	if g.Items == nil {
		return -1
	}
	for i, v := range g.Items {
		if v == it {
			return i
		}
	}
	return -1
}

// SetCurVal sets the current value (CurVal) and the corresponding CurIndex for that item on the current Items list (adds to items list if not found) -- returns that index -- and sets the text to the string value of that value (using standard Stringer string conversion)
func (g *ComboBox) SetCurVal(it interface{}) int {
	g.CurVal = it
	g.CurIndex = g.FindItem(it)
	if g.CurIndex < 0 { // add to list if not found..
		g.CurIndex = len(g.Items)
		g.Items = append(g.Items, it)
	}
	g.SetText(ToLabel(it))
	return g.CurIndex
}

// SetCurIndex sets the current index (CurIndex) and the corresponding CurVal for that item on the current Items list (-1 if not found) -- returns value -- and sets the text to the string value of that value (using standard Stringer string conversion)
func (g *ComboBox) SetCurIndex(idx int) interface{} {
	g.CurIndex = idx
	if idx < 0 || idx >= len(g.Items) {
		g.CurVal = nil
		g.SetText(fmt.Sprintf("idx %v > len", idx))
	} else {
		g.CurVal = g.Items[idx]
		g.SetText(ToLabel(g.CurVal))
	}
	return g.CurVal
}

// SelectItem selects a given item and emits the index as the ComboSig signal and the selected item as the data
func (g *ComboBox) SelectItem(idx int) {
	g.SetCurIndex(idx)
	g.ComboSig.Emit(g.This, int64(g.CurIndex), g.CurVal)
}

// set the text and update button -- does NOT change the currently-selected value or index
func (g *ComboBox) SetText(txt string) {
	SetButtonText(g, txt)
}

// set the Icon (could be nil) and update button
func (g *ComboBox) SetIcon(ic *Icon) {
	SetButtonIcon(g, ic)
}

// make menu of all the items
func (g *ComboBox) MakeItemsMenu() {
	if g.ItemsMenu == nil {
		g.ItemsMenu = make(ki.Slice, 0, len(g.Items))
	}
	sz := len(g.ItemsMenu)
	for i, it := range g.Items {
		var ac *Action
		if sz > i {
			ac = g.ItemsMenu[i].(*Action)
		} else {
			ac = &Action{}
			ac.Init(ac)
			g.ItemsMenu = append(g.ItemsMenu, ac.This.(Node2D))
		}
		txt := ToLabel(it)
		nm := fmt.Sprintf("Item_%v", i)
		ac.SetName(nm)
		ac.Text = txt
		ac.Data = i // index is the data
		ac.SetSelected(i == g.CurIndex)
		ac.SetAsMenu()
		ac.ActionSig.ConnectOnly(g.This, func(recv, send ki.Ki, sig int64, data interface{}) {
			idx := data.(int)
			cb := recv.(*ComboBox)
			cb.SelectItem(idx)
		})
	}
}

func (g *ComboBox) Init2D() {
	g.Init2DWidget()
	g.ConfigParts()
	Init2DButtonEvents(g)
}

func (g *ComboBox) ConfigParts() {
	config, icIdx, lbIdx := g.ConfigPartsIconLabel(g.Icon, g.Text)
	wrIdx := -1
	icnm := kit.ToString(g.Prop("indicator", false, false))
	if icnm == "" || icnm == "nil" {
		icnm = "widget-wedge-down"
	}
	if icnm != "none" {
		config.Add(KiT_Stretch, "indic-stretch")
		wrIdx = len(config)
		config.Add(KiT_Icon, "indicator")
	}
	mods, updt := g.Parts.ConfigChildren(config, false) // not unique names
	g.ConfigPartsSetIconLabel(g.Icon, g.Text, icIdx, lbIdx)
	if g.MaxLength > 0 && lbIdx >= 0 {
		lbl := g.Parts.Child(lbIdx).(*Label)
		lbl.SetMinPrefWidth(units.NewValue(float32(g.MaxLength), units.Ex))
	}
	if wrIdx >= 0 {
		ic := g.Parts.Child(wrIdx).(*Icon)
		if !ic.HasChildren() || ic.UniqueNm != icnm {
			ic.CopyFrom(IconByName(icnm))
			ic.UniqueNm = icnm
			g.StylePart(ic.This)
		}
	}
	if mods {
		g.UpdateEnd(updt)
	}
}

func (g *ComboBox) ConfigPartsIfNeeded() {
	if !g.PartsNeedUpdateIconLabel(g.Icon, g.Text) {
		return
	}
	g.ConfigParts()
}

func (g *ComboBox) Size2D() {
	g.Size2DWidget()
}

func (g *ComboBox) Layout2D(parBBox image.Rectangle) {
	g.ConfigParts()
	g.Layout2DWidget(parBBox)
	for i := 0; i < int(ButtonStatesN); i++ {
		g.StateStyles[i].CopyUnitContext(&g.Style.UnContext)
	}
	g.Layout2DChildren()
}

func (g *ComboBox) Render2D() {
	if g.PushBounds() {
		g.Style = g.StateStyles[g.State] // get current styles
		g.ConfigPartsIfNeeded()
		if !g.HasChildren() {
			g.Render2DDefaultStyle()
		} else {
			g.Render2DChildren()
		}
		g.PopBounds()
	}
}

// render using a default style if not otherwise styled
func (g *ComboBox) Render2DDefaultStyle() {
	st := &g.Style
	g.RenderStdBox(st)
	g.Render2DParts()
}

func (g *ComboBox) FocusChanged2D(gotFocus bool) {
	if gotFocus {
		g.SetButtonState(ButtonFocus)
	} else {
		g.SetButtonState(ButtonActive) // lose any hover state but whatever..
	}
	g.UpdateSig()
}
