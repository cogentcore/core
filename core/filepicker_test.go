// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"testing"
	"time"

	"cogentcore.org/core/base/dirsort"
	"cogentcore.org/core/base/fileinfo"
	"github.com/stretchr/testify/assert"
)

func TestFilePicker(t *testing.T) {
	b := NewBody()
	// The files of the directory do not change, unlike those of the
	// current directory, which are shown by default.
	NewFilePicker(b).SetFilename("testdata/file-picker/files/")
	b.AssertRender(t, "file-picker/basic")
}

// TestFilePickerSort tests that the files of a directory are in its sort
// order (see [dirsort.For]), and that sorting them by a column sets it.
func TestFilePickerSort(t *testing.T) {
	skipOnWeb(t, "uses temporary directories of the operating system")
	dir := t.TempDir()
	t0 := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	for i, f := range []struct {
		name string
		size int
	}{{"B.txt", 20}, {"a.txt", 10}, {"c.txt", 20}} {
		fn := filepath.Join(dir, f.name)
		os.WriteFile(fn, make([]byte, f.size), 0644)
		mt := t0.Add(time.Duration(i) * time.Hour)
		os.Chtimes(fn, mt, mt)
	}
	// (the times of files are all the same in tests)
	assert.NoError(t, dirsort.Set(dir, dirsort.Sort{By: dirsort.Size, Down: true}))
	b := NewBody()
	fp := NewFilePicker(b).SetFilename(dir + string(filepath.Separator))
	names := func() []string {
		var ns []string
		for _, fi := range fp.files {
			ns = append(ns, fi.Name)
		}
		return ns
	}
	b.RenderImage(func() {}, func() {
		b.AsyncLock() // (it is rendered meanwhile)
		defer b.AsyncUnlock()
		tb := fp.filesTable
		assert.Equal(t, "Size:down", tb.SortFieldName())
		assert.Equal(t, []string{"c.txt", "B.txt", "a.txt"}, names(), "equal sizes by name, down")
		size := slices.IndexFunc(tb.visibleFields, func(f reflect.StructField) bool { return f.Name == "Size" })
		tb.SortColumn(size) // (as by clicking on its header, again: up)
		assert.Equal(t, []string{"a.txt", "B.txt", "c.txt"}, names())
		s, ok := dirsort.Saved(dir)
		assert.True(t, ok)
		assert.Equal(t, dirsort.Sort{By: dirsort.Size}, s, "set for the directory")
		name := slices.IndexFunc(tb.visibleFields, func(f reflect.StructField) bool { return f.Name == "Name" })
		tb.SortColumn(name)
		tb.SortColumn(name) // (down)
		assert.Equal(t, []string{"c.txt", "B.txt", "a.txt"}, names(), "ignoring case")
	})
}

func TestFilePickerWatch(t *testing.T) {
	skipOnWeb(t, "watches the files of the operating system")
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "a.txt"), []byte("a\n"), 0644)
	b := NewBody()
	fp := NewFilePicker(b).SetFilename(dir + string(filepath.Separator))
	b.runAndShowNewWindow()
	has := func(name string) bool {
		b.AsyncLock()
		defer b.AsyncUnlock()
		return slices.ContainsFunc(fp.files, func(fi *fileinfo.FileInfo) bool { return fi.Name == name })
	}
	waitFor := func(what string, cond func() bool) {
		t.Helper()
		deadline := time.Now().Add(5 * time.Second)
		for !cond() {
			if time.Now().After(deadline) {
				t.Fatal("timed out waiting for", what)
			}
			time.Sleep(20 * time.Millisecond)
		}
	}
	if !has("a.txt") {
		t.Fatal("a.txt not listed")
	}
	time.Sleep(300 * time.Millisecond) // watcher start
	os.WriteFile(filepath.Join(dir, "b.txt"), []byte("b\n"), 0644)
	waitFor("b.txt listed", func() bool { return has("b.txt") })
	os.Remove(filepath.Join(dir, "a.txt"))
	waitFor("a.txt removed", func() bool { return !has("a.txt") })

	rc := b.Scene.renderContext()
	rc.Lock()
	b.Close()
	rc.Unlock()
}
