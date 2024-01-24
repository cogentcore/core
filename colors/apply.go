// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package colors

import (
	"image"
	"image/color"
)

// Applier is an image.Image wrapper that applies a color transformation
// to the output of a source image, using the given ApplyFunc
type Applier struct {
	image.Image
	ApplyFunc func(c color.Color) color.Color
}

// NewApplier returns a new applier for given image and apply function
func NewApplier(img image.Image, fun func(c color.Color) color.Color) *Applier {
	return &Applier{Image: img, ApplyFunc: fun}
}

func (ap *Applier) At(x, y int) color.Color {
	return ap.ApplyFunc(ap.Image.At(x, y))
}

// Apply returns a copy of the given image with the given color function
// applied to each pixel of the image. It handles [image.Uniform] and
// as a special case, only calling the function for the uniform color
func Apply(img image.Image, f func(c color.Color) color.Color) image.Image {
	if img == nil {
		return nil
	}
	switch img := img.(type) {
	case *image.Uniform:
		return image.NewUniform(f(AsRGBA(img)))
	default:
		return NewApplier(img, f)
	}
}

// ApplyOpacityImage applies the given opacity to the given image,
// handling [image.Uniform] as a special case, and using
// an Applier for the general case.  Gradients should get the
// opacity in their Update function for a more optimized floating-point
// integration with gradients of opacity (e.g., this happens in paint.Paint)
func ApplyOpacityImage(img image.Image, opacity float32) image.Image {
	if img == nil {
		return nil
	}
	switch img := img.(type) {
	case *image.Uniform:
		return image.NewUniform(ApplyOpacity(AsRGBA(img), opacity))
	default:
		return NewApplier(img, func(c color.Color) color.Color {
			return ApplyOpacity(c, opacity)
		})
	}
}
