// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gpu

import (
	"image"

	"github.com/cogentcore/webgpu/wgpu"
)

// Renderer is an interface for something that can actually be rendered to.
// It returns a TextureView to render into, and then Presents the result.
// Surface and RenderTexture are the two main implementers of this interface.
type Renderer interface {
	// GetCurrentTexture returns a TextureView that is the current
	// target for rendering.
	GetCurrentTexture() (*wgpu.TextureView, error)

	// Present presents the rendered texture to the window
	// and finalizes the current render pass.
	Present()

	// Device returns the device for this renderer,
	// which serves as the source device for the GraphicsSystem
	// and all of its components.
	Device() *Device

	// Render returns the Render object for this renderer,
	// which supports Multisampling and Depth buffers,
	// and handles all the render pass logic and state.
	Render() *Render

	// When the render surface (e.g., window) is resized, call this function.
	// WebGPU does not have any internal mechanism for tracking this, so we
	// need to drive it from external events.
	// This is efficient to call if already the same size.
	SetSize(size image.Point)

	// Release frees associated GPU resources.
	Release()
}
