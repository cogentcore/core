// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package tensor

import (
	"fmt"
	"log"
	"reflect"
	"unsafe"

	"cogentcore.org/core/base/metadata"
	"cogentcore.org/core/base/reflectx"
	"cogentcore.org/core/base/slicesx"
)

// Base is an n-dim array of float64s.
type Base[T any] struct {
	shape  Shape
	Values []T
	Meta   metadata.Data
}

// Shape returns a pointer to the shape that fully parametrizes the tensor shape
func (tsr *Base[T]) Shape() *Shape { return &tsr.shape }

// Metadata returns the metadata for this tensor, which can be used
// to encode plotting options, etc.
func (tsr *Base[T]) Metadata() *metadata.Data { return &tsr.Meta }

// Len returns the number of elements in the tensor (product of shape dimensions).
func (tsr *Base[T]) Len() int { return tsr.shape.Len() }

// NumDims returns the total number of dimensions.
func (tsr *Base[T]) NumDims() int { return tsr.shape.NumDims() }

// DimSize returns size of given dimension
func (tsr *Base[T]) DimSize(dim int) int { return tsr.shape.DimSize(dim) }

// RowCellSize returns the size of the outer-most Row shape dimension,
// and the size of all the remaining inner dimensions (the "cell" size).
// Used for Tensors that are columns in a data table.
func (tsr *Base[T]) RowCellSize() (rows, cells int) {
	return tsr.shape.RowCellSize()
}

// DataType returns the type of the data elements in the tensor.
// Bool is returned for the Bits tensor type.
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

func (tsr *Base[T]) Value(i []int) T    { j := tsr.shape.Offset(i); return tsr.Values[j] }
func (tsr *Base[T]) Value1D(i int) T    { return tsr.Values[i] }
func (tsr *Base[T]) Set(i []int, val T) { j := tsr.shape.Offset(i); tsr.Values[j] = val }
func (tsr *Base[T]) Set1D(i int, val T) { tsr.Values[i] = val }

// SetShape sets the shape params, resizing backing storage appropriately
func (tsr *Base[T]) SetShape(sizes []int, names ...string) {
	tsr.shape.SetShape(sizes, names...)
	nln := tsr.Len()
	tsr.Values = slicesx.SetLength(tsr.Values, nln)
}

// SetNumRows sets the number of rows (outer-most dimension) in a RowMajor organized tensor.
func (tsr *Base[T]) SetNumRows(rows int) {
	rows = max(1, rows) // must be > 0
	_, cells := tsr.shape.RowCellSize()
	nln := rows * cells
	tsr.shape.Sizes[0] = rows
	tsr.Values = slicesx.SetLength(tsr.Values, nln)
}

// subSpaceImpl returns a new tensor with innermost subspace at given
// offset(s) in outermost dimension(s) (len(offs) < NumDims).
// The new tensor points to the values of the this tensor (i.e., modifications
// will affect both), as its Values slice is a view onto the original (which
// is why only inner-most contiguous supsaces are supported).
// Use Clone() method to separate the two.
func (tsr *Base[T]) subSpaceImpl(offs []int) *Base[T] {
	nd := tsr.NumDims()
	od := len(offs)
	if od >= nd {
		return nil
	}
	stsr := &Base[T]{}
	stsr.SetShape(tsr.shape.Sizes[od:], tsr.shape.Names[od:]...)
	sti := make([]int, nd)
	copy(sti, offs)
	stoff := tsr.shape.Offset(sti)
	sln := stsr.Len()
	stsr.Values = tsr.Values[stoff : stoff+sln]
	return stsr
}

func (tsr *Base[T]) StringValue(i []int) string {
	j := tsr.shape.Offset(i)
	return reflectx.ToString(tsr.Values[j])
}
func (tsr *Base[T]) String1D(off int) string { return reflectx.ToString(tsr.Values[off]) }

func (tsr *Base[T]) StringRowCell(row, cell int) string {
	_, sz := tsr.shape.RowCellSize()
	return reflectx.ToString(tsr.Values[row*sz+cell])
}

// Label satisfies the core.Labeler interface for a summary description of the tensor
func (tsr *Base[T]) Label() string {
	return fmt.Sprintf("Tensor: %s", tsr.shape.String())
}

// Dims is the gonum/mat.Matrix interface method for returning the dimensionality of the
// 2D Matrix.  Assumes Row-major ordering and logs an error if NumDims < 2.
func (tsr *Base[T]) Dims() (r, c int) {
	nd := tsr.NumDims()
	if nd < 2 {
		log.Println("tensor Dims gonum Matrix call made on Tensor with dims < 2")
		return 0, 0
	}
	return tsr.shape.DimSize(nd - 2), tsr.shape.DimSize(nd - 1)
}

// Symmetric is the gonum/mat.Matrix interface method for returning the dimensionality of a symmetric
// 2D Matrix.
func (tsr *Base[T]) Symmetric() (r int) {
	nd := tsr.NumDims()
	if nd < 2 {
		log.Println("tensor Symmetric gonum Matrix call made on Tensor with dims < 2")
		return 0
	}
	if tsr.shape.DimSize(nd-2) != tsr.shape.DimSize(nd-1) {
		log.Println("tensor Symmetric gonum Matrix call made on Tensor that is not symmetric")
		return 0
	}
	return tsr.shape.DimSize(nd - 1)
}

// SymmetricDim returns the number of rows/columns in the matrix.
func (tsr *Base[T]) SymmetricDim() int {
	nd := tsr.NumDims()
	if nd < 2 {
		log.Println("tensor Symmetric gonum Matrix call made on Tensor with dims < 2")
		return 0
	}
	if tsr.shape.DimSize(nd-2) != tsr.shape.DimSize(nd-1) {
		log.Println("tensor Symmetric gonum Matrix call made on Tensor that is not symmetric")
		return 0
	}
	return tsr.shape.DimSize(nd - 1)
}
