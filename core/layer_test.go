// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"image"
	"image/color"
	"testing"

	"cogentcore.org/core/colors"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/abilities"
	"cogentcore.org/core/styles/units"
	"cogentcore.org/core/tree"
	"github.com/stretchr/testify/assert"
)

// layerTestBox is a box that counts its renders, rendered as a whole (as a
// video) if group is set.
type layerTestBox struct {
	Frame
	group   bool
	renders int
}

func (tb *layerTestBox) RenderGroup() bool { return tb.group }

func (tb *layerTestBox) Render() {
	tb.renders++
	tb.Frame.Render()
}

// layerTestScene makes a body with a box rendered as a whole (a video) of the
// given color, under a fixed translucent overlay (the backdrop of a dialog of a
// web page) with a white dialog box in its bottom right corner.
func layerTestScene(c color.RGBA) (b *Body, video, dialog *layerTestBox) {
	b = NewBody()
	b.Styler(func(s *styles.Style) { s.Min.Set(units.Dp(400), units.Dp(300)) })
	video = tree.New[layerTestBox](b)
	video.group = true
	video.Styler(func(s *styles.Style) {
		s.Min.Set(units.Dp(200), units.Dp(150))
		s.Background = colors.Uniform(c)
	})
	overlay := NewFrame(b)
	overlay.Styler(func(s *styles.Style) {
		s.Position = styles.PositionFixed
		s.Inset.Set(units.Zero())
		s.InsetSet.Set(true)
		s.Background = colors.Uniform(colors.WithAF32(colors.Black, 0.5))
	})
	dialog = tree.New[layerTestBox](overlay)
	dialog.Styler(func(s *styles.Style) {
		s.Position = styles.PositionAbsolute
		s.Inset.Right.Zero()
		s.Inset.Bottom.Zero()
		s.InsetSet.Right, s.InsetSet.Bottom = true, true
		s.Min.Set(units.Dp(100), units.Dp(50))
		s.Background = colors.Uniform(colors.White)
	})
	return
}

// TestLayerRenderGroup tests that a widget rendered as a whole (as a
// video) under a fixed translucent overlay (as the backdrop of a dialog of a
// web page) re-renders alone into the image of the scene, without rendering
// the overlay, which is in the layer of the scene (see sceneLayer) and is
// composed over it, so that the image shows both, the same as rendering the
// whole scene into one image does; and that a change in the overlay renders
// the layer again, without rendering the video.
func TestLayerRenderGroup(t *testing.T) {
	red, blue := colors.Red, colors.Blue
	b, video, dialog := layerTestScene(red)
	sc := b.Scene
	var videoRenders, dialogRenders, layerRenders int
	var vbb, dbb image.Rectangle
	img := b.RenderImage(func() {
		b.AsyncLock()
		defer b.AsyncUnlock()
		assert.True(t, sc.layer.active, "the scene has a layer")
		assert.False(t, video.hasFlag(widgetInLayer), "the video is not in the layer")
		assert.True(t, dialog.hasFlag(widgetInLayer), "the dialog is in the layer")
		assert.False(t, video.isCovered(), "the layer does not cover the video for rendering")
		videoRenders, dialogRenders, layerRenders = video.renders, dialog.renders, sc.layer.renders
		vbb = video.Geom.TotalBBox
		// a new frame of the video
		video.Styles.Background = colors.Uniform(blue)
		video.NeedsRender()
	})
	// (the window is closed: nothing else uses the widgets now)
	assert.Equal(t, videoRenders+1, video.renders, "the video is rendered")
	assert.Equal(t, dialogRenders, dialog.renders, "the dialog is not rendered")
	assert.Equal(t, layerRenders, sc.layer.renders, "the layer is not rendered")
	if !assert.NotNil(t, img) {
		return
	}
	// the new frame under the backdrop
	assertColorNear(t, img.At(vbb.Min.X+5, vbb.Min.Y+5), color.RGBA{0, 0, 128, 255}, "the video under the backdrop")

	// the same as one image of the whole scene
	layersEnabled = false
	b2, _, _ := layerTestScene(blue)
	img2 := b2.RenderImage()
	layersEnabled = true
	if assert.NotNil(t, img2) {
		assertSameImage(t, img2, img)
	}

	// a change in the dialog renders the layer, not the video
	b3, video3, dialog3 := layerTestScene(red)
	img3 := b3.RenderImage(func() {
		b3.AsyncLock()
		defer b3.AsyncUnlock()
		videoRenders, layerRenders = video3.renders, b3.Scene.layer.renders
		dbb = dialog3.Geom.TotalBBox
		dialog3.Styles.Background = colors.Uniform(colors.Green)
		dialog3.NeedsRender()
	})
	assert.Equal(t, videoRenders, video3.renders, "the video is not rendered")
	assert.Equal(t, layerRenders+1, b3.Scene.layer.renders, "the layer is rendered")
	if assert.NotNil(t, img3) {
		assertColorNear(t, img3.At(dbb.Min.X+5, dbb.Min.Y+5), colors.Green, "the new dialog")
	}
}

