// Copyright 2022 The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package vdraw

import (
	"image"
	"sync"

	"github.com/goki/vgpu/vgpu"
)

// Drawer is the vDraw implementation, which draws Textures
// or Fills solid colors to a render target (Surface, RenderFrame).
// Image and color palette must be set prior to a given render pass.
// Multiple fill operations can be performed in one pass, but only
// one Image can be used at a time.
type Drawer struct {
	Sys     vgpu.System       `desc:"drawing system"`
	Surf    *vgpu.Surface     `desc:"surface if render target"`
	Frame   *vgpu.RenderFrame `desc:"render frame if render target"`
	YIsDown bool              `desc:"render so the Y axis points down, with 0,0 at the upper left, which is the Vulkan standard.  default is Y is up, with 0,0 at bottom left, which is OpenGL default.  this must be set prior to configuring, the surface, as it determines the rendering parameters."`
	Impl    DrawerImpl        `desc:"implementation state -- ignore"`
	UpdtMu  sync.Mutex        `view:"-" copy:"-" json:"-" xml:"-" desc:"mutex on updating"`
}

// ConfigSurface configures the Drawer to use given surface as a render target.
// maxTextures is maximum number of images that can be used per pass.
// If maxTextures > vgpu.MaxTexturesPerSet (16) then multiple descriptor sets
// are used to hold more textures.
func (dw *Drawer) ConfigSurface(sf *vgpu.Surface, maxTextures int) {
	dw.Impl.MaxTextures = maxTextures
	dw.Surf = sf
	dw.Sys.InitGraphics(sf.GPU, "vdraw.Drawer", &sf.Device)
	dw.Sys.ConfigRender(&dw.Surf.Format, vgpu.UndefType)
	sf.SetRender(&dw.Sys.Render)

	dw.ConfigSys()
}

// ConfigFrame configures the Drawer to use a RenderFrame as a render target,
// of given size.  Use dw.Frame.SetSize to resize later.
// Frame is owned and managed by the Drawer.
// Uses given Device -- if nil, one is made.
// If maxTextures > vgpu.MaxTexturesPerSet (16) then multiple descriptor sets
// are used to hold more textures.
func (dw *Drawer) ConfigFrame(dev *vgpu.Device, size image.Point, maxTextures int) {
	dw.Impl.MaxTextures = maxTextures
	dw.Frame = vgpu.NewRenderFrame(vgpu.TheGPU, dev, size)
	dw.Sys.InitGraphics(vgpu.TheGPU, "vdraw.Drawer", &dw.Frame.Device)
	dw.Sys.ConfigRenderNonSurface(&dw.Frame.Format, vgpu.UndefType)
	dw.Frame.SetRender(&dw.Sys.Render)

	dw.ConfigSys()
}

// SetMaxTextures updates the max number of textures for drawing
// Must call this prior to doing any allocation of images.
func (dw *Drawer) SetMaxTextures(maxTextures int) {
	sy := &dw.Sys
	vars := sy.Vars()
	txset := vars.SetMap[0]
	txset.ConfigVals(maxTextures)
	dw.Impl.MaxTextures = maxTextures
	vars.NDescs = vgpu.NDescForTextures(dw.Impl.MaxTextures)
	vars.Config() // update after config changes
}

func (dw *Drawer) Destroy() {
	dw.Sys.Destroy()
	if dw.Frame != nil {
		dw.Frame.Destroy()
		dw.Frame = nil
	}
}

// DestSize returns the size of the render destination
func (dw *Drawer) DestSize() image.Point {
	if dw.Surf != nil {
		return dw.Surf.Format.Size
	} else {
		return dw.Frame.Format.Size
	}
}
