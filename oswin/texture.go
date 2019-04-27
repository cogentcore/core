// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// based on golang.org/x/exp/shiny:
// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package oswin

import (
	"image"
	"image/color"
	"image/draw"

	"github.com/goki/gi/mat32"
)

// Texture is a pixel buffer on the GPU, and is synonymous with the
// gpu.Texture2D interface.
//
// It must be defined here at the oswin level because it provides the
// updatable backing for a Window: you render to a Texture which is
// then drawn to the window during PublishTex().
//
// Images can be uploaded to Textures, and Textures can be drawn on Windows.
// Textures can also be drawn onto Textures, and can grab rendered output
// from 3D graphics rendering (e.g., via gpu.FrameBuffer and gi3d packages).
//
// Please use the gpu.Texture2D version for GPU-based texture uses (3D rendering)
// for greater clarity.
//
// For GPU-level uses, a Texture can be created and configured prior to calling
// Activate(), which is when the GPU-side version of the texture is created
// and configured.  Window-backing Textures are always Activated and are
// automatically resized etc along with their parent window.
//
// When specifying a sub-Texture via Draw, a Texture's top-left pixel is always
// (0, 0) in its own coordinate space.
type Texture interface {
	// Name returns the name of the texture (filename without extension
	// by default)
	Name() string

	// SetName sets the name of the texture
	SetName(name string)

	// Open loads texture image from file.
	// Format inferred from filename -- JPEG and PNG supported by default.
	// Generally call this prior to doing Activate --
	// If Activate()'d, then must be called with a valid gpu context
	// and on proper thread for that context.
	Open(path string) error

	// Image returns an Image of the texture, as an *image.RGBA.
	// If this Texture has been Activate'd then this retrieves
	// the current contents of the Texture, e.g., if it has been
	// used as a rendering target.
	// If Activate()'d, then must be called with a valid gpu context
	// and on proper thread for that context.
	Image() image.Image

	// SetImage sets entire contents of the Texture from given image
	// (including setting the size of the texture from that of the img).
	// This is most efficiently done using an image.RGBA, but other
	// formats will be converted as necessary.
	// Can be called prior to doing Activate(), in which case the image
	// pixels will then initialize the GPU version of the texture.
	// (most efficient case for standard GPU / 3D usage).
	// If called after Activate then the image is copied up to the GPU
	// and texture is left in an Activate state.
	// If Activate()'d, then must be called with a valid gpu context
	// and on proper thread for that context.
	SetImage(img image.Image) error

	// SetSubImage uploads the sub-Image defined by src and sr to the texture.
	// such that sr.Min in src-space aligns with dp in dst-space.
	// The textures's contents are overwritten; the draw operator
	// is implicitly draw.Src. Texture must be Activate'd to the GPU for this
	// to proceed -- if Activate() has not yet been called, it will be (on texture 0).
	// Must be called with a valid gpu context and on proper thread for that context.
	SetSubImage(dp image.Point, src image.Image, sr image.Rectangle) error

	// Size returns the size of the texture
	Size() image.Point

	// SetSize sets the size of the texture.
	// If texture has been Activate'd, then this resizes the GPU side as well.
	// If Activate()'d, then must be called with a valid gpu context
	// and on proper thread for that context.
	SetSize(size image.Point)

	// Bounds returns the bounds of the Texture's image. It is equal to
	// image.Rectangle{Max: t.Size()}.
	Bounds() image.Rectangle

	// Activate establishes the GPU resources and handle for the
	// texture, using the given texture number (0-31 range).
	// If an image has already been set for this texture, then it is
	// copied up to the GPU at this point -- otherwise the texture
	// is nil initialized.
	// Must be called with a valid gpu context and on proper thread for that context.
	Activate(texNo int)

	// Handle returns the GPU handle for the texture -- only
	// valid after Activate.
	Handle() uint32

	// Delete deletes the GPU resources associated with this texture
	// (requires Activate to re-establish a new one).
	// Should be called prior to Go object being deleted
	// (ref counting can be done externally).
	// Must be called with a valid gpu context and on proper thread for that context.
	Delete()

	// ActivateFramebuffer creates a gpu.Framebuffer for rendering onto
	// this texture (if not already created) and activates it for
	// rendering.  The gpu.Texture2D interface can provide direct access
	// to the created framebuffer.
	// Call gpu.TheGPU.RenderToWindow() or DeActivateFramebuffer
	// to return to window rendering.
	// Must be called with a valid gpu context and on proper thread for that context.
	ActivateFramebuffer()

	// DeActivateFramebuffer de-activates this texture's framebuffer
	// for rendering (just calls gpu.TheGPU.RenderToWindow())
	// Must be called with a valid gpu context and on proper thread for that context.
	DeActivateFramebuffer()

	// DeleteFramebuffer deletes this Texture's framebuffer
	// created during ActivateFramebuffer.
	// Must be called with a valid gpu context and on proper thread for that context.
	DeleteFramebuffer()

	Drawer
}

