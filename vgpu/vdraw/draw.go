// Copyright 2022 Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package vdraw

import (
	"fmt"
	"image"
	"image/draw"
	"log"
	"unsafe"

	"cogentcore.org/core/math32"
	"cogentcore.org/core/vgpu"
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
	dw.UpdateMu.Lock()
	_, tx, _ := dw.Sys.Vars().ValueByIndexTry(0, "Tex", idx)
	err := tx.SetGoImage(img, layer, flipY)
	if err != nil && vgpu.Debug {
		fmt.Println(err)
	}
	dw.UpdateMu.Unlock()
}

// GetImageValue returns vgpu Value value of Image for given index
func (dw *Drawer) GetImageValue(idx int) *vgpu.Value {
	_, tx, _ := dw.Sys.Vars().ValueByIndexTry(0, "Tex", idx)
	return tx
}

// ConfigImageDefaultFormat configures the draw image at the given index
// to fit the default image format specified by the given width, height,
// and number of layers.
func (dw *Drawer) ConfigImageDefaultFormat(idx int, width int, height int, layers int) {
	dw.ConfigImage(idx, vgpu.NewImageFormat(width, height, layers))
}

// ConfigImage configures the draw image at given index
// to fit the given image format and number of layers as a drawing source.
func (dw *Drawer) ConfigImage(idx int, fmt *vgpu.ImageFormat) {
	dw.UpdateMu.Lock()
	_, tx, _ := dw.Sys.Vars().ValueByIndexTry(0, "Tex", idx)
	tx.Texture.Format = *fmt
	tx.Texture.Format.SetMultisample(1) // can't be multi
	tx.Texture.AllocTexture()
	dw.UpdateMu.Unlock()
}

// SetFrameImage sets given vgpu.Framebuffer image as a drawing source at index,
// used in subsequent Draw methods.  Must have already been configured to fit!
func (dw *Drawer) SetFrameImage(idx int, fbi any) {
	fb := fbi.(*vgpu.Framebuffer)
	if fb == nil {
		return
	}
	dw.UpdateMu.Lock()
	_, tx, _ := dw.Sys.Vars().ValueByIndexTry(0, "Tex", idx)
	if fb.Format.Size != tx.Texture.Format.Size {
		dw.UpdateMu.Unlock()
		dw.ConfigImage(idx, &fb.Format)
		dw.UpdateMu.Lock()
	}
	cmd := dw.Sys.MemCmdStart()
	fb.CopyToImage(&tx.Texture.Image, dw.Sys.Device.Device, cmd)
	dw.Sys.MemCmdEndSubmitWaitFree()
	dw.UpdateMu.Unlock()
}

////////////////////////////////////////////////////////////////
// Names

// SetImageName sets name of image at given index, to enable name-based
// access for subsequent calls.  Returns error if name already exists.
func (dw *Drawer) SetImageName(idx int, name string) error {
	vr := dw.Sys.Vars().SetMap[0].Vars[0]
	_, err := vr.Values.SetName(idx, name)
	return err
}

// ImageIndexByName returns index of image val by name.
// Logs error if not found, and returns 0.
func (dw *Drawer) ImageIndexByName(name string) int {
	vr := dw.Sys.Vars().SetMap[0].Vars[0]
	vl, err := vr.Values.ValueByNameTry(name)
	if err != nil {
		log.Println(err)
		return 0
	}
	return vl.Index
}

// SetGoImageName sets given Go image as a drawing source to given image name,
// and layer, used in subsequent Draw methods. (use SetImageName to set names)
// A standard Go image is rendered upright on a standard Vulkan surface.
// Set flipY to true to flip. This can be used directly without pre-configuring.
func (dw *Drawer) SetGoImageName(name string, layer int, img image.Image, flipY bool) {
	idx := dw.ImageIndexByName(name)
	dw.SetGoImage(idx, layer, img, flipY)
}

// ConfigImageName configures the draw image at given name
// to fit the given image format as a drawing source.
func (dw *Drawer) ConfigImageName(name string, fmt *vgpu.ImageFormat) {
	idx := dw.ImageIndexByName(name)
	dw.ConfigImage(idx, fmt)
}

