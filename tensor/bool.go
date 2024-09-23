// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package tensor

import (
	"fmt"
	"reflect"

	"cogentcore.org/core/base/metadata"
	"cogentcore.org/core/base/num"
	"cogentcore.org/core/base/reflectx"
	"cogentcore.org/core/base/slicesx"
	"cogentcore.org/core/tensor/bitslice"
)

// Bool is a tensor of bits backed by a [bitslice.Slice] for efficient storage
// of binary, boolean data. Bool does not support [Tensor.SubSpace] access
// and related methods due to the nature of the underlying data representation.
type Bool struct {
	shape  Shape
	Values bitslice.Slice
	Meta   metadata.Data
}

// NewBool returns a new n-dimensional tensor of bit values
// with the given sizes per dimension (shape).
func NewBool(sizes ...int) *Bool {
	tsr := &Bool{}
	tsr.SetShapeSizes(sizes...)
	tsr.Values = bitslice.Make(tsr.Len(), 0)
	return tsr
}

// NewBoolShape returns a new n-dimensional tensor of bit values
// using given shape.
func NewBoolShape(shape *Shape) *Bool {
	tsr := &Bool{}
	tsr.shape.CopyFrom(shape)
	tsr.Values = bitslice.Make(tsr.Len(), 0)
	return tsr
}

// Float64ToBool converts float64 value to bool.
func Float64ToBool(val float64) bool {
	return num.ToBool(val)
}

// BoolToFloat64 converts bool to float64 value.
func BoolToFloat64(bv bool) float64 {
	return num.FromBool[float64](bv)
}

// IntToBool converts int value to bool.
func IntToBool(val int) bool {
	return num.ToBool(val)
}

// BoolToInt converts bool to int value.
func BoolToInt(bv bool) int {
	return num.FromBool[int](bv)
}

// String satisfies the fmt.Stringer interface for string of tensor data.
func (tsr *Bool) String() string { return sprint(tsr, 0) }

// Label satisfies the core.Labeler interface for a summary description of the tensor
func (tsr *Bool) Label() string {
	return label(tsr.Metadata().Name(), tsr.Shape())
}

func (tsr *Bool) IsString() bool { return false }

func (tsr *Bool) AsValues() Values { return tsr }

// DataType returns the type of the data elements in the tensor.
// Bool is returned for the Bool tensor type.
func (tsr *Bool) DataType() reflect.Kind { return reflect.Bool }

func (tsr *Bool) Sizeof() int64 { return int64(len(tsr.Values)) }

func (tsr *Bool) Bytes() []byte { return slicesx.ToBytes(tsr.Values) }

func (tsr *Bool) Shape() *Shape { return &tsr.shape }

// ShapeSizes returns the sizes of each dimension as a slice of ints.
// This is the preferred access for Go code.
func (tsr *Bool) ShapeSizes() []int { return tsr.shape.Sizes }

// Metadata returns the metadata for this tensor, which can be used
// to encode plotting options, etc.
func (tsr *Bool) Metadata() *metadata.Data { return &tsr.Meta }

// Len returns the number of elements in the tensor (product of shape dimensions).
func (tsr *Bool) Len() int { return tsr.shape.Len() }

// NumDims returns the total number of dimensions.
func (tsr *Bool) NumDims() int { return tsr.shape.NumDims() }

// DimSize returns size of given dimension
func (tsr *Bool) DimSize(dim int) int { return tsr.shape.DimSize(dim) }

// RowCellSize returns the size of the outermost Row shape dimension,
// and the size of all the remaining inner dimensions (the "cell" size).
// Used for Tensors that are columns in a data table.
func (tsr *Bool) RowCellSize() (rows, cells int) {
	return tsr.shape.RowCellSize()
}

func (tsr *Bool) SetShapeSizes(sizes ...int) {
	tsr.shape.SetShapeSizes(sizes...)
	nln := tsr.Len()
	tsr.Values.SetLen(nln)
}

