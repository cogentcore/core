// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package tensor

import (
	"fmt"
	"log"
	"math"
	"strconv"
	"strings"

	"cogentcore.org/core/base/num"
	"gonum.org/v1/gonum/mat"
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

// Byte is an alias for Number[byte].
type Byte = Number[byte]

// NewFloat32 returns a new [Float32] tensor
// with the given sizes per dimension (shape), and optional dimension names.
func NewFloat32(sizes ...int) *Float32 {
	return New[float32](sizes...).(*Float32)
}

// AsFloat32 returns the tensor as a [Float32] tensor.
// If already is a Float32, it is returned as such.
// Otherwise, a new Float32 tensor is created and values are copied.
func AsFloat32(tsr Tensor) *Float32 {
	if f, ok := tsr.(*Float32); ok {
		return f
	}
	f := NewFloat32(tsr.Shape().Sizes...)
	f.SetNames(tsr.Shape().Names...)
	f.CopyFrom(tsr)
	return f
}

// NewFloat64 returns a new [Float64] tensor
// with the given sizes per dimension (shape), and optional dimension names.
func NewFloat64(sizes ...int) *Float64 {
	return New[float64](sizes...).(*Float64)
}

// AsFloat64 returns the tensor as a [Float64] tensor.
// If already is a Float64, it is returned as such.
// Otherwise, a new Float64 tensor is created and values are copied.
// Use this function for interfacing with gonum or other apis that
// only operate on float64 types.
func AsFloat64(tsr Tensor) *Float64 {
	if f, ok := tsr.(*Float64); ok {
		return f
	}
	f := NewFloat64(tsr.Shape().Sizes...)
	f.SetNames(tsr.Shape().Names...)
	f.CopyFrom(tsr)
	return f
}

// NewInt returns a new Int tensor
// with the given sizes per dimension (shape), and optional dimension names.
func NewInt(sizes ...int) *Int {
	return New[int](sizes...).(*Int)
}

// NewInt32 returns a new Int32 tensor
// with the given sizes per dimension (shape), and optional dimension names.
func NewInt32(sizes ...int) *Int32 {
	return New[int32](sizes...).(*Int32)
}

// NewByte returns a new Byte tensor
// with the given sizes per dimension (shape), and optional dimension names.
func NewByte(sizes ...int) *Byte {
	return New[uint8](sizes...).(*Byte)
}

// NewNumber returns a new n-dimensional tensor of numerical values
// with the given sizes per dimension (shape), and optional dimension names.
func NewNumber[T num.Number](sizes ...int) *Number[T] {
	tsr := &Number[T]{}
	tsr.SetShape(sizes...)
	tsr.Values = make([]T, tsr.Len())
	return tsr
}

// NewNumberShape returns a new n-dimensional tensor of numerical values
// using given shape.
func NewNumberShape[T num.Number](shape *Shape) *Number[T] {
	tsr := &Number[T]{}
	tsr.shape.CopyShape(shape)
	tsr.Values = make([]T, tsr.Len())
	return tsr
}

// NewNumberFromSlice returns a new 1-dimensional tensor of given value type
// initialized directly from the given slice values, which are not copied.
// The resulting Tensor thus "wraps" the given values.
func NewNumberFromSlice[T num.Number](vals []T) Tensor {
	n := len(vals)
	tsr := &Number[T]{}
	tsr.Values = vals
	tsr.SetShape(n)
	return tsr
}

func (tsr *Number[T]) IsString() bool {
	return false
}

func (tsr *Number[T]) SetString(val string, i ...int) {
	if fv, err := strconv.ParseFloat(val, 64); err == nil {
		tsr.Values[tsr.shape.Offset(i...)] = T(fv)
	}
}

func (tsr Number[T]) SetString1D(val string, off int) {
	if fv, err := strconv.ParseFloat(val, 64); err == nil {
		tsr.Values[off] = T(fv)
	}
}

func (tsr *Number[T]) SetStringRowCell(val string, row, cell int) {
	if fv, err := strconv.ParseFloat(val, 64); err == nil {
		_, sz := tsr.shape.RowCellSize()
		tsr.Values[row*sz+cell] = T(fv)
	}
}

// String satisfies the fmt.Stringer interface for string of tensor data
func (tsr *Number[T]) String() string {
	str := tsr.Label()
	sz := len(tsr.Values)
	if sz > 1000 {
		return str
	}
	var b strings.Builder
	b.WriteString(str)
	b.WriteString("\n")
	oddRow := true
	rows, cols, _, _ := Projection2DShape(&tsr.shape, oddRow)
	for r := 0; r < rows; r++ {
		rc, _ := Projection2DCoords(&tsr.shape, oddRow, r, 0)
		b.WriteString(fmt.Sprintf("%v: ", rc))
		for c := 0; c < cols; c++ {
			vl := Projection2DValue(tsr, oddRow, r, c)
			b.WriteString(fmt.Sprintf("%7g ", vl))
		}
		b.WriteString("\n")
	}
	return b.String()
}

func (tsr *Number[T]) Float(i ...int) float64 {
	return float64(tsr.Values[tsr.shape.Offset(i...)])
}

func (tsr *Number[T]) SetFloat(val float64, i ...int) {
	tsr.Values[tsr.shape.Offset(i...)] = T(val)
}

func (tsr *Number[T]) Float1D(i int) float64 {
	return float64(tsr.Values[i])
}

func (tsr *Number[T]) SetFloat1D(val float64, i int) {
	tsr.Values[i] = T(val)
}

func (tsr *Number[T]) FloatRowCell(row, cell int) float64 {
	_, sz := tsr.shape.RowCellSize()
	i := row*sz + cell
	return float64(tsr.Values[i])
}

func (tsr *Number[T]) SetFloatRowCell(val float64, row, cell int) {
	_, sz := tsr.shape.RowCellSize()
	tsr.Values[row*sz+cell] = T(val)
}

// At is the gonum/mat.Matrix interface method for returning 2D matrix element at given
// row, column index.  Assumes Row-major ordering and logs an error if NumDims < 2.
func (tsr *Number[T]) At(i, j int) float64 {
	nd := tsr.NumDims()
	if nd < 2 {
		log.Println("tensor Dims gonum Matrix call made on Tensor with dims < 2")
		return 0
	} else if nd == 2 {
		return tsr.Float(i, j)
	} else {
		ix := make([]int, nd)
		ix[nd-2] = i
		ix[nd-1] = j
		return tsr.Float(ix...)
	}
}

// T is the gonum/mat.Matrix transpose method.
// It performs an implicit transpose by returning the receiver inside a Transpose.
func (tsr *Number[T]) T() mat.Matrix {
	return mat.Transpose{tsr}
}

// Range returns the min, max (and associated indexes, -1 = no values) for the tensor.
// This is needed for display and is thus in the core api in optimized form
// Other math operations can be done using gonum/floats package.
func (tsr *Number[T]) Range() (min, max float64, minIndex, maxIndex int) {
	minIndex = -1
	maxIndex = -1
	for j, vl := range tsr.Values {
		fv := float64(vl)
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

// SetZeros is simple convenience function initialize all values to 0
func (tsr *Number[T]) SetZeros() {
	for j := range tsr.Values {
		tsr.Values[j] = 0
	}
}

// Clone clones this tensor, creating a duplicate copy of itself with its
// own separate memory representation of all the values, and returns
// that as a Tensor (which can be converted into the known type as needed).
func (tsr *Number[T]) Clone() Tensor {
	csr := NewNumberShape[T](&tsr.shape)
	copy(csr.Values, tsr.Values)
	return csr
}

func (tsr *Number[T]) View() Tensor {
	return &Number[T]{*tsr.view()}
}

// CopyFrom copies all avail values from other tensor into this tensor, with an
// optimized implementation if the other tensor is of the same type, and
// otherwise it goes through appropriate standard type.
func (tsr *Number[T]) CopyFrom(frm Tensor) {
	if fsm, ok := frm.(*Number[T]); ok {
		copy(tsr.Values, fsm.Values)
		return
	}
	sz := min(len(tsr.Values), frm.Len())
	for i := 0; i < sz; i++ {
		tsr.Values[i] = T(frm.Float1D(i))
	}
}

// AppendFrom appends values from other tensor into this tensor,
// which must have the same cell size as this tensor.
// It uses and optimized implementation if the other tensor
// is of the same type, and otherwise it goes through
// appropriate standard type.
func (tsr *Number[T]) AppendFrom(frm Tensor) error {
	rows, cell := tsr.RowCellSize()
	frows, fcell := frm.RowCellSize()
	if cell != fcell {
		return fmt.Errorf("tensor.AppendFrom: cell sizes do not match: %d != %d", cell, fcell)
	}
	tsr.SetNumRows(rows + frows)
	st := rows * cell
	fsz := frows * fcell
	if fsm, ok := frm.(*Number[T]); ok {
		copy(tsr.Values[st:st+fsz], fsm.Values)
		return nil
	}
	for i := 0; i < fsz; i++ {
		tsr.Values[st+i] = T(frm.Float1D(i))
	}
	return nil
}

// SetShapeFrom copies just the shape from given source tensor
// calling SetShape with the shape params from source (see for more docs).
func (tsr *Number[T]) SetShapeFrom(frm Tensor) {
	sh := frm.Shape()
	tsr.SetShape(sh.Sizes...)
	tsr.SetNames(sh.Names...)
}

// CopyCellsFrom copies given range of values from other tensor into this tensor,
// using flat 1D indexes: to = starting index in this Tensor to start copying into,
// start = starting index on from Tensor to start copying from, and n = number of
// values to copy.  Uses an optimized implementation if the other tensor is
// of the same type, and otherwise it goes through appropriate standard type.
func (tsr *Number[T]) CopyCellsFrom(frm Tensor, to, start, n int) {
	if fsm, ok := frm.(*Number[T]); ok {
		for i := range n {
			tsr.Values[to+i] = fsm.Values[start+i]
		}
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
// Use Clone() method to separate the two.
func (tsr *Number[T]) SubSpace(offs ...int) Tensor {
	b := tsr.subSpaceImpl(offs...)
	rt := &Number[T]{Base: *b}
	return rt
}

// RowTensor is a convenience version of [Tensor.SubSpace] to return the
// SubSpace for the outermost row dimension. [Indexed] defines a version
// of this that indirects through the row indexes.
func (tsr *Number[T]) RowTensor(row int) Tensor {
	return tsr.SubSpace(row)
}

// SetRowTensor sets the values of the SubSpace at given row to given values.
func (tsr *Number[T]) SetRowTensor(val Tensor, row int) {
	_, cells := tsr.RowCellSize()
	st := row * cells
	mx := min(val.Len(), cells)
	tsr.CopyCellsFrom(val, st, 0, mx)
}
