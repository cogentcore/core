// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"testing"

	"cogentcore.org/core/events/key"
	"cogentcore.org/core/keymap"
	"cogentcore.org/core/styles"
	"github.com/stretchr/testify/assert"
)

// TestFrameFocusOnName tests focusing the children of a frame by typing
// the start of their labels, and going to the next match with tab.
func TestFrameFocusOnName(t *testing.T) {
	b := NewBody()
	fr := NewFrame(b)
	fr.Styler(func(s *styles.Style) { s.Direction = styles.Column })
	apple := NewButton(fr).SetText("Apple")
	NewText(fr).SetText("Avocado") // cannot be focused
	apricot := NewButton(fr).SetText("Apricot")
	banana := NewButton(fr).SetText("Banana")
	AssertRender(t, b, "frame/focus-on-name", Locked(b, func() {
		em := fr.Events()
		SendKey(fr, 'b', key.CodeB)
		assert.Equal(t, Widget(banana), em.focus)
		fr.focusName = "" // as after the timeout
		SendKey(fr, 'a', key.CodeA)
		assert.Equal(t, Widget(apple), em.focus)
		SendKey(fr, 'p', key.CodeP)
		assert.Equal(t, Widget(apple), em.focus, "ap")
		SendKey(fr, 'r', key.CodeR)
		assert.Equal(t, Widget(apricot), em.focus, "apr")
		SendKey(fr, 'z', key.CodeZ)
		assert.Equal(t, "", fr.focusName, "no match")

		SendKey(fr, 'a', key.CodeA)
		assert.Equal(t, Widget(apple), em.focus)
		SendKeyFunction(fr, keymap.FocusNext)
		assert.Equal(t, Widget(apricot), em.focus, "the next match")

		fr.focusName = ""
		SendKeyFunction(fr, keymap.FocusNext)
		assert.Nil(t, fr.focusNameLast, "tab with no name")
		SendKey(fr, 'a', key.CodeA, key.Control)
		assert.Equal(t, "", fr.focusName, "not with modifiers")
		assert.Nil(t, childByLabelCanFocus(fr, "q", nil))
	}))
}
