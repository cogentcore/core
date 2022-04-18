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

// SwapchainDims describes the size and format of the swapchain.
type SwapchainDims struct {
	// Width of the swapchain.
	Width uint32
	// Height of the swapchain.
	Height uint32
	// Format is the pixel format of the swapchain.
	Format vk.Format
}

// Surface manages the swapchain for presenting images to a window surface
type Surface struct {
	GPU     *GPU
	Surface vk.Surface
	Device  vk.Device `desc:"device for this surface -- each present surface has its own device"`

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

	PresentQueueIndex uint32
	PresentQueue      vk.Queue

	GraphicsQueueIndex uint32 // todo: unclear if we need this
	GraphicsQueue      vk.Queue

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

func (sf *Surface) Defaults() {
	sf.FrameLag = 3
}

// Init initializes the device for the surface
func (sf *Surface) Init(gp *GPU) {
	sf.GPU = gp
	// Get queue family properties
	var queueCount uint32
	vk.GetPhysicalDeviceQueueFamilyProperties(sf.GPU.Gpu, &queueCount, nil)
	queueProperties := make([]vk.QueueFamilyProperties, queueCount)
	vk.GetPhysicalDeviceQueueFamilyProperties(sf.GPU.Gpu, &queueCount, queueProperties)
	if queueCount == 0 { // probably should try another GPU
		return nil, errors.New("vulkan error: no queue families found on GPU 0")
	}

	// Find a suitable queue family for the target Vulkan mode
	found := false
	for i := uint32(0); i < queueCount; i++ {
		var supportsPresent vk.Bool32
		vk.GetPhysicalDeviceSurfaceSupport(sf.Gpu, i, sf.Surface, &supportsPresent)
		if supportsPresent.B() {
			sf.PresentQueueIndex = i
			found = true
			break
		}
		// if mode.Has(Compute) {
		// 	required |= vk.QueueFlags(vk.QueueComputeBit)
		// }
		// if mode.Has(Graphics) {
		// 	required |= vk.QueueFlags(vk.QueueGraphicsBit)
		// }
		vk.GetPhysicalDeviceSurfaceSupport(sf.Gpu, i, sf.Surface, &supportsPresent)
		if supportsPresent.B() {
			sf.PresentQueueIndex = i
			break
		}
		// queueProperties[i].Deref()
		// if queueProperties[i].QueueFlags&required != 0 {
		// 	if !needsPresent || (needsPresent && supportsPresent.B()) {
		// 		sf.GraphicsQueueIndex = i
		// 		graphicsFound = true
		// 		break
		// 	} else if needsPresent {
		// 		sf.GraphicsQueueIndex = i
		// 		graphicsFound = true
		// 		// need present, but this one doesn't support
		// 		// continue lookup
		// 	}
		// }
	}
	if !found {
		err := errors.New("Surface vulkan error: could not found queue with present capabilities")
		return nil, err
	}

	queueInfos = append(queueInfos, vk.DeviceQueueCreateInfo{
		SType:            vk.StructureTypeDeviceQueueCreateInfo,
		QueueFamilyIndex: sf.PresentQueueIndex,
		QueueCount:       1,
		PQueuePriorities: []float32{1.0},
	})

	var device vk.Device
	ret = vk.CreateDevice(sf.Gpu.Gpu, &vk.DeviceCreateInfo{
		SType:                   vk.StructureTypeDeviceCreateInfo,
		QueueCreateInfoCount:    uint32(len(queueInfos)),
		PQueueCreateInfos:       queueInfos,
		EnabledExtensionCount:   uint32(len(deviceExts)),
		PpEnabledExtensionNames: deviceExts,
		EnabledLayerCount:       uint32(len(validationLayers)),
		PpEnabledLayerNames:     validationLayers,
	}, nil, &device)
	IfPanic(NewError(ret))
	sf.Device = device

	var queue vk.Queue
	vk.GetDeviceQueue(sf.Device, sf.PresentQueueIndex, 0, &queue)
	sf.PresentQueue = queue

	return sf, nil
}

func (sf *Surface) PreparePresent() {
	// Create semaphores to synchronize acquiring presentable buffers before
	// rendering and waiting for drawing to be complete before presenting
	semaphoreCreateInfo := &vk.SemaphoreCreateInfo{
		SType: vk.StructureTypeSemaphoreCreateInfo,
	}
	sf.ImageAcquiredSemaphores = make([]vk.Semaphore, sf.FrameLag)
	sf.DrawCompleteSemaphores = make([]vk.Semaphore, sf.FrameLag)
	sf.ImageOwnershipSemaphores = make([]vk.Semaphore, sf.FrameLag)
	for i := 0; i < sf.FrameLag; i++ {
		ret := vk.CreateSemaphore(sf.Device, semaphoreCreateInfo, nil, &sf.ImageAcquiredSemaphores[i])
		IfPanic(NewError(ret))
		ret = vk.CreateSemaphore(sf.Device, semaphoreCreateInfo, nil, &sf.DrawCompleteSemaphores[i])
		IfPanic(NewError(ret))
		if sf.GPU.HasSeparatePresentQueue() {
			ret = vk.CreateSemaphore(sf.Device, semaphoreCreateInfo, nil, &sf.ImageOwnershipSemaphores[i])
			IfPanic(NewError(ret))
		}
	}
}

func (sf *Surface) Destroy() {
	func() (err error) {
		CheckErr(&err)
		if sf.OnCleanup != nil {
			err = sf.OnCleanup()
		}
		return
	}()

	for i := 0; i < sf.FrameLag; i++ {
		vk.DestroySemaphore(sf.Device, sf.ImageAcquiredSemaphores[i], nil)
		vk.DestroySemaphore(sf.Device, sf.DrawCompleteSemaphores[i], nil)
		if sf.GPU.HasSeparatePresentQueue() {
			vk.DestroySemaphore(sf.Device, sf.ImageOwnershipSemaphores[i], nil)
		}
	}
	for i := 0; i < len(sf.ImageResources); i++ {
		sf.ImageResources[i].Destroy(sf.Device, sf.CmdPool)
	}
	sf.ImageResources = nil
	if sf.Swapchain != vk.NullSwapchain {
		vk.DestroySwapchain(sf.Device, sf.Swapchain, nil)
		sf.Swapchain = vk.NullSwapchain
	}
	vk.DestroyCommandPool(sf.Device, sf.CmdPool, nil)
	if sf.GPU.HasSeparatePresentQueue() {
		vk.DestroyCommandPool(sf.Device, sf.PresentCmdPool, nil)
	}
	if sf.Surface != vk.NullSurface {
		vk.DestroySurface(sf.GPU.Instance, sf.Surface, nil)
		sf.Surface = vk.NullSurface
	}
	if sf.Device != nil {
		vk.DeviceWaitIdle(sf.Device)
		vk.DestroyDevice(sf.Device, nil)
		sf.Device = nil
	}
	sf.GPU = nil
}

func (sf *Surface) Prepare(needCleanup bool) {
	vk.DeviceWaitIdle(sf.Device)

	if needCleanup {
		if sf.OnCleanup != nil {
			IfPanic(sf.OnCleanup())
		}

		vk.DestroyCommandPool(sf.Device, sf.CmdPool, nil)
		if sf.GPU.HasSeparatePresentQueue() {
			vk.DestroyCommandPool(sf.Device, sf.PresentCmdPool, nil)
		}
	}

	if sf.GPU.HasSeparatePresentQueue() {
		var presentQueue vk.Queue
		vk.GetDeviceQueue(sf.Device, sf.PresentQueueIndex, 0, &presentQueue)
		sf.PresQueue = presentQueue
	}

	var CmdPool vk.CommandPool
	ret := vk.CreateCommandPool(sf.Device, &vk.CommandPoolCreateInfo{
		SType:            vk.StructureTypeCommandPoolCreateInfo,
		QueueFamilyIndex: sf.GPU.GraphicsQueueIndex,
	}, nil, &CmdPool)
	IfPanic(NewError(ret))
	sf.CmdPool = CmdPool

	var CmdBuff = make([]vk.CommandBuffer, 1)
	ret = vk.AllocateCommandBuffers(sf.Device, &vk.CommandBufferAllocateInfo{
		SType:              vk.StructureTypeCommandBufferAllocateInfo,
		CommandPool:        sf.CmdPool,
		Level:              vk.CommandBufferLevelPrimary,
		CommandBufferCount: 1,
	}, CmdBuff)
	IfPanic(NewError(ret))
	sf.CmdBuff = CmdBuff[0]

	ret = vk.BeginCommandBuffer(sf.CmdBuff, &vk.CommandBufferBeginInfo{
		SType: vk.StructureTypeCommandBufferBeginInfo,
	})
	IfPanic(NewError(ret))

	for i := 0; i < len(sf.ImageResources); i++ {
		var CmdBuff = make([]vk.CommandBuffer, 1)
		vk.AllocateCommandBuffers(sf.Device, &vk.CommandBufferAllocateInfo{
			SType:              vk.StructureTypeCommandBufferAllocateInfo,
			CommandPool:        sf.CmdPool,
			Level:              vk.CommandBufferLevelPrimary,
			CommandBufferCount: 1,
		}, CmdBuff)
		IfPanic(NewError(ret))
		sf.ImageResources[i].CmdBuff = CmdBuff[0]
	}

	if sf.GPU.HasSeparatePresentQueue() {
		var CmdPool vk.CommandPool
		ret = vk.CreateCommandPool(sf.Device, &vk.CommandPoolCreateInfo{
			SType:            vk.StructureTypeCommandPoolCreateInfo,
			QueueFamilyIndex: sf.GPU.PresentQueueIndex,
		}, nil, &CmdPool)
		IfPanic(NewError(ret))
		sf.PresentCmdPool = CmdPool

		for i := 0; i < len(sf.ImageResources); i++ {
			var CmdBuff = make([]vk.CommandBuffer, 1)
			ret = vk.AllocateCommandBuffers(sf.Device, &vk.CommandBufferAllocateInfo{
				SType:              vk.StructureTypeCommandBufferAllocateInfo,
				CommandPool:        sf.PresentCmdPool,
				Level:              vk.CommandBufferLevelPrimary,
				CommandBufferCount: 1,
			}, CmdBuff)
			IfPanic(NewError(ret))
			sf.ImageResources[i].GraphicsToPresentCmd = CmdBuff[0]

			sf.ImageResources[i].SetImageOwnership(
				sf.GPU.GraphicsQueueIndex, sf.GPU.PresentQueueIndex)
		}
	}

	for i := 0; i < len(sf.ImageResources); i++ {
		var view vk.ImageView
		ret = vk.CreateImageView(sf.Device, &vk.ImageViewCreateInfo{
			SType:  vk.StructureTypeImageViewCreateInfo,
			Format: sf.SwapchainDims.Format,
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
			Image:    sf.ImageResources[i].Image,
		}, nil, &view)
		IfPanic(NewError(ret))
		sf.ImageResources[i].View = view
	}

	if sf.OnPrepare != nil {
		IfPanic(sf.OnPrepare())
	}
	sf.FlushInitCmd()

	dims := &SwapchainDims{
		// some default preferences here
		Width: 640, Height: 480,
		Format: vk.FormatB8g8r8a8Unorm,
	}
	// if iface, ok := app.(AppSwapchainDims); ok {
	// 	dimensions = iface.SwapchainDims()
	// }
	sf.PrepareSwapchain(sf.Gpu, sf.Surface, dims)
}

func (sf *Surface) FlushInitCmd() {
	if sf.CmdBuff == nil {
		return
	}
	ret := vk.EndCommandBuffer(sf.CmdBuff)
	IfPanic(NewError(ret))

	var fence vk.Fence
	ret = vk.CreateFence(sf.Device, &vk.FenceCreateInfo{
		SType: vk.StructureTypeFenceCreateInfo,
	}, nil, &fence)
	IfPanic(NewError(ret))

	cmdBufs := []vk.CommandBuffer{sf.CmdBuff}
	ret = vk.QueueSubmit(sf.GPU.GraphicsQueue, 1, []vk.SubmitInfo{{
		SType:              vk.StructureTypeSubmitInfo,
		CommandBufferCount: 1,
		PCommandBuffers:    cmdBufs,
	}}, fence)
	IfPanic(NewError(ret))

	ret = vk.WaitForFences(sf.Device, 1, []vk.Fence{fence}, vk.True, vk.MaxUint64)
	IfPanic(NewError(ret))

	vk.FreeCommandBuffers(sf.Device, sf.CmdPool, 1, cmdBufs)
	vk.DestroyFence(sf.Device, fence, nil)
	sf.CmdBuff = nil
}

func (sf *Surface) PrepareSwapchain(gpu vk.PhysicalDevice, surface vk.Surface, dims *SwapchainDims) {

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
			format.Format = dims.Format
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
		swapchainSize.Width = dims.Width
		swapchainSize.Height = dims.Height
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
	oldSwapchain := sf.Swapchain
	ret = vk.CreateSwapchain(sf.Device, &vk.SwapchainCreateInfo{
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
		vk.DestroySwapchain(sf.Device, oldSwapchain, nil)
	}
	sf.Swapchain = swapchain

	sf.SwapchainDims = &SwapchainDims{
		Width:  swapchainSize.Width,
		Height: swapchainSize.Height,
		Format: format.Format,
	}

	var imageCount uint32
	ret = vk.GetSwapchainImages(sf.Device, sf.Swapchain, &imageCount, nil)
	IfPanic(NewError(ret))
	swapchainImages := make([]vk.Image, imageCount)
	ret = vk.GetSwapchainImages(sf.Device, sf.Swapchain, &imageCount, swapchainImages)
	IfPanic(NewError(ret))
	for i := 0; i < len(sf.ImageResources); i++ {
		sf.ImageResources[i].Destroy(sf.Device, sf.CmdPool)
	}
	sf.ImageResources = make([]*ImageResources, 0, imageCount)
	for i := 0; i < len(swapchainImages); i++ {
		sf.ImageResources = append(sf.ImageResources, &ImageResources{
			Image: swapchainImages[i],
		})
	}
}

func (sf *Surface) AcquireNextImage() (imageIndex int, outdated bool, err error) {
	defer CheckErr(&err)

	// Get the index of the next available swapchain image
	var idx uint32
	ret := vk.AcquireNextImage(sf.Device, sf.Swapchain, vk.MaxUint64,
		sf.ImageAcquiredSemaphores[sf.FrameIndex], vk.NullFence, &idx)
	imageIndex = int(idx)
	if sf.OnInvalidate != nil {
		IfPanic(sf.OnInvalidate(imageIndex))
	}
	switch ret {
	case vk.ErrorOutOfDate:
		sf.FrameIndex++
		sf.FrameIndex = sf.FrameIndex % sf.FrameLag
		sf.PrepareSwapchain(sf.GPU.Gpu, sf.GPU.Surface, sf.SwapchainDims)
		sf.Prepare(true)
		outdated = true
		return
	case vk.Suboptimal, vk.Success:
	default:
		IfPanic(NewError(ret))
	}

	graphicsQueue := sf.GPU.GraphicsQueue
	var nullFence vk.Fence
	ret = vk.QueueSubmit(graphicsQueue, 1, []vk.SubmitInfo{{
		SType: vk.StructureTypeSubmitInfo,
		PWaitDstStageMask: []vk.SurfaceStageFlags{
			vk.SurfaceStageFlags(vk.SurfaceStageColorAttachmentOutputBit),
		},
		WaitSemaphoreCount: 1,
		PWaitSemaphores: []vk.Semaphore{
			sf.ImageAcquiredSemaphores[sf.FrameIndex],
		},
		CommandBufferCount: 1,
		PCommandBuffers: []vk.CommandBuffer{
			sf.ImageResources[idx].CmdBuff,
		},
		SignalSemaphoreCount: 1,
		PSignalSemaphores: []vk.Semaphore{
			sf.DrawCompleteSemaphores[sf.FrameIndex],
		},
	}}, nullFence)
	IfPanic(NewError(ret))

	if sf.GPU.HasSeparatePresentQueue() {
		presentQueue := sf.GPU.PresentQueue()

		var nullFence vk.Fence
		ret = vk.QueueSubmit(presentQueue, 1, []vk.SubmitInfo{{
			SType: vk.StructureTypeSubmitInfo,
			PWaitDstStageMask: []vk.SurfaceStageFlags{
				vk.SurfaceStageFlags(vk.SurfaceStageColorAttachmentOutputBit),
			},
			WaitSemaphoreCount: 1,
			PWaitSemaphores: []vk.Semaphore{
				sf.ImageAcquiredSemaphores[sf.FrameIndex],
			},
			CommandBufferCount: 1,
			PCommandBuffers: []vk.CommandBuffer{
				sf.ImageResources[idx].GraphicsToPresentCmd,
			},
			SignalSemaphoreCount: 1,
			PSignalSemaphores: []vk.Semaphore{
				sf.ImageOwnershipSemaphores[sf.FrameIndex],
			},
		}}, nullFence)
		IfPanic(NewError(ret))
	}
	return
}

func (sf *Surface) PresentImage(imageIdx int) (outdated bool, err error) {
	// If we are using separate queues we have to wait for image ownership,
	// otherwise wait for draw complete.
	var semaphore vk.Semaphore
	if sf.GPU.HasSeparatePresentQueue() {
		semaphore = sf.ImageOwnershipSemaphores[sf.FrameIndex]
	} else {
		semaphore = sf.DrawCompleteSemaphores[sf.FrameIndex]
	}
	presentQueue := sf.GPU.PresentQueue()
	ret := vk.QueuePresent(presentQueue, &vk.PresentInfo{
		SType:              vk.StructureTypePresentInfo,
		WaitSemaphoreCount: 1,
		PWaitSemaphores:    []vk.Semaphore{semaphore},
		SwapchainCount:     1,
		PSwapchains:        []vk.Swapchain{sf.Swapchain},
		PImageIndices:      []uint32{uint32(imageIdx)},
	})
	sf.FrameIndex++
	sf.FrameIndex = sf.FrameIndex % sf.FrameLag

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
