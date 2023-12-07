// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package filetree

import (
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"

	"goki.dev/fi"
	"goki.dev/gi/v2/gi"
	"goki.dev/gi/v2/giv"
	"goki.dev/goosi"
	"goki.dev/goosi/events"
	"goki.dev/vci/v2"
)

// OSOpenCommand returns the generic file 'open' command to open file with default app
// open on Mac, xdg-open on Linux, and start on Windows
func OSOpenCommand() string {
	switch goosi.TheApp.Platform() {
	case goosi.MacOS:
		return "open"
	case goosi.LinuxX11:
		return "xdg-open"
	case goosi.Windows:
		return "start"
	}
	return "open"
}

// Filer is an interface for Filetree File actions
type Filer interface {
	// OpenFilesDefault opens selected files with default app for that file type (os defined).
	// runs open on Mac, xdg-open on Linux, and start on Windows
	OpenFilesDefault()

	// OpenFileDefault opens file with default app for that file type (os defined)
	// runs open on Mac, xdg-open on Linux, and start on Windows
	OpenFileDefault() error

	// OpenFilesWith opens selected files with user-specified command.
	OpenFilesWith()

	// OpenFileWith opens file with given command.
	// does not wait for command to finish in this routine (separate routine Waits)
	OpenFileWith(command string) error

	// DuplicateFiles makes a copy of selected files
	DuplicateFiles()

	// DuplicateFile creates a copy of given file -- only works for regular files, not
	// directories
	DuplicateFile() error

	// DeleteFiles deletes any selected files or directories. If any directory is selected,
	// all files and subdirectories in that directory are also deleted.
	DeleteFiles()

	// DeleteFilesImpl does the actual deletion, no prompts
	DeleteFilesImpl()

	// DeleteFile deletes this file
	DeleteFile() error

	// RenameFiles renames any selected files
	RenameFiles()

	// RenameFile renames file to new name
	RenameFile(newpath string) error

	// NewFiles makes a new file in selected directory
	NewFiles(filename string, addToVcs bool)

	// NewFile makes a new file in this directory node
	NewFile(filename string, addToVcs bool)

	// NewFolders makes a new folder in the given selected directory
	NewFolders(foldername string)

	// NewFolder makes a new folder (directory) in this directory node
	NewFolder(foldername string)

	// CopyFileToDir copies given file path into node that is a directory.
	// This does NOT check for overwriting -- that must be done at higher level!
	CopyFileToDir(filename string, perm os.FileMode)

	// Shows file information about selected file(s)
	ShowFileInfo()
}

// check for interface impl
var _ Filer = (*Node)(nil)

// OpenFilesDefault opens selected files with default app for that file type (os defined).
// runs open on Mac, xdg-open on Linux, and start on Windows
func (fn *Node) OpenFilesDefault() { //gti:add
	sels := fn.SelectedViews()
	for i := len(sels) - 1; i >= 0; i-- {
		sn := AsNode(sels[i].This())
		if sn == nil {
			continue
		}
		sn.This().(Filer).OpenFileDefault()
	}
}

// OpenFileDefault opens file with default app for that file type (os defined)
// runs open on Mac, xdg-open on Linux, and start on Windows
func (fn *Node) OpenFileDefault() error {
	cstr := OSOpenCommand()
	cmd := exec.Command(cstr, string(fn.FPath))
	out, err := cmd.CombinedOutput()
	fmt.Printf("%s\n", out)
	return err
}

// OpenFilesWith opens selected files with user-specified command.
func (fn *Node) OpenFilesWith() {
	sels := fn.SelectedViews()
	for i := len(sels) - 1; i >= 0; i-- {
		sn := AsNode(sels[i].This())
		if sn == nil {
			continue
		}
		giv.CallFunc(sn, sn.OpenFileWith) // todo: not using interface?
	}
}

// OpenFileWith opens file with given command.
// does not wait for command to finish in this routine (separate routine Waits)
func (fn *Node) OpenFileWith(command string) error {
	cmd := exec.Command(command, string(fn.FPath))
	err := cmd.Start()
	go func() {
		err := cmd.Wait()
		if err != nil {
			slog.Error(err.Error())
		}
	}()
	return err
}

