// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package filetree

import (
	"fmt"
	"log"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"time"

	"goki.dev/gi/v2/gi"
	"goki.dev/glop/dirs"
	"goki.dev/goosi"
	"goki.dev/gti"
	"goki.dev/ki/v2"
	"goki.dev/vci/v2"
	"gopkg.in/fsnotify.v1"
)

const (
	// ExternalFilesName is the name of the node that represents external files
	ExternalFilesName = "[external files]"
)

// Tree is the root of a tree representing files in a given directory
// (and subdirectories thereof), and has some overall management state for how to
// view things.  The Tree can be viewed by a TreeView to provide a GUI
// interface into it.
type Tree struct {
	Node

	// external files outside the root path of the tree -- abs paths are stored -- these are shown in the first sub-node if present -- use AddExtFile to add and update
	ExtFiles []string

	// records state of directories within the tree (encoded using paths relative to root),
	// e.g., open (have been opened by the user) -- can persist this to restore prior view of a tree
	Dirs DirFlagMap

	// if true, then all directories are placed at the top of the tree view
	// otherwise everything is mixed
	DirsOnTop bool

	// type of node to create -- defaults to giv.Node but can use custom node types
	NodeType *gti.Type `view:"-" json:"-" xml:"-"`

	// if true, we are in midst of an OpenAll call -- nodes should open all dirs
	InOpenAll bool

	// change notify for all dirs
	Watcher *fsnotify.Watcher `view:"-"`

	// channel to close watcher watcher
	DoneWatcher chan bool `view:"-"`

	// map of paths that have been added to watcher -- only active if bool = true
	WatchedPaths map[string]bool `view:"-"`

	// last path updated by watcher
	LastWatchUpdt string `view:"-"`

	// timestamp of last update
	LastWatchTime time.Time `view:"-"`

	// Update mutex
	UpdtMu sync.Mutex `view:"-"`
}

func (ft *Tree) CopyFieldsFrom(frm any) {
	fr := frm.(*Tree)
	ft.Node.CopyFieldsFrom(&fr.Node)
	ft.DirsOnTop = fr.DirsOnTop
	ft.NodeType = fr.NodeType
}

func (fv *Tree) Destroy() {
	if fv.Watcher != nil {
		fv.Watcher.Close()
		fv.Watcher = nil
	}
	if fv.DoneWatcher != nil {
		fv.DoneWatcher <- true
		close(fv.DoneWatcher)
		fv.DoneWatcher = nil
	}
	fv.TreeView.Destroy()
}

// OpenPath opens a filetree at given directory path -- reads all the files at
// given path into this tree -- uses config children to preserve extra info
// already stored about files.  Only paths listed in Dirs will be opened.
func (ft *Tree) OpenPath(path string) {
	ft.FRoot = ft // we are our own root..
	if ft.NodeType == nil {
		ft.NodeType = NodeType
	}
	effpath, err := filepath.EvalSymlinks(path)
	if err != nil {
		effpath = path
	}
	abs, err := filepath.Abs(effpath)
	if err != nil {
		log.Printf("giv.Tree:OpenPath: %s\n", err)
		abs = effpath
	}
	ft.FPath = gi.FileName(abs)
	ft.Open()
	ft.SetDirOpen(gi.FileName(abs))
	ft.UpdateAll()
}

// UpdateAll does a full update of the tree -- calls ReadDir on current path
func (ft *Tree) UpdateAll() {
	updt := ft.UpdateStartAsync() // note: safe for async updating
	ft.UpdtMu.Lock()
	ft.Dirs.ClearMarks()
	ft.ReadDir(string(ft.FPath))
	// the problem here is that closed dirs are not visited but we want to keep their settings:
	// ft.Dirs.DeleteStale()
	ft.Update()
	ft.TreeViewChanged(nil)
	ft.SetNeedsLayout(true)
	ft.UpdtMu.Unlock()
	ft.UpdateEndAsyncLayout(updt)
}

// UpdatePath updates the tree at the directory level for given path
// and everything below it
// func (ft *Tree) UpdatePath(path string) {
// 	ft.UpdtMu.Lock()
// 	ft.UpdtMu.Unlock()
// }

// todo: rewrite below to use UpdatePath

