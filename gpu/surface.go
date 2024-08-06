// Copyright (c) 2022, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// This is initially adapted from https://github.com/WebGPU-go/asche
// Copyright © 2017 Maxim Kupriianov <max@kc.vc>, under the MIT License

package gpu

import (
	"image"
	"sync"

	"cogentcore.org/core/base/errors"
	"github.com/cogentcore/webgpu/wgpu"
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

	// WebGPU handle for surface
	surface *wgpu.Surface `display:"-"`

	swapChainConfig *wgpu.SwapChainDescriptor

	// WebGPU handle for swapchain
	swapChain *wgpu.SwapChain `display:"-"`

	// current texture: must release at end
	curTexture *wgpu.TextureView

	needsReconfig bool

	sync.Mutex
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
	sf.Format.SetMultisample(1) // good default // TODO(wgpu): set to 4
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
	sf.Format.Size.X = width
	sf.Format.Size.Y = height
	sf.ConfigSwapChain()
	return nil
}

func (sf *Surface) AcquireNextTexture() (*wgpu.TextureView, error) {
	if sf.needsReconfig {
		sf.ReConfigSwapChain()
	}
	sf.Lock() // we remain locked until submit!
	view, err := sf.swapChain.GetCurrentTextureView()
	if errors.Log(err) != nil {
		return nil, err
	}
	sf.curTexture = view
	return view, nil
}

func (sf *Surface) SubmitRender(rp *wgpu.RenderPassEncoder, cmd *wgpu.CommandEncoder) error {
	cmdBuffer, err := cmd.Finish(nil)
	if errors.Log(err) != nil {
		return err
	}
	sf.Device.Queue.Submit(cmdBuffer)
	rp.Release()
	cmdBuffer.Release()
	return nil
}

func (sf *Surface) Present() {
	sf.swapChain.Present()
	sf.curTexture.Release()
	sf.curTexture = nil
	sf.Unlock()
}

// ConfigSwapChain configures the swapchain for surface.
// This assumes that all existing items have been destroyed.
func (sf *Surface) ConfigSwapChain() error {
	caps := sf.surface.GetCapabilities(sf.GPU.GPU)

	// fmt.Println(reflectx.StringJSON(caps))

	trgFmt := caps.Formats[0]
	var viewfmts []wgpu.TextureFormat
	switch trgFmt {
	case wgpu.TextureFormatBGRA8Unorm:
		viewfmts = append(viewfmts, wgpu.TextureFormatBGRA8UnormSrgb)
	case wgpu.TextureFormatRGBA8Unorm:
		viewfmts = append(viewfmts, wgpu.TextureFormatRGBA8UnormSrgb)
	}

	sf.swapChainConfig = &wgpu.SwapChainDescriptor{
		Usage:       wgpu.TextureUsageRenderAttachment,
		Format:      trgFmt,
		Width:       uint32(sf.Format.Size.X),
		Height:      uint32(sf.Format.Size.Y),
		PresentMode: wgpu.PresentModeFifo,
		AlphaMode:   caps.AlphaModes[0],
		ViewFormats: viewfmts,
	}

	sf.Format.Format = caps.Formats[0]
	return sf.CreateSwapChain()
}

func (sf *Surface) CreateSwapChain() error {
	sc, err := sf.Device.Device.CreateSwapChain(sf.surface, sf.swapChainConfig)
	if err != nil {
		return err
	}
	sf.swapChain = sc
	// fmt.Println("sc:", sf.Format.String())
	return nil
}

// ReleaseSwapChain frees any existing swawpchain (for ReInit or Release)
func (sf *Surface) ReleaseSwapChain() {
	if sf.swapChain == nil {
		return
	}
	sf.Device.WaitDone()
	sf.swapChain.Release()
	sf.swapChain = nil
}

// When the render surface (e.g., window) is resized, call this function.
// WebGPU does not have any internal mechanism for tracking this, so we
// need to drive it from external events.
func (sf *Surface) Resized(newSize image.Point) {
	sf.Format.Size = newSize
	sf.swapChainConfig.Width = uint32(sf.Format.Size.X)
	sf.swapChainConfig.Height = uint32(sf.Format.Size.Y)
	sf.needsReconfig = true
}

// ReConfigSwapChain does a re-create of swapchain, freeing existing.
// This must be called when the window is resized.
// must update the swapChainConfig parameters prior to calling!
// It returns false if the swapchain size is zero.
func (sf *Surface) ReConfigSwapChain() bool {
	sf.Lock()
	defer sf.Unlock()
	sf.needsReconfig = false
	sf.ReleaseSwapChain()
	if sf.CreateSwapChain() != nil {
		return false
	}
	if sf.Render != nil {
		sf.Render.SetSize(sf.Format.Size)
	}
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
