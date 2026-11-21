// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package rasterx_test

import (
	"image"
	"image/color"
	"testing"

	"cogentcore.org/core/base/goscript"
	"cogentcore.org/core/base/iox/imagex"
	"cogentcore.org/core/colors"
	"cogentcore.org/core/colors/gradient"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/paint"
	"cogentcore.org/core/paint/ppath"
	"cogentcore.org/core/paint/render"
	_ "cogentcore.org/core/paint/renderers"
	. "cogentcore.org/core/paint/renderers/rasterx"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/sides"
	"cogentcore.org/core/styles/units"
	"cogentcore.org/core/text/rich"
	"cogentcore.org/core/text/shaped"
	_ "cogentcore.org/core/text/shaped/shapers"
	"cogentcore.org/core/text/text"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// renderPainter renders the painter with a new rasterx renderer.
func renderPainter(t *testing.T, pc *paint.Painter) *image.RGBA {
	t.Helper()
	return renderItems(t, pc.Size, pc.RenderDone())
}

// renderItems renders the given items with a new rasterx renderer.
func renderItems(t *testing.T, size math32.Vector2, r render.Render) *image.RGBA {
	t.Helper()
	rs := New(size).(*Renderer)
	un, sz := rs.Size()
	assert.Equal(t, units.UnitDot, un)
	assert.Equal(t, size, sz)
	assert.Nil(t, rs.Source())
	rs.Render(r)
	img, ok := rs.Image().(*image.RGBA)
	require.True(t, ok)
	return img
}

// newPainter returns a painter of the given size with a white background.
func newPainter(w, h float32) *paint.Painter {
	pc := paint.NewPainter(math32.Vec2(w, h))
	pc.FillBox(math32.Vector2{}, pc.Size, colors.Uniform(colors.White))
	return pc
}

// rgba returns the color of the given pixel.
func rgba(img *image.RGBA, x, y int) color.RGBA {
	return img.RGBAAt(x, y)
}

// TestRendererStrokes tests stroking paths with each cap and join, with
// dashes, scaled by a transform, and filling with a gradient.
func TestRendererStrokes(t *testing.T) {
	pc := newPainter(240, 160)
	caps := []ppath.Caps{ppath.CapButt, ppath.CapRound, ppath.CapSquare}
	joins := []ppath.Joins{ppath.JoinMiter, ppath.JoinMiterClip, ppath.JoinRound, ppath.JoinBevel, ppath.JoinArcs, ppath.JoinArcsClip}
	for i, jn := range joins {
		pc.Stroke.Color = colors.Uniform(colors.Blue)
		pc.Stroke.Width.Dot(6)
		pc.Stroke.Cap = caps[i%len(caps)]
		pc.Stroke.Join = jn
		pc.Fill.Color = nil
		x := 10 + float32(i)*38
		pc.MoveTo(x, 50)
		pc.LineTo(x+15, 15)
		pc.LineTo(x+30, 50)
		pc.Draw()
	}
	pc.Stroke.Dashes = []float32{6, 3}
	pc.Stroke.Cap = ppath.CapButt
	pc.Line(10, 80, 230, 80)
	pc.Draw()
	pc.Stroke.Dashes = nil

	// scaled by a transform, so the width is scaled
	pc.Transform = math32.Scale2D(2, 2)
	pc.Stroke.Width.Dot(2)
	pc.Line(5, 50, 50, 50)
	pc.Draw()
	pc.Transform = math32.Identity2()

	lin := gradient.NewLinear().AddStop(colors.Red, 0).AddStop(colors.Yellow, 1)
	pc.Fill.Color = lin
	pc.Stroke.Color = nil
	pc.Rectangle(150, 110, 80, 40)
	pc.Draw()

	img := renderPainter(t, pc)
	imagex.Assert(t, img, "renderer-strokes")
	assert.Equal(t, colors.Blue, rgba(img, 20, 80), "a dash")
	assert.Equal(t, colors.White, rgba(img, 17, 80), "a gap")
	b := rgba(img, 50, 100)
	assert.Equal(t, uint8(255), b.B, "the scaled line is 4 dots wide")
	assert.Equal(t, colors.White, rgba(img, 50, 104))
	g := rgba(img, 152, 130)
	assert.Equal(t, uint8(255), g.R, "the start of the gradient")
	assert.Less(t, g.G, uint8(40))
}

// TestRendererLayers tests rendering within contexts that are layers: with
// opacity, with a clip mask, and with a luminance mask.
func TestRendererLayers(t *testing.T) {
	pc := newPainter(120, 40)
	red := styles.NewPaint()
	red.Fill.Color = colors.Uniform(colors.Red)
	red.Stroke.Color = nil
	rect := func(x, y, w, h float32) *render.Path {
		return render.NewPath(*ppath.New().Rectangle(x, y, w, h), red, pc.Context())
	}
	layer := func(ctx render.Context, items ...render.Item) {
		pc.Render.Add(&render.ContextPush{Context: ctx})
		for _, it := range items {
			pc.Render.Add(it)
		}
		pc.Render.Add(&render.ContextPop{})
	}
	base := *render.NewContext(red, render.NewBounds(0, 0, 120, 40, sides.Floats{}), nil)

	half := base
	half.Opacity = 0.5
	layer(half, rect(0, 0, 40, 40))

	clip := base
	clip.ClipMask = render.Render{rect(40, 0, 20, 40)} // the left half
	layer(clip, rect(40, 0, 40, 40))

	white := styles.NewPaint()
	white.Fill.Color = colors.Uniform(colors.White)
	white.Stroke.Color = nil
	mask := base
	mask.Mask = render.Render{render.NewPath(*ppath.New().Rectangle(80, 0, 40, 20), white, pc.Context())} // the top half
	layer(mask, rect(80, 0, 40, 40))

	empty := base
	empty.ClipMask = render.Render{rect(0, 0, 0, 0)} // nothing
	layer(empty, rect(0, 0, 120, 40))

	// a context that is not popped is popped at the end
	pc.Render.Add(&render.ContextPush{Context: half})
	pc.Render.Add(rect(0, 30, 10, 10))

	img := renderPainter(t, pc)
	h := rgba(img, 20, 20)
	assert.Equal(t, uint8(255), h.R)
	assert.InDelta(t, 128, h.G, 2, "half opacity over white")
	assert.Equal(t, colors.Red, rgba(img, 50, 20), "within the clip")
	assert.Equal(t, colors.White, rgba(img, 70, 20), "outside of the clip")
	assert.Equal(t, colors.Red, rgba(img, 100, 10), "where the mask is white")
	assert.Equal(t, colors.White, rgba(img, 100, 30), "where the mask is empty")
	assert.InDelta(t, 64, rgba(img, 5, 35).G, 3, "the unpopped half over half")
}

// TestRendererText tests rendering text with decorations, colors, a
// stroke, sub and superscripts, emoji, and in a rotated context.
func TestRendererText(t *testing.T) {
	goscript.Skip(t, "the js fonts have only metrics, so the image renderer cannot draw text glyphs")
	pc := newPainter(320, 140)
	sh := shaped.NewShaper()
	tsty := text.NewStyle()
	tsty.ToDots(&pc.UnitContext)
	draw := func(sty *rich.Style, s string, x, y float32) {
		tx := rich.NewText(sty, []rune(s))
		pc.DrawText(sh.WrapLines(tx, sty, tsty, pc.Size), math32.Vec2(x, y))
	}
	plain := rich.NewStyle()
	under := rich.NewStyle().SetDecoration(rich.Underline, rich.Overline)
	dotted := rich.NewStyle().SetDecoration(rich.DottedUnderline, rich.LineThrough)
	colored := rich.NewStyle().SetFillColor(colors.Blue).SetBackground(colors.Yellow).SetStrokeColor(colors.Red)
	sup := rich.NewStyle()
	sup.Special = rich.Super
	sub := rich.NewStyle()
	sub.Special = rich.Sub
	draw(plain, "Plain", 5, 5)
	draw(under, "Under", 80, 5)
	draw(dotted, "Dotted", 160, 5)
	draw(colored, "Colored", 5, 40)
	draw(sup, "Sup", 120, 40)
	draw(sub, "Sub", 170, 40)
	draw(plain, "emoji 🎉", 5, 80)
	pc.Transform = math32.Rotate2DAround(0.3, math32.Vec2(200, 100))
	draw(plain, "Rotated", 200, 90)
	pc.Transform = math32.Identity2()

	r := pc.RenderDone()
	img := renderItems(t, pc.Size, r)
	imagex.Assert(t, img, "renderer-text")
	// the background of the colored text
	bg := rgba(img, 6, 42)
	assert.Equal(t, uint8(255), bg.R)
	assert.Equal(t, uint8(255), bg.G)
	assert.Less(t, bg.B, uint8(100), "yellow")

	// rendering again uses the cached glyphs
	img2 := renderItems(t, pc.Size, r)
	assert.Equal(t, img.Pix, img2.Pix)
}

// TestRendererEmptyBounds tests that nothing is drawn in a context with
// empty bounds, as for a widget scrolled out of view: the small glyphs of
// punctuation, which are not cached, and paths, which are drawn by the
// scanner, were drawn without clipping.
func TestRendererEmptyBounds(t *testing.T) {
	goscript.Skip(t, "the js fonts have only metrics, so the image renderer cannot draw text glyphs")
	pc := newPainter(100, 60)
	sh := shaped.NewShaper()
	tsty := text.NewStyle()
	tsty.ToDots(&pc.UnitContext)
	sty := rich.NewStyle()
	sty.Size = 2
	pc.PushContext(nil, render.NewBoundsRect(image.Rect(0, 80, 100, 80), sides.Floats{}))
	tx := rich.NewText(sty, []rune(`"a", 'b'; .,:`))
	pc.DrawText(sh.WrapLines(tx, sty, tsty, pc.Size), math32.Vec2(5, 5))
	pc.Fill.Color = colors.Uniform(colors.Black)
	pc.Rectangle(10, 10, 50, 30)
	pc.Draw()
	pc.PopContext()
	img := renderPainter(t, pc)
	for y := range 60 {
		for x := range 100 {
			require.Equal(t, color.RGBA{255, 255, 255, 255}, rgba(img, x, y), "pixel %d, %d", x, y)
		}
	}
}
