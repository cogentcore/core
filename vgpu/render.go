// Copyright (c) 2022, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// This is initially adapted from https://github.com/vulkan-go/asche
// Copyright © 2017 Maxim Kupriianov <max@kc.vc>, under the MIT License

package vgpu

import (
	"errors"
	"image"
	"log"

	vk "github.com/goki/vulkan"
)

// Render manages various elements needed for rendering,
// including a vulkan RenderPass object,
// which specifies parameters for rendering to a Framebuffer.
// It holds the Depth buffer if one is used, and a multisampling image too.
// The Render object lives on the System, and any associated Surface,
// RenderFrame, and Framebuffers point to it.
type Render struct {
	Sys        *System         `desc:"system that we belong to and manages all shared resources (Memory, Vars, Vals, etc), etc"`
	Dev        vk.Device       `desc:"the device we're associated with -- this must be the same device that owns the Framebuffer -- e.g., the Surface"`
	Format     ImageFormat     `desc:"image format information for the framebuffer we render to"`
	Depth      Image           `desc:"the associated depth buffer, if set"`
	HasDepth   bool            `desc:"is true if configured with depth buffer"`
	Multi      Image           `desc:"for multisampling, this is the multisampled image that is the actual render target"`
	HasMulti   bool            `desc:"is true if multsampled image configured"`
	Grab       Image           `desc:"host-accessible image that is used to transfer back from a render color attachment to host memory -- requires a different format than color attachment, and is ImageOnHostOnly flagged."`
	GrabDepth  MemBuff         `desc:"host-accessible buffer for grabbing the depth map -- must go to a buffer and not an image"`
	NotSurface bool            `desc:"set this to true if it is not using a Surface render target (i.e., it is a RenderFrame)"`
	ClearVals  []vk.ClearValue `desc:"values for clearing image when starting render pass"`

	VkClearPass vk.RenderPass `desc:"the vulkan renderpass config that clears target first"`
	VkLoadPass  vk.RenderPass `desc:"the vulkan renderpass config that does not clear target first (loads previous)"`
}

func (rp *Render) Destroy() {
	if rp.VkClearPass == vk.NullRenderPass {
		return
	}
	vk.DestroyRenderPass(rp.Dev, rp.VkClearPass, nil)
	vk.DestroyRenderPass(rp.Dev, rp.VkLoadPass, nil)
	rp.VkClearPass = vk.NullRenderPass
	rp.VkLoadPass = vk.NullRenderPass
	rp.Depth.Destroy()
	rp.Multi.Destroy()
	rp.Grab.Destroy()
	rp.GrabDepth.Free(rp.Dev)
}

// Config configures the render pass for given device,
// Using standard parameters for graphics rendering,
// based on the given image format and depth image format
// (pass UndefType for no depth buffer).
func (rp *Render) Config(dev vk.Device, imgFmt *ImageFormat, depthFmt Types, notSurface bool) {
	rp.NotSurface = notSurface
	rp.SetClearColor(0, 0, 0, 1)
	rp.SetClearDepthStencil(1, 0)
	rp.VkClearPass = rp.ConfigImpl(dev, imgFmt, depthFmt, true)
	rp.VkLoadPass = rp.ConfigImpl(dev, imgFmt, depthFmt, false)
}

