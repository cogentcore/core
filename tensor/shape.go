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
// Per C / Go / Python conventions, indexes are Row-Major, ordered from
// outer to inner left-to-right, so the inner-most is right-most.
type Shape struct {

	// size per dimension
	Sizes []int

	// offsets for each dimension
	Strides []int `display:"-"`

	// names of each dimension
	Names []string `display:"-"`
}

// NewShape returns a new shape with given sizes and optional dimension names.
// RowMajor ordering is used by default.
func NewShape(sizes []int, names ...string) *Shape {
	sh := &Shape{}
	sh.SetShape(sizes, names...)
	return sh
}

// SetShape sets the shape size and optional names
// RowMajor ordering is used by default.
func (sh *Shape) SetShape(sizes []int, names ...string) {
	sh.Sizes = slices.Clone(sizes)
	sh.Strides = RowMajorStrides(sizes)
	sh.Names = make([]string, len(sh.Sizes))
	if len(names) == len(sizes) {
		copy(sh.Names, names)
	}
}

// CopyShape copies the shape parameters from another Shape struct.
// copies the data so it is not accidentally subject to updates.
func (sh *Shape) CopyShape(cp *Shape) {
	sh.Sizes = slices.Clone(cp.Sizes)
	sh.Strides = slices.Clone(cp.Strides)
	sh.Names = slices.Clone(cp.Names)
}

// Len returns the total length of elements in the tensor
// (i.e., the product of the shape sizes)
func (sh *Shape) Len() int {
	if len(sh.Sizes) == 0 {
		return 0
	}
	o := int(1)
	for _, v := range sh.Sizes {
		o *= v
	}
	return int(o)
}

// NumDims returns the total number of dimensions.
func (sh *Shape) NumDims() int { return len(sh.Sizes) }

// DimSize returns the size of given dimension.
func (sh *Shape) DimSize(i int) int { return sh.Sizes[i] }

// DimName returns the name of given dimension.
func (sh *Shape) DimName(i int) string { return sh.Names[i] }

// DimByName returns the index of the given dimension name.
// returns -1 if not found.
func (sh *Shape) DimByName(name string) int {
	for i, nm := range sh.Names {
		if nm == name {
			return i
		}
	}
	return -1
}

// DimSizeByName returns the size of given dimension, specified by name.
// will crash if name not found.
func (sh *Shape) DimSizeByName(name string) int {
	return sh.DimSize(sh.DimByName(name))
}

// IndexIsValid() returns true if given index is valid (within ranges for all dimensions)
func (sh *Shape) IndexIsValid(idx []int) bool {
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
	if !EqualInts(sh.Sizes, oth.Sizes) {
		return false
	}
	if !EqualInts(sh.Strides, oth.Strides) {
		return false
	}
	return true
}

// RowCellSize returns the size of the outer-most Row shape dimension,
// and the size of all the remaining inner dimensions (the "cell" size).
// Used for Tensors that are columns in a data table.
func (sh *Shape) RowCellSize() (rows, cells int) {
	rows = sh.Sizes[0]
	if len(sh.Sizes) == 1 {
		cells = 1
	} else {
		cells = sh.Len() / rows
	}
	return
}

// Offset returns the "flat" 1D array index into an element at the given n-dimensional index.
// No checking is done on the length or size of the index values relative to the shape of the tensor.
func (sh *Shape) Offset(index []int) int {
	var offset int
	for i, v := range index {
		offset += v * sh.Strides[i]
	}
	return offset
}

// Index returns the n-dimensional index from a "flat" 1D array index.
func (sh *Shape) Index(offset int) []int {
	nd := len(sh.Sizes)
	index := make([]int, nd)
	rem := offset
	for i := nd - 1; i >= 0; i-- {
		s := sh.Sizes[i]
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
		nm := sh.Names[i]
		if nm != "" {
			str += nm + ": "
		}
		str += fmt.Sprintf("%d", sh.Sizes[i])
		if i < len(sh.Sizes)-1 {
			str += ", "
		}
	}
	str += "]"
	return str
}

// RowMajorStrides returns strides for sizes where the first dimension is outer-most
// and subsequent dimensions are progressively inner.
func RowMajorStrides(sizes []int) []int {
	rem := int(1)
	for _, v := range sizes {
		rem *= v
	}

	if rem == 0 {
		strides := make([]int, len(sizes))
		rem := int(1)
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
func ColMajorStrides(sizes []int) []int {
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

// EqualInts compares two int slices and returns true if they are equal
func EqualInts(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// AddShapes returns a new shape by adding two shapes one after the other.
func AddShapes(shape1, shape2 *Shape) *Shape {
	sh1 := shape1.Sizes
	sh2 := shape2.Sizes
	nsh := make([]int, len(sh1)+len(sh2))
	copy(nsh, sh1)
	copy(nsh[len(sh1):], sh2)
	nms := make([]string, len(sh1)+len(sh2))
	copy(nms, shape1.Names)
	copy(nms[len(sh1):], shape2.Names)
	return NewShape(nsh, nms...)
}
