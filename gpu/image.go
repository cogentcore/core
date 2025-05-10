// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gpu

import (
	"image"

	"cogentcore.org/core/math32"
)

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
	if lin <= 0.0031308 {
		return 12.92 * lin
	}
	return (1.055*math32.Pow(lin, 1/2.4) + 0.055)
}

// SRGBToLinear converts set of sRGB components to linear values,
// removing gamma correction.
func SRGBToLinear(r, g, b float32) (rl, gl, bl float32) {
	rl = SRGBToLinearComp(r)
	gl = SRGBToLinearComp(g)
	bl = SRGBToLinearComp(b)
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

func ImgCompToUint8(val float32) uint8 {
	if val > 1.0 {
		val = 1.0
	}
	return uint8(val * float32(0xff))
}

// TextureSRGBFromLinear returns a sRGB colorspace version of given linear
// colorspace image
func TextureSRGBFromLinear(img *image.RGBA) *image.RGBA {
	out := image.NewRGBA(img.Rect)
	sz := len(img.Pix)
	tof := 1.0 / float32(0xff)
	for i := 0; i < sz; i += 4 {
		r := float32(img.Pix[i]) * tof
		g := float32(img.Pix[i+1]) * tof
		b := float32(img.Pix[i+2]) * tof
		a := img.Pix[i+3]
		rs, gs, bs := SRGBFromLinear(r, g, b)
		out.Pix[i] = ImgCompToUint8(rs)
		out.Pix[i+1] = ImgCompToUint8(gs)
		out.Pix[i+2] = ImgCompToUint8(bs)
		out.Pix[i+3] = a
	}
	return out
}

// TextureSRGBToLinear returns a linear colorspace version of sRGB
// colorspace image
func TextureSRGBToLinear(img *image.RGBA) *image.RGBA {
	out := image.NewRGBA(img.Rect)
	sz := len(img.Pix)
	tof := 1.0 / float32(0xff)
	for i := 0; i < sz; i += 4 {
		r := float32(img.Pix[i]) * tof
		g := float32(img.Pix[i+1]) * tof
		b := float32(img.Pix[i+2]) * tof
		a := img.Pix[i+3]
		rs, gs, bs := SRGBToLinear(r, g, b)
		out.Pix[i] = ImgCompToUint8(rs)
		out.Pix[i+1] = ImgCompToUint8(gs)
		out.Pix[i+2] = ImgCompToUint8(bs)
		out.Pix[i+3] = a
	}
	return out
}

// SetTextureSRGBFromLinear sets in place the pixel values to sRGB colorspace
// version of given linear colorspace image.
// This directly modifies the given image!
func SetTextureSRGBFromLinear(img *image.RGBA) {
	sz := len(img.Pix)
	tof := 1.0 / float32(0xff)
	for i := 0; i < sz; i += 4 {
		r := float32(img.Pix[i]) * tof
		g := float32(img.Pix[i+1]) * tof
		b := float32(img.Pix[i+2]) * tof
		a := img.Pix[i+3]
		rs, gs, bs := SRGBFromLinear(r, g, b)
		img.Pix[i] = ImgCompToUint8(rs)
		img.Pix[i+1] = ImgCompToUint8(gs)
		img.Pix[i+2] = ImgCompToUint8(bs)
		img.Pix[i+3] = a
	}
}

// SetTextureSRGBToLinear sets in place the pixel values to linear colorspace
// version of sRGB colorspace image.
// This directly modifies the given image!
func SetTextureSRGBToLinear(img *image.RGBA) {
	sz := len(img.Pix)
	tof := 1.0 / float32(0xff)
	for i := 0; i < sz; i += 4 {
		r := float32(img.Pix[i]) * tof
		g := float32(img.Pix[i+1]) * tof
		b := float32(img.Pix[i+2]) * tof
		a := img.Pix[i+3]
		rs, gs, bs := SRGBToLinear(r, g, b)
		img.Pix[i] = ImgCompToUint8(rs)
		img.Pix[i+1] = ImgCompToUint8(gs)
		img.Pix[i+2] = ImgCompToUint8(bs)
		img.Pix[i+3] = a
	}
}
