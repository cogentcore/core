// Copyright (c) 2022, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gpu

import (
	"github.com/cogentcore/webgpu/wgpu"
)

// Framebuffer combines an Texture and Render info (which has a depth buffer)
type Framebuffer struct {

	// target framebuffer format -- if multisampling is active then Texture has samples = 1, Render.Multi has full samples
	Format TextureFormat

	// the image behind the framebuffer, includes the format -- this
	Texture Texture

	// pointer to the associated renderpass and depth buffer
	Render *Render

	// WebGPU framebuffer
	// Framebuffer vk.Framebuffer

	// has this framebuffer been cleared yet?  if not, must be prior to use as a non-clearing Load case
	HasCleared bool
}

// ConfigSurfaceTexture configures settings for given existing surface image
// and format.  Does not yet make the Framebuffer because it
// still needs the Render (see ConfigAll for all)
func (fb *Framebuffer) ConfigSurfaceTexture(gp *GPU, dev *Device, fmt TextureFormat, img *wgpu.Texture) {
	// fb.Format = fmt
	// fb.Texture.Format.Defaults()
	// fb.Texture.Format = fmt
	// fb.Texture.Format.SetMultisample(1) // cannot multisample main image
	// fb.Texture.SetFlag(true, FramebufferTexture)
	// fb.Texture.SetVkTexture(gp, dev, img) // makes view
}

// ConfigRenderTexture configures a new image for a standalone framebuffer
// not associated with an existing surface, for RenderFrame target.
// In general it is recommended to use vk.SampleCount4Bit to avoid aliasing.
// Does not yet make the Framebuffer because it still needs the Render
// (see ConfigRender)
func (fb *Framebuffer) ConfigRenderTexture(gp *GPU, dev *Device, fmt TextureFormat) {
	fb.Format = fmt
	fb.Texture.ConfigFramebuffer(dev, &fmt)
}

// ConfigRender configures for Render, assuming image is already set
// and Configs the Framebuffer based on that.
func (fb *Framebuffer) ConfigRender(rp *Render) {
	fb.Render = rp
	if fb.Texture.device.Device != rp.device.Device { // device must be same as renderpass
		panic("gpu.Framebuffer:ConfigRender -- image and renderpass have different devices -- this will not work -- e.g., must set Surface to use System's device or vice-versa")
	}
	fb.Config()
}

// Release destroys everything
func (fb *Framebuffer) Release() {
	fb.ReleaseFrame()
	fb.Texture.Release()
	fb.Render = nil
}

// ReleaseFrame destroys the framebuffer if non-nil
func (fb *Framebuffer) ReleaseFrame() {
	// if fb.Render == nil || fb.Render.Dev == nil || fb.Framebuffer == vk.NullFramebuffer {
	// 	return
	// }
	// vk.ReleaseFramebuffer(fb.Render.Dev, fb.Framebuffer, nil)
	// fb.Framebuffer = vk.NullFramebuffer
}

// Config configures a new WebGPU framebuffer object with current settings,
// destroying any existing
func (fb *Framebuffer) Config() {
	fb.ReleaseFrame()
	// ivs := []vk.TextureView{}
	// if fb.Render.HasMulti {
	// 	ivs = append(ivs, fb.Render.Multi.View)
	// } else {
	// 	ivs = append(ivs, fb.Texture.View)
	// }
	// if fb.Render.HasDepth {
	// 	ivs = append(ivs, fb.Render.Depth.View)
	// }
	// if fb.Render.HasMulti {
	// 	ivs = append(ivs, fb.Texture.View)
	// }
	// w, h := fb.Texture.Format.Size32()
	// var frameBuff vk.Framebuffer
	// ret := vk.CreateFramebuffer(fb.Render.Dev, &vk.FramebufferCreateInfo{
	// 	SType:           vk.StructureTypeFramebufferCreateInfo,
	// 	RenderPass:      fb.Render.VkClearPass,
	// 	AttachmentCount: uint32(len(ivs)),
	// 	PAttachments:    ivs,
	// 	Width:           w,
	// 	Height:          h,
	// 	Layers:          1,
	// }, nil, &frameBuff)
	// IfPanic(NewError(ret))
	// fb.Framebuffer = frameBuff
	// fb.HasCleared = false
}

/////////////////////////////////////////////////////////////////
// RenderFrame functionality

// https://stackoverflow.com/questions/51477954/WebGPU-off-screen-rendering-tiling-optimal-or-linear

// https://github.com/SaschaWillems/Vulkan/tree/b9f0ac91d2adccc3055a904d3a8f6553b10ff6cd/examples/renderheadless/renderheadless.cpp

// GrabTexture grabs the current framebuffer image, using given command buffer
// which must have the cmdBegin called already.
// call this after: sys.MemCmdEndSubmitWaitFree()
func (fb *Framebuffer) GrabTexture(dev *Device, cmd *wgpu.CommandEncoder) {
	fb.Render.ConfigGrab(dev) // ensure image grab setup
	// first, prepare TextureGrab to receive copy from render image.
	// apparently, the color attachment, with src flag already set, does not need this.

	// todo: for Surface frame, transition src image

	// fb.Render.Grab.TransitionForDst(cmd, vk.PipelineStageTransferBit) // no idea why, but SaschaWillems does
	// vk.CmdCopyTexture(cmd, fb.Texture.Texture, vk.TextureLayoutTransferSrcOptimal, fb.Render.Grab.Texture, vk.TextureLayoutTransferDstOptimal, 1, []vk.TextureCopy{fb.Render.Grab.CopyTextureRec()})
	// fb.Render.Grab.TransitionDstToGeneral(cmd)
}

// CopyToTexture copies the current framebuffer image to given image dest
// using given command buffer which must have the cmdBegin called already.
// func (fb *Framebuffer) CopyToTexture(toImg *Texture, dev *Device, cmd *wgpu.CommandEncoder) {
// 	toImg.TransitionForDst(cmd, vk.PipelineStageTransferBit) // no idea why, but SaschaWillems does
// 	vk.CmdCopyTexture(cmd, fb.Texture.Texture, vk.TextureLayoutTransferSrcOptimal, toImg.Texture, vk.TextureLayoutTransferDstOptimal, 1, []vk.TextureCopy{toImg.CopyTextureRec()})
// }
