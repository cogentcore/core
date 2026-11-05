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
)

func TestGridRowSpan(t *testing.T) {
	b := NewBody()
	grid := NewFrame(b)
	grid.Styler(func(s *styles.Style) {
		s.Display = styles.Grid
		s.Columns = 2
		s.Gap.Zero()
		s.Padding.Zero()
	})
	tall := fixedFrame(grid, 50, 100)
	tall.Styler(func(s *styles.Style) { s.RowSpan = 2 })
	top := fixedFrame(grid, 60, 30)
	bottom := fixedFrame(grid, 60, 30)
	next := fixedFrame(grid, 40, 20) // in the row below the spanning one
	layoutGeom(b, func() {
		dp := grid.Styles.UnitContext.Dots(units.UnitDp)
		assert.Equal(t, image.Point{1, 0}, top.Geom.Cell)
		assert.Equal(t, image.Point{1, 1}, bottom.Geom.Cell)
		assert.Equal(t, image.Point{0, 2}, next.Geom.Cell)
		x0, y0 := grid.Geom.Pos.Content.X, grid.Geom.Pos.Content.Y
		// the rows are heightened evenly to fit the cell that spans them
		assert.InDelta(t, y0+50*dp, bottom.Geom.Pos.Total.Y, 2)
		assert.InDelta(t, x0+50*dp, bottom.Geom.Pos.Total.X, 2)
		assert.InDelta(t, y0+100*dp, next.Geom.Pos.Total.Y, 2)
		assert.InDelta(t, x0, next.Geom.Pos.Total.X, 1)
	})
}
