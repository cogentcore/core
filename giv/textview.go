// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package giv

import (
	"bytes"
	"image"
	"log"
	"strings"
	"time"
	"unicode"

	"github.com/alecthomas/chroma"
	"github.com/alecthomas/chroma/formatters/html"
	"github.com/alecthomas/chroma/lexers"
	"github.com/alecthomas/chroma/styles"

	"github.com/chewxy/math32"
	"github.com/goki/gi"
	"github.com/goki/gi/complete"
	"github.com/goki/gi/oswin"
	"github.com/goki/gi/oswin/cursor"
	"github.com/goki/gi/oswin/key"
	"github.com/goki/gi/oswin/mimedata"
	"github.com/goki/gi/oswin/mouse"
	"github.com/goki/gi/units"
	"github.com/goki/ki"
	"github.com/goki/ki/kit"
)

// TextView is a widget for editing multiple lines of text (as compared to
// TextField for a single line).  The underlying data model is just plain
// simple lines (ended by \n) with any number of characters per line.  These
// lines are displayed using wrap-around text into the editor.  Currently only
// works on in-memory strings.  Set the min
type TextView struct {
	gi.WidgetBase
	Buf         *TextBuf                  `json:"-" xml:"-" desc:"the text buffer that we're editing"`
	Placeholder string                    `json:"-" xml:"placeholder" desc:"text that is displayed when the field is empty, in a lower-contrast manner"`
	TabWidth    int                       `desc:"how many spaces is a tab"`
	HiLang      string                    `desc:"language for syntax highlighting the code"`
	HiStyle     string                    `desc:"syntax highlighting style"`
	HiCSS       gi.StyleSheet             `json:"-" xml:"-" desc:"CSS StyleSheet for given highlighting style"`
	Edited      bool                      `json:"-" xml:"-" desc:"true if the text has been edited relative to the original"`
	FocusActive bool                      `json:"-" xml:"-" desc:"true if the keyboard focus is active or not -- when we lose active focus we apply changes"`
	NLines      int                       `json:"-" xml:"-" desc:"number of lines in the view"`
	Markup      [][]byte                  `json:"-" xml:"-" desc:"marked-up version of the edit text lines, after being run through the syntax highlighting process -- this is what is actually rendered"`
	Renders     []gi.TextRender           `json:"-" xml:"-" desc:"renders of the text lines, with one render per line (each line could visibly wrap-around, so these are logical lines, not display lines)"`
	Offs        []float32                 `json:"-" xml:"-" desc:"starting offsets for top of each line"`
	LinesSize   gi.Vec2D                  `json:"-" xml:"-" desc:"total size of all lines as rendered"`
	RenderSz    gi.Vec2D                  `json:"-" xml:"-" desc:"size params to use in render call"`
	MaxWidthReq int                       `desc:"maximum width that field will request, in characters, during Size2D process -- if 0 then is 50 -- ensures that large strings don't request super large values -- standard max-width can override"`
	CursorPos   TextPos                   `json:"-" xml:"-" desc:"current cursor position"`
	CursorCol   int                       `json:"-" xml:"-" desc:"desired cursor column -- where the cursor was last when moved using left / right arrows -- used when doing up / down to not always go to short line columns"`
	CharWidth   int                       `xml:"-" desc:"approximate number of chars that can be displayed at any time -- computed from font size etc"`
	SelectStart TextPos                   `xml:"-" desc:"starting position of selection"`
	SelectEnd   TextPos                   `xml:"-" desc:"ending position of selection"`
	SelectMode  bool                      `xml:"-" desc:"if true, select text as cursor moves"`
	TextViewSig ki.Signal                 `json:"-" xml:"-" view:"-" desc:"signal for text viewt -- see TextViewSignals for the types"`
	StateStyles [TextViewStatesN]gi.Style `json:"-" xml:"-" desc:"normal style and focus style"`
	FontHeight  float32                   `json:"-" xml:"-" desc:"font height, cached during styling"`
	BlinkOn     bool                      `json:"-" xml:"-" oscillates between on and off for blinking"`
	Completion  *gi.Complete              `json:"-" xml:"-" desc:"functions and data for textfield completion"`
	// chroma highlighting
	lastHiLang  string
	lastHiStyle string
	lexer       chroma.Lexer
	formatter   *html.Formatter
	style       *chroma.Style
}

var KiT_TextView = kit.Types.AddType(&TextView{}, TextViewProps)

var TextViewProps = ki.Props{
	"font-family":      "Go Mono",
	"border-width":     units.NewValue(1, units.Px), // this also determines the cursor
	"border-color":     &gi.Prefs.Colors.Border,
	"border-style":     gi.BorderSolid,
	"padding":          units.NewValue(2, units.Px),
	"margin":           units.NewValue(2, units.Px),
	"vertical-align":   gi.AlignTop,
	"text-align":       gi.AlignLeft,
	"color":            &gi.Prefs.Colors.Font,
	"background-color": &gi.Prefs.Colors.Control,
	TextViewSelectors[TextViewActive]: ki.Props{
		"background-color": "lighter-0",
	},
	TextViewSelectors[TextViewFocus]: ki.Props{
		"border-width":     units.NewValue(2, units.Px),
		"background-color": "samelight-80",
	},
	TextViewSelectors[TextViewInactive]: ki.Props{
		"background-color": "highlight-10",
	},
	TextViewSelectors[TextViewSel]: ki.Props{
		"background-color": &gi.Prefs.Colors.Select,
	},
}

// TextViewSignals are signals that text view can send
type TextViewSignals int64

const (
	// return was pressed and an edit was completed -- data is the text
	TextViewDone TextViewSignals = iota

	// some text was selected (for Inactive state, selection is via WidgetSig)
	TextViewSelected

	TextViewSignalsN
)

//go:generate stringer -type=TextViewSignals

// TextViewStates are mutually-exclusive textfield states -- determines appearance
type TextViewStates int32

const (
	// normal state -- there but not being interacted with
	TextViewActive TextViewStates = iota

	// textfield is the focus -- will respond to keyboard input
	TextViewFocus

	// inactive -- not editable
	TextViewInactive

	// selected -- for inactive state, can select entire element
	TextViewSel

	TextViewStatesN
)