func (rp *Render) ConfigImpl(dev vk.Device, imgFmt *ImageFormat, depthFmt Types, clear bool) vk.RenderPass {
	// The initial layout for the color and depth attachments will be vk.LayoutUndefined
	// because at the start of the renderpass, we don't care about their contents.
	// At the start of the subpass, the color attachment's layout will be transitioned
	// to vk.LayoutColorAttachmentOptimal and the depth stencil attachment's layout
	// will be transitioned to vk.LayoutDepthStencilAttachmentOptimal.  At the end of
	// the renderpass, the color attachment's layout will be transitioned to
	// vk.LayoutPresentSrc to be ready to present.  This is all done as part of
	// the renderpass, no barriers are necessary.
	rp.Dev = dev
	rp.Format = *imgFmt
	rp.HasDepth = false

	ca := vk.AttachmentDescription{
		Format:         rp.Format.Format,
		Samples:        rp.Format.Samples,
		LoadOp:         vk.AttachmentLoadOpClear,
		StoreOp:        vk.AttachmentStoreOpStore,
		StencilLoadOp:  vk.AttachmentLoadOpDontCare,
		StencilStoreOp: vk.AttachmentStoreOpDontCare,
		InitialLayout:  vk.ImageLayoutUndefined,
		FinalLayout:    vk.ImageLayoutPresentSrc,
	}

	if !clear {
		ca.LoadOp = vk.AttachmentLoadOpLoad
		ca.InitialLayout = vk.ImageLayoutPresentSrc
	}

	atta := []vk.AttachmentDescription{ca}

	if depthFmt != UndefType {
		rp.HasDepth = true
		rp.Depth.ConfigDepth(rp.Sys.GPU, dev, depthFmt, imgFmt)
		depthAttach := vk.AttachmentDescription{
			Format:         rp.Depth.Format.Format,
			Samples:        rp.Depth.Format.Samples,
			LoadOp:         vk.AttachmentLoadOpClear,
			StoreOp:        vk.AttachmentStoreOpDontCare,
			StencilLoadOp:  vk.AttachmentLoadOpDontCare,
			StencilStoreOp: vk.AttachmentStoreOpDontCare,
			InitialLayout:  vk.ImageLayoutUndefined,
			FinalLayout:    vk.ImageLayoutDepthStencilAttachmentOptimal,
		}
		atta = append(atta, depthAttach)
	}

	if rp.Format.Samples != vk.SampleCount1Bit {
		rp.HasMulti = true
		ca.FinalLayout = vk.ImageLayoutColorAttachmentOptimal
		rp.Multi.ConfigMulti(rp.Sys.GPU, dev, &rp.Format)
		resolveAttach := vk.AttachmentDescription{
			Format:         rp.Format.Format,
			Samples:        vk.SampleCount1Bit,
			LoadOp:         vk.AttachmentLoadOpDontCare,
			StoreOp:        vk.AttachmentStoreOpStore,
			StencilLoadOp:  vk.AttachmentLoadOpDontCare,
			StencilStoreOp: vk.AttachmentStoreOpDontCare,
			InitialLayout:  vk.ImageLayoutUndefined,
			FinalLayout:    vk.ImageLayoutPresentSrc,
		}
		if rp.NotSurface { // transfer source
			resolveAttach.FinalLayout = vk.ImageLayoutTransferSrcOptimal
		}
		atta = append(atta, resolveAttach)
	} else {
		if rp.NotSurface { // transfer from color attach
			ca.FinalLayout = vk.ImageLayoutTransferSrcOptimal
		}
	}

	var renderPass vk.RenderPass
	rpcreate := &vk.RenderPassCreateInfo{
		SType:           vk.StructureTypeRenderPassCreateInfo,
		AttachmentCount: uint32(len(atta)),
		PAttachments:    atta,
		SubpassCount:    1,
		PSubpasses: []vk.SubpassDescription{{
			PipelineBindPoint:    vk.PipelineBindPointGraphics,
			ColorAttachmentCount: 1,
			PColorAttachments: []vk.AttachmentReference{{
				Attachment: 0,
				Layout:     vk.ImageLayoutColorAttachmentOptimal,
			}},
		}},
	}
	if rp.HasDepth {
		rpcreate.PSubpasses[0].PDepthStencilAttachment = &vk.AttachmentReference{
			Attachment: 1,
			Layout:     vk.ImageLayoutDepthStencilAttachmentOptimal,
		}
		dep := vk.SubpassDependency{
			SrcSubpass:    vk.SubpassExternal,
			DstSubpass:    0,
			SrcStageMask:  vk.PipelineStageFlags(vk.PipelineStageColorAttachmentOutputBit | vk.PipelineStageEarlyFragmentTestsBit),
			SrcAccessMask: 0,
			DstStageMask:  vk.PipelineStageFlags(vk.PipelineStageColorAttachmentOutputBit | vk.PipelineStageEarlyFragmentTestsBit),
			DstAccessMask: vk.AccessFlags(vk.AccessColorAttachmentWriteBit | vk.AccessDepthStencilAttachmentWriteBit),
		}
		rpcreate.DependencyCount = 1
		rpcreate.PDependencies = []vk.SubpassDependency{dep}
	}
	if rp.HasMulti {
		dpat := 2
		if !rp.HasDepth {
			dpat = 1
		}
		rpcreate.PSubpasses[0].PResolveAttachments = []vk.AttachmentReference{{
			Attachment: uint32(dpat),
			Layout:     vk.ImageLayoutColorAttachmentOptimal,
		}}
	}

	ret := vk.CreateRenderPass(dev, rpcreate, nil, &renderPass)
	IfPanic(NewError(ret))
	return renderPass
}

