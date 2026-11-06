// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package svg_test

import (
	"image"
	"image/color"
	"path/filepath"
	"strings"
	"testing"

	"cogentcore.org/core/colors"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/paint"
	_ "cogentcore.org/core/paint/renderers"
	. "cogentcore.org/core/svg"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// renderFile renders the given SVG file of the testdata at 640x480.
func renderFile(t *testing.T, fname string) *image.RGBA {
	size := math32.Vec2(640, 480)
	sv := NewSVG(size)
	require.NoError(t, sv.OpenXML(filepath.Join("testdata", "svg", fname)))
	img := paint.NewImageRenderer(size).Render(sv.Render(nil).RenderDone()).Image()
	return img.(*image.RGBA)
}

// TestClipMask tests clip paths (in user space and objectBoundingBox
// units, with clip-rule, on shapes and groups, in style sheets, with
// transforms), masks (by luminance, with their regions), the opacity of
// groups, and that elements that are not supported (foreignObject) do not
// render, as in browsers (see testdata/svg/clip-mask.svg).
func TestClipMask(t *testing.T) {
	img := renderFile(t, "clip-mask.svg")
	alpha := func(x, y int) uint8 { return img.RGBAAt(x, y).A }
	assert.Equal(t, uint8(255), alpha(100, 100), "in the clip circle")
	assert.Equal(t, uint8(0), alpha(45, 45), "in the rect, outside of the clip circle")
	assert.Equal(t, uint8(255), alpha(260, 140), "in the clip triangle (objectBoundingBox)")
	assert.Equal(t, uint8(0), alpha(205, 45), "outside of the clip triangle")
	assert.Equal(t, uint8(255), alpha(350, 50), "in the ring (evenodd)")
	assert.Equal(t, uint8(0), alpha(400, 100), "in the hole of the ring")
	assert.Equal(t, uint8(255), alpha(560, 100), "in the transformed clip circle")
	assert.Equal(t, uint8(0), alpha(505, 45), "outside of the transformed clip circle")
	assert.Greater(t, alpha(50, 300), uint8(220), "the white end of the mask gradient")
	assert.Less(t, alpha(190, 300), uint8(40), "the black end of the mask gradient")
	assert.Equal(t, uint8(0), alpha(320, 300), "in the black circle of the mask")
	assert.Equal(t, uint8(255), alpha(250, 230), "in the white of the mask")
	assert.InDelta(t, 128, int(alpha(450, 230)), 2, "the group opacity")
	assert.Equal(t, img.RGBAAt(450, 230), img.RGBAAt(520, 300), "overlapping items in a group with opacity")
	assert.Equal(t, uint8(255), alpha(100, 440), "in a link")
	assert.Equal(t, uint8(0), alpha(250, 440), "in a foreignObject")
}

// TestUseSymbol tests use elements: of symbols (scaled from their view
// box to the size of the use), of groups, with transforms, and of
// elements after them (see testdata/svg/use-symbol.svg).
func TestUseSymbol(t *testing.T) {
	img := renderFile(t, "use-symbol.svg")
	alpha := func(x, y int) uint8 { return img.RGBAAt(x, y).A }
	assert.Equal(t, uint8(255), alpha(80, 80), "the use of a circle after it")
	assert.Equal(t, uint8(255), alpha(60, 60), "the circle")
	assert.Equal(t, uint8(255), alpha(250, 70), "the center of the scaled star")
	assert.Equal(t, uint8(0), alpha(205, 25), "the corner of the scaled star")
	assert.Equal(t, uint8(255), alpha(420, 70), "the star centered in a wider use")
	assert.Equal(t, uint8(0), alpha(330, 70), "the star centered in a wider use: not at its left")
	assert.Equal(t, uint8(255), alpha(300, 400), "the use of a group")
	assert.Equal(t, uint8(0), alpha(100, 300), "symbols are not rendered themselves")
}

// TestPattern tests pattern fills and strokes: in user space and
// objectBoundingBox units, with patternTransform, viewBox, and href (see
// testdata/svg/pattern.svg).
func TestPattern(t *testing.T) {
	img := renderFile(t, "pattern.svg")
	at := func(x, y int) [4]uint8 { c := img.RGBAAt(x, y); return [4]uint8{c.R, c.G, c.B, c.A} }
	assert.Equal(t, [4]uint8{255, 0, 0, 255}, at(30, 30), "a dot")
	assert.Equal(t, [4]uint8{255, 255, 224, 255}, at(40, 40), "between dots")
	assert.Equal(t, [4]uint8{0, 0, 0, 255}, at(250, 50), "a black square of the checker (objectBoundingBox)")
	assert.Equal(t, uint8(0), at(270, 50)[3], "a transparent square of the checker")
	assert.Equal(t, [4]uint8{0, 128, 0, 255}, at(45, 285), "a square of the view box pattern")
	assert.Equal(t, uint8(0), at(65, 285)[3], "between the squares of the view box pattern")
	assert.Equal(t, uint8(0), at(275, 285)[3], "the pattern of the href is moved by its x")
	assert.Equal(t, [4]uint8{0, 128, 0, 255}, at(295, 285), "the pattern of the href is moved by its x")
	assert.Equal(t, uint8(0), at(540, 340)[3], "inside of the pattern stroke")
	assert.Equal(t, uint8(255), at(470, 340)[3], "the pattern stroke")
	n := 0
	for x := 430; x < 610; x++ {
		if c := at(x, 110); c[2] > 100 && c[0] < 50 {
			n++
		}
	}
	assert.Greater(t, n, 40, "navy stripes")
	assert.Less(t, n, 140, "stripes, not a solid fill")
}

// TestCurrentColor tests that currentColor is the color property of the
// element or its ancestors (as octicons: fill="currentColor"), or the
// Color of the SVG (as the CSS color of the element of a page that it is
// in), black by default.
func TestCurrentColor(t *testing.T) {
	src := `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 30 10" width="30" height="10" fill="currentColor">` +
		`<rect width="10" height="10"/><g color="#00ff00"><rect x="10" width="10" height="10"/></g>` +
		`<rect x="20" width="10" height="10" style="color: #0000ff; fill: currentColor"/></svg>`
	render := func(clr image.Image) *image.RGBA {
		sv := NewSVG(math32.Vec2(30, 10))
		sv.Color = clr
		require.NoError(t, sv.ReadXML(strings.NewReader(src)))
		return paint.NewImageRenderer(math32.Vec2(30, 10)).Render(sv.Render(nil).RenderDone()).Image().(*image.RGBA)
	}
	img := render(nil)
	assert.Equal(t, color.RGBA{0, 0, 0, 255}, img.RGBAAt(5, 5), "black by default")
	assert.Equal(t, color.RGBA{0, 255, 0, 255}, img.RGBAAt(15, 5), "the color of the group")
	assert.Equal(t, color.RGBA{0, 0, 255, 255}, img.RGBAAt(25, 5), "the color of the style")
	img = render(colors.Uniform(color.RGBA{255, 0, 0, 255}))
	assert.Equal(t, color.RGBA{255, 0, 0, 255}, img.RGBAAt(5, 5), "the color of the SVG")
	assert.Equal(t, color.RGBA{0, 255, 0, 255}, img.RGBAAt(15, 5), "the color of the group")
}

// TestFeatures tests features of SVG compared with browsers (see
// testdata/svg/features.svg): nested svg elements (with their view box,
// clipped to their viewport), percentage lengths, text spans laid out
// after each other, text-anchor, the opacity of text, space-separated
// dashes, and markers with orient auto-start-reverse.
func TestFeatures(t *testing.T) {
	img := renderFile(t, "features.svg")
	at := func(x, y int) color.RGBA { return img.RGBAAt(x, y) }
	assert.Equal(t, color.RGBA{128, 0, 128, 255}, at(80, 80), "the center of the nested svg circle")
	assert.Equal(t, uint8(0), at(150, 80).A, "outside of the nested svg")
	red := 0
	for x := 70; x < 140; x++ {
		for y := 245; y < 275; y++ {
			if c := at(x, y); c.R > 200 && c.G < 60 && c.B < 60 {
				red++
			}
		}
	}
	assert.Greater(t, red, 20, "the red span after the text")
	left := 640
	for x := 0; x < 160; x++ {
		for y := 235; y < 275; y++ {
			if at(x, y).A > 128 && x < left {
				left = x
			}
		}
	}
	assert.Less(t, left, 60, "the text is centered on its x (80)")
	maxA := uint8(0)
	for x := 500; x < 560; x++ {
		for y := 180; y < 205; y++ {
			maxA = max(maxA, at(x, y).A)
		}
	}
	assert.Greater(t, maxA, uint8(64), "the text before its end (560)")
	assert.Less(t, maxA, uint8(160), "the text with an opacity of 0.5")
	assert.Equal(t, uint8(0), at(45, 300).A, "a gap of the dashes")
	assert.Equal(t, uint8(255), at(30, 300).A, "a dash")
	c := at(334, 320)
	assert.True(t, c.R > 150 && c.G < 80, "the start marker points back, its tip at the left: %v", c)
}
