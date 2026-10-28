// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/base/fileinfo/mimedata"
	"cogentcore.org/core/base/fsx"
	"cogentcore.org/core/events"
	"cogentcore.org/core/styles/states"
)

// handleFilesDND sets up drag-and-drop for the files table:
// files can be dragged out of it, to other widgets, windows and apps,
// and dropped files are copied (or moved) into the current directory.
func (fp *FilePicker) handleFilesDND(w *Table) {
	w.On(events.DragStart, func(e events.Event) {
		fp.filesDragStart(e)
	})
	w.On(events.DragEnter, func(e events.Event) {
		w.SetState(true, states.DragHovered)
		w.NeedsRender()
		e.SetHandled()
	})
	w.On(events.DragLeave, func(e events.Event) {
		w.SetState(false, states.DragHovered)
		w.NeedsRender()
		e.SetHandled()
	})
	w.On(events.Drop, func(e events.Event) {
		fp.filesDrop(e)
	})
}

// filesDragStart starts dragging the file under the mouse.
func (fp *FilePicker) filesDragStart(e events.Event) {
	w := fp.filesTable
	row, _, ok := w.ListGrid.indexFromPixel(e.Pos())
	if !ok {
		return
	}
	idx := row + w.StartIndex
	if idx < 0 || idx >= len(fp.files) {
		return
	}
	if !w.indexIsSelected(idx) {
		w.updateSelectRow(row, events.SelectOne)
	}
	path := filepath.Join(fp.directory, fp.files[idx].Name)
	rw, ok := w.rowFirstWidget(row)
	if !ok {
		return
	}
	fp.Scene.Events.DragStart(rw.This.(Widget), mimedata.NewText(path), e)
	fp.Scene.Events.SetDragFiles(path)
	e.SetHandled()
}

// filesDrop copies the files dropped on the files table into the
// current directory, or moves them for [events.DropMove], first
// asking whether to overwrite any existing files of the same names.
func (fp *FilePicker) filesDrop(e events.Event) {
	de := e.(*events.DragDrop)
	e.SetHandled()
	fp.filesTable.SetState(false, states.DragHovered)
	paths := DroppedFiles(de)
	var existing []string
	for _, p := range paths {
		tp := filepath.Join(fp.directory, filepath.Base(p))
		if tp != p && errors.Ignore1(fsx.FileExists(tp)) {
			existing = append(existing, filepath.Base(p))
		}
	}
	move := de.DropMod == events.DropMove
	drop := func() {
		var errs []error
		for _, p := range paths {
			tp := filepath.Join(fp.directory, filepath.Base(p))
			if tp == p {
				continue
			}
			var err error
			if move {
				err = movePath(tp, p)
			} else {
				err = fsx.CopyPath(tp, p)
			}
			errs = append(errs, err)
		}
		fp.updateFilesEvent()
		ErrorSnackbar(fp, errors.Join(errs...), "Error copying file")
	}
	if len(existing) == 0 {
		drop()
		return
	}
	d := NewBody("Overwrite?")
	NewText(d).SetType(TextSupporting).SetText(fmt.Sprintf("Overwrite existing file(s) in %s: %s?", fp.directory, strings.Join(existing, ", ")))
	d.AddBottomBar(func(bar *Frame) {
		d.AddCancel(bar)
		d.AddOK(bar).SetText("Overwrite").OnClick(func(e events.Event) {
			drop()
		})
	})
	d.RunDialog(fp)
}

// movePath moves the file or directory at src to dst, overwriting an
// existing file. If it can't be renamed, for example on another drive,
// it is copied, and then deleted if the copy succeeds.
func movePath(dst, src string) error {
	if os.Rename(src, dst) == nil {
		return nil
	}
	if err := fsx.CopyPath(dst, src); err != nil {
		return err
	}
	return os.RemoveAll(src)
}
