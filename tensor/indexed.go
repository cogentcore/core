// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package tensor

import (
	"reflect"
	"slices"

	"cogentcore.org/core/base/metadata"
	"cogentcore.org/core/base/reflectx"
)

// Indexed provides an arbitrarily indexed view onto another "source" [Tensor]
// with each index value providing a full n-dimensional index into the source.
// The shape of this view is determined by the shape of the [Indexed.Indexes]
// tensor up to the final innermost dimension, which holds the index values.
// Thus the innermost dimension size of the indexes is equal to the number
// of dimensions in the source tensor. Given the essential role of the
// indexes in this view, it is not usable without the indexes.
// This view is not memory-contiguous and does not support the [RowMajor]
// interface or efficient access to inner-dimensional subspaces.
// To produce a new concrete [Values] that has raw data actually
// organized according to the indexed order (i.e., the copy function
// of numpy), call [Indexed.AsValues].
type Indexed struct { //types:add

	// Tensor source that we are an indexed view onto.
	Tensor Tensor

	// Indexes is the list of indexes into the source tensor,
	// with the innermost dimension providing the index values
	// (size = number of dimensions in the source tensor), and
	// the remaining outer dimensions determine the shape
	// of this [Indexed] tensor view.
	Indexes *Int
}

// NewIndexed returns a new [Indexed] view of given tensor,
// with tensor of indexes into the source tensor.
func NewIndexed(tsr Tensor, idx *Int) *Indexed {
	ix := &Indexed{Tensor: tsr}
	ix.Indexes = idx
	return ix
}

// AsIndexed returns the tensor as a [Indexed] view, if it is one.
// Otherwise, it returns nil; there is no usable "null" Indexed view.
func AsIndexed(tsr Tensor) *Indexed {
	if ix, ok := tsr.(*Indexed); ok {
		return ix
	}
	return nil
}

// SetTensor sets as indexes into given tensor with sequential initial indexes.
func (ix *Indexed) SetTensor(tsr Tensor) {
	ix.Tensor = tsr
}

// SourceIndexes returns the actual indexes into underlying source tensor
// based on given list of indexes into the [Indexed.Indexes] tensor,
// _excluding_ the final innermost dimension.
func (ix *Indexed) SourceIndexes(i ...int) []int {
	idx := slices.Clone(i)
	idx = append(idx, 0) // first index
	oned := ix.Indexes.Shape().IndexTo1D(idx...)
	nd := ix.Tensor.NumDims()
	return ix.Indexes.Values[oned : oned+nd]
}

// SourceIndexesFrom1D returns the full indexes into source tensor based on the
// given 1d index, which is based on the outer dimensions, excluding the
// final innermost dimension.
func (ix *Indexed) SourceIndexesFrom1D(oned int) []int {
	nd := ix.Tensor.NumDims()
	oned *= nd
	return ix.Indexes.Values[oned : oned+nd]
}

func (ix *Indexed) Label() string            { return label(ix.Metadata().Name(), ix.Shape()) }
func (ix *Indexed) String() string           { return sprint(ix, 0) }
func (ix *Indexed) Metadata() *metadata.Data { return ix.Tensor.Metadata() }
func (ix *Indexed) IsString() bool           { return ix.Tensor.IsString() }
func (ix *Indexed) DataType() reflect.Kind   { return ix.Tensor.DataType() }
func (ix *Indexed) Shape() *Shape            { return NewShape(ix.ShapeSizes()...) }
func (ix *Indexed) Len() int                 { return ix.Shape().Len() }
func (ix *Indexed) NumDims() int             { return ix.Indexes.NumDims() - 1 }
func (ix *Indexed) DimSize(dim int) int      { return ix.Indexes.DimSize(dim) }

func (ix *Indexed) ShapeSizes() []int {
	si := slices.Clone(ix.Indexes.ShapeSizes())
	return si[:len(si)-1] // exclude last dim
}

