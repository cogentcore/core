// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"testing"

	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/units"
	"github.com/stretchr/testify/assert"
)

// TestAlignItemsWithGrowingSibling tests that a child is aligned on the
// cross axis within the size of its line, which a sibling that grows can
// make larger than the children (as the sign up form of github.com, next
// to its heading, in a column whose items are centered).
func TestAlignItemsWithGrowingSibling(t *testing.T) {
	b := NewBody()
	fr := NewFrame(b)
	fr.Styler(func(s *styles.Style) {
		s.Direction = styles.Column
		s.Align.Items = styles.Center
		s.Min.X.Dp(600)
	})
	wide := NewFrame(fr)
	wide.Styler(func(s *styles.Style) { s.Grow.Set(1, 0); s.Min.Y.Dp(10) })
	small := NewFrame(fr)
	small.Styler(func(s *styles.Style) { s.Min.Set(units.Dp(100), units.Dp(10)) })
	b.RenderImage(func() {
		wpos, wsz := wide.Geom.Pos.Total.X, wide.Geom.Size.Actual.Total.X
		spos, ssz := small.Geom.Pos.Total.X, small.Geom.Size.Actual.Total.X
		assert.Greater(t, wsz, 2*ssz)
		assert.InDelta(t, wpos+wsz/2, spos+ssz/2, 1, "centered within the width of the line")
	})
}
