// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package colors

import (
	"image"
	"image/color"
	"math"
)

// Uniform returns a new uniform [image.Image] filled completely with the given color.
func Uniform(c color.Color) image.Image {
	return image.NewUniform(c)
}

// ToUniform converts the given image to a uniform color.
func ToUniform(img image.Image) color.Color {
	// TODO(kai/imageColor): we probably shouldn't need this anymore once everyone is an image
	if img == nil {
		return nil
	}
	return img.At(0, 0)
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
	return image.Rect(math.MinInt, math.MinInt, math.MaxInt, math.MaxInt)
}

func (p *pattern) At(x, y int) color.Color {
	return p.f(x, y)
}