//go:generate stringer -type=TextViewStates

// Style selector names for the different states
var TextViewSelectors = []string{":active", ":focus", ":inactive", ":selected"}

// Label returns the display label for this node, satisfying the Labeler interface
func (tv *TextView) Label() string {
	return tv.Nm
}

// EditDone completes editing and copies the active edited text to the text --
// called when the return key is pressed or goes out of focus
func (tv *TextView) EditDone() {
	if tv.Buf != nil {
		tv.Buf.EditDone()
	}
	tv.ClearSelected()
}

// Revert aborts editing and reverts to last saved text
func (tv *TextView) Revert() {
	updt := tv.UpdateStart()
	defer tv.UpdateEnd(updt)
	tv.Edited = false
	tv.LayoutLines()
	// todo: signal buffer?
	tv.SelectReset()
}

//////////////////////////////////////////////////////////////////////////////////////////
//  Buffer communication

// SetBuf sets the TextBuf that this is a view of, and interconnects their signals
func (tv *TextView) SetBuf(buf *TextBuf) {
	tv.Buf = buf
	buf.AddView(tv)
	tv.Revert()
}

// TextViewBufSigRecv receives a signal from the buffer and updates view accordingly
func TextViewBufSigRecv(rvwki, sbufki ki.Ki, sig int64, data interface{}) {
	tv := rvwki.Embed(KiT_TextView).(*TextView)
	switch TextBufSignals(sig) {
	case TextBufDone:
	case TextBufNew:
		tv.LayoutLines()
		tv.UpdateSig()
	case TextBufInsert:
		tbe := data.(*TextBufEdit)
		tv.LayoutLines() // todo: optimized!
		tv.RenderLines(tbe.Start.Ln, tbe.End.Ln)
	case TextBufDelete:
		tbe := data.(*TextBufEdit)
		tv.LayoutLines() // todo: optimized!
		tv.RenderLines(tbe.Start.Ln, tbe.End.Ln)
	}
}

//////////////////////////////////////////////////////////////////////////////////////////
//  Text formatting and rendering

// HasHi returns true if there are highighting parameters set
func (tv *TextView) HasHi() bool {
	if tv.HiLang == "" || tv.HiStyle == "" {
		return false
	}
	return true
}

// HiInit initializes the syntax highlighting for current Hi params
func (tv *TextView) HiInit() {
	if !tv.HasHi() {
		return
	}
	if tv.HiLang == tv.lastHiLang && tv.HiStyle == tv.lastHiStyle {
		return
	}
	tv.lexer = chroma.Coalesce(lexers.Get(tv.HiLang))
	tv.formatter = html.New(html.WithClasses(), html.TabWidth(tv.TabWidth))
	tv.style = styles.Get(tv.HiStyle)
	if tv.style == nil {
		tv.style = styles.Fallback
	}
	var cssBuf bytes.Buffer
	err := tv.formatter.WriteCSS(&cssBuf, tv.style)
	if err != nil {
		log.Println(err)
		return
	}
	csstr := cssBuf.String()
	csstr = strings.Replace(csstr, " .chroma .", " .", -1)
	// lnidx := strings.Index(csstr, "\n")
	// csstr = csstr[lnidx+1:]
	tv.HiCSS.ParseString(csstr)
	tv.CSS = tv.HiCSS.CSSProps()

	if chp, ok := ki.SubProps(tv.CSS, ".chroma"); ok {
		for ky, vl := range chp { // apply to top level
			tv.SetProp(ky, vl)
		}
	}

	tv.lastHiLang = tv.HiLang
	tv.lastHiStyle = tv.HiStyle
}

// RenderSize is the size we should pass to text rendering, based on alloc
func (tv *TextView) RenderSize() gi.Vec2D {
	st := &tv.Sty
	st.Font.LoadFont(&st.UnContext)
	tv.FontHeight = st.Font.Height
	spc := tv.Sty.BoxSpace()
	sz := tv.LayData.AllocSize
	if sz.IsZero() {
		sz = tv.LayData.SizePrefOrMax()
	}
	if !sz.IsZero() {
		sz.SetSubVal(2 * spc)
	}
	tv.RenderSz = sz
	return sz
}

// LayoutLines generates TextRenders of lines from our TextBuf, using any
// highlighter that might be present
func (tv *TextView) LayoutLines() {
	if tv.Buf == nil || tv.Buf.NLines == 0 {
		tv.NLines = 0
		tv.LinesSize = gi.Vec2DZero
		return
	}

	tv.HiInit()

	tv.NLines = tv.Buf.NLines
	tv.Markup = make([][]byte, tv.NLines)
	tv.Renders = make([]gi.TextRender, tv.NLines)
	tv.Offs = make([]float32, tv.NLines)

	if tv.HasHi() {
		var htmlBuf bytes.Buffer
		iterator, err := tv.lexer.Tokenise(nil, string(tv.Buf.Txt)) // todo: unfortunate conversion here..
		err = tv.formatter.Format(&htmlBuf, tv.style, iterator)
		if err != nil {
			log.Println(err)
			return
		}
		mtlns := bytes.Split(htmlBuf.Bytes(), []byte("\n"))

		maxln := len(mtlns) - 1
		for ln := 0; ln < maxln; ln++ {
			mt := mtlns[ln]
			if ln == 0 {
				mt = bytes.TrimPrefix(mt, []byte(`<pre class="chroma">`))
			}
			mt = bytes.TrimPrefix(mt, []byte(`</span>`)) // leftovers
			tv.Markup[ln] = mt
		}
	} else {
		for ln := 0; ln < tv.NLines; ln++ {
			tv.Markup[ln] = []byte(string(tv.Buf.Lines[ln]))
		}
	}

	sz := tv.RenderSize()
	st := &tv.Sty
	off := float32(0)
	mxwd := float32(0)
	for ln := 0; ln < tv.NLines; ln++ {
		tv.Renders[ln].SetHTMLPre(tv.Markup[ln], &st.Font, &st.UnContext, tv.CSS)
		tv.Renders[ln].LayoutStdLR(&st.Text, &st.Font, &st.UnContext, sz)
		tv.Offs[ln] = off
		lsz := tv.Renders[ln].Size.Y
		if lsz < tv.FontHeight {
			lsz = tv.FontHeight
		}
		off += lsz
		mxwd = gi.Max32(mxwd, tv.Renders[ln].Size.X)
	}
	tv.LinesSize.Set(mxwd, off)
}

