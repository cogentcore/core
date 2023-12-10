// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package colors

import (
	"image/color"

	"goki.dev/cam/hct"
	"goki.dev/colors/matcolor"
)

// List returns a list of n colors with the given HCT chroma and tone
// and varying hues spaced equally in order to minimize the number of similar colors.
// This can be useful for automatically generating colors for things like graph lines.
func List(n int, chroma float32, tone float32) []color.RGBA {
	res := []color.RGBA{}
	if n == 0 {
		return res
	}
	inc := 360 / float32(min(n, 6))
	for i := 0; i < n; i++ {
		h := hct.New(float32(i)*inc, chroma, tone)
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
