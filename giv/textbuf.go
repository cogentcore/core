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
	"regexp"
	"sync"
	"time"

	"github.com/goki/gi/gi"
	"github.com/goki/gi/gist"
	"github.com/goki/gi/giv/textbuf"
	"github.com/goki/gi/histyle"
	"github.com/goki/gi/units"
	"github.com/goki/ki/indent"
	"github.com/goki/ki/ints"
	"github.com/goki/ki/ki"
	"github.com/goki/ki/kit"
	"github.com/goki/ki/runes"
	"github.com/goki/pi/complete"
	"github.com/goki/pi/filecat"
	"github.com/goki/pi/lex"
	"github.com/goki/pi/pi"
	"github.com/goki/pi/spell"
	"github.com/goki/pi/token"
)

// TextBufMaxScopeLines is the maximum lines to search for a scope marker, e.g. '}'
var TextBufMaxScopeLines = 100

// TextBufDiffRevertLines is max number of lines to use the
// diff-based revert, which results in faster reverts but only
// if the file isn't too big..
var TextBufDiffRevertLines = 10000

// TextBufDiffRevertDiffs is max number of difference regions
// to apply for diff-based revert otherwise just reopens file
var TextBufDiffRevertDiffs = 20

// TextBufMarkupDelayMSec is the number of milliseconds to wait
// before starting a new background markup process, after
// text is entered in the line
var TextBufMarkupDelayMSec = 1000

// TextBuf is a buffer of text, which can be viewed by TextView(s).  It holds
// the raw text lines (in original string and rune formats, and marked-up from
// syntax highlighting), and sends signals for making edits to the text and
// coordinating those edits across multiple views.  Views always only view a
// single buffer, so they directly call methods on the buffer to drive
// updates, which are then broadcast.  It also has methods for loading and
// saving buffers to files.  Unlike GUI Widgets, its methods are generally
// signaling, without an explicit Action suffix.  Internally, the buffer
// represents new lines using \n = LF, but saving and loading can deal with
// Windows/DOS CRLF format.
type TextBuf struct {
	ki.Node
	Txt              []byte              `json:"-" xml:"text" desc:"the current value of the entire text being edited -- using []byte slice for greater efficiency"`
	Autosave         bool                `desc:"if true, auto-save file after changes (in a separate routine)"`
	Opts             textbuf.Opts        `desc:"options for how text editing / viewing works"`
	Filename         gi.FileName         `json:"-" xml:"-" desc:"filename of file last loaded or saved"`
	Info             FileInfo            `desc:"full info about file"`
	PiState          pi.FileStates       `desc:"Pi parsing state info for file"`
	Hi               HiMarkup            `desc:"syntax highlighting markup parameters (language, style, etc)"`
	NLines           int                 `json:"-" xml:"-" desc:"number of lines"`
	LineIcons        map[int]string      `desc:"icons for given lines -- use SetLineIcon and DeleteLineIcon"`
	LineColors       map[int]gist.Color  `desc:"special line number colors given lines -- use SetLineColor and DeleteLineColor"`
	Icons            map[string]*gi.Icon `json:"-" xml:"-" desc:"icons for each LineIcons being used"`
	Lines            [][]rune            `json:"-" xml:"-" desc:"the live lines of text being edited, with latest modifications -- encoded as runes per line, which is necessary for one-to-one rune / glyph rendering correspondence -- all TextPos positions etc are in *rune* indexes, not byte indexes!"`
	LineBytes        [][]byte            `json:"-" xml:"-" desc:"the live lines of text being edited, with latest modifications -- encoded in bytes per line translated from Lines, and used for input to markup -- essential to use Lines and not LineBytes when dealing with TextPos positions, which are in runes"`
	Tags             []lex.Line          `json:"extra custom tagged regions for each line"`
	HiTags           []lex.Line          `json:"syntax highlighting tags -- auto-generated"`
	Markup           [][]byte            `json:"-" xml:"-" desc:"marked-up version of the edit text lines, after being run through the syntax highlighting process etc -- this is what is actually rendered"`
	MarkupEdits      []*textbuf.Edit     `json:"-" xml:"-" desc:"edits that have been made since last full markup"`
	ByteOffs         []int               `json:"-" xml:"-" desc:"offsets for start of each line in Txt []byte slice -- this is NOT updated with edits -- call SetByteOffs to set it when needed -- used for re-generating the Txt in LinesToBytes, and set on initial open in BytesToLines"`
	TotalBytes       int                 `json:"-" xml:"-" desc:"total bytes in document -- see ByteOffs for when it is updated"`
	LinesMu          sync.RWMutex        `json:"-" xml:"-" desc:"mutex for updating lines"`
	MarkupMu         sync.RWMutex        `json:"-" xml:"-" desc:"mutex for updating markup"`
	MarkupDelayTimer *time.Timer         `json:"-" xml:"-" desc:"markup delay timer"`
	MarkupDelayMu    sync.Mutex          `json:"-" xml:"-" desc:"mutex for updating markup delay timer"`
	TextBufSig       ki.Signal           `json:"-" xml:"-" view:"-" desc:"signal for buffer -- see TextBufSignals for the types"`
	Views            []*TextView         `json:"-" xml:"-" desc:"the TextViews that are currently viewing this buffer"`
	Undos            textbuf.Undo        `json:"-" xml:"-" desc:"undo manager"`
	PosHistory       []lex.Pos           `json:"-" xml:"-" desc:"history of cursor positions -- can move back through them"`
	Complete         *gi.Complete        `json:"-" xml:"-" desc:"functions and data for text completion"`
	Spell            *gi.Spell           `json:"-" xml:"-" desc:"functions and data for spelling correction"`
	CurView          *TextView           `json:"-" xml:"-" desc:"current textview -- e.g., the one that initiated Complete or Correct process -- update cursor position in this view -- is reset to nil after usage always"`
}

var KiT_TextBuf = kit.Types.AddType(&TextBuf{}, TextBufProps)

func (tb *TextBuf) Disconnect() {
	tb.Node.Disconnect()
	tb.TextBufSig.DisconnectAll()
	tb.DeleteSpell()
	tb.DeleteCompleter()
}

var TextBufProps = ki.Props{
	"EnumType:Flag": KiT_TextBufFlags,
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
	// textbuf.Edit describing change -- the TextBuf always reflects the
	// current state *after* the edit.
	TextBufInsert

	// TextBufDelete signals that some text was deleted -- data is
	// textbuf.Edit describing change -- the TextBuf always reflects the
	// current state *after* the edit.
	TextBufDelete

	// TextBufMarkUpdt signals that the Markup text has been updated -- this
	// signal is typically sent from a separate goroutine so should be used
	// with a mutex
	TextBufMarkUpdt

	// TextBufClosed signals that the textbuf was closed
	TextBufClosed

	TextBufSignalsN
)

//go:generate stringer -type=TextBufSignals

// TextBufFlags extend NodeBase NodeFlags to hold TextBuf state
type TextBufFlags int

//go:generate stringer -type=TextBufFlags

var KiT_TextBufFlags = kit.Enums.AddEnumExt(gi.KiT_NodeFlags, TextBufFlagsN, kit.BitFlag, nil)

const (
	// TextBufAutoSaving is used in atomically safe way to protect autosaving
	TextBufAutoSaving TextBufFlags = TextBufFlags(gi.NodeFlagsN) + iota

	// TextBufMarkingUp indicates current markup operation in progress -- don't redo
	TextBufMarkingUp

	// TextBufChanged indicates if the text has been changed (edited) relative to the
	// original, since last save
	TextBufChanged

	// TextBufFileModOk have already asked about fact that file has changed since being
	// opened, user is ok
	TextBufFileModOk

	TextBufFlagsN
)

// IsChanged indicates if the text has been changed (edited) relative to
// the original, since last save
func (tb *TextBuf) IsChanged() bool {
	return tb.HasFlag(int(TextBufChanged))
}

// SetChanged marks buffer as changed
func (tb *TextBuf) SetChanged() {
	tb.SetFlag(int(TextBufChanged))
}

// ClearChanged marks buffer as un-changed
func (tb *TextBuf) ClearChanged() {
	tb.ClearFlag(int(TextBufChanged))
}

// SetText sets the text to given bytes
func (tb *TextBuf) SetText(txt []byte) {
	tb.Defaults()
	tb.Txt = txt
	tb.BytesToLines()
	tb.InitialMarkup()
	tb.Refresh()
	tb.ReMarkup()
}

// SetTextLines sets the text to given lines of bytes
// if cpy is true, make a copy of bytes -- otherwise use
func (tb *TextBuf) SetTextLines(lns [][]byte, cpy bool) {
	tb.Defaults()
	tb.LinesMu.Lock()
	tb.NLines = len(lns)
	tb.LinesMu.Unlock()
	tb.New(tb.NLines)
	tb.LinesMu.Lock()
	bo := 0
	for ln, txt := range lns {
		tb.ByteOffs[ln] = bo
		tb.Lines[ln] = bytes.Runes(txt)
		if cpy {
			tb.LineBytes[ln] = make([]byte, len(txt))
			copy(tb.LineBytes[ln], txt)
		} else {
			tb.LineBytes[ln] = txt
		}
		tb.Markup[ln] = HTMLEscapeRunes(tb.Lines[ln])
		bo += len(txt) + 1 // lf
	}
	tb.TotalBytes = bo
	tb.LinesMu.Unlock()
	tb.LinesToBytes()
	tb.InitialMarkup()
	tb.Refresh()
	tb.ReMarkup()
}

// EditDone finalizes any current editing, sends signal
func (tb *TextBuf) EditDone() {
	tb.AutoSaveDelete()
	tb.ClearChanged()
	tb.LinesToBytes()
	tb.TextBufSig.Emit(tb.This(), int64(TextBufDone), tb.Txt)
}

// Text returns the current text as a []byte array, applying all current
// changes -- calls EditDone and will generate that signal if there have been
// changes
func (tb *TextBuf) Text() []byte {
	tb.EditDone()
	return tb.Txt
}

// NumLines is the concurrent-safe accessor to NLines
func (tb *TextBuf) NumLines() int {
	tb.LinesMu.RLock()
	defer tb.LinesMu.RUnlock()
	return tb.NLines
}

// IsValidLine returns true if given line is in range
func (tb *TextBuf) IsValidLine(ln int) bool {
	if ln < 0 {
		return false
	}
	nln := tb.NumLines()
	if ln >= nln {
		return false
	}
	return true
}

// Line is the concurrent-safe accessor to specific Line of Lines runes
func (tb *TextBuf) Line(ln int) []rune {
	tb.LinesMu.RLock()
	defer tb.LinesMu.RUnlock()
	if ln >= tb.NLines || ln < 0 {
		return nil
	}
	return tb.Lines[ln]
}

// LineLen is the concurrent-safe accessor to length of specific Line of Lines runes
func (tb *TextBuf) LineLen(ln int) int {
	tb.LinesMu.RLock()
	defer tb.LinesMu.RUnlock()
	if ln >= tb.NLines || ln < 0 {
		return 0
	}
	return len(tb.Lines[ln])
}

// BytesLine is the concurrent-safe accessor to specific Line of LineBytes
func (tb *TextBuf) BytesLine(ln int) []byte {
	tb.LinesMu.RLock()
	defer tb.LinesMu.RUnlock()
	if ln >= tb.NLines || ln < 0 {
		return nil
	}
	return tb.LineBytes[ln]
}

// SetHiStyle sets the highlighting style -- needs to be protected by mutex
func (tb *TextBuf) SetHiStyle(style gi.HiStyleName) {
	tb.MarkupMu.Lock()
	tb.Hi.SetHiStyle(style)
	tb.MarkupMu.Unlock()
}

