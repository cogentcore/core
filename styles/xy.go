// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package styles

import (
	"fmt"

	"goki.dev/mat32/v2"
)

// XY represents X,Y values
type XY[T any] struct { //gti:add
	// X is the horizontal axis value
	X T

	// Y is the vertical axis value
	Y T
}

// String implements the fmt.Stringer interface.
func (xy *XY[T]) String() string {
	return fmt.Sprintf("(%v, %v)", xy.X, xy.Y)
}

// Set sets the X, Y values according to the given values.
// no values: set to 0.
// 1 value: set both to that value.
// 2 values, set X, Y to the two values respectively.
func (xy *XY[T]) Set(v ...T) {
	switch len(v) {
	case 0:
		var zv T
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

// return the value for given dimension
func (xy *XY[T]) Dim(d mat32.Dims) T {
	switch d {
	case mat32.X:
		return xy.X
	case mat32.Y:
		return xy.Y
	default:
		panic("styles.XY dimension invalid")
	}
}

// set the value for given dimension
func (xy *XY[T]) SetDim(d mat32.Dims, val T) {
	switch d {
	case mat32.X:
		xy.X = val
	case mat32.Y:
		xy.Y = val
	default:
		panic("styles.XY dimension invalid")
	}
}
