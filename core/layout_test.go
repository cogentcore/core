// Copyright 2023 Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/base/strcase"
	"cogentcore.org/core/colors"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/units"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func layoutTestDir(t *testing.T) string {
	tnm := strcase.ToSnake(strings.TrimPrefix(t.Name(), "TestLayout"))
	n := filepath.Join("layout", tnm)
	p := filepath.Join("testdata", n)
	errors.Log(os.MkdirAll(p, 0750))
	return n
}

func TestLayoutFramesAlignItems(t *testing.T) {
	wraps := []bool{false, true}
	dirs := []styles.Directions{styles.Row, styles.Column}
	aligns := []styles.Aligns{styles.Start, styles.Center, styles.End}
	tdir := layoutTestDir(t)
	for _, wrap := range wraps {
		for _, dir := range dirs {
			for _, align := range aligns {
				tnm := fmt.Sprintf("wrap_%v_dir_%v_align_%v", wrap, dir, align)
				b := NewBody()
				b.Styler(func(s *styles.Style) {
					s.Overflow.Set(styles.OverflowVisible)
					s.Direction = dir
					s.Wrap = wrap
					s.Align.Items = align
				})
				plainFrames(b, math32.Vec2(0, 0))
				b.AssertRender(t, tdir+tnm)
			}
		}
	}
}

func TestLayoutFramesAlignContent(t *testing.T) {
	wraps := []bool{false, true}
	dirs := []styles.Directions{styles.Row, styles.Column}
	aligns := []styles.Aligns{styles.Start, styles.Center, styles.End}
	tdir := layoutTestDir(t)
	for _, wrap := range wraps {
		for _, dir := range dirs {
			for _, align := range aligns {
				tnm := fmt.Sprintf("wrap-%v-dir-%v-align-%v", wrap, dir, align)
				b := NewBody()
				b.Styler(func(s *styles.Style) {
					if dir == styles.Row {
						s.Min.Y.Dp(300)
					} else {
						s.Min.X.Dp(300)
					}
					s.Overflow.Set(styles.OverflowVisible)
					s.Direction = dir
					s.Wrap = wrap
					s.Align.Content = align
				})
				plainFrames(b, math32.Vec2(0, 0))
				b.AssertRender(t, tdir+tnm)
			}
		}
	}
}

func TestLayoutFramesJustifyContent(t *testing.T) {
	wraps := []bool{false, true}
	dirs := []styles.Directions{styles.Row, styles.Column}
	aligns := []styles.Aligns{styles.Start, styles.Center, styles.End}
	tdir := layoutTestDir(t)
	for _, wrap := range wraps {
		dsz := float32(600)
		if wrap {
			dsz = 400
		}
		for _, dir := range dirs {
			for _, align := range aligns {
				tnm := fmt.Sprintf("wrap_%v_dir_%v_align_%v", wrap, dir, align)
				b := NewBody()
				b.Styler(func(s *styles.Style) {
					if dir == styles.Row {
						s.Min.X.Dp(dsz)
					} else {
						s.Min.Y.Dp(dsz)
					}
					s.Overflow.Set(styles.OverflowVisible)
					s.Direction = dir
					s.Wrap = wrap
					s.Justify.Content = align
				})
				plainFrames(b, math32.Vec2(0, 0))
				b.AssertRender(t, tdir+tnm)
			}
		}
	}
}

func TestLayoutFramesJustifyItems(t *testing.T) {
	aligns := []styles.Aligns{styles.Start, styles.Center, styles.End}
	tdir := layoutTestDir(t)
	// dsz := float32(600)
	for _, align := range aligns {
		tnm := fmt.Sprintf("align_%v", align)
		b := NewBody()
		b.Styler(func(s *styles.Style) {
			s.Overflow.Set(styles.OverflowVisible)
			s.Display = styles.Grid
			s.Columns = 2
			s.Justify.Items = align
		})
		plainFrames(b, math32.Vec2(0, 0))
		b.AssertRender(t, tdir+tnm)
	}
}

func TestLayoutFramesJustifySelf(t *testing.T) {
	aligns := []styles.Aligns{styles.Start, styles.Center, styles.End}
	tdir := layoutTestDir(t)
	// dsz := float32(600)
	for ai, align := range aligns {
		tnm := fmt.Sprintf("align_%v", align)
		b := NewBody()
		b.Styler(func(s *styles.Style) {
			s.Overflow.Set(styles.OverflowVisible)
			s.Display = styles.Grid
			s.Columns = 2
			s.Justify.Items = align
		})
		plainFrames(b, math32.Vec2(0, 0))
		b.Child(2).(Widget).AsWidget().Styler(func(s *styles.Style) {
			s.Justify.Self = aligns[(ai+1)%len(aligns)]
		})
		b.AssertRender(t, tdir+tnm)
	}
}

func TestLayoutFramesAlignSelf(t *testing.T) {
	aligns := []styles.Aligns{styles.Start, styles.Center, styles.End}
	tdir := layoutTestDir(t)
	// dsz := float32(600)
	for ai, align := range aligns {
		tnm := fmt.Sprintf("align_%v", align)
		b := NewBody()
		b.Styler(func(s *styles.Style) {
			s.Overflow.Set(styles.OverflowVisible)
			s.Display = styles.Grid
			s.Columns = 2
			s.Align.Items = align
		})
		plainFrames(b, math32.Vec2(0, 0))
		b.Child(2).(Widget).AsWidget().Styler(func(s *styles.Style) {
			s.Align.Self = aligns[(ai+1)%len(aligns)]
		})
		b.AssertRender(t, tdir+tnm)
	}
}

