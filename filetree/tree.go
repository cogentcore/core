// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package filetree

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"time"

	"cogentcore.org/core/base/dirs"
	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/base/vcs"
	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/system"
	"cogentcore.org/core/tree"
	"cogentcore.org/core/types"
	"github.com/fsnotify/fsnotify"
)

const (
	// ExternalFilesName is the name of the node that represents external files
	ExternalFilesName = "[external files]"
)

// Tree is the root widget of a file tree representing files in a given directory
// (and subdirectories thereof), and has some overall management state for how to
// view things.
type Tree struct {
	Node

	// ExternalFiles are external files outside the root path of the tree.
	// They are stored in terms of their absolute paths. These are shown
	// in the first sub-node if present; use [Tree.AddExternalFile] to add one.
	ExternalFiles []string `set:"-"`

	// records state of directories within the tree (encoded using paths relative to root),
	// e.g., open (have been opened by the user) -- can persist this to restore prior view of a tree
	Dirs DirFlagMap `set:"-"`

	// if true, then all directories are placed at the top of the tree.
	// Otherwise everything is mixed.
	DirsOnTop bool

	// type of node to create; defaults to [Node] but can use custom node types
	FileNodeType *types.Type `display:"-" json:"-" xml:"-"`

	// DoubleClickFunc is a function to call when a node receives an [events.DoubleClick] event.
	// If not set, defaults to OpenEmptyDir() (for folders).
	DoubleClickFunc func(e events.Event) `copier:"-" display:"-" json:"-" xml:"-"`

	// if true, we are in midst of an OpenAll call; nodes should open all dirs
	inOpenAll bool

	// change notify for all dirs
	watcher *fsnotify.Watcher

	// channel to close watcher watcher
	doneWatcher chan bool

	// map of paths that have been added to watcher; only active if bool = true
	watchedPaths map[string]bool

	// last path updated by watcher
	lastWatchUpdate string

	// timestamp of last update
	lastWatchTime time.Time

	// Update mutex
	updateMu sync.Mutex
}

func (ft *Tree) Init() {
	ft.Node.Init()
	ft.FileRoot = ft
	ft.FileNodeType = NodeType
	ft.OpenDepth = 4
}

func (fv *Tree) Destroy() {
	if fv.watcher != nil {
		fv.watcher.Close()
		fv.watcher = nil
	}
	if fv.doneWatcher != nil {
		fv.doneWatcher <- true
		close(fv.doneWatcher)
		fv.doneWatcher = nil
	}
	fv.Tree.Destroy()
}

// OpenPath opens the filetree at the given directory path. It reads all the files at
// the given path into this tree. Only paths listed in [Tree.Dirs] will be opened.
func (ft *Tree) OpenPath(path string) *Tree {
	if ft.FileNodeType == nil {
		ft.FileNodeType = NodeType
	}
	effpath, err := filepath.EvalSymlinks(path)
	if err != nil {
		effpath = path
	}
	abs, err := filepath.Abs(effpath)
	if errors.Log(err) != nil {
		abs = effpath
	}
	ft.Filepath = core.Filename(abs)
	ft.Open()
	ft.SetDirOpen(core.Filename(abs))
	ft.UpdateAll()
	return ft
}

// UpdateAll does a full update of the tree -- calls SetPath on current path
func (ft *Tree) UpdateAll() {
	// update := ft.AsyncLock() // note: safe for async updating
	ft.updateMu.Lock()
	ft.Dirs.ClearMarks()
	ft.SetPath(string(ft.Filepath))
	// the problem here is that closed dirs are not visited but we want to keep their settings:
	// ft.Dirs.DeleteStale()
	ft.Update()
	ft.TreeChanged(nil)
	ft.updateMu.Unlock()
	// ft.AsyncUnlock(update) // todo:
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
	if ft.watcher != nil {
		return nil
	}
	ft.watchedPaths = make(map[string]bool)
	var err error
	ft.watcher, err = fsnotify.NewWatcher()
	return err
}

