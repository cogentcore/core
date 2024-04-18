// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package hct

import (
	"image/color"

	"cogentcore.org/core/math32"
)

// Lighten returns a color that is lighter by the
// given absolute HCT tone amount (0-100, ranges enforced)
func Lighten(c color.Color, amount float32) color.RGBA {
	h := FromColor(c)
	h.SetTone(h.Tone + amount)
	return h.AsRGBA()
}

// Darken returns a color that is darker by the
// given absolute HCT tone amount (0-100, ranges enforced)
func Darken(c color.Color, amount float32) color.RGBA {
	h := FromColor(c)
	h.SetTone(h.Tone - amount)
	return h.AsRGBA()
}

// Highlight returns a color that is lighter or darker by the
// given absolute HCT tone amount (0-100, ranges enforced),
// making the color darker if it is light (tone >= 50) and
// lighter otherwise. It is the opposite of [Samelight].
func Highlight(c color.Color, amount float32) color.RGBA {
	h := FromColor(c)
	if h.Tone >= 50 {
		h.SetTone(h.Tone - amount)
	} else {
		h.SetTone(h.Tone + amount)
	}
	return h.AsRGBA()
}

// Samelight returns a color that is lighter or darker by the
// given absolute HCT tone amount (0-100, ranges enforced),
// making the color lighter if it is light (tone >= 50) and
// darker otherwise. It is the opposite of [Highlight].
func Samelight(c color.Color, amount float32) color.RGBA {
	h := FromColor(c)
	if h.Tone >= 50 {
		h.SetTone(h.Tone + amount)
	} else {
		h.SetTone(h.Tone - amount)
	}
	return h.AsRGBA()
}

// Saturate returns a color that is more saturated by the
// given absolute HCT chroma amount (0-max that depends
// on other params but is around 150, ranges enforced)
func Saturate(c color.Color, amount float32) color.RGBA {
	h := FromColor(c)
	h.SetChroma(h.Chroma + amount)
	return h.AsRGBA()
}

// Desaturate returns a color that is less saturated by the
// given absolute HCT chroma amount (0-max that depends
// on other params but is around 150, ranges enforced)
func Desaturate(c color.Color, amount float32) color.RGBA {
	h := FromColor(c)
	h.SetChroma(h.Chroma - amount)
	return h.AsRGBA()
}

// Spin returns a color that has a different hue by the
// given absolute HCT hue amount (±0-360, ranges enforced)
func Spin(c color.Color, amount float32) color.RGBA {
	h := FromColor(c)
	h.SetHue(h.Hue + amount)
	return h.AsRGBA()
}

// MinHueDistance finds the minimum distance between two hues.
// A positive number means add to a to get to b.
// A negative number means subtract from a to get to b.
func MinHueDistance(a, b float32) float32 {
	d1 := b - a
	d2 := (b + 360) - a
	d3 := (b - (a + 360))
	d1a := math32.Abs(d1)
	d2a := math32.Abs(d2)
	d3a := math32.Abs(d3)
	if d1a < d2a && d1a < d3a {
		return d1
	}
	if d2a < d1a && d2a < d3a {
		return d2
	}
	return d3
}

// Blend returns a color that is the given percent blend between the first
// and second color; 10 = 10% of the first and 90% of the second, etc;
// blending is done directly on non-premultiplied HCT values, and
// a correctly premultiplied color is returned.
func Blend(pct float32, x, y color.Color) color.RGBA {
	hx := FromColor(x)
	hy := FromColor(y)
	pct = math32.Clamp(pct, 0, 100)
	px := pct / 100
	py := 1 - px

	dhue := MinHueDistance(hx.Hue, hy.Hue)

	// weight as a function of chroma strength: if near grey, hue is unreliable
	cpy := py * hy.Chroma / (px*hx.Chroma + py*hy.Chroma)
	hue := hx.Hue + cpy*dhue

	chroma := px*hx.Chroma + py*hy.Chroma
	tone := px*hx.Tone + py*hy.Tone
	hr := New(hue, chroma, tone)
	hr.A = px*hx.A + py*hy.A
	return hr.AsRGBA()
}

// IsLight returns whether the given color is light
// (has an HCT tone greater than or equal to 50)
func IsLight(c color.Color) bool {
	h := FromColor(c)
	return h.Tone >= 50
}

// IsDark returns whether the given color is dark
// (has an HCT tone less than 50)
func IsDark(c color.Color) bool {
	h := FromColor(c)
	return h.Tone < 50
}
