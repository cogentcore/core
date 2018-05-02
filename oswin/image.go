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
)

// Image is an in-memory pixel buffer. Its pixels can be modified by any Go
// code that takes an *image.RGBA, such as the standard library's image/draw
// package. A Image is essentially an *image.RGBA, but not all *image.RGBA
// values (including those returned by image.NewRGBA) are valid Images, as a
// driver may assume that the memory backing a Image's pixels are specially
// allocated.
//
// To see a Image's contents on a screen, upload it to a Texture (and then
// draw the Texture on a Window) or upload it directly to a Window.
//
// When specifying a sub-Image via Upload, a Image's top-left pixel is always
// (0, 0) in its own coordinate space.
type Image interface {
	// Release releases the Image's resources, after all pending uploads and
	// draws resolve.
	//
	// The behavior of the Image after Release, whether calling its methods or
	// passing it as an argument, is undefined.
	Release()

	// Size returns the size of the Image's image.
	Size() image.Point

	// Bounds returns the bounds of the Image's image. It is equal to
	// image.Rectangle{Max: b.Size()}.
	Bounds() image.Rectangle

	// RGBA returns the pixel buffer as an *image.RGBA.
	//
	// Its contents should not be accessed while the Image is uploading.
	//
	// The contents of the returned *image.RGBA's Pix field (of type []byte)
	// can be modified at other times, but that Pix slice itself (i.e. its
	// underlying pointer, length and capacity) should not be modified at any
	// time.
	//
	// The following is valid:
	//	m := buffer.RGBA()
	//	if len(m.Pix) >= 4 {
	//		m.Pix[0] = 0xff
	//		m.Pix[1] = 0x00
	//		m.Pix[2] = 0x00
	//		m.Pix[3] = 0xff
	//	}
	// or, equivalently:
	//	m := buffer.RGBA()
	//	m.SetRGBA(m.Rect.Min.X, m.Rect.Min.Y, color.RGBA{0xff, 0x00, 0x00, 0xff})
	// and using the standard library's image/draw package is also valid:
	//	dst := buffer.RGBA()
	//	draw.Draw(dst, dst.Bounds(), etc)
	// but the following is invalid:
	//	m := buffer.RGBA()
	//	m.Pix = anotherByteSlice
	// and so is this:
	//	*buffer.RGBA() = anotherImageRGBA
	RGBA() *image.RGBA
}

// Uploader is something you can upload a Image to.
type Uploader interface {
	// Upload uploads the sub-Image defined by src and sr to the destination
	// (the method receiver), such that sr.Min in src-space aligns with dp in
	// dst-space. The destination's contents are overwritten; the draw operator
	// is implicitly draw.Src.
	//
	// It is valid to upload a Image while another upload of the same Image
	// is in progress, but a Image's image.RGBA pixel contents should not be
	// accessed while it is uploading. A Image is re-usable, in that its pixel
	// contents can be further modified, once all outstanding calls to Upload
	// have returned.
	//
	// TODO: make it optional that a Image's contents is preserved after
	// Upload? Undoing a swizzle is a non-trivial amount of work, and can be
	// redundant if the next paint cycle starts by clearing the buffer.
	//
	// When uploading to a Window, there will not be any visible effect until
	// Publish is called.
	Upload(dp image.Point, src Image, sr image.Rectangle)

	// Fill fills that part of the destination (the method receiver) defined by
	// dr with the given color.
	//
	// When filling a Window, there will not be any visible effect until
	// Publish is called.
	Fill(dr image.Rectangle, src color.Color, op draw.Op)
}

// TODO: have a Downloader interface? Not every graphical app needs to be
// interactive or involve a window. You could use the GPU for hardware-
// accelerated image manipulation: upload a buffer, do some texture ops, then
// download the result.
