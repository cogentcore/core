// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package tensor

import (
	"math"

	"cogentcore.org/core/base/errors"
)

// Clone returns a copy of the given tensor.
// If it is raw [Values] then a [Values.Clone] is returned.
// Otherwise if it is a view, then [Tensor.AsValues] is returned.
// This is equivalent to the NumPy copy function.
func Clone(tsr Tensor) Values {
	if vl, ok := tsr.(Values); ok {
		return vl.Clone()
	}
	return tsr.AsValues()
}

// Flatten returns a copy of the given tensor as a 1D flat list
// of values, by calling Clone(As1D(tsr)).
// It is equivalent to the NumPy flatten function.
func Flatten(tsr Tensor) Values {
	return Clone(As1D(tsr))
}

// Squeeze a [Reshaped] view of given tensor with all singleton
// (size = 1) dimensions removed (if none, just returns the tensor).
func Squeeze(tsr Tensor) Tensor {
	nd := tsr.NumDims()
	sh := tsr.ShapeSizes()
	reshape := make([]int, 0, nd)
	for _, sz := range sh {
		if sz > 1 {
			reshape = append(reshape, sz)
		}
	}
	if len(reshape) == nd {
		return tsr
	}
	return NewReshaped(tsr, reshape...)
}

// As1D returns a 1D tensor, which is either the input tensor if it is
// already 1D, or a new [Reshaped] 1D view of it.
// This can be useful e.g., for stats and metric functions that operate
// on a 1D list of values. See also [Flatten].
func As1D(tsr Tensor) Tensor {
	if tsr.NumDims() == 1 {
		return tsr
	}
	return NewReshaped(tsr, tsr.Len())
}

// Cells1D returns a flat 1D view of the innermost cells for given row index.
// For a [RowMajor] tensor, it uses the [RowTensor] subspace directly,
// otherwise it uses [Sliced] to extract the cells. In either case,
// [As1D] is used to ensure the result is a 1D tensor.
func Cells1D(tsr Tensor, row int) Tensor {
	if rm, ok := tsr.(RowMajor); ok {
		return As1D(rm.RowTensor(row))
	}
	return As1D(NewSliced(tsr, []int{row}))
}

// MustBeValues returns the given tensor as a [Values] subtype, or nil and
// an error if it is not one. Typically outputs of compute operations must
// be values, and are reshaped to hold the results as needed.
func MustBeValues(tsr Tensor) (Values, error) {
	vl, ok := tsr.(Values)
	if !ok {
		return nil, errors.New("tensor.MustBeValues: tensor must be a Values type")
	}
	return vl, nil
}

// MustBeSameShape returns an error if the two tensors do not have the same shape.
func MustBeSameShape(a, b Tensor) error {
	if !a.Shape().IsEqual(b.Shape()) {
		return errors.New("tensor.MustBeSameShape: tensors must have the same shape")
	}
	return nil
}

// SetShape sets the dimension sizes from given Shape
func SetShape(vals Values, sh *Shape) {
	vals.SetShapeSizes(sh.Sizes...)
}

// SetShapeSizesFromTensor sets the dimension sizes as 1D int values from given tensor.
// The backing storage is resized appropriately, retaining all existing data that fits.
func SetShapeSizesFromTensor(vals Values, sizes Tensor) {
	vals.SetShapeSizes(AsIntSlice(sizes)...)
}

// SetShapeFrom sets shape of given tensor from a source tensor.
func SetShapeFrom(vals Values, from Tensor) {
	vals.SetShapeSizes(from.ShapeSizes()...)
}

// SetShapeFromMustBeValues sets shape of given tensor from a source tensor,
// calling [MustBeValues] on the destination tensor to ensure it is a [Values],
// type, returning an error if not. This is used extensively for output
// tensors in functions, and all such output tensors _must_ be Values tensors.
func SetShapeFromMustBeValues(tsr Tensor, from Tensor) error {
	return SetShapeSizesMustBeValues(tsr, from.ShapeSizes()...)
}

// SetShapeSizesMustBeValues sets shape of given tensor from given sizes,
// calling [MustBeValues] on the destination tensor to ensure it is a [Values],
// type, returning an error if not. This is used extensively for output
// tensors in functions, and all such output tensors _must_ be Values tensors.
func SetShapeSizesMustBeValues(tsr Tensor, sizes ...int) error {
	vals, err := MustBeValues(tsr)
	if err != nil {
		return err
	}
	vals.SetShapeSizes(sizes...)
	return nil
}

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
	f := NewFloat64(tsr.ShapeSizes()...)
	f.CopyFrom(tsr.AsValues())
	return f
}

// AsFloat32Tensor returns the tensor as a [Float32] tensor.
// If already is a Float32, it is returned as such.
// Otherwise, a new Float32 tensor is created and values are copied.
func AsFloat32Tensor(tsr Tensor) *Float32 {
	if f, ok := tsr.(*Float32); ok {
		return f
	}
	f := NewFloat32(tsr.ShapeSizes()...)
	f.CopyFrom(tsr.AsValues())
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
	f := NewString(tsr.ShapeSizes()...)
	f.CopyFrom(tsr.AsValues())
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
	f := NewInt(tsr.ShapeSizes()...)
	f.CopyFrom(tsr.AsValues())
	return f
}

// Range returns the min, max (and associated indexes, -1 = no values) for the tensor.
// This is needed for display and is thus in the tensor api on Values.
func Range(vals Values) (min, max float64, minIndex, maxIndex int) {
	minIndex = -1
	maxIndex = -1
	n := vals.Len()
	for j := range n {
		fv := vals.Float1D(n)
		if math.IsNaN(fv) {
			continue
		}
		if fv < min || minIndex < 0 {
			min = fv
			minIndex = j
		}
		if fv > max || maxIndex < 0 {
			max = fv
			maxIndex = j
		}
	}
	return
}
