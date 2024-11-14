// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package tensor

import (
	"fmt"
	"strconv"

	"cogentcore.org/core/base/num"
	"cogentcore.org/core/base/reflectx"
)

// Number is a tensor of numerical values
type Number[T num.Number] struct {
	Base[T]
}

// Float64 is an alias for Number[float64].
type Float64 = Number[float64]

// Float32 is an alias for Number[float32].
type Float32 = Number[float32]

// Int is an alias for Number[int].
type Int = Number[int]

// Int32 is an alias for Number[int32].
type Int32 = Number[int32]

// Uint32 is an alias for Number[uint32].
type Uint32 = Number[uint32]

// Byte is an alias for Number[byte].
type Byte = Number[byte]

// NewFloat32 returns a new [Float32] tensor
// with the given sizes per dimension (shape).
func NewFloat32(sizes ...int) *Float32 {
	return New[float32](sizes...).(*Float32)
}

// NewFloat64 returns a new [Float64] tensor
// with the given sizes per dimension (shape).
func NewFloat64(sizes ...int) *Float64 {
	return New[float64](sizes...).(*Float64)
}

// NewInt returns a new Int tensor
// with the given sizes per dimension (shape).
func NewInt(sizes ...int) *Int {
	return New[int](sizes...).(*Int)
}

// NewInt32 returns a new Int32 tensor
// with the given sizes per dimension (shape).
func NewInt32(sizes ...int) *Int32 {
	return New[int32](sizes...).(*Int32)
}

// NewUint32 returns a new Uint32 tensor
// with the given sizes per dimension (shape).
func NewUint32(sizes ...int) *Uint32 {
	return New[uint32](sizes...).(*Uint32)
}

// NewByte returns a new Byte tensor
// with the given sizes per dimension (shape).
func NewByte(sizes ...int) *Byte {
	return New[uint8](sizes...).(*Byte)
}

// NewNumber returns a new n-dimensional tensor of numerical values
// with the given sizes per dimension (shape).
func NewNumber[T num.Number](sizes ...int) *Number[T] {
	tsr := &Number[T]{}
	tsr.SetShapeSizes(sizes...)
	tsr.Values = make([]T, tsr.Len())
	return tsr
}

// NewNumberShape returns a new n-dimensional tensor of numerical values
// using given shape.
func NewNumberShape[T num.Number](shape *Shape) *Number[T] {
	tsr := &Number[T]{}
	tsr.shape.CopyFrom(shape)
	tsr.Values = make([]T, shape.Header+tsr.Len())
	return tsr
}

// todo: this should in principle work with yaegi:add but it is crashing
// will come back to it later.

// NewNumberFromValues returns a new 1-dimensional tensor of given value type
// initialized directly from the given slice values, which are not copied.
// The resulting Tensor thus "wraps" the given values.
func NewNumberFromValues[T num.Number](vals ...T) *Number[T] {
	n := len(vals)
	tsr := &Number[T]{}
	tsr.Values = vals
	tsr.SetShapeSizes(n)
	return tsr
}

// String satisfies the fmt.Stringer interface for string of tensor data.
func (tsr *Number[T]) String() string { return Sprintf("", tsr, 0) }

func (tsr *Number[T]) IsString() bool { return false }

func (tsr *Number[T]) AsValues() Values { return tsr }

func (tsr *Number[T]) SetAdd(val T, i ...int) {
	tsr.Values[tsr.shape.IndexTo1D(i...)] += val
}
func (tsr *Number[T]) SetSub(val T, i ...int) {
	tsr.Values[tsr.shape.IndexTo1D(i...)] -= val
}
func (tsr *Number[T]) SetMul(val T, i ...int) {
	tsr.Values[tsr.shape.IndexTo1D(i...)] *= val
}
func (tsr *Number[T]) SetDiv(val T, i ...int) {
	tsr.Values[tsr.shape.IndexTo1D(i...)] /= val
}

/////////////////////  Strings

func (tsr *Number[T]) SetString(val string, i ...int) {
	if fv, err := strconv.ParseFloat(val, 64); err == nil {
		tsr.Values[tsr.shape.IndexTo1D(i...)] = T(fv)
	}
}

func (tsr Number[T]) SetString1D(val string, i int) {
	if fv, err := strconv.ParseFloat(val, 64); err == nil {
		tsr.Values[tsr.shape.Header+i] = T(fv)
	}
}

func (tsr *Number[T]) SetStringRow(val string, row, cell int) {
	if fv, err := strconv.ParseFloat(val, 64); err == nil {
		_, sz := tsr.shape.RowCellSize()
		tsr.Values[tsr.shape.Header+row*sz+cell] = T(fv)
	}
}

/////////////////////  Floats

func (tsr *Number[T]) Float(i ...int) float64 {
	return float64(tsr.Values[tsr.shape.IndexTo1D(i...)])
}

func (tsr *Number[T]) SetFloat(val float64, i ...int) {
	tsr.Values[tsr.shape.IndexTo1D(i...)] = T(val)
}

func (tsr *Number[T]) Float1D(i int) float64 {
	return float64(tsr.Values[tsr.shape.Header+i])
}

func (tsr *Number[T]) SetFloat1D(val float64, i int) {
	tsr.Values[tsr.shape.Header+i] = T(val)
}

func (tsr *Number[T]) FloatRow(row, cell int) float64 {
	_, sz := tsr.shape.RowCellSize()
	i := row*sz + cell
	return float64(tsr.Values[tsr.shape.Header+i])
}

