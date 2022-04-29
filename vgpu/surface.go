// Copyright (c) 2022, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// This is initially adapted from https://github.com/vulkan-go/asche
// Copyright © 2017 Maxim Kupriianov <max@kc.vc>, under the MIT License

package vgpu

import (
	"errors"
	"fmt"

	vk "github.com/vulkan-go/vulkan"
)

/*
func (s *SurfaceFrame) SetImageOwnership(graphicsQueueIndex, presentQueueIndex uint32) {
	ret := vk.BeginCommandBuffer(s.GraphicsToPresentCmd, &vk.CommandBufferBeginInfo{
		SType: vk.StructureTypeCommandBufferBeginInfo,
		Flags: vk.CommandBufferUsageFlags(vk.CommandBufferUsageSimultaneousUseBit),
	})
	IfPanic(NewError(ret))

	vk.CmdPipelineBarrier(s.GraphicsToPresentCmd,
		vk.PipelineStageFlags(vk.PipelineStageColorAttachmentOutputBit),
		vk.PipelineStageFlags(vk.PipelineStageColorAttachmentOutputBit),
		0, 0, nil, 0, nil, 1, []vk.ImageMemoryBarrier{{
			SType:               vk.StructureTypeImageMemoryBarrier,
			DstAccessMask:       vk.AccessFlags(vk.AccessColorAttachmentWriteBit),
			OldLayout:           vk.ImageLayoutPresentSrc,
			NewLayout:           vk.ImageLayoutPresentSrc,
			SrcQueueFamilyIndex: graphicsQueueIndex,
			DstQueueFamilyIndex: presentQueueIndex,
			Image:               s.Image,

			SubresourceRange: vk.ImageSubresourceRange{
				AspectMask: vk.ImageAspectFlags(vk.ImageAspectColorBit),
				LevelCount: 1,
				LayerCount: 1,
			},
		}})

	ret = vk.EndCommandBuffer(s.GraphicsToPresentCmd)
	IfPanic(NewError(ret))
}
*/

// Surface manages the physical device for the visible image
// of a window surface, and the swapchain for presenting images.
type Surface struct {
	GPU                      *GPU        `desc:"pointer to gpu device, for convenience"`
	Device                   Device      `desc:"device for this surface -- each window surface has its own device, configured for that surface"`
	RenderPass               *RenderPass `desc:"the RenderPass for this Surface, typically from a System"`
	CmdPool                  CmdPool
	Format                   ImageFormat    `desc:"has the current swapchain image format and dimensions"`
	NFrames                  int            `desc:"number of frames to maintain in the swapchain -- e.g., 2 = double-buffering, 3 = triple-buffering -- initially set to a requested amount, and after Init reflects actual number"`
	Frames                   []*Framebuffer `desc:"data for each visible image owned by the Surface -- we iterate through these in rendering subsequent frames"`
	FrameIndex               int            `desc:"index for current frame"`
	ImageAcquiredSemaphores  []vk.Semaphore
	DrawCompleteSemaphores   []vk.Semaphore
	ImageOwnershipSemaphores []vk.Semaphore

	Surface   vk.Surface   `desc:"vulkan handle for surface"`
	Swapchain vk.Swapchain `desc:"vulkan handle for swapchain"`
}

func (sf *Surface) Defaults() {
	sf.NFrames = 2                                  // requested, will be updated with actual
	sf.Format.Set(1024, 768, vk.FormatR8g8b8a8Srgb) // requested, will be updated with actual
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
	sf.CmdPool.Init(&sf.Device, 0) // todo: not clear what we need this for
	sf.InitSwapchain()
	return nil
}