// SetSize sets updated size of the render target -- resizes depth and multi buffers as needed
func (rp *Render) SetSize(size image.Point) {
	rp.Format.Size = size
	if rp.HasDepth {
		if rp.Depth.SetSize(size) {
			rp.Depth.ConfigDepthView()
		}
	}
	if rp.HasMulti {
		if rp.Multi.SetSize(size) {
			rp.Multi.ConfigStdView()
		}
	}
}

// SetClearColor sets the RGBA colors to set when starting new render
func (rp *Render) SetClearColor(r, g, b, a float32) {
	if len(rp.ClearVals) == 0 {
		rp.ClearVals = make([]vk.ClearValue, 2)
	}
	rp.ClearVals[0].SetColor([]float32{r, g, b, a})
}

// SetClearDepthStencil sets the depth and stencil values when starting new render
func (rp *Render) SetClearDepthStencil(depth float32, stencil uint32) {
	if len(rp.ClearVals) == 0 {
		rp.ClearVals = make([]vk.ClearValue, 2)
	}
	rp.ClearVals[1].SetDepthStencil(depth, stencil)
}

// BeginRenderPass adds commands to the given command buffer
// to start the render pass on given framebuffer.
// Clears the frame first, according to the ClearVals
// See BeginRenderPassNoClear for non-clearing version.
func (rp *Render) BeginRenderPass(cmd vk.CommandBuffer, fr *Framebuffer) {
	rp.BeginRenderPassImpl(cmd, fr, true)
}

// BeginRenderPassNoClear adds commands to the given command buffer
// to start the render pass on given framebuffer.
// does NOT clear the frame first -- loads prior state.
func (rp *Render) BeginRenderPassNoClear(cmd vk.CommandBuffer, fr *Framebuffer) {
	rp.BeginRenderPassImpl(cmd, fr, false)
}

// BeginRenderPassImpl adds commands to the given command buffer
// to start the render pass on given framebuffer.
// If clear = true, clears the frame according to the ClearVals.
func (rp *Render) BeginRenderPassImpl(cmd vk.CommandBuffer, fr *Framebuffer, clear bool) {
	w, h := fr.Image.Format.Size32()
	clearVals := rp.ClearVals
	vrp := rp.VkClearPass
	if !clear && fr.HasCleared {
		clearVals = nil
		vrp = rp.VkLoadPass
	}
	fr.HasCleared = true
	vk.CmdBeginRenderPass(cmd, &vk.RenderPassBeginInfo{
		SType:       vk.StructureTypeRenderPassBeginInfo,
		RenderPass:  vrp,
		Framebuffer: fr.Framebuffer,
		RenderArea: vk.Rect2D{
			Offset: vk.Offset2D{X: 0, Y: 0},
			Extent: vk.Extent2D{Width: w, Height: h},
		},
		ClearValueCount: uint32(len(clearVals)),
		PClearValues:    clearVals,
	}, vk.SubpassContentsInline)

	vk.CmdSetViewport(cmd, 0, 1, []vk.Viewport{{
		Width:    float32(w),
		Height:   float32(h),
		MinDepth: 0.0,
		MaxDepth: 1.0,
	}})

	vk.CmdSetScissor(cmd, 0, 1, []vk.Rect2D{{
		Offset: vk.Offset2D{X: 0, Y: 0},
		Extent: vk.Extent2D{Width: w, Height: h},
	}})
}

