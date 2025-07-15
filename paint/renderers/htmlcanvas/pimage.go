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

	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/base/iox/imagex"
	"cogentcore.org/core/colors"
	"cogentcore.org/core/colors/gradient"
	"cogentcore.org/core/paint/pimage"
)

func (rs *Renderer) RenderImage(pr *pimage.Params) {
	usrc := imagex.Unwrap(pr.Source)
	umask := imagex.Unwrap(pr.Mask)

	nilSrc := usrc == nil
	if pr.Rect == (image.Rectangle{}) {
		pr.Rect = image.Rectangle{Max: rs.size.ToPoint()}
	}

	// todo: handle masks!

	// Fast path for [image.Uniform]
	if u, ok := usrc.(*image.Uniform); nilSrc || ok && umask == nil {
		if u == nil || u.C == colors.Transparent {
			rs.ctx.Call("clearRect", pr.Rect.Min.X, pr.Rect.Min.Y, pr.Rect.Dx(), pr.Rect.Dy())
		} else {
			rs.ctx.Set("fillStyle", rs.imageToStyle(u))
			rs.ctx.Call("fillRect", pr.Rect.Min.X, pr.Rect.Min.Y, pr.Rect.Dx(), pr.Rect.Dy())
		}
		return
	}

	if gr, ok := usrc.(gradient.Gradient); ok {
		rs.curRect = pr.Rect
		rs.ctx.Set("fillStyle", rs.imageToStyle(gr))
		rs.ctx.Call("fillRect", pr.Rect.Min.X, pr.Rect.Min.Y, pr.Rect.Dx(), pr.Rect.Dy())
		return
	}

	ji := pr.Source.(*imagex.JSRGBA)
	if ji.JS.Bitmap.IsNull() || ji.JS.Bitmap.IsUndefined() {
		errors.Log(errors.New("htmlcanvas.pimage: nil JS bitmap in image"))
		return
	}

	sbb := pr.Source.Bounds()
	sw := min(pr.Rect.Dx(), sbb.Dx())
	sh := min(pr.Rect.Dy(), sbb.Dy())
	// fmt.Println(pr.Cmd, pr.Rect, pr.Op, pr.SourcePos, sw, sh, sbb)

	if pr.Op == draw.Over || (ji.JS.Data.IsUndefined() || ji.JS.Data.IsNull()) {
		rs.ctx.Call("drawImage", ji.JS.Bitmap, pr.SourcePos.X, pr.SourcePos.Y, sw, sh, pr.Rect.Min.X, pr.Rect.Min.Y, sw, sh)
	} else {
		rs.ctx.Call("putImageData", ji.JS.Data, pr.Rect.Min.X, pr.Rect.Min.Y, pr.SourcePos.X, pr.SourcePos.Y, sw, sh)
	}
}
