// Copyright (c) 2019, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// based extensively on vulkan-go/asche
// The MIT License (MIT)
// Copyright © 2017 Maxim Kupriianov <max@kc.vc>

package gpu

import (
	"errors"

	"github.com/goki/gi/oswin"
	"github.com/vulkan-go/vulkan"
)

// Context provides an OpenGL-style graphical rendering context
// that is specific to a particular rendering surface
type Context interface {
	// SetOnPrepare sets callback that will be invoked to initialize and prepare application's vulkan state
	// upon context prepare step. onCreate could create textures and pipelines,
	// descriptor layouts and render passes.
	SetOnPrepare(onPrepare func() error)

	// SetOnCleanup sets callback that will be invoked to cleanup application's vulkan state
	// upon context prepare step. onCreate could destroy textures and pipelines,
	// descriptor layouts and render passes.
	SetOnCleanup(onCleanup func() error)

	// SetOnInvalidate sets callback that will be invoked when context has been invalidated,
	// the application must update its state and prepare the corresponding swapchain image to be presented.
	// onInvalidate could compute new vertex and color data in swapchain image resource buffers.
	SetOnInvalidate(onInvalidate func(imageIdx int) error)

	// Device returns the Vulkan device assigned to the context.
	Device() vulkan.Device

	// GPU returns the current oswin.GPU info (also avail as oswin.TheGPU)
	GPU() oswin.GPU

	// Surface returns the surface to render on
	Surface() vulkan.Surface

	// SetSurface sets the surface to use
	SetSurface(surf vulkan.Surface)

	// CommandBuffer returns a command buffer currently active.
	CommandBuffer() vulkan.CommandBuffer

	// SwapchainDimensions returns the current swapchain dimensions, including pixel format.
	SwapchainDimensions() *SwapchainDimensions

	// SwapchainImageResources exposes the swapchain initialized image resources.
	SwapchainImageResources() []*SwapchainImageResources

	// AcquireNextImage
	AcquireNextImage() (imageIndex int, outdated bool, err error)

	// PresentImage
	PresentImage(imageIdx int) (outdated bool, err error)

	// Destroy frees resources associated with the context
	Destroy()
}

// ctxtBase provides the base implementation of Context
// different OS platforms can add additional fields / functions
// as needed
type ctxtBase struct {
	surface vulkan.Surface
	theGPU  oswin.GPU
	device  vulkan.Device

	onPrepare    func() error
	onCleanup    func() error
	onInvalidate func(imageIdx int) error

	cmd            vulkan.CommandBuffer
	cmdPool        vulkan.CommandPool
	presentCmdPool vulkan.CommandPool

	swapchain               vulkan.Swapchain
	swapchainDimensions     *SwapchainDimensions
	swapchainImageResources []*SwapchainImageResources
	frameLag                int

	imageAcquiredSemaphores  []vulkan.Semaphore
	drawCompleteSemaphores   []vulkan.Semaphore
	imageOwnershipSemaphores []vulkan.Semaphore

	frameIndex int
}

func (c *ctxtBase) initSwapchain() {
	// if separateQueue {
	// 	var presentQueue vulkan.Queue
	// 	vulkan.GetDeviceQueue(gp.device, gp.presentQueueIndex, 0, &presentQueue)
	// 	gp.presentQueue = presentQueue
	// }

	c.preparePresent()
	dimensions := &SwapchainDimensions{
		// some default preferences here
		Width: 640, Height: 480,
		Format: vulkan.FormatB8g8r8a8Unorm,
	}
	// if iface, ok := app.(ApplicationSwapchainDimensions); ok {
	// 	dimensions = iface.VulkanSwapchainDimensions()
	// }

	c.prepareSwapchain(c.theGPU.PhysicalDevice(), c.surface, dimensions)

	// if iface, ok := app.(ApplicationContextPrepare); ok {
	// 	gp.context.SetOnPrepare(iface.VulkanContextPrepare)
	// }
	// if iface, ok := app.(ApplicationContextCleanup); ok {
	// 	gp.context.SetOnCleanup(iface.VulkanContextCleanup)
	// }
	// if iface, ok := app.(ApplicationContextInvalidate); ok {
	// 	gp.context.SetOnInvalidate(iface.VulkanContextInvalidate)
	// }

	c.prepare(false)
}

