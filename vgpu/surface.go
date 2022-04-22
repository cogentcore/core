// Copyright (c) 2022, The Emergent Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// This is initially adapted from https://github.com/vulkan-go/asche
// Copyright © 2017 Maxim Kupriianov <max@kc.vc>, under the MIT License

package vgpu

import (
	"errors"

	vk "github.com/vulkan-go/vulkan"
)

// SwapchainDims describes the size and format of the swapchain.
type SwapchainDims struct {
	Width  uint32    `desc:"Width of the swapchain -- should be window width"`
	Height uint32    `desc:"Height of the swapchain  -- should be window height"`
	Format vk.Format `desc:"Format is the pixel format of the swapchain."`
}

func (sd *SwapchainDims) Set(w, h uint32, ft vk.Format) {
	sd.Width = w
	sd.Height = h
	sd.Format = ft
}

type ImageResources struct {
	Image vk.Image
	View  vk.ImageView
	// CmdBuff              vk.CommandBuffer
	GraphicsToPresentCmd vk.CommandBuffer
	// Framebuffer   vk.Framebuffer
	// DescriptorSet vk.DescriptorSet
	//
	// UniformBuffer vk.Buffer
	// UniformMemory vk.DeviceMemory
}

func (s *ImageResources) Destroy(dev vk.Device, CmdPool ...vk.CommandPool) {
	// vk.DestroyFramebuffer(dev, s.Framebuffer, nil)
	// vk.DestroyImageView(dev, s.View, nil)
	// if len(CmdPool) > 0 {
	// 	vk.FreeCommandBuffers(dev, CmdPool[0], 1, []vk.CommandBuffer{
	// 		s.CmdBuff,
	// 	})
	// }
	// vk.DestroyBuffer(dev, s.UniformBuffer, nil)
	// vk.FreeMemory(dev, s.UniformMemory, nil)
}

