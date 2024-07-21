// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package texteditor

import (
	"bytes"
	"fmt"
	"image"
	"io/fs"
	"log"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/base/fileinfo"
	"cogentcore.org/core/base/fsx"
	"cogentcore.org/core/base/indent"
	"cogentcore.org/core/base/runes"
	"cogentcore.org/core/base/slicesx"
	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/parse"
	"cogentcore.org/core/parse/complete"
	"cogentcore.org/core/parse/lexer"
	"cogentcore.org/core/parse/token"
	"cogentcore.org/core/spell"
	"cogentcore.org/core/texteditor/highlighting"
	"cogentcore.org/core/texteditor/textbuf"
)

// Buffer is a buffer of text, which can be viewed by [Editor](s).
// It holds the raw text lines (in original string and rune formats,
// and marked-up from syntax highlighting), and sends signals for making
// edits to the text and coordinating those edits across multiple views.
// Editors always only view a single buffer, so they directly call methods
// on the buffer to drive updates, which are then broadcast.
// It also has methods for loading and saving buffers to files.
// Unlike GUI widgets, its methods generally send events, without an
// explicit Event suffix.
// Internally, the buffer represents new lines using \n = LF, but saving
// and loading can deal with Windows/DOS CRLF format.
type Buffer struct { //types:add

	// Filename is the filename of the file that was last loaded or saved.
	// It is used when highlighting code.
	Filename core.Filename `json:"-" xml:"-"`

	// text is the current value of the entire text being edited,
	// represented as a byte slice for efficiency.
	text []byte

	// Autosave specifies whether the file should be automatically
	// saved after changes are made.
	Autosave bool

	// Options are the options for how text editing and viewing works.
	Options textbuf.Options

	// Info is the full information about the current file.
	Info fileinfo.FileInfo

	// ParseState is the parsing state information for the file.
	ParseState parse.FileStates

	// Highlighting is the syntax highlighting markup parameters, such as the language and style.
	Highlighting Highlighting

	// NumLines is the number of lines in the buffer.
	NumLines int `json:"-" xml:"-"`

	// LineColors are the colors to use for rendering circles
	// next to the line numbers of certain lines.
	LineColors map[int]image.Image

	// Lines are the live lines of text being edited, with the latest modifications.
	// They are encoded as runes per line, which is necessary for one-to-one rune/glyph
	// rendering correspondence. All TextPos positions are in rune indexes, not byte
	// indexes.
	Lines [][]rune `json:"-" xml:"-"`

	// lineBytes are the live lines of text being edited, with the latest modifications.
	// They are encoded in bytes per line, translated from Lines, and used for input to
	// markup. It is essential to use Lines and not lineBytes when dealing with TextPos
	// positions, which are in runes.
	lineBytes [][]byte

	// tags are the extra custom tagged regions for each line.
	tags []lexer.Line

	// hiTags are the syntax highlighting tags, which are auto-generated.
	hiTags []lexer.Line

	// Markup is the marked-up version of the edited text lines, after being run
	// through the syntax highlighting process. This is what is actually rendered.
	Markup [][]byte `json:"-" xml:"-"`

	// markupEdits are the edits that have been made since the last full markup.
	markupEdits []*textbuf.Edit

	// byteOffsets are the offsets for the start of each line in the [Buffer.text] byte
	// slice. This is not updated with edits. Call SetByteOffs to set it when needed.
	// It is used for re-generating the text in LinesToBytes and set on initial open in
	// BytesToLines.
	byteOffsets []int

	// totalBytes is the total number of bytes in the document.
	// See byteOffsets for when it is updated.
	totalBytes int

	// LinesMu is the mutex for updating lines.
	LinesMu sync.RWMutex `json:"-" xml:"-"`

	// markupMu is the mutex for updating markup.
	markupMu sync.RWMutex

	// markupDelayTimer is the markup delay timer.
	markupDelayTimer *time.Timer

	// markupDelayMu is the mutex for updating the markup delay timer.
	markupDelayMu sync.Mutex

	// editors are the editors that are currently viewing this buffer.
	editors []*Editor

	// posHistory is the history of cursor positions.
	// It can be used to move back through them.
	posHistory []lexer.Pos

	// Complete is the functions and data for text completion.
	Complete *core.Complete `json:"-" xml:"-"`

	// spell is the functions and data for spelling correction.
	spell *spellCheck

	// currentEditor is the current text editor, such as the one that initiated the
	// Complete or Correct process. The cursor position in this view is updated, and
	// it is reset to nil after usage.
	currentEditor *Editor

	// listeners is used for sending standard system events.
	// Change is sent for BufferDone, BufferInsert, and BufferDelete.
	listeners events.Listeners

	// Bool flags:

	// autoSaving is used in atomically safe way to protect autosaving
	autoSaving bool

	// markingUp indicates current markup operation in progress -- don't redo
	markingUp bool

	// Changed indicates if the text has been Changed (edited) relative to the
	// original, since last EditDone
	Changed bool

	// NotSaved indicates if the text has been changed (edited) relative to the
	// original, since last Save
	NotSaved bool

	// fileModOK have already asked about fact that file has changed since being
	// opened, user is ok
	fileModOK bool
}

// NewBuffer makes a new [Buffer] with default settings
// and initializes it.
func NewBuffer() *Buffer {
	tb := &Buffer{}
	tb.SetHighlighting(highlighting.StyleDefault)
	tb.Options.EditorSettings = core.SystemSettings.Editor
	tb.SetText([]byte{}) // to initialize
	return tb
}

// bufferSignals are signals that [Buffer] can send to [Editor].
type bufferSignals int32 //enums:enum -trim-prefix buffer

const (
	// bufferDone means that editing was completed and applied to Txt field
	// -- data is Txt bytes
	bufferDone bufferSignals = iota

	// bufferNew signals that entirely new text is present.
	// All views should do full layout update.
	bufferNew

	// bufferMods signals that potentially diffuse modifications
	// have been made.  Views should do a Layout and Render.
	bufferMods

	// bufferInsert signals that some text was inserted.
	// data is textbuf.Edit describing change.
	// The Buf always reflects the current state *after* the edit.
	bufferInsert

	// bufferDelete signals that some text was deleted.
	// data is textbuf.Edit describing change.
	// The Buf always reflects the current state *after* the edit.
	bufferDelete

	// bufferMarkupUpdated signals that the Markup text has been updated
	// This signal is typically sent from a separate goroutine,
	// so should be used with a mutex
	bufferMarkupUpdated

	// bufferClosed signals that the textbuf was closed.
	bufferClosed
)

// signalEditors sends the given signal and optional edit info
// to all the [Editor]s for this [Buffer]
func (tb *Buffer) signalEditors(sig bufferSignals, edit *textbuf.Edit) {
	for _, vw := range tb.editors {
		vw.bufferSignal(sig, edit)
	}
	if sig == bufferDone {
		e := &events.Base{Typ: events.Change}
		e.Init()
		tb.listeners.Call(e)
	} else if sig == bufferInsert || sig == bufferDelete {
		e := &events.Base{Typ: events.Input}
		e.Init()
		tb.listeners.Call(e)
	}
}

// OnChange adds an event listener function for the [events.Change] event.
func (tb *Buffer) OnChange(fun func(e events.Event)) {
	tb.listeners.Add(events.Change, fun)
}

// OnInput adds an event listener function for the [events.Input] event.
func (tb *Buffer) OnInput(fun func(e events.Event)) {
	tb.listeners.Add(events.Input, fun)
}

// clearNotSaved sets Changed and NotSaved to false.
func (tb *Buffer) clearNotSaved() {
	tb.Changed = false
	tb.NotSaved = false
}

// setChanged sets Changed and NotSaved to true.
func (tb *Buffer) setChanged() {
	tb.Changed = true
	tb.NotSaved = true
}

// SetText sets the text to the given bytes.
func (tb *Buffer) SetText(text []byte) *Buffer {
	tb.text = text
	tb.bytesToLines()
	tb.initialMarkup()
	tb.signalEditors(bufferNew, nil)
	tb.reMarkup()
	return tb
}

// SetString sets the text to the given string.
func (tb *Buffer) SetString(txt string) *Buffer {
	return tb.SetText([]byte(txt))
}

func (tb *Buffer) Update() {
	tb.signalMods()
}

// setTextLines sets the text to given lines of bytes
// if cpy is true, make a copy of bytes -- otherwise use
func (tb *Buffer) setTextLines(lns [][]byte, cpy bool) {
	tb.LinesMu.Lock()
	tb.NumLines = len(lns)
	tb.LinesMu.Unlock()
	tb.NewBuffer(tb.NumLines)
	tb.LinesMu.Lock()
	bo := 0
	for ln, txt := range lns {
		tb.byteOffsets[ln] = bo
		tb.Lines[ln] = bytes.Runes(txt)
		if cpy {
			tb.lineBytes[ln] = make([]byte, len(txt))
			copy(tb.lineBytes[ln], txt)
		} else {
			tb.lineBytes[ln] = txt
		}
		tb.Markup[ln] = htmlEscapeRunes(tb.Lines[ln])
		bo += len(txt) + 1 // lf
	}
	tb.totalBytes = bo
	tb.LinesMu.Unlock()
	tb.linesToBytes()
	tb.initialMarkup()
	tb.signalEditors(bufferNew, nil)
	tb.reMarkup()
}

// editDone finalizes any current editing, sends signal
func (tb *Buffer) editDone() {
	tb.AutoSaveDelete()
	tb.Changed = false
	tb.linesToBytes()
	tb.signalEditors(bufferDone, nil)
}

// Text returns the current text as a []byte array, applying all current
// changes by calling editDone, which will generate a signal if there have been
// changes.
func (tb *Buffer) Text() []byte {
	tb.editDone()
	return tb.text
}

// String returns the current text as a string, applying all current
// changes by calling editDone, which will generate a signal if there have been
// changes.
func (tb *Buffer) String() string {
	return string(tb.Text())
}

// numLines is the concurrent-safe accessor to NLines
func (tb *Buffer) numLines() int {
	tb.LinesMu.RLock()
	defer tb.LinesMu.RUnlock()
	return tb.NumLines
}

// IsValidLine returns true if given line is in range
func (tb *Buffer) IsValidLine(ln int) bool {
	if ln < 0 {
		return false
	}
	nln := tb.numLines()
	return ln < nln
}

// line is the concurrent-safe accessor to specific line of Lines runes
func (tb *Buffer) line(ln int) []rune {
	tb.LinesMu.RLock()
	defer tb.LinesMu.RUnlock()
	if ln >= tb.NumLines || ln < 0 {
		return nil
	}
	return tb.Lines[ln]
}

