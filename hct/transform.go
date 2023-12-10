// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package hct

import (
	"image/color"

	"goki.dev/cam/cam16"
	"goki.dev/mat32/v2"
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
// given absolute HCT hue amount (0-360, ranges enforced)
func Spin(c color.Color, amount float32) color.RGBA {
	h := FromColor(c)
	h.SetHue(h.Hue + amount)
	return h.AsRGBA()
}

// Blend returns a color that is the given percent blend between the first
// and second color; 10 = 10% of the second and 90% of the first, etc;
// blending is done directly on non-premultiplied HCT values, and
// a correctly premultiplied color is returned.
func Blend(pct float32, x, y color.Color) color.RGBA {
	// TODO(kai): finish Blend
	pct = mat32.Clamp(pct, 0, 100)
	amt := pct / 100

	hx := FromColor(x)
	hy := FromColor(y)

	cx := cam16.FromSRGB(hx.R, hx.G, hx.B)
	cy := cam16.FromSRGB(hy.R, hy.G, hy.B)

	xj, _, xa, xb := cx.UCS()
	yj, _, ya, yb := cy.UCS()

	j := yj + (xj-yj)*amt
	a := ya + (xa-ya)*amt
	b := yb + (xb-yb)*amt

	rc := cam16.FromUCS(j, a, b)
	_ = rc

	return color.RGBAModel.Convert(x).(color.RGBA)
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
