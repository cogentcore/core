// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build !js

package imagex

import (
	"image"
	"image/draw"

	"github.com/anthonynsimon/bild/transform"
)

// WrapJS returns a JavaScript optimized wrapper around the given
// [image.Image] on web, and just returns the image on other platforms.
func WrapJS(src image.Image) image.Image {
	return src
}

// Resize returns a resized version of the source image (which can be
// [Wrapped]), returning a [WrapJS] image handle on web and using web-native
// optimized code. Otherwise, uses medium quality Linear resize.
func Resize(src image.Image, size image.Point) image.Image {
	return transform.Resize(src, size.X, size.Y, transform.Linear)
}

// Crop returns a cropped region of the source image (which can be
// [Wrapped]), returning a [WrapJS] image handle on web and using web-native
// optimized code.
func Crop(src image.Image, rect image.Rectangle) image.Image {
	dst := image.NewRGBA(rect)
	draw.Draw(dst, rect, src, image.Point{}, draw.Src)
	return dst
}
