// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package colors

import (
	"image"
	"image/color"
)

// Render represents a color used for rendering. It can either be a solid color or a [Func].
// If Func is nil, it is a solid color; otherwise, it is a [Func].
type Render struct {
	Solid color.RGBA
	Func  Func

	// If non-zero, points outside of Clip will be clipped and represented as [Transparent]
	Clip image.Rectangle
}

// SolidRender returns a new [Render] corresponding to the given solid color.
func SolidRender(solid color.Color) *Render {
	return &Render{Solid: AsRGBA(solid)}
}

// FuncRender returns a new [Render] corresponding to the given color [Func].
func FuncRender(f Func) *Render {
	return &Render{Func: f}
}

// At returns the color that should be used for rendering at the given point.
func (r *Render) At(x, y int) color.Color {
	p := image.Pt(x, y)
	if r.Clip != (image.Rectangle{}) && !p.In(r.Clip) {
		return Transparent
	}
	if r.Func != nil {
		return r.Func(x, y)
	}
	return r.Solid
}
