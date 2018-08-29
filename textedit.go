// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"bytes"
	"fmt"
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

// TextPos represents line, character positions within the TextEdit
type TextPos struct {
	Ln, Ch int
}

// TextEdit is a widget for editing multiple lines of text (as compared to
// TextField for a single line).  The underlying data model is just plain
// simple lines (ended by \n) with any number of characters per line.  These
// lines are displayed using wrap-around text into the editor.  Currently only
// works on in-memory strings.  Set the min
type TextEdit struct {
	WidgetBase
	Txt         string `json:"-" xml:"text" desc:"the last saved value of the text string being edited"`
	Placeholder string `json:"-" xml:"placeholder" desc:"text that is displayed when the field is empty, in a lower-contrast manner"`
	Edited      bool   `json:"-" xml:"-" desc:"true if the text has been edited relative to the original"`
	TabWidth    int    `desc:"how many spaces is a tab"`
	HiLang      string `desc:"language for syntax highlighting the code"`
	HiStyle     string `desc:"syntax highlighting style"`
	HiCSS       StyleSheet
	FocusActive bool `json:"-" xml:"-" desc:"true if the keyboard focus is active or not -- when we lose active focus we apply changes"`
	NLines      int
	EditLines   [][]rune     `json:"-" xml:"-" desc:"the live text being edited, with latest modifications -- encoded as runes per line"`
	EditTxt     []rune       `json:"-" xml:"-" desc:"the live text being edited, with latest modifications -- encoded as runes per line"`
	MarkupTxt   [][]byte     `json:"-" xml:"-" desc:"marked-up version of the edit text, after being run through the syntax highlighting process -- this is what is actually rendered"`
	Render      []TextRender `json:"-" xml:"-" desc:"render of the text -- what is actually visible -- per line"`
	MaxWidthReq int          `desc:"maximum width that field will request, in characters, during Size2D process -- if 0 then is 50 -- ensures that large strings don't request super large values -- standard max-width can override"`
	CursorPos   int          `xml:"-" desc:"current cursor position"`
	StartPos    int
	EndPos      int
	RenderAll   TextRender
	CharWidth   int                    `xml:"-" desc:"approximate number of chars that can be displayed at any time -- computed from font size etc"`
	SelectStart int                    `xml:"-" desc:"starting position of selection in the string"`
	SelectEnd   int                    `xml:"-" desc:"ending position of selection in the string"`
	SelectMode  bool                   `xml:"-" desc:"if true, select text as cursor moves"`
	TextEditSig ki.Signal              `json:"-" xml:"-" view:"-" desc:"signal for line edit -- see TextEditSignals for the types"`
	StateStyles [TextEditStatesN]Style `json:"-" xml:"-" desc:"normal style and focus style"`
	FontHeight  float32                `json:"-" xml:"-" desc:"font height, cached during styling"`
	BlinkOn     bool                   `json:"-" xml:"-" oscillates between on and off for blinking"`
	Completion  Complete               `json:"-" xml:"-" desc:"functions and data for textfield completion"`
}

var KiT_TextEdit = kit.Types.AddType(&TextEdit{}, TextEditProps)

var TextEditProps = ki.Props{
	"font-family":      "Go Mono",
	"border-width":     units.NewValue(1, units.Px), // this also determines the cursor
	"border-color":     &Prefs.Colors.Border,
	"border-style":     BorderSolid,
	"padding":          units.NewValue(4, units.Px),
	"margin":           units.NewValue(1, units.Px),
	"text-align":       AlignLeft,
	"color":            &Prefs.Colors.Font,
	"background-color": &Prefs.Colors.Control,
	TextEditSelectors[TextEditActive]: ki.Props{
		"background-color": "lighter-0",
	},
	TextEditSelectors[TextEditFocus]: ki.Props{
		"border-width":     units.NewValue(2, units.Px),
		"background-color": "samelight-80",
	},
	TextEditSelectors[TextEditInactive]: ki.Props{
		"background-color": "highlight-10",
	},
	TextEditSelectors[TextEditSel]: ki.Props{
		"background-color": &Prefs.Colors.Select,
	},
}