func boxFrame(parent Widget) *Frame {
	fr := NewFrame(parent)
	fr.Styler(func(s *styles.Style) {
		s.Border.Width.Set(units.Dp(2))
	})
	return fr
}

func plainFrames(parent Widget, grow math32.Vector2) {
	for _, sz := range frameSizes {
		fr := boxFrame(parent)
		fr.Styler(func(s *styles.Style) {
			s.Min.X.Dp(sz.X)
			s.Min.Y.Dp(sz.Y)
			s.Grow = grow
		})
	}
}

var (
	longText = "This is a test of the layout logic, which is pretty complex and requires some experimenting to understand how it all works.  The styling and behavior is the same as the CSS / HTML Flex model, except we only support Grow, not Shrink. "

	frameSizes = [5]math32.Vector2{
		{20, 100},
		{80, 20},
		{60, 80},
		{40, 120},
		{150, 100},
	}
)

func TestLayoutScrollLabel(t *testing.T) {
	b := NewBody()
	b.Styler(func(s *styles.Style) {
		s.Max.Set(units.Dp(50))
		s.Overflow.Set(styles.OverflowAuto)
	})
	NewText(b).SetText(longText)
	b.AssertRender(t, "layout/scroll/label")
}

func TestParentRelativeSize(t *testing.T) {
	b := NewBody()
	b.Styler(func(s *styles.Style) {
		s.Min.Set(units.Dp(100))
	})
	fr := NewFrame(b)
	fr.SetName("target")
	fr.Styler(func(s *styles.Style) {
		s.Direction = styles.Row
		s.Grow.Set(1, 1)
		s.Overflow.Set(styles.OverflowAuto)
	})
	NewFrame(fr).Styler(func(s *styles.Style) {
		s.Background = colors.Scheme.Select.Container
		s.Grow.Set(0, 0)
		s.Min.Set(units.Pw(50))
	})
	NewFrame(fr).Styler(func(s *styles.Style) {
		s.Background = colors.Scheme.Error.Base
		s.Grow.Set(0, 0)
		s.Min.Set(units.Pw(50))
	})
	b.AssertRender(t, "layout/parent-relative")
}

func TestParentRelativeSizeSplits(t *testing.T) {
	b := NewBody()
	b.Styler(func(s *styles.Style) {
		s.Min.Set(units.Dp(400))
	})

	splitFrame := NewSplits(b)
	splitFrame.Styler(func(s *styles.Style) {
		s.Grow.Set(1, 1)
		s.Direction = styles.Row
	})
	splitFrame.SetSplits(20, 60, 20)
	//
	firstFrame := NewFrame(splitFrame)
	firstFrame.Styler(func(s *styles.Style) {
		s.Grow.Set(1, 1)
		s.Border.Width.Set(units.Dp(4))
		s.CenterAll()
	})
	NewText(firstFrame).SetText("20% Split Frame")

	// The center of the split, created so that we can put child widgets in it
	centerFrame := NewFrame(splitFrame)
	centerFrame.Styler(func(s *styles.Style) {
		s.Direction = styles.Column
		s.Grow.Set(1, 1)
		//s.Overflow.Set(styles.OverflowHidden)
	})

	// represents a top "menu" bar that has buttons in it
	centerMenu := NewFrame(centerFrame)
	centerMenu.Styler(func(s *styles.Style) {
		s.Min.Set(units.Pw(100), units.Ph(10))
		s.Max.Set(units.Pw(100), units.Ph(10))
		s.Border.Width.Set(units.Dp(4))
		s.CenterAll()
	})
	NewText(centerMenu).SetText("Menu Bar")

	// the frame that the child widget will use to wrap its content so that from the outside we only have a single frame
	centerContentArea := NewFrame(centerFrame)
	centerContentArea.Styler(func(s *styles.Style) {
		s.Grow.Set(1, 1)
		s.Border.Width.Set(units.Dp(4))
		s.Overflow.Set(styles.OverflowScroll)
	})
	//
	// represents content that will be too large for this frame, user will need to scroll this on both
	// axes to interact with it correctly
	internalCenter := NewFrame(centerContentArea)
	internalCenter.Styler(func(s *styles.Style) {
		s.Min.Set(units.Pw(125), units.Ph(125))
		s.CenterAll()
	})
	NewText(internalCenter).SetText("Content in center frame")
	//
	lastFrame := NewFrame(splitFrame)
	lastFrame.Styler(func(s *styles.Style) {
		s.Grow.Set(1, 1)
		s.CenterAll()
		s.Border.Width.Set(units.Dp(4))
	})
	NewText(lastFrame).SetText("20% Split Frame")
	b.AssertRender(t, "layout/parent-relative-splits")
}

func TestCustomLayout(t *testing.T) {
	b := NewBody()
	b.Styler(func(s *styles.Style) {
		s.Min.Set(units.Dp(100))
	})
	fr := NewFrame(b)
	fr.Styler(func(s *styles.Style) {
		s.Display = styles.Custom
		s.Grow.Set(1, 1)
	})
	NewFrame(fr).Styler(func(s *styles.Style) {
		s.Background = colors.Scheme.Select.Container
		s.Min.Set(units.Dp(40))
		s.Pos.Set(units.Dp(5))
	})
	NewFrame(fr).Styler(func(s *styles.Style) {
		s.Background = colors.Scheme.Error.Base
		s.Min.Set(units.Dp(40))
		s.Pos.Set(units.Dp(50))
	})
	b.AssertRender(t, "layout/custom")
}

