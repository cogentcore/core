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
	"github.com/goki/ki/bitflag"
	"github.com/goki/ki/kit"
)

// TextView is a widget for editing multiple lines of text (as compared to
// TextField for a single line).  The underlying data model is just plain
// simple lines (ended by \n) with any number of characters per line.  These
// lines are displayed using wrap-around text into the editor.  Currently only
// works on in-memory strings.  Set the min
type TextView struct {
	gi.WidgetBase
	Buf         *TextBuf        `json:"-" xml:"-" desc:"the text buffer that we're editing"`
	Placeholder string          `json:"-" xml:"placeholder" desc:"text that is displayed when the field is empty, in a lower-contrast manner"`
	TabWidth    int             `desc:"how many spaces is a tab"`
	HiLang      string          `desc:"language for syntax highlighting the code"`
	HiStyle     string          `desc:"syntax highlighting style"`
	HiCSS       gi.StyleSheet   `json:"-" xml:"-" desc:"CSS StyleSheet for given highlighting style"`
	FocusActive bool            `json:"-" xml:"-" desc:"true if the keyboard focus is active or not -- when we lose active focus we apply changes"`
	NLines      int             `json:"-" xml:"-" desc:"number of lines in the view"`
	Markup      [][]byte        `json:"-" xml:"-" desc:"marked-up version of the edit text lines, after being run through the syntax highlighting process -- this is what is actually rendered"`
	Render      []gi.TextRender `json:"-" xml:"-" desc:"render of the text lines, with one render per line (each line could visibly wrap-around, so these are logical lines, not display lines)"`
	Offs        []float32       `json:"-" xml:"-" desc:"starting offsets for top of each line"`
	LinesSize   gi.Vec2D        `json:"-" xml:"-" desc:"total size of all lines as rendered"`
	RenderSz    gi.Vec2D        `json:"-" xml:"-" desc:"size params to use in render call"`
	MaxWidthReq int             `desc:"maximum width that field will request, in characters, during Size2D process -- if 0 then is 50 -- ensures that large strings don't request super large values -- standard max-width can override"`
	CursorPos   int             `xml:"-" desc:"current cursor position"`
	StartPos    int
	EndPos      int
	RenderAll   gi.TextRender
	CharWidth   int                       `xml:"-" desc:"approximate number of chars that can be displayed at any time -- computed from font size etc"`
	SelectStart int                       `xml:"-" desc:"starting position of selection in the string"`
	SelectEnd   int                       `xml:"-" desc:"ending position of selection in the string"`
	SelectMode  bool                      `xml:"-" desc:"if true, select text as cursor moves"`
	TextViewSig ki.Signal                 `json:"-" xml:"-" view:"-" desc:"signal for text viewt -- see TextViewSignals for the types"`
	StateStyles [TextViewStatesN]gi.Style `json:"-" xml:"-" desc:"normal style and focus style"`
	FontHeight  float32                   `json:"-" xml:"-" desc:"font height, cached during styling"`
	BlinkOn     bool                      `json:"-" xml:"-" oscillates between on and off for blinking"`
	Completion  gi.Complete               `json:"-" xml:"-" desc:"functions and data for textfield completion"`
	// chroma highlighting
	lastHiLang  string
	lastHiStyle string
	lexer       chroma.Lexer
	formatter   *html.Formatter
	style       *chroma.Style
	EditTxt     []rune
	Edited      bool `json:"-" xml:"-" desc:"true if the text has been edited relative to the original"`
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
func (tx *TextView) Label() string {
	return tx.Nm
}

// EditDone completes editing and copies the active edited text to the text --
// called when the return key is pressed or goes out of focus
func (tx *TextView) EditDone() {
	if tx.Buf != nil {
		tx.Buf.EditDone()
	}
	tx.ClearSelected()
}

// Revert aborts editing and reverts to last saved text
func (tx *TextView) Revert() {
	updt := tx.UpdateStart()
	defer tx.UpdateEnd(updt)
	// tx.Edited = false
	tx.StartPos = 0
	tx.EndPos = tx.CharWidth
	tx.LayoutLines()
	// todo: signal buffer?
	tx.SelectReset()
}

//////////////////////////////////////////////////////////////////////////////////////////
//  Buffer communication

// SetBuf sets the TextBuf that this is a view of, and interconnects their signals
func (tx *TextView) SetBuf(buf *TextBuf) {
	tx.Buf = buf
	buf.AddView(tx)
	tx.Revert()
}

// TextViewBufSigRecv receives a signal from the buffer and updates view accordingly
func TextViewBufSigRecv(rvwki, sbufki ki.Ki, sig int64, data interface{}) {
	tx := rvwki.Embed(KiT_TextView).(*TextView)
	switch TextBufSignals(sig) {
	case TextBufDone:
	case TextBufNew:
		tx.LayoutLines()
	case TextBufInsert:
	case TextBufDelete:
	}
}

//////////////////////////////////////////////////////////////////////////////////////////
//  Text formatting and rendering

// HasHi returns true if there are highighting parameters set
func (tx *TextView) HasHi() bool {
	if tx.HiLang == "" || tx.HiStyle == "" {
		return false
	}
	return true
}

// HiInit initializes the syntax highlighting for current Hi params
func (tx *TextView) HiInit() {
	if !tx.HasHi() {
		return
	}
	if tx.HiLang == tx.lastHiLang && tx.HiStyle == tx.lastHiStyle {
		return
	}
	tx.lexer = chroma.Coalesce(lexers.Get(tx.HiLang))
	tx.formatter = html.New(html.WithClasses(), html.TabWidth(tx.TabWidth))
	tx.style = styles.Get(tx.HiStyle)
	if tx.style == nil {
		tx.style = styles.Fallback
	}
	var cssBuf bytes.Buffer
	err := tx.formatter.WriteCSS(&cssBuf, tx.style)
	if err != nil {
		log.Println(err)
		return
	}
	csstr := cssBuf.String()
	csstr = strings.Replace(csstr, " .chroma .", " .", -1)
	// lnidx := strings.Index(csstr, "\n")
	// csstr = csstr[lnidx+1:]
	tx.HiCSS.ParseString(csstr)
	tx.CSS = tx.HiCSS.CSSProps()

	if chp, ok := ki.SubProps(tx.CSS, ".chroma"); ok {
		for ky, vl := range chp { // apply to top level
			tx.SetProp(ky, vl)
		}
	}

	tx.lastHiLang = tx.HiLang
	tx.lastHiStyle = tx.HiStyle
}

// RenderSize is the size we should pass to text rendering, based on alloc
func (tx *TextView) RenderSize() gi.Vec2D {
	st := &tx.Sty
	st.Font.LoadFont(&st.UnContext)
	tx.FontHeight = st.Font.Height
	spc := tx.Sty.BoxSpace()
	sz := tx.LayData.AllocSize
	if sz.IsZero() {
		sz = tx.LayData.SizePrefOrMax()
	}
	if !sz.IsZero() {
		sz.SetSubVal(2 * spc)
	}
	tx.RenderSz = sz
	return sz
}

// LayoutLines generates TextRenders of lines from our TextBuf, using any
// highlighter that might be present
func (tx *TextView) LayoutLines() {
	if tx.Buf == nil || tx.Buf.NLines == 0 {
		tx.NLines = 0
		tx.LinesSize = gi.Vec2DZero
		return
	}

	tx.HiInit()

	tx.NLines = tx.Buf.NLines
	tx.Markup = make([][]byte, tx.NLines)
	tx.Render = make([]gi.TextRender, tx.NLines)
	tx.Offs = make([]float32, tx.NLines)

	if tx.HasHi() {
		var htmlBuf bytes.Buffer
		iterator, err := tx.lexer.Tokenise(nil, string(tx.Buf.Txt)) // todo: unfortunate conversion here..
		err = tx.formatter.Format(&htmlBuf, tx.style, iterator)
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
			tx.Markup[ln] = mt
		}
	} else {
		for ln := 0; ln < tx.NLines; ln++ {
			tx.Markup[ln] = []byte(string(tx.Buf.Lines[ln]))
		}
	}

	sz := tx.RenderSize()
	st := &tx.Sty
	off := float32(0)
	mxwd := float32(0)
	for ln := 0; ln < tx.NLines; ln++ {
		tx.Render[ln].SetHTMLPre(tx.Markup[ln], &st.Font, &st.UnContext, tx.CSS)
		tx.Render[ln].LayoutStdLR(&st.Text, &st.Font, &st.UnContext, sz)
		tx.Offs[ln] = off
		lsz := tx.Render[ln].Size.Y
		if lsz < tx.FontHeight {
			lsz = tx.FontHeight
		}
		off += lsz
		mxwd = gi.Max32(mxwd, tx.Render[ln].Size.X)
	}
	tx.LinesSize.Set(mxwd, off)
}

