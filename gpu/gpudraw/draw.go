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

const (
	// Unchanged should be used for the unchanged argument in drawer calls,
	// when the caller knows that the image is unchanged.
	Unchanged = true

	// Changed should be used for the unchanged argument to drawer calls,
	// when the image has changed since last time or its status is unknown
	Changed
)

// Copy copies the given Go source image to the render target, with the
// same semantics as golang.org/x/image/draw.Copy, with the destination
// implicit in the Drawer target.
//   - Must have called Start first!
//   - dp is the destination point.
//   - src is the source image. If an image.Uniform, fast Fill is done.
//   - sr is the source region, if zero full src is used; must have for Uniform.
//   - op is the drawing operation: Src = copy source directly (blit),
//     Over = alpha blend with existing.
//   - unchanged should be true if caller knows that this image is unchanged
//     from the last time it was used -- saves re-uploading to gpu.
func (dw *Drawer) Copy(dp image.Point, src image.Image, sr image.Rectangle, op draw.Op, unchanged bool) {
	if u, ok := src.(*image.Uniform); ok {
		dr := sr
		del := dp.Sub(sr.Min)
		dr.Min = dp
		dr.Max = dr.Max.Add(del)
		dw.Fill(u.At(0, 0), dr, op)
		return
	}
	dw.UseGoImage(src, unchanged)
	dw.CopyUsed(dp, sr, op, false)
}

// Scale copies the given Go source image to the render target,
// scaling the region defined by src and sr to the destination
// such that sr in src-space is mapped to dr in dst-space,
// and applying an optional rotation of the source image.
// Has the same general semantics as golang.org/x/image/draw.Scale, with the
// destination implicit in the Drawer target.
// If src image is an
//   - Must have called Start first!
//   - dr is the destination rectangle; if zero uses full dest image.
//   - src is the source image. Uniform does not work (or make sense) here.
//   - sr is the source region, if zero full src is used; must have for Uniform.
//   - rotateDeg = rotation degrees to apply in the mapping:
//     90 = left, -90 = right, 180 = invert.
//   - op is the drawing operation: Src = copy source directly (blit),
//     Over = alpha blend with existing.
//   - unchanged should be true if caller knows that this image is unchanged
//     from the last time it was used -- saves re-uploading to gpu.
func (dw *Drawer) Scale(dr image.Rectangle, src image.Image, sr image.Rectangle, rotateDeg float32, op draw.Op, unchanged bool) {
	dw.UseGoImage(src, unchanged)
	dw.ScaleUsed(dr, sr, rotateDeg, op, false)
}

// Transform copies the given Go source image to the render target,
// with the same semantics as golang.org/x/image/draw.Transform, with the
// destination implicit in the Drawer target.
//   - xform is the transform mapping source to destination coordinates.
//   - src is the source image. Uniform does not work (or make sense) here.
//   - sr is the source region, if zero full src is used; must have for Uniform.
//   - op is the drawing operation: Src = copy source directly (blit),
//     Over = alpha blend with existing.
//   - unchanged should be true if caller knows that this image is unchanged
//     from the last time it was used -- saves re-uploading to gpu.
func (dw *Drawer) Transform(xform math32.Matrix3, src image.Image, sr image.Rectangle, op draw.Op, unchanged bool) {
	dw.UseGoImage(src, unchanged)
	dw.TransformUsed(xform, sr, op, false)
}
