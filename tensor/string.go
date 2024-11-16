// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package tensor

import (
	"fmt"
	"strconv"

	"cogentcore.org/core/base/errors"
)

// String is a tensor of string values
type String struct {
	Base[string]
}

// NewString returns a new n-dimensional tensor of string values
// with the given sizes per dimension (shape).
func NewString(sizes ...int) *String {
	tsr := &String{}
	tsr.SetShapeSizes(sizes...)
	tsr.Values = make([]string, tsr.Len())
	return tsr
}

// NewStringShape returns a new n-dimensional tensor of string values
// using given shape.
func NewStringShape(shape *Shape) *String {
	tsr := &String{}
	tsr.shape.CopyFrom(shape)
	tsr.Values = make([]string, shape.Header+tsr.Len())
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

// String satisfies the fmt.Stringer interface for string of tensor data.
func (tsr *String) String() string {
	return Sprintf("", tsr, 0)
}

func (tsr *String) IsString() bool {
	return true
}

func (tsr *String) AsValues() Values { return tsr }

///////  Strings

func (tsr *String) SetString(val string, i ...int) {
	tsr.Values[tsr.shape.IndexTo1D(i...)] = val
}

func (tsr *String) String1D(i int) string {
	return tsr.Values[tsr.shape.Header+i]
}

func (tsr *String) SetString1D(val string, i int) {
	tsr.Values[tsr.shape.Header+NegIndex(i, len(tsr.Values))] = val
}

func (tsr *String) StringRow(row, cell int) string {
	_, sz := tsr.shape.RowCellSize()
	return tsr.Values[tsr.shape.Header+row*sz+cell]
}

func (tsr *String) SetStringRow(val string, row, cell int) {
	_, sz := tsr.shape.RowCellSize()
	tsr.Values[tsr.shape.Header+row*sz+cell] = val
}

// AppendRowString adds a row and sets string value(s), up to number of cells.
func (tsr *String) AppendRowString(val ...string) {
	if tsr.NumDims() == 0 {
		tsr.SetShapeSizes(0)
	}
	nrow, sz := tsr.shape.RowCellSize()
	tsr.SetNumRows(nrow + 1)
	mx := min(sz, len(val))
	for i := range mx {
		tsr.SetStringRow(val[i], nrow, i)
	}
}

///////  Floats

func (tsr *String) Float(i ...int) float64 {
	return StringToFloat64(tsr.Values[tsr.shape.IndexTo1D(i...)])
}

func (tsr *String) SetFloat(val float64, i ...int) {
	tsr.Values[tsr.shape.IndexTo1D(i...)] = Float64ToString(val)
}

func (tsr *String) Float1D(i int) float64 {
	return StringToFloat64(tsr.Values[tsr.shape.Header+NegIndex(i, len(tsr.Values))])
}

func (tsr *String) SetFloat1D(val float64, i int) {
	tsr.Values[tsr.shape.Header+NegIndex(i, len(tsr.Values))] = Float64ToString(val)
}

func (tsr *String) FloatRow(row, cell int) float64 {
	_, sz := tsr.shape.RowCellSize()
	return StringToFloat64(tsr.Values[tsr.shape.Header+row*sz+cell])
}

func (tsr *String) SetFloatRow(val float64, row, cell int) {
	_, sz := tsr.shape.RowCellSize()
	tsr.Values[tsr.shape.Header+row*sz+cell] = Float64ToString(val)
}

// AppendRowFloat adds a row and sets float value(s), up to number of cells.
func (tsr *String) AppendRowFloat(val ...float64) {
	if tsr.NumDims() == 0 {
		tsr.SetShapeSizes(0)
	}
	nrow, sz := tsr.shape.RowCellSize()
	tsr.SetNumRows(nrow + 1)
	mx := min(sz, len(val))
	for i := range mx {
		tsr.SetFloatRow(val[i], nrow, i)
	}
}

///////  Ints

func (tsr *String) Int(i ...int) int {
	return errors.Ignore1(strconv.Atoi(tsr.Values[tsr.shape.IndexTo1D(i...)]))
}

func (tsr *String) SetInt(val int, i ...int) {
	tsr.Values[tsr.shape.IndexTo1D(i...)] = strconv.Itoa(val)
}

func (tsr *String) Int1D(i int) int {
	return errors.Ignore1(strconv.Atoi(tsr.Values[tsr.shape.Header+NegIndex(i, len(tsr.Values))]))
}

func (tsr *String) SetInt1D(val int, i int) {
	tsr.Values[tsr.shape.Header+NegIndex(i, len(tsr.Values))] = strconv.Itoa(val)
}

func (tsr *String) IntRow(row, cell int) int {
	_, sz := tsr.shape.RowCellSize()
	return errors.Ignore1(strconv.Atoi(tsr.Values[tsr.shape.Header+row*sz+cell]))
}

func (tsr *String) SetIntRow(val int, row, cell int) {
	_, sz := tsr.shape.RowCellSize()
	tsr.Values[tsr.shape.Header+row*sz+cell] = strconv.Itoa(val)
}

// AppendRowInt adds a row and sets int value(s), up to number of cells.
func (tsr *String) AppendRowInt(val ...int) {
	if tsr.NumDims() == 0 {
		tsr.SetShapeSizes(0)
	}
	nrow, sz := tsr.shape.RowCellSize()
	tsr.SetNumRows(nrow + 1)
	mx := min(sz, len(val))
	for i := range mx {
		tsr.SetIntRow(val[i], nrow, i)
	}
}

// SetZeros is a simple convenience function initialize all values to the
// zero value of the type (empty strings for string type).
func (tsr *String) SetZeros() {
	n := len(tsr.Values)
	for j := tsr.shape.Header; j < n; j++ {
		tsr.Values[j] = ""
	}
}

// Clone clones this tensor, creating a duplicate copy of itself with its
// own separate memory representation of all the values, and returns
// that as a Tensor (which can be converted into the known type as needed).
func (tsr *String) Clone() Values {
	csr := NewStringShape(&tsr.shape)
	copy(csr.Values, tsr.Values)
	return csr
}

// CopyFrom copies all avail values from other tensor into this tensor, with an
// optimized implementation if the other tensor is of the same type, and
// otherwise it goes through appropriate standard type.
func (tsr *String) CopyFrom(frm Values) {
	if fsm, ok := frm.(*String); ok {
		copy(tsr.Values[tsr.shape.Header:], fsm.Values[fsm.shape.Header:])
		return
	}
	sz := min(tsr.Len(), frm.Len())
	for i := 0; i < sz; i++ {
		tsr.Values[tsr.shape.Header+i] = Float64ToString(frm.Float1D(i))
	}
}

// AppendFrom appends values from other tensor into this tensor,
// which must have the same cell size as this tensor.
// It uses and optimized implementation if the other tensor
// is of the same type, and otherwise it goes through
// appropriate standard type.
func (tsr *String) AppendFrom(frm Values) error {
	rows, cell := tsr.shape.RowCellSize()
	frows, fcell := frm.Shape().RowCellSize()
	if cell != fcell {
		return fmt.Errorf("tensor.AppendFrom: cell sizes do not match: %d != %d", cell, fcell)
	}
	tsr.SetNumRows(rows + frows)
	st := tsr.shape.Header + rows*cell
	fsz := frows * fcell
	if fsm, ok := frm.(*String); ok {
		copy(tsr.Values[st:st+fsz], fsm.Values[fsm.shape.Header:])
		return nil
	}
	for i := 0; i < fsz; i++ {
		tsr.Values[st+i] = Float64ToString(frm.Float1D(i))
	}
	return nil
}

// CopyCellsFrom copies given range of values from other tensor into this tensor,
// using flat 1D indexes: to = starting index in this Tensor to start copying into,
// start = starting index on from Tensor to start copying from, and n = number of
// values to copy.  Uses an optimized implementation if the other tensor is
// of the same type, and otherwise it goes through appropriate standard type.
func (tsr *String) CopyCellsFrom(frm Values, to, start, n int) {
	to += tsr.shape.Header
	if fsm, ok := frm.(*String); ok {
		start += fsm.shape.Header
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
func (tsr *String) SubSpace(offs ...int) Values {
	b := tsr.subSpaceImpl(offs...)
	rt := &String{Base: *b}
	return rt
}

// RowTensor is a convenience version of [RowMajor.SubSpace] to return the
// SubSpace for the outermost row dimension. [Rows] defines a version
// of this that indirects through the row indexes.
func (tsr *String) RowTensor(row int) Values {
	return tsr.SubSpace(row)
}

// SetRowTensor sets the values of the SubSpace at given row to given values.
func (tsr *String) SetRowTensor(val Values, row int) {
	_, cells := tsr.shape.RowCellSize()
	st := row * cells
	mx := min(val.Len(), cells)
	tsr.CopyCellsFrom(val, st, 0, mx)
}

// AppendRow adds a row and sets values to given values.
func (tsr *String) AppendRow(val Values) {
	if tsr.NumDims() == 0 {
		tsr.SetShapeSizes(0)
	}
	nrow := tsr.DimSize(0)
	tsr.SetNumRows(nrow + 1)
	tsr.SetRowTensor(val, nrow)
}
