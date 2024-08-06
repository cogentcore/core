// Copyright (c) 2022, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gpu

import (
	"image"

	"github.com/rajveermalviya/go-webgpu/wgpu"
)

// RenderTexture is an offscreen, non-window-backed rendering target,
// functioning like a Surface.
type RenderTexture struct {
	// pointer to gpu device, for convenience
	GPU *GPU

	// device for this Frame.  we do NOT own this device.
	Device Device

	// the Render for this RenderTexture, typically from a System.
	Render *Render

	// Format has the current image format and dimensions.
	// The Samples here are the desired value, whereas our Frames
	// always have Samples = 1.
	Format TextureFormat

	// number of frames to maintain in the simulated swapchain.
	// e.g., 2 = double-buffering, 3 = triple-buffering.
	NFrames int

	// Textures that we iterate through in rendering subsequent frames.
	Frames []*Texture
}

// NewRenderTexture returns a new rendertarget for given GPU, device,
// of given size.  If using in conjunction with a Surface, the device
// must be from that surface so frames can be transitioned there.
// If doing pure offscreen rendering, then make a new device and Release
// it when the RenderTexture is released.
// samples is the multisampling anti-aliasing parameter: 1 = none
// 4 = typical default value for smooth "no jaggy" edges.
func NewRenderTexture(gp *GPU, dev *Device, size image.Point, samples int) *RenderTexture {
	rt := &RenderTexture{}
	rt.Defaults()
	rt.init(gp, dev, size, samples)
	return rt
}

func (rt *RenderTexture) Defaults() {
	rt.NFrames = 1
	rt.Format.Defaults()
	// note: screen-correct results obtained by using Srgb here, which forces
	// this format in the final output.  Looks like what comes out from direct rendering.
	rt.Format.Set(1024, 768, wgpu.TextureFormatRGBA8UnormSrgb)
	rt.Format.SetMultisample(4)
}

func (rt *RenderTexture) init(gp *GPU, dev *Device, size image.Point, samples int) {
	rt.GPU = gp
	rt.Device = *dev
	rt.Format.Size = size
	rt.Format.SetMultisample(samples)
}

// SetRender sets the Render and configures the frames accordingly.
func (rt *RenderTexture) SetRender(rp *Render) {
	rt.Release()
	rt.Render = rp
	rt.Frames = make([]*Texture, rt.NFrames)
	for i := range rt.NFrames {
		fr := NewTexture(&rt.Device)
		fr.ConfigRenderTexture(dev, rt.Format)
		rt.Frames[i] = fr
	}
}

// SetSize sets the size for the render frame,
// doesn't do anything if already that size (returns fale)
func (rt *RenderTexture) SetSize(size image.Point) bool {
	if rt.Format.Size == size {
		return false
	}
	rt.Format.Size = size
	rt.ReConfig()
	return true
}

// ReConfig reconfigures rendering
func (rt *RenderTexture) ReConfig() {
	rt.Render.SetSize(rt.Format.Size)
	rt.ReConfigFrames()
}

// ReConfigFrames re-configures the Famebuffers
// using exiting settings.
// Assumes Config has been called.
func (rt *RenderTexture) ReConfigFrames() {
	for _, fr := range rt.Frames {
		fr.ConfigRenderTexture(rt.GPU, &rt.Device, rt.Format)
		fr.ConfigRender(rt.Render)
	}
}

func (rt *RenderTexture) Release() {
	for _, fr := range rt.Frames {
		fr.Release()
	}
	rt.Frames = nil
}

// SubmitRender submits a rendering command that must have been added
// to the given command buffer, calling CmdEnd on the buffer first.
// This buffer triggers the associated Fence logic to control the
// sequencing of render commands over time.
// The TextureAcquired semaphore before the command is run.
func (rt *RenderTexture) SubmitRender(cmd *wgpu.CommandEncoder) {
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
func (rt *RenderTexture) WaitForRender() {
	// dev := rf.Device.Device
	// vk.WaitForFences(dev, 1, []vk.Fence{rf.RenderFence}, vk.True, vk.MaxUint64)
	// vk.ResetFences(dev, 1, []vk.Fence{rf.RenderFence})
}

// GrabTexture grabs rendered image of given index to RenderTexture.TextureGrab.
// must have waited for render already.
func (rt *RenderTexture) GrabTexture(cmd *wgpu.CommandEncoder, idx int) {
	rt.Frames[idx].GrabTexture(&rt.Device, cmd)
}

// GrabDepthTexture grabs rendered depth image from the Render,
// must have waited for render already.
func (rt *RenderTexture) GrabDepthTexture(cmd *wgpu.CommandEncoder) {
	rt.Render.GrabDepthTexture(&rt.Device, cmd)
}
