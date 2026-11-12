// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package filetree

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"os"
	"path/filepath"
	"testing"
	"time"

	"cogentcore.org/core/base/dirsort"
	"cogentcore.org/core/base/fsx"
	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/icons"
)

func TestTree(t *testing.T) {
	b := core.NewBody()
	// testdata/events is a copy of the files of the events package at one
	// point (their contents matter for generated files, which are shown
	// differently), so that the image does not change when they change.
	NewTree(b).OpenPath("testdata/events")
	b.AssertRender(t, "basic")
}

// TestTreeReopen tests that the nodes of a tree opened at another path
// have the paths of that one, for directories with the same names.
func TestTreeReopen(t *testing.T) {
	a, b := t.TempDir(), t.TempDir()
	for _, dir := range []string{a, b} {
		require.NoError(t, os.Mkdir(filepath.Join(dir, "same"), 0o755))
	}
	require.NoError(t, os.Mkdir(filepath.Join(b, "other"), 0o755))
	bd := core.NewBody()
	ft := NewTree(bd)
	bd.AssertRender(t, "reopen", func() {
		bd.AsyncLock()
		defer bd.AsyncUnlock()
		ft.OpenPath(a)
		ft.OpenPath(b)
		root := string(ft.Filepath)
		var paths []string
		for _, c := range ft.Children {
			if fn, ok := c.(*Node); ok {
				paths = append(paths, string(fn.Filepath))
			}
		}
		assert.Equal(t, []string{filepath.Join(root, "other"), filepath.Join(root, "same")}, paths)
	})
}

// TestTreeDirSort tests that the files of a directory are listed in its
// sort order (see [dirsort.For]), with the directories first, and that
// sorting it by a field sets its sort order.
func TestTreeDirSort(t *testing.T) {
	root := t.TempDir()
	t0 := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	for i, f := range []struct {
		name string
		size int
	}{{"b.txt", 30}, {"A.txt", 10}, {"c.txt", 20}} {
		fn := filepath.Join(root, f.name)
		require.NoError(t, os.WriteFile(fn, make([]byte, f.size), 0o644))
		mt := t0.Add(time.Duration(i) * time.Hour)
		require.NoError(t, os.Chtimes(fn, mt, mt))
	}
	require.NoError(t, os.Mkdir(filepath.Join(root, "sub"), 0o755))
	names := func(ft *Tree) []string {
		var ns []string
		for _, info := range ft.dirFileList() {
			ns = append(ns, info.Name())
		}
		return ns
	}
	ft := NewTree(core.NewBody())
	ft.NoWatch = true
	ft.DirsOnTop = true
	ft.OpenPath(root)
	assert.Equal(t, []string{"sub", "A.txt", "b.txt", "c.txt"}, names(ft), "by name, up (ignoring case)")
	require.NoError(t, dirsort.Set(root, dirsort.Sort{By: dirsort.Size, Down: true}))
	assert.Equal(t, []string{"sub", "b.txt", "c.txt", "A.txt"}, names(ft))
	ft.sortBy(dirsort.Sort{By: dirsort.ModTime, Down: true})
	s, ok := dirsort.Saved(root)
	assert.True(t, ok)
	assert.Equal(t, dirsort.Sort{By: dirsort.ModTime, Down: true}, s)
	assert.Equal(t, []string{"sub", "c.txt", "A.txt", "b.txt"}, names(ft))
}

// TestTreeApp tests that clicking an app bundle (see [IsAppBundle]) does
// not open it in the tree, and that double-clicking it opens it as an app,
// with the mouse (clicks sent to the window), unless the tree shows apps
// as directories ([Tree.AppsAsDirs]), in which case clicking opens it.
func TestTreeApp(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(root, "Tool.app", "Contents"), 0o755))
	prevBundles, prevOpen := AppBundles, openURL
	t.Cleanup(func() { AppBundles, openURL = prevBundles, prevOpen })
	AppBundles = true
	var opened []string
	openURL = func(url string) { opened = append(opened, url) }
	b := core.NewBody()
	ft := NewTree(b)
	ft.NoWatch = true
	ft.OpenPath(root)
	var app *Node
	b.RenderImage(func() {
		b.AsyncLock()
		defer b.AsyncUnlock()
		app = AsNode(ft.Child(0))
		require.NotNil(t, app)
		assert.True(t, app.IsApp())
		assert.Equal(t, icons.Apps, app.Branch.IconOff)
	}, func() {
		src := b.Scene.RenderWindow().SystemWindow.Events()
		click := func(n int) {
			b.AsyncLock()
			bb := app.Parts.Geom.TotalBBox
			b.AsyncUnlock()
			pt := bb.Min.Add(bb.Size().Div(2))
			for range n {
				src.MouseButton(events.MouseDown, events.Left, pt, 0)
				src.MouseButton(events.MouseUp, events.Left, pt, 0)
			}
			time.Sleep(300 * time.Millisecond)
		}
		closed := func() bool {
			b.AsyncLock()
			defer b.AsyncUnlock()
			return app.Closed
		}
		click(1)
		assert.True(t, closed(), "a click does not open an app")
		assert.Empty(t, opened)
		click(2)
		assert.True(t, closed(), "a double click does not open an app in the tree")
		assert.Equal(t, []string{fsx.FileURL(filepath.Join(string(ft.Filepath), "Tool.app"))}, opened)

		b.AsyncLock()
		ft.AppsAsDirs = true
		assert.False(t, app.IsApp())
		b.AsyncUnlock()
		time.Sleep(600 * time.Millisecond) // (not a triple click)
		click(1)
		assert.False(t, closed(), "a click opens an app shown as a directory")
		assert.Len(t, opened, 1)
	})
}