func (s *ImageResources) SetImageOwnership(graphicsQueueIndex, presentQueueIndex uint32) {
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

// Surface manages the swapchain for presenting images to a window surface
type Surface struct {
	GPU     *GPU
	Surface vk.Surface
	Device  Device `desc:"device for this surface -- each present surface has its own device"`
	CmdPool CmdPool

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

	Swapchain vk.Swapchain
	Dims      SwapchainDims `desc:"has the current swapchain dimensions, including pixel format."`

	// ImageResources exposes the swapchain initialized image resources.
	ImageResources []*ImageResources

	FrameLag int

	ImageAcquiredSemaphores  []vk.Semaphore
	DrawCompleteSemaphores   []vk.Semaphore
	ImageOwnershipSemaphores []vk.Semaphore

	FrameIndex int
}

func (sf *Surface) Defaults() {
	sf.FrameLag = 2
	sf.Dims.Set(1024, 768, vk.FormatB8g8r8a8Unorm) // todo: get from window
}

// Init initializes the device for the surface
func (sf *Surface) Init(gp *GPU) error {
	sf.GPU = gp
	// Get queue family properties
	var queueCount uint32
	vk.GetPhysicalDeviceQueueFamilyProperties(sf.GPU.GPU, &queueCount, nil)
	queueProperties := make([]vk.QueueFamilyProperties, queueCount)
	vk.GetPhysicalDeviceQueueFamilyProperties(sf.GPU.GPU, &queueCount, queueProperties)
	if queueCount == 0 { // probably should try another GPU
		return errors.New("vulkan error: no queue families found on GPU 0")
	}

	// Find a suitable queue family for the target Vulkan mode
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
	sf.PrepareSwapchain()
	return nil
}

func (sf *Surface) PrepareSwapchain() {
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
			format.Format = sf.Dims.Format
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
		swapchainSize.Width = sf.Dims.Width
		swapchainSize.Height = sf.Dims.Height
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
	ret = vk.CreateSwapchain(sf.Device.Device, &vk.SwapchainCreateInfo{
		SType:           vk.StructureTypeSwapchainCreateInfo,
		Surface:         sf.Surface,
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
		vk.DestroySwapchain(sf.Device.Device, oldSwapchain, nil)
	}
	sf.Swapchain = swapchain
	sf.Dims.Set(swapchainSize.Width, swapchainSize.Width, format.Format)

	var imageCount uint32
	ret = vk.GetSwapchainImages(sf.Device.Device, sf.Swapchain, &imageCount, nil)
	IfPanic(NewError(ret))
	swapchainImages := make([]vk.Image, imageCount)
	ret = vk.GetSwapchainImages(sf.Device.Device, sf.Swapchain, &imageCount, swapchainImages)
	IfPanic(NewError(ret))
	for i := 0; i < len(sf.ImageResources); i++ {
		sf.ImageResources[i].Destroy(sf.Device.Device, sf.CmdPool.Pool)
	}
	sf.ImageResources = make([]*ImageResources, 0, imageCount)
	for i := 0; i < len(swapchainImages); i++ {
		sf.ImageResources = append(sf.ImageResources, &ImageResources{
			Image: swapchainImages[i],
		})
	}

	sf.Prepare(false)
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
		ret := vk.CreateSemaphore(sf.Device.Device, semaphoreCreateInfo, nil, &sf.ImageAcquiredSemaphores[i])
		IfPanic(NewError(ret))
		ret = vk.CreateSemaphore(sf.Device.Device, semaphoreCreateInfo, nil, &sf.DrawCompleteSemaphores[i])
		IfPanic(NewError(ret))
		ret = vk.CreateSemaphore(sf.Device.Device, semaphoreCreateInfo, nil, &sf.ImageOwnershipSemaphores[i])
		IfPanic(NewError(ret))
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
		vk.DestroySemaphore(sf.Device.Device, sf.ImageAcquiredSemaphores[i], nil)
		vk.DestroySemaphore(sf.Device.Device, sf.DrawCompleteSemaphores[i], nil)
		vk.DestroySemaphore(sf.Device.Device, sf.ImageOwnershipSemaphores[i], nil)
	}
	for i := 0; i < len(sf.ImageResources); i++ {
		sf.ImageResources[i].Destroy(sf.Device.Device, sf.CmdPool.Pool)
	}
	sf.ImageResources = nil
	if sf.Swapchain != vk.NullSwapchain {
		vk.DestroySwapchain(sf.Device.Device, sf.Swapchain, nil)
		sf.Swapchain = vk.NullSwapchain
	}
	vk.DestroyCommandPool(sf.Device.Device, sf.CmdPool.Pool, nil)
	if sf.Surface != vk.NullSurface {
		vk.DestroySurface(sf.GPU.Instance, sf.Surface, nil)
		sf.Surface = vk.NullSurface
	}
	if sf.Device.Device != nil {
		vk.DeviceWaitIdle(sf.Device.Device)
		vk.DestroyDevice(sf.Device.Device, nil)
		sf.Device.Device = nil
	}
	sf.GPU = nil
}

func (sf *Surface) Prepare(needCleanup bool) {
	vk.DeviceWaitIdle(sf.Device.Device)

	if needCleanup {
		if sf.OnCleanup != nil {
			IfPanic(sf.OnCleanup())
		}
		sf.CmdPool.Destroy(&sf.Device)
		var presentQueue vk.Queue
		vk.GetDeviceQueue(sf.Device.Device, sf.Device.QueueIndex, 0, &presentQueue)
		sf.Device.Queue = presentQueue
	}

	sf.CmdPool.Init(&sf.Device, 0)

	for i := 0; i < len(sf.ImageResources); i++ {
		var cmdBuff = make([]vk.CommandBuffer, 1)
		ret := vk.AllocateCommandBuffers(sf.Device.Device, &vk.CommandBufferAllocateInfo{
			SType:              vk.StructureTypeCommandBufferAllocateInfo,
			CommandPool:        sf.CmdPool.Pool,
			Level:              vk.CommandBufferLevelPrimary,
			CommandBufferCount: 1,
		}, cmdBuff)
		IfPanic(NewError(ret))
		sf.ImageResources[i].GraphicsToPresentCmd = cmdBuff[0]

		sf.ImageResources[i].SetImageOwnership( // this does the transfer?
			sf.GPU.Device.QueueIndex, sf.Device.QueueIndex)
	}

	for i := 0; i < len(sf.ImageResources); i++ {
		var view vk.ImageView
		ret := vk.CreateImageView(sf.Device.Device, &vk.ImageViewCreateInfo{
			SType:  vk.StructureTypeImageViewCreateInfo,
			Format: sf.Dims.Format,
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
	defer CheckErr(&err)

	// Get the index of the next available swapchain image
	var idx uint32
	ret := vk.AcquireNextImage(sf.Device.Device, sf.Swapchain, vk.MaxUint64,
		sf.ImageAcquiredSemaphores[sf.FrameIndex], vk.NullFence, &idx)
	imageIndex = int(idx)
	if sf.OnInvalidate != nil {
		IfPanic(sf.OnInvalidate(imageIndex))
	}
	switch ret {
	case vk.ErrorOutOfDate:
		sf.FrameIndex++
		sf.FrameIndex = sf.FrameIndex % sf.FrameLag
		sf.PrepareSwapchain()
		sf.Prepare(true)
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
			sf.ImageResources[idx].GraphicsToPresentCmd,
		},
		SignalSemaphoreCount: 1,
		PSignalSemaphores: []vk.Semaphore{
			sf.ImageOwnershipSemaphores[sf.FrameIndex],
		},
	}}, nullFence)
	IfPanic(NewError(ret))
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