// AsValues returns a copy of this tensor as raw [Values].
// This "renders" the Indexed view into a fully contiguous
// and optimized memory representation of that view, which will be faster
// to access for further processing, and enables all the additional
// functionality provided by the [Values] interface.
func (ix *Indexed) AsValues() Values {
	dt := ix.Tensor.DataType()
	vt := NewOfType(dt, ix.ShapeSizes()...)
	n := ix.Len()
	switch {
	case ix.Tensor.IsString():
		for i := range n {
			vt.SetString1D(ix.String1D(i), i)
		}
	case reflectx.KindIsFloat(dt):
		for i := range n {
			vt.SetFloat1D(ix.Float1D(i), i)
		}
	default:
		for i := range n {
			vt.SetInt1D(ix.Int1D(i), i)
		}
	}
	return vt
}

/////////////////////  Floats

// Float returns the value of given index as a float64.
// The indexes are indirected through the [Indexed.Indexes].
func (ix *Indexed) Float(i ...int) float64 {
	return ix.Tensor.Float(ix.SourceIndexes(i...)...)
}

// SetFloat sets the value of given index as a float64
// The indexes are indirected through the [Indexed.Indexes].
func (ix *Indexed) SetFloat(val float64, i ...int) {
	ix.Tensor.SetFloat(val, ix.SourceIndexes(i...)...)
}

// Float1D is somewhat expensive if indexes are set, because it needs to convert
// the flat index back into a full n-dimensional index and then use that api.
func (ix *Indexed) Float1D(i int) float64 {
	return ix.Tensor.Float(ix.SourceIndexesFrom1D(i)...)
}

// SetFloat1D is somewhat expensive if indexes are set, because it needs to convert
// the flat index back into a full n-dimensional index and then use that api.
func (ix *Indexed) SetFloat1D(val float64, i int) {
	ix.Tensor.SetFloat(val, ix.SourceIndexesFrom1D(i)...)
}

/////////////////////  Strings

// StringValue returns the value of given index as a string.
// The indexes are indirected through the [Indexed.Indexes].
func (ix *Indexed) StringValue(i ...int) string {
	return ix.Tensor.StringValue(ix.SourceIndexes(i...)...)
}

// SetString sets the value of given index as a string
// The indexes are indirected through the [Indexed.Indexes].
func (ix *Indexed) SetString(val string, i ...int) {
	ix.Tensor.SetString(val, ix.SourceIndexes(i...)...)
}

// String1D is somewhat expensive if indexes are set, because it needs to convert
// the flat index back into a full n-dimensional index and then use that api.
func (ix *Indexed) String1D(i int) string {
	return ix.Tensor.StringValue(ix.SourceIndexesFrom1D(i)...)
}

// SetString1D is somewhat expensive if indexes are set, because it needs to convert
// the flat index back into a full n-dimensional index and then use that api.
func (ix *Indexed) SetString1D(val string, i int) {
	ix.Tensor.SetString(val, ix.SourceIndexesFrom1D(i)...)
}

/////////////////////  Ints

// Int returns the value of given index as an int.
// The indexes are indirected through the [Indexed.Indexes].
func (ix *Indexed) Int(i ...int) int {
	return ix.Tensor.Int(ix.SourceIndexes(i...)...)
}

// SetInt sets the value of given index as an int
// The indexes are indirected through the [Indexed.Indexes].
func (ix *Indexed) SetInt(val int, i ...int) {
	ix.Tensor.SetInt(val, ix.SourceIndexes(i...)...)
}

// Int1D is somewhat expensive if indexes are set, because it needs to convert
// the flat index back into a full n-dimensional index and then use that api.
func (ix *Indexed) Int1D(i int) int {
	return ix.Tensor.Int(ix.SourceIndexesFrom1D(i)...)
}

// SetInt1D is somewhat expensive if indexes are set, because it needs to convert
// the flat index back into a full n-dimensional index and then use that api.
func (ix *Indexed) SetInt1D(val int, i int) {
	ix.Tensor.SetInt(val, ix.SourceIndexesFrom1D(i)...)
}

// check for interface impl
var _ Tensor = (*Indexed)(nil)
