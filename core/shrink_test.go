// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"strings"
	"testing"

	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/units"
	"github.com/stretchr/testify/assert"
)

// wrapText is text that is much wider than 300 dp on one line.
var wrapText = strings.Repeat("Some words that wrap. ", 12)

// shrinkTest makes a frame 300 dp wide, styled by the given function,
// and calls add to add its children before laying it out.
func shrinkTest(t *testing.T, style func(s *styles.Style), add func(fr *Frame), check func(fr *Frame, dp float32)) {
	b := NewBody()
	fr := NewFrame(b)
	fr.Styler(func(s *styles.Style) {
		s.Min.X.Dp(300)
		s.Max.X.Dp(300)
		s.Gap.Zero()
		s.Padding.Zero()
		style(s)
	})
	add(fr)
	layoutGeom(b, func() {
		check(fr, fr.Styles.UnitContext.Dots(units.UnitDp))
	})
}

// shrinkingText adds text that shrinks to the given frame.
func shrinkingText(fr *Frame, txt string) *Text {
	tx := NewText(fr).SetText(txt)
	tx.Styler(func(s *styles.Style) { s.Shrink.X = 1 })
	return tx
}

func TestShrinkRow(t *testing.T) {
	var a, c *Text
	shrinkTest(t, func(s *styles.Style) {}, func(fr *Frame) {
		a = shrinkingText(fr, wrapText)
		c = shrinkingText(fr, wrapText[:len(wrapText)/2])
	}, func(fr *Frame, dp float32) {
		wa, wc := a.Geom.Size.Alloc.Total.X, c.Geom.Size.Alloc.Total.X
		assert.LessOrEqual(t, wa+wc, 300*dp+1)
		// in proportion to their widths
		assert.Greater(t, wa, wc)
		// and the text wraps in them
		assert.LessOrEqual(t, a.Geom.Size.Actual.Total.X, wa+1)
		assert.Greater(t, a.Geom.Size.Actual.Total.Y, c.Geom.Size.Actual.Total.Y/2)
	})
}

func TestShrinkGrid(t *testing.T) {
	var cells []*Text
	shrinkTest(t, func(s *styles.Style) {
		s.Display = styles.Grid
		s.Columns = 2
	}, func(fr *Frame) {
		for range 4 {
			cells = append(cells, shrinkingText(fr, wrapText))
		}
	}, func(fr *Frame, dp float32) {
		assert.LessOrEqual(t, fr.Geom.Size.Actual.Total.X, 300*dp+1)
		w0, w1 := cells[0].Geom.Size.Alloc.Total.X, cells[1].Geom.Size.Alloc.Total.X
		assert.LessOrEqual(t, w0+w1, 300*dp+1)
		assert.InDelta(t, w0, cells[2].Geom.Size.Alloc.Total.X, 1) // columns are consistent
	})
}

func TestShrinkColumn(t *testing.T) {
	var inner *Frame
	var fixed *Frame
	shrinkTest(t, func(s *styles.Style) { s.Direction = styles.Column }, func(fr *Frame) {
		// a frame that does not grow, but shrinks, with wide text in it
		inner = NewFrame(fr)
		inner.Styler(func(s *styles.Style) { s.Shrink.X = 1; s.Direction = styles.Column })
		NewText(inner).SetText(wrapText).Styler(func(s *styles.Style) { s.Shrink.X = 1 })
		// a frame that does not shrink keeps its size
		fixed = fixedFrame(fr, 400, 10)
	}, func(fr *Frame, dp float32) {
		assert.LessOrEqual(t, inner.Geom.Size.Actual.Total.X, 300*dp+1)
		assert.InDelta(t, 400*dp, fixed.Geom.Size.Actual.Total.X, 1)
	})
}

func TestShrinkWrapRow(t *testing.T) {
	var row *Frame
	shrinkTest(t, func(s *styles.Style) { s.Direction = styles.Column }, func(fr *Frame) {
		// a shrinking wrapped row (that does not grow) is as wide as
		// its content on one line, up to the width of the column
		row = NewFrame(fr)
		row.Styler(func(s *styles.Style) { s.Wrap = true; s.Shrink.X = 1; s.Gap.Zero() })
		for range 3 {
			fixedFrame(row, 50, 10)
		}
	}, func(fr *Frame, dp float32) {
		assert.InDelta(t, 150*dp, row.Geom.Size.Actual.Total.X, 2)
		assert.InDelta(t, 10*dp, row.Geom.Size.Actual.Total.Y, 1)
	})
	shrinkTest(t, func(s *styles.Style) { s.Direction = styles.Column }, func(fr *Frame) {
		row = NewFrame(fr)
		row.Styler(func(s *styles.Style) { s.Wrap = true; s.Shrink.X = 1; s.Gap.Zero() })
		for range 8 {
			fixedFrame(row, 50, 10)
		}
	}, func(fr *Frame, dp float32) {
		assert.LessOrEqual(t, row.Geom.Size.Actual.Total.X, 300*dp+1)
		assert.InDelta(t, 20*dp, row.Geom.Size.Actual.Total.Y, 2) // two lines
	})
}
