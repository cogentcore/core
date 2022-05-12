// Copyright (c) 2022, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// This is initially adapted from https://github.com/vulkan-go/asche
// Copyright © 2017 Maxim Kupriianov <max@kc.vc>, under the MIT License

package vgpu

import (
	"errors"
	"fmt"

	vk "github.com/goki/vulkan"
)

// Surface manages the physical device for the visible image
// of a window surface, and the swapchain for presenting images.
type Surface struct {
	GPU            *GPU           `desc:"pointer to gpu device, for convenience"`
	Device         Device         `desc:"device for this surface -- each window surface has its own device, configured for that surface"`
	RenderPass     *RenderPass    `desc:"the RenderPass for this Surface, typically from a System"`
	CmdPool        CmdPool        `desc:"command pool which must be used for all surface rendering commands, to enable the sync logic to work properly.  It is created in Init and can be Reset() between uses."`
	Format         ImageFormat    `desc:"has the current swapchain image format and dimensions"`
	DesiredFormats []vk.Format    `desc:"ordered list of surface formats to select"`
	NFrames        int            `desc:"number of frames to maintain in the swapchain -- e.g., 2 = double-buffering, 3 = triple-buffering -- initially set to a requested amount, and after Init reflects actual number"`
	Frames         []*Framebuffer `desc:"Framebuffers representing the visible Image owned by the Surface -- we iterate through these in rendering subsequent frames"`
	Surface        vk.Surface     `view:"-" desc:"vulkan handle for surface"`
	Swapchain      vk.Swapchain   `view:"-" desc:"vulkan handle for swapchain"`
	ImageAcquired  vk.Semaphore   `view:"-" desc:"semaphore used internally for waiting on acquisition of next frame"`
	RenderDone     vk.Semaphore   `view:"-" desc:"semaphore that surface user can wait on, will be activated when image has been acquired in AcquireNextFrame method"`
	RenderFence    vk.Fence       `view:"-" desc:"fence for rendering command running"`
}

// NewSurface returns a new surface initialized for given GPU and vulkan
// Surface handle, obtained from a valid window.
func NewSurface(gp *GPU, vsurf vk.Surface) *Surface {
	sf := &Surface{}
	sf.Defaults()
	sf.Init(gp, vsurf)
	return sf
}

func (sf *Surface) Defaults() {
	sf.NFrames = 2 // requested, will be updated with actual
	sf.Format.Defaults()
	sf.Format.Set(1024, 768, vk.FormatR8g8b8a8Srgb)
	sf.DesiredFormats = []vk.Format{
		vk.FormatR8g8b8a8Srgb,
		vk.FormatB8g8r8a8Srgb,
	}
}

// Init initializes the device and all other resources for the surface
// based on the vulkan surface handle which must be obtained from the
// OS-specific window, created first (e.g., via glfw)
func (sf *Surface) Init(gp *GPU, vs vk.Surface) error {
	sf.GPU = gp
	sf.Surface = vs
	// Get queue family properties
	var queueCount uint32
	vk.GetPhysicalDeviceQueueFamilyProperties(sf.GPU.GPU, &queueCount, nil)
	queueProperties := make([]vk.QueueFamilyProperties, queueCount)
	vk.GetPhysicalDeviceQueueFamilyProperties(sf.GPU.GPU, &queueCount, queueProperties)
	if queueCount == 0 { // probably should try another GPU
		return errors.New("vulkan error: no queue families found on GPU 0")
	}

	// Find a suitable queue family for the target Vulkan mode
	// note: this differs from generic Device.FindQueue in
	// specifying the surface.
	found := false
	for i := uint32(0); i < queueCount; i++ {
		var supportsPresent vk.Bool32
		vk.GetPhysicalDeviceSurfaceSupport(sf.GPU.GPU, i, sf.Surface, &supportsPresent)
		if supportsPresent.B() {
			sf.Device.QueueIndex = i
			found = true
			break
		}
	}
	if !found {
		err := errors.New("Surface vulkan error: could not found queue with present capabilities")
		return err
	}

	sf.Device.MakeDevice(gp)
	sf.CmdPool.ConfigResettable(&sf.Device)
	sf.CmdPool.NewBuffer(&sf.Device)
	sf.ConfigSwapchain()
	return nil
}

