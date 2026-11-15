// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gpudraw

import (
	"image"
	"image/color"
	"image/draw"
	"testing"

	"cogentcore.org/core/base/iox/imagex"
	"cogentcore.org/core/gpu"
	"cogentcore.org/core/math32"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// drawTest is a drawer that draws to a render texture.
type drawTest struct {
	gp *gpu.GPU
	rt *gpu.RenderTexture
	dw *Drawer
}

// newDrawTest returns a new [drawTest] with a render texture of the given size.
func newDrawTest(t *testing.T, size image.Point) *drawTest {
	gpu.SkipIfNoGPU(t)
	dt := &drawTest{}
	gp, dev, err := gpu.NoDisplayGPU()
	require.NoError(t, err)
	dt.gp = gp
	dt.rt = gpu.NewRenderTexture(gp, dev, size, 1, gpu.UndefinedType)
	dt.dw = NewDrawer(gp, dt.rt)
	return dt
}

func (dt *drawTest) release() {
	dt.dw.Release()
	dt.rt.Release()
	dt.gp.Release()
}

// image reads back the render texture after the last End.
func (dt *drawTest) image(t *testing.T) *image.NRGBA {
	fr := dt.rt.CurrentFrame()
	require.NoError(t, fr.ConfigReadBuffer())
	sy := dt.dw.System
	cmd, err := sy.NewCommandEncoder()
	require.NoError(t, err)
	dt.rt.ReadFrame(cmd)
	buf, err := cmd.TryFinish(nil)
	require.NoError(t, err)
	sy.Device().Queue.Submit(buf)
	buf.Release()
	cmd.Release()
	img, err := fr.ReadGoImage()
	require.NoError(t, err)
	return img
}

// assertPixel asserts that the pixel of the image at the given position
// is near the given color.
func assertPixel(t *testing.T, img image.Image, x, y int, want color.RGBA, msg string) {
	t.Helper()
	got := color.RGBAModel.Convert(img.At(x, y)).(color.RGBA)
	assert.True(t, imagex.ColorsEqual(got, want, 6), "%s: at (%d, %d), got %v, want %v", msg, x, y, got, want)
}

// solidImage returns an image of the given size and color.
func solidImage(w, h int, c color.RGBA) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	draw.Draw(img, img.Bounds(), image.NewUniform(c), image.Point{}, draw.Src)
	return img
}

var (
	red   = color.RGBA{255, 0, 0, 255}
	green = color.RGBA{0, 255, 0, 255}
	blue  = color.RGBA{0, 0, 255, 255}
)

// TestDrawerFill tests filling rectangles, with Src and Over.
func TestDrawerFill(t *testing.T) {
	dt := newDrawTest(t, image.Pt(100, 80))
	defer dt.release()
	dw := dt.dw
	assert.Equal(t, image.Pt(100, 80), dw.DestSize())
	assert.Equal(t, image.Rect(0, 0, 100, 80), dw.DestBounds())
	assert.Equal(t, dw, dw.AsGPUDrawer())
	assert.Equal(t, dt.rt, dw.Renderer())

	dw.Start()
	dw.Fill(red, image.Rect(0, 0, 100, 80), draw.Src)
	dw.Fill(blue, image.Rect(10, 10, 40, 40), draw.Src)
	dw.Fill(color.RGBA{0, 128, 0, 128}, image.Rect(50, 10, 90, 40), draw.Over) // half green, premultiplied
	dw.Copy(image.Pt(10, 50), image.NewUniform(green), image.Rect(0, 0, 20, 20), draw.Src, Unchanged)
	dw.End()
	img := dt.image(t)

	assertPixel(t, img, 5, 5, red, "background fill")
	assertPixel(t, img, 25, 25, blue, "src fill")
	// blending is in linear space, which is 187 in sRGB for one half
	assertPixel(t, img, 70, 25, color.RGBA{187, 188, 0, 255}, "over fill blends half green on red")
	assertPixel(t, img, 20, 60, green, "copy of a uniform is a fill")
	assertPixel(t, img, 35, 60, red, "outside of the uniform copy")
	assertPixel(t, img, 95, 75, red, "corner")
}