// Defaults sets default parameters if they haven't been yet --
// if Hi.Style is empty, then it considers it to not have been set
func (tb *TextBuf) Defaults() {
	if tb.Hi.Style != "" {
		return
	}
	tb.SetHiStyle(histyle.StyleDefault)
	tb.Opts.EditorPrefs = gi.Prefs.Editor
}

// Refresh signals any views to refresh views
func (tb *TextBuf) Refresh() {
	tb.TextBufSig.Emit(tb.This(), int64(TextBufNew), tb.Txt)
}

// SetInactive sets the buffer in an inactive state if inactive = true
// otherwise is in active state.  Inactive = don't save Undos.
func (tb *TextBuf) SetInactive(inactive bool) {
	tb.Undos.Off = inactive
}

// todo: use https://github.com/andybalholm/crlf to deal with cr/lf etc --
// internally just use lf = \n

// New initializes a new buffer with n blank lines
func (tb *TextBuf) New(nlines int) {
	tb.Defaults()
	nlines = ints.MaxInt(nlines, 1)
	tb.LinesMu.Lock()
	tb.MarkupMu.Lock()
	tb.Undos.Reset()
	tb.Lines = make([][]rune, nlines)
	tb.LineBytes = make([][]byte, nlines)
	tb.Tags = make([]lex.Line, nlines)
	tb.HiTags = make([]lex.Line, nlines)
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

	tb.PiState.SetSrc(string(tb.Filename), "", tb.Info.Sup)
	tb.Hi.Init(&tb.Info, &tb.PiState)

	tb.MarkupMu.Unlock()
	tb.LinesMu.Unlock()
	tb.Refresh()
}

// Stat gets info about the file, including highlighting language
func (tb *TextBuf) Stat() error {
	tb.ClearFlag(int(TextBufFileModOk))
	err := tb.Info.InitFile(string(tb.Filename))
	if err != nil {
		return err
	}
	tb.ConfigSupported()
	return nil
}

// ConfigSupported configures options based on the supported language info in GoPi
// returns true if supported
func (tb *TextBuf) ConfigSupported() bool {
	if tb.Info.Sup != filecat.NoSupport {
		if tb.Spell == nil {
			tb.SetSpell()
		}
		if tb.Complete == nil {
			tb.SetCompleter(&tb.PiState, CompletePi, CompleteEditPi, LookupPi)
		}
		return tb.Opts.ConfigSupported(tb.Info.Sup)
	}
	return false
}

// FileModCheck checks if the underlying file has been modified since last
// Stat (open, save) -- if haven't yet prompted, user is prompted to ensure
// that this is OK.  returns true if file was modified
func (tb *TextBuf) FileModCheck() bool {
	if tb.HasFlag(int(TextBufFileModOk)) {
		return false
	}
	info, err := os.Stat(string(tb.Filename))
	if err != nil {
		return false
	}
	if info.ModTime() != time.Time(tb.Info.ModTime) {
		vp := tb.ViewportFromView()
		gi.ChoiceDialog(vp, gi.DlgOpts{Title: "File Changed on Disk: " + DirAndFile(string(tb.Filename)),
			Prompt: fmt.Sprintf("File has changed on Disk since being opened or saved by you -- what do you want to do?  If you <code>Revert from Disk</code>, you will lose any existing edits in open buffer.  If you <code>Ignore and Proceed</code>, the next save will overwrite the changed file on disk, losing any changes there.  File: %v", tb.Filename)},
			[]string{"Save As to diff File", "Revert from Disk", "Ignore and Proceed"},
			tb.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
				switch sig {
				case 0:
					CallMethod(tb, "SaveAs", vp)
				case 1:
					tb.Revert()
				case 2:
					tb.SetFlag(int(TextBufFileModOk))
				}
			})
		return true
	}
	return false
}

// Open loads text from a file into the buffer
func (tb *TextBuf) Open(filename gi.FileName) error {
	tb.Defaults()
	err := tb.OpenFile(filename)
	if err != nil {
		vp := tb.ViewportFromView()
		gi.PromptDialog(vp, gi.DlgOpts{Title: "File could not be Opened", Prompt: err.Error()}, gi.AddOk, gi.NoCancel, nil, nil)
		log.Println(err)
		return err
	}
	tb.SetName(string(filename))

	tb.InitialMarkup()
	tb.Refresh()
	tb.ReMarkup()
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

// Revert re-opens text from current file, if filename set -- returns false if
// not -- uses an optimized diff-based update to preserve existing formatting
// -- very fast if not very different
func (tb *TextBuf) Revert() bool {
	tb.AutoSaveDelete() // justin case
	if tb.Filename == "" {
		return false
	}

	didDiff := false
	if tb.NLines < TextBufDiffRevertLines {
		ob := &TextBuf{}
		ob.InitName(ob, "revert-tmp")
		err := ob.OpenFile(tb.Filename)
		if err != nil {
			vp := tb.ViewportFromView()
			if vp != nil { // only if viewing
				gi.PromptDialog(vp, gi.DlgOpts{Title: "File could not be Re-Opened", Prompt: err.Error()}, gi.AddOk, gi.NoCancel, nil, nil)
			}
			log.Println(err)
			return false
		}
		tb.Stat() // "own" the new file..
		if ob.NLines < TextBufDiffRevertLines {
			diffs := tb.DiffBufs(ob)
			if len(diffs) < TextBufDiffRevertDiffs {
				tb.PatchFromBuf(ob, diffs, true) // true = send sigs for each update -- better than full, assuming changes are minor
				didDiff = true
			}
		}
	}
	if !didDiff {
		tb.OpenFile(tb.Filename)
	}
	tb.ClearChanged()
	tb.AutoSaveDelete()
	tb.Refresh()
	tb.ReMarkup()
	return true
}

// SaveAsFunc saves the current text into given file -- does an EditDone first to save edits
// and checks for an existing file -- if it does exist then prompts to overwrite or not.
// If afterFunc is non-nil, then it is called with the status of the user action.
func (tb *TextBuf) SaveAsFunc(filename gi.FileName, afterFunc func(canceled bool)) {
	// todo: filemodcheck!
	tb.EditDone()
	if _, err := os.Stat(string(filename)); os.IsNotExist(err) {
		tb.SaveFile(filename)
		if afterFunc != nil {
			afterFunc(false)
		}
	} else {
		vp := tb.ViewportFromView()
		gi.ChoiceDialog(vp, gi.DlgOpts{Title: "File Exists, Overwrite?",
			Prompt: fmt.Sprintf("File already exists, overwrite?  File: %v", filename)},
			[]string{"Cancel", "Overwrite"},
			tb.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
				cancel := false
				switch sig {
				case 0:
					cancel = true
				case 1:
					tb.SaveFile(filename)
				}
				if afterFunc != nil {
					afterFunc(cancel)
				}
			})
	}
}

// SaveAs saves the current text into given file -- does an EditDone first to save edits
// and checks for an existing file -- if it does exist then prompts to overwrite or not.
func (tb *TextBuf) SaveAs(filename gi.FileName) {
	tb.SaveAsFunc(filename, nil)
}

// SaveFile writes current buffer to file, with no prompting, etc
func (tb *TextBuf) SaveFile(filename gi.FileName) error {
	err := ioutil.WriteFile(string(filename), tb.Txt, 0644)
	if err != nil {
		gi.PromptDialog(nil, gi.DlgOpts{Title: "Could not Save to File", Prompt: err.Error()}, gi.AddOk, gi.NoCancel, nil, nil)
		log.Println(err)
	} else {
		tb.Filename = filename
		tb.SetName(string(filename))
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
			tb.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
				switch sig {
				case 0:
					CallMethod(tb, "SaveAs", vp)
				case 1:
					tb.Revert()
				case 2:
					tb.SaveFile(tb.Filename)
				}
			})
	}
	return tb.SaveFile(tb.Filename)
}

// Close closes the buffer -- prompts to save if changes, and disconnects from views
// if afterFun is non-nil, then it is called with the status of the user action
func (tb *TextBuf) Close(afterFun func(canceled bool)) bool {
	if tb.IsChanged() {
		vp := tb.ViewportFromView()
		if tb.Filename != "" {
			gi.ChoiceDialog(vp, gi.DlgOpts{Title: "Close Without Saving?",
				Prompt: fmt.Sprintf("Do you want to save your changes to file: %v?", tb.Filename)},
				[]string{"Save", "Close Without Saving", "Cancel"},
				tb.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
					switch sig {
					case 0:
						tb.Save()
						tb.Close(afterFun) // 2nd time through won't prompt
					case 1:
						tb.ClearChanged()
						tb.AutoSaveDelete()
						tb.Close(afterFun)
					case 2:
						if afterFun != nil {
							afterFun(true)
						}
					}
				})
		} else {
			gi.ChoiceDialog(vp, gi.DlgOpts{Title: "Close Without Saving?",
				Prompt: "Do you want to save your changes (no filename for this buffer yet)?  If so, Cancel and then do Save As"},
				[]string{"Close Without Saving", "Cancel"},
				tb.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
					switch sig {
					case 0:
						tb.ClearChanged()
						tb.AutoSaveDelete()
						tb.Close(afterFun)
					case 1:
						if afterFun != nil {
							afterFun(true)
						}
					}
				})
		}
		return false // awaiting decisions..
	}
	tb.TextBufSig.Emit(tb.This(), int64(TextBufClosed), nil)
	// for _, tve := range tb.Views {
	// 	tve.SetBuf(nil) // automatically disconnects signals, views
	// }
	tb.New(1)
	tb.Filename = ""
	tb.ClearChanged()
	if afterFun != nil {
		afterFun(false)
	}
	return true
}

////////////////////////////////////////////////////////////////////////////////////////
//		AutoSave

// AutoSaveOff turns off autosave and returns the prior state of Autosave flag --
// call AutoSaveRestore with rval when done -- good idea to turn autosave off
// for anything that does a block of updates
func (tb *TextBuf) AutoSaveOff() bool {
	asv := tb.Autosave
	tb.Autosave = false
	return asv
}

// AutoSaveRestore restores prior Autosave setting, from AutoSaveOff()
func (tb *TextBuf) AutoSaveRestore(asv bool) {
	tb.Autosave = asv
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
	if tb.HasFlag(int(TextBufAutoSaving)) {
		return nil
	}
	tb.SetFlag(int(TextBufAutoSaving))
	asfn := tb.AutoSaveFilename()
	b := tb.LinesToBytesCopy()
	err := ioutil.WriteFile(asfn, b, 0644)
	if err != nil {
		log.Printf("giv.TextBuf: Could not AutoSave file: %v, error: %v\n", asfn, err)
	}
	tb.ClearFlag(int(TextBufAutoSaving))
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
func (tb *TextBuf) EndPos() lex.Pos {
	tb.LinesMu.RLock()
	defer tb.LinesMu.RUnlock()

	if tb.NLines == 0 {
		return lex.PosZero
	}
	ed := lex.Pos{tb.NLines - 1, len(tb.Lines[tb.NLines-1])}
	return ed
}

// AppendText appends new text to end of buffer, using insert, returns edit
func (tb *TextBuf) AppendText(text []byte, signal bool) *textbuf.Edit {
	if len(text) == 0 {
		return &textbuf.Edit{}
	}
	ed := tb.EndPos()
	return tb.InsertText(ed, text, signal)
}

// AppendTextLine appends one line of new text to end of buffer, using insert,
// and appending a LF at the end of the line if it doesn't already have one.
// Returns the edit region.
func (tb *TextBuf) AppendTextLine(text []byte, signal bool) *textbuf.Edit {
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
	tbe := tb.InsertText(ed, efft, signal)
	return tbe
}

// AppendTextMarkup appends new text to end of buffer, using insert, returns
// edit, and uses supplied markup to render it
func (tb *TextBuf) AppendTextMarkup(text []byte, markup []byte, signal bool) *textbuf.Edit {
	if len(text) == 0 {
		return &textbuf.Edit{}
	}
	ed := tb.EndPos()
	tbe := tb.InsertText(ed, text, false) // no sig -- we do later

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
		tb.TextBufSig.Emit(tb.This(), int64(TextBufInsert), tbe)
	}
	return tbe
}

