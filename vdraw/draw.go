// Copyright 2022 The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package vdraw

import (
	"fmt"
	"image"
	"image/draw"
	"log"
	"unsafe"

	"github.com/goki/mat32"
	vk "github.com/goki/vulkan"
	"goki.dev/vgpu/vgpu"
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
	dw.UpdtMu.Lock()
	_, tx, _ := dw.Sys.Vars().ValByIdxTry(0, "Tex", idx)
	err := tx.SetGoImage(img, layer, flipY)
	if err != nil && vgpu.Debug {
		fmt.Println(err)
	}
	dw.UpdtMu.Unlock()
}

// GetImageVal returns vgpu Val value of Image for given index
func (dw *Drawer) GetImageVal(idx int) *vgpu.Val {
	_, tx, _ := dw.Sys.Vars().ValByIdxTry(0, "Tex", idx)
	return tx
}

// ConfigImage configures the draw image at given index
// to fit the given image format and number of layers as a drawing source.
func (dw *Drawer) ConfigImage(idx int, fmt *vgpu.ImageFormat) {
	dw.UpdtMu.Lock()
	_, tx, _ := dw.Sys.Vars().ValByIdxTry(0, "Tex", idx)
	tx.Texture.Format = *fmt
	tx.Texture.Format.SetMultisample(1) // can't be multi
	tx.Texture.AllocTexture()
	dw.UpdtMu.Unlock()
}

// SetFrameImage sets given Framebuffer image as a drawing source at index,
// used in subsequent Draw methods.  Must have already been configured to fit!
func (dw *Drawer) SetFrameImage(idx int, fb *vgpu.Framebuffer) {
	dw.UpdtMu.Lock()
	_, tx, _ := dw.Sys.Vars().ValByIdxTry(0, "Tex", idx)
	if fb.Format.Size != tx.Texture.Format.Size {
		dw.UpdtMu.Unlock()
		dw.ConfigImage(idx, &fb.Format)
		dw.UpdtMu.Lock()
	}
	cmd := dw.Sys.MemCmdStart()
	fb.CopyToImage(&tx.Texture.Image, dw.Sys.Device.Device, cmd)
	dw.Sys.MemCmdEndSubmitWaitFree()
	dw.UpdtMu.Unlock()
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
	dw.UpdtMu.Lock()
	sy := &dw.Sys
	sy.Mem.SyncToGPU()
	vars := sy.Vars()
	vk.DeviceWaitIdle(sy.Device.Device)
	vars.BindAllTextureVars(0) // set = 0, iterates over multiple desc sets
	dw.UpdtMu.Unlock()
}

/////////////////////////////////////////////////////////////////////
// Drawing

// Copy copies texture at given index and layer to render target.
// dp is the destination point,
// sr is the source region (set to image.ZR zero rect for all),
// op is the drawing operation: Src = copy source directly (blit),
// Over = alpha blend with existing
// flipY = flipY axis when drawing this image
func (dw *Drawer) Copy(idx, layer int, dp image.Point, sr image.Rectangle, op draw.Op, flipY bool) error {
	mat := mat32.Mat3{
		1, 0, 0,
		0, 1, 0,
		float32(dp.X - sr.Min.X), float32(dp.Y - sr.Min.Y), 1,
	}
	return dw.Draw(idx, layer, mat, sr, op, flipY)
}

// Scale copies texture at given index and layer to render target,
// scaling the region defined by src and sr to the destination
// such that sr in src-space is mapped to dr in dst-space.
// dr is the destination rectangle
// sr is the source region (set to image.ZR zero rect for all),
// op is the drawing operation: Src = copy source directly (blit),
// Over = alpha blend with existing
// flipY = flipY axis when drawing this image
func (dw *Drawer) Scale(idx, layer int, dr image.Rectangle, sr image.Rectangle, op draw.Op, flipY bool) error {
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
	return dw.Draw(idx, layer, mat, sr, op, flipY)
}

// Draw draws texture at index and layer to render target.
// Must have called StartDraw first.
// src2dst is the transform mapping source to destination
// coordinates (translation, scaling),
// sr is the source region (set to image.ZR for all)
// op is the drawing operation: Src = copy source directly (blit),
// Over = alpha blend with existing
func (dw *Drawer) Draw(idx, layer int, src2dst mat32.Mat3, sr image.Rectangle, op draw.Op, flipY bool) error {
	dw.UpdtMu.Lock()
	sy := &dw.Sys
	dpl := dw.SelectPipeline(op)
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

	tmat := dw.ConfigMtxs(src2dst, tx.Texture.Format.Size, sr, op, flipY)
	// fmt.Printf("idx: %d sr: %v  sz: %v  omat: %v  tmat: %v \n", idx, sr, tx.Texture.Format.Size, src2dst, tmat)
	tmat.MVP[3*4] = float32(txIdx) // pack in unused 4th column
	tmat.MVP[3*4+1] = float32(layer)
	matv, _ := vars.VarByNameTry(vgpu.PushSet, "Mtxs")
	dpl.Push(cmd, matv, unsafe.Pointer(tmat))
	dpl.DrawVertex(cmd, 0)
	dw.UpdtMu.Unlock()
	return nil
}

// UseTextureSet selects the descriptor set to use --
// choose this based on the bank of 16
// texture values if number of textures > MaxTexturesPerSet.
func (dw *Drawer) UseTextureSet(descIdx int) {
	dw.UpdtMu.Lock()
	sy := &dw.Sys
	cmd := sy.CmdPool.Buff
	sy.CmdBindVars(cmd, descIdx)
	dw.UpdtMu.Unlock()
}

// StartDraw starts image drawing rendering process on render target
// No images can be added or set after this point.
// descIdx is the descriptor set to use -- choose this based on the bank of 16
// texture values if number of textures > MaxTexturesPerSet.
func (dw *Drawer) StartDraw(descIdx int) {
	dw.UpdtMu.Lock()
	sy := &dw.Sys
	cmd := sy.CmdPool.Buff
	if dw.Surf != nil {
		dw.Impl.SurfIdx = dw.Surf.AcquireNextImage()
		sy.ResetBeginRenderPassNoClear(cmd, dw.Surf.Frames[dw.Impl.SurfIdx], descIdx)
	} else {
		sy.ResetBeginRenderPassNoClear(cmd, dw.Frame.Frames[0], descIdx)
	}
	dw.Impl.LastOp = draw.Src
	dpl := sy.PipelineMap["draw_src"]
	dpl.BindPipeline(cmd)
	dw.UpdtMu.Unlock()
}

// SelectPipeline selects the pipeline based on draw op
// only changes if not last one used.  Default is Src
func (dw *Drawer) SelectPipeline(op draw.Op) *vgpu.Pipeline {
	bind := dw.Impl.LastOp != op
	sy := &dw.Sys
	cmd := sy.CmdPool.Buff
	var dpl *vgpu.Pipeline
	switch op {
	case draw.Src:
		dpl = sy.PipelineMap["draw_src"]
	case draw.Over:
		dpl = sy.PipelineMap["draw_over"]
	}
	if bind {
		dpl.BindPipeline(cmd)
	}
	dw.Impl.LastOp = op
	return dpl
}

// EndDraw ends image drawing rendering process on render target
func (dw *Drawer) EndDraw() {
	dw.UpdtMu.Lock()
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
	dw.UpdtMu.Unlock()
}
