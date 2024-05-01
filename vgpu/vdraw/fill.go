// Copyright 2022 Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package vdraw

import (
	"image"
	"image/color"
	"image/draw"
	"unsafe"

	"cogentcore.org/core/math32"
	"cogentcore.org/core/vgpu"
)

// FillRect fills given color to render target, to given region.
// op is the drawing operation: Src = copy source directly (blit),
// Over = alpha blend with existing
func (dw *Drawer) FillRect(clr color.Color, reg image.Rectangle, op draw.Op) error {
	return dw.Fill(clr, math32.Identity3(), reg, op)
}

// Fill fills given color to to render target.
// src2dst is the transform mapping source to destination
// coordinates (translation, scaling),
// reg is the region to fill
// op is the drawing operation: Src = copy source directly (blit),
// Over = alpha blend with existing
func (dw *Drawer) Fill(clr color.Color, src2dst math32.Matrix3, reg image.Rectangle, op draw.Op) error {
	sy := &dw.Sys
	cmd := sy.CmdPool.Buff
	vars := sy.Vars()

	dsz := dw.DestSize()
	tmat := dw.ConfigMtxs(src2dst, dsz, reg, op, false)
	clr4 := math32.NewVector4Color(clr)
	clr4.ToSlice(tmat.UVP[:], 12) // last column holds color

	matv, _ := vars.VarByNameTry(vgpu.PushSet, "Mtxs")
	fpl := sy.PipelineMap["fill"]
	fpl.Push(cmd, matv, unsafe.Pointer(tmat))
	fpl.BindDrawVertex(cmd, 0)

	return nil
}

// StartFill starts color fill drawing rendering process on render target.
// It returns false if rendering can not proceed.
func (dw *Drawer) StartFill() bool {
	sy := &dw.Sys
	fpl := sy.PipelineMap["fill"]
	cmd := sy.CmdPool.Buff
	if dw.Surf != nil {
		idx, ok := dw.Surf.AcquireNextImage()
		if !ok {
			return false
		}
		dw.Impl.SurfIndex = idx
		sy.ResetBeginRenderPassNoClear(cmd, dw.Surf.Frames[dw.Impl.SurfIndex], 0)
	} else {
		sy.ResetBeginRenderPassNoClear(cmd, dw.Frame.Frames[0], 0)
	}
	fpl.BindPipeline(cmd)
	return true
}

// EndFill ends color filling rendering process on render target
func (dw *Drawer) EndFill() {
	sy := &dw.Sys
	cmd := sy.CmdPool.Buff
	sy.EndRenderPass(cmd)
	if dw.Surf != nil {
		dw.Surf.SubmitRender(cmd) // this is where it waits for the 16 msec
		dw.Surf.PresentImage(dw.Impl.SurfIndex)
	} else {
		dw.Frame.SubmitRender(cmd)
		dw.Frame.WaitForRender()
	}
}
