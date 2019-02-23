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

	"github.com/goki/gi/gi"
	"github.com/goki/gi/histyle"
	"github.com/goki/gi/spell"
	"github.com/goki/ki"
	"github.com/goki/ki/indent"
	"github.com/goki/ki/ints"
	"github.com/goki/ki/kit"
	"github.com/goki/ki/nptime"
	"github.com/goki/ki/runes"
	"github.com/goki/pi/complete"
	"github.com/goki/pi/filecat"
	"github.com/goki/pi/lex"
	"github.com/goki/pi/pi"
	"github.com/goki/pi/token"
	"github.com/pmezard/go-difflib/difflib"
)

// TextBufOpts contains options for TextBufs -- contains everything necessary to
// conditionalize editing of a given text file
type TextBufOpts struct {
	SpaceIndent  bool   `desc:"use spaces, not tabs, for indentation -- tab-size property in TextStyle has the tab size, used for either tabs or spaces"`
	TabSize      int    `desc:"size of a tab, in chars -- also determines indent level for space indent"`
	AutoIndent   bool   `desc:"auto-indent on newline (enter) or tab"`
	LineNos      bool   `desc:"show line numbers at left end of editor"`
	Completion   bool   `desc:"use the completion system to suggest options while typing"`
	SpellCorrect bool   `desc:"use spell checking to suggest corrections while typing"`
	EmacsUndo    bool   `desc:"use emacs-style undo, where after a non-undo command, all the current undo actions are added to the undo stack, such that a subsequent undo is actually a redo"`
	DepthColor   bool   `desc:"colorize the background according to nesting depth"`
	CommentLn    string `desc:"character(s) that start a single-line comment -- if empty then multi-line comment syntax will be used"`
	CommentSt    string `desc:"character(s) that start a multi-line comment or one that requires both start and end"`
	CommentEd    string `desc:"character(s) that end a multi-line comment or one that requires both start and end"`
}

// MaxScopeLines	 is the maximum lines to search for a scope marker, e.g. '}'
var MaxScopeLines = 100

// TextBufDiffRevertLines is max number of lines to use the diff-based revert, which results
// in faster reverts but only if the file isn't too big..
var TextBufDiffRevertLines = 10000

// TextBufDiffRevertDiffs is max number of difference regions to apply for diff-based revert
// otherwise just reopens file
var TextBufDiffRevertDiffs = 20

// CommentStrs returns the comment start and end strings, using line-based CommentLn first if set
// and falling back on multi-line / general purpose start / end syntax
func (tb *TextBufOpts) CommentStrs() (comst, comed string) {
	comst = tb.CommentLn
	if comst == "" {
		comst = tb.CommentSt
		comed = tb.CommentEd
	}
	return
}

// IndentChar returns the indent character based on SpaceIndent option
func (tb *TextBufOpts) IndentChar() indent.Char {
	if tb.SpaceIndent {
		return indent.Space
	}
	return indent.Tab
}

// ConfigSupported configures options based on the supported language info in GoPi
// returns true if supported
func (tb *TextBufOpts) ConfigSupported(sup filecat.Supported) bool {
	if sup == filecat.NoSupport {
		return false
	}
	lp, ok := pi.StdLangProps[sup]
	if !ok {
		return false
	}
	tb.CommentLn = lp.CommentLn
	tb.CommentSt = lp.CommentSt
	tb.CommentEd = lp.CommentEd
	for _, flg := range lp.Flags {
		switch flg {
		case pi.IndentSpace:
			tb.SpaceIndent = true
		case pi.IndentTab:
			tb.SpaceIndent = false
		}
	}
	return true
}

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
	Txt          []byte           `json:"-" xml:"text" desc:"the current value of the entire text being edited -- using []byte slice for greater efficiency"`
	Autosave     bool             `desc:"if true, auto-save file after changes (in a separate routine)"`
	Opts         TextBufOpts      `desc:"options for how text editing / viewing works"`
	Filename     gi.FileName      `json:"-" xml:"-" desc:"filename of file last loaded or saved"`
	Info         FileInfo         `desc:"full info about file"`
	PiState      pi.FileState     `desc:"Pi parsing state info for file"`
	Hi           HiMarkup         `desc:"syntax highlighting markup parameters (language, style, etc)"`
	NLines       int              `json:"-" xml:"-" desc:"number of lines"`
	Lines        [][]rune         `json:"-" xml:"-" desc:"the live lines of text being edited, with latest modifications -- encoded as runes per line, which is necessary for one-to-one rune / glyph rendering correspondence -- all TextPos positions etc are in *rune* indexes, not byte indexes!"`
	LineBytes    [][]byte         `json:"-" xml:"-" desc:"the live lines of text being edited, with latest modifications -- encoded in bytes per line translated from Lines, and used for input to markup -- essential to use Lines and not LineBytes when dealing with TextPos positions, which are in runes"`
	Tags         []lex.Line       `json:"extra custom tagged regions for each line"`
	HiTags       []lex.Line       `json:"syntax highlighting tags -- auto-generated"`
	Markup       [][]byte         `json:"-" xml:"-" desc:"marked-up version of the edit text lines, after being run through the syntax highlighting process etc -- this is what is actually rendered"`
	ByteOffs     []int            `json:"-" xml:"-" desc:"offsets for start of each line in Txt []byte slice -- this is NOT updated with edits -- call SetByteOffs to set it when needed -- used for re-generating the Txt in LinesToBytes, and set on initial open in BytesToLines"`
	TotalBytes   int              `json:"-" xml:"-" desc:"total bytes in document -- see ByteOffs for when it is updated"`
	LinesMu      sync.RWMutex     `json:"-" xml:"-" desc:"mutex for updating lines"`
	MarkupMu     sync.RWMutex     `json:"-" xml:"-" desc:"mutex for updating markup"`
	TextBufSig   ki.Signal        `json:"-" xml:"-" view:"-" desc:"signal for buffer -- see TextBufSignals for the types"`
	Views        []*TextView      `json:"-" xml:"-" desc:"the TextViews that are currently viewing this buffer"`
	Undos        []*TextBufEdit   `json:"-" xml:"-" desc:"undo stack of edits"`
	UndoUndos    []*TextBufEdit   `json:"-" xml:"-" desc:"undo stack of *undo* edits -- added to "`
	UndoPos      int              `json:"-" xml:"-" desc:"undo position"`
	PosHistory   []TextPos        `json:"-" xml:"-" desc:"history of cursor positions -- can move back through them"`
	Complete     *gi.Complete     `json:"-" xml:"-" desc:"functions and data for text completion"`
	SpellCorrect *gi.SpellCorrect `json:"-" xml:"-" desc:"functions and data for spelling correction"`
	CurView      *TextView        `json:"-" xml:"-" desc:"current textview -- e.g., the one that initiated Complete or Correct process -- update cursor position in this view -- is reset to nil after usage always"`
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

