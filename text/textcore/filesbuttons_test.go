// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package textcore

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/text/lines"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// clickTop clicks the button with the given text in the dialog with the
// given title on top of the window of the body, waiting for it.
func clickTop(t *testing.T, b *core.Body, title, button string) {
	t.Helper()
	for range 500 {
		b.AsyncLock()
		if sc := core.TopScene(b); sc != nil && sc.Body != nil && sc.Body.Title == title {
			bt := core.ButtonByText(sc, button)
			if bt != nil {
				bt.Send(events.Click)
			}
			b.AsyncUnlock()
			require.NotNil(t, bt, button)
			return
		}
		b.AsyncUnlock()
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("no dialog %q", title)
}

// TestSaveCloseButtons tests the buttons of the dialogs of saving a file
// changed on disk, and of closing a file with unsaved changes.
func TestSaveCloseButtons(t *testing.T) {
	fname := filepath.Join(t.TempDir(), "f.txt")
	require.NoError(t, os.WriteFile(fname, []byte("one\n"), 0o644))
	ln := lines.NewLines()
	require.NoError(t, ln.Open(fname))
	b := core.NewBody()
	showed := make(chan struct{})
	b.OnFinal(events.Show, func(e events.Event) { showed <- struct{}{} })
	b.RunWindow()
	<-showed
	t.Cleanup(func() { b.AsyncLock(); b.Close(); b.AsyncUnlock() })
	locked := func(f func()) { b.AsyncLock(); defer b.AsyncUnlock(); f() }
	changeOnDisk := func(s string) {
		time.Sleep(10 * time.Millisecond) // a different modification time
		require.NoError(t, os.WriteFile(fname, []byte(s), 0o644))
	}
	read := func() string { got, _ := os.ReadFile(fname); return string(got) }

	// overwriting the file changed on disk
	changeOnDisk("disk\n")
	locked(func() { ln.SetText([]byte("mine\n")); assert.NoError(t, Save(b.Scene, ln)) })
	clickTop(t, b, "File Changed on Disk", "Save file, overwriting")
	assert.Eventually(t, func() bool { return read() == "mine\n" }, 2*time.Second, 10*time.Millisecond)

	// opening it from disk instead
	changeOnDisk("disk 2\n")
	locked(func() { ln.SetText([]byte("lost\n")); assert.NoError(t, Save(b.Scene, ln)) })
	clickTop(t, b, "File Changed on Disk", "Open from disk, losing changes")
	assert.Eventually(t, func() bool { return ln.String() == "disk 2\n" }, 2*time.Second, 10*time.Millisecond)

	// closing with unsaved changes: cancel, then save
	var after []bool
	afterFunc := func(canceled bool) { after = append(after, canceled) }
	locked(func() {
		ln.SetText([]byte("saved\n"))
		ln.InsertText(pos(0, 0), []rune("un")) // (an edit, which is not saved)
		require.True(t, ln.IsNotSaved())
		assert.False(t, Close(b.Scene, ln, afterFunc))
	})
	clickTop(t, b, "Close without saving?", "Cancel")
	assert.Eventually(t, func() bool { return len(after) == 1 }, 2*time.Second, 10*time.Millisecond)
	assert.Equal(t, []bool{true}, after)
	locked(func() { assert.False(t, Close(b.Scene, ln, afterFunc)) })
	clickTop(t, b, "Close without saving?", "Save")
	assert.Eventually(t, func() bool { return read() == "unsaved\n" }, 2*time.Second, 10*time.Millisecond, "saved before closing")
	assert.Eventually(t, func() bool { return len(after) == 2 }, 2*time.Second, 10*time.Millisecond)
	assert.Equal(t, false, after[1], "closed")
}