func TestCustomLayoutButton(t *testing.T) {
	b := NewBody()
	b.Styler(func(s *styles.Style) {
		s.Display = styles.Custom
		s.Min.Set(units.Dp(200), units.Dp(100))
	})
	bt := NewButton(b).SetText("Hello")
	bt.Styler(func(s *styles.Style) {
		s.Min.X.Dp(100)
		s.Pos.Set(units.Dp(25))
	})
	b.AssertRender(t, "layout/custom-button")
}

func TestLayoutMaxGrow(t *testing.T) { // issue #1557
	names := []string{"none", "first", "two", "all"}
	for i, name := range names {
		b := NewBody()
		b.Styler(func(s *styles.Style) {
			s.Min.Set(units.Dp(400), units.Dp(50))
		})
		container := NewFrame(b)
		container.Styler(func(s *styles.Style) {
			s.Grow.Set(1, 1)
			s.Border.Width.Set(units.Dp(4))
		})

		firstPane := NewFrame(container)
		firstPane.Styler(func(s *styles.Style) {
			s.Grow.Set(1, 1)
			s.Border.Width.Set(units.Dp(2))
			if i >= 1 {
				s.Max.X.Dp(50)
			}
		})

		secondPane := NewFrame(container)
		secondPane.Styler(func(s *styles.Style) {
			s.Grow.Set(1, 1)
			s.Border.Width.Set(units.Dp(2))
			if i == 3 {
				s.Max.X.Dp(50)
			}
		})

		thirdPane := NewFrame(container)
		thirdPane.Styler(func(s *styles.Style) {
			s.Grow.Set(1, 1)
			s.Border.Width.Set(units.Dp(2))
			if i >= 2 {
				s.Max.X.Dp(50)
			}
		})
		b.AssertRender(t, "layout/maxgrow/"+name)
	}
}

// These tests are for the intrinsic widths of the layout
// (layoutintrinsic.go), and the issues found with them.

// layoutAtWidth lays out the scene of the body at the given width and
// height in dp, as when its window is resized, and then calls check with
// the size of a dp in dots.
func layoutAtWidth(b *Body, w, h float32, check func(dp float32)) {
	b.RenderImage(func() {
		dp := b.Styles.UnitContext.Dots(units.UnitDp)
		b.AsyncLock()
		b.Scene.LayoutFrame(math32.Vec2(w*dp, h*dp))
		check(dp)
		b.AsyncUnlock()
	})
}

// scrollFrame adds a frame that scrolls and grows, in a column.
func scrollFrame(parent Widget) *Frame {
	fr := NewFrame(parent)
	fr.Styler(func(s *styles.Style) {
		s.Overflow.Set(styles.OverflowAuto)
		s.Grow.Set(1, 1)
		s.Direction = styles.Column
	})
	return fr
}

// columnFrame adds a column frame that grows, with a Min width in dp
// if it is above 0.
func columnFrame(parent Widget, minw float32) *Frame {
	fr := NewFrame(parent)
	fr.Styler(func(s *styles.Style) {
		s.Grow.Set(1, 1)
		s.Direction = styles.Column
		if minw > 0 {
			s.Min.X.Dp(minw)
		}
	})
	return fr
}

// wideRow adds a row that does not wrap, of n fixed frames w dp wide.
func wideRow(parent Widget, n int, w float32) *Frame {
	row := NewFrame(parent)
	row.Styler(func(s *styles.Style) {
		s.Gap.Zero()
		s.Padding.Zero()
	})
	for range n {
		fixedFrame(row, w, 20)
	}
	return row
}

// TestLayoutNestedScrollMin is for nested frames that scroll, in frames
// with a Min width, as tabs in tabs (the demo): when the window is
// narrower than the Min, the frames with the Min still have it, and only
// the frame that scrolls inside them scrolls in width, not also the
// frame that scrolls around them.
func TestLayoutNestedScrollMin(t *testing.T) {
	b := NewBody()
	outer := columnFrame(b, 0)
	holder := columnFrame(outer, 160)
	mid := scrollFrame(holder) // like the Widgets tab
	inner := columnFrame(mid, 0)
	holder2 := columnFrame(inner, 160)
	scroller := scrollFrame(holder2) // like the Inputs tab
	wideRow(scroller, 3, 100)
	layoutAtWidth(b, 120, 400, func(dp float32) {
		assert.GreaterOrEqual(t, holder.Geom.Size.Alloc.Total.X, 160*dp-1, "the frame keeps its Min")
		assert.GreaterOrEqual(t, holder2.Geom.Size.Alloc.Total.X, 160*dp-1, "the inner frame keeps its Min")
		assert.False(t, mid.HasScroll[math32.X], "the frame around does not also scroll in width")
		assert.True(t, scroller.HasScroll[math32.X], "the inner frame scrolls in width")
	})
}

