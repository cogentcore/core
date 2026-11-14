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
	"github.com/hack-pad/hackpadfs"
)

// The files of a [Tree.FS] are changed with the interfaces of hackpadfs
// for each operation, which extend those of [io/fs], as the os functions:
// [hackpadfs.MkdirAllFS] (or MkdirFS) for New folder, [hackpadfs.RenameFS]
// for Rename, [hackpadfs.RemoveAllFS] (or RemoveFS) for Delete,
// [hackpadfs.WriteFileFS] (or OpenFileFS) for New file, and [CopyFS], or
// else those of making directories and writing files, for copying, as in
// Duplicate, drag and drop, and copy and paste. Those that an FS does not
// have are disabled, as for a read-only FS. A [hackpadfs.MountFS] can
// have any of them, in its mounted filesystems.

// CopyFS is an [fs.FS] that copies its own files and directories, as for
// those that are more than their bytes, such as typed data. Those of
// another FS are copied by reading and writing them (see [Tree.FS]).
type CopyFS interface {
	fs.FS

	// CopyPath copies the file or directory at the source path to the
	// destination path, replacing anything there.
	CopyPath(dst, src string) error
}

// errUnsupported is the error of a change to the files of a [Tree.FS] that
// it does not support, as for a read-only one.
var errUnsupported = errors.New("the filesystem of the files does not support this change")

// fsErr returns the given error of an operation of hackpadfs, with
// [errUnsupported] for an operation that the FS does not have.
func fsErr(err error) error {
	if errors.Is(err, hackpadfs.ErrNotImplemented) {
		return fmt.Errorf("%w: %w", errUnsupported, err)
	}
	return err
}

// fsHas returns whether the [Tree.FS] of the node has any of the
// interfaces of the given values, or is a [hackpadfs.MountFS], which can
// have them in its mounted filesystems. It is true for the os.
func fsHas(fn *Node, has ...func(fs.FS) bool) bool {
	fsys := fn.FileRoot().FS
	if fsys == nil {
		return true
	}
	if _, ok := fsys.(hackpadfs.MountFS); ok {
		return true
	}
	for _, h := range has {
		if h(fsys) {
			return true
		}
	}
	return false
}

// is returns whether the given FS has the interface T.
func is[T any](fsys fs.FS) bool {
	_, ok := fsys.(T)
	return ok
}

// canMkdir, canRename, canRemove, canCreate, and canCopy return whether
// the files of the tree of the node can be changed with the operations of
// those names (see [CopyFS]), as those of the os can.
func (fn *Node) canMkdir() bool {
	return fsHas(fn, is[hackpadfs.MkdirAllFS], is[hackpadfs.MkdirFS])
}

func (fn *Node) canRename() bool {
	return fsHas(fn, is[hackpadfs.RenameFS])
}

func (fn *Node) canRemove() bool {
	return fsHas(fn, is[hackpadfs.RemoveAllFS], is[hackpadfs.RemoveFS])
}

func (fn *Node) canCreate() bool {
	return fsHas(fn, is[hackpadfs.WriteFileFS], is[hackpadfs.OpenFileFS])
}

func (fn *Node) canCopy() bool {
	return fsHas(fn, is[CopyFS]) || (fn.canMkdir() && fn.canCreate())
}

// mkdirAll, rename, and removeAll do those operations of hackpadfs on the
// [Tree.FS] of the node.
func (fn *Node) mkdirAll(p string) error {
	return fsErr(hackpadfs.MkdirAll(fn.FileRoot().FS, p, 0o775))
}

func (fn *Node) rename(oldpath, newpath string) error {
	return fsErr(hackpadfs.Rename(fn.FileRoot().FS, oldpath, newpath))
}

func (fn *Node) removeAll(p string) error {
	return fsErr(hackpadfs.RemoveAll(fn.FileRoot().FS, p))
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
// destination path, in the tree of the node: with [CopyFS] in a
// [Tree.FS] that has it, and otherwise by reading and writing the files.
func (fn *Node) copyPath(dst, src string) error {
	fsys := fn.FileRoot().FS
	if fsys == nil {
		return fsx.CopyPath(dst, src)
	}
	if cfs, ok := fsys.(CopyFS); ok {
		return cfs.CopyPath(dst, src)
	}
	if !fn.canCopy() {
		return errUnsupported
	}
	return fs.WalkDir(fsys, src, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		to := dst + p[len(src):]
		info, err := d.Info()
		if err != nil {
			return err
		}
		if d.IsDir() {
			return fsErr(hackpadfs.MkdirAll(fsys, to, info.Mode().Perm()|0o700))
		}
		b, err := fs.ReadFile(fsys, p)
		if err != nil {
			return err
		}
		return fsErr(hackpadfs.WriteFullFile(fsys, to, b, info.Mode().Perm()))
	})
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