// AppendTextLineMarkup appends one line of new text to end of buffer, using
// insert, and appending a LF at the end of the line if it doesn't already
// have one.  user-supplied markup is used.  Returns the edit region.
func (tb *TextBuf) AppendTextLineMarkup(text []byte, markup []byte, signal bool) *textbuf.Edit {
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
	tbe := tb.InsertText(ed, efft, false)
	tb.Markup[tbe.Reg.Start.Ln] = markup
	if signal {
		tb.TextBufSig.Emit(tb.This(), int64(TextBufInsert), tbe)
	}
	return tbe
}

/////////////////////////////////////////////////////////////////////////////
//   Views

// AddView adds a viewer of this buffer -- connects our signals to the viewer
func (tb *TextBuf) AddView(vw *TextView) {
	tb.Views = append(tb.Views, vw)
	tb.TextBufSig.Connect(vw.This(), TextViewBufSigRecv)
}

// DeleteView removes given viewer from our buffer
func (tb *TextBuf) DeleteView(vw *TextView) {
	for i, tve := range tb.Views {
		if tve == vw {
			tb.Views = append(tb.Views[:i], tb.Views[i+1:]...)
			break
		}
	}
	tb.TextBufSig.Disconnect(vw.This())
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
		if tv != nil && tv.This() != nil {
			tv.CursorPos = tb.EndPos()
			tv.ScrollCursorInView()
		}
	}
}

// RefreshViews does a refresh draw on all views
func (tb *TextBuf) RefreshViews() {
	for _, tv := range tb.Views {
		if tv != nil && tv.This() != nil {
			tv.Refresh()
		}
	}
}

// BatchUpdateStart call this when starting a batch of updates to the buffer --
// it blocks the window updates for views until all the updates are done,
// and calls AutoSaveOff.  Calls UpdateStart on Buf too.
// Returns buf updt, win updt and autosave restore state.
// Must call BatchUpdateEnd at end with the result of this call.
func (tb *TextBuf) BatchUpdateStart() (bufUpdt, winUpdt, autoSave bool) {
	tb.Undos.NewGroup()
	bufUpdt = tb.UpdateStart()
	autoSave = tb.AutoSaveOff()
	winUpdt = false
	vp := tb.ViewportFromView()
	if vp == nil {
		return
	}
	winUpdt = vp.TopUpdateStart()
	return
}

// BatchUpdateEnd call to complete BatchUpdateStart
func (tb *TextBuf) BatchUpdateEnd(bufUpdt, winUpdt, autoSave bool) {
	tb.AutoSaveRestore(autoSave)
	if winUpdt {
		vp := tb.ViewportFromView()
		if vp != nil {
			vp.TopUpdateEnd(winUpdt)
		}
	}
	tb.UpdateEnd(bufUpdt) // nobody listening probably, but flag avail for testing
}

// AddFileNode adds the FileNode to the list or receivers of changes to buffer
func (tb *TextBuf) AddFileNode(fn *FileNode) {
	tb.TextBufSig.Connect(fn.This(), FileNodeBufSigRecv)
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
	tb.LinesMu.RLock()
	defer tb.LinesMu.RUnlock()

	if tb.NLines == 0 {
		if tb.Txt != nil {
			tb.Txt = tb.Txt[:0]
		}
		return
	}

	txt := bytes.Join(tb.LineBytes, []byte("\n"))
	txt = append(txt, '\n')
	tb.Txt = txt
}

// LinesToBytesCopy converts current Lines into a separate text byte copy --
// e.g., for autosave or other "offline" uses of the text -- doesn't affect
// byte offsets etc
func (tb *TextBuf) LinesToBytesCopy() []byte {
	tb.LinesMu.RLock()
	defer tb.LinesMu.RUnlock()

	txt := bytes.Join(tb.LineBytes, []byte("\n"))
	txt = append(txt, '\n')
	return txt
}

// BytesToLines converts current Txt bytes into lines, and initializes markup
// with raw text
func (tb *TextBuf) BytesToLines() {
	if len(tb.Txt) == 0 {
		tb.New(1)
		return
	}
	tb.LinesMu.Lock()
	lns := bytes.Split(tb.Txt, []byte("\n"))
	tb.NLines = len(lns)
	if len(lns[tb.NLines-1]) == 0 { // lines have lf at end typically
		tb.NLines--
		lns = lns[:tb.NLines]
	}
	tb.LinesMu.Unlock()
	tb.New(tb.NLines)
	tb.LinesMu.Lock()
	bo := 0
	for ln, txt := range lns {
		tb.ByteOffs[ln] = bo
		tb.Lines[ln] = bytes.Runes(txt)
		tb.LineBytes[ln] = make([]byte, len(txt))
		copy(tb.LineBytes[ln], txt)
		tb.Markup[ln] = HTMLEscapeRunes(tb.Lines[ln])
		bo += len(txt) + 1 // lf
	}
	tb.TotalBytes = bo
	tb.LinesMu.Unlock()
}

// Strings returns the current text as []string array.
// If addNewLn is true, each string line has a \n appended at end.
func (tb *TextBuf) Strings(addNewLn bool) []string {
	tb.LinesMu.RLock()
	defer tb.LinesMu.RUnlock()
	str := make([]string, tb.NLines)
	for i, l := range tb.Lines {
		str[i] = string(l)
		if addNewLn {
			str[i] += "\n"
		}
	}
	return str
}

// Search looks for a string (no regexp) within buffer,
// with given case-sensitivity, returning number of occurrences
// and specific match position list. column positions are in runes.
func (tb *TextBuf) Search(find []byte, ignoreCase, lexItems bool) (int, []textbuf.Match) {
	tb.LinesMu.RLock()
	defer tb.LinesMu.RUnlock()
	if lexItems {
		tb.MarkupMu.RLock()
		defer tb.MarkupMu.RUnlock()
		return textbuf.SearchLexItems(tb.Lines, tb.HiTags, find, ignoreCase)
	} else {
		return textbuf.SearchRuneLines(tb.Lines, find, ignoreCase)
	}
}

// SearchRegexp looks for a string (regexp) within buffer,
// returning number of occurrences and specific match position list.
// Column positions are in runes.
func (tb *TextBuf) SearchRegexp(re *regexp.Regexp) (int, []textbuf.Match) {
	tb.LinesMu.RLock()
	defer tb.LinesMu.RUnlock()
	return textbuf.SearchByteLinesRegexp(tb.LineBytes, re)
}

// BraceMatch finds the brace, bracket, or parens that is the partner
// of the one passed to function.
func (tb *TextBuf) BraceMatch(r rune, st lex.Pos) (en lex.Pos, found bool) {
	tb.LinesMu.RLock()
	defer tb.LinesMu.RUnlock()
	tb.MarkupMu.RLock()
	defer tb.MarkupMu.RUnlock()
	return lex.BraceMatch(tb.Lines, tb.HiTags, r, st, TextBufMaxScopeLines)
}

/////////////////////////////////////////////////////////////////////////////
//   Edits

// ValidPos returns a position that is in a valid range
func (tb *TextBuf) ValidPos(pos lex.Pos) lex.Pos {
	tb.LinesMu.RLock()
	defer tb.LinesMu.RUnlock()

	if tb.NLines == 0 {
		return lex.PosZero
	}
	if pos.Ln < 0 {
		pos.Ln = 0
	}
	if pos.Ln >= len(tb.Lines) {
		pos.Ln = len(tb.Lines) - 1
		pos.Ch = len(tb.Lines[pos.Ln])
		return pos
	}
	pos.Ln = ints.MinInt(pos.Ln, len(tb.Lines)-1)
	llen := len(tb.Lines[pos.Ln])
	pos.Ch = ints.MinInt(pos.Ch, llen)
	if pos.Ch < 0 {
		pos.Ch = 0
	}
	return pos
}

const (
	// EditSignal is used as an arg for edit methods with a signal arg, indicating
	// that a signal should be emitted.
	EditSignal = true

	// EditNoSignal is used as an arg for edit methods with a signal arg, indicating
	// that a signal should NOT be emitted.
	EditNoSignal = false

	// ReplaceMatchCase is used for MatchCase arg in ReplaceText method
	ReplaceMatchCase = true

	// ReplaceNoMatchCase is used for MatchCase arg in ReplaceText method
	ReplaceNoMatchCase = false
)

// DeleteText is the primary method for deleting text from the buffer.
// It deletes region of text between start and end positions,
// optionally signaling views after text lines have been updated.
// Sets the timestamp on resulting Edit to now.
// An Undo record is automatically saved depending on Undo.Off setting.
func (tb *TextBuf) DeleteText(st, ed lex.Pos, signal bool) *textbuf.Edit {
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
	tb.SetChanged()
	tb.LinesMu.Lock()
	tbe := tb.DeleteTextImpl(st, ed)
	tb.SaveUndo(tbe)
	tb.LinesMu.Unlock()
	if signal {
		tb.TextBufSig.Emit(tb.This(), int64(TextBufDelete), tbe)
	}
	if tb.Autosave {
		go tb.AutoSave()
	}
	return tbe
}

// DeleteTextImpl deletes region of text between start and end positions.
// Sets the timestamp on resulting textbuf.Edit to now.  Must be called under
// LinesMu.Lock.
func (tb *TextBuf) DeleteTextImpl(st, ed lex.Pos) *textbuf.Edit {
	tbe := tb.RegionImpl(st, ed)
	if tbe == nil {
		return nil
	}
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
	return tbe
}

// DeleteTextRect deletes rectangular region of text between start, end
// defining the upper-left and lower-right corners of a rectangle.
// Fails if st.Ch >= ed.Ch. Sets the timestamp on resulting textbuf.Edit to now.
// An Undo record is automatically saved depending on Undo.Off setting.
func (tb *TextBuf) DeleteTextRect(st, ed lex.Pos, signal bool) *textbuf.Edit {
	st = tb.ValidPos(st)
	ed = tb.ValidPos(ed)
	if st == ed {
		return nil
	}
	if !st.IsLess(ed) {
		log.Printf("giv.TextBuf DeleteTextRect: starting position must be less than ending!: st: %v, ed: %v\n", st, ed)
		return nil
	}
	tb.FileModCheck()
	tb.SetChanged()
	tb.LinesMu.Lock()
	tbe := tb.DeleteTextRectImpl(st, ed)
	tb.SaveUndo(tbe)
	tb.LinesMu.Unlock()
	if signal {
		tb.Refresh()
	}
	if tb.Autosave {
		go tb.AutoSave()
	}
	return tbe
}

