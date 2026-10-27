// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"

	"cogentcore.org/core/base/fileinfo"
)

func TestFilePicker(t *testing.T) {
	b := NewBody()
	// The files of the directory do not change, unlike those of the
	// current directory, which are shown by default.
	NewFilePicker(b).SetFilename("testdata/file-picker/files/")
	b.AssertRender(t, "file-picker/basic")
}

func TestFilePickerWatch(t *testing.T) {
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
