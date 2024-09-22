// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package tensor

import (
	"fmt"
	"slices"
)

// Shape manages a tensor's shape information, including strides and dimension names
// and can compute the flat index into an underlying 1D data storage array based on an
// n-dimensional index (and vice-versa).
// Per Go / C / Python conventions, indexes are Row-Major, ordered from
// outer to inner left-to-right, so the inner-most is right-most.
type Shape struct {

	// size per dimension.
	Sizes []int

	// offsets for each dimension.
	Strides []int `display:"-"`
}

// NewShape returns a new shape with given sizes.
// RowMajor ordering is used by default.
func NewShape(sizes ...int) *Shape {
	sh := &Shape{}
	sh.SetShapeSizes(sizes...)
	return sh
}

// SetShapeSizes sets the shape sizes from list of ints.
// RowMajor ordering is used by default.
func (sh *Shape) SetShapeSizes(sizes ...int) {
	sh.Sizes = slices.Clone(sizes)
	sh.Strides = RowMajorStrides(sizes...)
}

// SetShapeSizesFromTensor sets the shape sizes from given tensor.
// RowMajor ordering is used by default.
func (sh *Shape) SetShapeSizesFromTensor(sizes Tensor) {
	sh.SetShapeSizes(AsIntSlice(sizes)...)
}

// SizesAsTensor returns shape sizes as an Int Tensor.
func (sh *Shape) SizesAsTensor() *Int {
	return NewIntFromValues(sh.Sizes...)
}

// CopyFrom copies the shape parameters from another Shape struct.
// copies the data so it is not accidentally subject to updates.
func (sh *Shape) CopyFrom(cp *Shape) {
	sh.Sizes = slices.Clone(cp.Sizes)
	sh.Strides = slices.Clone(cp.Strides)
}

// Len returns the total length of elements in the tensor
// (i.e., the product of the shape sizes).
func (sh *Shape) Len() int {
	if len(sh.Sizes) == 0 {
		return 0
	}
	ln := 1
	for _, v := range sh.Sizes {
		ln *= v
	}
	return ln
}

// NumDims returns the total number of dimensions.
func (sh *Shape) NumDims() int { return len(sh.Sizes) }

// DimSize returns the size of given dimension.
func (sh *Shape) DimSize(i int) int {
	// if sh.Sizes == nil {
	// 	return 0
	// }
	return sh.Sizes[i]
}

// IndexIsValid() returns true if given index is valid (within ranges for all dimensions)
func (sh *Shape) IndexIsValid(idx ...int) bool {
	if len(idx) != sh.NumDims() {
		return false
	}
	for i, v := range sh.Sizes {
		if idx[i] < 0 || idx[i] >= v {
			return false
		}
	}
	return true
}

// IsEqual returns true if this shape is same as other (does not compare names)
func (sh *Shape) IsEqual(oth *Shape) bool {
	if slices.Compare(sh.Sizes, oth.Sizes) != 0 {
		return false
	}
	if slices.Compare(sh.Strides, oth.Strides) != 0 {
		return false
	}
	return true
}

// RowCellSize returns the size of the outermost Row shape dimension,
// and the size of all the remaining inner dimensions (the "cell" size).
// Used for Tensors that are columns in a data table.
func (sh *Shape) RowCellSize() (rows, cells int) {
	rows = sh.Sizes[0]
	if len(sh.Sizes) == 1 {
		cells = 1
	} else if rows > 0 {
		cells = sh.Len() / rows
	} else {
		ln := 1
		for _, v := range sh.Sizes[1:] {
			ln *= v
		}
		cells = ln
	}
	return
}

// IndexTo1D returns the flat 1D index from given n-dimensional indicies.
// No checking is done on the length or size of the index values relative
// to the shape of the tensor.
func (sh *Shape) IndexTo1D(index ...int) int {
	var oned int
	for i, v := range index {
		oned += v * sh.Strides[i]
	}
	return oned
}

// IndexFrom1D returns the n-dimensional index from a "flat" 1D array index.
func (sh *Shape) IndexFrom1D(oned int) []int {
	nd := len(sh.Sizes)
	index := make([]int, nd)
	rem := oned
	for i := nd - 1; i >= 0; i-- {
		s := sh.Sizes[i]
		if s == 0 {
			return index
		}
		iv := rem % s
		rem /= s
		index[i] = iv
	}
	return index
}

// String satisfies the fmt.Stringer interface
func (sh *Shape) String() string {
	str := "["
	for i := range sh.Sizes {
		str += fmt.Sprintf("%d", sh.Sizes[i])
		if i < len(sh.Sizes)-1 {
			str += ", "
		}
	}
	str += "]"
	return str
}

// RowMajorStrides returns strides for sizes where the first dimension is outermost
// and subsequent dimensions are progressively inner.
func RowMajorStrides(sizes ...int) []int {
	if len(sizes) == 0 {
		return nil
	}
	sizes[0] = max(1, sizes[0]) // critical for strides to not be nil due to rows = 0
	rem := int(1)
	for _, v := range sizes {
		rem *= v
	}

	if rem == 0 {
		strides := make([]int, len(sizes))
		for i := range strides {
			strides[i] = rem
		}
		return strides
	}

	strides := make([]int, len(sizes))
	for i, v := range sizes {
		rem /= v
		strides[i] = rem
	}
	return strides
}

// ColMajorStrides returns strides for sizes where the first dimension is inner-most
// and subsequent dimensions are progressively outer
func ColMajorStrides(sizes ...int) []int {
	total := int(1)
	for _, v := range sizes {
		if v == 0 {
			strides := make([]int, len(sizes))
			for i := range strides {
				strides[i] = total
			}
			return strides
		}
	}

	strides := make([]int, len(sizes))
	for i, v := range sizes {
		strides[i] = total
		total *= v
	}
	return strides
}

// AddShapes returns a new shape by adding two shapes one after the other.
func AddShapes(shape1, shape2 *Shape) *Shape {
	sh1 := shape1.Sizes
	sh2 := shape2.Sizes
	nsh := make([]int, len(sh1)+len(sh2))
	copy(nsh, sh1)
	copy(nsh[len(sh1):], sh2)
	sh := NewShape(nsh...)
	return sh
}
