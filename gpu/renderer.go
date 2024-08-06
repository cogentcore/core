// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gpu

import "github.com/rajveermalviya/go-webgpu/wgpu"

// Renderer is an interface for something that can actually be rendered to.
// It returns a TextureView to render into, and then Presents the result.
// Surface and RenderTexture are the two main implementers of this interface.
type Renderer interface {
	// GetCurrentTexture returns a TextureView that is the current
	// target for rendering.
	GetCurrentTexture() (*wgpu.TextureView, error)

	// SetRender sets the Render helper for Multisampling
	// and Depth buffer.
	SetRender(rd *Render)

	// Present presents the rendered texture to the window
	// and finalizes the current render pass.
	Present()
}