func (c *ctxtBase) preparePresent() {
	// Create semaphores to synchronize acquiring presentable buffers before
	// rendering and waiting for drawing to be complete before presenting
	semaphoreCreateInfo := &vulkan.SemaphoreCreateInfo{
		SType: vulkan.StructureTypeSemaphoreCreateInfo,
	}
	c.imageAcquiredSemaphores = make([]vulkan.Semaphore, c.frameLag)
	c.drawCompleteSemaphores = make([]vulkan.Semaphore, c.frameLag)
	c.imageOwnershipSemaphores = make([]vulkan.Semaphore, c.frameLag)
	for i := 0; i < c.frameLag; i++ {
		ret := vulkan.CreateSemaphore(c.device, semaphoreCreateInfo, nil, &c.imageAcquiredSemaphores[i])
		orPanic(NewError(ret))
		ret = vulkan.CreateSemaphore(c.device, semaphoreCreateInfo, nil, &c.drawCompleteSemaphores[i])
		orPanic(NewError(ret))
		if c.theGPU.HasSeparatePresentQueue() {
			ret = vulkan.CreateSemaphore(c.device, semaphoreCreateInfo, nil, &c.imageOwnershipSemaphores[i])
			orPanic(NewError(ret))
		}
	}
}

// DestroyBase destroys everything in the base-level context
func (c *ctxtBase) DestroyBase() {
	func() (err error) {
		checkErr(&err)
		if c.onCleanup != nil {
			err = c.onCleanup()
		}
		return
	}()

	for i := 0; i < c.frameLag; i++ {
		vulkan.DestroySemaphore(c.device, c.imageAcquiredSemaphores[i], nil)
		vulkan.DestroySemaphore(c.device, c.drawCompleteSemaphores[i], nil)
		if c.theGPU.HasSeparatePresentQueue() {
			vulkan.DestroySemaphore(c.device, c.imageOwnershipSemaphores[i], nil)
		}
	}
	for i := 0; i < len(c.swapchainImageResources); i++ {
		c.swapchainImageResources[i].Destroy(c.device, c.cmdPool)
	}
	c.swapchainImageResources = nil
	if c.swapchain != vulkan.NullSwapchain {
		vulkan.DestroySwapchain(c.device, c.swapchain, nil)
		c.swapchain = vulkan.NullSwapchain
	}
	vulkan.DestroyCommandPool(c.device, c.cmdPool, nil)
	if c.theGPU.HasSeparatePresentQueue() {
		vulkan.DestroyCommandPool(c.device, c.presentCmdPool, nil)
	}

	if c.surface != vulkan.NullSurface {
		vulkan.DestroySurface(c.theGPU.Instance(), c.surface, nil)
		c.surface = vulkan.NullSurface
	}

	c.theGPU = nil
}

func (c *ctxtBase) Device() vulkan.Device {
	return c.device
}

func (c *ctxtBase) GPU() oswin.GPU {
	return c.theGPU
}

func (c *ctxtBase) Surface() vulkan.Surface {
	return c.surface
}

func (c *ctxtBase) SetSurface(surf vulkan.Surface) {
	c.surface = surf
}

func (c *ctxtBase) CommandBuffer() vulkan.CommandBuffer {
	return c.cmd
}

func (c *ctxtBase) SwapchainDimensions() *SwapchainDimensions {
	return c.swapchainDimensions
}

func (c *ctxtBase) SwapchainImageResources() []*SwapchainImageResources {
	return c.swapchainImageResources
}

func (c *ctxtBase) SetOnPrepare(onPrepare func() error) {
	c.onPrepare = onPrepare
}

func (c *ctxtBase) SetOnCleanup(onCleanup func() error) {
	c.onCleanup = onCleanup
}

func (c *ctxtBase) SetOnInvalidate(onInvalidate func(imageIdx int) error) {
	c.onInvalidate = onInvalidate
}