// these extend NodeBase NodeFlags to hold TextBuf state
const (
	// TextBufAutoSaving is used in atomically safe way to protect autosaving
	TextBufAutoSaving gi.NodeFlags = gi.NodeFlagsN + iota

	// TextBufMarkingUp indicates current markup operation in progress -- don't redo
	TextBufMarkingUp

	// TextBufChanged indicates if the text has been changed (edited) relative to the
	// original, since last save
	TextBufChanged

	// TextBufFileModOk have already asked about fact that file has changed since being
	// opened, user is ok
	TextBufFileModOk
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
	tb.Refresh()
}

// EditDone finalizes any current editing, sends signal
func (tb *TextBuf) EditDone() {
	if tb.IsChanged() {
		tb.AutoSaveDelete()
		tb.ClearChanged()
		tb.LinesToBytes()
		tb.TextBufSig.Emit(tb.This(), int64(TextBufDone), tb.Txt)
	}
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
func (tb *TextBuf) SetHiStyle(style histyle.StyleName) {
	tb.MarkupMu.Lock()
	tb.Hi.Style = style
	tb.MarkupMu.Unlock()
}

// Defaults sets default parameters if they haven't been yet --
// if Hi.Style is empty, then it considers it to not have been set
func (tb *TextBuf) Defaults() {
	if tb.Hi.Style != "" {
		return
	}
	tb.SetHiStyle(histyle.StyleDefault)
	tb.Opts.AutoIndent = true
	tb.Opts.TabSize = 4
}

// Refresh signals any views to refresh views
func (tb *TextBuf) Refresh() {
	tb.TextBufSig.Emit(tb.This(), int64(TextBufNew), tb.Txt)
}

// todo: use https://github.com/andybalholm/crlf to deal with cr/lf etc --
// internally just use lf = \n

// New initializes a new buffer with n blank lines
func (tb *TextBuf) New(nlines int) {
	tb.Defaults()
	nlines = ints.MaxInt(nlines, 1)
	tb.LinesMu.Lock()
	tb.MarkupMu.Lock()
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

	tb.PiState.SetSrc(&tb.Lines, string(tb.Filename), tb.Info.Sup)
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
		if tb.SpellCorrect == nil {
			tb.SetSpellCorrect(tb, SpellCorrectEdit)
		}
		if tb.Complete == nil {
			tb.SetCompleter(&tb.PiState, CompletePi, CompleteEditPi)
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
		gi.ChoiceDialog(vp, gi.DlgOpts{Title: "File Changed on Disk",
			Prompt: fmt.Sprintf("File has changed on disk since being opened or saved by you -- what do you want to do?  File: %v", tb.Filename)},
			[]string{"Save To Different File", "Open From Disk, Losing Changes", "Ignore and Proceed"},
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
		gi.PromptDialog(vp, gi.DlgOpts{Title: "File could not be Opened", Prompt: err.Error()}, true, false, nil, nil)
		log.Println(err)
		return err
	}
	tb.SetName(string(filename)) // todo: modify in any way?

	// markup the first 100 lines
	mxhi := ints.MinInt(100, tb.NLines-1)
	tb.MarkupLinesLock(0, mxhi)

	// update views
	tb.TextBufSig.Emit(tb.This(), int64(TextBufNew), tb.Txt)

	// do slow full update in background
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
				gi.PromptDialog(vp, gi.DlgOpts{Title: "File could not be Re-Opened", Prompt: err.Error()}, true, false, nil, nil)
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
	for _, tve := range tb.Views {
		tve.SetBuf(nil) // automatically disconnects signals, views
	}
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
func (tb *TextBuf) EndPos() TextPos {
	tb.LinesMu.RLock()
	defer tb.LinesMu.RUnlock()

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
		tb.TextBufSig.Emit(tb.This(), int64(TextBufInsert), tbe)
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
	bufUpdt = tb.UpdateStart()
	autoSave = tb.AutoSaveOff()
	winUpdt = false
	vp := tb.ViewportFromView()
	if vp == nil || vp.Win == nil {
		return
	}
	winUpdt = vp.Win.UpdateStart()
	return
}

// BatchUpdateEnd call to complete BatchUpdateStart
func (tb *TextBuf) BatchUpdateEnd(bufUpdt, winUpdt, autoSave bool) {
	tb.AutoSaveRestore(autoSave)
	if winUpdt {
		vp := tb.ViewportFromView()
		if vp != nil && vp.Win != nil {
			vp.Win.UpdateEnd(winUpdt)
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
		tb.Markup[ln] = HTMLEscapeBytes(tb.LineBytes[ln])
		bo += len(txt) + 1 // lf
	}
	tb.TotalBytes = bo
	tb.LinesMu.Unlock()
}

/////////////////////////////////////////////////////////////////////////////
//   Search

// FileSearchMatch records one match for search within file
type FileSearchMatch struct {
	Reg  TextRegion `desc:"region surrounding the match"`
	Text []byte     `desc:"text surrounding the match, at most FileSearchContext on either side (within a single line)"`
}

// FileSearchContext is how much text to include on either side of the search match
var FileSearchContext = 30

var mst = []byte("<mark>")
var mstsz = len(mst)
var med = []byte("</mark>")
var medsz = len(med)

// NewFileSearchMatch returns a new FileSearchMatch entry for given rune line with match starting
// at st and ending before ed, on given line
func NewFileSearchMatch(rn []rune, st, ed, ln int) FileSearchMatch {
	sz := len(rn)
	reg := NewTextRegion(ln, st, ln, ed)
	cist := ints.MaxInt(st-FileSearchContext, 0)
	cied := ints.MinInt(ed+FileSearchContext, sz)
	sctx := []byte(string(rn[cist:st]))
	fstr := []byte(string(rn[st:ed]))
	ectx := []byte(string(rn[ed:cied]))
	tlen := mstsz + medsz + len(sctx) + len(fstr) + len(ectx)
	txt := make([]byte, tlen)
	copy(txt, sctx)
	ti := st - cist
	copy(txt[ti:], mst)
	ti += mstsz
	copy(txt[ti:], fstr)
	ti += len(fstr)
	copy(txt[ti:], med)
	ti += medsz
	copy(txt[ti:], ectx)
	return FileSearchMatch{Reg: reg, Text: txt}
}

// Search looks for a string (no regexp) within buffer, with given case-sensitivity
// returning number of occurrences and specific match position list.
// column positions are in runes
func (tb *TextBuf) Search(find []byte, ignoreCase bool) (int, []FileSearchMatch) {
	fr := bytes.Runes(find)
	fsz := len(fr)
	if fsz == 0 {
		return 0, nil
	}
	tb.LinesMu.RLock()
	defer tb.LinesMu.RUnlock()
	cnt := 0
	var matches []FileSearchMatch
	for ln, rn := range tb.Lines {
		sz := len(rn)
		ci := 0
		for ci < sz {
			var i int
			if ignoreCase {
				i = runes.IndexFold(rn[ci:], fr)
			} else {
				i = runes.Index(rn[ci:], fr)
			}
			if i < 0 {
				break
			}
			i += ci
			ci = i + fsz
			mat := NewFileSearchMatch(rn, i, ci, ln)
			matches = append(matches, mat)
			cnt++
		}
	}
	return cnt, matches
}

/////////////////////////////////////////////////////////////////////////////
//   TextPos, TextRegion, TextBufEdit

// TextPos represents line, character positions within the TextBuf and TextView
// the Ch character position is in *runes* not bytes!
type TextPos struct {
	Ln, Ch int
}

// TextPosZero is the uninitialized zero text position (which is
// still a valid position)
var TextPosZero = TextPos{}

// TextPosErr represents an error text position (-1 for both line and char)
// used as a return value for cases where error positions are possible
var TextPosErr = TextPos{-1, -1}

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

// TextRegion represents a text region as a start / end position, and includes
// a Time stamp for when the region was created as valid positions into the TextBuf.
// The character end position is an *exclusive* position (i.e., the region ends at
// the character just prior to that character) but the lines are always *inclusive*
// (i.e., it is the actual line, not the next line).
type TextRegion struct {
	Start TextPos
	End   TextPos
	Time  nptime.Time `desc:"time when region was set -- needed for updating locations in the text based on time stamp (using efficient non-pointer time)"`
}

// TextRegionNil is the empty (zero) text region -- all zeros
var TextRegionNil TextRegion

// IsNil checks if the region is empty, because the start is after or equal to the end
func (tr *TextRegion) IsNil() bool {
	return !tr.Start.IsLess(tr.End)
}

// TimeNow grabs the current time as the edit time
func (tr *TextRegion) TimeNow() {
	tr.Time.Now()
}

// NewTextRegion creates a new text region using separate line and char
// values for start and end, and also sets the time stamp to now
func NewTextRegion(stLn, stCh, edLn, edCh int) TextRegion {
	tr := TextRegion{Start: TextPos{Ln: stLn, Ch: stCh}, End: TextPos{Ln: edLn, Ch: edCh}}
	tr.TimeNow()
	return tr
}

// NewTextRegionPos creates a new text region using position values
// and also sets the time stamp to now
func NewTextRegionPos(st, ed TextPos) TextRegion {
	tr := TextRegion{Start: st, End: ed}
	tr.TimeNow()
	return tr
}

// IsAfterTime reports if this region's time stamp is after given time value
// if region Time stamp has not been set, it always returns true
func (tr *TextRegion) IsAfterTime(t time.Time) bool {
	if tr.Time.IsZero() {
		return true
	}
	return tr.Time.Time().After(t)
}

// FromString decodes text region from a string representation of form:
// [#]LxxCxx-LxxCxx -- used in e.g., URL links -- returns true if successful
func (tr *TextRegion) FromString(link string) bool {
	link = strings.TrimPrefix(link, "#")
	fmt.Sscanf(link, "L%dC%d-L%dC%d", &tr.Start.Ln, &tr.Start.Ch, &tr.End.Ln, &tr.End.Ch)
	tr.Start.Ln--
	tr.Start.Ch--
	tr.End.Ln--
	tr.End.Ch--
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

// TextBufEdit describes an edit action to a buffer -- this is the data passed
// via signals to viewers of the buffer.  Actions are only deletions and
// insertions (a change is a sequence of those, given normal editing
// processes).  The TextBuf always reflects the current state *after* the
// edit.
type TextBufEdit struct {
	Reg    TextRegion `desc:"region for the edit (start is same for previous and current, end is in original pre-delete text for a delete, and in new lines data for an insert.  Also contains the Time stamp for this edit."`
	Delete bool       `desc:"action is either a deletion or an insertion"`
	Text   [][]rune   `desc:"text to be inserted"`
}

// ToBytes returns the Text of this edit record to a byte string, with
// newlines at end of each line -- nil if Text is empty
func (te *TextBufEdit) ToBytes() []byte {
	if te == nil {
		return nil
	}
	sz := len(te.Text)
	if sz == 0 {
		return nil
	}
	if sz == 1 {
		return []byte(string(te.Text[0]))
	}
	tsz := 0
	for i := range te.Text {
		tsz += len(te.Text[i]) + 10 // don't bother converting to runes, just extra slack
	}
	b := make([]byte, 0, tsz)
	for i := range te.Text {
		b = append(b, []byte(string(te.Text[i]))...)
		if i < sz-1 {
			b = append(b, '\n')
		}
	}
	return b
}

// AdjustPosDel determines what to do with positions within deleted region
type AdjustPosDel int

// these are options for what to do with positions within deleted region
// for the AdjustPos function
const (
	// AdjustPosDelErr means return a TextPosErr when in deleted region
	AdjustPosDelErr AdjustPosDel = iota

	// AdjustPosDelStart means return start of deleted region
	AdjustPosDelStart

	// AdjustPosDelEnd means return end of deleted region
	AdjustPosDelEnd
)

// AdjustPos adjusts the given text position as a function of the edit.
// if the position was within a deleted region of text, del determines
// what is returned
func (te *TextBufEdit) AdjustPos(pos TextPos, del AdjustPosDel) TextPos {
	if te == nil {
		return pos
	}
	if pos.IsLess(te.Reg.Start) || pos == te.Reg.Start {
		return pos
	}
	dl := te.Reg.End.Ln - te.Reg.Start.Ln
	if pos.Ln > te.Reg.End.Ln {
		if te.Delete {
			pos.Ln -= dl
		} else {
			pos.Ln += dl
		}
		return pos
	}
	if te.Delete {
		if pos.Ln < te.Reg.End.Ln || pos.Ch < te.Reg.End.Ch {
			switch del {
			case AdjustPosDelStart:
				return te.Reg.Start
			case AdjustPosDelEnd:
				return te.Reg.End
			case AdjustPosDelErr:
				return TextPosErr
			}
		}
		// this means pos.Ln == te.Reg.End.Ln, Ch >= end
		if dl == 0 {
			pos.Ch -= (te.Reg.End.Ch - te.Reg.Start.Ch)
		} else {
			pos.Ch -= te.Reg.End.Ch
		}
	} else {
		if dl == 0 {
			pos.Ch += (te.Reg.End.Ch - te.Reg.Start.Ch)
		} else {
			pos.Ln += dl
		}
	}
	return pos
}

// AdjustPosIfAfterTime checks the time stamp and IfAfterTime,
// it adjusts the given text position as a function of the edit
// del determines what to do with positions within a deleted region
// either move to start or end of the region, or return an error.
func (te *TextBufEdit) AdjustPosIfAfterTime(pos TextPos, t time.Time, del AdjustPosDel) TextPos {
	if te == nil {
		return pos
	}
	if te.Reg.IsAfterTime(t) {
		return te.AdjustPos(pos, del)
	}
	return pos
}

// AdjustReg adjusts the given text region as a function of the edit, including
// checking that the timestamp on the region is after the edit time, if
// the region has a valid Time stamp (otherwise always does adjustment).
// If the starting position is within a deleted region, it is moved to the
// end of the deleted region, and if the ending position was within a deleted
// region, it is moved to the start.  If the region becomes empty, TextRegionNil
// will be returned.
func (te *TextBufEdit) AdjustReg(reg TextRegion) TextRegion {
	if te == nil {
		return reg
	}
	if !reg.Time.IsZero() && !te.Reg.IsAfterTime(reg.Time.Time()) {
		return reg
	}
	reg.Start = te.AdjustPos(reg.Start, AdjustPosDelEnd)
	reg.End = te.AdjustPos(reg.End, AdjustPosDelStart)
	if reg.IsNil() {
		return TextRegionNil
	}
	return reg
}

// PunctGpMatch returns the matching grouping punctuation for given rune, which must be
// a left or right brace {}, bracket [] or paren () -- also returns true if it is *right*
func PunctGpMatch(r rune) (match rune, right bool) {
	right = false
	switch r {
	case '{':
		match = '}'
	case '}':
		right = true
		match = '{'
	case '(':
		match = ')'
	case ')':
		right = true
		match = '('
	case '[':
		match = ']'
	case ']':
		right = true
		match = '['
	}
	return
}

// FindScopeMatch finds the brace or parenthesis that is the partner of the one passed to function
func (tb *TextBuf) FindScopeMatch(r rune, st TextPos) (en TextPos, found bool) {
	tb.LinesMu.RLock()
	defer tb.LinesMu.RUnlock()

	en.Ln = -1
	found = false
	match, rt := PunctGpMatch(r)
	var left int
	var right int
	if rt {
		right++
	} else {
		left++
	}
	ch := st.Ch
	ln := st.Ln
	max := tb.NLines - ln
	if MaxScopeLines < tb.NLines {
		max = ln + MaxScopeLines
	}
	txt := tb.Line(ln)
	if left > right {
		for l := ln; l < max; l++ {
			for i := ch + 1; i < len(txt); i++ {
				if txt[i] == r {
					left++
					continue
				}
				if txt[i] == match {
					right++
					if left == right {
						en.Ln = l
						en.Ch = i
						break
					}
				}
			}
			if en.Ln >= 0 {
				found = true
				break
			}
			ln++
			txt = tb.Line(ln)
			ch = -1
		}
	} else {
		for l := ln; l >= 0; l-- {
			ch = ints.MinInt(ch, len(txt))
			for i := ch - 1; i >= 0; i-- {
				if txt[i] == r {
					right++
					continue
				}
				if txt[i] == match {
					left++
					if left == right {
						en.Ln = l
						en.Ch = i
						break
					}
				}
			}
			if en.Ln >= 0 {
				found = true
				break
			}
			ln--
			txt = tb.Line(ln)
			ch = len(txt)
		}
	}
	return en, found
}

/////////////////////////////////////////////////////////////////////////////
//   Edits

// ValidPos returns a position that is in a valid range
func (tb *TextBuf) ValidPos(pos TextPos) TextPos {
	tb.LinesMu.RLock()
	defer tb.LinesMu.RUnlock()

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
// views after text lines have been updated.  Sets the timestamp on resulting TextBufEdit
// to now
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
	tb.FileModCheck() // note: could bail if modified but not clear that is better?
	tbe := tb.Region(st, ed)
	tb.SetChanged()
	tb.LinesMu.Lock()
	tbe.Delete = true
	if ed.Ln == st.Ln {
		tb.Lines[st.Ln] = append(tb.Lines[st.Ln][:st.Ch], tb.Lines[st.Ln][ed.Ch:]...)
		tb.LinesMu.Unlock()
		if saveUndo {
			tb.SaveUndo(tbe)
		}
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
		tb.LinesMu.Unlock()
		if saveUndo {
			tb.SaveUndo(tbe)
		}
		tb.LinesDeleted(tbe)
	}

	if signal {
		tb.TextBufSig.Emit(tb.This(), int64(TextBufDelete), tbe)
	}
	if tb.Autosave {
		go tb.AutoSave()
	}
	return tbe
}

// Insert inserts new text at given starting position, signaling views after
// text has been inserted.  Sets the timestamp on resulting TextBufEdit to now
func (tb *TextBuf) InsertText(st TextPos, text []byte, saveUndo, signal bool) *TextBufEdit {
	if len(text) == 0 {
		return nil
	}
	if len(tb.Lines) == 0 {
		tb.New(1)
	}
	st = tb.ValidPos(st)
	tb.FileModCheck()
	tb.LinesMu.Lock()
	tb.SetChanged()
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
		tb.LinesMu.Unlock()
		tbe = tb.Region(st, ed)
		if saveUndo {
			tb.SaveUndo(tbe)
		}
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
		tb.LinesMu.Unlock()
		tbe = tb.Region(st, ed)
		if saveUndo {
			tb.SaveUndo(tbe)
		}
		tb.LinesInserted(tbe)
	}
	if signal {
		tb.TextBufSig.Emit(tb.This(), int64(TextBufInsert), tbe)
	}
	if tb.Autosave {
		go tb.AutoSave()
	}
	return tbe
}

// Region returns a TextBufEdit representation of text between start and end positions
// returns nil if not a valid region.  sets the timestamp on the TextBufEdit to now
func (tb *TextBuf) Region(st, ed TextPos) *TextBufEdit {
	st = tb.ValidPos(st)
	ed = tb.ValidPos(ed)
	if st == ed {
		return nil
	}
	if !st.IsLess(ed) {
		log.Printf("giv.TextBuf : starting position must be less than ending!: st: %v, ed: %v\n", st, ed)
		return nil
	}
	tbe := &TextBufEdit{Reg: NewTextRegionPos(st, ed)}
	tb.LinesMu.RLock()
	defer tb.LinesMu.RUnlock()
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

// SavePosHistory saves the cursor position in history stack of cursor positions --
// tracks across views -- returns false if position was on same line as last one saved
func (tb *TextBuf) SavePosHistory(pos TextPos) bool {
	if tb.PosHistory == nil {
		tb.PosHistory = make([]TextPos, 0, 1000)
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

// LinesInserted inserts new lines in Markup corresponding to lines
// inserted in Lines text.  Locks and unlocks the Markup mutex
func (tb *TextBuf) LinesInserted(tbe *TextBufEdit) {
	stln := tbe.Reg.Start.Ln + 1
	nsz := (tbe.Reg.End.Ln - tbe.Reg.Start.Ln)

	tb.LinesMu.Lock()
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

	tb.PiState.Src.LinesInserted(stln, nsz)

	st, ed := tbe.Reg.Start.Ln, tbe.Reg.End.Ln
	bo := tb.ByteOffs[st]
	for ln := st; ln <= ed; ln++ {
		tb.LineBytes[ln] = []byte(string(tb.Lines[ln]))
		tb.Markup[ln] = HTMLEscapeBytes(tb.LineBytes[ln])
		tb.ByteOffs[ln] = bo
		bo += len(tb.LineBytes[ln]) + 1
	}
	tb.MarkupLines(st, ed)
	tb.MarkupMu.Unlock()
	tb.LinesMu.Unlock()
}

// LinesDeleted deletes lines in Markup corresponding to lines
// deleted in Lines text.  Locks and unlocks the Markup mutex.
func (tb *TextBuf) LinesDeleted(tbe *TextBufEdit) {
	tb.LinesMu.Lock()
	tb.MarkupMu.Lock()

	stln := tbe.Reg.Start.Ln
	edln := tbe.Reg.End.Ln

	tb.LineBytes = append(tb.LineBytes[:stln], tb.LineBytes[edln:]...)
	tb.Markup = append(tb.Markup[:stln], tb.Markup[edln:]...)
	tb.Tags = append(tb.Tags[:stln], tb.Tags[edln:]...)
	tb.HiTags = append(tb.HiTags[:stln], tb.HiTags[edln:]...)
	tb.ByteOffs = append(tb.ByteOffs[:stln], tb.ByteOffs[edln:]...)

	tb.PiState.Src.LinesDeleted(stln, edln)

	st := tbe.Reg.Start.Ln
	tb.LineBytes[st] = []byte(string(tb.Lines[st]))
	tb.Markup[st] = HTMLEscapeBytes(tb.LineBytes[st])
	tb.MarkupLines(st, st)
	tb.MarkupMu.Unlock()
	tb.LinesMu.Unlock()
	// probably don't need to do global markup here..
}

// LinesEdited re-marks-up lines in edit (typically only 1).  Locks and
// unlocks the Markup mutex.
func (tb *TextBuf) LinesEdited(tbe *TextBufEdit) {
	tb.LinesMu.Lock()
	tb.MarkupMu.Lock()

	st, ed := tbe.Reg.Start.Ln, tbe.Reg.End.Ln
	for ln := st; ln <= ed; ln++ {
		tb.LineBytes[ln] = []byte(string(tb.Lines[ln]))
		tb.Markup[ln] = HTMLEscapeBytes(tb.LineBytes[ln])
	}
	tb.MarkupLines(st, ed)
	tb.MarkupMu.Unlock()
	tb.LinesMu.Unlock()
	// probably don't need to do global markup here..
}

// IsMarkingUp is true if the MarkupAllLines process is currently running
func (tb *TextBuf) IsMarkingUp() bool {
	return tb.HasFlag(int(TextBufMarkingUp))
}

// ReMarkup runs re-markup on text in background
func (tb *TextBuf) ReMarkup() {
	if !tb.Hi.HasHi() || tb.NLines == 0 {
		return
	}
	if tb.IsMarkingUp() {
		return
	}
	go tb.MarkupAllLines()
}

// AdjustedTags updates tag positions for edits
func (tb *TextBuf) AdjustedTags(ln int) lex.Line {
	sz := len(tb.Tags[ln])
	if sz == 0 {
		return tb.Tags[ln]
	}
	ntags := make(lex.Line, 0, sz)
	for _, tg := range tb.Tags[ln] {
		reg := TextRegion{Start: TextPos{Ln: ln, Ch: tg.St}, End: TextPos{Ln: ln, Ch: tg.Ed}}
		reg.Time = tg.Time
		reg = tb.AdjustReg(reg)
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
// designed to be called in a separate goroutine
func (tb *TextBuf) MarkupAllLines() {
	if !tb.Hi.HasHi() || tb.NLines == 0 {
		return
	}
	if tb.IsMarkingUp() {
		return
	}
	tb.SetFlag(int(TextBufMarkingUp))

	tb.LinesToBytes()
	tb.MarkupMu.Lock()
	mtags, err := tb.Hi.MarkupTagsAll(tb.Txt)
	if err != nil {
		tb.MarkupMu.Unlock()
		tb.ClearFlag(int(TextBufMarkingUp))
		return
	}

	maxln := ints.MinInt(len(mtags), tb.NLines)
	if tb.Hi.UsingPi() {
		for ln := 0; ln < maxln; ln++ {
			tb.HiTags[ln] = tb.PiState.LexLine(ln) // does clone, combines comments too
		}
	} else {
		for ln := 0; ln < maxln; ln++ {
			tb.HiTags[ln] = mtags[ln] // chroma tags are freshly allocated
		}
	}
	for ln := 0; ln < maxln; ln++ {
		tb.Tags[ln] = tb.AdjustedTags(ln)
		tb.Markup[ln] = tb.Hi.MarkupLine(tb.LineBytes[ln], tb.HiTags[ln], tb.Tags[ln])
	}
	tb.MarkupMu.Unlock()
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
		tb.Markup[ln] = tb.Hi.MarkupLine(tb.LineBytes[ln], tb.HiTags[ln], nil)
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
		ltxt := tb.LineBytes[ln]
		mt, err := tb.Hi.MarkupTagsLine(ln, ltxt)
		if err == nil {
			tb.HiTags[ln] = mt
			tb.Markup[ln] = tb.Hi.MarkupLine(ltxt, mt, tb.AdjustedTags(ln))
		} else {
			tb.Markup[ln] = HTMLEscapeBytes(ltxt)
			allgood = false
		}
	}
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
func (tb *TextBuf) SaveUndo(tbe *TextBufEdit) {
	if tb.UndoPos < len(tb.Undos) {
		// fmt.Printf("undo resetting to pos: %v len was: %v\n", tb.UndoPos, len(tb.Undos))
		tb.Undos = tb.Undos[:tb.UndoPos]
	}
	// fmt.Printf("save undo pos: %v: %v\n", tb.UndoPos, string(tbe.ToBytes()))
	tb.Undos = append(tb.Undos, tbe)
	tb.UndoPos = len(tb.Undos)
}

// Undo undoes next item on the undo stack, and returns that record -- nil if no more
func (tb *TextBuf) Undo() *TextBufEdit {
	if tb.UndoPos == 0 {
		tb.ClearChanged()
		tb.AutoSaveDelete()
		return nil
	}
	tb.UndoPos--
	tbe := tb.Undos[tb.UndoPos]
	if tbe == nil {
		return nil
	}
	if tbe.Delete {
		// fmt.Printf("undo pos: %v undoing delete at: %v text: %v\n", tb.UndoPos, tbe.Reg, string(tbe.ToBytes()))
		tbe := tb.InsertText(tbe.Reg.Start, tbe.ToBytes(), false, true) // don't save to reg und
		if tb.Opts.EmacsUndo {
			tb.UndoUndos = append(tb.UndoUndos, tbe)
		}
	} else {
		// fmt.Printf("undo pos: %v undoing insert at: %v text: %v\n", tb.UndoPos, tbe.Reg, string(tbe.ToBytes()))
		tbe := tb.DeleteText(tbe.Reg.Start, tbe.Reg.End, false, true)
		if tb.Opts.EmacsUndo {
			tb.UndoUndos = append(tb.UndoUndos, tbe)
		}
	}
	return tbe
}

// EmacsUndoSave if EmacsUndo mode is active, saves the UndoUndos to the regular Undo stack
// at the end, and moves undo to the very end -- undo is a constant stream..
func (tb *TextBuf) EmacsUndoSave() {
	if !tb.Opts.EmacsUndo || len(tb.UndoUndos) == 0 {
		return
	}
	for _, utbe := range tb.UndoUndos {
		tb.Undos = append(tb.Undos, utbe)
	}
	tb.UndoPos = len(tb.Undos)
	// fmt.Printf("emacs undo save new pos: %v\n", tb.UndoPos)
	tb.UndoUndos = nil
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

// AdjustPos adjusts given text position, which was recorded at given time
// for any edits that have taken place since that time (using the Undo stack).
// del determines what to do with positions within a deleted region -- either move
// to start or end of the region, or return an error
func (tb *TextBuf) AdjustPos(pos TextPos, t time.Time, del AdjustPosDel) TextPos {
	for _, utbe := range tb.Undos {
		pos = utbe.AdjustPosIfAfterTime(pos, t, del)
		if pos == TextPosErr {
			return pos
		}
	}
	return pos
}

// AdjustReg adjusts given text region for any edits that
// have taken place since time stamp on region (using the Undo stack)
// If region was wholly within a deleted region, then TextRegionNil will be
// returned -- otherwise it is clipped appropriately as function of deletes.
func (tb *TextBuf) AdjustReg(reg TextRegion) TextRegion {
	for _, utbe := range tb.Undos {
		reg = utbe.AdjustReg(reg)
		if reg == TextRegionNil {
			return reg
		}
	}
	return reg
}

/////////////////////////////////////////////////////////////////////////////
//   Tags

// AddTag adds a new custom tag for given line, at given position
func (tb *TextBuf) AddTag(ln, st, ed int, tag token.Tokens) {
	if !tb.IsValidLine(ln) {
		return
	}
	tr := lex.NewLex(token.KeyToken{Tok: tag}, st, ed)
	tr.Time.Now()
	if len(tb.Tags[ln]) == 0 {
		tb.Tags[ln] = append(tb.Tags[ln], tr)
	} else {
		tb.Tags[ln] = tb.AdjustedTags(ln) // must re-adjust before adding new ones!
		tb.Tags[ln].AddSort(tr)
	}
	tb.MarkupLinesLock(ln, ln)
}

// AddTagEdit adds a new custom tag for given line, using TextBufEdit for location
func (tb *TextBuf) AddTagEdit(tbe *TextBufEdit, tag token.Tokens) {
	tb.AddTag(tbe.Reg.Start.Ln, tbe.Reg.Start.Ch, tbe.Reg.End.Ch, tag)
}

// TagAt returns tag at given text position, if one exists -- returns false if not
func (tb *TextBuf) TagAt(pos TextPos) (reg lex.Lex, ok bool) {
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
func (tb *TextBuf) RemoveTag(pos TextPos, tag token.Tokens) (reg lex.Lex, ok bool) {
	if !tb.IsValidLine(pos.Ln) {
		return
	}
	tb.Tags[pos.Ln] = tb.AdjustedTags(pos.Ln) // re-adjust for current info
	for i, t := range tb.Tags[pos.Ln] {
		if t.ContainsPos(pos.Ch) {
			if tag > 0 && t.Tok.Tok != tag {
				continue
			}
			tb.Tags[pos.Ln] = append(tb.Tags[pos.Ln][:i], tb.Tags[pos.Ln][i+1:]...)
			reg = t
			ok = true
			break
		}
	}
	if ok {
		tb.MarkupLinesLock(pos.Ln, pos.Ln)
	}
	return
}

/////////////////////////////////////////////////////////////////////////////
//   Indenting

// LineIndent returns the number of tabs or spaces at start of given line --
// if line starts with tabs, then those are counted, else spaces --
// combinations of tabs and spaces won't produce sensible results
func (tb *TextBuf) LineIndent(ln int, tabSz int) (n int, ichr indent.Char) {
	tb.LinesMu.RLock()
	defer tb.LinesMu.RUnlock()

	ichr = indent.Tab
	sz := len(tb.Lines[ln])
	if sz == 0 {
		return
	}
	txt := tb.Lines[ln]
	if txt[0] == ' ' {
		ichr = indent.Space
		n = 1
	} else if txt[0] != '\t' {
		return
	} else {
		n = 1
	}
	if ichr == indent.Space {
		for i := 1; i < sz; i++ {
			if txt[i] == ' ' {
				n++
			} else {
				n /= tabSz
				return
			}
		}
		n /= tabSz
		return
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

// IndentLine indents line by given number of tab stops, using tabs or spaces,
// for given tab size (if using spaces) -- either inserts or deletes to reach
// target
func (tb *TextBuf) IndentLine(ln, n int) *TextBufEdit {
	asv := tb.AutoSaveOff()
	defer tb.AutoSaveRestore(asv)

	tabSz := tb.Opts.TabSize
	ichr := indent.Tab
	if tb.Opts.SpaceIndent {
		ichr = indent.Space
	}

	curli, _ := tb.LineIndent(ln, tabSz)
	if n > curli {
		// fmt.Printf("autoindent: ins %v\n", n)
		return tb.InsertText(TextPos{Ln: ln}, indent.Bytes(ichr, n-curli, tabSz), true, true)
	} else if n < curli {
		spos := indent.Len(ichr, n, tabSz)
		cpos := indent.Len(ichr, curli, tabSz)
		tb.DeleteText(TextPos{Ln: ln, Ch: spos}, TextPos{Ln: ln, Ch: cpos}, true, true)
		// fmt.Printf("IndentLine deleted: %v at: %v\n", string(tbe.ToBytes()), tbe.Reg)
	}
	return nil
}

// PrevLineIndent returns previous line from given line that has indentation -- skips blank lines
func (tb *TextBuf) PrevLineIndent(ln int) (n int, ichr indent.Char, txt string) {
	ichr = tb.Opts.IndentChar()
	tabSz := tb.Opts.TabSize
	comst, _ := tb.Opts.CommentStrs()
	tb.LinesMu.RLock()
	defer tb.LinesMu.RUnlock()
	ln--
	for ln >= 0 {
		if len(tb.Lines[ln]) == 0 {
			ln--
			continue
		}
		n, ichr = tb.LineIndent(ln, tabSz)
		txt = strings.TrimSpace(string(tb.Lines[ln]))
		if cmidx := strings.Index(txt, comst); cmidx > 0 {
			txt = strings.TrimSpace(txt[:cmidx])
		}
		return
	}
	n = 0
	return
}

// AutoIndent indents given line to the level of the prior line, adjusted
// appropriately if the current line starts with one of the given un-indent
// strings, or the prior line ends with one of the given indent strings.  Will
// have to be replaced with a smarter parsing-based mechanism for indent /
// unindent but this will do for now.  Returns any edit that took place (could
// be nil), along with the auto-indented level and character position for the
// indent of the current line.
func (tb *TextBuf) AutoIndent(ln int, indents, unindents []string) (tbe *TextBufEdit, indLev, chPos int) {
	tabSz := tb.Opts.TabSize
	ichr := tb.Opts.IndentChar()

	li, _, prvln := tb.PrevLineIndent(ln)
	tb.LinesMu.RLock()
	curln := strings.TrimSpace(string(tb.Lines[ln]))
	tb.LinesMu.RUnlock()
	ind := false
	und := false
	for _, us := range unindents {
		if curln == us {
			und = true
			break
		}
	}
	if prvln != "" { // unindent overrides indent
		for _, is := range indents {
			if strings.HasSuffix(prvln, is) {
				ind = true
				break
			}
		}
	}
	switch {
	case ind && und:
		return tb.IndentLine(ln, li), li, indent.Len(ichr, li, tabSz)
	case ind:
		return tb.IndentLine(ln, li+1), li + 1, indent.Len(ichr, li+1, tabSz)
	case und:
		return tb.IndentLine(ln, li-1), li - 1, indent.Len(ichr, li-1, tabSz)
	default:
		return tb.IndentLine(ln, li), li, indent.Len(ichr, li, tabSz)
	}
}

// AutoIndentRegion does auto-indent over given region -- end is *exclusive*
func (tb *TextBuf) AutoIndentRegion(st, ed int, indents, unindents []string) {
	bufUpdt, winUpdt, autoSave := tb.BatchUpdateStart()
	defer tb.BatchUpdateEnd(bufUpdt, winUpdt, autoSave)

	for ln := st; ln < ed; ln++ {
		if ln >= tb.NLines {
			break
		}
		tb.AutoIndent(ln, indents, unindents)
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
func (tb *TextBuf) InComment(pos TextPos) bool {
	cs := tb.CommentStart(pos.Ln)
	if cs < 0 {
		return false
	}
	return pos.Ch > cs
}

// LineCommented returns true if the given line is a full-comment line (i.e., starts with a comment)
func (tb *TextBuf) LineCommented(ln int) bool {
	cs := tb.CommentStart(ln)
	if cs < 0 {
		return false
	}
	li, _ := tb.LineIndent(ln, tb.Opts.TabSize)
	return cs == li
}

// CommentRegion inserts comment marker on given lines -- end is *exclusive*
func (tb *TextBuf) CommentRegion(st, ed int) {
	bufUpdt, winUpdt, autoSave := tb.BatchUpdateStart()
	defer tb.BatchUpdateEnd(bufUpdt, winUpdt, autoSave)

	ch := 0
	li, _ := tb.LineIndent(st, tb.Opts.TabSize)
	if li > 0 {
		if tb.Opts.SpaceIndent {
			ch = tb.Opts.TabSize * li
		} else {
			ch = li
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
			tb.InsertText(TextPos{Ln: ln, Ch: ch}, []byte(comst), true, true)
			if comed != "" {
				lln := len(tb.Lines[ln])
				tb.InsertText(TextPos{Ln: ln, Ch: lln}, []byte(comed), true, true)
			}
		} else {
			idx := tb.CommentStart(ln)
			if idx >= 0 {
				tb.DeleteText(TextPos{Ln: ln, Ch: idx}, TextPos{Ln: ln, Ch: idx + len(comst)}, true, true)
			}
			if comed != "" {
				idx := runes.IndexFold(tb.Line(ln), []rune(comed))
				if idx >= 0 {
					tb.DeleteText(TextPos{Ln: ln, Ch: idx}, TextPos{Ln: ln, Ch: idx + len(comed)}, true, true)
				}
			}
		}
	}
}

///////////////////////////////////////////////////////////////////////////////
//    Complete and Spell

// SetCompleter sets completion functions so that completions will
// automatically be offered as the user types
func (tb *TextBuf) SetCompleter(data interface{}, matchFun complete.MatchFunc, editFun complete.EditFunc) {
	if matchFun == nil || editFun == nil {
		if tb.Complete != nil {
			tb.Complete.CompleteSig.Disconnect(tb.This())
		}
		tb.Complete.Destroy()
		tb.Complete = nil
		return
	}
	if tb.Complete != nil {
		if tb.Complete.Context == data {
			tb.Complete.MatchFunc = matchFun
			tb.Complete.EditFunc = editFun
			return
		}
	}
	tb.Complete = &gi.Complete{}
	tb.Complete.InitName(tb.Complete, "tb-completion") // needed for standalone Ki's
	tb.Complete.Context = data
	tb.Complete.MatchFunc = matchFun
	tb.Complete.EditFunc = editFun
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

// CompleteText edits the text using the string chosen from the completion menu
func (tb *TextBuf) CompleteText(s string) {
	if s == "" {
		return
	}
	// give the completer a chance to edit the completion before insert,
	// also it return a number of runes past the cursor to delete
	st := TextPos{tb.Complete.SrcLn, 0}
	en := TextPos{tb.Complete.SrcLn, tb.LineLen(tb.Complete.SrcLn)}
	var tbes string
	tbe := tb.Region(st, en)
	if tbe != nil {
		tbes = string(tbe.ToBytes())
	}
	c := tb.Complete.GetCompletion(s)
	pos := TextPos{tb.Complete.SrcLn, tb.Complete.SrcCh}
	ed := tb.Complete.EditFunc(tb.Complete.Context, tbes, tb.Complete.SrcCh, c, tb.Complete.Seed)
	if ed.ForwardDelete > 0 {
		delEn := TextPos{tb.Complete.SrcLn, tb.Complete.SrcCh + ed.ForwardDelete}
		tb.DeleteText(pos, delEn, true, false)
	}
	// now the normal completion insertion
	st = pos
	st.Ch -= len(tb.Complete.Seed)
	tb.DeleteText(st, pos, true, false)
	tb.InsertText(st, []byte(ed.NewText), true, true)
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
	pos := TextPos{tb.Complete.SrcLn, tb.Complete.SrcCh}
	st := pos
	st.Ch -= len(tb.Complete.Seed)
	tb.DeleteText(st, pos, true, false)
	tb.InsertText(st, []byte(s), true, true)
	if tb.CurView != nil {
		ep := st
		ep.Ch += len(s)
		tb.CurView.SetCursorShow(ep)
		tb.CurView = nil
	}
}

// IsSpellCorrectEnabled returns true if spelling correction is enabled,
// taking into account given position in text if it is relevant for cases
// where it is only conditionally enabled
func (tb *TextBuf) IsSpellCorrectEnabled(pos TextPos) bool {
	if tb.SpellCorrect == nil || !tb.Opts.SpellCorrect {
		return false
	}
	switch tb.Info.Cat {
	case filecat.Doc: // always
		return true
	case filecat.Code:
		return tb.InComment(pos)
	default:
		return false
	}
}

// SetSpellCorrect sets spell correct functions so that spell correct will
// automatically be offered as the user types
func (tb *TextBuf) SetSpellCorrect(data interface{}, editFun spell.EditFunc) {
	if editFun == nil {
		if tb.SpellCorrect != nil {
			tb.SpellCorrect.SpellSig.Disconnect(tb.This())
		}
		tb.SpellCorrect.Destroy()
		tb.SpellCorrect = nil
		return
	}
	if tb.SpellCorrect != nil {
		if tb.SpellCorrect.Context == data {
			tb.SpellCorrect.EditFunc = editFun
			return
		}
	}
	gi.InitSpell()
	tb.SpellCorrect = &gi.SpellCorrect{}
	tb.SpellCorrect.InitName(tb.SpellCorrect, "tb-spellcorrect") // needed for standalone Ki's
	tb.SpellCorrect.Context = data
	tb.SpellCorrect.EditFunc = editFun
	// note: only need to connect once..
	tb.SpellCorrect.SpellSig.ConnectOnly(tb.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
		if sig == int64(gi.SpellSelect) {
			tbf, _ := recv.Embed(KiT_TextBuf).(*TextBuf)
			tbf.CorrectText(data.(string)) // always use data
		} else if sig == int64(gi.SpellIgnore) {
			tbf, _ := recv.Embed(KiT_TextBuf).(*TextBuf)
			tbf.CorrectText(data.(string)) // always use data
		}
	})
}

// CorrectText edits the text using the string chosen from the correction menu
func (tb *TextBuf) CorrectText(s string) {
	st := TextPos{tb.SpellCorrect.SrcLn, tb.SpellCorrect.SrcCh} // start of word
	tb.RemoveTag(st, token.TextSpellErr)
	oend := st
	oend.Ch += len(tb.SpellCorrect.Word)
	ed := tb.SpellCorrect.EditFunc(tb.SpellCorrect.Context, s, tb.SpellCorrect.Word)
	tb.DeleteText(st, oend, true, true)
	tb.InsertText(st, []byte(ed.NewText), true, true)
	if tb.CurView != nil {
		ep := st
		ep.Ch += len(ed.NewText)
		tb.CurView.SetCursorShow(ep)
		tb.CurView = nil
	}
}

func (tb *TextBuf) CorrectClear(s string) {
	st := TextPos{tb.SpellCorrect.SrcLn, tb.SpellCorrect.SrcCh} // start of word
	tb.RemoveTag(st, token.TextSpellErr)
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
	tb.LinesMu.RLock()
	ob.LinesMu.RLock()
	defer tb.LinesMu.RUnlock()
	defer ob.LinesMu.RUnlock()
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
	tb.LinesMu.RLock()
	ob.LinesMu.RLock()
	defer tb.LinesMu.RUnlock()
	defer ob.LinesMu.RUnlock()
	if tb.NLines == 0 || ob.NLines == 0 {
		return nil
	}
	astr := make([]string, tb.NLines)
	bstr := make([]string, ob.NLines)

	for ai, al := range tb.Lines {
		astr[ai] = string(al) + "\n"
	}
	for bi, bl := range ob.Lines {
		bstr[bi] = string(bl) + "\n"
	}

	ud := difflib.UnifiedDiff{A: astr, FromFile: string(tb.Filename), FromDate: tb.Info.ModTime.String(),
		B: bstr, ToFile: string(ob.Filename), ToDate: ob.Info.ModTime.String(), Context: context}
	var buf bytes.Buffer
	difflib.WriteUnifiedDiff(&buf, ud)
	return buf.Bytes()
}

// PrintDiffs prints out the diffs
func PrintDiffs(diffs TextDiffs) {
	for _, df := range diffs {
		switch df.Tag {
		case 'r':
			fmt.Printf("delete lines: %v - %v, insert lines: %v - %v\n", df.I1, df.I2, df.J1, df.J2)
		case 'd':
			fmt.Printf("delete lines: %v - %v\n", df.I1, df.I2)
		case 'i':
			fmt.Printf("insert lines at %v: %v - %v\n", df.I1, df.J1, df.J2)
		}
	}
}

// PatchFromBuf patches (edits) this buffer using content from other buffer,
// according to diff operations (e.g., as generated from DiffBufs).  signal
// determines whether each patch is signaled -- if an overall signal will be
// sent at the end, then that would not be necessary (typical)
func (tb *TextBuf) PatchFromBuf(ob *TextBuf, diffs TextDiffs, signal bool) bool {
	bufUpdt, winUpdt, autoSave := tb.BatchUpdateStart()
	defer tb.BatchUpdateEnd(bufUpdt, winUpdt, autoSave)

	sz := len(diffs)
	mods := false
	for i := sz - 1; i >= 0; i-- { // go in reverse so changes are valid!
		df := diffs[i]
		switch df.Tag {
		case 'r':
			tb.DeleteText(TextPos{Ln: df.I1}, TextPos{Ln: df.I2}, false, signal)
			// fmt.Printf("patch rep del: %v %v\n", tbe.Reg, string(tbe.ToBytes()))
			ot := ob.Region(TextPos{Ln: df.J1}, TextPos{Ln: df.J2})
			tb.InsertText(TextPos{Ln: df.I1}, ot.ToBytes(), false, signal)
			// fmt.Printf("patch rep ins: %v %v\n", tbe.Reg, string(tbe.ToBytes()))
			mods = true
		case 'd':
			tb.DeleteText(TextPos{Ln: df.I1}, TextPos{Ln: df.I2}, false, signal)
			// fmt.Printf("patch del: %v %v\n", tbe.Reg, string(tbe.ToBytes()))
			mods = true
		case 'i':
			ot := ob.Region(TextPos{Ln: df.J1}, TextPos{Ln: df.J2})
			tb.InsertText(TextPos{Ln: df.I1}, ot.ToBytes(), false, signal)
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
