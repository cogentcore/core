// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package system

import (
	"image"
	"image/color"
	"image/draw"

	"cogentcore.org/core/math32"
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
// rendering to a window surface. It is implemented by [*cogentcore.org/core/vgpu/vdraw.Drawer]
// and internal web and offscreen drivers.
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

	// Copy copies texture at given index and layer to render target.
	// dp is the destination point,
	// sr is the source region (set to image.ZR zero rect for all),
	// op is the drawing operation: Src = copy source directly (blit),
	// Over = alpha blend with existing
	// flipY = flipY axis when drawing this image
	Copy(idx, layer int, dp image.Point, sr image.Rectangle, op draw.Op, flipY bool) error

	// Scale copies texture at given index and layer to render target,
	// scaling the region defined by src and sr to the destination
	// such that sr in src-space is mapped to dr in dst-space.
	// dr is the destination rectangle
	// sr is the source region (set to image.ZR zero rect for all),
	// op is the drawing operation: Src = copy source directly (blit),
	// Over = alpha blend with existing
	// flipY = flipY axis when drawing this image
	// rotDeg = rotation degrees to apply in the mapping, e.g., 90
	// rotates 90 degrees to the left, -90 = right.
	Scale(idx, layer int, dr image.Rectangle, sr image.Rectangle, op draw.Op, flipY bool, rotDeg float32) error

	// UseTextureSet selects the descriptor set to use --
	// choose this based on the bank of 16
	// texture values if number of textures > MaxTexturesPerSet.
	UseTextureSet(descIndex int)

	// StartDraw starts image drawing rendering process on render target
	// No images can be added or set after this point.
	// descIndex is the descriptor set to use -- choose this based on the bank of 16
	// texture values if number of textures > MaxTexturesPerSet.
	// It returns false if rendering can not proceed.
	StartDraw(descIndex int) bool

	// EndDraw ends image drawing rendering process on render target
	EndDraw()

	// Fill fills given color to to render target.
	// src2dst is the transform mapping source to destination
	// coordinates (translation, scaling),
	// reg is the region to fill
	// op is the drawing operation: Src = copy source directly (blit),
	// Over = alpha blend with existing
	Fill(clr color.Color, src2dst math32.Mat3, reg image.Rectangle, op draw.Op) error

	// StartFill starts color fill drawing rendering process on render target
	// It returns false if rendering can not proceed.
	StartFill() bool

	// EndFill ends color filling rendering process on render target
	EndFill()

	// Surface is the vgpu device being drawn to.
	// Could be nil on unsupported devices (web).
	Surface() any

	// SetFrameImage does direct rendering from a *vgpu.Framebuffer image.
	// This is much more efficient for GPU-resident images, as in 3D or video.
	SetFrameImage(idx int, fb any)
}

// DrawerBase is a base implementation of [Drawer] with basic no-ops
// for most methods. Embedders need to implement DestBounds and EndDraw.
type DrawerBase struct {
	// MaxTxts is the max number of textures
	MaxTxts int

	// Image is the target render image
	Image *image.RGBA

	// Images is a stack of images indexed by render scene index and then layer number
	Images [][]*image.RGBA
}

// SetMaxTextures updates the max number of textures for drawing
// Must call this prior to doing any allocation of images.
func (dw *DrawerBase) SetMaxTextures(maxTextures int) {
	dw.MaxTxts = maxTextures
}

// MaxTextures returns the max number of textures for drawing
func (dw *DrawerBase) MaxTextures() int {
	return dw.MaxTxts
}

// SetGoImage sets given Go image as a drawing source to given image index,
// and layer, used in subsequent Draw methods.
// A standard Go image is rendered upright on a standard surface.
// Set flipY to true to flip.
func (dw *DrawerBase) SetGoImage(idx, layer int, img image.Image, flipY bool) {
	for len(dw.Images) <= idx {
		dw.Images = append(dw.Images, nil)
	}
	imgs := &dw.Images[idx]
	for len(*imgs) <= layer {
		*imgs = append(*imgs, nil)
	}
	(*imgs)[layer] = img.(*image.RGBA)
}

