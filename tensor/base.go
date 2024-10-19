// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package tensor

import (
	"reflect"
	"slices"
	"unsafe"

	"cogentcore.org/core/base/metadata"
	"cogentcore.org/core/base/reflectx"
	"cogentcore.org/core/base/slicesx"
)

// Base is the base Tensor implementation for given type.
type Base[T any] struct {
	shape  Shape
	Values []T
	Meta   metadata.Data
}

// Metadata returns the metadata for this tensor, which can be used
// to encode plotting options, etc.
func (tsr *Base[T]) Metadata() *metadata.Data { return &tsr.Meta }

func (tsr *Base[T]) Shape() *Shape { return &tsr.shape }

// ShapeSizes returns the sizes of each dimension as a slice of ints.
// This is the preferred access for Go code.
func (tsr *Base[T]) ShapeSizes() []int { return slices.Clone(tsr.shape.Sizes) }

// SetShapeSizes sets the dimension sizes of the tensor, and resizes
// backing storage appropriately, retaining all existing data that fits.
func (tsr *Base[T]) SetShapeSizes(sizes ...int) {
	tsr.shape.SetShapeSizes(sizes...)
	nln := tsr.shape.Header + tsr.shape.Len()
	tsr.Values = slicesx.SetLength(tsr.Values, nln)
}

// Len returns the number of elements in the tensor (product of shape dimensions).
func (tsr *Base[T]) Len() int { return tsr.shape.Len() }

// NumDims returns the total number of dimensions.
func (tsr *Base[T]) NumDims() int { return tsr.shape.NumDims() }

// DimSize returns size of given dimension.
func (tsr *Base[T]) DimSize(dim int) int { return tsr.shape.DimSize(dim) }

// DataType returns the type of the data elements in the tensor.
// Bool is returned for the Bool tensor type.
func (tsr *Base[T]) DataType() reflect.Kind {
	var v T
	return reflect.TypeOf(v).Kind()
}

func (tsr *Base[T]) Sizeof() int64 {
	var v T
	return int64(unsafe.Sizeof(v)) * int64(tsr.Len())
}

func (tsr *Base[T]) Bytes() []byte {
	return slicesx.ToBytes(tsr.Values)
}

func (tsr *Base[T]) Value(i ...int) T {
	return tsr.Values[tsr.shape.IndexTo1D(i...)]
}

func (tsr *Base[T]) Set(val T, i ...int) {
	tsr.Values[tsr.shape.IndexTo1D(i...)] = val
}

func (tsr *Base[T]) Value1D(i int) T { return tsr.Values[tsr.shape.Header+i] }

func (tsr *Base[T]) Set1D(val T, i int) { tsr.Values[tsr.shape.Header+i] = val }

// SetNumRows sets the number of rows (outermost dimension) in a RowMajor organized tensor.
// It is safe to set this to 0. For incrementally growing tensors (e.g., a log)
// it is best to first set the anticipated full size, which allocates the
// full amount of memory, and then set to 0 and grow incrementally.
func (tsr *Base[T]) SetNumRows(rows int) {
	_, cells := tsr.shape.RowCellSize()
	nln := tsr.shape.Header + rows*cells
	tsr.shape.Sizes[0] = rows
	tsr.Values = slicesx.SetLength(tsr.Values, nln)
}

// subSpaceImpl returns a new tensor with innermost subspace at given
// offset(s) in outermost dimension(s) (len(offs) < NumDims).
// The new tensor points to the values of the this tensor (i.e., modifications
// will affect both), as its Values slice is a view onto the original (which
// is why only inner-most contiguous supsaces are supported).
// Use AsValues() method to separate the two.
func (tsr *Base[T]) subSpaceImpl(offs ...int) *Base[T] {
	nd := tsr.NumDims()
	od := len(offs)
	if od > nd {
		return nil
	}
	var ssz []int
	if od == nd { // scalar subspace
		ssz = []int{1}
	} else {
		ssz = tsr.shape.Sizes[od:]
	}
	stsr := &Base[T]{}
	stsr.SetShapeSizes(ssz...)
	sti := make([]int, nd)
	copy(sti, offs)
	stoff := tsr.shape.IndexTo1D(sti...)
	sln := stsr.Len()
	stsr.Values = tsr.Values[stoff : stoff+sln]
	return stsr
}

/////////////////////  Strings

func (tsr *Base[T]) StringValue(i ...int) string {
	return reflectx.ToString(tsr.Values[tsr.shape.IndexTo1D(i...)])
}

func (tsr *Base[T]) String1D(i int) string {
	return reflectx.ToString(tsr.Values[tsr.shape.Header+i])
}

func (tsr *Base[T]) StringRowCell(row, cell int) string {
	_, sz := tsr.shape.RowCellSize()
	return reflectx.ToString(tsr.Values[tsr.shape.Header+row*sz+cell])
}

// Label satisfies the core.Labeler interface for a summary description of the tensor.
func (tsr *Base[T]) Label() string {
	return label(tsr.Meta.Name(), &tsr.shape)
}
