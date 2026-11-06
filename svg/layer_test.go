// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package svg_test

import (
	"image"
	"path/filepath"
	"testing"

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
