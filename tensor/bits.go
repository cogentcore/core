// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package tensor

import (
	"fmt"
	"log/slog"
	"reflect"

	"cogentcore.org/core/base/metadata"
	"cogentcore.org/core/base/num"
	"cogentcore.org/core/base/reflectx"
	"cogentcore.org/core/base/slicesx"
	"cogentcore.org/core/tensor/bitslice"
	"gonum.org/v1/gonum/mat"
)

// Bits is a tensor of bits backed by a [bitslice.Slice] for efficient storage
// of binary data.  Bits does not support [Tensor.SubSpace] access and related
// methods due to the nature of the underlying data representation.
type Bits struct {
	shape  Shape
	Values bitslice.Slice
	Meta   metadata.Data
}

// NewBits returns a new n-dimensional tensor of bit values
// with the given sizes per dimension (shape), and optional dimension names.
func NewBits(sizes ...int) *Bits {
	tsr := &Bits{}
	tsr.SetShapeInts(sizes...)
	tsr.Values = bitslice.Make(tsr.Len(), 0)
	return tsr
}

// NewBitsShape returns a new n-dimensional tensor of bit values
// using given shape.
func NewBitsShape(shape *Shape) *Bits {
	tsr := &Bits{}
	tsr.shape.CopyShape(shape)
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
func (tsr *Bits) String() string {
	return stringIndexed(tsr, 0, nil)
}

func (tsr *Bits) IsString() bool {
	return false
}

// DataType returns the type of the data elements in the tensor.
// Bool is returned for the Bits tensor type.
func (tsr *Bits) DataType() reflect.Kind {
	return reflect.Bool
}

func (tsr *Bits) Sizeof() int64 {
	return int64(len(tsr.Values))
}

func (tsr *Bits) Bytes() []byte {
	return slicesx.ToBytes(tsr.Values)
}

func (tsr *Bits) Shape() *Shape { return &tsr.shape }

// ShapeSizes returns the sizes of each dimension as an int tensor.
func (tsr *Bits) ShapeSizes() Tensor { return tsr.shape.AsTensor() }

// ShapeInts returns the sizes of each dimension as a slice of ints.
// This is the preferred access for Go code.
func (tsr *Bits) ShapeInts() []int { return tsr.shape.Sizes }

// Metadata returns the metadata for this tensor, which can be used
// to encode plotting options, etc.
func (tsr *Bits) Metadata() *metadata.Data { return &tsr.Meta }

// Len returns the number of elements in the tensor (product of shape dimensions).
func (tsr *Bits) Len() int { return tsr.shape.Len() }

// NumDims returns the total number of dimensions.
func (tsr *Bits) NumDims() int { return tsr.shape.NumDims() }

// DimSize returns size of given dimension
func (tsr *Bits) DimSize(dim int) int { return tsr.shape.DimSize(dim) }

// RowCellSize returns the size of the outermost Row shape dimension,
// and the size of all the remaining inner dimensions (the "cell" size).
// Used for Tensors that are columns in a data table.
func (tsr *Bits) RowCellSize() (rows, cells int) {
	return tsr.shape.RowCellSize()
}

// Value returns value at given tensor index
func (tsr *Bits) Value(i ...int) bool {
	return tsr.Values.Index(tsr.shape.IndexTo1D(i...))
}

// Value1D returns value at given tensor 1D (flat) index
func (tsr *Bits) Value1D(i int) bool { return tsr.Values.Index(i) }

func (tsr *Bits) Set(val bool, i ...int) {
	tsr.Values.Set(val, tsr.shape.IndexTo1D(i...))
}

func (tsr *Bits) Set1D(val bool, i int) { tsr.Values.Set(val, i) }

func (tsr *Bits) SetShape(sizes Tensor) {
	tsr.shape.SetShape(sizes)
	nln := tsr.Len()
	tsr.Values.SetLen(nln)
}

func (tsr *Bits) SetShapeInts(sizes ...int) {
	tsr.shape.SetShapeInts(sizes...)
	nln := tsr.Len()
	tsr.Values.SetLen(nln)
}

// SetNumRows sets the number of rows (outermost dimension) in a RowMajor organized tensor.
// It is safe to set this to 0. For incrementally growing tensors (e.g., a log)
// it is best to first set the anticipated full size, which allocates the
// full amount of memory, and then set to 0 and grow incrementally.
func (tsr *Bits) SetNumRows(rows int) {
	_, cells := tsr.shape.RowCellSize()
	nln := rows * cells
	tsr.shape.Sizes[0] = rows
	tsr.Values.SetLen(nln)
}

// SubSpace is not possible with Bits.
func (tsr *Bits) SubSpace(offs ...int) Tensor {
	return nil
}

// RowTensor not possible with Bits.
func (tsr *Bits) RowTensor(row int) Tensor {
	return nil
}

// SetRowTensor not possible with Bits.
func (tsr *Bits) SetRowTensor(val Tensor, row int) {

}

/////////////////////  Strings

func (tsr *Bits) String1D(off int) string {
	return reflectx.ToString(tsr.Values.Index(off))
}

func (tsr *Bits) SetString1D(val string, off int) {
	if bv, err := reflectx.ToBool(val); err == nil {
		tsr.Values.Set(bv, off)
	}
}

func (tsr *Bits) StringValue(i ...int) string {
	return reflectx.ToString(tsr.Values.Index(tsr.shape.IndexTo1D(i...)))
}

func (tsr *Bits) SetString(val string, i ...int) {
	if bv, err := reflectx.ToBool(val); err == nil {
		tsr.Values.Set(bv, tsr.shape.IndexTo1D(i...))
	}
}

func (tsr *Bits) StringRowCell(row, cell int) string {
	_, sz := tsr.RowCellSize()
	return reflectx.ToString(tsr.Values.Index(row*sz + cell))
}

func (tsr *Bits) SetStringRowCell(val string, row, cell int) {
	if bv, err := reflectx.ToBool(val); err == nil {
		_, sz := tsr.RowCellSize()
		tsr.Values.Set(bv, row*sz+cell)
	}
}

func (tsr *Bits) StringRow(row int) string {
	return tsr.StringRowCell(row, 0)
}

func (tsr *Bits) SetStringRow(val string, row int) {
	tsr.SetStringRowCell(val, row, 0)
}

/////////////////////  Floats

func (tsr *Bits) Float(i ...int) float64 {
	return BoolToFloat64(tsr.Values.Index(tsr.shape.IndexTo1D(i...)))
}

func (tsr *Bits) SetFloat(val float64, i ...int) {
	tsr.Values.Set(Float64ToBool(val), tsr.shape.IndexTo1D(i...))
}

func (tsr *Bits) Float1D(off int) float64 {
	return BoolToFloat64(tsr.Values.Index(off))
}

func (tsr *Bits) SetFloat1D(val float64, off int) {
	tsr.Values.Set(Float64ToBool(val), off)
}

func (tsr *Bits) FloatRowCell(row, cell int) float64 {
	_, sz := tsr.RowCellSize()
	return BoolToFloat64(tsr.Values.Index(row*sz + cell))
}

func (tsr *Bits) SetFloatRowCell(val float64, row, cell int) {
	_, sz := tsr.RowCellSize()
	tsr.Values.Set(Float64ToBool(val), row*sz+cell)
}

func (tsr *Bits) FloatRow(row int) float64 {
	return tsr.FloatRowCell(row, 0)
}

func (tsr *Bits) SetFloatRow(val float64, row int) {
	tsr.SetFloatRowCell(val, row, 0)
}

/////////////////////  Ints

func (tsr *Bits) Int(i ...int) int {
	return BoolToInt(tsr.Values.Index(tsr.shape.IndexTo1D(i...)))
}

func (tsr *Bits) SetInt(val int, i ...int) {
	tsr.Values.Set(IntToBool(val), tsr.shape.IndexTo1D(i...))
}

func (tsr *Bits) Int1D(off int) int {
	return BoolToInt(tsr.Values.Index(off))
}

func (tsr *Bits) SetInt1D(val int, off int) {
	tsr.Values.Set(IntToBool(val), off)
}

func (tsr *Bits) IntRowCell(row, cell int) int {
	_, sz := tsr.RowCellSize()
	return BoolToInt(tsr.Values.Index(row*sz + cell))
}

func (tsr *Bits) SetIntRowCell(val int, row, cell int) {
	_, sz := tsr.RowCellSize()
	tsr.Values.Set(IntToBool(val), row*sz+cell)
}

func (tsr *Bits) IntRow(row int) int {
	return tsr.IntRowCell(row, 0)
}

func (tsr *Bits) SetIntRow(val int, row int) {
	tsr.SetIntRowCell(val, row, 0)
}

// Label satisfies the core.Labeler interface for a summary description of the tensor
func (tsr *Bits) Label() string {
	return fmt.Sprintf("tensor.Bits: %s", tsr.shape.String())
}

// Range is not applicable to Bits tensor
func (tsr *Bits) Range() (min, max float64, minIndex, maxIndex int) {
	minIndex = -1
	maxIndex = -1
	return
}

// SetZeros is simple convenience function initialize all values to 0
func (tsr *Bits) SetZeros() {
	ln := tsr.Len()
	for j := 0; j < ln; j++ {
		tsr.Values.Set(false, j)
	}
}

// Clone clones this tensor, creating a duplicate copy of itself with its
// own separate memory representation of all the values, and returns
// that as a Tensor (which can be converted into the known type as needed).
func (tsr *Bits) Clone() Tensor {
	csr := NewBitsShape(&tsr.shape)
	csr.Values = tsr.Values.Clone()
	return csr
}

func (tsr *Bits) View() Tensor {
	nw := &Bits{}
	nw.shape.CopyShape(&tsr.shape)
	nw.Values = tsr.Values
	nw.Meta = tsr.Meta
	return nw
}

// CopyFrom copies all avail values from other tensor into this tensor, with an
// optimized implementation if the other tensor is of the same type, and
// otherwise it goes through appropriate standard type.
func (tsr *Bits) CopyFrom(frm Tensor) {
	if fsm, ok := frm.(*Bits); ok {
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
func (tsr *Bits) AppendFrom(frm Tensor) error {
	rows, cell := tsr.RowCellSize()
	frows, fcell := frm.RowCellSize()
	if cell != fcell {
		return fmt.Errorf("tensor.AppendFrom: cell sizes do not match: %d != %d", cell, fcell)
	}
	tsr.SetNumRows(rows + frows)
	st := rows * cell
	fsz := frows * fcell
	if fsm, ok := frm.(*Bits); ok {
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
func (tsr *Bits) CopyCellsFrom(frm Tensor, to, start, n int) {
	if fsm, ok := frm.(*Bits); ok {
		for i := 0; i < n; i++ {
			tsr.Values.Set(fsm.Values.Index(start+i), to+i)
		}
		return
	}
	for i := 0; i < n; i++ {
		tsr.Values.Set(Float64ToBool(frm.Float1D(start+i)), to+i)
	}
}

// Dims is the gonum/mat.Matrix interface method for returning the dimensionality of the
// 2D Matrix.  Not supported for Bits -- do not call!
func (tsr *Bits) Dims() (r, c int) {
	slog.Error("tensor Dims gonum Matrix call made on Bits Tensor; not supported")
	return 0, 0
}

// At is the gonum/mat.Matrix interface method for returning 2D matrix element at given
// row, column index.  Not supported for Bits -- do not call!
func (tsr *Bits) At(i, j int) float64 {
	slog.Error("tensor At gonum Matrix call made on Bits Tensor; not supported")
	return 0
}

// T is the gonum/mat.Matrix transpose method.
// Not supported for Bits -- do not call!
func (tsr *Bits) T() mat.Matrix {
	slog.Error("tensor T gonum Matrix call made on Bits Tensor; not supported")
	return mat.Transpose{tsr}
}
