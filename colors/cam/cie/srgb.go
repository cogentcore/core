// Copyright (c) 2021, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package cie

import "cogentcore.org/core/math32"

// SRGBToLinearComp converts an sRGB rgb component to linear space (removes gamma).
// Used in converting from sRGB to XYZ colors.
func SRGBToLinearComp(srgb float32) float32 {
	if srgb <= 0.04045 {
		return srgb / 12.92
	}
	return math32.Pow((srgb+0.055)/1.055, 2.4)
}

// SRGBFromLinearComp converts an sRGB rgb linear component
// to non-linear (gamma corrected) sRGB value
// Used in converting from XYZ to sRGB.
func SRGBFromLinearComp(lin float32) float32 {
	var gv float32
	if lin <= 0.0031308 {
		gv = 12.92 * lin
	} else {
		gv = (1.055*math32.Pow(lin, 1.0/2.4) - 0.055)
	}
	return math32.Clamp(gv, 0, 1)
}

// SRGBToLinear converts set of sRGB components to linear values,
// removing gamma correction.
func SRGBToLinear(r, g, b float32) (rl, gl, bl float32) {
	rl = SRGBToLinearComp(r)
	gl = SRGBToLinearComp(g)
	bl = SRGBToLinearComp(b)
	return
}

// SRGB100ToLinear converts set of sRGB components to linear values,
// removing gamma correction.  returns 100-base RGB values
func SRGB100ToLinear(r, g, b float32) (rl, gl, bl float32) {
	rl = 100 * SRGBToLinearComp(r)
	gl = 100 * SRGBToLinearComp(g)
	bl = 100 * SRGBToLinearComp(b)
	return
}

// SRGBFromLinear converts set of sRGB components from linear values,
// adding gamma correction.
func SRGBFromLinear(rl, gl, bl float32) (r, g, b float32) {
	r = SRGBFromLinearComp(rl)
	g = SRGBFromLinearComp(gl)
	b = SRGBFromLinearComp(bl)
	return
}

// SRGBFromLinear100 converts set of sRGB components from linear values in 0-100 range,
// adding gamma correction.
func SRGBFromLinear100(rl, gl, bl float32) (r, g, b float32) {
	r = SRGBFromLinearComp(rl / 100)
	g = SRGBFromLinearComp(gl / 100)
	b = SRGBFromLinearComp(bl / 100)
	return
}

// SRGBFloatToUint8 converts the given non-alpha-premuntiplied sRGB float32
// values to alpha-premultiplied sRGB uint8 values.
func SRGBFloatToUint8(rf, gf, bf, af float32) (r, g, b, a uint8) {
	r = uint8(rf*af*255 + 0.5)
	g = uint8(gf*af*255 + 0.5)
	b = uint8(bf*af*255 + 0.5)
	a = uint8(af*255 + 0.5)
	return
}

// SRGBFloatToUint32 converts the given non-alpha-premuntiplied sRGB float32
// values to alpha-premultiplied sRGB uint32 values.
func SRGBFloatToUint32(rf, gf, bf, af float32) (r, g, b, a uint32) {
	r = uint32(rf*af*65535 + 0.5)
	g = uint32(gf*af*65535 + 0.5)
	b = uint32(bf*af*65535 + 0.5)
	a = uint32(af*65535 + 0.5)
	return
}

// SRGBUint8ToFloat converts the given alpha-premultiplied sRGB uint8 values
// to non-alpha-premuntiplied sRGB float32 values.
func SRGBUint8ToFloat(r, g, b, a uint8) (fr, fg, fb, fa float32) {
	fa = float32(a) / 255
	fr = (float32(r) / 255) / fa
	fg = (float32(g) / 255) / fa
	fb = (float32(b) / 255) / fa
	return
}

// SRGBUint32ToFloat converts the given alpha-premultiplied sRGB uint32 values
// to non-alpha-premuntiplied sRGB float32 values.
func SRGBUint32ToFloat(r, g, b, a uint32) (fr, fg, fb, fa float32) {
	fa = float32(a) / 65535
	fr = (float32(r) / 65535) / fa
	fg = (float32(g) / 65535) / fa
	fb = (float32(b) / 65535) / fa
	return
}
