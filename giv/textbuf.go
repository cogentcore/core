// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package giv

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/alecthomas/chroma/lexers"
	"github.com/goki/gi"
	"github.com/goki/ki"
	"github.com/goki/ki/ints"
	"github.com/goki/ki/kit"
	"github.com/pmezard/go-difflib/difflib"
)

// TextBuf is a buffer of text, which can be viewed by TextView(s).  It holds
// the raw text lines (in original string and rune formats, and marked-up from
// syntax highlighting), and sends signals for making edits to the text and
// coordinating those edits across multiple views.  Views always only view a
// single buffer, so they directly call methods on the buffer to drive
// updates, which are then broadast.  It also has methods for loading and
// saving buffers to files.  Unlike GUI Widgets, its methods are generally
// signaling, without an explicit Action suffix.  Internally, the buffer
// represents new lines using \n = LF, but saving and loading can deal with
// Windows/DOS CRLF format.
type TextBuf struct {
	ki.Node
	Txt        []byte         `json:"-" xml:"text" desc:"the current value of the entire text being edited -- using []byte slice for greater efficiency"`
	Autosave   bool           `desc:"if true, auto-save file after changes (in a separate routine)"`
	Changed    bool           `json:"-" xml:"-" desc:"true if the text has been changed (edited) relative to the original, since last save"`
	Filename   gi.FileName    `json:"-" xml:"-" desc:"filename of file last loaded or saved"`
	Info       FileInfo       `desc:"full info about file"`
	Hi         HiMarkup       `desc:"syntax highlighting markup parameters (language, style, etc)"`
	NLines     int            `json:"-" xml:"-" desc:"number of lines"`
	Lines      [][]rune       `json:"-" xml:"-" desc:"the live lines of text being edited, with latest modifications -- encoded as runes per line, which is necessary for one-to-one rune / glyph rendering correspondence"`
	LineBytes  [][]byte       `json:"-" xml:"-" desc:"the live lines of text being edited, with latest modifications -- encoded in bytes per line -- these are initially just pointers into source Txt bytes"`
	Markup     [][]byte       `json:"-" xml:"-" desc:"marked-up version of the edit text lines, after being run through the syntax highlighting process -- this is what is actually rendered"`
	ByteOffs   []int          `json:"-" xml:"-" desc:"offsets for start of each line in Txt []byte slice -- this is NOT updated with edits -- call SetByteOffs to set it when needed -- used for re-generating the Txt in LinesToBytes, and set on initial open in BytesToLines"`
	TotalBytes int            `json:"-" xml:"-" desc:"total bytes in document -- see ByteOffs for when it is updated"`
	MarkupMu   sync.Mutex     `json:"-" xml:"-" desc:"mutex for updating markup"`
	TextBufSig ki.Signal      `json:"-" xml:"-" view:"-" desc:"signal for buffer -- see TextBufSignals for the types"`
	Views      []*TextView    `json:"-" xml:"-" desc:"the TextViews that are currently viewing this buffer"`
	Undos      []*TextBufEdit `json:"-" xml:"-" desc:"undo stack of edits"`
	UndoPos    int            `json:"-" xml:"-" desc:"undo position"`
	FileModOk  bool           `json:"-" xml:"-" desc:"have already asked about fact that file has changed since being opened, user is ok"`
	PosHistory []TextPos      `json:"-" xml:"-" desc:"history of cursor positions -- can move back through them"`
}

var KiT_TextBuf = kit.Types.AddType(&TextBuf{}, TextBufProps)

var TextBufProps = ki.Props{
	"CallMethods": ki.PropSlice{
		{"SaveAs", ki.Props{
			"Args": ki.PropSlice{
				{"File Name", ki.Props{
					"default-field": "Filename",
				}},
			},
		}},
	},
}

// TextBufSignals are signals that text buffer can send
type TextBufSignals int64

const (
	// TextBufDone means that editing was completed and applied to Txt field
	// -- data is Txt bytes
	TextBufDone TextBufSignals = iota

	// TextBufNew signals that entirely new text is present -- all views
	// update -- data is Txt bytes.
	TextBufNew

	// TextBufInsert signals that some text was inserted -- data is
	// TextBufEdit describing change -- the TextBuf always reflects the
	// current state *after* the edit.
	TextBufInsert

	// TextBufDelete signals that some text was deleted -- data is
	// TextBufEdit describing change -- the TextBuf always reflects the
	// current state *after* the edit.
	TextBufDelete

	// TextBufMarkUpdt signals that the Markup text has been updated -- this
	// signal is typically sent from a separate goroutine so should be used
	// with a mutex
	TextBufMarkUpdt

	TextBufSignalsN
)

//go:generate stringer -type=TextBufSignals

// EditDone finalizes any current editing, sends signal
func (tb *TextBuf) EditDone() {
	if tb.Changed {
		tb.AutoSaveDelete()
		tb.Changed = false
		tb.LinesToBytes()
		tb.TextBufSig.Emit(tb.This, int64(TextBufDone), tb.Txt)
	}
}

// Text returns the current text as a []byte array, applying all current
// changes -- calls EditDone and will generate that signal if there have been
// changes
func (tb *TextBuf) Text() []byte {
	tb.EditDone()
	return tb.Txt
}

// Refresh signals any views to refresh views
func (tb *TextBuf) Refresh() {
	tb.TextBufSig.Emit(tb.This, int64(TextBufNew), tb.Txt)
}

// todo: use https://github.com/andybalholm/crlf to deal with cr/lf etc --
// internally just use lf = \n

// New initializes a new buffer with n blank lines
func (tb *TextBuf) New(nlines int) {
	tb.MarkupMu.Lock()
	tb.Lines = make([][]rune, nlines)
	tb.LineBytes = make([][]byte, nlines)
	tb.Markup = make([][]byte, nlines)

	if cap(tb.ByteOffs) >= nlines {
		tb.ByteOffs = tb.ByteOffs[:nlines]
	} else {
		tb.ByteOffs = make([]int, nlines)
	}

	if nlines == 1 { // this is used for a new blank doc
		tb.ByteOffs[0] = 0 // by definition
		tb.Lines[0] = []rune("")
		tb.LineBytes[0] = []byte("")
		tb.Markup[0] = []byte("")
	}

	tb.NLines = nlines
	tb.MarkupMu.Unlock()
	tb.Refresh()
}

