// Copyright (c) 2022, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package vgpu

import (
	"image"

	vk "github.com/vulkan-go/vulkan"
)

// Framebuffer combines an Image and RenderPass info (which has a depth buffer)
type Framebuffer struct {
	Image       Image          `desc:"the image behind the framebuffer, includes the format"`
	RenderPass  *RenderPass    `desc:"pointer to the associated renderpass and depth buffer"`
	Framebuffer vk.Framebuffer `desc:"vulkan framebuffer"`
}

// ConfigImage configures settings for given existing image format
// and image.  Does not yet make the Framebuffer because it
// still needs the RenderPass (see ConfigAll for all)
func (fb *Framebuffer) ConfigImage(dev vk.Device, fmt ImageFormat, img vk.Image) {
	fb.Image.Format.Defaults()
	fb.Image.Format = fmt
	fb.Image.SetVkImage(dev, img) // makes view
	fb.Image.SetFlag(int(FramebufferImage))
}

// ConfigAll configures settings for given existing image format
// image, and RenderPass, and Makes the Framebuffer based on that.
func (fb *Framebuffer) ConfigAll(dev vk.Device, fmt ImageFormat, img vk.Image, rp *RenderPass) {
	fb.ConfigImage(dev, fmt, img)
	fb.RenderPass = rp
	fb.Config()
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

// ConfigNewImage configures a new image for a standalone framebuffer
// not associated with an existing surface, to be used as a rendering target.
// In general it is recommended to use vk.SampleCount4Bit to avoid aliasing.
// Does not yet make the Framebuffer because it still needs the RenderPass
// (see ConfigRenderPass)
func (fb *Framebuffer) ConfigNewImage(dev vk.Device, fmt ImageFormat, size image.Point, samples vk.SampleCountFlagBits) {
	fb.Image.Format.Defaults()
	fb.Image.Format = fmt
	fb.Image.Format.Size = size
	fb.Image.Format.Samples = samples
	fb.Image.SetFlag(int(FramebufferImage))
	fb.Image.Dev = dev
	fb.Image.AllocImage()
}

// Destroy destroys everything
func (fb *Framebuffer) Destroy() {
	fb.DestroyFrame()
	fb.Image.Destroy()
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
	if fb.RenderPass.Depth.IsActive() {
		ivs = append(ivs, fb.RenderPass.Depth.View)
	}
	w, h := fb.Image.Format.Size32()
	var frameBuff vk.Framebuffer
	ret := vk.CreateFramebuffer(fb.RenderPass.Dev, &vk.FramebufferCreateInfo{
		SType:           vk.StructureTypeFramebufferCreateInfo,
		RenderPass:      fb.RenderPass.RenderPass,
		AttachmentCount: uint32(len(ivs)),
		PAttachments:    ivs,
		Width:           w,
		Height:          h,
		Layers:          1,
	}, nil, &frameBuff)
	IfPanic(NewError(ret))
	fb.Framebuffer = frameBuff
}

// SetSize re-allocates an backing framebuffer Image of given size.
// This should be used for standalone framebuffers, not Surface framebuffers
// that get their Image from the Swapchain.
// If the RenderPass is set, then it re-sizes any corresponding Depth buffer
// and re-makes the framebuffer.
func (fb *Framebuffer) SetSize(size image.Point) {
	fb.Image.SetSize(size)
	if fb.RenderPass.Depth.IsActive() {
		fb.RenderPass.Depth.SetSize(size)
	}
	fb.Config()
}
