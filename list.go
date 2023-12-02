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
func List(n int, chroma float32, tone float32) []color.RGBA {
	res := []color.RGBA{}
	if n == 0 {
		return res
	}
	inc := 360 / float32(n)
	for i := float32(0); i < 360; i += inc {
		h := hct.New(i, chroma, tone)
		res = append(res, h.AsRGBA())
	}
	return res
}

// AccentList calls [List] with standard chroma and tone values that will result
// in matcolor-style base accent colors appropriate for the current color theme
// (light vs dark).
func AccentList(n int) []color.RGBA {
	if matcolor.SchemeIsDark {
		return List(n, 48, 80)
	}
	return List(n, 48, 40)
}
