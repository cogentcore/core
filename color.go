// Copyright 2019 The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package mat32

import "image/color"

// NewGoColor3 returns a Vec3 from Go standard color.Color
func NewGoColor3(clr color.Color) Vec3 {
	v3 := Vec3{}
	SetGoColor3(&v3, clr)
	return v3
}

// SetGoColor3 sets a Vec3 from Go standard color.Color
func SetGoColor3(v3 *Vec3, clr color.Color) {
	r, g, b, _ := clr.RGBA()
	v3.X = float32(r) / 0xffff
	v3.Y = float32(g) / 0xffff
	v3.Z = float32(b) / 0xffff
}

// NewGoColor4 returns a Vec4 from Go standard color.Color
func NewGoColor4(clr color.Color) Vec4 {
	v4 := Vec4{}
	SetGoColor4(&v4, clr)
	return v4
}

// SetGoColor4 sets a Vec4 from Go standard color.Color
func SetGoColor4(v4 *Vec4, clr color.Color) {
	r, g, b, a := clr.RGBA()
	v4.X = float32(r) / 0xffff
	v4.Y = float32(g) / 0xffff
	v4.Z = float32(b) / 0xffff
	v4.W = float32(a) / 0xffff
}
