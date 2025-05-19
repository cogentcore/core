// Copyright (c) 2022, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gpu

import (
	"image"

	"github.com/cogentcore/webgpu/wgpu"
)

// RenderTexture is an offscreen, non-window-backed rendering target,
// functioning like a Surface.
type RenderTexture struct {
	// Render helper for this RenderTexture.
	render Render

	// Format has the current image format and dimensions.
	// The Samples here are the desired value, whereas our Frames
	// always have Samples = 1, and use render for multisampling.
	Format TextureFormat

	// number of frames to maintain in the simulated swapchain.
	// e.g., 2 = double-buffering, 3 = triple-buffering.
	NFrames int

	// Textures that we iterate through in rendering subsequent frames.
	Frames []*Texture

	// pointer to gpu device, for convenience
	GPU *GPU

	// current frame number
	curFrame int

	// device, which we do NOT own.
	device Device
}

// NewRenderTexture returns a new standalone texture render
// target for given GPU and device, suitable for offscreen rendering
// or intermediate use of the render output for other purposes.
//   - device should be from a Surface if one is being used, otherwise
//     can be created anew for offscreen rendering, and released at end.
//   - size should reflect the actual size of the surface,
//     and can be updated with SetSize method.
//   - samples is the multisampling anti-aliasing parameter: 1 = none
//     4 = typical default value for smooth "no jaggy" edges.
//   - depthFmt is the depth buffer format.  use UndefinedType for none
//     or Depth32 recommended for best performance.
func NewRenderTexture(gp *GPU, dev *Device, size image.Point, samples int, depthFmt Types) *RenderTexture {
	rt := &RenderTexture{}
	rt.Defaults()
	rt.init(gp, dev, size, samples, depthFmt)
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

func (rt *RenderTexture) init(gp *GPU, dev *Device, size image.Point, samples int, depthFmt Types) {
	rt.GPU = gp
	rt.device = *dev
	rt.Format.Size = size
	rt.Format.SetMultisample(samples)
	rt.render.Config(&rt.device, &rt.Format, depthFmt)
	rt.ConfigFrames()
}

func (rt *RenderTexture) Device() *Device { return &rt.device }
func (rt *RenderTexture) Render() *Render { return &rt.render }

// GetCurrentTexture returns a TextureView that is the current
// target for rendering.
func (rt *RenderTexture) GetCurrentTexture() (*wgpu.TextureView, error) {
	cf := rt.curFrame
	rt.curFrame = (rt.curFrame + 1) % rt.NFrames
	return rt.Frames[cf].view, nil
}

// GetCurrentTextureObject returns the current texture itself, not the view.
func (rt *RenderTexture) GetCurrentTextureObject() (*Texture, error) {
	cf := rt.curFrame
	return rt.Frames[cf], nil
}

// ConfigFrames configures the frames, calling ReleaseFrames
// so it is safe for re-use.
func (rt *RenderTexture) ConfigFrames() {
	rt.ReleaseFrames()
	rt.Frames = make([]*Texture, rt.NFrames)
	for i := range rt.NFrames {
		fr := NewTexture(&rt.device)
		fr.ConfigRenderTexture(&rt.device, &rt.Format)
		rt.Frames[i] = fr
	}
}

// SetSize sets the size for the render frame,
// doesn't do anything if already that size.
func (rt *RenderTexture) SetSize(size image.Point) {
	if rt.Format.Size == size {
		return
	}
	rt.render.SetSize(size)
	rt.Format.Size = size
	rt.ConfigFrames()
}

func (rt *RenderTexture) ReleaseFrames() {
	for _, fr := range rt.Frames {
		fr.Release()
	}
	rt.Frames = nil
}

func (rt *RenderTexture) Release() {
	rt.ReleaseFrames()
	rt.render.Release()
}

func (rt *RenderTexture) Present() {
	// no-op
}

// GrabTexture grabs rendered image of given index to RenderTexture.TextureGrab.
// must have waited for render already.
func (rt *RenderTexture) GrabTexture(cmd *wgpu.CommandEncoder, idx int) {
	// rt.Frames[idx].GrabTexture(&rt.device, cmd)
}

// GrabDepthTexture grabs rendered depth image from the Render,
// must have waited for render already.
func (rt *RenderTexture) GrabDepthTexture(cmd *wgpu.CommandEncoder) {
	// rt.render.GrabDepthTexture(&rt.device, cmd)
}
