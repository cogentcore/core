// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package giv

import (
	"bytes"
	"fmt"
	"go/token"
	"image"
	"image/draw"
	"log"
	"strings"
	"sync"
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

const force = true
const dontforce = false

// TextViewOpts contains options for TextView editing
type TextViewOpts struct {
	SpaceIndent bool `desc:"use spaces, not tabs, for indentation -- tab-size property in TextStyle has the tab size, used for either tabs or spaces"`
	AutoIndent  bool `desc:"auto-indent on newline (enter) or tab"`
	LineNos     bool `desc:"show line numbers at left end of editor"`
	Completion  bool `desc:"use the completion system to suggest options while typing"`
}

// TextView is a widget for editing multiple lines of text (as compared to
// TextField for a single line).  The underlying data model is just plain
// simple lines (ended by \n) with any number of characters per line.  These
// lines are displayed using wrap-around text into the editor.  Currently only
// works on in-memory strings.
type TextView struct {
	gi.WidgetBase
	Buf               *TextBuf                  `json:"-" xml:"-" desc:"the text buffer that we're editing"`
	Placeholder       string                    `json:"-" xml:"placeholder" desc:"text that is displayed when the field is empty, in a lower-contrast manner"`
	Opts              TextViewOpts              `desc:"options for how text editing / viewing works"`
	CursorWidth       units.Value               `xml:"cursor-width" desc:"width of cursor -- set from cursor-width property (inherited)"`
	HiStyle           HiStyleName               `desc:"syntax highlighting style"`
	HiCSS             gi.StyleSheet             `json:"-" xml:"-" desc:"CSS StyleSheet for given highlighting style"`
	LineIcons         map[int]gi.IconName       `desc:"icons for each line -- use SetLineIcon and DeleteLineIcon"`
	FocusActive       bool                      `json:"-" xml:"-" desc:"true if the keyboard focus is active or not -- when we lose active focus we apply changes"`
	NLines            int                       `json:"-" xml:"-" desc:"number of lines in the view -- sync'd with the Buf after edits, but always reflects storage size of Renders etc"`
	Markup            [][]byte                  `json:"-" xml:"-" desc:"marked-up version of the edit text lines, after being run through the syntax highlighting process -- this is what is actually rendered"`
	HasMarkup         []bool                    `json:"-" xml:"-" desc:"is markup version available?  for each line of text"`
	MarkupMu          sync.Mutex                `json:"-" xml:"-" desc:"mutex for accessing HasMarkup -- markup routine and main routine use this to coordinate"`
	Renders           []gi.TextRender           `json:"-" xml:"-" desc:"renders of the text lines, with one render per line (each line could visibly wrap-around, so these are logical lines, not display lines)"`
	Offs              []float32                 `json:"-" xml:"-" desc:"starting offsets for top of each line"`
	LineNoDigs        int                       `json:"-" xml:"-" number of line number digits needed"`
	LineNoOff         float32                   `json:"-" xml:"-" desc:"horizontal offset for start of text after line numbers"`
	LineNoRender      gi.TextRender             `json:"-" xml:"-" desc:"render for line numbers"`
	LinesSize         image.Point               `json:"-" xml:"-" desc:"total size of all lines as rendered"`
	RenderSz          gi.Vec2D                  `json:"-" xml:"-" desc:"size params to use in render call"`
	CursorPos         TextPos                   `json:"-" xml:"-" desc:"current cursor position"`
	CursorCol         int                       `json:"-" xml:"-" desc:"desired cursor column -- where the cursor was last when moved using left / right arrows -- used when doing up / down to not always go to short line columns"`
	SelectReg         TextRegion                `json:"-" xml:"-" desc:"current selection region"`
	PrevSelectReg     TextRegion                `json:"-" xml:"-" desc:"previous selection region, that was actually rendered -- needed to update render"`
	Highlights        []TextRegion              `json:"-" xml:"-" desc:"highlighed regions, e.g., for search results"`
	SelectMode        bool                      `json:"-" xml:"-" desc:"if true, select text as cursor moves"`
	ISearchMode       bool                      `json:"-" xml:"-" desc:"if true, in interactive search mode"`
	ISearchString     string                    `json:"-" xml:"-" desc:"current interactive search string"`
	ISearchCase       bool                      `json:"-" xml:"-" desc:"pay attention to case in isearch -- triggered by typing an upper-case letter"`
	SearchMatches     []TextPos                 `json:"-" xml:"-" desc:"current search matches"`
	SearchPos         int                       `json:"-" xml:"-" desc:"position within isearch matches"`
	PrevISearchString string                    `json:"-" xml:"-" desc:"previous interactive search string"`
	PrevISearchCase   bool                      `json:"-" xml:"-" desc:"prev: pay attention to case in isearch -- triggered by typing an upper-case letter"`
	ISearchStartPos   TextPos                   `json:"-" xml:"-" desc:"starting position for search -- returns there after on cancel"`
	TextViewSig       ki.Signal                 `json:"-" xml:"-" view:"-" desc:"signal for text viewt -- see TextViewSignals for the types"`
	LinkSig           ki.Signal                 `json:"-" xml:"-" view:"-" desc:"signal for clicking on a link -- data is a string of the URL -- if nobody receiving this signal, calls TextLinkHandler then URLHandler"`
	StateStyles       [TextViewStatesN]gi.Style `json:"-" xml:"-" desc:"normal style and focus style"`
	FontHeight        float32                   `json:"-" xml:"-" desc:"font height, cached during styling"`
	LineHeight        float32                   `json:"-" xml:"-" desc:"line height, cached during styling"`
	VisSize           image.Point               `json:"-" xml:"-" desc:"height in lines and width in chars of the visible area"`
	BlinkOn           bool                      `json:"-" xml:"-" oscillates between on and off for blinking"`
	Complete          *gi.Complete              `json:"-" xml:"-" desc:"functions and data for textfield completion"`
	// chroma highlighting
	lastHiLang   string
	lastHiStyle  HiStyleName
	lexer        chroma.Lexer
	formatter    *html.Formatter
	style        *chroma.Style
	reLayout     bool
	lastRecenter int
}

var KiT_TextView = kit.Types.AddType(&TextView{}, TextViewProps)

var TextViewProps = ki.Props{
	"white-space":      gi.WhiteSpacePreWrap,
	"font-family":      "Go Mono",
	"border-width":     0, // don't render our own border
	"cursor-width":     units.NewValue(3, units.Px),
	"border-color":     &gi.Prefs.Colors.Border,
	"border-style":     gi.BorderSolid,
	"padding":          units.NewValue(2, units.Px),
	"margin":           units.NewValue(2, units.Px),
	"vertical-align":   gi.AlignTop,
	"text-align":       gi.AlignLeft,
	"tab-size":         4,
	"color":            &gi.Prefs.Colors.Font,
	"background-color": &gi.Prefs.Colors.Background,
	TextViewSelectors[TextViewActive]: ki.Props{
		"background-color": "lighter-0",
	},
	TextViewSelectors[TextViewFocus]: ki.Props{
		"background-color": "samelight-80",
	},
	TextViewSelectors[TextViewInactive]: ki.Props{
		"background-color": "highlight-10",
	},
	TextViewSelectors[TextViewSel]: ki.Props{
		"background-color": &gi.Prefs.Colors.Select,
	},
	TextViewSelectors[TextViewHighlight]: ki.Props{
		"background-color": &gi.Prefs.Colors.Highlight,
	},
}

// TextViewSignals are signals that text view can send
type TextViewSignals int64

const (
	// return was pressed and an edit was completed -- data is the text
	TextViewDone TextViewSignals = iota

	// some text was selected (for Inactive state, selection is via WidgetSig)
	TextViewSelected

	// cursor moved emitted for every cursor movement -- e.g., for displaying cursor pos
	TextViewCursorMoved

	// ISearch emitted for every update of interactive search process -- see
	// ISearch* members for current state
	TextViewISearch

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

	// selected
	TextViewSel

	// highlighted
	TextViewHighlight

	TextViewStatesN
)

//go:generate stringer -type=TextViewStates

// Style selector names for the different states
var TextViewSelectors = []string{":active", ":focus", ":inactive", ":selected", ":highlight"}

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

// Refresh re-displays everything anew from the buffer
func (tv *TextView) Refresh() {
	tv.LayoutAllLines(false)
	tv.RenderAllLines()
}

func (tv *TextView) IsChanged() bool {
	if tv.Buf != nil && tv.Buf.Changed {
		return true
	}
	return false
}

///////////////////////////////////////////////////////////////////////////////
//  Buffer communication

// ResetState resets all the random state variables, when opening a new buffer etc
func (tv *TextView) ResetState() {
	tv.SelectReset()
	tv.Highlights = nil
	tv.ISearchMode = false
	if tv.Buf == nil || tv.NLines != tv.Buf.NLines { // don't reset if reopening..
		tv.CursorPos = TextPos{}
	}
}

// SetBuf sets the TextBuf that this is a view of, and interconnects their signals
func (tv *TextView) SetBuf(buf *TextBuf) {
	if buf != nil && tv.Buf == buf {
		return
	}
	if tv.Buf != nil {
		tv.Buf.DeleteView(tv)
	}
	tv.Buf = buf
	tv.ResetState()
	if buf != nil {
		buf.AddView(tv)
	}
	tv.LayoutAllLines(false)
	tv.SetFullReRender()
	tv.UpdateSig()
}

// InsertLines inserts new lines of text and reformats them
func (tv *TextView) InsertLines(tbe *TextBufEdit) {
	tv.MarkupMu.Lock()

	stln := tbe.Reg.Start.Ln + 1
	nsz := (tbe.Reg.End.Ln - tbe.Reg.Start.Ln)

	// Markup
	tmpmu := make([][]byte, nsz)
	nmu := append(tv.Markup, tmpmu...) // first append to end to extend capacity
	copy(nmu[stln+nsz:], nmu[stln:])   // move stuff to end
	copy(nmu[stln:], tmpmu)            // copy into position
	tv.Markup = nmu

	// HasMarkup
	tmphm := make([]bool, nsz)
	nhm := append(tv.HasMarkup, tmphm...)
	copy(nhm[stln+nsz:], nhm[stln:])
	copy(nhm[stln:], tmphm)
	tv.HasMarkup = nhm

	// Renders
	tmprn := make([]gi.TextRender, nsz)
	nrn := append(tv.Renders, tmprn...)
	copy(nrn[stln+nsz:], nrn[stln:])
	copy(nrn[stln:], tmprn)
	tv.Renders = nrn

	// Offs
	tmpof := make([]float32, nsz)
	nof := append(tv.Offs, tmpof...)
	copy(nof[stln+nsz:], nof[stln:])
	copy(nof[stln:], tmpof)
	tv.Offs = nof

	tv.NLines += nsz
	tv.MarkupMu.Unlock()

	tv.LayoutLines(tbe.Reg.Start.Ln, tbe.Reg.End.Ln, false)
	tv.RenderAllLines()
}

// DeleteLines deletes lines of text and reformats remaining one
func (tv *TextView) DeleteLines(tbe *TextBufEdit) {
	tv.MarkupMu.Lock()

	stln := tbe.Reg.Start.Ln
	edln := tbe.Reg.End.Ln
	dsz := edln - stln

	tv.Markup = append(tv.Markup[:stln], tv.Markup[edln:]...)
	tv.HasMarkup = append(tv.HasMarkup[:stln], tv.HasMarkup[edln:]...)
	tv.Renders = append(tv.Renders[:stln], tv.Renders[edln:]...)
	tv.Offs = append(tv.Offs[:stln], tv.Offs[edln:]...)

	tv.NLines -= dsz
	tv.MarkupMu.Unlock()

	tv.LayoutLines(tbe.Reg.Start.Ln, tbe.Reg.Start.Ln, true)
	tv.RenderAllLines()
}

// TextViewBufSigRecv receives a signal from the buffer and updates view accordingly
func TextViewBufSigRecv(rvwki, sbufki ki.Ki, sig int64, data interface{}) {
	tv := rvwki.Embed(KiT_TextView).(*TextView)
	switch TextBufSignals(sig) {
	case TextBufDone:
	case TextBufNew:
		tv.ResetState()
		tv.LayoutAllLines(false)
		tv.SetFullReRender()
		tv.UpdateSig()
	case TextBufInsert:
		if tv.Renders == nil { // not init yet
			return
		}
		tbe := data.(*TextBufEdit)
		// fmt.Printf("tv %v got %v\n", tv.Nm, tbe.Reg.Start)
		if tbe.Reg.Start.Ln != tbe.Reg.End.Ln {
			tv.InsertLines(tbe)
		} else {
			rerend := tv.LayoutLines(tbe.Reg.Start.Ln, tbe.Reg.End.Ln, false)
			if rerend {
				fmt.Printf("rend all\n")
				tv.RenderAllLines()
			} else {
				tv.RenderLines(tbe.Reg.Start.Ln, tbe.Reg.End.Ln)
			}
		}
	case TextBufDelete:
		if tv.Renders == nil { // not init yet
			return
		}
		tbe := data.(*TextBufEdit)
		if tbe.Reg.Start.Ln != tbe.Reg.End.Ln {
			tv.DeleteLines(tbe)
		} else {
			rerend := tv.LayoutLines(tbe.Reg.Start.Ln, tbe.Reg.End.Ln, true)
			if rerend {
				tv.RenderAllLines()
			} else {
				tv.RenderLines(tbe.Reg.Start.Ln, tbe.Reg.End.Ln)
			}
		}
	}
}

///////////////////////////////////////////////////////////////////////////////
//  Text formatting and rendering

// HasHi returns true if there are highighting parameters set
func (tv *TextView) HasHi() bool {
	if tv.Buf == nil {
		return false
	}
	if tv.Buf.HiLang == "" || tv.HiStyle == "" {
		return false
	}
	return true
}

// HiInit initializes the syntax highlighting for current Hi params
func (tv *TextView) HiInit() {
	if !tv.HasHi() {
		return
	}
	if tv.Buf.HiLang == tv.lastHiLang && tv.HiStyle == tv.lastHiStyle {
		return
	}
	tv.lexer = chroma.Coalesce(lexers.Get(tv.Buf.HiLang))
	tv.formatter = html.New(html.WithClasses(), html.TabWidth(tv.Sty.Text.TabSize))
	tv.style = styles.Get(string(tv.HiStyle))
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

	tv.lastHiLang = tv.Buf.HiLang
	tv.lastHiStyle = tv.HiStyle
}

// RenderSize is the size we should pass to text rendering, based on alloc
func (tv *TextView) RenderSize() gi.Vec2D {
	spc := tv.Sty.BoxSpace()
	if tv.Par == nil {
		return gi.Vec2DZero
	}
	pari, _ := gi.KiToNode2D(tv.Par)
	parw := pari.AsLayout2D()
	if parw == nil {
		log.Printf("giv.TextView Programmer Error: A TextView MUST be located within a parent Layout object -- instead parent is %v at: %v\n", pari.Type(), tv.PathUnique())
		return gi.Vec2DZero
	}
	parw.SetReRenderAnchor()
	paloc := parw.LayData.AllocSizeOrig
	if !paloc.IsZero() {
		tv.RenderSz = paloc.Sub(parw.ExtraSize).SubVal(spc * 2)
	} else {
		sz := tv.LayData.AllocSizeOrig
		if sz.IsZero() {
			sz = tv.LayData.SizePrefOrMax()
		}
		if !sz.IsZero() {
			sz.SetSubVal(2 * spc)
		}
		tv.RenderSz = sz
		// fmt.Printf("alloc rendersz: %v\n", tv.RenderSz)
	}
	tv.RenderSz.X -= tv.LineNoOff
	// fmt.Printf("rendersz: %v\n", tv.RenderSz)
	return tv.RenderSz
}

// HiAllLines does highlighting of all lines, in a separate goroutine because it is slow
func (tv *TextView) HiAllLines() {
	var htmlBuf bytes.Buffer
	iterator, err := tv.lexer.Tokenise(nil, string(tv.Buf.Txt)) // todo: unfortunate conversion here..
	err = tv.formatter.Format(&htmlBuf, tv.style, iterator)
	if err != nil {
		log.Println(err)
		return
	}
	mtlns := bytes.Split(htmlBuf.Bytes(), []byte("\n"))

	sz := tv.RenderSz
	sty := &tv.Sty
	fst := sty.Font
	fst.BgColor.SetColor(nil)

	// tv.MarkupMu.Lock() // todo: turn these on if using goroutine
	maxln := len(mtlns) - 1
	for ln := 0; ln < maxln; ln++ {
		if len(tv.Markup) != tv.NLines { // update happend!
			break
		}
		mt := mtlns[ln]
		mt = bytes.TrimPrefix(mt, []byte(`</span>`)) // leftovers
		tv.Markup[ln] = mt
		tv.HasMarkup[ln] = true
		tv.Renders[ln].SetHTMLPre(tv.Markup[ln], &fst, &sty.Text, &sty.UnContext, tv.CSS)
		tv.Renders[ln].LayoutStdLR(&sty.Text, &sty.Font, &sty.UnContext, sz)
	}
	// tv.MarkupMu.Unlock()
	// tv.UpdateSig() // turn this on if using goroutine
}

// LayoutAllLines generates TextRenders of lines from our TextBuf, using any
// highlighter that might be present, and returns whether the current rendered
// size is different from what it was previously
func (tv *TextView) LayoutAllLines(inLayout bool) bool {
	if inLayout && tv.reLayout {
		return false
	}
	if tv.Buf == nil || tv.Buf.NLines == 0 {
		tv.NLines = 0
		return tv.ResizeIfNeeded(image.ZP)
	}
	if tv.Sty.Font.Size.Val == 0 { // not yet styled
		tv.StyleTextView()
	}

	tv.MarkupMu.Lock() // wait for prior markup if it is still happening
	tv.HiInit()

	// fmt.Printf("layout all: %v\n", tv.Nm)

	tv.NLines = tv.Buf.NLines
	nln := tv.NLines
	if cap(tv.Markup) >= nln {
		tv.Markup = tv.Markup[:nln]
	} else {
		tv.Markup = make([][]byte, nln)
	}
	if cap(tv.HasMarkup) >= nln {
		tv.HasMarkup = tv.HasMarkup[:nln]
		for i := range tv.HasMarkup {
			tv.HasMarkup[i] = false
		}
	} else {
		tv.HasMarkup = make([]bool, nln)
	}
	if cap(tv.Renders) >= nln {
		tv.Renders = tv.Renders[:nln]
	} else {
		tv.Renders = make([]gi.TextRender, nln)
	}
	if cap(tv.Offs) >= nln {
		tv.Offs = tv.Offs[:nln]
	} else {
		tv.Offs = make([]float32, nln)
	}

	tv.VisSizes()
	sz := tv.RenderSz

	if tv.HasHi() {
		//		go tv.HiAllLines() // this is not sufficiently reliable right now - just do it
		tv.HiAllLines()
	}

	// fmt.Printf("rendersize: %v\n", sz)
	sty := &tv.Sty
	fst := sty.Font
	fst.BgColor.SetColor(nil)
	off := float32(0)
	mxwd := float32(0)

	for ln := 0; ln < nln; ln++ {
		if tv.HasMarkup[ln] {
			tv.Renders[ln].SetHTMLPre(tv.Markup[ln], &fst, &sty.Text, &sty.UnContext, tv.CSS)
		} else {
			tv.Renders[ln].SetHTMLPre([]byte(string(tv.Buf.Lines[ln])), &fst, &sty.Text, &sty.UnContext, tv.CSS)
		}
		tv.Renders[ln].LayoutStdLR(&sty.Text, &sty.Font, &sty.UnContext, sz)
		tv.Offs[ln] = off
		lsz := gi.Max32(tv.Renders[ln].Size.Y, tv.LineHeight)
		off += lsz
		mxwd = gi.Max32(mxwd, tv.Renders[ln].Size.X)
	}
	tv.MarkupMu.Unlock()
	extraHalf := tv.LineHeight * 0.5 * float32(tv.VisSize.Y)
	nwSz := gi.Vec2D{mxwd, off + extraHalf}.ToPointCeil()
	// fmt.Printf("lay lines: diff: %v  old: %v  new: %v\n", diff, tv.LinesSize, nwSz)
	if inLayout {
		tv.LinesSize = nwSz
		return tv.SetSize()
	} else {
		return tv.ResizeIfNeeded(nwSz)
	}
}

// SetSize updates our size only if larger than our allocation
func (tv *TextView) SetSize() bool {
	sty := &tv.Sty
	spc := sty.BoxSpace()
	rndsz := tv.RenderSz
	rndsz.X += tv.LineNoOff
	netsz := gi.Vec2D{float32(tv.LinesSize.X) + tv.LineNoOff, float32(tv.LinesSize.Y)}
	cursz := tv.LayData.AllocSize.SubVal(2 * spc)
	if cursz.X < 10 || cursz.Y < 10 {
		nwsz := netsz.Max(rndsz)
		tv.Size2DFromWH(nwsz.X, nwsz.Y)
		return true
	}
	// fmt.Printf("netsz: %v  cursz: %v rndsz: %v\n", netsz, cursz, rndsz)
	cursz = cursz.Max(rndsz)
	if netsz.X > cursz.X || netsz.Y > cursz.Y {
		nwsz := netsz.Max(cursz)
		tv.Size2DFromWH(nwsz.X, nwsz.Y)
		return true
	}
	return false
}

// ResizeIfNeeded resizes the edit area if different from current setting --
// returns true if resizing was performed
func (tv *TextView) ResizeIfNeeded(nwSz image.Point) bool {
	if nwSz == tv.LinesSize {
		return false
	}
	// fmt.Printf("needs resize: %v\n", nwSz)
	tv.LinesSize = nwSz
	diff := tv.SetSize()
	if !diff {
		return false
	}
	ly := tv.ParentScrollLayout()
	if ly != nil {
		tv.reLayout = true
		ly.GatherSizes() // can't call Size2D b/c that resets layout
		ly.Layout2DTree()
		tv.reLayout = false
	}
	// tv.SetFullReRender()
	return true
}

// LayoutLines generates render of given range of lines (including
// highlighting). end is *inclusive* line.  if highlighter generates an error
// on a line, or word-wrap causes lines to increase in number of spans, then
// calls LayoutAllLines to do a full-reparse, and returns true to indicate
// need for a full re-render -- otherwise returns false and just these lines
// need to be re-rendered..  isDel means this is a delete and thus offsets for all
// higher lines need to be recomputed
func (tv *TextView) LayoutLines(st, ed int, isDel bool) bool {
	// tv.MarkupMu.Lock()
	sty := &tv.Sty
	fst := sty.Font
	fst.BgColor.SetColor(nil)
	mxwd := float32(tv.LinesSize.X)
	rerend := false
	for ln := st; ln <= ed; ln++ {
		curspans := len(tv.Renders[ln].Spans)
		if tv.HasHi() {
			var htmlBuf bytes.Buffer
			iterator, err := tv.lexer.Tokenise(nil, string(tv.Buf.Lines[ln])+"\n")
			// add \n b/c it needs to see that for comments..
			err = tv.formatter.Format(&htmlBuf, tv.style, iterator)
			if err != nil {
				log.Println(err)
				tv.Buf.LinesToBytes() // need to update buffer -- todo: redundant across views
				tv.LayoutAllLines(false)
				return true
			}
			b := htmlBuf.Bytes()
			lfidx := bytes.Index(b, []byte("\n"))
			if lfidx > 0 {
				b = b[:lfidx]
			}
			tv.Markup[ln] = b
			tv.HasMarkup[ln] = true
			tv.Renders[ln].SetHTMLPre(tv.Markup[ln], &fst, &sty.Text, &sty.UnContext, tv.CSS)
		} else {
			tv.Renders[ln].SetHTMLPre([]byte(string(tv.Buf.Lines[ln])), &fst, &sty.Text, &sty.UnContext, tv.CSS)
		}
		tv.Renders[ln].LayoutStdLR(&sty.Text, &sty.Font, &sty.UnContext, tv.RenderSz)
		nwspans := len(tv.Renders[ln].Spans)
		if nwspans != curspans && (nwspans > 1 || curspans > 1) {
			rerend = true
		}
		mxwd = gi.Max32(mxwd, tv.Renders[ln].Size.X)
	}
	// tv.MarkupMu.Unlock()

	// update all offsets to end of text
	if rerend || isDel || st != ed {
		ofst := st - 1
		if ofst < 0 {
			ofst = 0
		}
		off := tv.Offs[ofst]
		for ln := ofst; ln < tv.NLines; ln++ {
			tv.Offs[ln] = off
			lsz := gi.Max32(tv.Renders[ln].Size.Y, tv.LineHeight)
			off += lsz
		}
		extraHalf := tv.LineHeight * 0.5 * float32(tv.VisSize.Y)
		nwSz := gi.Vec2D{mxwd, off + extraHalf}.ToPointCeil()
		tv.ResizeIfNeeded(nwSz)
	} else {
		nwSz := gi.Vec2D{mxwd, 0}.ToPointCeil()
		nwSz.Y = tv.LinesSize.Y
		tv.ResizeIfNeeded(nwSz)
	}
	return rerend
}

///////////////////////////////////////////////////////////////////////////////
//  Cursor Navigation

// CursorMovedSig sends the signal that cursor has moved
func (tv *TextView) CursorMovedSig() {
	tv.TextViewSig.Emit(tv.This, int64(TextViewCursorMoved), tv.CursorPos)
}

// ValidCursor returns a cursor that is in a valid range
func (tv *TextView) ValidCursor(pos TextPos) TextPos {
	if tv.NLines == 0 {
		return TextPosZero
	}
	if pos.Ln < 0 {
		pos.Ln = 0
	}
	pos.Ln = gi.MinInt(pos.Ln, len(tv.Buf.Lines)-1)
	llen := len(tv.Buf.Lines[pos.Ln])
	pos.Ch = gi.MinInt(pos.Ch, llen)
	if pos.Ch < 0 {
		pos.Ch = 0
	}
	return pos
}

// ValidateCursor sets current cursor to a valid cursor position
func (tv *TextView) ValidateCursor() {
	tv.CursorPos = tv.ValidCursor(tv.CursorPos)
}

// SetCursor sets a new cursor position, enforcing it in range
func (tv *TextView) SetCursor(pos TextPos) {
	if tv.NLines == 0 {
		tv.CursorPos = TextPosZero
		return
	}
	tv.CursorPos = tv.ValidCursor(pos)
	tv.CursorMovedSig()
}

// SetCursorShow sets a new cursor position, enforcing it in range, and shows
// the cursor (scroll to if hidden, render)
func (tv *TextView) SetCursorShow(pos TextPos) {
	tv.SetCursor(pos)
	tv.ScrollCursorToCenterIfHidden()
	tv.RenderCursor(true)
}

// CursorSelect updates selection based on cursor movements, given starting
// cursor position and tv.CursorPos is current
func (tv *TextView) CursorSelect(org TextPos) {
	if !tv.SelectMode {
		return
	}
	if org.IsLess(tv.SelectReg.Start) {
		tv.SelectReg.Start = tv.CursorPos
	} else if !tv.CursorPos.IsLess(tv.SelectReg.Start) { // >
		tv.SelectReg.End = tv.CursorPos
	} else {
		tv.SelectReg.Start = tv.CursorPos
	}
	tv.RenderSelectLines()
}

// CursorForward moves the cursor forward
func (tv *TextView) CursorForward(steps int) {
	tv.ValidateCursor()
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
	if wln := tv.WrappedLines(tv.CursorPos.Ln); wln > 1 {
		si, ri, ok := tv.WrappedLineNo(tv.CursorPos)
		if ok && si > 0 {
			tv.CursorCol = ri
		} else {
			tv.CursorCol = tv.CursorPos.Ch
		}
	}
	tv.SetCursorShow(tv.CursorPos)
	tv.CursorSelect(org)
}

// WrappedLines returns the number of wrapped lines (spans) for given line number
func (tv *TextView) WrappedLines(ln int) int {
	if ln >= len(tv.Renders) {
		return 0
	}
	return len(tv.Renders[ln].Spans)
}

// WrappedLineNo returns the wrapped line number (span index) and rune index
// within that span of the given character position within line in position,
// and false if out of range
func (tv *TextView) WrappedLineNo(pos TextPos) (si, ri int, ok bool) {
	if pos.Ln >= len(tv.Renders) {
		return 0, 0, false
	}
	return tv.Renders[pos.Ln].RuneSpanPos(pos.Ch)
}

// CursorDown moves the cursor down line(s)
func (tv *TextView) CursorDown(steps int) {
	tv.ValidateCursor()
	org := tv.CursorPos
	pos := tv.CursorPos
	for i := 0; i < steps; i++ {
		gotwrap := false
		if wln := tv.WrappedLines(pos.Ln); wln > 1 {
			si, ri, ok := tv.WrappedLineNo(pos)
			if ok && si < wln-1 {
				nwc, ok := tv.Renders[pos.Ln].SpanPosToRuneIdx(si+1, ri)
				if ok {
					pos.Ch = nwc
					gotwrap = true
				}
			}
		}
		if !gotwrap {
			pos.Ln++
			if pos.Ln >= tv.NLines {
				pos.Ln = tv.NLines - 1
				break
			}
			mxlen := gi.MinInt(len(tv.Buf.Lines[pos.Ln]), tv.CursorCol)
			if tv.CursorCol < mxlen {
				pos.Ch = tv.CursorCol
			} else {
				pos.Ch = mxlen
			}
		}
	}
	tv.SetCursorShow(pos)
	tv.CursorSelect(org)
}

// CursorPageDown moves the cursor down page(s), where a page is defined
// dynamically as just moving the cursor off the screen
func (tv *TextView) CursorPageDown(steps int) {
	tv.ValidateCursor()
	org := tv.CursorPos
	for i := 0; i < steps; i++ {
		lvln := tv.LastVisibleLine(tv.CursorPos.Ln)
		tv.CursorPos.Ln = lvln
		if tv.CursorPos.Ln >= tv.NLines {
			tv.CursorPos.Ln = tv.NLines - 1
		}
		tv.CursorPos.Ch = gi.MinInt(len(tv.Buf.Lines[tv.CursorPos.Ln]), tv.CursorCol)
		tv.ScrollCursorToTop()
		tv.RenderCursor(true)
	}
	tv.SetCursor(tv.CursorPos)
	tv.CursorSelect(org)
}

// CursorBackward moves the cursor backward
func (tv *TextView) CursorBackward(steps int) {
	tv.ValidateCursor()
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
	if wln := tv.WrappedLines(tv.CursorPos.Ln); wln > 1 {
		si, ri, ok := tv.WrappedLineNo(tv.CursorPos)
		if ok && si > 0 {
			tv.CursorCol = ri
		} else {
			tv.CursorCol = tv.CursorPos.Ch
		}
	}
	tv.SetCursorShow(tv.CursorPos)
	tv.CursorSelect(org)
}

// CursorUp moves the cursor up line(s)
func (tv *TextView) CursorUp(steps int) {
	tv.ValidateCursor()
	org := tv.CursorPos
	pos := tv.CursorPos
	for i := 0; i < steps; i++ {
		gotwrap := false
		if wln := tv.WrappedLines(pos.Ln); wln > 1 {
			si, ri, ok := tv.WrappedLineNo(pos)
			if ok && si > 0 {
				ri = tv.CursorCol
				nwc, _ := tv.Renders[pos.Ln].SpanPosToRuneIdx(si-1, ri)
				pos.Ch = nwc
				gotwrap = true
			}
		}
		if !gotwrap {
			pos.Ln--
			if pos.Ln < 0 {
				pos.Ln = 0
				break
			}
			if wln := tv.WrappedLines(pos.Ln); wln > 1 { // just entered end of wrapped line
				si := wln - 1
				ri := tv.CursorCol
				nwc, _ := tv.Renders[pos.Ln].SpanPosToRuneIdx(si, ri)
				pos.Ch = nwc
			} else {
				mxlen := gi.MinInt(len(tv.Buf.Lines[pos.Ln]), tv.CursorCol)
				if tv.CursorCol < mxlen {
					pos.Ch = tv.CursorCol
				} else {
					pos.Ch = mxlen
				}
			}
		}
	}
	tv.SetCursorShow(pos)
	tv.CursorSelect(org)
}

// CursorPageUp moves the cursor up page(s), where a page is defined
// dynamically as just moving the cursor off the screen
func (tv *TextView) CursorPageUp(steps int) {
	tv.ValidateCursor()
	org := tv.CursorPos
	for i := 0; i < steps; i++ {
		lvln := tv.FirstVisibleLine(tv.CursorPos.Ln)
		tv.CursorPos.Ln = lvln
		if tv.CursorPos.Ln <= 0 {
			tv.CursorPos.Ln = 0
		}
		tv.CursorPos.Ch = gi.MinInt(len(tv.Buf.Lines[tv.CursorPos.Ln]), tv.CursorCol)
		tv.ScrollCursorToBottom()
		tv.RenderCursor(true)
	}
	tv.SetCursor(tv.CursorPos)
	tv.CursorSelect(org)
}

// CursorRecenter re-centers the view around the cursor position, toggling
// between putting cursor in middle, top, and bottom of view
func (tv *TextView) CursorRecenter() {
	tv.ValidateCursor()
	cur := (tv.lastRecenter + 1) % 3
	switch cur {
	case 0:
		tv.ScrollCursorToBottom()
	case 1:
		tv.ScrollCursorToVertCenter()
	case 2:
		tv.ScrollCursorToTop()
	}
	tv.lastRecenter = cur
}

// CursorStartLine moves the cursor to the start of the line, updating selection
// if select mode is active
func (tv *TextView) CursorStartLine() {
	tv.ValidateCursor()
	org := tv.CursorPos
	tv.CursorPos.Ch = 0
	tv.CursorCol = tv.CursorPos.Ch
	tv.SetCursor(tv.CursorPos)
	tv.ScrollCursorToLeft()
	tv.RenderCursor(true)
	tv.CursorSelect(org)
}

// CursorStartDoc moves the cursor to the start of the text, updating selection
// if select mode is active
func (tv *TextView) CursorStartDoc() {
	tv.ValidateCursor()
	org := tv.CursorPos
	tv.CursorPos.Ln = 0
	tv.CursorPos.Ch = 0
	tv.CursorCol = tv.CursorPos.Ch
	tv.SetCursor(tv.CursorPos)
	tv.ScrollCursorToTop()
	tv.RenderCursor(true)
	tv.CursorSelect(org)
}

// CursorEndLine moves the cursor to the end of the text
func (tv *TextView) CursorEndLine() {
	tv.ValidateCursor()
	org := tv.CursorPos
	tv.CursorPos.Ch = len(tv.Buf.Lines[tv.CursorPos.Ln])
	if wln := tv.WrappedLines(tv.CursorPos.Ln); wln > 1 {
		si, ri, ok := tv.WrappedLineNo(tv.CursorPos)
		if ok && si > 0 {
			tv.CursorCol = ri
		} else {
			tv.CursorCol = tv.CursorPos.Ch
		}
	}
	tv.SetCursor(tv.CursorPos)
	tv.ScrollCursorToRight()
	tv.RenderCursor(true)
	tv.CursorSelect(org)
}

// CursorEndDoc moves the cursor to the end of the text, updating selection if
// select mode is active
func (tv *TextView) CursorEndDoc() {
	tv.ValidateCursor()
	updt := tv.UpdateStart()
	defer tv.UpdateEnd(updt)
	org := tv.CursorPos
	tv.CursorPos.Ln = gi.MaxInt(tv.NLines-1, 0)
	tv.CursorPos.Ch = len(tv.Buf.Lines[tv.CursorPos.Ln])
	tv.CursorCol = tv.CursorPos.Ch
	tv.SetCursor(tv.CursorPos)
	tv.ScrollCursorToBottom()
	tv.RenderCursor(true)
	tv.CursorSelect(org)
}

// todo: ctrl+backspace = delete word
// shift+arrow = select
// uparrow = start / down = end

// CursorBackspace deletes character(s) immediately before cursor
func (tv *TextView) CursorBackspace(steps int) {
	tv.ValidateCursor()
	org := tv.CursorPos
	if tv.HasSelection() {
		tv.DeleteSelection()
		tv.SetCursorShow(org)
		return
	}
	// note: no update b/c signal from buf will drive update
	tv.CursorBackward(steps)
	tv.ScrollCursorToCenterIfHidden()
	tv.RenderCursor(true)
	tv.Buf.DeleteText(tv.CursorPos, org, true)
}

// CursorDelete deletes character(s) immediately after the cursor
func (tv *TextView) CursorDelete(steps int) {
	tv.ValidateCursor()
	if tv.HasSelection() {
		tv.DeleteSelection()
		return
	}
	// note: no update b/c signal from buf will drive update
	org := tv.CursorPos
	tv.CursorForward(steps)
	tv.Buf.DeleteText(org, tv.CursorPos, true)
	tv.SetCursorShow(org)
}

// CursorKill deletes text from cursor to end of text
func (tv *TextView) CursorKill() {
	tv.ValidateCursor()
	org := tv.CursorPos
	if tv.CursorPos.Ch == 0 && len(tv.Buf.Lines[tv.CursorPos.Ln]) == 0 {
		tv.CursorForward(1)
	} else {
		tv.CursorEndLine()
	}
	tv.Buf.DeleteText(org, tv.CursorPos, true)
	tv.SetCursorShow(org)
}

// JumpToLinePrompt jumps to given line number (minus 1) from prompt
func (tv *TextView) JumpToLinePrompt() {
	gi.StringPromptDialog(tv.Viewport, "", "Line no..",
		gi.DlgOpts{Title: "Jump To Line", Prompt: "Line Number to jump to"},
		tv.This, func(recv, send ki.Ki, sig int64, data interface{}) {
			dlg := send.(*gi.Dialog)
			if sig == int64(gi.DialogAccepted) {
				val := gi.StringPromptDialogValue(dlg)
				ln, ok := kit.ToInt(val)
				if ok {
					tv.JumpToLine(int(ln))
				}
			}
		})

}

// JumpToLine jumps to given line number (minus 1)
func (tv *TextView) JumpToLine(ln int) {
	tv.SetCursorShow(TextPos{Ln: ln})
}

///////////////////////////////////////////////////////////////////////////////
//    Undo / Redo

// Undo undoes previous action
func (tv *TextView) Undo() {
	tbe := tv.Buf.Undo()
	if tbe != nil {
		if tbe.Delete { // now an insert
			tv.SetCursor(tbe.Reg.End)
		} else {
			tv.SetCursor(tbe.Reg.Start)
		}
	}
	tv.ScrollCursorToCenterIfHidden()
	tv.RenderCursor(true)
}

// Redo redoes previously undone action
func (tv *TextView) Redo() {
	tbe := tv.Buf.Redo()
	if tbe != nil {
		if tbe.Delete {
			tv.SetCursor(tbe.Reg.Start)
		} else {
			tv.SetCursor(tbe.Reg.End)
		}
	}
	tv.ScrollCursorToCenterIfHidden()
	tv.RenderCursor(true)
}

///////////////////////////////////////////////////////////////////////////////
//    Search / Find

// TextViewMaxFindHighlights is the maximum number of regions to highlight on find
var TextViewMaxFindHighlights = 50

// FindMatches finds the matches with given search string (literal, not regex)
// and case sensitivity, updates highlights for all.  returns false if none
// found
func (tv *TextView) FindMatches(find string, useCase bool) bool {
	fsz := len(find)
	if fsz == 0 {
		tv.Highlights = nil
		return false
	}
	if useCase {
		_, tv.SearchMatches = tv.Buf.Search(find)
	} else {
		_, tv.SearchMatches = tv.Buf.SearchCI(find)
	}
	matches := tv.SearchMatches
	if len(matches) == 0 {
		tv.Highlights = nil
		return false
	}
	hi := make([]TextRegion, len(matches))
	for i, m := range matches {
		hi[i] = NewTextRegionLen(m, fsz)
		if i > TextViewMaxFindHighlights {
			break
		}
	}
	tv.Highlights = hi
	tv.RenderAllLines()
	return true
}

// ISearchMatches finds ISearch matches -- returns true if there are any
func (tv *TextView) ISearchMatches() bool {
	return tv.FindMatches(tv.ISearchString, tv.ISearchCase)
}

// ISearchSig sends the signal that ISearch is updated
func (tv *TextView) ISearchSig() {
	tv.TextViewSig.Emit(tv.This, int64(TextViewISearch), tv.CursorPos)
}

// ISearch is an emacs-style interactive search mode -- this is called when
// the search command itself is entered
func (tv *TextView) ISearch() {
	if tv.ISearchMode {
		if tv.ISearchString != "" { // already searching -- find next
			sz := len(tv.SearchMatches)
			if sz > 0 {
				if tv.SearchPos < sz-1 {
					tv.SearchPos++
				} else {
					tv.SearchPos = 0
				}
				pos := tv.SearchMatches[tv.SearchPos]
				tv.SelectReg = NewTextRegionLen(pos, len(tv.ISearchString))
				tv.SetCursor(pos)
				tv.ScrollCursorToCenterIfHidden()
				tv.RenderSelectLines()
				tv.ISearchSig()
			}
		} else { // restore prev
			if tv.PrevISearchString != "" {
				tv.ISearchString = tv.PrevISearchString
				tv.ISearchCase = tv.PrevISearchCase
				tv.PrevISearchString = "" // prevents future resets
				tv.SearchPos = -1
				tv.ISearchMatches()
				tv.ISearch()
			}
			// nothing..
		}
	} else {
		tv.ISearchMode = true
		tv.ISearchStartPos = tv.CursorPos
		tv.ISearchCase = false
		tv.SearchMatches = nil
		tv.SearchPos = -1
		tv.ISearchSig()
	}
}

// ISearchKeyInput is an emacs-style interactive search mode -- this is called
// when keys are typed while in search mode
func (tv *TextView) ISearchKeyInput(r rune) {
	if tv.ISearchString == tv.PrevISearchString { // undo starting point
		tv.ISearchString = ""
	}
	if unicode.IsUpper(r) { // todo: more complex
		tv.ISearchCase = true
	}
	tv.ISearchString += string(r)
	tv.ISearchMatches()
	sz := len(tv.SearchMatches)
	if sz == 0 {
		tv.SearchPos = -1
		tv.ISearchSig()
		return
	}
	got := false
	for i, pos := range tv.SearchMatches {
		if tv.CursorPos.IsLess(pos) {
			tv.SearchPos = i
			tv.SelectReg = NewTextRegionLen(pos, len(tv.ISearchString))
			tv.SetCursor(pos)
			tv.ScrollCursorToCenterIfHidden()
			tv.RenderSelectLines()
			tv.ISearchSig()
			got = true
			break
		}
	}
	if !got {
		tv.SearchPos = 0
		pos := tv.SearchMatches[0]
		tv.SelectReg = NewTextRegionLen(pos, len(tv.ISearchString))
		tv.SetCursor(pos)
		tv.ScrollCursorToCenterIfHidden()
		tv.RenderSelectLines()
		tv.ISearchSig()
	}
}

// ISearchBackspace gets rid of one item in search string
func (tv *TextView) ISearchBackspace() {
	if tv.ISearchString == tv.PrevISearchString { // undo starting point
		tv.ISearchString = ""
		tv.SearchMatches = nil
		tv.SearchPos = -1
		tv.ISearchSig()
	}
	if len(tv.ISearchString) <= 1 {
		tv.ISearchString = ""
		tv.ISearchCase = false
		return
	}
	tv.ISearchString = tv.ISearchString[:len(tv.ISearchString)-1]
	tv.ISearchMatches()
	sz := len(tv.SearchMatches)
	if sz == 0 {
		tv.SearchPos = -1
		tv.ISearchSig()
		return
	}
	got := false
	for i, pos := range tv.SearchMatches {
		if pos.Ln >= tv.CursorPos.Ln {
			tv.SearchPos = i
			tv.SetCursor(pos)
			tv.ScrollCursorToCenterIfHidden()
			tv.ISearchSig()
			got = true
			break
		}
	}
	if !got {
		tv.SearchPos = 0
		pos := tv.SearchMatches[0]
		tv.SetCursor(pos)
		tv.ScrollCursorToCenterIfHidden()
		tv.ISearchSig()
	}
}

// ISearchCancel cancels ISearch mode
func (tv *TextView) ISearchCancel() {
	if !tv.ISearchMode {
		return
	}
	tv.PrevISearchString = tv.ISearchString
	tv.PrevISearchCase = tv.ISearchCase
	tv.ISearchString = ""
	tv.ISearchCase = false
	tv.ISearchMode = false
	tv.SearchPos = -1
	tv.SearchMatches = nil
	tv.Highlights = nil
	tv.Refresh()
	tv.ISearchSig()
}

// EscPressed emitted for KeyFunAbort or KeyFunCancelSelect -- effect depends on state..
func (tv *TextView) EscPressed() {
	switch {
	case tv.ISearchMode:
		tv.ISearchCancel()
		tv.SetCursorShow(tv.ISearchStartPos)
	case tv.HasSelection():
		tv.SelectReset()
	}
}

///////////////////////////////////////////////////////////////////////////////
//    Selection

// ClearSelected resets both the global selected flag and any current selection
func (tv *TextView) ClearSelected() {
	tv.WidgetBase.ClearSelected()
	tv.SelectReset()
}

// HasSelection returns whether there is a selected region of text
func (tv *TextView) HasSelection() bool {
	if tv.SelectReg.Start.IsLess(tv.SelectReg.End) {
		return true
	}
	return false
}

// Selection returns the currently selected text as a TextBufEdit, which
// captures start, end, and full lines in between -- nil if no selection
func (tv *TextView) Selection() *TextBufEdit {
	if tv.HasSelection() {
		return tv.Buf.Region(tv.SelectReg.Start, tv.SelectReg.End)
	}
	return nil
}

// SelectModeToggle toggles the SelectMode, updating selection with cursor movement
func (tv *TextView) SelectModeToggle() {
	if tv.SelectMode {
		tv.SelectMode = false
	} else {
		tv.SelectMode = true
		tv.SelectReg.Start = tv.CursorPos
		tv.SelectReg.End = tv.SelectReg.Start
	}
}

// SelectAll selects all the text
func (tv *TextView) SelectAll() {
	updt := tv.UpdateStart()
	tv.SelectReg.Start = TextPosZero
	tv.SelectReg.End = TextPos{tv.Buf.NLines - 1, len(tv.Buf.Lines[tv.Buf.NLines-1])}
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
	// tv.SelectReg.Start = tv.CursorPos
	// if tv.SelectReg.Start >= sz {
	// 	tv.SelectReg.Start = sz - 2
	// }
	// if !tv.IsWordBreak(tv.EditTxt[tv.SelectReg.Start]) {
	// 	for tv.SelectReg.Start > 0 {
	// 		if tv.IsWordBreak(tv.EditTxt[tv.SelectReg.Start-1]) {
	// 			break
	// 		}
	// 		tv.SelectReg.Start--
	// 	}
	// 	tv.SelectReg.End = tv.CursorPos + 1
	// 	for tv.SelectReg.End < sz {
	// 		if tv.IsWordBreak(tv.EditTxt[tv.SelectReg.End]) {
	// 			break
	// 		}
	// 		tv.SelectReg.End++
	// 	}
	// } else { // keep the space start -- go to next space..
	// 	tv.SelectReg.End = tv.CursorPos + 1
	// 	for tv.SelectReg.End < sz {
	// 		if !tv.IsWordBreak(tv.EditTxt[tv.SelectReg.End]) {
	// 			break
	// 		}
	// 		tv.SelectReg.End++
	// 	}
	// 	for tv.SelectReg.End < sz {
	// 		if tv.IsWordBreak(tv.EditTxt[tv.SelectReg.End]) {
	// 			break
	// 		}
	// 		tv.SelectReg.End++
	// 	}
	// }
}

// SelectReset resets the selection
func (tv *TextView) SelectReset() {
	tv.SelectMode = false
	if !tv.HasSelection() {
		return
	}
	stln := tv.SelectReg.Start.Ln
	edln := tv.SelectReg.End.Ln
	tv.SelectReg = TextRegionZero
	tv.PrevSelectReg = TextRegionZero
	tv.RenderLines(stln, edln)
}

// RenderSelectLines renders the lines within the current selection region
func (tv *TextView) RenderSelectLines() {
	if tv.PrevSelectReg == TextRegionZero {
		tv.RenderLines(tv.SelectReg.Start.Ln, tv.SelectReg.End.Ln)
	} else {
		stln := gi.MinInt(tv.SelectReg.Start.Ln, tv.PrevSelectReg.Start.Ln)
		edln := gi.MaxInt(tv.SelectReg.End.Ln, tv.PrevSelectReg.End.Ln)
		tv.RenderLines(stln, edln)
	}
	tv.PrevSelectReg = tv.SelectReg
}

///////////////////////////////////////////////////////////////////////////////
//    Cut / Copy / Paste

// Cut cuts any selected text and adds it to the clipboard, also returns cut text
func (tv *TextView) Cut() *TextBufEdit {
	org := tv.SelectReg.Start
	cut := tv.DeleteSelection()
	if cut != nil {
		oswin.TheApp.ClipBoard().Write(mimedata.NewTextBytes(cut.ToBytes()))
	}
	tv.SetCursorShow(org)
	return cut
}

// DeleteSelection deletes any selected text, without adding to clipboard --
// returns text deleted as TextBufEdit (nil if none)
func (tv *TextView) DeleteSelection() *TextBufEdit {
	tbe := tv.Buf.DeleteText(tv.SelectReg.Start, tv.SelectReg.End, true)
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
		if tv.SelectReg.Start.IsLess(tv.CursorPos) && tv.CursorPos.IsLess(tv.SelectReg.End) {
			tv.DeleteSelection()
		}
		tv.InsertAtCursor(data.TypeData(mimedata.TextPlain))
	}
}