// DuplicateFiles makes a copy of selected files
func (fn *Node) DuplicateFiles() { //gti:add
	root := fn.FRoot
	sels := fn.SelectedViews()
	for i := len(sels) - 1; i >= 0; i-- {
		sn := AsNode(sels[i].This())
		if sn == nil {
			continue
		}
		sn.This().(Filer).DuplicateFile()
	}
	root.UpdateDir()
}

// DuplicateFile creates a copy of given file -- only works for regular files, not
// directories
func (fn *Node) DuplicateFile() error {
	_, err := fn.Info.Duplicate()
	if err == nil && fn.Par != nil {
		fnp := AsNode(fn.Par)
		fnp.UpdateNode()
	}
	return err
}

// deletes any selected files or directories. If any directory is selected,
// all files and subdirectories in that directory are also deleted.
func (fn *Node) DeleteFiles() { //gti:add
	d := gi.NewBody().AddTitle("Delete Files?").
		AddText("Ok to delete file(s)?  This is not undoable and files are not moving to trash / recycle bin. If any selections are directories all files and subdirectories will also be deleted.")
	d.AddBottomBar(func(pw gi.Widget) {
		d.AddCancel(pw)
		d.AddOk(pw).SetText("Delete Files").OnClick(func(e events.Event) {
			fn.This().(Filer).DeleteFilesImpl()
		})
	})
	d.NewDialog(fn).Run()
}

// DeleteFilesImpl does the actual deletion, no prompts
func (fn *Node) DeleteFilesImpl() {
	root := fn.FRoot
	sels := fn.SelectedViews()
	for i := len(sels) - 1; i >= 0; i-- {
		fn := AsNode(sels[i].This())
		if !fn.Info.IsDir() {
			fn.DeleteFile()
			continue
		}
		openList := []string{}
		var fns []string
		fn.Info.FileNames(&fns)
		ft := fn.FRoot
		for _, filename := range fns {
			fn, ok := ft.FindFile(filename)
			if !ok {
				return
			}
			if fn.Buf != nil {
				openList = append(openList, filename)
			}
		}
		if len(openList) > 0 {
			for _, filename := range openList {
				fn, _ := ft.FindFile(filename)
				fn.CloseBuf()
			}
		}
		fn.This().(Filer).DeleteFile()
	}
	root.UpdateDir()
}

// DeleteFile deletes this file
func (fn *Node) DeleteFile() error {
	if fn.IsExternal() {
		return nil
	}
	fn.CloseBuf()
	repo, _ := fn.Repo()
	var err error
	if !fn.Info.IsDir() && repo != nil && fn.Info.Vcs >= vci.Stored {
		// fmt.Printf("del repo: %v\n", fn.FPath)
		err = repo.Delete(string(fn.FPath))
	} else {
		// fmt.Printf("del raw: %v\n", fn.FPath)
		err = fn.Info.Delete()
	}
	if err == nil {
		fn.Delete(true)
	}
	return err
}

// renames any selected files
func (fn *Node) RenameFiles() { //gti:add
	root := fn.FRoot
	sels := fn.SelectedViews()
	for i := len(sels) - 1; i >= 0; i-- {
		sn := AsNode(sels[i].This())
		if sn == nil || sn.IsExternal() {
			continue
		}
		giv.CallFunc(sn, sn.RenameFile) // todo: not using interface?
	}
	root.UpdateDir()
}

