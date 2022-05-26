// Copyright 2019 The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package mat32

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

// NewVec3Color returns a Vec3 from Go standard color.Color
// (R,G,B components only)
func NewVec3Color(clr color.Color) Vec3 {
	v3 := Vec3{}
	v3.SetColor(clr)
	return v3
}

// SetColor sets from Go standard color.Color
// (R,G,B components only)
func (v *Vec3) SetColor(clr color.Color) {
	r, g, b, _ := clr.RGBA()
	v.X = float32(r) / 0xffff
	v.Y = float32(g) / 0xffff
	v.Z = float32(b) / 0xffff
}

// NewVec4Color returns a Vec4 from Go standard color.Color
// (full R,G,B,A components)
func NewVec4Color(clr color.Color) Vec4 {
	v4 := Vec4{}
	v4.SetColor(clr)
	return v4
}

// SetColor sets a Vec4 from Go standard color.Color
func (v *Vec4) SetColor(clr color.Color) {
	r, g, b, a := clr.RGBA()
	v.X = float32(r) / 0xffff
	v.Y = float32(g) / 0xffff
	v.Z = float32(b) / 0xffff
	v.W = float32(a) / 0xffff
}

// SRGBFromLinear returns an SRGB color space value from a linear source
func (v Vec3) SRGBFromLinear() Vec3 {
	nv := Vec3{}
	nv.X = SRGBFromLinear(v.X)
	nv.Y = SRGBFromLinear(v.Y)
	nv.Z = SRGBFromLinear(v.Z)
	return nv
}

// SRGBToLinear returns a linear color space value from a SRGB source
func (v Vec3) SRGBToLinear() Vec3 {
	nv := Vec3{}
	nv.X = SRGBToLinear(v.X)
	nv.Y = SRGBToLinear(v.Y)
	nv.Z = SRGBToLinear(v.Z)
	return nv
}

// SRGBFromLinear returns an SRGB color space value from a linear source
func (v Vec4) SRGBFromLinear() Vec4 {
	nv := Vec4{}
	nv.X = SRGBFromLinear(v.X)
	nv.Y = SRGBFromLinear(v.Y)
	nv.Z = SRGBFromLinear(v.Z)
	nv.W = v.W
	return nv
}

// SRGBToLinear returns a linear color space value from a SRGB source
func (v Vec4) SRGBToLinear() Vec4 {
	nv := Vec4{}
	nv.X = SRGBToLinear(v.X)
	nv.Y = SRGBToLinear(v.Y)
	nv.Z = SRGBToLinear(v.Z)
	nv.W = v.W
	return nv
}
