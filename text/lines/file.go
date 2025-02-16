// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package lines

import (
	"io/fs"
	"log"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/base/fileinfo"
)

//////// exported file api

// Filename returns the current filename
func (ls *Lines) Filename() string {
	ls.Lock()
	defer ls.Unlock()
	return ls.filename
}

// SetFilename sets the filename associated with the buffer and updates
// the code highlighting information accordingly.
func (ls *Lines) SetFilename(fn string) *Lines {
	ls.Lock()
	defer ls.Unlock()
	return ls.setFilename(fn)
}

// Stat gets info about the file, including the highlighting language.
func (ls *Lines) Stat() error {
	ls.Lock()
	defer ls.Unlock()
	return ls.stat()
}

// ConfigKnown configures options based on the supported language info in parse.
// Returns true if supported.
func (ls *Lines) ConfigKnown() bool {
	ls.Lock()
	defer ls.Unlock()
	return ls.configKnown()
}

// SetFileInfo sets the syntax highlighting and other parameters
// based on the type of file specified by given [fileinfo.FileInfo].
func (ls *Lines) SetFileInfo(info *fileinfo.FileInfo) *Lines {
	ls.Lock()
	defer ls.Unlock()
	ls.setFileInfo(info)
	return ls
}

// SetFileType sets the syntax highlighting and other parameters
// based on the given fileinfo.Known file type
func (ls *Lines) SetLanguage(ftyp fileinfo.Known) *Lines {
	return ls.SetFileInfo(fileinfo.NewFileInfoType(ftyp))
}

// SetFileExt sets syntax highlighting and other parameters
// based on the given file extension (without the . prefix),
// for cases where an actual file with [fileinfo.FileInfo] is not
// available.
func (ls *Lines) SetFileExt(ext string) *Lines {
	if len(ext) == 0 {
		return ls
	}
	if ext[0] == '.' {
		ext = ext[1:]
	}
	fn := "_fake." + strings.ToLower(ext)
	fi, _ := fileinfo.NewFileInfo(fn)
	return ls.SetFileInfo(fi)
}

// Open loads the given file into the buffer.
func (ls *Lines) Open(filename string) error { //types:add
	ls.Lock()
	err := ls.openFile(filename)
	ls.Unlock()
	ls.sendChange()
	return err
}

// OpenFS loads the given file in the given filesystem into the buffer.
func (ls *Lines) OpenFS(fsys fs.FS, filename string) error {
	ls.Lock()
	err := ls.openFileFS(fsys, filename)
	ls.Unlock()
	ls.sendChange()
	return err
}

// Revert re-opens text from the current file,
// if the filename is set; returns false if not.
// It uses an optimized diff-based update to preserve
// existing formatting, making it very fast if not very different.
func (ls *Lines) Revert() bool { //types:add
	ls.Lock()
	did := ls.revert()
	ls.Unlock()
	ls.sendChange()
	return did
}

// IsNotSaved returns true if buffer was changed (edited) since last Save.
func (ls *Lines) IsNotSaved() bool {
	ls.Lock()
	defer ls.Unlock()
	return ls.notSaved
}

// ClearNotSaved sets Changed and NotSaved to false.
func (ls *Lines) ClearNotSaved() {
	ls.Lock()
	defer ls.Unlock()
	ls.clearNotSaved()
}

// EditDone is called externally (e.g., by Editor widget) when the user
// has indicated that editing is done, and the results have been consumed.
func (ls *Lines) EditDone() {
	ls.Lock()
	ls.autosaveDelete()
	ls.changed = true
	ls.Unlock()
	ls.sendChange()
}

// SetReadOnly sets whether the buffer is read-only.
func (ls *Lines) SetReadOnly(readonly bool) *Lines {
	ls.Lock()
	defer ls.Unlock()
	return ls.setReadOnly(readonly)
}

// AutosaveFilename returns the autosave filename.
func (ls *Lines) AutosaveFilename() string {
	ls.Lock()
	defer ls.Unlock()
	return ls.autosaveFilename()
}

////////  Unexported implementation

// clearNotSaved sets Changed and NotSaved to false.
func (ls *Lines) clearNotSaved() {
	ls.changed = false
	ls.notSaved = false
}

// setReadOnly sets whether the buffer is read-only.
// read-only buffers also do not record undo events.
func (ls *Lines) setReadOnly(readonly bool) *Lines {
	ls.readOnly = readonly
	ls.undos.Off = readonly
	return ls
}

// setFilename sets the filename associated with the buffer and updates
// the code highlighting information accordingly.
func (ls *Lines) setFilename(fn string) *Lines {
	ls.filename = fn
	ls.stat()
	ls.setFileInfo(&ls.fileInfo)
	return ls
}

// stat gets info about the file, including the highlighting language.
func (ls *Lines) stat() error {
	ls.fileModOK = false
	err := ls.fileInfo.InitFile(string(ls.filename))
	ls.configKnown() // may have gotten file type info even if not existing
	return err
}

// configKnown configures options based on the supported language info in parse.
// Returns true if supported.
func (ls *Lines) configKnown() bool {
	if ls.fileInfo.Known != fileinfo.Unknown {
		// if ls.spell == nil {
		// 	ls.setSpell()
		// }
		// if ls.Complete == nil {
		// 	ls.setCompleter(&ls.ParseState, completeParse, completeEditParse, lookupParse)
		// }
		return ls.Settings.ConfigKnown(ls.fileInfo.Known)
	}
	return false
}