// lineLen is the concurrent-safe accessor to length of specific Line of Lines runes
func (tb *Buffer) lineLen(ln int) int {
	tb.LinesMu.RLock()
	defer tb.LinesMu.RUnlock()
	if ln >= tb.NumLines || ln < 0 {
		return 0
	}
	return len(tb.Lines[ln])
}

// BytesLine is the concurrent-safe accessor to specific Line of lineBytes.
func (tb *Buffer) BytesLine(ln int) []byte {
	tb.LinesMu.RLock()
	defer tb.LinesMu.RUnlock()
	if ln >= tb.NumLines || ln < 0 {
		return nil
	}
	return tb.lineBytes[ln]
}

// SetLanguage sets the language for highlighting and updates
// the highlighting style and buffer accordingly.
func (tb *Buffer) SetLanguage(language string) *Buffer {
	language = strings.ToLower(language)
	tb.SetFilename("_placeholder." + language)
	tb.SetText(tb.text) // to update it
	return tb
}

// SetHighlighting sets the highlighting style of the buffer.
func (tb *Buffer) SetHighlighting(style core.HighlightingName) *Buffer {
	tb.markupMu.Lock()
	tb.Highlighting.setStyle(style)
	tb.markupMu.Unlock()
	return tb
}

// signalMods sends the BufMods signal for misc, potentially
// widespread modifications to buffer.
func (tb *Buffer) signalMods() {
	tb.signalEditors(bufferMods, nil)
}

// SetReadOnly sets whether the buffer is read-only.
func (tb *Buffer) SetReadOnly(readonly bool) *Buffer {
	tb.Undos.Off = readonly
	return tb
}

// SetFilename sets the filename associated with the buffer and updates
// the code highlighting information accordingly.
func (tb *Buffer) SetFilename(fn string) *Buffer {
	tb.Filename = core.Filename(fn)
	tb.Stat()
	tb.Highlighting.init(&tb.Info, &tb.ParseState)
	return tb
}

// todo: use https://github.com/andybalholm/crlf to deal with cr/lf etc --
// internally just use lf = \n

// New initializes a new buffer with n blank lines.
func (tb *Buffer) NewBuffer(nlines int) {
	nlines = max(nlines, 1)
	tb.LinesMu.Lock()
	tb.markupMu.Lock()
	tb.Undos.Reset()
	tb.Lines = make([][]rune, nlines)
	tb.lineBytes = make([][]byte, nlines)
	tb.tags = make([]lexer.Line, nlines)
	tb.hiTags = make([]lexer.Line, nlines)
	tb.Markup = make([][]byte, nlines)

	tb.byteOffsets = slicesx.SetLength(tb.byteOffsets, nlines)

	if nlines == 1 { // this is used for a new blank doc
		tb.byteOffsets[0] = 0 // by definition
		tb.Lines[0] = []rune("")
		tb.lineBytes[0] = []byte("")
		tb.Markup[0] = []byte("")
	}

	tb.NumLines = nlines

	tb.ParseState.SetSrc(string(tb.Filename), "", tb.Info.Known)
	tb.Highlighting.init(&tb.Info, &tb.ParseState)

	tb.markupMu.Unlock()
	tb.LinesMu.Unlock()
	tb.signalEditors(bufferNew, nil)
}

// Stat gets info about the file, including the highlighting language.
func (tb *Buffer) Stat() error {
	tb.fileModOK = false
	err := tb.Info.InitFile(string(tb.Filename))
	tb.ConfigKnown() // may have gotten file type info even if not existing
	return err
}

// ConfigKnown configures options based on the supported language info in parse.
// Returns true if supported.
func (tb *Buffer) ConfigKnown() bool {
	if tb.Info.Known != fileinfo.Unknown {
		if tb.spell == nil {
			tb.setSpell()
		}
		if tb.Complete == nil {
			tb.setCompleter(&tb.ParseState, completeParse, completeEditParse, lookupParse)
		}
		return tb.Options.ConfigKnown(tb.Info.Known)
	}
	return false
}

// FileModCheck checks if the underlying file has been modified since last
// Stat (open, save); if haven't yet prompted, user is prompted to ensure
// that this is OK. It returns true if the file was modified.
func (tb *Buffer) FileModCheck() bool {
	if tb.fileModOK {
		return false
	}
	info, err := os.Stat(string(tb.Filename))
	if err != nil {
		return false
	}
	if info.ModTime() != time.Time(tb.Info.ModTime) {
		if !tb.Changed { // we haven't edited: just revert
			tb.Revert()
			return true
		}
		sc := tb.sceneFromEditor()
		d := core.NewBody().AddTitle("File changed on disk: " + fsx.DirAndFile(string(tb.Filename))).
			AddText(fmt.Sprintf("File has changed on disk since being opened or saved by you; what do you want to do?  If you <code>Revert from Disk</code>, you will lose any existing edits in open buffer.  If you <code>Ignore and Proceed</code>, the next save will overwrite the changed file on disk, losing any changes there.  File: %v", tb.Filename))
		d.AddBottomBar(func(parent core.Widget) {
			core.NewButton(parent).SetText("Save as to different file").OnClick(func(e events.Event) {
				d.Close()
				core.CallFunc(sc, tb.SaveAs)
			})
			core.NewButton(parent).SetText("Revert from disk").OnClick(func(e events.Event) {
				d.Close()
				tb.Revert()
			})
			core.NewButton(parent).SetText("Ignore and proceed").OnClick(func(e events.Event) {
				d.Close()
				tb.fileModOK = true
			})
		})
		d.RunDialog(sc)
		return true
	}
	return false
}

// Open loads the given file into the buffer.
func (tb *Buffer) Open(filename core.Filename) error { //types:add
	err := tb.openFile(filename)
	if err != nil {
		return err
	}
	tb.initialMarkup()
	tb.signalEditors(bufferNew, nil)
	tb.reMarkup()
	return nil
}

// OpenFS loads the given file in the given filesystem into the buffer.
func (tb *Buffer) OpenFS(fsys fs.FS, filename string) error {
	txt, err := fs.ReadFile(fsys, filename)
	if err != nil {
		return err
	}
	tb.text = txt
	tb.SetFilename(filename)
	tb.bytesToLines()
	tb.initialMarkup()
	tb.signalEditors(bufferNew, nil)
	tb.reMarkup()
	return nil
}

// openFile just loads the given file into the buffer, without doing
// any markup or signaling. It is typically used in other functions or
// for temporary buffers.
func (tb *Buffer) openFile(filename core.Filename) error {
	txt, err := os.ReadFile(string(filename))
	if err != nil {
		return err
	}
	tb.text = txt
	tb.SetFilename(string(filename))
	tb.bytesToLines()
	return nil
}

// Revert re-opens text from the current file, if the filename is set; returns false if
// not. It uses an optimized diff-based update to preserve existing formatting, making it
// very fast if not very different.
func (tb *Buffer) Revert() bool { //types:add
	tb.stopDelayedReMarkup()
	tb.AutoSaveDelete() // justin case
	if tb.Filename == "" {
		return false
	}

	didDiff := false
	if tb.NumLines < diffRevertLines {
		ob := NewBuffer()
		err := ob.openFile(tb.Filename)
		if errors.Log(err) != nil {
			sc := tb.sceneFromEditor()
			if sc != nil { // only if viewing
				core.ErrorSnackbar(sc, err, "Error reopening file")
			}
			return false
		}
		tb.Stat() // "own" the new file..
		if ob.NumLines < diffRevertLines {
			diffs := tb.diffBuffers(ob)
			if len(diffs) < diffRevertDiffs {
				tb.patchFromBuffer(ob, diffs, true) // true = send sigs for each update -- better than full, assuming changes are minor
				didDiff = true
			}
		}
	}
	if !didDiff {
		tb.openFile(tb.Filename)
	}
	tb.clearNotSaved()
	tb.AutoSaveDelete()
	tb.signalEditors(bufferNew, nil)
	tb.reMarkup()
	return true
}

// SaveAsFunc saves the current text into the given file.
// Does an editDone first to save edits and checks for an existing file.
// If it does exist then prompts to overwrite or not.
// If afterFunc is non-nil, then it is called with the status of the user action.
func (tb *Buffer) SaveAsFunc(filename core.Filename, afterFunc func(canceled bool)) {
	// todo: filemodcheck!
	tb.editDone()
	if !errors.Log1(fsx.FileExists(string(filename))) {
		tb.saveFile(filename)
		if afterFunc != nil {
			afterFunc(false)
		}
	} else {
		sc := tb.sceneFromEditor()
		d := core.NewBody().AddTitle("File exists").
			AddText(fmt.Sprintf("The file already exists; do you want to overwrite it?  File: %v", filename))
		d.AddBottomBar(func(parent core.Widget) {
			d.AddCancel(parent).OnClick(func(e events.Event) {
				if afterFunc != nil {
					afterFunc(true)
				}
			})
			d.AddOK(parent).OnClick(func(e events.Event) {
				tb.saveFile(filename)
				if afterFunc != nil {
					afterFunc(false)
				}
			})
		})
		d.RunDialog(sc)
	}
}

// SaveAs saves the current text into given file; does an editDone first to save edits
// and checks for an existing file; if it does exist then prompts to overwrite or not.
func (tb *Buffer) SaveAs(filename core.Filename) { //types:add
	tb.SaveAsFunc(filename, nil)
}

// saveFile writes current buffer to file, with no prompting, etc
func (tb *Buffer) saveFile(filename core.Filename) error {
	err := os.WriteFile(string(filename), tb.text, 0644)
	if err != nil {
		core.ErrorSnackbar(tb.sceneFromEditor(), err)
		slog.Error(err.Error())
	} else {
		tb.clearNotSaved()
		tb.Filename = filename
		tb.Stat()
	}
	return err
}

// Save saves the current text into the current filename associated with this buffer.
func (tb *Buffer) Save() error { //types:add
	if tb.Filename == "" {
		return fmt.Errorf("core.Buf: filename is empty for Save")
	}
	tb.editDone()
	info, err := os.Stat(string(tb.Filename))
	if err == nil && info.ModTime() != time.Time(tb.Info.ModTime) {
		sc := tb.sceneFromEditor()
		d := core.NewBody().AddTitle("File Changed on Disk").
			AddText(fmt.Sprintf("File has changed on disk since you opened or saved it; what do you want to do?  File: %v", tb.Filename))
		d.AddBottomBar(func(parent core.Widget) {
			core.NewButton(parent).SetText("Save to different file").OnClick(func(e events.Event) {
				d.Close()
				core.CallFunc(sc, tb.SaveAs)
			})
			core.NewButton(parent).SetText("Open from disk, losing changes").OnClick(func(e events.Event) {
				d.Close()
				tb.Revert()
			})
			core.NewButton(parent).SetText("Save file, overwriting").OnClick(func(e events.Event) {
				d.Close()
				tb.saveFile(tb.Filename)
			})
		})
		d.RunDialog(sc)
	}
	return tb.saveFile(tb.Filename)
}

