// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package giv

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"mime"
	"os"
	"path/filepath"
	"strings"

	"github.com/goki/gi"
	"github.com/goki/ki"
	"github.com/goki/ki/kit"
)

// TextBuf is a buffer of text, which can be viewed by TextView(s).  It just
// holds the raw text lines (in original string and rune formats), and sends
// signals for making edits to the text and coordinating those edits across
// multiple views.  Views always only view a single buffer, so they directly
// call methods on the buffer to drive updates, which are then broadast.  It
// also has methods for loading and saving buffers to files.  Unlike GUI
// Widgets, its methods are generally signaling, without an explicit Action
// suffix.  Internally, the buffer represents new lines using \n = LF, but
// saving and loading can deal with Windows/DOS CRLF format.
type TextBuf struct {
	ki.Node
	Txt        []byte         `json:"-" xml:"text" desc:"the current value of the entire text being edited -- using []byte slice for greater efficiency"`
	HiLang     string         `desc:"language for syntax highlighting the code"`
	Autosave   bool           `desc:"if true, auto-save file after changes (in a separate routine)"`
	Changed    bool           `json:"-" xml:"-" desc:"true if the text has been changed (edited) relative to the original, since last save"`
	Filename   gi.FileName    `json:"-" xml:"-" desc:"filename of file last loaded or saved"`
	Mimetype   string         `json:"-" xml:"-" desc:"mime type of the contents"`
	NLines     int            `json:"-" xml:"-" desc:"number of lines"`
	Lines      [][]rune       `json:"-" xml:"-" desc:"the live lines of text being edited, with latest modifications -- encoded as runes per line"`
	ByteOffs   []int          `json:"-" xml:"-" desc:"offset for start of each line in Txt []byte slice -- to enable more efficient updating of that via edits"`
	TextBufSig ki.Signal      `json:"-" xml:"-" view:"-" desc:"signal for buffer -- see TextBufSignals for the types"`
	Views      []*TextView    `json:"-" xml:"-" desc:"the TextViews that are currently viewing this buffer"`
	Undos      []*TextBufEdit `json:"-" xml:"-" desc:"undo stack of edits"`
	UndoPos    int            `json:"-" xml:"-" desc:"undo position"`
}

var KiT_TextBuf = kit.Types.AddType(&TextBuf{}, TextBufProps)

var TextBufProps = ki.Props{}

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
	tb.Lines = make([][]rune, nlines)
	if nlines == 1 {
		tb.Lines[0] = []rune("")
	}
	if cap(tb.ByteOffs) >= nlines {
		tb.ByteOffs = tb.ByteOffs[:nlines]
	} else {
		tb.ByteOffs = make([]int, nlines)
	}
	tb.NLines = nlines
	tb.Refresh()
}

// Open loads text from a file into the buffer
func (tb *TextBuf) Open(filename gi.FileName) error {
	fp, err := os.Open(string(filename))
	if err != nil {
		gi.PromptDialog(nil, gi.DlgOpts{Title: "File Not Found", Prompt: err.Error()}, true, false, nil, nil)
		log.Println(err)
		return err
	}
	tb.Txt, err = ioutil.ReadAll(fp)
	fp.Close()
	tb.Filename = filename
	tb.SetName(string(filename)) // todo: modify in any way?
	tb.SetMimetype(string(filename))
	tb.BytesToLines()
	return nil
}

// ReOpen re-opens text from current file, if filename set -- returns false if not
func (tb *TextBuf) ReOpen() bool {
	tb.AutoSaveDelete() // justin case
	if tb.Filename == "" {
		return false
	}
	err := tb.Open(tb.Filename)
	if err != nil {
		return false
	}
	return true
}

// SaveAs saves the current text into given file -- does an EditDone first to save edits
func (tb *TextBuf) SaveAs(filename gi.FileName) error {
	tb.EditDone()
	err := ioutil.WriteFile(string(filename), tb.Txt, 0644)
	if err != nil {
		gi.PromptDialog(nil, gi.DlgOpts{Title: "Could not Save to File", Prompt: err.Error()}, true, false, nil, nil)
		log.Println(err)
	}
	tb.Filename = filename
	tb.SetName(string(filename)) // todo: modify in any way?
	tb.SetMimetype(string(filename))
	return err
}

