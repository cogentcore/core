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

	"gonum.org/v1/gonum/mat"
)

// String is a tensor of string values
type String struct {
	Base[string]
}

// NewString returns a new n-dimensional tensor of string values
// with the given sizes per dimension (shape).
func NewString(sizes ...int) *String {
	tsr := &String{}
	tsr.SetShape(sizes...)
	tsr.Values = make([]string, tsr.Len())
	return tsr
}

// NewStringShape returns a new n-dimensional tensor of string values
// using given shape.
func NewStringShape(shape *Shape) *String {
	tsr := &String{}
	tsr.shape.CopyShape(shape)
	tsr.Values = make([]string, tsr.Len())
	return tsr
}

// NewStringFromSlice returns a new 1-dimensional tensor of given value type
// initialized directly from the given slice values, which are not copied.
// The resulting Tensor thus "wraps" the given values.
func NewStringFromSlice(vals ...string) Tensor {
	n := len(vals)
	tsr := &String{}
	tsr.Values = vals
	tsr.SetShape(n)
	return tsr
}

// StringToFloat64 converts string value to float64 using strconv,
// returning 0 if any error
func StringToFloat64(str string) float64 {
	if fv, err := strconv.ParseFloat(str, 64); err == nil {
		return fv
	}
	return 0
}

// Float64ToString converts float64 to string value using strconv, g format
func Float64ToString(val float64) string {
	return strconv.FormatFloat(val, 'g', -1, 64)
}

func (tsr *String) IsString() bool {
	return true
}

func (tsr *String) SetString(val string, i ...int) {
	j := tsr.shape.Offset(i...)
	tsr.Values[j] = val
}

func (tsr String) SetString1D(val string, off int) {
	tsr.Values[off] = val
}

func (tsr *String) SetStringRowCell(val string, row, cell int) {
	_, sz := tsr.shape.RowCellSize()
	tsr.Values[row*sz+cell] = val
}

// String satisfies the fmt.Stringer interface for string of tensor data
func (tsr *String) String() string {
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
			idx := Projection2DIndex(tsr.Shape(), oddRow, r, c)
			vl := tsr.Values[idx]
			b.WriteString(vl)
		}
		b.WriteString("\n")
	}
	return b.String()
}

func (tsr *String) Float(i ...int) float64 {
	return StringToFloat64(tsr.Values[tsr.shape.Offset(i...)])
}

func (tsr *String) SetFloat(val float64, i ...int) {
	tsr.Values[tsr.shape.Offset(i...)] = Float64ToString(val)
}

func (tsr *String) Float1D(off int) float64 {
	return StringToFloat64(tsr.Values[off])
}

func (tsr *String) SetFloat1D(val float64, off int) {
	tsr.Values[off] = Float64ToString(val)
}

func (tsr *String) FloatRowCell(row, cell int) float64 {
	_, sz := tsr.shape.RowCellSize()
	return StringToFloat64(tsr.Values[row*sz+cell])
}

func (tsr *String) SetFloatRowCell(val float64, row, cell int) {
	_, sz := tsr.shape.RowCellSize()
	tsr.Values[row*sz+cell] = Float64ToString(val)
}

// At is the gonum/mat.Matrix interface method for returning 2D matrix element at given
// row, column index.  Assumes Row-major ordering and logs an error if NumDims < 2.
func (tsr *String) At(i, j int) float64 {
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
func (tsr *String) T() mat.Matrix {
	return mat.Transpose{tsr}
}

// Range returns the min, max (and associated indexes, -1 = no values) for the tensor.
// This is needed for display and is thus in the core api in optimized form
// Other math operations can be done using gonum/floats package.
func (tsr *String) Range() (min, max float64, minIndex, maxIndex int) {
	minIndex = -1
	maxIndex = -1
	for j, vl := range tsr.Values {
		fv := StringToFloat64(vl)
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

// SetZeros is a simple convenience function initialize all values to the
// zero value of the type (empty strings for string type).
func (tsr *String) SetZeros() {
	for j := range tsr.Values {
		tsr.Values[j] = ""
	}
}

// Clone clones this tensor, creating a duplicate copy of itself with its
// own separate memory representation of all the values, and returns
// that as a Tensor (which can be converted into the known type as needed).
func (tsr *String) Clone() Tensor {
	csr := NewStringShape(&tsr.shape)
	copy(csr.Values, tsr.Values)
	return csr
}

func (tsr *String) View() Tensor {
	return &String{*tsr.view()}
}

// CopyFrom copies all avail values from other tensor into this tensor, with an
// optimized implementation if the other tensor is of the same type, and
// otherwise it goes through appropriate standard type.
func (tsr *String) CopyFrom(frm Tensor) {
	if fsm, ok := frm.(*String); ok {
		copy(tsr.Values, fsm.Values)
		return
	}
	sz := min(len(tsr.Values), frm.Len())
	for i := 0; i < sz; i++ {
		tsr.Values[i] = Float64ToString(frm.Float1D(i))
	}
}

// SetShapeFrom copies just the shape from given source tensor
// calling SetShape with the shape params from source (see for more docs).
func (tsr *String) SetShapeFrom(frm Tensor) {
	sh := frm.Shape()
	tsr.SetShape(sh.Sizes...)
	tsr.SetNames(sh.Names...)
}

// CopyCellsFrom copies given range of values from other tensor into this tensor,
// using flat 1D indexes: to = starting index in this Tensor to start copying into,
// start = starting index on from Tensor to start copying from, and n = number of
// values to copy.  Uses an optimized implementation if the other tensor is
// of the same type, and otherwise it goes through appropriate standard type.
func (tsr *String) CopyCellsFrom(frm Tensor, to, start, n int) {
	if fsm, ok := frm.(*String); ok {
		for i := 0; i < n; i++ {
			tsr.Values[to+i] = fsm.Values[start+i]
		}
		return
	}
	for i := 0; i < n; i++ {
		tsr.Values[to+i] = Float64ToString(frm.Float1D(start + i))
	}
}

// SubSpace returns a new tensor with innermost subspace at given
// offset(s) in outermost dimension(s) (len(offs) < NumDims).
// The new tensor points to the values of the this tensor (i.e., modifications
// will affect both), as its Values slice is a view onto the original (which
// is why only inner-most contiguous supsaces are supported).
// Use Clone() method to separate the two.
func (tsr *String) SubSpace(offs ...int) Tensor {
	b := tsr.subSpaceImpl(offs...)
	rt := &String{Base: *b}
	return rt
}