// LayoutLine generates render of given line (including highlighting)
func (tv *TextView) LayoutLine(ln int) {
	if tv.HasHi() {
		var htmlBuf bytes.Buffer
		iterator, err := tv.lexer.Tokenise(nil, string(tv.Buf.Lines[ln]))
		err = tv.formatter.Format(&htmlBuf, tv.style, iterator)
		if err != nil {
			log.Println(err)
			return
		}
		tv.Markup[ln] = htmlBuf.Bytes()
	} else {
		tv.Markup[ln] = []byte(string(tv.Buf.Lines[ln]))
	}

	st := &tv.Sty
	tv.Renders[ln].SetHTMLPre(tv.Markup[ln], &st.Font, &st.UnContext, tv.CSS)
	tv.Renders[ln].LayoutStdLR(&st.Text, &st.Font, &st.UnContext, tv.RenderSz)
}

//////////////////////////////////////////////////////////////////////////////////////////
//  Cursor Navigation

// CursorSelect updates selection based on cursor movements, given starting
// cursor position and tv.CursorPos is current
func (tv *TextView) CursorSelect(org TextPos) {
	if !tv.SelectMode {
		return
	}
	if org.IsLess(tv.SelectStart) {
		tv.SelectStart = tv.CursorPos
	} else if !tv.CursorPos.IsLess(tv.SelectStart) { // >
		tv.SelectEnd = tv.CursorPos
	} else {
		tv.SelectStart = tv.CursorPos
	}
	tv.SelectUpdate()
}

// CursorForward moves the cursor forward
func (tv *TextView) CursorForward(steps int) {
	tv.RenderCursor(false)
	org := tv.CursorPos
	for i := 0; i < steps; i++ {
		tv.CursorPos.Ch++
		if tv.CursorPos.Ch > len(tv.Buf.Lines[tv.CursorPos.Ln]) {
			if tv.CursorPos.Ln < tv.NLines-1 {
				tv.CursorPos.Ch = 0
				tv.CursorPos.Ln++
			} else {
				tv.CursorPos.Ch = len(tv.Buf.Lines[tv.CursorPos.Ln])
			}
		}
	}
	tv.CursorCol = tv.CursorPos.Ch
	tv.RenderCursor(true)
	tv.CursorSelect(org)
}

// CursorDown moves the cursor down line(s)
func (tv *TextView) CursorDown(steps int) {
	tv.RenderCursor(false)
	org := tv.CursorPos
	tv.CursorPos.Ln += steps
	if tv.CursorPos.Ln >= tv.NLines {
		tv.CursorPos.Ln = tv.NLines - 1
	}
	tv.CursorPos.Ch = gi.MinInt(len(tv.Buf.Lines[tv.CursorPos.Ln]), tv.CursorCol)
	tv.RenderCursor(true)
	tv.CursorSelect(org)
}

// CursorBackward moves the cursor backward
func (tv *TextView) CursorBackward(steps int) {
	tv.RenderCursor(false)
	org := tv.CursorPos
	for i := 0; i < steps; i++ {
		tv.CursorPos.Ch--
		if tv.CursorPos.Ch < 0 {
			if tv.CursorPos.Ln > 0 {
				tv.CursorPos.Ln--
				tv.CursorPos.Ch = len(tv.Buf.Lines[tv.CursorPos.Ln])
			} else {
				tv.CursorPos.Ch = 0
			}
		}
	}
	tv.CursorCol = tv.CursorPos.Ch
	tv.RenderCursor(true)
	tv.CursorSelect(org)
}

// CursorUp moves the cursor up line(s)
func (tv *TextView) CursorUp(steps int) {
	tv.RenderCursor(false)
	org := tv.CursorPos
	tv.CursorPos.Ln -= steps
	if tv.CursorPos.Ln < 0 {
		tv.CursorPos.Ln = 0
	}
	tv.CursorPos.Ch = gi.MinInt(len(tv.Buf.Lines[tv.CursorPos.Ln]), tv.CursorCol)
	tv.RenderCursor(true)
	tv.CursorSelect(org)
}

// CursorStartLine moves the cursor to the start of the line, updating selection
// if select mode is active
func (tv *TextView) CursorStartLine() {
	tv.RenderCursor(false)
	org := tv.CursorPos
	tv.CursorPos.Ch = 0
	tv.CursorCol = tv.CursorPos.Ch
	tv.RenderCursor(true)
	tv.CursorSelect(org)
}

// CursorStart moves the cursor to the start of the text, updating selection
// if select mode is active
func (tv *TextView) CursorStart() {
	tv.RenderCursor(false)
	org := tv.CursorPos
	tv.CursorPos.Ln = 0
	tv.CursorPos.Ch = 0
	tv.CursorCol = tv.CursorPos.Ch
	tv.RenderCursor(true)
	tv.CursorSelect(org)
}

// CursorEndLine moves the cursor to the end of the text
func (tv *TextView) CursorEndLine() {
	tv.RenderCursor(false)
	org := tv.CursorPos
	tv.CursorPos.Ch = len(tv.Buf.Lines[tv.CursorPos.Ln])
	tv.CursorCol = tv.CursorPos.Ch
	tv.RenderCursor(true)
	tv.CursorSelect(org)
}

// CursorEnd moves the cursor to the end of the text, updating selection if
// select mode is active
func (tv *TextView) CursorEnd() {
	updt := tv.UpdateStart()
	defer tv.UpdateEnd(updt)
	tv.RenderCursor(false)
	org := tv.CursorPos
	tv.CursorPos.Ln = gi.MaxInt(tv.NLines-1, 0)
	tv.CursorPos.Ch = len(tv.Buf.Lines[tv.CursorPos.Ln])
	tv.CursorCol = tv.CursorPos.Ch
	tv.RenderCursor(true)
	tv.CursorSelect(org)
}

