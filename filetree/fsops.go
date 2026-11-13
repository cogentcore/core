// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package filetree

import (
	"errors"
	"fmt"
	"io/fs"
	"path"
	"path/filepath"
	"strings"

	"cogentcore.org/core/base/dirsort"
	"cogentcore.org/core/base/fsx"
)

// WriteFS is an [fs.FS] that a file tree can change, such as an in-memory
// filesystem, with the file operations of its context menu, drag and drop,
// and copy and paste (see [Tree.FS]). Its paths are those of an fs.FS,
// separated by /. A tree of an fs.FS that is not a WriteFS is read-only.
type WriteFS interface {
	fs.FS

	// MkdirAll makes the directory at the given path, with any of its
	// parent directories that do not exist yet.
	MkdirAll(path string) error

	// Rename renames (moves) the file or directory at the old path to the
	// new path, which does not exist.
	Rename(oldpath, newpath string) error

	// RemoveAll removes the file or directory at the given path, with all
	// that it contains.
	RemoveAll(path string) error

	// CopyPath copies the file or directory at the source path to the
	// destination path, replacing anything there.
	CopyPath(dst, src string) error
}

// errReadOnly is the error of a change to a read-only [Tree.FS].
var errReadOnly = errors.New("the files cannot be changed: their filesystem is read-only")

// writeFS returns the [WriteFS] of the tree of the node, or nil if it is
// that of the operating system, or the error of a read-only [Tree.FS].
func (fn *Node) writeFS() (WriteFS, error) {
	fsys := fn.FileRoot().FS
	if fsys == nil {
		return nil, nil
	}
	if wfs, ok := fsys.(WriteFS); ok {
		return wfs, nil
	}
	return nil, errReadOnly
}

// canWrite returns whether the files of the tree of the node can be
// changed: those of the operating system, and of a [WriteFS].
func (fn *Node) canWrite() bool {
	_, err := fn.writeFS()
	return err == nil
}

// isFS returns whether the tree of the node is of a [Tree.FS].
func (fn *Node) isFS() bool {
	return fn.FileRoot().FS != nil
}

// cleanPath, dirOf, and baseOf are [filepath.Clean], [filepath.Dir], and
// [filepath.Base] for the paths of the tree of the node, which use / in a
// [Tree.FS] (see [Node.joinPath]).
func (fn *Node) cleanPath(p string) string {
	if fn.isFS() {
		return path.Clean(p)
	}
	return filepath.Clean(p)
}

func (fn *Node) dirOf(p string) string {
	if fn.isFS() {
		return path.Dir(p)
	}
	return filepath.Dir(p)
}

func (fn *Node) baseOf(p string) string {
	if fn.isFS() {
		return path.Base(p)
	}
	return filepath.Base(p)
}

// pathExists returns whether a file or directory exists at the given path
// of the tree of the node.
func (fn *Node) pathExists(p string) bool {
	if fsys := fn.FileRoot().FS; fsys != nil {
		_, err := fs.Stat(fsys, p)
		return err == nil
	}
	return fsx.Exists(p)
}

// isDirPath returns whether there is a directory at the given path of the
// tree of the node.
func (fn *Node) isDirPath(p string) bool {
	if fsys := fn.FileRoot().FS; fsys != nil {
		info, err := fs.Stat(fsys, p)
		return err == nil && info.IsDir()
	}
	ok, _ := fsx.DirExists(p)
	return ok
}

// copyPath copies the file or directory at the source path to the
// destination path, in the tree of the node.
func (fn *Node) copyPath(dst, src string) error {
	wfs, err := fn.writeFS()
	switch {
	case err != nil:
		return err
	case wfs != nil:
		return wfs.CopyPath(dst, src)
	}
	return fsx.CopyPath(dst, src)
}

// duplicatePath copies the file or directory at the given path to a new
// one in the same directory, named with _Copy, as [fileinfo.FileInfo.Duplicate],
// returning its path.
func (fn *Node) duplicatePath(p string) (string, error) {
	name := fn.baseOf(p)
	ext := path.Ext(name)
	if fn.isDirPath(p) || ext == name {
		ext = ""
	}
	noext := strings.TrimSuffix(p, ext)
	dst := noext + "_Copy" + ext
	for i := 1; fn.pathExists(dst); i++ {
		dst = noext + fmt.Sprintf("_Copy%d", i) + ext
	}
	return dst, fn.copyPath(dst, p)
}

// fsSort returns the sort order of the directory at the given path of a
// [Tree.FS]: the one set for it with Sort by, or else the [Tree.FSSort].
func (ft *Tree) fsSort(dir string) dirsort.Sort {
	if s, ok := ft.fsSorts[dir]; ok {
		return s
	}
	return ft.FSSort
}

// dirSort returns the sort order of the files of this directory: that of
// [dirsort.For], or of [Tree.fsSort] in a [Tree.FS].
func (fn *Node) dirSort() dirsort.Sort {
	if fn.isFS() {
		return fn.FileRoot().fsSort(string(fn.Filepath))
	}
	return dirsort.For(string(fn.Filepath))
}

// fsRenamePath returns the new path of this node of a [Tree.FS] for
// renaming it to the given name, or path within the tree, or "" if it is
// the same, with an error if something else is there.
func (fn *Node) fsRenamePath(name string) (string, error) {
	if name == "" {
		return "", errors.New("the new name is empty")
	}
	newpath := name
	if !strings.Contains(name, "/") {
		newpath = path.Join(path.Dir(string(fn.Filepath)), name)
	}
	newpath = path.Clean(newpath)
	if newpath == path.Clean(string(fn.Filepath)) {
		return "", nil
	}
	if fn.pathExists(newpath) {
		return "", fmt.Errorf("cannot rename %s: %s already exists", fn.Name, newpath)
	}
	return newpath, nil
}