// TestLayoutOverflowScrolls is for a row that does not wrap (as segmented
// buttons) in a frame that scrolls and is narrower than it: the row is
// narrowed as a block, and the frame scrolls to all of its width, instead
// of clipping it.
func TestLayoutOverflowScrolls(t *testing.T) {
	b := NewBody()
	scroller := scrollFrame(b)
	scroller.Styler(func(s *styles.Style) {
		s.Max.X.Dp(200)
	})
	NewText(scroller).SetText(wrapText)
	row := wideRow(scroller, 3, 120)
	layoutAtWidth(b, 400, 600, func(dp float32) {
		assert.True(t, scroller.HasScroll[math32.X], "the frame scrolls in width")
		assert.GreaterOrEqual(t, row.Geom.Size.Actual.Total.X, 360*dp-1, "the row is as wide as its content")
		assert.GreaterOrEqual(t, scroller.Geom.Size.Internal.X, row.Geom.Size.Actual.Total.X-1, "the frame scrolls to all of the row")
	})
}

// TestLayoutPartsNotNarrowed is for widgets with parts (such as tree
// nodes), which keep their width in a frame narrower than them, as their
// parts are laid out within their actual size, even when the text in
// their parts could wrap: the label stays on one line, and the frame
// scrolls to it.
func TestLayoutPartsNotNarrowed(t *testing.T) {
	b := NewBody()
	col := scrollFrame(b)
	col.Styler(func(s *styles.Style) {
		s.Max.X.Dp(100)
	})
	tr := NewTree(col).SetText("A tree node with a long label")
	var label *Text
	b.RenderImage(func() {
		// the label of the tree node wraps
		label = tr.Parts.ChildByName("text", 1).(*Text)
		label.Styler(func(s *styles.Style) { s.SetTextWrap(true) })
		label.Style()
		tr.NeedsLayout()
	}, func() {
		dp := b.Styles.UnitContext.Dots(units.UnitDp)
		b.AsyncLock()
		b.Scene.LayoutFrame(math32.Vec2(400*dp, 400*dp))
		sz := &tr.Geom.Size
		assert.Equal(t, sz.prefWidth, sz.minWidth, "its minimum is its preferred width")
		assert.Greater(t, sz.minWidth, 100*dp, "which is wider than the frame")
		assert.True(t, label.Styles.Text.WhiteSpace.HasWordWrap())
		assert.Len(t, label.paintText.Lines, 1, "its label is on one line")
		assert.True(t, col.HasScroll[math32.X], "the frame scrolls to it")
		b.AsyncUnlock()
	})
}

// TestLayoutTextWrapHeight is for text that starts out on one line and is
// then wrapped in a narrow width: all of its lines are laid out, not only
// those that fit in the height of one line that it was allocated.
func TestLayoutTextWrapHeight(t *testing.T) {
	b := NewBody()
	col := columnFrame(b, 0)
	col.Styler(func(s *styles.Style) {
		s.Max.X.Dp(150)
	})
	tx := NewText(col).SetText(wrapText)
	layoutAtWidth(b, 400, 2000, func(dp float32) {
		lines := len(tx.paintText.Lines)
		assert.Greater(t, lines, 10, "the text wraps in many lines")
		lht := tx.paintText.LineHeight
		assert.GreaterOrEqual(t, tx.Geom.Size.Actual.Content.Y, float32(lines)*lht-1, "it is as tall as all of its lines")
		assert.Contains(t, string(tx.paintText.Source.Join()), "wrap.", "all of its text is laid out")
		last := tx.paintText.Lines[lines-1]
		assert.Equal(t, len(tx.paintText.Source.Join()), last.SourceRange.End, "the last line ends the text")
	})
}

// TestLayoutShrinkToMin is for text in a row that is narrower than the
// longest word of the text: it shrinks to the width of that word (and
// the row scrolls), instead of staying as wide as it is on one line.
func TestLayoutShrinkToMin(t *testing.T) {
	b := NewBody()
	row := NewFrame(b)
	row.Styler(func(s *styles.Style) {
		s.Max.Set(units.Dp(100))
		s.Overflow.Set(styles.OverflowAuto)
	})
	tx := NewText(row).SetText("Some text with an extraordinarily long word in it, " + wrapText)
	layoutAtWidth(b, 400, 400, func(dp float32) {
		sz := &tx.Geom.Size
		assert.Less(t, sz.minWidth, sz.prefWidth)
		assert.InDelta(t, sz.minWidth, sz.Alloc.Total.X, 1, "it shrinks to its min-content width")
		assert.Greater(t, sz.Actual.Total.Y, 5*tx.paintText.LineHeight, "and wraps in it")
		assert.True(t, row.HasScroll[math32.X], "the row scrolls to the long word")
	})
}

// TestLayoutShrinkMaxBasis is for columns of a row that grow up to the
// same Max (as a CSS width of 100%), with texts of different lengths:
// they shrink to the same width, as for the flex basis of CSS.
func TestLayoutShrinkMaxBasis(t *testing.T) {
	b := NewBody()
	row := NewFrame(b)
	row.Styler(func(s *styles.Style) {
		s.Min.X.Dp(300)
		s.Max.X.Dp(300)
		s.Gap.Zero()
		s.Padding.Zero()
	})
	var cols []*Frame
	for i := range 3 {
		col := columnFrame(row, 0)
		col.Styler(func(s *styles.Style) {
			s.Max.X.Pw(100)
			s.Padding.Zero()
		})
		NewText(col).SetText(strings.Repeat("Words. ", 10*(i+1)))
		cols = append(cols, col)
	}
	layoutAtWidth(b, 400, 600, func(dp float32) {
		w0 := cols[0].Geom.Size.Alloc.Total.X
		for _, col := range cols[1:] {
			assert.InDelta(t, w0, col.Geom.Size.Alloc.Total.X, 2, "the columns have the same width")
			assert.InDelta(t, w0, col.Geom.Size.Actual.Total.X, 2, "and their text wraps in it")
		}
		assert.InDelta(t, 100*dp, w0, 2)
	})
}

