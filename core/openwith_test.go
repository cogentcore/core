// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"image"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"cogentcore.org/core/base/fileinfo"
	"cogentcore.org/core/base/fileinfo/apps"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// menuTexts returns the texts of the buttons of the given menu.
func menuTexts(menus ...func(m *Scene, pos image.Point)) []string {
	m := NewScene()
	for _, menu := range menus {
		menu(m, image.Point{})
	}
	var items []string
	for _, c := range m.Children {
		if bt, ok := c.(*Button); ok {
			items = append(items, bt.Text)
		}
	}
	return items
}

// TestOpenWith tests the items of [OpenWithMenuItems], with the apps that
// open a file, and the chooser of the apps, when the system has apps for
// text files, and that the [FilePicker] has them for files only.
func TestOpenWith(t *testing.T) {
	skipOnWeb(t, "opens files of the operating system with its apps")
	dir := t.TempDir()
	file := filepath.Join(dir, "b.txt")
	require.NoError(t, os.WriteFile(file, []byte("b"), 0644))
	require.NoError(t, os.Mkdir(filepath.Join(dir, "sub"), 0750))
	as, _ := apps.For(file)
	if len(as) == 0 {
		t.Skip("no apps for text files on this system")
	}
	items := []string{"Open with", "Change default app"}
	assert.Equal(t, items, menuTexts(func(m *Scene, pos image.Point) { OpenWithMenuItems(m, file) }))

	// the menu of the apps
	m := NewScene()
	OpenWithMenuItems(m, file)
	m.Child(0).(*Button).Menu(m, image.Point{}) // adds the apps to m
	var sub []string
	for _, c := range m.Children[2:] {
		if bt, ok := c.(*Button); ok {
			sub = append(sub, bt.Text)
		}
	}
	first := as[0].Name
	if as[0].Default { // not on linux without a default app for the type
		first += " (default)"
	}
	assert.Equal(t, first, sub[0])
	assert.Len(t, sub, len(as)+1)
	assert.Equal(t, otherApps, sub[len(sub)-1])

	// the chooser of the apps has all of them for Other...
	ch := newAppChooser(NewBody(), as)
	assert.Len(t, ch.Items, len(as)+1)
	app, ok := chosenApp(ch)
	assert.True(t, ok)
	assert.Equal(t, as[0], app, "the default")
	ch.SetCurrentValue(otherApps)
	ch.SendChange()
	all, err := apps.All()
	require.NoError(t, err)
	assert.Len(t, ch.Items, len(all))
	assert.Greater(t, len(all), len(as))
	assert.Len(t, newAppChooser(NewBody(), nil).Items, len(all), "all, with no apps for the file")

	// the file picker, for files only
	b := NewBody()
	fp := NewFilePicker(b).SetFilename(dir + string(filepath.Separator))
	b.RenderImage(func() {}, func() {
		tb := fp.filesTable
		index := func(name string) int {
			return slices.IndexFunc(fp.files, func(fi *fileinfo.FileInfo) bool { return fi.Name == name })
		}
		tb.SelectedIndex = index("b.txt")
		assert.Subset(t, menuTexts(tb.ContextMenus...), items)
		tb.SelectedIndex = index("sub")
		assert.NotContains(t, menuTexts(tb.ContextMenus...), items[0])
	})
}
