// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package imagex

import (
	"image"

	"github.com/anthonynsimon/bild/clone"
)

// Wrapped extends the [image.Image] interface with two methods that manage
// the wrapping of an underlying Go [image.Image]. This can be used for images that
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
	// such as [draw.Draw]. Do NOT use this for functions that will
	// directly handle the wrapped image!
	Underlying() image.Image
}

// Update calls [Wrapped.Update] on a [Wrapped] if it is one.
// It does nothing otherwise.
func Update(src image.Image) {
	if wr, ok := src.(Wrapped); ok {
		wr.Update()
	}
}

// Unwrap calls [Wrapped.Underlying] on a [Wrapped] if it is one.
// It returns the original image otherwise.
func Unwrap(src image.Image) image.Image {
	if wr, ok := src.(Wrapped); ok {
		return wr.Underlying()
	}
	return src
}

// CloneAsRGBA returns an [*image.RGBA] copy of the supplied image.
// It calls [Unwrap] first. See also [AsRGBA].
func CloneAsRGBA(src image.Image) *image.RGBA {
	return clone.AsRGBA(Unwrap(src))
}

// AsRGBA returns the image as an [*image.RGBA]. If it already is one,
// it returns that image directly. Otherwise it returns a clone.
// It calls [Unwrap] first. See also [CloneAsRGBA].
func AsRGBA(src image.Image) *image.RGBA {
	return clone.AsShallowRGBA(Unwrap(src))
}