// LayoutLine generates render of given line (including highlighting)
func (tx *TextView) LayoutLine(ln int) {
	if tx.HasHi() {
		var htmlBuf bytes.Buffer
		iterator, err := tx.lexer.Tokenise(nil, string(tx.Buf.Lines[ln]))
		err = tx.formatter.Format(&htmlBuf, tx.style, iterator)
		if err != nil {
			log.Println(err)
			return
		}
		tx.Markup[ln] = htmlBuf.Bytes()
	} else {
		tx.Markup[ln] = []byte(string(tx.Buf.Lines[ln]))
	}

	st := &tx.Sty
	tx.Render[ln].SetHTMLPre(tx.Markup[ln], &st.Font, &st.UnContext, tx.CSS)
	tx.Render[ln].LayoutStdLR(&st.Text, &st.Font, &st.UnContext, tx.RenderSz)
}

//////////////////////////////////////////////////////////////////////////////////////////
//  Cursor Navigation

// CursorForward moves the cursor forward
func (tx *TextView) CursorForward(steps int) {
	updt := tx.UpdateStart()
	defer tx.UpdateEnd(updt)
	tx.CursorPos += steps
	if tx.CursorPos > len(tx.EditTxt) {
		tx.CursorPos = len(tx.EditTxt)
	}
	if tx.CursorPos > tx.EndPos {
		inc := tx.CursorPos - tx.EndPos
		tx.EndPos += inc
	}
	if tx.SelectMode {
		if tx.CursorPos-steps < tx.SelectStart {
			tx.SelectStart = tx.CursorPos
		} else if tx.CursorPos > tx.SelectStart {
			tx.SelectEnd = tx.CursorPos
		} else {
			tx.SelectStart = tx.CursorPos
		}
		tx.SelectUpdate()
	}
}

// CursorForward moves the cursor backward
func (tx *TextView) CursorBackward(steps int) {
	updt := tx.UpdateStart()
	defer tx.UpdateEnd(updt)
	tx.CursorPos -= steps
	if tx.CursorPos < 0 {
		tx.CursorPos = 0
	}
	if tx.CursorPos <= tx.StartPos {
		dec := kit.MinInt(tx.StartPos, 8)
		tx.StartPos -= dec
	}
	if tx.SelectMode {
		if tx.CursorPos+steps < tx.SelectStart {
			tx.SelectStart = tx.CursorPos
		} else if tx.CursorPos > tx.SelectStart {
			tx.SelectEnd = tx.CursorPos
		} else {
			tx.SelectStart = tx.CursorPos
		}
		tx.SelectUpdate()
	}
}

