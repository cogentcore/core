// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gpu

import (
	"image"
	"sync"

	"cogentcore.org/core/base/errors"
	"github.com/cogentcore/webgpu/wgpu"
)

// Surface manages the physical device for the visible image
// of a window surface, and the swapchain for presenting images.
// It provides an encapsulated source of TextureView textures
// for the rendering process to draw on.
// It implements the Renderer interface, which defines the
// primary API (GetCurrentTexture() -> Present()).
type Surface struct {
	// Render helper for this Surface.
	render Render

	// Format has the current rendering surface size and
	// rendering texture format.  This format may be different
	// from the actual physical swapchain format, in case there
	// is a different view (e.g., srgb)
	Format TextureFormat

	// pointer to gpu device, needed for properties.
	GPU *GPU

	// Device for this surface, which we own.
	// Each window surface has its own device, configured for that surface.
	device *Device

	// WebGPU handle for surface
	surface *wgpu.Surface

	config *wgpu.SurfaceConfiguration

	// current texture: must release at end
	curTexture *wgpu.TextureView

	sync.Mutex
}

// NewSurface returns a new surface initialized for given GPU and WebGPU
// Surface handle, obtained from a valid window.
//   - size should reflect the actual size of the surface,
//     and can be updated with SetSize method.
//   - samples is the multisampling anti-aliasing parameter: 1 = none
//     4 = typical default value for smooth "no jaggy" edges.
//   - depthFmt is the depth buffer format.  use UndefinedType for none
//     or Depth32 recommended for best performance.
func NewSurface(gp *GPU, wsurf *wgpu.Surface, size image.Point, samples int, depthFmt Types) *Surface {
	sf := &Surface{}
	sf.Defaults()
	sf.init(gp, wsurf, size, samples, depthFmt)
	return sf
}

func (sf *Surface) Defaults() {
	// sf.NFrames = 3 // requested, will be updated with actual
	sf.Format.Defaults()
	sf.Format.Set(1024, 768, wgpu.TextureFormatRGBA8UnormSrgb)
	sf.Format.SetMultisample(4)
}

func (sf *Surface) init(gp *GPU, ws *wgpu.Surface, size image.Point, samples int, depthFmt Types) error {
	sf.GPU = gp
	sf.surface = ws
	dev, err := gp.NewDevice() // surface owns this device
	if errors.Log(err) != nil {
		return err
	}
	sf.device = dev
	// note: Format.Format will be determined in InitConfig,
	// based on GetCapabilities call.
	sf.Format.SetMultisample(samples)
	sf.Format.Size = size
	sf.InitConfig() // can change the format
	sf.render.Config(sf.device, &sf.Format, depthFmt)
	return nil
}

func (sf *Surface) Device() *Device { return sf.device }
func (sf *Surface) Render() *Render { return &sf.render }

// When the render surface (e.g., window) is resized, call this function.
// WebGPU does not have any internal mechanism for tracking this, so we
// need to drive it from external events.
func (sf *Surface) SetSize(sz image.Point) {
	if sf.Format.Size == sz || sz.X == 0 || sz.Y == 0 {
		return
	}
	sf.render.SetSize(sz)
	sf.Format.Size = sz
	sf.config.Width = uint32(sf.Format.Size.X)
	sf.config.Height = uint32(sf.Format.Size.Y)
	sf.Reconfig()
}

// GetCurrentTexture returns a TextureView that is the current
// target for rendering.
func (sf *Surface) GetCurrentTexture() (*wgpu.TextureView, error) {
	sf.Lock() // we remain locked until submit!
	texture, err := sf.surface.GetCurrentTexture()
	if errors.Log(err) != nil {
		return nil, err
	}
	// Note: we need to specify a descriptor here so that we use the correct
	// format, which may be different from the default format, such as when
	// it is srgb.
	view, err := texture.CreateView(&wgpu.TextureViewDescriptor{
		MipLevelCount:   texture.GetMipLevelCount(),
		ArrayLayerCount: texture.GetDepthOrArrayLayers(),
		Format:          sf.Format.Format,
	})
	if errors.Log(err) != nil {
		return nil, err
	}
	sf.curTexture = view
	return view, nil
}

// Present is the final step for showing the rendered texture to the window.
// The current texture is automatically Released and Unlock() is called.
func (sf *Surface) Present() {
	sf.surface.Present()
	if sf.curTexture != nil {
		sf.curTexture.Release()
		sf.curTexture = nil
	}
	sf.Unlock()
}

// InitConfig does the initial configuration of the surface.
// This assumes that all existing items have been destroyed.
func (sf *Surface) InitConfig() error {
	caps := sf.surface.GetCapabilities(sf.GPU.GPU)
	trgFmt := caps.Formats[0]
	// fmt.Println(reflectx.StringJSON(caps), trgFmt)
	viewFmt := trgFmt
	switch trgFmt {
	case wgpu.TextureFormatBGRA8Unorm:
		viewFmt = wgpu.TextureFormatBGRA8UnormSrgb
	case wgpu.TextureFormatRGBA8Unorm:
		viewFmt = wgpu.TextureFormatRGBA8UnormSrgb
	}
	var viewFmts []wgpu.TextureFormat
	if viewFmt != trgFmt {
		viewFmts = append(viewFmts, viewFmt)
	}

	sf.config = &wgpu.SurfaceConfiguration{
		Usage:       wgpu.TextureUsageRenderAttachment,
		Format:      trgFmt,
		Width:       uint32(sf.Format.Size.X),
		Height:      uint32(sf.Format.Size.Y),
		PresentMode: wgpu.PresentModeFifo,
		AlphaMode:   caps.AlphaModes[0],
		ViewFormats: viewFmts,
	}

	sf.Format.Format = viewFmt
	sf.Config()
	return nil
}

// Config configures the surface based on the surface configuration.
func (sf *Surface) Config() {
	sf.surface.Configure(sf.GPU.GPU, sf.device.Device, sf.config)
}

// Reconfig reconfigures the surface.
// This must be called when the window is resized.
// Must update the swapChainConfig parameters prior to calling!
// It returns false if the swapchain size is zero.
func (sf *Surface) Reconfig() bool {
	sf.Lock()
	defer sf.Unlock()
	sf.Config()
	sf.render.SetSize(sf.Format.Size)
	return true
}

func (sf *Surface) Release() {
	sf.render.Release()
	if sf.surface != nil {
		sf.surface.Release()
		sf.surface = nil
	}
	if sf.device != nil {
		sf.device.Release()
		sf.device = nil
	}
	sf.GPU = nil
}