// InsertAtCursor inserts given text at current cursor position
func (tv *TextView) InsertAtCursor(txt []byte) {
	if tv.HasSelection() {
		tv.Cut()
	}
	sz := len(txt)
	if sz == 1 {
		npos := tv.CursorPos
		npos.Ch++
		tv.SetCursorShow(npos)
		// tv.SetCursor(npos)
	}
	tbe := tv.Buf.InsertText(tv.CursorPos, txt, true)
	if tv.CursorPos != tbe.Reg.End {
		tv.SetCursorShow(tbe.Reg.End)
		// tv.SetCursor(tbe.Reg.End)
	}
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

///////////////////////////////////////////////////////////////////////////////
//    Complete

// OfferComplete pops up a menu of possible completions
func (tv *TextView) OfferComplete(forcecomplete bool) {
	if tv.Complete == nil || tv.ISearchMode {
		return
	}
	if !tv.Opts.Completion && !forcecomplete {
		return
	}
	win := tv.ParentWindow()
	if gi.PopupIsCompleter(win.Popup) {
		win.ClosePopup(win.Popup)
	}

	st := TextPos{tv.CursorPos.Ln, 0}
	en := TextPos{tv.CursorPos.Ln, tv.CursorPos.Ch}
	tbe := tv.Buf.Region(st, en)
	var s string
	if tbe != nil {
		s = string(tbe.ToBytes())
		s = strings.TrimLeft(s, " \t") // trim ' ' and '\t'
	}
	if len(s) == 0 && !forcecomplete {
		return
	}

	tpos := token.Position{} // text position
	count := tv.Buf.ByteOffs[tv.CursorPos.Ln] + tv.CursorPos.Ch
	tpos.Line = tv.CursorPos.Ln
	tpos.Column = tv.CursorPos.Ch
	tpos.Offset = count
	tpos.Filename = ""
	cpos := tv.CharStartPos(tv.CursorPos).ToPoint() // physical location
	cpos.X += 5
	cpos.Y += 10
	tv.Complete.ShowCompletions(s, tpos, tv.Viewport, cpos)
}

// CompleteText edits the text using the string chosen from the completion menu
func (tv *TextView) CompleteText(s string) {
	win := tv.ParentWindow()
	win.ClosePopup(win.Popup)

	st := TextPos{tv.CursorPos.Ln, 0}
	en := TextPos{tv.CursorPos.Ln, tv.CursorPos.Ch}
	tbe := tv.Buf.Region(st, en)
	tbes := string(tbe.ToBytes())

	ns, _ := tv.Complete.EditFunc(tv.Complete.Context, tbes, tv.CursorPos.Ch, s, tv.Complete.Seed)
	fmt.Println(ns)
	tv.Buf.DeleteText(st, tv.CursorPos, true)
	tv.CursorPos = st
	tv.InsertAtCursor([]byte(ns))
	//tv.CursorForward(delta)
}

// SetCompleter sets completion functions so that completions will
// automatically be offered as the user types
func (tv *TextView) SetCompleter(data interface{}, matchFun complete.MatchFunc, editFun complete.EditFunc) {
	if matchFun == nil || editFun == nil {
		if tv.Complete != nil {
			tv.Complete.CompleteSig.Disconnect(tv.This)
		}
		tv.Complete.Destroy()
		tv.Complete = nil
		return
	}
	tv.Complete = &gi.Complete{}
	tv.Complete.InitName(tv.Complete, "tv-completion") // needed for standalone Ki's
	tv.Complete.Context = data
	tv.Complete.MatchFunc = matchFun
	tv.Complete.EditFunc = editFun
	// note: only need to connect once..
	tv.Complete.CompleteSig.ConnectOnly(tv.This, func(recv, send ki.Ki, sig int64, data interface{}) {
		tvf, _ := recv.Embed(KiT_TextView).(*TextView)
		if sig == int64(gi.CompleteSelect) {
			tvf.CompleteText(data.(string)) // always use data
		} else if sig == int64(gi.CompleteExtend) {
			tvf.CompleteExtend(data.(string)) // always use data
		}
	})
}

// CompleteExtend inserts the extended seed at the current cursor position
func (tv *TextView) CompleteExtend(s string) {
	if s != "" {
		win := tv.ParentWindow()
		win.ClosePopup(win.Popup)
		tv.InsertAtCursor([]byte(s))
		tv.OfferComplete(dontforce)
	}
}

///////////////////////////////////////////////////////////////////////////////
//    Scrolling

// ScrollInView tells any parent scroll layout to scroll to get given box
// (e.g., cursor BBox) in view -- returns true if scrolled
func (tv *TextView) ScrollInView(bbox image.Rectangle) bool {
	ly := tv.ParentScrollLayout()
	if ly == nil {
		return false
	}
	return ly.ScrollToBox(bbox)
}

// ScrollCursorInView tells any parent scroll layout to scroll to get cursor
// in view -- returns true if scrolled
func (tv *TextView) ScrollCursorInView() bool {
	curBBox := tv.CursorBBox(tv.CursorPos)
	return tv.ScrollInView(curBBox)
}

// AutoScroll tells any parent scroll layout to scroll to do its autoscroll
// based on given location -- for dragging
func (tv *TextView) AutoScroll(pos image.Point) bool {
	ly := tv.ParentScrollLayout()
	if ly == nil {
		return false
	}
	return ly.AutoScroll(pos)
}

// ScrollCursorToCenterIfHidden checks if the cursor is not visible, and if
// so, scrolls to the center, along both dimensions.
func (tv *TextView) ScrollCursorToCenterIfHidden() bool {
	curBBox := tv.CursorBBox(tv.CursorPos)
	did := false
	if curBBox.Max.Y < tv.VpBBox.Min.Y || curBBox.Min.Y > tv.VpBBox.Max.Y {
		did = tv.ScrollCursorToVertCenter()
	}
	if curBBox.Max.X < tv.VpBBox.Min.X || curBBox.Min.X > tv.VpBBox.Max.X {
		did = did || tv.ScrollCursorToHorizCenter()
	}
	return false
}

///////////////////////////////////////////////////////////////////////////////
//    Scrolling -- Vertical

// ScrollToTop tells any parent scroll layout to scroll to get given vertical
// coordinate at top of view to extent possible -- returns true if scrolled
func (tv *TextView) ScrollToTop(pos int) bool {
	ly := tv.ParentScrollLayout()
	if ly == nil {
		return false
	}
	return ly.ScrollDimToStart(gi.Y, pos)
}

// ScrollCursorToTop tells any parent scroll layout to scroll to get cursor
// at top of view to extent possible -- returns true if scrolled.
func (tv *TextView) ScrollCursorToTop() bool {
	curBBox := tv.CursorBBox(tv.CursorPos)
	return tv.ScrollToTop(curBBox.Min.Y)
}

// ScrollToBottom tells any parent scroll layout to scroll to get given
// vertical coordinate at bottom of view to extent possible -- returns true if
// scrolled
func (tv *TextView) ScrollToBottom(pos int) bool {
	ly := tv.ParentScrollLayout()
	if ly == nil {
		return false
	}
	return ly.ScrollDimToEnd(gi.Y, pos)
}

// ScrollCursorToBottom tells any parent scroll layout to scroll to get cursor
// at bottom of view to extent possible -- returns true if scrolled.
func (tv *TextView) ScrollCursorToBottom() bool {
	curBBox := tv.CursorBBox(tv.CursorPos)
	return tv.ScrollToBottom(curBBox.Max.Y)
}

// ScrollToVertCenter tells any parent scroll layout to scroll to get given
// vertical coordinate to center of view to extent possible -- returns true if
// scrolled
func (tv *TextView) ScrollToVertCenter(pos int) bool {
	ly := tv.ParentScrollLayout()
	if ly == nil {
		return false
	}
	return ly.ScrollDimToCenter(gi.Y, pos)
}

// ScrollCursorToVertCenter tells any parent scroll layout to scroll to get
// cursor at vert center of view to extent possible -- returns true if
// scrolled.
func (tv *TextView) ScrollCursorToVertCenter() bool {
	curBBox := tv.CursorBBox(tv.CursorPos)
	mid := (curBBox.Min.Y + curBBox.Max.Y) / 2
	return tv.ScrollToVertCenter(mid)
}

///////////////////////////////////////////////////////////////////////////////
//    Scrolling -- Horizontal

// ScrollToLeft tells any parent scroll layout to scroll to get given
// horizontal coordinate at left of view to extent possible -- returns true if
// scrolled
func (tv *TextView) ScrollToLeft(pos int) bool {
	ly := tv.ParentScrollLayout()
	if ly == nil {
		return false
	}
	return ly.ScrollDimToStart(gi.X, pos)
}

// ScrollCursorToLeft tells any parent scroll layout to scroll to get cursor
// at left of view to extent possible -- returns true if scrolled.
func (tv *TextView) ScrollCursorToLeft() bool {
	if tv.CursorPos.Ch == 0 {
		return tv.ScrollToLeft(tv.ObjBBox.Min.X - int(tv.Sty.BoxSpace()) - 2)
	}
	curBBox := tv.CursorBBox(tv.CursorPos)
	return tv.ScrollToLeft(curBBox.Min.X)
}

// ScrollToRight tells any parent scroll layout to scroll to get given
// horizontal coordinate at right of view to extent possible -- returns true
// if scrolled
func (tv *TextView) ScrollToRight(pos int) bool {
	ly := tv.ParentScrollLayout()
	if ly == nil {
		return false
	}
	return ly.ScrollDimToEnd(gi.X, pos)
}

// ScrollCursorToRight tells any parent scroll layout to scroll to get cursor
// at right of view to extent possible -- returns true if scrolled.
func (tv *TextView) ScrollCursorToRight() bool {
	curBBox := tv.CursorBBox(tv.CursorPos)
	return tv.ScrollToRight(curBBox.Max.X)
}

// ScrollToHorizCenter tells any parent scroll layout to scroll to get given
// horizontal coordinate to center of view to extent possible -- returns true if
// scrolled
func (tv *TextView) ScrollToHorizCenter(pos int) bool {
	ly := tv.ParentScrollLayout()
	if ly == nil {
		return false
	}
	return ly.ScrollDimToCenter(gi.X, pos)
}

// ScrollCursorToHorizCenter tells any parent scroll layout to scroll to get
// cursor at horiz center of view to extent possible -- returns true if
// scrolled.
func (tv *TextView) ScrollCursorToHorizCenter() bool {
	curBBox := tv.CursorBBox(tv.CursorPos)
	mid := (curBBox.Min.X + curBBox.Max.X) / 2
	return tv.ScrollToHorizCenter(mid)
}

///////////////////////////////////////////////////////////////////////////////
//    Rendering

// CharStartPos returns the starting (top left) render coords for the given
// position -- makes no attempt to rationalize that pos (i.e., if not in
// visible range, position will be out of range too)
func (tv *TextView) CharStartPos(pos TextPos) gi.Vec2D {
	spos := tv.RenderStartPos()
	spos.X += tv.LineNoOff
	if pos.Ln >= len(tv.Offs) {
		if len(tv.Offs) > 0 {
			pos.Ln = len(tv.Offs) - 1
		} else {
			return spos
		}
	} else {
		spos.Y += tv.Offs[pos.Ln] + gi.FixedToFloat32(tv.Sty.Font.Face.Metrics().Descent)
	}
	if len(tv.Renders[pos.Ln].Spans) > 0 {
		// note: Y from rune pos is baseline
		rrp, _, _, _ := tv.Renders[pos.Ln].RuneRelPos(pos.Ch)
		spos.X += rrp.X
		spos.Y += rrp.Y - tv.Renders[pos.Ln].Spans[0].RelPos.Y // relative
	}
	return spos
}

// CharEndPos returns the ending (bottom right) render coords for the given
// position -- makes no attempt to rationalize that pos (i.e., if not in
// visible range, position will be out of range too)
func (tv *TextView) CharEndPos(pos TextPos) gi.Vec2D {
	spos := tv.RenderStartPos()
	if pos.Ln >= tv.NLines {
		spos.Y += float32(tv.LinesSize.Y)
		spos.X += tv.LineNoOff
		return spos
	}
	spos.Y += tv.Offs[pos.Ln] + gi.FixedToFloat32(tv.Sty.Font.Face.Metrics().Descent)
	spos.X += tv.LineNoOff
	if len(tv.Renders[pos.Ln].Spans) > 0 {
		// note: Y from rune pos is baseline
		rrp, _, _, _ := tv.Renders[pos.Ln].RuneEndPos(pos.Ch)
		spos.X += rrp.X
		spos.Y += rrp.Y - tv.Renders[pos.Ln].Spans[0].RelPos.Y // relative
	}
	spos.Y += tv.LineHeight // end of that line
	return spos
}

// TextViewBlinker is the time.Ticker for blinking cursors for text fields,
// only one of which can be active at at a time
var TextViewBlinker *time.Ticker

// BlinkingTextView is the text field that is blinking
var BlinkingTextView *TextView

// TextViewSpriteName is the name of the window sprite used for the cursor
var TextViewSpriteName = "giv.TextView.Cursor"

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
		if win == nil || win.IsResizing() || win.IsClosed() {
			continue
		}
		tv.BlinkOn = !tv.BlinkOn
		tv.RenderCursor(tv.BlinkOn)
	}
}

