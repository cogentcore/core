// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package pdfrender

import (
	"bytes"
	"image"
	"image/color"
	"image/draw"
	"strings"
	"testing"

	"cogentcore.org/core/colors"
	"cogentcore.org/core/colors/gradient"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/paint/pimage"
	"cogentcore.org/core/paint/ppath"
	"cogentcore.org/core/paint/render"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/sides"
	"cogentcore.org/core/styles/units"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestRenderer returns a new renderer of 100x80 dots, at 72 dpi.
func newTestRenderer() *Renderer {
	ctx := units.NewContext()
	ctx.DPI = 72
	return New(math32.Vec2(100, 80), ctx).(*Renderer)
}

// TestRenderer tests rendering paths, contexts, and images to a PDF.
func TestRenderer(t *testing.T) {
	rs := newTestRenderer()
	un, sz := rs.Size()
	assert.Equal(t, units.UnitDot, un)
	assert.Equal(t, math32.Vec2(100, 80), sz)
	assert.Nil(t, rs.Image(), "no image")
	assert.Nil(t, rs.Source(), "nothing rendered yet")

	sty := styles.NewPaint()
	sty.Fill.Color = colors.Uniform(colors.Red)
	sty.Stroke.Color = nil
	ctx := render.NewContext(sty, render.NewBounds(0, 0, 100, 80, sides.Floats{}), nil)
	ctx.Transform = math32.Translate2D(5, 5)

	img := image.NewRGBA(image.Rect(0, 0, 4, 4))
	draw.Draw(img, img.Bounds(), image.NewUniform(color.RGBA{0, 0, 255, 255}), image.Point{}, draw.Src)
	big := image.NewRGBA(image.Rect(0, 0, 8, 8)) // opaque, so it has no mask
	draw.Draw(big, big.Bounds(), image.NewUniform(color.RGBA{0, 255, 0, 255}), image.Point{}, draw.Src)
	anchored := pimage.NewDraw(image.Rect(10, 10, 20, 20), colors.Uniform(colors.Green), nil, image.Point{}, draw.Over)
	anchored.Anchor = "here"

	var r render.Render
	r = append(r, &render.ContextPush{Context: *ctx})
	r = append(r, render.NewPath(*ppath.New().Rectangle(0, 0, 10, 10), sty, ctx))
	r = append(r, &render.ContextPop{})
	r = append(r, anchored)
	r = append(r, pimage.NewDraw(image.Rect(30, 30, 34, 34), img, nil, image.Point{}, draw.Over))
	r = append(r, pimage.NewDraw(image.Rect(40, 40, 44, 44), img, big, image.Point{}, draw.Over))
	r = append(r, pimage.NewTransform(math32.Rotate2D(0.3), math32.Identity2(), image.Rect(50, 50, 54, 54), img, nil, draw.Over))
	r = append(r, pimage.NewDraw(image.Rect(0, 0, 4, 4), gradient.NewLinear().AddStop(colors.Red, 0).AddStop(colors.Blue, 1), nil, image.Point{}, draw.Over))
	rs.Render(r)

	src := string(rs.Source())
	require.NotEmpty(t, src)
	assert.True(t, strings.HasPrefix(src, "%PDF-1.7"))
	assert.Contains(t, src, "/MediaBox[0 0 100 80]")
	assert.Contains(t, src, "1 0 0 rg", "the red path")
	assert.Contains(t, src, " q", "the context is pushed")
	assert.Contains(t, src, " Q", "and popped")
	assert.Contains(t, src, "/Dests", "the anchor")
	assert.Equal(t, 2, strings.Count(src, "/Subtype/Image"), "the image, and the larger original for the scaled one")
	assert.Contains(t, src, "/Width 8", "the original, as it is larger")
	assert.Contains(t, src, "/Pattern cs", "the gradient image fills its rectangle")

	rs.SetSize(units.UnitPt, math32.Vec2(50, 50))
	un, sz = rs.Size()
	assert.Equal(t, units.UnitPt, un)
	var b bytes.Buffer
	rs.StartRender(&b)
	rs.AddPage()
	rs.EndRender()
	assert.Contains(t, b.String(), "/Count 2", "two pages")
	assert.Contains(t, b.String(), "/MediaBox[0 0 50 50]")
}
