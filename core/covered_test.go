// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"image"
	"testing"

	"cogentcore.org/core/colors"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/states"
	"cogentcore.org/core/styles/units"
	"github.com/stretchr/testify/assert"
)

// TestIsInvisibleCovered tests that a widget is invisible when a widget
// that paints is rendered over most of it (as a dialog in a page over a
// video), and not when that one is transparent, small, under it, or hidden.
func TestIsInvisibleCovered(t *testing.T) {
	check := func(name string, want bool, bg image.Image, z int, hidden bool) {
		b := NewBody()
		b.Styler(func(s *styles.Style) { s.Min.Set(units.Dp(400), units.Dp(300)) })
		target := NewFrame(b)
		target.Styler(func(s *styles.Style) { s.Min.Set(units.Dp(200), units.Dp(100)) })
		inner := NewFrame(target) // (its own children never cover it)
		inner.Styler(func(s *styles.Style) {
			s.Position = styles.PositionAbsolute
			s.Inset.Set(units.Zero())
			s.InsetSet.Set(true)
			s.Background = colors.Uniform(colors.Black)
		})
		overlay := NewFrame(b)
		overlay.Styler(func(s *styles.Style) {
			s.Position = styles.PositionFixed
			s.Inset.Set(units.Zero())
			s.InsetSet.Set(true)
			s.Background = bg
			s.ZIndex = z
		})
		if hidden {
			overlay.SetState(true, states.Invisible)
		}
		b.RenderImage(func() {}, func() {
			b.AsyncLock()
			defer b.AsyncUnlock()
			assert.True(t, target.IsVisible(), name)
			assert.Equal(t, want, target.IsInvisible(), name)
		})
	}
	black := colors.Uniform(colors.Black)
	check("opaque overlay", true, black, 0, false)
	check("translucent overlay", true, colors.Uniform(colors.WithAF32(colors.Black, 0.5)), 0, false)
	check("transparent overlay", false, nil, 0, false)
	check("overlay under it", false, black, -1, false)
	check("hidden overlay", false, black, 0, true)
}
