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

	"cogentcore.org/core/gox/num"
	"cogentcore.org/core/tensor/bitslice"
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

// NewNumber returns a new n-dimensional tensor of numerical values
// with the given sizes per dimension (shape), and optional dimension names.
// Nulls are initialized to nil.
func NewNumber[T num.Number](sizes []int, names ...string) *Number[T] {
	tsr := &Number[T]{}
	tsr.SetShape(sizes, names...)
	tsr.Values = make([]T, tsr.Len())
	return tsr
}

// NewNumberShape returns a new n-dimensional tensor of numerical values
// using given shape.
// Nulls are initialized to nil.
func NewNumberShape[T num.Number](shape *Shape) *Number[T] {
	tsr := &Number[T]{}
	tsr.Shp.CopyShape(shape)
	tsr.Values = make([]T, tsr.Len())
	return tsr
}

func (tsr *Number[T]) IsString() bool {
	return false
}

func (tsr *Number[T]) AddScalar(i []int, val float64) float64 {
	j := tsr.Shp.Offset(i)
	tsr.Values[j] += T(val)
	return float64(tsr.Values[j])
}

func (tsr *Number[T]) MulScalar(i []int, val float64) float64 {
	j := tsr.Shp.Offset(i)
	tsr.Values[j] *= T(val)
	return float64(tsr.Values[j])
}

func (tsr *Number[T]) SetString(i []int, val string) {
	if fv, err := strconv.ParseFloat(val, 64); err == nil {
		j := tsr.Shp.Offset(i)
		tsr.Values[j] = T(fv)
	}
}

