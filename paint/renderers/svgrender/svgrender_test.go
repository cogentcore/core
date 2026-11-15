// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package svgrender_test

import (
	"image"
	"image/color"
	"image/draw"
	"strings"
	"testing"

	"cogentcore.org/core/colors"
	"cogentcore.org/core/colors/gradient"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/paint"
	"cogentcore.org/core/paint/pimage"
	_ "cogentcore.org/core/paint/renderers"
	. "cogentcore.org/core/paint/renderers/svgrender"
	"cogentcore.org/core/styles/units"
	"cogentcore.org/core/text/rich"
	"cogentcore.org/core/text/shaped"
	"cogentcore.org/core/text/text"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRenderer tests rendering paths, contexts, text, and images to an
// SVG document, and rendering that back to an image.
func TestRenderer(t *testing.T) {
	size := math32.Vec2(100, 80)
	rs := New(size).(*Renderer)
	un, sz := rs.Size()
	assert.Equal(t, units.UnitDot, un)
	assert.Equal(t, size, sz)
	assert.Nil(t, rs.Image(), "nothing rendered yet")
	assert.Nil(t, rs.Source())

	pc := paint.NewPainter(size)
	pc.Fill.Color = colors.Uniform(colors.Red)
	pc.Rectangle(5, 5, 10, 10)
	pc.Draw()
	pc.PushContext(nil, nil)
	pc.Transform = math32.Translate2D(20, 0)
	pc.Rectangle(0, 0, 10, 10)
	pc.Draw()
	pc.PopContext()
	pc.Transform = math32.Identity2()

	tsty := text.NewStyle()
	fsty := rich.NewStyle()
	tsty.ToDots(&pc.UnitContext)
	tx := rich.NewText(fsty, []rune("Hi there"))
	lns := shaped.NewShaper().WrapLines(tx, fsty, tsty, math32.Vec2(100, 40))
	pc.DrawText(lns, math32.Vec2(5, 30))

	img := image.NewRGBA(image.Rect(0, 0, 4, 4))
	draw.Draw(img, img.Bounds(), image.NewUniform(color.RGBA{0, 0, 255, 255}), image.Point{}, draw.Src)
	big := image.NewRGBA(image.Rect(0, 0, 8, 8))
	pc.DrawImage(img, big, image.Rect(50, 50, 54, 54), image.Point{}, draw.Over)
	pc.DrawImage(colors.Uniform(colors.Green), nil, image.Rect(60, 60, 70, 70), image.Point{}, draw.Over)
	pc.Render.Add(&pimage.Params{Cmd: pimage.Draw, Op: draw.Over})
	pc.DrawImage(gradient.NewLinear().AddStop(colors.Red, 0).AddStop(colors.Blue, 1), nil, image.Rect(0, 0, 4, 4), image.Point{}, draw.Over)
	pc.Render.Add(pimage.NewTransform(math32.Rotate2D(0.3), math32.Identity2(), image.Rect(70, 10, 74, 14), img, nil, draw.Over))

	rs.Render(pc.RenderDone())
	src := string(rs.Source())
	require.NotEmpty(t, src)
	assert.Equal(t, 2, strings.Count(src, "<path"), src)
	assert.Contains(t, src, "translate(20", "the pushed context transform")
	assert.Contains(t, src, ">Hi there<", "the text run")
	assert.Equal(t, 2, strings.Count(src, "<rect"), "the uniform color, and the nil image, which fills the whole size")
	assert.Contains(t, src, `width="100"`)
	assert.Contains(t, src, "fill:#008000", "the green rectangle")
	assert.Contains(t, src, `viewBox="0 0 100 80"`)
	assert.Equal(t, 2, strings.Count(src, "<image"), "the drawn and transformed images")

	im := rs.Image()
	require.NotNil(t, im)
	assert.Equal(t, image.Rect(0, 0, 100, 80), im.Bounds())
	r, _, _, _ := im.At(10, 10).RGBA()
	assert.Equal(t, uint32(0xffff), r, "the red rectangle")

	rs.SetSize(units.UnitDot, math32.Vec2(50, 50))
	_, sz = rs.Size()
	assert.Equal(t, math32.Vec2(50, 50), sz)
}