// Save saves the current text into current Filename associated with this
// buffer
func (tb *TextBuf) Save() error {
	if tb.Filename == "" {
		return fmt.Errorf("giv.TextBuf: filename is empty for Save")
	}
	return tb.SaveAs(tb.Filename)
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

// LinesToBytes converts current Lines back to the Txt slice of bytes
func (tb *TextBuf) LinesToBytes() {
	if tb.Txt != nil {
		tb.Txt = tb.Txt[:0]
	} else {
		tb.Txt = make([]byte, 0, tb.NLines*40)
	}
	if cap(tb.ByteOffs) >= tb.NLines {
		tb.ByteOffs = tb.ByteOffs[:tb.NLines]
	} else {
		tb.ByteOffs = make([]int, tb.NLines)
	}
	bo := 0
	for ln, lr := range tb.Lines {
		tb.ByteOffs[ln] = bo
		tb.Txt = append(tb.Txt, []byte(string(lr))...)
		tb.Txt = append(tb.Txt, '\n')
		bo += len(lr) + 1
	}
}

// LinesToBytesCopy converts current Lines into a separate text byte copy --
// e.g., for autosave or other "offline" uses of the text -- doesn't affect
// byte offsets etc
func (tb *TextBuf) LinesToBytesCopy() []byte {
	b := make([]byte, 0, tb.NLines*40)
	for _, lr := range tb.Lines {
		b = append(b, []byte(string(lr))...)
		b = append(b, '\n')
	}
	return b
}

// BytesToLines converts current Txt bytes into lines, and signals that new text is available
func (tb *TextBuf) BytesToLines() {
	if len(tb.Txt) == 0 {
		tb.New(1)
		return
	}
	lns := bytes.Split(tb.Txt, []byte("\n"))
	tb.NLines = len(lns)
	tb.New(tb.NLines)
	bo := 0
	for ln, txt := range lns {
		tb.ByteOffs[ln] = bo
		tb.Lines[ln] = bytes.Runes(txt)
		bo += len(txt) + 1 // lf
	}
	tb.TextBufSig.Emit(tb.This, int64(TextBufNew), tb.Txt)
}

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

// SetMimetype sets the Mimetype and HiLang based on the given filename
func (tb *TextBuf) SetMimetype(filename string) {
	// todo: use chroma too
	ext := strings.ToLower(filepath.Ext(filename))
	strings.TrimSuffix(ext, "#") // autosave
	tb.Mimetype = mime.TypeByExtension(ext)
	if hl, ok := ExtToHiLangMap[ext]; ok {
		tb.HiLang = hl
		// fmt.Printf("set language to: %v for extension: %v\n", hl, ext)
	} else if strings.HasSuffix(filename, "Makefile") {
		tb.HiLang = "Makefile"
	} else {
		fmt.Printf("failed to set language for extension: %v\n", ext)
	}
	// else try something else..
}

//////////////////////////////////////////////////////////////////////////////////////
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

//////////////////////////////////////////////////////////////////////////////////////
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
	case lidx >= 0:
		fmt.Sscanf(link, "L%d", &tp.Ln)
		tp.Ln-- // link is 1-based, we use 0-based
	case cidx >= 0:
		fmt.Sscanf(link, "C%d", &tp.Ch)
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

//////////////////////////////////////////////////////////////////////////////////////
//   Edits

func (tb *TextBuf) SaveUndo(tbe *TextBufEdit) {
	if tb.UndoPos < len(tb.Undos) {
		tb.Undos = tb.Undos[:tb.UndoPos]
	}
	tb.Undos = append(tb.Undos, tbe)
	tb.UndoPos = len(tb.Undos)
}

// DeleteText deletes region of text between start and end positions, signaling
// views after text lines have been updated.
func (tb *TextBuf) DeleteText(st, ed TextPos, saveUndo bool) *TextBufEdit {
	for ed.Ln >= len(tb.Lines) {
		ed.Ln--
	}
	if st == ed {
		return nil
	}
	if !st.IsLess(ed) {
		log.Printf("giv.TextBuf DeleteText: starting position must be less than ending!: st: %v, ed: %v\n", st, ed)
		return nil
	}
	tb.Changed = true
	tbe := tb.Region(st, ed)
	tbe.Delete = true
	if ed.Ln == st.Ln {
		tb.Lines[st.Ln] = append(tb.Lines[st.Ln][:st.Ch], tb.Lines[st.Ln][ed.Ch:]...)
		// no lines to bytes for single-line ops
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
		tb.LinesToBytes()
	}
	tb.TextBufSig.Emit(tb.This, int64(TextBufDelete), tbe)
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
func (tb *TextBuf) InsertText(st TextPos, text []byte, saveUndo bool) *TextBufEdit {
	if len(text) == 0 {
		return nil
	}
	if len(tb.Lines) == 0 {
		tb.New(1)
	}
	tb.Changed = true
	lns := bytes.Split(text, []byte("\n"))
	sz := len(lns)
	rs := bytes.Runes(lns[0])
	rsz := len(rs)
	ed := st
	if sz == 1 {
		nt := append(tb.Lines[st.Ln], rs...) // first append to end to extend capacity
		copy(nt[st.Ch+rsz:], nt[st.Ch:])     // move stuff to end
		copy(nt[st.Ch:], rs)                 // copy into position
		tb.Lines[st.Ln] = nt
		ed.Ch += rsz
		// no lines to bytes
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
		tb.LinesToBytes()
	}
	tbe := tb.Region(st, ed)
	tb.TextBufSig.Emit(tb.This, int64(TextBufInsert), tbe)
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
			tbe.Text[0] = make([]rune, sz)
			copy(tbe.Text[0][0:sz], tb.Lines[st.Ln][st.Ch:])
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
		tb.InsertText(tbe.Reg.Start, tbe.ToBytes(), false)
	} else {
		// fmt.Printf("undoing insert at: %v text: %v\n", tbe.Reg, string(tbe.ToBytes()))
		tb.DeleteText(tbe.Reg.Start, tbe.Reg.End, false)
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
		tb.DeleteText(tbe.Reg.Start, tbe.Reg.End, false)
	} else {
		tb.InsertText(tbe.Reg.Start, tbe.ToBytes(), false)
	}
	tb.UndoPos++
	return tbe
}

// EndPos returns the ending position at end of buffer
func (tb *TextBuf) EndPos() TextPos {
	if tb.NLines == 0 {
		return TextPosZero
	}
	ed := TextPos{tb.NLines - 1, len(tb.Lines[tb.NLines-1])}
	return ed
}

// AppendText appends new text to end of buffer, using insert, returns edit
func (tb *TextBuf) AppendText(text []byte) *TextBufEdit {
	ed := tb.EndPos()
	return tb.InsertText(ed, text, true)
}

// AppendTextLine appends one line of new text to end of buffer, using insert, returns edit
func (tb *TextBuf) AppendTextLine(text []byte) *TextBufEdit {
	ed := tb.EndPos()
	sz := len(text)
	tcpy := make([]byte, sz+1)
	copy(tcpy, text)
	tcpy[sz] = '\n'
	return tb.InsertText(ed, tcpy, true)
}

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
		return tb.InsertText(TextPos{Ln: ln}, IndentBytes(n-curli, tabSz, spc), true)
	} else if n < curli {
		spos := IndentCharPos(n, tabSz, spc)
		cpos := IndentCharPos(curli, tabSz, spc)
		return tb.DeleteText(TextPos{Ln: ln, Ch: spos}, TextPos{Ln: ln, Ch: cpos}, true)
	}
	return nil
}

