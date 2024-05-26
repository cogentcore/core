// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package units

import (
	"cogentcore.org/core/math32"
)

// XY represents unit Value for X and Y dimensions
type XY struct { //types:add
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

// Zero sets values to 0
func (xy *XY) Zero() {
	xy.X.Zero()
	xy.Y.Zero()
}

// Set sets the x and y values according to the given values.
// No values: set both to 0.
// One value: set both to that value.
// Two values: set x to the first value and y to the second value.
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
func (xy *XY) Dim(d math32.Dims) Value {
	switch d {
	case math32.X:
		return xy.X
	case math32.Y:
		return xy.Y
	default:
		panic("units.XY dimension invalid")
	}
}

// SetDim sets the value for given dimension
func (xy *XY) SetDim(d math32.Dims, val Value) {
	switch d {
	case math32.X:
		xy.X = val
	case math32.Y:
		xy.Y = val
	default:
		panic("units.XY dimension invalid")
	}
}

// Dots returns the dots values as a math32.Vector2 vector
func (xy *XY) Dots() math32.Vector2 {
	return math32.Vec2(xy.X.Dots, xy.Y.Dots)
}
