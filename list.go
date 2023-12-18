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

// Spaced returns a maximally widely-spaced sequence of colors
// for progressive values of the index, using the HCT space.
// This is useful, for example, for assigning colors in graphs.
func Spaced(idx int) color.RGBA {
	if matcolor.SchemeIsDark {
		return SpacedDark(idx)
	}
	return SpacedLight(idx)
}

// SpacedLight is the Light mode version of Spaced
func SpacedLight(idx int) color.RGBA {
	// red, blue, green, yellow, violet, aqua, orange, blueviolet
	// hues := []float32{30, 280, 140, 110, 330, 200, 70, 305}
	hues := []float32{25, 255, 150, 105, 340, 210, 60, 300}
	// even 45:       30, 75, 120, 165, 210, 255, 300, 345,
	toffs := []float32{0, -10, 0, 10, 0, 0, 5, 0}
	tones := []float32{65, 80, 45, 65, 80}
	chromas := []float32{90, 90, 90, 20, 20}
	ncats := len(hues)
	ntc := len(tones)
	hi := idx % ncats
	hr := idx / ncats
	tci := hr % ntc
	hue := hues[hi]
	tone := toffs[hi] + tones[tci]
	chroma := chromas[tci]
	return hct.New(hue, float32(chroma), tone).AsRGBA()
}

// SpacedDark is the Dark mode version of Spaced
func SpacedDark(idx int) color.RGBA {
	// red, blue, green, yellow, violet, aqua, orange, blueviolet
	// hues := []float32{30, 280, 140, 110, 330, 200, 70, 305}
	hues := []float32{25, 255, 150, 105, 340, 210, 60, 300}
	// even 45:       30, 75, 120, 165, 210, 255, 300, 345,
	toffs := []float32{0, -10, 0, 10, 0, 0, 5, 0}
	tones := []float32{65, 80, 45, 65, 80}
	chromas := []float32{90, 90, 90, 20, 20}
	ncats := len(hues)
	ntc := len(tones)
	hi := idx % ncats
	hr := idx / ncats
	tci := hr % ntc
	hue := hues[hi]
	tone := toffs[hi] + tones[tci]
	chroma := chromas[tci]
	return hct.New(hue, float32(chroma), tone).AsRGBA()
}
