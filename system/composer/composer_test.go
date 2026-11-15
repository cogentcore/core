// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build !js

package composer

import (
	"image"
	"image/color"
	"image/draw"
	"testing"

	"cogentcore.org/core/math32"
	"github.com/stretchr/testify/assert"
)

// testDrawer is a [Drawer] that draws to an image with [DrawerBase],
// recording the calls of its other methods.
type testDrawer struct {
	DrawerBase
	calls []string
}

func (td *testDrawer) Start()  { td.calls = append(td.calls, "start") }
func (td *testDrawer) End()    { td.calls = append(td.calls, "end") }
func (td *testDrawer) Redraw() { td.calls = append(td.calls, "redraw") }

// fillSource is a [Source] that fills a rectangle with a color.
type fillSource struct {
	r   image.Rectangle
	clr color.RGBA
}

func (fs *fillSource) Draw(c Composer) {
	cd := c.(*ComposerDrawer)
	cd.Drawer.Copy(fs.r.Min, image.NewUniform(fs.clr), fs.r, draw.Src, Unchanged)
}

var (
	red  = color.RGBA{255, 0, 0, 255}
	blue = color.RGBA{0, 0, 255, 255}
)

func TestComposerDrawer(t *testing.T) {
	td := &testDrawer{Image: image.NewRGBA(image.Rect(0, 0, 20, 20))}
	cd := &ComposerDrawer{Drawer: td}
	var _ Composer = cd

	cd.Start()
	cd.Add(&fillSource{image.Rect(0, 0, 10, 10), red}, nil)
	cd.Add(nil, nil) // ignored
	cd.Add(&fillSource{image.Rect(5, 5, 15, 15), blue}, nil)
	assert.Len(t, cd.Sources, 2)
	cd.Compose()
	assert.Equal(t, []string{"start", "end"}, td.calls)
	assert.Equal(t, red, td.Image.RGBAAt(2, 2))
	assert.Equal(t, blue, td.Image.RGBAAt(7, 7), "later sources are drawn over earlier ones")
	assert.Equal(t, color.RGBA{}, td.Image.RGBAAt(17, 17), "only the region of the source is filled")

	cd.Redraw()
	assert.Equal(t, "redraw", td.calls[2])
	cd.Start()
	assert.Empty(t, cd.Sources, "start resets the sources")
}

func TestDrawerBaseCopy(t *testing.T) {
	dw := &DrawerBase{Image: image.NewRGBA(image.Rect(0, 0, 10, 10))}
	src := image.NewRGBA(image.Rect(0, 0, 4, 4))
	draw.Draw(src, image.Rect(0, 0, 2, 4), image.NewUniform(red), image.Point{}, draw.Src)
	draw.Draw(src, image.Rect(2, 0, 4, 4), image.NewUniform(blue), image.Point{}, draw.Src)

	dw.Copy(image.Pt(1, 1), src, image.Rectangle{}, draw.Src, Changed)
	assert.Equal(t, red, dw.Image.RGBAAt(1, 1), "a zero region copies all of the source")
	assert.Equal(t, blue, dw.Image.RGBAAt(4, 4))
	assert.Equal(t, color.RGBA{}, dw.Image.RGBAAt(5, 5))

	dw.Image = image.NewRGBA(image.Rect(0, 0, 10, 10))
	dw.Copy(image.Pt(6, 6), src, image.Rect(2, 0, 4, 2), draw.Src, Changed)
	assert.Equal(t, blue, dw.Image.RGBAAt(6, 6), "the region is copied from its min")
	assert.Equal(t, blue, dw.Image.RGBAAt(7, 7))
	assert.Equal(t, color.RGBA{}, dw.Image.RGBAAt(8, 8), "only the size of the region is copied")

	half := color.RGBA{0, 0, 128, 128}
	dw.Copy(image.Pt(6, 6), image.NewUniform(half), image.Rect(6, 6, 7, 7), draw.Over, Unchanged)
	assert.Equal(t, color.RGBA{0, 0, 255, 255}, dw.Image.RGBAAt(6, 6), "over blends with what is there")
	dw.Copy(image.Pt(0, 0), image.NewUniform(half), image.Rect(0, 0, 1, 1), draw.Src, Unchanged)
	assert.Equal(t, half, dw.Image.RGBAAt(0, 0), "src replaces")
	assert.Equal(t, color.RGBA{}, dw.Image.RGBAAt(1, 1))
}

func TestDrawerBaseScaleTransform(t *testing.T) {
	dw := &DrawerBase{Image: image.NewRGBA(image.Rect(0, 0, 10, 10))}
	dw.Start()
	assert.Nil(t, dw.Renderer())
	src := image.NewUniform(red)
	dw.Scale(image.Rect(2, 2, 4, 4), src, image.Rect(0, 0, 2, 2), 0, draw.Src, Changed)
	assert.Equal(t, red, dw.Image.RGBAAt(3, 3))
	assert.Equal(t, color.RGBA{}, dw.Image.RGBAAt(5, 5))
	dw.Transform(math32.Identity3(), image.NewUniform(blue), image.Rect(6, 6, 8, 8), draw.Src, Changed)
	assert.Equal(t, blue, dw.Image.RGBAAt(7, 7))
}
