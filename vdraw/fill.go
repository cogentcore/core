// Copyright 2022 The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package vdraw

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"log"
	"unsafe"

	"github.com/goki/mat32"
	"github.com/goki/vgpu/vgpu"
)

type ColorIdx struct {
	Color color.Color `desc:"color"`
	Idx   int         `desc:"index"`
}

// Palette is a collection of named colors with associated index
// for use with Fill function.
type Palette map[string]*ColorIdx

// Add adds a color to the palette
func (pl *Palette) Add(name string, clr color.Color) {
	if *pl == nil {
		*pl = make(Palette)
	}
	nc := &ColorIdx{Color: clr, Idx: len(*pl)}
	(*pl)[name] = nc
}

// Index returns the index of a color by name, -1 if not found
func (pl *Palette) Index(name string) int {
	nc, has := (*pl)[name]
	if !has {
		return -1
	}
	return nc.Idx
}

// SetPalette sets fill colors based on palette
func (dw *Drawer) SetPalette(pal Palette) error {
	var cerr error
	for _, nc := range pal {
		err := dw.SetColor(nc.Idx, nc.Color)
		if err != nil {
			cerr = err
			break
		}
	}
	dw.ColorsUpdated()
	return cerr
}

// SetColor sets fill color at given index (must be < MaxColors)
func (dw *Drawer) SetColor(idx int, src color.Color) error {
	if idx >= dw.Impl.MaxColors {
		err := fmt.Errorf("vdraw.SetColor: index: %d exceed maximum number of colors: %d -- set max higher in Config", idx, dw.Impl.MaxColors)
		if vgpu.TheGPU.Debug {
			log.Println(err)
		}
		return err
	}
	vars := dw.Sys.Vars()

	r, g, b, a := src.RGBA()
	clr := mat32.NewVec4(float32(r)/65535, float32(g)/65535, float32(b)/65535, float32(a)/65535)
	_, fc, _ := vars.ValByIdxTry(1, "Color", idx)
	fcv := fc.Floats32()
	fcv.SetVec4(0, clr)
	fc.SetMod()
	return nil
}

// ColorsUpdated must be called after all the colors have been updated
func (dw *Drawer) ColorsUpdated() {
	dw.Sys.Mem.SyncToGPU()
}

// FillRect fills color at given index to render target, to given region.
// op is the drawing operation: Src = copy source directly (blit),
// Over = alpha blend with existing
func (dw *Drawer) FillRect(idx int, reg image.Rectangle, op draw.Op) error {
	mat := mat32.Mat3{
		1, 0, 0,
		0, 1, 0,
		0, 0, 1,
	}
	return dw.Fill(idx, mat, reg, op)
}

// Fill fills color at given index to render target.
// src2dst is the transform mapping source to destination
// coordinates (translation, scaling),
// reg is the region to fill
// op is the drawing operation: Src = copy source directly (blit),
// Over = alpha blend with existing
func (dw *Drawer) Fill(idx int, src2dst mat32.Mat3, reg image.Rectangle, op draw.Op) error {
	sy := &dw.Sys
	cmd := sy.CmdPool.Buff
	vars := sy.Vars()
	vars.BindDynValIdx(1, "Color", idx)
	sy.CmdBindVars(cmd, 0) // required to get offsets!

	tmat := dw.ConfigMats(src2dst, reg.Max, reg, op, false)
	matv, _ := vars.VarByNameTry(vgpu.PushSet, "Mats")
	fpl := sy.PipelineMap["fill"]
	fpl.Push(cmd, matv, vgpu.VertexShader, unsafe.Pointer(tmat))
	fpl.BindDrawVertex(cmd, 0)

	return nil
}

// StartFill starts color fill drawing rendering process on render target
func (dw *Drawer) StartFill() {
	sy := &dw.Sys
	fpl := sy.PipelineMap["fill"]
	if dw.Surf != nil {
		dw.Impl.SurfIdx = dw.Surf.AcquireNextImage()
		cmd := sy.CmdPool.Buff
		sy.ResetBeginRenderPass(cmd, dw.Surf.Frames[dw.Impl.SurfIdx], 0)
		fpl.BindPipeline(cmd)
	}
}

// EndFill ends color filling rendering process on render target
func (dw *Drawer) EndFill() {
	sy := &dw.Sys
	cmd := sy.CmdPool.Buff
	if dw.Surf != nil {
		sy.EndRenderPass(cmd)
		dw.Surf.SubmitRender(cmd) // this is where it waits for the 16 msec
		dw.Surf.PresentImage(dw.Impl.SurfIdx)
	}
}
