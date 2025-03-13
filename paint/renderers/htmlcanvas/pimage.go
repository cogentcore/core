// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// This is adapted from https://github.com/tdewolff/canvas
// Copyright (c) 2015 Taco de Wolff, under an MIT License.

//go:build js

package htmlcanvas

import (
	"image"
	"image/color"
	"image/draw"
	"syscall/js"

	"cogentcore.org/core/colors"
	"cogentcore.org/core/colors/gradient"
	"cogentcore.org/core/paint/pimage"
	"github.com/cogentcore/webgpu/wgpu"
)

func (rs *Renderer) RenderImage(pr *pimage.Params) {
	nilSrc := pr.Source == nil
	if r, ok := pr.Source.(*image.RGBA); ok && r == nil {
		nilSrc = true
	}
	if pr.Rect == (image.Rectangle{}) {
		pr.Rect = image.Rectangle{Max: rs.size.ToPoint()}
	}

	// Fast path for [image.Uniform]
	if u, ok := pr.Source.(*image.Uniform); nilSrc || ok && pr.Mask == nil {
		if nilSrc {
			rs.style.Fill.Color = colors.Uniform(color.Transparent)
		} else {
			rs.style.Fill.Color = u
		}
		rs.ctx.Set("fillStyle", rs.imageToStyle(u))
		rs.ctx.Call("fillRect", pr.Rect.Min.X, pr.Rect.Min.Y, pr.Rect.Dx(), pr.Rect.Dy())
		return
	}

	if gr, ok := pr.Source.(gradient.Gradient); ok {
		_ = gr
		// TODO: fill with gradient
		// rs.style.Fill.Color = u
		// rs.ctx.Set("fillStyle", rs.imageToStyle(u))
		// rs.ctx.Call("fillRect", pr.Rect.Min.X, pr.Rect.Min.Y, pr.Rect.Dx(), pr.Rect.Dy())
		return
	}

	if nimg, ok := pr.Source.(*image.NRGBA); ok {
		_ = nimg
		// TODO: this happens on docs/color
		return
	}

	// TODO: images possibly comparatively not performant on web, so there
	// might be a better path for things like FillBox.
	// TODO: have a fast path for [image.RGBA]?
	// size := pr.Rect.Size() // TODO: is this right?
	// TODO: clean this up
	jsBuf := wgpu.BytesToJS(pr.Source.(*image.RGBA).Pix)
	sbb := pr.Source.Bounds()
	imageData := js.Global().Get("ImageData").New(jsBuf, sbb.Dx(), sbb.Dy())

	sw := min(pr.Rect.Dx(), sbb.Dx())
	sh := min(pr.Rect.Dy(), sbb.Dy())
	// fmt.Println(pr.Cmd, pr.Rect, pr.Op, pr.SourcePos, sw, sh, sbb)

	// origin := m.Dot(canvas.Point{0, float64(img.Bounds().Size().Y)}).Mul(rs.dpm)
	// m = m.Scale(rs.dpm, rs.dpm)
	// rs.ctx.Call("setTransform", m[0][0], m[0][1], m[1][0], m[1][1], origin.X, rs.height-origin.Y)
	if pr.Op == draw.Over {
		imageBitmapPromise := js.Global().Call("createImageBitmap", imageData)
		imageBitmap, ok := jsAwait(imageBitmapPromise)
		if !ok {
			panic("error while waiting for createImageBitmap promise")
		}
		rs.ctx.Call("drawImage", imageBitmap, pr.SourcePos.X, pr.SourcePos.Y, sw, sh, pr.Rect.Min.X, pr.Rect.Min.Y, sw, sh)
	} else {
		rs.ctx.Call("putImageData", imageData, pr.Rect.Min.X, pr.Rect.Min.Y, pr.SourcePos.X, pr.SourcePos.Y, sw, sh)
	}
	// rs.ctx.Call("putImageData", imageData, pr.Rect.Min.X, pr.Rect.Min.Y)
	// rs.ctx.Call("setTransform", 1.0, 0.0, 0.0, 1.0, 0.0, 0.0)
}
