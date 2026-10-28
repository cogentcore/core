// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package filetree

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/base/fileinfo"
	"cogentcore.org/core/base/fileinfo/mimedata"
	"cogentcore.org/core/base/fsx"
	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/text/textcore"
	"cogentcore.org/core/tree"
)

// MimeData adds mimedata for this node: three items, a text/plain of the
// tree path, text/plain of the file path, and the file contents (or a
// text/plain message for a directory or a file that is too big or can't be read).
func (fn *Node) MimeData(md *mimedata.Mimes) {
	froot := fn.FileRoot()
	path := string(fn.Filepath)
	punq := fn.PathFrom(froot) // note: tree paths have . escaped -> \,
	*md = append(*md, mimedata.NewTextData(punq))
	*md = append(*md, mimedata.NewTextData(path))
	switch {
	case fn.IsDir():
		*md = append(*md, mimedata.NewTextData("Folder"))
	case int(fn.Info.Size) >= core.SystemSettings.BigFileSize:
		*md = append(*md, mimedata.NewTextData("File exceeds BigFileSize"))
	default:
		b, err := os.ReadFile(path) // closes the file, which Windows can't delete while open
		if err != nil {
			slog.Error(err.Error())
			*md = append(*md, mimedata.NewTextData(err.Error()))
			return
		}
		*md = append(*md, &mimedata.Data{Type: fn.Info.Mime, Data: b})
	}
}

// Cut copies the selected files to the clipboard and then deletes them.
func (fn *Node) Cut() { //types:add
	if fn.IsRoot("Cut") {
		return
	}
	fn.Copy()
	// todo: move files somewhere temporary, then use those temps for paste..
	core.MessageDialog(fn, "File names were copied to clipboard and can be pasted to copy elsewhere, but files are not deleted because contents of files are not placed on the clipboard and thus cannot be pasted as such.  Use Delete to delete files.")
}

// Paste copies the files on the clipboard, from [Node.Copy] or as
// absolute paths in text, into this directory, or the directory of this file.
func (fn *Node) Paste() { //types:add
	md := fn.Clipboard().Read([]string{fileinfo.TextPlain})
	fn.pasteFiles(clipboardPaths(md), false, nil)
}

func (fn *Node) DragDrop(e events.Event) {
	de := e.(*events.DragDrop)
	e.SetHandled()
	md, _ := de.Data.(mimedata.Mimes)
	var paths []string
	if sn, ok := de.Source.(tree.Node); ok && AsNode(sn) != nil {
		// from a file tree, possibly in another window
		paths = filePaths(md)
	} else { // from another widget or app
		paths = core.DroppedFiles(de)
	}
	fn.pasteFiles(paths, de.DropMod == events.DropMove, func(inPlace []string) {
		if len(inPlace) > 0 { // already where they were moved to: not to be deleted
			de.Data = withoutFiles(md, inPlace)
		}
		fn.DropFinalize(de)
	})
}

// DragFiles returns the files of this node and the other selected nodes,
// for dragging them to other apps (see [core.FileDragger]).
func (fn *Node) DragFiles() []string {
	files := []string{string(fn.Filepath)}
	for _, sn := range fn.GetSelectedNodes() {
		if sfn := AsNode(sn); sfn != nil && sfn != fn {
			files = append(files, string(sfn.Filepath))
		}
	}
	return topPaths(files)
}

// filePaths returns the file paths in the given mime data from [Node.MimeData],
// which has three items per file, with the file path second.
func filePaths(md mimedata.Mimes) []string {
	nf := len(md) / 3
	paths := make([]string, nf)
	for i := range nf {
		paths[i] = string(md[i*3+1].Data)
	}
	return paths
}

// isNodeData returns whether the given mime data is from [Node.MimeData].
func isNodeData(md mimedata.Mimes) bool {
	if len(md) == 0 || len(md)%3 != 0 {
		return false
	}
	for i := 0; i < len(md); i += 3 {
		if md[i].Type != fileinfo.TextPlain || md[i+1].Type != fileinfo.TextPlain || !filepath.IsAbs(string(md[i+1].Data)) {
			return false
		}
	}
	return true
}

