// Copyright 2019 The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package mat32

import "golang.org/x/image/math/fixed"

// ToFixed converts a float32 value to a fixed.Int26_6
func ToFixed(x float32) fixed.Int26_6 {
	return fixed.Int26_6(x * 64)
}

// FromFixed converts a fixed.Int26_6 to a float32
func FromFixed(x fixed.Int26_6) float32 {
	const shift, mask = 6, 1<<6 - 1
	if x >= 0 {
		return float32(x>>shift) + float32(x&mask)/64
	}
	x = -x
	if x >= 0 {
		return -(float32(x>>shift) + float32(x&mask)/64)
	}
	return 0
}

// ToFixedPoint converts  float32 x,y values to a fixed.Point26_6
func ToFixedPoint(x, y float32) fixed.Point26_6 {
	return fixed.Point26_6{X: ToFixed(x), Y: ToFixed(y)}
}
