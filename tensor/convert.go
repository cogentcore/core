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

// Cells1D returns a flat 1D [Tensor] view of the cells for given row
// index.  This is useful for passing to other functions e.g.,
// in stats or metrics that process a 1D tensor.
func Cells1D(tsr Tensor, row int) Tensor {
	return New1DViewOf(tsr.SubSpace(row))
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

// AsFloat64Scalar returns the first value of tensor as a float64 scalar.
// Returns 0 if no values.
func AsFloat64Scalar(tsr Tensor) float64 {
	if tsr.Len() == 0 {
		return 0
	}
	return tsr.Float1D(0)
}

// AsIntScalar returns the first value of tensor as an int scalar.
// Returns 0 if no values.
func AsIntScalar(tsr Tensor) int {
	if tsr.Len() == 0 {
		return 0
	}
	return tsr.Int1D(0)
}

// AsStringScalar returns the first value of tensor as a string scalar.
// Returns "" if no values.
func AsStringScalar(tsr Tensor) string {
	if tsr.Len() == 0 {
		return ""
	}
	return tsr.String1D(0)
}

// AsFloat64Slice returns all the tensor values as a slice of float64's.
// This allocates a new slice for the return values, and is not
// a good option for performance-critical code.
func AsFloat64Slice(tsr Tensor) []float64 {
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

// AsIntSlice returns all the tensor values as a slice of ints.
// This allocates a new slice for the return values, and is not
// a good option for performance-critical code.
func AsIntSlice(tsr Tensor) []int {
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

// AsStringSlice returns all the tensor values as a slice of strings.
// This allocates a new slice for the return values, and is not
// a good option for performance-critical code.
func AsStringSlice(tsr Tensor) []string {
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

// AsFloat64Tensor returns the tensor as a [Float64] tensor.
// If already is a Float64, it is returned as such.
// Otherwise, a new Float64 tensor is created and values are copied.
// Use this function for interfacing with gonum or other apis that
// only operate on float64 types.
func AsFloat64Tensor(tsr Tensor) *Float64 {
	if f, ok := tsr.(*Float64); ok {
		return f
	}
	f := NewFloat64(tsr.ShapeInts()...)
	f.CopyFrom(tsr)
	return f
}

// AsFloat32Tensor returns the tensor as a [Float32] tensor.
// If already is a Float32, it is returned as such.
// Otherwise, a new Float32 tensor is created and values are copied.
func AsFloat32Tensor(tsr Tensor) *Float32 {
	if f, ok := tsr.(*Float32); ok {
		return f
	}
	f := NewFloat32(AsIntSlice(tsr.ShapeSizes())...)
	f.CopyFrom(tsr)
	return f
}

// AsStringTensor returns the tensor as a [String] tensor.
// If already is a String, it is returned as such.
// Otherwise, a new String tensor is created and values are copied.
// Use this function for interfacing with gonum or other apis that
// only operate on float64 types.
func AsStringTensor(tsr Tensor) *String {
	if f, ok := tsr.(*String); ok {
		return f
	}
	f := NewString(tsr.ShapeInts()...)
	f.CopyFrom(tsr)
	return f
}

// AsIntTensor returns the tensor as a [Int] tensor.
// If already is a Int, it is returned as such.
// Otherwise, a new Int tensor is created and values are copied.
// Use this function for interfacing with gonum or other apis that
// only operate on float64 types.
func AsIntTensor(tsr Tensor) *Int {
	if f, ok := tsr.(*Int); ok {
		return f
	}
	f := NewInt(tsr.ShapeInts()...)
	f.CopyFrom(tsr)
	return f
}