// Close closes the buffer, prompting to save if there are changes, and disconnects
// from editors. If afterFun is non-nil, then it is called with the status of the user
// action.
func (tb *Buffer) Close(afterFun func(canceled bool)) bool {
	if tb.Changed {
		tb.stopDelayedReMarkup()
		sc := tb.sceneFromEditor()
		if tb.Filename != "" {
			d := core.NewBody().AddTitle("Close without saving?").
				AddText(fmt.Sprintf("Do you want to save your changes to file: %v?", tb.Filename))
			d.AddBottomBar(func(parent core.Widget) {
				core.NewButton(parent).SetText("Cancel").OnClick(func(e events.Event) {
					d.Close()
					if afterFun != nil {
						afterFun(true)
					}
				})
				core.NewButton(parent).SetText("Close without saving").OnClick(func(e events.Event) {
					d.Close()
					tb.clearNotSaved()
					tb.AutoSaveDelete()
					tb.Close(afterFun)
				})
				core.NewButton(parent).SetText("Save").OnClick(func(e events.Event) {
					tb.Save()
					tb.Close(afterFun) // 2nd time through won't prompt
				})
			})
			d.RunDialog(sc)
		} else {
			d := core.NewBody().AddTitle("Close without saving?").
				AddText("Do you want to save your changes (no filename for this buffer yet)?  If so, Cancel and then do Save As")
			d.AddBottomBar(func(parent core.Widget) {
				d.AddCancel(parent).OnClick(func(e events.Event) {
					if afterFun != nil {
						afterFun(true)
					}
				})
				d.AddOK(parent).SetText("Close without saving").OnClick(func(e events.Event) {
					tb.clearNotSaved()
					tb.AutoSaveDelete()
					tb.Close(afterFun)
				})
			})
			d.RunDialog(sc)
		}
		return false // awaiting decisions..
	}
	tb.signalEditors(bufferClosed, nil)
	tb.NewBuffer(1)
	tb.Filename = ""
	tb.clearNotSaved()
	if afterFun != nil {
		afterFun(false)
	}
	return true
}

////////////////////////////////////////////////////////////////////////////////////////
//		AutoSave

// autoSaveOff turns off autosave and returns the
// prior state of Autosave flag.
// Call AutoSaveRestore with rval when done.
// See BatchUpdate methods for auto-use of this.
func (tb *Buffer) autoSaveOff() bool {
	asv := tb.Autosave
	tb.Autosave = false
	return asv
}

// autoSaveRestore restores prior Autosave setting,
// from AutoSaveOff
func (tb *Buffer) autoSaveRestore(asv bool) {
	tb.Autosave = asv
}

// AutoSaveFilename returns the autosave filename.
func (tb *Buffer) AutoSaveFilename() string {
	path, fn := filepath.Split(string(tb.Filename))
	if fn == "" {
		fn = "new_file"
	}
	asfn := filepath.Join(path, "#"+fn+"#")
	return asfn
}

// autoSave does the autosave -- safe to call in a separate goroutine
func (tb *Buffer) autoSave() error {
	if tb.autoSaving {
		return nil
	}
	tb.autoSaving = true
	asfn := tb.AutoSaveFilename()
	b := tb.linesToBytesCopy()
	err := os.WriteFile(asfn, b, 0644)
	if err != nil {
		log.Printf("core.Buf: Could not AutoSave file: %v, error: %v\n", asfn, err)
	}
	tb.autoSaving = false
	return err
}

// AutoSaveDelete deletes any existing autosave file
func (tb *Buffer) AutoSaveDelete() {
	asfn := tb.AutoSaveFilename()
	err := os.Remove(asfn)
	// the file may not exist, which is fine
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		errors.Log(err)
	}
}

// AutoSaveCheck checks if an autosave file exists; logic for dealing with
// it is left to larger app; call this before opening a file.
func (tb *Buffer) AutoSaveCheck() bool {
	asfn := tb.AutoSaveFilename()
	if _, err := os.Stat(asfn); os.IsNotExist(err) {
		return false // does not exist
	}
	return true
}

/////////////////////////////////////////////////////////////////////////////
//   Appending Lines

// endPos returns the ending position at end of buffer
func (tb *Buffer) endPos() lexer.Pos {
	tb.LinesMu.RLock()
	defer tb.LinesMu.RUnlock()

	if tb.NumLines == 0 {
		return lexer.PosZero
	}
	ed := lexer.Pos{tb.NumLines - 1, len(tb.Lines[tb.NumLines-1])}
	return ed
}

// AppendTextMarkup appends new text to end of buffer, using insert, returns
// edit, and uses supplied markup to render it.
func (tb *Buffer) AppendTextMarkup(text []byte, markup []byte, signal bool) *textbuf.Edit {
	if len(text) == 0 {
		return &textbuf.Edit{}
	}
	ed := tb.endPos()
	tbe := tb.insertText(ed, text, false) // no sig -- we do later

	st := tbe.Reg.Start.Ln
	el := tbe.Reg.End.Ln
	sz := (el - st) + 1
	msplt := bytes.Split(markup, []byte("\n"))
	if len(msplt) < sz {
		log.Printf("Buf AppendTextMarkup: markup text less than appended text: is: %v, should be: %v\n", len(msplt), sz)
		el = min(st+len(msplt)-1, el)
	}
	for ln := st; ln <= el; ln++ {
		tb.Markup[ln] = msplt[ln-st]
	}
	if signal {
		tb.signalEditors(bufferInsert, tbe)
	}
	return tbe
}

// AppendTextLineMarkup appends one line of new text to end of buffer, using
// insert, and appending a LF at the end of the line if it doesn't already
// have one. User-supplied markup is used. Returns the edit region.
func (tb *Buffer) AppendTextLineMarkup(text []byte, markup []byte, signal bool) *textbuf.Edit {
	ed := tb.endPos()
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
	tbe := tb.insertText(ed, efft, false)
	tb.Markup[tbe.Reg.Start.Ln] = markup
	if signal {
		tb.signalEditors(bufferInsert, tbe)
	}
	return tbe
}

/////////////////////////////////////////////////////////////////////////////
//   Editors

// addEditor adds a editor of this buffer, connecting our signals to the editor
func (tb *Buffer) addEditor(vw *Editor) {
	tb.editors = append(tb.editors, vw)
	// tb.BufSig.Connect(vw.This, ViewBufSigRecv)
}

// deleteEditor removes given editor from our buffer
func (tb *Buffer) deleteEditor(vw *Editor) {
	tb.markupMu.Lock()
	tb.LinesMu.Lock()
	defer func() {
		tb.LinesMu.Unlock()
		tb.markupMu.Unlock()
	}()

	for i, ede := range tb.editors {
		if ede == vw {
			tb.editors = append(tb.editors[:i], tb.editors[i+1:]...)
			break
		}
	}
}

// sceneFromEditor returns Scene from text editor, if avail
func (tb *Buffer) sceneFromEditor() *core.Scene {
	if len(tb.editors) > 0 {
		return tb.editors[0].Scene
	}
	return nil
}

// AutoScrollEditors ensures that our editors are always viewing the end of the buffer
func (tb *Buffer) AutoScrollEditors() {
	for _, ed := range tb.editors {
		if ed != nil && ed.This != nil {
			ed.renderLayout()
			ed.SetCursorTarget(tb.endPos())
		}
	}
}

// batchUpdateStart call this when starting a batch of updates.
// It calls AutoSaveOff and returns the prior state of that flag
// which must be restored using BatchUpdateEnd.
func (tb *Buffer) batchUpdateStart() (autoSave bool) {
	tb.Undos.NewGroup()
	autoSave = tb.autoSaveOff()
	return
}

// batchUpdateEnd call to complete BatchUpdateStart
func (tb *Buffer) batchUpdateEnd(autoSave bool) {
	tb.autoSaveRestore(autoSave)
}

/////////////////////////////////////////////////////////////////////////////
//   Accessing Text

// setByteOffs sets the byte offsets for each line into the raw text
func (tb *Buffer) setByteOffs() {
	bo := 0
	for ln, txt := range tb.lineBytes {
		tb.byteOffsets[ln] = bo
		bo += len(txt) + 1 // lf
	}
	tb.totalBytes = bo
}

// linesToBytes converts current Lines back to the [Buffer.text] slice of bytes.
func (tb *Buffer) linesToBytes() {
	tb.LinesMu.RLock()
	defer tb.LinesMu.RUnlock()

	if tb.NumLines == 0 {
		if tb.text != nil {
			tb.text = tb.text[:0]
		}
		return
	}

	txt := bytes.Join(tb.lineBytes, []byte("\n"))
	txt = append(txt, '\n')
	tb.text = txt
}

// linesToBytesCopy converts current Lines into a separate text byte copy --
// e.g., for autosave or other "offline" uses of the text -- doesn't affect
// byte offsets etc
func (tb *Buffer) linesToBytesCopy() []byte {
	tb.LinesMu.RLock()
	defer tb.LinesMu.RUnlock()

	txt := bytes.Join(tb.lineBytes, []byte("\n"))
	txt = append(txt, '\n')
	return txt
}

// bytesToLines converts current Txt bytes into lines, and initializes markup
// with raw text
func (tb *Buffer) bytesToLines() {
	if len(tb.text) == 0 {
		tb.NewBuffer(1)
		return
	}
	tb.LinesMu.Lock()
	lns := bytes.Split(tb.text, []byte("\n"))
	tb.NumLines = len(lns)
	if len(lns[tb.NumLines-1]) == 0 { // lines have lf at end typically
		tb.NumLines--
		lns = lns[:tb.NumLines]
	}
	tb.LinesMu.Unlock()
	tb.NewBuffer(tb.NumLines)
	tb.LinesMu.Lock()
	bo := 0
	for ln, txt := range lns {
		tb.byteOffsets[ln] = bo
		tb.Lines[ln] = bytes.Runes(txt)
		tb.lineBytes[ln] = make([]byte, len(txt))
		copy(tb.lineBytes[ln], txt)
		tb.Markup[ln] = htmlEscapeRunes(tb.Lines[ln])
		bo += len(txt) + 1 // lf
	}
	tb.totalBytes = bo
	tb.LinesMu.Unlock()
}

