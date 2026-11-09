// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package paint_test

import (
	"image"
	"image/color"
	"testing"

	"cogentcore.org/core/colors"
	"cogentcore.org/core/math32"
	. "cogentcore.org/core/paint"
	"cogentcore.org/core/paint/render"
	_ "cogentcore.org/core/paint/renderers"
	"github.com/stretchr/testify/assert"
)

// renderLayer renders a red square (10..90) in a context with the clip
// mask, mask, and opacity of the given functions, which paint with pc,
// and returns the image.
func renderLayer(t *testing.T, clip, mask func(pc *Painter), opacity float32) *image.RGBA {
	size := math32.Vec2(100, 100)
	pc := NewPainter(size)
	pc.PushContext(nil, nil)
	var cr, mr render.Render
	if clip != nil {
		cr = pc.RenderSeparately(func() { clip(pc) })
	}
	if mask != nil {
		mr = pc.RenderSeparately(func() { mask(pc) })
	}
	pc.SetLayer(cr, mr, opacity)
	pc.Fill.Color = colors.Uniform(colors.Red)
	pc.Stroke.Color = nil
	pc.Rectangle(10, 10, 80, 80)
	pc.Draw()
	pc.Rectangle(30, 30, 40, 40) // overlaps: the same with group opacity
	pc.Draw()
	pc.PopContext()
	img := NewImageRenderer(size).Render(pc.RenderDone()).Image()
	return img.(*image.RGBA)
}

// TestLayerClipMask tests the clipping of the rendering of a context to
// the shapes of its clip mask, whatever their colors.
func TestLayerClipMask(t *testing.T) {
	img := renderLayer(t, func(pc *Painter) {
		pc.Fill.Color = colors.Uniform(color.RGBA{0, 0, 255, 255})
		pc.Stroke.Color = nil
		pc.Circle(50, 50, 30)
		pc.Draw()
	}, nil, 0)
	assert.Equal(t, color.RGBA{255, 0, 0, 255}, img.RGBAAt(50, 50), "inside")
	assert.Equal(t, color.RGBA{255, 0, 0, 255}, img.RGBAAt(50, 22), "inside, near the edge")
	assert.Equal(t, color.RGBA{}, img.RGBAAt(15, 15), "in the square, outside of the circle")
	assert.Equal(t, color.RGBA{}, img.RGBAAt(50, 5), "outside of both")
}

// TestLayerMask tests masking by the luminance times the alpha of the
// rendering of the mask.
func TestLayerMask(t *testing.T) {
	img := renderLayer(t, nil, func(pc *Painter) {
		pc.Stroke.Color = nil
		pc.Fill.Color = colors.Uniform(colors.White)
		pc.Rectangle(0, 0, 50, 100)
		pc.Draw()
		pc.Fill.Color = colors.Uniform(color.RGBA{0, 0, 0, 255}) // black: masked out
		pc.Rectangle(50, 0, 25, 100)
		pc.Draw()
		pc.Fill.Color = colors.Uniform(color.RGBA{128, 128, 128, 255}) // gray: half
		pc.Rectangle(75, 0, 25, 100)
		pc.Draw()
	}, 0)
	assert.Equal(t, color.RGBA{255, 0, 0, 255}, img.RGBAAt(20, 50), "white")
	assert.Equal(t, color.RGBA{}, img.RGBAAt(60, 50), "black")
	c := img.RGBAAt(85, 50)
	assert.InDelta(t, 128, int(c.A), 2, "gray")
	assert.Equal(t, color.RGBA{}, img.RGBAAt(5, 5), "outside of the square")
}

// TestLayerOpacity tests the opacity of a group: its items that overlap
// do not show through each other.
func TestLayerOpacity(t *testing.T) {
	img := renderLayer(t, nil, nil, 0.5)
	inner, outer := img.RGBAAt(50, 50), img.RGBAAt(15, 15)
	assert.InDelta(t, 128, int(outer.A), 1)
	assert.Equal(t, outer, inner, "overlapping items have the opacity of the group")
}
