// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gpu

import (
	"errors"
	"image"
	"log"

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

	// is true if configured with depth buffer
	HasDepth bool

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

	sys    *System
	device *Device
}

func (rp *Render) Release() {
	rp.Depth.Release()
	rp.Multi.Release()
	rp.Grab.Release()
	// rp.GrabDepth.Free(rp.Dev)
}

// Config configures the render pass for given device,
// Using standard parameters for graphics rendering,
// based on the given image format and depth image format
// (pass UndefType for no depth buffer).
func (rp *Render) Config(dev *Device, depthFmt Types, notSurface bool) {
	rp.NotSurface = notSurface
	rp.ClearColor = colors.Black
	rp.SetClearDepthStencil(1, 0)
}

// ClearRenderPass returns a render pass descriptor that clears the framebuffer
func (rp *Render) ClearRenderPass(view *wgpu.TextureView) *wgpu.RenderPassDescriptor {
	r, g, b, a := colors.ToFloat32(rp.ClearColor)
	return &wgpu.RenderPassDescriptor{
		ColorAttachments: []wgpu.RenderPassColorAttachment{{
			View:   view,
			LoadOp: wgpu.LoadOp_Clear,
			ClearValue: wgpu.Color{
				R: r,
				G: g,
				B: b,
				A: a,
			},
			StoreOp: wgpu.StoreOp_Store,
		}},
	}
}

// LoadRenderPass returns a render pass descriptor that loads previous framebuffer
func (rp *Render) LoadRenderPass(view *wgpu.TextureView) *wgpu.RenderPassDescriptor {
	return &wgpu.RenderPassDescriptor{
		ColorAttachments: []wgpu.RenderPassColorAttachment{{
			View:    view,
			LoadOp:  wgpu.LoadOpLoad,
			StoreOp: wgpu.StoreOpStore,
		}},
	}
}

// SetClearDepthStencil sets the depth and stencil values when starting new render
func (rp *Render) SetClearDepthStencil(depth float32, stencil uint32) {
	// if len(rp.ClearValues) == 0 {
	// 	rp.ClearValues = make([]vk.ClearValue, 2)
	// }
	// rp.ClearValues[1].SetDepthStencil(depth, stencil)
}

// BeginRenderPass adds commands to the given command buffer
// to start the render pass on given framebuffer.
// Clears the frame first, according to the ClearValues
// See BeginRenderPassNoClear for non-clearing version.
func (rp *Render) BeginRenderPass(cmd *wgpu.CommandEncoder, view *wgpu.TextureView) *wgpu.RenderPass {
	return rp.BeginRenderPassImpl(cmd, view, true)
}

// BeginRenderPassNoClear adds commands to the given command buffer
// to start the render pass on given framebuffer.
// does NOT clear the frame first -- loads prior state.
func (rp *Render) BeginRenderPassNoClear(cmd *wgpu.CommandEncoder, view *wgpu.TextureView) *wgpu.RenderPass {
	return rp.BeginRenderPassImpl(cmd, view, false)
}

// BeginRenderPassImpl adds commands to the given command buffer
// to start the render pass on given framebuffer.
// If clear = true, clears the frame according to the ClearColor.
func (rp *Render) BeginRenderPassImpl(cmd *wgpu.CommandEncoder, view *wgpu.TextureView, clear bool) *wgpu.RenderPass {
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
	if rp.Grab.IsActive() {
		if rp.Grab.Format.Size == rp.Format.Size {
			return
		}
		rp.Grab.SetSize(rp.Format.Size)
		return
	}
	rp.Grab.Format.Defaults()
	rp.Grab.Format = rp.Format
	rp.Grab.Format.SetMultisample(1) // can't have for grabs
	// rp.Grab.SetFlag(true, TextureOnHostOnly)
	rp.Grab.Dev = dev
	rp.Grab.GPU = rp.Sys.GPU
	// rp.Grab.AllocTexture()
}

// https://www.reddit.com/r/WebGPU/comments/7yhvep/retrieve_depth_attachment_from_framebuffer/
// https://pastebin.com/33MxSNmh
// have to copy to a buffer then sync that back down
// create the buffer as the GrabDepth item