// assertColorNear asserts that the given colors are the same within 2.
func assertColorNear(t *testing.T, got color.Color, want color.RGBA, msg string) {
	t.Helper()
	g := colors.AsRGBA(got)
	near := func(a, b uint8) bool { return max(a, b)-min(a, b) <= 2 }
	if !near(g.R, want.R) || !near(g.G, want.G) || !near(g.B, want.B) || !near(g.A, want.A) {
		t.Errorf("%s: got %v, want %v", msg, g, want)
	}
}

// assertSameImage asserts that the given images are the same within 2 in each
// channel of each pixel.
func assertSameImage(t *testing.T, want, got *image.RGBA) {
	t.Helper()
	if !assert.Equal(t, want.Bounds(), got.Bounds()) {
		return
	}
	bad := 0
	for y := want.Rect.Min.Y; y < want.Rect.Max.Y; y++ {
		for x := want.Rect.Min.X; x < want.Rect.Max.X; x++ {
			w, g := want.RGBAAt(x, y), got.RGBAAt(x, y)
			d := func(a, b uint8) bool { return max(a, b)-min(a, b) > 2 }
			if d(w.R, g.R) || d(w.G, g.G) || d(w.B, g.B) || d(w.A, g.A) {
				if bad == 0 {
					t.Errorf("first differing pixel at %d,%d: want %v, got %v", x, y, w, g)
				}
				bad++
			}
		}
	}
	if bad > 0 {
		t.Errorf("%d pixels differ", bad)
	}
}

// TestLayerEvents tests that the widgets under the mouse are looked for only
// in the layer of the scene (see sceneLayer) when one of the widgets in it is
// under the mouse (as on a dialog of a web page), with the same widget getting
// the events as when all of the widgets of the scene are gone through, and in
// all of them when none of those in the layer is under it (as when the overlay
// has no pointer events).
func TestLayerEvents(t *testing.T) {
	for _, noPointer := range []bool{false, true} {
		b, video, dialog := layerTestScene(colors.Red)
		overlay := AsWidget(dialog.Parent)
		if noPointer {
			overlay.Styler(func(s *styles.Style) {
				s.SetAbilities(true, abilities.NoPointerEvents)
			})
		}
		sc := b.Scene
		b.RenderImage(func() {
			b.AsyncLock()
			defer b.AsyncUnlock()
			em := &sc.Events
			// the widget at the position from all of the widgets of the scene
			all := func(pos image.Point) Widget {
				em.mouseInBBox = nil
				em.getMouseInBBox(sc, pos)
				ws := sortByRenderLayer(em.mouseInBBox)
				if len(ws) == 0 {
					return nil
				}
				return ws[len(ws)-1]
			}
			vbb, dbb := video.Geom.TotalBBox, dialog.Geom.TotalBBox
			onVideo := vbb.Min.Add(image.Pt(5, 5))
			onDialog := dbb.Min.Add(image.Pt(5, 5))
			elsewhere := image.Pt(dbb.Min.X-5, vbb.Max.Y+5)
			for _, pos := range []image.Point{onVideo, onDialog, elsewhere} {
				assert.Equal(t, all(pos), sc.WidgetAt(pos), "noPointer %v at %v", noPointer, pos)
			}
			assert.Equal(t, Widget(dialog), sc.WidgetAt(onDialog), "the dialog")
			em.mouseInBBox = nil
			em.getMouseInScene(sc, onVideo)
			if noPointer {
				assert.Equal(t, Widget(video), sc.WidgetAt(onVideo), "the video through the overlay")
			} else {
				assert.Equal(t, overlay.This, sc.WidgetAt(onVideo), "the overlay over the video")
				assert.NotContains(t, em.mouseInBBox, Widget(video), "the video is not gone through")
			}
		})
	}
}
