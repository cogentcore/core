// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"testing"

	"cogentcore.org/core/colors"
	"cogentcore.org/core/colors/gradient"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/units"
)

func frameTestButtons(fr *Frame) {
	NewButton(fr).SetText("First")
	NewButton(fr).SetText("Second")
	NewButton(fr).SetText("Third")
}

func TestFrame(t *testing.T) {
	b := NewBody()
	fr := NewFrame(b)
	frameTestButtons(fr)
	b.AssertRender(t, "frame/basic")
}

func TestFrameBackground(t *testing.T) {
	b := NewBody()
	fr := NewFrame(b)
	fr.Styler(func(s *styles.Style) {
		s.Background = colors.Scheme.Warn.Container
	})
	frameTestButtons(fr)
	b.AssertRender(t, "frame/background")
}

func TestFrameGradient(t *testing.T) {
	b := NewBody()
	fr := NewFrame(b)
	fr.Styler(func(s *styles.Style) {
		s.Background = gradient.NewLinear().AddStop(colors.Yellow, 0).AddStop(colors.Orange, 0.5).AddStop(colors.Red, 1)
	})
	frameTestButtons(fr)
	b.AssertRender(t, "frame/gradient")
}

func TestFrameBorder(t *testing.T) {
	b := NewBody()
	fr := NewFrame(b)
	fr.Styler(func(s *styles.Style) {
		s.Border.Width.Set(units.Dp(4))
	})
	frameTestButtons(fr)
	b.AssertRender(t, "frame/border")
}
func TestFrameBorderRadius(t *testing.T) {
	b := NewBody()
	fr := NewFrame(b)
	fr.Styler(func(s *styles.Style) {
		s.Border.Radius = styles.BorderRadiusLarge
		s.Border.Width.Set(units.Dp(4))
	})
	frameTestButtons(fr)
	b.AssertRender(t, "frame/border-radius")
}

func TestFrameNoGrow(t *testing.T) {
	b := NewBody()
	fr := NewFrame(b)
	fr.Styler(func(s *styles.Style) {
		s.Grow.Set(0, 0)
		s.Border.Width.Set(units.Dp(4))
	})
	frameTestButtons(fr)
	b.AssertRender(t, "frame/no-grow")
}

func TestFrameScrollNoMargin(t *testing.T) {
	b := NewBody()
	fr := NewFrame(b)
	fr.Styler(func(s *styles.Style) {
		s.Overflow.Set(styles.OverflowAuto)
		s.Border.Width.Set(units.Dp(4))
		s.Max.Set(units.Dp(40))
	})
	NewFrame(fr).Styler(func(s *styles.Style) {
		s.Min.Set(units.Dp(80))
		s.Background = colors.Scheme.Select.Container
	})
	b.AssertRender(t, "frame/scroll-no-margin")
}

func TestFrameScrollMargin(t *testing.T) {
	b := NewBody()
	fr := NewFrame(b)
	fr.Styler(func(s *styles.Style) {
		s.Overflow.Set(styles.OverflowAuto)
		s.Border.Width.Set(units.Dp(4))
		s.Margin.Set(units.Dp(8))
		s.Max.Set(units.Dp(40))
	})
	NewFrame(fr).Styler(func(s *styles.Style) {
		s.Min.Set(units.Dp(80))
		s.Background = colors.Scheme.Select.Container
	})
	b.AssertRender(t, "frame/scroll-margin")
}

func TestFrameScrollMarginPadding(t *testing.T) {
	b := NewBody()
	fr := NewFrame(b)
	fr.Styler(func(s *styles.Style) {
		s.Overflow.Set(styles.OverflowAuto)
		s.Border.Width.Set(units.Dp(4))
		s.Margin.Set(units.Dp(8))
		s.Padding.Set(units.Dp(16))
		s.Max.Set(units.Dp(40))
	})
	NewFrame(fr).Styler(func(s *styles.Style) {
		s.Min.Set(units.Dp(80))
		s.Background = colors.Scheme.Select.Container
	})
	b.AssertRender(t, "frame/scroll-margin-padding")
}
