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
	"github.com/rajveermalviya/go-webgpu/wgpu"
)

// Render manages various elements needed for rendering,
// including a function to get a WebGPU RenderPass object,
// which specifies parameters for rendering to a Framebuffer.
// It holds the Depth buffer if one is used, and a multisampling image too.
// The Render object lives on the System, and any associated Surface,
// RenderFrame, and Framebuffers point to it.
type Render struct {

	// image format information for the framebuffer we render to
	Format TextureFormat

	// the associated depth buffer, if set
	Depth Texture

	// if this is not UndefType, depth format is used
	DepthFormat Types

	// for multisampling, this is the multisampled image that is the actual render target
	Multi Texture

	// is true if multsampled image configured
	HasMulti bool

	// host-accessible image that is used to transfer back from a render color attachment to host memory -- requires a different format than color attachment, and is TextureOnHostOnly flagged.
	Grab Texture

	// set this to true if it is not using a Surface render target
	// (i.e., it is a RenderFrame)
	NotSurface bool

	// values for clearing image when starting render pass
	ClearColor color.Color

	ClearDepth float32

	ClearStencil uint32

	device Device
}

func (rp *Render) Release() {
	rp.Depth.Release()
	rp.Multi.Release()
	rp.Grab.Release()
	// rp.GrabDepth.Free(rp.Dev)
}

func (rp *Render) SetSize(sz image.Point) {
	if rp.Format.Size != sz {
		rp.Format.Size = sz
		if rp.HasMulti {
			rp.Multi.ConfigMulti(&rp.device, &rp.Format)
		}
		if rp.DepthFormat != UndefType {
			rp.Depth.ConfigDepth(&rp.device, rp.DepthFormat, &rp.Format)
		}
	}
}

// Config configures the render pass for given device,
// Using standard parameters for graphics rendering,
// based on the given image format and depth image format
// (pass UndefType for no depth buffer).
func (rp *Render) Config(dev *Device, imgFmt *TextureFormat, depthFmt Types, notSurface bool) {
	rp.device = *dev
	rp.Format = *imgFmt
	rp.NotSurface = notSurface
	rp.ClearColor = colors.Black
	rp.ClearDepth = 1
	rp.ClearStencil = 0
	rp.DepthFormat = depthFmt
	if depthFmt != UndefType {
		rp.Depth.ConfigDepth(dev, rp.DepthFormat, imgFmt)
	}
	if rp.Format.Samples > 1 {
		rp.Multi.ConfigMulti(dev, imgFmt)
		rp.HasMulti = true
	}
}

// ClearRenderPass returns a render pass descriptor that clears the framebuffer
func (rp *Render) ClearRenderPass(view *wgpu.TextureView) *wgpu.RenderPassDescriptor {
	r, g, b, a := colors.ToFloat64(rp.ClearColor)
	rpd := &wgpu.RenderPassDescriptor{}
	if rp.Format.Samples > 1 && rp.Multi.view != nil {
		rpd.ColorAttachments = []wgpu.RenderPassColorAttachment{{
			View:          rp.Multi.view,
			ResolveTarget: view,
			LoadOp:        wgpu.LoadOpClear,
			ClearValue: wgpu.Color{
				R: r,
				G: g,
				B: b,
				A: a,
			},
			StoreOp: wgpu.StoreOpStore,
		}}
	} else {
		rpd.ColorAttachments = []wgpu.RenderPassColorAttachment{{
			View:   view,
			LoadOp: wgpu.LoadOpClear,
			ClearValue: wgpu.Color{
				R: r,
				G: g,
				B: b,
				A: a,
			},
			StoreOp: wgpu.StoreOpStore,
		}}
	}
	rp.SetDepthDescriptor(rpd)
	return rpd
}