// clipboardPaths returns the paths of the existing files in the given
// clipboard data: those from [Node.MimeData], or the absolute paths
// (or file:// URLs) on the lines of text.
func clipboardPaths(md mimedata.Mimes) []string {
	if isNodeData(md) {
		return filePaths(md)
	}
	var paths []string
	for _, d := range md {
		if !mimedata.IsText(d.Type) {
			continue
		}
		for _, ln := range strings.Split(string(d.Data), "\n") {
			p := strings.TrimPrefix(strings.TrimSpace(ln), "file://")
			if filepath.IsAbs(p) && fsx.Exists(p) {
				paths = append(paths, p)
			}
		}
	}
	return paths
}

// withoutFiles returns the given mime data from [Node.MimeData]
// without the items for the given file paths.
func withoutFiles(md mimedata.Mimes, paths []string) mimedata.Mimes {
	var res mimedata.Mimes
	for i := 0; i+2 < len(md); i += 3 {
		if !slices.Contains(paths, string(md[i+1].Data)) {
			res = append(res, md[i:i+3]...)
		}
	}
	return res
}

// pasteDone reports any error from copying the files of a paste or drop,
// and otherwise calls dropFinal if non-nil, which deletes the source
// files of a move (except those in inPlace): these are kept if anything failed.
func (fn *Node) pasteDone(err error, inPlace []string, dropFinal func(inPlace []string)) {
	if err != nil {
		core.ErrorSnackbar(fn, err, "Error copying file")
		return
	}
	if dropFinal != nil {
		dropFinal(inPlace)
	}
}

// pasteFiles copies the files at the given paths into or onto this node,
// for a paste or drop. For a move, dropFinal deletes the source files,
// except the given ones that were already in place (it is nil for a paste).
// Multiple files, and directories, are copied into this directory or the
// directory of this file. A single file can also replace this file.
func (fn *Node) pasteFiles(paths []string, move bool, dropFinal func(inPlace []string)) {
	paths = topPaths(paths)
	if len(paths) == 0 || fn.isExternal() {
		return
	}
	if fn.IsDir() {
		fn.pasteToDir(fn, paths, move, dropFinal)
		return
	}
	if len(paths) > 1 || errors.Ignore1(fsx.DirExists(paths[0])) {
		fn.pasteToDir(AsNode(fn.Parent), paths, move, dropFinal)
		return
	}
	fn.pasteOntoFile(paths[0], dropFinal)
}

// pasteToDir copies the files at the given paths into the given directory,
// first asking whether to overwrite any existing files. Files that are already
// in the directory are duplicated for a copy, and left alone for a move.
func (fn *Node) pasteToDir(tdir *Node, paths []string, move bool, dropFinal func(inPlace []string)) {
	dir := string(tdir.Filepath)
	var copies, inPlace, existing []string
	for _, p := range paths {
		tp := filepath.Join(dir, filepath.Base(p))
		switch {
		case tp == filepath.Clean(p):
			inPlace = append(inPlace, p)
		case fsx.Exists(tp):
			existing = append(existing, filepath.Base(p))
			fallthrough
		default:
			copies = append(copies, p)
		}
	}
	paste := func() {
		var errs []error
		if !move {
			for _, p := range inPlace {
				fi, err := fileinfo.NewFileInfo(p)
				if err == nil {
					_, err = fi.Duplicate()
				}
				errs = append(errs, err)
			}
		}
		for _, p := range copies {
			errs = append(errs, tdir.copyFileToDir(p))
		}
		fn.FileRoot().UpdatePath(dir)
		fn.pasteDone(errors.Join(errs...), inPlace, dropFinal)
	}
	if len(existing) == 0 {
		paste()
		return
	}
	d := core.NewBody("Overwrite?")
	core.NewText(d).SetType(core.TextSupporting).SetText(fmt.Sprintf("Overwrite existing file(s) in %s: %s?", dir, strings.Join(existing, ", ")))
	d.AddBottomBar(func(bar *core.Frame) {
		d.AddCancel(bar)
		d.AddOK(bar).SetText("Overwrite").OnClick(func(e events.Event) {
			paste()
		})
	})
	d.RunDialog(fn)
}

