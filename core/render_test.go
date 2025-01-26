// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"testing"

	"cogentcore.org/core/colors"
	"cogentcore.org/core/colors/gradient"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/units"
	"cogentcore.org/core/tree"
)

// For https://github.com/cogentcore/core/issues/614
func TestRenderOneSideBorder(t *testing.T) {
	b := NewBody()
	NewWidgetBase(b).Styler(func(s *styles.Style) {
		s.Min.Set(units.Dp(100))
		s.Border.Width.Bottom.Dp(10)
		s.Background = colors.Scheme.SurfaceContainerHigh
	})
	b.AssertRender(t, "render/one-side-border")
}

// For https://github.com/cogentcore/core/issues/660
func TestRenderParentBorderRadius(t *testing.T) {
	b := NewBody()
	NewButton(b).SetText("Test").Styler(func(s *styles.Style) {
		s.Padding.Zero()
	})
	b.AssertRender(t, "render/parent-border-radius")
}

// For https://github.com/cogentcore/core/issues/989
func TestRenderParentBorderRadiusVerticalToolbar(t *testing.T) {
	b := NewBody()
	b.Scene.renderBBoxes = true
	b.Styler(func(s *styles.Style) {
		s.Min.Y.Em(10)
	})
	tb := NewToolbar(b)
	tb.Styler(func(s *styles.Style) {
		s.Direction = styles.Column
		s.Background = colors.Scheme.Select.Container
	})
	tree.AddChild(tb, func(w *Button) {
		w.SetIcon(icons.Close)
		w.Styler(func(s *styles.Style) {
			s.Background = colors.Scheme.Error.Base
			s.Border.Radius.Zero()
		})
	})
	b.AssertRender(t, "render/parent-border-radius-vertical-toolbar")
}

// For https://github.com/cogentcore/core/issues/989
func TestRenderParentBorderRadiusHorizontalToolbar(t *testing.T) {
	b := NewBody()
	b.Scene.renderBBoxes = true
	b.Styler(func(s *styles.Style) {
		s.Min.X.Em(10)
	})
	tb := NewToolbar(b)
	tb.Styler(func(s *styles.Style) {
		s.Direction = styles.Row
		s.Background = colors.Scheme.Select.Container
	})
	tree.AddChild(tb, func(w *Button) {
		w.SetIcon(icons.Close)
		w.Styler(func(s *styles.Style) {
			s.Background = colors.Scheme.Error.Base
			s.Border.Radius.Zero()
		})
	})
	b.AssertRender(t, "render/parent-border-radius-horizontal-toolbar")
}

func TestRenderBorderLeak(t *testing.T) {
	b := NewBody()
	NewFrame(b).Styler(func(s *styles.Style) {
		s.Min.Set(units.Em(5))
		s.Border.Width.Set(units.Dp(4))
		s.Border.Color.Set(colors.Scheme.Outline)
	})
	NewFrame(b).Styler(func(s *styles.Style) {
		s.Min.Set(units.Em(5))
		s.Border.Style.Set(styles.BorderDotted)
		s.Border.Width.Set(units.Dp(4))
		s.Border.Color.Set(colors.Scheme.Error.Base)
	})
	NewFrame(b).Styler(func(s *styles.Style) {
		s.Min.Set(units.Em(5))
		s.Border.Width.Set(units.Dp(4))
		s.Border.Color.Set(colors.Scheme.Outline)
	})
	b.AssertRender(t, "render/border-leak")
}

// For https://github.com/cogentcore/core/issues/810
func TestRenderButtonAlignment(t *testing.T) {
	b := NewBody()
	bt := NewButton(b).SetType(ButtonAction).SetIcon(icons.Square)
	bt.Styler(func(s *styles.Style) {
		s.Background = colors.Scheme.SurfaceContainerHighest
		s.Border = styles.Border{}
		s.MaxBorder = styles.Border{}
	})
	b.AssertRender(t, "render/button-alignment")
}

// For https://github.com/cogentcore/core/issues/810
func TestRenderFrameAlignment(t *testing.T) {
	b := NewBody()
	outer := NewFrame(b)
	outer.Styler(func(s *styles.Style) {
		s.Background = colors.Uniform(colors.Orange)
		s.Min.Set(units.Dp(30))
		s.Padding.Zero()
		s.Gap.Zero()
	})
	NewWidgetBase(outer).Styler(func(s *styles.Style) {
		s.Background = colors.Uniform(colors.Blue)
		s.Grow.Set(1, 1)
	})
	b.AssertRender(t, "render/frame-alignment")
}