// StartCursor starts the cursor blinking and renders it
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

// StopCursor stops the cursor from blinking
func (tv *TextView) StopCursor() {
	if BlinkingTextView == tv {
		BlinkingTextView = nil
	}
}

// CursorBBox returns a bounding-box for a cursor at given position
func (tv *TextView) CursorBBox(pos TextPos) image.Rectangle {
	st := &tv.Sty
	cpos := tv.CharStartPos(pos)
	cbmin := cpos.SubVal(st.Border.Width.Dots)
	cbmax := cpos.AddVal(st.Border.Width.Dots)
	cbmax.Y += tv.FontHeight
	curBBox := image.Rectangle{cbmin.ToPointFloor(), cbmax.ToPointCeil()}
	return curBBox
}

// RenderCursor renders the cursor on or off, as a sprite that is either on or off
func (tv *TextView) RenderCursor(on bool) {
	win := tv.Viewport.Win
	if win == nil {
		return
	}
	if tv.Renders == nil {
		return
	}
	if tv.PushBounds() {
		sp := tv.CursorSprite()
		if on {
			win.ActivateSprite(sp.Nm)
		} else {
			win.InactivateSprite(sp.Nm)
		}
		sp.Geom.Pos = tv.CharStartPos(tv.CursorPos).ToPointFloor()
		win.RenderOverlays() // needs an explicit call!
		tv.PopBounds()
		win.UpdateSig() // publish
	}
}

