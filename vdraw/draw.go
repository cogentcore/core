// Copyright 2022 The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package vdraw

import (
	"image"
	"image/draw"
	"log"
	"unsafe"

	"github.com/goki/mat32"
	"github.com/goki/vgpu/vgpu"
	vk "github.com/goki/vulkan"
)

// These draw.Op constants are provided so that users of this package don't
// have to explicitly import "image/draw".
const (
	Over = draw.Over
	Src  = draw.Src
)

// SetGoImage sets given Go image as a drawing source to given image index,
// used in subsequent Draw methods.
// A standard Go image is rendered upright on a standard
// Vulkan surface. Set flipY to true to flip.
func (dw *Drawer) SetGoImage(idx int, img image.Image, flipY bool) {
	_, tx, _ := dw.Sys.Vars().ValByIdxTry(0, "Tex", idx)
	tx.SetGoImage(img, flipY)
}

// ConfigImage configures the draw image at given index
// to fit the given image format as a drawing source.
func (dw *Drawer) ConfigImage(idx int, fmt *vgpu.ImageFormat) {
	_, tx, _ := dw.Sys.Vars().ValByIdxTry(0, "Tex", idx)
	tx.Texture.Format = *fmt
	tx.Texture.Format.SetMultisample(1) // can't be multi
	tx.Texture.AllocTexture()
}

// SetFrameImage sets given Framebuffer image as a drawing source at index,
// used in subsequent Draw methods.  Must have already been configured to fit!
func (dw *Drawer) SetFrameImage(idx int, fb *vgpu.Framebuffer) {
	_, tx, _ := dw.Sys.Vars().ValByIdxTry(0, "Tex", idx)
	if fb.Format.Size != tx.Texture.Format.Size {
		dw.ConfigImage(idx, &fb.Format)
	}
	cmd := dw.Sys.MemCmdStart()
	fb.CopyToImage(&tx.Texture.Image, dw.Sys.Device.Device, cmd)
	dw.Sys.MemCmdSubmitWaitFree()
}

////////////////////////////////////////////////////////////////
// Names

// SetImageName sets name of image at given index, to enable name-based
// access for subsequent calls.  Returns error if name already exists.
func (dw *Drawer) SetImageName(idx int, name string) error {
	vr := dw.Sys.Vars().SetMap[0].Vars[0]
	_, err := vr.Vals.SetName(idx, name)
	return err
}

// ImageIdxByName returns index of image val by name.
// Logs error if not found, and returns 0.
func (dw *Drawer) ImageIdxByName(name string) int {
	vr := dw.Sys.Vars().SetMap[0].Vars[0]
	vl, err := vr.Vals.ValByNameTry(name)
	if err != nil {
		log.Println(err)
		return 0
	}
	return vl.Idx
}

// SetGoImageName sets given Go image as a drawing source to given image name,
// used in subsequent Draw methods. (use SetImageName to set names)
// A standard Go image is rendered upright on a standard
// Vulkan surface. Set flipY to true to flip.
func (dw *Drawer) SetGoImageName(name string, img image.Image, flipY bool) {
	idx := dw.ImageIdxByName(name)
	dw.SetGoImage(idx, img, flipY)
}

// ConfigImageName configures the draw image at given name
// to fit the given image format as a drawing source.
func (dw *Drawer) ConfigImageName(name string, fmt *vgpu.ImageFormat) {
	idx := dw.ImageIdxByName(name)
	dw.ConfigImage(idx, fmt)
}

// SetFrameImageName sets given Framebuffer image as a drawing source at name,
// used in subsequent Draw methods.  Must have already been configured to fit!
func (dw *Drawer) SetFrameImageName(name string, fb *vgpu.Framebuffer) {
	idx := dw.ImageIdxByName(name)
	dw.SetFrameImage(idx, fb)
}

// SyncImages must be called after images have been updated, to sync
// memory up to the GPU.
func (dw *Drawer) SyncImages() {
	sy := &dw.Sys
	sy.Mem.SyncToGPU()
	vars := sy.Vars()
	vk.DeviceWaitIdle(sy.Device.Device)
	vars.BindVarsStart(0)
	vars.BindStatVars(0) // binds images
	vars.BindVarsEnd()
}

