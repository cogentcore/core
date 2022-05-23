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
// and layer, used in subsequent Draw methods.
// A standard Go image is rendered upright on a standard Vulkan surface.
// Set flipY to true to flip.
func (dw *Drawer) SetGoImage(idx, layer int, img image.Image, flipY bool) {
	_, tx, _ := dw.Sys.Vars().ValByIdxTry(0, "Tex", idx)
	tx.SetGoImage(img, layer, flipY)
}

// ConfigImage configures the draw image at given index
// to fit the given image format and number of layers as a drawing source.
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
// and layer, used in subsequent Draw methods. (use SetImageName to set names)
// A standard Go image is rendered upright on a standard Vulkan surface.
// Set flipY to true to flip. This can be used directly without pre-configuring.
func (dw *Drawer) SetGoImageName(name string, layer int, img image.Image, flipY bool) {
	idx := dw.ImageIdxByName(name)
	dw.SetGoImage(idx, layer, img, flipY)
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
	vars.BindAllTextureVars(0) // set = 0, iterates over multiple desc sets
}

/////////////////////////////////////////////////////////////////////
// Drawing

// Copy copies texture at given index and layer to render target.
// dp is the destination point,
// sr is the source region (set to image.ZR zero rect for all),
// op is the drawing operation: Src = copy source directly (blit),
// Over = alpha blend with existing
func (dw *Drawer) Copy(idx, layer int, dp image.Point, sr image.Rectangle, op draw.Op) error {
	mat := mat32.Mat3{
		1, 0, 0,
		0, 1, 0,
		float32(dp.X - sr.Min.X), float32(dp.Y - sr.Min.Y), 1,
	}
	return dw.Draw(idx, layer, mat, sr, op)
}

// Scale copies texture at given index and layer to render target,
// scaling the region defined by src and sr to the destination
// such that sr in src-space is mapped to dr in dst-space.
// dr is the destination rectangle
// sr is the source region (set to image.ZR zero rect for all),
// op is the drawing operation: Src = copy source directly (blit),
// Over = alpha blend with existing
func (dw *Drawer) Scale(idx, layer int, dr image.Rectangle, sr image.Rectangle, op draw.Op) error {
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
	return dw.Draw(idx, layer, mat, sr, op)
}

// Draw draws texture at index and layer to render target.
// Must have called StartDraw first.
// src2dst is the transform mapping source to destination
// coordinates (translation, scaling),
// sr is the source region (set to image.ZR for all)
// op is the drawing operation: Src = copy source directly (blit),
// Over = alpha blend with existing
func (dw *Drawer) Draw(idx, layer int, src2dst mat32.Mat3, sr image.Rectangle, op draw.Op) error {
	sy := &dw.Sys
	dpl := sy.PipelineMap["draw"]
	vars := sy.Vars()
	cmd := sy.CmdPool.Buff

	txIdx, _, _, err := sy.CmdBindTextureVarIdx(cmd, 0, "Tex", idx)
	if err != nil {
		return err
	}
	_, tx, _ := vars.ValByIdxTry(0, "Tex", idx)
	if sr == image.ZR {
		sr = tx.Texture.Format.Bounds()
	}

	tmat := dw.ConfigMtxs(src2dst, tx.Texture.Format.Size, sr, op, false)
	// fmt.Printf("idx: %d sr: %v  sz: %v  omat: %v  tmat: %v \n", idx, sr, tx.Texture.Format.Size, src2dst, tmat)
	tmat.MVP[3*4] = float32(txIdx) // pack in unused 4th column
	tmat.MVP[3*4+1] = float32(layer)
	matv, _ := vars.VarByNameTry(vgpu.PushSet, "Mtxs")
	dpl.Push(cmd, matv, unsafe.Pointer(tmat))
	dpl.DrawVertex(cmd, 0)
	return nil
}

// UseTextureSet selects the descriptor set to use -- choose this based on the bank of 16
// texture values if number of textures > MaxTexturesPerSet.
func (dw *Drawer) UseTextureSet(descIdx int) {
	sy := &dw.Sys
	cmd := sy.CmdPool.Buff
	sy.CmdBindVars(cmd, descIdx)
}

// StartDraw starts image drawing rendering process on render target
// No images can be added or set after this point.
// descIdx is the descriptor set to use -- choose this based on the bank of 16
// texture values if number of textures > MaxTexturesPerSet.
func (dw *Drawer) StartDraw(descIdx int) {
	sy := &dw.Sys
	dpl := sy.PipelineMap["draw"]
	cmd := sy.CmdPool.Buff
	if dw.Surf != nil {
		dw.Impl.SurfIdx = dw.Surf.AcquireNextImage()
		sy.ResetBeginRenderPassNoClear(cmd, dw.Surf.Frames[dw.Impl.SurfIdx], descIdx)
	} else {
		sy.ResetBeginRenderPassNoClear(cmd, dw.Frame.Frames[0], descIdx)
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
