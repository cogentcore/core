// Copyright 2022 The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package vdraw

import (
	"image"
	"log"

	"github.com/goki/vgpu/vgpu"
)

// maxPerStageDescriptorSamplers is only 16 on mac -- this is the relevant limit on textures!
// also maxPerStageDescriptorSampledImages is basically the same:
// https://vulkan.gpuinfo.org/displaydevicelimit.php?name=maxPerStageDescriptorSamplers&platform=all
// https://vulkan.gpuinfo.org/displaydevicelimit.php?name=maxPerStageDescriptorSampledImages&platform=all

const (
	// MaxImages is the maximum number of image variables that can be used
	// in one render pass.  Multiple render passes can be used to get around this,
	// but it is much more efficient to stay within this limit.
	// Arrays of textures are possible for sprites, for example.
	MaxImages = 16
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
}

// ConfigSurface configures the Drawer to use given surface as a render target.
// maxImages is maximum number of images that can be used per pass
// this cannot exceed MaxImages (16).
func (dw *Drawer) ConfigSurface(sf *vgpu.Surface, maxImages int) {
	if maxImages > MaxImages {
		log.Printf("vdraw.Drawer can only use a maximum of %d images, due to limitations across platforms.  The requested number: %d will be reduced accordingly\n", MaxImages, maxImages)
		maxImages = MaxImages
	}
	dw.Impl.MaxImages = maxImages
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
// maxImages is maximum number of images that can be used per pass.
// this cannot exceed MaxImages (16).
func (dw *Drawer) ConfigFrame(dev *vgpu.Device, size image.Point, maxImages int) {
	if maxImages > MaxImages {
		log.Printf("vdraw.Drawer can only use a maximum of %d images, due to limitations across platforms.  The requested number: %d will be reduced accordingly\n", MaxImages, maxImages)
		maxImages = MaxImages
	}
	dw.Impl.MaxImages = maxImages
	dw.Frame = vgpu.NewRenderFrame(vgpu.TheGPU, dev, size)
	dw.Sys.InitGraphics(vgpu.TheGPU, "vdraw.Drawer", &dw.Frame.Device)
	dw.Sys.ConfigRenderNonSurface(&dw.Frame.Format, vgpu.UndefType)
	dw.Frame.SetRender(&dw.Sys.Render)

	dw.ConfigSys()
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