// todo: ctrl+backspace = delete word
// shift+arrow = select
// uparrow = start / down = end

// CursorBackspace deletes character(s) immediately before cursor
func (tv *TextView) CursorBackspace(steps int) {
	if tv.HasSelection() {
		tv.DeleteSelection()
		return
	}
	// note: no update b/c signal from buf will drive update
	org := tv.CursorPos
	tv.RenderCursor(false)
	tv.CursorBackward(steps)
	tv.RenderCursor(true)
	tv.Buf.DeleteText(tv.CursorPos, org)
}

// CursorDelete deletes character(s) immediately after the cursor
func (tv *TextView) CursorDelete(steps int) {
	if tv.HasSelection() {
		tv.DeleteSelection()
	}
	// note: no update b/c signal from buf will drive update
	org := tv.CursorPos
	tv.RenderCursor(false)
	tv.CursorForward(steps)
	tv.RenderCursor(true)
	tv.Buf.DeleteText(org, tv.CursorPos)
}

// CursorKill deletes text from cursor to end of text
func (tv *TextView) CursorKill() {
	org := tv.CursorPos
	tv.CursorEnd()
	tv.Buf.DeleteText(org, tv.CursorPos)
}

// ClearSelected resets both the global selected flag and any current selection
func (tv *TextView) ClearSelected() {
	tv.WidgetBase.ClearSelected()
	tv.SelectReset()
}

// HasSelection returns whether there is a selected region of text
func (tv *TextView) HasSelection() bool {
	tv.SelectUpdate()
	if tv.SelectStart.IsLess(tv.SelectEnd) {
		return true
	}
	return false
}

// Selection returns the currently selected text as a TextBufEdit, which
// captures start, end, and full lines in between -- nil if no selection
func (tv *TextView) Selection() *TextBufEdit {
	if tv.HasSelection() {
		return tv.Buf.Region(tv.SelectStart, tv.SelectEnd)
	}
	return nil
}

// SelectModeToggle toggles the SelectMode, updating selection with cursor movement
func (tv *TextView) SelectModeToggle() {
	if tv.SelectMode {
		tv.SelectMode = false
	} else {
		tv.SelectMode = true
		tv.SelectStart = tv.CursorPos
		tv.SelectEnd = tv.SelectStart
	}
}

// SelectAll selects all the text
func (tv *TextView) SelectAll() {
	updt := tv.UpdateStart()
	tv.SelectStart = TextPosZero
	tv.SelectEnd = TextPos{tv.Buf.NLines - 1, len(tv.Buf.Lines[tv.Buf.NLines-1])}
	tv.UpdateEnd(updt)
}

// IsWordBreak defines what counts as a word break for the purposes of selecting words
func (tv *TextView) IsWordBreak(r rune) bool {
	if unicode.IsSpace(r) || unicode.IsSymbol(r) || unicode.IsPunct(r) {
		return true
	}
	return false
}

// SelectWord selects the word (whitespace delimited) that the cursor is on
func (tv *TextView) SelectWord() {
	// updt := tv.UpdateStart()
	// defer tv.UpdateEnd(updt)
	// sz := len(tv.EditTxt)
	// if sz <= 3 {
	// 	tv.SelectAll()
	// 	return
	// }
	// tv.SelectStart = tv.CursorPos
	// if tv.SelectStart >= sz {
	// 	tv.SelectStart = sz - 2
	// }
	// if !tv.IsWordBreak(tv.EditTxt[tv.SelectStart]) {
	// 	for tv.SelectStart > 0 {
	// 		if tv.IsWordBreak(tv.EditTxt[tv.SelectStart-1]) {
	// 			break
	// 		}
	// 		tv.SelectStart--
	// 	}
	// 	tv.SelectEnd = tv.CursorPos + 1
	// 	for tv.SelectEnd < sz {
	// 		if tv.IsWordBreak(tv.EditTxt[tv.SelectEnd]) {
	// 			break
	// 		}
	// 		tv.SelectEnd++
	// 	}
	// } else { // keep the space start -- go to next space..
	// 	tv.SelectEnd = tv.CursorPos + 1
	// 	for tv.SelectEnd < sz {
	// 		if !tv.IsWordBreak(tv.EditTxt[tv.SelectEnd]) {
	// 			break
	// 		}
	// 		tv.SelectEnd++
	// 	}
	// 	for tv.SelectEnd < sz {
	// 		if tv.IsWordBreak(tv.EditTxt[tv.SelectEnd]) {
	// 			break
	// 		}
	// 		tv.SelectEnd++
	// 	}
	// }
}

// SelectReset resets the selection
func (tv *TextView) SelectReset() {
	tv.SelectMode = false
	zp := TextPosZero
	if tv.SelectStart == zp && tv.SelectEnd == zp {
		return
	}
	updt := tv.UpdateStart()
	tv.SelectStart = zp
	tv.SelectEnd = zp
	tv.UpdateEnd(updt)
}

// SelectUpdate updates the select region after any change to the text, to keep it in range
func (tv *TextView) SelectUpdate() {
	// if tv.SelectStart < tv.SelectEnd {
	// 	ed := len(tv.EditTxt)
	// 	if tv.SelectStart < 0 {
	// 		tv.SelectStart = 0
	// 	}
	// 	if tv.SelectEnd > ed {
	// 		tv.SelectEnd = ed
	// 	}
	// } else {
	// 	tv.SelectReset()
	// }
}

// Cut cuts any selected text and adds it to the clipboard, also returns cut text
func (tv *TextView) Cut() *TextBufEdit {
	cut := tv.DeleteSelection()
	if cut != nil {
		oswin.TheApp.ClipBoard().Write(mimedata.NewTextBytes(cut.ToBytes()))
	}
	return cut
}

// DeleteSelection deletes any selected text, without adding to clipboard --
// returns text deleted as TextBufEdit (nil if none)
func (tv *TextView) DeleteSelection() *TextBufEdit {
	tbe := tv.Selection()
	if tbe == nil {
		return nil
	}
	tv.Buf.DeleteText(tv.SelectStart, tv.SelectEnd)
	tv.SelectReset()
	return tbe
}

