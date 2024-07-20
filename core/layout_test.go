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
						s.Min.Y.Px(300)
					} else {
						s.Min.X.Px(300)
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
						s.Min.X.Px(dsz)
					} else {
						s.Min.Y.Px(dsz)
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
			s.Min.X.Px(sz.X)
			s.Min.Y.Px(sz.Y)
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