// CursorStart moves the cursor to the start of the text, updating selection
// if select mode is active
func (tx *TextView) CursorStart() {
	updt := tx.UpdateStart()
	defer tx.UpdateEnd(updt)
	tx.CursorPos = 0
	tx.StartPos = 0
	tx.EndPos = kit.MinInt(len(tx.EditTxt), tx.StartPos+tx.CharWidth)
	if tx.SelectMode {
		tx.SelectStart = 0
		tx.SelectUpdate()
	}
}

// CursorEnd moves the cursor to the end of the text
func (tx *TextView) CursorEnd() {
	updt := tx.UpdateStart()
	defer tx.UpdateEnd(updt)
	ed := len(tx.EditTxt)
	tx.CursorPos = ed
	tx.EndPos = len(tx.EditTxt) // try -- display will adjust
	tx.StartPos = kit.MaxInt(0, tx.EndPos-tx.CharWidth)
	if tx.SelectMode {
		tx.SelectEnd = ed
		tx.SelectUpdate()
	}
}

// todo: ctrl+backspace = delete word
// shift+arrow = select
// uparrow = start / down = end

// CursorBackspace deletes character(s) immediately before cursor
func (tx *TextView) CursorBackspace(steps int) {
	if tx.HasSelection() {
		tx.DeleteSelection()
		return
	}
	if tx.CursorPos < steps {
		steps = tx.CursorPos
	}
	if steps <= 0 {
		return
	}
	updt := tx.UpdateStart()
	defer tx.UpdateEnd(updt)
	tx.Edited = true
	tx.EditTxt = append(tx.EditTxt[:tx.CursorPos-steps], tx.EditTxt[tx.CursorPos:]...)
	tx.CursorBackward(steps)
	if tx.CursorPos > tx.SelectStart && tx.CursorPos <= tx.SelectEnd {
		tx.SelectEnd -= steps
	} else if tx.CursorPos < tx.SelectStart {
		tx.SelectStart -= steps
		tx.SelectEnd -= steps
	}
	tx.SelectUpdate()
}

// CursorDelete deletes character(s) immediately after the cursor
func (tx *TextView) CursorDelete(steps int) {
	if tx.HasSelection() {
		tx.DeleteSelection()
	}
	if tx.CursorPos+steps > len(tx.EditTxt) {
		steps = len(tx.EditTxt) - tx.CursorPos
	}
	if steps <= 0 {
		return
	}
	updt := tx.UpdateStart()
	defer tx.UpdateEnd(updt)
	tx.Edited = true
	tx.EditTxt = append(tx.EditTxt[:tx.CursorPos], tx.EditTxt[tx.CursorPos+steps:]...)
	if tx.CursorPos > tx.SelectStart && tx.CursorPos <= tx.SelectEnd {
		tx.SelectEnd -= steps
	} else if tx.CursorPos < tx.SelectStart {
		tx.SelectStart -= steps
		tx.SelectEnd -= steps
	}
	tx.SelectUpdate()
}

// CursorKill deletes text from cursor to end of text
func (tx *TextView) CursorKill() {
	steps := len(tx.EditTxt) - tx.CursorPos
	tx.CursorDelete(steps)
}

// ClearSelected resets both the global selected flag and any current selection
func (tx *TextView) ClearSelected() {
	tx.WidgetBase.ClearSelected()
	tx.SelectReset()
}

// HasSelection returns whether there is a selected region of text
func (tx *TextView) HasSelection() bool {
	tx.SelectUpdate()
	if tx.SelectStart < tx.SelectEnd {
		return true
	}
	return false
}

// Selection returns the currently selected text
func (tx *TextView) Selection() string {
	if tx.HasSelection() {
		// return string(tx.EditTxt[tx.SelectStart:tx.SelectEnd])
	}
	return ""
}

// SelectModeToggle toggles the SelectMode, updating selection with cursor movement
func (tx *TextView) SelectModeToggle() {
	if tx.SelectMode {
		tx.SelectMode = false
	} else {
		tx.SelectMode = true
		tx.SelectStart = tx.CursorPos
		tx.SelectEnd = tx.SelectStart
	}
}

// SelectAll selects all the text
func (tx *TextView) SelectAll() {
	updt := tx.UpdateStart()
	tx.SelectStart = 0
	tx.SelectEnd = len(tx.EditTxt)
	tx.UpdateEnd(updt)
}

// IsWordBreak defines what counts as a word break for the purposes of selecting words
func (tx *TextView) IsWordBreak(r rune) bool {
	if unicode.IsSpace(r) || unicode.IsSymbol(r) || unicode.IsPunct(r) {
		return true
	}
	return false
}

// SelectWord selects the word (whitespace delimited) that the cursor is on
func (tx *TextView) SelectWord() {
	updt := tx.UpdateStart()
	defer tx.UpdateEnd(updt)
	sz := len(tx.EditTxt)
	if sz <= 3 {
		tx.SelectAll()
		return
	}
	tx.SelectStart = tx.CursorPos
	if tx.SelectStart >= sz {
		tx.SelectStart = sz - 2
	}
	if !tx.IsWordBreak(tx.EditTxt[tx.SelectStart]) {
		for tx.SelectStart > 0 {
			if tx.IsWordBreak(tx.EditTxt[tx.SelectStart-1]) {
				break
			}
			tx.SelectStart--
		}
		tx.SelectEnd = tx.CursorPos + 1
		for tx.SelectEnd < sz {
			if tx.IsWordBreak(tx.EditTxt[tx.SelectEnd]) {
				break
			}
			tx.SelectEnd++
		}
	} else { // keep the space start -- go to next space..
		tx.SelectEnd = tx.CursorPos + 1
		for tx.SelectEnd < sz {
			if !tx.IsWordBreak(tx.EditTxt[tx.SelectEnd]) {
				break
			}
			tx.SelectEnd++
		}
		for tx.SelectEnd < sz {
			if tx.IsWordBreak(tx.EditTxt[tx.SelectEnd]) {
				break
			}
			tx.SelectEnd++
		}
	}
}