// Copy copies any selected text to the clipboard, and returns that text,
// optionaly resetting the current selection
func (tv *TextView) Copy(reset bool) *TextBufEdit {
	tbe := tv.Selection()
	if tbe == nil {
		return nil
	}
	oswin.TheApp.ClipBoard().Write(mimedata.NewTextBytes(tbe.ToBytes()))
	if reset {
		tv.SelectReset()
	}
	return tbe
}

// Paste inserts text from the clipboard at current cursor position -- if
// cursor is within a current selection, that selection is
func (tv *TextView) Paste() {
	data := oswin.TheApp.ClipBoard().Read([]string{mimedata.TextPlain})
	if data != nil {
		if tv.SelectStart.IsLess(tv.CursorPos) && tv.CursorPos.IsLess(tv.SelectEnd) {
			tv.DeleteSelection()
		}
		tv.InsertAtCursor(data.TypeData(mimedata.TextPlain))
	}
}

// InsertAtCursor inserts given text at current cursor position
func (tv *TextView) InsertAtCursor(txt []byte) {
	updt := tv.UpdateStart()
	defer tv.UpdateEnd(updt)
	if tv.HasSelection() {
		tv.Cut()
	}
	tbe := tv.Buf.InsertText(tv.CursorPos, txt)
	tv.CursorPos = tbe.End
}

func (tv *TextView) MakeContextMenu(m *gi.Menu) {
	cpsc := gi.ActiveKeyMap.ChordForFun(gi.KeyFunCopy)
	ac := m.AddAction(gi.ActOpts{Label: "Copy", Shortcut: cpsc},
		tv.This, func(recv, send ki.Ki, sig int64, data interface{}) {
			txf := recv.Embed(KiT_TextView).(*TextView)
			txf.Copy(true)
		})
	ac.SetActiveState(tv.HasSelection())
	if !tv.IsInactive() {
		ctsc := gi.ActiveKeyMap.ChordForFun(gi.KeyFunCut)
		ptsc := gi.ActiveKeyMap.ChordForFun(gi.KeyFunPaste)
		ac = m.AddAction(gi.ActOpts{Label: "Cut", Shortcut: ctsc},
			tv.This, func(recv, send ki.Ki, sig int64, data interface{}) {
				txf := recv.Embed(KiT_TextView).(*TextView)
				txf.Cut()
			})
		ac.SetActiveState(tv.HasSelection())
		ac = m.AddAction(gi.ActOpts{Label: "Paste", Shortcut: ptsc},
			tv.This, func(recv, send ki.Ki, sig int64, data interface{}) {
				txf := recv.Embed(KiT_TextView).(*TextView)
				txf.Paste()
			})
		ac.SetInactiveState(oswin.TheApp.ClipBoard().IsEmpty())
	}
}

// OfferCompletions pops up a menu of possible completions
func (tv *TextView) OfferCompletions() {
	if tv.Completion == nil {
		return
	}
	// win := tv.ParentWindow()
	// if gi.PopupIsCompleter(win.Popup) {
	// 	win.ClosePopup(win.Popup)
	// }

	// s := string(tv.EditTxt[0:tv.CursorPos])
	// cpos := tv.CharStartPos(tv.CursorPos).ToPoint()

	// tv.Completion.ShowCompletions(s, tv.Viewport, cpos.X+5, cpos.Y+10)
}

// Complete edits the text field using the string chosen from the completion menu
func (tv *TextView) Complete(str string) {
	// txt := string(tv.EditTxt) // John: do NOT call tv.Text() in an active editing context!!!
	// s, delta := tv.Completion.EditFunc(txt, tv.CursorPos, str, tv.Completion.Seed)
	// tv.EditTxt = []rune(s)
	// tv.CursorForward(delta)
}

// PixelToCursor finds the cursor position that corresponds to the given pixel location
func (tv *TextView) PixelToCursor(pixOff float32) TextPos {
	// st := &tv.Sty

	// spc := st.BoxSpace()
	// px := pixOff - spc

	// // for selection to work correctly, we need this to be deterministic

	// sz := len(tv.EditTxt)
	// c := tv.StartPos + int(float64(px/st.UnContext.ToDotsFactor(units.Ch)))
	// c = kit.MinInt(c, sz)

	// w := tv.TextWidth(tv.StartPos, c)
	// if w > px {
	// 	for w > px {
	// 		c--
	// 		if c <= tv.StartPos {
	// 			c = tv.StartPos
	// 			break
	// 		}
	// 		w = tv.TextWidth(tv.StartPos, c)
	// 	}
	// } else if w < px {
	// 	for c < tv.EndPos {
	// 		wn := tv.TextWidth(tv.StartPos, c+1)
	// 		if wn > px {
	// 			break
	// 		} else if wn == px {
	// 			c++
	// 			break
	// 		}
	// 		c++
	// 	}
	// }
	// return c
	return TextPosZero
}

func (tv *TextView) SetCursorFromPixel(pixOff float32, selMode mouse.SelectModes) {
	updt := tv.UpdateStart()
	defer tv.UpdateEnd(updt)
	oldPos := tv.CursorPos
	tv.CursorPos = tv.PixelToCursor(pixOff)
	if tv.SelectMode || selMode != mouse.NoSelectMode {
		if !tv.SelectMode && selMode != mouse.NoSelectMode {
			tv.SelectStart = oldPos
			tv.SelectMode = true
		}
		if !tv.IsDragging() && tv.SelectStart.IsLess(tv.CursorPos) && tv.CursorPos.IsLess(tv.SelectEnd) {
			tv.SelectReset()
		} else if tv.SelectStart.IsLess(tv.CursorPos) {
			tv.SelectEnd = tv.CursorPos
		} else {
			tv.SelectStart = tv.CursorPos
		}
		tv.SelectUpdate()
	} else if tv.HasSelection() {
		tv.SelectReset()
	}
}

