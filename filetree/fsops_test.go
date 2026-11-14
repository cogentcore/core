// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package filetree

import (
	"image"
	"io/fs"
	"maps"
	"path"
	"slices"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"cogentcore.org/core/base/dirsort"
	"cogentcore.org/core/base/fileinfo/mimedata"
	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"github.com/hack-pad/hackpadfs"
	"github.com/hack-pad/hackpadfs/mem"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// memFS is an in-memory FS for the tests, with the hackpadfs interfaces
// of the operations, other than writing files, and [CopyFS], whose
// copies it counts.
type memFS struct {
	fstest.MapFS
	copies int
}

func (m *memFS) MkdirAll(p string, perm fs.FileMode) error {
	for d := p; d != "." && d != "/"; d = path.Dir(d) {
		if _, ok := m.MapFS[d]; !ok {
			m.MapFS[d] = &fstest.MapFile{Mode: fs.ModeDir | perm, ModTime: time.Now()}
		}
	}
	return nil
}

// within returns the paths at and within the given one.
func (m *memFS) within(p string) []string {
	var ps []string
	for k := range m.MapFS {
		if k == p || strings.HasPrefix(k, p+"/") {
			ps = append(ps, k)
		}
	}
	return ps
}

func (m *memFS) Rename(oldpath, newpath string) error {
	if _, err := fs.Stat(m, oldpath); err != nil {
		return err
	}
	for _, k := range m.within(oldpath) {
		m.MapFS[newpath+k[len(oldpath):]] = m.MapFS[k]
		delete(m.MapFS, k)
	}
	return nil
}

func (m *memFS) RemoveAll(p string) error {
	for _, k := range m.within(p) {
		delete(m.MapFS, k)
	}
	return nil
}

func (m *memFS) CopyPath(dst, src string) error {
	if _, err := fs.Stat(m, src); err != nil {
		return err
	}
	m.copies++
	m.RemoveAll(dst)
	for _, k := range m.within(src) {
		f := *m.MapFS[k]
		f.Data = slices.Clone(f.Data)
		m.MapFS[dst+k[len(src):]] = &f
	}
	return nil
}

// testFSKinds are the kinds of FS of the tests: the in-memory one of
// hackpadfs (with Mkdir, Remove, OpenFile, and others), a [memFS], and a
// read-only one.
type testFSKind int

const (
	hackpadMem testFSKind = iota
	copyFS
	readOnly
)

// newFSTree returns a tree of a new FS of the given kind, with a.txt,
// sub/b.txt, and the directory other, in a window.
func newFSTree(t *testing.T, kind testFSKind) (*Tree, fs.FS) {
	files := fstest.MapFS{
		"a.txt":     {Data: []byte("a\n"), Mode: 0o644, ModTime: time.Now()},
		"sub":       {Mode: fs.ModeDir | 0o755},
		"sub/b.txt": {Data: []byte("b\n"), Mode: 0o644},
		"other":     {Mode: fs.ModeDir | 0o755},
	}
	var fsys fs.FS = files
	switch kind {
	case hackpadMem:
		mfs, err := mem.NewFS()
		require.NoError(t, err)
		for _, p := range slices.Sorted(maps.Keys(files)) {
			if files[p].Mode.IsDir() {
				require.NoError(t, hackpadfs.MkdirAll(mfs, p, 0o755))
			} else {
				require.NoError(t, hackpadfs.WriteFullFile(mfs, p, files[p].Data, 0o644))
			}
		}
		fsys = mfs
	case copyFS:
		fsys = &memFS{MapFS: files}
	}
	b := core.NewBody()
	ft := NewTree(b)
	ft.Dirs.setOpen("sub", true)
	ft.Dirs.setOpen("other", true)
	ft.OpenPathFS(fsys, ".")
	runTestWindow(t, b)
	return ft, fsys
}

// fsNode returns the node of the given path of the tree, or nil.
func fsNode(ft *Tree, p string) *Node {
	ft.AsyncLock()
	defer ft.AsyncUnlock()
	return fsNode2(ft, p)
}

// fsNode2 is [fsNode] under the lock.
func fsNode2(ft *Tree, p string) *Node {
	n, ok := ft.FindFile(p)
	if !ok || string(n.Filepath) != p {
		return nil
	}
	return n
}

// TestWriteFS tests the file operations of a tree of an FS that can be
// changed: making a folder and a file, renaming, duplicating, and
// deleting, moving with drag and drop, and copying with copy and paste,
// with the hackpadfs interfaces (and the fallbacks of hackpadfs, as for
// the in-memory one of hackpadfs, without MkdirAll and RemoveAll), and
// with [CopyFS] or by reading and writing the files.
func TestWriteFS(t *testing.T) {
	for _, kind := range []testFSKind{hackpadMem, copyFS} {
		testWriteFS(t, kind)
	}
}

func testWriteFS(t *testing.T, kind testFSKind) {
	ft, fsys := newFSTree(t, kind)
	do := func(f func()) {
		ft.AsyncLock()
		defer ft.AsyncUnlock()
		f()
	}
	read := func(p string) string {
		b, err := fs.ReadFile(fsys, p)
		if err != nil {
			return "(none)"
		}
		return string(b)
	}
	exists := func(p string) bool {
		_, err := fs.Stat(fsys, p)
		return err == nil
	}
	waitFor := func(what string, cond func() bool) {
		t.Helper()
		deadline := time.Now().Add(5 * time.Second)
		for !cond() {
			if time.Now().After(deadline) {
				t.Fatalf("%d: timeout waiting for %s", kind, what)
			}
			time.Sleep(10 * time.Millisecond)
		}
	}
	has := func(p string) func() bool { return func() bool { return fsNode(ft, p) != nil } }

	// a new folder, and a new file in the hackpadfs one (with OpenFile)
	do(func() { AsNode(ft.This).newFolder("new") })
	assert.True(t, exists("new"))
	waitFor("the new folder", has("new"))
	if kind == hackpadMem {
		do(func() { AsNode(ft.This).newFile("n.txt", false) })
		assert.Equal(t, "", read("n.txt"))
		waitFor("the new file", has("n.txt"))
		do(func() { require.NoError(t, fsNode2(ft, "n.txt").DeleteFile()) })
	}

	// rename a file and a directory, with its files
	do(func() { require.NoError(t, fsNode2(ft, "a.txt").RenameFile("c.txt")) })
	assert.Equal(t, "a\n", read("c.txt"))
	assert.False(t, exists("a.txt"))
	waitFor("the renamed file", has("c.txt"))
	do(func() { require.NoError(t, fsNode2(ft, "sub").RenameFile("sub2")) })
	assert.Equal(t, "b\n", read("sub2/b.txt"))
	waitFor("the renamed directory", has("sub2/b.txt"))
	do(func() {
		assert.Error(t, fsNode2(ft, "c.txt").RenameFile("sub2"), "exists")
	})

	// duplicate, and delete
	do(func() { require.NoError(t, fsNode2(ft, "c.txt").duplicateFile()) })
	assert.Equal(t, "a\n", read("c_Copy.txt"))
	waitFor("the copy", has("c_Copy.txt"))
	do(func() { require.NoError(t, fsNode2(ft, "c_Copy.txt").DeleteFile()) })
	assert.False(t, exists("c_Copy.txt"))
	waitFor("the deletion", func() bool { return fsNode(ft, "c_Copy.txt") == nil })

	// move a directory with drag and drop
	var md mimedata.Mimes
	do(func() {
		src := fsNode2(ft, "sub2")
		src.MimeData(&md)
		de := &events.DragDrop{DropMod: events.DropMove, Data: md, Source: src.This}
		de.Typ = events.Drop
		fsNode2(ft, "other").DragDrop(de)
	})
	waitFor("the moved directory", has("other/sub2"))
	assert.Equal(t, "b\n", read("other/sub2/b.txt"))
	waitFor("the source deleted", func() bool { return !exists("sub2") && fsNode(ft, "sub2") == nil })

	// copy and paste a file into a directory
	do(func() {
		var md mimedata.Mimes // (as copied: the clipboard of the tests is empty)
		fsNode2(ft, "c.txt").MimeData(&md)
		fsNode2(ft, "new").pasteMimeData(md)
		// (paths of the os are not pasted)
		fsNode2(ft, "new").pasteMimeData(mimedata.NewText(t.TempDir()))
	})
	assert.Equal(t, "a\n", read("new/c.txt"), "pasted")
	ents, _ := fs.ReadDir(fsys, "new")
	assert.Len(t, ents, 1, "only that")
	assert.Equal(t, "a\n", read("c.txt"), "copied")
	if m, ok := fsys.(*memFS); ok {
		assert.Equal(t, 3, m.copies, "with CopyPath")
	}
}

// TestReadOnlyFS tests that the files of a tree of an [fs.FS] without the
// interfaces of the operations are not changed, and that a drop from the
// files of the os into the tree of an FS is not done.
func TestReadOnlyFS(t *testing.T) {
	ft, fsys := newFSTree(t, readOnly)
	m := fsys.(fstest.MapFS)
	ft.AsyncLock()
	defer ft.AsyncUnlock()
	n := fsNode2(ft, "a.txt")
	require.NotNil(t, n)
	assert.False(t, n.canRename() || n.canRemove() || n.canMkdir() || n.canCreate() || n.canCopy())
	assert.ErrorIs(t, n.RenameFile("c.txt"), errUnsupported)
	assert.ErrorIs(t, n.DeleteFile(), errUnsupported)
	assert.ErrorIs(t, n.copyPath("d.txt", "a.txt"), errUnsupported)
	AsNode(ft.This).newFolder("new")
	assert.Nil(t, m["new"])
	assert.NotNil(t, m["a.txt"])

	de := &events.DragDrop{DropMod: events.DropCopy, Data: mimedata.Mimes{mimedata.NewTextData(t.TempDir())}}
	de.Typ = events.Drop
	fsNode2(ft, "sub").DragDrop(de)
	assert.Len(t, m, 4, "nothing dropped")
}

// TestFSSort tests the sort order of the directories of a [Tree.FS]: the
// [Tree.FSSort], and that set for a directory with Sort by.
func TestFSSort(t *testing.T) {
	ft, fsys := newFSTree(t, copyFS)
	m := fsys.(*memFS)
	ft.AsyncLock()
	defer ft.AsyncUnlock()
	m.MapFS["sub/a.txt"] = &fstest.MapFile{Data: []byte("aaaa\n"), ModTime: time.Now().Add(time.Hour)}
	m.MapFS["sub/b.txt"].ModTime = time.Now()
	names := func(dir string) []string {
		var ns []string
		for _, kid := range fsNode2(ft, dir).Children {
			ns = append(ns, kid.AsTree().Name)
		}
		return ns
	}
	sub := fsNode2(ft, "sub")
	sub.Update()
	assert.Equal(t, []string{"a.txt", "b.txt"}, names("sub"), "by name")
	ft.FSSort = dirsort.Sort{By: dirsort.ModTime, Down: true}
	sub.Update()
	assert.Equal(t, []string{"a.txt", "b.txt"}, names("sub"), "newest first")
	sub.sortBy(dirsort.Sort{By: dirsort.ModTime})
	assert.Equal(t, []string{"b.txt", "a.txt"}, names("sub"), "set for it")
	assert.Equal(t, dirsort.Sort{By: dirsort.ModTime}, sub.dirSort())
	_, saved := dirsort.Saved("sub")
	assert.False(t, saved, "not saved for the directory of the os")
}

// TestFSSubdirRoot tests finding the files of a tree of an FS opened at
// one of its directories, whose paths start with it, rather than ".".
func TestFSSubdirRoot(t *testing.T) {
	m := &memFS{MapFS: fstest.MapFS{
		"top/sub/b.txt": {Data: []byte("b\n")},
	}}
	b := core.NewBody()
	ft := NewTree(b)
	ft.OpenPathFS(m, "top")
	runTestWindow(t, b)
	ft.AsyncLock()
	defer ft.AsyncUnlock()
	n := fsNode2(ft, "top/sub/b.txt")
	require.NotNil(t, n, "found, opening its directory")
	require.NoError(t, n.RenameFile("c.txt"))
	assert.NotNil(t, m.MapFS["top/sub/c.txt"])
	assert.NotNil(t, fsNode2(ft, "top/sub/c.txt"))
}

// TestFSMenu tests that the items of the context menu that change files
// are enabled for the operations that the FS has, and disabled for a
// read-only FS.
func TestFSMenu(t *testing.T) {
	for _, kind := range []testFSKind{hackpadMem, copyFS, readOnly} {
		ft, _ := newFSTree(t, kind)
		ft.AsyncLock()
		n := fsNode2(ft, "sub")
		n.Select()
		m := core.NewScene()
		n.contextMenu(m, image.Point{})
		enabled := map[string]bool{}
		for _, kid := range m.Children {
			if fb, ok := kid.(*core.FuncButton); ok {
				enabled[fb.Text] = !fb.IsDisabled()
			}
		}
		ft.AsyncUnlock()
		for _, item := range []string{"Duplicate", "Delete", "Rename", "New folder", "Paste"} {
			assert.Equal(t, kind != readOnly, enabled[item], "%s, kind: %d", item, kind)
		}
		assert.Equal(t, kind == hackpadMem, enabled["New file"], "with OpenFile, kind: %d", kind)
		_, ok := enabled["Remove from externs"]
		assert.False(t, ok, "not in an FS")
	}
}