func (tsr *Number[T]) SetFloatRow(val float64, row, cell int) {
	_, sz := tsr.shape.RowCellSize()
	tsr.Values[tsr.shape.Header+row*sz+cell] = T(val)
}

/////////////////////  Ints

func (tsr *Number[T]) Int(i ...int) int {
	return int(tsr.Values[tsr.shape.IndexTo1D(i...)])
}

func (tsr *Number[T]) SetInt(val int, i ...int) {
	tsr.Values[tsr.shape.IndexTo1D(i...)] = T(val)
}

func (tsr *Number[T]) Int1D(i int) int {
	return int(tsr.Values[tsr.shape.Header+i])
}

func (tsr *Number[T]) SetInt1D(val int, i int) {
	tsr.Values[tsr.shape.Header+i] = T(val)
}

func (tsr *Number[T]) IntRow(row, cell int) int {
	_, sz := tsr.shape.RowCellSize()
	i := row*sz + cell
	return int(tsr.Values[tsr.shape.Header+i])
}

func (tsr *Number[T]) SetIntRow(val int, row, cell int) {
	_, sz := tsr.shape.RowCellSize()
	tsr.Values[tsr.shape.Header+row*sz+cell] = T(val)
}

// SetZeros is simple convenience function initialize all values to 0
func (tsr *Number[T]) SetZeros() {
	n := len(tsr.Values)
	for j := tsr.shape.Header; j < n; j++ {
		tsr.Values[j] = 0
	}
}

// Clone clones this tensor, creating a duplicate copy of itself with its
// own separate memory representation of all the values.
func (tsr *Number[T]) Clone() Values {
	csr := NewNumberShape[T](&tsr.shape)
	copy(csr.Values, tsr.Values)
	return csr
}

// CopyFrom copies all avail values from other tensor into this tensor, with an
// optimized implementation if the other tensor is of the same type, and
// otherwise it goes through appropriate standard type.
func (tsr *Number[T]) CopyFrom(frm Values) {
	if fsm, ok := frm.(*Number[T]); ok {
		copy(tsr.Values[tsr.shape.Header:], fsm.Values[fsm.shape.Header:])
		return
	}
	sz := min(tsr.Len(), frm.Len())
	if reflectx.KindIsInt(tsr.DataType()) {
		for i := range sz {
			tsr.Values[tsr.shape.Header+i] = T(frm.Int1D(i))
		}
	} else {
		for i := range sz {
			tsr.Values[tsr.shape.Header+i] = T(frm.Float1D(i))
		}
	}
}

// AppendFrom appends values from other tensor into this tensor,
// which must have the same cell size as this tensor.
// It uses and optimized implementation if the other tensor
// is of the same type, and otherwise it goes through
// appropriate standard type.
func (tsr *Number[T]) AppendFrom(frm Values) error {
	rows, cell := tsr.shape.RowCellSize()
	frows, fcell := frm.Shape().RowCellSize()
	if cell != fcell {
		return fmt.Errorf("tensor.AppendFrom: cell sizes do not match: %d != %d", cell, fcell)
	}
	tsr.SetNumRows(rows + frows)
	st := tsr.shape.Header + rows*cell
	fsz := frows * fcell
	if fsm, ok := frm.(*Number[T]); ok {
		copy(tsr.Values[st:st+fsz], fsm.Values[fsm.shape.Header:])
		return nil
	}
	for i := 0; i < fsz; i++ {
		tsr.Values[st+i] = T(frm.Float1D(i))
	}
	return nil
}

// CopyCellsFrom copies given range of values from other tensor into this tensor,
// using flat 1D indexes: to = starting index in this Tensor to start copying into,
// start = starting index on from Tensor to start copying from, and n = number of
// values to copy.  Uses an optimized implementation if the other tensor is
// of the same type, and otherwise it goes through appropriate standard type.
func (tsr *Number[T]) CopyCellsFrom(frm Values, to, start, n int) {
	to += tsr.shape.Header
	if fsm, ok := frm.(*Number[T]); ok {
		start += fsm.shape.Header
		copy(tsr.Values[to:to+n], fsm.Values[start:start+n])
		return
	}
	for i := range n {
		tsr.Values[to+i] = T(frm.Float1D(start + i))
	}
}

// SubSpace returns a new tensor with innermost subspace at given
// offset(s) in outermost dimension(s) (len(offs) < NumDims).
// The new tensor points to the values of the this tensor (i.e., modifications
// will affect both), as its Values slice is a view onto the original (which
// is why only inner-most contiguous supsaces are supported).
// Use AsValues() method to separate the two.
func (tsr *Number[T]) SubSpace(offs ...int) Values {
	b := tsr.subSpaceImpl(offs...)
	rt := &Number[T]{Base: *b}
	return rt
}

// RowTensor is a convenience version of [RowMajor.SubSpace] to return the
// SubSpace for the outermost row dimension. [Rows] defines a version
// of this that indirects through the row indexes.
func (tsr *Number[T]) RowTensor(row int) Values {
	return tsr.SubSpace(row)
}

// SetRowTensor sets the values of the SubSpace at given row to given values.
func (tsr *Number[T]) SetRowTensor(val Values, row int) {
	_, cells := tsr.shape.RowCellSize()
	st := row * cells
	mx := min(val.Len(), cells)
	tsr.CopyCellsFrom(val, st, 0, mx)
}

// AppendRow adds a row and sets values to given values.
func (tsr *Number[T]) AppendRow(val Values) {
	if tsr.NumDims() == 0 {
		tsr.SetShapeSizes(0)
	}
	nrow := tsr.DimSize(0)
	tsr.SetNumRows(nrow + 1)
	tsr.SetRowTensor(val, nrow)
}