// SetFrameImageName sets given Framebuffer image as a drawing source at name,
// used in subsequent Draw methods.  Must have already been configured to fit!
func (dw *Drawer) SetFrameImageName(name string, fb *vgpu.Framebuffer) {
	idx := dw.ImageIndexByName(name)
	dw.SetFrameImage(idx, fb)
}

// SyncImages must be called after images have been updated, to sync
// memory up to the GPU.
func (dw *Drawer) SyncImages() {
	dw.UpdateMu.Lock()
	sy := &dw.Sys
	sy.Mem.SyncToGPU()
	vars := sy.Vars()
	vk.DeviceWaitIdle(sy.Device.Device)
	vars.BindAllTextureVars(0) // set = 0, iterates over multiple desc sets
	dw.UpdateMu.Unlock()
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
	mat := math32.Matrix3{
		1, 0, 0,
		0, 1, 0,
		float32(dp.X - sr.Min.X), float32(dp.Y - sr.Min.Y), 1,
	}
	return dw.Draw(idx, layer, mat, sr, op, flipY)
}

// TransformMatrix returns a transformation matrix for the generic Draw function
// that scales, translates, and rotates the source image by the given degrees.
// to make it fit within the destination rectangle dr, given its original size sr (unrotated).
// To avoid scaling, ensure that the dr and sr are the same dimensions (post rotation).
// rotDeg = rotation degrees to apply in the mapping: 90 = left, -90 = right, 180 = invert
func TransformMatrix(dr image.Rectangle, sr image.Rectangle, rotDeg float32) math32.Matrix3 {
	sx := float32(dr.Dx()) / float32(sr.Dx())
	sy := float32(dr.Dy()) / float32(sr.Dy())
	tx := float32(dr.Min.X) - sx*float32(sr.Min.X)
	ty := float32(dr.Min.Y) - sy*float32(sr.Min.Y)

	if rotDeg == 0 {
		return math32.Matrix3{
			sx, 0, 0,
			0, sy, 0,
			tx, ty, 1,
		}
	}
	rad := math32.DegToRad(rotDeg)
	dsz := math32.FromPoint(dr.Size())
	rmat := math32.Rotate2D(rad)

	dmnr := rmat.MulVector2AsPoint(math32.FromPoint(dr.Min))
	dmxr := rmat.MulVector2AsPoint(math32.FromPoint(dr.Max))
	sx = math32.Abs(dmxr.X-dmnr.X) / float32(sr.Dx())
	sy = math32.Abs(dmxr.Y-dmnr.Y) / float32(sr.Dy())
	tx = dmnr.X - sx*float32(sr.Min.X)
	ty = dmnr.Y - sy*float32(sr.Min.Y)

	if rotDeg < -45 && rotDeg > -135 {
		ty -= dsz.X
	} else if rotDeg > 45 && rotDeg < 135 {
		tx -= dsz.Y
	} else if rotDeg > 135 || rotDeg < -135 {
		ty -= dsz.Y
		tx -= dsz.X
	}

	mat := math32.Matrix3{
		sx, 0, 0,
		0, sy, 0,
		tx, ty, 1,
	}

	return mat.Mul(math32.Matrix3FromMatrix2(rmat))

	/*  stuff that didn't work, but theoretically should?
	rad := math32.DegToRad(rotDeg)
	dsz := math32.FromPoint(dr.Size())
	dctr := dsz.MulScalar(0.5)
	_ = dctr
	// mat2 := math32.Translate2D(dctr.X, 0).Mul(math32.Rotate2D(rad)).Mul(math32.Translate2D(tx, ty)).Mul(math32.Scale2D(sx, sy))
	mat2 := math32.Translate2D(tx, ty).Mul(math32.Scale2D(sx, sy)).Mul(math32.Translate2D(dctr.X, 0)).Mul(math32.Rotate2D(rad))
	// mat2 := math32.Rotate2D(rad).MulCtr(math32.Translate2D(tx, ty).Mul(math32.Scale2D(sx, sy)), dctr)
	mat := math32.Matrix3FromMatrix2(mat2)
	*/
}