// signals that buttons can send
type TextEditSignals int64

const (
	// main signal -- return was pressed and an edit was completed -- data is the text
	TextEditDone TextEditSignals = iota

	// some text was selected (for Inactive state, selection is via WidgetSig)
	TextEditSelected

	TextEditSignalsN
)

//go:generate stringer -type=TextEditSignals

// TextEditStates are mutually-exclusive textfield states -- determines appearance
type TextEditStates int32

const (
	// normal state -- there but not being interacted with
	TextEditActive TextEditStates = iota

	// textfield is the focus -- will respond to keyboard input
	TextEditFocus

	// inactive -- not editable
	TextEditInactive

	// selected -- for inactive state, can select entire element
	TextEditSel

	TextEditStatesN
)

//go:generate stringer -type=TextEditStates

// Style selector names for the different states
var TextEditSelectors = []string{":active", ":focus", ":inactive", ":selected"}

// Text returns the current text -- applies any unapplied changes first
func (tx *TextEdit) Text() string {
	tx.EditDone()
	return tx.Txt
}

// SetText sets the text to be edited and reverts any current edit to reflect this new text
func (tx *TextEdit) SetText(txt string) {
	tx.Txt = txt
	tx.RevertEdit()
}

// Label returns the display label for this node, satisfying the Labeler interface
func (tx *TextEdit) Label() string {
	return tx.Nm
}

// EditDone completes editing and copies the active edited text to the text --
// called when the return key is pressed or goes out of focus
func (tx *TextEdit) EditDone() {
	if tx.Edited {
		tx.Edited = false
		// tx.Txt = string(tx.EditTxt)
		tx.TextEditSig.Emit(tx.This, int64(TextEditDone), tx.Txt)
	}
	tx.ClearSelected()
}

// RevertEdit aborts editing and reverts to last saved text
func (tx *TextEdit) RevertEdit() {
	updt := tx.UpdateStart()
	defer tx.UpdateEnd(updt)
	// tx.EditTxt = []rune(tx.Txt)
	tx.Edited = false
	tx.StartPos = 0
	tx.EndPos = tx.CharWidth
	tx.SelectReset()
}

//////////////////////////////////////////////////////////////////////////////////////////
//  Text formatting and rendering

func (tx *TextEdit) RenderFullText() {
	lns := strings.Split(tx.Txt, "\n")
	tx.NLines = len(lns)
	tx.EditLines = make([][]rune, tx.NLines)
	for ln, txt := range lns {
		tx.EditLines[ln] = []rune(txt)
	}

	// syntax highlighting:
	lexer := chroma.Coalesce(lexers.Get(tx.HiLang))
	formatter := html.New(html.WithClasses(), html.WithLineNumbers(), html.TabWidth(tx.TabWidth))
	style := styles.Get(tx.HiStyle)
	if style == nil {
		style = styles.Fallback
	}
	var cssBuf bytes.Buffer
	err := formatter.WriteCSS(&cssBuf, style)
	if err != nil {
		log.Println(err)
		return
	}
	csstr := cssBuf.String()
	csstr = strings.Replace(csstr, " .chroma", "", -1)
	lnidx := strings.Index(csstr, "\n")
	csstr = csstr[lnidx+1:]
	// fmt.Printf("=================\nCSS:\n%v\n\n", csstr)
	tx.HiCSS.ParseString(csstr)
	tx.CSS = tx.HiCSS.CSSProps()

	var htmlBuf bytes.Buffer
	iterator, err := lexer.Tokenise(nil, tx.Txt)
	err = formatter.Format(&htmlBuf, style, iterator)
	if err != nil {
		log.Println(err)
		return
	}
	// htmlstr := htmlBuf.String()
	// fmt.Printf("=================\nHTML:\n%v\n\n", htmlstr)
	mtlns := bytes.Split(htmlBuf.Bytes(), []byte("\n"))

	tx.MarkupTxt = make([][]byte, tx.NLines)
	tx.Render = make([]TextRender, tx.NLines)

	st := &tx.Sty
	st.Font.LoadFont(&st.UnContext)

	spc := tx.Sty.BoxSpace()
	sz := tx.LayData.AllocSize
	if sz.IsZero() {
		sz = tx.LayData.SizePrefOrMax()
	}
	if !sz.IsZero() {
		sz.SetSubVal(2 * spc)
	}

	exln := 4
	maxln := len(mtlns) - exln
	fmt.Printf("Nlines: %v  mkup lns: %v\n", tx.NLines, maxln)
	for ln := 1; ln <= maxln; ln++ {
		mt := mtlns[ln]
		if ln == 1 {
			mt = bytes.TrimPrefix(mt, []byte(`<pre class="chroma">`))
		}
		tx.MarkupTxt[ln-1] = mt

		// todo: going to require a custom <span> parser just for this b/c
		// the go xml parser does not preserve whitespace
		tx.Render[ln-1].SetHTML(string(mt), &st.Font, &st.UnContext, tx.CSS)
		tx.Render[ln-1].LayoutStdLR(&st.Text, &st.Font, &st.UnContext, sz)
	}
}