// DeleteTextRectImpl deletes rectangular region of text between start, end
// defining the upper-left and lower-right corners of a rectangle.
// Fails if st.Ch >= ed.Ch. Sets the timestamp on resulting textbuf.Edit to now.
// Must be called under LinesMu.Lock.
func (tb *TextBuf) DeleteTextRectImpl(st, ed lex.Pos) *textbuf.Edit {
	tbe := tb.RegionRectImpl(st, ed)
	if tbe == nil {
		return nil
	}
	tbe.Delete = true
	for ln := st.Ln; ln <= ed.Ln; ln++ {
		ls := tb.Lines[ln]
		if len(ls) > st.Ch {
			tb.Lines[ln] = append(ls[:st.Ch], ls[ed.Ch:]...) // should be ok even if shorter?
		}
	}
	tb.LinesEdited(tbe)
	return tbe
}

// InsertText is the primary method for inserting text into the buffer.
// It inserts new text at given starting position, optionally signaling
// views after text has been inserted.  Sets the timestamp on resulting Edit to now.
// An Undo record is automatically saved depending on Undo.Off setting.
func (tb *TextBuf) InsertText(st lex.Pos, text []byte, signal bool) *textbuf.Edit {
	if len(text) == 0 {
		return nil
	}
	st = tb.ValidPos(st)
	tb.FileModCheck() // will just revert changes if shouldn't have changed
	tb.SetChanged()
	if len(tb.Lines) == 0 {
		tb.New(1)
	}
	tb.LinesMu.Lock()
	tbe := tb.InsertTextImpl(st, text)
	tb.SaveUndo(tbe)
	tb.LinesMu.Unlock()
	if signal {
		tb.TextBufSig.Emit(tb.This(), int64(TextBufInsert), tbe)
	}
	if tb.Autosave {
		go tb.AutoSave()
	}
	return tbe
}

// InsertTextImpl does the raw insert of new text at given starting position, returning
// a new Edit with timestamp of Now.  LinesMu must be locked surrounding this call.
func (tb *TextBuf) InsertTextImpl(st lex.Pos, text []byte) *textbuf.Edit {
	lns := bytes.Split(text, []byte("\n"))
	sz := len(lns)
	rs := bytes.Runes(lns[0])
	rsz := len(rs)
	ed := st
	var tbe *textbuf.Edit
	if sz == 1 {
		nt := append(tb.Lines[st.Ln], rs...) // first append to end to extend capacity
		copy(nt[st.Ch+rsz:], nt[st.Ch:])     // move stuff to end
		copy(nt[st.Ch:], rs)                 // copy into position
		tb.Lines[st.Ln] = nt
		ed.Ch += rsz
		tbe = tb.RegionImpl(st, ed)
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
		tbe = tb.RegionImpl(st, ed)
		tb.LinesInserted(tbe)
	}
	return tbe
}

// InsertTextRect inserts a rectangle of text defined in given textbuf.Edit record,
// (e.g., from RegionRect or DeleteRect), optionally signaling
// views after text has been inserted.
// Returns a copy of the Edit record with an updated timestamp.
// An Undo record is automatically saved depending on Undo.Off setting.
func (tb *TextBuf) InsertTextRect(tbe *textbuf.Edit, signal bool) *textbuf.Edit {
	if tbe == nil {
		return nil
	}
	tb.FileModCheck() // will just revert changes if shouldn't have changed
	tb.SetChanged()
	tb.LinesMu.Lock()
	nln := tb.NLines
	re := tb.InsertTextRectImpl(tbe)
	tb.SaveUndo(re)
	tb.LinesMu.Unlock()
	if signal {
		if re.Reg.End.Ln >= nln {
			ie := &textbuf.Edit{}
			ie.Reg.Start.Ln = nln - 1
			ie.Reg.End.Ln = re.Reg.End.Ln
			tb.TextBufSig.Emit(tb.This(), int64(TextBufInsert), ie)
		} else {
			tb.Refresh()
		}
	}
	if tb.Autosave {
		go tb.AutoSave()
	}
	return re
}

// InsertTextRectImpl does the raw insert of new text at given starting position,
// using a Rect textbuf.Edit  (e.g., from RegionRect or DeleteRect).
// Returns a copy of the Edit record with an updated timestamp.
func (tb *TextBuf) InsertTextRectImpl(tbe *textbuf.Edit) *textbuf.Edit {
	st := tbe.Reg.Start
	ed := tbe.Reg.End
	nlns := (ed.Ln - st.Ln) + 1
	if nlns <= 0 {
		return nil
	}
	// make sure there are enough lines -- add as needed
	cln := len(tb.Lines)
	if cln == 0 {
		tb.New(nlns)
	} else if cln <= ed.Ln {
		nln := (1 + ed.Ln) - cln
		tmp := make([][]rune, nln)
		tb.Lines = append(tb.Lines, tmp...) // first append to end to extend capacity
		tb.NLines = len(tb.Lines)
		ie := &textbuf.Edit{}
		ie.Reg.Start.Ln = cln - 1
		ie.Reg.End.Ln = ed.Ln
		tb.LinesInserted(ie)
	}
	nch := (ed.Ch - st.Ch)
	for i := 0; i < nlns; i++ {
		ln := st.Ln + i
		lr := tb.Lines[ln]
		ir := tbe.Text[i]
		if len(lr) < st.Ch {
			lr = append(lr, runes.Repeat([]rune(" "), st.Ch-len(lr))...)
		}
		nt := append(lr, ir...)          // first append to end to extend capacity
		copy(nt[st.Ch+nch:], nt[st.Ch:]) // move stuff to end
		copy(nt[st.Ch:], ir)             // copy into position
		tb.Lines[ln] = nt
	}
	re := tbe.Clone()
	re.Delete = false
	re.Reg.TimeNow()
	tb.LinesEdited(re)
	return re
}

// Region returns a textbuf.Edit representation of text between start and end positions
// returns nil if not a valid region.  sets the timestamp on the textbuf.Edit to now
func (tb *TextBuf) Region(st, ed lex.Pos) *textbuf.Edit {
	st = tb.ValidPos(st)
	ed = tb.ValidPos(ed)
	tb.LinesMu.RLock()
	defer tb.LinesMu.RUnlock()
	return tb.RegionImpl(st, ed)
}

