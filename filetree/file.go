// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package filetree

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"cogentcore.org/core/base/fsx"
	"cogentcore.org/core/base/vcs"
	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/tree"
)

// Filer is an interface for file tree file actions that all [Node]s satisfy.
// This allows apps to intervene and apply any additional logic for these actions.
type Filer interface { //types:add
	core.Treer

	// AsFileNode returns the [Node]
	AsFileNode() *Node

	// RenameFiles renames any selected files.
	RenameFiles()

	// DeleteFiles deletes any selected files.
	DeleteFiles()

	// GetFileInfo updates the .Info for this file
	GetFileInfo() error

	// OpenFile opens the file for node. This is called by OpenFilesDefault
	OpenFile() error
}

var _ Filer = (*Node)(nil)

// SelectedPaths returns the paths of selected nodes.
func (fn *Node) SelectedPaths() []string {
	sels := fn.GetSelectedNodes()
	n := len(sels)
	if n == 0 {
		return nil
	}
	paths := make([]string, 0, n)
	fn.SelectedFunc(func(sn *Node) {
		paths = append(paths, string(sn.Filepath))
	})
	return paths
}

// OpenFilesDefault opens selected files with default app for that file type (os defined).
// runs open on Mac, xdg-open on Linux, and start on Windows
func (fn *Node) OpenFilesDefault() { //types:add
	fn.SelectedFunc(func(sn *Node) {
		sn.This.(Filer).OpenFile()
	})
}

// OpenFile just does OpenFileDefault
func (fn *Node) OpenFile() error {
	return fn.OpenFileDefault()
}

// OpenFileDefault opens file with default app for that file type (os defined)
// runs open on Mac, xdg-open on Linux, and start on Windows
func (fn *Node) OpenFileDefault() error {
	core.TheApp.OpenURL(fsx.FileURL(string(fn.Filepath)))
	return nil
}

// duplicateFiles makes a copy of selected files
func (fn *Node) duplicateFiles() { //types:add
	root := fn.FileRoot()
	root.NeedsLayout()
	var errs []error
	fn.SelectedFunc(func(sn *Node) {
		errs = append(errs, sn.duplicateFile())
	})
	core.ErrorSnackbar(root, errors.Join(errs...), "Error duplicating file")
}

// duplicateFile creates a copy of given file -- only works for regular files, not
// directories
func (fn *Node) duplicateFile() error {
	_, err := fn.Info.Duplicate()
	if err == nil && fn.Parent != nil {
		fnp := AsNode(fn.Parent)
		fnp.Update()
	}
	return err
}

// DeleteFiles deletes any selected files or directories. If any directory is selected,
// all files and subdirectories in that directory are also deleted.
func (fn *Node) DeleteFiles() { //types:add
	d := core.NewBody("Delete Files?")
	core.NewText(d).SetType(core.TextSupporting).SetText("OK to delete file(s)? They are moved to the trash / recycle bin where available, and otherwise permanently deleted. If any selections are directories, all files and subdirectories in them are also deleted.")
	d.AddBottomBar(func(bar *core.Frame) {
		d.AddCancel(bar)
		d.AddOK(bar).SetText("Delete Files").OnClick(func(e events.Event) {
			fn.DeleteFilesNoPrompts()
		})
	})
	d.RunDialog(fn)
}

// DeleteFilesNoPrompts does the actual deletion, no prompts.
// Errors are reported in a snackbar.
func (fn *Node) DeleteFilesNoPrompts() {
	root := fn.FileRoot()
	root.NeedsLayout()
	var errs []error
	for _, sn := range fn.selectedTop() {
		errs = append(errs, sn.DeleteFile())
	}
	core.ErrorSnackbar(root, errors.Join(errs...), "Error deleting file")
}

// DeleteFile deletes this file
func (fn *Node) DeleteFile() error {
	if fn.isExternal() {
		return nil
	}
	pari := fn.Parent
	var parent *Node
	if pari != nil {
		parent = AsNode(pari)
	}
	repo, _ := fn.Repo()
	var err error
	if !fn.Info.IsDir() && repo != nil && fn.Info.VCS >= vcs.Stored {
		// fmt.Printf("del repo: %v\n", fn.FPath)
		err = repo.Delete(string(fn.Filepath))
	} else {
		// fmt.Printf("del raw: %v\n", fn.FPath)
		err = fn.Info.Delete()
	}
	if err == nil {
		fn.Delete()
	}
	if parent != nil {
		parent.Update()
	}
	return err
}

// renames any selected files
func (fn *Node) RenameFiles() { //types:add
	fn.FileRoot().NeedsLayout()
	fn.SelectedFunc(func(sn *Node) {
		fb := core.NewSoloFuncButton(sn).SetFunc(sn.RenameFile)
		fb.Args[0].SetValue(sn.Name)
		fb.CallFunc()
	})
}