// Stat gets info about the file, including highlighting language
func (tb *TextBuf) Stat() error {
	tb.FileModOk = false
	err := tb.Info.InitFile(string(tb.Filename))
	if err != nil {
		return err
	}
	lexer := lexers.Match(tb.Info.Name)
	if lexer == nil && tb.NLines > 0 {
		lexer = lexers.Analyse(string(tb.Txt))
	}
	if lexer != nil {
		tb.Hi.Lang = lexer.Config().Name
	}
	return nil
}

// FileModCheck checks if the underlying file has been modified since last
// Stat (open, save) -- if haven't yet prompted, user is prompted to ensure
// that this is OK
func (tb *TextBuf) FileModCheck() {
	if tb.FileModOk {
		return
	}
	info, err := os.Stat(string(tb.Filename))
	if err != nil {
		return
	}
	if info.ModTime() != time.Time(tb.Info.ModTime) {
		vp := tb.ViewportFromView()
		gi.ChoiceDialog(vp, gi.DlgOpts{Title: "File Changed on Disk",
			Prompt: fmt.Sprintf("File has changed on disk since being opened or saved by you -- what do you want to do?  File: %v", tb.Filename)},
			[]string{"Save To Different File", "Open From Disk, Losing Changes", "Ignore and Proceed"},
			tb.This, func(recv, send ki.Ki, sig int64, data interface{}) {
				switch sig {
				case 0:
					CallMethod(tb, "SaveAs", vp)
				case 1:
					tb.ReOpen()
				case 2:
					tb.FileModOk = true
				}
			})
	}
}

// Open loads text from a file into the buffer
func (tb *TextBuf) Open(filename gi.FileName) error {
	err := tb.OpenFile(filename)
	if err != nil {
		vp := tb.ViewportFromView()
		gi.PromptDialog(vp, gi.DlgOpts{Title: "File could not be Opened", Prompt: err.Error()}, true, false, nil, nil)
		log.Println(err)
		return err
	}
	tb.SetName(string(filename)) // todo: modify in any way?

	// markup the first 100 lines
	mxhi := ints.MinInt(100, tb.NLines-1)
	tb.MarkupLines(0, mxhi)

	// update views
	tb.TextBufSig.Emit(tb.This, int64(TextBufNew), tb.Txt)

	// do slow full update in background
	go tb.MarkupAllLines() // then do all in background
	return nil
}

// OpenFile just loads a file into the buffer -- doesn't do any markup or
// notification -- for temp bufs
func (tb *TextBuf) OpenFile(filename gi.FileName) error {
	fp, err := os.Open(string(filename))
	if err != nil {
		return err
	}
	tb.Txt, err = ioutil.ReadAll(fp)
	fp.Close()
	tb.Filename = filename
	tb.Stat()
	tb.BytesToLines()
	return nil
}

// ReOpen re-opens text from current file, if filename set -- returns false if
// not -- uses an optimized diff-based update to preserve existing formatting
// -- very fast if not very different
func (tb *TextBuf) ReOpen() bool {
	tb.AutoSaveDelete() // justin case
	if tb.Filename == "" {
		return false
	}

	ob := &TextBuf{}
	ob.InitName(ob, "re-open-tmp")
	err := ob.OpenFile(tb.Filename)
	if err != nil {
		vp := tb.ViewportFromView()
		if vp != nil { // only if viewing
			gi.PromptDialog(vp, gi.DlgOpts{Title: "File could not be Re-Opened", Prompt: err.Error()}, true, false, nil, nil)
		}
		log.Println(err)
		return false
	}
	tb.Stat() // "own" the new file..
	diffs := tb.DiffBufs(ob)
	tb.PatchFromBuf(ob, diffs, true) // true = send sigs for each update -- better than full, assuming changes are minor
	tb.Changed = false
	go tb.MarkupAllLines() // always do global reformat in bg
	return true
}

// SaveAs saves the current text into given file -- does an EditDone first to save edits
func (tb *TextBuf) SaveAs(filename gi.FileName) error {
	// todo: filemodcheck!
	tb.EditDone()
	if _, err := os.Stat(string(filename)); !os.IsNotExist(err) {
		vp := tb.ViewportFromView()
		gi.ChoiceDialog(vp, gi.DlgOpts{Title: "File Exists, Overwrite?",
			Prompt: fmt.Sprintf("File already exists, overwrite?  File: %v", filename)},
			[]string{"Cancel", "Overwrite"},
			tb.This, func(recv, send ki.Ki, sig int64, data interface{}) {
				switch sig {
				case 0:
					// do nothing
				case 1:
					tb.SaveFile(filename)
				}
			})
		return nil
	} else {
		return tb.SaveFile(filename)
	}
}

// SaveFile writes current buffer to file, with no prompting, etc
func (tb *TextBuf) SaveFile(filename gi.FileName) error {
	err := ioutil.WriteFile(string(filename), tb.Txt, 0644)
	if err != nil {
		gi.PromptDialog(nil, gi.DlgOpts{Title: "Could not Save to File", Prompt: err.Error()}, true, false, nil, nil)
		log.Println(err)
	} else {
		tb.Filename = filename
		tb.SetName(string(filename)) // todo: modify in any way?
		tb.Stat()
	}
	return err
}

// Save saves the current text into current Filename associated with this
// buffer
func (tb *TextBuf) Save() error {
	if tb.Filename == "" {
		return fmt.Errorf("giv.TextBuf: filename is empty for Save")
	}
	tb.EditDone()
	info, err := os.Stat(string(tb.Filename))
	if err == nil && info.ModTime() != time.Time(tb.Info.ModTime) {
		vp := tb.ViewportFromView()
		gi.ChoiceDialog(vp, gi.DlgOpts{Title: "File Changed on Disk",
			Prompt: fmt.Sprintf("File has changed on disk since being opened or saved by you -- what do you want to do?  File: %v", tb.Filename)},
			[]string{"Save To Different File", "Open From Disk, Losing Changes", "Save File, Overwriting"},
			tb.This, func(recv, send ki.Ki, sig int64, data interface{}) {
				switch sig {
				case 0:
					CallMethod(tb, "SaveAs", vp)
				case 1:
					tb.ReOpen()
				case 2:
					tb.SaveFile(tb.Filename)
				}
			})
	}
	return tb.SaveFile(tb.Filename)
}

