// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package canvasrast

import (
	"image"

	"cogentcore.org/core/math32"
	"cogentcore.org/core/paint/pimage"
	"cogentcore.org/core/paint/ptext"
	"cogentcore.org/core/paint/render"
	"cogentcore.org/core/paint/renderers/rasterx"
	"cogentcore.org/core/paint/renderers/rasterx/scan"
	"cogentcore.org/core/styles/units"
	"golang.org/x/image/vector"
)

type Renderer struct {
	size  math32.Vector2
	image *image.RGBA

	useRasterx bool

	ras *vector.Rasterizer

	// scan Filler
	Filler *rasterx.Filler
	// scan scanner
	Scanner *scan.Scanner
	// scan spanner
	ImgSpanner *scan.ImgSpanner
}

func New(size math32.Vector2) render.Renderer {
	rs := &Renderer{}
	rs.useRasterx = false

	rs.SetSize(units.UnitDot, size)
	if !rs.useRasterx {
		rs.ras = &vector.Rasterizer{}
		// rs.ras.DrawOp = draw.Src // makes no diff on performance
	}
	return rs
}

func (rs *Renderer) IsImage() bool      { return true }
func (rs *Renderer) Image() *image.RGBA { return rs.image }
func (rs *Renderer) Code() []byte       { return nil }

func (rs *Renderer) Size() (units.Units, math32.Vector2) {
	return units.UnitDot, rs.size
}

func (rs *Renderer) SetSize(un units.Units, size math32.Vector2) {
	if rs.size == size {
		return
	}
	rs.size = size
	psz := size.ToPointCeil()
	rs.image = image.NewRGBA(image.Rectangle{Max: psz})
	if rs.useRasterx {
		rs.ImgSpanner = scan.NewImgSpanner(rs.image)
		rs.Scanner = scan.NewScanner(rs.ImgSpanner, psz.X, psz.Y)
		rs.Filler = rasterx.NewFiller(psz.X, psz.Y, rs.Scanner)
	}
}

func (rs *Renderer) Render(r render.Render) {
	for _, ri := range r {
		switch x := ri.(type) {
		case *render.Path:
			rs.RenderPath(x)
		case *pimage.Params:
			x.Render(rs.image)
		case *ptext.Text:
			x.Render(rs.image, rs)
		}
	}
}
