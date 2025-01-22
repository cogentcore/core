// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package texteditor

import (
	"fmt"
	"image"
	"io/fs"
	"log"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"time"

	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/base/fileinfo"
	"cogentcore.org/core/base/fsx"
	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/parse/complete"
	"cogentcore.org/core/parse/lexer"
	"cogentcore.org/core/parse/token"
	"cogentcore.org/core/spell"
	"cogentcore.org/core/texteditor/highlighting"
	"cogentcore.org/core/texteditor/text"
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
	text.Lines

	// Filename is the filename of the file that was last loaded or saved.
	// It is used when highlighting code.
	Filename core.Filename `json:"-" xml:"-"`

	// Autosave specifies whether the file should be automatically
	// saved after changes are made.
	Autosave bool

	// Info is the full information about the current file.
	Info fileinfo.FileInfo

	// LineColors are the colors to use for rendering circles
	// next to the line numbers of certain lines.
	LineColors map[int]image.Image

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

	// notSaved indicates if the text has been changed (edited) relative to the
	// original, since last Save.  This can be true even when changed flag is
	// false, because changed is cleared on EditDone, e.g., when texteditor
	// is being monitored for OnChange and user does Control+Enter.
	// Use IsNotSaved() method to query state.
	notSaved bool

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
	tb.SetText(nil) // to initialize
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
	// data is text.Edit describing change.
	// The Buf always reflects the current state *after* the edit.
	bufferInsert

	// bufferDelete signals that some text was deleted.
	// data is text.Edit describing change.
	// The Buf always reflects the current state *after* the edit.
	bufferDelete

	// bufferMarkupUpdated signals that the Markup text has been updated
	// This signal is typically sent from a separate goroutine,
	// so should be used with a mutex
	bufferMarkupUpdated

	// bufferClosed signals that the text was closed.
	bufferClosed
)

