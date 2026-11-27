// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"testing"

	"cogentcore.org/core/events"
	"cogentcore.org/core/events/key"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/abilities"
	"cogentcore.org/core/styles/units"
	"cogentcore.org/core/tree"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestScrollNoScrollbarWidth tests that a frame whose scrollbars have no
// width (and are not shown) still scrolls by the wheel, as with
// scrollbar-width: none in CSS.
func TestScrollNoScrollbarWidth(t *testing.T) {
	b := NewBody()
	fr := NewFrame(b)
	fr.Styler(func(s *styles.Style) {
		s.Overflow.Y = styles.OverflowAuto
		s.Max.Y.Dp(100)
		s.Min.X.Dp(100)
		s.ScrollbarWidth.Zero()
	})
	tall := NewFrame(fr)
	tall.Styler(func(s *styles.Style) { s.Min.Set(units.Dp(50), units.Dp(1000)) })
	RenderImage(b, func() {}, func() {
		b.AsyncLock()
		defer b.AsyncUnlock()
		sb := fr.Scrolls[math32.Y]
		if !assert.True(t, fr.HasScroll[math32.Y]) || !assert.NotNil(t, sb) {
			return
		}
		assert.Greater(t, sb.Max, float32(500), "its scroll range")
		assert.True(t, fr.scrollActionDelta(math32.Y, 50), "it scrolls")
		assert.Greater(t, sb.Value, float32(0))
	})
}

// TestScrollbarsMovedScene tests that the scrollbars of a frame that is
// moved into another scene (as the content of a web page, read in one and
// shown in another) are in that scene, in which they are rendered (they
// were not shown).
func TestScrollbarsMovedScene(t *testing.T) {
	b1 := NewBody()
	fr := NewFrame(b1)
	fr.Styler(func(s *styles.Style) {
		s.Overflow.Set(styles.OverflowAuto)
		s.Min.Set(units.Dp(100), units.Dp(100))
		s.Max.Set(units.Dp(100), units.Dp(100))
	})
	tall := NewFrame(fr)
	tall.Styler(func(s *styles.Style) { s.Min.Set(units.Dp(50), units.Dp(500)) })
	RenderImage(b1, func() {
		b1.AsyncLock()
		defer b1.AsyncUnlock()
		require.NotNil(t, fr.Scrolls[1])
		b2 := NewBody()
		tree.MoveToParent(fr, b2)
		assert.Same(t, b2.Scene, fr.Scene)
		assert.Same(t, b2.Scene, fr.Scrolls[1].Scene, "the scrollbar is in the scene of its frame")
	})
}

// TestScrollKeys tests that a frame that has the focus itself, as a panel
// of text to read, scrolls with the arrow keys, PageUp and PageDown, and
// Home and End, and that it does not when it does not have the focus.
func TestScrollKeys(t *testing.T) {
	b := NewBody()
	fr := NewFrame(b)
	fr.Styler(func(s *styles.Style) {
		s.Overflow.Y = styles.OverflowAuto
		s.Max.Y.Dp(100)
		s.Min.X.Dp(100)
		s.SetAbilities(true, abilities.Focusable)
	})
	tall := NewFrame(fr)
	tall.Styler(func(s *styles.Style) { s.Min.Set(units.Dp(50), units.Dp(1000)) })
	press := func(code key.Codes) {
		b.Scene.Events.handleEvent(events.NewKey(events.KeyChord, 0, code, 0))
	}
	RenderImage(b, func() {}, func() {
		b.AsyncLock()
		defer b.AsyncUnlock()
		sb := fr.Scrolls[math32.Y]
		require.NotNil(t, sb)
		sb.SetValue(0)
		press(key.CodeDownArrow)
		assert.Zero(t, sb.Value, "not without the focus")
		fr.SetFocusQuiet()
		press(key.CodeDownArrow)
		line := sb.Value
		assert.Greater(t, line, float32(0), "a line")
		press(key.CodePageDown)
		assert.Greater(t, sb.Value-line, line, "a page, more than a line")
		press(key.CodeUpArrow)
		press(key.CodeEnd)
		end := sb.Value
		assert.Greater(t, end, 5*line, "the end")
		press(key.CodePageDown)
		assert.Equal(t, end, sb.Value, "no further than the end")
		press(key.CodeHome)
		assert.Zero(t, sb.Value, "the start")
	})
}
