// Copyright 2024 Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gpudraw

import (
	"image"
	"image/color"
	"image/draw"

	"cogentcore.org/core/math32"
)

// Fill fills given color to render target, to given destination region dr.
// op is the drawing operation: Src = copy source directly (blit),
// Over = alpha blend with existing
func (dw *Drawer) Fill(clr color.Color, dr image.Rectangle, op draw.Op) {
	dw.FillTransform(clr, math32.Identity3(), dr, op)
}

// FillTransform fills given color to to render target, using
// xform to mapping source to destination coordinates (translation, scaling),
// reg is the region to fill.
// op is the drawing operation: Src = copy source directly (blit),
// Over = alpha blend with existing
func (dw *Drawer) FillTransform(clr color.Color, xform math32.Matrix3, reg image.Rectangle, op draw.Op) {
	dsz := dw.DestSize()
	tmat := ConfigMatrix(dsz, xform, reg.Max, reg, false)
	clr4 := math32.NewVector4Color(clr)
	clr4.ToSlice(tmat.UVP[:], 12) // last column holds color
	dw.addOp(Fill, tmat)
}
