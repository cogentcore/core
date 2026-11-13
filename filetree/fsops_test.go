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
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// memFS is a [WriteFS] in memory, for the tests.
type memFS struct {
	fstest.MapFS
}

func (m *memFS) MkdirAll(p string) error {
	for d := p; d != "." && d != "/"; d = path.Dir(d) {
		if _, ok := m.MapFS[d]; !ok {
			m.MapFS[d] = &fstest.MapFile{Mode: fs.ModeDir | 0o755, ModTime: time.Now()}
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
	m.RemoveAll(dst)
	for _, k := range m.within(src) {
		f := *m.MapFS[k]
		f.Data = slices.Clone(f.Data)
		m.MapFS[dst+k[len(src):]] = &f
	}
	return nil
}

// newFSTree returns a tree of a new [memFS], or of a read-only one, with
// a.txt, sub/b.txt, and the directory other, in a window.
func newFSTree(t *testing.T, writable bool) (*Tree, *memFS) {
	m := &memFS{fstest.MapFS{
		"a.txt":     {Data: []byte("a\n"), ModTime: time.Now()},
		"sub":       {Mode: fs.ModeDir | 0o755},
		"sub/b.txt": {Data: []byte("b\n")},
		"other":     {Mode: fs.ModeDir | 0o755},
	}}
	b := core.NewBody()
	ft := NewTree(b)
	ft.Dirs.setOpen("sub", true)
	ft.Dirs.setOpen("other", true)
	if writable {
		ft.OpenPathFS(m, ".")
	} else {
		ft.OpenPathFS(m.MapFS, ".")
	}
	runTestWindow(t, b)
	return ft, m
}

// fsNode returns the node of the given path of the tree, or nil.
func fsNode(ft *Tree, p string) *Node {
	ft.AsyncLock()
	defer ft.AsyncUnlock()
	n, ok := ft.FindFile(p)
	if !ok || string(n.Filepath) != p {
		return nil
	}
	return n
}

// TestWriteFS tests the file operations of a tree of a [WriteFS]: making
// a folder, renaming, duplicating, and deleting, moving with drag and
// drop, and copying with copy and paste.
func TestWriteFS(t *testing.T) {
	ft, m := newFSTree(t, true)
	do := func(f func()) {
		ft.AsyncLock()
		defer ft.AsyncUnlock()
		f()
	}
	waitFor := func(what string, cond func() bool) {
		t.Helper()
		deadline := time.Now().Add(5 * time.Second)
		for !cond() {
			if time.Now().After(deadline) {
				t.Fatalf("timeout waiting for %s; files: %v", what, slices.Sorted(maps.Keys(m.MapFS)))
			}
			time.Sleep(10 * time.Millisecond)
		}
	}
	has := func(p string) func() bool { return func() bool { return fsNode(ft, p) != nil } }

	// a new folder
	do(func() { AsNode(ft.This).newFolder("new") })
	assert.True(t, m.MapFS["new"].Mode.IsDir())
	waitFor("the new folder", has("new"))

	// rename a file and a directory, with its files
	do(func() { require.NoError(t, fsNode2(ft, "a.txt").RenameFile("c.txt")) })
	assert.Equal(t, "a\n", string(m.MapFS["c.txt"].Data))
	assert.Nil(t, m.MapFS["a.txt"])
	waitFor("the renamed file", has("c.txt"))
	do(func() { require.NoError(t, fsNode2(ft, "sub").RenameFile("sub2")) })
	assert.Equal(t, "b\n", string(m.MapFS["sub2/b.txt"].Data))
	waitFor("the renamed directory", has("sub2/b.txt"))
	do(func() {
		assert.Error(t, fsNode2(ft, "c.txt").RenameFile("sub2"), "exists")
	})

	// duplicate, and delete
	do(func() { require.NoError(t, fsNode2(ft, "c.txt").duplicateFile()) })
	assert.Equal(t, "a\n", string(m.MapFS["c_Copy.txt"].Data))
	waitFor("the copy", has("c_Copy.txt"))
	do(func() { require.NoError(t, fsNode2(ft, "c_Copy.txt").DeleteFile()) })
	assert.Nil(t, m.MapFS["c_Copy.txt"])
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
	assert.Equal(t, "b\n", string(m.MapFS["other/sub2/b.txt"].Data))
	waitFor("the source deleted", func() bool { return m.MapFS["sub2"] == nil && fsNode(ft, "sub2") == nil })

	// copy and paste a file into a directory
	do(func() {
		var md mimedata.Mimes // (as copied: the clipboard of the tests is empty)
		fsNode2(ft, "c.txt").MimeData(&md)
		fsNode2(ft, "new").pasteMimeData(md)
		// (paths of the os are not pasted)
		fsNode2(ft, "new").pasteMimeData(mimedata.NewText(t.TempDir()))
	})
	require.NotNil(t, m.MapFS["new/c.txt"], "pasted")
	assert.Equal(t, "a\n", string(m.MapFS["new/c.txt"].Data))
	assert.Len(t, m.within("new"), 2, "only that")
	assert.Equal(t, "a\n", string(m.MapFS["c.txt"].Data), "copied")
}

// fsNode2 is [fsNode] under the lock.
func fsNode2(ft *Tree, p string) *Node {
	n, ok := ft.FindFile(p)
	if !ok || string(n.Filepath) != p {
		return nil
	}
	return n
}

// TestReadOnlyFS tests that the files of a tree of an [fs.FS] that is
// not a [WriteFS] are not changed, and that a drop from the files of the
// os into the tree of an FS is not done.
func TestReadOnlyFS(t *testing.T) {
	ft, m := newFSTree(t, false)
	ft.AsyncLock()
	defer ft.AsyncUnlock()
	n := fsNode2(ft, "a.txt")
	require.NotNil(t, n)
	assert.False(t, n.canWrite())
	assert.ErrorIs(t, n.RenameFile("c.txt"), errReadOnly)
	assert.ErrorIs(t, n.DeleteFile(), errReadOnly)
	AsNode(ft.This).newFolder("new")
	assert.Nil(t, m.MapFS["new"])
	assert.NotNil(t, m.MapFS["a.txt"])

	de := &events.DragDrop{DropMod: events.DropCopy, Data: mimedata.Mimes{mimedata.NewTextData(t.TempDir())}}
	de.Typ = events.Drop
	fsNode2(ft, "sub").DragDrop(de)
	assert.Len(t, m.MapFS, 4, "nothing dropped")
}

// TestFSSort tests the sort order of the directories of a [Tree.FS]: the
// [Tree.FSSort], and that set for a directory with Sort by.
func TestFSSort(t *testing.T) {
	ft, m := newFSTree(t, true)
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
	m := &memFS{fstest.MapFS{
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
// are enabled for a [WriteFS], and disabled for a read-only FS.
func TestFSMenu(t *testing.T) {
	for _, writable := range []bool{true, false} {
		ft, _ := newFSTree(t, writable)
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
			assert.Equal(t, writable, enabled[item], "%s, writable: %v", item, writable)
		}
		assert.False(t, enabled["New file"], "not in an FS")
		_, ok := enabled["Remove from externs"]
		assert.False(t, ok, "not in an FS")
	}
}