// SetNumRows sets the number of rows (outermost dimension) in a RowMajor organized tensor.
// It is safe to set this to 0. For incrementally growing tensors (e.g., a log)
// it is best to first set the anticipated full size, which allocates the
// full amount of memory, and then set to 0 and grow incrementally.
func (tsr *Bool) SetNumRows(rows int) {
	_, cells := tsr.shape.RowCellSize()
	nln := rows * cells
	tsr.shape.Sizes[0] = rows
	tsr.Values.SetLen(nln)
}

// SubSpace is not possible with Bool.
func (tsr *Bool) SubSpace(offs ...int) Values { return nil }

// RowTensor not possible with Bool.
func (tsr *Bool) RowTensor(row int) Values { return nil }

// SetRowTensor not possible with Bool.
func (tsr *Bool) SetRowTensor(val Values, row int) {}

/////////////////////  Strings

func (tsr *Bool) String1D(off int) string {
	return reflectx.ToString(tsr.Values.Index(off))
}

func (tsr *Bool) SetString1D(val string, off int) {
	if bv, err := reflectx.ToBool(val); err == nil {
		tsr.Values.Set(bv, off)
	}
}

func (tsr *Bool) StringValue(i ...int) string {
	return reflectx.ToString(tsr.Values.Index(tsr.shape.IndexTo1D(i...)))
}

func (tsr *Bool) SetString(val string, i ...int) {
	if bv, err := reflectx.ToBool(val); err == nil {
		tsr.Values.Set(bv, tsr.shape.IndexTo1D(i...))
	}
}

func (tsr *Bool) StringRowCell(row, cell int) string {
	_, sz := tsr.RowCellSize()
	return reflectx.ToString(tsr.Values.Index(row*sz + cell))
}

func (tsr *Bool) SetStringRowCell(val string, row, cell int) {
	if bv, err := reflectx.ToBool(val); err == nil {
		_, sz := tsr.RowCellSize()
		tsr.Values.Set(bv, row*sz+cell)
	}
}

func (tsr *Bool) StringRow(row int) string {
	return tsr.StringRowCell(row, 0)
}

func (tsr *Bool) SetStringRow(val string, row int) {
	tsr.SetStringRowCell(val, row, 0)
}

/////////////////////  Floats

func (tsr *Bool) Float(i ...int) float64 {
	return BoolToFloat64(tsr.Values.Index(tsr.shape.IndexTo1D(i...)))
}

func (tsr *Bool) SetFloat(val float64, i ...int) {
	tsr.Values.Set(Float64ToBool(val), tsr.shape.IndexTo1D(i...))
}

func (tsr *Bool) Float1D(off int) float64 {
	return BoolToFloat64(tsr.Values.Index(off))
}

func (tsr *Bool) SetFloat1D(val float64, off int) {
	tsr.Values.Set(Float64ToBool(val), off)
}

func (tsr *Bool) FloatRowCell(row, cell int) float64 {
	_, sz := tsr.RowCellSize()
	return BoolToFloat64(tsr.Values.Index(row*sz + cell))
}

func (tsr *Bool) SetFloatRowCell(val float64, row, cell int) {
	_, sz := tsr.RowCellSize()
	tsr.Values.Set(Float64ToBool(val), row*sz+cell)
}

func (tsr *Bool) FloatRow(row int) float64 {
	return tsr.FloatRowCell(row, 0)
}

func (tsr *Bool) SetFloatRow(val float64, row int) {
	tsr.SetFloatRowCell(val, row, 0)
}

/////////////////////  Ints

func (tsr *Bool) Int(i ...int) int {
	return BoolToInt(tsr.Values.Index(tsr.shape.IndexTo1D(i...)))
}

func (tsr *Bool) SetInt(val int, i ...int) {
	tsr.Values.Set(IntToBool(val), tsr.shape.IndexTo1D(i...))
}

func (tsr *Bool) Int1D(off int) int {
	return BoolToInt(tsr.Values.Index(off))
}

func (tsr *Bool) SetInt1D(val int, off int) {
	tsr.Values.Set(IntToBool(val), off)
}

