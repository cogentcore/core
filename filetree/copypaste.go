// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package filetree

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/fi"
	"cogentcore.org/core/giv"
	"cogentcore.org/core/glop/dirs"
	"cogentcore.org/core/grr"
	"cogentcore.org/core/mimedata"
	"cogentcore.org/core/texteditor"
)

// MimeData adds mimedata for this node: a text/plain of the Path,
// text/plain of filename, and text/
func (fn *Node) MimeData(md *mimedata.Mimes) {
	froot := fn.FRoot
	path := string(fn.FPath)
	punq := fn.PathFrom(froot) // note: ki paths have . escaped -> \,
	*md = append(*md, mimedata.NewTextData(punq))
	*md = append(*md, mimedata.NewTextData(path))
	if int(fn.Info.Size) < core.SystemSettings.BigFileSize {
		in, err := os.Open(path)
		if err != nil {
			slog.Error(err.Error())
			return
		}
		b, err := io.ReadAll(in)
		if err != nil {
			slog.Error(err.Error())
			return
		}
		fd := &mimedata.Data{fn.Info.Mime, b}
		*md = append(*md, fd)
	} else {
		*md = append(*md, mimedata.NewTextData("File exceeds BigFileSize"))
	}
}

// Cut copies to goosi.Clipboard and deletes selected items
func (fn *Node) Cut() {
	if fn.IsRoot("Cut") {
		return
	}
	fn.Copy(false)
	// todo: move files somewhere temporary, then use those temps for paste..
	core.MessageDialog(fn, "File names were copied to clipboard and can be pasted to copy elsewhere, but files are not deleted because contents of files are not placed on the clipboard and thus cannot be pasted as such.  Use Delete to delete files.")
}

func (fn *Node) Paste() {
	md := fn.Clipboard().Read([]string{fi.TextPlain})
	if md != nil {
		fn.PasteFiles(md, false, nil)
	}
}

func (fn *Node) DragDrop(e events.Event) {
	de := e.(*events.DragDrop)
	md := de.Data.(mimedata.Mimes)
	fn.PasteFiles(md, de.Source == nil, func() {
		fn.This().(giv.TreeViewer).DropFinalize(de)
	})
}

// PasteCheckExisting checks for existing files in target node directory if
// that is non-nil (otherwise just uses absolute path), and returns list of existing
// and node for last one if exists.
func (fn *Node) PasteCheckExisting(tfn *Node, md mimedata.Mimes, externalDrop bool) ([]string, *Node) {
	froot := fn.FRoot
	tpath := ""
	if tfn != nil {
		tpath = string(tfn.FPath)
	}
	nf := len(md)
	if !externalDrop {
		nf /= 3
	}
	var sfn *Node
	var existing []string
	for i := 0; i < nf; i++ {
		var d *mimedata.Data
		if !externalDrop {
			d = md[i*3+1]
			npath := string(md[i*3].Data)
			sfni := froot.FindPath(npath)
			if sfni != nil {
				sfn = AsNode(sfni)
			}
		} else {
			d = md[i] // just a list
		}
		if d.Type != fi.TextPlain {
			continue
		}
		path := string(d.Data)
		path = strings.TrimPrefix(path, "file://")
		if tfn != nil {
			_, fnm := filepath.Split(path)
			path = filepath.Join(tpath, fnm)
		}
		if grr.Log1(dirs.FileExists(path)) {
			existing = append(existing, path)
		}
	}
	return existing, sfn
}

// PasteCopyFiles copies files in given data into given target directory
func (fn *Node) PasteCopyFiles(tdir *Node, md mimedata.Mimes, externalDrop bool) {
	froot := fn.FRoot
	nf := len(md)
	if !externalDrop {
		nf /= 3
	}
	for i := 0; i < nf; i++ {
		var d *mimedata.Data
		mode := os.FileMode(0664)
		if !externalDrop {
			d = md[i*3+1]
			npath := string(md[i*3].Data)
			sfni := froot.FindPath(npath)
			if sfni == nil {
				slog.Error("filetree.Node: could not find path", "path", npath)
				continue
			}
			sfn := AsNode(sfni)
			mode = sfn.Info.Mode
		} else {
			d = md[i] // just a list
		}
		if d.Type != fi.TextPlain {
			continue
		}
		path := string(d.Data)
		path = strings.TrimPrefix(path, "file://")
		tdir.CopyFileToDir(path, mode)
	}
}

// PasteCopyFilesCheck copies files into given directory node,
// first checking if any already exist -- if they exist, prompts.
func (fn *Node) PasteCopyFilesCheck(tdir *Node, md mimedata.Mimes, externalDrop bool, dropFinal func()) {
	existing, _ := fn.PasteCheckExisting(tdir, md, externalDrop)
	if len(existing) == 0 {
		fn.PasteCopyFiles(tdir, md, externalDrop)
		if dropFinal != nil {
			dropFinal()
		}
		return
	}
	d := core.NewBody().AddTitle("File(s) Exist in Target Dir, Overwrite?").
		AddText(fmt.Sprintf("File(s): %v exist, do you want to overwrite?", existing))
	d.AddBottomBar(func(parent core.Widget) {
		d.AddCancel(parent)
		d.AddOK(parent).SetText("Overwrite").OnClick(func(e events.Event) {
			fn.PasteCopyFiles(tdir, md, externalDrop)
			if dropFinal != nil {
				dropFinal()
			}
		})
	})
	d.NewDialog(fn).Run()
}

