// Copyright 2022 The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package vdraw

import (
	"fmt"
	"image"
	"image/draw"
	"unsafe"

	"github.com/goki/mat32"
	"github.com/goki/vgpu/vgpu"
)

// SetGoImage sets given Go image as a drawing source,
// used in subsequent Draw methods.
// A standard Go image is rendered upright on a standard
// Vulkan surface. Set flipY to true to flip.
func (dw *Drawer) SetGoImage(img image.Image, flipY bool) {
	_, tx, _ := dw.Sys.Vars().ValByIdxTry(0, "Tex", 0)
	tx.SetGoImage(img, false)
	dw.Impl.FlipY = flipY
}

// ConfigImage configures the draw image to fit the given image format as a drawing source.
func (dw *Drawer) ConfigImage(fmt *vgpu.ImageFormat) {
	_, tx, _ := dw.Sys.Vars().ValByIdxTry(0, "Tex", 0)
	tx.Texture.Format = *fmt
	tx.Texture.Format.SetMultisample(1) // can't be multi
	tx.Texture.AllocTexture()
}

// SetFrameImage sets given Framebuffer image as a drawing source,
// used in subsequent Draw methods.
func (dw *Drawer) SetFrameImage(fb *vgpu.Framebuffer, flipY bool) {
	_, tx, _ := dw.Sys.Vars().ValByIdxTry(0, "Tex", 0)
	if fb.Format.Size != tx.Texture.Format.Size {
		fmt.Printf("size mismatch\n")
	}
	cmd := dw.Sys.MemCmdStart()
	fb.CopyToImage(&tx.Texture.Image, dw.Sys.Device.Device, cmd)
	dw.Sys.MemCmdSubmitWaitFree()
	dw.Impl.FlipY = flipY
}

// Copy copies current texture to render target.
// dp is the destination point,
// sr is the source region (set to tex.Format.Bounds() for all),
// op is the drawing operation: Src = copy source directly (blit),
// Over = alpha blend with existing
func (dw *Drawer) Copy(dp image.Point, sr image.Rectangle, op draw.Op) error {
	mat := mat32.Mat3{
		1, 0, 0,
		0, 1, 0,
		float32(dp.X - sr.Min.X), float32(dp.Y - sr.Min.Y), 1,
	}
	return dw.Draw(mat, sr, op)
}

// Scale copies current texture to render target,
// scaling the region defined by src and sr to the destination
// such that sr in src-space is mapped to dr in dst-space.
// dr is the destination rectangle
// sr is the source region (set to tex.Format.Bounds() for all),
// op is the drawing operation: Src = copy source directly (blit),
// Over = alpha blend with existing
func (dw *Drawer) Scale(dr image.Rectangle, sr image.Rectangle, op draw.Op) error {
	rx := float32(dr.Dx()) / float32(sr.Dx())
	ry := float32(dr.Dy()) / float32(sr.Dy())
	mat := mat32.Mat3{
		rx, 0, 0,
		0, ry, 0,
		float32(dr.Min.X) - rx*float32(sr.Min.X),
		float32(dr.Min.Y) - ry*float32(sr.Min.Y), 1,
	}
	return dw.Draw(mat, sr, op)
}

// Draw draws current texture to render target.
// src2dst is the transform mapping source to destination
// coordinates (translation, scaling),
// sr is the source region (set to tex.Format.Bounds() for all)
// op is the drawing operation: Src = copy source directly (blit),
// Over = alpha blend with existing
func (dw *Drawer) Draw(src2dst mat32.Mat3, sr image.Rectangle, op draw.Op) error {
	dw.StartDraw()
	dw.DrawImpl(src2dst, sr, op)
	dw.EndDraw()
	return nil
}

// DrawImpl is impl that draws current texture to render target.
// Must have called StartDraw first.
// src2dst is the transform mapping source to destination
// coordinates (translation, scaling),
// sr is the source region (set to tex.Format.Bounds() for all)
// op is the drawing operation: Src = copy source directly (blit),
// Over = alpha blend with existing
func (dw *Drawer) DrawImpl(src2dst mat32.Mat3, sr image.Rectangle, op draw.Op) error {
	vars := dw.Sys.Vars()
	_, tx, _ := vars.ValByIdxTry(0, "Tex", 0)

	tmat := dw.ConfigMats(src2dst, tx.Texture.Format.Size, sr, op, false)
	matv, _ := vars.VarByNameTry(vgpu.PushSet, "Mats")
	dpl := dw.Sys.PipelineMap["draw"]

	cmd := dw.Sys.CmdPool.Buff
	dpl.Push(cmd, matv, vgpu.VertexShader, unsafe.Pointer(tmat))
	dpl.DrawVertex(cmd, 0)
	return nil
}

// StartDraw starts image drawing rendering process on render target
// No images can be added or set after this point.
func (dw *Drawer) StartDraw() {
	sy := &dw.Sys
	sy.Mem.SyncToGPU()
	vars := sy.Vars()
	vars.BindVarsStart(0)
	vars.BindStatVars(0) // binds images
	vars.BindVarsEnd()
	dpl := sy.PipelineMap["draw"]
	if dw.Surf != nil {
		dw.Impl.SurfIdx = dw.Surf.AcquireNextImage()
		cmd := sy.CmdPool.Buff
		sy.ResetBeginRenderPassNoClear(cmd, dw.Surf.Frames[dw.Impl.SurfIdx], 0)
		dpl.BindPipeline(cmd)
	}
}

// EndDraw ends image drawing rendering process on render target
func (dw *Drawer) EndDraw() {
	sy := &dw.Sys
	cmd := sy.CmdPool.Buff
	if dw.Surf != nil {
		sidx := dw.Impl.SurfIdx
		sy.EndRenderPass(cmd)
		dw.Surf.SubmitRender(cmd) // this is where it waits for the 16 msec
		dw.Surf.PresentImage(sidx)
	}
}