// openFile just loads the given file into the buffer, without doing
// any markup or signaling. It is typically used in other functions or
// for temporary buffers.
func (ls *Lines) openFile(filename string) error {
	txt, err := os.ReadFile(string(filename))
	if err != nil {
		return err
	}
	ls.setFilename(filename)
	ls.setText(txt)
	return nil
}

// openFileOnly just loads the given file into the buffer, without doing
// any markup or signaling. It is typically used in other functions or
// for temporary buffers.
func (ls *Lines) openFileOnly(filename string) error {
	txt, err := os.ReadFile(string(filename))
	if err != nil {
		return err
	}
	ls.setFilename(filename)
	ls.bytesToLines(txt) // not setText!
	return nil
}

// openFileFS loads the given file in the given filesystem into the buffer.
func (ls *Lines) openFileFS(fsys fs.FS, filename string) error {
	ls.Lock()
	defer ls.Unlock()
	txt, err := fs.ReadFile(fsys, filename)
	if err != nil {
		return err
	}
	ls.setFilename(filename)
	ls.setText(txt)
	return nil
}

// revert re-opens text from the current file,
// if the filename is set; returns false if not.
// It uses an optimized diff-based update to preserve
// existing formatting, making it very fast if not very different.
func (ls *Lines) revert() bool {
	if ls.filename == "" {
		return false
	}

	ls.stopDelayedReMarkup()
	ls.autosaveDelete() // justin case

	didDiff := false
	if ls.numLines() < diffRevertLines {
		ob := &Lines{}
		err := ob.openFileOnly(ls.filename)
		if errors.Log(err) != nil {
			// sc := tb.sceneFromEditor() // todo:
			// if sc != nil { // only if viewing
			// 	core.ErrorSnackbar(sc, err, "Error reopening file")
			// }
			return false
		}
		ls.stat() // "own" the new file..
		if ob.NumLines() < diffRevertLines {
			diffs := ls.DiffBuffers(ob)
			if len(diffs) < diffRevertDiffs {
				ls.PatchFromBuffer(ob, diffs)
				didDiff = true
			}
		}
	}
	if !didDiff {
		ls.openFile(ls.filename)
	}
	ls.clearNotSaved()
	ls.autosaveDelete()
	return true
}

// saveFile writes current buffer to file, with no prompting, etc
func (ls *Lines) saveFile(filename string) error {
	err := os.WriteFile(string(filename), ls.Bytes(), 0644)
	if err != nil {
		// core.ErrorSnackbar(tb.sceneFromEditor(), err) // todo:
		slog.Error(err.Error())
	} else {
		ls.clearNotSaved()
		ls.filename = filename
		ls.stat()
	}
	return err
}

// fileModCheck checks if the underlying file has been modified since last
// Stat (open, save); if haven't yet prompted, user is prompted to ensure
// that this is OK. It returns true if the file was modified.
func (ls *Lines) fileModCheck() bool {
	if ls.filename == "" || ls.fileModOK {
		return false
	}
	info, err := os.Stat(string(ls.filename))
	if err != nil {
		return false
	}
	if info.ModTime() != time.Time(ls.fileInfo.ModTime) {
		if !ls.notSaved { // we haven't edited: just revert
			ls.revert()
			return true
		}
		if ls.FileModPromptFunc != nil {
			ls.FileModPromptFunc() // todo: this could be called under lock -- need to figure out!
		}
		return true
	}
	return false
}

//////// Autosave

// autoSaveOff turns off autosave and returns the
// prior state of Autosave flag.
// Call AutosaveRestore with rval when done.
// See BatchUpdate methods for auto-use of this.
func (ls *Lines) autoSaveOff() bool {
	asv := ls.Autosave
	ls.Autosave = false
	return asv
}

// autoSaveRestore restores prior Autosave setting,
// from AutosaveOff
func (ls *Lines) autoSaveRestore(asv bool) {
	ls.Autosave = asv
}

// autosaveFilename returns the autosave filename.
func (ls *Lines) autosaveFilename() string {
	path, fn := filepath.Split(ls.filename)
	if fn == "" {
		fn = "new_file"
	}
	asfn := filepath.Join(path, "#"+fn+"#")
	return asfn
}

// autoSave does the autosave -- safe to call in a separate goroutine
func (ls *Lines) autoSave() error {
	if ls.autoSaving {
		return nil
	}
	ls.autoSaving = true
	asfn := ls.autosaveFilename()
	b := ls.bytes(0)
	err := os.WriteFile(asfn, b, 0644)
	if err != nil {
		log.Printf("Lines: Could not Autosave file: %v, error: %v\n", asfn, err)
	}
	ls.autoSaving = false
	return err
}

// autosaveDelete deletes any existing autosave file
func (ls *Lines) autosaveDelete() {
	asfn := ls.autosaveFilename()
	err := os.Remove(asfn)
	// the file may not exist, which is fine
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		errors.Log(err)
	}
}

// autosaveCheck checks if an autosave file exists; logic for dealing with
// it is left to larger app; call this before opening a file.
func (ls *Lines) autosaveCheck() bool {
	asfn := ls.autosaveFilename()
	if _, err := os.Stat(asfn); os.IsNotExist(err) {
		return false // does not exist
	}
	return true
}
