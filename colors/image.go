// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package colors

import (
	"image"
	"image/color"
)

// Uniform returns a new [image.Uniform] filled completely with the given color.
// See [ToUniform] for the converse.
func Uniform(c color.Color) image.Image {
	if c == nil {
		return nil
	}
	return image.NewUniform(c)
}

// ToUniform converts the given image to a uniform [color.RGBA] color.
// See [Uniform] for the converse.
func ToUniform(img image.Image) color.RGBA {
	if img == nil {
		return color.RGBA{}
	}
	return AsRGBA(img.At(0, 0))
}

// CloneUniform returns a copy of a Uniform image color, if image is one.
// Otherwise just returns the image back. This is useful when modifying
// an image color, to ensure that it doesn't modify all users of this image.
func CloneUniform(img image.Image) image.Image {
	if img == nil {
		return nil
	}
	u, ok := img.(*image.Uniform)
	if !ok {
		return img
	}
	return image.NewUniform(AsRGBA(u.C))
}

// Pattern returns a new unbounded [image.Image] represented by the given pattern function.
func Pattern(f func(x, y int) color.Color) image.Image {
	return &pattern{f}
}

type pattern struct {
	f func(x, y int) color.Color
}

func (p *pattern) ColorModel() color.Model {
	return color.RGBAModel
}

func (p *pattern) Bounds() image.Rectangle {
	return image.Rect(-1e9, -1e9, 1e9, 1e9)
}

func (p *pattern) At(x, y int) color.Color {
	return p.f(x, y)
}