// RegionImpl returns a textbuf.Edit representation of text between
// start and end positions. Returns nil if not a valid region.
// Sets the timestamp on the textbuf.Edit to now.
// Impl version must be called under LinesMu.RLock or Lock
func (tb *TextBuf) RegionImpl(st, ed lex.Pos) *textbuf.Edit {
	if st == ed || ed.IsLess(st) {
		return nil
	}
	if !st.IsLess(ed) {
		log.Printf("giv.TextBuf.Region: starting position must be less than ending!: st: %v, ed: %v\n", st, ed)
		return nil
	}
	tbe := &textbuf.Edit{Reg: textbuf.NewRegionPos(st, ed)}
	if ed.Ln == st.Ln {
		sz := ed.Ch - st.Ch
		if sz <= 0 {
			return nil
		}
		tbe.Text = make([][]rune, 1)
		tbe.Text[0] = make([]rune, sz)
		copy(tbe.Text[0][:sz], tb.Lines[st.Ln][st.Ch:ed.Ch])
	} else {
		// first get chars on start and end
		if ed.Ln >= len(tb.Lines) {
			ed.Ln = len(tb.Lines) - 1
			ed.Ch = len(tb.Lines[ed.Ln])
		}
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

// RegionRect returns a textbuf.Edit representation of text between start and end positions
// as a rectangle,
// returns nil if not a valid region.  sets the timestamp on the textbuf.Edit to now
func (tb *TextBuf) RegionRect(st, ed lex.Pos) *textbuf.Edit {
	st = tb.ValidPos(st)
	ed = tb.ValidPos(ed)
	tb.LinesMu.RLock()
	defer tb.LinesMu.RUnlock()
	return tb.RegionRectImpl(st, ed)
}

// RegionRectImpl returns a textbuf.Edit representation of rectangle of
// text between start (upper left) and end (bottom right) positions.
// Returns nil if not a valid region.
// All lines in Text are guaranteed to be of the same size,
// even if line had fewer chars.
// Sets the timestamp on the textbuf.Edit to now.
// Impl version must be called under LinesMu.RLock or Lock
func (tb *TextBuf) RegionRectImpl(st, ed lex.Pos) *textbuf.Edit {
	if st == ed {
		return nil
	}
	if !st.IsLess(ed) || st.Ch >= ed.Ch {
		log.Printf("giv.TextBuf.RegionRect: starting position must be less than ending!: st: %v, ed: %v\n", st, ed)
		return nil
	}
	tbe := &textbuf.Edit{Reg: textbuf.NewRegionPos(st, ed)}
	tbe.Rect = true
	// first get chars on start and end
	nlns := (ed.Ln - st.Ln) + 1
	nch := (ed.Ch - st.Ch)
	tbe.Text = make([][]rune, nlns)
	for i := 0; i < nlns; i++ {
		ln := st.Ln + i
		lr := tb.Lines[ln]
		ll := len(lr)
		var txt []rune
		if ll > st.Ch {
			sz := ints.MinInt(ll-st.Ch, nch)
			txt = make([]rune, sz, nch)
			copy(txt, lr[st.Ch:ed.Ch])
		}
		if len(txt) < nch { // rect
			txt = append(txt, runes.Repeat([]rune(" "), nch-len(txt))...)
		}
		tbe.Text[i] = txt
	}
	return tbe
}

// ReplaceText does DeleteText for given region, and then InsertText at given position
// (typically same as delSt but not necessarily), optionally emitting a signal after the insert.
// if matchCase is true, then the lex.MatchCase function is called to match the
// case (upper / lower) of the new inserted text to that of the text being replaced.
// returns the textbuf.Edit for the inserted text.
func (tb *TextBuf) ReplaceText(delSt, delEd, insPos lex.Pos, insTxt string, signal, matchCase bool) *textbuf.Edit {
	if matchCase {
		red := tb.Region(delSt, delEd)
		cur := string(red.ToBytes())
		insTxt = lex.MatchCase(cur, insTxt)
	}
	if len(insTxt) > 0 {
		tb.DeleteText(delSt, delEd, EditNoSignal)
		return tb.InsertText(insPos, []byte(insTxt), signal)
	}
	return tb.DeleteText(delSt, delEd, signal)
}

// SavePosHistory saves the cursor position in history stack of cursor positions --
// tracks across views -- returns false if position was on same line as last one saved
func (tb *TextBuf) SavePosHistory(pos lex.Pos) bool {
	if tb.PosHistory == nil {
		tb.PosHistory = make([]lex.Pos, 0, 1000)
	}
	sz := len(tb.PosHistory)
	if sz > 0 {
		if tb.PosHistory[sz-1].Ln == pos.Ln {
			return false
		}
	}
	tb.PosHistory = append(tb.PosHistory, pos)
	// fmt.Printf("saved pos hist: %v\n", pos)
	return true
}

/////////////////////////////////////////////////////////////////////////////
//   Syntax Highlighting Markup

// LinesEdited re-marks-up lines in edit (typically only 1).  Locks and
// unlocks the Markup mutex.  Must be called under Lines mutex lock.
func (tb *TextBuf) LinesEdited(tbe *textbuf.Edit) {
	tb.MarkupMu.Lock()
	st, ed := tbe.Reg.Start.Ln, tbe.Reg.End.Ln
	for ln := st; ln <= ed; ln++ {
		tb.LineBytes[ln] = []byte(string(tb.Lines[ln]))
		tb.Markup[ln] = HTMLEscapeRunes(tb.Lines[ln])
	}
	tb.MarkupLines(st, ed)
	tb.MarkupMu.Unlock()
	tb.StartDelayedReMarkup()
}

// LinesInserted inserts new lines in Markup corresponding to lines
// inserted in Lines text.  Locks and unlocks the Markup mutex, and
// must be called under lines mutex
func (tb *TextBuf) LinesInserted(tbe *textbuf.Edit) {
	stln := tbe.Reg.Start.Ln + 1
	nsz := (tbe.Reg.End.Ln - tbe.Reg.Start.Ln)

	tb.MarkupMu.Lock()
	tb.MarkupEdits = append(tb.MarkupEdits, tbe)

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

	// Tags
	tmptg := make([]lex.Line, nsz)
	ntg := append(tb.Tags, tmptg...)
	copy(ntg[stln+nsz:], ntg[stln:])
	copy(ntg[stln:], tmptg)
	tb.Tags = ntg

	// HiTags
	tmpht := make([]lex.Line, nsz)
	nht := append(tb.HiTags, tmpht...)
	copy(nht[stln+nsz:], nht[stln:])
	copy(nht[stln:], tmpht)
	tb.HiTags = nht

	// ByteOffs -- maintain mem updt
	tmpof := make([]int, nsz)
	nof := append(tb.ByteOffs, tmpof...)
	copy(nof[stln+nsz:], nof[stln:])
	copy(nof[stln:], tmpof)
	tb.ByteOffs = nof

	if tb.Hi.UsingPi() {
		pfs := tb.PiState.Done()
		pfs.Src.LinesInserted(stln, nsz)
	}

	st, ed := tbe.Reg.Start.Ln, tbe.Reg.End.Ln
	bo := tb.ByteOffs[st]
	for ln := st; ln <= ed; ln++ {
		tb.LineBytes[ln] = []byte(string(tb.Lines[ln]))
		tb.Markup[ln] = HTMLEscapeRunes(tb.Lines[ln])
		tb.ByteOffs[ln] = bo
		bo += len(tb.LineBytes[ln]) + 1
	}
	tb.MarkupLines(st, ed)
	tb.MarkupMu.Unlock()
	tb.StartDelayedReMarkup()
}

// LinesDeleted deletes lines in Markup corresponding to lines
// deleted in Lines text.  Locks and unlocks the Markup mutex, and
// must be called under lines mutex.
func (tb *TextBuf) LinesDeleted(tbe *textbuf.Edit) {
	tb.MarkupMu.Lock()

	tb.MarkupEdits = append(tb.MarkupEdits, tbe)

	stln := tbe.Reg.Start.Ln
	edln := tbe.Reg.End.Ln

	tb.LineBytes = append(tb.LineBytes[:stln], tb.LineBytes[edln:]...)
	tb.Markup = append(tb.Markup[:stln], tb.Markup[edln:]...)
	tb.Tags = append(tb.Tags[:stln], tb.Tags[edln:]...)
	tb.HiTags = append(tb.HiTags[:stln], tb.HiTags[edln:]...)
	tb.ByteOffs = append(tb.ByteOffs[:stln], tb.ByteOffs[edln:]...)

	if tb.Hi.UsingPi() {
		pfs := tb.PiState.Done()
		pfs.Src.LinesDeleted(stln, edln)
	}

	st := tbe.Reg.Start.Ln
	tb.LineBytes[st] = []byte(string(tb.Lines[st]))
	tb.Markup[st] = HTMLEscapeRunes(tb.Lines[st])
	tb.MarkupLines(st, st)
	tb.MarkupMu.Unlock()
	tb.StartDelayedReMarkup()
}

//////////////////////////////////////////////////////////////////////////////////////////////////
//  Markup

// MarkupLine does markup on a single line
func (tb *TextBuf) MarkupLine(ln int) {
	tb.LinesMu.Lock()
	tb.MarkupMu.Lock()

	if ln >= 0 && ln < len(tb.Markup) {
		tb.LineBytes[ln] = []byte(string(tb.Lines[ln]))
		tb.Markup[ln] = HTMLEscapeRunes(tb.Lines[ln])
		tb.MarkupLines(ln, ln)
	}
	tb.MarkupMu.Unlock()
	tb.LinesMu.Unlock()
}

// IsMarkingUp is true if the MarkupAllLines process is currently running
func (tb *TextBuf) IsMarkingUp() bool {
	return tb.HasFlag(int(TextBufMarkingUp))
}

// InitialMarkup does the first-pass markup on the file
func (tb *TextBuf) InitialMarkup() {
	if tb.Hi.UsingPi() {
		fs := tb.PiState.Done() // initialize
		fs.Src.SetBytes(tb.Txt)
	}
	mxhi := ints.MinInt(100, tb.NLines-1)
	tb.MarkupAllLines(mxhi)
}

// StartDelayedReMarkup starts a timer for doing markup after an interval
func (tb *TextBuf) StartDelayedReMarkup() {
	tb.MarkupDelayMu.Lock()
	defer tb.MarkupDelayMu.Unlock()
	if !tb.Hi.HasHi() || tb.NLines == 0 {
		return
	}
	if tb.MarkupDelayTimer != nil {
		tb.MarkupDelayTimer.Stop()
		tb.MarkupDelayTimer = nil
	}
	vp := tb.ViewportFromView()
	if vp != nil {
		cpop := vp.Win.CurPopup()
		if gi.PopupIsCompleter(cpop) {
			return
		}
	}
	if tb.Complete != nil && tb.Complete.IsAboutToShow() {
		return
	}
	tb.MarkupDelayTimer = time.AfterFunc(time.Duration(TextBufMarkupDelayMSec)*time.Millisecond,
		func() {
			// fmt.Printf("delayed remarkup\n")
			tb.MarkupDelayTimer = nil
			tb.ReMarkup()
		})
}

// StopDelayedReMarkup stops timer for doing markup after an interval
func (tb *TextBuf) StopDelayedReMarkup() {
	tb.MarkupDelayMu.Lock()
	defer tb.MarkupDelayMu.Unlock()
	if tb.MarkupDelayTimer != nil {
		tb.MarkupDelayTimer.Stop()
		tb.MarkupDelayTimer = nil
	}
}

// ReMarkup runs re-markup on text in background
func (tb *TextBuf) ReMarkup() {
	if !tb.Hi.HasHi() || tb.NLines == 0 {
		return
	}
	if tb.IsMarkingUp() {
		return
	}
	go tb.MarkupAllLines(-1)
}

// AdjustedTags updates tag positions for edits
// must be called under MarkupMu lock
func (tb *TextBuf) AdjustedTags(ln int) lex.Line {
	return tb.AdjustedTagsImpl(tb.Tags[ln], ln)
}

// AdjustedTagsImpl updates tag positions for edits, for given list of tags
func (tb *TextBuf) AdjustedTagsImpl(tags lex.Line, ln int) lex.Line {
	sz := len(tags)
	if sz == 0 {
		return nil
	}
	ntags := make(lex.Line, 0, sz)
	for _, tg := range tags {
		reg := textbuf.Region{Start: lex.Pos{Ln: ln, Ch: tg.St}, End: lex.Pos{Ln: ln, Ch: tg.Ed}}
		reg.Time = tg.Time
		reg = tb.Undos.AdjustReg(reg)
		if !reg.IsNil() {
			ntr := ntags.AddLex(tg.Tok, reg.Start.Ch, reg.End.Ch)
			ntr.Time.Now()
		}
	}
	// lex.LexsCleanup(&ntags)
	return ntags
}

// MarkupAllLines does syntax highlighting markup for all lines in buffer,
// calling MarkupMu mutex when setting the marked-up lines with the result --
// designed to be called in a separate goroutine.
// if maxLines > 0 then it specifies a maximum number of lines (for InitialMarkup)
func (tb *TextBuf) MarkupAllLines(maxLines int) {
	if !tb.Hi.HasHi() || tb.NLines == 0 {
		return
	}
	if tb.IsMarkingUp() {
		return
	}
	tb.SetFlag(int(TextBufMarkingUp))

	tb.MarkupMu.Lock()
	tb.MarkupEdits = nil
	tb.MarkupMu.Unlock()

	var txt []byte
	if maxLines > 0 {
		tb.LinesMu.RLock()
		mln := ints.MinInt(maxLines, len(tb.LineBytes))
		txt = bytes.Join(tb.LineBytes[:mln], []byte("\n"))
		txt = append(txt, '\n')
		tb.LinesMu.RUnlock()
	} else {
		txt = tb.LinesToBytesCopy()
	}
	mtags, err := tb.Hi.MarkupTagsAll(txt) // does full parse, outside of markup lock
	if err != nil {
		tb.ClearFlag(int(TextBufMarkingUp))
		return
	}

	// by this point mtags could be out of sync with deletes that have happened
	tb.LinesMu.Lock()
	tb.MarkupMu.Lock()

	maxln := ints.MinInt(len(tb.Markup), tb.NLines)
	if maxLines > 0 {
		maxln = ints.MinInt(maxln, maxLines)
	}

	if tb.Hi.UsingPi() {
		pfs := tb.PiState.Done()
		// first update mtags with any changes since it was generated
		for _, tbe := range tb.MarkupEdits {
			if tbe.Delete {
				stln := tbe.Reg.Start.Ln
				edln := tbe.Reg.End.Ln
				pfs.Src.LinesDeleted(stln, edln)
			} else {
				stln := tbe.Reg.Start.Ln + 1
				nlns := (tbe.Reg.End.Ln - tbe.Reg.Start.Ln)
				pfs.Src.LinesInserted(stln, nlns)
			}
		}
		tb.MarkupEdits = nil
		// if maxln > 1 && len(pfs.Src.Lexs)-1 != maxln {
		// 	fmt.Printf("error: markup out of sync: %v != %v len(Lexs)\n", maxln, len(pfs.Src.Lexs)-1)
		// }
		for ln := 0; ln < maxln; ln++ {
			tb.HiTags[ln] = pfs.LexLine(ln) // does clone, combines comments too
		}
	} else {
		// first update mtags with any changes since it was generated
		for _, tbe := range tb.MarkupEdits {
			if tbe.Delete {
				stln := tbe.Reg.Start.Ln
				edln := tbe.Reg.End.Ln
				mtags = append(mtags[:stln], mtags[edln:]...)
			} else {
				stln := tbe.Reg.Start.Ln + 1
				nlns := (tbe.Reg.End.Ln - tbe.Reg.Start.Ln)
				tmpht := make([]lex.Line, nlns)
				nht := append(mtags, tmpht...)
				copy(nht[stln+nlns:], nht[stln:])
				copy(nht[stln:], tmpht)
				mtags = nht
			}
		}
		tb.MarkupEdits = nil
		// if maxln > 0 && len(mtags) != maxln {
		// 	fmt.Printf("error: markup out of sync: %v != %v len(mtags)\n", maxln, len(mtags))
		// }
		maxln = ints.MinInt(maxln, len(mtags))
		for ln := 0; ln < maxln; ln++ {
			tb.HiTags[ln] = mtags[ln] // chroma tags are freshly allocated
		}
	}
	for ln := 0; ln < maxln; ln++ {
		tb.Tags[ln] = tb.AdjustedTags(ln)
		tb.Markup[ln] = tb.Hi.MarkupLine(tb.Lines[ln], tb.HiTags[ln], tb.Tags[ln])
	}
	tb.MarkupMu.Unlock()
	tb.LinesMu.Unlock()
	tb.ClearFlag(int(TextBufMarkingUp))
	tb.TextBufSig.Emit(tb.This(), int64(TextBufMarkUpdt), tb.Txt)
}

// MarkupFromTags does syntax highlighting markup using existing HiTags without
// running new tagging -- for special case where tagging is under external
// control
func (tb *TextBuf) MarkupFromTags() {
	tb.MarkupMu.Lock()
	// getting the lock means we are in control of the flag
	tb.SetFlag(int(TextBufMarkingUp))

	maxln := ints.MinInt(len(tb.HiTags), tb.NLines)
	for ln := 0; ln < maxln; ln++ {
		tb.Markup[ln] = tb.Hi.MarkupLine(tb.Lines[ln], tb.HiTags[ln], nil)
	}
	tb.MarkupMu.Unlock()
	tb.ClearFlag(int(TextBufMarkingUp))
	tb.TextBufSig.Emit(tb.This(), int64(TextBufMarkUpdt), tb.Txt)
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
		ltxt := tb.Lines[ln]
		mt, err := tb.Hi.MarkupTagsLine(ln, ltxt)
		if err == nil {
			tb.HiTags[ln] = mt
			tb.Markup[ln] = tb.Hi.MarkupLine(ltxt, mt, tb.AdjustedTags(ln))
		} else {
			tb.Markup[ln] = HTMLEscapeRunes(ltxt)
			allgood = false
		}
	}
	// Now we trigger a background reparse of everything in a separate pi.FilesState
	// that gets switched into the current.
	return allgood
}

// MarkupLinesLock does MarkupLines and gets the mutex lock first
func (tb *TextBuf) MarkupLinesLock(st, ed int) bool {
	tb.MarkupMu.Lock()
	defer tb.MarkupMu.Unlock()
	return tb.MarkupLines(st, ed)
}

/////////////////////////////////////////////////////////////////////////////
//   Undo

// SaveUndo saves given edit to undo stack
func (tb *TextBuf) SaveUndo(tbe *textbuf.Edit) {
	tb.Undos.Save(tbe)
}

// Undo undoes next group of items on the undo stack
func (tb *TextBuf) Undo() *textbuf.Edit {
	tb.LinesMu.Lock()
	tbe := tb.Undos.UndoPop()
	if tbe == nil {
		tb.LinesMu.Unlock()
		tb.ClearChanged()
		tb.AutoSaveDelete()
		return nil
	}
	bufUpdt, winUpdt, autoSave := tb.BatchUpdateStart()
	defer tb.BatchUpdateEnd(bufUpdt, winUpdt, autoSave)
	stgp := tbe.Group
	last := tbe
	for {
		if tbe.Rect {
			if tbe.Delete {
				utbe := tb.InsertTextRectImpl(tbe)
				utbe.Group = stgp + tbe.Group
				if tb.Opts.EmacsUndo {
					tb.Undos.SaveUndo(utbe)
				}
				tb.LinesMu.Unlock()
				tb.Refresh()
			} else {
				utbe := tb.DeleteTextRectImpl(tbe.Reg.Start, tbe.Reg.End)
				utbe.Group = stgp + tbe.Group
				if tb.Opts.EmacsUndo {
					tb.Undos.SaveUndo(utbe)
				}
				tb.LinesMu.Unlock()
				tb.Refresh()
			}
		} else {
			if tbe.Delete {
				utbe := tb.InsertTextImpl(tbe.Reg.Start, tbe.ToBytes())
				utbe.Group = stgp + tbe.Group
				if tb.Opts.EmacsUndo {
					tb.Undos.SaveUndo(utbe)
				}
				tb.LinesMu.Unlock()
				tb.TextBufSig.Emit(tb.This(), int64(TextBufInsert), utbe)
			} else {
				utbe := tb.DeleteTextImpl(tbe.Reg.Start, tbe.Reg.End)
				utbe.Group = stgp + tbe.Group
				if tb.Opts.EmacsUndo {
					tb.Undos.SaveUndo(utbe)
				}
				tb.LinesMu.Unlock()
				tb.TextBufSig.Emit(tb.This(), int64(TextBufDelete), utbe)
			}
		}
		tb.LinesMu.Lock()
		tbe = tb.Undos.UndoPopIfGroup(stgp)
		if tbe == nil {
			break
		}
		last = tbe
	}
	tb.LinesMu.Unlock()
	if tb.Undos.Pos == 0 {
		tb.ClearChanged()
		tb.AutoSaveDelete()
	}
	return last
}

// EmacsUndoSave is called by TextView at end of latest set of undo commands.
// If EmacsUndo mode is active, saves the current UndoStack to the regular Undo stack
// at the end, and moves undo to the very end -- undo is a constant stream.
func (tb *TextBuf) EmacsUndoSave() {
	if !tb.Opts.EmacsUndo {
		return
	}
	tb.Undos.UndoStackSave()
}

// Redo redoes next group of items on the undo stack,
// and returns the last record, nil if no more
func (tb *TextBuf) Redo() *textbuf.Edit {
	tb.LinesMu.Lock()
	tbe := tb.Undos.RedoNext()
	if tbe == nil {
		tb.LinesMu.Unlock()
		return nil
	}
	bufUpdt, winUpdt, autoSave := tb.BatchUpdateStart()
	defer tb.BatchUpdateEnd(bufUpdt, winUpdt, autoSave)
	stgp := tbe.Group
	last := tbe
	for {
		if tbe.Rect {
			if tbe.Delete {
				tb.DeleteTextRectImpl(tbe.Reg.Start, tbe.Reg.End)
				tb.LinesMu.Unlock()
				tb.Refresh()
			} else {
				tb.InsertTextRectImpl(tbe)
				tb.LinesMu.Unlock()
				tb.Refresh()
			}
		} else {
			if tbe.Delete {
				tb.DeleteTextImpl(tbe.Reg.Start, tbe.Reg.End)
				tb.LinesMu.Unlock()
				tb.TextBufSig.Emit(tb.This(), int64(TextBufDelete), tbe)
			} else {
				tb.InsertTextImpl(tbe.Reg.Start, tbe.ToBytes())
				tb.LinesMu.Unlock()
				tb.TextBufSig.Emit(tb.This(), int64(TextBufInsert), tbe)
			}
		}
		tb.LinesMu.Lock()
		tbe = tb.Undos.RedoNextIfGroup(stgp)
		if tbe == nil {
			break
		}
		last = tbe
	}
	tb.LinesMu.Unlock()
	return last
}

// AdjustPos adjusts given text position, which was recorded at given time
// for any edits that have taken place since that time (using the Undo stack).
// del determines what to do with positions within a deleted region -- either move
// to start or end of the region, or return an error
func (tb *TextBuf) AdjustPos(pos lex.Pos, t time.Time, del textbuf.AdjustPosDel) lex.Pos {
	return tb.Undos.AdjustPos(pos, t, del)
}

// AdjustReg adjusts given text region for any edits that
// have taken place since time stamp on region (using the Undo stack).
// If region was wholly within a deleted region, then RegionNil will be
// returned -- otherwise it is clipped appropriately as function of deletes.
func (tb *TextBuf) AdjustReg(reg textbuf.Region) textbuf.Region {
	return tb.Undos.AdjustReg(reg)
}

/////////////////////////////////////////////////////////////////////////////
//   Tags

// AddTag adds a new custom tag for given line, at given position
func (tb *TextBuf) AddTag(ln, st, ed int, tag token.Tokens) {
	if !tb.IsValidLine(ln) {
		return
	}
	tb.MarkupMu.Lock()
	tr := lex.NewLex(token.KeyToken{Tok: tag}, st, ed)
	tr.Time.Now()
	if len(tb.Tags[ln]) == 0 {
		tb.Tags[ln] = append(tb.Tags[ln], tr)
	} else {
		tb.Tags[ln] = tb.AdjustedTags(ln) // must re-adjust before adding new ones!
		tb.Tags[ln].AddSort(tr)
	}
	tb.MarkupMu.Unlock()
	tb.MarkupLinesLock(ln, ln)
}

// AddTagEdit adds a new custom tag for given line, using textbuf.Edit for location
func (tb *TextBuf) AddTagEdit(tbe *textbuf.Edit, tag token.Tokens) {
	tb.AddTag(tbe.Reg.Start.Ln, tbe.Reg.Start.Ch, tbe.Reg.End.Ch, tag)
}

// TagAt returns tag at given text position, if one exists -- returns false if not
func (tb *TextBuf) TagAt(pos lex.Pos) (reg lex.Lex, ok bool) {
	tb.MarkupMu.Lock()
	defer tb.MarkupMu.Unlock()
	if !tb.IsValidLine(pos.Ln) {
		return
	}
	tb.Tags[pos.Ln] = tb.AdjustedTags(pos.Ln) // re-adjust for current info
	for _, t := range tb.Tags[pos.Ln] {
		if t.St >= pos.Ch && t.Ed < pos.Ch {
			return t, true
		}
	}
	return
}

// RemoveTag removes tag (optionally only given tag if non-zero) at given position
// if it exists -- returns tag
func (tb *TextBuf) RemoveTag(pos lex.Pos, tag token.Tokens) (reg lex.Lex, ok bool) {
	if !tb.IsValidLine(pos.Ln) {
		return
	}
	tb.MarkupMu.Lock()
	tb.Tags[pos.Ln] = tb.AdjustedTags(pos.Ln) // re-adjust for current info
	for i, t := range tb.Tags[pos.Ln] {
		if t.ContainsPos(pos.Ch) {
			if tag > 0 && t.Tok.Tok != tag {
				continue
			}
			tb.Tags[pos.Ln].DeleteIdx(i)
			reg = t
			ok = true
			break
		}
	}
	tb.MarkupMu.Unlock()
	if ok {
		tb.MarkupLinesLock(pos.Ln, pos.Ln)
	}
	return
}

// HiTagAtPos returns the highlighting (markup) lexical tag at given position
// using current Markup tags, and index, -- could be nil if none or out of range
func (tb *TextBuf) HiTagAtPos(pos lex.Pos) (*lex.Lex, int) {
	tb.MarkupMu.Lock()
	defer tb.MarkupMu.Unlock()
	if !tb.IsValidLine(pos.Ln) {
		return nil, -1
	}
	return tb.HiTags[pos.Ln].AtPos(pos.Ch)
}

// LexString returns the string associated with given Lex (Tag) at given line
func (tb *TextBuf) LexString(ln int, lx *lex.Lex) string {
	tb.LinesMu.RLock()
	defer tb.LinesMu.RUnlock()
	if !tb.IsValidLine(ln) {
		return ""
	}
	rns := tb.Lines[ln][lx.St:lx.Ed]
	return string(rns)
}

// LexObjPathString returns the string at given lex, and including prior
// lex-tagged regions that include sequences of PunctSepPeriod and NameTag
// which are used for object paths -- used for e.g., debugger to pull out
// variable expressions that can be evaluated.
func (tb *TextBuf) LexObjPathString(ln int, lx *lex.Lex) string {
	tb.LinesMu.RLock()
	defer tb.LinesMu.RUnlock()
	if !tb.IsValidLine(ln) {
		return ""
	}
	stlx := lex.ObjPathAt(tb.HiTags[ln], lx)
	rns := tb.Lines[ln][stlx.St:lx.Ed]
	return string(rns)
}

// InTokenSubCat returns true if the given text position is marked with lexical
// type in given SubCat sub-category
func (tb *TextBuf) InTokenSubCat(pos lex.Pos, subCat token.Tokens) bool {
	lx, _ := tb.HiTagAtPos(pos)
	return lx != nil && lx.Tok.Tok.InSubCat(subCat)
}

// InLitString returns true if position is in a string literal
func (tb *TextBuf) InLitString(pos lex.Pos) bool {
	return tb.InTokenSubCat(pos, token.LitStr)
}

// InTokenCode returns true if position is in a Keyword, Name, Operator, or Punctuation.
// This is useful for turning off spell checking in docs
func (tb *TextBuf) InTokenCode(pos lex.Pos) bool {
	lx, _ := tb.HiTagAtPos(pos)
	if lx == nil {
		return false
	}
	return lx.Tok.Tok.IsCode()
}

/////////////////////////////////////////////////////////////////////////////
//   LineIcons / Colors

// SetLineIcon sets given icon at given line (0 starting)
func (tb *TextBuf) SetLineIcon(ln int, icon string) {
	tb.LinesMu.Lock()
	defer tb.LinesMu.Unlock()
	if tb.LineIcons == nil {
		tb.LineIcons = make(map[int]string)
		tb.Icons = make(map[string]*gi.Icon)
	}
	tb.LineIcons[ln] = icon
	ic, has := tb.Icons[icon]
	if !has {
		ic = &gi.Icon{}
		ic.InitName(ic, icon)
		ic.SetIcon(icon)
		ic.SetProp("width", units.NewEm(1))
		ic.SetProp("height", units.NewEm(1))
		tb.Icons[icon] = ic
	}
}

// DeleteLineIcon deletes any icon at given line (0 starting)
// if ln = -1 then delete all line icons.
func (tb *TextBuf) DeleteLineIcon(ln int) {
	tb.LinesMu.Lock()
	defer tb.LinesMu.Unlock()
	if ln < 0 {
		tb.LineIcons = nil
		return
	}
	if tb.LineIcons == nil {
		return
	}
	delete(tb.LineIcons, ln)
}

// SetLineColor sets given color (name or hex string) at given line (0 starting)
func (tb *TextBuf) SetLineColor(ln int, color string) {
	tb.LinesMu.Lock()
	defer tb.LinesMu.Unlock()
	if tb.LineColors == nil {
		tb.LineColors = make(map[int]gist.Color)
	}
	clr, _ := gist.ColorFromString(color, nil)
	tb.LineColors[ln] = clr
}

// HasLineColor checks if given line has a line color set
func (tb *TextBuf) HasLineColor(ln int) bool {
	tb.LinesMu.Lock()
	defer tb.LinesMu.Unlock()
	if ln < 0 {
		return false
	}
	if tb.LineColors == nil {
		return false
	}
	_, has := tb.LineColors[ln]
	return has
}

// HasLineColor
func (tb *TextBuf) DeleteLineColor(ln int) {
	tb.LinesMu.Lock()
	defer tb.LinesMu.Unlock()
	if ln < 0 {
		tb.LineColors = nil
		return
	}
	if tb.LineColors == nil {
		return
	}
	delete(tb.LineColors, ln)
}

/////////////////////////////////////////////////////////////////////////////
//   Indenting

// see pi/lex/indent.go for support functions

// IndentLine indents line by given number of tab stops, using tabs or spaces,
// for given tab size (if using spaces) -- either inserts or deletes to reach target.
// Returns edit record for any change.
func (tb *TextBuf) IndentLine(ln, ind int) *textbuf.Edit {
	asv := tb.AutoSaveOff()
	defer tb.AutoSaveRestore(asv)

	tabSz := tb.Opts.TabSize
	ichr := indent.Tab
	if tb.Opts.SpaceIndent {
		ichr = indent.Space
	}

	tb.LinesMu.RLock()
	curind, _ := lex.LineIndent(tb.Lines[ln], tabSz)
	tb.LinesMu.RUnlock()
	if ind > curind {
		return tb.InsertText(lex.Pos{Ln: ln}, indent.Bytes(ichr, ind-curind, tabSz), EditSignal)
	} else if ind < curind {
		spos := indent.Len(ichr, ind, tabSz)
		cpos := indent.Len(ichr, curind, tabSz)
		return tb.DeleteText(lex.Pos{Ln: ln, Ch: spos}, lex.Pos{Ln: ln, Ch: cpos}, EditSignal)
	}
	return nil
}

// AutoIndent indents given line to the level of the prior line, adjusted
// appropriately if the current line starts with one of the given un-indent
// strings, or the prior line ends with one of the given indent strings.
// Returns any edit that took place (could be nil), along with the auto-indented
// level and character position for the indent of the current line.
func (tb *TextBuf) AutoIndent(ln int) (tbe *textbuf.Edit, indLev, chPos int) {
	tabSz := tb.Opts.TabSize

	tb.LinesMu.RLock()
	tb.MarkupMu.RLock()
	lp, _ := pi.LangSupport.Props(tb.PiState.Sup)
	var pInd, delInd int
	if lp != nil && lp.Lang != nil {
		pInd, delInd, _, _ = lp.Lang.IndentLine(&tb.PiState, tb.Lines, tb.HiTags, ln, tabSz)
	} else {
		pInd, delInd, _, _ = lex.BracketIndentLine(tb.Lines, tb.HiTags, ln, tabSz)
	}
	tb.MarkupMu.RUnlock()
	tb.LinesMu.RUnlock()
	ichr := tb.Opts.IndentChar()

	indLev = pInd + delInd
	chPos = indent.Len(ichr, indLev, tabSz)
	tbe = tb.IndentLine(ln, indLev)
	return
}

// AutoIndentRegion does auto-indent over given region -- end is *exclusive*
func (tb *TextBuf) AutoIndentRegion(st, ed int) {
	bufUpdt, winUpdt, autoSave := tb.BatchUpdateStart()
	defer tb.BatchUpdateEnd(bufUpdt, winUpdt, autoSave)
	for ln := st; ln < ed; ln++ {
		if ln >= tb.NLines {
			break
		}
		tb.AutoIndent(ln)
	}
}

// CommentStart returns the char index where the comment starts on given line, -1 if no comment
func (tb *TextBuf) CommentStart(ln int) int {
	if !tb.IsValidLine(ln) {
		return -1
	}
	comst, _ := tb.Opts.CommentStrs()
	if comst == "" {
		return -1
	}
	tb.LinesMu.RLock()
	defer tb.LinesMu.RUnlock()
	return runes.Index(tb.Line(ln), []rune(comst))
}

// InComment returns true if the given text position is within a commented region
func (tb *TextBuf) InComment(pos lex.Pos) bool {
	if tb.InTokenSubCat(pos, token.Comment) {
		return true
	}
	cs := tb.CommentStart(pos.Ln)
	if cs < 0 {
		return false
	}
	return pos.Ch > cs
}

// LineCommented returns true if the given line is a full-comment line (i.e., starts with a comment)
func (tb *TextBuf) LineCommented(ln int) bool {
	tb.MarkupMu.RLock()
	defer tb.MarkupMu.RUnlock()
	tags := tb.HiTags[ln]
	if len(tags) == 0 {
		return false
	}
	if tags[0].Tok.Tok.InCat(token.Comment) {
		return true
	}
	return false
}

// CommentRegion inserts comment marker on given lines -- end is *exclusive*
func (tb *TextBuf) CommentRegion(st, ed int) {
	bufUpdt, winUpdt, autoSave := tb.BatchUpdateStart()
	defer tb.BatchUpdateEnd(bufUpdt, winUpdt, autoSave)

	tabSz := tb.Opts.TabSize

	ch := 0
	tb.LinesMu.RLock()
	ind, _ := lex.LineIndent(tb.Lines[st], tabSz)
	tb.LinesMu.RUnlock()

	if ind > 0 {
		if tb.Opts.SpaceIndent {
			ch = tb.Opts.TabSize * ind
		} else {
			ch = ind
		}
	}

	comst, comed := tb.Opts.CommentStrs()
	if comst == "" {
		fmt.Printf("giv.TextBuf: %v attempt to comment region without any comment syntax defined\n", tb.Nm)
		return
	}

	eln := ints.MinInt(tb.NumLines(), ed)
	ncom := 0
	nln := eln - st
	for ln := st; ln < eln; ln++ {
		if tb.LineCommented(ln) {
			ncom++
		}
	}
	trgln := ints.MaxInt(nln-2, 1)
	doCom := true
	if ncom >= trgln {
		doCom = false
	}

	for ln := st; ln < eln; ln++ {
		if doCom {
			tb.InsertText(lex.Pos{Ln: ln, Ch: ch}, []byte(comst), EditSignal)
			if comed != "" {
				lln := len(tb.Lines[ln])
				tb.InsertText(lex.Pos{Ln: ln, Ch: lln}, []byte(comed), EditSignal)
			}
		} else {
			idx := tb.CommentStart(ln)
			if idx >= 0 {
				tb.DeleteText(lex.Pos{Ln: ln, Ch: idx}, lex.Pos{Ln: ln, Ch: idx + len(comst)}, EditSignal)
			}
			if comed != "" {
				idx := runes.IndexFold(tb.Line(ln), []rune(comed))
				if idx >= 0 {
					tb.DeleteText(lex.Pos{Ln: ln, Ch: idx}, lex.Pos{Ln: ln, Ch: idx + len(comed)}, EditSignal)
				}
			}
		}
	}
}

// JoinParaLines merges sequences of lines with hard returns forming paragraphs,
// separated by blank lines, into a single line per paragraph,
// within the given line regions -- edLn is *inclusive*
func (tb *TextBuf) JoinParaLines(stLn, edLn int) {
	bufUpdt, winUpdt, autoSave := tb.BatchUpdateStart()

	curEd := edLn                      // current end of region being joined == last blank line
	for ln := edLn; ln >= stLn; ln-- { // reverse order
		lb := tb.LineBytes[ln]
		lbt := bytes.TrimSpace(lb)
		if len(lbt) == 0 || ln == stLn {
			if ln < curEd-1 {
				stp := lex.Pos{Ln: ln + 1}
				if ln == stLn {
					stp.Ln--
				}
				ep := lex.Pos{Ln: curEd - 1}
				if curEd == edLn {
					ep.Ln = curEd
				}
				eln := tb.Lines[ep.Ln]
				ep.Ch = len(eln)
				tlb := bytes.Join(tb.LineBytes[stp.Ln:ep.Ln+1], []byte(" "))
				tb.ReplaceText(stp, ep, stp, string(tlb), EditSignal, ReplaceNoMatchCase)
			}
			curEd = ln
		}
	}
	tb.BatchUpdateEnd(bufUpdt, winUpdt, autoSave)
	tb.Refresh()
}

// TabsToSpaces replaces tabs with spaces in given line.
func (tb *TextBuf) TabsToSpaces(ln int) {
	tabSz := tb.Opts.TabSize

	lr := tb.Lines[ln]
	st := lex.Pos{Ln: ln}
	ed := lex.Pos{Ln: ln}
	i := 0
	for {
		if i >= len(lr) {
			break
		}
		r := lr[i]
		if r == '\t' {
			po := i % tabSz
			nspc := tabSz - po
			st.Ch = i
			ed.Ch = i + 1
			tb.ReplaceText(st, ed, st, indent.Spaces(1, nspc), EditNoSignal, ReplaceNoMatchCase)
			i += nspc
			lr = tb.Lines[ln]
		} else {
			i++
		}
	}
}

// TabsToSpacesRegion replaces tabs with spaces over given region -- end is *exclusive*
func (tb *TextBuf) TabsToSpacesRegion(st, ed int) {
	bufUpdt, winUpdt, autoSave := tb.BatchUpdateStart()
	for ln := st; ln < ed; ln++ {
		if ln >= tb.NLines {
			break
		}
		tb.TabsToSpaces(ln)
	}
	tb.BatchUpdateEnd(bufUpdt, winUpdt, autoSave)
	tb.Refresh()
}

// SpacesToTabs replaces spaces with tabs in given line.
func (tb *TextBuf) SpacesToTabs(ln int) {
	tabSz := tb.Opts.TabSize

	lr := tb.Lines[ln]
	st := lex.Pos{Ln: ln}
	ed := lex.Pos{Ln: ln}
	i := 0
	nspc := 0
	for {
		if i >= len(lr) {
			break
		}
		r := lr[i]
		if r == ' ' {
			nspc++
			if nspc == tabSz {
				st.Ch = i - (tabSz - 1)
				ed.Ch = i + 1
				tb.ReplaceText(st, ed, st, "\t", EditNoSignal, ReplaceNoMatchCase)
				i -= tabSz - 1
				lr = tb.Lines[ln]
				nspc = 0
			} else {
				i++
			}
		} else {
			nspc = 0
			i++
		}
	}
}

// SpacesToTabsRegion replaces tabs with spaces over given region -- end is *exclusive*
func (tb *TextBuf) SpacesToTabsRegion(st, ed int) {
	bufUpdt, winUpdt, autoSave := tb.BatchUpdateStart()
	for ln := st; ln < ed; ln++ {
		if ln >= tb.NLines {
			break
		}
		tb.SpacesToTabs(ln)
	}
	tb.BatchUpdateEnd(bufUpdt, winUpdt, autoSave)
	tb.Refresh()
}

///////////////////////////////////////////////////////////////////////////////
//    Complete and Spell

// SetCompleter sets completion functions so that completions will
// automatically be offered as the user types
func (tb *TextBuf) SetCompleter(data interface{}, matchFun complete.MatchFunc, editFun complete.EditFunc,
	lookupFun complete.LookupFunc) {
	if tb.Complete != nil {
		if tb.Complete.Context == data {
			tb.Complete.MatchFunc = matchFun
			tb.Complete.EditFunc = editFun
			tb.Complete.LookupFunc = lookupFun
			return
		}
		tb.DeleteCompleter()
	}
	tb.Complete = &gi.Complete{}
	tb.Complete.InitName(tb.Complete, "tb-completion") // needed for standalone Ki's
	tb.Complete.Context = data
	tb.Complete.MatchFunc = matchFun
	tb.Complete.EditFunc = editFun
	tb.Complete.LookupFunc = lookupFun
	// note: only need to connect once..
	tb.Complete.CompleteSig.ConnectOnly(tb.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
		tbf, _ := recv.Embed(KiT_TextBuf).(*TextBuf)
		if sig == int64(gi.CompleteSelect) {
			tbf.CompleteText(data.(string)) // always use data
		} else if sig == int64(gi.CompleteExtend) {
			tbf.CompleteExtend(data.(string)) // always use data
		}
	})
}

