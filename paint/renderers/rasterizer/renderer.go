// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package rasterizer

import (
	"image"

	"cogentcore.org/core/math32"
	"cogentcore.org/core/paint"
	"cogentcore.org/core/styles/units"
	"golang.org/x/image/vector"
)

// Renderer is the overall renderer for rasterizer.
type Renderer struct {
	// Size is the size of the render target.
	Size math32.Vector2

	// Image is the image we are rendering to.
	Image *image.RGBA

	ras *vector.Rasterizer // reuse
}

// New returns a new rasterx Renderer, rendering to given image.
func New(size math32.Vector2, img *image.RGBA) paint.Renderer {
	rs := &Renderer{Size: size, Image: img}
	// psz := size.ToPointCeil()
	rs.ras = &vector.Rasterizer{}
	return rs
}

// RenderSize returns the size of the render target, in dots (pixels).
func (rs *Renderer) RenderSize() (units.Units, math32.Vector2) {
	return units.UnitDot, rs.Size
}

// Render is the main rendering function.
func (rs *Renderer) Render(r paint.Render) {
	for _, ri := range r {
		switch x := ri.(type) {
		case *paint.Path:
			rs.RenderPath(x)
		}
	}
}