// Scale copies texture at given index and layer to render target,
// scaling the region defined by src and sr to the destination
// such that sr in src-space is mapped to dr in dst-space.
// dr is the destination rectangle
// sr is the source region (set to image.Rectangle{} zero rect for all),
// op is the drawing operation: Src = copy source directly (blit),
// Over = alpha blend with existing
// flipY = flipY axis when drawing this image
// rotDeg = rotation degrees to apply in the mapping: 90 = left, -90 = right, 180 = invert
func (dw *Drawer) Scale(idx, layer int, dr image.Rectangle, sr image.Rectangle, op draw.Op, flipY bool, rotDeg float32) error {
	zr := image.Rectangle{}
	if sr == zr {
		_, tx, _ := dw.Sys.Vars().ValueByIndexTry(0, "Tex", idx)
		sr = tx.Texture.Format.Bounds()
	}
	return dw.Draw(idx, layer, TransformMatrix(dr, sr, rotDeg), sr, op, flipY)
}

// Draw draws texture at index and layer to render target.
// Must have called StartDraw first.
// src2dst is the transform mapping source to destination
// coordinates (translation, scaling),
// sr is the source region (set to image.ZR for all)
// op is the drawing operation: Src = copy source directly (blit),
// Over = alpha blend with existing
func (dw *Drawer) Draw(idx, layer int, src2dst math32.Matrix3, sr image.Rectangle, op draw.Op, flipY bool) error {
	dw.UpdateMu.Lock()
	sy := &dw.Sys
	dpl := dw.SelectPipeline(op)
	vars := sy.Vars()
	cmd := sy.CmdPool.Buff

	txIndex, _, _, err := sy.CmdBindTextureVarIndex(cmd, 0, "Tex", idx)
	if err != nil {
		return err
	}
	_, tx, _ := vars.ValueByIndexTry(0, "Tex", idx)
	if sr == image.ZR {
		sr = tx.Texture.Format.Bounds()
	}

	tmat := dw.ConfigMtxs(src2dst, tx.Texture.Format.Size, sr, op, flipY)
	// fmt.Printf("idx: %d sr: %v  sz: %v  omat: %v  tmat: %v \n", idx, sr, tx.Texture.Format.Size, src2dst, tmat)
	tmat.MVP[3*4] = float32(txIndex) // pack in unused 4th column
	tmat.MVP[3*4+1] = float32(layer)
	matv, _ := vars.VarByNameTry(vgpu.PushSet, "Mtxs")
	dpl.Push(cmd, matv, unsafe.Pointer(tmat))
	dpl.DrawVertex(cmd, 0)
	dw.UpdateMu.Unlock()
	return nil
}

// UseTextureSet selects the descriptor set to use --
// choose this based on the bank of 16
// texture values if number of textures > MaxTexturesPerSet.
func (dw *Drawer) UseTextureSet(descIndex int) {
	dw.UpdateMu.Lock()
	sy := &dw.Sys
	cmd := sy.CmdPool.Buff
	sy.CmdBindVars(cmd, descIndex)
	dw.UpdateMu.Unlock()
}

// StartDraw starts image drawing rendering process on render target
// No images can be added or set after this point.
// descIndex is the descriptor set to use -- choose this based on the bank of 16
// texture values if number of textures > MaxTexturesPerSet.
// It returns false if rendering can not proceed.
func (dw *Drawer) StartDraw(descIndex int) bool {
	dw.UpdateMu.Lock()
	defer dw.UpdateMu.Unlock()
	sy := &dw.Sys
	cmd := sy.CmdPool.Buff
	if dw.Surf != nil {
		idx, ok := dw.Surf.AcquireNextImage()
		if !ok {
			return false
		}
		dw.Impl.SurfIndex = idx
		sy.ResetBeginRenderPassNoClear(cmd, dw.Surf.Frames[dw.Impl.SurfIndex], descIndex)
	} else {
		sy.ResetBeginRenderPassNoClear(cmd, dw.Frame.Frames[0], descIndex)
	}
	dw.Impl.LastOp = draw.Src
	dpl := sy.PipelineMap["draw_src"]
	dpl.BindPipeline(cmd)
	return true
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
	dw.UpdateMu.Lock()
	sy := &dw.Sys
	cmd := sy.CmdPool.Buff
	sy.EndRenderPass(cmd)
	if dw.Surf != nil {
		sidx := dw.Impl.SurfIndex
		dw.Surf.SubmitRender(cmd)
		dw.Surf.PresentImage(sidx)
	} else {
		dw.Frame.SubmitRender(cmd)
		dw.Frame.WaitForRender()
	}
	dw.UpdateMu.Unlock()
}
