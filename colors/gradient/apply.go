// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Based on https://github.com/srwiley/rasterx:
// Copyright 2018 by the rasterx Authors. All rights reserved.
// Created 2018 by S.R.Wiley

package gradient

import (
	"image"
	"image/color"

	"cogentcore.org/core/colors"
	"github.com/anthonynsimon/bild/adjust"
)

// ApplyOpacity applies the given opacity to the given image, handling
// [image.Uniform] and [Gradient] as special cases.
func ApplyOpacity(img image.Image, opacity float32) image.Image {
	return Apply(img, func(c color.RGBA) color.RGBA {
		return colors.ApplyOpacity(c, opacity)
	})
}

// Apply returns a copy of the given image with the given color function
// applied to each pixel of the image. It handles [image.Uniform] and
// [Gradient] as special cases, only calling the function for the uniform
// color and each stop color, respectively.
func Apply(img image.Image, f func(c color.RGBA) color.RGBA) image.Image {
	if img == nil {
		return nil
	}
	switch img := img.(type) {
	case *image.Uniform:
		return image.NewUniform(f(colors.AsRGBA(img)))
	case Gradient:
		res := CopyOf(img)
		gb := res.AsBase()
		for i, s := range gb.Stops {
			s.Color = f(colors.AsRGBA(s.Color))
			gb.Stops[i] = s
		}
		return res
	default:
		return adjust.Apply(img, f)
	}
}
