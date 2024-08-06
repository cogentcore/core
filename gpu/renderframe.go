// Copyright (c) 2022, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gpu

import (
	"image"

	"github.com/cogentcore/webgpu/wgpu"
)

// RenderFrame is an offscreen, non-window-backed rendering target,
// functioning like a Surface
type RenderFrame struct {

	// pointer to gpu device, for convenience
	GPU *GPU

	// device for this surface -- each window surface has its own device, configured for that surface
	Device Device

	// the Render for this RenderFrame, typically from a System
	Render *Render

	// has the current image format and dimensions
	Format TextureFormat

	// number of frames to maintain in the swapchain -- e.g., 2 = double-buffering, 3 = triple-buffering -- initially set to a requested amount, and after Init reflects actual number
	NFrames int

	// Framebuffers representing the Texture owned by the RenderFrame -- we iterate through these in rendering subsequent frames
	Frames []*Framebuffer

	// semaphore used internally for waiting on acquisition of next frame
	// TextureAcquired vk.Semaphore `display:"-"`
	//
	// // semaphore that surface user can wait on, will be activated when image has been acquired in AcquireNextFrame method
	// RenderDone vk.Semaphore `display:"-"`

	// fence for rendering command running
	// RenderFence vk.Fence `display:"-"`

	// do we own the device?
	OwnDevice bool
}

// NewRenderFrameOwnDevice returns a new renderframe initialized for given GPU,
// of given size.
// This version creates a new Graphics device -- for purely offscreen usage.
func NewRenderFrameOwnDevice(gp *GPU, size image.Point) *RenderFrame {
	rf := &RenderFrame{}
	rf.Defaults()
	rf.Format.Size = size
	rf.Init(gp, true) // make own device
	return rf
}

// NewRenderFrame returns a new renderframe initialized for given GPU,
// of given size.
// using given device, e.g., from a Surface -- to transition images
// from renderframe to surface, they must use the same device.
// if device is nil, own device is created.
func NewRenderFrame(gp *GPU, dev *Device, size image.Point) *RenderFrame {
	rf := &RenderFrame{}
	rf.Defaults()
	rf.Format.Size = size
	if dev != nil {
		rf.Device = *dev
		rf.Init(gp, false)
	} else {
		rf.Init(gp, true) // make dev
	}
	return rf
}

func (rf *RenderFrame) Defaults() {
	rf.NFrames = 1
	rf.Format.Defaults()
	// note: screen-correct results obtained by using Srgb here, which forces
	// this format in the final output.  Looks like what comes out from direct rendering.
	rf.Format.Set(1024, 768, wgpu.TextureFormatRGBA8UnormSrgb)
	// rf.Format.Set(1024, 768, wgpu.TextureFormatR8g8b8a8Unorm)
	rf.Format.SetMultisample(4)
}

// Init initializes the device and all other resources for the renderframe.
func (rf *RenderFrame) Init(gp *GPU, makeDevice bool) error {
	rf.GPU = gp
	if makeDevice {
		rf.OwnDevice = true
		// rf.Device.Init(gp, vk.QueueGraphicsBit)
	} else {
		rf.OwnDevice = false
	}
	rf.Config()
	return nil
}

// Config configures the framebuffers etc
func (rf *RenderFrame) Config() {
	// dev := rf.Device.Device
	// rf.TextureAcquired = NewSemaphore(dev)
	// rf.RenderDone = NewSemaphore(dev)
	// rf.RenderFence = NewFence(dev)
	//
	// rf.Frames = make([]*Framebuffer, rf.NFrames)
	// for i := 0; i < rf.NFrames; i++ {
	// 	fr := &Framebuffer{}
	// 	fr.ConfigRenderTexture(rf.GPU, dev, rf.Format)
	// 	rf.Frames[i] = fr
	// }
}