// KeyInput handles keyboard input into the text field and from the completion menu
func (tv *TextView) KeyInput(kt *key.ChordEvent) {
	kf := gi.KeyFun(kt.ChordString())
	win := tv.ParentWindow()

	if gi.PopupIsCompleter(win.Popup) {
		switch kf {
		case gi.KeyFunFocusNext: // tab will complete if single item or try to extend if multiple items
			count := len(tv.Completion.Completions)
			if count > 0 {
				if count == 1 { // just complete
					tv.Complete(tv.Completion.Completions[0])
					win.ClosePopup(win.Popup)
				} else { // try to extend the seed
					s := complete.ExtendSeed(tv.Completion.Completions, tv.Completion.Seed)
					if s != "" {
						win.ClosePopup(win.Popup)
						tv.InsertAtCursor([]byte(s))
						tv.OfferCompletions()
					}
				}
			}
			kt.SetProcessed()
		default:
			//fmt.Printx("some char\n")
		}
	}

	if kt.IsProcessed() {
		return
	}

	// first all the keys that work for both inactive and active
	switch kf {
	case gi.KeyFunMoveRight:
		kt.SetProcessed()
		tv.CursorForward(1)
		tv.OfferCompletions()
	case gi.KeyFunMoveLeft:
		kt.SetProcessed()
		tv.CursorBackward(1)
		tv.OfferCompletions()
	case gi.KeyFunMoveUp:
		kt.SetProcessed()
		tv.CursorUp(1)
	case gi.KeyFunMoveDown:
		kt.SetProcessed()
		tv.CursorDown(1)
	case gi.KeyFunHome:
		kt.SetProcessed()
		tv.CursorStartLine()
	case gi.KeyFunEnd:
		kt.SetProcessed()
		tv.CursorEndLine()
	case gi.KeyFunSelectMode:
		kt.SetProcessed()
		tv.SelectModeToggle()
	case gi.KeyFunCancelSelect:
		kt.SetProcessed()
		tv.SelectReset()
	case gi.KeyFunSelectAll:
		kt.SetProcessed()
		tv.SelectAll()
	case gi.KeyFunCopy:
		kt.SetProcessed()
		tv.Copy(true) // reset
	}
	if tv.IsInactive() || kt.IsProcessed() {
		return
	}
	switch kf {
	case gi.KeyFunSelectItem: // enter
		fallthrough
	case gi.KeyFunAccept: // ctrl+enter
		tv.EditDone()
		kt.SetProcessed()
		tv.FocusNext()
	case gi.KeyFunAbort: // esc
		tv.Revert()
		kt.SetProcessed()
		tv.FocusNext()
	case gi.KeyFunBackspace:
		kt.SetProcessed()
		tv.CursorBackspace(1)
		tv.OfferCompletions()
	case gi.KeyFunKill:
		kt.SetProcessed()
		tv.CursorKill()
	case gi.KeyFunDelete:
		kt.SetProcessed()
		tv.CursorDelete(1)
	case gi.KeyFunCut:
		kt.SetProcessed()
		tv.Cut()
	case gi.KeyFunPaste:
		kt.SetProcessed()
		tv.Paste()
	case gi.KeyFunComplete:
		kt.SetProcessed()
		tv.OfferCompletions()
	case gi.KeyFunNil:
		if unicode.IsPrint(kt.Rune) {
			if !kt.HasAnyModifier(key.Control, key.Meta) {
				kt.SetProcessed()
				tv.InsertAtCursor([]byte(string(kt.Rune)))
				tv.OfferCompletions()
			}
		}
	}
}

// MouseEvent handles the mouse.Event
func (tv *TextView) MouseEvent(me *mouse.Event) {
	if !tv.IsInactive() && !tv.HasFocus() {
		tv.GrabFocus()
	}
	me.SetProcessed()
	switch me.Button {
	case mouse.Left:
		if me.Action == mouse.Press {
			if tv.IsInactive() {
				tv.SetSelectedState(!tv.IsSelected())
				tv.EmitSelectedSignal()
				tv.UpdateSig()
			} else {
				pt := tv.PointToRelPos(me.Pos())
				tv.SetCursorFromPixel(float32(pt.X), me.SelectMode())
			}
		} else if me.Action == mouse.DoubleClick {
			me.SetProcessed()
			// if tv.HasSelection() {
			// 	if tv.SelectStart == TextPosZero && tv.SelectEnd == tv.Buf.EndPos() {
			// 		tv.SelectReset()
			// 	} else {
			// 		tv.SelectAll()
			// 	}
			// } else {
			tv.SelectWord()
			// }
		}
	case mouse.Middle:
		if !tv.IsInactive() && me.Action == mouse.Press {
			me.SetProcessed()
			pt := tv.PointToRelPos(me.Pos())
			tv.SetCursorFromPixel(float32(pt.X), me.SelectMode())
			tv.Paste()
		}
	case mouse.Right:
		if me.Action == mouse.Press {
			me.SetProcessed()
			tv.EmitContextMenuSignal()
			tv.This.(gi.Node2D).ContextMenu()
		}
	}
}

func (tv *TextView) TextViewEvents() {
	tv.HoverTooltipEvent()
	tv.ConnectEvent(oswin.MouseDragEvent, gi.RegPri, func(recv, send ki.Ki, sig int64, d interface{}) {
		me := d.(*mouse.DragEvent)
		me.SetProcessed()
		txf := recv.Embed(KiT_TextView).(*TextView)
		if !txf.SelectMode {
			txf.SelectModeToggle()
		}
		pt := txf.PointToRelPos(me.Pos())
		txf.SetCursorFromPixel(float32(pt.X), mouse.NoSelectMode)
	})
	tv.ConnectEvent(oswin.MouseEvent, gi.RegPri, func(recv, send ki.Ki, sig int64, d interface{}) {
		txf := recv.Embed(KiT_TextView).(*TextView)
		me := d.(*mouse.Event)
		txf.MouseEvent(me)
	})
	tv.ConnectEvent(oswin.MouseFocusEvent, gi.RegPri, func(recv, send ki.Ki, sig int64, d interface{}) {
		txf := recv.Embed(KiT_TextView).(*TextView)
		if txf.IsInactive() {
			return
		}
		me := d.(*mouse.FocusEvent)
		me.SetProcessed()
		if me.Action == mouse.Enter {
			oswin.TheApp.Cursor().PushIfNot(cursor.IBeam)
		} else {
			oswin.TheApp.Cursor().PopIf(cursor.IBeam)
		}
	})
	tv.ConnectEvent(oswin.KeyChordEvent, gi.RegPri, func(recv, send ki.Ki, sig int64, d interface{}) {
		txf := recv.Embed(KiT_TextView).(*TextView)
		kt := d.(*key.ChordEvent)
		txf.KeyInput(kt)
	})
	if dlg, ok := tv.Viewport.This.(*gi.Dialog); ok {
		dlg.DialogSig.Connect(tv.This, func(recv, send ki.Ki, sig int64, data interface{}) {
			txf, _ := recv.Embed(KiT_TextView).(*TextView)
			if sig == int64(gi.DialogAccepted) {
				txf.EditDone()
			}
		})
	}
}