func (tb *TextBuf) DeleteCompleter() {
	if tb.Complete == nil {
		return
	}
	tb.Complete.CompleteSig.Disconnect(tb.This())
	tb.Complete.Destroy()
	tb.Complete = nil
}

// CompleteText edits the text using the string chosen from the completion menu
func (tb *TextBuf) CompleteText(s string) {
	if s == "" {
		return
	}
	// give the completer a chance to edit the completion before insert,
	// also it return a number of runes past the cursor to delete
	st := lex.Pos{tb.Complete.SrcLn, 0}
	en := lex.Pos{tb.Complete.SrcLn, tb.LineLen(tb.Complete.SrcLn)}
	var tbes string
	tbe := tb.Region(st, en)
	if tbe != nil {
		tbes = string(tbe.ToBytes())
	}
	c := tb.Complete.GetCompletion(s)
	pos := lex.Pos{tb.Complete.SrcLn, tb.Complete.SrcCh}
	ed := tb.Complete.EditFunc(tb.Complete.Context, tbes, tb.Complete.SrcCh, c, tb.Complete.Seed)
	if ed.ForwardDelete > 0 {
		delEn := lex.Pos{tb.Complete.SrcLn, tb.Complete.SrcCh + ed.ForwardDelete}
		tb.DeleteText(pos, delEn, EditNoSignal)
	}
	// now the normal completion insertion
	st = pos
	st.Ch -= len(tb.Complete.Seed)
	tb.ReplaceText(st, pos, st, ed.NewText, EditSignal, ReplaceNoMatchCase)
	if tb.CurView != nil {
		ep := st
		ep.Ch += len(ed.NewText) + ed.CursorAdjust
		tb.CurView.SetCursorShow(ep)
		tb.CurView = nil
	}
}