// Close closes the buffer -- prompts to save if changes, and disconnects from views
func (tb *TextBuf) Close() bool {
	if tb.Changed {
		vp := tb.ViewportFromView()
		if tb.Filename != "" {
			gi.ChoiceDialog(vp, gi.DlgOpts{Title: "Close Without Saving?",
				Prompt: fmt.Sprintf("Do you want to save your changes to file: %v?", tb.Filename)},
				[]string{"Save", "Close Without Saving", "Cancel"},
				tb.This, func(recv, send ki.Ki, sig int64, data interface{}) {
					switch sig {
					case 0:
						tb.Save()
						tb.Close() // 2nd time through won't prompt
					case 1:
						tb.Changed = false
						tb.AutoSaveDelete()
						tb.Close()
					}
				})
		} else {
			gi.ChoiceDialog(vp, gi.DlgOpts{Title: "Close Without Saving?",
				Prompt: "Do you want to save your changes (no filename for this buffer yet)?  If so, Cancel and then do Save As"},
				[]string{"Close Without Saving", "Cancel"},
				tb.This, func(recv, send ki.Ki, sig int64, data interface{}) {
					switch sig {
					case 0:
						tb.Changed = false
						tb.AutoSaveDelete()
						tb.Close()
					case 1:
					}
				})
		}
		return false // awaiting decisions..
	}
	for _, tve := range tb.Views {
		tve.SetBuf(nil) // automatically disconnects signals, views
	}
	tb.New(1)
	tb.Filename = ""
	tb.Changed = false
	return true
}

// AutoSaveFilename returns the autosave filename
func (tb *TextBuf) AutoSaveFilename() string {
	path, fn := filepath.Split(string(tb.Filename))
	if fn == "" {
		fn = "new_file_" + tb.Nm
	}
	asfn := filepath.Join(path, "#"+fn+"#")
	return asfn
}

// AutoSave does the autosave -- safe to call in a separate goroutine
func (tb *TextBuf) AutoSave() error {
	asfn := tb.AutoSaveFilename()
	b := tb.LinesToBytesCopy()
	err := ioutil.WriteFile(asfn, b, 0644)
	if err != nil {
		log.Printf("giv.TextBuf: Could not AutoSave file: %v, error: %v\n", asfn, err)
	}
	return err
}

// AutoSaveDelete deletes any existing autosave file
func (tb *TextBuf) AutoSaveDelete() {
	asfn := tb.AutoSaveFilename()
	os.Remove(asfn)
}

// AutoSaveCheck checks if an autosave file exists -- logic for dealing with
// it is left to larger app -- call this before opening a file
func (tb *TextBuf) AutoSaveCheck() bool {
	asfn := tb.AutoSaveFilename()
	if _, err := os.Stat(asfn); os.IsNotExist(err) {
		return false // does not exist
	}
	return true
}

/////////////////////////////////////////////////////////////////////////////
//   Appending Lines

// EndPos returns the ending position at end of buffer
func (tb *TextBuf) EndPos() TextPos {
	if tb.NLines == 0 {
		return TextPosZero
	}
	ed := TextPos{tb.NLines - 1, len(tb.Lines[tb.NLines-1])}
	return ed
}

// AppendText appends new text to end of buffer, using insert, returns edit
func (tb *TextBuf) AppendText(text []byte, saveUndo, signal bool) *TextBufEdit {
	if len(text) == 0 {
		return &TextBufEdit{}
	}
	ed := tb.EndPos()
	return tb.InsertText(ed, text, saveUndo, signal)
}

// AppendTextLine appends one line of new text to end of buffer, using insert,
// and appending a LF at the end of the line if it doesn't already have one.
// Returns the edit region.
func (tb *TextBuf) AppendTextLine(text []byte, saveUndo, signal bool) *TextBufEdit {
	ed := tb.EndPos()
	sz := len(text)
	addLF := false
	if sz > 0 {
		if text[sz-1] != '\n' {
			addLF = true
		}
	} else {
		addLF = true
	}
	efft := text
	if addLF {
		tcpy := make([]byte, sz+1)
		copy(tcpy, text)
		tcpy[sz] = '\n'
		efft = tcpy
	}
	tbe := tb.InsertText(ed, efft, saveUndo, signal)
	return tbe
}

// AppendTextMarkup appends new text to end of buffer, using insert, returns
// edit, and uses supplied markup to render it
func (tb *TextBuf) AppendTextMarkup(text []byte, markup []byte, saveUndo, signal bool) *TextBufEdit {
	if len(text) == 0 {
		return &TextBufEdit{}
	}
	ed := tb.EndPos()
	tbe := tb.InsertText(ed, text, saveUndo, false) // no sig -- we do later

	st := tbe.Reg.Start.Ln
	el := tbe.Reg.End.Ln
	sz := (el - st) + 1
	msplt := bytes.Split(markup, []byte("\n"))
	if len(msplt) < sz {
		log.Printf("TextBuf AppendTextMarkup: markup text less than appended text: is: %v, should be: %v\n", len(msplt), sz)
		el = ints.MinInt(st+len(msplt)-1, el)
	}
	for ln := st; ln <= el; ln++ {
		tb.Markup[ln] = msplt[ln-st]
	}
	if signal {
		tb.TextBufSig.Emit(tb.This, int64(TextBufInsert), tbe)
	}
	return tbe
}

// AppendTextLineMarkup appends one line of new text to end of buffer, using
// insert, and appending a LF at the end of the line if it doesn't already
// have one.  user-supplied markup is used.  Returns the edit region.
func (tb *TextBuf) AppendTextLineMarkup(text []byte, markup []byte, saveUndo, signal bool) *TextBufEdit {
	ed := tb.EndPos()
	sz := len(text)
	addLF := false
	if sz > 0 {
		if text[sz-1] != '\n' {
			addLF = true
		}
	} else {
		addLF = true
	}
	efft := text
	if addLF {
		tcpy := make([]byte, sz+1)
		copy(tcpy, text)
		tcpy[sz] = '\n'
		efft = tcpy
	}
	tbe := tb.InsertText(ed, efft, saveUndo, false)
	tb.Markup[tbe.Reg.Start.Ln] = markup
	if signal {
		tb.TextBufSig.Emit(tb.This, int64(TextBufInsert), tbe)
	}
	return tbe
}

