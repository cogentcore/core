// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package content

import (
	"io/fs"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"cogentcore.org/core/core"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/units"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mountIcon returns an SVG icon of the given color for [TestMountContent].
func mountIcon(color string) *fstest.MapFile {
	return &fstest.MapFile{Data: []byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 10 10"><rect width="10" height="10" fill="` + color + `"/></svg>`)}
}

// mountRootFS is the root filesystem of the [MountFS] of the tests: a site
// home page with a settings page, which has the name of the settings page
// of the lists docs.
var mountRootFS = fstest.MapFS{
	"home.md":     {Data: []byte("+++\nURL = \"\"\n+++\n\nThe apps: [[Cogent Lists]] and [[Files]].\n")},
	"settings.md": {Data: []byte("Site settings.\n")},
	"docs/old.md": {Data: []byte("An old page.\n")},
}

// mountLists and mountFiles are the docs of two apps, each with a home page
// with a relative icon image, as used for the help of the apps.
var mountLists = fstest.MapFS{
	"home.md":     {Data: []byte("+++\nURL = \"\"\nTitle = \"Cogent Lists\"\n+++\n\n<img src=\"icon.svg\" width=\"64\">\n\nThe lists app. See [[Settings (Cogent Lists)]].\n")},
	"settings.md": {Data: []byte("Lists settings.\n")},
	"icon.svg":    mountIcon("red"),
}

var mountFiles = fstest.MapFS{
	"home.md":       {Data: []byte("+++\nURL = \"\"\n+++\n\n<img src=\"icon.svg\" width=\"64\">\n\nThe files app.\n")},
	"guide/tabs.md": {Data: []byte("<img src=\"../icon.svg\" width=\"32\">\n\nTabs.\n")},
	"icon.svg":      mountIcon("blue"),
}

// newTestMountFS returns the [MountFS] of the tests.
func newTestMountFS() *MountFS {
	return NewMountFS(mountRootFS).Mount("docs/lists", mountLists).Mount("docs/files", mountFiles)
}

// TestMountFS tests the filesystem of a [MountFS] with [fstest.TestFS],
// including a directory ("docs") that is in the root and has mounts in it.
func TestMountFS(t *testing.T) {
	m := newTestMountFS()
	require.NoError(t, fstest.TestFS(m, "home.md", "settings.md", "docs/old.md",
		"docs/lists/home.md", "docs/lists/settings.md", "docs/lists/icon.svg",
		"docs/files/home.md", "docs/files/guide/tabs.md", "docs/files/icon.svg"))
	b, err := fs.ReadFile(m, "docs/files/icon.svg")
	require.NoError(t, err)
	assert.Contains(t, string(b), "blue")

	only := NewMountFS(nil).Mount("a/b", mountLists)
	require.NoError(t, fstest.TestFS(only, "a/b/home.md", "a/b/icon.svg"))
	_, err = only.Open("home.md")
	assert.ErrorIs(t, err, fs.ErrNotExist)

	assert.Panics(t, func() { NewMountFS(nil).Mount(".", mountLists) })
	assert.Panics(t, func() { NewMountFS(nil).Mount("/a", mountLists) })
	assert.Panics(t, func() { newTestMountFS().Mount("docs/lists", mountFiles) })
}

// TestMountContent tests the pages of a [Content] with a [MountFS] source:
// the URLs and names of the mounted pages, and that the relative images of
// the mounted pages are relative to them.
func TestMountContent(t *testing.T) {
	ct := NewContent(core.NewBody()).SetSource(newTestMountFS())
	urls := map[string]string{}
	for _, pg := range ct.pages {
		urls[pg.URL] = pg.Name
	}
	assert.Equal(t, map[string]string{
		"":                      "Home",
		"settings":              "Settings",
		"docs/old":              "Docs/old", // (as before, for a page in a directory)
		"docs/lists":            "Cogent Lists",
		"docs/lists/settings":   "Settings (Cogent Lists)",
		"docs/files":            "Files",
		"docs/files/guide/tabs": "Guide/tabs",
	}, urls)
	assert.Equal(t, "Cogent Lists", ct.pageByName("cogent lists").Title)
	assert.Equal(t, "Files", ct.pagesByURL["docs/files"].Title)
	assert.True(t, ct.isPageURL("docs/lists/settings#heading"))
	assert.False(t, ct.isPageURL("docs/lists/icon.svg"))

	pages := []struct {
		url, base string
		images    int
	}{
		{"", "", 0},
		{"docs/lists", "docs/lists/", 1},
		{"docs/files/guide/tabs", "docs/files/guide/", 1},
		{"settings", "", 0},
	}
	for _, tp := range pages {
		name := tp.url
		if name == "" {
			name = "home"
		}
		t.Run(name, func(t *testing.T) {
			b := core.NewBody()
			b.Styler(func(s *styles.Style) { s.Min.Set(units.Dp(800), units.Dp(500)) })
			ct := NewContent(b).SetSource(newTestMountFS())
			core.AssertRender(t, b, filepath.Join("mount", strings.ReplaceAll(name, "/", "-")), func() {
				ct.AsyncLock()
				ct.Open(tp.url)
				ct.AsyncUnlock()
			}, func() {
				time.Sleep(200 * time.Millisecond) // to give the image time to load
				ct.AsyncLock()
				defer ct.AsyncUnlock()
				assert.Equal(t, tp.url, ct.current.Page.URL)
				assert.Equal(t, tp.base, ct.Context.PageURL)
				assert.Equal(t, tp.images, countImages(ct), "the icon of the app, resolved relative to the page")
			})
		})
	}
}

// countImages returns the number of images in the page of the given content
// that have their image loaded.
func countImages(ct *Content) int {
	n := 0
	ct.rightFrame.WidgetWalkDown(func(cw core.Widget, cwb *core.WidgetBase) bool {
		if im, ok := cw.(*core.Image); ok && im.Image != nil {
			n++
		}
		if sv, ok := cw.(*core.SVG); ok && sv.SVG != nil && sv.SVG.Root.NumChildren() > 0 {
			n++
		}
		return true
	})
	return n
}
