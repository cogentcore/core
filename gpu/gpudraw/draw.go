// Copyright 2024 Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gpudraw

import (
	"image"
	"image/draw"

	"cogentcore.org/core/gpu"
	"cogentcore.org/core/math32"
)

// UseGoImage uses the given Go image.Image as the source image
// for the next Draw operation.
func (dw *Drawer) UseGoImage(img image.Image) {
	dw.Lock()
	defer dw.Unlock()
	dw.curImageSize = img.Bounds().Size()
	tvv := dw.getNextTextureValue()
	tvv.SetFromGoImage(img, 0)
}

// getNextTextureValue gets the next Texture image value, for Use
// methods.
func (dw *Drawer) getNextTextureValue() *gpu.Value {
	sy := dw.Sys
	tvr := sy.Vars.VarByName(1, "TexSampler")
	ni := dw.curTexIdx
	dw.curTexIdx++
	nv := len(tvr.Values.Values)
	if ni >= nv {
		tvr.SetNValues(&sy.Device, nv+AllocChunk)
	}
	return tvr.Values.Values[ni]
}

// SetFrameTexture sets given gpu.Framebuffer image as a drawing source at index,
// used in subsequent Draw methods.  Must have already been configured to fit!
func (dw *Drawer) SetFrameTexture(idx int, fbi any) {
	// todo use above api
	// fb := fbi.(*gpu.Framebuffer)
	// if fb == nil {
	// 	return
	// }
	// dw.Lock()
	// _, tx, _ := dw.Sys.Vars().ValueByIndexTry(0, "Tex", idx)
	// if fb.Format.Size != tx.Texture.Format.Size {
	// 	dw.Unlock()
	// 	dw.ConfigTexture(idx, &fb.Format)
	// 	dw.Lock()
	// }
	// cmd := dw.Sys.MemCmdStart()
	// fb.CopyToTexture(&tx.Texture.Texture, dw.Sys.Device.Device, cmd)
	// dw.Sys.MemCmdEndSubmitWaitFree()
	// dw.Unlock()
}

// Copy copies the current Use* texture to render target.
// Must have called StartDraw and a Use* method first!
// dp is the destination point,
// sr is the source region (set to image.ZR zero rect for all),
// op is the drawing operation: Src = copy source directly (blit),
// Over = alpha blend with existing
// flipY = flipY axis when drawing this image
func (dw *Drawer) Copy(dp image.Point, sr image.Rectangle, op draw.Op, flipY bool) {
	if sr == (image.Rectangle{}) {
		sr.Max = dw.curImageSize
	}
	mat := math32.Matrix3{
		1, 0, 0,
		0, 1, 0,
		float32(dp.X - sr.Min.X), float32(dp.Y - sr.Min.Y), 1,
	}
	dw.Draw(mat, sr, op, flipY)
}

// Scale copies the current Use* texture to render target.
// Must have called StartDraw and a Use* method first!
// Scales the region defined by src and sr to the destination
// such that sr in src-space is mapped to dr in dst-space.
// dr is the destination rectangle
// sr is the source region (set to image.Rectangle{} zero rect for all),
// op is the drawing operation: Src = copy source directly (blit),
// Over = alpha blend with existing
// flipY = flipY axis when drawing this image
// rotDeg = rotation degrees to apply in the mapping: 90 = left, -90 = right, 180 = invert
func (dw *Drawer) Scale(dr image.Rectangle, sr image.Rectangle, op draw.Op, flipY bool, rotDeg float32) {
	if sr == (image.Rectangle{}) {
		sr.Max = dw.curImageSize
	}
	dw.Draw(TransformMatrix(dr, sr, rotDeg), sr, op, flipY)
}

// Draw draws the current Use* texture to render target.
// Must have called StartDraw and a Use* method first!
// src2dst is the transform mapping source to destination
// coordinates (translation, scaling),
// sr is the source region (set to image.ZR for all)
// op is the drawing operation: Src = copy source directly (blit),
// Over = alpha blend with existing
func (dw *Drawer) Draw(src2dst math32.Matrix3, sr image.Rectangle, op draw.Op, flipY bool) {
	dw.Lock()
	defer dw.Unlock()
	if sr == (image.Rectangle{}) {
		sr.Max = dw.curImageSize
	}

	tmat := ConfigMatrix(dw.DestSize(), src2dst, dw.curImageSize, sr, flipY)
	// fmt.Printf("sr: %v  sz: %v  omat: %v  tmat: %v \n", sr, dw.curImageSize, src2dst, tmat)
	dw.addOp(op, tmat)
}

func (dw *Drawer) addOp(op draw.Op, mtx *Matrix) {
	oi := len(dw.opList)
	mvr := dw.Sys.Vars.VarByName(0, "Matrix")
	mvl := mvr.Values.Values[0]
	nv := mvl.DynamicN
	if oi >= nv {
		mvl.DynamicN += AllocChunk
	}
	gpu.SetDynamicValueFrom(mvl, oi, []Matrix{*mtx})
	dw.opList = append(dw.opList, op)
}