// ConfigSwapchain configures the swapchain for surface.
// This assumes that all existing items have been destroyed.
func (sf *Surface) ConfigSwapchain() {
	dev := sf.Device.Device

	// Read sf.Surface capabilities
	var surfaceCapabilities vk.SurfaceCapabilities
	ret := vk.GetPhysicalDeviceSurfaceCapabilities(sf.GPU.GPU, sf.Surface, &surfaceCapabilities)
	IfPanic(NewError(ret))
	surfaceCapabilities.Deref()

	// Get available surface pixel formats
	var formatCount uint32
	vk.GetPhysicalDeviceSurfaceFormats(sf.GPU.GPU, sf.Surface, &formatCount, nil)
	formats := make([]vk.SurfaceFormat, formatCount)
	vk.GetPhysicalDeviceSurfaceFormats(sf.GPU.GPU, sf.Surface, &formatCount, formats)

	// Select a proper surface format
	var format vk.SurfaceFormat
	if formatCount == 1 {
		formats[0].Deref()
		if formats[0].Format == vk.FormatUndefined {
			format = formats[0]
			format.Format = sf.Format.Format
		} else {
			format = formats[0]
		}
	} else if formatCount == 0 {
		IfPanic(errors.New("vulkan error: surface has no pixel formats"))
	} else {
		got := false
		for _, df := range sf.DesiredFormats {
			for _, ft := range formats {
				ft.Deref()
				if ft.Format == df {
					format = ft
					got = true
					break
				}
			}
			if got {
				break
			}
		}
		if !got {
			formats[0].Deref()
			format = formats[0]
			if sf.GPU.Debug {
				dfs := make([]string, len(sf.DesiredFormats))
				for i, df := range sf.DesiredFormats {
					dfs[i] = ImageFormatNames[df]
				}
				fmt.Printf("vgpu.Surface:Init unable to find desired format: %v, using first one: %s\n", dfs, ImageFormatNames[format.Format])
			}
		}
	}

	// Setup swapchain parameters
	var swapchainSize vk.Extent2D
	surfaceCapabilities.CurrentExtent.Deref()
	if surfaceCapabilities.CurrentExtent.Width == vk.MaxUint32 {
		w, h := sf.Format.Size32()
		swapchainSize.Width = w
		swapchainSize.Height = h
	} else {
		swapchainSize = surfaceCapabilities.CurrentExtent
	}

	// The FIFO present mode is guaranteed by the spec to be supported
	// and to have no tearing.  It's a great default present mode to use.
	swapchainPresentMode := vk.PresentModeFifo

	// Determine the number of VkImage's to use in the swapchain.
	// Ideally, we desire to own 1 image at a time, the rest of the images can either be rendered to and/or
	// being queued up for display.
	desiredSwapchainImages := uint32(sf.NFrames)
	if surfaceCapabilities.MaxImageCount > 0 && desiredSwapchainImages > surfaceCapabilities.MaxImageCount {
		// App must settle for fewer images than desired.
		desiredSwapchainImages = surfaceCapabilities.MaxImageCount
	}

	// Figure out a suitable surface transform.
	var preTransform vk.SurfaceTransformFlagBits
	requiredTransforms := vk.SurfaceTransformIdentityBit
	supportedTransforms := surfaceCapabilities.SupportedTransforms
	if vk.SurfaceTransformFlagBits(supportedTransforms)&requiredTransforms != 0 {
		preTransform = requiredTransforms
	} else {
		preTransform = surfaceCapabilities.CurrentTransform
	}

	// Find a supported composite alpha mode - one of these is guaranteed to be set
	compositeAlpha := vk.CompositeAlphaOpaqueBit
	compositeAlphaFlags := []vk.CompositeAlphaFlagBits{
		vk.CompositeAlphaOpaqueBit, // this only affects blending with other windows in OS
		vk.CompositeAlphaPreMultipliedBit,
		vk.CompositeAlphaPostMultipliedBit,
		vk.CompositeAlphaInheritBit,
	}
	for i := 0; i < len(compositeAlphaFlags); i++ {
		alphaFlags := vk.CompositeAlphaFlags(compositeAlphaFlags[i])
		flagSupported := surfaceCapabilities.SupportedCompositeAlpha&alphaFlags != 0
		if flagSupported {
			compositeAlpha = compositeAlphaFlags[i]
			break
		}
	}

	// Create a swapchain
	var swapchain vk.Swapchain
	oldSwapchain := sf.Swapchain
	swci := &vk.SwapchainCreateInfo{
		SType:           vk.StructureTypeSwapchainCreateInfo,
		Surface:         sf.Surface,
		MinImageCount:   desiredSwapchainImages,
		ImageFormat:     format.Format,
		ImageColorSpace: format.ColorSpace,
		ImageExtent: vk.Extent2D{
			Width:  swapchainSize.Width,
			Height: swapchainSize.Height,
		},
		ImageUsage:       vk.ImageUsageFlags(vk.ImageUsageColorAttachmentBit),
		PreTransform:     preTransform,
		CompositeAlpha:   compositeAlpha,
		ImageArrayLayers: 1,
		ImageSharingMode: vk.SharingModeExclusive,
		PresentMode:      swapchainPresentMode,
		OldSwapchain:     oldSwapchain,
		Clipped:          vk.True,
	}
	ret = vk.CreateSwapchain(dev, swci, nil, &swapchain)
	IfPanic(NewError(ret))
	if oldSwapchain != vk.NullSwapchain {
		vk.DestroySwapchain(dev, oldSwapchain, nil)
	}
	sf.Swapchain = swapchain
	sf.Format.Set(int(swapchainSize.Width), int(swapchainSize.Height), format.Format)

	var imageCount uint32
	ret = vk.GetSwapchainImages(dev, sf.Swapchain, &imageCount, nil)
	IfPanic(NewError(ret))
	sf.NFrames = int(imageCount)
	swapchainImages := make([]vk.Image, imageCount)
	ret = vk.GetSwapchainImages(dev, sf.Swapchain, &imageCount, swapchainImages)
	IfPanic(NewError(ret))

	sf.ImageAcquired = NewSemaphore(dev)
	sf.RenderDone = NewSemaphore(dev)
	sf.RenderFence = NewFence(dev)

	sf.Frames = make([]*Framebuffer, sf.NFrames)
	for i := 0; i < sf.NFrames; i++ {
		fr := &Framebuffer{}
		fr.ConfigImage(dev, sf.Format, swapchainImages[i])
		sf.Frames[i] = fr
	}
}

