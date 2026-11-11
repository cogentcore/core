// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"testing"

	"cogentcore.org/core/math32"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/units"
	"github.com/stretchr/testify/assert"
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
