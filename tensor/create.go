// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package tensor

import (
	"math/rand"
	"slices"
)

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
	VectorizeThreaded(1, func(tsr ...Tensor) int { return tsr[0].Len() },
		func(idx int, tsr ...Tensor) {
			tsr[0].SetFloat1D(val, idx)
		}, tsr)
}

// SetAllInt sets all values of given tensor to given value.
func SetAllInt(tsr Tensor, val int) {
	VectorizeThreaded(1, func(tsr ...Tensor) int { return tsr[0].Len() },
		func(idx int, tsr ...Tensor) {
			tsr[0].SetInt1D(val, idx)
		}, tsr)
}

// SetAllString sets all values of given tensor to given value.
func SetAllString(tsr Tensor, val string) {
	VectorizeThreaded(1, func(tsr ...Tensor) int { return tsr[0].Len() },
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

// NewFloat64Rand returns a new tensor full of random numbers from
// global random source, of given shape sizes.
func NewFloat64Rand(sizes ...int) *Float64 {
	tsr := NewFloat64(sizes...)
	FloatSetFunc(1, func(idx int) float64 { return rand.Float64() }, tsr)
	return tsr
}

// NewIntRange returns a new [Int] [Tensor] with given [Slice]
// range parameters, with the same semantics as NumPy arange based on
// the number of arguments passed:
//   - 1 = stop
//   - 2 = start, stop
//   - 3 = start, stop, step
func NewIntRange(svals ...int) *Int {
	if len(svals) == 0 {
		return NewInt()
	}
	sl := Slice{}
	switch len(svals) {
	case 1:
		sl.Stop = svals[0]
	case 2:
		sl.Start = svals[0]
		sl.Stop = svals[1]
	case 3:
		sl.Start = svals[0]
		sl.Stop = svals[1]
		sl.Step = svals[2]
	}
	return sl.IntTensor(sl.Stop)
}

// NewFloat64SpacedLinear returns a new [Float64] tensor with num linearly
// spaced numbers between start and stop values, as tensors, which
// must be the same length and determine the cell shape of the output.
// If num is 0, then a default of 50 is used.
// If endpoint = true, then the stop value is _inclusive_, i.e., it will
// be the final value, otherwise it is exclusive.
// This corresponds to the NumPy linspace function.
func NewFloat64SpacedLinear(start, stop Tensor, num int, endpoint bool) *Float64 {
	if num <= 0 {
		num = 50
	}
	fnum := float64(num)
	if endpoint {
		fnum -= 1
	}
	step := Clone(start)
	n := step.Len()
	for i := range n {
		step.SetFloat1D((stop.Float1D(i)-start.Float1D(i))/fnum, i)
	}
	tsz := slices.Clone(start.Shape().Sizes)
	tsz = append([]int{num}, tsz...)
	tsr := NewFloat64(tsz...)
	for r := range num {
		for i := range n {
			tsr.SetFloatRowCell(start.Float1D(i)+float64(r)*step.Float1D(i), r, i)
		}
	}
	return tsr
}
