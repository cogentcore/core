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
	tsr.Values = make([]string, tsr.Len())
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
	return Sprintf(tsr, 0, "")
}

func (tsr *String) IsString() bool {
	return true
}

func (tsr *String) AsValues() Values { return tsr }

/////////////////////  Strings

func (tsr *String) SetString(val string, i ...int) {
	j := tsr.shape.IndexTo1D(i...)
	tsr.Values[j] = val
}

func (tsr String) SetString1D(val string, off int) {
	tsr.Values[off] = val
}

func (tsr *String) SetStringRowCell(val string, row, cell int) {
	_, sz := tsr.shape.RowCellSize()
	tsr.Values[row*sz+cell] = val
}

func (tsr *String) StringRow(row int) string {
	return tsr.StringRowCell(row, 0)
}

func (tsr *String) SetStringRow(val string, row int) {
	tsr.SetStringRowCell(val, row, 0)
}

/////////////////////  Floats

func (tsr *String) Float(i ...int) float64 {
	return StringToFloat64(tsr.Values[tsr.shape.IndexTo1D(i...)])
}

func (tsr *String) SetFloat(val float64, i ...int) {
	tsr.Values[tsr.shape.IndexTo1D(i...)] = Float64ToString(val)
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

func (tsr *String) FloatRow(row int) float64 {
	return tsr.FloatRowCell(row, 0)
}

func (tsr *String) SetFloatRow(val float64, row int) {
	tsr.SetFloatRowCell(val, row, 0)
}

/////////////////////  Ints

func (tsr *String) Int(i ...int) int {
	return errors.Ignore1(strconv.Atoi(tsr.Values[tsr.shape.IndexTo1D(i...)]))
}

func (tsr *String) SetInt(val int, i ...int) {
	tsr.Values[tsr.shape.IndexTo1D(i...)] = strconv.Itoa(val)
}

func (tsr *String) Int1D(off int) int {
	return errors.Ignore1(strconv.Atoi(tsr.Values[off]))
}

func (tsr *String) SetInt1D(val int, off int) {
	tsr.Values[off] = strconv.Itoa(val)
}

func (tsr *String) IntRowCell(row, cell int) int {
	_, sz := tsr.shape.RowCellSize()
	return errors.Ignore1(strconv.Atoi(tsr.Values[row*sz+cell]))
}

func (tsr *String) SetIntRowCell(val int, row, cell int) {
	_, sz := tsr.shape.RowCellSize()
	tsr.Values[row*sz+cell] = strconv.Itoa(val)
}

func (tsr *String) IntRow(row int) int {
	return tsr.IntRowCell(row, 0)
}

func (tsr *String) SetIntRow(val int, row int) {
	tsr.SetIntRowCell(val, row, 0)
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
		copy(tsr.Values, fsm.Values)
		return
	}
	sz := min(len(tsr.Values), frm.Len())
	for i := 0; i < sz; i++ {
		tsr.Values[i] = Float64ToString(frm.Float1D(i))
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
	st := rows * cell
	fsz := frows * fcell
	if fsm, ok := frm.(*String); ok {
		copy(tsr.Values[st:st+fsz], fsm.Values)
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
func (tsr *String) SubSpace(offs ...int) Values {
	b := tsr.subSpaceImpl(offs...)
	rt := &String{Base: *b}
	return rt
}

// RowTensor is a convenience version of [Tensor.SubSpace] to return the
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