// TestLayoutGridGrowShrinkable is for a grid column that grows into the
// extra width of the grid and has content that could shrink (a grid with
// a column that grows up to a Max, with wrapping text, beside a fixed
// column, as for grid template areas in CSS): it gets all of the extra
// width, not only its preferred width.
func TestLayoutGridGrowShrinkable(t *testing.T) {
	b := NewBody()
	grid := NewFrame(b)
	grid.Styler(func(s *styles.Style) {
		s.Display = styles.Grid
		s.Columns = 2
		s.Min.X.Dp(600)
		s.Grow.Set(1, 0)
		s.Gap.Zero()
		s.Padding.Zero()
	})
	fixedFrame(grid, 200, 20)
	inner := NewFrame(grid)
	inner.Styler(func(s *styles.Style) {
		s.Display = styles.Grid
		s.Columns = 2
		s.Grow.Set(1, 1)
		s.Shrink.X = 1
		s.Gap.Zero()
		s.Padding.Zero()
	})
	title := columnFrame(inner, 0)
	title.Styler(func(s *styles.Style) {
		s.Max.X.Dp(300)
		s.Shrink.X = 1
		s.Padding.Zero()
	})
	NewText(title).SetText("Title")
	fixedFrame(inner, 100, 20)
	body := columnFrame(inner, 0)
	body.Styler(func(s *styles.Style) {
		s.Max.X.Dp(300)
		s.Shrink.X = 1
		s.Padding.Zero()
	})
	NewText(body).SetText("The body text of the page, which is long enough to wrap in the width of its column of the grid.")
	layoutAtWidth(b, 800, 400, func(dp float32) {
		gw := grid.Geom.Size.Actual.Content.X
		assert.InDelta(t, gw-200*dp, inner.Geom.Size.Alloc.Total.X, 2, "the growing column has all of the extra width")
	})
}

// TestLayoutContentSizingMeasure is for a window sized to its content
// with a long text: it is limited to a readable measure, within which the
// text wraps, instead of being as wide as the text on one line.
func TestLayoutContentSizingMeasure(t *testing.T) {
	b := NewBody()
	tx := NewText(b).SetText(wrapText)
	b.RenderImage(func() {
		measure := units.Ch(contentSizingMeasure)
		measure.ToDots(&tx.Styles.UnitContext)
		w := b.Scene.Geom.Size.Actual.Total.X
		assert.LessOrEqual(t, w, measure.Dots+b.Styles.BoxSpace().Size().X+b.Scene.Styles.BoxSpace().Size().X+2, "the window is within the measure")
		assert.Greater(t, len(tx.paintText.Lines), 2, "the text wraps in it")
	})
}

// TestLayoutTextScrollbar is for text in a frame that only gets a vertical
// scrollbar once its text is wrapped (and so taller), which makes it
// narrower: the text is wrapped again in that width, so that it is not
// hidden under the scrollbar. (When the scrollbar appears after the
// second iteration of SizeDown, as for nested frames in the w3c snapshot
// of htmlcore, the text is wrapped again because its width changed.)
func TestLayoutTextScrollbar(t *testing.T) {
	b := NewBody()
	scroller := scrollFrame(b)
	scroller.Styler(func(s *styles.Style) {
		s.Min.Set(units.Dp(300), units.Dp(120))
		s.Max.Set(units.Dp(300), units.Dp(120))
	})
	var txs []*Text
	for range 3 {
		txs = append(txs, NewText(scroller).SetText(wrapText[:len(wrapText)/3]))
	}
	layoutAtWidth(b, 400, 400, func(dp float32) {
		assert.True(t, scroller.HasScroll[math32.Y], "the frame scrolls vertically")
		visible := scroller.Geom.Size.Alloc.Content.X
		for _, tx := range txs {
			assert.LessOrEqual(t, tx.Geom.Size.Actual.Total.X, visible+1, "the text is not wider than the frame within its scrollbar")
		}
	})
}

// TestLayoutOverflowBeyondMax is for content that overflows a frame with a
// Max narrower than it, in a frame that scrolls (as a row of cards in a
// page with a max-width in CSS): the overflow is still visible, and the
// frame that scrolls scrolls over all of it, not only to the Max.
func TestLayoutOverflowBeyondMax(t *testing.T) {
	b := NewBody()
	scroller := scrollFrame(b)
	scroller.Styler(func(s *styles.Style) {
		s.Max.X.Dp(300)
	})
	page := columnFrame(scroller, 0)
	page.Styler(func(s *styles.Style) {
		s.Max.X.Dp(200)
		s.Padding.Zero()
	})
	row := wideRow(page, 3, 150)
	layoutAtWidth(b, 800, 400, func(dp float32) {
		assert.LessOrEqual(t, page.Geom.Size.Actual.Total.X, 200*dp+1, "the page is within its Max")
		assert.GreaterOrEqual(t, row.Geom.Size.Actual.Total.X, 450*dp-1, "the row overflows it")
		assert.True(t, scroller.HasScroll[math32.X], "the frame scrolls in width")
		maxSize, _, _ := scroller.ScrollValues(math32.X)
		assert.GreaterOrEqual(t, maxSize, 450*dp-1, "over all of the row")
	})
}