// LoadRenderPass returns a render pass descriptor that loads previous framebuffer
func (rp *Render) LoadRenderPass(view *wgpu.TextureView) *wgpu.RenderPassDescriptor {
	rpd := &wgpu.RenderPassDescriptor{}
	if rp.Format.Samples > 1 && rp.Multi.view != nil {
		rpd.ColorAttachments = []wgpu.RenderPassColorAttachment{{
			View:          rp.Multi.view,
			ResolveTarget: view,
			LoadOp:        wgpu.LoadOpLoad,
			StoreOp:       wgpu.StoreOpStore,
		}}
	} else {
		rpd.ColorAttachments = []wgpu.RenderPassColorAttachment{{
			View:    view,
			LoadOp:  wgpu.LoadOpLoad,
			StoreOp: wgpu.StoreOpStore,
		}}
	}
	rp.SetDepthDescriptor(rpd)
	return rpd
}

func (rp *Render) SetDepthDescriptor(rpd *wgpu.RenderPassDescriptor) {
	if rp.Depth.texture == nil {
		return
	}
	rpd.DepthStencilAttachment = &wgpu.RenderPassDepthStencilAttachment{
		View:              rp.Depth.view,
		DepthClearValue:   rp.ClearDepth,
		DepthLoadOp:       wgpu.LoadOpClear,
		DepthStoreOp:      wgpu.StoreOpStore,
		DepthReadOnly:     false,
		StencilClearValue: rp.ClearStencil,
		StencilLoadOp:     wgpu.LoadOpClear,
		StencilStoreOp:    wgpu.StoreOpStore,
		StencilReadOnly:   true,
	}

}

// BeginRenderPass adds commands to the given command buffer
// to start the render pass on given framebuffer.
// Clears the frame first, according to the ClearValues
// See BeginRenderPassNoClear for non-clearing version.
func (rp *Render) BeginRenderPass(cmd *wgpu.CommandEncoder, view *wgpu.TextureView) *wgpu.RenderPassEncoder {
	return rp.BeginRenderPassImpl(cmd, view, true)
}

// BeginRenderPassNoClear adds commands to the given command buffer
// to start the render pass on given framebuffer.
// does NOT clear the frame first -- loads prior state.
func (rp *Render) BeginRenderPassNoClear(cmd *wgpu.CommandEncoder, view *wgpu.TextureView) *wgpu.RenderPassEncoder {
	return rp.BeginRenderPassImpl(cmd, view, false)
}

// BeginRenderPassImpl adds commands to the given command buffer
// to start the render pass on given framebuffer.
// If clear = true, clears the frame according to the ClearColor.
func (rp *Render) BeginRenderPassImpl(cmd *wgpu.CommandEncoder, view *wgpu.TextureView, clear bool) *wgpu.RenderPassEncoder {
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
		rpd = rp.ClearRenderPass(view)
	} else {
		rpd = rp.LoadRenderPass(view)
	}
	return cmd.BeginRenderPass(rpd)
}

// ConfigGrab configures the Grab for copying rendered image
// back to host memory.  Uses format of current Texture.
func (rp *Render) ConfigGrab(dev *Device) {
	if rp.Grab.texture != nil && rp.Grab.Format.Size == rp.Format.Size {
		return
	}
	rp.Grab.Format.Defaults()
	rp.Grab.Format = rp.Format
	rp.Grab.Format.SetMultisample(1) // can't have for grabs
	rp.Grab.device = *dev
	rp.Grab.CreateTexture(wgpu.TextureUsageCopySrc) // todo: not sure what else?
}

// https://www.reddit.com/r/WebGPU/comments/7yhvep/retrieve_depth_attachment_from_framebuffer/
// https://pastebin.com/33MxSNmh
// have to copy to a buffer then sync that back down
// create the buffer as the GrabDepth item

// ConfigGrabDepth configures the GrabDepth for copying depth image
// back to host memory.  Uses format of current Depth image.
func (rp *Render) ConfigGrabDepth(dev *Device) {
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
func (rp *Render) GrabDepthTexture(dev *Device, cmd *wgpu.CommandEncoder) error {
	nsamp := rp.Format.Samples
	if nsamp > 1 {
		err := errors.New("gpu.Render.GrabDepthTexture(): does not work if multisampling is > 1")
		if Debug {
			log.Println(err)
		}
		return err
	}
	rp.ConfigGrabDepth(dev) // ensure image grab setup
	return nil
}

// DepthTextureArray returns the float values from the last GrabDepthTexture call
// automatically handles down-sampling from multisampling.
func (rp *Render) DepthTextureArray() ([]float32, error) {
	return nil, nil
}
