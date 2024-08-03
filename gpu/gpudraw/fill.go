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

// FillRect fills given color to render target, to given region.
// op is the drawing operation: Src = copy source directly (blit),
// Over = alpha blend with existing
func (dw *Drawer) FillRect(clr color.Color, reg image.Rectangle, op draw.Op) {
	dw.Fill(clr, math32.Identity3(), reg, op)
}

// Fill fills given color to to render target.
// src2dst is the transform mapping source to destination
// coordinates (translation, scaling),
// reg is the region to fill
// op is the drawing operation: Src = copy source directly (blit),
// Over = alpha blend with existing
func (dw *Drawer) Fill(clr color.Color, src2dst math32.Matrix3, reg image.Rectangle, op draw.Op) {
	dsz := dw.DestSize()
	tmat := dw.ConfigMatrix(src2dst, dsz, reg, op, false)
	clr4 := math32.NewVector4Color(clr)
	clr4.ToSlice(tmat.UVP[:], 12) // last column holds color
	dw.addOp(Fill, tmat)
}
