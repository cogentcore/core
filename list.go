// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package colors

import (
	"image/color"
	"math"

	"goki.dev/cam/hct"
	"goki.dev/colors/matcolor"
	"goki.dev/mat32/v2"
)

// BinarySpacedNumber returns a floating point number in the 0-1 range based on the
// binary representation of the given input number, such that the biggest differences
// are in the lowest-order bits, with progressively smaller differences for higher powers.
// 0 = 0; 1 = 0.5; 2 = 0.25; 3 = 0.75; 4 = 0.125; 5 = 0.625...
func BinarySpacedNumber(idx int) float32 {
	nb := int(mat32.Ceil(mat32.Log(float32(idx)) / math.Ln2))
	rv := float32(0)
	for i := 0; i <= nb; i++ {
		pbase := 1 << i
		base := 1 << (i + 1)
		dv := (idx % base) / pbase
		iv := float32(dv) * (1 / float32(base))
		rv += iv
	}
	return rv
}

// BinarySpacedColor returns a maximally widely-spaced sequence of colors
// for prgressive values of the index, using the Hue value of the HCT space.
// This is useful for assigning colors in graphs etc.
func BinarySpacedColor(idx int, chroma, tone float32) color.RGBA {
	h := hct.New(360*BinarySpacedNumber(idx), chroma, tone)
	return h.AsRGBA()
}

// List returns a list of n colors with the given HCT chroma and tone
// and varying hues spaced equally in order to minimize the number of similar colors.
// This can be useful for automatically generating colors for things like graph lines.
func List(n int, chroma float32, tone float32) []color.RGBA {
	res := []color.RGBA{}
	if n == 0 {
		return res
	}
	fn := float32(n)
	inc := 360 / float32(min(n, 6))
	for i := float32(0); i < fn; i++ {
		hue := float32(i) * inc
		hue -= mat32.Mod(hue, 360) * (i / fn)

		h := hct.New(hue, chroma, tone)
		res = append(res, h.AsRGBA())
	}
	return res
}

// AccentList calls [List] with standard chroma and tone values that will result
// in matcolor-style base accent colors appropriate for the current color theme
// (light vs dark). These colors will satisfy text contrast requirements when placed
// on standard scheme backgrounds.
func AccentList(n int) []color.RGBA {
	if matcolor.SchemeIsDark {
		return List(n, 48, 80)
	}
	return List(n, 48, 40)
}

// AccentVariantList calls [List] with standard chroma and tone values that will result
// in variant versions of matcolor-style base accent colors appropriate for the current
// color theme (light vs dark). These colors will not necessarily satisfy text contrast
// requirements, and they are designed for things like graph lines that do not need to
// stand out as much.
func AccentVariantList(n int) []color.RGBA {
	if matcolor.SchemeIsDark {
		return List(n, 48, 60)
	}
	return List(n, 48, 50)
}