// For https://github.com/cogentcore/core/issues/810
func TestRenderFrameAlignmentCenter(t *testing.T) {
	b := NewBody()
	outer := NewFrame(b)
	outer.Styler(func(s *styles.Style) {
		s.Background = colors.Uniform(colors.Orange)
		s.Min.Set(units.Dp(30))
		s.Padding.Zero()
		s.Gap.Zero()
		s.CenterAll()
	})
	NewWidgetBase(outer).Styler(func(s *styles.Style) {
		s.Background = colors.Uniform(colors.Blue)
		s.Min.Set(units.Dp(15))
	})
	b.AssertRender(t, "render/frame-alignment-center")
}

// For https://github.com/cogentcore/core/issues/808
func TestOverflowAutoDefinedMax(t *testing.T) {
	b := NewBody()
	fr := NewFrame(b)
	fr.Styler(func(s *styles.Style) {
		s.Max.Set(units.Dp(100))
		s.Overflow.Set(styles.OverflowAuto)
	})
	NewText(fr).SetText("This is long text that I have written for the purpose of demonstrating an issue with overflow auto on elements with a defined max")
	b.AssertRender(t, "render/overflow-auto-defined-max")
}

// For https://github.com/cogentcore/core/issues/615
func TestRenderNestedScroll(t *testing.T) {
	t.Skip("TODO(#1456): fix this test")
	b := NewBody()
	b.Styler(func(s *styles.Style) {
		s.Max.Set(units.Dp(300))
		s.Background = colors.Uniform(colors.Orange)
		s.Overflow.Set(styles.OverflowAuto)
		s.Direction = styles.Row
	})
	fr := NewFrame(b)
	fr.Styler(func(s *styles.Style) {
		s.Background = colors.Uniform(colors.Blue)
		s.Min.Set(units.Dp(200))
		s.Max.Set(units.Dp(200))
		s.Overflow.Set(styles.OverflowAuto)
	})
	NewFrame(fr).Styler(func(s *styles.Style) {
		s.Background = colors.Uniform(colors.Red)
		s.Min.Set(units.Dp(400))
	})
	NewFrame(b).Styler(func(s *styles.Style) {
		s.Background = colors.Uniform(colors.Purple)
		s.Min.Set(units.Dp(300))
	})
	b.AssertRender(t, "render/nested-scroll")
}

// For https://github.com/cogentcore/core/issues/1011
func TestRenderElementCutOff(t *testing.T) {
	b := NewBody()
	b.Styler(func(s *styles.Style) {
		s.Min.X.Em(50)
	})
	sp := NewSplits(b)
	NewFrame(sp)
	fr := NewFrame(sp)
	NewTextField(fr).Styler(func(s *styles.Style) {
		s.Max.X.Zero()
	})
	NewButton(fr).SetIcon(icons.Send)
	b.AssertRender(t, "render/element-cut-off")
}

// For https://github.com/cogentcore/core/issues/1012
func TestRenderGridCenteredFrame(t *testing.T) {
	b := NewBody()
	grid := NewFrame(b)
	grid.Styler(func(s *styles.Style) {
		s.Display = styles.Grid
		s.Columns = 2
	})
	nameLabel := NewText(grid).SetText("Name")
	nameLabel.Styler(func(s *styles.Style) {
		s.SetTextWrap(false)
	})
	NewTextField(grid).SetText("widget-base")
	label := NewText(grid).SetText("Children")
	label.Styler(func(s *styles.Style) {
		s.SetTextWrap(false)
	})
	fr := NewFrame(grid)
	fr.Styler(func(s *styles.Style) {
		s.Justify.Content = styles.Center
		s.Align.Items = styles.Center
		s.Background = colors.Scheme.Select.Container
	})
	tx := NewText(fr).SetText("0 nodes").SetType(TextLabelLarge)
	tx.Styler(func(s *styles.Style) {
		s.SetTextWrap(false)
	})
	b.AssertRender(t, "render/grid-centered-frame")
}

// For https://github.com/cogentcore/core/issues/1034
func TestRenderParentGradient(t *testing.T) {
	b := NewBody()
	b.Styler(func(s *styles.Style) {
		s.Direction = styles.Row
		s.Background = gradient.NewLinear().AddStop(colors.White, 0).AddStop(colors.Black, 1)
	})
	for range 3 {
		NewFrame(b).Styler(func(s *styles.Style) {
			s.Min.Set(units.Em(2))
		})
	}
	b.AssertRender(t, "render/parent-gradient")
}