// RenameFile renames file to new name
func (fn *Node) RenameFile(newpath string) error { //types:add
	if fn.isExternal() {
		return nil
	}
	root := fn.FileRoot()
	var err error
	orgpath := fn.Filepath
	newpath, err = fn.Info.RenamePath(newpath) // the rename is done below
	if len(newpath) == 0 || err != nil {
		return err
	}
	if fn.IsDir() {
		if fn.FileRoot().isDirOpen(orgpath) {
			fn.FileRoot().setDirOpen(core.Filename(newpath))
		}
	}
	repo, _ := fn.Repo()
	if repo != nil && (fn.IsDir() || fn.Info.VCS >= vcs.Stored) {
		err = repo.Move(string(orgpath), newpath)
		if err != nil && fn.IsDir() { // no files in the repository
			err = os.Rename(string(orgpath), newpath)
		}
	} else {
		err = os.Rename(string(orgpath), newpath)
	}
	if err == nil {
		err = fn.Info.InitFile(newpath)
	}
	if err == nil {
		fn.Filepath = core.Filename(fn.Info.Path)
		fn.SetName(fn.Info.Name)
		fn.SetText(fn.Info.Name)
		if fn.IsDir() {
			fn.movedDir(string(orgpath))
		}
	}
	if root != nil {
		root.UpdatePath(string(orgpath))
		root.UpdatePath(newpath)
	}
	return err
}

// movedDir updates the paths of the nodes within this directory node,
// which has been renamed (moved) from the given path, along with the
// open state of the directories. Existing nodes are kept by name when
// the directory is updated, with their paths unchanged.
func (fn *Node) movedDir(orgpath string) {
	root := fn.FileRoot()
	newpath := string(fn.Filepath)
	fn.WalkDown(func(k tree.Node) bool {
		sfn := AsNode(k)
		if sfn == nil || sfn == fn {
			return tree.Continue
		}
		opath := string(sfn.Filepath)
		if !isWithin(opath, orgpath) {
			return tree.Continue
		}
		sfn.Filepath = core.Filename(newpath + opath[len(orgpath):])
		sfn.This.(Filer).GetFileInfo()
		if sfn.IsDir() && root.isDirOpen(core.Filename(opath)) {
			root.setDirOpen(sfn.Filepath)
		}
		return tree.Continue
	})
}

// newFiles makes a new file in selected directory
func (fn *Node) newFiles(filename string, addToVCS bool) { //types:add
	done := false
	fn.SelectedFunc(func(sn *Node) {
		if !done {
			sn.newFile(filename, addToVCS)
			done = true
		}
	})
}

// newFile makes a new file in this directory node
func (fn *Node) newFile(filename string, addToVCS bool) { //types:add
	if fn.isExternal() {
		return
	}
	ppath := string(fn.Filepath)
	if !fn.IsDir() {
		ppath, _ = filepath.Split(ppath)
	}
	np := filepath.Join(ppath, filename)
	f, err := os.OpenFile(np, os.O_RDWR|os.O_CREATE|os.O_EXCL, 0666) // not truncating an existing file
	if err == nil {
		err = f.Close()
	}
	if err != nil {
		core.ErrorSnackbar(fn, err, "Error creating file")
		return
	}
	if addToVCS {
		fn.FileRoot().UpdatePath(ppath)
		nfn, ok := fn.FileRoot().FindFile(np)
		if ok && !nfn.IsRoot() && string(nfn.Filepath) == np {
			core.MessageSnackbar(fn, "Adding new file to VCS: "+fsx.DirAndFile(string(nfn.Filepath)))
			nfn.AddToVCS()
		}
	}
	fn.FileRoot().UpdatePath(ppath)
}

// makes a new folder in the given selected directory
func (fn *Node) newFolders(foldername string) { //types:add
	done := false
	fn.SelectedFunc(func(sn *Node) {
		if !done {
			sn.newFolder(foldername)
			done = true
		}
	})
}

// newFolder makes a new folder (directory) in this directory node
func (fn *Node) newFolder(foldername string) { //types:add
	if fn.isExternal() {
		return
	}
	ppath := string(fn.Filepath)
	if !fn.IsDir() {
		ppath, _ = filepath.Split(ppath)
	}
	np := filepath.Join(ppath, foldername)
	err := os.MkdirAll(np, 0775)
	if err != nil {
		core.ErrorSnackbar(fn, err, "Error creating folder")
		return
	}
	fn.FileRoot().UpdatePath(ppath)
}

// copyFileToDir copies the file or directory at the given path into this
// directory node, overwriting any existing file. A copy of a file that is
// in version control in this tree is added to version control.
func (fn *Node) copyFileToDir(filename string) error {
	if fn.isExternal() {
		return nil
	}
	ppath := string(fn.Filepath)
	tpath := filepath.Join(ppath, filepath.Base(filename))
	err := fsx.CopyPath(tpath, filename)
	fn.FileRoot().UpdatePath(ppath)
	if err != nil {
		return err
	}
	ofn, ok := fn.FileRoot().FindFile(filename)
	if ok && !ofn.IsDir() && ofn.Info.VCS >= vcs.Stored {
		nfn, ok := fn.FileRoot().FindFile(tpath)
		if ok && !nfn.IsRoot() {
			if string(nfn.Filepath) != tpath {
				fmt.Printf("error: nfn.FPath != tpath; %q != %q, see bug #453\n", nfn.Filepath, tpath)
			} else {
				nfn.AddToVCS() // todo: this sometimes is not just tpath!  See bug #453
			}
			nfn.Update()
		}
	}
	return nil
}

// Shows file information about selected file(s)
func (fn *Node) showFileInfo() { //types:add
	fn.SelectedFunc(func(sn *Node) {
		d := core.NewBody("File info")
		core.NewForm(d).SetStruct(&sn.Info).SetReadOnly(true)
		d.AddOKOnly().RunWindowDialog(sn)
	})
}