// signalEditors sends the given signal and optional edit info
// to all the [Editor]s for this [Buffer]
func (tb *Buffer) signalEditors(sig bufferSignals, edit *text.Edit) {
	for _, vw := range tb.editors {
		if vw != nil && vw.This != nil { // editor can be deleting
			vw.bufferSignal(sig, edit)
		}
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

// IsNotSaved returns true if buffer was changed (edited) since last Save.
func (tb *Buffer) IsNotSaved() bool {
	// note: could use a mutex on this if there are significant race issues
	return tb.notSaved
}

// clearNotSaved sets Changed and NotSaved to false.
func (tb *Buffer) clearNotSaved() {
	tb.SetChanged(false)
	tb.notSaved = false
}

// Init initializes the buffer.  Called automatically in SetText.
func (tb *Buffer) Init() {
	if tb.MarkupDoneFunc != nil {
		return
	}
	tb.MarkupDoneFunc = func() {
		tb.signalEditors(bufferMarkupUpdated, nil)
	}
	tb.ChangedFunc = func() {
		tb.notSaved = true
	}
}

// SetText sets the text to the given bytes.
// Pass nil to initialize an empty buffer.
func (tb *Buffer) SetText(text []byte) *Buffer {
	tb.Init()
	tb.Lines.SetText(text)
	tb.signalEditors(bufferNew, nil)
	return tb
}

// SetString sets the text to the given string.
func (tb *Buffer) SetString(txt string) *Buffer {
	return tb.SetText([]byte(txt))
}

func (tb *Buffer) Update() {
	tb.signalMods()
}

// editDone finalizes any current editing, sends signal
func (tb *Buffer) editDone() {
	tb.AutoSaveDelete()
	tb.SetChanged(false)
	tb.signalEditors(bufferDone, nil)
}

// Text returns the current text as a []byte array, applying all current
// changes by calling editDone, which will generate a signal if there have been
// changes.
func (tb *Buffer) Text() []byte {
	tb.editDone()
	return tb.Bytes()
}

// String returns the current text as a string, applying all current
// changes by calling editDone, which will generate a signal if there have been
// changes.
func (tb *Buffer) String() string {
	return string(tb.Text())
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
	tb.SetFileInfo(&tb.Info)
	return tb
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

// SetFileExt sets syntax highlighting and other parameters
// based on the given file extension (without the . prefix),
// for cases where an actual file with [fileinfo.FileInfo] is not
// available.
func (tb *Buffer) SetFileExt(ext string) *Buffer {
	tb.Lines.SetFileExt(ext)
	return tb
}

// SetFileType sets the syntax highlighting and other parameters
// based on the given fileinfo.Known file type
func (tb *Buffer) SetLanguage(ftyp fileinfo.Known) *Buffer {
	tb.Lines.SetLanguage(ftyp)
	return tb
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
		if !tb.IsNotSaved() { // we haven't edited: just revert
			tb.Revert()
			return true
		}
		sc := tb.sceneFromEditor()
		d := core.NewBody("File changed on disk: " + fsx.DirAndFile(string(tb.Filename)))
		core.NewText(d).SetType(core.TextSupporting).SetText(fmt.Sprintf("File has changed on disk since being opened or saved by you; what do you want to do?  If you <code>Revert from Disk</code>, you will lose any existing edits in open buffer.  If you <code>Ignore and Proceed</code>, the next save will overwrite the changed file on disk, losing any changes there.  File: %v", tb.Filename))
		d.AddBottomBar(func(bar *core.Frame) {
			core.NewButton(bar).SetText("Save as to different file").OnClick(func(e events.Event) {
				d.Close()
				core.CallFunc(sc, tb.SaveAs)
			})
			core.NewButton(bar).SetText("Revert from disk").OnClick(func(e events.Event) {
				d.Close()
				tb.Revert()
			})
			core.NewButton(bar).SetText("Ignore and proceed").OnClick(func(e events.Event) {
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
	return nil
}

// OpenFS loads the given file in the given filesystem into the buffer.
func (tb *Buffer) OpenFS(fsys fs.FS, filename string) error {
	txt, err := fs.ReadFile(fsys, filename)
	if err != nil {
		return err
	}
	tb.SetFilename(filename)
	tb.SetText(txt)
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
	tb.SetFilename(string(filename))
	tb.SetText(txt)
	return nil
}

// Revert re-opens text from the current file,
// if the filename is set; returns false if not.
// It uses an optimized diff-based update to preserve
// existing formatting, making it very fast if not very different.
func (tb *Buffer) Revert() bool { //types:add
	tb.StopDelayedReMarkup()
	tb.AutoSaveDelete() // justin case
	if tb.Filename == "" {
		return false
	}

	didDiff := false
	if tb.NumLines() < diffRevertLines {
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
		if ob.NumLines() < diffRevertLines {
			diffs := tb.DiffBuffers(&ob.Lines)
			if len(diffs) < diffRevertDiffs {
				tb.PatchFromBuffer(&ob.Lines, diffs)
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
	return true
}

// SaveAsFunc saves the current text into the given file.
// Does an editDone first to save edits and checks for an existing file.
// If it does exist then prompts to overwrite or not.
// If afterFunc is non-nil, then it is called with the status of the user action.
func (tb *Buffer) SaveAsFunc(filename core.Filename, afterFunc func(canceled bool)) {
	tb.editDone()
	if !errors.Log1(fsx.FileExists(string(filename))) {
		tb.saveFile(filename)
		if afterFunc != nil {
			afterFunc(false)
		}
	} else {
		sc := tb.sceneFromEditor()
		d := core.NewBody("File exists")
		core.NewText(d).SetType(core.TextSupporting).SetText(fmt.Sprintf("The file already exists; do you want to overwrite it?  File: %v", filename))
		d.AddBottomBar(func(bar *core.Frame) {
			d.AddCancel(bar).OnClick(func(e events.Event) {
				if afterFunc != nil {
					afterFunc(true)
				}
			})
			d.AddOK(bar).OnClick(func(e events.Event) {
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
	err := os.WriteFile(string(filename), tb.Bytes(), 0644)
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
		return errors.New("core.Buf: filename is empty for Save")
	}
	tb.editDone()
	info, err := os.Stat(string(tb.Filename))
	if err == nil && info.ModTime() != time.Time(tb.Info.ModTime) {
		sc := tb.sceneFromEditor()
		d := core.NewBody("File Changed on Disk")
		core.NewText(d).SetType(core.TextSupporting).SetText(fmt.Sprintf("File has changed on disk since you opened or saved it; what do you want to do?  File: %v", tb.Filename))
		d.AddBottomBar(func(bar *core.Frame) {
			core.NewButton(bar).SetText("Save to different file").OnClick(func(e events.Event) {
				d.Close()
				core.CallFunc(sc, tb.SaveAs)
			})
			core.NewButton(bar).SetText("Open from disk, losing changes").OnClick(func(e events.Event) {
				d.Close()
				tb.Revert()
			})
			core.NewButton(bar).SetText("Save file, overwriting").OnClick(func(e events.Event) {
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
	if tb.IsNotSaved() {
		tb.StopDelayedReMarkup()
		sc := tb.sceneFromEditor()
		if tb.Filename != "" {
			d := core.NewBody("Close without saving?")
			core.NewText(d).SetType(core.TextSupporting).SetText(fmt.Sprintf("Do you want to save your changes to file: %v?", tb.Filename))
			d.AddBottomBar(func(bar *core.Frame) {
				core.NewButton(bar).SetText("Cancel").OnClick(func(e events.Event) {
					d.Close()
					if afterFun != nil {
						afterFun(true)
					}
				})
				core.NewButton(bar).SetText("Close without saving").OnClick(func(e events.Event) {
					d.Close()
					tb.clearNotSaved()
					tb.AutoSaveDelete()
					tb.Close(afterFun)
				})
				core.NewButton(bar).SetText("Save").OnClick(func(e events.Event) {
					tb.Save()
					tb.Close(afterFun) // 2nd time through won't prompt
				})
			})
			d.RunDialog(sc)
		} else {
			d := core.NewBody("Close without saving?")
			core.NewText(d).SetType(core.TextSupporting).SetText("Do you want to save your changes (no filename for this buffer yet)?  If so, Cancel and then do Save As")
			d.AddBottomBar(func(bar *core.Frame) {
				d.AddCancel(bar).OnClick(func(e events.Event) {
					if afterFun != nil {
						afterFun(true)
					}
				})
				d.AddOK(bar).SetText("Close without saving").OnClick(func(e events.Event) {
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
	tb.SetText(nil)
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
	b := tb.Bytes()
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

// AppendTextMarkup appends new text to end of buffer, using insert, returns
// edit, and uses supplied markup to render it.
func (tb *Buffer) AppendTextMarkup(text []byte, markup []byte, signal bool) *text.Edit {
	tbe := tb.Lines.AppendTextMarkup(text, markup)
	if tbe != nil && signal {
		tb.signalEditors(bufferInsert, tbe)
	}
	return tbe
}

// AppendTextLineMarkup appends one line of new text to end of buffer, using
// insert, and appending a LF at the end of the line if it doesn't already
// have one. User-supplied markup is used. Returns the edit region.
func (tb *Buffer) AppendTextLineMarkup(text []byte, markup []byte, signal bool) *text.Edit {
	tbe := tb.Lines.AppendTextLineMarkup(text, markup)
	if tbe != nil && signal {
		tb.signalEditors(bufferInsert, tbe)
	}
	return tbe
}

/////////////////////////////////////////////////////////////////////////////
//   Editors

// addEditor adds a editor of this buffer, connecting our signals to the editor
func (tb *Buffer) addEditor(vw *Editor) {
	tb.editors = append(tb.editors, vw)
}

// deleteEditor removes given editor from our buffer
func (tb *Buffer) deleteEditor(vw *Editor) {
	tb.editors = slices.DeleteFunc(tb.editors, func(e *Editor) bool {
		return e == vw
	})
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
			ed.SetCursorTarget(tb.EndPos())
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
func (tb *Buffer) DeleteText(st, ed lexer.Pos, signal bool) *text.Edit {
	tb.FileModCheck()
	tbe := tb.Lines.DeleteText(st, ed)
	if tbe == nil {
		return tbe
	}
	if signal {
		tb.signalEditors(bufferDelete, tbe)
	}
	if tb.Autosave {
		go tb.autoSave()
	}
	return tbe
}

// deleteTextRect deletes rectangular region of text between start, end
// defining the upper-left and lower-right corners of a rectangle.
// Fails if st.Ch >= ed.Ch. Sets the timestamp on resulting text.Edit to now.
// An Undo record is automatically saved depending on Undo.Off setting.
func (tb *Buffer) deleteTextRect(st, ed lexer.Pos, signal bool) *text.Edit {
	tb.FileModCheck()
	tbe := tb.Lines.DeleteTextRect(st, ed)
	if tbe == nil {
		return tbe
	}
	if signal {
		tb.signalMods()
	}
	if tb.Autosave {
		go tb.autoSave()
	}
	return tbe
}

// insertText is the primary method for inserting text into the buffer.
// It inserts new text at given starting position, optionally signaling
// views after text has been inserted.  Sets the timestamp on resulting Edit to now.
// An Undo record is automatically saved depending on Undo.Off setting.
func (tb *Buffer) insertText(st lexer.Pos, text []byte, signal bool) *text.Edit {
	tb.FileModCheck() // will just revert changes if shouldn't have changed
	tbe := tb.Lines.InsertText(st, text)
	if tbe == nil {
		return tbe
	}
	if signal {
		tb.signalEditors(bufferInsert, tbe)
	}
	if tb.Autosave {
		go tb.autoSave()
	}
	return tbe
}

// insertTextRect inserts a rectangle of text defined in given text.Edit record,
// (e.g., from RegionRect or DeleteRect), optionally signaling
// views after text has been inserted.
// Returns a copy of the Edit record with an updated timestamp.
// An Undo record is automatically saved depending on Undo.Off setting.
func (tb *Buffer) insertTextRect(tbe *text.Edit, signal bool) *text.Edit {
	tb.FileModCheck() // will just revert changes if shouldn't have changed
	nln := tb.NumLines()
	re := tb.Lines.InsertTextRect(tbe)
	if re == nil {
		return re
	}
	if signal {
		if re.Reg.End.Ln >= nln {
			ie := &text.Edit{}
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

// ReplaceText does DeleteText for given region, and then InsertText at given position
// (typically same as delSt but not necessarily), optionally emitting a signal after the insert.
// if matchCase is true, then the lexer.MatchCase function is called to match the
// case (upper / lower) of the new inserted text to that of the text being replaced.
// returns the text.Edit for the inserted text.
func (tb *Buffer) ReplaceText(delSt, delEd, insPos lexer.Pos, insTxt string, signal, matchCase bool) *text.Edit {
	tbe := tb.Lines.ReplaceText(delSt, delEd, insPos, insTxt, matchCase)
	if tbe == nil {
		return tbe
	}
	if signal {
		tb.signalMods() // todo: could be more specific?
	}
	if tb.Autosave {
		go tb.autoSave()
	}
	return tbe
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
//   Undo

// undo undoes next group of items on the undo stack
func (tb *Buffer) undo() []*text.Edit {
	autoSave := tb.batchUpdateStart()
	defer tb.batchUpdateEnd(autoSave)
	tbe := tb.Lines.Undo()
	if tbe == nil || tb.Undos.Pos == 0 { // no more undo = fully undone
		tb.SetChanged(false)
		tb.notSaved = false
		tb.AutoSaveDelete()
	}
	tb.signalMods()
	return tbe
}

// redo redoes next group of items on the undo stack,
// and returns the last record, nil if no more
func (tb *Buffer) redo() []*text.Edit {
	autoSave := tb.batchUpdateStart()
	defer tb.batchUpdateEnd(autoSave)
	tbe := tb.Lines.Redo()
	if tbe != nil {
		tb.signalMods()
	}
	return tbe
}

/////////////////////////////////////////////////////////////////////////////
//   LineColors

// SetLineColor sets the color to use for rendering a circle next to the line
// number at the given line.
func (tb *Buffer) SetLineColor(ln int, color image.Image) {
	if tb.LineColors == nil {
		tb.LineColors = make(map[int]image.Image)
	}
	tb.LineColors[ln] = color
}

// HasLineColor checks if given line has a line color set
func (tb *Buffer) HasLineColor(ln int) bool {
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
func (tb *Buffer) indentLine(ln, ind int) *text.Edit {
	autoSave := tb.batchUpdateStart()
	defer tb.batchUpdateEnd(autoSave)
	tbe := tb.Lines.IndentLine(ln, ind)
	tb.signalMods()
	return tbe
}

// AutoIndentRegion does auto-indent over given region; end is *exclusive*
func (tb *Buffer) AutoIndentRegion(start, end int) {
	autoSave := tb.batchUpdateStart()
	defer tb.batchUpdateEnd(autoSave)
	tb.Lines.AutoIndentRegion(start, end)
	tb.signalMods()
}

// CommentRegion inserts comment marker on given lines; end is *exclusive*
func (tb *Buffer) CommentRegion(start, end int) {
	autoSave := tb.batchUpdateStart()
	defer tb.batchUpdateEnd(autoSave)
	tb.Lines.CommentRegion(start, end)
	tb.signalMods()
}

// JoinParaLines merges sequences of lines with hard returns forming paragraphs,
// separated by blank lines, into a single line per paragraph,
// within the given line regions; endLine is *inclusive*
func (tb *Buffer) JoinParaLines(startLine, endLine int) {
	autoSave := tb.batchUpdateStart()
	defer tb.batchUpdateEnd(autoSave)
	tb.Lines.JoinParaLines(startLine, endLine)
	tb.signalMods()
}

// TabsToSpaces replaces tabs with spaces over given region; end is *exclusive*
func (tb *Buffer) TabsToSpaces(start, end int) {
	autoSave := tb.batchUpdateStart()
	defer tb.batchUpdateEnd(autoSave)
	tb.Lines.TabsToSpaces(start, end)
	tb.signalMods()
}

// SpacesToTabs replaces tabs with spaces over given region; end is *exclusive*
func (tb *Buffer) SpacesToTabs(start, end int) {
	autoSave := tb.batchUpdateStart()
	defer tb.batchUpdateEnd(autoSave)
	tb.Lines.SpacesToTabs(start, end)
	tb.signalMods()
}

// DiffBuffersUnified computes the diff between this buffer and the other buffer,
// returning a unified diff with given amount of context (default of 3 will be
// used if -1)
func (tb *Buffer) DiffBuffersUnified(ob *Buffer, context int) []byte {
	astr := tb.Strings(true) // needs newlines for some reason
	bstr := ob.Strings(true)

	return text.DiffLinesUnified(astr, bstr, context, string(tb.Filename), tb.Info.ModTime.String(),
		string(ob.Filename), ob.Info.ModTime.String())
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
	en := lexer.Pos{tb.Complete.SrcLn, tb.LineLen(tb.Complete.SrcLn)}
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
		return !tb.InTokenCode(pos)
	case fileinfo.Code:
		return tb.InComment(pos) || tb.InLitString(pos)
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
	tb.RemoveTag(st, token.TextSpellErr)
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
	return spell.CheckLexLine(tb.Line(ln), tb.HiTags(ln))
}

// spellCheckLineTag runs spell check on given line, and sets Tags for any
// misspelled words and updates markup for that line.
func (tb *Buffer) spellCheckLineTag(ln int) {
	if !tb.IsValidLine(ln) {
		return
	}
	ser := tb.SpellCheckLineErrors(ln)
	ntgs := tb.AdjustedTags(ln)
	ntgs.DeleteToken(token.TextSpellErr)
	for _, t := range ser {
		ntgs.AddSort(t)
	}
	tb.SetTags(ln, ntgs)
	tb.MarkupLines(ln, ln)
	tb.StartDelayedReMarkup()
}
