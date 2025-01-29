// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package rasterizer

import (
	"image"

	"cogentcore.org/core/math32"
	"cogentcore.org/core/paint"
	"cogentcore.org/core/paint/pimage"
	"cogentcore.org/core/styles/units"
	"golang.org/x/image/vector"
)

type Renderer struct {
	size  math32.Vector2
	image *image.RGBA
	ras   *vector.Rasterizer
}

func New(size math32.Vector2, img *image.RGBA) paint.Renderer {
	psz := size.ToPointCeil()
	if img == nil {
		img = image.NewRGBA(image.Rectangle{Max: psz})
	}
	rs := &Renderer{size: size, image: img}
	rs.ras = &vector.Rasterizer{}
	return rs
}

func (rs *Renderer) IsImage() bool      { return true }
func (rs *Renderer) Image() *image.RGBA { return rs.image }
func (rs *Renderer) Code() []byte       { return nil }

func (rs *Renderer) Size() (units.Units, math32.Vector2) {
	return units.UnitDot, rs.size
}

func (rs *Renderer) SetSize(un units.Units, size math32.Vector2, img *image.RGBA) {
	if rs.size == size {
		return
	}
	rs.size = size
	if img != nil {
		rs.image = img
		return
	}
	psz := size.ToPointCeil()
	rs.image = image.NewRGBA(image.Rectangle{Max: psz})
}

func (rs *Renderer) Render(r paint.Render) {
	for _, ri := range r {
		switch x := ri.(type) {
		case *paint.Path:
			rs.RenderPath(x)
		case *pimage.Params:
			x.Render(rs.image)
		}
	}
}
