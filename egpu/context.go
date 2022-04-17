// Copyright (c) 2022, The Emergent Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// This is initially adapted from https://github.com/vulkan-go/asche
// Copyright © 2017 Maxim Kupriianov <max@kc.vc>, under the MIT License

package egpu

import (
	"errors"

	vk "github.com/vulkan-go/vulkan"
)

// Context holds GPU context
type Context struct {
	Platform *Platform
	Device   vk.Device

	// OnPrepare is a callback that will be invoked to initialize
	// and prepare vulkan state upon context prepare step.
	// Could create textures and pipelines, descriptor layouts
	// and render passes.
	OnPrepare func() error

	// OnCleanup is a callback that will be invoked to cleanup
	// application's vulkan state upon context destruction step.
	// Should destroy resources created during OnPrepare
	OnCleanup func() error

	// OnInvalidate is a callback that will be invoked when
	// context has been invalidated.
	// The application must update its state and prepare the
	// corresponding swapchain image to be presented.
	// Could compute new vertex and color data in swapchain
	// image resource buffers.
	OnInvalidate func(imageIdx int) error

	CmdBuff vk.CommandBuffer

	CmdPool        vk.CommandPool
	PresentCmdPool vk.CommandPool

	Swapchain vk.Swapchain

	// SwapchainDims has the current swapchain dimensions, including pixel format.
	SwapchainDims *SwapchainDims

	// ImageResources exposes the swapchain initialized image resources.
	ImageResources []*ImageResources

	FrameLag int

	ImageAcquiredSemaphores  []vk.Semaphore
	DrawCompleteSemaphores   []vk.Semaphore
	ImageOwnershipSemaphores []vk.Semaphore

	FrameIndex int
}

func (c *Context) PreparePresent() {
	// Create semaphores to synchronize acquiring presentable buffers before
	// rendering and waiting for drawing to be complete before presenting
	semaphoreCreateInfo := &vk.SemaphoreCreateInfo{
		SType: vk.StructureTypeSemaphoreCreateInfo,
	}
	c.ImageAcquiredSemaphores = make([]vk.Semaphore, c.FrameLag)
	c.DrawCompleteSemaphores = make([]vk.Semaphore, c.FrameLag)
	c.ImageOwnershipSemaphores = make([]vk.Semaphore, c.FrameLag)
	for i := 0; i < c.FrameLag; i++ {
		ret := vk.CreateSemaphore(c.Device, semaphoreCreateInfo, nil, &c.ImageAcquiredSemaphores[i])
		IfPanic(NewError(ret))
		ret = vk.CreateSemaphore(c.Device, semaphoreCreateInfo, nil, &c.DrawCompleteSemaphores[i])
		IfPanic(NewError(ret))
		if c.Platform.HasSeparatePresentQueue() {
			ret = vk.CreateSemaphore(c.Device, semaphoreCreateInfo, nil, &c.ImageOwnershipSemaphores[i])
			IfPanic(NewError(ret))
		}
	}
}

func (c *Context) Destroy() {
	func() (err error) {
		CheckErr(&err)
		if c.OnCleanup != nil {
			err = c.OnCleanup()
		}
		return
	}()

	for i := 0; i < c.FrameLag; i++ {
		vk.DestroySemaphore(c.Device, c.ImageAcquiredSemaphores[i], nil)
		vk.DestroySemaphore(c.Device, c.DrawCompleteSemaphores[i], nil)
		if c.Platform.HasSeparatePresentQueue() {
			vk.DestroySemaphore(c.Device, c.ImageOwnershipSemaphores[i], nil)
		}
	}
	for i := 0; i < len(c.ImageResources); i++ {
		c.ImageResources[i].Destroy(c.Device, c.CmdPool)
	}
	c.ImageResources = nil
	if c.Swapchain != vk.NullSwapchain {
		vk.DestroySwapchain(c.Device, c.Swapchain, nil)
		c.Swapchain = vk.NullSwapchain
	}
	vk.DestroyCommandPool(c.Device, c.CmdPool, nil)
	if c.Platform.HasSeparatePresentQueue() {
		vk.DestroyCommandPool(c.Device, c.PresentCmdPool, nil)
	}
	c.Platform = nil
}