// AutoIndent indents given line to the level of the prior line, adjusted
// appropriately if the current line starts with one of the given un-indent
// strings, or the prior line ends with one of the given indent strings.  Will
// have to be replaced with a smarter parsing-based mechanism for indent /
// unindent but this will do for now.  Returns any edit that took place (could
// be nil), along with the auto-indented level and character position for the
// indent of the current line.
func (tb *TextBuf) AutoIndent(ln int, spc bool, tabSz int, indents, unindents []string) (tbe *TextBufEdit, indLev, chPos int) {
	prvln := ""
	li := 0
	if ln > 0 {
		prvln = strings.TrimSpace(string(tb.Lines[ln-1]))
		li, _ = tb.LineIndent(ln-1, tabSz)
	}
	ind := false
	und := false
	if prvln != "" {
		for _, us := range unindents {
			if strings.HasPrefix(prvln, us) {
				und = true
				break
			}
		}
		if !und {
			for _, is := range indents {
				if strings.HasSuffix(prvln, is) {
					ind = true
					break
				}
			}
		}
	}
	switch {
	case ind:
		return tb.IndentLine(ln, li+1, tabSz, spc), li + 1, IndentCharPos(li+1, tabSz, spc)
	case und: // operates on previous line and current one
		if li-1 > 0 {
			if ln > 0 {
				tb.IndentLine(ln-1, li-1, tabSz, spc)
			}
			return tb.IndentLine(ln, li-1, tabSz, spc), li - 1, IndentCharPos(li-1, tabSz, spc)
		}
		return nil, 0, 0
	default:
		return tb.IndentLine(ln, li, tabSz, spc), li, IndentCharPos(li, tabSz, spc)
	}
}

//////////////////////////////////////////////////////////////////////////////////////
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

///////////////////////////////////////////////////////////////////////////////
//  extension to highighting style map

var ExtToHiLangMap = map[string]string{
	".go":    "Go",
	".md":    "markdown",
	".css":   "CSS",
	".html":  "HTML",
	".htm":   "HTML",
	".tex":   "TeX",
	".cpp":   "C++",
	".c":     "C",
	".h":     "C++",
	".sh":    "Bash",
	".plist": "XML",
	".svg":   "XML",
}