// UpdateNewFile should be called with path to a new file that has just been
// created -- will update view to show that file, and if that file doesn't
// exist, it updates the directory containing that file
func (ft *Tree) UpdateNewFile(filename string) {
	ft.DirsTo(filename)
	fpath, _ := filepath.Split(filename)
	fpath = filepath.Clean(fpath)
	if fn, ok := ft.FindFile(filename); ok {
		// fmt.Printf("updating node for file: %v\n", filename)
		fn.UpdateNode()
	} else if fn, ok := ft.FindFile(fpath); ok {
		// fmt.Printf("updating node for path: %v\n", fpath)
		fn.UpdateNode()
		// } else {
		// log.Printf("giv.Tree UpdateNewFile: no node found for path to update: %v\n", filename)
	}
}

// ConfigWatcher configures a new watcher for tree
func (ft *Tree) ConfigWatcher() error {
	if ft.Watcher != nil {
		return nil
	}
	ft.WatchedPaths = make(map[string]bool)
	var err error
	ft.Watcher, err = fsnotify.NewWatcher()
	return err
}

// WatchWatcher monitors the watcher channel for update events.
// It must be called once some paths have been added to watcher --
// safe to call multiple times.
func (ft *Tree) WatchWatcher() {
	if ft.Watcher == nil || ft.Watcher.Events == nil {
		return
	}
	if ft.DoneWatcher != nil {
		return
	}
	ft.DoneWatcher = make(chan bool)
	go func() {
		watch := ft.Watcher
		done := ft.DoneWatcher
		for {
			select {
			case <-done:
				return
			case event := <-watch.Events:
				switch {
				case event.Op&fsnotify.Create == fsnotify.Create ||
					event.Op&fsnotify.Remove == fsnotify.Remove ||
					event.Op&fsnotify.Rename == fsnotify.Rename:
					ft.WatchUpdt(event.Name)
				}
			case err := <-watch.Errors:
				_ = err
			}
		}
	}()
}

// WatchUpdt does the update for given path
func (ft *Tree) WatchUpdt(path string) {
	ft.UpdtMu.Lock()
	defer ft.UpdtMu.Unlock()
	// fmt.Println(path)

	dir, _ := filepath.Split(path)
	rp := ft.RelPath(gi.FileName(dir))
	if rp == ft.LastWatchUpdt {
		now := time.Now()
		lagMs := int(now.Sub(ft.LastWatchTime) / time.Millisecond)
		if lagMs < 100 {
			// fmt.Printf("skipping update to: %s  due to lag: %v\n", rp, lagMs)
			return // no update
		}
	}
	fn, err := ft.FindDirNode(rp)
	if err != nil {
		// slog.Error(err.Error())
		return
	}
	ft.LastWatchUpdt = rp
	ft.LastWatchTime = time.Now()
	if !fn.IsOpen() {
		// fmt.Printf("warning: watcher updating closed node: %s\n", rp)
		return // shouldn't happen
	}
	// update node
	fn.UpdateNode()
}

// WatchPath adds given path to those watched
func (ft *Tree) WatchPath(path gi.FileName) error {
	return nil // disable for all platforms for now -- getting some issues
	if goosi.TheApp.Platform() == goosi.MacOS {
		return nil // mac is not supported in a high-capacity fashion at this point
	}
	rp := ft.RelPath(path)
	on, has := ft.WatchedPaths[rp]
	if on || has {
		return nil
	}
	ft.ConfigWatcher()
	// fmt.Printf("watching path: %s\n", path)
	err := ft.Watcher.Add(string(path))
	if err == nil {
		ft.WatchedPaths[rp] = true
		ft.WatchWatcher()
	} else {
		slog.Error(err.Error())
	}
	return err
}

// UnWatchPath removes given path from those watched
func (ft *Tree) UnWatchPath(path gi.FileName) {
	rp := ft.RelPath(path)
	on, has := ft.WatchedPaths[rp]
	if !on || !has {
		return
	}
	ft.ConfigWatcher()
	ft.Watcher.Remove(string(path))
	ft.WatchedPaths[rp] = false
}

// IsDirOpen returns true if given directory path is open (i.e., has been
// opened in the view)
func (ft *Tree) IsDirOpen(fpath gi.FileName) bool {
	if fpath == ft.FPath { // we are always open
		return true
	}
	return ft.Dirs.IsOpen(ft.RelPath(fpath))
}

