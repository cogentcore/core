// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package tensor

// NewFloat64Scalar is a convenience method for a Tensor
// representation of a single float64 scalar value.
func NewFloat64Scalar(val float64) *Float64 {
	return NewNumberFromValues(val)
}

// NewIntScalar is a convenience method for a Tensor
// representation of a single int scalar value.
func NewIntScalar(val int) *Int {
	return NewNumberFromValues(val)
}

// NewStringScalar is a convenience method for a Tensor
// representation of a single string scalar value.
func NewStringScalar(val string) *String {
	return NewStringFromValues(val)
}

// NewFloat64FromValues returns a new 1-dimensional tensor of given value type
// initialized directly from the given slice values, which are not copied.
// The resulting Tensor thus "wraps" the given values.
func NewFloat64FromValues(vals ...float64) *Float64 {
	return NewNumberFromValues(vals...)
}

// NewIntFromValues returns a new 1-dimensional tensor of given value type
// initialized directly from the given slice values, which are not copied.
// The resulting Tensor thus "wraps" the given values.
func NewIntFromValues(vals ...int) *Int {
	return NewNumberFromValues(vals...)
}

// NewStringFromValues returns a new 1-dimensional tensor of given value type
// initialized directly from the given slice values, which are not copied.
// The resulting Tensor thus "wraps" the given values.
func NewStringFromValues(vals ...string) *String {
	n := len(vals)
	tsr := &String{}
	tsr.Values = vals
	tsr.SetShapeSizes(n)
	return tsr
}

// SetAllFloat64 sets all values of given tensor to given value.
func SetAllFloat64(tsr Tensor, val float64) {
	VectorizeThreaded(1, func(tsr ...Tensor) int {
		return tsr[0].Len()
	},
		func(idx int, tsr ...Tensor) {
			tsr[0].SetFloat1D(val, idx)
		}, tsr)
}

// SetAllInt sets all values of given tensor to given value.
func SetAllInt(tsr Tensor, val int) {
	VectorizeThreaded(1, func(tsr ...Tensor) int {
		return tsr[0].Len()
	},
		func(idx int, tsr ...Tensor) {
			tsr[0].SetInt1D(val, idx)
		}, tsr)
}

// SetAllString sets all values of given tensor to given value.
func SetAllString(tsr Tensor, val string) {
	VectorizeThreaded(1, func(tsr ...Tensor) int {
		return tsr[0].Len()
	},
		func(idx int, tsr ...Tensor) {
			tsr[0].SetString1D(val, idx)
		}, tsr)
}

// NewFloat64Full returns a new tensor full of given scalar value,
// of given shape sizes.
func NewFloat64Full(val float64, sizes ...int) *Float64 {
	tsr := NewFloat64(sizes...)
	SetAllFloat64(tsr, val)
	return tsr
}

// NewFloat64Ones returns a new tensor full of 1s,
// of given shape sizes.
func NewFloat64Ones(sizes ...int) *Float64 {
	tsr := NewFloat64(sizes...)
	SetAllFloat64(tsr, 1.0)
	return tsr
}

// NewIntFull returns a new tensor full of given scalar value,
// of given shape sizes.
func NewIntFull(val int, sizes ...int) *Int {
	tsr := NewInt(sizes...)
	SetAllInt(tsr, val)
	return tsr
}

// NewStringFull returns a new tensor full of given scalar value,
// of given shape sizes.
func NewStringFull(val string, sizes ...int) *String {
	tsr := NewString(sizes...)
	SetAllString(tsr, val)
	return tsr
}