func (tsr *Bool) IntRowCell(row, cell int) int {
	_, sz := tsr.RowCellSize()
	return BoolToInt(tsr.Values.Index(row*sz + cell))
}

func (tsr *Bool) SetIntRowCell(val int, row, cell int) {
	_, sz := tsr.RowCellSize()
	tsr.Values.Set(IntToBool(val), row*sz+cell)
}

func (tsr *Bool) IntRow(row int) int {
	return tsr.IntRowCell(row, 0)
}

func (tsr *Bool) SetIntRow(val int, row int) {
	tsr.SetIntRowCell(val, row, 0)
}

/////////////////////  Bools

func (tsr *Bool) Bool(i ...int) bool {
	return tsr.Values.Index(tsr.shape.IndexTo1D(i...))
}

func (tsr *Bool) SetBool(val bool, i ...int) {
	tsr.Values.Set(val, tsr.shape.IndexTo1D(i...))
}

func (tsr *Bool) Bool1D(off int) bool {
	return tsr.Values.Index(off)
}

func (tsr *Bool) SetBool1D(val bool, off int) {
	tsr.Values.Set(val, off)
}

// SetZeros is a convenience function initialize all values to 0 (false).
func (tsr *Bool) SetZeros() {
	ln := tsr.Len()
	for j := 0; j < ln; j++ {
		tsr.Values.Set(false, j)
	}
}

// SetTrue is simple convenience function initialize all values to 0
func (tsr *Bool) SetTrue() {
	ln := tsr.Len()
	for j := 0; j < ln; j++ {
		tsr.Values.Set(true, j)
	}
}

// Clone clones this tensor, creating a duplicate copy of itself with its
// own separate memory representation of all the values, and returns
// that as a Tensor (which can be converted into the known type as needed).
func (tsr *Bool) Clone() Values {
	csr := NewBoolShape(&tsr.shape)
	csr.Values = tsr.Values.Clone()
	return csr
}

// CopyFrom copies all avail values from other tensor into this tensor, with an
// optimized implementation if the other tensor is of the same type, and
// otherwise it goes through appropriate standard type.
func (tsr *Bool) CopyFrom(frm Values) {
	if fsm, ok := frm.(*Bool); ok {
		copy(tsr.Values, fsm.Values)
		return
	}
	sz := min(len(tsr.Values), frm.Len())
	for i := 0; i < sz; i++ {
		tsr.Values.Set(Float64ToBool(frm.Float1D(i)), i)
	}
}

// AppendFrom appends values from other tensor into this tensor,
// which must have the same cell size as this tensor.
// It uses and optimized implementation if the other tensor
// is of the same type, and otherwise it goes through
// appropriate standard type.
func (tsr *Bool) AppendFrom(frm Values) error {
	rows, cell := tsr.RowCellSize()
	frows, fcell := frm.RowCellSize()
	if cell != fcell {
		return fmt.Errorf("tensor.AppendFrom: cell sizes do not match: %d != %d", cell, fcell)
	}
	tsr.SetNumRows(rows + frows)
	st := rows * cell
	fsz := frows * fcell
	if fsm, ok := frm.(*Bool); ok {
		copy(tsr.Values[st:st+fsz], fsm.Values)
		return nil
	}
	for i := 0; i < fsz; i++ {
		tsr.Values.Set(Float64ToBool(frm.Float1D(i)), st+i)
	}
	return nil
}

// CopyCellsFrom copies given range of values from other tensor into this tensor,
// using flat 1D indexes: to = starting index in this Tensor to start copying into,
// start = starting index on from Tensor to start copying from, and n = number of
// values to copy.  Uses an optimized implementation if the other tensor is
// of the same type, and otherwise it goes through appropriate standard type.
func (tsr *Bool) CopyCellsFrom(frm Values, to, start, n int) {
	if fsm, ok := frm.(*Bool); ok {
		for i := 0; i < n; i++ {
			tsr.Values.Set(fsm.Values.Index(start+i), to+i)
		}
		return
	}
	for i := 0; i < n; i++ {
		tsr.Values.Set(Float64ToBool(frm.Float1D(start+i)), to+i)
	}
}