// Free frees any existing (for ReInit or Release)
func (rf *RenderFrame) Free() {
	// dev := rf.Device.Device
	// vk.DeviceWaitIdle(dev)
	// vk.ReleaseSemaphore(dev, rf.TextureAcquired, nil)
	// vk.ReleaseSemaphore(dev, rf.RenderDone, nil)
	// vk.ReleaseFence(dev, rf.RenderFence, nil)
	// for _, fr := range rf.Frames {
	// 	fr.Release()
	// }
	// rf.Frames = nil
}

// SetRender sets the Render and updates frames accordingly
func (rf *RenderFrame) SetRender(rp *Render) {
	rf.Render = rp
	for _, fr := range rf.Frames {
		fr.ConfigRender(rp)
	}
}

// SetSize sets the size for the render frame,
// doesn't do anything if already that size (returns fale)
func (rf *RenderFrame) SetSize(size image.Point) bool {
	if rf.Format.Size == size {
		return false
	}
	rf.Format.Size = size
	rf.ReConfig()
	return true
}

// ReConfig reconfigures rendering
func (rf *RenderFrame) ReConfig() {
	rf.Render.SetSize(rf.Format.Size)
	rf.ReConfigFrames()
}

// ReConfigFrames re-configures the Famebuffers
// using exiting settings.
// Assumes Config has been called.
func (rf *RenderFrame) ReConfigFrames() {
	for _, fr := range rf.Frames {
		fr.ConfigRenderTexture(rf.GPU, &rf.Device, rf.Format)
		fr.ConfigRender(rf.Render)
	}
}

func (rf *RenderFrame) Release() {
	rf.Free()
	if rf.OwnDevice {
		// rf.Device.Release()
	}
	rf.GPU = nil
}

// SubmitRender submits a rendering command that must have been added
// to the given command buffer, calling CmdEnd on the buffer first.
// This buffer triggers the associated Fence logic to control the
// sequencing of render commands over time.
// The TextureAcquired semaphore before the command is run.
func (rf *RenderFrame) SubmitRender(cmd *wgpu.CommandEncoder) {
	// dev := rf.Device.Device
	// vk.ResetFences(dev, 1, []vk.Fence{rf.RenderFence})
	// CmdEnd(cmd)
	// ret := vk.QueueSubmit(rf.Device.Queue, 1, []vk.SubmitInfo{{
	// 	SType: vk.StructureTypeSubmitInfo,
	// 	PWaitDstStageMask: []vk.PipelineStageFlags{
	// 		vk.PipelineStageFlags(vk.PipelineStageColorAttachmentOutputBit),
	// 	},
	// 	// WaitSemaphoreCount:   1,
	// 	// PWaitSemaphores:      []vk.Semaphore{rf.TextureAcquired},
	// 	CommandBufferCount: 1,
	// 	PCommandBuffers:    []*wgpu.CommandEncoder{cmd},
	// 	// SignalSemaphoreCount: 1,
	// 	// PSignalSemaphores:    []vk.Semaphore{rf.RenderDone},
	// }}, rf.RenderFence)
	// IfPanic(NewError(ret))
}

// WaitForRender waits until the last submitted render completes
func (rf *RenderFrame) WaitForRender() {
	// dev := rf.Device.Device
	// vk.WaitForFences(dev, 1, []vk.Fence{rf.RenderFence}, vk.True, vk.MaxUint64)
	// vk.ResetFences(dev, 1, []vk.Fence{rf.RenderFence})
}

// GrabTexture grabs rendered image of given index to Framebuffer.TextureGrab.
// must have waited for render already.
func (rf *RenderFrame) GrabTexture(cmd *wgpu.CommandEncoder, idx int) {
	rf.Frames[idx].GrabTexture(&rf.Device, cmd)
}

// GrabDepthTexture grabs rendered depth image from the Render,
// must have waited for render already.
func (rf *RenderFrame) GrabDepthTexture(cmd *wgpu.CommandEncoder) {
	rf.Render.GrabDepthTexture(&rf.Device, cmd)
}
