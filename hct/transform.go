// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package hct

import (
	"image/color"
)

// Lighten returns a color that is lighter by the
// given absolute HCT tone amount (0-100, ranges enforced)
func Lighten(c color.Color, amount float32) color.RGBA {
	h := Uint32ToHCT(c.RGBA())
	h.SetTone(h.Tone + amount)
	return h.AsRGBA()
}

// Darken returns a color that is darker by the
// given absolute HCT tone amount (0-100, ranges enforced)
func Darken(c color.Color, amount float32) color.RGBA {
	h := Uint32ToHCT(c.RGBA())
	h.SetTone(h.Tone - amount)
	return h.AsRGBA()
}

/*

// Highlight returns a color that is lighter or darker by the
// given absolute HCT tone (0-100), based on whether the color is already
// light or dark; if the color is lighter than or equal to 50%,
// it becomes darker, and if it is darker than 50%, it becomes lighter.
// For example, 50 = 50% lighter or darker, relative to the maximum
// possible lightness; it converts to HSL, adds or subtracts
// from the L factor, and then converts back to RGBA.
func (c Color) Highlight(pct float32) Color {
	hsl := HSLAModel.Convert(c).(HSLA)
	if hsl.L >= .5 {
		hsl.L -= pct / 100
	} else {
		hsl.L += pct / 100
	}
	hsl.L = mat32.Clamp(hsl.L, 0, 1)
	return ColorModel.Convert(hsl).(Color)
}

// Samelight is the opposite of [Color.Highlight];
// it makes a color darker if it is already
// darker than 50%, and lighter if already
// lighter than or equal to 50%
func (c Color) Samelight(pct float32) Color {
	hsl := HSLAModel.Convert(c).(HSLA)
	if hsl.L >= .5 {
		hsl.L += pct / 100
	} else {
		hsl.L -= pct / 100
	}
	hsl.L = mat32.Clamp(hsl.L, 0, 1)
	return ColorModel.Convert(hsl).(Color)
}

// Saturate returns a color that is more saturated by the
// given absolute HSL percent, e.g., 50 = 50%
// more saturated, relative to the maximum possible saturation;
// it converts to HSL, adds to the S factor,
// and then converts back to RGBA.
func (c Color) Saturate(pct float32) Color {
	hsl := HSLAModel.Convert(c).(HSLA)
	hsl.S += pct / 100
	hsl.S = mat32.Clamp(hsl.S, 0, 1)
	return ColorModel.Convert(hsl).(Color)
}

// Pastel returns a color that is less saturated (more pastel-like) by the
// given absolute HSL percent, e.g., 50 = 50%
// less saturated, relative to the maximum possible saturation;
// it converts to HSL, subtracts from the S factor,
// and then converts back to RGBA.
func (c Color) Pastel(pct float32) Color {
	hsl := HSLAModel.Convert(c).(HSLA)
	hsl.S -= pct / 100
	hsl.S = mat32.Clamp(hsl.S, 0, 1)
	return ColorModel.Convert(hsl).(Color)
}

// Clearer returns a color that is the given percent
// more transparent (lower alpha value)
// relative to the maximum possible alpha level
func (c Color) Clearer(pct float32) Color {
	f32 := NRGBAf32Model.Convert(c).(NRGBAf32)
	f32.A -= pct / 100
	f32.A = mat32.Clamp(f32.A, 0, 1)
	return ColorModel.Convert(f32).(Color)
}

// Opaquer returns a color that is the given percent
// more opaque (higher alpha value)
// relative to the maximum possible alpha level
func (c Color) Opaquer(pct float32) Color {
	f32 := NRGBAf32Model.Convert(c).(NRGBAf32)
	f32.A += pct / 100
	f32.A = mat32.Clamp(f32.A, 0, 1)
	return ColorModel.Convert(f32).(Color)
}
*/