// CompleteExtend inserts the extended seed at the current cursor position
func (tb *TextBuf) CompleteExtend(s string) {
	if s == "" {
		return
	}
	pos := lex.Pos{tb.Complete.SrcLn, tb.Complete.SrcCh}
	st := pos
	st.Ch -= len(tb.Complete.Seed)
	tb.ReplaceText(st, pos, st, s, EditSignal, ReplaceNoMatchCase)
	if tb.CurView != nil {
		ep := st
		ep.Ch += len(s)
		tb.CurView.SetCursorShow(ep)
		tb.CurView = nil
	}
}

// IsSpellEnabled returns true if spelling correction is enabled,
// taking into account given position in text if it is relevant for cases
// where it is only conditionally enabled
func (tb *TextBuf) IsSpellEnabled(pos lex.Pos) bool {
	if tb.Spell == nil || !tb.Opts.SpellCorrect {
		return false
	}
	switch tb.Info.Cat {
	case filecat.Doc: // not in code!
		return !tb.InTokenCode(pos)
	case filecat.Code:
		return tb.InComment(pos) || tb.InLitString(pos)
	default:
		return false
	}
}

// SetSpell sets spell correct functions so that spell correct will
// automatically be offered as the user types
func (tb *TextBuf) SetSpell() {
	if tb.Spell != nil {
		return
	}
	gi.InitSpell()
	tb.Spell = &gi.Spell{}
	tb.Spell.InitName(tb.Spell, "tb-spellcorrect") // needed for standalone Ki's
	// note: only need to connect once..
	tb.Spell.SpellSig.ConnectOnly(tb.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
		if sig == int64(gi.SpellSelect) {
			tbf, _ := recv.Embed(KiT_TextBuf).(*TextBuf)
			tbf.CorrectText(data.(string)) // always use data
		} else if sig == int64(gi.SpellIgnore) {
			tbf, _ := recv.Embed(KiT_TextBuf).(*TextBuf)
			tbf.CorrectText(data.(string)) // always use data
		}
	})
}

