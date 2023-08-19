// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package hsl

import (
	"image/color"

	"github.com/goki/mat32"
)

// Lighten returns a color that is lighter by the
// given absolute HSL lightness amount (0-100, ranges enforced)
func Lighten(c color.Color, amount float32) color.RGBA {
	h := FromColor(c)
	h.L += amount / 100
	h.L = mat32.Clamp(h.L, 0, 100)
	return h.AsRGBA()
}

// Darken returns a color that is darker by the
// given absolute HSL lightness amount (0-100, ranges enforced)
func Darken(c color.Color, amount float32) color.RGBA {
	h := FromColor(c)
	h.L -= amount / 100
	h.L = mat32.Clamp(h.L, 0, 100)
	return h.AsRGBA()
}

// Highlight returns a color that is lighter or darker by the
// given absolute HSL lightness amount (0-100, ranges enforced),
// making the color darker if it is light (tone >= 0.5) and
// lighter otherwise. It is the opposite of [Samelight].
func Highlight(c color.Color, amount float32) color.RGBA {
	h := FromColor(c)
	if h.L >= 0.5 {
		h.L -= amount / 100
	} else {
		h.L += amount / 100
	}
	h.L = mat32.Clamp(h.L, 0, 100)
	return h.AsRGBA()
}

// Samelight returns a color that is lighter or darker by the
// given absolute HSL lightness amount (0-100, ranges enforced),
// making the color lighter if it is light (tone >= 0.5) and
// darker otherwise. It is the opposite of [Highlight].
func Samelight(c color.Color, amount float32) color.RGBA {
	h := FromColor(c)
	if h.L >= 0.5 {
		h.L += amount
	} else {
		h.L -= amount
	}
	h.L = mat32.Clamp(h.L, 0, 100)
	return h.AsRGBA()
}

// Saturate returns a color that is more saturated by the
// given absolute HSL saturation amount (0-100, ranges enforced)
func Saturate(c color.Color, amount float32) color.RGBA {
	h := FromColor(c)
	h.S += amount
	h.S = mat32.Clamp(h.S, 0, 100)
	return h.AsRGBA()
}

// Desaturate returns a color that is less saturated by the
// given absolute HSL saturation amount (0-100, ranges enforced)
func Desaturate(c color.Color, amount float32) color.RGBA {
	h := FromColor(c)
	h.S -= amount
	h.S = mat32.Clamp(h.S, 0, 100)
	return h.AsRGBA()
}

// Spin returns a color that has a different hue by the
// given absolute HSL hue amount (0-360, ranges enforced)
func Spin(c color.Color, amount float32) color.RGBA {
	h := FromColor(c)
	h.H += amount
	h.H = mat32.Clamp(h.H, 0, 360)
	return h.AsRGBA()
}

// IsLight returns whether the given color is light
// (has an HSL lightness greater than or equal to 0.6)
func IsLight(c color.Color) bool {
	h := FromColor(c)
	return h.L >= 0.6
}

// IsDark returns whether the given color is dark
// (has an HSL lightness less than 0.6)
func IsDark(c color.Color) bool {
	h := FromColor(c)
	return h.L < 0.6
}

// ContrastColor returns the color that should
// be used to contrast this color (white or black),
// based on the result of [IsLight].
func ContrastColor(c color.Color) color.RGBA {
	if IsLight(c) {
		return color.RGBA{0, 0, 0, 255}
	}
	return color.RGBA{255, 255, 255, 255}
}