func (c *Context) Prepare(needCleanup bool) {
	vk.DeviceWaitIdle(c.Device)

	if needCleanup {
		if c.OnCleanup != nil {
			IfPanic(c.OnCleanup())
		}

		vk.DestroyCommandPool(c.Device, c.CmdPool, nil)
		if c.Platform.HasSeparatePresentQueue() {
			vk.DestroyCommandPool(c.Device, c.PresentCmdPool, nil)
		}
	}

	var CmdPool vk.CommandPool
	ret := vk.CreateCommandPool(c.Device, &vk.CommandPoolCreateInfo{
		SType:            vk.StructureTypeCommandPoolCreateInfo,
		QueueFamilyIndex: c.Platform.GraphicsQueueIndex,
	}, nil, &CmdPool)
	IfPanic(NewError(ret))
	c.CmdPool = CmdPool

	var CmdBuff = make([]vk.CommandBuffer, 1)
	ret = vk.AllocateCommandBuffers(c.Device, &vk.CommandBufferAllocateInfo{
		SType:              vk.StructureTypeCommandBufferAllocateInfo,
		CommandPool:        c.CmdPool,
		Level:              vk.CommandBufferLevelPrimary,
		CommandBufferCount: 1,
	}, CmdBuff)
	IfPanic(NewError(ret))
	c.CmdBuff = CmdBuff[0]

	ret = vk.BeginCommandBuffer(c.CmdBuff, &vk.CommandBufferBeginInfo{
		SType: vk.StructureTypeCommandBufferBeginInfo,
	})
	IfPanic(NewError(ret))

	for i := 0; i < len(c.ImageResources); i++ {
		var CmdBuff = make([]vk.CommandBuffer, 1)
		vk.AllocateCommandBuffers(c.Device, &vk.CommandBufferAllocateInfo{
			SType:              vk.StructureTypeCommandBufferAllocateInfo,
			CommandPool:        c.CmdPool,
			Level:              vk.CommandBufferLevelPrimary,
			CommandBufferCount: 1,
		}, CmdBuff)
		IfPanic(NewError(ret))
		c.ImageResources[i].CmdBuff = CmdBuff[0]
	}

	if c.Platform.HasSeparatePresentQueue() {
		var CmdPool vk.CommandPool
		ret = vk.CreateCommandPool(c.Device, &vk.CommandPoolCreateInfo{
			SType:            vk.StructureTypeCommandPoolCreateInfo,
			QueueFamilyIndex: c.Platform.PresentQueueIndex,
		}, nil, &CmdPool)
		IfPanic(NewError(ret))
		c.PresentCmdPool = CmdPool

		for i := 0; i < len(c.ImageResources); i++ {
			var CmdBuff = make([]vk.CommandBuffer, 1)
			ret = vk.AllocateCommandBuffers(c.Device, &vk.CommandBufferAllocateInfo{
				SType:              vk.StructureTypeCommandBufferAllocateInfo,
				CommandPool:        c.PresentCmdPool,
				Level:              vk.CommandBufferLevelPrimary,
				CommandBufferCount: 1,
			}, CmdBuff)
			IfPanic(NewError(ret))
			c.ImageResources[i].GraphicsToPresentCmd = CmdBuff[0]

			c.ImageResources[i].SetImageOwnership(
				c.Platform.GraphicsQueueIndex, c.Platform.PresentQueueIndex)
		}
	}

	for i := 0; i < len(c.ImageResources); i++ {
		var view vk.ImageView
		ret = vk.CreateImageView(c.Device, &vk.ImageViewCreateInfo{
			SType:  vk.StructureTypeImageViewCreateInfo,
			Format: c.SwapchainDims.Format,
			Components: vk.ComponentMapping{
				R: vk.ComponentSwizzleR,
				G: vk.ComponentSwizzleG,
				B: vk.ComponentSwizzleB,
				A: vk.ComponentSwizzleA,
			},
			SubresourceRange: vk.ImageSubresourceRange{
				AspectMask: vk.ImageAspectFlags(vk.ImageAspectColorBit),
				LevelCount: 1,
				LayerCount: 1,
			},
			ViewType: vk.ImageViewType2d,
			Image:    c.ImageResources[i].Image,
		}, nil, &view)
		IfPanic(NewError(ret))
		c.ImageResources[i].View = view
	}

	if c.OnPrepare != nil {
		IfPanic(c.OnPrepare())
	}
	c.FlushInitCmd()
}

