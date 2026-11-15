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

// TestTableInsertDeleteSearch tests inserting, deleting, selecting by a
// field value, and searching the rows of a table.
func TestTableInsertDeleteSearch(t *testing.T) {
	b := NewBody()
	langs := []language{{"Go", 10}, {"Python", 5}, {"C", 3}}
	tb := NewTable(b).SetSlice(&langs)
	RenderImage(b, Locked(b, func() {
		tb.NewAt(1)
		assert.Equal(t, []language{{"Go", 10}, {}, {"Python", 5}, {"C", 3}}, langs)
		assert.Equal(t, 1, tb.SelectedIndex)
		tb.NewAt(-1)
		assert.Len(t, langs, 5, "-1 appends")
		tb.DeleteAt(4)
		tb.DeleteAt(1)
		tb.DeleteAt(9)
		assert.Equal(t, []language{{"Go", 10}, {"Python", 5}, {"C", 3}}, langs)

		assert.True(t, tb.selectFieldValue("Name", "C"))
		assert.Equal(t, 2, tb.SelectedIndex)
		assert.False(t, tb.selectFieldValue("Name", "Rust"))
		assert.False(t, tb.selectFieldValue("NoField", "C"))

		res := SearchWidgets(b, "o", IgnoreCase)
		assert.Equal(t, 2, res.Len(), "Go, Python")
		r, _ := res.AtIndex(1)
		assert.Equal(t, 2, r.Matches[1].Region.Start.Line, "the line of the name of Python (row 1, field 0, of 2 fields)")
		SearchHighlight(b, res)
		SearchSelect(res, 1, SelectSet)
		SearchHighlight(b, nil)
		assert.Nil(t, tb.TextRunes())
	}))
}

// TestScrollDims tests scrolling a frame to the start, end, and center
// of its content, and to positions.
func TestScrollDims(t *testing.T) {
	b := NewBody()
	b.Styler(func(s *styles.Style) {
		s.Min.Set(units.Dp(200), units.Dp(200))
		s.Max.Set(units.Dp(200), units.Dp(200))
	})
	fr := scrollFrame(b)
	for range 20 {
		f := NewFrame(fr)
		f.Styler(func(s *styles.Style) {
			s.Min.Set(units.Dp(100), units.Dp(50))
		})
	}
	y := math32.Y
	RenderImage(b, Locked(b, func() {
		assert.True(t, fr.HasScroll[y])
		assert.False(t, fr.HasScroll[math32.X])
		// (note: the initial scroll value is 0.5, not 0)
		assert.True(t, fr.ScrollDimToContentStart(y))
		assert.True(t, fr.IsDimAtContentStart(y))
		assert.False(t, fr.IsDimAtContentEnd(y))
		assert.True(t, fr.ScrollDimToContentEnd(y))
		assert.True(t, fr.IsDimAtContentEnd(y))
		assert.False(t, fr.IsDimAtContentStart(y))
		assert.True(t, fr.ScrollDimToContentStart(y))
		assert.True(t, fr.IsDimAtContentStart(y))

		cmin, cmax := fr.Geom.contentRangeDim(y)
		assert.False(t, fr.ScrollDimToStart(y, int(cmin)), "already at the start")
		assert.True(t, fr.ScrollDimToStart(y, int(cmin)+100))
		assert.InDelta(t, 100, fr.Scrolls[y].Value, 1)
		assert.True(t, fr.ScrollDimToEnd(y, int(cmax)+50))
		assert.InDelta(t, 150, fr.Scrolls[y].Value, 1)
		mid := 0.5 * (cmin + cmax)
		assert.True(t, fr.ScrollDimToCenter(y, int(mid)-150))
		assert.InDelta(t, 0, fr.Scrolls[y].Value, 1, "clamped at the start")

		x := math32.X
		assert.False(t, fr.ScrollDimToContentEnd(x), "no scrolling in x")
		assert.False(t, fr.IsDimAtContentStart(x))
		assert.False(t, fr.ScrollDimToCenter(x, 10))
	}))
}
