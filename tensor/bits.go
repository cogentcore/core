// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package tensor

import (
	"fmt"
	"log/slog"
	"reflect"
	"strings"

	"cogentcore.org/core/base/reflectx"
	"cogentcore.org/core/base/slicesx"
	"cogentcore.org/core/tensor/bitslice"
	"gonum.org/v1/gonum/mat"
)

// Bits is a tensor of bits backed by a bitslice.Slice for efficient storage
// of binary data
type Bits struct {
	Shp    Shape
	Values bitslice.Slice
	Meta   map[string]string
}

// NewBits returns a new n-dimensional tensor of bit values
// with the given sizes per dimension (shape), and optional dimension names.
func NewBits(sizes []int, names ...string) *Bits {
	tsr := &Bits{}
	tsr.SetShape(sizes, names...)
	tsr.Values = bitslice.Make(tsr.Len(), 0)
	return tsr
}

// NewBitsShape returns a new n-dimensional tensor of bit values
// using given shape.
func NewBitsShape(shape *Shape) *Bits {
	tsr := &Bits{}
	tsr.Shp.CopyShape(shape)
	tsr.Values = bitslice.Make(tsr.Len(), 0)
	return tsr
}

func Float64ToBool(val float64) bool {
	bv := true
	if val == 0 {
		bv = false
	}
	return bv
}

func BoolToFloat64(bv bool) float64 {
	if bv {
		return 1
	} else {
		return 0
	}
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

// Shape returns a pointer to the shape that fully parametrizes the tensor shape
func (tsr *Bits) Shape() *Shape { return &tsr.Shp }

// Len returns the number of elements in the tensor (product of shape dimensions).
func (tsr *Bits) Len() int { return tsr.Shp.Len() }

// NumDims returns the total number of dimensions.
func (tsr *Bits) NumDims() int { return tsr.Shp.NumDims() }

// DimSize returns size of given dimension
func (tsr *Bits) DimSize(dim int) int { return tsr.Shp.DimSize(dim) }

// RowCellSize returns the size of the outer-most Row shape dimension,
// and the size of all the remaining inner dimensions (the "cell" size).
// Used for Tensors that are columns in a data table.
func (tsr *Bits) RowCellSize() (rows, cells int) {
	return tsr.Shp.RowCellSize()
}

// Value returns value at given tensor index
func (tsr *Bits) Value(i []int) bool { j := int(tsr.Shp.Offset(i)); return tsr.Values.Index(j) }

// Value1D returns value at given tensor 1D (flat) index
func (tsr *Bits) Value1D(i int) bool { return tsr.Values.Index(i) }

func (tsr *Bits) Set(i []int, val bool) { j := int(tsr.Shp.Offset(i)); tsr.Values.Set(j, val) }
func (tsr *Bits) Set1D(i int, val bool) { tsr.Values.Set(i, val) }

// SetShape sets the shape params, resizing backing storage appropriately
func (tsr *Bits) SetShape(sizes []int, names ...string) {
	tsr.Shp.SetShape(sizes, names...)
	nln := tsr.Len()
	tsr.Values.SetLen(nln)
}

// SetNumRows sets the number of rows (outer-most dimension) in a RowMajor organized tensor.
func (tsr *Bits) SetNumRows(rows int) {
	rows = max(1, rows) // must be > 0
	_, cells := tsr.Shp.RowCellSize()
	nln := rows * cells
	tsr.Shp.Sizes[0] = rows
	tsr.Values.SetLen(nln)
}

// SubSpace is not possible with Bits
func (tsr *Bits) SubSpace(offs []int) Tensor {
	return nil
}

func (tsr *Bits) Float(i []int) float64 {
	j := tsr.Shp.Offset(i)
	return BoolToFloat64(tsr.Values.Index(j))
}

func (tsr *Bits) SetFloat(i []int, val float64) {
	j := tsr.Shp.Offset(i)
	tsr.Values.Set(j, Float64ToBool(val))
}

func (tsr *Bits) StringValue(i []int) string {
	j := tsr.Shp.Offset(i)
	return reflectx.ToString(tsr.Values.Index(j))
}

func (tsr *Bits) SetString(i []int, val string) {
	if bv, err := reflectx.ToBool(val); err == nil {
		j := tsr.Shp.Offset(i)
		tsr.Values.Set(j, bv)
	}
}

func (tsr *Bits) Float1D(off int) float64 {
	return BoolToFloat64(tsr.Values.Index(off))
}
func (tsr *Bits) SetFloat1D(off int, val float64) {
	tsr.Values.Set(off, Float64ToBool(val))
}

func (tsr *Bits) FloatRowCell(row, cell int) float64 {
	_, sz := tsr.RowCellSize()
	return BoolToFloat64(tsr.Values.Index(row*sz + cell))
}
func (tsr *Bits) SetFloatRowCell(row, cell int, val float64) {
	_, sz := tsr.RowCellSize()
	tsr.Values.Set(row*sz+cell, Float64ToBool(val))
}

func (tsr *Bits) Floats(flt *[]float64) {
	sz := tsr.Len()
	*flt = slicesx.SetLength(*flt, sz)
	for j := 0; j < sz; j++ {
		(*flt)[j] = BoolToFloat64(tsr.Values.Index(j))
	}
}

// SetFloats sets tensor values from a []float64 slice (copies values).
func (tsr *Bits) SetFloats(vals []float64) {
	sz := min(tsr.Len(), len(vals))
	for j := 0; j < sz; j++ {
		tsr.Values.Set(j, Float64ToBool(vals[j]))
	}
}

func (tsr *Bits) String1D(off int) string {
	return reflectx.ToString(tsr.Values.Index(off))
}

func (tsr *Bits) SetString1D(off int, val string) {
	if bv, err := reflectx.ToBool(val); err == nil {
		tsr.Values.Set(off, bv)
	}
}

func (tsr *Bits) StringRowCell(row, cell int) string {
	_, sz := tsr.RowCellSize()
	return reflectx.ToString(tsr.Values.Index(row*sz + cell))
}

func (tsr *Bits) SetStringRowCell(row, cell int, val string) {
	if bv, err := reflectx.ToBool(val); err == nil {
		_, sz := tsr.RowCellSize()
		tsr.Values.Set(row*sz+cell, bv)
	}
}

// Label satisfies the core.Labeler interface for a summary description of the tensor
func (tsr *Bits) Label() string {
	return fmt.Sprintf("tensor.Bits: %s", tsr.Shp.String())
}

// SetMetaData sets a key=value meta data (stored as a map[string]string).
// For TensorGrid display: top-zero=+/-, odd-row=+/-, image=+/-,
// min, max set fixed min / max values, background=color
func (tsr *Bits) SetMetaData(key, val string) {
	if tsr.Meta == nil {
		tsr.Meta = make(map[string]string)
	}
	tsr.Meta[key] = val
}

// MetaData retrieves value of given key, bool = false if not set
func (tsr *Bits) MetaData(key string) (string, bool) {
	if tsr.Meta == nil {
		return "", false
	}
	val, ok := tsr.Meta[key]
	return val, ok
}

// MetaDataMap returns the underlying map used for meta data
func (tsr *Bits) MetaDataMap() map[string]string {
	return tsr.Meta
}

// CopyMetaData copies meta data from given source tensor
func (tsr *Bits) CopyMetaData(frm Tensor) {
	fmap := frm.MetaDataMap()
	if len(fmap) == 0 {
		return
	}
	if tsr.Meta == nil {
		tsr.Meta = make(map[string]string)
	}
	for k, v := range fmap {
		tsr.Meta[k] = v
	}
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
		tsr.Values.Set(j, false)
	}
}

