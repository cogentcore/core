// Copyright (c) 2024, The Cogent Core Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package tensor

import (
	"fmt"
	"log"
	"math"
	"strconv"
	"strings"

	"cogentcore.org/core/bitslice"
	"gonum.org/v1/gonum/mat"
)

// Float64 is a tensor of float64 values
type Float64 struct {
	Base[float64]
}

// NewFloat64 returns a new n-dimensional tensor of float64s.
// Nulls are initialized to nil.
func NewFloat64(sizes []int, names ...string) *Float64 {
	tsr := &Float64{}
	tsr.SetShape(sizes, names...)
	tsr.Values = make([]float64, tsr.Len())
	return tsr
}

// NewFloat64Shape returns a new n-dimensional tensor of float64s.
// Using shape and optionally existing values if vals != nil
// (must be of proper length). We directly set our internal
// Values = vals, thereby sharing the same
// underlying data. Nulls are initialized to nil.
func NewFloat64Shape(shape *Shape, vals []float64) *Float64 {
	tsr := &Float64{}
	tsr.Shp.CopyShape(shape)
	if vals != nil {
		if len(vals) != tsr.Len() {
			log.Printf("tensor.NewFloat64Shape: length of provided vals: %d not proper length: %d", len(vals), tsr.Len())
			tsr.Values = make([]float64, tsr.Len())
		} else {
			tsr.Values = vals
		}
	} else {
		tsr.Values = make([]float64, tsr.Len())
	}
	return tsr
}

func (tsr *Float64) AddScalar(i []int, val float64) float64 {
	j := tsr.Shp.Offset(i)
	tsr.Values[j] += val
	return tsr.Values[j]
}

func (tsr *Float64) MulScalar(i []int, val float64) float64 {
	j := tsr.Shp.Offset(i)
	tsr.Values[j] *= val
	return tsr.Values[j]
}

func (tsr *Float64) SetString(i []int, val string) {
	if fv, err := strconv.ParseFloat(val, 64); err == nil {
		j := tsr.Shp.Offset(i)
		tsr.Values[j] = float64(fv)
	}
}

func (tsr Float64) SetString1D(off int, val string) {
	if fv, err := strconv.ParseFloat(val, 64); err == nil {
		tsr.Values[off] = float64(fv)
	}
}
func (tsr *Float64) SetStringRowCell(row, cell int, val string) {
	if fv, err := strconv.ParseFloat(val, 64); err == nil {
		_, sz := tsr.Shp.RowCellSize()
		tsr.Values[row*sz+cell] = float64(fv)
	}
}

// String satisfies the fmt.Stringer interface for string of tensor data
func (tsr *Float64) String() string {
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

func (tsr *Float64) Float(i []int) float64 {
	j := tsr.Shp.Offset(i)
	return float64(tsr.Values[j])
}

func (tsr *Float64) SetFloat(i []int, val float64) {
	j := tsr.Shp.Offset(i)
	tsr.Values[j] = float64(val)
}

func (tsr *Float64) Float1D(off int) float64 {
	return float64(tsr.Values[off])
}

func (tsr *Float64) SetFloat1D(off int, val float64) {
	tsr.Values[off] = float64(val)
}

func (tsr *Float64) FloatRowCell(row, cell int) float64 {
	_, sz := tsr.Shp.RowCellSize()
	return float64(tsr.Values[row*sz+cell])
}

func (tsr *Float64) SetFloatRowCell(row, cell int, val float64) {
	_, sz := tsr.Shp.RowCellSize()
	tsr.Values[row*sz+cell] = float64(val)
}

// Floats sets []float64 slice of all elements in the tensor
// (length is ensured to be sufficient).
// This can be used for all of the gonum/floats methods
// for basic math, gonum/stats, etc.
func (tsr *Float64) Floats(flt *[]float64) {
	SetFloat64SliceLen(flt, len(tsr.Values))
	copy(*flt, tsr.Values) // diff: blit from values directly
}

// SetFloats sets tensor values from a []float64 slice (copies values).
func (tsr *Float64) SetFloats(vals []float64) {
	copy(tsr.Values, vals) // diff: blit from values directly
}

// At is the gonum/mat.Matrix interface method for returning 2D matrix element at given
// row, column index.  Assumes Row-major ordering and logs an error if NumDims < 2.
func (tsr *Float64) At(i, j int) float64 {
	nd := tsr.NumDims()
	if nd < 2 {
		log.Println("etensor Dims gonum Matrix call made on Tensor with dims < 2")
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
func (tsr *Float64) T() mat.Matrix {
	return mat.Transpose{tsr}
}

// Range returns the min, max (and associated indexes, -1 = no values) for the tensor.
// This is needed for display and is thus in the core api in optimized form
// Other math operations can be done using gonum/floats package.
func (tsr *Float64) Range() (min, max float64, minIndex, maxIndex int) {
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
func (tsr *Float64) SetZeros() {
	for j := range tsr.Values {
		tsr.Values[j] = 0
	}
}

// Clone clones this tensor, creating a duplicate copy of itself with its
// own separate memory representation of all the values, and returns
// that as a Tensor (which can be converted into the known type as needed).
func (tsr *Float64) Clone() Tensor {
	csr := NewFloat64Shape(&tsr.Shp, nil)
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
func (tsr *Float64) CopyFrom(frm Tensor) {
	if fsm, ok := frm.(*Float64); ok {
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
		tsr.Values[i] = float64(frm.Float1D(i))
		if frm.IsNull1D(i) {
			tsr.SetNull1D(i, true)
		}
	}
}

// CopyShapeFrom copies just the shape from given source tensor
// calling SetShape with the shape params from source (see for more docs).
func (tsr *Float64) CopyShapeFrom(frm Tensor) {
	tsr.SetShape(frm.Shape().Sizes, frm.Shape().Names...)
}

// CopyCellsFrom copies given range of values from other tensor into this tensor,
// using flat 1D indexes: to = starting index in this Tensor to start copying into,
// start = starting index on from Tensor to start copying from, and n = number of
// values to copy.  Uses an optimized implementation if the other tensor is
// of the same type, and otherwise it goes through appropriate standard type.
func (tsr *Float64) CopyCellsFrom(frm Tensor, to, start, n int) {
	if fsm, ok := frm.(*Float64); ok {
		for i := 0; i < n; i++ {
			tsr.Values[to+i] = fsm.Values[start+i]
			if fsm.IsNull1D(start + i) {
				tsr.SetNull1D(to+i, true)
			}
		}
		return
	}
	for i := 0; i < n; i++ {
		tsr.Values[to+i] = float64(frm.Float1D(start + i))
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
func (tsr *Float64) SubSpace(offs []int) Tensor {
	b := tsr.subSpaceImpl(offs)
	rt := &Float64{Base: *b}
	return rt
}

// Check for interface implementation
var _ Tensor = (*Float64)(nil)
