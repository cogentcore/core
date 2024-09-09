// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package tensor

import (
	"fmt"
	"log"
	"reflect"
	"unsafe"

	"cogentcore.org/core/base/reflectx"
	"cogentcore.org/core/base/slicesx"
)

// Base is an n-dim array of float64s.
type Base[T any] struct {
	Shp    Shape
	Values []T
	Meta   map[string]string
}

// Shape returns a pointer to the shape that fully parametrizes the tensor shape
func (tsr *Base[T]) Shape() *Shape { return &tsr.Shp }

// Len returns the number of elements in the tensor (product of shape dimensions).
func (tsr *Base[T]) Len() int { return tsr.Shp.Len() }

// NumDims returns the total number of dimensions.
func (tsr *Base[T]) NumDims() int { return tsr.Shp.NumDims() }

// DimSize returns size of given dimension
func (tsr *Base[T]) DimSize(dim int) int { return tsr.Shp.DimSize(dim) }

// RowCellSize returns the size of the outer-most Row shape dimension,
// and the size of all the remaining inner dimensions (the "cell" size).
// Used for Tensors that are columns in a data table.
func (tsr *Base[T]) RowCellSize() (rows, cells int) {
	return tsr.Shp.RowCellSize()
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

func (tsr *Base[T]) Value(i []int) T    { j := tsr.Shp.Offset(i); return tsr.Values[j] }
func (tsr *Base[T]) Value1D(i int) T    { return tsr.Values[i] }
func (tsr *Base[T]) Set(i []int, val T) { j := tsr.Shp.Offset(i); tsr.Values[j] = val }
func (tsr *Base[T]) Set1D(i int, val T) { tsr.Values[i] = val }

// SetShape sets the shape params, resizing backing storage appropriately
func (tsr *Base[T]) SetShape(sizes []int, names ...string) {
	tsr.Shp.SetShape(sizes, names...)
	nln := tsr.Len()
	tsr.Values = slicesx.SetLength(tsr.Values, nln)
}

// SetNumRows sets the number of rows (outer-most dimension) in a RowMajor organized tensor.
func (tsr *Base[T]) SetNumRows(rows int) {
	rows = max(1, rows) // must be > 0
	_, cells := tsr.Shp.RowCellSize()
	nln := rows * cells
	tsr.Shp.Sizes[0] = rows
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
	stsr.SetShape(tsr.Shp.Sizes[od:], tsr.Shp.Names[od:]...)
	sti := make([]int, nd)
	copy(sti, offs)
	stoff := tsr.Shp.Offset(sti)
	sln := stsr.Len()
	stsr.Values = tsr.Values[stoff : stoff+sln]
	return stsr
}

func (tsr *Base[T]) StringValue(i []int) string {
	j := tsr.Shp.Offset(i)
	return reflectx.ToString(tsr.Values[j])
}
func (tsr *Base[T]) String1D(off int) string { return reflectx.ToString(tsr.Values[off]) }

func (tsr *Base[T]) StringRowCell(row, cell int) string {
	_, sz := tsr.Shp.RowCellSize()
	return reflectx.ToString(tsr.Values[row*sz+cell])
}

// Label satisfies the core.Labeler interface for a summary description of the tensor
func (tsr *Base[T]) Label() string {
	return fmt.Sprintf("Tensor: %s", tsr.Shp.String())
}

// Dims is the gonum/mat.Matrix interface method for returning the dimensionality of the
// 2D Matrix.  Assumes Row-major ordering and logs an error if NumDims < 2.
func (tsr *Base[T]) Dims() (r, c int) {
	nd := tsr.NumDims()
	if nd < 2 {
		log.Println("tensor Dims gonum Matrix call made on Tensor with dims < 2")
		return 0, 0
	}
	return tsr.Shp.DimSize(nd - 2), tsr.Shp.DimSize(nd - 1)
}

// Symmetric is the gonum/mat.Matrix interface method for returning the dimensionality of a symmetric
// 2D Matrix.
func (tsr *Base[T]) Symmetric() (r int) {
	nd := tsr.NumDims()
	if nd < 2 {
		log.Println("tensor Symmetric gonum Matrix call made on Tensor with dims < 2")
		return 0
	}
	if tsr.Shp.DimSize(nd-2) != tsr.Shp.DimSize(nd-1) {
		log.Println("tensor Symmetric gonum Matrix call made on Tensor that is not symmetric")
		return 0
	}
	return tsr.Shp.DimSize(nd - 1)
}

// SymmetricDim returns the number of rows/columns in the matrix.
func (tsr *Base[T]) SymmetricDim() int {
	nd := tsr.NumDims()
	if nd < 2 {
		log.Println("tensor Symmetric gonum Matrix call made on Tensor with dims < 2")
		return 0
	}
	if tsr.Shp.DimSize(nd-2) != tsr.Shp.DimSize(nd-1) {
		log.Println("tensor Symmetric gonum Matrix call made on Tensor that is not symmetric")
		return 0
	}
	return tsr.Shp.DimSize(nd - 1)
}

// SetMetaData sets a key=value meta data (stored as a map[string]string).
// For TensorGrid display: top-zero=+/-, odd-row=+/-, image=+/-,
// min, max set fixed min / max values, background=color
func (tsr *Base[T]) SetMetaData(key, val string) {
	if tsr.Meta == nil {
		tsr.Meta = make(map[string]string)
	}
	tsr.Meta[key] = val
}

// MetaData retrieves value of given key, bool = false if not set
func (tsr *Base[T]) MetaData(key string) (string, bool) {
	if tsr.Meta == nil {
		return "", false
	}
	val, ok := tsr.Meta[key]
	return val, ok
}

// MetaDataMap returns the underlying map used for meta data
func (tsr *Base[T]) MetaDataMap() map[string]string {
	return tsr.Meta
}

// CopyMetaData copies meta data from given source tensor
func (tsr *Base[T]) CopyMetaData(frm Tensor) {
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
