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