//////////////////////////////////////////////////////////////////////////////////////////
//  Cursor Navigation

// CursorForward moves the cursor forward
func (tx *TextEdit) CursorForward(steps int) {
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
func (tx *TextEdit) CursorBackward(steps int) {
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
func (tx *TextEdit) CursorStart() {
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
func (tx *TextEdit) CursorEnd() {
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
func (tx *TextEdit) CursorBackspace(steps int) {
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
func (tx *TextEdit) CursorDelete(steps int) {
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
func (tx *TextEdit) CursorKill() {
	steps := len(tx.EditTxt) - tx.CursorPos
	tx.CursorDelete(steps)
}

// ClearSelected resets both the global selected flag and any current selection
func (tx *TextEdit) ClearSelected() {
	tx.WidgetBase.ClearSelected()
	tx.SelectReset()
}

// HasSelection returns whether there is a selected region of text
func (tx *TextEdit) HasSelection() bool {
	tx.SelectUpdate()
	if tx.SelectStart < tx.SelectEnd {
		return true
	}
	return false
}

// Selection returns the currently selected text
func (tx *TextEdit) Selection() string {
	if tx.HasSelection() {
		// return string(tx.EditTxt[tx.SelectStart:tx.SelectEnd])
	}
	return ""
}

// SelectModeToggle toggles the SelectMode, updating selection with cursor movement
func (tx *TextEdit) SelectModeToggle() {
	if tx.SelectMode {
		tx.SelectMode = false
	} else {
		tx.SelectMode = true
		tx.SelectStart = tx.CursorPos
		tx.SelectEnd = tx.SelectStart
	}
}

// SelectAll selects all the text
func (tx *TextEdit) SelectAll() {
	updt := tx.UpdateStart()
	tx.SelectStart = 0
	tx.SelectEnd = len(tx.EditTxt)
	tx.UpdateEnd(updt)
}

// IsWordBreak defines what counts as a word break for the purposes of selecting words
func (tx *TextEdit) IsWordBreak(r rune) bool {
	if unicode.IsSpace(r) || unicode.IsSymbol(r) || unicode.IsPunct(r) {
		return true
	}
	return false
}

// SelectWord selects the word (whitespace delimited) that the cursor is on
func (tx *TextEdit) SelectWord() {
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
func (tx *TextEdit) SelectReset() {
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
func (tx *TextEdit) SelectUpdate() {
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
func (tx *TextEdit) Cut() string {
	cut := tx.DeleteSelection()
	if cut != "" {
		oswin.TheApp.ClipBoard().Write(mimedata.NewText(cut))
	}
	return cut
}

// DeleteSelection deletes any selected text, without adding to clipboard --
// returns text deleted
func (tx *TextEdit) DeleteSelection() string {
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
func (tx *TextEdit) Copy(reset bool) string {
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
func (tx *TextEdit) Paste() {
	data := oswin.TheApp.ClipBoard().Read([]string{mimedata.TextPlain})
	if data != nil {
		if tx.CursorPos >= tx.SelectStart && tx.CursorPos < tx.SelectEnd {
			tx.DeleteSelection()
		}
		tx.InsertAtCursor(data.Text(mimedata.TextPlain))
	}
}

// InsertAtCursor inserts given text at current cursor position
func (tx *TextEdit) InsertAtCursor(str string) {
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

func (tx *TextEdit) MakeContextMenu(m *Menu) {
	cpsc := ActiveKeyMap.ChordForFun(KeyFunCopy)
	ac := m.AddAction(ActOpts{Label: "Copy", Shortcut: cpsc},
		tx.This, func(recv, send ki.Ki, sig int64, data interface{}) {
			txf := recv.Embed(KiT_TextEdit).(*TextEdit)
			txf.Copy(true)
		})
	ac.SetActiveState(tx.HasSelection())
	if !tx.IsInactive() {
		ctsc := ActiveKeyMap.ChordForFun(KeyFunCut)
		ptsc := ActiveKeyMap.ChordForFun(KeyFunPaste)
		ac = m.AddAction(ActOpts{Label: "Cut", Shortcut: ctsc},
			tx.This, func(recv, send ki.Ki, sig int64, data interface{}) {
				txf := recv.Embed(KiT_TextEdit).(*TextEdit)
				txf.Cut()
			})
		ac.SetActiveState(tx.HasSelection())
		ac = m.AddAction(ActOpts{Label: "Paste", Shortcut: ptsc},
			tx.This, func(recv, send ki.Ki, sig int64, data interface{}) {
				txf := recv.Embed(KiT_TextEdit).(*TextEdit)
				txf.Paste()
			})
		ac.SetInactiveState(oswin.TheApp.ClipBoard().IsEmpty())
	}
}

// OfferCompletions pops up a menu of possible completions
func (tx *TextEdit) OfferCompletions() {
	win := tx.ParentWindow()
	if PopupIsCompleter(win.Popup) {
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
		var m Menu
		for i := 0; i < count; i++ {
			s := tx.Completion.Completions[i]
			m.AddAction(ActOpts{Label: s},
				tx.This, func(recv, send ki.Ki, sig int64, data interface{}) {
					txf := recv.Embed(KiT_TextEdit).(*TextEdit)
					txf.Complete(s)
				})
		}
		cpos := tx.CharStartPos(tx.CursorPos).ToPoint()
		// todo: figure popup placement using font and line height
		vp := PopupMenu(m, cpos.X+15, cpos.Y+50, tx.Viewport, "tx-completion-menu")
		bitflag.Set(&vp.Flag, int(VpFlagCompleter))
		vp.KnownChild(0).SetProp("no-focus-name", true) // disable name focusing -- grabs key events in popup instead of in textxield!
	}
}

// Complete edits the text field using the string chosen from the completion menu
func (tx *TextEdit) Complete(str string) {
	txt := string(tx.EditTxt) // John: do NOT call tx.Text() in an active editing context!!!
	s, delta := tx.Completion.EditFunc(txt, tx.CursorPos, str, tx.Completion.Seed)
	tx.EditTxt = []rune(s)
	tx.CursorForward(delta)
}

// PixelToCursor finds the cursor position that corresponds to the given pixel location
func (tx *TextEdit) PixelToCursor(pixOff float32) int {
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

func (tx *TextEdit) SetCursorFromPixel(pixOff float32, selMode mouse.SelectModes) {
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
func (tx *TextEdit) KeyInput(kt *key.ChordEvent) {
	kf := KeyFun(kt.ChordString())
	win := tx.ParentWindow()

	if PopupIsCompleter(win.Popup) {
		switch kf {
		case KeyFunFocusNext: // tab will complete if single item or try to extend if multiple items
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
	case KeyFunMoveRight:
		kt.SetProcessed()
		tx.CursorForward(1)
		tx.OfferCompletions()
	case KeyFunMoveLeft:
		kt.SetProcessed()
		tx.CursorBackward(1)
		tx.OfferCompletions()
	case KeyFunHome:
		kt.SetProcessed()
		tx.CursorStart()
	case KeyFunEnd:
		kt.SetProcessed()
		tx.CursorEnd()
	case KeyFunSelectMode:
		kt.SetProcessed()
		tx.SelectModeToggle()
	case KeyFunCancelSelect:
		kt.SetProcessed()
		tx.SelectReset()
	case KeyFunSelectAll:
		kt.SetProcessed()
		tx.SelectAll()
	case KeyFunCopy:
		kt.SetProcessed()
		tx.Copy(true) // reset
	}
	if tx.IsInactive() || kt.IsProcessed() {
		return
	}
	switch kf {
	case KeyFunSelectItem: // enter
		fallthrough
	case KeyFunAccept: // ctrl+enter
		tx.EditDone()
		kt.SetProcessed()
		tx.FocusNext()
	case KeyFunAbort: // esc
		tx.RevertEdit()
		kt.SetProcessed()
		tx.FocusNext()
	case KeyFunBackspace:
		kt.SetProcessed()
		tx.CursorBackspace(1)
		tx.OfferCompletions()
	case KeyFunKill:
		kt.SetProcessed()
		tx.CursorKill()
	case KeyFunDelete:
		kt.SetProcessed()
		tx.CursorDelete(1)
	case KeyFunCut:
		kt.SetProcessed()
		tx.Cut()
	case KeyFunPaste:
		kt.SetProcessed()
		tx.Paste()
	case KeyFunComplete:
		kt.SetProcessed()
		tx.OfferCompletions()
	case KeyFunNil:
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
func (tx *TextEdit) MouseEvent(me *mouse.Event) {
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
			tx.This.(Node2D).ContextMenu()
		}
	}
}

func (tx *TextEdit) TextEditEvents() {
	tx.HoverTooltipEvent()
	tx.ConnectEvent(oswin.MouseDragEvent, RegPri, func(recv, send ki.Ki, sig int64, d interface{}) {
		me := d.(*mouse.DragEvent)
		me.SetProcessed()
		tx := recv.Embed(KiT_TextEdit).(*TextEdit)
		if !tx.SelectMode {
			tx.SelectModeToggle()
		}
		pt := tx.PointToRelPos(me.Pos())
		tx.SetCursorFromPixel(float32(pt.X), mouse.NoSelectMode)
	})
	tx.ConnectEvent(oswin.MouseEvent, RegPri, func(recv, send ki.Ki, sig int64, d interface{}) {
		txf := recv.Embed(KiT_TextEdit).(*TextEdit)
		me := d.(*mouse.Event)
		txf.MouseEvent(me)
	})
	tx.ConnectEvent(oswin.MouseFocusEvent, RegPri, func(recv, send ki.Ki, sig int64, d interface{}) {
		txf := recv.Embed(KiT_TextEdit).(*TextEdit)
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
	tx.ConnectEvent(oswin.KeyChordEvent, RegPri, func(recv, send ki.Ki, sig int64, d interface{}) {
		txf := recv.Embed(KiT_TextEdit).(*TextEdit)
		kt := d.(*key.ChordEvent)
		txf.KeyInput(kt)
	})
	if dlg, ok := tx.Viewport.This.(*Dialog); ok {
		dlg.DialogSig.Connect(tx.This, func(recv, send ki.Ki, sig int64, data interface{}) {
			txf, _ := recv.Embed(KiT_TextEdit).(*TextEdit)
			if sig == int64(DialogAccepted) {
				txf.EditDone()
			}
		})
	}
}

////////////////////////////////////////////////////
//  Node2D Interface

func (tx *TextEdit) Init2D() {
	tx.Init2DWidget()
	tx.EditTxt = []rune(tx.Txt)
	tx.Edited = false
}

func (tx *TextEdit) Style2D() {
	tx.SetCanFocusIfActive()
	tx.Style2DWidget()
	pst := &(tx.Par.(Node2D).AsWidget().Sty)
	for i := 0; i < int(TextEditStatesN); i++ {
		tx.StateStyles[i].CopyFrom(&tx.Sty)
		tx.StateStyles[i].SetStyleProps(pst, tx.StyleProps(TextEditSelectors[i]))
		tx.StateStyles[i].StyleCSS(tx.This.(Node2D), tx.CSSAgg, TextEditSelectors[i])
		tx.StateStyles[i].CopyUnitContext(&tx.Sty.UnContext)
	}
	tx.RenderFullText()
}

func (tx *TextEdit) UpdateRenderAll() bool {
	st := &tx.Sty
	st.Font.LoadFont(&st.UnContext)
	tx.RenderAll.SetRunes(tx.EditTxt, &st.Font, &st.UnContext, &st.Text, true, 0, 0)
	return true
}

func (tx *TextEdit) Size2D(iter int) {
	tmptxt := tx.EditTxt
	if len(tx.Txt) == 0 && len(tx.Placeholder) > 0 {
		tx.EditTxt = []rune(tx.Placeholder)
	} else {
		tx.EditTxt = []rune(tx.Txt)
	}
	tx.Edited = false
	tx.StartPos = 0
	maxlen := tx.MaxWidthReq
	if maxlen <= 0 {
		maxlen = 50
	}
	tx.EndPos = kit.MinInt(len(tx.EditTxt), maxlen)
	tx.UpdateRenderAll()
	tx.FontHeight = tx.RenderAll.Size.Y
	w := tx.TextWidth(tx.StartPos, tx.EndPos)
	w += 2.0 // give some extra buffer
	// fmt.Printx("fontheight: %v width: %v\n", tx.FontHeight, w)
	tx.Size2DFromWH(w, tx.FontHeight)
	tx.EditTxt = tmptxt
}

func (tx *TextEdit) Layout2D(parBBox image.Rectangle, iter int) bool {
	tx.Layout2DBase(parBBox, true, iter) // init style
	for i := 0; i < int(TextEditStatesN); i++ {
		tx.StateStyles[i].CopyUnitContext(&tx.Sty.UnContext)
	}
	return tx.Layout2DChildren(iter)
}

// StartCharPos returns the starting position of the given rune
func (tx *TextEdit) StartCharPos(idx int) float32 {
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
func (tx *TextEdit) TextWidth(st, ed int) float32 {
	return tx.StartCharPos(ed) - tx.StartCharPos(st)
}

// CharStartPos returns the starting render coords for the given character
// position in string -- makes no attempt to rationalize that pos (i.e., if
// not in visible range, position will be out of range too)
func (tx *TextEdit) CharStartPos(charidx int) Vec2D {
	st := &tx.Sty
	spc := st.BoxSpace()
	pos := tx.LayData.AllocPos.AddVal(spc)
	cpos := tx.TextWidth(tx.StartPos, charidx)
	return Vec2D{pos.X + cpos, pos.Y}
}

// TextEditBlinker is the time.Ticker for blinking cursors for text fields,
// only one of which can be active at at a time
var TextEditBlinker *time.Ticker

// BlinkingTextEdit is the text field that is blinking
var BlinkingTextEdit *TextEdit

// TextEditBlink is function that blinks text field cursor
func TextEditBlink() {
	for {
		if TextEditBlinker == nil {
			return // shutdown..
		}
		<-TextEditBlinker.C
		if BlinkingTextEdit == nil {
			continue
		}
		if BlinkingTextEdit.IsDestroyed() || BlinkingTextEdit.IsDeleted() {
			BlinkingTextEdit = nil
			continue
		}
		tx := BlinkingTextEdit
		if tx.Viewport == nil || !tx.HasFocus() || !tx.FocusActive || tx.VpBBox == image.ZR {
			BlinkingTextEdit = nil
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

func (tx *TextEdit) StartCursor() {
	tx.BlinkOn = true
	if CursorBlinkMSec == 0 {
		tx.RenderCursor(true)
		return
	}
	if TextEditBlinker == nil {
		TextEditBlinker = time.NewTicker(time.Duration(CursorBlinkMSec) * time.Millisecond)
		go TextEditBlink()
	}
	tx.BlinkOn = true
	win := tx.ParentWindow()
	if win != nil && !win.IsResizing() {
		tx.RenderCursor(true)
	}
	BlinkingTextEdit = tx
}

func (tx *TextEdit) StopCursor() {
	if BlinkingTextEdit == tx {
		BlinkingTextEdit = nil
	}
}

func (tx *TextEdit) RenderCursor(on bool) {
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

func (tx *TextEdit) RenderSelect() {
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
	st := &tx.StateStyles[TextEditSel]
	tsz := tx.TextWidth(effst, effed)
	pc.FillBox(rs, spos, Vec2D{tsz, tx.FontHeight}, &st.Font.BgColor)
}

// AutoScroll scrolls the starting position to keep the cursor visible
func (tx *TextEdit) AutoScroll() {
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
	tx.CursorPos = InRangeInt(tx.CursorPos, 0, sz)

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

func (tx *TextEdit) Render2D() {
	if tx.FullReRenderIfNeeded() {
		return
	}
	if tx.PushBounds() {
		// tx.TextEditEvents()
		// tx.AutoScroll() // inits paint with our style

		if tx.IsInactive() {
			if tx.IsSelected() {
				tx.Sty = tx.StateStyles[TextEditSel]
			} else {
				tx.Sty = tx.StateStyles[TextEditInactive]
			}
		} else if tx.HasFocus() {
			if tx.FocusActive {
				tx.Sty = tx.StateStyles[TextEditFocus]
			} else {
				tx.Sty = tx.StateStyles[TextEditActive]
			}
		} else if tx.IsSelected() {
			tx.Sty = tx.StateStyles[TextEditSel]
		} else {
			tx.Sty = tx.StateStyles[TextEditActive]
		}
		rs := &tx.Viewport.Render
		st := &tx.Sty
		st.Font.LoadFont(&st.UnContext)
		tx.RenderStdBox(st)
		// cur := tx.EditTxt[tx.StartPos:tx.EndPos]
		// tx.RenderSelect()
		pos := tx.LayData.AllocPos.AddVal(st.BoxSpace())
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

		for ln := 0; ln < tx.NLines; ln++ {
			tx.Render[ln].RenderTopPos(rs, pos)
			pos.Y += tx.Render[ln].Size.Y
			if pos.Y > float32(tx.VpBBox.Max.Y) {
				break
			}
		}

		tx.Render2DChildren()
		tx.PopBounds()
	} else {
		tx.DisconnectAllEvents(RegPri)
	}
}

func (tx *TextEdit) FocusChanged2D(change FocusChanges) {
	switch change {
	case FocusLost:
		tx.FocusActive = false
		tx.EditDone()
		tx.UpdateSig()
	case FocusGot:
		tx.FocusActive = true
		tx.ScrollToMe()
		//tx.CursorEnd()
		tx.EmitFocusedSignal()
		tx.UpdateSig()
	case FocusInactive:
		tx.FocusActive = false
		tx.EditDone()
		tx.UpdateSig()
	case FocusActive:
		tx.FocusActive = true
		tx.ScrollToMe()
		// tx.UpdateSig()
		// todo: see about cursor
	}
}

func (tx *TextEdit) SetCompleter(data interface{}, matchFun complete.MatchFunc, editFun complete.EditFunc) {
	if matchFun == nil || editFun == nil {
		return
	}
	tx.Completion.Context = data
	tx.Completion.MatchFunc = matchFun
	tx.Completion.EditFunc = editFun
}