/////////////////////////////////////////////////////////////////////////////
//   Views

// AddView adds a viewer of this buffer -- connects our signals to the viewer
func (tb *TextBuf) AddView(vw *TextView) {
	tb.Views = append(tb.Views, vw)
	tb.TextBufSig.Connect(vw.This, TextViewBufSigRecv)
}

// DeleteView removes given viewer from our buffer
func (tb *TextBuf) DeleteView(vw *TextView) {
	for i, tve := range tb.Views {
		if tve == vw {
			tb.Views = append(tb.Views[:i], tb.Views[i+1:]...)
			break
		}
	}
	tb.TextBufSig.Disconnect(vw.This)
}

// ViewportFromView returns Viewport from textview, if avail
func (tb *TextBuf) ViewportFromView() *gi.Viewport2D {
	if len(tb.Views) > 0 {
		return tb.Views[0].Viewport
	}
	return nil
}

// AutoscrollViews ensures that views are always viewing the end of the buffer
func (tb *TextBuf) AutoScrollViews() {
	for _, tv := range tb.Views {
		tv.CursorPos = tb.EndPos()
		tv.ScrollCursorInView()
	}
}

/////////////////////////////////////////////////////////////////////////////
//   Accessing Text

// SetByteOffs sets the byte offsets for each line into the raw text
func (tb *TextBuf) SetByteOffs() {
	bo := 0
	for ln, txt := range tb.LineBytes {
		tb.ByteOffs[ln] = bo
		bo += len(txt) + 1 // lf
	}
	tb.TotalBytes = bo
}

// LinesToBytes converts current Lines back to the Txt slice of bytes.
func (tb *TextBuf) LinesToBytes() {
	if tb.NLines == 0 {
		if tb.Txt != nil {
			tb.Txt = tb.Txt[:0]
		}
		return
	}

	tb.Txt = tb.LinesToBytesCopy()

	// the following does not work because LineBytes is just pointers into txt!
	// tb.SetByteOffs()
	// totsz := tb.TotalBytes

	// if cap(tb.Txt) < totsz {
	// 	tb.Txt = make([]byte, totsz)
	// } else {
	// 	tb.Txt = tb.Txt[:totsz]
	// }

	// for ln := range tb.Lines {
	// 	bo := tb.ByteOffs[ln]
	// 	lsz := len(tb.LineBytes[ln])
	// 	copy(tb.Txt[bo:bo+lsz], tb.LineBytes[ln])
	// 	tb.Txt[bo+lsz] = '\n'
	// }
}

// LinesToBytesCopy converts current Lines into a separate text byte copy --
// e.g., for autosave or other "offline" uses of the text -- doesn't affect
// byte offsets etc
func (tb *TextBuf) LinesToBytesCopy() []byte {
	txt := bytes.Join(tb.LineBytes, []byte("\n"))
	txt = append(txt, '\n')
	return txt
}

// BytesToLines converts current Txt bytes into lines, and initializes markup
// with raw text
func (tb *TextBuf) BytesToLines() {
	tb.Hi.Init()
	if len(tb.Txt) == 0 {
		tb.New(1)
		return
	}
	lns := bytes.Split(tb.Txt, []byte("\n"))
	tb.NLines = len(lns)
	if len(lns[tb.NLines-1]) == 0 { // lines have lf at end typically
		tb.NLines--
		lns = lns[:tb.NLines]
	}
	tb.New(tb.NLines)
	bo := 0
	for ln, txt := range lns {
		tb.ByteOffs[ln] = bo
		tb.Lines[ln] = bytes.Runes(txt)
		tb.LineBytes[ln] = txt
		tb.Markup[ln] = tb.LineBytes[ln]
		bo += len(txt) + 1 // lf
	}
	tb.TotalBytes = bo
}

/////////////////////////////////////////////////////////////////////////////
//   Search

// Search looks for a string (no regexp) within buffer, in a case-sensitive
// way, returning number of occurences and specific match position list.
// Currently ONLY returning byte char positions, not rune ones..
func (tb *TextBuf) Search(find string) (int, []TextPos) {
	fsz := len(find)
	if fsz == 0 {
		return 0, nil
	}
	cnt := 0
	var matches []TextPos
	for ln, lr := range tb.Lines {
		lstr := string(lr)
		sz := len(lstr)
		ci := 0
		for ci < sz {
			i := strings.Index(lstr[ci:], find)
			if i < 0 {
				break
			}
			i += ci
			ci = i + fsz
			matches = append(matches, TextPos{ln, i})
			cnt++
		}
	}
	return cnt, matches
}

// SearchCI looks for a string (no regexp) within buffer, in a
// case-INsensitive way, returning number of occurences.  Currently ONLY
// returning byte char positions, not rune ones..
func (tb *TextBuf) SearchCI(find string) (int, []TextPos) {
	fsz := len(find)
	if fsz == 0 {
		return 0, nil
	}
	find = strings.ToLower(find)
	cnt := 0
	var matches []TextPos
	for ln, lr := range tb.Lines {
		lstr := strings.ToLower(string(lr))
		sz := len(lstr)
		ci := 0
		for ci < sz {
			i := strings.Index(lstr[ci:], find)
			if i < 0 {
				break
			}
			i += ci
			ci = i + fsz
			matches = append(matches, TextPos{ln, i})
			cnt++
		}
	}
	return cnt, matches
}

/////////////////////////////////////////////////////////////////////////////
//   TextPos, TextRegion, TextBufEdit

// TextPos represents line, character positions within the TextBuf and TextView
type TextPos struct {
	Ln, Ch int
}

var TextPosZero = TextPos{}

// IsLess returns true if receiver position is less than given comparison
func (tp *TextPos) IsLess(cmp TextPos) bool {
	switch {
	case tp.Ln < cmp.Ln:
		return true
	case tp.Ln == cmp.Ln:
		return tp.Ch < cmp.Ch
	default:
		return false
	}
}

