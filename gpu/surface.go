// Copyright (c) 2022, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// This is initially adapted from https://github.com/WebGPU-go/asche
// Copyright © 2017 Maxim Kupriianov <max@kc.vc>, under the MIT License

package gpu

import (
	"cogentcore.org/core/base/errors"
	"github.com/rajveermalviya/go-webgpu/wgpu"
)

// Surface manages the physical device for the visible image
// of a window surface, and the swapchain for presenting images.
type Surface struct {

	// pointer to gpu device, for convenience.
	GPU *GPU

	// Device for this surface, which we own.
	// Each window surface has its own device, configured for that surface.
	Device *Device

	// Render for this Surface, typically from a System.
	Render *Render

	// Format has the current swapchain image format and dimensions.
	// the Size values here are definitive for the target size of the surface.
	Format TextureFormat

	// ordered list of surface formats to select.
	DesiredFormats []wgpu.TextureFormat

	// number of frames to maintain in the swapchain
	// e.g., 2 = double-buffering, 3 = triple-buffering.
	// Initially set to a requested amount, and after Init reflects actual number
	// NFrames int

	// Framebuffers representing the visible Texture owned
	// by the Surface. we iterate through these in rendering subsequent frames
	// Frames []*Framebuffer

	// WebGPU handle for surface
	surface *wgpu.Surface `display:"-"`

	swapChainConfig *wgpu.SwapChainDescriptor

	// WebGPU handle for swapchain
	swapChain *wgpu.SwapChain `display:"-"`

	// semaphore used internally for waiting on acquisition of next frame
	// TextureAcquired vk.Semaphore `display:"-"`

	// semaphore that surface user can wait on, will be activated
	// when image has been acquired in AcquireNextFrame method
	// RenderDone vk.Semaphore `display:"-"`

	// fence for rendering command running
	// RenderFence vk.Fence `display:"-"`

	// NeedsConfig is whether the surface needs to be configured again
	// without freeing the swapchain.
	// This is set internally to allow for correct recovery
	// from sudden minimization events that are
	// only detected at the point of swapchain reconfiguration.
	NeedsConfig bool
}

// NewSurface returns a new surface initialized for given GPU and WebGPU
// Surface handle, obtained from a valid window.
func NewSurface(gp *GPU, wsurf *wgpu.Surface, width, height int) *Surface {
	sf := &Surface{}
	sf.Defaults()
	sf.Init(gp, wsurf, width, height)
	return sf
}

func (sf *Surface) Defaults() {
	// sf.NFrames = 3 // requested, will be updated with actual
	sf.Format.Defaults()
	sf.Format.Set(1024, 768, wgpu.TextureFormatRGBA8UnormSrgb)
	sf.Format.SetMultisample(4) // good default
	sf.DesiredFormats = []wgpu.TextureFormat{
		wgpu.TextureFormatRGBA8UnormSrgb,
		// wgpu.TextureFormatR8g8b8a8Unorm, // these def too dark
		// wgpu.TextureFormatB8g8r8a8Unorm,
	}
}

// Init initializes the device and all other resources for the surface
// based on the WebGPU surface handle which must be obtained from the
// OS-specific window, created first (e.g., via glfw)
func (sf *Surface) Init(gp *GPU, ws *wgpu.Surface, width, height int) error {
	sf.GPU = gp
	sf.surface = ws
	dev, err := gp.NewDevice() // surface owns this device
	if errors.Log(err) != nil {
		return err
	}
	sf.Device = dev
	sf.Format.Format = ws.GetPreferredFormat(gp.GPU)
	sf.Format.Samples = 4
	sf.Format.Size.X = width
	sf.Format.Size.Y = height
	sf.ConfigSwapChain()
	return nil
}

func (sf *Surface) AcquireNextTexture() (*wgpu.TextureView, error) {
	return sf.swapChain.GetCurrentTextureView()
}

func (sf *Surface) SubmitRender(cmd *wgpu.CommandEncoder) error {
	cmdBuffer, err := cmd.Finish(nil)
	if errors.Log(err) != nil {
		return err
	}
	defer cmdBuffer.Release()
	sf.Device.Queue.Submit(cmdBuffer)
	return nil
}

func (sf *Surface) Present() {
	sf.swapChain.Present()
}

