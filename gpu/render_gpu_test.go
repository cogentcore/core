// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gpu

import (
	"image"
	"image/color"
	"testing"

	"cogentcore.org/core/base/iox/imagex"
	"github.com/oliverbestmann/webgpu/wgpu"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestRender returns a new graphics system rendering to a render
// texture of the given size, without multisampling, so that the pixels
// are exact away from edges.
func newTestRender(t *testing.T, size image.Point) (*GPU, *RenderTexture, *GraphicsSystem) {
	gp, dev, err := NoDisplayGPU()
	require.NoError(t, err)
	rt := NewRenderTexture(gp, dev, size, 1, Depth32)
	sy := NewGraphicsSystem(gp, "test", rt)
	return gp, rt, sy
}

// grabImage ends the render pass and returns the rendered image.
func grabImage(t *testing.T, sy *GraphicsSystem, rp *wgpu.RenderPassEncoder) *image.NRGBA {
	img := sy.EndRenderPassGrabImage(rp)
	require.NotNil(t, img)
	return img
}

// assertPixel asserts that the pixel of the image at the given position
// is near the given color.
func assertPixel(t *testing.T, img image.Image, x, y int, want color.RGBA, msg string) {
	t.Helper()
	got := color.RGBAModel.Convert(img.At(x, y)).(color.RGBA)
	assert.True(t, imagex.ColorsEqual(got, want, 6), "%s: at (%d, %d), got %v, want %v", msg, x, y, got, want)
}

// TestRenderClear tests clearing a render texture, and changing its size.
func TestRenderClear(t *testing.T) {
	SkipIfNoGPU(t)
	gp, rt, sy := newTestRender(t, image.Pt(64, 48))
	defer gp.Release()
	sy.SetClearColor(color.RGBA{10, 200, 30, 255})
	sy.Config()
	rt.CurrentFrame().ConfigReadBuffer()

	rp, err := sy.BeginRenderPass()
	require.NoError(t, err)
	rp.End()
	img := grabImage(t, sy, rp)
	assert.Equal(t, image.Rect(0, 0, 64, 48), img.Bounds())
	assertPixel(t, img, 5, 5, color.RGBA{10, 200, 30, 255}, "clear color")
	assertPixel(t, img, 63, 47, color.RGBA{10, 200, 30, 255}, "clear color")

	sy.SetSize(image.Pt(32, 20))
	assert.Equal(t, image.Pt(32, 20), rt.Size())
	rt.CurrentFrame().ConfigReadBuffer()
	sy.SetClearColor(color.RGBA{255, 0, 0, 255})
	rp, err = sy.BeginRenderPass()
	require.NoError(t, err)
	rp.End()
	img = grabImage(t, sy, rp)
	assert.Equal(t, image.Rect(0, 0, 32, 20), img.Bounds())
	assertPixel(t, img, 16, 10, color.RGBA{255, 0, 0, 255}, "after resize")
	sy.Release()
}

// quadShader draws a quad in clip coordinates, with a texture and a
// uniform tint, which multiplies the color of the texture.
const quadShader = `
struct Tint {
	color: vec4<f32>,
}

@group(1) @binding(0)
var<uniform> tint: Tint;

struct VertexInput {
	@location(0) position: vec2<f32>,
	@location(1) tex_coord: vec2<f32>,
}

struct VertexOutput {
	@builtin(position) clip_position: vec4<f32>,
	@location(0) tex_coord: vec2<f32>,
}

@vertex
fn vs_main(in: VertexInput) -> VertexOutput {
	var out: VertexOutput;
	out.clip_position = vec4<f32>(in.position, 0.0, 1.0);
	out.tex_coord = in.tex_coord;
	return out;
}

@group(0) @binding(0)
var t_tex: texture_2d<f32>;
@group(0) @binding(1)
var s_tex: sampler;

@fragment
fn fs_main(in: VertexOutput) -> @location(0) vec4<f32> {
	return textureSample(t_tex, s_tex, in.tex_coord) * tint.color;
}
`

// quadSystem is a graphics system that draws [quadShader].
type quadSystem struct {
	gp   *GPU
	rt   *RenderTexture
	sy   *GraphicsSystem
	pl   *GraphicsPipeline
	tex  *Var
	tint *Var
}

// newQuadSystem returns a new [quadSystem] with the given number of
// texture values, drawing a quad over the whole render texture.
func newQuadSystem(t *testing.T, ntex int) *quadSystem {
	qs := &quadSystem{}
	qs.gp, qs.rt, qs.sy = newTestRender(t, image.Pt(80, 60))
	qs.sy.SetClearColor(color.Black)
	qs.pl = qs.sy.AddGraphicsPipeline("quad")
	qs.pl.SetCullMode(wgpu.CullModeNone)
	sh := qs.pl.AddShader("quad")
	require.NoError(t, sh.OpenCode(quadShader))
	qs.pl.AddEntry(sh, VertexShader, "vs_main")
	qs.pl.AddEntry(sh, FragmentShader, "fs_main")
	assert.NotNil(t, qs.pl.VertexEntry())
	assert.NotNil(t, qs.pl.FragmentEntry())
	assert.Equal(t, sh, qs.pl.ShaderByName("quad"))

	vars := qs.sy.Vars()
	vgp := vars.AddVertexGroup()
	tgp := vars.AddGroup(SampledTexture)
	ugp := vars.AddGroup(Uniform)
	posv := vgp.Add("Pos", Float32Vector2, 0, VertexShader)
	tcv := vgp.Add("TexCoord", Float32Vector2, 0, VertexShader)
	idxv := vgp.Add("Index", Uint16, 0, VertexShader)
	idxv.Role = Index
	qs.tex = tgp.Add("Tex", TextureRGBA32, 1, FragmentShader)
	qs.tint = ugp.Add("Tint", Float32Vector4, 1, FragmentShader)
	vgp.SetNValues(1)
	tgp.SetNValues(ntex)
	ugp.SetNValues(1)
	qs.sy.Config()

	// the whole clip space; texture coordinate (0,0) is the top left
	require.NoError(t, SetValueFrom(posv.Values.Values[0], []float32{-1, -1, 1, -1, 1, 1, -1, 1}))
	require.NoError(t, SetValueFrom(tcv.Values.Values[0], []float32{0, 1, 1, 1, 1, 0, 0, 0}))
	require.NoError(t, SetValueFrom(idxv.Values.Values[0], []uint16{0, 1, 2, 0, 2, 3}))
	qs.setTint(t, 1, 1, 1, 1)
	return qs
}

func (qs *quadSystem) setTint(t *testing.T, r, g, b, a float32) {
	require.NoError(t, SetValueFrom(qs.tint.Values.Values[0], []float32{r, g, b, a}))
}

// draw draws the quad and returns the image.
func (qs *quadSystem) draw(t *testing.T) *image.NRGBA {
	qs.rt.CurrentFrame().ConfigReadBuffer()
	rp, err := qs.sy.BeginRenderPass()
	require.NoError(t, err)
	require.NoError(t, qs.pl.BindPipeline(rp))
	qs.pl.BindDrawIndexed(rp)
	rp.End()
	return grabImage(t, qs.sy, rp)
}

func (qs *quadSystem) release() {
	qs.sy.Release()
	qs.gp.Release()
}

// quadrantImage returns an image with a different color in each of its
// quadrants: red, green (top), blue, white (bottom).
func quadrantImage(n int) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, n, n))
	h := n / 2
	for y := range n {
		for x := range n {
			var c color.RGBA
			switch {
			case x < h && y < h:
				c = color.RGBA{255, 0, 0, 255}
			case y < h:
				c = color.RGBA{0, 255, 0, 255}
			case x < h:
				c = color.RGBA{0, 0, 255, 255}
			default:
				c = color.RGBA{255, 255, 255, 255}
			}
			img.SetRGBA(x, y, c)
		}
	}
	return img
}