////////////////////////////////////////////////////
//  Node2D Interface

func (tv *TextView) Init2D() {
	tv.Init2DWidget()
}

func (tv *TextView) Style2D() {
	tv.HiInit()
	tv.SetCanFocusIfActive()
	tv.Style2DWidget()
	pst := &(tv.Par.(gi.Node2D).AsWidget().Sty)
	for i := 0; i < int(TextViewStatesN); i++ {
		tv.StateStyles[i].CopyFrom(&tv.Sty)
		tv.StateStyles[i].SetStyleProps(pst, tv.StyleProps(TextViewSelectors[i]))
		tv.StateStyles[i].StyleCSS(tv.This.(gi.Node2D), tv.CSSAgg, TextViewSelectors[i])
		tv.StateStyles[i].CopyUnitContext(&tv.Sty.UnContext)
	}
}

func (tv *TextView) Size2D(iter int) {
	// if len(tv.Txt) == 0 && len(tv.Placeholder) > 0 {
	// 	text = tv.Placeholder
	// } else {
	// 	text = tv.Txt
	// }
	// tv.Edited = false
	// maxlen := tv.MaxWidthReq
	// if maxlen <= 0 {
	// 	maxlen = 50
	// }
	tv.LayoutLines()
	tv.Size2DFromWH(tv.LinesSize.X, tv.LinesSize.Y)
}

func (tv *TextView) Layout2D(parBBox image.Rectangle, iter int) bool {
	tv.Layout2DBase(parBBox, true, iter) // init style
	for i := 0; i < int(TextViewStatesN); i++ {
		tv.StateStyles[i].CopyUnitContext(&tv.Sty.UnContext)
	}
	return tv.Layout2DChildren(iter)
}

// CharStartPos returns the starting render coords for the given position --
// makes no attempt to rationalize that pos (i.e., if not in visible range,
// position will be out of range too)
func (tv *TextView) CharStartPos(pos TextPos) gi.Vec2D {
	spos := tv.RenderStartPos()
	spos.Y += tv.Offs[pos.Ln]
	if len(tv.Renders[pos.Ln].Spans) > 0 {
		spos.X += tv.Renders[pos.Ln].Spans[0].RuneRelPos(pos.Ch).X
	}
	return spos
}

// TextViewBlinker is the time.Ticker for blinking cursors for text fields,
// only one of which can be active at at a time
var TextViewBlinker *time.Ticker

// BlinkingTextView is the text field that is blinking
var BlinkingTextView *TextView

// TextViewBlink is function that blinks text field cursor
func TextViewBlink() {
	for {
		if TextViewBlinker == nil {
			return // shutdown..
		}
		<-TextViewBlinker.C
		if BlinkingTextView == nil {
			continue
		}
		if BlinkingTextView.IsDestroyed() || BlinkingTextView.IsDeleted() {
			BlinkingTextView = nil
			continue
		}
		tv := BlinkingTextView
		if tv.Viewport == nil || !tv.HasFocus() || !tv.FocusActive || tv.VpBBox == image.ZR {
			BlinkingTextView = nil
			continue
		}
		win := tv.ParentWindow()
		if win == nil || win.IsResizing() {
			continue
		}
		tv.BlinkOn = !tv.BlinkOn
		tv.RenderCursor(tv.BlinkOn)
	}
}

func (tv *TextView) StartCursor() {
	tv.BlinkOn = true
	if gi.CursorBlinkMSec == 0 {
		tv.RenderCursor(true)
		return
	}
	if TextViewBlinker == nil {
		TextViewBlinker = time.NewTicker(time.Duration(gi.CursorBlinkMSec) * time.Millisecond)
		go TextViewBlink()
	}
	tv.BlinkOn = true
	win := tv.ParentWindow()
	if win != nil && !win.IsResizing() {
		tv.RenderCursor(true)
	}
	BlinkingTextView = tv
}

func (tv *TextView) StopCursor() {
	if BlinkingTextView == tv {
		BlinkingTextView = nil
	}
}

func (tv *TextView) RenderCursor(on bool) {
	if tv.PushBounds() {
		st := &tv.Sty
		cpos := tv.CharStartPos(tv.CursorPos)
		rs := &tv.Viewport.Render
		pc := &rs.Paint
		if on {
			pc.StrokeStyle.SetColor(&st.Font.Color)
		} else {
			pc.StrokeStyle.SetColor(&st.Font.BgColor.Color)
		}
		pc.StrokeStyle.Width = st.Border.Width
		if on {
			pc.StrokeStyle.Width.Dots -= .1 // try to get rid of halo
		}
		pc.DrawLine(rs, cpos.X, cpos.Y, cpos.X, cpos.Y+tv.FontHeight)
		pc.Stroke(rs)
		tv.PopBounds()

		// compute bbox just for the cursor
		cbmin := cpos.SubVal(st.Border.Width.Dots)
		cbmax := cpos.AddVal(st.Border.Width.Dots)
		cbmax.Y += tv.FontHeight
		curBBox := image.Rectangle{cbmin.ToPointFloor(), cbmax.ToPointCeil()}
		vprel := curBBox.Min.Sub(tv.VpBBox.Min)
		curWinBBox := tv.WinBBox.Add(vprel)

		vp := tv.Viewport
		updt := vp.Win.UpdateStart()
		vp.Win.UploadVpRegion(vp, curBBox, curWinBBox) // bigger than necc.
		vp.Win.UpdateEnd(updt)
	}
}

