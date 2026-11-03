// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"errors"
	"math/rand"
	"testing"
	"time"

	"cogentcore.org/core/base/iox/imagex"
	"cogentcore.org/core/events"
	"cogentcore.org/core/events/key"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/keymap"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/units"
	"github.com/stretchr/testify/assert"
)

func TestTextField(t *testing.T) {
	b := NewBody()
	NewTextField(b)
	b.AssertRender(t, "text-field/basic")
}

func TestTextFieldPlaceholder(t *testing.T) {
	b := NewBody()
	NewTextField(b).SetPlaceholder("Jane Doe")
	b.AssertRender(t, "text-field/placeholder")
}

func TestTextFieldText(t *testing.T) {
	b := NewBody()
	NewTextField(b).SetText("Hello, world!")
	b.AssertRender(t, "text-field/text")
}

func TestTextFieldOverflow(t *testing.T) {
	b := NewBody()
	NewTextField(b).SetText("This is a long sentence that demonstrates how text field content can overflow onto multiple lines")
	b.AssertRender(t, "text-field/overflow")
}

func TestTextFieldOutlined(t *testing.T) {
	b := NewBody()
	NewTextField(b).SetType(TextFieldOutlined)
	b.AssertRender(t, "text-field/outlined")
}

func TestTextFieldPassword(t *testing.T) {
	b := NewBody()
	NewTextField(b).SetTypePassword().SetText("my password")
	b.AssertRender(t, "text-field/password")
}

func TestTextFieldPasswordClick(t *testing.T) {
	b := NewBody()
	tf := NewTextField(b).SetTypePassword().SetText("my password")
	b.AssertRender(t, "text-field/password-click", func() {
		b.AsyncLock()
		defer b.AsyncUnlock()
		tf.trailingIconButton.Send(events.Click)
	})
}

func TestTextFieldClear(t *testing.T) {
	b := NewBody()
	NewTextField(b).AddClearButton().SetText("Hello, world!")
	b.AssertRender(t, "text-field/clear")
}

func TestTextFieldClearClick(t *testing.T) {
	b := NewBody()
	tf := NewTextField(b).AddClearButton().SetText("Hello, world!")
	b.AssertRender(t, "text-field/clear-click", func() {
		b.AsyncLock()
		defer b.AsyncUnlock()
		tf.trailingIconButton.Send(events.Click)
		assert.Equal(t, "", tf.Text())
	})
}

func TestTextFieldIcons(t *testing.T) {
	b := NewBody()
	clicked := false
	tf := NewTextField(b).SetTrailingIcon(icons.Euro).SetLeadingIcon(icons.OpenInNew, func(e events.Event) {
		clicked = true
	})
	b.AssertRender(t, "text-field/icons", func() {
		b.AsyncLock()
		defer b.AsyncUnlock()
		tf.leadingIconButton.Send(events.Click)
		assert.True(t, clicked)
	})
}

func TestTextFieldValidatorValid(t *testing.T) {
	b := NewBody()
	tf := NewTextField(b).SetText("my secure password")
	tf.SetValidator(func() error {
		if len(tf.Text()) < 12 {
			return errors.New("password must be at least 12 characters")
		}
		return nil
	})
	b.AssertRender(t, "text-field/validator-valid", func() {
		b.AsyncLock()
		defer b.AsyncUnlock()
		tf.SendChange() // trigger validation
	})
}

func TestTextFieldValidatorInvalid(t *testing.T) {
	b := NewBody()
	b.Styler(func(s *styles.Style) {
		s.Min.Set(units.Em(15), units.Em(10))
	})
	tf := NewTextField(b).SetText("my password")
	tf.SetValidator(func() error {
		if len(tf.Text()) < 12 {
			return errors.New("password must be at least 12 characters")
		}
		return nil
	})
	b.AssertRender(t, "text-field/validator-invalid", func() {
		b.AsyncLock()
		defer b.AsyncUnlock()
		tf.SendChange() // trigger validation
	})
}

func TestTextFieldChange(t *testing.T) {
	b := NewBody()
	tf := NewTextField(b)
	text := ""
	tf.OnChange(func(e events.Event) {
		text = tf.Text()
	})
	b.AssertRender(t, "text-field/change", func() {
		b.AsyncLock()
		defer b.AsyncUnlock()
		tf.HandleEvent(events.NewKey(events.KeyChord, 'G', 0, 0))
		tf.HandleEvent(events.NewKey(events.KeyChord, 'o', 0, 0))
		assert.Equal(t, "", text)
		tf.HandleEvent(events.NewKey(events.KeyChord, 0, key.CodeReturnEnter, 0))
		assert.Equal(t, "Go", text)
	})
}