// Strings returns the current text as []string array.
// If addNewLine is true, each string line has a \n appended at end.
func (tb *Buffer) Strings(addNewLine bool) []string {
	tb.LinesMu.RLock()
	defer tb.LinesMu.RUnlock()
	str := make([]string, tb.NumLines)
	for i, l := range tb.Lines {
		str[i] = string(l)
		if addNewLine {
			str[i] += "\n"
		}
	}
	return str
}

// Search looks for a string (no regexp) within buffer,
// with given case-sensitivity, returning number of occurrences
// and specific match position list. Column positions are in runes.
func (tb *Buffer) Search(find []byte, ignoreCase, lexItems bool) (int, []textbuf.Match) {
	tb.LinesMu.RLock()
	defer tb.LinesMu.RUnlock()
	if lexItems {
		tb.markupMu.RLock()
		defer tb.markupMu.RUnlock()
		return textbuf.SearchLexItems(tb.Lines, tb.hiTags, find, ignoreCase)
	} else {
		return textbuf.SearchRuneLines(tb.Lines, find, ignoreCase)
	}
}

// SearchRegexp looks for a string (regexp) within buffer,
// returning number of occurrences and specific match position list.
// Column positions are in runes.
func (tb *Buffer) SearchRegexp(re *regexp.Regexp) (int, []textbuf.Match) {
	tb.LinesMu.RLock()
	defer tb.LinesMu.RUnlock()
	return textbuf.SearchByteLinesRegexp(tb.lineBytes, re)
}

// braceMatch finds the brace, bracket, or parens that is the partner
// of the one passed to function.
func (tb *Buffer) braceMatch(r rune, st lexer.Pos) (en lexer.Pos, found bool) {
	tb.LinesMu.RLock()
	defer tb.LinesMu.RUnlock()
	tb.markupMu.RLock()
	defer tb.markupMu.RUnlock()
	return lexer.BraceMatch(tb.Lines, tb.hiTags, r, st, maxScopeLines)
}

/////////////////////////////////////////////////////////////////////////////
//   Edits

