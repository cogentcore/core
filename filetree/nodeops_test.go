// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package filetree

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"cogentcore.org/core/base/fileinfo"
	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newOpsTree returns a tree in a window of a temporary directory with
// a.go, b.go, c.txt, and sub/d.go.
func newOpsTree(t *testing.T) (*Tree, string) {
	t.Helper()
	dir, err := filepath.EvalSymlinks(t.TempDir())
	require.NoError(t, err)
	for _, f := range []string{"a.go", "b.go", "c.txt", "sub/d.go"} {
		p := filepath.Join(dir, f)
		require.NoError(t, os.MkdirAll(filepath.Dir(p), 0o755))
		require.NoError(t, os.WriteFile(p, []byte("package a\n"), 0o644))
	}
	b := core.NewBody()
	ft := NewTree(b)
	ft.OpenPath(dir)
	runTestWindow(t, b)
	return ft, dir
}

// opsNode returns the node of the given relative path in the tree.
func opsNode(t *testing.T, ft *Tree, dir, rel string) *Node {
	t.Helper()
	n, ok := ft.FindFile(filepath.Join(dir, filepath.FromSlash(rel)))
	require.True(t, ok, rel)
	return n
}

// TestNodeCounts tests the counts of file extensions, the latest
// modification time, and the relative paths of nodes.
func TestNodeCounts(t *testing.T) {
	ft, dir := newOpsTree(t)
	ft.AsyncLock()
	defer ft.AsyncUnlock()
	ft.openAll()
	counts := ft.FileExtensionCounts(fileinfo.Code)
	require.NotEmpty(t, counts)
	assert.Equal(t, NodeNameCount{Name: ".go", Count: 3}, counts[0])
	all := ft.FileExtensionCounts(fileinfo.UnknownCategory)
	found := map[string]int{}
	for _, c := range all {
		found[c.Name] = c.Count
	}
	assert.Equal(t, 1, found[".txt"])
	assert.Equal(t, 3, found[".go"])
	for i := 1; i < len(all); i++ {
		assert.GreaterOrEqual(t, all[i-1].Count, all[i].Count, "sorted by count")
	}

	// (file info has a fixed mod time in tests: see fileinfo.SetFileInfo)
	future := time.Now().Add(time.Hour)
	d := opsNode(t, ft, dir, "sub/d.go")
	d.Info.ModTime = future
	assert.WithinDuration(t, future, ft.LatestFileMod(fileinfo.Code), time.Second)
	assert.True(t, ft.LatestFileMod(fileinfo.Doc).Before(future), "only of the category")
	assert.Equal(t, filepath.Join("sub", "d.go"), d.RelativePath())
	assert.Equal(t, "sub", opsNode(t, ft, dir, "sub").RelativePath())
	assert.Contains(t, ft.OpenPaths(), filepath.Join(dir, "sub"), "opened by openAll")
}

// TestNodeSelectedPaths tests the paths of the selected nodes.
func TestNodeSelectedPaths(t *testing.T) {
	ft, dir := newOpsTree(t)
	ft.AsyncLock()
	defer ft.AsyncUnlock()
	assert.Nil(t, ft.SelectedPaths())
	a := opsNode(t, ft, dir, "a.go")
	a.SelectEvent(events.SelectOne)
	opsNode(t, ft, dir, "c.txt").SelectEvent(events.ExtendOne)
	assert.ElementsMatch(t, []string{filepath.Join(dir, "a.go"), filepath.Join(dir, "c.txt")}, ft.SelectedPaths())
}

// TestExternalFiles tests adding external files to a tree, and removing
// them.
func TestExternalFiles(t *testing.T) {
	ft, _ := newOpsTree(t)
	ext := filepath.Join(t.TempDir(), "ext.go")
	require.NoError(t, os.WriteFile(ext, []byte("package ext\n"), 0o644))
	ft.AsyncLock()
	defer ft.AsyncUnlock()
	n, err := ft.AddExternalFile(ext)
	require.NoError(t, err)
	require.NotNil(t, n)
	assert.True(t, n.isExternal())
	again, err := ft.AddExternalFile(ext)
	require.NoError(t, err)
	assert.Same(t, n, again, "added once")
	_, err = ft.AddExternalFile(filepath.Join(t.TempDir(), "missing.go"))
	assert.Error(t, err)

	n.DeleteFilesNoPrompts()
	_, err = os.Stat(ext)
	assert.NoError(t, err, "an external file is not deleted")
	n.SelectEvent(events.SelectOne)
	n.removeFromExterns()
	has, _ := ft.hasExternalFile(ext)
	assert.False(t, has)
	assert.False(t, ft.removeExternalFile(ext), "already removed")
}

// TestNewFolderPaste tests making a new folder in the selected directory,
// and copying and pasting a file into it, through the clipboard.
func TestNewFolderPaste(t *testing.T) {
	ft, dir := newOpsTree(t)
	ft.AsyncLock()
	sub := opsNode(t, ft, dir, "sub")
	sub.SelectEvent(events.SelectOne)
	sub.newFolders("new")
	ft.AsyncUnlock()
	_, err := os.Stat(filepath.Join(dir, "sub", "new"))
	require.NoError(t, err, "the new folder")

	ft.AsyncLock()
	a := opsNode(t, ft, dir, "a.go")
	a.SelectEvent(events.SelectOne)
	a.Copy()
	sub.Update()
	nw := opsNode(t, ft, dir, "sub/new")
	nw.Paste()
	ft.AsyncUnlock()
	got, err := os.ReadFile(filepath.Join(dir, "sub", "new", "a.go"))
	require.NoError(t, err, "pasted")
	assert.Equal(t, "package a\n", string(got))
}
