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

	"goki.dev/gi/v2/gi"
	"goki.dev/gi/v2/giv"
	"goki.dev/girl/states"
	"goki.dev/goosi"
	"goki.dev/goosi/events"
	"goki.dev/pi/v2/filecat"
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

// OpenFilesDefault opens selected files with default app for that file type (os defined).
// runs open on Mac, xdg-open on Linux, and start on Windows
func (fn *Node) OpenFilesDefault() { //gti:add
	sels := fn.SelectedViews()
	for i := len(sels) - 1; i >= 0; i-- {
		sn := AsNode(sels[i].This())
		sn.OpenFileDefault()
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
		giv.NewFuncButton(sn, sn.OpenFileWith).CallFunc()
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

// makes a copy of selected files
func (fn *Node) DuplicateFiles() { //gti:add
	sels := fn.SelectedViews()
	for i := len(sels) - 1; i >= 0; i-- {
		sn := AsNode(sels[i].This())
		sn.DuplicateFile()
	}
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
	gi.NewDialog(fn).Title("Delete Files?").
		Prompt("Ok to delete file(s)?  This is not undoable and files are not moving to trash / recycle bin. If any selections are directories all files and subdirectories will also be deleted.").
		Cancel().Ok("Delete Files").
		OnAccept(func(e events.Event) {
			fn.DeleteFilesImpl()
		}).Run()
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
		fn.DeleteFile()
	}
	root.UpdateDir()
}

// DeleteFile deletes this file
func (fn *Node) DeleteFile() (err error) {
	if fn.IsExternal() {
		return nil
	}
	fn.CloseBuf()
	repo, _ := fn.Repo()
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
	sels := fn.SelectedViews()
	for i := len(sels) - 1; i >= 0; i-- {
		sn := AsNode(sels[i].This())
		if sn.IsExternal() {
			continue
		}
		giv.NewFuncButton(sn, sn.RenameFile).CallFunc()
	}
}

// RenameFile renames file to new name
func (fn *Node) RenameFile(newpath string) (err error) { //gti:add
	if fn.IsExternal() {
		return nil
	}
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
	}
	if stored {
		fn.AddToVcs()
	} else {
		// fn.SetNeedsRender() // todo
		fn.FRoot.UpdateDir() // need full update
	}
	return err
}

// makes a new file in selected directory
func (fn *Node) NewFiles(filename string, addToVcs bool) { //gti:add
	sels := fn.SelectedViews()
	sz := len(sels)
	if sz == 0 { // shouldn't happen
		return
	}
	sn := AsNode(sels[sz-1])
	sn.NewFile(filename, addToVcs)
}

// NewFile makes a new file in this directory node
func (fn *Node) NewFile(filename string, addToVcs bool) {
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
		// TODO(kai/snack)
		// gi.PromptDialog(nil, gi.DlgOpts{Title: "Couldn't Make File", Prompt: fmt.Sprintf("Could not make new file at: %v, err: %v", np, err), Ok: true, Cancel: false}, nil)
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
	sn.NewFolder(foldername)
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
		// TODO(kai/snack)
		// emsg := fmt.Sprintf("giv.FileNode at: %q: Error: %v", ppath, err)
		// gi.PromptDialog(nil, gi.DlgOpts{Title: "Couldn't Make Folder", Prompt: emsg, Ok: true, Cancel: false}, nil)
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
	filecat.CopyFile(tpath, filename, perm)
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
		d := gi.NewDialog(fn).Title("File info").FullWindow(true)
		giv.NewStructView(d).SetStruct(&fn.Info).SetState(true, states.ReadOnly)
		d.Run()
	}
}
