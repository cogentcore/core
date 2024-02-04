// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"fmt"
	"path/filepath"
	"testing"

	"cogentcore.org/core/colors"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/ki"
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
	b.AssertRender(t, filepath.Join("render", "one-side-border"))
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
	b.AssertRender(t, filepath.Join("render", "parent-border-radius"))
}

// For https://github.com/cogentcore/core/issues/810
func TestRenderButtonAlignment(t *testing.T) {
	b := NewBody()
	bt := NewButton(b).SetType(ButtonAction).SetIcon(icons.Square).Style(func(s *styles.Style) {
		s.Background = colors.C(colors.Scheme.SurfaceContainerHighest)
		s.Border = styles.Border{}
		s.MaxBorder = styles.Border{}
	})
	b.AssertRender(t, filepath.Join("render", "button-alignment"), func() {
		bt.WidgetWalkPre(func(kwi Widget, kwb *WidgetBase) bool {
			fmt.Printf("%v: %#v %#v\n\n", kwb, kwb.Styles.BoxSpace(), kwb.Geom)
			return ki.Continue
		})
	})
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
	b.AssertRender(t, filepath.Join("render", "frame-alignment"))
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
	b.AssertRender(t, filepath.Join("render", "frame-alignment-center"))
}
