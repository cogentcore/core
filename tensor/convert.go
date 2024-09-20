// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package tensor

// New1DViewOf returns a 1D view into the given tensor, using the same
// underlying values, and just changing the shape to a 1D view.
// This can be useful e.g., for stats and metric functions that report
// on the 1D list of values.
func New1DViewOf(tsr Tensor) Tensor {
	vw := tsr.View()
	vw.SetShapeInts(tsr.Len())
	return vw
}

// NewFloat64Scalar is a convenience method for a Tensor
// representation of a single float64 scalar value.
func NewFloat64Scalar(val float64) Tensor {
	return NewNumberFromSlice(val)
}

// NewIntScalar is a convenience method for a Tensor
// representation of a single int scalar value.
func NewIntScalar(val int) Tensor {
	return NewNumberFromSlice(val)
}

// NewStringScalar is a convenience method for a Tensor
// representation of a single string scalar value.
func NewStringScalar(val string) Tensor {
	return NewStringFromSlice(val)
}

// NewFloat64FromSlice returns a new 1-dimensional tensor of given value type
// initialized directly from the given slice values, which are not copied.
// The resulting Tensor thus "wraps" the given values.
func NewFloat64FromSlice(vals ...float64) Tensor {
	return NewNumberFromSlice(vals...)
}

// NewIntFromSlice returns a new 1-dimensional tensor of given value type
// initialized directly from the given slice values, which are not copied.
// The resulting Tensor thus "wraps" the given values.
func NewIntFromSlice(vals ...int) Tensor {
	return NewNumberFromSlice(vals...)
}

// NewStringFromSlice returns a new 1-dimensional tensor of given value type
// initialized directly from the given slice values, which are not copied.
// The resulting Tensor thus "wraps" the given values.
func NewStringFromSlice(vals ...string) Tensor {
	n := len(vals)
	tsr := &String{}
	tsr.Values = vals
	tsr.SetShapeInts(n)
	return tsr
}

// AsFloat64 returns the first value of tensor as an float64. Returns 0 if no values.
func AsFloat64(tsr Tensor) float64 {
	if tsr.Len() == 0 {
		return 0
	}
	return tsr.Float1D(0)
}

// AsInt returns the first value of tensor as an int. Returns 0 if no values.
func AsInt(tsr Tensor) int {
	if tsr.Len() == 0 {
		return 0
	}
	return tsr.Int1D(0)
}

// AsString returns the first value of tensor as an string. Returns "" if no values.
func AsString(tsr Tensor) string {
	if tsr.Len() == 0 {
		return ""
	}
	return tsr.String1D(0)
}

// AsFloat64s returns all the tensor values as a slice of float64s.
// This allocates a new slice for the return values, and is not
// a good option for performance-critical code.
func AsFloat64s(tsr Tensor) []float64 {
	if tsr.Len() == 0 {
		return nil
	}
	sz := tsr.Len()
	slc := make([]float64, sz)
	for i := range sz {
		slc[i] = tsr.Float1D(i)
	}
	return slc
}

// AsInts returns all the tensor values as a slice of ints.
// This allocates a new slice for the return values, and is not
// a good option for performance-critical code.
func AsInts(tsr Tensor) []int {
	if tsr.Len() == 0 {
		return nil
	}
	sz := tsr.Len()
	slc := make([]int, sz)
	for i := range sz {
		slc[i] = tsr.Int1D(i)
	}
	return slc
}

// AsStrings returns all the tensor values as a slice of strings.
// This allocates a new slice for the return values, and is not
// a good option for performance-critical code.
func AsStrings(tsr Tensor) []string {
	if tsr.Len() == 0 {
		return nil
	}
	sz := tsr.Len()
	slc := make([]string, sz)
	for i := range sz {
		slc[i] = tsr.String1D(i)
	}
	return slc
}
