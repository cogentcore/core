// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gradient

import (
	"image"
	"image/color"
	"testing"

	"goki.dev/colors"
	"goki.dev/mat32/v2"
)

func ExampleLinear() {
	NewLinear().AddStop(colors.White, 0).AddStop(colors.Black, 1)
}

func ExampleRadial() {
	NewRadial().AddStop(colors.Green, 0).AddStop(colors.Yellow, 0.5).AddStop(colors.Red, 1)
}

func TestColorAt(t *testing.T) {
	type value struct {
		x    int
		y    int
		want color.RGBA
	}
	type test struct {
		gr   Gradient
		want []value
	}
	tests := []test{
		{NewLinear().
			AddStop(colors.White, 0).
			AddStop(colors.Black, 1),
			[]value{
				{33, 71, color.RGBA{68, 67, 67, 255}},
				{78, 71, color.RGBA{68, 67, 67, 255}},
				{78, 17, color.RGBA{205, 205, 205, 255}},
				{33, 50, color.RGBA{118, 118, 117, 255}},
			}},
		{linearGoldRedTransformTest,
			[]value{
				{50, 50, color.RGBA{255, 141, 52, 255}},
				{7, 50, color.RGBA{255, 141, 52, 255}},
				{81, 23, color.RGBA{255, 185, 76, 255}},
				{81, 94, color.RGBA{254, 12, 0, 255}},
			}},
		{NewRadial().SetUnits(UserSpaceOnUse).
			SetCenter(mat32.V2(90, 50)).SetFocal(mat32.V2(90, 50)).
			AddStop(colors.Blue, 0).
			AddStop(colors.Yellow, 0.7),
			[]value{
				{90, 50, colors.Blue},
			}},
	}
	for i, test := range tests {
		for j, v := range test.want {
			have := test.gr.At(v.x, v.y)
			if have != v.want {
				t.Errorf("%d.%d: expected %v at %v but got %v", i, j, v.want, image.Pt(v.x, v.y), have)
			}
		}
	}
}
