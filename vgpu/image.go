// Copyright (c) 2022, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// This is initially adapted from https://github.com/vulkan-go/asche
// Copyright © 2017 Maxim Kupriianov <max@kc.vc>, under the MIT License

package vgpu

import (
	"image"

	vk "github.com/vulkan-go/vulkan"
)

// ImageFormat describes the size and format of an Image
type ImageFormat struct {
	Size   image.Point `desc:"Size of image"`
	Format vk.Format   `desc:"Image format -- FormatR8g8b8a8Srgb is a standard default"`
}

func (im *ImageFormat) SetSize(w, h int) {
	im.Size = image.Point{X: w, Y: h}
}

func (im *ImageFormat) Set(w, h int, ft vk.Format) {
	im.SetSize(w, h)
	im.Format = ft
}

// Size32 returns size as uint32 values
func (im *ImageFormat) Size32() (width, height uint32) {
	width = uint32(im.Size.X)
	height = uint32(im.Size.Y)
	return
}

// Image represents a vulkan image with an associated ImageView
// It owns the View but the Image itself is not owned and must
// be managed externally.
type Image struct {
	Format ImageFormat  `desc:"format & size of image"`
	Image  vk.Image     `desc:"vulkan image handle"`
	View   vk.ImageView `desc:"vulkan image view"`
	Dev    vk.Device    `desc:"keep track of device for destroying view"`
	Buff   MemBuff      `desc:"memory buffer for allocating image"`
}

// HasView returns true if the image is set and has a view
func (im *Image) HasView() bool {
	return im.View != nil
}

// SetImage sets a new image and generates a default 2D view
// based on existing format information (which must be set properly).
// Any exiting view is destroyed first.  Must pass the relevant device.
func (im *Image) SetImage(dev vk.Device, img vk.Image) {
	im.DestroyView()
	im.Image = img
	im.Dev = dev
	im.MakeStdView()
}

// MakeStdView makes a standard 2D image view, for current image,
// format, and device.
func (im *Image) MakeStdView() {
	var view vk.ImageView
	ret := vk.CreateImageView(im.Dev, &vk.ImageViewCreateInfo{
		SType:  vk.StructureTypeImageViewCreateInfo,
		Format: im.Format.Format,
		Components: vk.ComponentMapping{ // this is the default anyway
			R: vk.ComponentSwizzleIdentity,
			G: vk.ComponentSwizzleIdentity,
			B: vk.ComponentSwizzleIdentity,
			A: vk.ComponentSwizzleIdentity,
		},
		SubresourceRange: vk.ImageSubresourceRange{
			AspectMask: vk.ImageAspectFlags(vk.ImageAspectColorBit),
			LevelCount: 1,
			LayerCount: 1,
		},
		ViewType: vk.ImageViewType2d,
		Image:    im.Image,
	}, nil, &view)
	IfPanic(NewError(ret))
	im.View = view
}

// DestroyView destroys any existing view
func (im *Image) DestroyView() {
	if im.View == nil {
		return
	}
	vk.DestroyImageView(im.Dev, im.View, nil)
	im.View = nil
}

// Destroy destroys any existing view, nils fields
func (im *Image) Destroy() {
	im.DestroyView()
	im.Image = nil
	im.Dev = nil
}

// SetNil sets everything to nil, for shared image
func (im *Image) SetNil() {
	im.View = nil
	im.Image = nil
	im.Dev = nil
}

// SetSize allocates image of given size, in its own buffer.
// It only reallocates memory if new size is larger.
func (im *Image) SetSize(size image.Point) {
	// todo: allocate image!

	im.Format.Size = size
}