// TestLayoutGrowWithinRow is for a row with a column that grows up to a
// Max and has text that can shrink, beside a column that grows and cannot
// shrink (as the header of a page with a heading and a button): when they
// fit, they grow into the width that is left, and do not overflow the row.
func TestLayoutGrowWithinRow(t *testing.T) {
	b := NewBody()
	row := NewFrame(b)
	row.Styler(func(s *styles.Style) {
		s.Min.X.Dp(400)
		s.Max.X.Dp(400)
		s.Gap.Zero()
		s.Padding.Zero()
	})
	heading := columnFrame(row, 0)
	heading.Styler(func(s *styles.Style) {
		s.Max.X.Dp(300)
		s.Padding.Zero()
	})
	NewText(heading).SetText(wrapText)
	side := columnFrame(row, 0)
	side.Styler(func(s *styles.Style) {
		s.Max.X.Pw(30)
		s.Padding.Zero()
	})
	fixedFrame(side, 80, 20)
	layoutAtWidth(b, 800, 600, func(dp float32) {
		w := heading.Geom.Size.Alloc.Total.X + side.Geom.Size.Alloc.Total.X
		assert.LessOrEqual(t, w, 400*dp+1, "the columns fit in the row")
		assert.GreaterOrEqual(t, w, 400*dp-2, "and fill it")
		assert.LessOrEqual(t, row.Geom.Size.Internal.X, 400*dp+1, "the row does not overflow")
	})
}

// TestLayoutWrapInRow is for a wrapped row (that does not shrink) beside
// text in a row: the wrapped row has its preferred width (its items on
// one line) and its min-content width (its widest item), so that both it
// and the text get their share of the row, and its items wrap within it.
func TestLayoutWrapInRow(t *testing.T) {
	b := NewBody()
	row := NewFrame(b)
	row.Styler(func(s *styles.Style) {
		s.Min.X.Dp(300)
		s.Max.X.Dp(300)
		s.Gap.Zero()
		s.Padding.Zero()
	})
	tx := NewText(row).SetText(wrapText)
	wrap := NewFrame(row)
	wrap.Styler(func(s *styles.Style) {
		s.Wrap = true
		s.Gap.Zero()
		s.Padding.Zero()
	})
	for range 5 {
		fixedFrame(wrap, 50, 20)
	}
	layoutAtWidth(b, 800, 600, func(dp float32) {
		wa := wrap.Geom.Size.Alloc.Total.X
		ta := tx.Geom.Size.Alloc.Total.X
		assert.LessOrEqual(t, wa+ta, 300*dp+1, "they fit in the row")
		assert.GreaterOrEqual(t, wa, 50*dp-1, "the wrapped row has at least its widest item")
		assert.Greater(t, ta, 50*dp, "the text has some of the row")
		assert.LessOrEqual(t, wrap.Geom.Size.Actual.Total.X, wa+1, "the items wrap within the wrapped row")
	})
}

// TestLayoutOverflowWithinRoom is for a frame whose content overflows
// it visibly beyond its Max, at the start of a row that fills the width
// of a frame that scrolls (as the icons of a footer on Wikipedia): the
// overflow is within the room after the frame in the row, so the frame
// that scrolls does not scroll in width for it.
func TestLayoutOverflowWithinRoom(t *testing.T) {
	b := NewBody()
	scroller := scrollFrame(b)
	scroller.Styler(func(s *styles.Style) {
		s.Max.X.Dp(240)
	})
	page := columnFrame(scroller, 0)
	page.Styler(func(s *styles.Style) {
		s.Padding.Zero()
	})
	row := NewFrame(page)
	row.Styler(func(s *styles.Style) {
		s.Gap.Zero()
		s.Padding.Zero()
	})
	box := NewFrame(row)
	box.Styler(func(s *styles.Style) {
		s.Max.X.Dp(80)
		s.Padding.Zero()
	})
	wideRow(box, 1, 120) // overflows the box by 40
	fixedFrame(row, 160, 20)
	// the sizes are whole dots (at 1.0125 dots per dp), so that they fit
	layoutAtWidth(b, 800, 400, func(dp float32) {
		assert.LessOrEqual(t, box.Geom.Size.Actual.Total.X, 80*dp+1, "the box is within its Max")
		assert.False(t, scroller.HasScroll[math32.X], "the overflow of the box is within the row")
	})
}

// TestLayoutWrapEmptyItem is for a wrapped row with a gap, of a frame of
// text that is wider than the row on one line and an empty frame after it
// (as the table of contents of MDN and the empty element of its ads): as
// in CSS, the text frame takes the width of the row, and the empty frame
// is on the next line, so that the line does not overflow the row by the
// gap.
func TestLayoutWrapEmptyItem(t *testing.T) {
	b := NewBody()
	row := NewFrame(b)
	row.Styler(func(s *styles.Style) {
		s.Wrap = true
		s.Min.X.Dp(200)
		s.Max.X.Dp(200)
		s.Gap.Set(units.Dp(8))
		s.Padding.Zero()
	})
	textFrame := NewFrame(row)
	textFrame.Styler(func(s *styles.Style) {
		s.Padding.Zero()
	})
	NewText(textFrame).SetText(wrapText)
	empty := NewFrame(row)
	empty.Styler(func(s *styles.Style) {
		s.Padding.Zero()
	})
	layoutAtWidth(b, 800, 400, func(dp float32) {
		assert.LessOrEqual(t, row.Geom.Size.Actual.Total.X, 200*dp+1, "the line fits the row")
		assert.Equal(t, 1, empty.Geom.Cell.Y, "the empty frame is on the next line")
	})
}

