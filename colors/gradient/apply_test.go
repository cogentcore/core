// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gradient

//core:js-test

import (
	"image"
	"image/color"
	"testing"

	"cogentcore.org/core/colors"
	"cogentcore.org/core/math32"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// invert is an [ApplyFunc] that inverts the color.
func invert(c color.Color) color.Color {
	r := colors.AsRGBA(c)
	return color.RGBA{r.A - r.R, r.A - r.G, r.A - r.B, r.A}
}

// testBox is the box of the gradients of the tests.
var testBox = math32.B2(0, 0, 100, 100)

// TestApplyImages tests applying functions and opacity to uniform images,
// gradients, and other images.
func TestApplyImages(t *testing.T) {
	assert.Nil(t, Apply(nil, invert))
	assert.Nil(t, ApplyOpacity(nil, 0.5))

	u := Apply(colors.Uniform(colors.Red), invert)
	assert.Equal(t, color.RGBA{0, 255, 255, 255}, colors.AsRGBA(u.At(0, 0)))
	uo := ApplyOpacity(colors.Uniform(colors.Red), 0.5)
	assert.InDelta(t, 128, colors.AsRGBA(uo.At(0, 0)).A, 1)
	same := colors.Uniform(colors.Red)
	assert.Equal(t, same, ApplyOpacity(same, 1), "opacity 1 is the same image")

	lin := NewLinear().AddStop(colors.Black, 0).AddStop(colors.White, 1)
	lin.Update(1, testBox, math32.Identity2())
	gi := Apply(lin, invert)
	g, ok := gi.(*Linear)
	require.True(t, ok, "a gradient is copied")
	assert.Len(t, lin.ApplyFuncs, 0, "the original is not changed")
	g.Update(1, testBox, math32.Identity2())
	assert.Greater(t, colors.AsRGBA(g.At(1, 50)).R, uint8(245), "black inverted to white")

	go1 := ApplyOpacity(lin, 0.5)
	go1.(Gradient).Update(1, testBox, math32.Identity2())
	assert.InDelta(t, 128, colors.AsRGBA(go1.At(99, 50)).A, 1)

	img := image.NewRGBA(image.Rect(0, 0, 2, 2))
	img.Set(1, 1, colors.Blue)
	ai := Apply(img, invert)
	assert.IsType(t, &Applier{}, ai)
	assert.Equal(t, color.RGBA{255, 255, 0, 255}, colors.AsRGBA(ai.At(1, 1)))
	ao := ApplyOpacity(img, 0.5)
	assert.InDelta(t, 128, colors.AsRGBA(ao.At(1, 1)).A, 1)

	var fs ApplyFuncs
	assert.Nil(t, fs.Clone())
	fs.Add(invert)
	fs.Add(invert)
	assert.Len(t, fs.Clone(), 2)
	assert.Equal(t, colors.AsRGBA(colors.Red), colors.AsRGBA(fs.Apply(colors.Red)), "inverted twice")
}

// TestApplyOpacityToStops tests multiplying the opacities of the stops.
func TestApplyOpacityToStops(t *testing.T) {
	lin := NewLinear().AddStop(colors.Black, 0).AddStop(colors.White, 1, 0.5)
	lin.ApplyOpacityToStops(0.5)
	assert.Equal(t, float32(0.5), lin.Stops[0].Opacity)
	assert.Equal(t, float32(0.25), lin.Stops[1].Opacity)
}

// TestBlendStops tests that a gradient blended in a color space other
// than RGB gets the intermediate stops for that space, and blends
// between them.
func TestBlendStops(t *testing.T) {
	lin := NewLinear().AddStop(colors.Red, 0).AddStop(colors.Blue, 1)
	lin.SetBlend(colors.HCT)
	lin.Update(1, testBox, math32.Identity2())
	assert.Greater(t, len(lin.stopsRGB), 2, "intermediate stops in HCT")
	n := len(lin.stopsRGB)
	lin.Update(1, testBox, math32.Identity2())
	assert.Equal(t, n, len(lin.stopsRGB), "the same stops are kept")

	mid := colors.AsRGBA(lin.At(50, 50))
	rgb := NewLinear().AddStop(colors.Red, 0).AddStop(colors.Blue, 1)
	rgb.Update(1, testBox, math32.Identity2())
	assert.NotEqual(t, colors.AsRGBA(rgb.At(50, 50)), mid, "HCT blends differently from RGB")
	assert.Nil(t, rgb.stopsRGB)

	cp := CopyOf(lin).(*Linear)
	assert.Equal(t, lin.stopsRGB, cp.stopsRGB)
	start, end := lin.TransformedAxis()
	assert.Equal(t, math32.Vec2(0, 0), start)
	assert.Equal(t, math32.Vec2(100, 0), end)
}

// TestRadialFocal tests a radial gradient whose focal point is not its
// center, which is the color of the first stop at the focal point and of
// the last stop at the edge, and beyond it.
func TestRadialFocal(t *testing.T) {
	r := NewRadial().AddStop(colors.Black, 0).AddStop(colors.White, 1)
	r.Focal = math32.Vec2(0.25, 0.5)
	r.Update(1, testBox, math32.Identity2())
	c, f, _ := r.TransformedCoords()
	assert.Less(t, f.X, c.X, "the focal point is left of the center")
	assert.Equal(t, f.Y, c.Y)

	focal := colors.AsRGBA(r.At(24, 49))
	assert.Less(t, focal.R, uint8(30), "black at the focal point")
	edge := colors.AsRGBA(r.At(98, 50))
	assert.Greater(t, edge.R, uint8(220), "white at the edge")
	outside := colors.AsRGBA(r.At(0, 0))
	assert.Equal(t, uint8(255), outside.R, "the last stop outside")

	one := NewRadial().AddStop(colors.Red, 0)
	assert.Equal(t, colors.AsRGBA(colors.Red), colors.AsRGBA(one.At(0, 0)))
	assert.Equal(t, color.RGBA{}, NewRadial().At(0, 0))

	_, ok := rayCircleIntersectionF(math32.Vec2(5, 5), math32.Vec2(6, 5), math32.Vec2(0, 0), 1)
	assert.False(t, ok, "a ray that misses the circle")
}

// TestFromAny tests getting images from values of different types.
func TestFromAny(t *testing.T) {
	img, err := FromAny(colors.Red)
	assert.NoError(t, err)
	assert.Equal(t, colors.AsRGBA(colors.Red), colors.AsRGBA(img.At(0, 0)))
	var c color.Color = colors.Blue
	img, err = FromAny(&c)
	assert.NoError(t, err)
	assert.Equal(t, colors.AsRGBA(colors.Blue), colors.AsRGBA(img.At(0, 0)))
	u := colors.Uniform(colors.Green)
	img, err = FromAny(u)
	assert.NoError(t, err)
	assert.Equal(t, u, img)
	img, err = FromAny("red")
	assert.NoError(t, err)
	assert.Equal(t, colors.AsRGBA(colors.Red), colors.AsRGBA(img.At(0, 0)))
	s := "linear-gradient(red, blue)"
	img, err = FromAny(&s)
	assert.NoError(t, err)
	assert.IsType(t, &Linear{}, img)
	_, err = FromAny(3)
	assert.Error(t, err)

	ClearCache()
	assert.Equal(t, color.RGBAModel, NewLinear().ColorModel())
}
