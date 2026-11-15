// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"errors"
	"image"
	"testing"

	"cogentcore.org/core/events/key"
	"cogentcore.org/core/keymap"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/units"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestTextFieldBackspace tests deleting the rune before the cursor, and
// the selection, with backspace.
func TestTextFieldBackspace(t *testing.T) {
	b := NewBody()
	tf := NewTextField(b).SetText("hello")
	RenderImage(b, Locked(b, func() {
		tf.SetFocus()
		SendKeyFunction(tf, keymap.End)
		SendKeyFunction(tf, keymap.Backspace)
		SendKeyFunction(tf, keymap.Backspace)
		assert.Equal(t, "hel", string(tf.editText))
		SendKeyFunction(tf, keymap.Home)
		SendKeyFunction(tf, keymap.Backspace)
		assert.Equal(t, "hel", string(tf.editText), "nothing before the start")
		SendKeyFunction(tf, keymap.SelectAll)
		SendKeyFunction(tf, keymap.Backspace)
		assert.Equal(t, "", string(tf.editText), "the selection")
	}))
}

// TestTextFieldWrappedLines tests moving the cursor up and down the lines
// of a text field whose text wraps.
func TestTextFieldWrappedLines(t *testing.T) {
	b := NewBody()
	tf := NewTextField(b).SetText("one two three four five six seven eight nine ten eleven twelve")
	tf.Styler(func(s *styles.Style) {
		s.Min.X.Em(8)
		s.Max.X.Em(8)
	})
	AssertRender(t, b, "text-field/wrapped-lines", Locked(b, func() {
		require.Greater(t, tf.numLines, 2, "the text wraps")
		tf.SetFocus()
		SendKeyFunction(tf, keymap.Home)
		assert.Equal(t, 0, tf.cursorLine)
		SendKeyFunction(tf, keymap.MoveDown)
		SendKeyFunction(tf, keymap.MoveDown)
		assert.Equal(t, 2, tf.cursorLine)
		SendKeyFunction(tf, keymap.MoveUp)
		assert.Equal(t, 1, tf.cursorLine)
		pos := tf.cursorPos
		rn, code, mods, err := keymap.MoveDown.Chord().Decode()
		require.NoError(t, err)
		mods.SetFlag(true, key.Shift)
		sendKeyMods(tf, rn, code, mods)
		assert.True(t, tf.hasSelection(), "shift selects")
		assert.Greater(t, tf.cursorPos, pos)
		for range tf.numLines {
			SendKeyFunction(tf, keymap.MoveDown)
		}
		assert.Equal(t, tf.numLines-1, tf.cursorLine, "stops at the last line")
		for range tf.numLines {
			SendKeyFunction(tf, keymap.MoveUp)
		}
		assert.Equal(t, 0, tf.cursorLine, "stops at the first line")
	}))
}

// TestTextFieldContextMenuError tests the context menu of a text field,
// and clearing a validation error when the text becomes valid.
func TestTextFieldContextMenuError(t *testing.T) {
	b := NewBody()
	b.Styler(func(s *styles.Style) { s.Min.Set(units.Em(15), units.Em(6)) })
	tf := NewTextField(b).SetText("short")
	tf.SetValidator(func() error {
		if len(tf.Text()) < 8 {
			return errors.New("too short")
		}
		return nil
	})
	RenderImage(b, Locked(b, func() {
		m := NewScene()
		tf.contextMenu(m, image.Point{})
		assert.NotNil(t, ButtonByText(m, "Copy"))
		assert.NotNil(t, ButtonByText(m, "Paste"))
		tf.SetReadOnly(true)
		m = NewScene()
		tf.contextMenu(m, image.Point{})
		assert.Nil(t, ButtonByText(m, "Paste"), "read-only")
		tf.SetReadOnly(false)

		tf.SendChange()
		assert.Error(t, tf.error)
		tf.SetText("long enough")
		tf.SendChange()
		assert.NoError(t, tf.error, "cleared")
	}))
}