func (c *ctxtBase) prepare(needCleanup bool) {
	vulkan.DeviceWaitIdle(c.device)

	if needCleanup {
		if c.onCleanup != nil {
			orPanic(c.onCleanup())
		}

		vulkan.DestroyCommandPool(c.device, c.cmdPool, nil)
		if c.theGPU.HasSeparatePresentQueue() {
			vulkan.DestroyCommandPool(c.device, c.presentCmdPool, nil)
		}
	}

	var cmdPool vulkan.CommandPool
	ret := vulkan.CreateCommandPool(c.device, &vulkan.CommandPoolCreateInfo{
		SType:            vulkan.StructureTypeCommandPoolCreateInfo,
		QueueFamilyIndex: c.theGPU.GraphicsQueueFamilyIndex(),
	}, nil, &cmdPool)
	orPanic(NewError(ret))
	c.cmdPool = cmdPool

	var cmd = make([]vulkan.CommandBuffer, 1)
	ret = vulkan.AllocateCommandBuffers(c.device, &vulkan.CommandBufferAllocateInfo{
		SType:              vulkan.StructureTypeCommandBufferAllocateInfo,
		CommandPool:        c.cmdPool,
		Level:              vulkan.CommandBufferLevelPrimary,
		CommandBufferCount: 1,
	}, cmd)
	orPanic(NewError(ret))
	c.cmd = cmd[0]

	ret = vulkan.BeginCommandBuffer(c.cmd, &vulkan.CommandBufferBeginInfo{
		SType: vulkan.StructureTypeCommandBufferBeginInfo,
	})
	orPanic(NewError(ret))

	for i := 0; i < len(c.swapchainImageResources); i++ {
		var cmd = make([]vulkan.CommandBuffer, 1)
		vulkan.AllocateCommandBuffers(c.device, &vulkan.CommandBufferAllocateInfo{
			SType:              vulkan.StructureTypeCommandBufferAllocateInfo,
			CommandPool:        c.cmdPool,
			Level:              vulkan.CommandBufferLevelPrimary,
			CommandBufferCount: 1,
		}, cmd)
		orPanic(NewError(ret))
		c.swapchainImageResources[i].cmd = cmd[0]
	}

	if c.theGPU.HasSeparatePresentQueue() {
		var cmdPool vulkan.CommandPool
		ret = vulkan.CreateCommandPool(c.device, &vulkan.CommandPoolCreateInfo{
			SType:            vulkan.StructureTypeCommandPoolCreateInfo,
			QueueFamilyIndex: c.theGPU.PresentQueueFamilyIndex(),
		}, nil, &cmdPool)
		orPanic(NewError(ret))
		c.presentCmdPool = cmdPool

		for i := 0; i < len(c.swapchainImageResources); i++ {
			var cmd = make([]vulkan.CommandBuffer, 1)
			ret = vulkan.AllocateCommandBuffers(c.device, &vulkan.CommandBufferAllocateInfo{
				SType:              vulkan.StructureTypeCommandBufferAllocateInfo,
				CommandPool:        c.presentCmdPool,
				Level:              vulkan.CommandBufferLevelPrimary,
				CommandBufferCount: 1,
			}, cmd)
			orPanic(NewError(ret))
			c.swapchainImageResources[i].graphicsToPresentCmd = cmd[0]

			c.swapchainImageResources[i].SetImageOwnership(
				c.theGPU.GraphicsQueueFamilyIndex(), c.theGPU.PresentQueueFamilyIndex())
		}
	}

	for i := 0; i < len(c.swapchainImageResources); i++ {
		var view vulkan.ImageView
		ret = vulkan.CreateImageView(c.device, &vulkan.ImageViewCreateInfo{
			SType:  vulkan.StructureTypeImageViewCreateInfo,
			Format: c.swapchainDimensions.Format,
			Components: vulkan.ComponentMapping{
				R: vulkan.ComponentSwizzleR,
				G: vulkan.ComponentSwizzleG,
				B: vulkan.ComponentSwizzleB,
				A: vulkan.ComponentSwizzleA,
			},
			SubresourceRange: vulkan.ImageSubresourceRange{
				AspectMask: vulkan.ImageAspectFlags(vulkan.ImageAspectColorBit),
				LevelCount: 1,
				LayerCount: 1,
			},
			ViewType: vulkan.ImageViewType2d,
			Image:    c.swapchainImageResources[i].image,
		}, nil, &view)
		orPanic(NewError(ret))
		c.swapchainImageResources[i].view = view
	}

	if c.onPrepare != nil {
		orPanic(c.onPrepare())
	}
	c.flushInitCmd()
}

