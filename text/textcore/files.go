// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package textcore

import (
	"fmt"
	"os"
	"time"

	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/base/fsx"
	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/text/lines"
)

// SaveAs saves the given Lines text into the given file.
// Does an EditDone on the Lines first to save edits and checks for an existing file.
// If it does exist then prompts to overwrite or not.
// If afterFunc is non-nil, then it is called with the status of the user action.
func SaveAs(sc *core.Scene, lns *lines.Lines, filename string, afterFunc func(canceled bool)) {
	lns.EditDone()
	if !errors.Log1(fsx.FileExists(filename)) {
		lns.SaveFile(filename)
		if afterFunc != nil {
			afterFunc(false)
		}
	} else {
		d := core.NewBody("File exists")
		core.NewText(d).SetType(core.TextSupporting).SetText(fmt.Sprintf("The file already exists; do you want to overwrite it?  File: %v", filename))
		d.AddBottomBar(func(bar *core.Frame) {
			d.AddCancel(bar).OnClick(func(e events.Event) {
				if afterFunc != nil {
					afterFunc(true)
				}
			})
			d.AddOK(bar).OnClick(func(e events.Event) {
				lns.SaveFile(filename)
				if afterFunc != nil {
					afterFunc(false)
				}
			})
		})
		d.RunDialog(sc)
	}
}

// Save saves the given LInes into the current filename associated with this buffer,
// prompting if the file is changed on disk since the last save. Does an EditDone
// on the lines.
func Save(sc *core.Scene, lns *lines.Lines) error {
	fname := lns.Filename()
	if fname == "" {
		return errors.New("core.Editor: filename is empty for Save")
	}
	lns.EditDone()
	info, err := os.Stat(fname)
	if err == nil && info.ModTime() != time.Time(lns.FileInfo().ModTime) {
		d := core.NewBody("File Changed on Disk")
		core.NewText(d).SetType(core.TextSupporting).SetText(fmt.Sprintf("File has changed on disk since you opened or saved it; what do you want to do?  File: %v", fname))
		d.AddBottomBar(func(bar *core.Frame) {
			core.NewButton(bar).SetText("Save to different file").OnClick(func(e events.Event) {
				d.Close()
				fd := core.NewBody("Save file as")
				fv := core.NewFilePicker(fd).SetFilename(fname)
				fv.OnSelect(func(e events.Event) {
					SaveAs(sc, lns, fv.SelectedFile(), nil)
				})
				fd.RunWindowDialog(sc)
			})
			core.NewButton(bar).SetText("Open from disk, losing changes").OnClick(func(e events.Event) {
				d.Close()
				lns.Revert()
			})
			core.NewButton(bar).SetText("Save file, overwriting").OnClick(func(e events.Event) {
				d.Close()
				lns.SaveFile(fname)
			})
		})
		d.RunDialog(sc)
	}
	return lns.SaveFile(fname)
}

// Close closes the lines viewed by this editor, prompting to save if there are changes.
// If afterFunc is non-nil, then it is called with the status of the user action.
// Returns false if the file was actually not closed pending input from the user.
func Close(sc *core.Scene, lns *lines.Lines, afterFunc func(canceled bool)) bool {
	if !lns.IsNotSaved() {
		lns.Close()
		if afterFunc != nil {
			afterFunc(false)
		}
		return true
	}
	lns.StopDelayedReMarkup()
	fname := lns.Filename()
	if fname == "" {
		d := core.NewBody("Close without saving?")
		core.NewText(d).SetType(core.TextSupporting).SetText("Do you want to save your changes (no filename for this buffer yet)?  If so, Cancel and then do Save As")
		d.AddBottomBar(func(bar *core.Frame) {
			d.AddCancel(bar).OnClick(func(e events.Event) {
				if afterFunc != nil {
					afterFunc(true)
				}
			})
			d.AddOK(bar).SetText("Close without saving").OnClick(func(e events.Event) {
				lns.ClearNotSaved()
				lns.AutosaveDelete()
				Close(sc, lns, afterFunc)
			})
		})
		d.RunDialog(sc)
		return false // awaiting decisions..
	}

	d := core.NewBody("Close without saving?")
	core.NewText(d).SetType(core.TextSupporting).SetText(fmt.Sprintf("Do you want to save your changes to file: %v?", fname))
	d.AddBottomBar(func(bar *core.Frame) {
		core.NewButton(bar).SetText("Cancel").OnClick(func(e events.Event) {
			d.Close()
			if afterFunc != nil {
				afterFunc(true)
			}
		})
		core.NewButton(bar).SetText("Close without saving").OnClick(func(e events.Event) {
			d.Close()
			lns.ClearNotSaved()
			lns.AutosaveDelete()
			Close(sc, lns, afterFunc)
		})
		core.NewButton(bar).SetText("Save").OnClick(func(e events.Event) {
			Save(sc, lns)
			Close(sc, lns, afterFunc) // 2nd time through won't prompt
		})
	})
	d.RunDialog(sc)
	return false
}

// FileModPrompt is called when a file has been modified in the filesystem
// and it is about to be modified through an edit, in the fileModCheck function.
// The prompt determines whether the user wants to revert, overwrite, or
// save current version as a different file.
func FileModPrompt(sc *core.Scene, lns *lines.Lines) bool {
	fname := lns.Filename()
	d := core.NewBody("File changed on disk: " + fsx.DirAndFile(fname))
	core.NewText(d).SetType(core.TextSupporting).SetText(fmt.Sprintf("File has changed on disk since being opened or saved by you; what do you want to do?  If you <code>Revert from Disk</code>, you will lose any existing edits in open buffer.  If you <code>Ignore and Proceed</code>, the next save will overwrite the changed file on disk, losing any changes there.  File: %v", fname))
	d.AddBottomBar(func(bar *core.Frame) {
		core.NewButton(bar).SetText("Save as to different file").OnClick(func(e events.Event) {
			d.Close()
			fd := core.NewBody("Save file as")
			fv := core.NewFilePicker(fd).SetFilename(fname)
			fv.OnSelect(func(e events.Event) {
				SaveAs(sc, lns, fv.SelectedFile(), nil)
			})
			fd.RunWindowDialog(sc)
		})
		core.NewButton(bar).SetText("Revert from disk").OnClick(func(e events.Event) {
			d.Close()
			lns.Revert()
		})
		core.NewButton(bar).SetText("Ignore and proceed").OnClick(func(e events.Event) {
			d.Close()
			lns.SetFileModOK(true)
		})
	})
	d.RunDialog(sc)
	return true
}
