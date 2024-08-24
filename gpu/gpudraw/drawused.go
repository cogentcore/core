// Copyright 2024 Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gpudraw

import (
	"image"
	"image/draw"

	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/gpu"
	"cogentcore.org/core/gpu/drawmatrix"
	"cogentcore.org/core/math32"
)

// UseGoImage uses the given Go image.Image as the source image
// for the next Draw operation.  unchanged is a hint from the source
// about whether the image is unchanged from last use, in which case
// it does not need to be re-uploaded (if found).
func (dw *Drawer) UseGoImage(img image.Image, unchanged bool) {
	dw.Lock()
	defer dw.Unlock()
	dw.curImageSize = img.Bounds().Size()
	idx, exists := dw.images.use(img)
	if exists && unchanged {
		return
	}
	tvr := errors.Log1(dw.System.Vars().VarByName(1, "TexSampler"))
	nv := len(tvr.Values.Values)
	if idx >= nv { // new allocation
		tvr.SetNValues(dw.System.Device(), dw.images.capacity)
	}
	tvv := tvr.Values.Values[idx]
	tvv.SetFromGoImage(img, 0)
}

// UseTexture uses the given GPU resident Texture as the source image
// for the next Draw operation.
func (dw *Drawer) UseTexture(tx *gpu.Texture) {
	dw.Lock()
	defer dw.Unlock()
	dw.curImageSize = tx.Format.Bounds().Size()
	idx, _ := dw.images.use(tx)
	//	if exists && unchanged {
	//		return
	//	}
	tvr := errors.Log1(dw.System.Vars().VarByName(1, "TexSampler"))
	nv := len(tvr.Values.Values)
	if idx >= nv { // new allocation
		tvr.SetNValues(dw.System.Device(), dw.images.capacity)
	}
	tvv := tvr.Values.Values[idx]
	tvv.SetFromTexture(tx)
}

// CopyUsed copies the current Use* texture to render target.
// Must have called Start and a Use* method first!
//   - dp is the destination point.
//   - src is the source image. If an image.Uniform, fast Fill is done.
//   - sr is the source region, if zero full src is used.
//   - op is the drawing operation: Src = copy source directly (blit),
//     Over = alpha blend with existing.
//   - flipY = flipY axis when drawing this image.
func (dw *Drawer) CopyUsed(dp image.Point, sr image.Rectangle, op draw.Op, flipY bool) {
	if sr == (image.Rectangle{}) {
		sr.Max = dw.curImageSize
	}
	mat := math32.Matrix3{
		1, 0, 0,
		0, 1, 0,
		float32(dp.X - sr.Min.X), float32(dp.Y - sr.Min.Y), 1,
	}
	dw.TransformUsed(mat, sr, op, flipY)
}

// ScaleUsed copies the current Use* texture to render target,
// scaling the region defined by src and sr to the destination
// such that sr in src-space is mapped to dr in dst-space.
// Must have called Start and a Use* method first!
//   - dr is the destination rectangle; if zero uses full dest image.
//   - sr is the source region; if zero uses full src image.
//   - rotateDeg = rotation degrees to apply in the mapping:
//     90 = left, -90 = right, 180 = invert.
//   - op is the drawing operation: Src = copy source directly (blit),
//     Over = alpha blend with existing.
//   - flipY = flipY axis when drawing this image.
func (dw *Drawer) ScaleUsed(dr image.Rectangle, sr image.Rectangle, rotateDeg float32, op draw.Op, flipY bool) {
	if dr == (image.Rectangle{}) {
		dr.Max = dw.DestSize()
	}
	if sr == (image.Rectangle{}) {
		sr.Max = dw.curImageSize
	}
	dw.TransformUsed(drawmatrix.Transform(dr, sr, rotateDeg), sr, op, flipY)
}

// TransformUsed draws the current Use* texture to render target
// Must have called Start and a Use* method first!
//   - xform is the transform mapping source to destination coordinates.
//   - sr is the source region; if zero uses full src image.
//   - op is the drawing operation: Src = copy source directly (blit),
//     Over = alpha blend with existing.
func (dw *Drawer) TransformUsed(xform math32.Matrix3, sr image.Rectangle, op draw.Op, flipY bool) {
	dw.Lock()
	defer dw.Unlock()
	if sr == (image.Rectangle{}) {
		sr.Max = dw.curImageSize
	}

	tmat := drawmatrix.Config(dw.DestSize(), xform, dw.curImageSize, sr, flipY)
	dw.addOp(op, tmat)
}

// addOp adds matrix for given operation
func (dw *Drawer) addOp(op draw.Op, mtx *drawmatrix.Matrix) {
	oi := len(dw.opList)
	mvr := errors.Log1(dw.System.Vars().VarByName(0, "Matrix"))
	mvl := mvr.Values.Values[0]
	nv := mvl.DynamicN()
	if oi >= nv {
		mvl.SetDynamicN(nv + AllocChunk)
	}
	gpu.SetDynamicValueFrom(mvl, oi, []drawmatrix.Matrix{*mtx})
	dw.opList = append(dw.opList, op)
}