// CursorSprite returns the sprite Viewport2D that holds the cursor (which is
// only rendered once with a vertical bar, and just activated and inactivated
// depending on render status)
func (tv *TextView) CursorSprite() *gi.Viewport2D {
	win := tv.Viewport.Win
	if win == nil {
		return nil
	}
	sty := &tv.StateStyles[TextViewActive]
	spnm := fmt.Sprintf("%v-%v", TextViewSpriteName, tv.FontHeight)
	sp, ok := win.Sprites[spnm]
	if !ok {
		bbsz := image.Point{int(math32.Ceil(tv.CursorWidth.Dots)), int(math32.Ceil(tv.FontHeight))}
		if bbsz.X < 2 { // at least 2
			bbsz.X = 2
		}
		sp = win.AddSprite(spnm, bbsz, image.ZP)
		draw.Draw(sp.Pixels, sp.Pixels.Bounds(), &image.Uniform{sty.Font.Color}, image.ZP, draw.Src)
	}
	return sp
}

// RenderSelect renders the selection region as a selected background color
// -- always called within context of outer RenderLines or RenderAllLines
func (tv *TextView) RenderSelect() {
	if !tv.HasSelection() {
		return
	}
	tv.RenderRegionBox(tv.SelectReg, TextViewSel)
}

