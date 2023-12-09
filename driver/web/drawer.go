// Copyright 2023 The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build js

package web

import (
	"image"
	"image/draw"
	"syscall/js"
	"unsafe"
)

// Drawer is a TEMPORARY, low-performance implementation of [goosi.Drawer].
// It will be replaced with a full WebGPU based drawer at some point.
// TODO: replace Drawer with WebGPU
type Drawer struct {
	maxTextures int
	image       *image.RGBA     // target render image
	images      [][]*image.RGBA // stack of images indexed by render scene index and then layer number
}

// SetMaxTextures updates the max number of textures for drawing
// Must call this prior to doing any allocation of images.
func (dw *Drawer) SetMaxTextures(maxTextures int) {
	dw.maxTextures = maxTextures
}

// MaxTextures returns the max number of textures for drawing
func (dw *Drawer) MaxTextures() int {
	return dw.maxTextures
}

// DestBounds returns the bounds of the render destination
func (dw *Drawer) DestBounds() image.Rectangle {
	return TheApp.Scrn.Geometry
}

// SetGoImage sets given Go image as a drawing source to given image index,
// and layer, used in subsequent Draw methods.
// A standard Go image is rendered upright on a standard surface.
// Set flipY to true to flip.
func (dw *Drawer) SetGoImage(idx, layer int, img image.Image, flipY bool) {
	if dw.image == nil {
		dw.image = image.NewRGBA(image.Rect(0, 0, img.Bounds().Dx(), img.Bounds().Dy()))
	}
	for len(dw.images) <= idx {
		dw.images = append(dw.images, nil)
	}
	imgs := &dw.images[idx]
	for len(*imgs) <= layer {
		*imgs = append(*imgs, nil)
	}
	(*imgs)[layer] = img.(*image.RGBA)
}

// ConfigImageDefaultFormat configures the draw image at the given index
// to fit the default image format specified by the given width, height,
// and number of layers.
func (dw *Drawer) ConfigImageDefaultFormat(idx int, width int, height int, layers int) {
	// no-op
}

// ConfigImage configures the draw image at given index
// to fit the given image format and number of layers as a drawing source.
// ConfigImage(idx int, fmt *vgpu.ImageFormat)

// SyncImages must be called after images have been updated, to sync
// memory up to the GPU.
func (dw *Drawer) SyncImages() {
	// no-op
}

// Scale copies texture at given index and layer to render target,
// scaling the region defined by src and sr to the destination
// such that sr in src-space is mapped to dr in dst-space.
// dr is the destination rectangle
// sr is the source region (set to image.ZR zero rect for all),
// op is the drawing operation: Src = copy source directly (blit),
// Over = alpha blend with existing
// flipY = flipY axis when drawing this image
func (dw *Drawer) Scale(idx, layer int, dr image.Rectangle, sr image.Rectangle, op draw.Op, flipY bool) error {
	img := dw.images[idx][layer]
	draw.Draw(dw.image, dr, img, sr.Min, op)
	return nil
}

// Copy copies texture at given index and layer to render target.
// dp is the destination point,
// sr is the source region (set to image.ZR zero rect for all),
// op is the drawing operation: Src = copy source directly (blit),
// Over = alpha blend with existing
// flipY = flipY axis when drawing this image
func (dw *Drawer) Copy(idx, layer int, dp image.Point, sr image.Rectangle, op draw.Op, flipY bool) error {
	img := dw.images[idx][layer]
	// fmt.Println("cp", idx, layer, dp, dp.Add(img.Rect.Size()), sr.Min)
	draw.Draw(dw.image, image.Rectangle{dp, dp.Add(img.Rect.Size())}, img, sr.Min, op)
	return nil
}

// UseTextureSet selects the descriptor set to use --
// choose this based on the bank of 16
// texture values if number of textures > MaxTexturesPerSet.
func (dw *Drawer) UseTextureSet(descIdx int) {
	// no-op
}

// StartDraw starts image drawing rendering process on render target
// No images can be added or set after this point.
// descIdx is the descriptor set to use -- choose this based on the bank of 16
// texture values if number of textures > MaxTexturesPerSet.
func (dw *Drawer) StartDraw(descIdx int) {
	sz := dw.image.Bounds().Size()
	ptr := uintptr(unsafe.Pointer(&dw.image.Pix[0]))
	js.Global().Call("displayImage", ptr, len(dw.image.Pix), sz.X, sz.Y)
}

// EndDraw ends image drawing rendering process on render target
func (dw *Drawer) EndDraw() {
	// no-op
}

func (dw *Drawer) Surface() any {
	return nil
}