func TestTextFieldInput(t *testing.T) {
	b := NewBody()
	tf := NewTextField(b)
	n := 0
	text := ""
	tf.OnInput(func(e events.Event) {
		n++
		text = tf.Text()
	})
	b.AssertRender(t, "text-field/input", func() {
		b.AsyncLock()
		defer b.AsyncUnlock()
		tf.HandleEvent(events.NewKey(events.KeyChord, 'G', 0, 0))
		assert.Equal(t, 1, n)
		assert.Equal(t, "G", text)
		tf.HandleEvent(events.NewKey(events.KeyChord, 'o', 0, 0))
		assert.Equal(t, 2, n)
		assert.Equal(t, "Go", text)
		tf.HandleEvent(events.NewKey(events.KeyChord, 0, key.CodeReturnEnter, 0))
		assert.Equal(t, 2, n)
		assert.Equal(t, "Go", text)
	})
}

// TestTextFieldCursorArrows tests that the cursor stays on the line of a
// text field while moving back to the start of random texts that are typed
// in, and forward to the end, including within ligatures such as fi.
func TestTextFieldCursorArrows(t *testing.T) {
	if testing.Short() {
		t.Skip("random texts")
	}
	b := NewBody()
	tf := NewTextField(b)
	tf.Styler(func(s *styles.Style) {
		s.Min.X.Em(30)
	})
	chars := []rune("abcdefghijklmnopqrstuvwxyz ABCXYZ fffiflffi 0123 .,:;'\"-_()[]/\\!?@#$%&*+=<>")
	rnd := rand.New(rand.NewSource(1))
	texts := []string{"fix the bug", "off the file", "a  b"}
	for range 20 {
		rs := make([]rune, 1+rnd.Intn(20))
		for i := range rs {
			rs[i] = chars[rnd.Intn(len(chars))]
		}
		texts = append(texts, string(rs))
	}
	arrow := func(code key.Codes) {
		tf.HandleEvent(events.NewKey(events.KeyChord, 0, code, 0))
	}
	var steps []func()
	for _, s := range texts {
		steps = append(steps, func() { // types the text, which is laid out after
			b.AsyncLock()
			defer b.AsyncUnlock()
			tf.SetText("")
			for _, r := range s {
				tf.HandleEvent(events.NewKey(events.KeyChord, r, 0, 0))
			}
		}, func() {
			b.AsyncLock()
			defer b.AsyncUnlock()
			n := len([]rune(s))
			for range n {
				arrow(key.CodeLeftArrow)
			}
			assert.Equal(t, 0, tf.cursorPos, s)
			start := tf.charRenderPos(0)
			lastX := start.X
			for i := range n {
				arrow(key.CodeRightArrow)
				assert.Equal(t, i+1, tf.cursorPos, s)
				pos := tf.charRenderPos(tf.cursorPos)
				assert.InDelta(t, start.Y, pos.Y, 1, "%q: line of the cursor at %d", s, tf.cursorPos)
				assert.GreaterOrEqual(t, pos.X, lastX, "%q: the cursor goes right at %d", s, tf.cursorPos)
				lastX = pos.X
			}
		})
	}
	imagex.UpdateTestImagesFor(t) // the image is not compared
	b.AssertRender(t, "text-field/cursor-arrows", steps...)
}

// TestTextFieldCopy tests that copying the selected text, with the keys,
// clears the selection.
func TestTextFieldCopy(t *testing.T) {
	b := NewBody()
	tf := NewTextField(b).SetText("Hello world")
	b.AssertRender(t, "text-field/copy", func() {
		b.AsyncLock()
		tf.SetFocus()
		tf.selectAll()
		assert.True(t, tf.hasSelection())
		b.AsyncUnlock()
		src := b.Scene.RenderWindow().SystemWindow.Events()
		for ch, kf := range *keymap.ActiveMap {
			if kf != keymap.Copy {
				continue
			}
			r, code, mods, err := ch.Decode()
			assert.NoError(t, err)
			src.Key(events.KeyDown, r, code, mods)
			src.Key(events.KeyUp, r, code, mods)
			break
		}
		time.Sleep(200 * time.Millisecond)
		b.AsyncLock()
		defer b.AsyncUnlock()
		// the clipboard of the test driver does nothing, so it is not checked
		assert.False(t, tf.hasSelection(), "the selection is cleared")
	})
}

func TestTextFieldEdited(t *testing.T) {
	b := NewBody()
	tf := NewTextField(b).SetText("text")
	assert.False(t, tf.Edited())
	tf.editText = []rune("texts")
	tf.textEdited()
	assert.True(t, tf.Edited())
	tf.editDone()
	assert.False(t, tf.Edited(), "applied")
	assert.Equal(t, "texts", tf.Text())
}
