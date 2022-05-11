// Copyright 2022 The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package vdraw

import (
	"image"

	"github.com/goki/vgpu/vgpu"
)

// Drawer is the vDraw implementation, which draws Textures
// or Fills solid colors to a render target (Surface, Framebuffer).
// Image and color palette must be set prior to a given render pass.
// Multiple fill operations can be performed in one pass, but only
// one Image can be used at a time.
type Drawer struct {
	Sys     vgpu.System   `desc:"drawing system"`
	Surf    *vgpu.Surface `desc:"surface if render target"`
	YIsDown bool          `desc:"render so the Y axis points down, with 0,0 at the upper left, which is the Vulkan standard.  default is Y is up, with 0,0 at bottom left, which is OpenGL default.  this must be set prior to configuring, the surface, as it determines the rendering parameters."`
	Impl    DrawerImpl    `desc:"implementation state -- ignore"`
}

// ConfigSurface configures the Drawer to use given surface as a render target
// maxColors is maximum number of fill colors in palette
func (dw *Drawer) ConfigSurface(sf *vgpu.Surface, maxColors int) {
	dw.Impl.MaxColors = maxColors
	dw.Surf = sf
	dw.Sys.InitGraphics(sf.GPU, "vdraw.Drawer", &sf.Device)
	dw.Sys.RenderPass.NoClear = true
	dw.Sys.ConfigRenderPass(&dw.Surf.Format, vgpu.UndefType)
	sf.SetRenderPass(&dw.Sys.RenderPass)

	dw.ConfigSys()
}

func (dw *Drawer) Destroy() {
	dw.Sys.Destroy()
}

// DestSize returns the size of the render destination
func (dw *Drawer) DestSize() image.Point {
	if dw.Surf != nil {
		return dw.Surf.Format.Size
	}
	return image.Point{10, 10}
}
