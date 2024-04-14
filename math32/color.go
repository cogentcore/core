// Copyright 2019 Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package math32

import "image/color"

// official correct ones:

// SRGBFromLinear converts a color with linear gamma correction to SRGB standard 2.4 gamma
// with offsets.
func SRGBFromLinear(lin float32) float32 {
	if lin <= 0.0031308 {
		return lin * 12.92
	}
	return Pow(lin, 1.0/2.4)*1.055 - 0.055
}

// SRGBToLinear converts a color with SRGB gamma correction to SRGB standard 2.4 gamma
// with offsets
func SRGBToLinear(sr float32) float32 {
	if sr <= 0.04045 {
		return sr / 12.92
	}
	return Pow((sr+0.055)/1.055, 2.4)
}

/*
// rough-and-ready approx used in many cases:

func SRGBFromLinear(lin float32) float32 {
	return Pow(lin, 1.0/2.2)
}

func SRGBToLinear(sr float32) float32 {
	return Pow(sr, 2.2)
}

*/

// NewVector3Color returns a Vector3 from Go standard color.Color
// (R,G,B components only)
func NewVector3Color(clr color.Color) Vector3 {
	v3 := Vector3{}
	v3.SetColor(clr)
	return v3
}

// SetColor sets from Go standard color.Color
// (R,G,B components only)
func (v *Vector3) SetColor(clr color.Color) {
	r, g, b, _ := clr.RGBA()
	v.X = float32(r) / 0xffff
	v.Y = float32(g) / 0xffff
	v.Z = float32(b) / 0xffff
}

// NewVector4Color returns a Vector4 from Go standard color.Color
// (full R,G,B,A components)
func NewVector4Color(clr color.Color) Vector4 {
	v4 := Vector4{}
	v4.SetColor(clr)
	return v4
}

// SetColor sets a Vector4 from Go standard color.Color
func (v *Vector4) SetColor(clr color.Color) {
	r, g, b, a := clr.RGBA()
	v.X = float32(r) / 0xffff
	v.Y = float32(g) / 0xffff
	v.Z = float32(b) / 0xffff
	v.W = float32(a) / 0xffff
}

// SRGBFromLinear returns an SRGB color space value from a linear source
func (v Vector3) SRGBFromLinear() Vector3 {
	nv := Vector3{}
	nv.X = SRGBFromLinear(v.X)
	nv.Y = SRGBFromLinear(v.Y)
	nv.Z = SRGBFromLinear(v.Z)
	return nv
}

// SRGBToLinear returns a linear color space value from a SRGB source
func (v Vector3) SRGBToLinear() Vector3 {
	nv := Vector3{}
	nv.X = SRGBToLinear(v.X)
	nv.Y = SRGBToLinear(v.Y)
	nv.Z = SRGBToLinear(v.Z)
	return nv
}

// SRGBFromLinear returns an SRGB color space value from a linear source
func (v Vector4) SRGBFromLinear() Vector4 {
	nv := Vector4{}
	nv.X = SRGBFromLinear(v.X)
	nv.Y = SRGBFromLinear(v.Y)
	nv.Z = SRGBFromLinear(v.Z)
	nv.W = v.W
	return nv
}

// SRGBToLinear returns a linear color space value from a SRGB source
func (v Vector4) SRGBToLinear() Vector4 {
	nv := Vector4{}
	nv.X = SRGBToLinear(v.X)
	nv.Y = SRGBToLinear(v.Y)
	nv.Z = SRGBToLinear(v.Z)
	nv.W = v.W
	return nv
}