// RenderHighlights renders the highlight regions as a highlighted background
// color -- always called within context of outer RenderLines or
// RenderAllLines
func (tv *TextView) RenderHighlights(stln, edln int) {
	for _, reg := range tv.Highlights {
		if stln >= 0 && (reg.Start.Ln > edln || reg.End.Ln < stln) {
			continue
		}
		tv.RenderRegionBox(reg, TextViewHighlight)
	}
}

// RenderRegionBox renders a region in background color according to given state style
func (tv *TextView) RenderRegionBox(reg TextRegion, state TextViewStates) {
	st := reg.Start
	ed := reg.End
	spos := tv.CharStartPos(st)
	epos := tv.CharEndPos(ed)
	if int(math32.Ceil(epos.Y)) < tv.VpBBox.Min.Y || int(math32.Floor(spos.Y)) > tv.VpBBox.Max.Y {
		return
	}

	rs := &tv.Viewport.Render
	pc := &rs.Paint
	sty := &tv.StateStyles[state]
	spc := sty.BoxSpace()

	ed.Ch-- // end is exclusive
	rst := tv.RenderStartPos()

	// fmt.Printf("select: %v -- %v\n", st, ed)

	if st.Ln == ed.Ln {
		if wln := tv.WrappedLines(st.Ln); wln > 1 {
			stsi, _, stok := tv.WrappedLineNo(st)
			edsi, _, edok := tv.WrappedLineNo(ed)
			if !stok || !edok {
				return
			}
			if stsi == edsi {
				pc.FillBox(rs, spos, epos.Sub(spos), &sty.Font.BgColor)
			} else {
				if st.Ch > 0 {
					se := tv.CharEndPos(st)
					se.X = float32(tv.VpBBox.Max.X) - spc
					pc.FillBox(rs, spos, se.Sub(spos), &sty.Font.BgColor)
				}
				spos.Y += tv.LineHeight
				spos.X = rst.X + tv.LineNoOff
				nfull := edsi - stsi
				sed := spos
				if nfull >= 2 {
					beb := spos
					beb.Y += float32(nfull-1) * tv.LineHeight
					beb.X = float32(tv.VpBBox.Max.X) - spc
					pc.FillBox(rs, spos, beb.Sub(spos), &sty.Font.BgColor)
					sed.Y = beb.Y
				}
				pc.FillBox(rs, sed, epos.Sub(sed), &sty.Font.BgColor)
			}
		} else {
			pc.FillBox(rs, spos, epos.Sub(spos), &sty.Font.BgColor)
		}
	} else {
		if st.Ch > 0 {
			se := tv.CharEndPos(st)
			se.X = float32(tv.VpBBox.Max.X) - spc
			pc.FillBox(rs, spos, se.Sub(spos), &sty.Font.BgColor)
			spos.Y += tv.LineHeight
			spos.X = rst.X + tv.LineNoOff
		}
		beb := epos
		beb.Y -= tv.LineHeight
		beb.X = float32(tv.VpBBox.Max.X) - spc
		pc.FillBox(rs, spos, beb.Sub(spos), &sty.Font.BgColor)
		// now get anything on end
		if ed.Ch > 0 {
			beb.X = rst.X + tv.LineNoOff
			pc.FillBox(rs, beb, epos.Sub(beb), &sty.Font.BgColor)
		}
	}
}

