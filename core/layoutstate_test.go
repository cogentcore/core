// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"image"
	"testing"

	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/units"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestLayoutRowHeightColWidth tests the heights of the rows and widths of
// the columns of a grid, of rows that wrap, and of columns that wrap.
func TestLayoutRowHeightColWidth(t *testing.T) {
	b := NewBody()
	grid := NewFrame(b)
	grid.Styler(func(s *styles.Style) {
		s.Display = styles.Grid
		s.Columns = 2
		s.Gap.Zero()
	})
	for _, h := range []float32{10, 20, 30, 40} { // rows of 20 and 40
		fixedFrame(grid, h, h)
	}
	rows := NewFrame(b)
	rows.Styler(func(s *styles.Style) {
		s.Wrap = true
		s.Min.X.Dp(110) // (room for 2, not 3, with rounding)
		s.Max.X.Dp(110)
		s.Gap.Zero()
	})
	for _, h := range []float32{10, 30, 50, 20} { // 50 wide: 2 per row
		fixedFrame(rows, 50, h)
	}
	cols := NewFrame(b)
	cols.Styler(func(s *styles.Style) {
		s.Direction = styles.Column
		s.Wrap = true
		s.Min.Y.Dp(110)
		s.Max.Y.Dp(110)
		s.Gap.Zero()
	})
	for _, w := range []float32{10, 30, 50, 20} { // 50 high: 2 per column
		fixedFrame(cols, w, 50)
	}
	layoutAtWidth(b, 800, 600, func(dp float32) {
		near := func(want, got float32, err error, msg string) {
			t.Helper()
			require.NoError(t, err, msg)
			assert.InDelta(t, want*dp, got, 1, msg)
		}
		gl := &grid.layout
		h, err := gl.rowHeight(0, 0)
		near(20, h, err, "grid row 0")
		h, err = gl.rowHeight(1, 0)
		near(40, h, err, "grid row 1")
		w, err := gl.colWidth(0, 1)
		near(40, w, err, "grid column 1")
		_, err = gl.rowHeight(2, 0)
		assert.Error(t, err)
		_, err = gl.colWidth(0, 2)
		assert.Error(t, err)

		rl := &rows.layout
		require.Len(t, rl.Wraps, 2, "two rows")
		h, err = rl.rowHeight(0, 0)
		near(30, h, err, "wrapped row 0")
		h, err = rl.rowHeight(1, 1)
		near(50, h, err, "wrapped row 1")
		w, err = rl.colWidth(1, 0)
		near(50, w, err, "a column of wrapped row 1")
		_, err = rl.rowHeight(2, 0)
		assert.Error(t, err)
		_, err = rl.rowHeight(0, 2)
		assert.Error(t, err)
		_, err = rl.colWidth(2, 0)
		assert.Error(t, err)
		_, err = rl.colWidth(0, 2)
		assert.Error(t, err)

		cl := &cols.layout
		require.Len(t, cl.Wraps, 2, "two columns")
		w, err = cl.colWidth(0, 0)
		near(30, w, err, "wrapped column 0")
		w, err = cl.colWidth(0, 1)
		near(50, w, err, "wrapped column 1")
		h, err = cl.rowHeight(1, 1)
		near(50, h, err, "a row of wrapped column 1")
		_, err = cl.rowHeight(0, 2)
		assert.Error(t, err)
		_, err = cl.rowHeight(2, 0)
		assert.Error(t, err)
		_, err = cl.colWidth(0, 2)
		assert.Error(t, err)
		_, err = cl.colWidth(2, 0)
		assert.Error(t, err)

		assert.NotEmpty(t, gl.String())
		assert.NotEmpty(t, rl.String())
	})
}

// TestLayoutFlexBasis tests that the children of a flex row with a flex
// basis start from it as their preferred width, limited by their max.
func TestLayoutFlexBasis(t *testing.T) {
	b := NewBody()
	row := NewFrame(b)
	row.Styler(func(s *styles.Style) {
		s.Display = styles.Flex
		s.Direction = styles.Row
		s.Gap.Zero()
	})
	item := func(maxX float32) *Frame {
		fr := NewFrame(row)
		fr.Styler(func(s *styles.Style) {
			s.Min.Set(units.Dp(10), units.Dp(10))
			s.FlexBasis.Dp(100)
			if maxX > 0 {
				s.Max.X.Dp(maxX)
			}
		})
		return fr
	}
	a, c := item(0), item(60)
	layoutAtWidth(b, 800, 600, func(dp float32) {
		assert.InDelta(t, 100*dp, a.Geom.Size.preferred.X, 2, "the flex basis")
		assert.InDelta(t, 60*dp, c.Geom.Size.preferred.X, 2, "limited by the max")
	})
}

// TestLayoutRowBaseline tests that a row aligned on baselines within
// another one has the baseline of its first line, so that its text lines
// up with the text beside it.
func TestLayoutRowBaseline(t *testing.T) {
	b := NewBody()
	outer := NewFrame(b)
	outer.Styler(func(s *styles.Style) {
		s.Direction = styles.Row
		s.Align.Items = styles.Baseline
	})
	tx := NewText(outer).SetText("Outer")
	inner := NewFrame(outer)
	inner.Styler(func(s *styles.Style) {
		s.Direction = styles.Row
		s.Align.Items = styles.Baseline
	})
	fixedFrame(inner, 10, 60) // a tall item without text
	itx := NewText(inner).SetText("Inner")
	layoutAtWidth(b, 800, 600, func(dp float32) {
		base := func(wb *WidgetBase) float32 {
			bl, ok := wb.firstBaseline()
			require.True(t, ok)
			return wb.Geom.Pos.Total.Y + bl
		}
		ib, ok := inner.rowBaseline()
		require.True(t, ok)
		assert.Greater(t, ib, float32(0))
		assert.InDelta(t, base(tx.AsWidget()), base(itx.AsWidget()), 2, "the texts line up")
		_, ok = b.rowBaseline()
		assert.False(t, ok, "a column")
	})
}

// TestLayoutStacked tests that the children of a stacked frame all get its
// size, or only the top one with LayoutStackTopOnly, and the source and
// destination boxes of direct rendering.
func TestLayoutStacked(t *testing.T) {
	for _, topOnly := range []bool{false, true} {
		b := NewBody()
		st := NewFrame(b)
		st.Styler(func(s *styles.Style) {
			s.Display = styles.Stacked
			s.Min.Set(units.Dp(80), units.Dp(60))
		})
		st.LayoutStackTopOnly = topOnly
		a := fixedFrame(st, 10, 10)
		c := fixedFrame(st, 20, 20)
		st.StackTop = 1
		layoutAtWidth(b, 800, 600, func(dp float32) {
			csz := st.Geom.Size.Actual.Content
			assert.Equal(t, csz, c.Geom.Size.Alloc.Total, "the top gets the size")
			if !topOnly {
				assert.Equal(t, csz, a.Geom.Size.Alloc.Total, "all get the size")
			}
			dst, src, empty := c.DirectRenderDrawBBoxes(image.Rect(0, 0, 20, 20))
			assert.False(t, empty)
			assert.Equal(t, c.Geom.TotalBBox.Add(b.Scene.SceneGeom.Pos), dst)
			assert.Equal(t, image.Rect(0, 0, 20, 20), src)
			_, _, empty = c.DirectRenderDrawBBoxes(image.Rectangle{})
			assert.True(t, empty)
		})
	}
}