// TestLayoutWrapRoundingFits is for a wrapped row of items whose exact
// widths fit it together, in a frame that scrolls: their widths are each
// rounded up to whole dots, which keeps them on one line (see
// TestShrinkWrapRow), and the row is still as wide as the frame, which
// does not scroll in width for those dots (as the navigation of the Rust
// site).
func TestLayoutWrapRoundingFits(t *testing.T) {
	b := NewBody()
	scroller := scrollFrame(b)
	scroller.Styler(func(s *styles.Style) {
		s.Max.X.Dp(100)
		s.Padding.Zero()
	})
	row := NewFrame(scroller)
	row.Styler(func(s *styles.Style) {
		s.Wrap = true
		s.Grow.Set(1, 0)
		s.Gap.Zero()
		s.Padding.Zero()
	})
	var items []*Frame
	for range 4 { // 25.4 dots each, 26 rounded up, in 102 dots
		items = append(items, fixedFrame(row, 25.09, 20))
	}
	layoutAtWidth(b, 800, 400, func(dp float32) {
		for _, it := range items {
			assert.Equal(t, 0, it.Geom.Cell.Y, "the items are on one line")
		}
		assert.False(t, scroller.HasScroll[math32.X], "the frame does not scroll for the rounding")
	})
}

// TestTextWrapRounding is for a text that wraps, in a frame that is
// narrower than its preferred width (rounded up to whole dots) by less than
// a dot (as the menu items of pkg.go.dev, a text and an icon in a link
// whose width rounds down): it stays on one line.
func TestTextWrapRounding(t *testing.T) {
	var w float32
	b := NewBody()
	tx := NewText(b).SetText("Why Go")
	layoutAtWidth(b, 800, 400, func(dp float32) {
		w = tx.paintText.Bounds.Size().X // exactly, not rounded up
	})
	require.Greater(t, w, float32(0))
	b = NewBody()
	fr := NewFrame(b)
	fr.Styler(func(s *styles.Style) {
		s.Max.X.Dot(math32.Ceil(w) - 1) // less than w, by less than 1
		s.Padding.Zero()
	})
	tx = NewText(fr).SetText("Why Go")
	var h float32
	b2 := NewBody()
	one := NewText(b2).SetText("Why Go")
	layoutAtWidth(b2, 800, 400, func(dp float32) { h = one.Geom.Size.Actual.Content.Y })
	layoutAtWidth(b, 800, 400, func(dp float32) {
		assert.Equal(t, h, tx.Geom.Size.Actual.Content.Y, "one line")
	})
}

// TestLayoutFloatOfFloats is for a float that only has floats in it (as
// the icons in the footer of Wikipedia: a list that floats right, of
// items that float left): as in CSS, the float is as wide as its floats
// side by side, which are in it, within the column, instead of having no
// width and its floats overflowing the column on top of each other.
func TestLayoutFloatOfFloats(t *testing.T) {
	b := NewBody()
	page := columnFrame(b, 0)
	page.Styler(func(s *styles.Style) {
		s.Max.X.Dp(400)
		s.Padding.Zero()
	})
	fixedFrame(page, 100, 20)
	list := NewFrame(page)
	list.Styler(func(s *styles.Style) {
		s.Direction = styles.Column
		s.Float = styles.FloatRight
		s.Padding.Zero()
	})
	var items []*Frame
	for range 2 {
		it := fixedFrame(list, 50, 20)
		it.Styler(func(s *styles.Style) { s.Float = styles.FloatLeft })
		items = append(items, it)
	}
	layoutAtWidth(b, 800, 400, func(dp float32) {
		right := page.Geom.Pos.Content.X + page.Geom.Size.Actual.Content.X
		assert.InDelta(t, 100*dp, list.Geom.Size.Actual.Total.X, 2, "the list is as wide as its floats")
		assert.Less(t, items[0].Geom.Pos.Total.X+1, items[1].Geom.Pos.Total.X, "the floats are side by side")
		for _, it := range items {
			assert.LessOrEqual(t, it.Geom.Pos.Total.X+it.Geom.Size.Actual.Total.X, right+1, "within the column")
		}
	})
}

// TestLayoutFloatsBesideInline tests right floats after inline content that
// is on one line (as the theme and search box of the header of the Python
// docs, after its links), which are on that line, beside it, when they fit,
// as in CSS; after a block (which takes its line in CSS), a float is below.
func TestLayoutFloatsBesideInline(t *testing.T) {
	b := NewBody()
	page := columnFrame(b, 0)
	page.Styler(func(s *styles.Style) {
		s.Max.X.Dp(600)
		s.Padding.Zero()
	})
	tx := NewText(page).SetText("Python » Tutorial")
	var floats []*Frame
	for range 2 {
		f := fixedFrame(page, 80, 20)
		f.Styler(func(s *styles.Style) { s.Float = styles.FloatRight })
		floats = append(floats, f)
	}
	block := fixedFrame(page, 100, 20)
	after := fixedFrame(page, 80, 20)
	after.Styler(func(s *styles.Style) { s.Float = styles.FloatRight })
	layoutAtWidth(b, 800, 400, func(dp float32) {
		for _, f := range floats {
			assert.Equal(t, tx.Geom.TotalBBox.Min.Y, f.Geom.TotalBBox.Min.Y, "the float is on the line of the text")
		}
		assert.GreaterOrEqual(t, after.Geom.TotalBBox.Min.Y, block.Geom.TotalBBox.Max.Y-1, "a float after a block is below it")
	})
}

