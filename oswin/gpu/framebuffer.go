// Copyright (c) 2019, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gpu

import "image"

// Framebuffer is an offscreen render target.
// gi3d renders exclusively to a Framebuffer, which is then copied to
// the overall oswin.Window.WinTex texture that backs the window for
// final display to the user.
// Use gpu.TheGPU.NewFramebuffer for a new freestanding framebuffer,
// or gpu.Texture2D.ActivateFramebuffer to activate for rendering
// onto an existing texture.
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
	// Setting to a number > 0 automatically disables use of external
	// Texture2D that might have previously been set by SetTexture --
	// must call Texture() to get downsampled texture instead.
	SetSamples(samples int)

	// Samples returns the number of multi-sample samples
	Samples() int

	// SetTexture sets an existing Texture2D to serve as the color buffer target
	// for this framebuffer.  This also implies SetSamples(0), and that
	// the Texture() method just directly returns the texture set here.
	// If we have a non-zero size, then the texture is automatically resized
	// to the size of the framebuffer, otherwise the fb inherits size of texture.
	SetTexture(tex Texture2D)

	// Texture returns the current contents of the framebuffer as a Texture2D.
	// For Samples() > 0 this reduces the optimized internal render buffer to a
	// standard 2D texture -- the return texture is owned and managed by the
	// framebuffer, and re-used every time Texture() is called.
	// If SetTexture was called, then it just returns that texture
	// which was directly rendered to.
	Texture() Texture2D

	// DepthAt returns the depth-buffer value at given x,y coordinate in
	// framebuffer coordinates (i.e., Y = 0 is at bottom).
	// Must be called with a valid gpu context and on proper thread for that context,
	// with framebuffer active.
	DepthAt(x, y int) float32

	// DepthAll returns the entire depth buffer as a slice of float32 values
	// of same size as framebuffer.  This slice is pointer to internal reused
	// value -- copy to retain values or modify.
	// Must be called with a valid gpu context and on proper thread for that context,
	// with framebuffer active.
	DepthAll() []float32

	// Activate establishes the GPU resources and handle for the
	// framebuffer and all other associated buffers etc (if not already done).
	// It then sets this as the current rendering target for subsequent
	// gpu drawing commands.
	Activate() error

	// Rendered should be called after rendering to the framebuffer,
	// to ensure the update of data transferred from the framebuffer
	// (texture, depth buffer)
	Rendered()

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
