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
