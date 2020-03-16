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

	"github.com/goki/mat32"
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
// Unlike Window, the Drawer interface for Texture does *not* manage the
// TheApp.RunOnMain and context Activate steps needed to set GPU context properly -
// these must be done prior to calling any of those routines.  Also, a
// 0,0 = top, left coordinate system is assumed for all Draw routines, but
// when drawing onto a Texture, its 0,0 is actually bottom, left, so that is
// managed internally to preserve the same overall coordinate system.
//
// Please use the gpu.Texture2D version for GPU-based texture uses (3D rendering)
// for greater clarity.
//
// For GPU-level uses, a Texture can be created and configured prior to calling
// Activate(), which is when the GPU-side version of the texture is created
// and configured.  Window-backing Textures are always Activated and are
// automatically resized etc along with their parent window.
type Texture interface {
	// Name returns the name of the texture (filename without extension
	// by default)
	Name() string

	// SetName sets the name of the texture
	SetName(name string)

	// Open loads texture image from file.
	// Format inferred from filename -- JPEG and PNG supported by default.
	// Generally call this prior to doing Activate --
	// if Activate()'d, then must be called with a valid gpu context
	// and on proper thread for that context.
	Open(path string) error

	// Image returns the current image -- typically as an *image.RGBA.
	// This is the image that was last set using Open, SetImage, or GrabImage.
	// Use GrabImage to get the current GPU-side image, e.g., for rendering targets.
	Image() image.Image

	// GrabImage retrieves the current contents of the Texture, e.g., if it has been
	// used as a rendering target (Y axis flipped so top = 0).  Returns nil if not initialized.
	// Must be called with a valid gpu context and on proper thread for that context.
	// Returned image points to single internal image.RGBA used for this texture --
	// copy before modifying and to retain values.
	GrabImage() image.Image

	// SetImage sets entire contents of the Texture from given image
	// (including setting the size of the texture from that of the img).
	// This is most efficiently done using an image.RGBA, but other
	// formats will be converted as necessary.  Image Y axis is automatically
	// flipped when transferred up to the texture, so texture has bottom = 0.
	// Can be called prior to doing Activate(), in which case the image
	// pixels will then initialize the GPU version of the texture.
	// (most efficient case for standard GPU / 3D usage).
	// If called after Activate then the image is copied up to the GPU
	// and texture is left in an Activate state.
	// If Activate()'d, then must be called with a valid gpu context
	// and on proper thread for that context.
	SetImage(img image.Image) error

	// ImageFlipY flips the Y axis from a source image.RGBA into a dest.
	// both must be the same size else it panics.  This utility function
	// is needed for GrabImage and is made avail for general use here.
	ImageFlipY(dest, src *image.RGBA)

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

	// BotZero returns true if this texture has the Y=0 pixels at the bottom
	// of the image.  Otherwise, Y=0 is at the top, which is the default
	// for most images loaded from files.
	BotZero() bool

	// SetBotZero sets whether this texture has the Y=0 pixels at the bottom
	// of the image.  Otherwise, Y=0 is at the top, which is the default
	// for most images loaded from files.
	SetBotZero(botzero bool)

	// Activate establishes the GPU resources and handle for the
	// texture, using the given texture number (0-31 range).
	// If an image has already been set for this texture, then it is
	// copied up to the GPU at this point -- otherwise the texture
	// is nil initialized.
	// Must be called with a valid gpu context and on proper thread for that context.
	Activate(texNo int)

	// IsActive returns true if texture has already been Activate'd
	// and thus exists on the GPU
	IsActive() bool

	// Handle returns the GPU handle for the texture -- only
	// valid after Activate.
	Handle() uint32

	// Transfer copies current image up to the GPU, activating on given
	// texture number.  Returns false if there is no image to transfer.
	// Must be called with a valid gpu context and on proper thread for that context.
	Transfer(texNo int) bool

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

	// FrameDepthAt return depth (0-1) at given pixel location from texture used as a framebuffer
	// Must be called with a valid gpu context and on proper thread for that context.
	FrameDepthAt(x, y int) (float32, error)

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
// than the full matrix used by Draw.
//
// When drawing on a Window, there will not be any visible effect until Publish
// is called.
//
// Draw automatically takes into account the Y-axis orientation of the destination
// and source textures.
//
// If the destination is a Window, then its Y=0 is always at the bottom.
//
// If the destination is a Texture, its BotZero parameter determines whether
// Y=0 is at the bottom or top -- the default is Y=0 at the top, which is
// true for textures loaded from images files or rendered with that standard.
//
// The source texture also has its own BotZero parameter, so that determines
// whether it is flipped relative to the destination during the rendering.
//
// Finally, if the FlipY option is passed, the source is flipped relative to
// that determined by the default rules above.
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
	// (the method receiver), such that sr.Min in src-space aligns with dp in dst-space.
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
	// FlipY means flip the Y (vertical) axis of the source when rendering into destination,
	// relative to the default orientation of the source as determined by its BotZero setting.
	FlipY bool
}