// ConfigSwapChain configures the swapchain for surface.
// This assumes that all existing items have been destroyed.
func (sf *Surface) ConfigSwapChain() error {
	caps := sf.surface.GetCapabilities(sf.GPU.GPU)

	// fmt.Println(reflectx.StringJSON(caps))

	sf.swapChainConfig = &wgpu.SwapChainDescriptor{
		Usage:       wgpu.TextureUsageRenderAttachment,
		Format:      caps.Formats[0],
		Width:       uint32(sf.Format.Size.X),
		Height:      uint32(sf.Format.Size.Y),
		PresentMode: wgpu.PresentModeFifo,
		AlphaMode:   caps.AlphaModes[0],
	}

	sf.Format.Format = caps.Formats[0]

	sc, err := sf.Device.Device.CreateSwapChain(sf.surface, sf.swapChainConfig)
	if err != nil {
		return err
	}
	sf.swapChain = sc
	return nil

	// // Read sf.Surface capabilities
	// var surfaceCapabilities vk.SurfaceCapabilities
	// ret := vk.GetPhysicalDeviceSurfaceCapabilities(sf.GPU.GPU, sf.Surface, &surfaceCapabilities)
	// IfPanic(NewError(ret))
	// surfaceCapabilities.Deref()
	//
	// // Get available surface pixel formats
	// var formatCount uint32
	// vk.GetPhysicalDeviceSurfaceFormats(sf.GPU.GPU, sf.Surface, &formatCount, nil)
	// formats := make([]vk.SurfaceFormat, formatCount)
	// vk.GetPhysicalDeviceSurfaceFormats(sf.GPU.GPU, sf.Surface, &formatCount, formats)
	//
	// // Select a proper surface format
	// var format vk.SurfaceFormat
	// if formatCount == 1 {
	// 	formats[0].Deref()
	// 	if formats[0].Format == wgpu.TextureFormatUndefined {
	// 		format = formats[0]
	// 		format.Format = sf.Format.Format
	// 	} else {
	// 		format = formats[0]
	// 	}
	// } else if formatCount == 0 {
	// 	IfPanic(errors.New("WebGPU error: surface has no pixel formats"))
	// } else {
	// 	got := false
	// 	for _, df := range sf.DesiredFormats {
	// 		for _, ft := range formats {
	// 			ft.Deref()
	// 			if ft.Format == df {
	// 				format = ft
	// 				got = true
	// 				break
	// 			}
	// 		}
	// 		if got {
	// 			break
	// 		}
	// 	}
	// 	if !got {
	// 		formats[0].Deref()
	// 		format = formats[0]
	// 		if Debug {
	// 			dfs := make([]string, len(sf.DesiredFormats))
	// 			for i, df := range sf.DesiredFormats {
	// 				dfs[i] = TextureFormatNames[df]
	// 			}
	// 			fmt.Printf("gpu.Surface:Init unable to find desired format: %v, using first one: %s\n", dfs, TextureFormatNames[format.Format])
	// 		}
	// 	}
	// }

	// Setup swapchain parameters
	// var swapchainSize vk.Extent2D
	// surfaceCapabilities.CurrentExtent.Deref()
	// if surfaceCapabilities.CurrentExtent.Width == vk.MaxUint32 {
	// 	w, h := sf.Format.Size32()
	// 	swapchainSize.Width = w
	// 	swapchainSize.Height = h
	// } else {
	// 	swapchainSize = surfaceCapabilities.CurrentExtent
	// }
	//
	// if swapchainSize.Width == 0 || swapchainSize.Height == 0 {
	// 	return false
	// }

	// The FIFO present mode is guaranteed by the spec to be supported
	// and to have no tearing.  It's a great default present mode to use.
	// swapchainPresentMode := vk.PresentModeFifo

	// Determine the number of VkTexture's to use in the swapchain.
	// Ideally, we desire to own 1 image at a time, the rest of the images can either be rendered to and/or
	// being queued up for display.
	// desiredSwapchainTextures := uint32(sf.NFrames)
	// if surfaceCapabilities.MaxTextureCount > 0 && desiredSwapchainTextures > surfaceCapabilities.MaxTextureCount {
	// 	// App must settle for fewer images than desired.
	// 	desiredSwapchainTextures = surfaceCapabilities.MaxTextureCount
	// }

	// Figure out a suitable surface transform.
	// var preTransform vk.SurfaceTransformFlagBits
	// requiredTransforms := vk.SurfaceTransformIdentityBit
	// supportedTransforms := surfaceCapabilities.SupportedTransforms
	// if vk.SurfaceTransformFlagBits(supportedTransforms)&requiredTransforms != 0 {
	// 	preTransform = requiredTransforms
	// } else {
	// 	preTransform = surfaceCapabilities.CurrentTransform
	// }

	// Find a supported composite alpha mode - one of these is guaranteed to be set
	// compositeAlpha := vk.CompositeAlphaOpaqueBit
	// compositeAlphaFlags := []vk.CompositeAlphaFlagBits{
	// 	vk.CompositeAlphaPreMultipliedBit,
	// 	vk.CompositeAlphaOpaqueBit, // this only affects blending with other windows in OS
	// 	vk.CompositeAlphaPostMultipliedBit,
	// 	vk.CompositeAlphaInheritBit,
	// }
	// // goti := -1
	// for i := 0; i < len(compositeAlphaFlags); i++ {
	// 	alphaFlags := vk.CompositeAlphaFlags(compositeAlphaFlags[i])
	// 	flagSupported := surfaceCapabilities.SupportedCompositeAlpha&alphaFlags != 0
	// 	if flagSupported {
	// 		// goti = i
	// 		compositeAlpha = compositeAlphaFlags[i]
	// 		break
	// 	}
	// }

	//	fmt.Printf("Got alpha: %d\n", goti)

	// Create a swapchain
	// var swapchain vk.Swapchain
	// oldSwapchain := sf.Swapchain
	// swci := &vk.SwapchainCreateInfo{
	// 	SType:             vk.StructureTypeSwapchainCreateInfo,
	// 	Surface:           sf.Surface,
	// 	MinTextureCount:   desiredSwapchainTextures,
	// 	TextureFormat:     format.Format,
	// 	TextureColorSpace: format.ColorSpace,
	// 	TextureExtent: vk.Extent2D{
	// 		Width:  swapchainSize.Width,
	// 		Height: swapchainSize.Height,
	// 	},
	// 	TextureUsage:       vk.TextureUsageFlags(vk.TextureUsageColorAttachmentBit),
	// 	PreTransform:       preTransform,
	// 	CompositeAlpha:     compositeAlpha,
	// 	TextureArrayLayers: 1,
	// 	TextureSharingMode: vk.SharingModeExclusive,
	// 	PresentMode:        swapchainPresentMode,
	// 	OldSwapchain:       oldSwapchain,
	// 	Clipped:            vk.True,
	// }
	// ret = vk.CreateSwapchain(dev, swci, nil, &swapchain)
	// IfPanic(NewError(ret))
	// if oldSwapchain != vk.NullSwapchain {
	// 	vk.ReleaseSwapchain(dev, oldSwapchain, nil)
	// }
	// sf.Swapchain = swapchain
	// sf.Format.Set(int(swapchainSize.Width), int(swapchainSize.Height), format.Format)

	// var imageCount uint32
	// ret = vk.GetSwapchainTextures(dev, sf.Swapchain, &imageCount, nil)
	// IfPanic(NewError(ret))
	// sf.NFrames = int(imageCount)
	// swapchainTextures := make([]*wgpu.Texture, imageCount)
	// ret = vk.GetSwapchainTextures(dev, sf.Swapchain, &imageCount, swapchainTextures)
	// IfPanic(NewError(ret))
	//
	// sf.TextureAcquired = NewSemaphore(dev)
	// sf.RenderDone = NewSemaphore(dev)
	// sf.RenderFence = NewFence(dev)
	//
	// sf.Frames = make([]*Framebuffer, sf.NFrames)
	// for i := 0; i < sf.NFrames; i++ {
	// 	fr := &Framebuffer{}
	// 	fr.ConfigSurfaceTexture(sf.GPU, dev, sf.Format, swapchainTextures[i])
	// 	sf.Frames[i] = fr
	// }
	// return true
}