// ConfigGrabDepth configures the GrabDepth for copying depth image
// back to host memory.  Uses format of current Depth image.
func (rp *Render) ConfigGrabDepth(dev *Device) {
	bsz := rp.Format.Size.X * rp.Format.Size.Y * 4 // 32 bit = 4 bytes per pixel
	if rp.GrabDepth.Active {
		if rp.GrabDepth.Size == bsz {
			return
		}
		rp.GrabDepth.Free(dev)
	}
	rp.GrabDepth.GPU = rp.Sys.GPU
	rp.GrabDepth.Type = StorageBuffer
	// rp.GrabDepth.AllocMem(dev, bsz)
}

// GrabDepthTexture grabs the current render depth image, using given command buffer
// which must have the cmdBegin called already.  Uses the GrabDepth Storage Buffer.
// call this after: sys.MemCmdEndSubmitWaitFree()
func (rp *Render) GrabDepthTexture(dev *Device, cmd *wgpu.CommandEncoder) error {
	nsamp := rp.Format.NSamples()
	if nsamp > 1 {
		err := errors.New("gpu.Render.GrabDepthTexture(): does not work if multisampling is > 1")
		if Debug {
			log.Println(err)
		}
		return err
	}
	rp.ConfigGrabDepth(dev) // ensure image grab setup
	// first, prepare TextureGrab to receive copy from render image.
	// apparently, the color attachment, with src flag already set, does not need this.

	// reg := vk.BufferTextureCopy{BufferOffset: 0, BufferRowLength: 0, BufferTextureHeight: 0}
	// reg.TextureSubresource.AspectMask = vk.TextureAspectFlags(vk.TextureAspectDepthBit)
	// reg.TextureSubresource.MipLevel = 0
	// reg.TextureSubresource.BaseArrayLayer = 0
	// reg.TextureSubresource.LayerCount = 1
	// reg.TextureOffset.X, reg.TextureOffset.Y, reg.TextureOffset.Z = 0, 0, 0
	// reg.TextureExtent.Width = uint32(rp.Format.Size.X)
	// reg.TextureExtent.Height = uint32(rp.Format.Size.Y)
	// reg.TextureExtent.Depth = 1
	//
	// vk.CmdCopyTextureToBuffer(cmd, rp.Depth.Texture, vk.TextureLayoutTransferSrcOptimal, rp.GrabDepth.Host, 1, []vk.BufferTextureCopy{reg})
	return nil
}

// DepthTextureArray returns the float values from the last GrabDepthTexture call
// automatically handles down-sampling from multisampling.
func (rp *Render) DepthTextureArray() ([]float32, error) {
	/*
		if rp.GrabDepth.Host == vk.NullBuffer {
			err := errors.New("DepthTextureArray: No GrabDepth.Host buffer -- must call GrabDepthTexture")
			if Debug {
				log.Println(err)
			}
			return nil, err
		}
		sz := rp.Format.Size
		fsz := sz.X * sz.Y
		ary := make([]float32, fsz)
		fp := (*[ByteCopyMemoryLimit]float32)(rp.GrabDepth.HostPtr)[0:fsz]
		copy(ary, fp)
	*/
	// note: you cannot specify a greater width than actual width
	// and resolving depth images GPU-side is not exactly clear:
	// https://community.khronos.org/t/how-to-resolve-multi-sampled-depth-images/7584
	// https://www.reddit.com/r/WebGPU/comments/rpeywp/is_it_possible_to_resolve_a_depth_msaa_buffer/
	// furthermore, the function for resolving the multiple samples is not obvious either -- average
	// is implemented below:
	// for y := 0; y < sz.Y; y++ {
	// 	for x := 0; x < sz.X; x++ {
	// 		sum := float32(0)
	// 		for ys := 0; ys < ns2; ys++ {
	// 			for xs := 0; xs < ns2; xs++ {
	// 				si := (y*ns2+ys)*sz.X*ns2 + x*ns2 + xs
	// 				sum += fp[si]
	// 			}
	// 		}
	// 		di := y*sz.X + x
	// 		ary[di] = sum / float32(nsamp)
	// 	}
	// }
	return nil, nil
}
