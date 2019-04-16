// Copyright (c) 2019, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gpu

import "image"

// Framebuffer is an offscreen render target.
// gi3d renders exclusively to a Framebuffer, which is then copied to
// the overall oswin.Window.WinTex texture that backs the window for
// final display to the user.
type Framebuffer interface {
	// Name returns the name of the framebuffer
	Name() string

	// SetName sets the name of the framebuffer
	SetName(name string)

	// Size returns the size of the framebuffer
	Size() image.Point

	// SetSize sets the size of the framebuffer.
	// If framebuffer has been Activate'd, then this resizes the GPU side as well.
	SetSize(size image.Point)

	// Bounds returns the bounds of the framebuffer's image. It is equal to
	// image.Rectangle{Max: t.Size()}.
	Bounds() image.Rectangle

	// SetSamples sets the number of samples to use for multi-sample
	// anti-aliasing.  When using as a primary 3D render target,
	// the recommended number is 4 to produce much better-looking results.
	// If just using as an intermediate render buffer, then you may
	// want to turn off by setting to 0.
	SetSamples(samples int)

	// Samples returns the number of multi-sample samples
	Samples() int

	// SetTexture sets an existing Texture2D to serve as the color buffer target
	// for this framebuffer.  This also implies SetSamples(0), and that
	// the Texture() method just directly returns the texture set here.
	SetTexture(tex Texture2D)

	// Texture returns the current contents of the framebuffer as a Texture2D.
	// For Samples() > 0 this reduces the optimized internal render buffer to a
	// standard 2D texture.  If SetTexture was called, then it just returns that
	// texture which was directly rendered to.
	Texture() Texture2D

	// todo: methods to get the depth, stencil buffer output as well..

	// Activate establishes the GPU resources and handle for the
	// framebuffer and all other associated buffers etc (if not already done).
	// It then sets this as the current rendering target for subsequent
	// gpu drawing commands.
	Activate()

	// Handle returns the GPU handle for the framebuffer -- only
	// valid after Activate.
	Handle() uint32

	// Delete deletes the GPU resources associated with this framebuffer
	// (requires Activate to re-establish a new one).
	// Should be called prior to Go object being deleted
	// (ref counting can be done externally).
	// Does NOT delete any Texture set by SetTexture -- must be done externally.
	Delete()
}