// RenameFile renames file to new name
func (fn *Node) RenameFile(newpath string) error { //gti:add
	if fn.IsExternal() {
		return nil
	}
	var err error
	fn.CloseBuf() // invalid after this point
	orgpath := fn.FPath
	newpath, err = fn.Info.Rename(newpath)
	if len(newpath) == 0 || err != nil {
		return err
	}
	if fn.IsDir() {
		if fn.FRoot.IsDirOpen(orgpath) {
			fn.FRoot.SetDirOpen(gi.FileName(newpath))
		}
	}
	repo, _ := fn.Repo()
	stored := false
	if fn.IsDir() && !fn.HasChildren() {
		err = os.Rename(string(orgpath), newpath)
	} else if repo != nil && fn.Info.Vcs >= vci.Stored {
		stored = true
		err = repo.Move(string(orgpath), newpath)
	} else {
		err = os.Rename(string(orgpath), newpath)
	}
	if err == nil {
		err = fn.Info.InitFile(newpath)
	}
	if err == nil {
		fn.FPath = gi.FileName(fn.Info.Path)
		fn.SetName(fn.Info.Name)
		fn.SetText(fn.Info.Name)
	}
	if stored {
		fn.AddToVcs()
	}
	return err
}

// NewFiles makes a new file in selected directory
func (fn *Node) NewFiles(filename string, addToVcs bool) { //gti:add
	sels := fn.SelectedViews()
	sz := len(sels)
	if sz == 0 { // shouldn't happen
		return
	}
	sn := AsNode(sels[sz-1])
	if sn == nil {
		return
	}
	sn.This().(Filer).NewFile(filename, addToVcs)
}

// NewFile makes a new file in this directory node
func (fn *Node) NewFile(filename string, addToVcs bool) { //gti:add
	if fn.IsExternal() {
		return
	}
	ppath := string(fn.FPath)
	if !fn.IsDir() {
		ppath, _ = filepath.Split(ppath)
	}
	np := filepath.Join(ppath, filename)
	_, err := os.Create(np)
	if err != nil {
		gi.ErrorSnackbar(fn, err)
		return
	}
	fn.FRoot.UpdateNewFile(np)
	if addToVcs {
		nfn, ok := fn.FRoot.FindFile(np)
		if ok && nfn.This() != fn.FRoot.This() {
			nfn.AddToVcs()
		}
	}
}

// makes a new folder in the given selected directory
func (fn *Node) NewFolders(foldername string) { //gti:add
	sels := fn.SelectedViews()
	sz := len(sels)
	if sz == 0 { // shouldn't happen
		return
	}
	sn := AsNode(sels[sz-1])
	if sn == nil {
		return
	}
	sn.This().(Filer).NewFolder(foldername)
}

// NewFolder makes a new folder (directory) in this directory node
func (fn *Node) NewFolder(foldername string) { //gti:add
	if fn.IsExternal() {
		return
	}
	ppath := string(fn.FPath)
	if !fn.IsDir() {
		ppath, _ = filepath.Split(ppath)
	}
	np := filepath.Join(ppath, foldername)
	err := os.MkdirAll(np, 0775)
	if err != nil {
		gi.ErrorSnackbar(fn, err)
		return
	}
	fn.FRoot.UpdateNewFile(ppath)
}

// CopyFileToDir copies given file path into node that is a directory.
// This does NOT check for overwriting -- that must be done at higher level!
func (fn *Node) CopyFileToDir(filename string, perm os.FileMode) {
	if fn.IsExternal() {
		return
	}
	ppath := string(fn.FPath)
	sfn := filepath.Base(filename)
	tpath := filepath.Join(ppath, sfn)
	fi.CopyFile(tpath, filename, perm)
	fn.FRoot.UpdateNewFile(ppath)
	ofn, ok := fn.FRoot.FindFile(filename)
	if ok && ofn.Info.Vcs >= vci.Stored {
		nfn, ok := fn.FRoot.FindFile(tpath)
		if ok && nfn.This() != fn.FRoot.This() {
			nfn.AddToVcs()
			nfn.UpdateNode()
		}
	}
}

// Shows file information about selected file(s)
func (fn *Node) ShowFileInfo() { //gti:add
	sels := fn.SelectedViews()
	for i := len(sels) - 1; i >= 0; i-- {
		fn := AsNode(sels[i].This())
		d := gi.NewBody().AddTitle("File info")
		giv.NewStructView(d).SetStruct(&fn.Info).SetReadOnly(true)
		d.AddOkOnly().NewDialog(fn).Run()
	}
}
