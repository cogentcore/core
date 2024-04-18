// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"testing"

	"cogentcore.org/core/colors"
	"cogentcore.org/core/colors/gradient"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/units"
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
	fr.Style(func(s *styles.Style) {
		s.Background = colors.C(colors.Scheme.Warn.Container)
	})
	frameTestButtons(fr)
	b.AssertRender(t, "frame/background")
}

func TestFrameGradient(t *testing.T) {
	b := NewBody()
	fr := NewFrame(b)
	fr.Style(func(s *styles.Style) {
		s.Background = gradient.NewLinear().AddStop(colors.Yellow, 0).AddStop(colors.Orange, 0.5).AddStop(colors.Red, 1)
	})
	frameTestButtons(fr)
	b.AssertRender(t, "frame/gradient")
}

func TestFrameBorder(t *testing.T) {
	b := NewBody()
	fr := NewFrame(b)
	fr.Style(func(s *styles.Style) {
		s.Border.Width.Set(units.Dp(4))
		s.Border.Color.Set(colors.C(colors.Scheme.Outline))
	})
	frameTestButtons(fr)
	b.AssertRender(t, "frame/border")
}
func TestFrameBorderRadius(t *testing.T) {
	b := NewBody()
	fr := NewFrame(b)
	fr.Style(func(s *styles.Style) {
		s.Border.Radius = styles.BorderRadiusLarge
		s.Border.Width.Set(units.Dp(4))
		s.Border.Color.Set(colors.C(colors.Scheme.Outline))
	})
	frameTestButtons(fr)
	b.AssertRender(t, "frame/border-radius")
}

func TestFrameNoGrow(t *testing.T) {
	b := NewBody()
	fr := NewFrame(b)
	fr.Style(func(s *styles.Style) {
		s.Grow.Set(0, 0)
		s.Border.Width.Set(units.Dp(4))
		s.Border.Color.Set(colors.C(colors.Scheme.Outline))
	})
	frameTestButtons(fr)
	b.AssertRender(t, "frame/no-grow")
}
