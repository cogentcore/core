// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gradient

import (
	"image"
	"image/color"

	"cogentcore.org/core/colors"
)

// ApplyFunc is a function that transforms input color to an output color.
type ApplyFunc func(c color.Color) color.Color

// ApplyFuncs is a slice of ApplyFunc color functions applied in order
type ApplyFuncs []ApplyFunc

// Add adds a new function
func (af *ApplyFuncs) Add(fun ApplyFunc) {
	*af = append(*af, fun)
}

// Apply applies all functions in order to given input color
func (af ApplyFuncs) Apply(c color.Color) color.Color {
	for _, f := range af {
		c = f(c)
	}
	return c
}

func (af ApplyFuncs) Clone() ApplyFuncs {
	n := len(af)
	if n == 0 {
		return nil
	}
	c := make(ApplyFuncs, n)
	copy(c, af)
	return c
}

// Applier is an image.Image wrapper that applies a color transformation
// to the output of a source image, using the given ApplyFunc
type Applier struct {
	image.Image
	Func ApplyFunc
}

// NewApplier returns a new applier for given image and apply function
func NewApplier(img image.Image, fun func(c color.Color) color.Color) *Applier {
	return &Applier{Image: img, Func: fun}
}

func (ap *Applier) At(x, y int) color.Color {
	return ap.Func(ap.Image.At(x, y))
}

// Apply returns a copy of the given image with the given color function
// applied to each pixel of the image, handling special cases:
// [image.Uniform] is optimized and must be preserved as such: color is directly updated.
// [gradient.Gradient] must have Update called prior to rendering, with
// the current bounding box.
func Apply(img image.Image, f ApplyFunc) image.Image {
	if img == nil {
		return nil
	}
	switch im := img.(type) {
	case *image.Uniform:
		return image.NewUniform(f(colors.AsRGBA(im)))
	case Gradient:
		cp := CopyOf(im)
		cp.AsBase().ApplyFuncs.Add(f)
		return cp
	default:
		return NewApplier(img, f)
	}
}

// ApplyOpacity applies the given opacity (0-1) to the given image,
// handling the following special cases, and using an Applier for the general case.
// [image.Uniform] is optimized and must be preserved as such: color is directly updated.
// [gradient.Gradient] must have Update called prior to rendering, with
// the current bounding box. Multiplies the opacity of the stops.
func ApplyOpacity(img image.Image, opacity float32) image.Image {
	if img == nil {
		return nil
	}
	if opacity == 1 {
		return img
	}
	switch im := img.(type) {
	case *image.Uniform:
		return image.NewUniform(colors.ApplyOpacity(colors.AsRGBA(im), opacity))
	case Gradient:
		cp := CopyOf(im)
		cp.AsBase().ApplyOpacityToStops(opacity)
		return cp
	default:
		return NewApplier(img, func(c color.Color) color.Color {
			return colors.ApplyOpacity(c, opacity)
		})
	}
}