// FreeSwapchain frees any existing swawpchain (for ReInit or Destroy)
func (sf *Surface) FreeSwapchain() {
	dev := sf.Device.Device
	vk.DeviceWaitIdle(dev)
	vk.DestroySemaphore(dev, sf.ImageAcquired, nil)
	vk.DestroySemaphore(dev, sf.RenderDone, nil)
	vk.DestroyFence(dev, sf.RenderFence, nil)
	for _, fr := range sf.Frames {
		fr.Destroy()
	}
	sf.Frames = nil
	if sf.Swapchain != vk.NullSwapchain {
		vk.DestroySwapchain(dev, sf.Swapchain, nil)
		sf.Swapchain = vk.NullSwapchain
	}
}

// ReConfigSwapchain does a re-initialize of swapchain, freeing existing.
// This must be called when the window is resized.
func (sf *Surface) ReConfigSwapchain() {
	sf.FreeSwapchain()
	sf.ConfigSwapchain()
	sf.RenderPass.SetDepthSize(sf.Format.Size)
	sf.ReConfigFrames()
}

// SetRenderPass sets the RenderPass and updates frames accordingly
func (sf *Surface) SetRenderPass(rp *RenderPass) {
	sf.RenderPass = rp
	for _, fr := range sf.Frames {
		fr.ConfigRenderPass(rp)
	}
}