// FromString decodes text position from a string representation of form:
// [#]LxxCxx -- used in e.g., URL links -- returns true if successful
func (tp *TextPos) FromString(link string) bool {
	link = strings.TrimPrefix(link, "#")
	lidx := strings.Index(link, "L")
	cidx := strings.Index(link, "C")

	switch {
	case lidx >= 0 && cidx >= 0:
		fmt.Sscanf(link, "L%dC%d", &tp.Ln, &tp.Ch)
		tp.Ln-- // link is 1-based, we use 0-based
		tp.Ch-- // ditto
	case lidx >= 0:
		fmt.Sscanf(link, "L%d", &tp.Ln)
		tp.Ln-- // link is 1-based, we use 0-based
	case cidx >= 0:
		fmt.Sscanf(link, "C%d", &tp.Ch)
		tp.Ch--
	default:
		// todo: could support other formats
		return false
	}
	return true
}

// TextRegion represents a text region as a start / end position
type TextRegion struct {
	Start TextPos
	End   TextPos
}

// FromString decodes text region from a string representation of form:
// [#]LxxCxx-LxxCxx -- used in e.g., URL links -- returns true if successful
func (tp *TextRegion) FromString(link string) bool {
	link = strings.TrimPrefix(link, "#")
	fmt.Sscanf(link, "L%dC%d-L%dC%d", &tp.Start.Ln, &tp.Start.Ch, &tp.End.Ln, &tp.End.Ch)
	tp.Start.Ln--
	tp.Start.Ch--
	tp.End.Ln--
	tp.End.Ch--
	return true
}

// NewTextRegionLen makes a new TextRegion from a starting point and a length
// along same line
func NewTextRegionLen(start TextPos, len int) TextRegion {
	reg := TextRegion{}
	reg.Start = start
	reg.End = start
	reg.End.Ch += len
	return reg
}

var TextRegionZero = TextRegion{}

// TextBufEdit describes an edit action to a buffer -- this is the data passed
// via signals to viewers of the buffer.  Actions are only deletions and
// insertions (a change is a sequence of those, given normal editing
// processes).  The TextBuf always reflects the current state *after* the
// edit.
type TextBufEdit struct {
	Reg    TextRegion `desc:"region for the edit (start is same for previous and current, end is in original pre-delete text for a delete, and in new lines data for an insert"`
	Delete bool       `desc:"action is either a deletion or an insertion"`
	Text   [][]rune   `desc:"text to be inserted"`
}

// ToBytes returns the Text of this edit record to a byte string, with
// newlines at end of each line -- nil if Text is empty
func (te *TextBufEdit) ToBytes() []byte {
	sz := len(te.Text)
	if sz == 0 {
		return nil
	}
	if sz == 1 {
		return []byte(string(te.Text[0]))
	}
	var b []byte
	for i := range te.Text {
		b = append(b, []byte(string(te.Text[i]))...)
		if i < sz-1 {
			b = append(b, '\n')
		}
	}
	return b
}

/////////////////////////////////////////////////////////////////////////////
//   Edits

// ValidPos returns a position that is in a valid range
func (tb *TextBuf) ValidPos(pos TextPos) TextPos {
	if tb.NLines == 0 {
		return TextPosZero
	}
	if pos.Ln < 0 {
		pos.Ln = 0
	}
	pos.Ln = ints.MinInt(pos.Ln, len(tb.Lines)-1)
	llen := len(tb.Lines[pos.Ln])
	pos.Ch = ints.MinInt(pos.Ch, llen)
	if pos.Ch < 0 {
		pos.Ch = 0
	}
	return pos
}

// DeleteText deletes region of text between start and end positions, signaling
// views after text lines have been updated.
func (tb *TextBuf) DeleteText(st, ed TextPos, saveUndo, signal bool) *TextBufEdit {
	st = tb.ValidPos(st)
	ed = tb.ValidPos(ed)
	if st == ed {
		return nil
	}
	if !st.IsLess(ed) {
		log.Printf("giv.TextBuf DeleteText: starting position must be less than ending!: st: %v, ed: %v\n", st, ed)
		return nil
	}
	tb.FileModCheck()
	tb.Changed = true
	tbe := tb.Region(st, ed)
	tbe.Delete = true
	if ed.Ln == st.Ln {
		tb.Lines[st.Ln] = append(tb.Lines[st.Ln][:st.Ch], tb.Lines[st.Ln][ed.Ch:]...)
		tb.LinesEdited(tbe)
	} else {
		// first get chars on start and end
		stln := st.Ln + 1
		cpln := st.Ln
		tb.Lines[st.Ln] = tb.Lines[st.Ln][:st.Ch]
		eoedl := len(tb.Lines[ed.Ln][ed.Ch:])
		var eoed []rune
		if eoedl > 0 { // save it
			eoed = make([]rune, eoedl)
			copy(eoed, tb.Lines[ed.Ln][ed.Ch:])
		}
		tb.Lines = append(tb.Lines[:stln], tb.Lines[ed.Ln+1:]...)
		if eoed != nil {
			tb.Lines[cpln] = append(tb.Lines[cpln], eoed...)
		}
		tb.NLines = len(tb.Lines)
		tb.LinesDeleted(tbe)
	}
	if signal {
		tb.TextBufSig.Emit(tb.This, int64(TextBufDelete), tbe)
	}
	if tb.Autosave {
		go tb.AutoSave()
	}
	if saveUndo {
		tb.SaveUndo(tbe)
	}
	return tbe
}

