// Copyright (c) 2022, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package vgpu

import (
	vk "github.com/goki/vulkan"
)

// Framebuffer combines an Image and Render info (which has a depth buffer)
type Framebuffer struct {
	Format      ImageFormat    `desc:"target framebuffer format -- if multisampling is active then Image has samples = 1, Render.Multi has full samples"`
	Image       Image          `desc:"the image behind the framebuffer, includes the format -- this "`
	Render      *Render        `desc:"pointer to the associated renderpass and depth buffer"`
	Framebuffer vk.Framebuffer `desc:"vulkan framebuffer"`
	HasCleared  bool           `desc:"has this framebuffer been cleared yet?  if not, must be prior to use as a non-clearing Load case"`
}

// ConfigSurfaceImage configures settings for given existing surface image
// and format.  Does not yet make the Framebuffer because it
// still needs the Render (see ConfigAll for all)
func (fb *Framebuffer) ConfigSurfaceImage(gp *GPU, dev vk.Device, fmt ImageFormat, img vk.Image) {
	fb.Format = fmt
	fb.Image.Format.Defaults()
	fb.Image.Format = fmt
	fb.Image.Format.SetMultisample(1) // cannot multisample main image
	fb.Image.SetFlag(int(FramebufferImage))
	fb.Image.SetVkImage(gp, dev, img) // makes view
}

// ConfigRenderImage configures a new image for a standalone framebuffer
// not associated with an existing surface, for RenderFrame target.
// In general it is recommended to use vk.SampleCount4Bit to avoid aliasing.
// Does not yet make the Framebuffer because it still needs the Render
// (see ConfigRender)
func (fb *Framebuffer) ConfigRenderImage(gp *GPU, dev vk.Device, fmt ImageFormat) {
	fb.Format = fmt
	fb.Image.ConfigFramebuffer(gp, dev, &fmt)
}

// ConfigRender configures for Render, assuming image is already set
// and Configs the Framebuffer based on that.
func (fb *Framebuffer) ConfigRender(rp *Render) {
	fb.Render = rp
	if fb.Image.Dev != rp.Dev { // device must be same as renderpass
		panic("vgpu.Framebuffer:ConfigRender -- image and renderpass have different devices -- this will not work -- e.g., must set Surface to use System's device or vice-versa")
	}
	fb.Config()
}

// Destroy destroys everything
func (fb *Framebuffer) Destroy() {
	fb.DestroyFrame()
	fb.Image.Destroy()
	fb.Render = nil
}

// DestroyFrame destroys the framebuffer if non-nil
func (fb *Framebuffer) DestroyFrame() {
	if fb.Render == nil || fb.Render.Dev == nil || fb.Framebuffer == nil {
		return
	}
	vk.DestroyFramebuffer(fb.Render.Dev, fb.Framebuffer, nil)
	fb.Framebuffer = nil
}

// Config configures a new vulkan framebuffer object with current settings,
// destroying any existing
func (fb *Framebuffer) Config() {
	fb.DestroyFrame()
	ivs := []vk.ImageView{}
	if fb.Render.HasMulti {
		ivs = append(ivs, fb.Render.Multi.View)
	} else {
		ivs = append(ivs, fb.Image.View)
	}
	if fb.Render.HasDepth {
		ivs = append(ivs, fb.Render.Depth.View)
	}
	if fb.Render.HasMulti {
		ivs = append(ivs, fb.Image.View)
	}
	w, h := fb.Image.Format.Size32()
	var frameBuff vk.Framebuffer
	ret := vk.CreateFramebuffer(fb.Render.Dev, &vk.FramebufferCreateInfo{
		SType:           vk.StructureTypeFramebufferCreateInfo,
		RenderPass:      fb.Render.VkClearPass,
		AttachmentCount: uint32(len(ivs)),
		PAttachments:    ivs,
		Width:           w,
		Height:          h,
		Layers:          1,
	}, nil, &frameBuff)
	IfPanic(NewError(ret))
	fb.Framebuffer = frameBuff
	fb.HasCleared = false
}

/////////////////////////////////////////////////////////////////
// RenderFrame functionality

// https://stackoverflow.com/questions/51477954/vulkan-off-screen-rendering-tiling-optimal-or-linear

// https://github.com/SaschaWillems/Vulkan/tree/b9f0ac91d2adccc3055a904d3a8f6553b10ff6cd/examples/renderheadless/renderheadless.cpp

// GrabImage grabs the current framebuffer image, using given command buffer
// which must have the cmdBegin called already.
// call this after: sys.MemCmdEndSubmitWaitFree()
func (fb *Framebuffer) GrabImage(dev vk.Device, cmd vk.CommandBuffer) {
	fb.Render.ConfigGrab(dev) // ensure image grab setup
	// first, prepare ImageGrab to receive copy from render image.
	// apparently, the color attachment, with src flag already set, does not need this.

	// todo: for Surface frame, transition src image

	fb.Render.Grab.TransitionForDst(cmd, vk.PipelineStageTransferBit) // no idea why, but SaschaWillems does
	vk.CmdCopyImage(cmd, fb.Image.Image, vk.ImageLayoutTransferSrcOptimal, fb.Render.Grab.Image, vk.ImageLayoutTransferDstOptimal, 1, []vk.ImageCopy{fb.Render.Grab.CopyImageRec()})
	fb.Render.Grab.TransitionDstToGeneral(cmd)
}

// CopyToImage copies the current framebuffer image to given image dest
// using given command buffer which must have the cmdBegin called already.
func (fb *Framebuffer) CopyToImage(toImg *Image, dev vk.Device, cmd vk.CommandBuffer) {
	toImg.TransitionForDst(cmd, vk.PipelineStageTransferBit) // no idea why, but SaschaWillems does
	vk.CmdCopyImage(cmd, fb.Image.Image, vk.ImageLayoutTransferSrcOptimal, toImg.Image, vk.ImageLayoutTransferDstOptimal, 1, []vk.ImageCopy{toImg.CopyImageRec()})
}
