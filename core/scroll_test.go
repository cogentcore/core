// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"testing"

	"cogentcore.org/core/math32"
	"cogentcore.org/core/styles"
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
	b.RenderImage(func() {}, func() {
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
	b1.RenderImage(func() {
		b1.AsyncLock()
		defer b1.AsyncUnlock()
		require.NotNil(t, fr.Scrolls[1])
		b2 := NewBody()
		tree.MoveToParent(fr, b2)
		assert.Same(t, b2.Scene, fr.Scene)
		assert.Same(t, b2.Scene, fr.Scrolls[1].Scene, "the scrollbar is in the scene of its frame")
	})
}