// TestDrawerImages tests copying, scaling, and transforming images.
func TestDrawerImages(t *testing.T) {
	dt := newDrawTest(t, image.Pt(100, 80))
	defer dt.release()
	dw := dt.dw

	// left half red, right half blue
	img := image.NewRGBA(image.Rect(0, 0, 8, 4))
	draw.Draw(img, image.Rect(0, 0, 4, 4), image.NewUniform(red), image.Point{}, draw.Src)
	draw.Draw(img, image.Rect(4, 0, 8, 4), image.NewUniform(blue), image.Point{}, draw.Src)
	gimg := solidImage(10, 10, green)

	dw.Start()
	dw.Scale(image.Rectangle{}, img, image.Rectangle{}, 0, draw.Src, Changed) // whole destination
	dw.Copy(image.Pt(45, 5), gimg, image.Rectangle{}, draw.Src, Changed)
	dw.Copy(image.Pt(10, 60), gimg, image.Rect(0, 0, 5, 5), draw.Src, Unchanged) // a part
	dw.End()
	out := dt.image(t)
	assertPixel(t, out, 20, 40, red, "scaled left half")
	assertPixel(t, out, 80, 40, blue, "scaled right half")
	assertPixel(t, out, 50, 10, green, "copy")
	assertPixel(t, out, 12, 62, green, "copy of a part")
	assertPixel(t, out, 17, 67, red, "outside of the part")

	// rotated by 180: the halves are swapped
	dw.Start()
	dw.Scale(image.Rectangle{}, img, image.Rectangle{}, 180, draw.Src, Unchanged)
	dw.End()
	out = dt.image(t)
	assertPixel(t, out, 20, 40, blue, "rotated left half")
	assertPixel(t, out, 80, 40, red, "rotated right half")

	// a transform that translates and scales by 2
	dw.Start()
	dw.Fill(color.Black, dw.DestBounds(), draw.Src)
	dw.Transform(math32.Matrix3{2, 0, 0, 0, 2, 0, 20, 30, 1}, gimg, image.Rectangle{}, draw.Src, Unchanged)
	dw.End()
	out = dt.image(t)
	assertPixel(t, out, 38, 48, green, "inside the transformed image")
	assertPixel(t, out, 45, 55, color.RGBA{0, 0, 0, 255}, "beyond the transformed image")
	assertPixel(t, out, 15, 25, color.RGBA{0, 0, 0, 255}, "before the transformed image")

	dw.Redraw()
	assertPixel(t, dt.image(t), 38, 48, green, "redraw")
}

// TestDrawerOver tests drawing a translucent image over another.
func TestDrawerOver(t *testing.T) {
	dt := newDrawTest(t, image.Pt(40, 40))
	defer dt.release()
	dw := dt.dw
	half := solidImage(20, 20, color.RGBA{0, 0, 128, 128}) // half blue, premultiplied

	dw.Start()
	dw.Fill(red, dw.DestBounds(), draw.Src)
	dw.Copy(image.Pt(0, 0), half, image.Rectangle{}, draw.Over, Changed)
	dw.Copy(image.Pt(20, 20), half, image.Rectangle{}, draw.Src, Unchanged)
	dw.End()
	out := dt.image(t)
	assertPixel(t, out, 10, 10, color.RGBA{187, 0, 128, 255}, "over blends (in linear space)")
	assertPixel(t, out, 30, 10, red, "outside")
}

// TestDrawerUsed tests drawing with a used texture, flipped in Y.
func TestDrawerUsed(t *testing.T) {
	dt := newDrawTest(t, image.Pt(40, 40))
	defer dt.release()
	dw := dt.dw

	// top half red, bottom half blue
	img := image.NewRGBA(image.Rect(0, 0, 4, 4))
	draw.Draw(img, image.Rect(0, 0, 4, 2), image.NewUniform(red), image.Point{}, draw.Src)
	draw.Draw(img, image.Rect(0, 2, 4, 4), image.NewUniform(blue), image.Point{}, draw.Src)

	dw.Start()
	dw.UseGoImage(img, Changed)
	dw.ScaleUsed(image.Rectangle{}, image.Rectangle{}, 0, draw.Src, false)
	dw.End()
	out := dt.image(t)
	// at a quarter and three quarters: the sampler repeats, so the
	// edges blend with the opposite edges
	assertPixel(t, out, 20, 10, red, "top")
	assertPixel(t, out, 20, 30, blue, "bottom")

	dw.Start()
	dw.UseGoImage(img, Unchanged)
	dw.ScaleUsed(image.Rectangle{}, image.Rectangle{}, 0, draw.Src, true)
	dw.End()
	out = dt.image(t)
	assertPixel(t, out, 20, 10, blue, "flipped top")
	assertPixel(t, out, 20, 30, red, "flipped bottom")
}

// TestDrawerManyImages tests drawing more images and operations than
// the initial allocation of the drawer.
func TestDrawerManyImages(t *testing.T) {
	dt := newDrawTest(t, image.Pt(100, 100))
	defer dt.release()
	dw := dt.dw
	n := AllocChunk + 4
	imgs := make([]*image.RGBA, n)
	for i := range imgs {
		imgs[i] = solidImage(5, 5, color.RGBA{uint8(i * 10), 0, 255 - uint8(i*10), 255})
	}
	dw.Start()
	dw.Fill(color.Black, dw.DestBounds(), draw.Src)
	for i, img := range imgs {
		dw.Copy(image.Pt((i%10)*10, (i/10)*10), img, image.Rectangle{}, draw.Src, Changed)
	}
	dw.End()
	out := dt.image(t)
	for i := range imgs {
		assertPixel(t, out, (i%10)*10+2, (i/10)*10+2, imgs[i].RGBAAt(0, 0), "image")
	}
}
