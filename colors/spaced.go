// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package colors

import (
	"image/color"

	"cogentcore.org/core/colors/cam/hct"
	"cogentcore.org/core/colors/matcolor"
)

// Spaced returns a maximally widely spaced sequence of colors
// for progressive values of the index, using the HCT space.
// This is useful, for example, for assigning colors in graphs.
func Spaced(idx int) color.RGBA {
	if matcolor.SchemeIsDark {
		return spacedDark(idx)
	}
	return spacedLight(idx)
}

// spacedLight is the light mode version of [Spaced].
func spacedLight(idx int) color.RGBA {
	// blue, red, green, yellow, violet, aqua, orange, blueviolet
	// hues := []float32{30, 280, 140, 110, 330, 200, 70, 305}
	hues := []float32{255, 25, 150, 105, 340, 210, 60, 300}
	// even 45:       30, 75, 120, 165, 210, 255, 300, 345,
	toffs := []float32{0, -10, 0, 5, 0, 0, 5, 0}
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

// spacedDark is the dark mode version of [Spaced].
func spacedDark(idx int) color.RGBA {
	// blue, red, green, yellow, violet, aqua, orange, blueviolet
	// hues := []float32{30, 280, 140, 110, 330, 200, 70, 305}
	hues := []float32{255, 25, 150, 105, 340, 210, 60, 300}
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