// ConfigImageDefaultFormat configures the draw image at the given index
// to fit the default image format specified by the given width, height,
// and number of layers.
func (dw *DrawerBase) ConfigImageDefaultFormat(idx int, width int, height int, layers int) {
	// no-op
}

// SyncImages must be called after images have been updated, to sync
// memory up to the GPU.
func (dw *DrawerBase) SyncImages() {
	// no-op
}

// Copy copies texture at given index and layer to render target.
// dp is the destination point,
// sr is the source region (set to image.ZR zero rect for all),
// op is the drawing operation: Src = copy source directly (blit),
// Over = alpha blend with existing
// flipY = flipY axis when drawing this image
func (dw *DrawerBase) Copy(idx, layer int, dp image.Point, sr image.Rectangle, op draw.Op, flipY bool) error {
	img := dw.Images[idx][layer]
	draw.Draw(dw.Image, image.Rectangle{dp, dp.Add(img.Rect.Size())}, img, sr.Min, op)
	return nil
}

// Scale copies texture at given index and layer to render target,
// scaling the region defined by src and sr to the destination
// such that sr in src-space is mapped to dr in dst-space.
// dr is the destination rectangle
// sr is the source region (set to image.ZR zero rect for all),
// op is the drawing operation: Src = copy source directly (blit),
// Over = alpha blend with existing
// flipY = flipY axis when drawing this image
// rotDeg = rotation degrees to apply in the mapping, e.g., 90
// rotates 90 degrees to the left, -90 = right.
func (dw *DrawerBase) Scale(idx, layer int, dr image.Rectangle, sr image.Rectangle, op draw.Op, flipY bool, rotDeg float32) error {
	img := dw.Images[idx][layer]
	// todo: this needs to implement Scale!
	draw.Draw(dw.Image, dr, img, sr.Min, op)
	return nil
}

// UseTextureSet selects the descriptor set to use --
// choose this based on the bank of 16
// texture values if number of textures > MaxTexturesPerSet.
func (dw *DrawerBase) UseTextureSet(descIndex int) {
	// no-op
}

// StartDraw starts image drawing rendering process on render target
// No images can be added or set after this point.
// descIndex is the descriptor set to use -- choose this based on the bank of 16
// texture values if number of textures > MaxTexturesPerSet.
// This is a no-op on DrawerBase; if rendering logic is done here instead of
// EndDraw, everything is delayed by one render because Scale and Copy are
// called after StartDraw but before EndDraw, and we need them to be called
// before actually rendering the image to the capture channel.
// It returns false if rendering can not proceed.
func (dw *DrawerBase) StartDraw(descIndex int) bool {
	// no-op
	return true
}

// Fill fills given color to to render target.
// src2dst is the transform mapping source to destination
// coordinates (translation, scaling),
// reg is the region to fill
// op is the drawing operation: Src = copy source directly (blit),
// Over = alpha blend with existing
func (dw *DrawerBase) Fill(clr color.Color, src2dst math32.Mat3, reg image.Rectangle, op draw.Op) error {
	draw.Draw(dw.Image, reg, image.NewUniform(clr), image.Point{}, op)
	return nil
}

// StartFill starts color fill drawing rendering process on render target.
// It returns false if rendering can not proceed.
func (dw *DrawerBase) StartFill() bool {
	// no-op
	return true
}

// EndFill ends color filling rendering process on render target
func (dw *DrawerBase) EndFill() {
	// no-op
}

func (dw *DrawerBase) Surface() any {
	// no-op
	return nil
}

// SetFrameImage does direct rendering from a *vgpu.Framebuffer image.
// This is much more efficient for GPU-resident images, as in 3D or video.
func (dw *DrawerBase) SetFrameImage(idx int, fb any) {
	// no-op
}
