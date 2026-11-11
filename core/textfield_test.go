// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"errors"
	"image"
	"math/rand"
	"testing"
	"time"

	"cogentcore.org/core/base/iox/imagex"
	"cogentcore.org/core/events"
	"cogentcore.org/core/events/key"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/keymap"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/abilities"
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

// TestTextFieldWidths tests that a text field with a long text is as wide
// as its text on one line (its preferred width), within its Max, so that
// it has one line when there is room for it, and that it is narrower than
// its Min when it is given less width, down to the width of its longest
// word, wrapping its text, as text does. (The user agent of the settings
// of the web browser wrapped at its Min of 20ch, over many lines, in a
// window with room for it.)
func TestTextFieldWidths(t *testing.T) {
	const long = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) CogentCore/1.0 Safari/537.36"
	const short = "Mozilla/5.0 (Macintosh; Intel)"
	const words = "the quick brown fox jumps over the lazy dog, again and again and again"
	field := func(width float32, txt string) (tf *TextField, b *Body) {
		b = NewBody()
		fr := NewFrame(b)
		if width > 0 {
			fr.Styler(func(s *styles.Style) { s.Direction = styles.Column; s.Min.X.Dp(width); s.Max.X.Dp(width) })
		}
		tf = NewTextField(fr).SetText(txt)
		return
	}
	lines := func(tf *TextField) float32 {
		return tf.Geom.Size.Actual.Content.Y / tf.lineHeight
	}
	t.Run("short", func(t *testing.T) {
		tf, b := field(0, short)
		b.RenderImage(func() {}, func() {
			assert.InDelta(t, 1, lines(tf), 0.3, "one line")
			assert.Greater(t, tf.Geom.Size.Actual.Total.X, tf.styleMinDots().X, "wider than its Min, for its text")
		})
	})
	t.Run("max", func(t *testing.T) {
		tf, b := field(0, long)
		b.RenderImage(func() {}, func() {
			// as wide as its Max (the window is sized to it), wrapping in it
			// (the width of its lines, which wrap within it)
			assert.InDelta(t, tf.Geom.Size.Max.X, tf.Geom.Size.Actual.Content.X, 8)
			assert.LessOrEqual(t, lines(tf), float32(4.3), "a few lines at its Max")
		})
	})
	t.Run("narrow", func(t *testing.T) {
		tf, b := field(120, words)
		b.RenderImage(func() {}, func() {
			w := tf.Geom.Size.Actual.Content.X
			assert.Less(t, w, tf.styleMinDots().X, "narrower than its Min")
			assert.GreaterOrEqual(t, w, tf.minContentWidth(), "not narrower than its longest word")
			assert.Greater(t, lines(tf), float32(2.5), "wrapped")
		})
	})
	t.Run("form", func(t *testing.T) {
		// as in the settings window, which is sized to its content
		b := NewBody()
		st := &struct{ UserAgent string }{long}
		NewForm(b).SetStruct(st)
		b.RenderImage(func() {}, func() {
			var tf *TextField
			b.WidgetWalkDown(func(cw Widget, cwb *WidgetBase) bool {
				if x, ok := cw.(*TextField); ok {
					tf = x
				}
				return tf == nil
			})
			if assert.NotNil(t, tf) {
				// (a form has no Max for its fields, which fill its width)
				assert.Greater(t, tf.Geom.Size.Actual.Content.X, 1.5*tf.styleMinDots().X, "wider than its Min")
				assert.LessOrEqual(t, lines(tf), float32(4.3), "a few lines")
				// a window sized to it is not as wide as its text on one line
				assert.Greater(t, tf.Geom.Size.prefWidth, float32(800))
				m := units.Ch(contentSizingMeasure)
				m.ToDots(&tf.Styles.UnitContext)
				// (as wide as the measure, within which its text wraps, not as
				// wide as the screen, as it is offered)
				csz := float32(b.Scene.contentSize(image.Pt(4000, 4000)).X)
				assert.Greater(t, csz, m.Dots, "as wide as a readable measure")
				assert.Less(t, csz, m.Dots+200, "within a readable measure")
			}
		})
	})
}

// TestTextFieldWidthsRow tests that text fields in a row (as of an inline
// form) get the widths of their texts on one line when there is room.
func TestTextFieldWidthsRow(t *testing.T) {
	b := NewBody()
	fr := NewFrame(b)
	a := NewTextField(fr).SetText("MacStandard")
	d := NewTextField(fr).SetText("Standard Mac KeyMap")
	b.RenderImage(func() {}, func() {
		assert.InDelta(t, 1, d.Geom.Size.Actual.Content.Y/d.lineHeight, 0.3, "one line")
		assert.Less(t, a.Geom.Size.Actual.Total.X, d.Geom.Size.Actual.Total.X, "the one with the shorter text is narrower")
	})
}

// TestTextFieldSetCursorPos tests that the cursor of a text field is put at
// a position in its text (as for one that replaces another in which the
// user was typing), within its text.
func TestTextFieldSetCursorPos(t *testing.T) {
	b := NewBody()
	tf := NewTextField(b).SetText("hello world")
	b.RenderImage(func() {
		b.AsyncLock()
		defer b.AsyncUnlock()
		tf.SetCursorPos(5)
		assert.Equal(t, 5, tf.CursorPos())
		tf.SetCursorPos(100)
		assert.Equal(t, 11, tf.CursorPos(), "at most its end")
		tf.SetCursorPos(-1)
		assert.Equal(t, 0, tf.CursorPos())
	})
}

// TestTextFieldSlideOverDrag tests that sliding in a text field that has
// the focus (to select its text, as it is edited) has priority over
// dragging the widget that it is in (as a row of a tree table), and that
// the widget is dragged from a text field that does not have the focus.
func TestTextFieldSlideOverDrag(t *testing.T) {
	b := NewBody()
	row := NewFrame(b)
	row.Styler(func(s *styles.Style) { s.SetAbilities(true, abilities.Draggable) })
	dragged := 0
	row.On(events.DragStart, func(e events.Event) {
		dragged++
		row.Scene.Events.DragStart(row, "row", e)
	})
	tf := NewTextField(row).SetText("some text to select")
	slide := func() {
		b.AsyncLock()
		bb := tf.Geom.TotalBBox
		b.AsyncUnlock()
		from := image.Pt(bb.Min.X+8, (bb.Min.Y+bb.Max.Y)/2)
		ev := b.Scene.RenderWindow().SystemWindow.Events()
		ev.MouseButton(events.MouseDown, events.Left, from, 0)
		time.Sleep(200 * time.Millisecond) // (past the time to start a drag)
		for i := 1; i <= 8; i++ {
			ev.MouseMove(from.Add(image.Pt(10*i, 0)))
			time.Sleep(10 * time.Millisecond)
		}
		ev.MouseButton(events.MouseUp, events.Left, from.Add(image.Pt(80, 0)), 0)
		time.Sleep(50 * time.Millisecond)
	}
	unfocus := func() {
		b.AsyncLock()
		b.Scene.Events.focusClear() // (the first text field of a window has it)
		b.AsyncUnlock()
	}
	b.RenderImage(unfocus, slide, func() {
		assert.Equal(t, 1, dragged, "dragged from a text field without the focus")
		b.AsyncLock()
		tf.SetFocus()
		b.AsyncUnlock()
	}, slide, func() {
		assert.Equal(t, 1, dragged, "not dragged from the text field being edited")
		b.AsyncLock()
		defer b.AsyncUnlock()
		assert.True(t, tf.hasSelection(), "its text is selected")
	})
}
