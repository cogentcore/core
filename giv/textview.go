// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package giv

import (
	"bytes"
	"fmt"
	"image"
	"image/draw"
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

// TextViewOpts contains options for TextView editing
type TextViewOpts struct {
	AutoIndent bool `desc:"auto-indent on newline and decrease indent on } (todo: generalize)"`
}

// TextView is a widget for editing multiple lines of text (as compared to
// TextField for a single line).  The underlying data model is just plain
// simple lines (ended by \n) with any number of characters per line.  These
// lines are displayed using wrap-around text into the editor.  Currently only
// works on in-memory strings.
type TextView struct {
	gi.WidgetBase
	Buf           *TextBuf                  `json:"-" xml:"-" desc:"the text buffer that we're editing"`
	Placeholder   string                    `json:"-" xml:"placeholder" desc:"text that is displayed when the field is empty, in a lower-contrast manner"`
	Opts          TextViewOpts              `desc:"options for how text editing / viewing works"`
	CursorWidth   units.Value               `xml:"cursor-width" desc:"width of cursor -- set from cursor-width property (inherited)"`
	HiStyle       string                    `desc:"syntax highlighting style"`
	HiCSS         gi.StyleSheet             `json:"-" xml:"-" desc:"CSS StyleSheet for given highlighting style"`
	Edited        bool                      `json:"-" xml:"-" desc:"true if the text has been edited relative to the original"`
	LineNos       bool                      `desc:"show line numbers at left end of editor"`
	LineIcons     map[int]gi.IconName       `desc:"icons for each line -- use SetLineIcon and DeleteLineIcon"`
	FocusActive   bool                      `json:"-" xml:"-" desc:"true if the keyboard focus is active or not -- when we lose active focus we apply changes"`
	NLines        int                       `json:"-" xml:"-" desc:"number of lines in the view -- sync'd with the Buf after edits, but always reflects storage size of Renders etc"`
	Markup        [][]byte                  `json:"-" xml:"-" desc:"marked-up version of the edit text lines, after being run through the syntax highlighting process -- this is what is actually rendered"`
	Renders       []gi.TextRender           `json:"-" xml:"-" desc:"renders of the text lines, with one render per line (each line could visibly wrap-around, so these are logical lines, not display lines)"`
	Offs          []float32                 `json:"-" xml:"-" desc:"starting offsets for top of each line"`
	LineNoDigs    int                       `json:"-" xml:"-" number of line number digits needed"`
	LineNoOff     float32                   `json:"-" xml:"-" desc:"horizontal offset for start of text after line numbers"`
	LineNoRender  gi.TextRender             `json:"-" xml:"-" desc:"render for line numbers"`
	LinesSize     image.Point               `json:"-" xml:"-" desc:"total size of all lines as rendered"`
	RenderSz      gi.Vec2D                  `json:"-" xml:"-" desc:"size params to use in render call"`
	CursorPos     TextPos                   `json:"-" xml:"-" desc:"current cursor position"`
	CursorCol     int                       `json:"-" xml:"-" desc:"desired cursor column -- where the cursor was last when moved using left / right arrows -- used when doing up / down to not always go to short line columns"`
	SelectReg     TextRegion                `xml:"-" desc:"current selection region"`
	PrevSelectReg TextRegion                `xml:"-" desc:"previous selection region, that was actually rendered -- needed to update render"`
	SelectMode    bool                      `xml:"-" desc:"if true, select text as cursor moves"`
	TextViewSig   ki.Signal                 `json:"-" xml:"-" view:"-" desc:"signal for text viewt -- see TextViewSignals for the types"`
	StateStyles   [TextViewStatesN]gi.Style `json:"-" xml:"-" desc:"normal style and focus style"`
	FontHeight    float32                   `json:"-" xml:"-" desc:"font height, cached during styling"`
	LineHeight    float32                   `json:"-" xml:"-" desc:"line height, cached during styling"`
	VisSize       image.Point               `json:"-" xml:"-" desc:"height in lines and width in chars of the visible area"`
	BlinkOn       bool                      `json:"-" xml:"-" oscillates between on and off for blinking"`
	Completion    *gi.Complete              `json:"-" xml:"-" desc:"functions and data for textfield completion"`
	// chroma highlighting
	lastHiLang   string
	lastHiStyle  string
	lexer        chroma.Lexer
	formatter    *html.Formatter
	style        *chroma.Style
	reLayout     bool
	lastRecenter int
}

var KiT_TextView = kit.Types.AddType(&TextView{}, TextViewProps)

var TextViewProps = ki.Props{
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
	"background-color": &gi.Prefs.Colors.Control,
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
	tv.LayoutAllLines(false)
	// todo: signal buffer?
	tv.SelectReset()
}

///////////////////////////////////////////////////////////////////////////////
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
		tv.LayoutAllLines(false)
		tv.SetFullReRender()
		tv.UpdateSig()
	case TextBufInsert:
		tbe := data.(*TextBufEdit)
		tv.Edited = tv.Buf.Edited
		// fmt.Printf("tv %v got %v\n", tv.Nm, tbe.Reg.Start)
		if tbe.Reg.Start.Ln != tbe.Reg.End.Ln {
			tv.LayoutAllLines(false)
			tv.RenderAllLines()
		} else {
			rerend := tv.LayoutLines(tbe.Reg.Start.Ln, tbe.Reg.End.Ln)
			if rerend {
				tv.RenderAllLines()
			} else {
				tv.RenderLines(tbe.Reg.Start.Ln, tbe.Reg.End.Ln)
			}
		}
	case TextBufDelete:
		tbe := data.(*TextBufEdit)
		tv.Edited = tv.Buf.Edited
		if tbe.Reg.Start.Ln != tbe.Reg.End.Ln {
			tv.LayoutAllLines(false)
			tv.RenderAllLines()
		} else {
			rerend := tv.LayoutLines(tbe.Reg.Start.Ln, tbe.Reg.End.Ln)
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

	tv.lastHiLang = tv.Buf.HiLang
	tv.lastHiStyle = tv.HiStyle
}

// RenderSize is the size we should pass to text rendering, based on alloc
func (tv *TextView) RenderSize() gi.Vec2D {
	spc := tv.Sty.BoxSpace()
	pari, _ := gi.KiToNode2D(tv.Par)
	parw := pari.AsLayout2D()
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

	tv.HiInit()

	tv.NLines = tv.Buf.NLines
	nln := tv.NLines
	if cap(tv.Markup) >= nln {
		tv.Markup = tv.Markup[:nln]
	} else {
		tv.Markup = make([][]byte, nln)
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

	if tv.HasHi() {
		var htmlBuf bytes.Buffer
		iterator, err := tv.lexer.Tokenise(nil, string(tv.Buf.Txt)) // todo: unfortunate conversion here..
		err = tv.formatter.Format(&htmlBuf, tv.style, iterator)
		if err != nil {
			log.Println(err)
			return false
		}
		mtlns := bytes.Split(htmlBuf.Bytes(), []byte("\n"))

		maxln := len(mtlns) - 1
		for ln := 0; ln < maxln; ln++ {
			mt := mtlns[ln]
			mt = bytes.TrimPrefix(mt, []byte(`</span>`)) // leftovers
			tv.Markup[ln] = mt
		}
	} else {
		for ln := 0; ln < nln; ln++ {
			tv.Markup[ln] = []byte(string(tv.Buf.Lines[ln]))
		}
	}

	tv.VisSizes()
	sz := tv.RenderSize()
	// fmt.Printf("rendersize: %v\n", sz)
	sty := &tv.Sty
	fst := sty.Font
	fst.BgColor.SetColor(nil)
	off := float32(0)
	mxwd := float32(0)
	for ln := 0; ln < nln; ln++ {
		tv.Renders[ln].SetHTMLPre(tv.Markup[ln], &fst, &sty.UnContext, tv.CSS)
		tv.Renders[ln].LayoutStdLR(&sty.Text, &sty.Font, &sty.UnContext, sz)
		tv.Offs[ln] = off
		lsz := gi.Max32(tv.Renders[ln].Size.Y, tv.LineHeight)
		off += lsz
		mxwd = gi.Max32(mxwd, tv.Renders[ln].Size.X)
	}
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
	tv.SetFullReRender()
	return true
}

// LayoutLines generates render of given range of lines (including
// highlighting). end is *inclusive* line.  if highlighter generates an error
// on a line, or word-wrap causes lines to increase in number of spans, then
// calls LayoutAllLines to do a full-reparse, and returns true to indicate
// need for a full re-render -- otherwise returns false and just these lines
// need to be re-rendered.
func (tv *TextView) LayoutLines(st, ed int) bool {
	sty := &tv.Sty
	fst := sty.Font
	fst.BgColor.SetColor(nil)
	mxwd := float32(tv.LinesSize.X)
	for ln := st; ln <= ed; ln++ {
		if tv.HasHi() {
			var htmlBuf bytes.Buffer
			iterator, err := tv.lexer.Tokenise(nil, string(tv.Buf.Lines[ln]))
			err = tv.formatter.Format(&htmlBuf, tv.style, iterator)
			if err != nil {
				log.Println(err)
				tv.Buf.LinesToBytes() // need to update buffer -- todo: redundant across views
				tv.LayoutAllLines(false)
				return true
			}
			tv.Markup[ln] = htmlBuf.Bytes()
		} else {
			tv.Markup[ln] = []byte(string(tv.Buf.Lines[ln]))
		}
		curspans := len(tv.Renders[ln].Spans)
		tv.Renders[ln].SetHTMLPre(tv.Markup[ln], &fst, &sty.UnContext, tv.CSS)
		tv.Renders[ln].LayoutStdLR(&sty.Text, &sty.Font, &sty.UnContext, tv.RenderSz)
		nwspans := len(tv.Renders[ln].Spans)
		if nwspans != curspans && (nwspans > 1 || curspans > 1) {
			tv.Buf.LinesToBytes() // need to update buffer -- todo: redundant across views
			tv.LayoutAllLines(false)
			return true
		}
		mxwd = gi.Max32(mxwd, tv.Renders[ln].Size.X)
	}
	nwSz := gi.Vec2D{mxwd, 0}.ToPointCeil()
	nwSz.Y = tv.LinesSize.Y
	tv.ResizeIfNeeded(nwSz)
	return false
}

///////////////////////////////////////////////////////////////////////////////
//  Cursor Navigation

// SetCursor sets a new cursor position, enforcing it in range
func (tv *TextView) SetCursor(pos TextPos) {
	if tv.NLines == 0 {
		tv.CursorPos = TextPosZero
		return
	}
	if pos.Ln >= len(tv.Buf.Lines) {
		pos.Ln = len(tv.Buf.Lines) - 1
	}
	llen := len(tv.Buf.Lines[pos.Ln])
	if pos.Ch >= llen {
		pos.Ch = llen
	}
	if pos.Ch < 0 {
		pos.Ch = 0
	}
	tv.CursorPos = pos
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
	tv.ScrollCursorToCenterIfHidden()
	tv.RenderCursor(true)
	tv.CursorSelect(org)
}

// WrappedLines returns the number of wrapped lines (spans) for given line number
func (tv *TextView) WrappedLines(ln int) int {
	return len(tv.Renders[ln].Spans)
}

// WrappedLineNo returns the wrapped line number (span index) and rune index
// within that span of the given character position within line in position,
// and false if out of range
func (tv *TextView) WrappedLineNo(pos TextPos) (si, ri int, ok bool) {
	return tv.Renders[pos.Ln].RuneSpanPos(pos.Ch)
}

// CursorDown moves the cursor down line(s)
func (tv *TextView) CursorDown(steps int) {
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
	tv.CursorPos = pos
	tv.ScrollCursorToCenterIfHidden()
	tv.RenderCursor(true)
	tv.CursorSelect(org)
}

// CursorPageDown moves the cursor down page(s), where a page is defined
// dynamically as just moving the cursor off the screen
func (tv *TextView) CursorPageDown(steps int) {
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
	tv.CursorSelect(org)
}

// CursorBackward moves the cursor backward
func (tv *TextView) CursorBackward(steps int) {
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
	tv.ScrollCursorToCenterIfHidden()
	tv.RenderCursor(true)
	tv.CursorSelect(org)
}

// CursorUp moves the cursor up line(s)
func (tv *TextView) CursorUp(steps int) {
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
	tv.CursorPos = pos
	tv.ScrollCursorToCenterIfHidden()
	tv.RenderCursor(true)
	tv.CursorSelect(org)
}

// CursorPageUp moves the cursor up page(s), where a page is defined
// dynamically as just moving the cursor off the screen
func (tv *TextView) CursorPageUp(steps int) {
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
	tv.CursorSelect(org)
}

// CursorRecenter re-centers the view around the cursor position, toggling
// between putting cursor in middle, top, and bottom of view
func (tv *TextView) CursorRecenter() {
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
	org := tv.CursorPos
	tv.CursorPos.Ch = 0
	tv.CursorCol = tv.CursorPos.Ch
	tv.ScrollCursorToLeft()
	tv.RenderCursor(true)
	tv.CursorSelect(org)
}

// CursorStart moves the cursor to the start of the text, updating selection
// if select mode is active
func (tv *TextView) CursorStart() {
	org := tv.CursorPos
	tv.CursorPos.Ln = 0
	tv.CursorPos.Ch = 0
	tv.CursorCol = tv.CursorPos.Ch
	tv.ScrollCursorToTop()
	tv.RenderCursor(true)
	tv.CursorSelect(org)
}

// CursorEndLine moves the cursor to the end of the text
func (tv *TextView) CursorEndLine() {
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
	tv.ScrollCursorToRight()
	tv.RenderCursor(true)
	tv.CursorSelect(org)
}

// CursorEnd moves the cursor to the end of the text, updating selection if
// select mode is active
func (tv *TextView) CursorEnd() {
	updt := tv.UpdateStart()
	defer tv.UpdateEnd(updt)
	org := tv.CursorPos
	tv.CursorPos.Ln = gi.MaxInt(tv.NLines-1, 0)
	tv.CursorPos.Ch = len(tv.Buf.Lines[tv.CursorPos.Ln])
	tv.CursorCol = tv.CursorPos.Ch
	tv.ScrollCursorToBottom()
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
	tv.CursorBackward(steps)
	tv.ScrollCursorToCenterIfHidden()
	tv.RenderCursor(true)
	tv.Buf.DeleteText(tv.CursorPos, org, true)
}

// CursorDelete deletes character(s) immediately after the cursor
func (tv *TextView) CursorDelete(steps int) {
	if tv.HasSelection() {
		tv.DeleteSelection()
		return
	}
	// note: no update b/c signal from buf will drive update
	org := tv.CursorPos
	tv.CursorForward(steps)
	tv.Buf.DeleteText(org, tv.CursorPos, true)
	tv.SetCursor(org)
	tv.ScrollCursorToCenterIfHidden()
	tv.RenderCursor(true)
}

// CursorKill deletes text from cursor to end of text
func (tv *TextView) CursorKill() {
	org := tv.CursorPos
	if tv.CursorPos.Ch == 0 && len(tv.Buf.Lines[tv.CursorPos.Ln]) == 0 {
		tv.CursorForward(1)
	} else {
		tv.CursorEndLine()
	}
	tv.Buf.DeleteText(org, tv.CursorPos, true)
	tv.SetCursor(org)
	tv.ScrollCursorToCenterIfHidden()
	tv.RenderCursor(true)
}

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
	tv.Buf.DeleteText(tv.SelectReg.Start, tv.SelectReg.End, true)
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
	tbe := tv.Buf.InsertText(tv.CursorPos, txt, true)
	tv.SetCursor(tbe.Reg.End)
	tv.ScrollCursorToCenterIfHidden()
	tv.RenderCursor(true)
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

// OfferCompletions pops up a menu of possible completions
func (tv *TextView) OfferCompletions() {
	if tv.Completion == nil {
		return
	}
	win := tv.ParentWindow()
	if gi.PopupIsCompleter(win.Popup) {
		win.ClosePopup(win.Popup)
	}

	st := TextPos{tv.CursorPos.Ln, 0}
	en := TextPos{tv.CursorPos.Ln, tv.CursorPos.Ch}
	tbe := tv.Buf.Region(st, en)
	if tbe != nil {
		s := string(tbe.ToBytes())
		s = strings.TrimLeft(s, " \t") // trim ' ' and '\t'
		fmt.Println(s)
		cp := tv.CharStartPos(tv.CursorPos)
		tv.Completion.ShowCompletions(s, tv.Viewport, int(cp.X+5), int(cp.Y+10))
	}
}

// Complete edits the text using the string chosen from the completion menu
func (tv *TextView) Complete(s string) {
	win := tv.ParentWindow()
	win.ClosePopup(win.Popup)

	st := TextPos{tv.CursorPos.Ln, 0}
	en := TextPos{tv.CursorPos.Ln, tv.CursorPos.Ch}
	tbe := tv.Buf.Region(st, en)
	tbes := string(tbe.ToBytes())

	ns, _ := tv.Completion.EditFunc(tbes, tv.CursorPos.Ch, s, tv.Completion.Seed)
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
	// note: only need to connect once..
	tv.Completion.CompleteSig.ConnectOnly(tv.This, func(recv, send ki.Ki, sig int64, data interface{}) {
		tvf, _ := recv.Embed(KiT_TextView).(*TextView)
		if sig == int64(gi.CompleteSelect) {
			tvf.Complete(data.(string)) // always use data
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
		tv.OfferCompletions()
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
	spos.Y += tv.Offs[pos.Ln] + gi.FixedToFloat32(tv.Sty.Font.Face.Metrics().Descent)
	spos.X += tv.LineNoOff
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

// RenderSelect renders the selection region as a highlighted background color
// -- always called within context of outer RenderLines or RenderAllLines
func (tv *TextView) RenderSelect() {
	if !tv.HasSelection() {
		return
	}
	rs := &tv.Viewport.Render
	pc := &rs.Paint
	sty := &tv.StateStyles[TextViewSel]
	spc := sty.BoxSpace()

	st := tv.SelectReg.Start
	ed := tv.SelectReg.End
	ed.Ch-- // end is exclusive
	spos := tv.CharStartPos(st)
	epos := tv.CharEndPos(ed)

	// fmt.Printf("select: %v -- %v\n", st, ed)

	if st.Ln == ed.Ln {
		pc.FillBox(rs, spos, epos.Sub(spos), &sty.Font.BgColor)
	} else {
		if st.Ch > 0 {
			se := tv.CharEndPos(st)
			se.X = float32(tv.VpBBox.Max.X) - spc
			pc.FillBox(rs, spos, se.Sub(spos), &sty.Font.BgColor)
			st.Ln++
			st.Ch = 0
			spos = tv.CharStartPos(st)
		}
		lm1 := ed
		lm1.Ln--
		be := tv.CharEndPos(lm1)
		be.X = float32(tv.VpBBox.Max.X) - spc
		pc.FillBox(rs, spos, be.Sub(spos), &sty.Font.BgColor)
		// now get anything on end
		if ed.Ch > 0 {
			els := ed
			els.Ch = 0
			elsp := tv.CharStartPos(els)
			pc.FillBox(rs, elsp, epos.Sub(elsp), &sty.Font.BgColor)
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
	if tv.LineNos {
		tv.LineNoOff = float32(tv.LineNoDigs+3)*sty.Font.Ch + spc // space for icon
	} else {
		tv.LineNoOff = 0
	}
}

// RenderAllLines displays all the visible lines on the screen -- called
// during standard render
func (tv *TextView) RenderAllLines() {
	if tv.PushBounds() {
		vp := tv.Viewport
		updt := vp.Win.UpdateStart()

		sty := &tv.Sty
		sty.Font.OpenFont(&sty.UnContext)
		tv.VisSizes()
		tv.RenderStdBox(sty)
		tv.RenderLineNosBoxAll()
		tv.RenderSelect()
		rs := &tv.Viewport.Render
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

		tv.PopBounds()
		vp.Win.UploadVpRegion(vp, tv.VpBBox, tv.WinBBox)
		vp.Win.UpdateEnd(updt)
	}
}

// RenderLineNosBoxAll renders the background for the line numbers in a darker shade
func (tv *TextView) RenderLineNosBoxAll() {
	if !tv.LineNos {
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
	if !tv.LineNos {
		return
	}
	rs := &tv.Viewport.Render
	pc := &rs.Paint
	sty := &tv.Sty
	spc := sty.BoxSpace()
	clr := sty.Font.BgColor.Color.Highlight(10)
	spos := tv.CharStartPos(TextPos{Ln: st})
	spos.X = float32(tv.VpBBox.Min.X) + spc
	epos := tv.CharEndPos(TextPos{Ln: ed})
	epos.X = spos.X + tv.LineNoOff - spc
	pc.FillBoxColor(rs, spos, epos.Sub(spos), clr)
}

// RenderLineNo renders given line number -- called within context of other render
func (tv *TextView) RenderLineNo(ln int) {
	if !tv.LineNos {
		return
	}
	vp := tv.Viewport
	sty := &tv.Sty
	fst := sty.Font
	fst.BgColor.SetColor(nil)
	rs := &vp.Render
	lfmt := fmt.Sprintf("%v", tv.LineNoDigs)
	lfmt = "%0" + lfmt + "d"
	lnstr := fmt.Sprintf(lfmt, ln)
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

			tv.RenderSelect()
			tv.RenderLineNosBox(st, ed)

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

// SetCursorFromPixel sets cursor position from pixel location, e.g., from
// mouse action -- handles the selection updating etc.
func (tv *TextView) SetCursorFromPixel(pt image.Point, selMode mouse.SelectModes) {
	oldPos := tv.CursorPos
	newPos := tv.PixelToCursor(pt)
	if newPos == oldPos {
		return
	}
	tv.SetCursor(newPos)
	if tv.SelectMode || selMode != mouse.NoSelectMode {
		if !tv.SelectMode && selMode != mouse.NoSelectMode {
			tv.SelectReg.Start = oldPos
			tv.SelectMode = true
		}
		if !tv.IsDragging() && tv.SelectReg.Start.IsLess(tv.CursorPos) && tv.CursorPos.IsLess(tv.SelectReg.End) {
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

// KeyInput handles keyboard input into the text field and from the completion menu
func (tv *TextView) KeyInput(kt *key.ChordEvent) {
	kf := gi.KeyFun(kt.ChordString())
	win := tv.ParentWindow()

	if gi.PopupIsCompleter(win.Popup) {
		tv.Completion.KeyInput(kf)
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
	case gi.KeyFunPageUp:
		kt.SetProcessed()
		tv.CursorPageUp(1)
	case gi.KeyFunPageDown:
		kt.SetProcessed()
		tv.CursorPageDown(1)
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
	case gi.KeyFunUndo:
		kt.SetProcessed()
		tv.Undo()
	case gi.KeyFunRedo:
		kt.SetProcessed()
		tv.Redo()
	case gi.KeyFunComplete:
		kt.SetProcessed()
		tv.OfferCompletions()
	case gi.KeyFunRecenter:
		kt.SetProcessed()
		tv.CursorRecenter()
	case gi.KeyFunSelectItem: // enter
		if !kt.HasAnyModifier(key.Control, key.Meta) {
			kt.SetProcessed()
			tv.InsertAtCursor([]byte("\n"))
		}
	case gi.KeyFunFocusNext: // tab
		if !kt.HasAnyModifier(key.Control, key.Meta) {
			kt.SetProcessed()
			tv.InsertAtCursor([]byte("\t"))
		}
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
				tv.SetCursorFromPixel(pt, me.SelectMode())
			}
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
			pt := tv.PointToRelPos(me.Pos())
			tv.SetCursorFromPixel(pt, me.SelectMode())
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
		txf.SetCursorFromPixel(pt, mouse.NoSelectMode)
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
	tv.CursorWidth.SetFmInheritProp("cursor-width", tv.This, true, true) // inherit and get type defaults
	tv.CursorWidth.ToDots(&tv.Sty.UnContext)
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
		tv.EmitFocusedSignal()
		tv.UpdateSig()
	case gi.FocusInactive:
		tv.FocusActive = false
		tv.EditDone()
		tv.UpdateSig()
	case gi.FocusActive:
		tv.FocusActive = true
		// tv.UpdateSig()
		// todo: see about cursor
	}
}