// InitSwapchain initializes the swapchain for surface
func (sf *Surface) InitSwapchain() {
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
		formats[0].Deref()
		// select the first one available
		format = formats[0]
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
	fmt.Printf("swapchain size: %#v\n", swapchainSize)

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
	ret = vk.CreateSwapchain(sf.Device.Device, swci, nil, &swapchain)
	IfPanic(NewError(ret))
	if oldSwapchain != vk.NullSwapchain {
		vk.DestroySwapchain(sf.Device.Device, oldSwapchain, nil)
	}
	sf.Swapchain = swapchain
	sf.Format.Set(int(swapchainSize.Width), int(swapchainSize.Width), format.Format)

	var imageCount uint32
	ret = vk.GetSwapchainImages(sf.Device.Device, sf.Swapchain, &imageCount, nil)
	IfPanic(NewError(ret))
	sf.NFrames = int(imageCount)
	swapchainImages := make([]vk.Image, imageCount)
	ret = vk.GetSwapchainImages(sf.Device.Device, sf.Swapchain, &imageCount, swapchainImages)
	IfPanic(NewError(ret))
	for i := 0; i < len(sf.Frames); i++ {
		sf.Frames[i].Destroy()
	}
	sf.Frames = make([]*Framebuffer, sf.NFrames)
	for i := 0; i < len(swapchainImages); i++ {
		fr := &Framebuffer{}
		fr.InitImage(sf.Device.Device, sf.Format, swapchainImages[i])
		sf.Frames[i] = fr
	}
}

// FreeSwapchain frees any existing swawpchain (for ReInit or Destroy)
func (sf *Surface) FreeSwapchain() {
	vk.DeviceWaitIdle(sf.Device.Device)
	for i := 0; i < sf.NFrames; i++ {
		vk.DestroySemaphore(sf.Device.Device, sf.ImageAcquiredSemaphores[i], nil)
		vk.DestroySemaphore(sf.Device.Device, sf.DrawCompleteSemaphores[i], nil)
		vk.DestroySemaphore(sf.Device.Device, sf.ImageOwnershipSemaphores[i], nil)
	}
	for _, fr := range sf.Frames {
		fr.Destroy()
	}
	sf.Frames = nil
	if sf.Swapchain != vk.NullSwapchain {
		vk.DestroySwapchain(sf.Device.Device, sf.Swapchain, nil)
		sf.Swapchain = vk.NullSwapchain
	}
}

// ReInitSwapchain does a re-initialize of swapchain, freeing existing.
// This must be called when the window is resized.
func (sf *Surface) ReInitSwapchain() {
	sf.FreeSwapchain()
	sf.InitSwapchain()
	sf.RenderPass.SetDepthSize(sf.Format.Size)
	sf.ReInitFrames()
}

// SetRenderPass sets the RenderPass and updates frames accordingly
func (sf *Surface) SetRenderPass(rp *RenderPass) {
	sf.RenderPass = rp
	for _, fr := range sf.Frames {
		fr.InitRenderPass(rp)
	}
}

// ReInitFrames re-initializes the Frame framebuffers
// using exiting settings.  Assumes InitSwapchain has been called.
func (sf *Surface) ReInitFrames() {
	for _, fr := range sf.Frames {
		fr.InitRenderPass(sf.RenderPass)
	}
}

func (sf *Surface) PreparePresent() {
	// Create semaphores to synchronize acquiring presentable buffers before
	// rendering and waiting for drawing to be complete before presenting
	semaphoreCreateInfo := &vk.SemaphoreCreateInfo{
		SType: vk.StructureTypeSemaphoreCreateInfo,
	}
	sf.ImageAcquiredSemaphores = make([]vk.Semaphore, sf.NFrames)
	sf.DrawCompleteSemaphores = make([]vk.Semaphore, sf.NFrames)
	sf.ImageOwnershipSemaphores = make([]vk.Semaphore, sf.NFrames)
	for i := 0; i < sf.NFrames; i++ {
		ret := vk.CreateSemaphore(sf.Device.Device, semaphoreCreateInfo, nil, &sf.ImageAcquiredSemaphores[i])
		IfPanic(NewError(ret))
		ret = vk.CreateSemaphore(sf.Device.Device, semaphoreCreateInfo, nil, &sf.DrawCompleteSemaphores[i])
		IfPanic(NewError(ret))
		ret = vk.CreateSemaphore(sf.Device.Device, semaphoreCreateInfo, nil, &sf.ImageOwnershipSemaphores[i])
		IfPanic(NewError(ret))
	}
}

func (sf *Surface) Destroy() {
	sf.FreeSwapchain()
	if sf.Surface != vk.NullSurface {
		vk.DestroySurface(sf.GPU.Instance, sf.Surface, nil)
		sf.Surface = vk.NullSurface
	}
	sf.CmdPool.Destroy(sf.Device.Device)
	sf.Device.Destroy()
	sf.GPU = nil
}