// TestRenderTexture tests sampling a texture made from a Go image, with
// a uniform that tints it, and switching between textures.
func TestRenderTexture(t *testing.T) {
	SkipIfNoGPU(t)
	qs := newQuadSystem(t, 2)
	defer qs.release()

	qs.tex.Values.Values[0].SetFromGoImage(quadrantImage(16), 0)
	solid := image.NewRGBA(image.Rect(0, 0, 4, 4))
	for i := range solid.Pix {
		solid.Pix[i] = 128
	}
	tx := qs.tex.Values.Values[1].SetFromGoImage(solid, 0)
	require.NotNil(t, tx)
	assert.Equal(t, image.Pt(4, 4), tx.Format.Size)

	img := qs.draw(t)
	// the centers of the quadrants, away from the filtered edges
	assertPixel(t, img, 20, 15, color.RGBA{255, 0, 0, 255}, "top left")
	assertPixel(t, img, 60, 15, color.RGBA{0, 255, 0, 255}, "top right")
	assertPixel(t, img, 20, 45, color.RGBA{0, 0, 255, 255}, "bottom left")
	assertPixel(t, img, 60, 45, color.RGBA{255, 255, 255, 255}, "bottom right")

	qs.setTint(t, 0, 1, 0, 1) // only green
	img = qs.draw(t)
	assertPixel(t, img, 20, 15, color.RGBA{0, 0, 0, 255}, "tinted red")
	assertPixel(t, img, 60, 45, color.RGBA{0, 255, 0, 255}, "tinted white")

	qs.setTint(t, 1, 1, 1, 1)
	_, err := qs.sy.Vars().SetCurrentValue(0, "Tex", 1)
	require.NoError(t, err)
	img = qs.draw(t)
	assertPixel(t, img, 40, 30, color.RGBA{128, 128, 128, 255}, "the second texture (the alpha of the render is opaque)")
}

// TestTextureReadback tests copying a texture to a read buffer and
// reading it back as an image and data.
func TestTextureReadback(t *testing.T) {
	SkipIfNoGPU(t)
	gp, rt, sy := newTestRender(t, image.Pt(70, 30)) // a width that needs row padding
	defer gp.Release()
	sy.SetClearColor(color.RGBA{0, 0, 255, 255})
	sy.Config()
	fr := rt.CurrentFrame()
	require.NoError(t, fr.ConfigReadBuffer())

	rp, err := sy.BeginRenderPass()
	require.NoError(t, err)
	rp.End()
	rt.ReadFrame(sy.CommandEncoder)
	sy.EndRenderPass(rp)

	var data []byte
	require.NoError(t, fr.ReadData(&data, true))
	assert.Len(t, data, 70*30*4, "the padding is removed")
	assert.Equal(t, []byte{0, 0, 255, 255}, data[4*69:4*70], "the last pixel of the first row")
	img, err := fr.ReadGoImage()
	require.NoError(t, err)
	assertPixel(t, img, 69, 29, color.RGBA{0, 0, 255, 255}, "read image")

	dims := NewTextureBufferDims(image.Pt(70, 30))
	assert.False(t, dims.HasNoPadding())
	assert.Greater(t, dims.PaddedSize(), dims.UnpaddedSize())
	assert.True(t, NewTextureBufferDims(image.Pt(64, 30)).HasNoPadding())
	sy.Release()
}
