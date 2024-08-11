// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gpu

import (
	"errors"
	"image"
	"image/color"
	"log"

	"cogentcore.org/core/colors"
	"github.com/cogentcore/webgpu/wgpu"
)

// Render manages various elements needed for rendering,
// including a function to get a WebGPU RenderPass object,
// which specifies parameters for rendering to a Texture.
// It holds the Depth buffer if one is used, and a multisampling image too.
// The Render object is owned by a Renderer (Surface or RenderTexture).
type Render struct {
	// texture format information for the texture target we render to.
	// critically, this can be different from the surface actual format
	// in the case when that format is non-srgb, as is the case in web browsers.
	Format TextureFormat

	// the associated depth buffer, if set
	Depth Texture

	// if this is not UndefinedType, depth format is used
	DepthFormat Types

	// for multisampling, this is the multisampled image that is the actual render target
	Multi Texture

	// is true if multsampled image configured
	HasMulti bool

	// host-accessible image that is used to transfer back
	// from a render color attachment to host memory.
	// Requires a different format than color attachment
	Grab Texture

	// values for clearing image when starting render pass
	ClearColor color.Color

	ClearDepth float32

	ClearStencil uint32

	device Device
}

// Config configures the render pass for given device,
// Using standard parameters for graphics rendering,
// based on the given image format and depth image format
// (pass UndefinedType for no depth buffer, or Depth32).
func (rd *Render) Config(dev *Device, imgFmt *TextureFormat, depthFmt Types) {
	rd.device = *dev
	rd.Format = *imgFmt
	rd.ClearColor = colors.Black
	rd.ClearDepth = 1
	rd.ClearStencil = 0
	rd.DepthFormat = depthFmt
	if depthFmt != UndefinedType {
		rd.Depth.ConfigDepth(dev, rd.DepthFormat, imgFmt)
	}
	if rd.Format.Samples > 1 {
		rd.Multi.ConfigMulti(dev, imgFmt)
		rd.HasMulti = true
	}
}

func (rd *Render) Release() {
	rd.Depth.Release()
	rd.Multi.Release()
	rd.Grab.Release()
	// rp.GrabDepth.Free(rp.Dev)
}

func (rd *Render) SetSize(sz image.Point) {
	if rd.Format.Size == sz {
		return
	}
	rd.Format.Size = sz
	if rd.HasMulti {
		rd.Multi.ConfigMulti(&rd.device, &rd.Format)
	}
	if rd.DepthFormat != UndefinedType {
		rd.Depth.ConfigDepth(&rd.device, rd.DepthFormat, &rd.Format)
	}
}

// ClearRenderPass returns a render pass descriptor that clears the framebuffer
func (rd *Render) ClearRenderPass(view *wgpu.TextureView) *wgpu.RenderPassDescriptor {
	r, g, b, a := colors.ToFloat32(rd.ClearColor)
	r, g, b = SRGBToLinear(r, g, b)
	clearColor := wgpu.Color{R: float64(r), G: float64(g), B: float64(b), A: float64(a)}
	rpd := &wgpu.RenderPassDescriptor{}
	if rd.Format.Samples > 1 && rd.Multi.view != nil {
		rpd.Label = "ClearMulti1"
		rpd.ColorAttachments = []wgpu.RenderPassColorAttachment{{
			View:          rd.Multi.view,
			ResolveTarget: view,
			LoadOp:        wgpu.LoadOpClear,
			ClearValue:    clearColor,
			StoreOp:       wgpu.StoreOpStore,
		}}
	} else {
		rpd.Label = "Clear1"
		rpd.ColorAttachments = []wgpu.RenderPassColorAttachment{{
			View:       view,
			LoadOp:     wgpu.LoadOpClear,
			ClearValue: clearColor,
			StoreOp:    wgpu.StoreOpStore,
		}}
	}
	rd.SetDepthDescriptor(rpd)
	return rpd
}

// LoadRenderPass returns a render pass descriptor that loads previous framebuffer
func (rd *Render) LoadRenderPass(view *wgpu.TextureView) *wgpu.RenderPassDescriptor {
	rpd := &wgpu.RenderPassDescriptor{}
	if rd.Format.Samples > 1 && rd.Multi.view != nil {
		rpd.Label = "LoadMulti"
		rpd.ColorAttachments = []wgpu.RenderPassColorAttachment{{
			View:          rd.Multi.view,
			ResolveTarget: view,
			LoadOp:        wgpu.LoadOpLoad,
			StoreOp:       wgpu.StoreOpStore,
		}}
	} else {
		rpd.Label = "Load1"
		rpd.ColorAttachments = []wgpu.RenderPassColorAttachment{{
			View:    view,
			LoadOp:  wgpu.LoadOpLoad,
			StoreOp: wgpu.StoreOpStore,
		}}
	}
	rd.SetDepthDescriptor(rpd)
	return rpd
}

