// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package colors

import (
	"image/color"
)

// Render represents a color used for rendering. It can either be a solid color or a [Func].
// If Func is nil, it is a solid color; otherwise, it is a [Func].
type Render struct {
	Func  Func
	Solid color.RGBA
}

// SolidRender returns a new [Render] corresponding to the given solid color.
func SolidRender(solid color.Color) Render {
	return Render{Solid: AsRGBA(solid)}
}

// FuncRender returns a new [Render] corresponding to the given color [Func].
func FuncRender(f Func) Render {
	return Render{Func: f}
}

// SetSolid sets the render color to the given solid [color.Color],
// also setting the Func to nil.
func (r *Render) SetSolid(solid color.Color) {
	r.Solid = AsRGBA(solid)
	r.Func = nil
}