// DeleteSpell deletes any existing spell object
func (tb *TextBuf) DeleteSpell() {
	if tb.Spell == nil {
		return
	}
	tb.Spell.SpellSig.Disconnect(tb.This())
	tb.Spell.Destroy()
	tb.Spell = nil
}

// CorrectText edits the text using the string chosen from the correction menu
func (tb *TextBuf) CorrectText(s string) {
	st := lex.Pos{tb.Spell.SrcLn, tb.Spell.SrcCh} // start of word
	tb.RemoveTag(st, token.TextSpellErr)
	oend := st
	oend.Ch += len(tb.Spell.Word)
	tb.ReplaceText(st, oend, st, s, EditSignal, ReplaceNoMatchCase)
	if tb.CurView != nil {
		ep := st
		ep.Ch += len(s)
		tb.CurView.SetCursorShow(ep)
		tb.CurView = nil
	}
}

// CorrectClear clears the TextSpellErr tag for given word
func (tb *TextBuf) CorrectClear(s string) {
	st := lex.Pos{tb.Spell.SrcLn, tb.Spell.SrcCh} // start of word
	tb.RemoveTag(st, token.TextSpellErr)
}

// SpellCheckLineErrs runs spell check on given line, and returns Lex tags
// with token.TextSpellErr for any misspelled words
func (tb *TextBuf) SpellCheckLineErrs(ln int) lex.Line {
	if !tb.IsValidLine(ln) {
		return nil
	}
	tb.LinesMu.RLock()
	defer tb.LinesMu.RUnlock()
	tb.MarkupMu.RLock()
	defer tb.MarkupMu.RUnlock()
	return spell.CheckLexLine(tb.Lines[ln], tb.HiTags[ln])
}

// SpellCheckLineTag runs spell check on given line, and sets Tags for any
// misspelled words and updates markup for that line.
func (tb *TextBuf) SpellCheckLineTag(ln int) {
	if !tb.IsValidLine(ln) {
		return
	}
	ser := tb.SpellCheckLineErrs(ln)
	tb.MarkupMu.Lock()
	ntgs := tb.AdjustedTags(ln)
	ntgs.DeleteToken(token.TextSpellErr)
	for _, t := range ser {
		ntgs.AddSort(t)
	}
	tb.Tags[ln] = ntgs
	tb.MarkupMu.Unlock()
	tb.MarkupLinesLock(ln, ln)
	tb.StartDelayedReMarkup()
}

///////////////////////////////////////////////////////////////////
//  Diff

// DiffBufs computes the diff between this buffer and the other buffer,
// reporting a sequence of operations that would convert this buffer (a) into
// the other buffer (b).  Each operation is either an 'r' (replace), 'd'
// (delete), 'i' (insert) or 'e' (equal).  Everything is line-based (0, offset).
func (tb *TextBuf) DiffBufs(ob *TextBuf) textbuf.Diffs {
	astr := tb.Strings(false)
	bstr := ob.Strings(false)
	return textbuf.DiffLines(astr, bstr)
}

// DiffBufsUnified computes the diff between this buffer and the other buffer,
// returning a unified diff with given amount of context (default of 3 will be
// used if -1)
func (tb *TextBuf) DiffBufsUnified(ob *TextBuf, context int) []byte {
	astr := tb.Strings(true) // needs newlines for some reason
	bstr := ob.Strings(true)

	return textbuf.DiffLinesUnified(astr, bstr, context, string(tb.Filename), tb.Info.ModTime.String(),
		string(ob.Filename), ob.Info.ModTime.String())
}

// PatchFromBuf patches (edits) this buffer using content from other buffer,
// according to diff operations (e.g., as generated from DiffBufs).  signal
// determines whether each patch is signaled -- if an overall signal will be
// sent at the end, then that would not be necessary (typical)
func (tb *TextBuf) PatchFromBuf(ob *TextBuf, diffs textbuf.Diffs, signal bool) bool {
	bufUpdt, winUpdt, autoSave := tb.BatchUpdateStart()
	defer tb.BatchUpdateEnd(bufUpdt, winUpdt, autoSave)

	sz := len(diffs)
	mods := false
	for i := sz - 1; i >= 0; i-- { // go in reverse so changes are valid!
		df := diffs[i]
		switch df.Tag {
		case 'r':
			tb.DeleteText(lex.Pos{Ln: df.I1}, lex.Pos{Ln: df.I2}, signal)
			// fmt.Printf("patch rep del: %v %v\n", tbe.Reg, string(tbe.ToBytes()))
			ot := ob.Region(lex.Pos{Ln: df.J1}, lex.Pos{Ln: df.J2})
			tb.InsertText(lex.Pos{Ln: df.I1}, ot.ToBytes(), signal)
			// fmt.Printf("patch rep ins: %v %v\n", tbe.Reg, string(tbe.ToBytes()))
			mods = true
		case 'd':
			tb.DeleteText(lex.Pos{Ln: df.I1}, lex.Pos{Ln: df.I2}, signal)
			// fmt.Printf("patch del: %v %v\n", tbe.Reg, string(tbe.ToBytes()))
			mods = true
		case 'i':
			ot := ob.Region(lex.Pos{Ln: df.J1}, lex.Pos{Ln: df.J2})
			tb.InsertText(lex.Pos{Ln: df.I1}, ot.ToBytes(), signal)
			// fmt.Printf("patch ins: %v %v\n", tbe.Reg, string(tbe.ToBytes()))
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
