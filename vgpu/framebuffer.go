// Copyright (c) 2022, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package vgpu

import (
	"image"

	vk "github.com/goki/vulkan"
)

// Framebuffer combines an Image and RenderPass info (which has a depth buffer)
type Framebuffer struct {
	Image       Image          `desc:"the image behind the framebuffer, includes the format"`
	RenderPass  *RenderPass    `desc:"pointer to the associated renderpass and depth buffer"`
	Framebuffer vk.Framebuffer `desc:"vulkan framebuffer"`
	HasCleared  bool           `desc:"has this framebuffer been cleared yet?  if not, must be prior to use as a non-clearing Load case"`
	ImageGrab   Image          `desc:"for a RenderFrame, this is the image that is used to transfer back from the render color attachment to host memory -- requires a different format than color attachment, and is ImageOnHostOnly flagged."`
}

// ConfigSurfaceImage configures settings for given existing surface image
// and format.  Does not yet make the Framebuffer because it
// still needs the RenderPass (see ConfigAll for all)
func (fb *Framebuffer) ConfigSurfaceImage(dev vk.Device, fmt ImageFormat, img vk.Image) {
	fb.Image.Format.Defaults()
	fb.Image.Format = fmt
	fb.Image.SetVkImage(dev, img) // makes view
	fb.Image.SetFlag(int(FramebufferImage))
}

// ConfigRenderImage configures a new image for a standalone framebuffer
// not associated with an existing surface, for RenderFrame target.
// In general it is recommended to use vk.SampleCount4Bit to avoid aliasing.
// Does not yet make the Framebuffer because it still needs the RenderPass
// (see ConfigRenderPass)
func (fb *Framebuffer) ConfigRenderImage(dev vk.Device, fmt ImageFormat) {
	fb.Image.Format.Defaults()
	fb.Image.Format = fmt
	fb.Image.SetFlag(int(FramebufferImage))
	fb.Image.Dev = dev
	fb.Image.AllocImage()
	fb.Image.ConfigStdView()
	fb.ConfigImageGrab(dev)
}

// ConfigImageGrab configures the ImageGrab for copying rendered image
// back to host memory.  Uses format of current Image.
func (fb *Framebuffer) ConfigImageGrab(dev vk.Device) {
	if fb.ImageGrab.IsActive() {
		if fb.ImageGrab.Format.Size == fb.Image.Format.Size {
			return
		}
		fb.ImageGrab.SetSize(fb.Image.Format.Size)
		return
	}
	fb.ImageGrab.Format.Defaults()
	fb.ImageGrab.Format = fb.Image.Format
	fb.ImageGrab.SetFlag(int(ImageOnHostOnly))
	fb.ImageGrab.Dev = dev
	fb.ImageGrab.AllocImage()
}

// ConfigRenderPass configures for RenderPass, assuming image is already set
// and Configs the Framebuffer based on that.
func (fb *Framebuffer) ConfigRenderPass(rp *RenderPass) {
	fb.RenderPass = rp
	if fb.Image.Dev != rp.Dev { // device must be same as renderpass
		panic("vgpu.Framebuffer:ConfigRenderPass -- image and renderpass have different devices -- this will not work -- e.g., must set Surface to use System's device or vice-versa")
	}
	fb.Config()
}

// Destroy destroys everything
func (fb *Framebuffer) Destroy() {
	fb.DestroyFrame()
	fb.Image.Destroy()
	if fb.ImageGrab.Image != nil {
		fb.ImageGrab.Destroy()
	}
	fb.RenderPass = nil
}

// DestroyFrame destroys the framebuffer if non-nil
func (fb *Framebuffer) DestroyFrame() {
	if fb.RenderPass == nil || fb.RenderPass.Dev == nil || fb.Framebuffer == nil {
		return
	}
	vk.DestroyFramebuffer(fb.RenderPass.Dev, fb.Framebuffer, nil)
	fb.Framebuffer = nil
}

// Config configures a new vulkan framebuffer object with current settings,
// destroying any existing
func (fb *Framebuffer) Config() {
	fb.DestroyFrame()
	ivs := []vk.ImageView{fb.Image.View}
	if fb.RenderPass.HasDepth {
		ivs = append(ivs, fb.RenderPass.Depth.View)
	}
	w, h := fb.Image.Format.Size32()
	var frameBuff vk.Framebuffer
	ret := vk.CreateFramebuffer(fb.RenderPass.Dev, &vk.FramebufferCreateInfo{
		SType:           vk.StructureTypeFramebufferCreateInfo,
		RenderPass:      fb.RenderPass.VkClearPass,
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

// SetSize re-allocates an backing framebuffer Image of given size.
// This should be used for standalone framebuffers, not Surface framebuffers
// that get their Image from the Swapchain.
// If the RenderPass is set, then it re-sizes any corresponding Depth buffer
// and re-makes the framebuffer.
func (fb *Framebuffer) SetSize(size image.Point) {
	fb.Image.SetSize(size)
	fb.ImageGrab.SetSize(size)
	if fb.RenderPass.Depth.IsActive() {
		fb.RenderPass.Depth.SetSize(size)
	}
	fb.Config()
}

/////////////////////////////////////////////////////////////////
// RenderFrame functionality

// https://stackoverflow.com/questions/51477954/vulkan-off-screen-rendering-tiling-optimal-or-linear

// https://github.com/SaschaWillems/Vulkan/tree/b9f0ac91d2adccc3055a904d3a8f6553b10ff6cd/examples/renderheadless/renderheadless.cpp

// GrabImage grabs the current framebuffer image, using given command buffer
// which must have the cmdBegin called already.
func (fb *Framebuffer) GrabImage(dev vk.Device, cmd vk.CommandBuffer) {
	fb.ConfigImageGrab(dev) // ensure image grab setup
	// first, prepare ImageGrab to receive copy from render image.
	// apparently, the color attachment, with src flag already set, does not need this.

	// todo: for Surface frame, transition src image

	fb.ImageGrab.TransitionForDst(cmd, vk.PipelineStageTransferBit) // no idea why, but SaschaWillems does
	vk.CmdCopyImage(cmd, fb.Image.Image, vk.ImageLayoutTransferSrcOptimal, fb.ImageGrab.Image, vk.ImageLayoutTransferDstOptimal, 1, []vk.ImageCopy{fb.ImageGrab.CopyImageRec()})
	fb.ImageGrab.TransitionDstToGeneral(cmd)
}

// CopyToImage copies the current framebuffer image to given image dest
// using given command buffer which must have the cmdBegin called already.
func (fb *Framebuffer) CopyToImage(toImg *Image, dev vk.Device, cmd vk.CommandBuffer) {
	toImg.TransitionForDst(cmd, vk.PipelineStageTransferBit) // no idea why, but SaschaWillems does
	vk.CmdCopyImage(cmd, fb.Image.Image, vk.ImageLayoutTransferSrcOptimal, toImg.Image, vk.ImageLayoutTransferDstOptimal, 1, []vk.ImageCopy{toImg.CopyImageRec()})
}
