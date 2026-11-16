// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"testing"

	"cogentcore.org/core/events"
	"github.com/stretchr/testify/assert"
)

// TestTextFieldCompose tests composing text with an input method, as sent
// by the drivers: the text being composed is shown at the cursor without
// an Input event, and the committed text follows its end as KeyChords.
func TestTextFieldCompose(t *testing.T) {
	b := NewBody()
	tf := NewTextField(b).SetText("hello world")
	inputs := 0
	tf.OnInput(func(e events.Event) { inputs++ })
	compose := func(text string, start, end int) {
		tf.HandleEvent(events.NewTextComposition(text, start, end))
	}
	AssertRender(t, b, "text-field/compose", func() {
		b.AsyncLock() // (it is rendered meanwhile)
		defer b.AsyncUnlock()
		tf.SetFocus()
		tf.cursorPos = 6
		compose("にほん", 3, 3)
		assert.Equal(t, "hello にほんworld", string(tf.editText))
		assert.Equal(t, 9, tf.cursorPos)
		compose("日本", 0, 2) // converted, with the clause selected
		assert.Equal(t, "hello 日本world", string(tf.editText))
		assert.Equal(t, 8, tf.cursorPos)
		assert.Equal(t, 0, inputs)
	}, func() {
		b.AsyncLock() // (it is rendered meanwhile)
		defer b.AsyncUnlock()
		// commit
		compose("", 0, 0)
		assert.Equal(t, "hello world", string(tf.editText))
		assert.Equal(t, 6, tf.cursorPos)
		for _, r := range "日本" {
			tf.HandleEvent(events.NewKey(events.KeyChord, r, 0, 0))
		}
		assert.Equal(t, "hello 日本world", string(tf.editText))
		assert.Equal(t, 8, tf.cursorPos)

		// composing replaces the selection
		tf.selectRange.Start, tf.selectRange.End = 0, 5
		tf.cursorPos = 5
		compose("x", 1, 1)
		assert.Equal(t, "x 日本world", string(tf.editText))
		compose("", 0, 0)
		assert.Equal(t, " 日本world", string(tf.editText))
	})
}

// TestTextFieldTextInputBefore tests that a focused text field sets the
// text before its cursor in the text input state of its window, which
// the keyboard of iOS uses for its predictions: after the cursor moves,
// as by a tap, before the text being composed, and none for a password.
func TestTextFieldTextInputBefore(t *testing.T) {
	b := NewBody()
	tf := NewTextField(b).SetText("Milk and eggs")
	pw := NewTextField(b).SetTypePassword().SetText("secret")
	before := func() string {
		rw := b.Scene.RenderWindow()
		if rw.textInput == nil {
			return "<none>"
		}
		return rw.textInput.Before
	}
	b.runAndShowNewWindow()
	defer func() {
		b.AsyncLock()
		b.Close()
		b.AsyncUnlock()
	}()
	locked := func(fun func()) {
		b.AsyncLock()
		defer b.AsyncUnlock()
		fun()
	}
	locked(func() {
		tf.SetFocus()
		tf.cursorPos = 8
		tf.updateTextInput()
		assert.Equal(t, "Milk and", before())
		tf.cursorPos = 4 // moved, as by a tap
		tf.updateTextInput()
		assert.Equal(t, "Milk", before())
		tf.HandleEvent(events.NewTextComposition("にほん", 3, 3))
		tf.updateTextInput()
		assert.Equal(t, "Milk", before(), "before the text being composed")
		tf.HandleEvent(events.NewTextComposition("", 0, 0))
		pw.SetFocus()
		pw.cursorPos = 6
		pw.updateTextInput()
		assert.Equal(t, "", before(), "not the text of a password")
	})
}