// PasteFiles applies a paste / drop of mime data onto this node.
// always does a copy of files into / onto target.
// externalDrop is true if this is an externally-generated Drop event (from OS)
func (fn *Node) PasteFiles(md mimedata.Mimes, externalDrop bool, dropFinal func()) {
	if len(md) == 0 {
		return
	}
	if fn == nil || fn.IsExternal() {
		return
	}

	tpath := string(fn.FPath)
	isdir := fn.IsDir()
	if isdir {
		fn.PasteCopyFilesCheck(fn, md, externalDrop, dropFinal)
		return
	}
	if len(md) > 3 { // multiple files -- automatically goes into parent dir
		tdir := AsNode(fn.Parent())
		fn.PasteCopyFilesCheck(tdir, md, externalDrop, dropFinal)
		return
	}
	// single file dropped onto a single target file
	srcpath := ""
	if externalDrop {
		srcpath = string(md[0].Data) // just file path
	} else {
		srcpath = string(md[1].Data) // 1 has file path, 0 = ki path, 2 = file data
	}
	fname := filepath.Base(srcpath)
	tdir := AsNode(fn.Parent())
	existing, sfn := fn.PasteCheckExisting(tdir, md, externalDrop)
	mode := os.FileMode(0664)
	if sfn != nil {
		mode = sfn.Info.Mode
	}
	switch {
	case len(existing) == 1 && fname == fn.Nm:
		d := core.NewBody().AddTitle("Overwrite?").
			AddText(fmt.Sprintf("Overwrite target file: %s with source file of same name?, or diff (compare) two files?", fn.Nm))
		d.AddBottomBar(func(parent core.Widget) {
			d.AddCancel(parent)
			d.AddOK(parent).SetText("Diff (compare)").OnClick(func(e events.Event) {
				texteditor.DiffFiles(fn, tpath, srcpath)
			})
			d.AddOK(parent).SetText("Overwrite").OnClick(func(e events.Event) {
				fi.CopyFile(tpath, srcpath, mode)
				if dropFinal != nil {
					dropFinal()
				}
			})
		})
		d.NewDialog(fn).Run()
	case len(existing) > 0:
		d := core.NewBody().AddTitle("Overwrite?").
			AddText(fmt.Sprintf("Overwrite target file: %s with source file: %s, or overwrite existing file with same name as source file (%s), or diff (compare) files?", fn.Nm, fname, fname))
		d.AddBottomBar(func(parent core.Widget) {
			d.AddCancel(parent)
			d.AddOK(parent).SetText("Diff to target").OnClick(func(e events.Event) {
				texteditor.DiffFiles(fn, tpath, srcpath)
			})
			d.AddOK(parent).SetText("Diff to existing").OnClick(func(e events.Event) {
				npath := filepath.Join(string(tdir.FPath), fname)
				texteditor.DiffFiles(fn, npath, srcpath)
			})
			d.AddOK(parent).SetText("Overwrite target").OnClick(func(e events.Event) {
				fi.CopyFile(tpath, srcpath, mode)
				if dropFinal != nil {
					dropFinal()
				}
			})
			d.AddOK(parent).SetText("Overwrite existing").OnClick(func(e events.Event) {
				npath := filepath.Join(string(tdir.FPath), fname)
				fi.CopyFile(npath, srcpath, mode)
				if dropFinal != nil {
					dropFinal()
				}
			})
		})
		d.NewDialog(fn).Run()
	default:
		d := core.NewBody().AddTitle("Overwrite?").
			AddText(fmt.Sprintf("Overwrite target file: %s with source file: %s, or copy to: %s in current folder (which doesn't yet exist), or diff (compare) the two files?", fn.Nm, fname, fname))
		d.AddBottomBar(func(parent core.Widget) {
			d.AddCancel(parent)
			d.AddOK(parent).SetText("Diff (compare)").OnClick(func(e events.Event) {
				texteditor.DiffFiles(fn, tpath, srcpath)
			})
			d.AddOK(parent).SetText("Overwrite target").OnClick(func(e events.Event) {
				fi.CopyFile(tpath, srcpath, mode)
				if dropFinal != nil {
					dropFinal()
				}
			})
			d.AddOK(parent).SetText("Copy new file").OnClick(func(e events.Event) {
				tdir.CopyFileToDir(srcpath, mode)
				if dropFinal != nil {
					dropFinal()
				}
			})
		})
		d.NewDialog(fn).Run()
	}
}

// Dragged is called after target accepts the drop -- we just remove
// elements that were moved
// satisfies gi.DragNDropper interface and can be overridden by subtypes
func (fn *Node) DropDeleteSource(e events.Event) {
	de := e.(*events.DragDrop)
	froot := fn.FRoot
	if froot == nil || fn.IsExternal() {
		return
	}
	md := de.Data.(mimedata.Mimes)
	nf := len(md) / 3 // always internal
	for i := 0; i < nf; i++ {
		npath := string(md[i*3].Data)
		sfni := froot.FindPath(npath)
		if sfni == nil {
			slog.Error("filetree.Node: could not find path", "path", npath)
			continue
		}
		sfn := AsNode(sfni)
		if sfn == nil {
			continue
		}
		// fmt.Printf("dnd deleting: %v  path: %v\n", sfn.Path(), sfn.FPath)
		sfn.DeleteFile()
	}
}
