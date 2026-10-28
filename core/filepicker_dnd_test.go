// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"os"
	"path/filepath"
	"testing"

	"cogentcore.org/core/base/fileinfo/mimedata"
	"cogentcore.org/core/events"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func writeTestFile(t *testing.T, path, text string) {
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte(text), 0o644))
}

func readTestFile(t *testing.T, path string) string {
	b, err := os.ReadFile(path)
	require.NoError(t, err)
	return string(b)
}

// testFileDragger is a [FileDragger] drag source.
type testFileDragger struct {
	files []string
}

func (fd *testFileDragger) DragFiles() []string { return fd.files }

func TestDroppedFiles(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "f.txt")
	writeTestFile(t, f, "f\n")
	d := filepath.Join(dir, "d")
	require.NoError(t, os.Mkdir(d, 0o755))
	md := mimedata.Mimes{
		mimedata.NewTextData(f),
		mimedata.NewTextData("file://" + d), // directories too
		mimedata.NewTextData(filepath.Join(dir, "missing.txt")),
		mimedata.NewTextData("relative.txt"),
		mimedata.NewTextData("some text"),
		{Type: "image/png", Data: []byte(f)},
	}
	assert.Equal(t, []string{f, d}, DroppedFiles(&events.DragDrop{Data: md}))

	fd := &testFileDragger{files: []string{d}}
	assert.Equal(t, []string{d}, DroppedFiles(&events.DragDrop{Data: md, Source: fd}), "files of the source")
}

func TestFilePickerDrop(t *testing.T) {
	dir := t.TempDir()
	other := t.TempDir()
	writeTestFile(t, filepath.Join(dir, "a.txt"), "a\n")
	writeTestFile(t, filepath.Join(other, "a.txt"), "other a\n")
	writeTestFile(t, filepath.Join(other, "b.txt"), "b\n")
	writeTestFile(t, filepath.Join(other, "d", "c.txt"), "c\n")
	writeTestFile(t, filepath.Join(other, "m.txt"), "m\n")

	b := NewBody()
	fp := NewFilePicker(b).SetFilename(dir + string(filepath.Separator))
	b.runAndShowNewWindow()
	drop := func(mod events.DropMods, paths ...string) {
		var md mimedata.Mimes
		for _, p := range paths {
			md = append(md, mimedata.NewTextData(p))
		}
		b.AsyncLock()
		defer b.AsyncUnlock()
		de := &events.DragDrop{DropMod: mod, Data: md}
		de.Typ = events.Drop
		fp.filesDrop(de)
	}

	// copy a file and a directory
	drop(events.DropCopy, filepath.Join(other, "b.txt"), filepath.Join(other, "d"))
	assert.Equal(t, "b\n", readTestFile(t, filepath.Join(dir, "b.txt")))
	assert.Equal(t, "c\n", readTestFile(t, filepath.Join(dir, "d", "c.txt")))
	assert.FileExists(t, filepath.Join(other, "b.txt"))

	// move
	drop(events.DropMove, filepath.Join(other, "m.txt"))
	assert.Equal(t, "m\n", readTestFile(t, filepath.Join(dir, "m.txt")))
	assert.NoFileExists(t, filepath.Join(other, "m.txt"))

	// an existing file is not overwritten without asking
	drop(events.DropCopy, filepath.Join(other, "a.txt"))
	assert.Equal(t, "a\n", readTestFile(t, filepath.Join(dir, "a.txt")))

	// a file in the directory itself is left alone
	drop(events.DropMove, filepath.Join(dir, "a.txt"))
	assert.Equal(t, "a\n", readTestFile(t, filepath.Join(dir, "a.txt")))

	rc := b.Scene.renderContext()
	rc.Lock()
	b.Close()
	rc.Unlock()
}

func TestMovePath(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, filepath.Join(dir, "d", "x.txt"), "x\n")
	require.NoError(t, movePath(filepath.Join(dir, "e"), filepath.Join(dir, "d")))
	assert.Equal(t, "x\n", readTestFile(t, filepath.Join(dir, "e", "x.txt")))
	assert.NoDirExists(t, filepath.Join(dir, "d"))
}