// SelectReset resets the selection
func (tx *TextView) SelectReset() {
	tx.SelectMode = false
	if tx.SelectStart == 0 && tx.SelectEnd == 0 {
		return
	}
	updt := tx.UpdateStart()
	tx.SelectStart = 0
	tx.SelectEnd = 0
	tx.UpdateEnd(updt)
}

// SelectUpdate updates the select region after any change to the text, to keep it in range
func (tx *TextView) SelectUpdate() {
	if tx.SelectStart < tx.SelectEnd {
		ed := len(tx.EditTxt)
		if tx.SelectStart < 0 {
			tx.SelectStart = 0
		}
		if tx.SelectEnd > ed {
			tx.SelectEnd = ed
		}
	} else {
		tx.SelectReset()
	}
}

// Cut cuts any selected text and adds it to the clipboard, also returns cut text
func (tx *TextView) Cut() string {
	cut := tx.DeleteSelection()
	if cut != "" {
		oswin.TheApp.ClipBoard().Write(mimedata.NewText(cut))
	}
	return cut
}

// DeleteSelection deletes any selected text, without adding to clipboard --
// returns text deleted
func (tx *TextView) DeleteSelection() string {
	tx.SelectUpdate()
	if !tx.HasSelection() {
		return ""
	}
	updt := tx.UpdateStart()
	defer tx.UpdateEnd(updt)
	cut := tx.Selection()
	tx.Edited = true
	tx.EditTxt = append(tx.EditTxt[:tx.SelectStart], tx.EditTxt[tx.SelectEnd:]...)
	if tx.CursorPos > tx.SelectStart {
		if tx.CursorPos < tx.SelectEnd {
			tx.CursorPos = tx.SelectStart
		} else {
			tx.CursorPos -= tx.SelectEnd - tx.SelectStart
		}
	}
	tx.SelectReset()
	return cut
}

// Copy copies any selected text to the clipboard, and returns that text,
// optionaly resetting the current selection
func (tx *TextView) Copy(reset bool) string {
	tx.SelectUpdate()
	if !tx.HasSelection() {
		return ""
	}
	cpy := tx.Selection()
	oswin.TheApp.ClipBoard().Write(mimedata.NewText(cpy))
	if reset {
		tx.SelectReset()
	}
	return cpy
}

// Paste inserts text from the clipboard at current cursor position -- if
// cursor is within a current selection, that selection is
func (tx *TextView) Paste() {
	data := oswin.TheApp.ClipBoard().Read([]string{mimedata.TextPlain})
	if data != nil {
		if tx.CursorPos >= tx.SelectStart && tx.CursorPos < tx.SelectEnd {
			tx.DeleteSelection()
		}
		tx.InsertAtCursor(data.Text(mimedata.TextPlain))
	}
}

// InsertAtCursor inserts given text at current cursor position
func (tx *TextView) InsertAtCursor(str string) {
	updt := tx.UpdateStart()
	defer tx.UpdateEnd(updt)
	if tx.HasSelection() {
		tx.Cut()
	}
	tx.Edited = true
	rs := []rune(str)
	rsl := len(rs)
	nt := make([]rune, 0, cap(tx.EditTxt)+cap(rs))
	nt = append(nt, tx.EditTxt[:tx.CursorPos]...)
	nt = append(nt, rs...)
	nt = append(nt, tx.EditTxt[tx.CursorPos:]...)
	tx.EditTxt = nt
	tx.EndPos += rsl
	tx.CursorForward(rsl)
}

// cpos := tx.CharStartPos(tx.CursorPos).ToPoint()

func (tx *TextView) MakeContextMenu(m *gi.Menu) {
	cpsc := gi.ActiveKeyMap.ChordForFun(gi.KeyFunCopy)
	ac := m.AddAction(gi.ActOpts{Label: "Copy", Shortcut: cpsc},
		tx.This, func(recv, send ki.Ki, sig int64, data interface{}) {
			txf := recv.Embed(KiT_TextView).(*TextView)
			txf.Copy(true)
		})
	ac.SetActiveState(tx.HasSelection())
	if !tx.IsInactive() {
		ctsc := gi.ActiveKeyMap.ChordForFun(gi.KeyFunCut)
		ptsc := gi.ActiveKeyMap.ChordForFun(gi.KeyFunPaste)
		ac = m.AddAction(gi.ActOpts{Label: "Cut", Shortcut: ctsc},
			tx.This, func(recv, send ki.Ki, sig int64, data interface{}) {
				txf := recv.Embed(KiT_TextView).(*TextView)
				txf.Cut()
			})
		ac.SetActiveState(tx.HasSelection())
		ac = m.AddAction(gi.ActOpts{Label: "Paste", Shortcut: ptsc},
			tx.This, func(recv, send ki.Ki, sig int64, data interface{}) {
				txf := recv.Embed(KiT_TextView).(*TextView)
				txf.Paste()
			})
		ac.SetInactiveState(oswin.TheApp.ClipBoard().IsEmpty())
	}
}

