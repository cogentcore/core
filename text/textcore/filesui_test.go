// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package textcore

import (
	"os"
	"path/filepath"
	"testing"

	"cogentcore.org/core/core"
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
