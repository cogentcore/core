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
	"cogentcore.org/core/events/key"
	"cogentcore.org/core/text/lines"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// pressReturn presses the return key in the window of the given body,
// which activates the button with the focus in a dialog.
func pressReturn(b *core.Body) func() {
	return func() {
		b.Scene.RenderWindow().SystemWindow.Events().KeyChord(0, key.CodeReturnEnter, 0)
	}
}

// TestSaveAsClose tests saving as a new and an existing file, which asks
// first, and closing, which asks first for unsaved changes.
func TestSaveAsClose(t *testing.T) {
	dir := t.TempDir()
	defer core.TrackWindows()() // (the file picker of save as, before dir is removed)
	newf := filepath.Join(dir, "new.txt")
	oldf := filepath.Join(dir, "old.txt")
	require.NoError(t, os.WriteFile(oldf, []byte("old\n"), 0o644))
	b := core.NewBody()
	ed := NewEditor(b)
	ed.Lines.SetString("text\n")
	var after []bool
	afterFunc := func(canceled bool) { after = append(after, canceled) }
	core.RenderImage(b, core.Locked(b, func() {
		SaveAs(b.Scene, ed.Lines, newf, afterFunc)
		got, err := os.ReadFile(newf)
		require.NoError(t, err)
		assert.Equal(t, "text\n", string(got))
		assert.Equal(t, []bool{false}, after)

		ed.SaveAs(core.Filename(oldf)) // exists: a dialog asks first
	}), pressReturn(b), core.Locked(b, func() {
		assert.Error(t, Save(b.Scene, lines.NewLines()), "no filename")

		require.NoError(t, ed.Lines.Open(newf))
		after = nil
		assert.True(t, ed.Close(afterFunc), "closed without changes")
		assert.Equal(t, []bool{false}, after)

		ed.SetLines(lines.NewLines()) // the closed lines are dropped
		require.NoError(t, ed.Lines.Open(newf))
		ed.Lines.InsertText(pos(0, 0), []rune("changed "))
		require.True(t, ed.Lines.IsNotSaved())
		after = nil
		assert.False(t, ed.Close(afterFunc), "a dialog asks first")
	}), pressReturn(b), core.Locked(b, func() {
		ed.SetLines(lines.NewLines())
		ed.Lines.SetString("no file\n")
		ed.Lines.InsertText(pos(0, 0), []rune("x"))
		assert.False(t, Close(b.Scene, ed.Lines, afterFunc), "asks, with no filename")
	}), pressReturn(b), core.Locked(b, func() {
		ed.SetLines(lines.NewLines())
		require.NoError(t, ed.Lines.Open(oldf))
		assert.True(t, FileModPrompt(b.Scene, ed.Lines))
		require.NoError(t, ed.SaveQuiet())
	}), pressReturn(b))
}

// TestFileModPromptButtons tests the buttons of the prompt for a file
// changed on disk: reverting from disk, ignoring the change, and saving
// as a different file.
func TestFileModPromptButtons(t *testing.T) {
	fn := filepath.Join(t.TempDir(), "a.txt")
	require.NoError(t, os.WriteFile(fn, []byte("old\n"), 0o644))
	b := core.NewBody()
	ed := NewEditor(b)
	require.NoError(t, ed.Lines.Open(fn))
	// changeOnDisk writes new text to the file, with a later time.
	changeOnDisk := func(txt string) {
		require.NoError(t, os.WriteFile(fn, []byte(txt), 0o644))
		later := time.Now().Add(time.Hour)
		require.NoError(t, os.Chtimes(fn, later, later))
	}
	// click clicks the button of the prompt dialog with the given text.
	click := func(text string) {
		dlg := b.Scene.RenderWindow().MainScene()
		require.NotNil(t, dlg)
		bt := core.ButtonByText(dlg, text)
		require.NotNil(t, bt, text)
		bt.Send(events.Click)
	}
	var picker *core.Scene // the scene of the save as dialog, in any window
	core.RenderImage(b, core.Locked(b, func() {
		ed.Lines.InsertText(pos(0, 0), []rune("edit "))
		changeOnDisk("new\n")
		FileModPrompt(b.Scene, ed.Lines)
	}), core.Locked(b, func() {
		click("Revert from disk")
		assert.Equal(t, "new\n", ed.Lines.String())
	}), core.Locked(b, func() {
		ed.Lines.InsertText(pos(0, 0), []rune("edit "))
		changeOnDisk("newer\n")
		FileModPrompt(b.Scene, ed.Lines)
	}), core.Locked(b, func() {
		click("Ignore and proceed")
		prompted := 0
		ed.Lines.FileModPromptFunc = func() { prompted++ }
		ed.Lines.InsertText(pos(0, 0), []rune("more "))
		assert.Equal(t, 0, prompted, "not asked again")
		assert.Equal(t, "more edit new\n", ed.Lines.String(), "the edits are kept")
	}), core.Locked(b, func() {
		FileModPrompt(b.Scene, ed.Lines)
	}), core.Locked(b, func() {
		click("Save as to different file")
	}), core.Locked(b, func() {
		for _, w := range core.AllRenderWindows {
			sc := w.MainScene()
			if sc == nil {
				continue
			}
			for cw := range sc.AllWidgets() {
				if _, ok := cw.(*core.FilePicker); ok {
					picker = sc
				}
			}
		}
		require.NotNil(t, picker, "a file picker to save as")
	}), func() { // in its own window, with its own lock
		picker.AsyncLock()
		picker.Close()
		picker.AsyncUnlock()
	})
}
