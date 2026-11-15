// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"cogentcore.org/core/events"
	"os"
	"path/filepath"
	"testing"

	"cogentcore.org/core/base/fileinfo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newPickerDir returns a temporary directory with a.go, b.txt, and sub/c.go.
func newPickerDir(t *testing.T) string {
	t.Helper()
	dir, err := filepath.EvalSymlinks(t.TempDir())
	require.NoError(t, err)
	for _, f := range []string{"a.go", "b.txt", "sub/c.go"} {
		p := filepath.Join(dir, filepath.FromSlash(f))
		require.NoError(t, os.MkdirAll(filepath.Dir(p), 0o755))
		require.NoError(t, os.WriteFile(p, []byte("x"), 0o644))
	}
	return dir
}

// pickerNames returns the names of the files of the file picker.
func pickerNames(fp *FilePicker) []string {
	var names []string
	for _, f := range fp.files {
		names = append(names, f.Name)
	}
	return names
}

// TestFilePickerNavigate tests selecting files, opening a directory, going
// up, making a new folder, and the selected file.
func TestFilePickerNavigate(t *testing.T) {
	skipOnWeb(t, "uses temporary directories of the operating system")
	dir := newPickerDir(t)
	b := NewBody()
	fp := NewFilePicker(b).SetFilename(filepath.Join(dir, "a.go"))
	selects := 0
	fp.OnSelect(func(e events.Event) { selects++ })
	RenderImage(b, Locked(b, func() {
		assert.ElementsMatch(t, []string{"a.go", "b.txt", "sub"}, pickerNames(fp))
		assert.Equal(t, filepath.Join(dir, "a.go"), fp.SelectedFile())

		sub := -1
		for i, f := range fp.files {
			if f.Name == "sub" {
				sub = i
			}
		}
		fp.fileSelect(sub)
		assert.Equal(t, "sub", fp.SelectedFileInfo().Name)
		assert.False(t, fp.selectFile(), "a directory is opened, not selected")
		assert.Equal(t, filepath.Join(dir, "sub"), fp.directory)
		assert.Equal(t, []string{"c.go"}, pickerNames(fp))
		assert.Nil(t, fp.SelectedFileInfo())

		fp.fileSelect(0)
		assert.True(t, fp.selectFile(), "a file is selected")
		assert.Equal(t, filepath.Join(dir, "sub", "c.go"), fp.SelectedFile())
		fp.fileSelect(-1)

		fp.directoryUp()
		assert.Equal(t, dir, fp.directory)
		require.NoError(t, fp.newFolder("made"))
		assert.Contains(t, pickerNames(fp), "made")
		assert.Greater(t, selects, 1)
	}))
}

// TestFilePickerFilters tests the directory and extension filters, and
// setting the extensions.
func TestFilePickerFilters(t *testing.T) {
	skipOnWeb(t, "uses temporary directories of the operating system")
	dir := newPickerDir(t)
	b := NewBody()
	fp := NewFilePicker(b).SetFilename(dir + string(filepath.Separator))
	RenderImage(b, Locked(b, func() {
		fp.Filterer = FilePickerDirOnlyFilter
		fp.updateFilesEvent()
		assert.Equal(t, []string{"sub"}, pickerNames(fp))

		fp.SetExtensions("go, TXT")
		assert.Equal(t, map[string]string{".go": ".go", ".txt": ".TXT"}, fp.extensionMap)
		fp.SetExtensions(".go")
		fp.Filterer = FilePickerExtensionOnlyFilter
		fp.updateFilesEvent()
		assert.ElementsMatch(t, []string{"a.go", "sub"}, pickerNames(fp), "files of the extension, and directories")

		fp.selectedFilename = "b.txt"
		fp.SetExtensions("")
		assert.Equal(t, ".txt", fp.extensions, "the extension of the selected file")
	}))
}

// TestFilePickerComplete tests completing file names, and selecting a
// file by name, which adds the extension if it is not found.
func TestFilePickerComplete(t *testing.T) {
	skipOnWeb(t, "uses temporary directories of the operating system")
	dir := newPickerDir(t)
	b := NewBody()
	fp := NewFilePicker(b).SetFilename(dir + string(filepath.Separator))
	RenderImage(b, Locked(b, func() {
		md := fp.fileComplete(nil, "a", 0, 1)
		require.Len(t, md.Matches, 1)
		assert.Equal(t, "a.go", md.Matches[0].Text)
		all := fp.fileComplete(nil, "", 0, 0)
		assert.Len(t, all.Matches, 3)
		ed := fp.fileCompleteEdit(nil, "a", 1, md.Matches[0], "a")
		assert.Equal(t, "a.go", ed.NewText)

		fp.setSelectedFile("b.txt")
		assert.Equal(t, "b.txt", fp.selectedFilename)
		assert.Equal(t, "b.txt", fp.SelectedFileInfo().Name)
		fp.extensionField.SetText(".go")
		fp.setSelectedFile("new")
		assert.Equal(t, "new.go", fp.selectedFilename, "the extension is added to a new file")

		info := fp.SelectedFileInfo()
		assert.True(t, info == nil || info.Name != "new.go")
		var _ *fileinfo.FileInfo = info
	}))
}