// ReConfigFrames re-configures the Famebuffers
// using exiting settings.  Assumes ConfigSwapchain has been called.
func (sf *Surface) ReConfigFrames() {
	for _, fr := range sf.Frames {
		fr.ConfigRenderPass(sf.RenderPass)
	}
}

func (sf *Surface) Destroy() {
	dev := sf.Device.Device
	sf.FreeSwapchain()
	if sf.Surface != vk.NullSurface {
		vk.DestroySurface(sf.GPU.Instance, sf.Surface, nil)
		sf.Surface = vk.NullSurface
	}
	sf.CmdPool.Destroy(dev)
	sf.Device.Destroy()
	sf.GPU = nil
}

// AcquireNextImage gets the next frame index to render to.
// It automatically handles any issues with out-of-date swapchain.
// It triggers the ImageAcquired semaphore when image actually acquired.
// Must call SubmitRender with command to launch command contingent
// on that semaphore.
func (sf *Surface) AcquireNextImage() uint32 {
	dev := sf.Device.Device
	vk.WaitForFences(dev, 1, []vk.Fence{sf.RenderFence}, vk.True, vk.MaxUint64)
	vk.ResetFences(dev, 1, []vk.Fence{sf.RenderFence})
	var idx uint32
	ret := vk.AcquireNextImage(dev, sf.Swapchain, vk.MaxUint64, sf.ImageAcquired, vk.NullFence, &idx)
	switch ret {
	case vk.ErrorOutOfDate, vk.Suboptimal:
		sf.ReConfigSwapchain()
		if sf.GPU.Debug {
			fmt.Printf("vgpu.Surface:AcquireNextImage, new format: %#v\n", sf.Format)
		}
		return sf.AcquireNextImage() // try again
	case vk.Success:
	default:
		IfPanic(NewError(ret))
	}
	return idx
}

// SubmitRender submits a rendering command that must have been added
// to the sf.CmdPool.Buff buffer.  This buffer triggers the associated
// Fence logic to control the sequencing of render commands over time.
// The ImageAcquired semaphore efore the command is run.
func (sf *Surface) SubmitRender(cmd vk.CommandBuffer) {
	ret := vk.QueueSubmit(sf.Device.Queue, 1, []vk.SubmitInfo{{
		SType: vk.StructureTypeSubmitInfo,
		PWaitDstStageMask: []vk.PipelineStageFlags{
			vk.PipelineStageFlags(vk.PipelineStageColorAttachmentOutputBit),
		},
		WaitSemaphoreCount:   1,
		PWaitSemaphores:      []vk.Semaphore{sf.ImageAcquired},
		CommandBufferCount:   1,
		PCommandBuffers:      []vk.CommandBuffer{cmd},
		SignalSemaphoreCount: 1,
		PSignalSemaphores:    []vk.Semaphore{sf.RenderDone},
	}}, sf.RenderFence)
	IfPanic(NewError(ret))
}

// PresentImage waits on the RenderDone semaphore to present the
// rendered image to the surface, for the given frame index,
// as returned by AcquireNextImage.
func (sf *Surface) PresentImage(frameIdx uint32) error {
	ret := vk.QueuePresent(sf.Device.Queue, &vk.PresentInfo{
		SType:              vk.StructureTypePresentInfo,
		WaitSemaphoreCount: 1,
		PWaitSemaphores:    []vk.Semaphore{sf.RenderDone},
		SwapchainCount:     1,
		PSwapchains:        []vk.Swapchain{sf.Swapchain},
		PImageIndices:      []uint32{frameIdx},
	})

	switch ret {
	case vk.ErrorOutOfDate, vk.Suboptimal:
		sf.ReConfigSwapchain()
		if sf.GPU.Debug {
			fmt.Printf("vgpu.Surface:PresentImage, new format: %#v\n", sf.Format)
		}
		return fmt.Errorf("vgpu.Surface:PresentImage: swapchain was out of date, reinitialized -- not rendered")
	case vk.Success:
		return nil
	default:
		return NewError(ret)
	}
}