// Insert inserts new text at given starting position, signaling views after
// text has been inserted
func (tb *TextBuf) InsertText(st TextPos, text []byte, saveUndo, signal bool) *TextBufEdit {
	if len(text) == 0 {
		return nil
	}
	if len(tb.Lines) == 0 {
		tb.New(1)
	}
	st = tb.ValidPos(st)
	tb.FileModCheck()
	tb.Changed = true
	lns := bytes.Split(text, []byte("\n"))
	sz := len(lns)
	rs := bytes.Runes(lns[0])
	rsz := len(rs)
	ed := st
	var tbe *TextBufEdit
	if sz == 1 {
		nt := append(tb.Lines[st.Ln], rs...) // first append to end to extend capacity
		copy(nt[st.Ch+rsz:], nt[st.Ch:])     // move stuff to end
		copy(nt[st.Ch:], rs)                 // copy into position
		tb.Lines[st.Ln] = nt
		ed.Ch += rsz
		tbe = tb.Region(st, ed)
		tb.LinesEdited(tbe)
	} else {
		if tb.Lines[st.Ln] == nil {
			tb.Lines[st.Ln] = []rune("")
		}
		eostl := len(tb.Lines[st.Ln][st.Ch:]) // end of starting line
		var eost []rune
		if eostl > 0 { // save it
			eost = make([]rune, eostl)
			copy(eost, tb.Lines[st.Ln][st.Ch:])
		}
		tb.Lines[st.Ln] = append(tb.Lines[st.Ln][:st.Ch], rs...)
		nsz := sz - 1
		tmp := make([][]rune, nsz)
		for i := 1; i < sz; i++ {
			tmp[i-1] = bytes.Runes(lns[i])
		}
		stln := st.Ln + 1
		nt := append(tb.Lines, tmp...) // first append to end to extend capacity
		copy(nt[stln+nsz:], nt[stln:]) // move stuff to end
		copy(nt[stln:], tmp)           // copy into position
		tb.Lines = nt
		tb.NLines = len(tb.Lines)
		ed.Ln += nsz
		ed.Ch = len(tb.Lines[ed.Ln])
		if eost != nil {
			tb.Lines[ed.Ln] = append(tb.Lines[ed.Ln], eost...)
		}
		tbe = tb.Region(st, ed)
		tb.LinesInserted(tbe)
	}
	if signal {
		tb.TextBufSig.Emit(tb.This, int64(TextBufInsert), tbe)
	}
	if tb.Autosave {
		go tb.AutoSave()
	}
	if saveUndo {
		tb.SaveUndo(tbe)
	}
	return tbe
}

// Region returns a TextBufEdit representation of text between start and end positions
func (tb *TextBuf) Region(st, ed TextPos) *TextBufEdit {
	st = tb.ValidPos(st)
	ed = tb.ValidPos(ed)
	if st == ed {
		return nil
	}
	if !st.IsLess(ed) {
		log.Printf("giv.TextBuf TextRegion: starting position must be less than ending!: st: %v, ed: %v\n", st, ed)
		return nil
	}
	tbe := &TextBufEdit{Reg: TextRegion{Start: st, End: ed}}
	if ed.Ln == st.Ln {
		sz := ed.Ch - st.Ch
		tbe.Text = make([][]rune, 1)
		tbe.Text[0] = make([]rune, sz)
		copy(tbe.Text[0][:sz], tb.Lines[st.Ln][st.Ch:ed.Ch])
	} else {
		// first get chars on start and end
		nlns := (ed.Ln - st.Ln) + 1
		tbe.Text = make([][]rune, nlns)
		stln := st.Ln
		if st.Ch > 0 {
			ec := len(tb.Lines[st.Ln])
			sz := ec - st.Ch
			if sz > 0 {
				tbe.Text[0] = make([]rune, sz)
				copy(tbe.Text[0][0:sz], tb.Lines[st.Ln][st.Ch:])
			}
			stln++
		}
		edln := ed.Ln
		if ed.Ch < len(tb.Lines[ed.Ln]) {
			tbe.Text[ed.Ln-st.Ln] = make([]rune, ed.Ch)
			copy(tbe.Text[ed.Ln-st.Ln], tb.Lines[ed.Ln][:ed.Ch])
			edln--
		}
		for ln := stln; ln <= edln; ln++ {
			ti := ln - st.Ln
			sz := len(tb.Lines[ln])
			tbe.Text[ti] = make([]rune, sz)
			copy(tbe.Text[ti], tb.Lines[ln])
		}
	}
	return tbe
}

// SaveCursorPos saves the cursor position in history stack of cursor positions --
// tracks across views
func (tb *TextBuf) SaveCursorPos(pos TextPos) {
	if tb.PosHistory == nil {
		tb.PosHistory = make([]TextPos, 0, 1000)
	}
	tb.PosHistory = append(tb.PosHistory, pos)
}

/////////////////////////////////////////////////////////////////////////////
//   Syntax Highlighting Markup

// LinesInserted inserts new lines in Markup corresponding to lines
// inserted in Lines text.  Locks and unlocks the Markup mutex
func (tb *TextBuf) LinesInserted(tbe *TextBufEdit) {
	stln := tbe.Reg.Start.Ln + 1
	nsz := (tbe.Reg.End.Ln - tbe.Reg.Start.Ln)

	tb.MarkupMu.Lock()

	// LineBytes
	tmplb := make([][]byte, nsz)
	nlb := append(tb.LineBytes, tmplb...)
	copy(nlb[stln+nsz:], nlb[stln:])
	copy(nlb[stln:], tmplb)
	tb.LineBytes = nlb

	// Markup
	tmpmu := make([][]byte, nsz)
	nmu := append(tb.Markup, tmpmu...) // first append to end to extend capacity
	copy(nmu[stln+nsz:], nmu[stln:])   // move stuff to end
	copy(nmu[stln:], tmpmu)            // copy into position
	tb.Markup = nmu

	// ByteOffs -- maintain mem updt
	tmpof := make([]int, nsz)
	nof := append(tb.ByteOffs, tmpof...)
	copy(nof[stln+nsz:], nof[stln:])
	copy(nof[stln:], tmpof)
	tb.ByteOffs = nof

	st, ed := tbe.Reg.Start.Ln, tbe.Reg.End.Ln
	bo := tb.ByteOffs[st]
	for ln := st; ln <= ed; ln++ {
		tb.LineBytes[ln] = []byte(string(tb.Lines[ln]))
		tb.Markup[ln] = tb.LineBytes[ln]
		tb.ByteOffs[ln] = bo
		bo += len(tb.LineBytes[ln]) + 1
	}
	tb.MarkupLines(st, ed)
	tb.MarkupMu.Unlock()

	go tb.MarkupAllLines() // always do global reformat in bg
}

// LinesDeleted deletes lines in Markup corresponding to lines
// deleted in Lines text.  Locks and unlocks the Markup mutex.
func (tb *TextBuf) LinesDeleted(tbe *TextBufEdit) {
	tb.MarkupMu.Lock()

	stln := tbe.Reg.Start.Ln
	edln := tbe.Reg.End.Ln

	tb.LineBytes = append(tb.LineBytes[:stln], tb.LineBytes[edln:]...)
	tb.Markup = append(tb.Markup[:stln], tb.Markup[edln:]...)
	tb.ByteOffs = append(tb.ByteOffs[:stln], tb.ByteOffs[edln:]...)

	st := tbe.Reg.Start.Ln
	tb.LineBytes[st] = []byte(string(tb.Lines[st]))
	tb.Markup[st] = tb.LineBytes[st]
	tb.MarkupLines(st, st)
	tb.MarkupMu.Unlock()
	// probably don't need to do global markup here..
}