// RenderStartPos is absolute rendering start position from our allocpos
func (tv *TextView) RenderStartPos() gi.Vec2D {
	st := &tv.Sty
	spc := st.BoxSpace()
	pos := tv.LayData.AllocPos.AddVal(spc)
	return pos
}

// VisSizes computes the visible size of view given current parameters
func (tv *TextView) VisSizes() {
	if tv.Sty.Font.Size.Val == 0 { // not yet styled
		tv.StyleTextView()
	}
	sty := &tv.Sty
	spc := sty.BoxSpace()
	sty.Font.OpenFont(&sty.UnContext)
	tv.FontHeight = sty.Font.Height
	tv.LineHeight = tv.FontHeight * sty.Text.EffLineHeight()
	sz := tv.VpBBox.Size()
	if sz == image.ZP {
		tv.VisSize.Y = 40
		tv.VisSize.X = 80
	} else {
		tv.VisSize.Y = int(math32.Floor(float32(sz.Y) / tv.LineHeight))
		tv.VisSize.X = int(math32.Floor(float32(sz.X) / sty.Font.Ch))
	}
	tv.LineNoDigs = gi.MaxInt(1+int(math32.Log10(float32(tv.NLines))), 3)
	if tv.Opts.LineNos {
		tv.LineNoOff = float32(tv.LineNoDigs+3)*sty.Font.Ch + spc // space for icon
	} else {
		tv.LineNoOff = 0
	}
	tv.RenderSize()
}

// RenderAllLines displays all the visible lines on the screen -- this is
// called outside of update process and has its own bounds check and updating
func (tv *TextView) RenderAllLines() {
	if tv.PushBounds() {
		vp := tv.Viewport
		updt := vp.Win.UpdateStart()
		tv.RenderAllLinesInBounds()
		tv.PopBounds()
		vp.Win.UploadVpRegion(vp, tv.VpBBox, tv.WinBBox)
		vp.Win.UpdateEnd(updt)
	}
}

// RenderAllLinesInBounds displays all the visible lines on the screen --
// after PushBounds has already been called
func (tv *TextView) RenderAllLinesInBounds() {
	// fmt.Printf("render all: %v\n", tv.Nm)
	rs := &tv.Viewport.Render
	pc := &rs.Paint
	sty := &tv.Sty
	tv.VisSizes()
	if tv.NLines == 0 {
		pos := tv.RenderStartPos()
		pos.X += tv.LineNoOff
		sz := tv.RenderSz
		pc.FillBox(rs, pos, sz, &sty.Font.BgColor)
	} else {
		tv.RenderStdBox(sty)
	}
	tv.RenderLineNosBoxAll()
	tv.RenderHighlights(-1, -1) // all
	tv.RenderSelect()
	pos := tv.RenderStartPos()
	for ln := 0; ln < tv.NLines; ln++ {
		lst := pos.Y + tv.Offs[ln]
		led := lst + math32.Max(tv.Renders[ln].Size.Y, tv.LineHeight)
		if int(math32.Ceil(led)) < tv.VpBBox.Min.Y {
			continue
		}
		if int(math32.Floor(lst)) > tv.VpBBox.Max.Y {
			continue
		}
		lp := pos
		lp.Y = lst
		lp.X += tv.LineNoOff
		tv.Renders[ln].Render(rs, lp) // not top pos -- already has baseline offset
		tv.RenderLineNo(ln)
	}
}

