// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package filetree

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sync/atomic"

	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/base/vcs"
	"cogentcore.org/core/core"
	"cogentcore.org/core/tree"
	"cogentcore.org/core/types"
)

const (
	// externalFilesName is the name of the node that represents external files
	externalFilesName = "[external files]"
)

// Treer is an interface for getting the Root node if it implements [Treer].
type Treer interface {
	AsFileTree() *Tree
}

// AsTree returns the given value as a [Tree] if it has
// an AsFileTree() method, or nil otherwise.
func AsTree(n tree.Node) *Tree {
	if t, ok := n.(Treer); ok {
		return t.AsFileTree()
	}
	return nil
}

// Tree is the root widget of a file tree representing files in a given directory
// (and subdirectories thereof), and has some overall management state for how to
// view things.
type Tree struct {
	Node

	// externalFiles are external files outside the root path of the tree.
	// They are stored in terms of their absolute paths. These are shown
	// in the first sub-node if present; use [Tree.AddExternalFile] to add one.
	externalFiles []string

	// Dirs records state of directories within the tree (encoded using paths relative to root),
	// e.g., open (have been opened by the user) -- can persist this to restore prior view of a tree
	Dirs DirFlagMap `set:"-"`

	// DirsOnTop indicates whether all directories are placed at the top of the tree.
	// Otherwise everything is mixed.  This is the default.
	DirsOnTop bool

	// SortByModTime causes files to be sorted by modification time by default.
	// Otherwise it is a per-directory option.
	SortByModTime bool

	// FileNodeType is the type of node to create; defaults to [Node] but can use custom node types
	FileNodeType *types.Type `display:"-" json:"-" xml:"-"`

	// FilterFunc, if set, determines whether to include the given node in the tree.
	// return true to include, false to not.  This applies to files and directories alike.
	FilterFunc func(path string, info fs.FileInfo) bool

	// FS is the file system we are browsing, if it is an FS (nil = os filesystem)
	FS fs.FS

	// inOpenAll indicates whether we are in midst of an OpenAll call; nodes should open all dirs.
	inOpenAll bool

	// NoWatch disables the file system watcher, which otherwise updates
	// the tree when files change on disk. It must be set before [Tree.OpenPath].
	NoWatch bool

	// WatchFunc, if set, is called with each batch of changes detected by
	// the file system watcher, after the tree has been updated for them.
	// It is called under [core.WidgetBase.AsyncLock], so it can update the GUI
	// directly. Only the files directly within the root and the open
	// directories of the tree are watched, excluding hidden directories.
	WatchFunc func(changes []FileChange) `display:"-" json:"-" xml:"-"`

	// watch is the active file system watcher, if any.
	watch atomic.Pointer[treeWatcher]
}

func (ft *Tree) Init() {
	ft.Node.Init()
	ft.Root = ft
	ft.FileNodeType = types.For[Node]()
	ft.OpenDepth = 4
	ft.DirsOnTop = true
	ft.FirstMaker(func(p *tree.Plan) {
		if len(ft.externalFiles) == 0 {
			return
		}
		tree.AddNew(p, externalFilesName, func() Filer {
			return tree.NewOfType(ft.FileNodeType).(Filer)
		}, func(wf Filer) {
			w := wf.AsFileNode()
			w.Root = ft.Root
			w.Filepath = externalFilesName
			w.Info.Mode = os.ModeDir
			w.Info.VCS = vcs.Stored
		})
	})
}

func (fv *Tree) Destroy() {
	fv.stopWatcher()
	fv.Tree.Destroy()
}

func (ft *Tree) AsFileTree() *Tree {
	return ft
}

// OpenPath opens the filetree at the given os file system directory path.
// It reads all the files at the given path into this tree.
// Only paths listed in [Tree.Dirs] will be opened.
func (ft *Tree) OpenPath(path string) *Tree {
	if ft.FileNodeType == nil {
		ft.FileNodeType = types.For[Node]()
	}
	effpath, err := filepath.EvalSymlinks(path)
	if err != nil {
		effpath = path
	}
	abs, err := filepath.Abs(effpath)
	if errors.Log(err) != nil {
		abs = effpath
	}
	ft.FS = nil
	ft.Filepath = core.Filename(abs)
	ft.setDirOpen(core.Filename(abs))
	ft.detectVCSRepo()
	ft.This.(Filer).GetFileInfo()
	ft.Open()
	ft.Update()
	ft.startWatcher()
	return ft
}