// OfferCompletions pops up a menu of possible completions
func (tx *TextView) OfferCompletions() {
	win := tx.ParentWindow()
	if gi.PopupIsCompleter(win.Popup) {
		win.ClosePopup(win.Popup)
	}
	if tx.Completion.MatchFunc == nil {
		return
	}

	tx.Completion.Completions, tx.Completion.Seed = tx.Completion.MatchFunc(string(tx.EditTxt[0:tx.CursorPos]))
	count := len(tx.Completion.Completions)
	if count > 0 {
		if count == 1 && tx.Completion.Completions[0] == tx.Completion.Seed {
			return
		}
		var m gi.Menu
		for i := 0; i < count; i++ {
			s := tx.Completion.Completions[i]
			m.AddAction(gi.ActOpts{Label: s},
				tx.This, func(recv, send ki.Ki, sig int64, data interface{}) {
					txf := recv.Embed(KiT_TextView).(*TextView)
					txf.Complete(s)
				})
		}
		cpos := tx.CharStartPos(tx.CursorPos).ToPoint()
		// todo: figure popup placement using font and line height
		vp := gi.PopupMenu(m, cpos.X+15, cpos.Y+50, tx.Viewport, "tx-completion-menu")
		bitflag.Set(&vp.Flag, int(gi.VpFlagCompleter))
		vp.KnownChild(0).SetProp("no-focus-name", true) // disable name focusing -- grabs key events in popup instead of in textxield!
	}
}

// Complete edits the text field using the string chosen from the completion menu
func (tx *TextView) Complete(str string) {
	txt := string(tx.EditTxt) // John: do NOT call tx.Text() in an active editing context!!!
	s, delta := tx.Completion.EditFunc(txt, tx.CursorPos, str, tx.Completion.Seed)
	tx.EditTxt = []rune(s)
	tx.CursorForward(delta)
}

