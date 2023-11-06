// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package goosi

import (
	"image"
	"image/draw"
)

// Capture tells the app drawer to capture its next frame as an image and save it
// to the given filename. It is currently only supported with the offscreen build tag.
func Capture(filename string) {
	NeedsCapture = true
	CaptureFilename = filename
}

var (
	// NeedsCapture is whether the app drawer needs to capture its next frame (see [Capture])
	NeedsCapture bool
	// CaptureFilename is the filename the app drawer should save its capture to (see [Capture])
	CaptureFilename string
)

const (
	// MaxTexturesPerSet is the maximum number of image variables that can be used
	// in one descriptor set.  This value is a lowest common denominator across
	// platforms.  To overcome this limitation, when more Texture vals are allocated,
	// multiple NDescs are used, setting the and switch
	// across those -- each such Desc set can hold this many textures.
	// NValsPer on a Texture var can be set higher and only this many will be
	// allocated in the descriptor set, with bindings of values wrapping
	// around across as many such sets as are vals, with a warning if insufficient
	// numbers are present.
	MaxTexturesPerSet = 16

	// MaxImageLayers is the maximum number of layers per image
	MaxImageLayers = 128

	// FlipY used as named arg for flipping the Y axis of images, etc
	FlipY = true

	// NoFlipY used as named arg for not flipping the Y axis of images
	NoFlipY = false
)

// Drawer is an interface representing a type capable of high-performance
// rendering to a window surface. It is implemented by [*goki.dev/vgpu/v2/vdraw.Drawer]
// and an internal web driver.
type Drawer interface {
	// SetMaxTextures updates the max number of textures for drawing
	// Must call this prior to doing any allocation of images.
	SetMaxTextures(maxTextures int)

	// MaxTextures returns the max number of textures for drawing
	MaxTextures() int

	// DestBounds returns the bounds of the render destination
	DestBounds() image.Rectangle

	// SetGoImage sets given Go image as a drawing source to given image index,
	// and layer, used in subsequent Draw methods.
	// A standard Go image is rendered upright on a standard surface.
	// Set flipY to true to flip.
	SetGoImage(idx, layer int, img image.Image, flipY bool)

	// ConfigImageDefaultFormat configures the draw image at the given index
	// to fit the default image format specified by the given width, height,
	// and number of layers.
	ConfigImageDefaultFormat(idx int, width int, height int, layers int)

	// ConfigImage configures the draw image at given index
	// to fit the given image format and number of layers as a drawing source.
	// ConfigImage(idx int, fmt *vgpu.ImageFormat)

	// SyncImages must be called after images have been updated, to sync
	// memory up to the GPU.
	SyncImages()

	// Scale copies texture at given index and layer to render target,
	// scaling the region defined by src and sr to the destination
	// such that sr in src-space is mapped to dr in dst-space.
	// dr is the destination rectangle
	// sr is the source region (set to image.ZR zero rect for all),
	// op is the drawing operation: Src = copy source directly (blit),
	// Over = alpha blend with existing
	// flipY = flipY axis when drawing this image
	Scale(idx, layer int, dr image.Rectangle, sr image.Rectangle, op draw.Op, flipY bool) error

	// Copy copies texture at given index and layer to render target.
	// dp is the destination point,
	// sr is the source region (set to image.ZR zero rect for all),
	// op is the drawing operation: Src = copy source directly (blit),
	// Over = alpha blend with existing
	// flipY = flipY axis when drawing this image
	Copy(idx, layer int, dp image.Point, sr image.Rectangle, op draw.Op, flipY bool) error

	// UseTextureSet selects the descriptor set to use --
	// choose this based on the bank of 16
	// texture values if number of textures > MaxTexturesPerSet.
	UseTextureSet(descIdx int)

	// StartDraw starts image drawing rendering process on render target
	// No images can be added or set after this point.
	// descIdx is the descriptor set to use -- choose this based on the bank of 16
	// texture values if number of textures > MaxTexturesPerSet.
	StartDraw(descIdx int)

	// EndDraw ends image drawing rendering process on render target
	EndDraw()
}
