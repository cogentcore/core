// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"testing"

	"cogentcore.org/core/colors"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/units"
)

// For https://github.com/cogentcore/core/issues/614
func TestRenderOneSideBorder(t *testing.T) {
	b := NewBody()
	NewBox(b).Style(func(s *styles.Style) {
		s.Min.Set(units.Dp(100))
		s.Border.Width.Bottom.Dp(10)
		s.Border.Color.Bottom = colors.Scheme.Outline
		s.Background = colors.C(colors.Scheme.SurfaceContainerHigh)
	})
	b.AssertRender(t, "render/one-side-border")
}

// For https://github.com/cogentcore/core/issues/660
func TestRenderParentBorderRadius(t *testing.T) {
	b := NewBody()
	outer := NewFrame(b).Style(func(s *styles.Style) {
		s.Border.Radius = styles.BorderRadiusFull
		s.Background = colors.C(colors.Blue)
		s.Min.Set(units.Dp(100))
	})
	NewBox(outer).Style(func(s *styles.Style) {
		s.Background = colors.C(colors.Red)
		s.Min.Set(units.Dp(80))
	})
	b.AssertRender(t, "render/parent-border-radius")
}

// For https://github.com/cogentcore/core/issues/810
func TestRenderButtonAlignment(t *testing.T) {
	b := NewBody()
	NewButton(b).SetType(ButtonAction).SetIcon(icons.Square).Style(func(s *styles.Style) {
		s.Background = colors.C(colors.Scheme.SurfaceContainerHighest)
		s.Border = styles.Border{}
		s.MaxBorder = styles.Border{}
	})
	b.AssertRender(t, "render/button-alignment")
}

// For https://github.com/cogentcore/core/issues/810
func TestRenderFrameAlignment(t *testing.T) {
	b := NewBody()
	outer := NewFrame(b).Style(func(s *styles.Style) {
		s.Background = colors.C(colors.Orange)
		s.Min.Set(units.Dp(30))
		s.Padding.Zero()
		s.Gap.Zero()
	})
	NewBox(outer).Style(func(s *styles.Style) {
		s.Background = colors.C(colors.Blue)
		s.Grow.Set(1, 1)
	})
	b.AssertRender(t, "render/frame-alignment")
}

// For https://github.com/cogentcore/core/issues/810
func TestRenderFrameAlignmentCenter(t *testing.T) {
	b := NewBody()
	outer := NewFrame(b).Style(func(s *styles.Style) {
		s.Background = colors.C(colors.Orange)
		s.Min.Set(units.Dp(30))
		s.Padding.Zero()
		s.Gap.Zero()
		s.Justify.Content = styles.Center
		s.Justify.Items = styles.Center
		s.Align.Content = styles.Center
		s.Align.Items = styles.Center
	})
	NewBox(outer).Style(func(s *styles.Style) {
		s.Background = colors.C(colors.Blue)
		s.Min.Set(units.Dp(15))
	})
	b.AssertRender(t, "render/frame-alignment-center")
}

// For https://github.com/cogentcore/core/issues/615
func TestRenderNestedScroll(t *testing.T) {
	// TODO(#808)
	b := NewBody()
	b.Style(func(s *styles.Style) {
		s.Max.Set(units.Dp(300))
	})
	f0 := NewFrame(b).Style(func(s *styles.Style) {
		s.Background = colors.C(colors.Orange)
		s.Overflow.Set(styles.OverflowAuto)
	})
	f1 := NewFrame(f0).Style(func(s *styles.Style) {
		s.Background = colors.C(colors.Blue)
		s.Min.Set(units.Dp(200))
		s.Max.Set(units.Dp(200))
	})
	f2 := NewFrame(f1).Style(func(s *styles.Style) {
		s.Background = colors.C(colors.Green)
		s.Overflow.Set(styles.OverflowAuto)
	})
	NewFrame(f2).Style(func(s *styles.Style) {
		s.Background = colors.C(colors.Red)
		s.Min.Set(units.Dp(400))
	})
	NewFrame(f0).Style(func(s *styles.Style) {
		s.Background = colors.C(colors.Purple)
		s.Min.Set(units.Dp(200))
	})
	b.AssertRender(t, "render/nested-scroll")
}
