// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package hct

import (
	"image/color"
	"testing"

	"goki.dev/mat32/v2"
)

func TestContrastRatio(t *testing.T) {
	type data struct {
		a    color.Color
		b    color.Color
		want float32
	}
	tests := []data{
		{color.White, color.Black, 21},
		{color.Black, color.White, 21},
		{color.RGBA{100, 100, 100, 255}, color.RGBA{100, 100, 100, 255}, 1},
		{color.RGBA{0, 0, 255, 255}, color.RGBA{255, 255, 255, 255}, 8.59},
	}
	for i, test := range tests {
		res := ContrastRatio(test.a, test.b)
		if mat32.Abs(res-test.want) > 0.1 {
			t.Errorf("%d: expected %g but got %g", i, test.want, res)
		}
	}
}

func TestToneContrastRatio(t *testing.T) {
	type data struct {
		a    float32
		b    float32
		want float32
	}
	tests := []data{
		{0, 100, 21},
		{100, 0, 21},
		{50, 50, 1},
		{100, 32.302586, 8.59},
	}
	for i, test := range tests {
		res := ToneContrastRatio(test.a, test.b)
		if mat32.Abs(res-test.want) > 0.1 {
			t.Errorf("%d: expected %g but got %g", i, test.want, res)
		}
	}
}
