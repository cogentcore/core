// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package colors

import (
	"image/color"

	"goki.dev/cam/hct"
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
