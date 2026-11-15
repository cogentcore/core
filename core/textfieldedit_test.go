// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"testing"
	"time"

	"cogentcore.org/core/base/fileinfo/mimedata"
	"cogentcore.org/core/keymap"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/text/textpos"
	"github.com/stretchr/testify/assert"
)

// newEditTextField returns a new text field with the given text, in a
// new body, wide enough for the text to all be visible.
func newEditTextField(text string) (*Body, *TextField) {
	b := NewBody()
	tf := NewTextField(b).SetText(text)
	tf.Styler(func(s *styles.Style) {
		s.Min.X.Em(30)
	})
	return b, tf
}

// TestTextFieldWordMoves tests moving the cursor by words, and to the
// start and end, with the keyboard.
func TestTextFieldWordMoves(t *testing.T) {
	b, tf := newEditTextField("one two three")
	AssertRender(t, b, "text-field/word-moves", Locked(b, func() {
		tf.SetFocus()
		SendKeyFunction(tf, keymap.Home)
		assert.Equal(t, 0, tf.CursorPos())
		SendKeyFunction(tf, keymap.WordRight)
		assert.Equal(t, 3, tf.CursorPos())
		SendKeyFunction(tf, keymap.WordRight)
		assert.Equal(t, 7, tf.CursorPos())
		SendKeyFunction(tf, keymap.WordLeft)
		assert.Equal(t, 3, tf.CursorPos(), "word left stops at the space before the word")
		SendKeyFunction(tf, keymap.End)
		assert.Equal(t, 13, tf.CursorPos())
		SendKeyFunction(tf, keymap.WordLeft)
		assert.Equal(t, 7, tf.CursorPos())
		SendKeyFunction(tf, keymap.MoveLeft)
		assert.Equal(t, 6, tf.CursorPos())
		tf.SetCursorPos(100)
		assert.Equal(t, 13, tf.CursorPos(), "clamped to the end")
		tf.SetCursorPos(2).SetCursorEnd()
		assert.Equal(t, 13, tf.CursorPos())
	}))
}

// TestTextFieldDeleteWords tests deleting words and killing to the end
// of the text with the keyboard.
func TestTextFieldDeleteWords(t *testing.T) {
	b, tf := newEditTextField("one two three")
	AssertRender(t, b, "text-field/delete-words", Locked(b, func() {
		tf.SetFocus()
		SendKeyFunction(tf, keymap.End)
		SendKeyFunction(tf, keymap.BackspaceWord)
		assert.Equal(t, "one two", string(tf.editText))
		assert.Equal(t, 7, tf.CursorPos())
		SendKeyFunction(tf, keymap.Home)
		SendKeyFunction(tf, keymap.DeleteWord)
		assert.Equal(t, " two", string(tf.editText))
		assert.Equal(t, 0, tf.CursorPos(), "the cursor stays")
		SendKeyFunction(tf, keymap.MoveRight)
		SendKeyFunction(tf, keymap.Kill)
		assert.Equal(t, " ", string(tf.editText))
		assert.Equal(t, 1, tf.CursorPos())
	}))
}

// TestTextFieldCutPaste tests cutting, copying, and pasting the selected
// text through the clipboard.
func TestTextFieldCutPaste(t *testing.T) {
	b, tf := newEditTextField("hello world")
	AssertRender(t, b, "text-field/cut-paste", Locked(b, func() {
		tf.SetFocus()
		tf.SetSelectRange(textpos.Range{Start: 0, End: 5})
		assert.True(t, tf.hasSelection())
		SendKeyFunction(tf, keymap.Copy)
		assert.False(t, tf.hasSelection(), "copying clears the selection")
		assert.Equal(t, "hello", tf.Clipboard().Read([]string{mimedata.TextPlain}).Text(mimedata.TextPlain))

		tf.SetSelectRange(textpos.Range{Start: 5, End: 11})
		SendKeyFunction(tf, keymap.Cut)
		assert.Equal(t, "hello", string(tf.editText))
		assert.Equal(t, " world", tf.Clipboard().Read([]string{mimedata.TextPlain}).Text(mimedata.TextPlain))

		tf.SetCursorPos(0)
		SendKeyFunction(tf, keymap.Paste)
		assert.Equal(t, " worldhello", string(tf.editText))
		assert.Equal(t, 6, tf.CursorPos())

		tf.SetSelectRange(textpos.Range{Start: 6, End: 11})
		tf.SetCursorPos(7)
		SendKeyFunction(tf, keymap.Paste)
		assert.Equal(t, " world world", string(tf.editText), "replaces the selection with the cursor in it")
	}))
}

// TestTextFieldNoEchoCopy tests that the text of a password field is not
// copied or cut to the clipboard.
func TestTextFieldNoEchoCopy(t *testing.T) {
	b := NewBody()
	tf := NewTextField(b).SetTypePassword().SetText("secret")
	AssertRender(t, b, "text-field/no-echo-copy", Locked(b, func() {
		tf.Clipboard().Write(mimedata.NewText("clip"))
		tf.SetSelectRange(textpos.Range{Start: 0, End: 6})
		tf.copy()
		tf.cut()
		assert.Equal(t, "clip", tf.Clipboard().Read([]string{mimedata.TextPlain}).Text(mimedata.TextPlain))
		assert.Equal(t, "secret", string(tf.editText))
	}))
}

// TestTextFieldUndoRedo tests undoing and redoing edits, where quick
// edits are grouped into one undo.
func TestTextFieldUndoRedo(t *testing.T) {
	b, tf := newEditTextField("start")
	AssertRender(t, b, "text-field/undo-redo", Locked(b, func() {
		tf.SetFocus()
		SendKeyFunction(tf, keymap.Undo)
		assert.Equal(t, "start", string(tf.editText), "nothing to undo")
		tf.SetText("")

		TypeText(tf, "hello")
		tf.undos.lastSave = time.Time{} // ends the group of edits
		TypeText(tf, " you")
		assert.Equal(t, "hello you", string(tf.editText))

		SendKeyFunction(tf, keymap.Undo)
		assert.Equal(t, "hello", string(tf.editText), "undoes the last group")
		SendKeyFunction(tf, keymap.Undo)
		assert.Equal(t, "", string(tf.editText), "undoes the first group")
		SendKeyFunction(tf, keymap.Undo)
		assert.Equal(t, "", string(tf.editText))

		SendKeyFunction(tf, keymap.Redo)
		assert.Equal(t, "hello", string(tf.editText))
		SendKeyFunction(tf, keymap.Redo)
		assert.Equal(t, "hello you", string(tf.editText))
		SendKeyFunction(tf, keymap.Redo)
		assert.Equal(t, "hello you", string(tf.editText), "nothing to redo")
	}))
}

// TestTextFieldSearch tests searching the text of a text field.
func TestTextFieldSearch(t *testing.T) {
	b, tf := newEditTextField("abc ABC abc")
	AssertRender(t, b, "text-field/search", Locked(b, func() {
		res := SearchWidgets(b, "abc", UseCase)
		assert.Equal(t, 2, res.Len())
		assert.Equal(t, 3, SearchWidgets(b, "abc", IgnoreCase).Len())
		SearchHighlight(b, res)
		assert.Len(t, tf.highlights, 2)
		SearchSelect(res, 1, SelectSet)
		assert.Equal(t, textpos.Range{Start: 8, End: 11}, tf.selectRange)
		SearchSelect(res, 1, SelectReset)
		assert.False(t, tf.hasSelection())
		assert.Equal(t, "abc ABC abc", string(tf.TextRunes()))
	}))
}
