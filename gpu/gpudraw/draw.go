// Copyright 2024 Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gpudraw

import (
	"image"
	"image/draw"

	"cogentcore.org/core/math32"
)

// this file contains the official image/draw like interface

// Copy copies the given Go source image to the render target, with the
// same semantics as golang.org/x/image/draw.Copy, with the destination
// implicit in the Drawer target.
//   - Must have called StartDraw first!
//   - dp is the destination point.
//   - src is the source image. If an image.Uniform, fast Fill is done.
//   - sr is the source region, if zero full src is used; must have for Uniform.
//   - op is the drawing operation: Src = copy source directly (blit),
//     Over = alpha blend with existing.
func (dw *Drawer) Copy(dp image.Point, src image.Image, sr image.Rectangle, op draw.Op) {
	if u, ok := src.(*image.Uniform); ok {
		dr := sr
		del := dp.Sub(sr.Min)
		dr.Min = dp
		dr.Max.Add(del)
		dw.Fill(u.At(0, 0), dr, op)
		return
	}
	dw.UseGoImage(src)
	dw.CopyUsed(dp, sr, op, false)
}

// Scale copies the given Go source image to the render target,
// scaling the region defined by src and sr to the destination
// such that sr in src-space is mapped to dr in dst-space.
// with the same semantics as golang.org/x/image/draw.Scale, with the
// destination implicit in the Drawer target.
// If src image is an
//   - Must have called StartDraw first!
//   - dr is the destination rectangle; if zero uses full dest image.
//   - src is the source image. Uniform does not work (or make sense) here.
//   - sr is the source region, if zero full src is used; must have for Uniform.
//   - op is the drawing operation: Src = copy source directly (blit),
//     Over = alpha blend with existing.
func (dw *Drawer) Scale(dr image.Rectangle, src image.Image, sr image.Rectangle, op draw.Op) {
	dw.UseGoImage(src)
	dw.ScaleUsed(dr, sr, op, false, 0)
}

// Transform copies the given Go source image to the render target,
// with the same semantics as golang.org/x/image/draw.Transform, with the
// destination implicit in the Drawer target.
//   - xform is the transform mapping source to destination coordinates.
//   - src is the source image. Uniform does not work (or make sense) here.
//   - sr is the source region, if zero full src is used; must have for Uniform.
//   - op is the drawing operation: Src = copy source directly (blit),
//     Over = alpha blend with existing.
func (dw *Drawer) Transform(xform math32.Matrix3, src image.Image, sr image.Rectangle, op draw.Op) {
	dw.UseGoImage(src)
	dw.TransformUsed(xform, sr, op, false)
}