// Clone clones this tensor, creating a duplicate copy of itself with its
// own separate memory representation of all the values, and returns
// that as a Tensor (which can be converted into the known type as needed).
func (tsr *Bits) Clone() Tensor {
	csr := NewBitsShape(&tsr.Shp)
	csr.Values = tsr.Values.Clone()
	return csr
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
		tsr.Values.Set(i, Float64ToBool(frm.Float1D(i)))
	}
}

// CopyShapeFrom copies just the shape from given source tensor
// calling SetShape with the shape params from source (see for more docs).
func (tsr *Bits) CopyShapeFrom(frm Tensor) {
	tsr.SetShape(frm.Shape().Sizes, frm.Shape().Names...)
}

// CopyCellsFrom copies given range of values from other tensor into this tensor,
// using flat 1D indexes: to = starting index in this Tensor to start copying into,
// start = starting index on from Tensor to start copying from, and n = number of
// values to copy.  Uses an optimized implementation if the other tensor is
// of the same type, and otherwise it goes through appropriate standard type.
func (tsr *Bits) CopyCellsFrom(frm Tensor, to, start, n int) {
	if fsm, ok := frm.(*Bits); ok {
		for i := 0; i < n; i++ {
			tsr.Values.Set(to+i, fsm.Values.Index(start+i))
		}
		return
	}
	for i := 0; i < n; i++ {
		tsr.Values.Set(to+i, Float64ToBool(frm.Float1D(start+i)))
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

// String satisfies the fmt.Stringer interface for string of tensor data
func (tsr *Bits) String() string {
	str := tsr.Label()
	sz := tsr.Len()
	if sz > 1000 {
		return str
	}
	var b strings.Builder
	b.WriteString(str)
	b.WriteString("\n")
	oddRow := true
	rows, cols, _, _ := Projection2DShape(&tsr.Shp, oddRow)
	for r := 0; r < rows; r++ {
		rc, _ := Projection2DCoords(&tsr.Shp, oddRow, r, 0)
		b.WriteString(fmt.Sprintf("%v: ", rc))
		for c := 0; c < cols; c++ {
			vl := Projection2DValue(tsr, oddRow, r, c)
			b.WriteString(fmt.Sprintf("%g ", vl))
		}
		b.WriteString("\n")
	}
	return b.String()
}