// LinesEdited re-marks-up lines in edit (typically only 1).  Locks and
// unlocks the Markup mutex.
func (tb *TextBuf) LinesEdited(tbe *TextBufEdit) {
	tb.MarkupMu.Lock()

	st, ed := tbe.Reg.Start.Ln, tbe.Reg.End.Ln
	for ln := st; ln <= ed; ln++ {
		tb.LineBytes[ln] = []byte(string(tb.Lines[ln]))
		tb.Markup[ln] = tb.LineBytes[ln]
	}
	tb.MarkupLines(st, ed)
	tb.MarkupMu.Unlock()
	// probably don't need to do global markup here..
}

// MarkupAllLines does syntax highlighting markup for all lines in buffer,
// calling MarkupMu mutex when setting the marked-up lines with the result --
// designed to be called in a separate goroutine
func (tb *TextBuf) MarkupAllLines() {
	if !tb.Hi.HasHi() || tb.NLines == 0 || tb.Hi.lexer == nil {
		return
	}
	tb.LinesToBytes() // todo: could need another mutex here?
	mtlns, err := tb.Hi.MarkupText(tb.Txt)
	if err != nil {
		return
	}

	tb.MarkupMu.Lock()
	maxln := ints.MinInt(len(mtlns)-1, tb.NLines)
	for ln := 0; ln < maxln; ln++ {
		mt := mtlns[ln]
		tb.Markup[ln] = tb.Hi.FixMarkupLine(mt)
	}
	tb.MarkupMu.Unlock()
	tb.TextBufSig.Emit(tb.This, int64(TextBufMarkUpdt), tb.Txt)
}

// MarkupLines generates markup of given range of lines. end is *inclusive*
// line.  returns true if all lines were marked up successfully.  This does
// NOT lock the MarkupMu mutex (done at outer loop)
func (tb *TextBuf) MarkupLines(st, ed int) bool {
	if !tb.Hi.HasHi() || tb.NLines == 0 {
		return false
	}
	if ed >= tb.NLines {
		ed = tb.NLines - 1
	}
	allgood := true
	for ln := st; ln <= ed; ln++ {
		mu, err := tb.Hi.MarkupLine(tb.LineBytes[ln])
		if err == nil {
			tb.Markup[ln] = mu
		} else {
			allgood = false
		}
	}
	return allgood
}

/////////////////////////////////////////////////////////////////////////////
//   Undo

// SaveUndo saves given edit to undo stack
func (tb *TextBuf) SaveUndo(tbe *TextBufEdit) {
	if tb.UndoPos < len(tb.Undos) {
		tb.Undos = tb.Undos[:tb.UndoPos]
	}
	tb.Undos = append(tb.Undos, tbe)
	tb.UndoPos = len(tb.Undos)
}

// Undo undoes next item on the undo stack, and returns that record -- nil if no more
func (tb *TextBuf) Undo() *TextBufEdit {
	if tb.UndoPos == 0 {
		tb.Changed = false // should be!
		tb.AutoSaveDelete()
		return nil
	}
	tb.UndoPos--
	tbe := tb.Undos[tb.UndoPos]
	if tbe.Delete {
		// fmt.Printf("undoing delete at: %v text: %v\n", tbe.Reg, string(tbe.ToBytes()))
		tb.InsertText(tbe.Reg.Start, tbe.ToBytes(), false, true)
	} else {
		// fmt.Printf("undoing insert at: %v text: %v\n", tbe.Reg, string(tbe.ToBytes()))
		tb.DeleteText(tbe.Reg.Start, tbe.Reg.End, false, true)
	}
	return tbe
}

// Redo redoes next item on the undo stack, and returns that record, nil if no more
func (tb *TextBuf) Redo() *TextBufEdit {
	if tb.UndoPos >= len(tb.Undos) {
		return nil
	}
	tbe := tb.Undos[tb.UndoPos]
	if tbe.Delete {
		tb.DeleteText(tbe.Reg.Start, tbe.Reg.End, false, true)
	} else {
		tb.InsertText(tbe.Reg.Start, tbe.ToBytes(), false, true)
	}
	tb.UndoPos++
	return tbe
}

/////////////////////////////////////////////////////////////////////////////
//   Indenting

// LineIndent returns the number of tabs or spaces at start of given line --
// if line starts with tabs, then those are counted, else spaces --
// combinations of tabs and spaces won't produce sensible results
func (tb *TextBuf) LineIndent(ln int, tabSz int) (n int, spc bool) {
	sz := len(tb.Lines[ln])
	if sz == 0 {
		return
	}
	txt := tb.Lines[ln]
	if txt[0] == ' ' {
		spc = true
		n = 1
	} else if txt[0] != '\t' {
		return
	} else {
		n = 1
	}
	if spc {
		for i := 1; i < sz; i++ {
			if txt[i] == ' ' {
				n++
			} else {
				return
			}
		}
	} else {
		for i := 1; i < sz; i++ {
			if txt[i] == '\t' {
				n++
			} else {
				return
			}
		}
	}
	return
}

// IndentBytes returns an indentation string of given number of tab stops,
// using tabs or spaces, for given tab size (if using spaces)
func IndentBytes(n, tabSz int, spc bool) []byte {
	if spc {
		b := make([]byte, n*tabSz)
		for i := 0; i < n*tabSz; i++ {
			b[i] = '\t'
		}
		return b
	} else {
		b := make([]byte, n)
		for i := 0; i < n; i++ {
			b[i] = '\t'
		}
		return b
	}
}

// IndentCharPos returns character position for given level of indentation
func IndentCharPos(n, tabSz int, spc bool) int {
	if spc {
		return n * tabSz
	}
	return n
}

// IndentLine indents line by given number of tab stops, using tabs or spaces,
// for given tab size (if using spaces) -- either inserts or deletes to reach
// target
func (tb *TextBuf) IndentLine(ln int, n, tabSz int, spc bool) *TextBufEdit {
	curli, _ := tb.LineIndent(ln, tabSz)
	if n > curli {
		return tb.InsertText(TextPos{Ln: ln}, IndentBytes(n-curli, tabSz, spc), true, true)
	} else if n < curli {
		spos := IndentCharPos(n, tabSz, spc)
		cpos := IndentCharPos(curli, tabSz, spc)
		return tb.DeleteText(TextPos{Ln: ln, Ch: spos}, TextPos{Ln: ln, Ch: cpos}, true, true)
	}
	return nil
}