// TestLayoutFloatsRounding tests floats whose widths fit a row exactly but
// are each rounded up to whole dots (as the nine slides of the photos of
// the postings of craigslist.org, of 442px in a row of nine times that):
// they are all on one row, without the last one wrapping below.
func TestLayoutFloatsRounding(t *testing.T) {
	b := NewBody()
	row := NewFrame(b)
	row.Styler(func(s *styles.Style) {
		s.Direction = styles.Column
		s.Min.X.Dot(9 * 100.4)
		s.Max.X.Dot(9 * 100.4)
		s.Padding.Zero()
	})
	var slides []*Frame
	for range 9 {
		f := NewFrame(row)
		f.Styler(func(s *styles.Style) {
			s.Float = styles.FloatLeft
			s.Min.Set(units.Dot(100.4), units.Dot(20))
			s.Max.Set(units.Dot(100.4), units.Dot(20))
			s.Padding.Zero()
		})
		slides = append(slides, f)
	}
	layoutAtWidth(b, 1200, 400, func(dp float32) {
		for _, f := range slides {
			assert.Equal(t, slides[0].Geom.TotalBBox.Min.Y, f.Geom.TotalBBox.Min.Y, "on one row")
		}
	})
}

// TestBaselineFields tests the baselines of a text field and a chooser in
// a row aligned on baselines beside a text (as the search field and the
// theme select of the header of the Python docs, beside their labels):
// their texts are on the baseline of the text.
func TestBaselineFields(t *testing.T) {
	b := NewBody()
	row := NewFrame(b)
	row.Styler(func(s *styles.Style) {
		s.Direction = styles.Row
		s.Align.Items = styles.Baseline
	})
	tx := NewText(row).SetText("Theme")
	ch := NewChooser(row).SetStrings("Auto", "Light")
	tf := NewTextField(row).SetText("Quick search")
	layoutAtWidth(b, 800, 400, func(dp float32) {
		base := func(wb *WidgetBase) float32 {
			bl, ok := wb.firstBaseline()
			require.True(t, ok)
			return wb.Geom.Pos.Total.Y + bl
		}
		want := base(tx.AsWidget())
		assert.InDelta(t, want, base(ch.AsWidget()), 2, "the chooser")
		assert.InDelta(t, want, base(tf.AsWidget()), 2, "the text field")
	})
}

// TestLayoutGridSpanPercentMax tests that a percentage Max width of a
// child of a grid is of its grid area, the columns that it spans with the
// gaps between them, as in CSS: a child of 12 columns with a Max of 100%
// (as the block of the sections of nytimes.com/wirecutter, of max-width:
// 100%) is as wide as the grid, not narrower by its 11 gaps.
func TestLayoutGridSpanPercentMax(t *testing.T) {
	b := NewBody()
	g := NewFrame(b)
	g.FrColumns = true
	g.Styler(func(s *styles.Style) {
		s.Display = styles.Grid
		s.Columns = 12
		s.Gap.Set(units.Dp(20), units.Dp(0))
		s.Min.X.Dp(1024)
		s.Max.X.Dp(1024)
		s.Padding.Zero()
	})
	var wide *Frame
	for i, span := range []int{12, 8} {
		f := NewFrame(g)
		NewText(f).SetText("text")
		f.Styler(func(s *styles.Style) {
			s.ColSpan = span
			s.Grow.Set(1, 1)
			if i == 0 {
				s.Max.X = units.Pw(100)
			}
		})
		if i == 0 {
			wide = f
		}
	}
	b.RenderImage(func() {
		assert.InDelta(t, g.Geom.Size.Actual.Content.X, wide.Geom.Size.Actual.Total.X, 1, "as wide as the grid")
	})
}

// TestLayoutPercentPaddingFirst tests that a percentage padding is of the
// width of the parent in the first layout of a widget, not of the width
// that its unit context had when it was styled (as the boxes of the
// aspect ratios of the images of edition.cnn.com, of padding-bottom:
// 56.25%, which its scripts make).
func TestLayoutPercentPaddingFirst(t *testing.T) {
	b := NewBody()
	fr := NewFrame(b)
	fr.Styler(func(s *styles.Style) { s.Direction = styles.Column; s.Min.X.Dp(200); s.Max.X.Dp(200) })
	NewText(fr).SetText("before")
	b.RenderImage(func() {
		// a box added after the page is laid out (as by a script), in
		// one layout
		b.AsyncLock()
		defer b.AsyncUnlock()
		// (in a new parent, which has no size yet when it is styled)
		wrap := NewFrame(fr)
		wrap.Styler(func(s *styles.Style) { s.Grow.X = 1 })
		box := NewFrame(wrap)
		box.Styler(func(s *styles.Style) { s.Grow.X = 1; s.Padding.Bottom = units.Pw(50) })
		wrap.Update()
		b.Scene.LayoutScene()
		dp := b.Styles.UnitContext.Dots(units.UnitDp)
		assert.InDelta(t, 100*dp, box.Geom.Size.Actual.Total.Y, 3)
	})
}
