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

	"cogentcore.org/core/core"
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