func (sf *Surface) Prepare(needCleanup bool) {

	if needCleanup {
		sf.CmdPool.Destroy(sf.Device.Device)
		var presentQueue vk.Queue
		vk.GetDeviceQueue(sf.Device.Device, sf.Device.QueueIndex, 0, &presentQueue)
		sf.Device.Queue = presentQueue
	}

	// sf.FlushInitCmd()
}

func (sf *Surface) FlushInitCmd() {
	/*
		if sf.CmdBuff == nil {
			return
		}
		ret := vk.EndCommandBuffer(sf.CmdBuff)
		IfPanic(NewError(ret))

		var fence vk.Fence
		ret = vk.CreateFence(sf.Device.Device, &vk.FenceCreateInfo{
			SType: vk.StructureTypeFenceCreateInfo,
		}, nil, &fence)
		IfPanic(NewError(ret))

		cmdBufs := []vk.CommandBuffer{sf.CmdBuff}
		ret = vk.QueueSubmit(sf.GPU.Queue, 1, []vk.SubmitInfo{{
			SType:              vk.StructureTypeSubmitInfo,
			CommandBufferCount: 1,
			PCommandBuffers:    cmdBufs,
		}}, fence)
		IfPanic(NewError(ret))

		ret = vk.WaitForFences(sf.Device.Device, 1, []vk.Fence{fence}, vk.True, vk.MaxUint64)
		IfPanic(NewError(ret))

		vk.FreeCommandBuffers(sf.Device.Device, sf.CmdPool.Pool, 1, cmdBufs)
		vk.DestroyFence(sf.Device.Device, fence, nil)
		sf.CmdBuff = nil
	*/
}

func (sf *Surface) AcquireNextImage() (imageIndex int, outdated bool, err error) {
	/*
		defer CheckErr(&err)

		// Get the index of the next available swapchain image
		var idx uint32
		ret := vk.AcquireNextImage(sf.Device.Device, sf.Swapchain, vk.MaxUint64,
			sf.ImageAcquiredSemaphores[sf.FrameIndex], vk.NullFence, &idx)
		imageIndex = int(idx)
		switch ret {
		case vk.ErrorOutOfDate:
			sf.FrameIndex++
			sf.FrameIndex = sf.FrameIndex % sf.NFrames
			// sf.PrepareSwapchain()
			// sf.Prepare(true)
			outdated = true
			return
		case vk.Suboptimal, vk.Success:
		default:
			IfPanic(NewError(ret))
		}

		presentQueue := sf.Device.Queue

		var nullFence vk.Fence
		ret = vk.QueueSubmit(presentQueue, 1, []vk.SubmitInfo{{
			SType: vk.StructureTypeSubmitInfo,
			PWaitDstStageMask: []vk.PipelineStageFlags{
				vk.PipelineStageFlags(vk.PipelineStageColorAttachmentOutputBit),
			},
			WaitSemaphoreCount: 1,
			PWaitSemaphores: []vk.Semaphore{
				sf.ImageAcquiredSemaphores[sf.FrameIndex],
			},
			CommandBufferCount: 1,
			PCommandBuffers: []vk.CommandBuffer{
				sf.Frames[idx].GraphicsToPresentCmd,
			},
			SignalSemaphoreCount: 1,
			PSignalSemaphores: []vk.Semaphore{
				sf.ImageOwnershipSemaphores[sf.FrameIndex],
			},
		}}, nullFence)
		IfPanic(NewError(ret))
	*/
	return
}

func (sf *Surface) PresentImage(imageIdx int) (outdated bool, err error) {
	// If we are using separate queues we have to wait for image ownership,
	// otherwise wait for draw complete.
	var semaphore vk.Semaphore
	semaphore = sf.ImageOwnershipSemaphores[sf.FrameIndex]

	presentQueue := sf.Device.Queue
	ret := vk.QueuePresent(presentQueue, &vk.PresentInfo{
		SType:              vk.StructureTypePresentInfo,
		WaitSemaphoreCount: 1,
		PWaitSemaphores:    []vk.Semaphore{semaphore},
		SwapchainCount:     1,
		PSwapchains:        []vk.Swapchain{sf.Swapchain},
		PImageIndices:      []uint32{uint32(imageIdx)},
	})
	sf.FrameIndex++
	sf.FrameIndex = sf.FrameIndex % sf.NFrames

	switch ret {
	case vk.ErrorOutOfDate:
		outdated = true
		return
	case vk.Suboptimal, vk.Success:
		return
	default:
		err = NewError(ret)
		return
	}
}