// RenderLineNosBoxAll renders the background for the line numbers in a darker shade
func (tv *TextView) RenderLineNosBoxAll() {
	if !tv.Opts.LineNos {
		return
	}
	rs := &tv.Viewport.Render
	pc := &rs.Paint
	sty := &tv.Sty
	spc := sty.BoxSpace()
	clr := sty.Font.BgColor.Color.Highlight(10)
	spos := gi.NewVec2DFmPoint(tv.VpBBox.Min).AddVal(spc)
	epos := gi.NewVec2DFmPoint(tv.VpBBox.Max)
	epos.X = spos.X + tv.LineNoOff - spc
	pc.FillBoxColor(rs, spos, epos.Sub(spos), clr)
}

// RenderLineNosBox renders the background for the line numbers in given range, in a darker shade
func (tv *TextView) RenderLineNosBox(st, ed int) {
	if !tv.Opts.LineNos {
		return
	}
	rs := &tv.Viewport.Render
	pc := &rs.Paint
	sty := &tv.Sty
	spc := sty.BoxSpace()
	clr := sty.Font.BgColor.Color.Highlight(10)
	spos := tv.CharStartPos(TextPos{Ln: st})
	spos.X = float32(tv.VpBBox.Min.X) + spc
	epos := tv.CharEndPos(TextPos{Ln: ed + 1})
	epos.Y -= tv.LineHeight
	epos.X = spos.X + tv.LineNoOff - spc
	// fmt.Printf("line box: st %v ed: %v spos %v  epos %v\n", st, ed, spos, epos)
	pc.FillBoxColor(rs, spos, epos.Sub(spos), clr)
}

// RenderLineNo renders given line number -- called within context of other render
func (tv *TextView) RenderLineNo(ln int) {
	if !tv.Opts.LineNos {
		return
	}
	vp := tv.Viewport
	sty := &tv.Sty
	fst := sty.Font
	fst.BgColor.SetColor(nil)
	rs := &vp.Render
	lfmt := fmt.Sprintf("%v", tv.LineNoDigs)
	lfmt = "%0" + lfmt + "d"
	lnstr := fmt.Sprintf(lfmt, ln+1)
	tv.LineNoRender.SetString(lnstr, &fst, &sty.UnContext, &sty.Text, true, 0, 0)
	pos := tv.RenderStartPos()
	lst := tv.CharStartPos(TextPos{Ln: ln}).Y // note: charstart pos includes descent
	pos.Y = lst + gi.FixedToFloat32(sty.Font.Face.Metrics().Ascent) - +gi.FixedToFloat32(sty.Font.Face.Metrics().Descent)
	tv.LineNoRender.Render(rs, pos)
	// if ic, ok := tv.LineIcons[ln]; ok {
	// 	// todo: render icon!
	// }
}

// RenderLines displays a specific range of lines on the screen, also painting
// selection.  end is *inclusive* line.  returns false if nothing visible.
func (tv *TextView) RenderLines(st, ed int) bool {
	if tv.PushBounds() {
		vp := tv.Viewport
		updt := vp.Win.UpdateStart()
		sty := &tv.Sty
		rs := &vp.Render
		pc := &rs.Paint
		pos := tv.RenderStartPos()
		var boxMin, boxMax gi.Vec2D
		// first get the box to fill
		visSt := -1
		visEd := -1
		for ln := st; ln <= ed; ln++ {
			lst := tv.CharStartPos(TextPos{Ln: ln}).Y // note: charstart pos includes descent
			led := lst + math32.Max(tv.Renders[ln].Size.Y, tv.LineHeight)
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
		} else {
			boxMax.X = float32(tv.VpBBox.Max.X) // go all the way
			pc.FillBox(rs, boxMin, boxMax.Sub(boxMin), &sty.Font.BgColor)
			// fmt.Printf("lns: st: %v ed: %v vis st: %v ed %v box: min %v max: %v\n", st, ed, visSt, visEd, boxMin, boxMax)

			tv.RenderHighlights(visSt, visEd)
			tv.RenderSelect()
			tv.RenderLineNosBox(visSt, visEd)

			for ln := visSt; ln <= visEd; ln++ {
				lst := pos.Y + tv.Offs[ln]
				lp := pos
				lp.Y = lst
				lp.X += tv.LineNoOff
				tv.Renders[ln].Render(rs, lp) // not top pos -- already has baseline offset
				tv.RenderLineNo(ln)
			}

			tBBox := image.Rectangle{boxMin.ToPointFloor(), boxMax.ToPointCeil()}
			vprel := tBBox.Min.Sub(tv.VpBBox.Min)
			tWinBBox := tv.WinBBox.Add(vprel)
			vp.Win.UploadVpRegion(vp, tBBox, tWinBBox)
		}
		tv.PopBounds()
		vp.Win.UpdateEnd(updt)
	}
	return true
}

///////////////////////////////////////////////////////////////////////////////
//    View-specific helpers

// FirstVisibleLine finds the first visible line, starting at given line
// (typically cursor -- if zero, a visible line is first found) -- returns
// stln if nothing found above it.
func (tv *TextView) FirstVisibleLine(stln int) int {
	if stln == 0 {
		perln := float32(tv.LinesSize.Y) / float32(tv.NLines)
		stln = int(float32(tv.VpBBox.Min.Y-tv.ObjBBox.Min.Y)/perln) - 1
		if stln < 0 {
			stln = 0
		}
		for ln := stln; ln < tv.NLines; ln++ {
			cpos := tv.CharStartPos(TextPos{Ln: ln})
			if int(math32.Floor(cpos.Y)) >= tv.VpBBox.Min.Y { // top definitely on screen
				stln = ln
				break
			}
		}
	}
	lastln := stln
	for ln := stln - 1; ln >= 0; ln-- {
		cpos := tv.CharStartPos(TextPos{Ln: ln})
		if int(math32.Ceil(cpos.Y)) < tv.VpBBox.Min.Y { // top just offscreen
			break
		}
		lastln = ln
	}
	return lastln
}

// LastVisibleLine finds the last visible line, starting at given line
// (typically cursor) -- returns stln if nothing found beyond it.
func (tv *TextView) LastVisibleLine(stln int) int {
	lastln := stln
	for ln := stln + 1; ln < tv.NLines; ln++ {
		pos := TextPos{Ln: ln}
		cpos := tv.CharStartPos(pos)
		if int(math32.Floor(cpos.Y)) > tv.VpBBox.Max.Y { // just offscreen
			break
		}
		lastln = ln
	}
	return lastln
}

// PixelToCursor finds the cursor position that corresponds to the given pixel
// location (e.g., from mouse click) which has had WinBBox.Min subtracted from
// it (i.e, relative to upper left of text area)
func (tv *TextView) PixelToCursor(pt image.Point) TextPos {
	if tv.NLines == 0 {
		return TextPosZero
	}
	sty := &tv.Sty
	yoff := float32(tv.WinBBox.Min.Y)
	stln := tv.FirstVisibleLine(0)
	cln := stln
	fls := tv.CharStartPos(TextPos{Ln: stln}).Y - yoff
	if pt.Y < int(math32.Floor(fls)) {
		cln = stln
	} else if pt.Y > tv.WinBBox.Max.Y {
		cln = tv.NLines - 1
	} else {
		got := false
		for ln := stln; ln < tv.NLines; ln++ {
			ls := tv.CharStartPos(TextPos{Ln: ln}).Y - yoff
			es := ls
			es += math32.Max(tv.Renders[ln].Size.Y, tv.LineHeight)
			if pt.Y >= int(math32.Floor(ls)) && pt.Y < int(math32.Ceil(es)) {
				got = true
				cln = ln
				break
			}
		}
		if !got {
			cln = tv.NLines - 1
		}
	}
	// fmt.Printf("cln: %v  pt: %v\n", cln, pt)
	lnsz := len(tv.Buf.Lines[cln])
	if lnsz == 0 {
		return TextPos{Ln: cln, Ch: 0}
	}
	xoff := float32(tv.WinBBox.Min.X)
	scrl := tv.WinBBox.Min.X - tv.ObjBBox.Min.X
	sc := int(float32(pt.X+scrl) / sty.Font.Ch)
	sc -= sc / 4
	sc = gi.MaxInt(0, sc)
	cch := sc

	si := 0
	spoff := 0
	nspan := len(tv.Renders[cln].Spans)
	lstY := tv.CharStartPos(TextPos{Ln: cln}).Y - yoff
	if nspan > 1 {
		si = int((float32(pt.Y) - lstY) / tv.LineHeight)
		for i := 0; i < si; i++ {
			spoff += len(tv.Renders[cln].Spans[i].Text) + 1
		}
		// fmt.Printf("si: %v  spoff: %v\n", si, spoff)
	}
	cch += spoff

	tooBig := false
	if cch < lnsz {
		for c := cch; c < lnsz; c++ {
			rsp := math32.Floor(tv.CharStartPos(TextPos{Ln: cln, Ch: c}).X - xoff)
			rep := math32.Ceil(tv.CharEndPos(TextPos{Ln: cln, Ch: c}).X - xoff)
			// fmt.Printf("trying c: %v for pt: %v xoff: %v rsp: %v, rep: %v\n", c, pt, xoff, rsp, rep)
			if pt.X >= int(rsp) && pt.X < int(rep) {
				cch = c
				// fmt.Printf("got cch: %v for pt: %v rsp: %v, rep: %v\n", cch, pt, rsp, rep)
				break
			} else if int(rep) > pt.X {
				cch = c
				tooBig = true
				break
			}
		}
	} else {
		tooBig = true
		cch = lnsz - 1
	}
	if tooBig {
		for c := cch; c >= 0; c-- {
			rsp := math32.Floor(tv.CharStartPos(TextPos{Ln: cln, Ch: c}).X - xoff)
			rep := math32.Ceil(tv.CharEndPos(TextPos{Ln: cln, Ch: c}).X - xoff)
			// fmt.Printf("too big: trying c: %v for pt: %v rsp: %v, rep: %v\n", c, pt, rsp, rep)
			if pt.X >= int(rsp) && pt.X < int(rep) {
				cch = c
				// fmt.Printf("got cch: %v for pt: %v rsp: %v, rep: %v\n", cch, pt, rsp, rep)
				break
			}
		}
	}
	return TextPos{Ln: cln, Ch: cch}
}

// SetCursorFromMouse sets cursor position from mouse mouse action -- handles
// the selection updating etc.
func (tv *TextView) SetCursorFromMouse(pt image.Point, newPos TextPos, selMode mouse.SelectModes) {
	oldPos := tv.CursorPos
	if newPos == oldPos {
		return
	}
	tv.SetCursor(newPos)
	if tv.SelectMode || selMode != mouse.NoSelectMode {
		if !tv.SelectMode && selMode != mouse.NoSelectMode {
			tv.SelectReg.Start = oldPos
			tv.SelectMode = true
		}
		if !tv.IsDragging() && selMode == mouse.NoSelectMode {
			tv.SelectReset()
		} else if tv.SelectReg.Start.IsLess(tv.CursorPos) {
			tv.SelectReg.End = tv.CursorPos
		} else {
			tv.SelectReg.Start = tv.CursorPos
		}
		if tv.IsDragging() {
			tv.AutoScroll(pt.Add(tv.WinBBox.Min))
		} else {
			tv.ScrollCursorToCenterIfHidden()
		}
		tv.RenderSelectLines()
	} else if tv.HasSelection() {
		tv.SelectReset()
	}
	tv.RenderCursor(true)
}

///////////////////////////////////////////////////////////////////////////////
//    KeyInput handling

var defaultIndentStrings = []string{"{"}
var defaultUnindentStrings = []string{"}"}