// PrevLineIndent returns previous line from given line that has indentation -- skips blank lines
func (tb *TextBuf) PrevLineIndent(ln int, tabSz int) (n int, spc bool, txt string) {
	ln--
	for ln >= 0 {
		if len(tb.Lines[ln]) == 0 {
			ln--
			continue
		}
		n, spc = tb.LineIndent(ln, tabSz)
		txt = strings.TrimSpace(string(tb.Lines[ln]))
		return
	}
	return 0, false, ""
}

// AutoIndent indents given line to the level of the prior line, adjusted
// appropriately if the current line starts with one of the given un-indent
// strings, or the prior line ends with one of the given indent strings.  Will
// have to be replaced with a smarter parsing-based mechanism for indent /
// unindent but this will do for now.  Returns any edit that took place (could
// be nil), along with the auto-indented level and character position for the
// indent of the current line.
func (tb *TextBuf) AutoIndent(ln int, spc bool, tabSz int, indents, unindents []string) (tbe *TextBufEdit, indLev, chPos int) {
	li, _, prvln := tb.PrevLineIndent(ln, tabSz)
	curln := strings.TrimSpace(string(tb.Lines[ln]))
	ind := false
	und := false
	for _, us := range unindents {
		if curln == us {
			und = true
			break
		}
	}
	if !und && prvln != "" { // unindent overrides indent
		for _, is := range indents {
			if strings.HasSuffix(prvln, is) {
				ind = true
				break
			}
		}
	}
	switch {
	case ind:
		return tb.IndentLine(ln, li+1, tabSz, spc), li + 1, IndentCharPos(li+1, tabSz, spc)
	case und:
		return tb.IndentLine(ln, li-1, tabSz, spc), li - 1, IndentCharPos(li-1, tabSz, spc)
	default:
		return tb.IndentLine(ln, li, tabSz, spc), li, IndentCharPos(li, tabSz, spc)
	}
}

////////////////////////////////////////////////////////////////////////////
//   Diffs

// TextDiffs are raw differences between text, in terms of lines, reporting a
// sequence of operations that would convert one buffer (a) into the other
// buffer (b).  Each operation is either an 'r' (replace), 'd' (delete), 'i'
// (insert) or 'e' (equal).
type TextDiffs []difflib.OpCode

// DiffBufs computes the diff between this buffer and the other buffer,
// reporting a sequence of operations that would convert this buffer (a) into
// the other buffer (b).  Each operation is either an 'r' (replace), 'd'
// (delete), 'i' (insert) or 'e' (equal).  Everything is line-based (0, offset).
func (tb *TextBuf) DiffBufs(ob *TextBuf) TextDiffs {
	if tb.NLines == 0 || ob.NLines == 0 {
		return nil
	}
	astr := make([]string, tb.NLines)
	bstr := make([]string, ob.NLines)

	for ai, al := range tb.Lines {
		astr[ai] = string(al)
	}
	for bi, bl := range ob.Lines {
		bstr[bi] = string(bl)
	}

	m := difflib.NewMatcherWithJunk(astr, bstr, false, nil) // no junk
	return m.GetOpCodes()
}

// DiffBufsUnified computes the diff between this buffer and the other buffer,
// returning a unified diff with given amount of context (default of 3 will be
// used if -1)
func (tb *TextBuf) DiffBufsUnified(ob *TextBuf, context int) []byte {
	if tb.NLines == 0 || ob.NLines == 0 {
		return nil
	}
	astr := make([]string, tb.NLines)
	bstr := make([]string, ob.NLines)

	for ai, al := range tb.Lines {
		astr[ai] = string(al)
	}
	for bi, bl := range ob.Lines {
		bstr[bi] = string(bl)
	}

	ud := difflib.UnifiedDiff{A: astr, FromFile: string(tb.Filename), FromDate: tb.Info.ModTime.String(),
		B: bstr, ToFile: string(ob.Filename), ToDate: ob.Info.ModTime.String(), Context: context}
	var buf bytes.Buffer
	difflib.WriteUnifiedDiff(&buf, ud)
	return buf.Bytes()
}

// PatchFromBuf patches (edits) this buffer using content from other buffer,
// according to diff operations (e.g., as generated from DiffBufs).  signal
// determines whether each patch is signaled -- if an overall signal will be
// sent at the end, then that would not be necessary (typical)
func (tb *TextBuf) PatchFromBuf(ob *TextBuf, diffs TextDiffs, signal bool) bool {
	mods := false
	for _, df := range diffs {
		switch df.Tag {
		case 'r':
			tb.DeleteText(TextPos{Ln: df.I1}, TextPos{Ln: df.I2}, false, signal)
			ot := ob.Region(TextPos{Ln: df.J1}, TextPos{Ln: df.J2})
			tb.InsertText(TextPos{Ln: df.I1}, ot.ToBytes(), false, signal)
			mods = true
		case 'd':
			tb.DeleteText(TextPos{Ln: df.I1}, TextPos{Ln: df.I2}, false, signal)
			mods = true
		case 'i':
			ot := ob.Region(TextPos{Ln: df.J1}, TextPos{Ln: df.J2})
			tb.InsertText(TextPos{Ln: df.I1}, ot.ToBytes(), false, signal)
			mods = true
		}
	}
	return mods
}

////////////////////////////////////////////////////////////////////////////
//   TextBufList, TextBufs

// TextBufList is a list of text buffers, as a ki.Node, with buffers as children
type TextBufList struct {
	ki.Node
}

// New returns a new TextBuf buffer
func (tl *TextBufList) New() *TextBuf {
	tb := tl.AddNewChild(KiT_TextBuf, "newbuf").(*TextBuf)
	return tb
}

// TextBufs is the default list of TextBuf buffers for open texts
var TextBufs TextBufList

func init() {
	TextBufs.InitName(&TextBufs, "giv-text-bufs")
}

func NewTextBuf() *TextBuf {
	return TextBufs.New()
}
