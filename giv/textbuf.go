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

	"github.com/goki/gi"
	"github.com/goki/ki"
	"github.com/goki/ki/kit"
)

// TextPos represents line, character positions within the TextBuf and TextView
type TextPos struct {
	Ln, Ch int
}

// TextBufEdit describes an edit action to a buffer -- this is the data passed
// via signals to viewers of the buffer.  Actions are only deletions and
// insertions (a change is a sequence of those, given normal editing
// processes).  The TextBuf always reflects the current state *after* the
// edit.
type TextBufEdit struct {
	Start  TextPos `desc:"starting position for the edit (always same for previous and current)"`
	End    TextPos `desc:"ending position for the edit, in original lines data for a delete, and in new lines data for an insert"`
	Text   []byte  `desc:"text that was deleted or inserted"`
	Delete bool    `desc:"action is either a deletion or an insertion"`
}

// TextBuf is a buffer of text, which can be viewed by TextView(s).  It just
// holds the raw text lines (in original string and rune formats), and sends
// and receives signals for making edits to the text and coordinating those
// edits across multiple views.  It also has methods for loading and saving
// buffers to files.  Unlike GUI Widgets, all of its methods are generally
// signaling, without an explicit Action suffix.
type TextBuf struct {
	ki.Node
	Txt        []byte      `json:"-" xml:"text" desc:"the last saved value of the entire text being edited -- using []byte slice for greater efficiency"`
	Edited     bool        `json:"-" xml:"-" desc:"true if the text has been edited relative to the original"`
	Filename   gi.FileName `json:"-" xml:"-" desc:"filename of file last loaded or saved"`
	Mimetype   string      `json:"-" xml:"-" desc:"mime type of the contents"`
	NLines     int         `json:"-" xml:"-" desc:"number of lines"`
	Lines      [][]rune    `json:"-" xml:"-" desc:"the live lines of text being edited, with latest modifications -- encoded as runes per line"`
	TextBufSig ki.Signal   `json:"-" xml:"-" view:"-" desc:"signal for buffer -- see TextBufSignals for the types"`
	Views      []*TextView `json:"-" xml:"-" desc:"the TextViews that are currently viewing this buffer"`
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

// EditDone completes editing and copies the active edited text to the text
func (tb *TextBuf) EditDone() {
	if tb.Edited {
		tb.Edited = false
		tb.Txt = tb.LinesToBytes()
		tb.TextBufSig.Emit(tb.This, int64(TextBufDone), tb.Txt)
	}
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
	tb.BytesToLines(tb.Txt)
	return nil
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

// SetMimetype sets the Mimetype based on the given filename
func (tb *TextBuf) SetMimetype(filename string) {
	ext := filepath.Ext(filename)
	tb.Mimetype = mime.TypeByExtension(ext)
}

// LinesToBytes converts current Lines to a slice of bytes
func (tb *TextBuf) LinesToBytes() []byte {
	b := make([]byte, 0, tb.NLines*80)
	for _, lr := range tb.Lines {
		b = append(b, []byte(string(lr))...)
	}
	return b
}

// BytesToLines converts given text bytes into lines, and signals that new text is available
func (tb *TextBuf) BytesToLines(text []byte) {
	if len(text) == 0 {
		tb.NLines = 0
		if tb.Lines != nil {
			tb.Lines = tb.Lines[0:0]
		}
		return
	}
	lns := bytes.Split(text, []byte("\n")) // todo: other cr?
	tb.NLines = len(lns)
	if cap(tb.Lines) >= tb.NLines {
		tb.Lines = tb.Lines[0:0]
	} else {
		tb.Lines = make([][]rune, tb.NLines)
	}
	for ln, txt := range lns {
		tb.Lines[ln] = []rune(string(txt))
	}
	tb.TextBufSig.Emit(tb.This, int64(TextBufNew), tb.Txt)
}

// AddView adds a viewer of this buffer -- connects our signals to the viewer
func (tb *TextBuf) AddView(vw *TextView) {
	tb.Views = append(tb.Views, vw)
	tb.TextBufSig.Connect(vw.This, TextViewBufSigRecv)
	vw.TextViewSig.Connect(tb.This, TextBufViewSigRecv)
}

// TextBufViewSigRecv receives a signal from the view and updates buffer accordingly
func TextBufViewSigRecv(rbufki, svwki ki.Ki, sig int64, data interface{}) {
	// todo: view signals
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