// SetDirOpen sets the given directory path to be open
func (ft *Tree) SetDirOpen(fpath gi.FileName) {
	rp := ft.RelPath(fpath)
	// fmt.Printf("setdiropen: %s\n", rp)
	ft.Dirs.SetOpen(rp, true)
	ft.Dirs.SetMark(rp)
	ft.WatchPath(fpath)
}

// SetDirClosed sets the given directory path to be closed
func (ft *Tree) SetDirClosed(fpath gi.FileName) {
	rp := ft.RelPath(fpath)
	ft.Dirs.SetOpen(rp, false)
	ft.Dirs.SetMark(rp)
	ft.UnWatchPath(fpath)
}

// SetDirSortBy sets the given directory path sort by option
func (ft *Tree) SetDirSortBy(fpath gi.FileName, modTime bool) {
	ft.Dirs.SetSortBy(ft.RelPath(fpath), modTime)
}

// DirSortByName returns true if dir is sorted by name
func (ft *Tree) DirSortByName(fpath gi.FileName) bool {
	return ft.Dirs.SortByName(ft.RelPath(fpath))
}

// DirSortByModTime returns true if dir is sorted by mod time
func (ft *Tree) DirSortByModTime(fpath gi.FileName) bool {
	return ft.Dirs.SortByModTime(ft.RelPath(fpath))
}

// AddExtFile adds an external file outside of root of file tree
// and triggers an update, returning the Node for it, or
// error if filepath.Abs fails.
func (ft *Tree) AddExtFile(fpath string) (*Node, error) {
	pth, err := filepath.Abs(fpath)
	if err != nil {
		return nil, err
	}
	if _, err := os.Stat(pth); err != nil {
		return nil, err
	}
	if has, _ := ft.HasExtFile(pth); has {
		return ft.ExtNodeByPath(pth)
	}
	ft.ExtFiles = append(ft.ExtFiles, pth)
	ft.UpdateDir()
	return ft.ExtNodeByPath(pth)
}

// RemoveExtFile removes external file from maintained list,  returns true if removed
func (ft *Tree) RemoveExtFile(fpath string) bool {
	for i, ef := range ft.ExtFiles {
		if ef == fpath {
			ft.ExtFiles = append(ft.ExtFiles[:i], ft.ExtFiles[i+1:]...)
			return true
		}
	}
	return false
}

// HasExtFile returns true and index if given abs path exists on ExtFiles list.
// false and -1 if not.
func (ft *Tree) HasExtFile(fpath string) (bool, int) {
	for i, f := range ft.ExtFiles {
		if f == fpath {
			return true, i
		}
	}
	return false, -1
}

// ExtNodeByPath returns Node for given file path, and true, if it
// exists in the external files list.  Otherwise returns nil, false.
func (ft *Tree) ExtNodeByPath(fpath string) (*Node, error) {
	ehas, i := ft.HasExtFile(fpath)
	if !ehas {
		return nil, fmt.Errorf("ExtFile not found on list: %v", fpath)
	}
	ekid, err := ft.ChildByNameTry(ExternalFilesName, 0)
	if err != nil {
		return nil, fmt.Errorf("ExtFile not updated -- no ExtFiles node")
	}
	ekids := *ekid.Children()
	err = ekids.IsValidIndex(i)
	if err == nil {
		kn := AsNode(ekids.Elem(i))
		return kn, nil
	}
	return nil, fmt.Errorf("ExtFile not updated: %v", err)
}

// UpdateExtFiles returns a type-and-name list for configuring nodes
// for ExtFiles
func (ft *Tree) UpdateExtFiles(efn *Node) {
	efn.Info.Mode = os.ModeDir | os.ModeIrregular // mark as dir, irregular
	config := ki.Config{}
	typ := ft.NodeType
	for _, f := range ft.ExtFiles {
		config.Add(typ, dirs.DirAndFile(f))
	}
	mods, updt := efn.ConfigChildren(config) // NOT unique names
	if mods {
		// fmt.Printf("got mods: %v\n", path)
	}
	// always go through kids, regardless of mods
	for i, sfk := range efn.Kids {
		sf := AsNode(sfk)
		sf.FRoot = ft
		fp := ft.ExtFiles[i]
		sf.SetNodePath(fp)
		sf.Info.Vcs = vci.Stored // no vcs in general
	}
	if mods {
		efn.UpdateEnd(updt)
	}
}