// ShiftSelect activates selection mode if shift key is also pressed -- called
// along with cursor motion keys
func (tv *TextView) ShiftSelect(kt *key.ChordEvent) {
	hasShift := kt.HasAnyModifier(key.Shift)
	if hasShift {
		if !tv.SelectMode {
			tv.SelectMode = true
			tv.SelectReg.Start = tv.CursorPos
			tv.SelectReg.End = tv.CursorPos
		}
	}
}

// KeyInput handles keyboard input into the text field and from the completion menu
func (tv *TextView) KeyInput(kt *key.ChordEvent) {
	kf := gi.KeyFun(kt.Chord())
	win := tv.ParentWindow()

	if gi.PopupIsCompleter(win.Popup) {
		tv.Complete.KeyInput(kf)
	}

	if kt.IsProcessed() {
		return
	}

	// first all the keys that work for both inactive and active
	switch kf {
	case gi.KeyFunMoveRight:
		tv.ISearchCancel() // note: may need to generalize to cancel more stuff
		kt.SetProcessed()
		tv.ShiftSelect(kt)
		tv.CursorForward(1)
		tv.OfferComplete(dontforce)
	case gi.KeyFunMoveLeft:
		tv.ISearchCancel()
		kt.SetProcessed()
		tv.ShiftSelect(kt)
		tv.CursorBackward(1)
		tv.OfferComplete(dontforce)
	case gi.KeyFunMoveUp:
		tv.ISearchCancel()
		kt.SetProcessed()
		tv.ShiftSelect(kt)
		tv.CursorUp(1)
	case gi.KeyFunMoveDown:
		tv.ISearchCancel()
		kt.SetProcessed()
		tv.ShiftSelect(kt)
		tv.CursorDown(1)
	case gi.KeyFunPageUp:
		tv.ISearchCancel()
		kt.SetProcessed()
		tv.ShiftSelect(kt)
		tv.CursorPageUp(1)
	case gi.KeyFunPageDown:
		tv.ISearchCancel()
		kt.SetProcessed()
		tv.ShiftSelect(kt)
		tv.CursorPageDown(1)
	case gi.KeyFunHome:
		tv.ISearchCancel()
		kt.SetProcessed()
		tv.ShiftSelect(kt)
		tv.CursorStartLine()
	case gi.KeyFunEnd:
		tv.ISearchCancel()
		kt.SetProcessed()
		tv.ShiftSelect(kt)
		tv.CursorEndLine()
	case gi.KeyFunDocHome:
		tv.ISearchCancel()
		kt.SetProcessed()
		tv.ShiftSelect(kt)
		tv.CursorStartDoc()
	case gi.KeyFunDocEnd:
		tv.ISearchCancel()
		kt.SetProcessed()
		tv.ShiftSelect(kt)
		tv.CursorEndDoc()
	case gi.KeyFunSelectMode:
		tv.ISearchCancel()
		kt.SetProcessed()
		tv.SelectModeToggle()
	case gi.KeyFunCancelSelect:
		kt.SetProcessed()
		tv.EscPressed() // generic cancel
	case gi.KeyFunSelectAll:
		tv.ISearchCancel()
		kt.SetProcessed()
		tv.SelectAll()
	case gi.KeyFunCopy:
		tv.ISearchCancel()
		kt.SetProcessed()
		tv.Copy(true) // reset
	case gi.KeyFunSearch:
		kt.SetProcessed()
		tv.ISearch()
	case gi.KeyFunAbort:
		kt.SetProcessed()
		tv.EscPressed()
	case gi.KeyFunJump:
		kt.SetProcessed()
		tv.JumpToLinePrompt()
	}
	if tv.IsInactive() || kt.IsProcessed() {
		return
	}
	switch kf {
	case gi.KeyFunAccept: // ctrl+enter
		tv.ISearchCancel()
		// tv.EditDone()
		kt.SetProcessed()
		tv.FocusNext()
	case gi.KeyFunBackspace:
		if tv.ISearchMode {
			tv.ISearchBackspace()
		} else {
			kt.SetProcessed()
			tv.CursorBackspace(1)
			tv.OfferComplete(dontforce)
		}
	case gi.KeyFunKill:
		tv.ISearchCancel()
		kt.SetProcessed()
		tv.CursorKill()
	case gi.KeyFunDelete:
		tv.ISearchCancel()
		kt.SetProcessed()
		tv.CursorDelete(1)
	case gi.KeyFunCut:
		tv.ISearchCancel()
		kt.SetProcessed()
		tv.Cut()
	case gi.KeyFunPaste:
		tv.ISearchCancel()
		kt.SetProcessed()
		tv.Paste()
	case gi.KeyFunUndo:
		tv.ISearchCancel()
		kt.SetProcessed()
		tv.Undo()
	case gi.KeyFunRedo:
		tv.ISearchCancel()
		kt.SetProcessed()
		tv.Redo()
	case gi.KeyFunComplete:
		tv.ISearchCancel()
		kt.SetProcessed()
		tv.OfferComplete(force)
	case gi.KeyFunRecenter:
		kt.SetProcessed()
		tv.CursorRecenter()
	case gi.KeyFunSelectItem: // enter
		tv.ISearchCancel()
		if !kt.HasAnyModifier(key.Control, key.Meta) {
			kt.SetProcessed()
			tv.InsertAtCursor([]byte("\n"))
			if tv.Opts.AutoIndent {
				tv.Buf.AutoIndent(tv.CursorPos.Ln, tv.Opts.SpaceIndent, tv.Sty.Text.TabSize, defaultIndentStrings, defaultUnindentStrings)
				tv.CursorEndLine()
			}
		}
	case gi.KeyFunFocusNext: // tab
		tv.ISearchCancel()
		if !kt.HasAnyModifier(key.Control, key.Meta) {
			kt.SetProcessed()
			if tv.CursorPos.Ch == 0 && tv.Opts.AutoIndent { // todo: only at 1st pos now
				_, _, cpos := tv.Buf.AutoIndent(tv.CursorPos.Ln, tv.Opts.SpaceIndent, tv.Sty.Text.TabSize, defaultIndentStrings, defaultUnindentStrings)
				tv.CursorPos.Ch = cpos
				tv.RenderCursor(true)
			} else {
				if tv.Opts.SpaceIndent {
					tv.InsertAtCursor(IndentBytes(1, tv.Sty.Text.TabSize, true))
				} else {
					tv.InsertAtCursor([]byte("\t"))
				}
			}
		}
	case gi.KeyFunNil:
		if unicode.IsPrint(kt.Rune) {
			if !kt.HasAnyModifier(key.Control, key.Meta) {
				kt.SetProcessed()
				if tv.ISearchMode { // todo: need this in inactive mode
					tv.ISearchKeyInput(kt.Rune)
				} else {
					tv.InsertAtCursor([]byte(string(kt.Rune)))
				}
				tv.OfferComplete(dontforce)
			}
		}
	}
}

// OpenLink opens given link, either by sending LinkSig signal if there are
// receivers, or by calling the TextLinkHandler if non-nil, or URLHandler if
// non-nil (which by default opens user's default browser via
// oswin/App.OpenURL())
func (tv *TextView) OpenLink(tl *gi.TextLink) {
	// fmt.Printf("opening link: %v\n", tl.URL)
	if len(tv.LinkSig.Cons) == 0 {
		if gi.TextLinkHandler != nil {
			if gi.TextLinkHandler(*tl) {
				return
			}
			if gi.URLHandler != nil {
				gi.URLHandler(tl.URL)
			}
		}
		return
	}
	tv.LinkSig.Emit(tv.This, 0, tl.URL) // todo: could potentially signal different target=_blank kinds of options here with the sig
}

// MouseEvent handles the mouse.Event
func (tv *TextView) MouseEvent(me *mouse.Event) {
	if !tv.IsInactive() && !tv.HasFocus() {
		tv.GrabFocus()
	}
	me.SetProcessed()
	pt := tv.PointToRelPos(me.Pos())
	newPos := tv.PixelToCursor(pt)
	switch me.Button {
	case mouse.Left:
		if me.Action == mouse.Press {
			me.SetProcessed()
			if newPos.Ln < len(tv.Renders) && len(tv.Renders[newPos.Ln].Links) > 0 {
				lpos := tv.CharStartPos(TextPos{Ln: newPos.Ln})
				rend := &tv.Renders[newPos.Ln]
				for ti, _ := range rend.Links {
					tl := &rend.Links[ti]
					tlb := tl.Bounds(rend, lpos)
					if me.Where.In(tlb) {
						tv.OpenLink(tl)
						return
					}
				}
			}
			tv.SetCursorFromMouse(pt, newPos, me.SelectMode())
		} else if me.Action == mouse.DoubleClick {
			me.SetProcessed()
			// if tv.HasSelection() {
			// 	if tv.SelectReg.Start == TextPosZero && tv.SelectReg.End == tv.Buf.EndPos() {
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
			tv.SetCursorFromMouse(pt, newPos, me.SelectMode())
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
		newPos := txf.PixelToCursor(pt)
		txf.SetCursorFromMouse(pt, newPos, mouse.NoSelectMode)
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

func (tv *TextView) StyleTextView() {
	tv.HiInit()
	tv.Style2DWidget()
	pst := &(tv.Par.(gi.Node2D).AsWidget().Sty)
	for i := 0; i < int(TextViewStatesN); i++ {
		tv.StateStyles[i].CopyFrom(&tv.Sty)
		tv.StateStyles[i].SetStyleProps(pst, tv.StyleProps(TextViewSelectors[i]))
		tv.StateStyles[i].StyleCSS(tv.This.(gi.Node2D), tv.CSSAgg, TextViewSelectors[i])
		tv.StateStyles[i].CopyUnitContext(&tv.Sty.UnContext)
	}
	tv.CursorWidth.SetFmInheritProp("cursor-width", tv.This, true, true) // inherit and get type defaults
	tv.CursorWidth.ToDots(&tv.Sty.UnContext)
}

func (tv *TextView) Style2D() {
	tv.SetCanFocusIfActive()
	tv.StyleTextView()
	tv.LayData.SetFromStyle(&tv.Sty.Layout) // also does reset
}

func (tv *TextView) Size2D(iter int) {
	if iter > 0 {
		return
	} else {
		tv.InitLayout2D()
		tv.LayoutAllLines(true) // already sets the size
	}
}

func (tv *TextView) Layout2D(parBBox image.Rectangle, iter int) bool {
	tv.Layout2DBase(parBBox, true, iter) // init style
	for i := 0; i < int(TextViewStatesN); i++ {
		tv.StateStyles[i].CopyUnitContext(&tv.Sty.UnContext)
	}
	tv.Layout2DChildren(iter)
	redo := tv.LayoutAllLines(true) // is our size now different?  if so iterate..
	return redo
}

func (tv *TextView) Render2D() {
	// fmt.Printf("textview render: %v\n", tv.Nm)
	if tv.FullReRenderIfNeeded() {
		return
	}
	tv.VisSizes()
	if tv.NLines == 0 {
		sz := tv.RenderSz.ToPointCeil()
		tv.VpBBox.Max = tv.VpBBox.Min.Add(sz)
		tv.WinBBox.Max = tv.WinBBox.Min.Add(sz)
	}
	if tv.PushBounds() {
		tv.TextViewEvents()
		if tv.IsInactive() {
			if tv.IsSelected() {
				tv.Sty = tv.StateStyles[TextViewSel]
			} else {
				tv.Sty = tv.StateStyles[TextViewInactive]
			}
		} else if tv.NLines == 0 {
			tv.Sty = tv.StateStyles[TextViewInactive]
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
		tv.RenderAllLinesInBounds()
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
		// tv.EditDone()
		tv.UpdateSig()
	case gi.FocusGot:
		tv.FocusActive = true
		tv.EmitFocusedSignal()
		tv.UpdateSig()
	case gi.FocusInactive:
		tv.FocusActive = false
		// tv.EditDone()
		tv.UpdateSig()
	case gi.FocusActive:
		tv.FocusActive = true
		// tv.UpdateSig()
		// todo: see about cursor
	}
}