// Drawer is something you can Draw Textures on (i.e., a Window or another Texture).
//
// Draw is the most general purpose of this interface's methods. It supports
// arbitrary affine transformations, such as translations, scales and
// rotations.
//
// Copy and Scale are more specific versions of Draw. The affected dst pixels
// are an axis-aligned rectangle, quantized to the pixel grid. Copy copies
// pixels in a 1:1 manner, Scale is more general. They have simpler parameters
// than Draw, using ints instead of float64s.
//
// When drawing on a Window, there will not be any visible effect until Publish
// is called.
type Drawer interface {
	// Draw draws the sub-Texture defined by src and sr to the destination (the
	// method receiver). src2dst defines how to transform src coordinates to
	// dst coordinates. For example, if src2dst is the matrix
	//
	// m00 m01 m02
	// m10 m11 m12
	// 0   0   1
	//
	// then the src-space point (sx, sy) maps to the dst-space point
	// (m00*sx + m01*sy + m02, m10*sx + m11*sy + m12).
	// Must be called with a valid gpu context and on proper thread for that context.
	Draw(src2dst mat32.Mat3, src Texture, sr image.Rectangle, op draw.Op, opts *DrawOptions)

	// DrawUniform is like Draw except that the src is a uniform color instead
	// of a Texture.
	// Must be called with a valid gpu context and on proper thread for that context.
	DrawUniform(src2dst mat32.Mat3, src color.Color, sr image.Rectangle, op draw.Op, opts *DrawOptions)

	// Copy copies the sub-Texture defined by src and sr to the destination
	// (the method receiver), such that sr.Min in src-space aligns with dp in
	// dst-space.
	// Must be called with a valid gpu context and on proper thread for that context.
	Copy(dp image.Point, src Texture, sr image.Rectangle, op draw.Op, opts *DrawOptions)

	// Scale scales the sub-Texture defined by src and sr to the destination
	// (the method receiver), such that sr in src-space is mapped to dr in
	// dst-space.
	// Must be called with a valid gpu context and on proper thread for that context.
	Scale(dr image.Rectangle, src Texture, sr image.Rectangle, op draw.Op, opts *DrawOptions)

	// Fill fills that part of the destination (the method receiver) defined by
	// dr with the given color.
	//
	// When filling a Window, there will not be any visible effect until
	// Publish is called.
	// Must be called with a valid gpu context and on proper thread for that context.
	Fill(dr image.Rectangle, src color.Color, op draw.Op)
}

// These draw.Op constants are provided so that users of this package don't
// have to explicitly import "image/draw".
const (
	Over = draw.Over
	Src  = draw.Src
)

// DrawOptions are optional arguments to Draw.
type DrawOptions struct {
	// TODO: transparency in [0x0000, 0xffff]?
	// TODO: scaler (nearest neighbor vs linear)?
}
