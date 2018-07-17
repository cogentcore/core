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
	"github.com/goki/gi/oswin/mimedata"
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
	Text   string     `xml:"text" desc:"label to display"`
	Render TextRender `xml:"-" json:"-" desc:"render data for text label"`
}

var KiT_Label = kit.Types.AddType(&Label{}, LabelProps)

var LabelProps = ki.Props{
	"padding":          units.NewValue(2, units.Px),
	"margin":           units.NewValue(2, units.Px),
	"vertical-align":   AlignTop,
	"color":            &Prefs.FontColor,
	"background-color": color.Transparent,
}

// SetText sets the text and updates the rendered version
func (g *Label) SetText(txt string) {
	g.Text = txt
	g.Render.SetHTML(g.Text, &(g.Sty.Font), &(g.Sty.UnContext), g.CSSAgg)
	sz := g.LayData.AllocSize
	if sz.IsZero() {
		sz = g.LayData.SizePrefOrMax()
	}
	g.Render.LayoutStdLR(&(g.Sty.Text), &(g.Sty.Font), &(g.Sty.UnContext), sz)
}

// SetTextAction sets the text and triggers an update action
func (g *Label) SetTextAction(txt string) {
	g.SetText(txt)
	g.UpdateSig()
}

func (g *Label) Style2D() {
	g.Style2DWidget()
	g.Render.SetHTML(g.Text, &(g.Sty.Font), &(g.Sty.UnContext), g.CSSAgg)
	g.Render.LayoutStdLR(&(g.Sty.Text), &(g.Sty.Font), &(g.Sty.UnContext), g.LayData.SizePrefOrMax())
}

func (g *Label) Size2D() {
	g.InitLayout2D()
	g.Size2DFromWH(g.Render.Size.X, g.Render.Size.Y)
}

func (g *Label) Layout2D(parBBox image.Rectangle) {
	g.Layout2DBase(parBBox, true)
	g.Layout2DChildren()
	g.Render.LayoutStdLR(&(g.Sty.Text), &(g.Sty.Font), &(g.Sty.UnContext), g.LayData.AllocSize)
}