// WatchWatcher monitors the watcher channel for update events.
// It must be called once some paths have been added to watcher --
// safe to call multiple times.
func (ft *Tree) WatchWatcher() {
	if ft.watcher == nil || ft.watcher.Events == nil {
		return
	}
	if ft.doneWatcher != nil {
		return
	}
	ft.doneWatcher = make(chan bool)
	go func() {
		watch := ft.watcher
		done := ft.doneWatcher
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
	ft.updateMu.Lock()
	defer ft.updateMu.Unlock()
	// fmt.Println(path)

	dir, _ := filepath.Split(path)
	rp := ft.RelPath(core.Filename(dir))
	if rp == ft.lastWatchUpdate {
		now := time.Now()
		lagMs := int(now.Sub(ft.lastWatchTime) / time.Millisecond)
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
	ft.lastWatchUpdate = rp
	ft.lastWatchTime = time.Now()
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
	on, has := ft.watchedPaths[rp]
	if on || has {
		return nil
	}
	ft.ConfigWatcher()
	// fmt.Printf("watching path: %s\n", path)
	err := ft.watcher.Add(string(path))
	if err == nil {
		ft.watchedPaths[rp] = true
		ft.WatchWatcher()
	} else {
		slog.Error(err.Error())
	}
	return err
}

// UnWatchPath removes given path from those watched
func (ft *Tree) UnWatchPath(path core.Filename) {
	rp := ft.RelPath(path)
	on, has := ft.watchedPaths[rp]
	if !on || !has {
		return
	}
	ft.ConfigWatcher()
	ft.watcher.Remove(string(path))
	ft.watchedPaths[rp] = false
}

// IsDirOpen returns true if given directory path is open (i.e., has been
// opened in the view)
func (ft *Tree) IsDirOpen(fpath core.Filename) bool {
	if fpath == ft.Filepath { // we are always open
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

// AddExternalFile adds an external file outside of root of file tree
// and triggers an update, returning the Node for it, or
// error if [filepath.Abs] fails.
func (ft *Tree) AddExternalFile(fpath string) (*Node, error) {
	pth, err := filepath.Abs(fpath)
	if err != nil {
		return nil, err
	}
	if _, err := os.Stat(pth); err != nil {
		return nil, err
	}
	if has, _ := ft.HasExternalFile(pth); has {
		return ft.ExternalNodeByPath(pth)
	}
	ft.ExternalFiles = append(ft.ExternalFiles, pth)
	ft.SyncDir()
	return ft.ExternalNodeByPath(pth)
}

// RemoveExternalFile removes external file from maintained list; returns true if removed.
func (ft *Tree) RemoveExternalFile(fpath string) bool {
	for i, ef := range ft.ExternalFiles {
		if ef == fpath {
			ft.ExternalFiles = append(ft.ExternalFiles[:i], ft.ExternalFiles[i+1:]...)
			return true
		}
	}
	return false
}

// HasExternalFile returns true and index if given abs path exists on ExtFiles list.
// false and -1 if not.
func (ft *Tree) HasExternalFile(fpath string) (bool, int) {
	for i, f := range ft.ExternalFiles {
		if f == fpath {
			return true, i
		}
	}
	return false, -1
}

// ExternalNodeByPath returns Node for given file path, and true, if it
// exists in the external files list.  Otherwise returns nil, false.
func (ft *Tree) ExternalNodeByPath(fpath string) (*Node, error) {
	ehas, i := ft.HasExternalFile(fpath)
	if !ehas {
		return nil, fmt.Errorf("ExtFile not found on list: %v", fpath)
	}
	ekid := ft.ChildByName(ExternalFilesName, 0)
	if ekid == nil {
		return nil, fmt.Errorf("ExtFile not updated -- no ExtFiles node")
	}
	if n := ekid.AsTree().Child(i); n != nil {
		return AsNode(n), nil
	}
	return nil, fmt.Errorf("ExtFile not updated; index invalid")
}

// SyncExternalFiles returns a type-and-name list for configuring nodes
// for ExtFiles
func (ft *Tree) SyncExternalFiles(efn *Node) {
	efn.Info.Mode = os.ModeDir | os.ModeIrregular // mark as dir, irregular
	plan := tree.TypePlan{}
	typ := ft.FileNodeType
	for _, f := range ft.ExternalFiles {
		plan.Add(typ, dirs.DirAndFile(f))
	}
	tree.Update(efn, plan) // NOT unique names
	// always go through kids, regardless of mods
	for i, sfk := range efn.Children {
		sf := AsNode(sfk)
		sf.FileRoot = ft
		fp := ft.ExternalFiles[i]
		sf.SetNodePath(fp)
		sf.Info.VCS = vcs.Stored // no vcs in general
	}
}