// validPos returns a position that is in a valid range
func (tb *Buffer) validPos(pos lexer.Pos) lexer.Pos {
	tb.LinesMu.RLock()
	defer tb.LinesMu.RUnlock()

	if tb.NumLines == 0 {
		return lexer.PosZero
	}
	if pos.Ln < 0 {
		pos.Ln = 0
	}
	if pos.Ln >= len(tb.Lines) {
		pos.Ln = len(tb.Lines) - 1
		pos.Ch = len(tb.Lines[pos.Ln])
		return pos
	}
	pos.Ln = min(pos.Ln, len(tb.Lines)-1)
	llen := len(tb.Lines[pos.Ln])
	pos.Ch = min(pos.Ch, llen)
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
func (tb *Buffer) DeleteText(st, ed lexer.Pos, signal bool) *textbuf.Edit {
	st = tb.validPos(st)
	ed = tb.validPos(ed)
	if st == ed {
		return nil
	}
	if !st.IsLess(ed) {
		log.Printf("core.Buf DeleteText: starting position must be less than ending!: st: %v, ed: %v\n", st, ed)
		return nil
	}
	tb.FileModCheck()
	tb.setChanged()
	tb.LinesMu.Lock()
	tbe := tb.deleteTextImpl(st, ed)
	tb.saveUndo(tbe)
	tb.LinesMu.Unlock()
	if signal {
		tb.signalEditors(bufferDelete, tbe)
	}
	if tb.Autosave {
		go tb.autoSave()
	}
	return tbe
}

// deleteTextImpl deletes region of text between start and end positions.
// Sets the timestamp on resulting textbuf.Edit to now.  Must be called under
// LinesMu.Lock.
func (tb *Buffer) deleteTextImpl(st, ed lexer.Pos) *textbuf.Edit {
	tbe := tb.regionImpl(st, ed)
	if tbe == nil {
		return nil
	}
	tbe.Delete = true
	if ed.Ln == st.Ln {
		tb.Lines[st.Ln] = append(tb.Lines[st.Ln][:st.Ch], tb.Lines[st.Ln][ed.Ch:]...)
		tb.linesEdited(tbe)
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
		tb.NumLines = len(tb.Lines)
		tb.linesDeleted(tbe)
	}
	return tbe
}

// deleteTextRect deletes rectangular region of text between start, end
// defining the upper-left and lower-right corners of a rectangle.
// Fails if st.Ch >= ed.Ch. Sets the timestamp on resulting textbuf.Edit to now.
// An Undo record is automatically saved depending on Undo.Off setting.
func (tb *Buffer) deleteTextRect(st, ed lexer.Pos, signal bool) *textbuf.Edit {
	st = tb.validPos(st)
	ed = tb.validPos(ed)
	if st == ed {
		return nil
	}
	if !st.IsLess(ed) {
		log.Printf("core.Buf DeleteTextRect: starting position must be less than ending!: st: %v, ed: %v\n", st, ed)
		return nil
	}
	tb.FileModCheck()
	tb.setChanged()
	tb.LinesMu.Lock()
	tbe := tb.deleteTextRectImpl(st, ed)
	tb.saveUndo(tbe)
	tb.LinesMu.Unlock()
	if signal {
		tb.signalMods()
	}
	if tb.Autosave {
		go tb.autoSave()
	}
	return tbe
}

// deleteTextRectImpl deletes rectangular region of text between start, end
// defining the upper-left and lower-right corners of a rectangle.
// Fails if st.Ch >= ed.Ch. Sets the timestamp on resulting textbuf.Edit to now.
// Must be called under LinesMu.Lock.
func (tb *Buffer) deleteTextRectImpl(st, ed lexer.Pos) *textbuf.Edit {
	tbe := tb.regionRectImpl(st, ed)
	if tbe == nil {
		return nil
	}
	tbe.Delete = true
	for ln := st.Ln; ln <= ed.Ln; ln++ {
		ls := tb.Lines[ln]
		if len(ls) > st.Ch {
			if ed.Ch < len(ls)-1 {
				tb.Lines[ln] = append(ls[:st.Ch], ls[ed.Ch:]...)
			} else {
				tb.Lines[ln] = ls[:st.Ch]
			}
		}
	}
	tb.linesEdited(tbe)
	return tbe
}

// insertText is the primary method for inserting text into the buffer.
// It inserts new text at given starting position, optionally signaling
// views after text has been inserted.  Sets the timestamp on resulting Edit to now.
// An Undo record is automatically saved depending on Undo.Off setting.
func (tb *Buffer) insertText(st lexer.Pos, text []byte, signal bool) *textbuf.Edit {
	if len(text) == 0 {
		return nil
	}
	st = tb.validPos(st)
	tb.FileModCheck() // will just revert changes if shouldn't have changed
	tb.setChanged()
	if len(tb.Lines) == 0 {
		tb.NewBuffer(1)
	}
	tb.LinesMu.Lock()
	tbe := tb.insertTextImpl(st, text)
	tb.saveUndo(tbe)
	tb.LinesMu.Unlock()
	if signal {
		tb.signalEditors(bufferInsert, tbe)
	}
	if tb.Autosave {
		go tb.autoSave()
	}
	return tbe
}

// insertTextImpl does the raw insert of new text at given starting position, returning
// a new Edit with timestamp of Now.  LinesMu must be locked surrounding this call.
func (tb *Buffer) insertTextImpl(st lexer.Pos, text []byte) *textbuf.Edit {
	lns := bytes.Split(text, []byte("\n"))
	sz := len(lns)
	rs := bytes.Runes(lns[0])
	rsz := len(rs)
	ed := st
	var tbe *textbuf.Edit
	st.Ch = min(len(tb.Lines[st.Ln]), st.Ch)
	if sz == 1 {
		nt := append(tb.Lines[st.Ln], rs...) // first append to end to extend capacity
		copy(nt[st.Ch+rsz:], nt[st.Ch:])     // move stuff to end
		copy(nt[st.Ch:], rs)                 // copy into position
		tb.Lines[st.Ln] = nt
		ed.Ch += rsz
		tbe = tb.regionImpl(st, ed)
		tb.linesEdited(tbe)
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
		tb.NumLines = len(tb.Lines)
		ed.Ln += nsz
		ed.Ch = len(tb.Lines[ed.Ln])
		if eost != nil {
			tb.Lines[ed.Ln] = append(tb.Lines[ed.Ln], eost...)
		}
		tbe = tb.regionImpl(st, ed)
		tb.linesInserted(tbe)
	}
	return tbe
}

// insertTextRect inserts a rectangle of text defined in given textbuf.Edit record,
// (e.g., from RegionRect or DeleteRect), optionally signaling
// views after text has been inserted.
// Returns a copy of the Edit record with an updated timestamp.
// An Undo record is automatically saved depending on Undo.Off setting.
func (tb *Buffer) insertTextRect(tbe *textbuf.Edit, signal bool) *textbuf.Edit {
	if tbe == nil {
		return nil
	}
	tb.FileModCheck() // will just revert changes if shouldn't have changed
	tb.setChanged()
	tb.LinesMu.Lock()
	nln := tb.NumLines
	re := tb.insertTextRectImpl(tbe)
	tb.saveUndo(re)
	tb.LinesMu.Unlock()
	if signal {
		if re.Reg.End.Ln >= nln {
			ie := &textbuf.Edit{}
			ie.Reg.Start.Ln = nln - 1
			ie.Reg.End.Ln = re.Reg.End.Ln
			tb.signalEditors(bufferInsert, ie)
		} else {
			tb.signalMods()
		}
	}
	if tb.Autosave {
		go tb.autoSave()
	}
	return re
}

// insertTextRectImpl does the raw insert of new text at given starting position,
// using a Rect textbuf.Edit  (e.g., from RegionRect or DeleteRect).
// Returns a copy of the Edit record with an updated timestamp.
func (tb *Buffer) insertTextRectImpl(tbe *textbuf.Edit) *textbuf.Edit {
	st := tbe.Reg.Start
	ed := tbe.Reg.End
	nlns := (ed.Ln - st.Ln) + 1
	if nlns <= 0 {
		return nil
	}
	// make sure there are enough lines -- add as needed
	cln := len(tb.Lines)
	if cln == 0 {
		tb.NewBuffer(nlns)
	} else if cln <= ed.Ln {
		nln := (1 + ed.Ln) - cln
		tmp := make([][]rune, nln)
		tb.Lines = append(tb.Lines, tmp...) // first append to end to extend capacity
		tb.NumLines = len(tb.Lines)
		ie := &textbuf.Edit{}
		ie.Reg.Start.Ln = cln - 1
		ie.Reg.End.Ln = ed.Ln
		tb.linesInserted(ie)
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
	tb.linesEdited(re)
	return re
}

// Region returns a textbuf.Edit representation of text between start and end positions
// returns nil if not a valid region.  sets the timestamp on the textbuf.Edit to now
func (tb *Buffer) Region(st, ed lexer.Pos) *textbuf.Edit {
	st = tb.validPos(st)
	ed = tb.validPos(ed)
	tb.LinesMu.RLock()
	defer tb.LinesMu.RUnlock()
	return tb.regionImpl(st, ed)
}

// regionImpl returns a textbuf.Edit representation of text between
// start and end positions. Returns nil if not a valid region.
// Sets the timestamp on the textbuf.Edit to now.
// Impl version must be called under LinesMu.RLock or Lock
func (tb *Buffer) regionImpl(st, ed lexer.Pos) *textbuf.Edit {
	if st == ed || ed.IsLess(st) {
		return nil
	}
	if !st.IsLess(ed) {
		log.Printf("core.Buf.Region: starting position must be less than ending!: st: %v, ed: %v\n", st, ed)
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

// regionRect returns a textbuf.Edit representation of text between start and end positions
// as a rectangle,
// returns nil if not a valid region.  sets the timestamp on the textbuf.Edit to now
func (tb *Buffer) regionRect(st, ed lexer.Pos) *textbuf.Edit {
	st = tb.validPos(st)
	ed = tb.validPos(ed)
	tb.LinesMu.RLock()
	defer tb.LinesMu.RUnlock()
	return tb.regionRectImpl(st, ed)
}

// regionRectImpl returns a textbuf.Edit representation of rectangle of
// text between start (upper left) and end (bottom right) positions.
// Returns nil if not a valid region.
// All lines in Text are guaranteed to be of the same size,
// even if line had fewer chars.
// Sets the timestamp on the textbuf.Edit to now.
// Impl version must be called under LinesMu.RLock or Lock
func (tb *Buffer) regionRectImpl(st, ed lexer.Pos) *textbuf.Edit {
	if st == ed {
		return nil
	}
	if !st.IsLess(ed) || st.Ch >= ed.Ch {
		log.Printf("core.Buf.RegionRect: starting position must be less than ending!: st: %v, ed: %v\n", st, ed)
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
			sz := min(ll-st.Ch, nch)
			txt = make([]rune, sz, nch)
			edl := min(ed.Ch, ll)
			copy(txt, lr[st.Ch:edl])
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
// if matchCase is true, then the lexer.MatchCase function is called to match the
// case (upper / lower) of the new inserted text to that of the text being replaced.
// returns the textbuf.Edit for the inserted text.
func (tb *Buffer) ReplaceText(delSt, delEd, insPos lexer.Pos, insTxt string, signal, matchCase bool) *textbuf.Edit {
	if matchCase {
		red := tb.Region(delSt, delEd)
		cur := string(red.ToBytes())
		insTxt = lexer.MatchCase(cur, insTxt)
	}
	if len(insTxt) > 0 {
		tb.DeleteText(delSt, delEd, EditNoSignal)
		return tb.insertText(insPos, []byte(insTxt), signal)
	}
	return tb.DeleteText(delSt, delEd, signal)
}

// savePosHistory saves the cursor position in history stack of cursor positions --
// tracks across views -- returns false if position was on same line as last one saved
func (tb *Buffer) savePosHistory(pos lexer.Pos) bool {
	if tb.posHistory == nil {
		tb.posHistory = make([]lexer.Pos, 0, 1000)
	}
	sz := len(tb.posHistory)
	if sz > 0 {
		if tb.posHistory[sz-1].Ln == pos.Ln {
			return false
		}
	}
	tb.posHistory = append(tb.posHistory, pos)
	// fmt.Printf("saved pos hist: %v\n", pos)
	return true
}

/////////////////////////////////////////////////////////////////////////////
//   Syntax Highlighting Markup

// linesEdited re-marks-up lines in edit (typically only 1).  Locks and
// unlocks the Markup mutex.  Must be called under Lines mutex lock.
func (tb *Buffer) linesEdited(tbe *textbuf.Edit) {
	tb.markupMu.Lock()
	st, ed := tbe.Reg.Start.Ln, tbe.Reg.End.Ln
	for ln := st; ln <= ed; ln++ {
		tb.lineBytes[ln] = []byte(string(tb.Lines[ln]))
		tb.Markup[ln] = htmlEscapeRunes(tb.Lines[ln])
	}
	tb.markupLines(st, ed)
	tb.markupMu.Unlock()
	tb.StartDelayedReMarkup()
}

// linesInserted inserts new lines in Markup corresponding to lines
// inserted in Lines text.  Locks and unlocks the Markup mutex, and
// must be called under lines mutex
func (tb *Buffer) linesInserted(tbe *textbuf.Edit) {
	stln := tbe.Reg.Start.Ln + 1
	nsz := (tbe.Reg.End.Ln - tbe.Reg.Start.Ln)

	tb.markupMu.Lock()
	tb.markupEdits = append(tb.markupEdits, tbe)

	// LineBytes
	tmplb := make([][]byte, nsz)
	nlb := append(tb.lineBytes, tmplb...)
	copy(nlb[stln+nsz:], nlb[stln:])
	copy(nlb[stln:], tmplb)
	tb.lineBytes = nlb

	// Markup
	tmpmu := make([][]byte, nsz)
	nmu := append(tb.Markup, tmpmu...) // first append to end to extend capacity
	copy(nmu[stln+nsz:], nmu[stln:])   // move stuff to end
	copy(nmu[stln:], tmpmu)            // copy into position
	tb.Markup = nmu

	// Tags
	tmptg := make([]lexer.Line, nsz)
	ntg := append(tb.tags, tmptg...)
	copy(ntg[stln+nsz:], ntg[stln:])
	copy(ntg[stln:], tmptg)
	tb.tags = ntg

	// HiTags
	tmpht := make([]lexer.Line, nsz)
	nht := append(tb.hiTags, tmpht...)
	copy(nht[stln+nsz:], nht[stln:])
	copy(nht[stln:], tmpht)
	tb.hiTags = nht

	// ByteOffs -- maintain mem update
	tmpof := make([]int, nsz)
	nof := append(tb.byteOffsets, tmpof...)
	copy(nof[stln+nsz:], nof[stln:])
	copy(nof[stln:], tmpof)
	tb.byteOffsets = nof

	if tb.Highlighting.UsingParse() {
		pfs := tb.ParseState.Done()
		pfs.Src.LinesInserted(stln, nsz)
	}

	st, ed := tbe.Reg.Start.Ln, tbe.Reg.End.Ln
	bo := tb.byteOffsets[st]
	for ln := st; ln <= ed; ln++ {
		tb.lineBytes[ln] = []byte(string(tb.Lines[ln]))
		tb.Markup[ln] = htmlEscapeRunes(tb.Lines[ln])
		tb.byteOffsets[ln] = bo
		bo += len(tb.lineBytes[ln]) + 1
	}
	tb.markupLines(st, ed)
	tb.markupMu.Unlock()
	tb.StartDelayedReMarkup()
}

// linesDeleted deletes lines in Markup corresponding to lines
// deleted in Lines text.  Locks and unlocks the Markup mutex, and
// must be called under lines mutex.
func (tb *Buffer) linesDeleted(tbe *textbuf.Edit) {
	tb.markupMu.Lock()

	tb.markupEdits = append(tb.markupEdits, tbe)

	stln := tbe.Reg.Start.Ln
	edln := tbe.Reg.End.Ln

	tb.lineBytes = append(tb.lineBytes[:stln], tb.lineBytes[edln:]...)
	tb.Markup = append(tb.Markup[:stln], tb.Markup[edln:]...)
	tb.tags = append(tb.tags[:stln], tb.tags[edln:]...)
	tb.hiTags = append(tb.hiTags[:stln], tb.hiTags[edln:]...)
	tb.byteOffsets = append(tb.byteOffsets[:stln], tb.byteOffsets[edln:]...)

	if tb.Highlighting.UsingParse() {
		pfs := tb.ParseState.Done()
		pfs.Src.LinesDeleted(stln, edln)
	}

	st := tbe.Reg.Start.Ln
	tb.lineBytes[st] = []byte(string(tb.Lines[st]))
	tb.Markup[st] = htmlEscapeRunes(tb.Lines[st])
	tb.markupLines(st, st)
	tb.markupMu.Unlock()
	tb.StartDelayedReMarkup()
}

///////////////////////////////////////////////////////////////////////////////////////
//  Markup

// initialMarkup does the first-pass markup on the file
func (tb *Buffer) initialMarkup() {
	if tb.Highlighting.UsingParse() {
		fs := tb.ParseState.Done() // initialize
		fs.Src.SetBytes(tb.text)
	}
	mxhi := min(100, tb.NumLines-1)
	tb.markupAllLines(mxhi)
}

// StartDelayedReMarkup starts a timer for doing markup after an interval
func (tb *Buffer) StartDelayedReMarkup() {
	tb.markupDelayMu.Lock()
	defer tb.markupDelayMu.Unlock()
	if !tb.Highlighting.Has || tb.NumLines == 0 {
		return
	}
	if tb.markupDelayTimer != nil {
		tb.markupDelayTimer.Stop()
		tb.markupDelayTimer = nil
	}
	sc := tb.sceneFromEditor()
	_ = sc
	// if vp != nil {
	// 	cpop := vp.Win.CurPopup()
	// 	if core.PopupIsCompleter(cpop) {
	// 		return
	// 	}
	// }
	if tb.Complete != nil && tb.Complete.IsAboutToShow() {
		return
	}
	tb.markupDelayTimer = time.AfterFunc(markupDelay, func() {
		// fmt.Printf("delayed remarkup\n")
		tb.markupDelayTimer = nil
		tb.reMarkup()
	})
}

// stopDelayedReMarkup stops timer for doing markup after an interval
func (tb *Buffer) stopDelayedReMarkup() {
	tb.markupDelayMu.Lock()
	defer tb.markupDelayMu.Unlock()
	if tb.markupDelayTimer != nil {
		tb.markupDelayTimer.Stop()
		tb.markupDelayTimer = nil
	}
}

// reMarkup runs re-markup on text in background
func (tb *Buffer) reMarkup() {
	if !tb.Highlighting.Has || tb.NumLines == 0 {
		return
	}
	if tb.markingUp {
		return
	}
	go tb.markupAllLines(-1)
}

// adjustedTags updates tag positions for edits
// must be called under MarkupMu lock
func (tb *Buffer) adjustedTags(ln int) lexer.Line {
	return tb.AdjustedTagsImpl(tb.tags[ln], ln)
}

// AdjustedTagsImpl updates tag positions for edits, for given list of tags
func (tb *Buffer) AdjustedTagsImpl(tags lexer.Line, ln int) lexer.Line {
	sz := len(tags)
	if sz == 0 {
		return nil
	}
	ntags := make(lexer.Line, 0, sz)
	for _, tg := range tags {
		reg := textbuf.Region{Start: lexer.Pos{Ln: ln, Ch: tg.St}, End: lexer.Pos{Ln: ln, Ch: tg.Ed}}
		reg.Time = tg.Time
		reg = tb.Undos.AdjustRegion(reg)
		if !reg.IsNil() {
			ntr := ntags.AddLex(tg.Token, reg.Start.Ch, reg.End.Ch)
			ntr.Time.Now()
		}
	}
	// lexer.LexsCleanup(&ntags)
	return ntags
}

// markupAllLines does syntax highlighting markup for all lines in buffer,
// calling MarkupMu mutex when setting the marked-up lines with the result --
// designed to be called in a separate goroutine.
// if maxLines > 0 then it specifies a maximum number of lines (for InitialMarkup)
func (tb *Buffer) markupAllLines(maxLines int) {
	if !tb.Highlighting.Has || tb.NumLines == 0 {
		return
	}
	if tb.markingUp {
		return
	}
	tb.markingUp = true

	tb.markupMu.Lock()
	tb.markupEdits = nil
	tb.markupMu.Unlock()

	var txt []byte
	if maxLines > 0 {
		tb.LinesMu.RLock()
		mln := min(maxLines, len(tb.lineBytes))
		txt = bytes.Join(tb.lineBytes[:mln], []byte("\n"))
		txt = append(txt, '\n')
		tb.LinesMu.RUnlock()
	} else {
		txt = tb.linesToBytesCopy()
	}
	mtags, err := tb.Highlighting.markupTagsAll(txt) // does full parse, outside of markup lock
	if err != nil {
		tb.markingUp = false
		return
	}

	// by this point mtags could be out of sync with deletes that have happened
	tb.LinesMu.Lock()
	tb.markupMu.Lock()

	maxln := min(len(tb.Markup), tb.NumLines)
	if maxLines > 0 {
		maxln = min(maxln, maxLines)
	}

	if tb.Highlighting.UsingParse() {
		pfs := tb.ParseState.Done()
		// first update mtags with any changes since it was generated
		for _, tbe := range tb.markupEdits {
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
		tb.markupEdits = nil
		// if maxln > 1 && len(pfs.Src.Lexs)-1 != maxln {
		// 	fmt.Printf("error: markup out of sync: %v != %v len(Lexs)\n", maxln, len(pfs.Src.Lexs)-1)
		// }
		for ln := 0; ln < maxln; ln++ {
			tb.hiTags[ln] = pfs.LexLine(ln) // does clone, combines comments too
		}
	} else {
		// first update mtags with any changes since it was generated
		for _, tbe := range tb.markupEdits {
			if tbe.Delete {
				stln := tbe.Reg.Start.Ln
				edln := tbe.Reg.End.Ln
				mtags = append(mtags[:stln], mtags[edln:]...)
			} else {
				stln := tbe.Reg.Start.Ln + 1
				nlns := (tbe.Reg.End.Ln - tbe.Reg.Start.Ln)
				tmpht := make([]lexer.Line, nlns)
				nht := append(mtags, tmpht...)
				copy(nht[stln+nlns:], nht[stln:])
				copy(nht[stln:], tmpht)
				mtags = nht
			}
		}
		tb.markupEdits = nil
		// if maxln > 0 && len(mtags) != maxln {
		// 	fmt.Printf("error: markup out of sync: %v != %v len(mtags)\n", maxln, len(mtags))
		// }
		maxln = min(maxln, len(mtags))
		for ln := 0; ln < maxln; ln++ {
			tb.hiTags[ln] = mtags[ln] // chroma tags are freshly allocated
		}
	}
	for ln := 0; ln < maxln; ln++ {
		tb.tags[ln] = tb.adjustedTags(ln)
		tb.Markup[ln] = tb.Highlighting.markupLine(tb.Lines[ln], tb.hiTags[ln], tb.tags[ln])
	}
	tb.markupMu.Unlock()
	tb.LinesMu.Unlock()
	tb.markingUp = false
	tb.signalEditors(bufferMarkupUpdated, nil)
}

// markupLines generates markup of given range of lines. end is *inclusive*
// line.  returns true if all lines were marked up successfully.  This does
// NOT lock the MarkupMu mutex (done at outer loop)
func (tb *Buffer) markupLines(st, ed int) bool {
	if !tb.Highlighting.Has || tb.NumLines == 0 {
		return false
	}
	if ed >= tb.NumLines {
		ed = tb.NumLines - 1
	}

	allgood := true
	for ln := st; ln <= ed; ln++ {
		ltxt := tb.Lines[ln]
		mt, err := tb.Highlighting.markupTagsLine(ln, ltxt)
		if err == nil {
			tb.hiTags[ln] = mt
			tb.Markup[ln] = tb.Highlighting.markupLine(ltxt, mt, tb.adjustedTags(ln))
		} else {
			tb.Markup[ln] = htmlEscapeRunes(ltxt)
			allgood = false
		}
	}
	// Now we trigger a background reparse of everything in a separate parse.FilesState
	// that gets switched into the current.
	return allgood
}

// markupLinesLock does MarkupLines and gets the mutex lock first
func (tb *Buffer) markupLinesLock(st, ed int) bool {
	tb.markupMu.Lock()
	defer tb.markupMu.Unlock()
	return tb.markupLines(st, ed)
}

/////////////////////////////////////////////////////////////////////////////
//   Undo

// saveUndo saves given edit to undo stack
func (tb *Buffer) saveUndo(tbe *textbuf.Edit) {
	tb.Undos.Save(tbe)
}

// undo undoes next group of items on the undo stack
func (tb *Buffer) undo() *textbuf.Edit {
	tb.LinesMu.Lock()
	tbe := tb.Undos.UndoPop()
	if tbe == nil {
		tb.LinesMu.Unlock()
		tb.Changed = false
		tb.AutoSaveDelete()
		return nil
	}
	autoSave := tb.batchUpdateStart()
	defer tb.batchUpdateEnd(autoSave)
	stgp := tbe.Group
	last := tbe
	for {
		if tbe.Rect {
			if tbe.Delete {
				utbe := tb.insertTextRectImpl(tbe)
				utbe.Group = stgp + tbe.Group
				if tb.Options.EmacsUndo {
					tb.Undos.SaveUndo(utbe)
				}
				tb.LinesMu.Unlock()
				tb.signalMods()
			} else {
				utbe := tb.deleteTextRectImpl(tbe.Reg.Start, tbe.Reg.End)
				utbe.Group = stgp + tbe.Group
				if tb.Options.EmacsUndo {
					tb.Undos.SaveUndo(utbe)
				}
				tb.LinesMu.Unlock()
				tb.signalMods()
			}
		} else {
			if tbe.Delete {
				utbe := tb.insertTextImpl(tbe.Reg.Start, tbe.ToBytes())
				utbe.Group = stgp + tbe.Group
				if tb.Options.EmacsUndo {
					tb.Undos.SaveUndo(utbe)
				}
				tb.LinesMu.Unlock()
				tb.signalEditors(bufferInsert, utbe)
			} else {
				utbe := tb.deleteTextImpl(tbe.Reg.Start, tbe.Reg.End)
				utbe.Group = stgp + tbe.Group
				if tb.Options.EmacsUndo {
					tb.Undos.SaveUndo(utbe)
				}
				tb.LinesMu.Unlock()
				tb.signalEditors(bufferDelete, utbe)
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
		tb.Changed = false
		tb.AutoSaveDelete()
	}
	return last
}

// emacsUndoSave is called by View at end of latest set of undo commands.
// If EmacsUndo mode is active, saves the current UndoStack to the regular Undo stack
// at the end, and moves undo to the very end -- undo is a constant stream.
func (tb *Buffer) emacsUndoSave() {
	if !tb.Options.EmacsUndo {
		return
	}
	tb.Undos.UndoStackSave()
}

// redo redoes next group of items on the undo stack,
// and returns the last record, nil if no more
func (tb *Buffer) redo() *textbuf.Edit {
	tb.LinesMu.Lock()
	tbe := tb.Undos.RedoNext()
	if tbe == nil {
		tb.LinesMu.Unlock()
		return nil
	}
	autoSave := tb.batchUpdateStart()
	defer tb.batchUpdateEnd(autoSave)
	stgp := tbe.Group
	last := tbe
	for {
		if tbe.Rect {
			if tbe.Delete {
				tb.deleteTextRectImpl(tbe.Reg.Start, tbe.Reg.End)
				tb.LinesMu.Unlock()
				tb.signalMods()
			} else {
				tb.insertTextRectImpl(tbe)
				tb.LinesMu.Unlock()
				tb.signalMods()
			}
		} else {
			if tbe.Delete {
				tb.deleteTextImpl(tbe.Reg.Start, tbe.Reg.End)
				tb.LinesMu.Unlock()
				tb.signalEditors(bufferDelete, tbe)
			} else {
				tb.insertTextImpl(tbe.Reg.Start, tbe.ToBytes())
				tb.LinesMu.Unlock()
				tb.signalEditors(bufferInsert, tbe)
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

// AdjustRegion adjusts given text region for any edits that
// have taken place since time stamp on region (using the Undo stack).
// If region was wholly within a deleted region, then RegionNil will be
// returned -- otherwise it is clipped appropriately as function of deletes.
func (tb *Buffer) AdjustRegion(reg textbuf.Region) textbuf.Region {
	return tb.Undos.AdjustRegion(reg)
}

/////////////////////////////////////////////////////////////////////////////
//   Tags

// addTag adds a new custom tag for given line, at given position
func (tb *Buffer) addTag(ln, st, ed int, tag token.Tokens) {
	if !tb.IsValidLine(ln) {
		return
	}
	tb.markupMu.Lock()
	tr := lexer.NewLex(token.KeyToken{Token: tag}, st, ed)
	tr.Time.Now()
	if len(tb.tags[ln]) == 0 {
		tb.tags[ln] = append(tb.tags[ln], tr)
	} else {
		tb.tags[ln] = tb.adjustedTags(ln) // must re-adjust before adding new ones!
		tb.tags[ln].AddSort(tr)
	}
	tb.markupMu.Unlock()
	tb.markupLinesLock(ln, ln)
}

// addTagEdit adds a new custom tag for given line, using textbuf.Edit for location
func (tb *Buffer) addTagEdit(tbe *textbuf.Edit, tag token.Tokens) {
	tb.addTag(tbe.Reg.Start.Ln, tbe.Reg.Start.Ch, tbe.Reg.End.Ch, tag)
}

// removeTag removes tag (optionally only given tag if non-zero) at given position
// if it exists -- returns tag
func (tb *Buffer) removeTag(pos lexer.Pos, tag token.Tokens) (reg lexer.Lex, ok bool) {
	if !tb.IsValidLine(pos.Ln) {
		return
	}
	tb.markupMu.Lock()
	tb.tags[pos.Ln] = tb.adjustedTags(pos.Ln) // re-adjust for current info
	for i, t := range tb.tags[pos.Ln] {
		if t.ContainsPos(pos.Ch) {
			if tag > 0 && t.Token.Token != tag {
				continue
			}
			tb.tags[pos.Ln].DeleteIndex(i)
			reg = t
			ok = true
			break
		}
	}
	tb.markupMu.Unlock()
	if ok {
		tb.markupLinesLock(pos.Ln, pos.Ln)
	}
	return
}

// HiTagAtPos returns the highlighting (markup) lexical tag at given position
// using current Markup tags, and index, -- could be nil if none or out of range
func (tb *Buffer) HiTagAtPos(pos lexer.Pos) (*lexer.Lex, int) {
	tb.markupMu.Lock()
	defer tb.markupMu.Unlock()
	if !tb.IsValidLine(pos.Ln) {
		return nil, -1
	}
	return tb.hiTags[pos.Ln].AtPos(pos.Ch)
}

// LexObjPathString returns the string at given lex, and including prior
// lex-tagged regions that include sequences of PunctSepPeriod and NameTag
// which are used for object paths -- used for e.g., debugger to pull out
// variable expressions that can be evaluated.
func (tb *Buffer) LexObjPathString(ln int, lx *lexer.Lex) string {
	tb.LinesMu.RLock()
	defer tb.LinesMu.RUnlock()
	if !tb.IsValidLine(ln) {
		return ""
	}
	stlx := lexer.ObjPathAt(tb.hiTags[ln], lx)
	rns := tb.Lines[ln][stlx.St:lx.Ed]
	return string(rns)
}

// inTokenSubCat returns true if the given text position is marked with lexical
// type in given SubCat sub-category
func (tb *Buffer) inTokenSubCat(pos lexer.Pos, subCat token.Tokens) bool {
	lx, _ := tb.HiTagAtPos(pos)
	return lx != nil && lx.Token.Token.InSubCat(subCat)
}

// inLitString returns true if position is in a string literal
func (tb *Buffer) inLitString(pos lexer.Pos) bool {
	return tb.inTokenSubCat(pos, token.LitStr)
}

// inTokenCode returns true if position is in a Keyword, Name, Operator, or Punctuation.
// This is useful for turning off spell checking in docs
func (tb *Buffer) inTokenCode(pos lexer.Pos) bool {
	lx, _ := tb.HiTagAtPos(pos)
	if lx == nil {
		return false
	}
	return lx.Token.Token.IsCode()
}

/////////////////////////////////////////////////////////////////////////////
//   LineColors

// SetLineColor sets the color to use for rendering a circle next to the line
// number at the given line.
func (tb *Buffer) SetLineColor(ln int, color image.Image) {
	tb.LinesMu.Lock()
	defer tb.LinesMu.Unlock()
	if tb.LineColors == nil {
		tb.LineColors = make(map[int]image.Image)
	}
	tb.LineColors[ln] = color
}

// HasLineColor checks if given line has a line color set
func (tb *Buffer) HasLineColor(ln int) bool {
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

// DeleteLineColor deletes the line color at the given line.
func (tb *Buffer) DeleteLineColor(ln int) {
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

// see parse/lexer/indent.go for support functions

// indentLine indents line by given number of tab stops, using tabs or spaces,
// for given tab size (if using spaces) -- either inserts or deletes to reach target.
// Returns edit record for any change.
func (tb *Buffer) indentLine(ln, ind int) *textbuf.Edit {
	asv := tb.autoSaveOff()
	defer tb.autoSaveRestore(asv)

	tabSz := tb.Options.TabSize
	ichr := indent.Tab
	if tb.Options.SpaceIndent {
		ichr = indent.Space
	}

	tb.LinesMu.RLock()
	curind, _ := lexer.LineIndent(tb.Lines[ln], tabSz)
	tb.LinesMu.RUnlock()
	if ind > curind {
		return tb.insertText(lexer.Pos{Ln: ln}, indent.Bytes(ichr, ind-curind, tabSz), EditSignal)
	} else if ind < curind {
		spos := indent.Len(ichr, ind, tabSz)
		cpos := indent.Len(ichr, curind, tabSz)
		return tb.DeleteText(lexer.Pos{Ln: ln, Ch: spos}, lexer.Pos{Ln: ln, Ch: cpos}, EditSignal)
	}
	return nil
}

// autoIndent indents given line to the level of the prior line, adjusted
// appropriately if the current line starts with one of the given un-indent
// strings, or the prior line ends with one of the given indent strings.
// Returns any edit that took place (could be nil), along with the auto-indented
// level and character position for the indent of the current line.
func (tb *Buffer) autoIndent(ln int) (tbe *textbuf.Edit, indLev, chPos int) {
	tabSz := tb.Options.TabSize

	tb.LinesMu.RLock()
	tb.markupMu.RLock()
	lp, _ := parse.LanguageSupport.Properties(tb.ParseState.Sup)
	var pInd, delInd int
	if lp != nil && lp.Lang != nil {
		pInd, delInd, _, _ = lp.Lang.IndentLine(&tb.ParseState, tb.Lines, tb.hiTags, ln, tabSz)
	} else {
		pInd, delInd, _, _ = lexer.BracketIndentLine(tb.Lines, tb.hiTags, ln, tabSz)
	}
	tb.markupMu.RUnlock()
	tb.LinesMu.RUnlock()
	ichr := tb.Options.IndentChar()

	indLev = pInd + delInd
	chPos = indent.Len(ichr, indLev, tabSz)
	tbe = tb.indentLine(ln, indLev)
	return
}

// AutoIndentRegion does auto-indent over given region; end is *exclusive*
func (tb *Buffer) AutoIndentRegion(start, end int) {
	autoSave := tb.batchUpdateStart()
	defer tb.batchUpdateEnd(autoSave)
	for ln := start; ln < end; ln++ {
		if ln >= tb.NumLines {
			break
		}
		tb.autoIndent(ln)
	}
}

// commentStart returns the char index where the comment starts on given line, -1 if no comment
func (tb *Buffer) commentStart(ln int) int {
	if !tb.IsValidLine(ln) {
		return -1
	}
	comst, _ := tb.Options.CommentStrings()
	if comst == "" {
		return -1
	}
	tb.LinesMu.RLock()
	defer tb.LinesMu.RUnlock()
	return runes.Index(tb.line(ln), []rune(comst))
}

// inComment returns true if the given text position is within a commented region
func (tb *Buffer) inComment(pos lexer.Pos) bool {
	if tb.inTokenSubCat(pos, token.Comment) {
		return true
	}
	cs := tb.commentStart(pos.Ln)
	if cs < 0 {
		return false
	}
	return pos.Ch > cs
}

// lineCommented returns true if the given line is a full-comment line (i.e., starts with a comment)
func (tb *Buffer) lineCommented(ln int) bool {
	tb.markupMu.RLock()
	defer tb.markupMu.RUnlock()
	tags := tb.hiTags[ln]
	if len(tags) == 0 {
		return false
	}
	return tags[0].Token.Token.InCat(token.Comment)
}

// CommentRegion inserts comment marker on given lines; end is *exclusive*
func (tb *Buffer) CommentRegion(start, end int) {
	autoSave := tb.batchUpdateStart()
	defer tb.batchUpdateEnd(autoSave)

	tabSz := tb.Options.TabSize

	ch := 0
	tb.LinesMu.RLock()
	ind, _ := lexer.LineIndent(tb.Lines[start], tabSz)
	tb.LinesMu.RUnlock()

	if ind > 0 {
		if tb.Options.SpaceIndent {
			ch = tb.Options.TabSize * ind
		} else {
			ch = ind
		}
	}

	comst, comed := tb.Options.CommentStrings()
	if comst == "" {
		fmt.Printf("core.Buf: %v attempt to comment region without any comment syntax defined\n", tb.Filename)
		return
	}

	eln := min(tb.numLines(), end)
	ncom := 0
	nln := eln - start
	for ln := start; ln < eln; ln++ {
		if tb.lineCommented(ln) {
			ncom++
		}
	}
	trgln := max(nln-2, 1)
	doCom := true
	if ncom >= trgln {
		doCom = false
	}

	for ln := start; ln < eln; ln++ {
		if doCom {
			tb.insertText(lexer.Pos{Ln: ln, Ch: ch}, []byte(comst), EditSignal)
			if comed != "" {
				lln := len(tb.Lines[ln])
				tb.insertText(lexer.Pos{Ln: ln, Ch: lln}, []byte(comed), EditSignal)
			}
		} else {
			idx := tb.commentStart(ln)
			if idx >= 0 {
				tb.DeleteText(lexer.Pos{Ln: ln, Ch: idx}, lexer.Pos{Ln: ln, Ch: idx + len(comst)}, EditSignal)
			}
			if comed != "" {
				idx := runes.IndexFold(tb.line(ln), []rune(comed))
				if idx >= 0 {
					tb.DeleteText(lexer.Pos{Ln: ln, Ch: idx}, lexer.Pos{Ln: ln, Ch: idx + len(comed)}, EditSignal)
				}
			}
		}
	}
}

// JoinParaLines merges sequences of lines with hard returns forming paragraphs,
// separated by blank lines, into a single line per paragraph,
// within the given line regions; endLine is *inclusive*
func (tb *Buffer) JoinParaLines(startLine, endLine int) {
	autoSave := tb.batchUpdateStart()

	curEd := endLine                           // current end of region being joined == last blank line
	for ln := endLine; ln >= startLine; ln-- { // reverse order
		lb := tb.lineBytes[ln]
		lbt := bytes.TrimSpace(lb)
		if len(lbt) == 0 || ln == startLine {
			if ln < curEd-1 {
				stp := lexer.Pos{Ln: ln + 1}
				if ln == startLine {
					stp.Ln--
				}
				ep := lexer.Pos{Ln: curEd - 1}
				if curEd == endLine {
					ep.Ln = curEd
				}
				eln := tb.Lines[ep.Ln]
				ep.Ch = len(eln)
				tlb := bytes.Join(tb.lineBytes[stp.Ln:ep.Ln+1], []byte(" "))
				tb.ReplaceText(stp, ep, stp, string(tlb), EditSignal, ReplaceNoMatchCase)
			}
			curEd = ln
		}
	}
	tb.batchUpdateEnd(autoSave)
	tb.signalMods()
}

// tabsToSpaces replaces tabs with spaces in the given line.
func (tb *Buffer) tabsToSpaces(ln int) {
	tabSz := tb.Options.TabSize

	lr := tb.Lines[ln]
	st := lexer.Pos{Ln: ln}
	ed := lexer.Pos{Ln: ln}
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

// TabsToSpaces replaces tabs with spaces over given region; end is *exclusive*
func (tb *Buffer) TabsToSpaces(start, end int) {
	autoSave := tb.batchUpdateStart()
	for ln := start; ln < end; ln++ {
		if ln >= tb.NumLines {
			break
		}
		tb.tabsToSpaces(ln)
	}
	tb.batchUpdateEnd(autoSave)
	tb.signalMods()
}

// spacesToTabs replaces spaces with tabs in the given line.
func (tb *Buffer) spacesToTabs(ln int) {
	tabSz := tb.Options.TabSize

	lr := tb.Lines[ln]
	st := lexer.Pos{Ln: ln}
	ed := lexer.Pos{Ln: ln}
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

// SpacesToTabs replaces tabs with spaces over given region; end is *exclusive*
func (tb *Buffer) SpacesToTabs(start, end int) {
	autoSave := tb.batchUpdateStart()
	for ln := start; ln < end; ln++ {
		if ln >= tb.NumLines {
			break
		}
		tb.spacesToTabs(ln)
	}
	tb.batchUpdateEnd(autoSave)
	tb.signalMods()
}

///////////////////////////////////////////////////////////////////////////////
//    Complete and Spell

// setCompleter sets completion functions so that completions will
// automatically be offered as the user types
func (tb *Buffer) setCompleter(data any, matchFun complete.MatchFunc, editFun complete.EditFunc,
	lookupFun complete.LookupFunc) {
	if tb.Complete != nil {
		if tb.Complete.Context == data {
			tb.Complete.MatchFunc = matchFun
			tb.Complete.EditFunc = editFun
			tb.Complete.LookupFunc = lookupFun
			return
		}
		tb.deleteCompleter()
	}
	tb.Complete = core.NewComplete().SetContext(data).SetMatchFunc(matchFun).
		SetEditFunc(editFun).SetLookupFunc(lookupFun)
	tb.Complete.OnSelect(func(e events.Event) {
		tb.completeText(tb.Complete.Completion)
	})
	// todo: what about CompleteExtend event type?
	// TODO(kai/complete): clean this up and figure out what to do about Extend and only connecting once
	// note: only need to connect once..
	// tb.Complete.CompleteSig.ConnectOnly(func(dlg *core.Dialog) {
	// 	tbf, _ := recv.Embed(TypeBuf).(*Buf)
	// 	if sig == int64(core.CompleteSelect) {
	// 		tbf.CompleteText(data.(string)) // always use data
	// 	} else if sig == int64(core.CompleteExtend) {
	// 		tbf.CompleteExtend(data.(string)) // always use data
	// 	}
	// })
}

func (tb *Buffer) deleteCompleter() {
	if tb.Complete == nil {
		return
	}
	tb.Complete = nil
}

// completeText edits the text using the string chosen from the completion menu
func (tb *Buffer) completeText(s string) {
	if s == "" {
		return
	}
	// give the completer a chance to edit the completion before insert,
	// also it return a number of runes past the cursor to delete
	st := lexer.Pos{tb.Complete.SrcLn, 0}
	en := lexer.Pos{tb.Complete.SrcLn, tb.lineLen(tb.Complete.SrcLn)}
	var tbes string
	tbe := tb.Region(st, en)
	if tbe != nil {
		tbes = string(tbe.ToBytes())
	}
	c := tb.Complete.GetCompletion(s)
	pos := lexer.Pos{tb.Complete.SrcLn, tb.Complete.SrcCh}
	ed := tb.Complete.EditFunc(tb.Complete.Context, tbes, tb.Complete.SrcCh, c, tb.Complete.Seed)
	if ed.ForwardDelete > 0 {
		delEn := lexer.Pos{tb.Complete.SrcLn, tb.Complete.SrcCh + ed.ForwardDelete}
		tb.DeleteText(pos, delEn, EditNoSignal)
	}
	// now the normal completion insertion
	st = pos
	st.Ch -= len(tb.Complete.Seed)
	tb.ReplaceText(st, pos, st, ed.NewText, EditSignal, ReplaceNoMatchCase)
	if tb.currentEditor != nil {
		ep := st
		ep.Ch += len(ed.NewText) + ed.CursorAdjust
		tb.currentEditor.SetCursorShow(ep)
		tb.currentEditor = nil
	}
}

// isSpellEnabled returns true if spelling correction is enabled,
// taking into account given position in text if it is relevant for cases
// where it is only conditionally enabled
func (tb *Buffer) isSpellEnabled(pos lexer.Pos) bool {
	if tb.spell == nil || !tb.Options.SpellCorrect {
		return false
	}
	switch tb.Info.Cat {
	case fileinfo.Doc: // not in code!
		return !tb.inTokenCode(pos)
	case fileinfo.Code:
		return tb.inComment(pos) || tb.inLitString(pos)
	default:
		return false
	}
}

// setSpell sets spell correct functions so that spell correct will
// automatically be offered as the user types
func (tb *Buffer) setSpell() {
	if tb.spell != nil {
		return
	}
	initSpell()
	tb.spell = newSpell()
	tb.spell.onSelect(func(e events.Event) {
		tb.correctText(tb.spell.correction)
	})
}

// correctText edits the text using the string chosen from the correction menu
func (tb *Buffer) correctText(s string) {
	st := lexer.Pos{tb.spell.srcLn, tb.spell.srcCh} // start of word
	tb.removeTag(st, token.TextSpellErr)
	oend := st
	oend.Ch += len(tb.spell.word)
	tb.ReplaceText(st, oend, st, s, EditSignal, ReplaceNoMatchCase)
	if tb.currentEditor != nil {
		ep := st
		ep.Ch += len(s)
		tb.currentEditor.SetCursorShow(ep)
		tb.currentEditor = nil
	}
}

// SpellCheckLineErrors runs spell check on given line, and returns Lex tags
// with token.TextSpellErr for any misspelled words
func (tb *Buffer) SpellCheckLineErrors(ln int) lexer.Line {
	if !tb.IsValidLine(ln) {
		return nil
	}
	tb.LinesMu.RLock()
	defer tb.LinesMu.RUnlock()
	tb.markupMu.RLock()
	defer tb.markupMu.RUnlock()
	return spell.CheckLexLine(tb.Lines[ln], tb.hiTags[ln])
}

// spellCheckLineTag runs spell check on given line, and sets Tags for any
// misspelled words and updates markup for that line.
func (tb *Buffer) spellCheckLineTag(ln int) {
	if !tb.IsValidLine(ln) {
		return
	}
	ser := tb.SpellCheckLineErrors(ln)
	tb.markupMu.Lock()
	ntgs := tb.adjustedTags(ln)
	ntgs.DeleteToken(token.TextSpellErr)
	for _, t := range ser {
		ntgs.AddSort(t)
	}
	tb.tags[ln] = ntgs
	tb.markupMu.Unlock()
	tb.markupLinesLock(ln, ln)
	tb.StartDelayedReMarkup()
}

///////////////////////////////////////////////////////////////////
//  Diff

// diffBuffers computes the diff between this buffer and the other buffer,
// reporting a sequence of operations that would convert this buffer (a) into
// the other buffer (b).  Each operation is either an 'r' (replace), 'd'
// (delete), 'i' (insert) or 'e' (equal).  Everything is line-based (0, offset).
func (tb *Buffer) diffBuffers(ob *Buffer) textbuf.Diffs {
	astr := tb.Strings(false)
	bstr := ob.Strings(false)
	return textbuf.DiffLines(astr, bstr)
}

// DiffBuffersUnified computes the diff between this buffer and the other buffer,
// returning a unified diff with given amount of context (default of 3 will be
// used if -1)
func (tb *Buffer) DiffBuffersUnified(ob *Buffer, context int) []byte {
	astr := tb.Strings(true) // needs newlines for some reason
	bstr := ob.Strings(true)

	return textbuf.DiffLinesUnified(astr, bstr, context, string(tb.Filename), tb.Info.ModTime.String(),
		string(ob.Filename), ob.Info.ModTime.String())
}

// patchFromBuffer patches (edits) this buffer using content from other buffer,
// according to diff operations (e.g., as generated from DiffBufs).  signal
// determines whether each patch is signaled -- if an overall signal will be
// sent at the end, then that would not be necessary (typical)
func (tb *Buffer) patchFromBuffer(ob *Buffer, diffs textbuf.Diffs, signal bool) bool {
	autoSave := tb.batchUpdateStart()
	defer tb.batchUpdateEnd(autoSave)

	sz := len(diffs)
	mods := false
	for i := sz - 1; i >= 0; i-- { // go in reverse so changes are valid!
		df := diffs[i]
		switch df.Tag {
		case 'r':
			tb.DeleteText(lexer.Pos{Ln: df.I1}, lexer.Pos{Ln: df.I2}, signal)
			// fmt.Printf("patch rep del: %v %v\n", tbe.Reg, string(tbe.ToBytes()))
			ot := ob.Region(lexer.Pos{Ln: df.J1}, lexer.Pos{Ln: df.J2})
			tb.insertText(lexer.Pos{Ln: df.I1}, ot.ToBytes(), signal)
			// fmt.Printf("patch rep ins: %v %v\n", tbe.Reg, string(tbe.ToBytes()))
			mods = true
		case 'd':
			tb.DeleteText(lexer.Pos{Ln: df.I1}, lexer.Pos{Ln: df.I2}, signal)
			// fmt.Printf("patch del: %v %v\n", tbe.Reg, string(tbe.ToBytes()))
			mods = true
		case 'i':
			ot := ob.Region(lexer.Pos{Ln: df.J1}, lexer.Pos{Ln: df.J2})
			tb.insertText(lexer.Pos{Ln: df.I1}, ot.ToBytes(), signal)
			// fmt.Printf("patch ins: %v %v\n", tbe.Reg, string(tbe.ToBytes()))
			mods = true
		}
	}
	return mods
}
