// Copyright 2018 The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// based on golang.org/x/exp/shiny:
// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package glos

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"

	"github.com/go-gl/gl/v4.1-core/gl"
	"github.com/goki/gi/oswin"
	"github.com/goki/gi/oswin/gpu"
)

type textureImpl struct {
	w    *windowImpl
	id   uint32
	fb   uint32
	size image.Point
}

func (t *textureImpl) Size() image.Point       { return t.size }
func (t *textureImpl) Bounds() image.Rectangle { return image.Rectangle{Max: t.size} }

func (t *textureImpl) Release() {
	theGPU.UseContext(t.w)
	defer theGPU.ClearContext(t.w)

	t.w.DeleteTexture(t)

	if t.fb != 0 {
		gl.DeleteFramebuffers(1, &t.fb)
		t.fb = 0
	}
	gl.DeleteTextures(1, &t.id)
	t.id = 0
}

func (t *textureImpl) Upload(dp image.Point, src oswin.Image, sr image.Rectangle) {
	theApp.RunOnMain(func() {
		t.upload(dp, src, sr)
	})
}

func (t *textureImpl) upload(dp image.Point, src oswin.Image, sr image.Rectangle) {
	buf := src.(*imageImpl)
	buf.preUpload()

	// src2dst is added to convert from the src coordinate space to the dst
	// coordinate space. It is subtracted to convert the other way.
	src2dst := dp.Sub(sr.Min)

	// Clip to the source.
	sr = sr.Intersect(buf.Bounds())

	// Clip to the destination.
	dr := sr.Add(src2dst)
	dr = dr.Intersect(t.Bounds())
	if dr.Empty() {
		return
	}

	// Bring dr.Min in dst-space back to src-space to get the pixel image offset.
	pix := buf.rgba.Pix[buf.rgba.PixOffset(dr.Min.X-src2dst.X, dr.Min.Y-src2dst.Y):]

	theGPU.UseContext(t.w)
	defer theGPU.ClearContext(t.w)

	gl.ActiveTexture(gl.TEXTURE0)
	gl.BindTexture(gl.TEXTURE_2D, t.id)
	gpu.TheGPU.ErrCheck("tex upload tex")

	width := dr.Dx()
	if width*4 == buf.rgba.Stride {
		gl.TexSubImage2D(gl.TEXTURE_2D, 0, int32(dr.Min.X), int32(dr.Min.Y), int32(width), int32(dr.Dy()), gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(pix))
		gpu.TheGPU.ErrCheck("tex subimg")
		fmt.Printf("uploaded tex: dr: %+v\n", dr)
		return
	}
	// TODO: can we use GL_UNPACK_ROW_LENGTH with glPixelStorei for stride in
	// ES 3.0, instead of uploading the pixels row-by-row?
	for y, p := dr.Min.Y, 0; y < dr.Max.Y; y++ {
		gl.TexSubImage2D(gl.TEXTURE_2D, 0, int32(dr.Min.X), int32(y), int32(width), 1, gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(pix[p:]))
		p += buf.rgba.Stride
	}
	fmt.Printf("uploaded tex: dr: %+v\n", dr)
}

func (t *textureImpl) Fill(dr image.Rectangle, src color.Color, op draw.Op) {
	theApp.RunOnMain(func() {
		t.fill(dr, src, op)
	})
}

func (t *textureImpl) fill(dr image.Rectangle, src color.Color, op draw.Op) {
	minX := float64(dr.Min.X)
	minY := float64(dr.Min.Y)
	maxX := float64(dr.Max.X)
	maxY := float64(dr.Max.Y)
	mvp := calcMVP(
		t.size.X, t.size.Y,
		minX, minY,
		maxX, minY,
		minX, maxY,
	)

	theGPU.UseContext(t.w)
	defer theGPU.ClearContext(t.w)

	create := t.fb == 0
	if create {
		gl.GenFramebuffers(1, &t.fb)
	}
	gl.BindFramebuffer(gl.FRAMEBUFFER, t.fb)
	if create {
		gl.FramebufferTexture2D(gl.FRAMEBUFFER, gl.COLOR_ATTACHMENT0, gl.TEXTURE_2D, t.id, 0)
	}

	gl.Viewport(0, 0, int32(t.size.X), int32(t.size.Y))
	doFill(t.w.app, mvp, src, op)
}
