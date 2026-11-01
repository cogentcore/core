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
	b.AssertRender(t, "text-field/compose", func() {
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