func (c *ctxtBase) flushInitCmd() {
	if c.cmd == nil {
		return
	}
	ret := vulkan.EndCommandBuffer(c.cmd)
	orPanic(NewError(ret))

	var fence vulkan.Fence
	ret = vulkan.CreateFence(c.device, &vulkan.FenceCreateInfo{
		SType: vulkan.StructureTypeFenceCreateInfo,
	}, nil, &fence)
	orPanic(NewError(ret))

	cmdBufs := []vulkan.CommandBuffer{c.cmd}
	ret = vulkan.QueueSubmit(c.theGPU.GraphicsQueue(), 1, []vulkan.SubmitInfo{{
		SType:              vulkan.StructureTypeSubmitInfo,
		CommandBufferCount: 1,
		PCommandBuffers:    cmdBufs,
	}}, fence)
	orPanic(NewError(ret))

	ret = vulkan.WaitForFences(c.device, 1, []vulkan.Fence{fence}, vulkan.True, vulkan.MaxUint64)
	orPanic(NewError(ret))

	vulkan.FreeCommandBuffers(c.device, c.cmdPool, 1, cmdBufs)
	vulkan.DestroyFence(c.device, fence, nil)
	c.cmd = nil
}

func (c *ctxtBase) prepareSwapchain(gpu vulkan.PhysicalDevice, surface vulkan.Surface, dimensions *SwapchainDimensions) {
	// Read surface capabilities
	var surfaceCapabilities vulkan.SurfaceCapabilities
	ret := vulkan.GetPhysicalDeviceSurfaceCapabilities(gpu, surface, &surfaceCapabilities)
	orPanic(NewError(ret))
	surfaceCapabilities.Deref()

	// Get available surface pixel formats
	var formatCount uint32
	vulkan.GetPhysicalDeviceSurfaceFormats(gpu, surface, &formatCount, nil)
	formats := make([]vulkan.SurfaceFormat, formatCount)
	vulkan.GetPhysicalDeviceSurfaceFormats(gpu, surface, &formatCount, formats)

	// Select a proper surface format
	var format vulkan.SurfaceFormat
	if formatCount == 1 {
		formats[0].Deref()
		if formats[0].Format == vulkan.FormatUndefined {
			format = formats[0]
			format.Format = dimensions.Format
		} else {
			format = formats[0]
		}
	} else if formatCount == 0 {
		orPanic(errors.New("vulkan error: surface has no pixel formats"))
	} else {
		formats[0].Deref()
		// select the first one available
		format = formats[0]
	}

	// Setup swapchain parameters
	var swapchainSize vulkan.Extent2D
	surfaceCapabilities.CurrentExtent.Deref()
	if surfaceCapabilities.CurrentExtent.Width == vulkan.MaxUint32 {
		swapchainSize.Width = dimensions.Width
		swapchainSize.Height = dimensions.Height
	} else {
		swapchainSize = surfaceCapabilities.CurrentExtent
	}
	// The FIFO present mode is guaranteed by the spec to be supported
	// and to have no tearing.  It's a great default present mode to use.
	swapchainPresentMode := vulkan.PresentModeFifo

	// Determine the number of VkImage's to use in the swapchain.
	// Ideally, we desire to own 1 image at a time, the rest of the images can either be rendered to and/or
	// being queued up for display.
	desiredSwapchainImages := surfaceCapabilities.MinImageCount + 1
	if surfaceCapabilities.MaxImageCount > 0 && desiredSwapchainImages > surfaceCapabilities.MaxImageCount {
		// Application must settle for fewer images than desired.
		desiredSwapchainImages = surfaceCapabilities.MaxImageCount
	}

	// Figure out a suitable surface transform.
	var preTransform vulkan.SurfaceTransformFlagBits
	requiredTransforms := vulkan.SurfaceTransformIdentityBit
	supportedTransforms := surfaceCapabilities.SupportedTransforms
	if vulkan.SurfaceTransformFlagBits(supportedTransforms)&requiredTransforms != 0 {
		preTransform = requiredTransforms
	} else {
		preTransform = surfaceCapabilities.CurrentTransform
	}

	// Find a supported composite alpha mode - one of these is guaranteed to be set
	compositeAlpha := vulkan.CompositeAlphaOpaqueBit
	compositeAlphaFlags := []vulkan.CompositeAlphaFlagBits{
		vulkan.CompositeAlphaOpaqueBit,
		vulkan.CompositeAlphaPreMultipliedBit,
		vulkan.CompositeAlphaPostMultipliedBit,
		vulkan.CompositeAlphaInheritBit,
	}
	for i := 0; i < len(compositeAlphaFlags); i++ {
		alphaFlags := vulkan.CompositeAlphaFlags(compositeAlphaFlags[i])
		flagSupported := surfaceCapabilities.SupportedCompositeAlpha&alphaFlags != 0
		if flagSupported {
			compositeAlpha = compositeAlphaFlags[i]
			break
		}
	}

	// Create a swapchain
	var swapchain vulkan.Swapchain
	oldSwapchain := c.swapchain
	ret = vulkan.CreateSwapchain(c.device, &vulkan.SwapchainCreateInfo{
		SType:           vulkan.StructureTypeSwapchainCreateInfo,
		Surface:         surface,
		MinImageCount:   desiredSwapchainImages, // 1 - 3?
		ImageFormat:     format.Format,
		ImageColorSpace: format.ColorSpace,
		ImageExtent: vulkan.Extent2D{
			Width:  swapchainSize.Width,
			Height: swapchainSize.Height,
		},
		ImageUsage:       vulkan.ImageUsageFlags(vulkan.ImageUsageColorAttachmentBit),
		PreTransform:     preTransform,
		CompositeAlpha:   compositeAlpha,
		ImageArrayLayers: 1,
		ImageSharingMode: vulkan.SharingModeExclusive,
		PresentMode:      swapchainPresentMode,
		OldSwapchain:     oldSwapchain,
		Clipped:          vulkan.True,
	}, nil, &swapchain)
	orPanic(NewError(ret))
	if oldSwapchain != vulkan.NullSwapchain {
		vulkan.DestroySwapchain(c.device, oldSwapchain, nil)
	}
	c.swapchain = swapchain

	c.swapchainDimensions = &SwapchainDimensions{
		Width:  swapchainSize.Width,
		Height: swapchainSize.Height,
		Format: format.Format,
	}

	var imageCount uint32
	ret = vulkan.GetSwapchainImages(c.device, c.swapchain, &imageCount, nil)
	orPanic(NewError(ret))
	swapchainImages := make([]vulkan.Image, imageCount)
	ret = vulkan.GetSwapchainImages(c.device, c.swapchain, &imageCount, swapchainImages)
	orPanic(NewError(ret))
	for i := 0; i < len(c.swapchainImageResources); i++ {
		c.swapchainImageResources[i].Destroy(c.device, c.cmdPool)
	}
	c.swapchainImageResources = make([]*SwapchainImageResources, 0, imageCount)
	for i := 0; i < len(swapchainImages); i++ {
		c.swapchainImageResources = append(c.swapchainImageResources, &SwapchainImageResources{
			image: swapchainImages[i],
		})
	}
}

