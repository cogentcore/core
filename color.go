// Copyright 2019 The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package mat32

import "image/color"

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
