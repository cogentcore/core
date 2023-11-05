// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package units

import (
	"goki.dev/mat32/v2"
)

// XY represents unit Value for X and Y dimensions
type XY struct { //gti:add
	// X is the horizontal axis value
	X Value

	// Y is the vertical axis value
	Y Value
}

// ToDots converts value to raw display pixels (dots as in DPI),
// setting also the Dots field
func (xy *XY) ToDots(uc *Context) {
	xy.X.ToDots(uc)
	xy.Y.ToDots(uc)
}

// String implements the fmt.Stringer interface.
func (xy *XY) String() string {
	return "(" + xy.X.String() + ", " + xy.Y.String() + ")"
}

// Set sets the X, Y values according to the given values.
// no values: set to 0.
// 1 value: set both to that value.
// 2 values, set X, Y to the two values respectively.
func (xy *XY) Set(v ...Value) {
	switch len(v) {
	case 0:
		var zv Value
		xy.X = zv
		xy.Y = zv
	case 1:
		xy.X = v[0]
		xy.Y = v[0]
	default:
		xy.X = v[0]
		xy.Y = v[1]
	}
}

// Dim returns the value for given dimension
func (xy *XY) Dim(d mat32.Dims) Value {
	switch d {
	case mat32.X:
		return xy.X
	default:
		return xy.Y
	}
}

// Dots returns the dots values as a mat32.Vec2 vector
func (xy *XY) Dots() mat32.Vec2 {
	return mat32.NewVec2(xy.X.Dots, xy.Y.Dots)
}
