// Copyright 2023 The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build js

package web

import (
	"fmt"
	"image"
	"image/draw"
)

// drawerImpl is a TEMPORARY, low-performance implementation of [goosi.Drawer].
// It will be replaced with a full WebGPU based drawer at some point.
// TODO: replace drawerImpl with WebGPU
type drawerImpl struct {
	maxTextures int
	images      [][]image.Image
}

// SetMaxTextures updates the max number of textures for drawing
// Must call this prior to doing any allocation of images.
func (dw *drawerImpl) SetMaxTextures(maxTextures int) {
	dw.maxTextures = maxTextures
}

// MaxTextures returns the max number of textures for drawing
func (dw *drawerImpl) MaxTextures() int {
	return dw.maxTextures
}

// DestBounds returns the bounds of the render destination
func (dw *drawerImpl) DestBounds() image.Rectangle {
	return theApp.screen.Geometry
}

// SetGoImage sets given Go image as a drawing source to given image index,
// and layer, used in subsequent Draw methods.
// A standard Go image is rendered upright on a standard surface.
// Set flipY to true to flip.
func (dw *drawerImpl) SetGoImage(idx, layer int, img image.Image, flipY bool) {
	dw.images[idx][layer] = img
}

// ConfigImageDefaultFormat configures the draw image at the given index
// to fit the default image format specified by the given width, height,
// and number of layers.
func (dw *drawerImpl) ConfigImageDefaultFormat(idx int, width int, height int, layers int) {
	for len(dw.images) <= idx {
		dw.images = append(dw.images, nil)
	}
	dw.images[idx] = make([]image.Image, layers)
}

// ConfigImage configures the draw image at given index
// to fit the given image format and number of layers as a drawing source.
// ConfigImage(idx int, fmt *vgpu.ImageFormat)

// SyncImages must be called after images have been updated, to sync
// memory up to the GPU.
func (dw *drawerImpl) SyncImages() {}

// Scale copies texture at given index and layer to render target,
// scaling the region defined by src and sr to the destination
// such that sr in src-space is mapped to dr in dst-space.
// dr is the destination rectangle
// sr is the source region (set to image.ZR zero rect for all),
// op is the drawing operation: Src = copy source directly (blit),
// Over = alpha blend with existing
// flipY = flipY axis when drawing this image
func (dw *drawerImpl) Scale(idx, layer int, dr image.Rectangle, sr image.Rectangle, op draw.Op, flipY bool) error {
	return nil
}

// Copy copies texture at given index and layer to render target.
// dp is the destination point,
// sr is the source region (set to image.ZR zero rect for all),
// op is the drawing operation: Src = copy source directly (blit),
// Over = alpha blend with existing
// flipY = flipY axis when drawing this image
func (dw *drawerImpl) Copy(idx, layer int, dp image.Point, sr image.Rectangle, op draw.Op, flipY bool) error {
	return nil
}

// UseTextureSet selects the descriptor set to use --
// choose this based on the bank of 16
// texture values if number of textures > MaxTexturesPerSet.
func (dw *drawerImpl) UseTextureSet(descIdx int) {}

// StartDraw starts image drawing rendering process on render target
// No images can be added or set after this point.
// descIdx is the descriptor set to use -- choose this based on the bank of 16
// texture values if number of textures > MaxTexturesPerSet.
func (dw *drawerImpl) StartDraw(descIdx int) {
	fmt.Println("start draw", descIdx)
}

// EndDraw ends image drawing rendering process on render target
func (dw *drawerImpl) EndDraw() {}
