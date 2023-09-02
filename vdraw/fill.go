// Copyright 2022 The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package vdraw

import (
	"image"
	"image/color"
	"image/draw"
	"unsafe"

	"goki.dev/mat32/v2"
	"goki.dev/vgpu/v2/vgpu"
)

// FillRect fills given color to render target, to given region.
// op is the drawing operation: Src = copy source directly (blit),
// Over = alpha blend with existing
func (dw *Drawer) FillRect(clr color.Color, reg image.Rectangle, op draw.Op) error {
	mat := mat32.Mat3{
		1, 0, 0,
		0, 1, 0,
		0, 0, 1,
	}
	return dw.Fill(clr, mat, reg, op)
}

// Fill fills given color to to render target.
// src2dst is the transform mapping source to destination
// coordinates (translation, scaling),
// reg is the region to fill
// op is the drawing operation: Src = copy source directly (blit),
// Over = alpha blend with existing
func (dw *Drawer) Fill(clr color.Color, src2dst mat32.Mat3, reg image.Rectangle, op draw.Op) error {
	sy := &dw.Sys
	cmd := sy.CmdPool.Buff
	vars := sy.Vars()

	dsz := dw.DestSize()
	tmat := dw.ConfigMtxs(src2dst, dsz, reg, op, false)
	clr4 := mat32.NewVec4Color(clr)
	clr4.ToArray(tmat.UVP[:], 12) // last column holds color

	matv, _ := vars.VarByNameTry(vgpu.PushSet, "Mtxs")
	fpl := sy.PipelineMap["fill"]
	fpl.Push(cmd, matv, unsafe.Pointer(tmat))
	fpl.BindDrawVertex(cmd, 0)

	return nil
}

// StartFill starts color fill drawing rendering process on render target
func (dw *Drawer) StartFill() {
	sy := &dw.Sys
	fpl := sy.PipelineMap["fill"]
	cmd := sy.CmdPool.Buff
	if dw.Surf != nil {
		dw.Impl.SurfIdx = dw.Surf.AcquireNextImage()
		sy.ResetBeginRenderPassNoClear(cmd, dw.Surf.Frames[dw.Impl.SurfIdx], 0)
	} else {
		sy.ResetBeginRenderPassNoClear(cmd, dw.Frame.Frames[0], 0)
	}
	fpl.BindPipeline(cmd)
}

// EndFill ends color filling rendering process on render target
func (dw *Drawer) EndFill() {
	sy := &dw.Sys
	cmd := sy.CmdPool.Buff
	sy.EndRenderPass(cmd)
	if dw.Surf != nil {
		dw.Surf.SubmitRender(cmd) // this is where it waits for the 16 msec
		dw.Surf.PresentImage(dw.Impl.SurfIdx)
	} else {
		dw.Frame.SubmitRender(cmd)
		dw.Frame.WaitForRender()
	}
}