func (tv *TextView) RenderSelect() {
	if !tv.HasSelection() {
		return
	}

	// todo: requires separate logic for start / end lines of select
	spos := tv.CharStartPos(tv.SelectStart)
	epos := tv.CharStartPos(tv.SelectEnd)
	if epos.X < spos.X {
		tmp := spos.X
		spos.X = epos.X
		epos.X = tmp
	}

	rs := &tv.Viewport.Render
	pc := &rs.Paint
	st := &tv.StateStyles[TextViewSel]
	pc.FillBox(rs, spos, epos, &st.Font.BgColor)
}

// RenderStartPos is absolute rendering start position from our allocpos
func (tv *TextView) RenderStartPos() gi.Vec2D {
	st := &tv.Sty
	spc := st.BoxSpace()
	pos := tv.LayData.AllocPos.AddVal(spc)
	return pos
}

// RenderAllLines displays all the visible lines on the screen -- called
// during standard render
func (tv *TextView) RenderAllLines() {
	rs := &tv.Viewport.Render
	pos := tv.RenderStartPos()
	for ln := 0; ln < tv.NLines; ln++ {
		lst := pos.Y + tv.Offs[ln]
		led := lst + tv.Renders[ln].Size.Y
		if int(math32.Ceil(led)) < tv.VpBBox.Min.Y {
			continue
		}
		if int(math32.Floor(lst)) > tv.VpBBox.Max.Y {
			continue
		}
		lp := pos
		lp.Y = lst
		tv.Renders[ln].Render(rs, lp) // not top pos -- already has baseline offset
	}
}

// RenderLines displays a specific range of lines on the screen, also painting
// selection.  end is *inclusive* line.  returns false if nothing visible.
func (tv *TextView) RenderLines(st, ed int) bool {
	sty := &tv.Sty
	rs := &tv.Viewport.Render
	pc := &rs.Paint
	pos := tv.RenderStartPos()
	var boxMin, boxMax gi.Vec2D
	// first get the box to fill
	visSt := -1
	visEd := -1
	for ln := st; ln <= ed; ln++ {
		lst := pos.Y + tv.Offs[ln]
		led := lst + tv.Renders[ln].Size.Y
		if int(math32.Ceil(led)) < tv.VpBBox.Min.Y {
			continue
		}
		if int(math32.Floor(lst)) > tv.VpBBox.Max.Y {
			continue
		}
		lp := pos
		if visSt < 0 {
			visSt = ln
			lp.Y = lst
			boxMin = lp
		}
		visEd = ln // just keep updating
		lp.Y = led
		boxMax = lp
	}
	if visSt < 0 && visEd < 0 {
		return false
	}
	boxMax.X = float32(tv.VpBBox.Max.X) // go all the way
	pc.FillBox(rs, boxMin, boxMax.Sub(boxMin), &sty.Font.BgColor)
	// todo: selection!
	for ln := visSt; ln <= visEd; ln++ {
		lst := pos.Y + tv.Offs[ln]
		lp := pos
		lp.Y = lst
		tv.Renders[ln].Render(rs, lp) // not top pos -- already has baseline offset
	}
	return true
}

func (tv *TextView) Render2D() {
	if tv.FullReRenderIfNeeded() {
		return
	}
	if tv.PushBounds() {
		tv.TextViewEvents()
		if tv.IsInactive() {
			if tv.IsSelected() {
				tv.Sty = tv.StateStyles[TextViewSel]
			} else {
				tv.Sty = tv.StateStyles[TextViewInactive]
			}
		} else if tv.HasFocus() {
			if tv.FocusActive {
				tv.Sty = tv.StateStyles[TextViewFocus]
			} else {
				tv.Sty = tv.StateStyles[TextViewActive]
			}
		} else if tv.IsSelected() {
			tv.Sty = tv.StateStyles[TextViewSel]
		} else {
			tv.Sty = tv.StateStyles[TextViewActive]
		}
		st := &tv.Sty
		st.Font.LoadFont(&st.UnContext)
		tv.RenderStdBox(st)
		tv.RenderSelect()
		tv.RenderAllLines()
		if tv.HasFocus() && tv.FocusActive {
			tv.StartCursor()
		} else {
			tv.StopCursor()
		}
		tv.Render2DChildren()
		tv.PopBounds()
	} else {
		tv.DisconnectAllEvents(gi.RegPri)
	}
}

func (tv *TextView) FocusChanged2D(change gi.FocusChanges) {
	switch change {
	case gi.FocusLost:
		tv.FocusActive = false
		tv.EditDone()
		tv.UpdateSig()
	case gi.FocusGot:
		tv.FocusActive = true
		tv.ScrollToMe()
		//tv.CursorEnd()
		tv.EmitFocusedSignal()
		tv.UpdateSig()
	case gi.FocusInactive:
		tv.FocusActive = false
		tv.EditDone()
		tv.UpdateSig()
	case gi.FocusActive:
		tv.FocusActive = true
		tv.ScrollToMe()
		// tv.UpdateSig()
		// todo: see about cursor
	}
}

func (tv *TextView) SetCompleter(data interface{}, matchFun complete.MatchFunc, editFun complete.EditFunc) {
	if matchFun == nil || editFun == nil {
		if tv.Completion != nil {
			tv.Completion.CompleteSig.Disconnect(tv.This)
		}
		tv.Completion.Destroy()
		tv.Completion = nil
		return
	}
	tv.Completion = &gi.Complete{}
	tv.Completion.InitName(tv.Completion, "tv-completion") // needed for standalone Ki's
	tv.Completion.Context = data
	tv.Completion.MatchFunc = matchFun
	tv.Completion.EditFunc = editFun
	tv.Completion.CompleteSig.ConnectOnly(tv.This, func(recv, send ki.Ki, sig int64, data interface{}) {
		tvf, _ := recv.Embed(KiT_TextView).(*TextView)
		if sig == int64(gi.CompleteSelect) {
			tvf.Complete(data.(string))
		}
	})
}