func (c *ctxtBase) AcquireNextImage() (imageIndex int, outdated bool, err error) {
	defer checkErr(&err)

	// Get the index of the next available swapchain image
	var idx uint32
	ret := vulkan.AcquireNextImage(c.device, c.swapchain, vulkan.MaxUint64,
		c.imageAcquiredSemaphores[c.frameIndex], vulkan.NullFence, &idx)
	imageIndex = int(idx)
	if c.onInvalidate != nil {
		orPanic(c.onInvalidate(imageIndex))
	}
	switch ret {
	case vulkan.ErrorOutOfDate:
		c.frameIndex++
		c.frameIndex = c.frameIndex % c.frameLag
		c.prepareSwapchain(c.theGPU.PhysicalDevice(),
			c.surface, c.SwapchainDimensions())
		c.prepare(true)
		outdated = true
		return
	case vulkan.Suboptimal, vulkan.Success:
	default:
		orPanic(NewError(ret))
	}

	graphicsQueue := c.theGPU.GraphicsQueue()
	var nullFence vulkan.Fence
	ret = vulkan.QueueSubmit(graphicsQueue, 1, []vulkan.SubmitInfo{{
		SType: vulkan.StructureTypeSubmitInfo,
		PWaitDstStageMask: []vulkan.PipelineStageFlags{
			vulkan.PipelineStageFlags(vulkan.PipelineStageColorAttachmentOutputBit),
		},
		WaitSemaphoreCount: 1,
		PWaitSemaphores: []vulkan.Semaphore{
			c.imageAcquiredSemaphores[c.frameIndex],
		},
		CommandBufferCount: 1,
		PCommandBuffers: []vulkan.CommandBuffer{
			c.swapchainImageResources[idx].cmd,
		},
		SignalSemaphoreCount: 1,
		PSignalSemaphores: []vulkan.Semaphore{
			c.drawCompleteSemaphores[c.frameIndex],
		},
	}}, nullFence)
	orPanic(NewError(ret))

	if c.theGPU.HasSeparatePresentQueue() {
		presentQueue := c.theGPU.PresentQueue()

		var nullFence vulkan.Fence
		ret = vulkan.QueueSubmit(presentQueue, 1, []vulkan.SubmitInfo{{
			SType: vulkan.StructureTypeSubmitInfo,
			PWaitDstStageMask: []vulkan.PipelineStageFlags{
				vulkan.PipelineStageFlags(vulkan.PipelineStageColorAttachmentOutputBit),
			},
			WaitSemaphoreCount: 1,
			PWaitSemaphores: []vulkan.Semaphore{
				c.imageAcquiredSemaphores[c.frameIndex],
			},
			CommandBufferCount: 1,
			PCommandBuffers: []vulkan.CommandBuffer{
				c.swapchainImageResources[idx].graphicsToPresentCmd,
			},
			SignalSemaphoreCount: 1,
			PSignalSemaphores: []vulkan.Semaphore{
				c.imageOwnershipSemaphores[c.frameIndex],
			},
		}}, nullFence)
		orPanic(NewError(ret))
	}
	return
}

func (c *ctxtBase) PresentImage(imageIdx int) (outdated bool, err error) {
	// If we are using separate queues we have to wait for image ownership,
	// otherwise wait for draw complete.
	var semaphore vulkan.Semaphore
	if c.theGPU.HasSeparatePresentQueue() {
		semaphore = c.imageOwnershipSemaphores[c.frameIndex]
	} else {
		semaphore = c.drawCompleteSemaphores[c.frameIndex]
	}
	presentQueue := c.theGPU.PresentQueue()
	ret := vulkan.QueuePresent(presentQueue, &vulkan.PresentInfo{
		SType:              vulkan.StructureTypePresentInfo,
		WaitSemaphoreCount: 1,
		PWaitSemaphores:    []vulkan.Semaphore{semaphore},
		SwapchainCount:     1,
		PSwapchains:        []vulkan.Swapchain{c.swapchain},
		PImageIndices:      []uint32{uint32(imageIdx)},
	})
	c.frameIndex++
	c.frameIndex = c.frameIndex % c.frameLag

	switch ret {
	case vulkan.ErrorOutOfDate:
		outdated = true
		return
	case vulkan.Suboptimal, vulkan.Success:
		return
	default:
		err = NewError(ret)
		return
	}
}
