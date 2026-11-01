// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"image"
	"slices"

	"cogentcore.org/core/events"
	"cogentcore.org/core/styles/states"
	"cogentcore.org/core/text/rich"
)

// textFieldComposition is the state of text being composed with an input
// method (IME) in a [TextField], which is provisionally in its edit text,
// underlined, until the composition ends, when it is removed: any committed
// text then follows as KeyChord events, which insert it as typed text.
type textFieldComposition struct {

	// active is whether text is being composed.
	active bool

	// start is the position of the composition in the edit text, in runes,
	// and n is its length.
	start, n int
}

// compose handles a [events.Composition] event from an input method,
// showing the text being composed at the cursor, or ending the
// composition if it is empty.
func (tf *TextField) compose(e *events.TextComposition) {
	if tf.IsReadOnly() || tf.NoEcho {
		return
	}
	if e.Text == "" {
		tf.endComposition()
		return
	}
	cp := &tf.composition
	if !cp.active {
		if tf.hasSelection() { // typing replaces the selection
			tf.deleteSelection()
		}
		cp.active, cp.start, cp.n = true, tf.cursorPos, 0
	}
	txt := []rune(e.Text)
	tf.editText = slices.Replace(tf.editText, cp.start, cp.start+cp.n, txt...)
	tf.dispRange.End = max(0, min(tf.dispRange.End+len(txt)-cp.n, len(tf.editText)))
	cp.n = len(txt)
	tf.cursorPos = cp.start + max(0, min(e.End, cp.n)) // the input method cursor
	tf.renderVisible = nil
	tf.NeedsRender()
}

// endComposition ends any composition, removing its text.
func (tf *TextField) endComposition() {
	cp := &tf.composition
	if !cp.active {
		return
	}
	cp.active = false
	end := min(cp.start+cp.n, len(tf.editText))
	tf.editText = slices.Delete(tf.editText, cp.start, end)
	tf.dispRange.End = max(0, min(tf.dispRange.End-(end-cp.start), len(tf.editText)))
	tf.dispRange.Start = min(tf.dispRange.Start, tf.dispRange.End)
	tf.cursorPos = cp.start
	tf.renderVisible = nil
	tf.NeedsRender()
}

// compositionText returns the visible text as rich text,
// with the part of it that is being composed underlined.
func (tf *TextField) compositionText(sty *rich.Style, cur []rune) rich.Text {
	cp := &tf.composition
	st := max(cp.start-tf.dispRange.Start, 0)
	ed := min(cp.start+cp.n-tf.dispRange.Start, len(cur))
	if st >= ed {
		return rich.NewText(sty, cur)
	}
	usty := *sty
	usty.SetDecoration(rich.Underline)
	var tx rich.Text
	tx.AddSpan(sty, cur[:st]).AddSpan(&usty, cur[st:ed]).AddSpan(sty, cur[ed:])
	return tx
}

// updateTextInput updates the text input state of the window for the
// input method, with the position of the cursor, which is that of the
// input method within the text being composed, if any.
func (tf *TextField) updateTextInput() {
	if tf.IsReadOnly() || !tf.StateIs(states.Focused) || tf.renderVisible == nil {
		return
	}
	min := tf.charRenderPos(tf.cursorPos).ToPointFloor()
	h := int(tf.renderVisible.LineHeight + 0.5)
	tf.SetTextInput(image.Rectangle{Min: min, Max: min.Add(image.Pt(max(1, int(tf.CursorWidth.Dots)), h))})
}
