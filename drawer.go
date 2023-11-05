// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package goosi

import (
	"image"
	"image/draw"
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

	// SetGoImage sets given Go image as a drawing source to given image index,
	// and layer, used in subsequent Draw methods.
	// A standard Go image is rendered upright on a standard surface.
	// Set flipY to true to flip.
	SetGoImage(idx, layer int, img image.Image, flipY bool)

	// ConfigImage configures the draw image at given index
	// to fit the given image format and number of layers as a drawing source.
	// ConfigImage(idx int, fmt *vgpu.ImageFormat)

	// UseTextureSet selects the descriptor set to use --
	// choose this based on the bank of 16
	// texture values if number of textures > MaxTexturesPerSet.
	UseTextureSet(descIdx int)

	// Copy copies texture at given index and layer to render target.
	// dp is the destination point,
	// sr is the source region (set to image.ZR zero rect for all),
	// op is the drawing operation: Src = copy source directly (blit),
	// Over = alpha blend with existing
	// flipY = flipY axis when drawing this image
	Copy(idx, layer int, dp image.Point, sr image.Rectangle, op draw.Op, flipY bool) error
}