// ReleaseSwapChain frees any existing swawpchain (for ReInit or Release)
func (sf *Surface) ReleaseSwapChain() {
	// vk.DeviceWaitIdle(dev)
	// vk.ReleaseSemaphore(dev, sf.TextureAcquired, nil)
	// vk.ReleaseSemaphore(dev, sf.RenderDone, nil)
	// vk.ReleaseFence(dev, sf.RenderFence, nil)
	// for _, fr := range sf.Frames {
	// 	fr.Release()
	// }
	// sf.Frames = nil
	if sf.swapChain != nil {
		sf.swapChain.Release()
		sf.swapChain = nil
	}
}

// ReConfigSwapChain does a re-initialize of swapchain, freeing existing.
// This must be called when the window is resized.
// It returns false if the swapchain size is zero.
func (sf *Surface) ReConfigSwapChain() bool {
	sf.ReleaseSwapChain()
	if sf.ConfigSwapChain() != nil {
		return false
	}
	// sf.Render.SetSize(sf.Format.Size)
	// sf.ReConfigFrames()
	return true
}

func (sf *Surface) Release() {
	sf.ReleaseSwapChain()
	if sf.surface != nil {
		sf.surface.Release()
		sf.surface = nil
	}
	if sf.Device != nil {
		sf.Device.Release()
		sf.Device = nil
	}
	sf.GPU = nil
}
