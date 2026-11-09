// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"image"
	"testing"

	"cogentcore.org/core/math32"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/units"
	"github.com/stretchr/testify/assert"
)

// layoutGeom lays out the body and calls fun with the layout done.
func layoutGeom(b *Body, fun func()) {
	b.RenderImage(fun)
}

// fixedFrame returns a new frame of the given size in dp.
func fixedFrame(parent Widget, w, h float32) *Frame {
	fr := NewFrame(parent)
	fr.Styler(func(s *styles.Style) {
		s.Min.Set(units.Dp(w), units.Dp(h))
		s.Max.Set(units.Dp(w), units.Dp(h))
		s.Background = nil
	})
	return fr
}

func TestLayoutJustifySpaceBetween(t *testing.T) {
	for _, jc := range []styles.Aligns{styles.SpaceBetween, styles.SpaceAround, styles.SpaceEvenly} {
		b := NewBody()
		row := NewFrame(b)
		row.Styler(func(s *styles.Style) {
			s.Min.X.Dp(400)
			s.Gap.Zero()
			s.Justify.Content = jc
		})
		kids := []*Frame{fixedFrame(row, 50, 20), fixedFrame(row, 50, 20), fixedFrame(row, 50, 20)}
		layoutGeom(b, func() {
			x0 := row.Geom.Pos.Content.X
			w := row.Geom.Size.Actual.Content.X
			extra := w - 3*kids[0].Geom.Size.Actual.Total.X
			var want []float32
			switch jc {
			case styles.SpaceBetween:
				want = []float32{0, extra / 2, extra}
			case styles.SpaceAround:
				sp := extra / 3
				want = []float32{sp / 2, 1.5 * sp, 2.5 * sp}
			case styles.SpaceEvenly:
				sp := extra / 4
				want = []float32{sp, 2 * sp, 3 * sp}
			}
			for i, k := range kids {
				got := k.Geom.Pos.Total.X - x0 - float32(i)*k.Geom.Size.Actual.Total.X
				assert.InDelta(t, want[i], got, 1.5, "%v child %d", jc, i)
			}
		})
	}
}

func TestLayoutGridColSpan(t *testing.T) {
	b := NewBody()
	grid := NewFrame(b)
	grid.Styler(func(s *styles.Style) {
		s.Display = styles.Grid
		s.Columns = 3
		s.Gap.Set(units.Dp(4))
	})
	var kids []*Frame
	for range 3 {
		kids = append(kids, fixedFrame(grid, 60, 20))
	}
	span := fixedFrame(grid, 0, 20) // spans the first two columns
	span.Styler(func(s *styles.Style) {
		s.ColSpan = 2
		s.Max.X.Zero()
		s.Grow.X = 1
	})
	last := fixedFrame(grid, 60, 20)
	// a wide spanning cell in the last row widens its columns
	wide := fixedFrame(grid, 300, 20)
	wide.Styler(func(s *styles.Style) { s.ColSpan = 3 })
	layoutGeom(b, func() {
		assert.Equal(t, image.Point{0, 1}, span.Geom.Cell)
		assert.Equal(t, image.Point{2, 1}, last.Geom.Cell)
		assert.Equal(t, image.Point{0, 2}, wide.Geom.Cell)
		assert.Equal(t, image.Point{3, 3}, grid.layout.Shape)
		// the cell after the span is in the column of the third cell of the first row
		assert.InDelta(t, kids[2].Geom.Pos.Total.X, last.Geom.Pos.Total.X, 1)
		// the span covers the first two columns and the gap between them
		assert.InDelta(t, kids[1].Geom.Pos.Total.X+kids[1].Geom.Size.Alloc.Total.X-kids[0].Geom.Pos.Total.X,
			span.Geom.Size.Alloc.Total.X, 1.5)
		total := kids[2].Geom.Pos.Total.X + kids[2].Geom.Size.Alloc.Total.X - kids[0].Geom.Pos.Total.X
		assert.GreaterOrEqual(t, total+1, wide.Geom.Size.Actual.Total.X)
	})
}

func TestLayoutGrowMaxAlign(t *testing.T) {
	b := NewBody()
	col := NewFrame(b)
	col.Styler(func(s *styles.Style) {
		s.Direction = styles.Column
		s.Min.X.Dp(400)
		s.Grow.Set(0, 0)
	})
	fr := NewFrame(col)
	fr.Styler(func(s *styles.Style) {
		s.Min.Y.Dp(20)
		s.Max.X.Dp(200)
		s.Grow.Set(1, 0)
		s.Align.Self = styles.Center
	})
	layoutGeom(b, func() {
		// it grows to its max, and is centered in the space it would have had
		w := fr.Geom.Size.Actual.Total.X
		assert.InDelta(t, fr.Styles.Max.X.Dots, w, 1)
		off := fr.Geom.Pos.Total.X - col.Geom.Pos.Content.X
		assert.InDelta(t, (col.Geom.Size.Actual.Content.X-w)/2, off, 1.5)
	})
}

func TestLayoutImagePercentWidth(t *testing.T) {
	b := NewBody()
	b.Styler(func(s *styles.Style) { s.Min.X.Dp(400) })
	im := NewImage(b)
	im.SetImage(image.NewRGBA(image.Rect(0, 0, 200, 100)))
	im.Styler(func(s *styles.Style) {
		s.Min.X.Pw(50)
	})
	layoutGeom(b, func() {
		sz := im.Geom.Size.Actual.Content
		assert.Greater(t, sz.X, float32(0))
		assert.InDelta(t, sz.X/2, sz.Y, 1, "aspect ratio of %v", sz)
		_ = math32.Vector2{}
	})
}

// TestLayoutAlignBaseline tests a row aligned on baselines: texts of
// different sizes have the same baseline, as a group at the end of the
// line, and a frame without text is at the end of the line.
func TestLayoutAlignBaseline(t *testing.T) {
	b := NewBody()
	row := NewFrame(b)
	row.Styler(func(s *styles.Style) {
		s.Direction = styles.Row
		s.Align.Items = styles.Baseline
		s.Padding.Zero()
		s.Gap.Zero()
	})
	big := NewText(row).SetText("Big")
	big.Styler(func(s *styles.Style) { s.Font.Size = units.Dp(40) })
	small := NewText(row).SetText("small")
	small.Styler(func(s *styles.Style) { s.Font.Size = units.Dp(12) })
	box := fixedFrame(row, 10, 10)
	tall := fixedFrame(row, 10, 80) // the line is taller than the texts
	layoutGeom(b, func() {
		bigBase, ok := big.firstBaseline()
		assert.True(t, ok)
		smallBase, ok := small.firstBaseline()
		assert.True(t, ok)
		_, ok = box.firstBaseline()
		assert.False(t, ok, "no text")
		assert.InDelta(t, big.Geom.Pos.Total.Y+bigBase, small.Geom.Pos.Total.Y+smallBase, 1, "the same baseline")
		lineEnd := tall.Geom.Pos.Total.Y + tall.Geom.Size.Actual.Total.Y
		assert.InDelta(t, lineEnd, big.Geom.Pos.Total.Y+big.Geom.Size.Actual.Total.Y, 1, "the texts are at the end of the line")
		assert.InDelta(t, lineEnd, box.Geom.Pos.Total.Y+box.Geom.Size.Actual.Total.Y, 1, "a frame without text is at the end")
	})
}
