// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// This is adapted from https://github.com/tdewolff/canvas
// Copyright (c) 2015 Taco de Wolff, under an MIT License.

//go:build js

package htmlcanvas

import (
	"image"
	"image/draw"

	"cogentcore.org/core/colors/gradient"
	"cogentcore.org/core/paint/pimage"
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

	sbb := pr.Source.Bounds()

	sw := min(pr.Rect.Dx(), sbb.Dx())
	sh := min(pr.Rect.Dy(), sbb.Dy())
	// fmt.Println(pr.Cmd, pr.Rect, pr.Op, pr.SourcePos, sw, sh, sbb)

	// origin := m.Dot(canvas.Point{0, float64(img.Bounds().Size().Y)}).Mul(rs.dpm)
	// m = m.Scale(rs.dpm, rs.dpm)
	// rs.ctx.Call("setTransform", m[0][0], m[0][1], m[1][0], m[1][1], origin.X, rs.height-origin.Y)
	if pr.Op == draw.Over {
		imageBitmap := pimage.GetJSImageBitmap(pr)
		if !imageBitmap.IsUndefined() {
			rs.ctx.Call("drawImage", imageBitmap, pr.SourcePos.X, pr.SourcePos.Y, sw, sh, pr.Rect.Min.X, pr.Rect.Min.Y, sw, sh)
		}
	} else {
		imageData := pimage.GetJSImageData(pr)
		if !imageData.IsUndefined() {
			rs.ctx.Call("putImageData", imageData, pr.Rect.Min.X, pr.Rect.Min.Y, pr.SourcePos.X, pr.SourcePos.Y, sw, sh)
		}
	}
}
