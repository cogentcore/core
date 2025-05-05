// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package imagex

import (
	"image"
	"image/draw"
)

// Wrapped extends the image.Image interface with two methods that manage
// the wrapping of an underlying Go image.Image. This can be used for images that
// are actually GPU textures, and to manage JavaScript pointers on the js platform.
type Wrapped interface {
	image.Image

	// Update is called whenever the image data has been updated,
	// to update any additional data based on the new image.
	// This may copy an image to the GPU or update JavaScript pointers.
	Update()

	// Underlying returns the underlying image.Image, which should
	// be called whenever passing the image to some other Go-based
	// function that is likely to be optimized for different image types,
	// such as draw.Draw. Do NOT use this for functions that will
	// directly handle the wrapped image!
	Underlying() image.Image
}

// Update calls Update on a wrapped [imagex.Wrapped] if it is one.
func Update(src image.Image) {
	if im, ok := src.(Wrapped); ok {
		im.Update()
	}
}

// Unwrap calls Underlying on a wrapped [imagex.Wrapped] if it is one.
func Unwrap(src image.Image) image.Image {
	if im, ok := src.(Wrapped); ok {
		return im.Underlying()
	}
	return src
}

// CloneAsRGBA returns an RGBA copy of the supplied image.
// Unwraps imagex.Wrapped wrapped images.
func CloneAsRGBA(src image.Image) *image.RGBA {
	ui := Unwrap(src)
	if ui == nil {
		return nil
	}
	bounds := ui.Bounds()
	img := image.NewRGBA(bounds)
	draw.Draw(img, bounds, ui, bounds.Min, draw.Src)
	return img
}

// AsRGBA returns the image as an RGBA: if it already is one, then
// it returns that image directly. Otherwise it returns a clone.
// Unwraps imagex.Wrapped wrapped images.
func AsRGBA(src image.Image) *image.RGBA {
	if src == nil {
		return nil
	}
	ui := Unwrap(src)
	if rgba, ok := ui.(*image.RGBA); ok {
		return rgba
	}
	return CloneAsRGBA(ui)
}

// note: defined in _js and _notjs:

// WrapJS returns a JavaScript optimized wrapper around the given
// image.Image on web platform, and just returns the image otherwise.
// func WrapJS(src image.Image) image.Image