/////////////////////////////////////////////////////////////////////
// Drawing

// Copy copies texture at given index to render target.
// dp is the destination point,
// sr is the source region (set to image.ZR zero rect for all),
// op is the drawing operation: Src = copy source directly (blit),
// Over = alpha blend with existing
func (dw *Drawer) Copy(idx int, dp image.Point, sr image.Rectangle, op draw.Op) error {
	mat := mat32.Mat3{
		1, 0, 0,
		0, 1, 0,
		float32(dp.X - sr.Min.X), float32(dp.Y - sr.Min.Y), 1,
	}
	return dw.Draw(idx, mat, sr, op)
}

// Scale copies texture at given index to render target,
// scaling the region defined by src and sr to the destination
// such that sr in src-space is mapped to dr in dst-space.
// dr is the destination rectangle
// sr is the source region (set to image.ZR zero rect for all),
// op is the drawing operation: Src = copy source directly (blit),
// Over = alpha blend with existing
func (dw *Drawer) Scale(idx int, dr image.Rectangle, sr image.Rectangle, op draw.Op) error {
	if sr == image.ZR {
		_, tx, _ := dw.Sys.Vars().ValByIdxTry(0, "Tex", idx)
		sr = tx.Texture.Format.Bounds()
	}
	rx := float32(dr.Dx()) / float32(sr.Dx())
	ry := float32(dr.Dy()) / float32(sr.Dy())
	mat := mat32.Mat3{
		rx, 0, 0,
		0, ry, 0,
		float32(dr.Min.X) - rx*float32(sr.Min.X),
		float32(dr.Min.Y) - ry*float32(sr.Min.Y), 1,
	}
	return dw.Draw(idx, mat, sr, op)
}

// Draw draws texture at index to render target.
// Must have called StartDraw first.
// src2dst is the transform mapping source to destination
// coordinates (translation, scaling),
// sr is the source region (set to image.ZR for all)
// op is the drawing operation: Src = copy source directly (blit),
// Over = alpha blend with existing
func (dw *Drawer) Draw(idx int, src2dst mat32.Mat3, sr image.Rectangle, op draw.Op) error {
	vars := dw.Sys.Vars()
	_, tx, _ := vars.ValByIdxTry(0, "Tex", idx)

	if sr == image.ZR {
		sr = tx.Texture.Format.Bounds()
	}

	tmat := dw.ConfigMats(src2dst, tx.Texture.Format.Size, sr, op, false)
	tmat.MVP[3] = float32(idx) // pack it!
	matv, _ := vars.VarByNameTry(vgpu.PushSet, "Mats")
	dpl := dw.Sys.PipelineMap["draw"]

	cmd := dw.Sys.CmdPool.Buff
	dpl.Push(cmd, matv, unsafe.Pointer(tmat))
	dpl.DrawVertex(cmd, 0)
	return nil
}

// StartDraw starts image drawing rendering process on render target
// No images can be added or set after this point.
func (dw *Drawer) StartDraw() {
	sy := &dw.Sys
	dpl := sy.PipelineMap["draw"]
	cmd := sy.CmdPool.Buff
	if dw.Surf != nil {
		dw.Impl.SurfIdx = dw.Surf.AcquireNextImage()
		sy.ResetBeginRenderPassNoClear(cmd, dw.Surf.Frames[dw.Impl.SurfIdx], 0)
	} else {
		sy.ResetBeginRenderPassNoClear(cmd, dw.Frame.Frames[0], 0)
	}
	dpl.BindPipeline(cmd)
}

// EndDraw ends image drawing rendering process on render target
func (dw *Drawer) EndDraw() {
	sy := &dw.Sys
	cmd := sy.CmdPool.Buff
	sy.EndRenderPass(cmd)
	if dw.Surf != nil {
		sidx := dw.Impl.SurfIdx
		dw.Surf.SubmitRender(cmd)
		dw.Surf.PresentImage(sidx)
	} else {
		dw.Frame.SubmitRender(cmd)
		dw.Frame.WaitForRender()
	}
}
