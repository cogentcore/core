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
	"cogentcore.org/core/tree"
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
	// (a fixed overlay is in the layer of the scene, through which a
	// translucent one is seen, see sceneLayer)
	check("translucent overlay", false, colors.Uniform(colors.WithAF32(colors.Black, 0.5)), 0, false)
	check("transparent overlay", false, nil, 0, false)
	check("overlay under it", false, black, -1, false)
	check("hidden overlay", false, black, 0, true)
}

// testGroup is a widget rendered as a whole (as a video), which counts its
// renders.
type testGroup struct {
	Frame
	renders int
}

func (tg *testGroup) RenderGroup() bool { return true }

func (tg *testGroup) Render() {
	tg.renders++
	tg.Frame.Render()
}

// TestRenderGroupCovered tests that a widget rendered as a whole (as a
// video) is not rendered for a change in it while it is covered by a
// widget rendered over it, which it would paint over (as the videos of the
// search page of pexels.com, under the dialog of one of them), and is once
// it is not. The overlay is absolute, so it is not in the layer of the
// scene (see sceneLayer), but in its image, which the group would paint over.
func TestRenderGroupCovered(t *testing.T) {
	b := NewBody()
	b.Styler(func(s *styles.Style) { s.Min.Set(units.Dp(400), units.Dp(300)) })
	tg := tree.New[testGroup](b)
	tg.Styler(func(s *styles.Style) { s.Min.Set(units.Dp(200), units.Dp(100)) })
	child := NewText(tg).SetText("time")
	overlay := NewFrame(b)
	overlay.Styler(func(s *styles.Style) {
		s.Position = styles.PositionAbsolute
		s.Inset.Set(units.Zero())
		s.InsetSet.Set(true)
		s.Background = colors.Uniform(colors.WithAF32(colors.Black, 0.5))
	})
	b.RenderImage(func() {}, func() {
		b.AsyncLock()
		defer b.AsyncUnlock()
		n := tg.renders
		child.NeedsRender()
		b.Scene.doNeedsRender()
		assert.Equal(t, n, tg.renders, "not rendered while covered")
		overlay.SetState(true, states.Invisible)
		child.NeedsRender()
		b.Scene.doNeedsRender()
		assert.Equal(t, n+1, tg.renders, "rendered once not covered")
	})
}

// testOverlay is a frame that counts its renders.
type testOverlay struct {
	Frame
	renders int
}

func (to *testOverlay) Render() {
	to.renders++
	to.Frame.Render()
}

// TestRenderGroupOverlay tests that the widgets that are over a part of a
// widget rendered as a whole (as the controls of a player of a page over
// its playing video) are rendered again after it, which it paints over.
func TestRenderGroupOverlay(t *testing.T) {
	b := NewBody()
	b.Styler(func(s *styles.Style) { s.Min.Set(units.Dp(400), units.Dp(300)) })
	box := NewFrame(b)
	box.Styler(func(s *styles.Style) {
		s.Position = styles.PositionRelative
		s.Min.Set(units.Dp(300), units.Dp(200))
	})
	tg := tree.New[testGroup](box)
	tg.Styler(func(s *styles.Style) { s.Min.Set(units.Dp(300), units.Dp(200)) })
	child := NewText(tg).SetText("frame")
	bar := tree.New[testOverlay](box)
	bar.Styler(func(s *styles.Style) {
		s.Position = styles.PositionAbsolute
		s.Inset.Bottom = units.Zero()
		s.Inset.Left = units.Zero()
		s.InsetSet.Bottom = true
		s.InsetSet.Left = true
		s.Min.Set(units.Dp(300), units.Dp(30))
		s.Background = colors.Uniform(colors.WithAF32(colors.Black, 0.5))
	})
	b.RenderImage(func() {}, func() {
		b.AsyncLock()
		defer b.AsyncUnlock()
		n, bn := tg.renders, bar.renders
		child.NeedsRender()
		b.Scene.doNeedsRender()
		assert.Equal(t, n+1, tg.renders, "the group is rendered")
		assert.Equal(t, bn+1, bar.renders, "and the bar over it after it")
	})
}
