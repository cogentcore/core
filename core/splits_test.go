// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"strconv"
	"testing"

	"cogentcore.org/core/colors"
	"cogentcore.org/core/styles"
)

func makeSplits(n, w, h int) (*Body, *Splits) {
	b := NewBody()
	b.Styler(func(s *styles.Style) {
		s.Min.X.Em(float32(w))
		s.Min.Y.Em(float32(h))
	})
	sp := NewSplits(b)
	for i := range n {
		f := NewFrame(sp)
		f.Styler(func(s *styles.Style) {
			s.Background = colors.Scheme.Select.Container
		})
		NewText(f).SetText(strconv.Itoa(i))
	}
	return b, sp
}

func TestSplitsRow2(t *testing.T) {
	b, _ := makeSplits(2, 40, 20)
	b.AssertRender(t, "splits/row-2")
}

func TestSplitsColumn2(t *testing.T) {
	b, _ := makeSplits(2, 20, 40)
	b.AssertRender(t, "splits/column-2")
}

func TestSplitsRow4(t *testing.T) {
	b, _ := makeSplits(4, 40, 20)
	b.AssertRender(t, "splits/row-4")
}

func TestSplitsColumn4(t *testing.T) {
	b, _ := makeSplits(4, 20, 40)
	b.AssertRender(t, "splits/column-4")
}

func TestSplitsRow2Set(t *testing.T) {
	b, sp := makeSplits(2, 40, 20)
	sp.SetSplits(0.2, 0.8)
	b.AssertRender(t, "splits/row-2-set")
}

func TestSplitsRow2Column(t *testing.T) {
	b, sp := makeSplits(2, 40, 20)
	sp.Styler(func(s *styles.Style) {
		s.Direction = styles.Column
	})
	b.AssertRender(t, "splits/row-2-column")
}

func TestSplitsColumn2Row(t *testing.T) {
	b, sp := makeSplits(2, 20, 40)
	sp.Styler(func(s *styles.Style) {
		s.Direction = styles.Row
	})
	b.AssertRender(t, "splits/column-2-row")
}

// For https://github.com/cogentcore/core/issues/850
func TestMixedVerticalSplits(t *testing.T) {
	b := NewBody()
	txt := "This is a long sentence that I wrote for the purpose of testing vertical splits behavior"
	NewText(b).SetText(txt)
	sp := NewSplits(b)
	sp.Styler(func(s *styles.Style) {
		s.Direction = styles.Column
	})
	NewText(sp).SetText(txt)
	NewText(sp).SetText(txt)
	NewText(b).SetText(txt)
	b.AssertRender(t, "splits/mixed-vertical")
}

func TestSplitsTilesRowSpanFirst(t *testing.T) {
	b, sp := makeSplits(4, 40, 20)
	sp.SetTiles(TileSpan, TileFirstLong)
	b.AssertRender(t, "splits/tiles-row-span-first")
}

func TestSplitsTilesRowSpanSecond(t *testing.T) {
	b, sp := makeSplits(4, 40, 20)
	sp.SetTiles(TileSpan, TileSecondLong)
	b.AssertRender(t, "splits/tiles-row-span-second")
}

func TestSplitsTilesColumnSpanFirst(t *testing.T) {
	b, sp := makeSplits(4, 20, 40)
	sp.SetTiles(TileSpan, TileFirstLong)
	b.AssertRender(t, "splits/tiles-column-span-first")
}

func TestSplitsTilesColumnSpanSecond(t *testing.T) {
	b, sp := makeSplits(4, 20, 40)
	sp.SetTiles(TileSpan, TileSecondLong)
	b.AssertRender(t, "splits/tiles-column-span-second")
}

func TestSplitsTilesRowSplitSpanSplit(t *testing.T) {
	b, sp := makeSplits(5, 40, 20)
	sp.SetTiles(TileSplit, TileSpan, TileSplit)
	b.AssertRender(t, "splits/tiles-row-split-span-split")
}

func TestSplitsTilesRowSpanFirstSet(t *testing.T) {
	b, sp := makeSplits(4, 40, 20)
	sp.SetTiles(TileSpan, TileFirstLong)
	sp.SetSplits(0.2, 0.8, 0.4, 0.6)
	b.AssertRender(t, "splits/tiles-row-span-first-set")
}

func TestSplitsTilesRowSpanSecondSet(t *testing.T) {
	b, sp := makeSplits(4, 40, 20)
	sp.SetTiles(TileSpan, TileSecondLong)
	sp.SetSplits(0.2, 0.8, 0.2, 0.3)
	b.AssertRender(t, "splits/tiles-row-span-second-set")
}

func TestSplitsTilesRowSplitSpanSplitSet(t *testing.T) {
	b, sp := makeSplits(5, 40, 20)
	sp.SetTiles(TileSplit, TileSpan, TileSplit)
	sp.SetSplits(0.2, 0.8, 0.6, 0.6, 0.4)
	b.AssertRender(t, "splits/tiles-row-split-span-split-set")
}

func TestSplitsTilesRowSpanPlus(t *testing.T) {
	b, sp := makeSplits(5, 40, 20)
	sp.SetTiles(TileSpan, TilePlus)
	b.AssertRender(t, "splits/tiles-row-span-plus")
}