func (g *Label) Render2D() {
	if g.FullReRenderIfNeeded() {
		return
	}
	if g.PushBounds() {
		g.WidgetEvents()
		st := &g.Sty
		rs := &g.Viewport.Render
		g.RenderStdBox(st)
		pos := g.LayData.AllocPos.AddVal(st.BoxSpace())
		g.Render.Render(rs, pos)
		g.Render2DChildren()
		g.PopBounds()
	} else {
		g.DisconnectAllEvents()
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

// TextFieldStates are mutually-exclusive textfield states -- determines appearance
type TextFieldStates int32

const (
	// normal state -- there but not being interacted with
	TextFieldActive TextFieldStates = iota

	// textfield is the focus -- will respond to keyboard input
	TextFieldFocus

	// inactive -- not editable
	TextFieldInactive

	// selected -- for inactive state, can select entire element
	TextFieldSel

	TextFieldStatesN
)

//go:generate stringer -type=TextFieldStates

// Style selector names for the different states
var TextFieldSelectors = []string{":active", ":focus", ":inactive", ":selected"}

// TextField is a widget for editing a line of text
type TextField struct {
	WidgetBase
	Txt          string                  `json:"-" xml:"text" desc:"the last saved value of the text string being edited"`
	Edited       bool                    `json:"-" xml:"-" desc:"true if the text has been edited relative to the original"`
	EditTxt      []rune                  `json:"-" xml:"-" desc:"the live text string being edited, with latest modifications -- encoded as runes"`
	MaxWidthReq  int                     `desc:"maximum width that field will request, in characters, during Size2D process -- if 0 then is 50 -- ensures that large strings don't request super large values -- standard max-width can override"`
	StartPos     int                     `xml:"-" desc:"starting display position in the string"`
	EndPos       int                     `xml:"-" desc:"ending display position in the string"`
	CursorPos    int                     `xml:"-" desc:"current cursor position"`
	CharWidth    int                     `xml:"-" desc:"approximate number of chars that can be displayed at any time -- computed from font size etc"`
	SelectStart  int                     `xml:"-" desc:"starting position of selection in the string"`
	SelectEnd    int                     `xml:"-" desc:"ending position of selection in the string"`
	SelectMode   bool                    `xml:"-" desc:"if true, select text as cursor moves"`
	TextFieldSig ki.Signal               `json:"-" xml:"-" desc:"signal for line edit -- see TextFieldSignals for the types"`
	RenderAll    TextRender              `json:"-" xml:"-" desc:"render version of entire text, for sizing"`
	RenderVis    TextRender              `json:"-" xml:"-" desc:"render version of just visible text"`
	StateStyles  [TextFieldStatesN]Style `json:"-" xml:"-" desc:"normal style and focus style"`
	FontHeight   float32                 `json:"-" xml:"-" desc:"font height, cached during styling"`
}

var KiT_TextField = kit.Types.AddType(&TextField{}, TextFieldProps)

var TextFieldProps = ki.Props{
	"border-width":     units.NewValue(1, units.Px),
	"border-color":     &Prefs.BorderColor,
	"border-style":     BorderSolid,
	"padding":          units.NewValue(4, units.Px),
	"margin":           units.NewValue(1, units.Px),
	"text-align":       AlignLeft,
	"color":            &Prefs.FontColor,
	"background-color": &Prefs.ControlColor,
	TextFieldSelectors[TextFieldActive]: ki.Props{
		"background-color": "lighter-0",
	},
	TextFieldSelectors[TextFieldFocus]: ki.Props{
		"border-width":     units.NewValue(2, units.Px),
		"background-color": "samelight-80",
	},
	TextFieldSelectors[TextFieldInactive]: ki.Props{
		"background-color": "highlight-20",
	},
	TextFieldSelectors[TextFieldSel]: ki.Props{
		"background-color": &Prefs.SelectColor,
	},
}

// Text returns the current text -- applies any unapplied changes first
func (tf *TextField) Text() string {
	tf.EditDone()
	return tf.Txt
}

// SetText sets the text to be edited and reverts any current edit to reflect this new text
func (tf *TextField) SetText(txt string) {
	if tf.Txt == txt && !tf.Edited {
		return
	}
	tf.Txt = txt
	tf.RevertEdit()
}

// EditDone completes editing and copies the active edited text to the text --
// called when the return key is pressed or goes out of focus
func (tf *TextField) EditDone() {
	if tf.Edited {
		tf.Edited = false
		tf.Txt = string(tf.EditTxt)
		tf.TextFieldSig.Emit(tf.This, int64(TextFieldDone), tf.Txt)
	}
}

// RevertEdit aborts editing and reverts to last saved text
func (tf *TextField) RevertEdit() {
	updt := tf.UpdateStart()
	defer tf.UpdateEnd(updt)
	tf.EditTxt = []rune(tf.Txt)
	tf.Edited = false
	tf.StartPos = 0
	tf.EndPos = tf.CharWidth
	tf.SelectReset()
}

// CursorForward moves the cursor forward
func (tf *TextField) CursorForward(steps int) {
	updt := tf.UpdateStart()
	defer tf.UpdateEnd(updt)
	tf.CursorPos += steps
	if tf.CursorPos > len(tf.EditTxt) {
		tf.CursorPos = len(tf.EditTxt)
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
}

// CursorForward moves the cursor backward
func (tf *TextField) CursorBackward(steps int) {
	updt := tf.UpdateStart()
	defer tf.UpdateEnd(updt)
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
}

// CursorStart moves the cursor to the start of the text, updating selection
// if select mode is active
func (tf *TextField) CursorStart() {
	updt := tf.UpdateStart()
	defer tf.UpdateEnd(updt)
	tf.CursorPos = 0
	tf.StartPos = 0
	tf.EndPos = kit.MinInt(len(tf.EditTxt), tf.StartPos+tf.CharWidth)
	if tf.SelectMode {
		tf.SelectStart = 0
		tf.SelectUpdate()
	}
}

// CursorEnd moves the cursor to the end of the text
func (tf *TextField) CursorEnd() {
	updt := tf.UpdateStart()
	defer tf.UpdateEnd(updt)
	ed := len(tf.EditTxt)
	tf.CursorPos = ed
	tf.EndPos = len(tf.EditTxt) // try -- display will adjust
	tf.StartPos = kit.MaxInt(0, tf.EndPos-tf.CharWidth)
	if tf.SelectMode {
		tf.SelectEnd = ed
		tf.SelectUpdate()
	}
}

// todo: ctrl+backspace = delete word
// shift+arrow = select
// uparrow = start / down = end

// CursorBackspace deletes character(s) immediately before cursor
func (tf *TextField) CursorBackspace(steps int) {
	if tf.HasSelection() {
		tf.DeleteSelection()
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
	if tf.CursorPos > tf.SelectStart && tf.CursorPos <= tf.SelectEnd {
		tf.SelectEnd -= steps
	} else if tf.CursorPos < tf.SelectStart {
		tf.SelectStart -= steps
		tf.SelectEnd -= steps
	}
	tf.SelectUpdate()
}

// CursorDelete deletes character(s) immediately after the cursor
func (tf *TextField) CursorDelete(steps int) {
	if tf.HasSelection() {
		tf.DeleteSelection()
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
	if tf.CursorPos > tf.SelectStart && tf.CursorPos <= tf.SelectEnd {
		tf.SelectEnd -= steps
	} else if tf.CursorPos < tf.SelectStart {
		tf.SelectStart -= steps
		tf.SelectEnd -= steps
	}
	tf.SelectUpdate()
}

// CursorKill deletes text from cursor to end of text
func (tf *TextField) CursorKill() {
	steps := len(tf.EditTxt) - tf.CursorPos
	tf.CursorDelete(steps)
}

// ClearSelected resets both the global selected flag and any current selection
func (tf *TextField) ClearSelected() {
	tf.WidgetBase.ClearSelected()
	tf.SelectReset()
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
		tf.SelectStart = tf.CursorPos
		tf.SelectEnd = tf.SelectStart
	}
}

// SelectAll selects all the text
func (tf *TextField) SelectAll() {
	updt := tf.UpdateStart()
	tf.SelectStart = 0
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
		for tf.SelectEnd < sz {
			if tf.IsWordBreak(tf.EditTxt[tf.SelectEnd]) {
				break
			}
			tf.SelectEnd++
		}
	}
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

// Cut cuts any selected text and adds it to the clipboard, also returns cut text
func (tf *TextField) Cut() string {
	cut := tf.DeleteSelection()
	if cut != "" {
		oswin.TheApp.ClipBoard().Write(mimedata.NewText(cut))
	}
	return cut
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

// Copy copies any selected text to the clipboard, and returns that text,
// optionaly resetting the current selection
func (tf *TextField) Copy(reset bool) string {
	tf.SelectUpdate()
	if !tf.HasSelection() {
		return ""
	}
	cpy := tf.Selection()
	oswin.TheApp.ClipBoard().Write(mimedata.NewText(cpy))
	if reset {
		tf.SelectReset()
	}
	return cpy
}

// Paste inserts text from the clipboard at current cursor position -- if
// cursor is within a current selection, that selection is
func (tf *TextField) Paste() {
	data := oswin.TheApp.ClipBoard().Read([]string{mimedata.TextPlain})
	if data != nil {
		if tf.CursorPos >= tf.SelectStart && tf.CursorPos < tf.SelectEnd {
			tf.DeleteSelection()
		}
		tf.InsertAtCursor(data.Text(mimedata.TextPlain))
	}
}

// InsertAtCursor inserts given text at current cursor position
func (tf *TextField) InsertAtCursor(str string) {
	updt := tf.UpdateStart()
	defer tf.UpdateEnd(updt)
	if tf.HasSelection() {
		tf.Cut()
	}
	tf.Edited = true
	rs := []rune(str)
	rsl := len(rs)
	nt := make([]rune, 0, cap(tf.EditTxt)+cap(rs))
	nt = append(nt, tf.EditTxt[:tf.CursorPos]...)
	nt = append(nt, rs...)
	nt = append(nt, tf.EditTxt[tf.CursorPos:]...)
	tf.EditTxt = nt
	tf.EndPos += rsl
	tf.CursorForward(rsl)
}

// ActionMenu pops up a menu of various actions to perform
func (tf *TextField) ActionMenu() {
	var men Menu
	tf.MakeActionMenu(&men)
	cpos := tf.CharStartPos(tf.CursorPos).ToPoint()
	PopupMenu(men, cpos.X, cpos.Y, tf.Viewport, "tfActionMenu")
}

// MakeActionMenu makes the menu of actions that can be performed on each
// node.
func (tf *TextField) MakeActionMenu(m *Menu) {
	if len(*m) > 0 {
		return
	}
	// todo: add shortcuts
	m.AddMenuText("Copy", tf.This, nil, func(recv, send ki.Ki, sig int64, data interface{}) {
		tff := recv.EmbeddedStruct(KiT_TextField).(*TextField)
		tff.Copy(true)
	})
	if !tf.IsInactive() {
		m.AddMenuText("Cut", tf.This, nil, func(recv, send ki.Ki, sig int64, data interface{}) {
			tff := recv.EmbeddedStruct(KiT_TextField).(*TextField)
			tff.Cut()
		})
		m.AddMenuText("Paste", tf.This, nil, func(recv, send ki.Ki, sig int64, data interface{}) {
			tff := recv.EmbeddedStruct(KiT_TextField).(*TextField)
			tff.Paste()
		})
	}
}

func (tf *TextField) KeyInput(kt *key.ChordEvent) {
	kf := KeyFun(kt.ChordString())
	// first all the keys that work for both inactive and active
	switch kf {
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
	case KeyFunSelectMode:
		tf.SelectModeToggle()
		kt.SetProcessed()
	case KeyFunCancelSelect:
		tf.SelectReset()
		kt.SetProcessed()
	case KeyFunSelectAll:
		tf.SelectAll()
		kt.SetProcessed()
	case KeyFunCopy:
		tf.Copy(true) // reset
		kt.SetProcessed()
	}
	if tf.IsInactive() || kt.IsProcessed() {
		return
	}
	switch kf {
	case KeyFunSelectItem:
		tf.EditDone() // not processed, others could consume
	case KeyFunAccept:
		tf.EditDone() // not processed, others could consume
	case KeyFunAbort:
		tf.RevertEdit() // not processed, others could consume
	case KeyFunBackspace:
		tf.CursorBackspace(1)
		kt.SetProcessed()
	case KeyFunKill:
		tf.CursorKill()
		kt.SetProcessed()
	case KeyFunDelete:
		tf.CursorDelete(1)
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
	st := &tf.Sty

	spc := st.BoxSpace()
	px := pixOff - spc

	if px <= 0 {
		return tf.StartPos
	}

	// for selection to work correctly, we need this to be deterministic

	sz := len(tf.EditTxt)
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

func (tf *TextField) SetCursorFromPixel(pixOff float32, selMode mouse.SelectModes) {
	updt := tf.UpdateStart()
	defer tf.UpdateEnd(updt)
	oldPos := tf.CursorPos
	tf.CursorPos = tf.PixelToCursor(pixOff)
	if tf.SelectMode || selMode != mouse.NoSelectMode {
		if !tf.SelectMode && selMode != mouse.NoSelectMode {
			tf.SelectStart = oldPos
			tf.SelectMode = true
		}
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
}

func (tf *TextField) TextFieldEvents() {
	tf.WidgetEvents()
	tf.ConnectEventType(oswin.MouseDragEvent, func(recv, send ki.Ki, sig int64, d interface{}) {
		me := d.(*mouse.DragEvent)
		me.SetProcessed()
		tf := recv.EmbeddedStruct(KiT_TextField).(*TextField)
		if tf.IsDragging() {
			if !tf.SelectMode {
				tf.SelectModeToggle()
			}
			pt := tf.PointToRelPos(me.Pos())
			tf.SetCursorFromPixel(float32(pt.X), mouse.NoSelectMode)
		}
	})
	tf.ConnectEventType(oswin.MouseEvent, func(recv, send ki.Ki, sig int64, d interface{}) {
		tff := recv.EmbeddedStruct(KiT_TextField).(*TextField)
		me := d.(*mouse.Event)
		me.SetProcessed()
		if tff.IsInactive() {
			if me.Action == mouse.Press {
				tff.SetSelectedState(!tff.IsSelected())
				if tff.IsSelected() {
					tff.TextFieldSig.Emit(tff.This, int64(TextFieldSelected), tff.Txt)
				}
				tff.UpdateSig()
			}
			return
		}
		if !tff.HasFocus() {
			tff.GrabFocus()
		}
		switch me.Button {
		case mouse.Left:
			if me.Action == mouse.Press {
				pt := tff.PointToRelPos(me.Pos())
				tff.SetCursorFromPixel(float32(pt.X), me.SelectMode())
			} else if me.Action == mouse.DoubleClick {
				if tff.HasSelection() {
					if tff.SelectStart == 0 && tff.SelectEnd == len(tff.EditTxt) {
						tff.SelectReset()
					} else {
						tff.SelectAll()
					}
				} else {
					tff.SelectWord()
				}
			}
		case mouse.Middle:
			pt := tff.PointToRelPos(me.Pos())
			tff.SetCursorFromPixel(float32(pt.X), me.SelectMode())
			tff.Paste()
		case mouse.Right:
			tff.ActionMenu()
		}
	})
	tf.ConnectEventType(oswin.KeyChordEvent, func(recv, send ki.Ki, sig int64, d interface{}) {
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

////////////////////////////////////////////////////
//  Node2D Interface

func (tf *TextField) Init2D() {
	tf.Init2DWidget()
	tf.EditTxt = []rune(tf.Txt)
	tf.Edited = false
	bitflag.Set(&tf.Flag, int(InactiveEvents))
}

func (tf *TextField) Style2D() {
	tf.SetCanFocusIfActive()
	tf.Style2DWidget()
	pst := &(tf.Par.(Node2D).AsWidget().Sty)
	for i := 0; i < int(TextFieldStatesN); i++ {
		tf.StateStyles[i].CopyFrom(&tf.Sty)
		tf.StateStyles[i].SetStyleProps(pst, tf.StyleProps(TextFieldSelectors[i]))
		tf.StateStyles[i].CopyUnitContext(&tf.Sty.UnContext)
	}
}

func (tf *TextField) UpdateRenderAll() bool {
	st := &tf.Sty
	st.Font.LoadFont(&st.UnContext, "")
	tf.RenderAll.SetRunes(tf.EditTxt, &st.Font, &st.Text, true, 0, 0)
	return true
}

func (tf *TextField) Size2D() {
	tf.EditTxt = []rune(tf.Txt)
	tf.Edited = false
	tf.StartPos = 0
	maxlen := tf.MaxWidthReq
	if maxlen <= 0 {
		maxlen = 50
	}
	tf.EndPos = kit.MinInt(len(tf.EditTxt), maxlen)
	tf.UpdateRenderAll()
	tf.FontHeight = tf.RenderAll.Size.Y
	w := tf.TextWidth(tf.StartPos, tf.EndPos)
	w += 2.0 // give some extra buffer
	// fmt.Printf("fontheight: %v width: %v\n", tf.FontHeight, w)
	tf.Size2DFromWH(w, tf.FontHeight)
}

func (tf *TextField) Layout2D(parBBox image.Rectangle) {
	tf.Layout2DBase(parBBox, true) // init style
	for i := 0; i < int(TextFieldStatesN); i++ {
		tf.StateStyles[i].CopyUnitContext(&tf.Sty.UnContext)
	}
	tf.Layout2DChildren()
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

// TextWidth returns the text width in dots between the two text string
// positions (ed is exclusive -- +1 beyond actual char)
func (tf *TextField) TextWidth(st, ed int) float32 {
	return tf.StartCharPos(ed) - tf.StartCharPos(st)
}

// CharStartPos returns the starting render coords for the given character
// position in string -- makes no attempt to rationalize that pos (i.e., if
// not in visible range, position will be out of range too)
func (tf *TextField) CharStartPos(charidx int) Vec2D {
	st := &tf.Sty
	spc := st.BoxSpace()
	pos := tf.LayData.AllocPos.AddVal(spc)
	cpos := tf.TextWidth(tf.StartPos, charidx)
	return Vec2D{pos.X + cpos, pos.Y}
}

func (tf *TextField) RenderCursor() {
	cpos := tf.CharStartPos(tf.CursorPos)
	rs := &tf.Viewport.Render
	pc := &rs.Paint
	pc.DrawLine(rs, cpos.X, cpos.Y, cpos.X, cpos.Y+tf.FontHeight)
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

	spos := tf.CharStartPos(effst)

	rs := &tf.Viewport.Render
	pc := &rs.Paint
	st := &tf.StateStyles[TextFieldSel]
	tsz := tf.TextWidth(effst, effed)
	pc.FillBox(rs, spos, Vec2D{tsz, tf.FontHeight}, &st.Font.BgColor)
}

// AutoScroll scrolls the starting position to keep the cursor visible
func (tf *TextField) AutoScroll() {
	st := &tf.Sty

	tf.UpdateRenderAll()

	sz := len(tf.EditTxt)

	if sz == 0 || tf.LayData.AllocSize.X <= 0 {
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
	if tf.FullReRenderIfNeeded() {
		return
	}
	if tf.PushBounds() {
		tf.TextFieldEvents()
		tf.AutoScroll() // inits paint with our style
		if tf.IsInactive() {
			if tf.IsSelected() {
				tf.Sty = tf.StateStyles[TextFieldSel]
			} else {
				tf.Sty = tf.StateStyles[TextFieldInactive]
			}
		} else if tf.HasFocus() {
			tf.Sty = tf.StateStyles[TextFieldFocus]
		} else {
			tf.Sty = tf.StateStyles[TextFieldActive]
		}
		rs := &tf.Viewport.Render
		st := &tf.Sty
		st.Font.LoadFont(&st.UnContext, "")
		tf.RenderStdBox(st)
		cur := tf.EditTxt[tf.StartPos:tf.EndPos]
		tf.RenderSelect()
		pos := tf.LayData.AllocPos.AddVal(st.BoxSpace())
		tf.RenderVis.SetRunes(cur, &st.Font, &st.Text, true, 0, 0)
		tf.RenderVis.RenderTopPos(rs, pos)
		if tf.HasFocus() {
			tf.RenderCursor()
		}
		tf.Render2DChildren()
		tf.PopBounds()
	} else {
		tf.DisconnectAllEvents()
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

// SpinBox combines a TextField with up / down buttons for incrementing /
// decrementing values -- all configured within the Parts of the widget
type SpinBox struct {
	PartsWidgetBase
	Value      float32   `xml:"value" desc:"current value"`
	HasMin     bool      `xml:"has-min" desc:"is there a minimum value to enforce"`
	Min        float32   `xml:"min" desc:"minimum value in range"`
	HasMax     bool      `xml:"has-max" desc:"is there a maximumvalue to enforce"`
	Max        float32   `xml:"max" desc:"maximum value in range"`
	Step       float32   `xml:"step" desc:"smallest step size to increment"`
	PageStep   float32   `xml:"pagestep" desc:"larger PageUp / Dn step size"`
	Prec       int       `desc:"specifies the precision of decimal places (total, not after the decimal point) to use in representing the number -- this helps to truncate small weird floating point values in the nether regions"`
	UpIcon     IconName  `json:"-" xml:"-" desc:"icon to use for up button -- defaults to widget-wedge-up"`
	DownIcon   IconName  `json:"-" xml:"-" desc:"icon to use for down button -- defaults to widget-wedge-down"`
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
	defer g.UpdateEnd(updt)
	if g.Prec == 0 {
		g.Defaults()
	}
	g.Value = val
	if g.HasMax {
		g.Value = Min32(g.Value, g.Max)
	}
	if g.HasMin {
		g.Value = Max32(g.Value, g.Min)
	}
	g.Value = Truncate32(g.Value, g.Prec)
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
	if g.UpIcon.IsNil() {
		g.UpIcon = IconName("widget-wedge-up")
	}
	if g.DownIcon.IsNil() {
		g.DownIcon = IconName("widget-wedge-down")
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
		g.StylePart(Node2D(buts))
		buts.SetNChildren(2, KiT_Action, "but")
		// up
		up := buts.Child(0).(*Action)
		up.SetName("up")
		bitflag.SetState(up.Flags(), g.IsInactive(), int(Inactive))
		up.Icon = g.UpIcon
		g.StylePart(Node2D(up))
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
		g.StylePart(Node2D(dn))
		if !g.IsInactive() {
			dn.ActionSig.ConnectOnly(g.This, func(recv, send ki.Ki, sig int64, data interface{}) {
				sb := recv.EmbeddedStruct(KiT_SpinBox).(*SpinBox)
				sb.IncrValue(-1.0)
			})
		}
		// space
		g.StylePart(g.Parts.Child(sbSpaceIdx).(Node2D)) // also get the space
		// text-field
		tf := g.Parts.Child(sbTextFieldIdx).(*TextField)
		bitflag.SetState(tf.Flags(), g.IsInactive(), int(Inactive))
		g.StylePart(Node2D(tf))
		tf.Txt = fmt.Sprintf("%g", g.Value)
		if !g.IsInactive() {
			tf.TextFieldSig.ConnectOnly(g.This, func(recv, send ki.Ki, sig int64, data interface{}) {
				if sig == int64(TextFieldDone) {
					sb := recv.EmbeddedStruct(KiT_SpinBox).(*SpinBox)
					tf := send.(*TextField)
					vl, err := strconv.ParseFloat(tf.Text(), 32)
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
	if tf.Txt != txt {
		tf.SetText(txt)
	}
}

func (g *SpinBox) SpinBoxEvents() {
	g.ConnectEventType(oswin.MouseScrollEvent, func(recv, send ki.Ki, sig int64, d interface{}) {
		me := d.(*mouse.ScrollEvent)
		sb := recv.EmbeddedStruct(KiT_SpinBox).(*SpinBox)
		sb.IncrValue(float32(me.NonZeroDelta(false)))
		me.SetProcessed()
	})
}

func (g *SpinBox) Init2D() {
	g.Init2DWidget()
	g.ConfigParts()
}

func (g *SpinBox) Style2D() {
	if g.Step == 0 {
		g.Defaults()
	}
	g.Style2DWidget()
	g.ConfigParts()
}

func (g *SpinBox) Size2D() {
	g.Size2DParts()
	g.ConfigParts()
}

func (g *SpinBox) Layout2D(parBBox image.Rectangle) {
	g.ConfigPartsIfNeeded()
	g.Layout2DBase(parBBox, true) // init style
	g.Layout2DParts(parBBox)
	g.Layout2DChildren()
}

func (g *SpinBox) Render2D() {
	if g.FullReRenderIfNeeded() {
		return
	}
	if g.PushBounds() {
		g.SpinBoxEvents()
		// g.Sty = g.StateStyles[g.State] // get current styles
		g.ConfigPartsIfNeeded()
		g.Render2DChildren()
		g.Render2DParts()
		g.PopBounds()
	} else {
		g.DisconnectAllEvents()
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
	ItemsMenu Menu          `json:"-" xml:"-" desc:"the menu of actions for selecting items -- automatically generated from Items"`
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
	"background-color": &Prefs.ControlColor,
	"color":            &Prefs.FontColor,
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
	ButtonSelectors[ButtonActive]: ki.Props{
		"background-color": "linear-gradient(lighter-0, highlight-10)",
	},
	ButtonSelectors[ButtonInactive]: ki.Props{
		"border-color": "highlight-50",
		"color":        "highlight-50",
	},
	ButtonSelectors[ButtonHover]: ki.Props{
		"background-color": "linear-gradient(highlight-10, highlight-10)",
	},
	ButtonSelectors[ButtonFocus]: ki.Props{
		"border-width":     units.NewValue(2, units.Px),
		"background-color": "linear-gradient(samelight-50, highlight-10)",
	},
	ButtonSelectors[ButtonDown]: ki.Props{
		"color":            "highlight-90",
		"background-color": "linear-gradient(highlight-30, highlight-10)",
	},
	ButtonSelectors[ButtonSelected]: ki.Props{
		"background-color": "linear-gradient(pref(SelectColor), highlight-10)",
		"color":            "highlight-90",
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

func (g *ComboBox) ConfigParts() {
	config, icIdx, lbIdx := g.ConfigPartsIconLabel(string(g.Icon), g.Text)
	indIdx := g.ConfigPartsAddIndicator(&config, true)  // default on
	mods, updt := g.Parts.ConfigChildren(config, false) // not unique names
	g.ConfigPartsSetIconLabel(string(g.Icon), g.Text, icIdx, lbIdx)
	g.ConfigPartsIndicator(indIdx)
	if g.MaxLength > 0 && lbIdx >= 0 {
		lbl := g.Parts.Child(lbIdx).(*Label)
		lbl.SetMinPrefWidth(units.NewValue(float32(g.MaxLength), units.Ex))
	}
	if mods {
		g.UpdateEnd(updt)
	}
}

// MakeItems makes sure the Items list is made, and if not, or reset is true,
// creates one with the given capacity
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

// SetToMaxLength gets the maximum label length so that the width of the
// button label is automatically set according to the max length of all items
// in the list -- if maxLen > 0 then it is used as an upper do-not-exceed
// length
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

// ItemsFromTypes sets the Items list from a list of types -- see e.g.,
// AllImplementersOf or AllEmbedsOf in kit.TypeRegistry -- if setFirst then
// set current item to the first item in the list, sort sorts the list in
// ascending order, and maxLen if > 0 auto-sets the width of the button to the
// contents, with the given upper limit
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

// ItemsFromStringList sets the Items list from a list of string values -- if
// setFirst then set current item to the first item in the list, and maxLen if
// > 0 auto-sets the width of the button to the contents, with the given upper
// limit
func (g *ComboBox) ItemsFromStringList(el []string, setFirst bool, maxLen int) {
	sz := len(el)
	g.Items = make([]interface{}, sz)
	for i, str := range el {
		g.Items[i] = str
	}
	if maxLen > 0 {
		g.SetToMaxLength(maxLen)
	}
	if setFirst {
		g.SetCurIndex(0)
	}
}

// ItemsFromEnumList sets the Items list from a list of enum values (see
// kit.EnumRegistry) -- if setFirst then set current item to the first item in
// the list, and maxLen if > 0 auto-sets the width of the button to the
// contents, with the given upper limit
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

// ItemsFromEnum sets the Items list from an enum type, which must be
// registered on kit.EnumRegistry -- if setFirst then set current item to the
// first item in the list, and maxLen if > 0 auto-sets the width of the button
// to the contents, with the given upper limit -- see kit.EnumRegistry, and
// maxLen if > 0 auto-sets the width of the button to the contents, with the
// given upper limit
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

// SetCurVal sets the current value (CurVal) and the corresponding CurIndex
// for that item on the current Items list (adds to items list if not found)
// -- returns that index -- and sets the text to the string value of that
// value (using standard Stringer string conversion)
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

// SetCurIndex sets the current index (CurIndex) and the corresponding CurVal
// for that item on the current Items list (-1 if not found) -- returns value
// -- and sets the text to the string value of that value (using standard
// Stringer string conversion)
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

// SelectItem selects a given item and emits the index as the ComboSig signal
// and the selected item as the data
func (g *ComboBox) SelectItem(idx int) {
	updt := g.UpdateStart()
	g.SetCurIndex(idx)
	g.ComboSig.Emit(g.This, int64(g.CurIndex), g.CurVal)
	g.UpdateEnd(updt)
}

// MakeItemsMenu makes menu of all the items
func (g *ComboBox) MakeItemsMenu() {
	if g.ItemsMenu == nil {
		g.ItemsMenu = make(Menu, 0, len(g.Items))
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
		ac.SetSelectedState(i == g.CurIndex)
		ac.SetAsMenu()
		ac.ActionSig.ConnectOnly(g.This, func(recv, send ki.Ki, sig int64, data interface{}) {
			idx := data.(int)
			cb := recv.(*ComboBox)
			cb.SelectItem(idx)
		})
	}
}
