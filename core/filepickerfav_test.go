// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"path/filepath"
	"slices"
	"testing"

	"cogentcore.org/core/events"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestFilePickerFavorites tests adding the current directory to the
// favorites, going to a favorite, the toolbar, the recent paths, and the
// dialog for editing them. (The settings of the offscreen app are saved in
// a temporary directory.)
func TestFilePickerFavorites(t *testing.T) {
	skipOnWeb(t, "uses temporary directories of the operating system")
	dir := newPickerDir(t)
	prevFavs := slices.Clone(SystemSettings.FavPaths)
	defer func() { SystemSettings.FavPaths = prevFavs }()
	b := NewBody()
	fp := NewFilePicker(b).SetFilename(filepath.Join(dir, "a.go"))
	tb := NewToolbar(b)
	tb.Maker(fp.MakeToolbar)
	RenderImage(b, Locked(b, func() {
		assert.Len(t, tb.Children, 5, "up, favorite, update, new folder, and the overflow menu")

		sub := filepath.Join(dir, "sub")
		fp.directory = sub
		fp.updateFilesEvent()
		n := len(SystemSettings.FavPaths)
		fp.addPathToFavorites()
		require.Len(t, SystemSettings.FavPaths, n+1)
		fav := SystemSettings.FavPaths[n]
		assert.Equal(t, "sub", fav.Name)
		fp.addPathToFavorites()
		assert.Len(t, SystemSettings.FavPaths, n+1, "not added twice")

		fp.directoryUp()
		assert.Equal(t, dir, fp.directory)
		fp.favoritesSelect(n)
		assert.Equal(t, sub, fp.directory, "went to the favorite")
		fp.favoritesSelect(-1)
		fp.favoritesSelect(n + 1)
		assert.Equal(t, sub, fp.directory)

		var items []ChooserItem
		fp.addRecentPathItems(&items)
		assert.Contains(t, slices.Collect(func(yield func(any) bool) {
			for _, it := range items {
				if !yield(it.Value) {
					return
				}
			}
		}), dir, "the recent paths")
		fp.editRecentPaths()
	}), Locked(b, func() {
		d := TopScene(b)
		require.NotNil(t, d)
		assert.Equal(t, "Recent file paths", d.Body.Title)
		ok := ButtonByText(d, "OK")
		require.NotNil(t, ok)
		ok.Send(events.Click)
	}))
}
