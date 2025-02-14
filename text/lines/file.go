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

	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/base/fileinfo"
	"cogentcore.org/core/base/fsx"
)

// todo: cleanup locks / exported status:

//////// exported file api

// IsNotSaved returns true if buffer was changed (edited) since last Save.
func (ls *Lines) IsNotSaved() bool {
	ls.Lock()
	defer ls.Unlock()
	return ls.notSaved
}

// clearNotSaved sets Changed and NotSaved to false.
func (ls *Lines) clearNotSaved() {
	ls.SetChanged(false)
	ls.notSaved = false
}

// SetReadOnly sets whether the buffer is read-only.
func (ls *Lines) SetReadOnly(readonly bool) *Lines {
	ls.ReadOnly = readonly
	ls.undos.Off = readonly
	return ls
}

// SetFilename sets the filename associated with the buffer and updates
// the code highlighting information accordingly.
func (ls *Lines) SetFilename(fn string) *Lines {
	ls.Filename = fsx.Filename(fn)
	ls.Stat()
	ls.SetFileInfo(&ls.FileInfo)
	return ls
}

// Stat gets info about the file, including the highlighting language.
func (ls *Lines) Stat() error {
	ls.fileModOK = false
	err := ls.FileInfo.InitFile(string(ls.Filename))
	ls.ConfigKnown() // may have gotten file type info even if not existing
	return err
}

// ConfigKnown configures options based on the supported language info in parse.
// Returns true if supported.
func (ls *Lines) ConfigKnown() bool {
	if ls.FileInfo.Known != fileinfo.Unknown {
		// if ls.spell == nil {
		// 	ls.setSpell()
		// }
		// if ls.Complete == nil {
		// 	ls.setCompleter(&ls.ParseState, completeParse, completeEditParse, lookupParse)
		// }
		return ls.Settings.ConfigKnown(ls.FileInfo.Known)
	}
	return false
}

// Open loads the given file into the buffer.
func (ls *Lines) Open(filename fsx.Filename) error { //types:add
	err := ls.openFile(filename)
	if err != nil {
		return err
	}
	return nil
}

// OpenFS loads the given file in the given filesystem into the buffer.
func (ls *Lines) OpenFS(fsys fs.FS, filename string) error {
	txt, err := fs.ReadFile(fsys, filename)
	if err != nil {
		return err
	}
	ls.SetFilename(filename)
	ls.SetText(txt)
	return nil
}

// openFile just loads the given file into the buffer, without doing
// any markup or signaling. It is typically used in other functions or
// for temporary buffers.
func (ls *Lines) openFile(filename fsx.Filename) error {
	txt, err := os.ReadFile(string(filename))
	if err != nil {
		return err
	}
	ls.SetFilename(string(filename))
	ls.SetText(txt)
	return nil
}

// Revert re-opens text from the current file,
// if the filename is set; returns false if not.
// It uses an optimized diff-based update to preserve
// existing formatting, making it very fast if not very different.
func (ls *Lines) Revert() bool { //types:add
	ls.StopDelayedReMarkup()
	ls.AutoSaveDelete() // justin case
	if ls.Filename == "" {
		return false
	}
	ls.Lock()
	defer ls.Unlock()

	didDiff := false
	if ls.numLines() < diffRevertLines {
		ob := &Lines{}
		err := ob.openFile(ls.Filename)
		if errors.Log(err) != nil {
			// sc := tb.sceneFromEditor()
			// if sc != nil { // only if viewing
			// 	core.ErrorSnackbar(sc, err, "Error reopening file")
			// }
			return false
		}
		ls.Stat() // "own" the new file..
		if ob.NumLines() < diffRevertLines {
			diffs := ls.DiffBuffers(ob)
			if len(diffs) < diffRevertDiffs {
				ls.PatchFromBuffer(ob, diffs)
				didDiff = true
			}
		}
	}
	if !didDiff {
		ls.openFile(ls.Filename)
	}
	ls.clearNotSaved()
	ls.AutoSaveDelete()
	// ls.signalEditors(bufferNew, nil)
	return true
}

// saveFile writes current buffer to file, with no prompting, etc
func (ls *Lines) saveFile(filename fsx.Filename) error {
	err := os.WriteFile(string(filename), ls.Bytes(), 0644)
	if err != nil {
		// core.ErrorSnackbar(tb.sceneFromEditor(), err)
		slog.Error(err.Error())
	} else {
		ls.clearNotSaved()
		ls.Filename = filename
		ls.Stat()
	}
	return err
}

// autoSaveOff turns off autosave and returns the
// prior state of Autosave flag.
// Call AutoSaveRestore with rval when done.
// See BatchUpdate methods for auto-use of this.
func (ls *Lines) autoSaveOff() bool {
	asv := ls.Autosave
	ls.Autosave = false
	return asv
}

// autoSaveRestore restores prior Autosave setting,
// from AutoSaveOff
func (ls *Lines) autoSaveRestore(asv bool) {
	ls.Autosave = asv
}

// AutoSaveFilename returns the autosave filename.
func (ls *Lines) AutoSaveFilename() string {
	path, fn := filepath.Split(string(ls.Filename))
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
	asfn := ls.AutoSaveFilename()
	b := ls.bytes(0)
	err := os.WriteFile(asfn, b, 0644)
	if err != nil {
		log.Printf("Lines: Could not AutoSave file: %v, error: %v\n", asfn, err)
	}
	ls.autoSaving = false
	return err
}

// AutoSaveDelete deletes any existing autosave file
func (ls *Lines) AutoSaveDelete() {
	asfn := ls.AutoSaveFilename()
	err := os.Remove(asfn)
	// the file may not exist, which is fine
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		errors.Log(err)
	}
}

// AutoSaveCheck checks if an autosave file exists; logic for dealing with
// it is left to larger app; call this before opening a file.
func (ls *Lines) AutoSaveCheck() bool {
	asfn := ls.AutoSaveFilename()
	if _, err := os.Stat(asfn); os.IsNotExist(err) {
		return false // does not exist
	}
	return true
}

// batchUpdateStart call this when starting a batch of updates.
// It calls AutoSaveOff and returns the prior state of that flag
// which must be restored using BatchUpdateEnd.
func (ls *Lines) batchUpdateStart() (autoSave bool) {
	ls.undos.NewGroup()
	autoSave = ls.autoSaveOff()
	return
}

// batchUpdateEnd call to complete BatchUpdateStart
func (ls *Lines) batchUpdateEnd(autoSave bool) {
	ls.autoSaveRestore(autoSave)
}
