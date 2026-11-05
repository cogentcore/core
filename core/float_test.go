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

func TestFloat(t *testing.T) {
	b := NewBody()
	col := NewFrame(b)
	col.Styler(func(s *styles.Style) {
		s.Direction = styles.Column
		s.Min.X.Dp(400)
		s.Gap.Zero()
		s.Padding.Zero()
	})
	block := func(h float32) *Frame {
		fr := NewFrame(col)
		fr.Styler(func(s *styles.Style) {
			s.Min.Y.Dp(h)
			s.Grow.X = 1
		})
		return fr
	}
	first := block(30)
	float := fixedFrame(col, 100, 100)
	float.Styler(func(s *styles.Style) { s.Float = styles.FloatRight })
	beside := block(40)
	cleared := block(20)
	cleared.Styler(func(s *styles.Style) { s.Clear = styles.ClearBoth })
	layoutGeom(b, func() {
		dp := col.Styles.UnitContext.Dots(units.UnitDp)
		x0, y0 := col.Geom.Pos.Content.X, col.Geom.Pos.Content.Y
		w := col.Geom.Size.Actual.Content.X
		// the float is at the right, where it is in the column
		assert.InDelta(t, x0+w-100*dp, float.Geom.Pos.Total.X, 1.5)
		assert.InDelta(t, first.Geom.Pos.Total.Y+first.Geom.Size.Actual.Total.Y, float.Geom.Pos.Total.Y, 1.5)
		// the block after it is beside it, and the one before is not narrowed
		assert.InDelta(t, w, first.Geom.Size.Actual.Total.X, 1.5)
		assert.InDelta(t, w-100*dp, beside.Geom.Size.Actual.Total.X, 1.5)
		assert.InDelta(t, float.Geom.Pos.Total.Y, beside.Geom.Pos.Total.Y, 1.5)
		// the cleared block is below the float, at the full width
		assert.InDelta(t, float.Geom.Pos.Total.Y+100*dp, cleared.Geom.Pos.Total.Y, 1.5)
		assert.InDelta(t, w, cleared.Geom.Size.Actual.Total.X, 1.5)
		// the column is as tall as its content with the float
		assert.InDelta(t, cleared.Geom.Pos.Total.Y+20*dp-y0, col.Geom.Size.Actual.Content.Y, 2)
		_ = x0
	})
}

// TestFloatClearFlow tests that a float that clears the floats before it
// is below them, but the widgets after it are still beside them.
func TestFloatClearFlow(t *testing.T) {
	b := NewBody()
	col := NewFrame(b)
	col.Styler(func(s *styles.Style) {
		s.Direction = styles.Column
		s.Min.X.Dp(400)
		s.Gap.Zero()
		s.Padding.Zero()
	})
	first := fixedFrame(col, 100, 100)
	first.Styler(func(s *styles.Style) { s.Float = styles.FloatRight })
	second := fixedFrame(col, 100, 50)
	second.Styler(func(s *styles.Style) { s.Float = styles.FloatRight; s.Clear = styles.ClearRight })
	beside := NewFrame(col)
	beside.Styler(func(s *styles.Style) { s.Min.Y.Dp(20); s.Grow.X = 1 })
	layoutGeom(b, func() {
		dp := col.Styles.UnitContext.Dots(units.UnitDp)
		y0 := col.Geom.Pos.Content.Y
		assert.InDelta(t, y0+100*dp, second.Geom.Pos.Total.Y, 2)
		assert.InDelta(t, y0, beside.Geom.Pos.Total.Y, 2)
		assert.InDelta(t, 300*dp, beside.Geom.Size.Alloc.Total.X, 2)
	})
}

// TestFloatNotHigher tests that a float is not placed higher than the
// floats before it, even where there is room above.
func TestFloatNotHigher(t *testing.T) {
	b := NewBody()
	col := NewFrame(b)
	col.Styler(func(s *styles.Style) {
		s.Direction = styles.Column
		s.Min.X.Dp(300)
		s.Gap.Zero()
		s.Padding.Zero()
	})
	tall := fixedFrame(col, 100, 200)
	tall.Styler(func(s *styles.Style) { s.Float = styles.FloatLeft })
	short := fixedFrame(col, 100, 50)
	short.Styler(func(s *styles.Style) { s.Float = styles.FloatLeft })
	cleared := fixedFrame(col, 100, 50) // below both, at the left
	cleared.Styler(func(s *styles.Style) { s.Float = styles.FloatLeft; s.Clear = styles.ClearBoth })
	next := fixedFrame(col, 100, 50) // beside it, not in the room below the short one
	next.Styler(func(s *styles.Style) { s.Float = styles.FloatLeft })
	layoutGeom(b, func() {
		dp := col.Styles.UnitContext.Dots(units.UnitDp)
		y0 := col.Geom.Pos.Content.Y
		assert.InDelta(t, y0+200*dp, cleared.Geom.Pos.Total.Y, 2)
		assert.InDelta(t, y0+200*dp, next.Geom.Pos.Total.Y, 2)
	})
}
