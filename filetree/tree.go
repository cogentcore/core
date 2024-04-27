// Copyright (c) 2023, Cogent Core. All rights reserved.
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

	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/fileinfo/vcs"
	"cogentcore.org/core/gox/dirs"
	"cogentcore.org/core/system"
	"cogentcore.org/core/tree"
	"cogentcore.org/core/types"
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
	ExtFiles []string `set:"-"`

	// records state of directories within the tree (encoded using paths relative to root),
	// e.g., open (have been opened by the user) -- can persist this to restore prior view of a tree
	Dirs DirFlagMap `set:"-"`

	// if true, then all directories are placed at the top of the tree view
	// otherwise everything is mixed
	DirsOnTop bool

	// type of node to create -- defaults to filetree.Node but can use custom node types
	FileNodeType *types.Type `view:"-" json:"-" xml:"-"`

	// DoubleClickFun is a function to call when a node receives a DoubleClick event.
	// if not set, defaults to OpenEmptyDir() (for folders)
	DoubleClickFun func(e events.Event) `copier:"-" view:"-" json:"-" xml:"-"`

	// if true, we are in midst of an OpenAll call -- nodes should open all dirs
	InOpenAll bool `copier:"-" set:"-"`

	// change notify for all dirs
	Watcher *fsnotify.Watcher `copier:"-" set:"-" view:"-"`

	// channel to close watcher watcher
	DoneWatcher chan bool `copier:"-" set:"-" view:"-"`

	// map of paths that have been added to watcher -- only active if bool = true
	WatchedPaths map[string]bool `copier:"-" set:"-" view:"-"`

	// last path updated by watcher
	LastWatchUpdate string `copier:"-" set:"-" view:"-"`

	// timestamp of last update
	LastWatchTime time.Time `copier:"-" set:"-" view:"-"`

	// Update mutex
	UpdateMu sync.Mutex `copier:"-" set:"-" view:"-"`
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

// OpenPath opens the filetree at the given directory path. It reads all the files at
// the given path into this tree. It uses config children to preserve extra info
// already stored about files. Only paths listed in Dirs will be opened.
func (ft *Tree) OpenPath(path string) {
	ft.FRoot = ft // we are our own root..
	ft.Nm = "/"
	if ft.FileNodeType == nil {
		ft.FileNodeType = NodeType
	}
	effpath, err := filepath.EvalSymlinks(path)
	if err != nil {
		effpath = path
	}
	abs, err := filepath.Abs(effpath)
	if err != nil {
		log.Printf("views.Tree:OpenPath: %s\n", err)
		abs = effpath
	}
	ft.FPath = core.Filename(abs)
	ft.Open()
	ft.SetDirOpen(core.Filename(abs))
	ft.UpdateAll()
}

// UpdateAll does a full update of the tree -- calls ReadDir on current path
func (ft *Tree) UpdateAll() {
	// updt := ft.AsyncLock() // note: safe for async updating
	ft.UpdateMu.Lock()
	ft.Dirs.ClearMarks()
	ft.ReadDir(string(ft.FPath))
	// the problem here is that closed dirs are not visited but we want to keep their settings:
	// ft.Dirs.DeleteStale()
	ft.Update()
	ft.TreeViewChanged(nil)
	ft.UpdateMu.Unlock()
	// ft.AsyncUnlock(updt) // todo:
}

// UpdatePath updates the tree at the directory level for given path
// and everything below it.  It flags that it needs render update,
// but if a deletion or insertion happened, then NeedsLayout should also
// be called.
func (ft *Tree) UpdatePath(path string) {
	ft.NeedsRender()
	path = filepath.Clean(path)
	ft.DirsTo(path)
	if fn, ok := ft.FindFile(path); ok {
		if fn.IsDir() {
			fn.UpdateNode()
			return
		}
	}
	fpath, _ := filepath.Split(path)
	if fn, ok := ft.FindFile(fpath); ok {
		fn.UpdateNode()
		return
	}
	// core.MessageSnackbar(ft, "UpdatePath: path not found in tree: "+path)
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
					ft.WatchUpdate(event.Name)
				}
			case err := <-watch.Errors:
				_ = err
			}
		}
	}()
}

// WatchUpdate does the update for given path
func (ft *Tree) WatchUpdate(path string) {
	ft.UpdateMu.Lock()
	defer ft.UpdateMu.Unlock()
	// fmt.Println(path)

	dir, _ := filepath.Split(path)
	rp := ft.RelPath(core.Filename(dir))
	if rp == ft.LastWatchUpdate {
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
	ft.LastWatchUpdate = rp
	ft.LastWatchTime = time.Now()
	if !fn.IsOpen() {
		// fmt.Printf("warning: watcher updating closed node: %s\n", rp)
		return // shouldn't happen
	}
	// update node
	fn.UpdateNode()
}

// WatchPath adds given path to those watched
func (ft *Tree) WatchPath(path core.Filename) error {
	return nil // TODO: disable for all platforms for now -- getting some issues
	if core.TheApp.Platform() == system.MacOS {
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
func (ft *Tree) UnWatchPath(path core.Filename) {
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
func (ft *Tree) IsDirOpen(fpath core.Filename) bool {
	if fpath == ft.FPath { // we are always open
		return true
	}
	return ft.Dirs.IsOpen(ft.RelPath(fpath))
}

// SetDirOpen sets the given directory path to be open
func (ft *Tree) SetDirOpen(fpath core.Filename) {
	rp := ft.RelPath(fpath)
	// fmt.Printf("setdiropen: %s\n", rp)
	ft.Dirs.SetOpen(rp, true)
	ft.Dirs.SetMark(rp)
	ft.WatchPath(fpath)
}

// SetDirClosed sets the given directory path to be closed
func (ft *Tree) SetDirClosed(fpath core.Filename) {
	rp := ft.RelPath(fpath)
	ft.Dirs.SetOpen(rp, false)
	ft.Dirs.SetMark(rp)
	ft.UnWatchPath(fpath)
}

// SetDirSortBy sets the given directory path sort by option
func (ft *Tree) SetDirSortBy(fpath core.Filename, modTime bool) {
	ft.Dirs.SetSortBy(ft.RelPath(fpath), modTime)
}

// DirSortByName returns true if dir is sorted by name
func (ft *Tree) DirSortByName(fpath core.Filename) bool {
	return ft.Dirs.SortByName(ft.RelPath(fpath))
}

// DirSortByModTime returns true if dir is sorted by mod time
func (ft *Tree) DirSortByModTime(fpath core.Filename) bool {
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
	ekid := ft.ChildByName(ExternalFilesName, 0)
	if ekid == nil {
		return nil, fmt.Errorf("ExtFile not updated -- no ExtFiles node")
	}
	ekids := *ekid.Children()
	err := ekids.IsValidIndex(i)
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
	config := tree.Config{}
	typ := ft.FileNodeType
	for _, f := range ft.ExtFiles {
		config.Add(typ, dirs.DirAndFile(f))
	}
	efn.ConfigChildren(config) // NOT unique names
	// always go through kids, regardless of mods
	for i, sfk := range efn.Kids {
		sf := AsNode(sfk)
		sf.FRoot = ft
		fp := ft.ExtFiles[i]
		sf.SetNodePath(fp)
		sf.Info.VCS = vcs.Stored // no vcs in general
	}
}