// PixelToCursor finds the cursor position that corresponds to the given pixel location
func (tx *TextView) PixelToCursor(pixOff float32) int {
	st := &tx.Sty

	spc := st.BoxSpace()
	px := pixOff - spc

	if px <= 0 {
		return tx.StartPos
	}

	// for selection to work correctly, we need this to be deterministic

	sz := len(tx.EditTxt)
	c := tx.StartPos + int(float64(px/st.UnContext.ToDotsFactor(units.Ch)))
	c = kit.MinInt(c, sz)

	w := tx.TextWidth(tx.StartPos, c)
	if w > px {
		for w > px {
			c--
			if c <= tx.StartPos {
				c = tx.StartPos
				break
			}
			w = tx.TextWidth(tx.StartPos, c)
		}
	} else if w < px {
		for c < tx.EndPos {
			wn := tx.TextWidth(tx.StartPos, c+1)
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

func (tx *TextView) SetCursorFromPixel(pixOff float32, selMode mouse.SelectModes) {
	updt := tx.UpdateStart()
	defer tx.UpdateEnd(updt)
	oldPos := tx.CursorPos
	tx.CursorPos = tx.PixelToCursor(pixOff)
	if tx.SelectMode || selMode != mouse.NoSelectMode {
		if !tx.SelectMode && selMode != mouse.NoSelectMode {
			tx.SelectStart = oldPos
			tx.SelectMode = true
		}
		if !tx.IsDragging() && tx.CursorPos >= tx.SelectStart && tx.CursorPos < tx.SelectEnd {
			tx.SelectReset()
		} else if tx.CursorPos > tx.SelectStart {
			tx.SelectEnd = tx.CursorPos
		} else {
			tx.SelectStart = tx.CursorPos
		}
		tx.SelectUpdate()
	} else if tx.HasSelection() {
		tx.SelectReset()
	}
}

// KeyInput handles keyboard input into the text field and from the completion menu
func (tx *TextView) KeyInput(kt *key.ChordEvent) {
	kf := gi.KeyFun(kt.ChordString())
	win := tx.ParentWindow()

	if gi.PopupIsCompleter(win.Popup) {
		switch kf {
		case gi.KeyFunFocusNext: // tab will complete if single item or try to extend if multiple items
			count := len(tx.Completion.Completions)
			if count > 0 {
				if count == 1 { // just complete
					tx.Complete(tx.Completion.Completions[0])
					win.ClosePopup(win.Popup)
				} else { // try to extend the seed
					s := complete.ExtendSeed(tx.Completion.Completions, tx.Completion.Seed)
					if s != "" {
						win.ClosePopup(win.Popup)
						tx.InsertAtCursor(s)
						tx.OfferCompletions()
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
		tx.CursorForward(1)
		tx.OfferCompletions()
	case gi.KeyFunMoveLeft:
		kt.SetProcessed()
		tx.CursorBackward(1)
		tx.OfferCompletions()
	case gi.KeyFunHome:
		kt.SetProcessed()
		tx.CursorStart()
	case gi.KeyFunEnd:
		kt.SetProcessed()
		tx.CursorEnd()
	case gi.KeyFunSelectMode:
		kt.SetProcessed()
		tx.SelectModeToggle()
	case gi.KeyFunCancelSelect:
		kt.SetProcessed()
		tx.SelectReset()
	case gi.KeyFunSelectAll:
		kt.SetProcessed()
		tx.SelectAll()
	case gi.KeyFunCopy:
		kt.SetProcessed()
		tx.Copy(true) // reset
	}
	if tx.IsInactive() || kt.IsProcessed() {
		return
	}
	switch kf {
	case gi.KeyFunSelectItem: // enter
		fallthrough
	case gi.KeyFunAccept: // ctrl+enter
		tx.EditDone()
		kt.SetProcessed()
		tx.FocusNext()
	case gi.KeyFunAbort: // esc
		tx.Revert()
		kt.SetProcessed()
		tx.FocusNext()
	case gi.KeyFunBackspace:
		kt.SetProcessed()
		tx.CursorBackspace(1)
		tx.OfferCompletions()
	case gi.KeyFunKill:
		kt.SetProcessed()
		tx.CursorKill()
	case gi.KeyFunDelete:
		kt.SetProcessed()
		tx.CursorDelete(1)
	case gi.KeyFunCut:
		kt.SetProcessed()
		tx.Cut()
	case gi.KeyFunPaste:
		kt.SetProcessed()
		tx.Paste()
	case gi.KeyFunComplete:
		kt.SetProcessed()
		tx.OfferCompletions()
	case gi.KeyFunNil:
		if unicode.IsPrint(kt.Rune) {
			if !kt.HasAnyModifier(key.Control, key.Meta) {
				kt.SetProcessed()
				tx.InsertAtCursor(string(kt.Rune))
				tx.OfferCompletions()
			}
		}
	}
}

// MouseEvent handles the mouse.Event
func (tx *TextView) MouseEvent(me *mouse.Event) {
	if !tx.IsInactive() && !tx.HasFocus() {
		tx.GrabFocus()
	}
	me.SetProcessed()
	switch me.Button {
	case mouse.Left:
		if me.Action == mouse.Press {
			if tx.IsInactive() {
				tx.SetSelectedState(!tx.IsSelected())
				tx.EmitSelectedSignal()
				tx.UpdateSig()
			} else {
				pt := tx.PointToRelPos(me.Pos())
				tx.SetCursorFromPixel(float32(pt.X), me.SelectMode())
			}
		} else if me.Action == mouse.DoubleClick {
			me.SetProcessed()
			if tx.HasSelection() {
				if tx.SelectStart == 0 && tx.SelectEnd == len(tx.EditTxt) {
					tx.SelectReset()
				} else {
					tx.SelectAll()
				}
			} else {
				tx.SelectWord()
			}
		}
	case mouse.Middle:
		if !tx.IsInactive() && me.Action == mouse.Press {
			me.SetProcessed()
			pt := tx.PointToRelPos(me.Pos())
			tx.SetCursorFromPixel(float32(pt.X), me.SelectMode())
			tx.Paste()
		}
	case mouse.Right:
		if me.Action == mouse.Press {
			me.SetProcessed()
			tx.EmitContextMenuSignal()
			tx.This.(gi.Node2D).ContextMenu()
		}
	}
}

func (tx *TextView) TextViewEvents() {
	tx.HoverTooltipEvent()
	tx.ConnectEvent(oswin.MouseDragEvent, gi.RegPri, func(recv, send ki.Ki, sig int64, d interface{}) {
		me := d.(*mouse.DragEvent)
		me.SetProcessed()
		tx := recv.Embed(KiT_TextView).(*TextView)
		if !tx.SelectMode {
			tx.SelectModeToggle()
		}
		pt := tx.PointToRelPos(me.Pos())
		tx.SetCursorFromPixel(float32(pt.X), mouse.NoSelectMode)
	})
	tx.ConnectEvent(oswin.MouseEvent, gi.RegPri, func(recv, send ki.Ki, sig int64, d interface{}) {
		txf := recv.Embed(KiT_TextView).(*TextView)
		me := d.(*mouse.Event)
		txf.MouseEvent(me)
	})
	tx.ConnectEvent(oswin.MouseFocusEvent, gi.RegPri, func(recv, send ki.Ki, sig int64, d interface{}) {
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
	tx.ConnectEvent(oswin.KeyChordEvent, gi.RegPri, func(recv, send ki.Ki, sig int64, d interface{}) {
		txf := recv.Embed(KiT_TextView).(*TextView)
		kt := d.(*key.ChordEvent)
		txf.KeyInput(kt)
	})
	if dlg, ok := tx.Viewport.This.(*gi.Dialog); ok {
		dlg.DialogSig.Connect(tx.This, func(recv, send ki.Ki, sig int64, data interface{}) {
			txf, _ := recv.Embed(KiT_TextView).(*TextView)
			if sig == int64(gi.DialogAccepted) {
				txf.EditDone()
			}
		})
	}
}

////////////////////////////////////////////////////
//  Node2D Interface

func (tx *TextView) Init2D() {
	tx.Init2DWidget()
	// tx.EditTxt = []rune(tx.Txt)
	// tx.Edited = false
}

func (tx *TextView) Style2D() {
	tx.HiInit()
	tx.SetCanFocusIfActive()
	tx.Style2DWidget()
	pst := &(tx.Par.(gi.Node2D).AsWidget().Sty)
	for i := 0; i < int(TextViewStatesN); i++ {
		tx.StateStyles[i].CopyFrom(&tx.Sty)
		tx.StateStyles[i].SetStyleProps(pst, tx.StyleProps(TextViewSelectors[i]))
		tx.StateStyles[i].StyleCSS(tx.This.(gi.Node2D), tx.CSSAgg, TextViewSelectors[i])
		tx.StateStyles[i].CopyUnitContext(&tx.Sty.UnContext)
	}
}

func (tx *TextView) UpdateRenderAll() bool {
	st := &tx.Sty
	st.Font.LoadFont(&st.UnContext)
	tx.RenderAll.SetRunes(tx.EditTxt, &st.Font, &st.UnContext, &st.Text, true, 0, 0)
	return true
}

func (tx *TextView) Size2D(iter int) {
	// if len(tx.Txt) == 0 && len(tx.Placeholder) > 0 {
	// 	text = tx.Placeholder
	// } else {
	// 	text = tx.Txt
	// }
	tx.LayoutLines()
	// tx.Edited = false
	// maxlen := tx.MaxWidthReq
	// if maxlen <= 0 {
	// 	maxlen = 50
	// }
	tx.Size2DFromWH(tx.LinesSize.X, tx.LinesSize.Y)
}

func (tx *TextView) Layout2D(parBBox image.Rectangle, iter int) bool {
	tx.Layout2DBase(parBBox, true, iter) // init style
	for i := 0; i < int(TextViewStatesN); i++ {
		tx.StateStyles[i].CopyUnitContext(&tx.Sty.UnContext)
	}
	return tx.Layout2DChildren(iter)
}

// StartCharPos returns the starting position of the given rune
func (tx *TextView) StartCharPos(idx int) float32 {
	if idx <= 0 || len(tx.RenderAll.Spans) != 1 {
		return 0.0
	}
	sr := &(tx.RenderAll.Spans[0])
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
func (tx *TextView) TextWidth(st, ed int) float32 {
	return tx.StartCharPos(ed) - tx.StartCharPos(st)
}

// CharStartPos returns the starting render coords for the given character
// position in string -- makes no attempt to rationalize that pos (i.e., if
// not in visible range, position will be out of range too)
func (tx *TextView) CharStartPos(charidx int) gi.Vec2D {
	st := &tx.Sty
	spc := st.BoxSpace()
	pos := tx.LayData.AllocPos.AddVal(spc)
	cpos := tx.TextWidth(tx.StartPos, charidx)
	return gi.Vec2D{pos.X + cpos, pos.Y}
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
		tx := BlinkingTextView
		if tx.Viewport == nil || !tx.HasFocus() || !tx.FocusActive || tx.VpBBox == image.ZR {
			BlinkingTextView = nil
			continue
		}
		win := tx.ParentWindow()
		if win == nil || win.IsResizing() {
			continue
		}
		tx.BlinkOn = !tx.BlinkOn
		tx.RenderCursor(tx.BlinkOn)
	}
}

func (tx *TextView) StartCursor() {
	tx.BlinkOn = true
	if gi.CursorBlinkMSec == 0 {
		tx.RenderCursor(true)
		return
	}
	if TextViewBlinker == nil {
		TextViewBlinker = time.NewTicker(time.Duration(gi.CursorBlinkMSec) * time.Millisecond)
		go TextViewBlink()
	}
	tx.BlinkOn = true
	win := tx.ParentWindow()
	if win != nil && !win.IsResizing() {
		tx.RenderCursor(true)
	}
	BlinkingTextView = tx
}

func (tx *TextView) StopCursor() {
	if BlinkingTextView == tx {
		BlinkingTextView = nil
	}
}

func (tx *TextView) RenderCursor(on bool) {
	if tx.PushBounds() {
		st := &tx.Sty
		cpos := tx.CharStartPos(tx.CursorPos)
		rs := &tx.Viewport.Render
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
		pc.DrawLine(rs, cpos.X, cpos.Y, cpos.X, cpos.Y+tx.FontHeight)
		pc.Stroke(rs)
		tx.PopBounds()

		// compute bbox just for the cursor
		cbmin := cpos.SubVal(st.Border.Width.Dots)
		cbmax := cpos.AddVal(st.Border.Width.Dots)
		cbmax.Y += tx.FontHeight
		curBBox := image.Rectangle{cbmin.ToPointFloor(), cbmax.ToPointCeil()}
		vprel := curBBox.Min.Sub(tx.VpBBox.Min)
		curWinBBox := tx.WinBBox.Add(vprel)

		vp := tx.Viewport
		updt := vp.Win.UpdateStart()
		vp.Win.UploadVpRegion(vp, curBBox, curWinBBox) // bigger than necc.
		vp.Win.UpdateEnd(updt)
	}
}

func (tx *TextView) RenderSelect() {
	if tx.SelectEnd <= tx.SelectStart {
		return
	}
	effst := kit.MaxInt(tx.StartPos, tx.SelectStart)
	if effst >= tx.EndPos {
		return
	}
	effed := kit.MinInt(tx.EndPos, tx.SelectEnd)
	if effed < tx.StartPos {
		return
	}
	if effed <= effst {
		return
	}

	spos := tx.CharStartPos(effst)

	rs := &tx.Viewport.Render
	pc := &rs.Paint
	st := &tx.StateStyles[TextViewSel]
	tsz := tx.TextWidth(effst, effed)
	pc.FillBox(rs, spos, gi.Vec2D{tsz, tx.FontHeight}, &st.Font.BgColor)
}

// AutoScroll scrolls the starting position to keep the cursor visible
func (tx *TextView) AutoScroll() {
	st := &tx.Sty

	tx.UpdateRenderAll()

	sz := len(tx.EditTxt)

	if sz == 0 || tx.LayData.AllocSize.X <= 0 {
		tx.CursorPos = 0
		tx.EndPos = 0
		tx.StartPos = 0
		return
	}
	spc := st.BoxSpace()
	maxw := tx.LayData.AllocSize.X - 2.0*spc
	tx.CharWidth = int(maxw / st.UnContext.ToDotsFactor(units.Ch)) // rough guess in chars

	// first rationalize all the values
	if tx.EndPos == 0 || tx.EndPos > sz { // not init
		tx.EndPos = sz
	}
	if tx.StartPos >= tx.EndPos {
		tx.StartPos = kit.MaxInt(0, tx.EndPos-tx.CharWidth)
	}
	tx.CursorPos = gi.InRangeInt(tx.CursorPos, 0, sz)

	inc := int(math32.Ceil(.1 * float32(tx.CharWidth)))
	inc = kit.MaxInt(4, inc)

	// keep cursor in view with buffer
	startIsAnchor := true
	if tx.CursorPos < (tx.StartPos + inc) {
		tx.StartPos -= inc
		tx.StartPos = kit.MaxInt(tx.StartPos, 0)
		tx.EndPos = tx.StartPos + tx.CharWidth
		tx.EndPos = kit.MinInt(sz, tx.EndPos)
	} else if tx.CursorPos > (tx.EndPos - inc) {
		tx.EndPos += inc
		tx.EndPos = kit.MinInt(tx.EndPos, sz)
		tx.StartPos = tx.EndPos - tx.CharWidth
		tx.StartPos = kit.MaxInt(0, tx.StartPos)
		startIsAnchor = false
	}

	if startIsAnchor {
		gotWidth := false
		spos := tx.StartCharPos(tx.StartPos)
		for {
			w := tx.StartCharPos(tx.EndPos) - spos
			if w < maxw {
				if tx.EndPos == sz {
					break
				}
				nw := tx.StartCharPos(tx.EndPos+1) - spos
				if nw >= maxw {
					gotWidth = true
					break
				}
				tx.EndPos++
			} else {
				tx.EndPos--
			}
		}
		if gotWidth || tx.StartPos == 0 {
			return
		}
		// otherwise, try getting some more chars by moving up start..
	}

	// end is now anchor
	epos := tx.StartCharPos(tx.EndPos)
	for {
		w := epos - tx.StartCharPos(tx.StartPos)
		if w < maxw {
			if tx.StartPos == 0 {
				break
			}
			nw := epos - tx.StartCharPos(tx.StartPos-1)
			if nw >= maxw {
				break
			}
			tx.StartPos--
		} else {
			tx.StartPos++
		}
	}
}

// RenderText displays the lines on the screen
func (tx *TextView) RenderText() {
	rs := &tx.Viewport.Render
	st := &tx.Sty
	pos := tx.LayData.AllocPos.AddVal(st.Layout.Margin.Dots + st.Layout.Padding.Dots)
	pos.Y -= 0.5 * tx.FontHeight
	for ln := 0; ln < tx.NLines; ln++ {
		lst := pos.Y + tx.Offs[ln]
		led := lst + tx.Render[ln].Size.Y
		if int(math32.Ceil(led)) < tx.VpBBox.Min.Y {
			continue
		}
		if int(math32.Floor(lst)) > tx.VpBBox.Max.Y {
			continue
		}
		lp := pos
		lp.Y = lst
		tx.Render[ln].RenderTopPos(rs, lp)
	}
}

func (tx *TextView) Render2D() {
	if tx.FullReRenderIfNeeded() {
		return
	}
	if tx.PushBounds() {
		// tx.TextViewEvents()
		// tx.AutoScroll() // inits paint with our style

		if tx.IsInactive() {
			if tx.IsSelected() {
				tx.Sty = tx.StateStyles[TextViewSel]
			} else {
				tx.Sty = tx.StateStyles[TextViewInactive]
			}
		} else if tx.HasFocus() {
			if tx.FocusActive {
				tx.Sty = tx.StateStyles[TextViewFocus]
			} else {
				tx.Sty = tx.StateStyles[TextViewActive]
			}
		} else if tx.IsSelected() {
			tx.Sty = tx.StateStyles[TextViewSel]
		} else {
			tx.Sty = tx.StateStyles[TextViewActive]
		}
		// rs := &tx.Viewport.Render
		st := &tx.Sty
		st.Font.LoadFont(&st.UnContext)
		tx.RenderStdBox(st)
		// cur := tx.EditTxt[tx.StartPos:tx.EndPos]
		// tx.RenderSelect()
		// pos := tx.LayData.AllocPos.AddVal(st.BoxSpace())
		tx.RenderText()
		// if len(tx.EditTxt) == 0 && len(tx.Placeholder) > 0 {
		// 	st.Font.Color = st.Font.Color.Highlight(50)
		// 	tx.RenderVis.SetString(tx.Placeholder, &st.Font, &st.UnContext, &st.Text, true, 0, 0)
		// 	tx.RenderVis.RenderTopPos(rs, pos)

		// } else {
		// 	tx.RenderVis.SetRunes(cur, &st.Font, &st.UnContext, &st.Text, true, 0, 0)
		// 	tx.RenderVis.RenderTopPos(rs, pos)
		// }
		// if tx.HasFocus() && tx.FocusActive {
		// 	tx.StartCursor()
		// } else {
		// 	tx.StopCursor()
		// }
		tx.Render2DChildren()
		tx.PopBounds()
	} else {
		tx.DisconnectAllEvents(gi.RegPri)
	}
}

func (tx *TextView) FocusChanged2D(change gi.FocusChanges) {
	switch change {
	case gi.FocusLost:
		tx.FocusActive = false
		tx.EditDone()
		tx.UpdateSig()
	case gi.FocusGot:
		tx.FocusActive = true
		tx.ScrollToMe()
		//tx.CursorEnd()
		tx.EmitFocusedSignal()
		tx.UpdateSig()
	case gi.FocusInactive:
		tx.FocusActive = false
		tx.EditDone()
		tx.UpdateSig()
	case gi.FocusActive:
		tx.FocusActive = true
		tx.ScrollToMe()
		// tx.UpdateSig()
		// todo: see about cursor
	}
}

func (tx *TextView) SetCompleter(data interface{}, matchFun complete.MatchFunc, editFun complete.EditFunc) {
	if matchFun == nil || editFun == nil {
		return
	}
	tx.Completion.Context = data
	tx.Completion.MatchFunc = matchFun
	tx.Completion.EditFunc = editFun
}