// pasteOntoFile copies the file at the given path onto this file, or into
// the directory of this file, as chosen by the user, who can also compare them.
func (fn *Node) pasteOntoFile(srcpath string, dropFinal func(inPlace []string)) {
	tpath := string(fn.Filepath)
	if filepath.Clean(srcpath) == tpath { // dropped onto itself
		return
	}
	fname := filepath.Base(srcpath)
	tdir := AsNode(fn.Parent)
	npath := filepath.Join(string(tdir.Filepath), fname) // the source if in this directory
	copyTo := func(dst string) {
		err := fsx.CopyPath(dst, srcpath)
		fn.FileRoot().UpdatePath(dst)
		fn.pasteDone(err, nil, dropFinal)
	}
	d := core.NewBody("Overwrite?")
	text := core.NewText(d).SetType(core.TextSupporting)
	d.AddBottomBar(func(bar *core.Frame) {
		d.AddCancel(bar)
		switch {
		case fname == fn.Name:
			text.SetText(fmt.Sprintf("Overwrite target file: %s with source file of same name?, or diff (compare) two files?", fn.Name))
			d.AddOK(bar).SetText("Diff (compare)").OnClick(func(e events.Event) {
				textcore.DiffFiles(fn, tpath, srcpath)
			})
			d.AddOK(bar).SetText("Overwrite").OnClick(func(e events.Event) {
				copyTo(tpath)
			})
		case npath == filepath.Clean(srcpath) || !fsx.Exists(npath):
			if npath == filepath.Clean(srcpath) { // in this directory: can only replace the target
				text.SetText(fmt.Sprintf("Overwrite target file: %s with source file: %s, or diff (compare) the two files?", fn.Name, fname))
			} else {
				text.SetText(fmt.Sprintf("Overwrite target file: %s with source file: %s, or copy to: %s in current folder (which doesn't yet exist), or diff (compare) the two files?", fn.Name, fname, fname))
			}
			d.AddOK(bar).SetText("Diff (compare)").OnClick(func(e events.Event) {
				textcore.DiffFiles(fn, tpath, srcpath)
			})
			d.AddOK(bar).SetText("Overwrite target").OnClick(func(e events.Event) {
				copyTo(tpath)
			})
			if npath != filepath.Clean(srcpath) {
				d.AddOK(bar).SetText("Copy new file").OnClick(func(e events.Event) {
					copyTo(npath)
				})
			}
		default:
			text.SetText(fmt.Sprintf("Overwrite target file: %s with source file: %s, or overwrite existing file with same name as source file (%s), or diff (compare) files?", fn.Name, fname, fname))
			d.AddOK(bar).SetText("Diff to target").OnClick(func(e events.Event) {
				textcore.DiffFiles(fn, tpath, srcpath)
			})
			d.AddOK(bar).SetText("Diff to existing").OnClick(func(e events.Event) {
				textcore.DiffFiles(fn, npath, srcpath)
			})
			d.AddOK(bar).SetText("Overwrite target").OnClick(func(e events.Event) {
				copyTo(tpath)
			})
			d.AddOK(bar).SetText("Overwrite existing").OnClick(func(e events.Event) {
				copyTo(npath)
			})
		}
	})
	d.RunDialog(fn)
}

// Dragged is called after target accepts the drop -- we just remove
// elements that were moved
// satisfies core.DragNDropper interface and can be overridden by subtypes
func (fn *Node) DropDeleteSource(e events.Event) {
	de := e.(*events.DragDrop)
	froot := fn.FileRoot()
	if froot == nil || fn.isExternal() {
		return
	}
	md := de.Data.(mimedata.Mimes)
	top := topPaths(filePaths(md))
	nf := len(md) / 3 // always internal
	var errs []error
	for i := 0; i < nf; i++ {
		if !slices.Contains(top, string(md[i*3+1].Data)) { // deleted with its directory
			continue
		}
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
		errs = append(errs, sfn.DeleteFile())
	}
	core.ErrorSnackbar(froot, errors.Join(errs...), "Error moving file")
}