func (c *Context) FlushInitCmd() {
	if c.CmdBuff == nil {
		return
	}
	ret := vk.EndCommandBuffer(c.CmdBuff)
	IfPanic(NewError(ret))

	var fence vk.Fence
	ret = vk.CreateFence(c.Device, &vk.FenceCreateInfo{
		SType: vk.StructureTypeFenceCreateInfo,
	}, nil, &fence)
	IfPanic(NewError(ret))

	cmdBufs := []vk.CommandBuffer{c.CmdBuff}
	ret = vk.QueueSubmit(c.Platform.GraphicsQueue, 1, []vk.SubmitInfo{{
		SType:              vk.StructureTypeSubmitInfo,
		CommandBufferCount: 1,
		PCommandBuffers:    cmdBufs,
	}}, fence)
	IfPanic(NewError(ret))

	ret = vk.WaitForFences(c.Device, 1, []vk.Fence{fence}, vk.True, vk.MaxUint64)
	IfPanic(NewError(ret))

	vk.FreeCommandBuffers(c.Device, c.CmdPool, 1, cmdBufs)
	vk.DestroyFence(c.Device, fence, nil)
	c.CmdBuff = nil
}

func (c *Context) PrepareSwapchain(gpu vk.PhysicalDevice, surface vk.Surface, dimensions *SwapchainDims) {
	// Read surface capabilities
	var surfaceCapabilities vk.SurfaceCapabilities
	ret := vk.GetPhysicalDeviceSurfaceCapabilities(gpu, surface, &surfaceCapabilities)
	IfPanic(NewError(ret))
	surfaceCapabilities.Deref()

	// Get available surface pixel formats
	var formatCount uint32
	vk.GetPhysicalDeviceSurfaceFormats(gpu, surface, &formatCount, nil)
	formats := make([]vk.SurfaceFormat, formatCount)
	vk.GetPhysicalDeviceSurfaceFormats(gpu, surface, &formatCount, formats)

	// Select a proper surface format
	var format vk.SurfaceFormat
	if formatCount == 1 {
		formats[0].Deref()
		if formats[0].Format == vk.FormatUndefined {
			format = formats[0]
			format.Format = dimensions.Format
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
		swapchainSize.Width = dimensions.Width
		swapchainSize.Height = dimensions.Height
	} else {
		swapchainSize = surfaceCapabilities.CurrentExtent
	}
	// The FIFO present mode is guaranteed by the spec to be supported
	// and to have no tearing.  It's a great default present mode to use.
	swapchainPresentMode := vk.PresentModeFifo

	// Determine the number of VkImage's to use in the swapchain.
	// Ideally, we desire to own 1 image at a time, the rest of the images can either be rendered to and/or
	// being queued up for display.
	desiredSwapchainImages := surfaceCapabilities.MinImageCount + 1
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
		vk.CompositeAlphaOpaqueBit,
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
	oldSwapchain := c.Swapchain
	ret = vk.CreateSwapchain(c.Device, &vk.SwapchainCreateInfo{
		SType:           vk.StructureTypeSwapchainCreateInfo,
		Surface:         surface,
		MinImageCount:   desiredSwapchainImages, // 1 - 3?
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
	}, nil, &swapchain)
	IfPanic(NewError(ret))
	if oldSwapchain != vk.NullSwapchain {
		vk.DestroySwapchain(c.Device, oldSwapchain, nil)
	}
	c.Swapchain = swapchain

	c.SwapchainDims = &SwapchainDims{
		Width:  swapchainSize.Width,
		Height: swapchainSize.Height,
		Format: format.Format,
	}

	var imageCount uint32
	ret = vk.GetSwapchainImages(c.Device, c.Swapchain, &imageCount, nil)
	IfPanic(NewError(ret))
	swapchainImages := make([]vk.Image, imageCount)
	ret = vk.GetSwapchainImages(c.Device, c.Swapchain, &imageCount, swapchainImages)
	IfPanic(NewError(ret))
	for i := 0; i < len(c.ImageResources); i++ {
		c.ImageResources[i].Destroy(c.Device, c.CmdPool)
	}
	c.ImageResources = make([]*ImageResources, 0, imageCount)
	for i := 0; i < len(swapchainImages); i++ {
		c.ImageResources = append(c.ImageResources, &ImageResources{
			Image: swapchainImages[i],
		})
	}
}

func (c *Context) AcquireNextImage() (imageIndex int, outdated bool, err error) {
	defer CheckErr(&err)

	// Get the index of the next available swapchain image
	var idx uint32
	ret := vk.AcquireNextImage(c.Device, c.Swapchain, vk.MaxUint64,
		c.ImageAcquiredSemaphores[c.FrameIndex], vk.NullFence, &idx)
	imageIndex = int(idx)
	if c.OnInvalidate != nil {
		IfPanic(c.OnInvalidate(imageIndex))
	}
	switch ret {
	case vk.ErrorOutOfDate:
		c.FrameIndex++
		c.FrameIndex = c.FrameIndex % c.FrameLag
		c.PrepareSwapchain(c.Platform.Gpu, c.Platform.Surface, c.SwapchainDims)
		c.Prepare(true)
		outdated = true
		return
	case vk.Suboptimal, vk.Success:
	default:
		IfPanic(NewError(ret))
	}

	graphicsQueue := c.Platform.GraphicsQueue
	var nullFence vk.Fence
	ret = vk.QueueSubmit(graphicsQueue, 1, []vk.SubmitInfo{{
		SType: vk.StructureTypeSubmitInfo,
		PWaitDstStageMask: []vk.PipelineStageFlags{
			vk.PipelineStageFlags(vk.PipelineStageColorAttachmentOutputBit),
		},
		WaitSemaphoreCount: 1,
		PWaitSemaphores: []vk.Semaphore{
			c.ImageAcquiredSemaphores[c.FrameIndex],
		},
		CommandBufferCount: 1,
		PCommandBuffers: []vk.CommandBuffer{
			c.ImageResources[idx].CmdBuff,
		},
		SignalSemaphoreCount: 1,
		PSignalSemaphores: []vk.Semaphore{
			c.DrawCompleteSemaphores[c.FrameIndex],
		},
	}}, nullFence)
	IfPanic(NewError(ret))

	if c.Platform.HasSeparatePresentQueue() {
		presentQueue := c.Platform.PresentQueue()

		var nullFence vk.Fence
		ret = vk.QueueSubmit(presentQueue, 1, []vk.SubmitInfo{{
			SType: vk.StructureTypeSubmitInfo,
			PWaitDstStageMask: []vk.PipelineStageFlags{
				vk.PipelineStageFlags(vk.PipelineStageColorAttachmentOutputBit),
			},
			WaitSemaphoreCount: 1,
			PWaitSemaphores: []vk.Semaphore{
				c.ImageAcquiredSemaphores[c.FrameIndex],
			},
			CommandBufferCount: 1,
			PCommandBuffers: []vk.CommandBuffer{
				c.ImageResources[idx].GraphicsToPresentCmd,
			},
			SignalSemaphoreCount: 1,
			PSignalSemaphores: []vk.Semaphore{
				c.ImageOwnershipSemaphores[c.FrameIndex],
			},
		}}, nullFence)
		IfPanic(NewError(ret))
	}
	return
}

func (c *Context) PresentImage(imageIdx int) (outdated bool, err error) {
	// If we are using separate queues we have to wait for image ownership,
	// otherwise wait for draw complete.
	var semaphore vk.Semaphore
	if c.Platform.HasSeparatePresentQueue() {
		semaphore = c.ImageOwnershipSemaphores[c.FrameIndex]
	} else {
		semaphore = c.DrawCompleteSemaphores[c.FrameIndex]
	}
	presentQueue := c.Platform.PresentQueue()
	ret := vk.QueuePresent(presentQueue, &vk.PresentInfo{
		SType:              vk.StructureTypePresentInfo,
		WaitSemaphoreCount: 1,
		PWaitSemaphores:    []vk.Semaphore{semaphore},
		SwapchainCount:     1,
		PSwapchains:        []vk.Swapchain{c.Swapchain},
		PImageIndices:      []uint32{uint32(imageIdx)},
	})
	c.FrameIndex++
	c.FrameIndex = c.FrameIndex % c.FrameLag

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