func (rd *Render) SetDepthDescriptor(rpd *wgpu.RenderPassDescriptor) {
	if rd.Depth.texture == nil {
		return
	}
	rpd.DepthStencilAttachment = &wgpu.RenderPassDepthStencilAttachment{
		View:              rd.Depth.view,
		DepthClearValue:   rd.ClearDepth,
		DepthLoadOp:       wgpu.LoadOpClear,
		DepthStoreOp:      wgpu.StoreOpStore,
		DepthReadOnly:     false,
		StencilClearValue: rd.ClearStencil,
		StencilLoadOp:     wgpu.LoadOpClear,
		StencilStoreOp:    wgpu.StoreOpStore,
		StencilReadOnly:   true,
	}

}

// BeginRenderPass adds commands to the given command buffer
// to start the render pass on given framebuffer.
// Clears the frame first, according to the ClearValues
// See BeginRenderPassNoClear for non-clearing version.
func (rd *Render) BeginRenderPass(cmd *wgpu.CommandEncoder, view *wgpu.TextureView) *wgpu.RenderPassEncoder {
	return rd.beginRenderPass(cmd, view, true)
}

// BeginRenderPassNoClear adds commands to the given command buffer
// to start the render pass on given framebuffer.
// does NOT clear the frame first -- loads prior state.
func (rd *Render) BeginRenderPassNoClear(cmd *wgpu.CommandEncoder, view *wgpu.TextureView) *wgpu.RenderPassEncoder {
	return rd.beginRenderPass(cmd, view, false)
}

func (rd *Render) beginRenderPass(cmd *wgpu.CommandEncoder, view *wgpu.TextureView, clear bool) *wgpu.RenderPassEncoder {
	// w, h := fr.Texture.Format.Size32()
	// clearValues := rp.ClearValues
	// vrp := rp.VkClearPass
	// if !clear && fr.HasCleared {
	// 	clearValues = nil
	// 	vrp = rp.VkLoadPass
	// }
	// fr.HasCleared = true

	var rpd *wgpu.RenderPassDescriptor
	if clear {
		rpd = rd.ClearRenderPass(view)
	} else {
		rpd = rd.LoadRenderPass(view)
	}
	return cmd.BeginRenderPass(rpd)
}

// ConfigGrab configures the Grab for copying rendered image
// back to host memory.  Uses format of current Texture.
func (rd *Render) ConfigGrab(dev *Device) {
	if rd.Grab.texture != nil && rd.Grab.Format.Size == rd.Format.Size {
		return
	}
	rd.Grab.Format.Defaults()
	rd.Grab.Format = rd.Format
	rd.Grab.Format.SetMultisample(1) // can't have for grabs
	rd.Grab.device = *dev
	rd.Grab.CreateTexture(wgpu.TextureUsageCopySrc) // todo: not sure what else?
}

// https://www.reddit.com/r/WebGPU/comments/7yhvep/retrieve_depth_attachment_from_framebuffer/
// https://pastebin.com/33MxSNmh
// have to copy to a buffer then sync that back down
// create the buffer as the GrabDepth item

// ConfigGrabDepth configures the GrabDepth for copying depth image
// back to host memory.  Uses format of current Depth image.
func (rd *Render) ConfigGrabDepth(dev *Device) {
	// bsz := rp.Format.Size.X * rp.Format.Size.Y * 4 // 32 bit = 4 bytes per pixel
	// if rp.GrabDepth.Active {
	// 	if rp.GrabDepth.Size == bsz {
	// 		return
	// 	}
	// 	rp.GrabDepth.Free(dev)
	// }
	// rp.GrabDepth.Type = StorageBuffer
	// rp.GrabDepth.AllocMem(dev, bsz)
}

// GrabDepthTexture grabs the current render depth image, using given command buffer
// which must have the cmdBegin called already.  Uses the GrabDepth Storage Buffer.
// call this after: sys.MemCmdEndSubmitWaitFree()
func (rd *Render) GrabDepthTexture(dev *Device, cmd *wgpu.CommandEncoder) error {
	nsamp := rd.Format.Samples
	if nsamp > 1 {
		err := errors.New("gpu.Render.GrabDepthTexture(): does not work if multisampling is > 1")
		if Debug {
			log.Println(err)
		}
		return err
	}
	rd.ConfigGrabDepth(dev) // ensure image grab setup
	return nil
}

// DepthTextureArray returns the float values from the last GrabDepthTexture call
// automatically handles down-sampling from multisampling.
func (rd *Render) DepthTextureArray() ([]float32, error) {
	return nil, nil
}