// ConfigGrab configures the Grab for copying rendered image
// back to host memory.  Uses format of current Image.
func (rp *Render) ConfigGrab(dev vk.Device) {
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
	rp.Grab.SetFlag(int(ImageOnHostOnly))
	rp.Grab.Dev = dev
	rp.Grab.GPU = rp.Sys.GPU
	rp.Grab.AllocImage()
}

// https://www.reddit.com/r/vulkan/comments/7yhvep/retrieve_depth_attachment_from_framebuffer/
// https://pastebin.com/33MxSNmh
// have to copy to a buffer then sync that back down
// create the buffer as the GrabDepth item

// ConfigGrabDepth configures the GrabDepth for copying depth image
// back to host memory.  Uses format of current Depth image.
func (rp *Render) ConfigGrabDepth(dev vk.Device) {
	bsz := rp.Format.Size.X * rp.Format.Size.Y * 4 // 32 bit = 4 bytes per pixel
	if rp.GrabDepth.Active {
		if rp.GrabDepth.Size == bsz {
			return
		}
		rp.GrabDepth.Free(dev)
	}
	rp.GrabDepth.GPU = rp.Sys.GPU
	rp.GrabDepth.Type = StorageBuff
	rp.GrabDepth.AllocHost(dev, bsz)
}

// GrabDepthImage grabs the current render depth image, using given command buffer
// which must have the cmdBegin called already.  Uses the GrabDepth Storage Buffer.
// call this after: sys.MemCmdEndSubmitWaitFree()
func (rp *Render) GrabDepthImage(dev vk.Device, cmd vk.CommandBuffer) error {
	nsamp := rp.Format.NSamples()
	if nsamp > 1 {
		err := errors.New("vgpu.Render.GrabDepthImage(): does not work if multisampling is > 1")
		if Debug {
			log.Println(err)
		}
		return err
	}
	rp.ConfigGrabDepth(dev) // ensure image grab setup
	// first, prepare ImageGrab to receive copy from render image.
	// apparently, the color attachment, with src flag already set, does not need this.

	reg := vk.BufferImageCopy{BufferOffset: 0, BufferRowLength: 0, BufferImageHeight: 0}
	reg.ImageSubresource.AspectMask = vk.ImageAspectFlags(vk.ImageAspectDepthBit)
	reg.ImageSubresource.MipLevel = 0
	reg.ImageSubresource.BaseArrayLayer = 0
	reg.ImageSubresource.LayerCount = 1
	reg.ImageOffset.X, reg.ImageOffset.Y, reg.ImageOffset.Z = 0, 0, 0
	reg.ImageExtent.Width = uint32(rp.Format.Size.X)
	reg.ImageExtent.Height = uint32(rp.Format.Size.Y)
	reg.ImageExtent.Depth = 1

	vk.CmdCopyImageToBuffer(cmd, rp.Depth.Image, vk.ImageLayoutTransferSrcOptimal, rp.GrabDepth.Host, 1, []vk.BufferImageCopy{reg})
	return nil
}

// DepthImageArray returns the float values from the last GrabDepthImage call
// automatically handles down-sampling from multisampling.
func (rp *Render) DepthImageArray() ([]float32, error) {
	if rp.GrabDepth.Host == vk.NullBuffer {
		err := errors.New("DepthImageArray: No GrabDepth.Host buffer -- must call GrabDepthImage")
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
	// note: you cannot specify a greater width than actual width
	// and resolving depth images GPU-side is not exactly clear:
	// https://community.khronos.org/t/how-to-resolve-multi-sampled-depth-images/7584
	// https://www.reddit.com/r/vulkan/comments/rpeywp/is_it_possible_to_resolve_a_depth_msaa_buffer/
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
	return ary, nil
}