// OpenPathFS opens the filetree at the given [fs] file system directory path.
// It reads all the files at the given path into this tree.
// Only paths listed in [Tree.Dirs] will be opened.
func (ft *Tree) OpenPathFS(fsys fs.FS, path string) *Tree {
	if ft.FileNodeType == nil {
		ft.FileNodeType = types.For[Node]()
	}
	ft.stopWatcher()
	ft.FS = fsys
	ft.Filepath = core.Filename(path)
	ft.setDirOpen(core.Filename(path))
	ft.This.(Filer).GetFileInfo()
	ft.Open()
	ft.Update()
	return ft
}

// UpdatePath updates the tree at the directory level for given path
// and everything below it. It flags that it needs render update,
// but if a deletion or insertion happened, then NeedsLayout should also
// be called.
func (ft *Tree) UpdatePath(path string) {
	ft.NeedsRender()
	path = filepath.Clean(path)
	ft.dirsTo(path)
	if fn, ok := ft.FindFile(path); ok {
		if fn.IsDir() {
			fn.Update()
			return
		}
	}
	fpath, _ := filepath.Split(path)
	if fn, ok := ft.FindFile(fpath); ok {
		fn.Update()
		return
	}
	// core.MessageSnackbar(ft, "UpdatePath: path not found in tree: "+path)
}

// OpenPaths returns a list of open paths.
func (ft *Tree) OpenPaths() []string {
	return ft.Dirs.openPaths(string(ft.Filepath))
}

// isDirOpen returns true if given directory path is open (i.e., has been
// opened in the view)
func (ft *Tree) isDirOpen(fpath core.Filename) bool {
	if fpath == ft.Filepath { // we are always open
		return true
	}
	return ft.Dirs.isOpen(ft.RelativePathFrom(fpath))
}

// setDirOpen sets the given directory path to be open
func (ft *Tree) setDirOpen(fpath core.Filename) {
	rp := ft.RelativePathFrom(fpath)
	// fmt.Printf("setdiropen: %s\n", rp)
	ft.Dirs.setOpen(rp, true)
}

// setDirClosed sets the given directory path to be closed
func (ft *Tree) setDirClosed(fpath core.Filename) {
	rp := ft.RelativePathFrom(fpath)
	ft.Dirs.setOpen(rp, false)
}

// setDirSortBy sets the given directory path sort by option
func (ft *Tree) setDirSortBy(fpath core.Filename, modTime bool) {
	ft.Dirs.setSortBy(ft.RelativePathFrom(fpath), modTime)
}

// dirSortByModTime returns true if dir is sorted by mod time
func (ft *Tree) dirSortByModTime(fpath core.Filename) bool {
	return ft.Dirs.sortByModTime(ft.RelativePathFrom(fpath))
}

// dirSortByName returns true if dir is sorted by name
func (ft *Tree) dirSortByName(fpath core.Filename) bool {
	return ft.Dirs.sortByName(ft.RelativePathFrom(fpath))
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
	if has, _ := ft.hasExternalFile(pth); has {
		return ft.externalNodeByPath(pth)
	}
	newExt := len(ft.externalFiles) == 0
	ft.externalFiles = append(ft.externalFiles, pth)
	if newExt {
		ft.Update()
	} else {
		ft.Child(0).(Filer).AsFileNode().Update()
	}
	return ft.externalNodeByPath(pth)
}

// removeExternalFile removes external file from maintained list; returns true if removed.
func (ft *Tree) removeExternalFile(fpath string) bool {
	for i, ef := range ft.externalFiles {
		if ef == fpath {
			ft.externalFiles = append(ft.externalFiles[:i], ft.externalFiles[i+1:]...)
			return true
		}
	}
	return false
}

// hasExternalFile returns true and index if given abs path exists on ExtFiles list.
// false and -1 if not.
func (ft *Tree) hasExternalFile(fpath string) (bool, int) {
	for i, f := range ft.externalFiles {
		if f == fpath {
			return true, i
		}
	}
	return false, -1
}

// externalNodeByPath returns Node for given file path, and true, if it
// exists in the external files list.  Otherwise returns nil, false.
func (ft *Tree) externalNodeByPath(fpath string) (*Node, error) {
	ehas, i := ft.hasExternalFile(fpath)
	if !ehas {
		return nil, fmt.Errorf("ExtFile not found on list: %v", fpath)
	}
	ekid := ft.ChildByName(externalFilesName, 0)
	if ekid == nil {
		return nil, errors.New("ExtFile not updated -- no ExtFiles node")
	}
	if n := ekid.AsTree().Child(i); n != nil {
		return AsNode(n), nil
	}
	return nil, errors.New("ExtFile not updated; index invalid")
}
