// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gpu

import (
	"fmt"
	"image"
	"log"

	"github.com/cogentcore/webgpu/wgpu"
)

// TextureFormat describes the size and WebGPU format of a Texture.
// If Layers > 1, all must be the same size.
type TextureFormat struct {
	// Size of image
	Size image.Point

	// Texture format: RGBA8UnormSrgb is default
	Format wgpu.TextureFormat

	// number of samples. set higher for RenderTexture rendering
	// but otherwise default of 1
	Samples int

	// number of layers for texture arrays
	Layers int
}

// NewTextureFormat returns a new TextureFormat with default format and given size
// and number of layers
func NewTextureFormat(width, height, layers int) *TextureFormat {
	im := &TextureFormat{}
	im.Defaults()
	im.Size = image.Point{width, height}
	im.Layers = layers
	return im
}

func (im *TextureFormat) Defaults() {
	im.Format = wgpu.TextureFormatRGBA8UnormSrgb
	im.Samples = 1
	im.Layers = 1
}

// String returns human-readable version of format
func (im *TextureFormat) String() string {
	return fmt.Sprintf("Size: %v  Format: %s  MultiSample: %d  Layers: %d", im.Size, TextureFormatNames[im.Format], im.Samples, im.Layers)
}

// IsStdRGBA returns true if image format is the standard
// wgpu.TextureFormatRGBA8UnormSrgb
// which is compatible with go image.RGBA format.
func (im *TextureFormat) IsStdRGBA() bool {
	return im.Format == wgpu.TextureFormatRGBA8UnormSrgb
}

// IsRGBAUnorm returns true if image format is the
// wgpu.TextureFormatRGBA8Unorm format
// which is compatible with go image.RGBA format with colorspace conversion.
func (im *TextureFormat) IsRGBAUnorm() bool {
	return im.Format == wgpu.TextureFormatRGBA8Unorm
}

// SetSize sets the width, height
func (im *TextureFormat) SetSize(w, h int) {
	im.Size = image.Point{X: w, Y: h}
}

// Set sets width, height and format
func (im *TextureFormat) Set(w, h int, ft wgpu.TextureFormat) {
	im.SetSize(w, h)
	im.Format = ft
}

// SetFormat sets the format using vgpu standard Types
func (im *TextureFormat) SetFormat(ft Types) {
	im.Format = ft.TextureFormat()
}

// SetMultisample sets the number of multisampling to decrease aliasing
// 4 is typically sufficient.  Values must be power of 2.
func (im *TextureFormat) SetMultisample(nsamp int) {
	im.Samples = nsamp
}

// Size32 returns size as uint32 values
func (im *TextureFormat) Size32() (width, height uint32) {
	width = uint32(im.Size.X)
	height = uint32(im.Size.Y)
	return
}

func (im *TextureFormat) Extent3D() wgpu.Extent3D {
	ex := wgpu.Extent3D{
		Width:              uint32(im.Size.X),
		Height:             uint32(im.Size.Y),
		DepthOrArrayLayers: uint32(im.Layers),
	}
	return ex
}

// Aspect returns the aspect ratio X / Y
func (im *TextureFormat) Aspect() float32 {
	if im.Size.Y > 0 {
		return float32(im.Size.X) / float32(im.Size.Y)
	}
	return 1.3
}

// Bounds returns the rectangle defining this image: 0,0,w,h
func (im *TextureFormat) Bounds() image.Rectangle {
	return image.Rectangle{Max: im.Size}
}

// BytesPerPixel returns number of bytes required to represent
// one Pixel (in Host memory at least).  TODO only works
// for known formats -- need to add more as needed.
func (im *TextureFormat) BytesPerPixel() int {
	bpp := TextureFormatSizes[im.Format]
	if bpp > 0 {
		return bpp
	}
	log.Println("gpu.TextureFormat:BytesPerPixel() -- format not yet supported!")
	return 0
}

// LayerByteSize returns number of bytes required to represent one layer of
// image in Host memory.  TODO only works
// for known formats -- need to add more as needed.
func (im *TextureFormat) LayerByteSize() int {
	return im.BytesPerPixel() * im.Size.X * im.Size.Y
}

// TotalByteSize returns total number of bytes required to represent all layers of
// images in Host memory.  TODO only works
// for known formats -- need to add more as needed.
func (im *TextureFormat) TotalByteSize() int {
	return im.LayerByteSize() * im.Layers
}

// Stride returns number of bytes per image row.  TODO only works
// for known formats -- need to add more as needed.
func (im *TextureFormat) Stride() int {
	return im.BytesPerPixel() * im.Size.X
}

//////////////////////////////////////////////////////////////////////

// TextureBufferDims represents the sizes required in Buffer to
// represent a texture of a given size.
type TextureBufferDims struct {
	Width           uint64
	Height          uint64
	UnpaddedRowSize uint64
	PaddedRowSize   uint64
}

func NewTextureBufferDims(size image.Point) *TextureBufferDims {
	td := &TextureBufferDims{}
	td.Set(size)
	return td
}

func (td *TextureBufferDims) Set(size image.Point) {
	td.Width = uint64(size.X)
	td.Height = uint64(size.Y)
	const bytesPerPixel = 4 // unsafe.Sizeof(uint32(0))
	td.UnpaddedRowSize = uint64(td.Width * bytesPerPixel)
	align := uint64(wgpu.CopyBytesPerRowAlignment)
	padding := (align - td.UnpaddedRowSize%align) % align
	td.PaddedRowSize = td.UnpaddedRowSize + padding
}

// PaddedSize returns the total padded size of data
func (td *TextureBufferDims) PaddedSize() uint64 {
	return td.PaddedRowSize * td.Height
}

// UnpaddedSize returns the total unpadded size of data
func (td *TextureBufferDims) UnpaddedSize() uint64 {
	return td.UnpaddedRowSize * td.Height
}

// HasNoPadding returns true if the Unpadded and Padded row sizes
// are the same.
func (td *TextureBufferDims) HasNoPadding() bool {
	return td.UnpaddedRowSize == td.PaddedRowSize
}