func (tsr Number[T]) SetString1D(off int, val string) {
	if fv, err := strconv.ParseFloat(val, 64); err == nil {
		tsr.Values[off] = T(fv)
	}
}
func (tsr *Number[T]) SetStringRowCell(row, cell int, val string) {
	if fv, err := strconv.ParseFloat(val, 64); err == nil {
		_, sz := tsr.Shp.RowCellSize()
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
	rows, cols, _, _ := Prjn2DShape(&tsr.Shp, oddRow)
	for r := 0; r < rows; r++ {
		rc, _ := Prjn2DCoords(&tsr.Shp, oddRow, r, 0)
		b.WriteString(fmt.Sprintf("%v: ", rc))
		for c := 0; c < cols; c++ {
			vl := Prjn2DValue(tsr, oddRow, r, c)
			b.WriteString(fmt.Sprintf("%7g ", vl))
		}
		b.WriteString("\n")
	}
	return b.String()
}

func (tsr *Number[T]) Float(i []int) float64 {
	j := tsr.Shp.Offset(i)
	return float64(tsr.Values[j])
}

func (tsr *Number[T]) SetFloat(i []int, val float64) {
	j := tsr.Shp.Offset(i)
	tsr.Values[j] = T(val)
}

func (tsr *Number[T]) Float1D(off int) float64 {
	return float64(tsr.Values[off])
}

func (tsr *Number[T]) SetFloat1D(off int, val float64) {
	tsr.Values[off] = T(val)
}

func (tsr *Number[T]) FloatRowCell(row, cell int) float64 {
	_, sz := tsr.Shp.RowCellSize()
	return float64(tsr.Values[row*sz+cell])
}

func (tsr *Number[T]) SetFloatRowCell(row, cell int, val float64) {
	_, sz := tsr.Shp.RowCellSize()
	tsr.Values[row*sz+cell] = T(val)
}

// Floats sets []float64 slice of all elements in the tensor
// (length is ensured to be sufficient).
// This can be used for all of the gonum/floats methods
// for basic math, gonum/stats, etc.
func (tsr *Number[T]) Floats(flt *[]float64) {
	*flt = SetSliceLength(*flt, len(tsr.Values))
	switch vals := any(tsr.Values).(type) {
	case []float64:
		copy(*flt, vals)
	default:
		for i, v := range tsr.Values {
			(*flt)[i] = float64(v)
		}
	}
}

// SetFloats sets tensor values from a []float64 slice (copies values).
func (tsr *Number[T]) SetFloats(flt []float64) {
	switch vals := any(tsr.Values).(type) {
	case []float64:
		copy(vals, flt)
	default:
		for i, v := range flt {
			tsr.Values[i] = T(v)
		}
	}
}

// At is the gonum/mat.Matrix interface method for returning 2D matrix element at given
// row, column index.  Assumes Row-major ordering and logs an error if NumDims < 2.
func (tsr *Number[T]) At(i, j int) float64 {
	nd := tsr.NumDims()
	if nd < 2 {
		log.Println("tensor Dims gonum Matrix call made on Tensor with dims < 2")
		return 0
	} else if nd == 2 {
		return tsr.Float([]int{i, j})
	} else {
		ix := make([]int, nd)
		ix[nd-2] = i
		ix[nd-1] = j
		return tsr.Float(ix)
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
	csr := NewNumberShape[T](&tsr.Shp)
	copy(csr.Values, tsr.Values)
	if tsr.Nulls != nil {
		csr.Nulls = tsr.Nulls.Clone()
	}
	return csr
}

// CopyFrom copies all avail values from other tensor into this tensor, with an
// optimized implementation if the other tensor is of the same type, and
// otherwise it goes through appropriate standard type.
// Copies Null state as well if present.
func (tsr *Number[T]) CopyFrom(frm Tensor) {
	if fsm, ok := frm.(*Number[T]); ok {
		copy(tsr.Values, fsm.Values)
		if fsm.Nulls != nil {
			if tsr.Nulls == nil {
				tsr.Nulls = bitslice.Make(tsr.Len(), 0)
			}
			copy(tsr.Nulls, fsm.Nulls)
		}
		return
	}
	sz := min(len(tsr.Values), frm.Len())
	for i := 0; i < sz; i++ {
		tsr.Values[i] = T(frm.Float1D(i))
		if frm.IsNull1D(i) {
			tsr.SetNull1D(i, true)
		}
	}
}

// CopyShapeFrom copies just the shape from given source tensor
// calling SetShape with the shape params from source (see for more docs).
func (tsr *Number[T]) CopyShapeFrom(frm Tensor) {
	tsr.SetShape(frm.Shape().Sizes, frm.Shape().Names...)
}

// CopyCellsFrom copies given range of values from other tensor into this tensor,
// using flat 1D indexes: to = starting index in this Tensor to start copying into,
// start = starting index on from Tensor to start copying from, and n = number of
// values to copy.  Uses an optimized implementation if the other tensor is
// of the same type, and otherwise it goes through appropriate standard type.
func (tsr *Number[T]) CopyCellsFrom(frm Tensor, to, start, n int) {
	if fsm, ok := frm.(*Number[T]); ok {
		for i := 0; i < n; i++ {
			tsr.Values[to+i] = fsm.Values[start+i]
			if fsm.IsNull1D(start + i) {
				tsr.SetNull1D(to+i, true)
			}
		}
		return
	}
	for i := 0; i < n; i++ {
		tsr.Values[to+i] = T(frm.Float1D(start + i))
		if frm.IsNull1D(start + i) {
			tsr.SetNull1D(to+i, true)
		}
	}
}

// SubSpace returns a new tensor with innermost subspace at given
// offset(s) in outermost dimension(s) (len(offs) < NumDims).
// The new tensor points to the values of the this tensor (i.e., modifications
// will affect both), as its Values slice is a view onto the original (which
// is why only inner-most contiguous supsaces are supported).
// Use Clone() method to separate the two.
// Null value bits are NOT shared but are copied if present.
func (tsr *Number[T]) SubSpace(offs []int) Tensor {
	b := tsr.subSpaceImpl(offs)
	rt := &Number[T]{Base: *b}
	return rt
}
