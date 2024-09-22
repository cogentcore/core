// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package tensor

import (
	"reflect"
	"slices"

	"cogentcore.org/core/base/metadata"
	"cogentcore.org/core/base/reflectx"
	"cogentcore.org/core/base/slicesx"
)

// Indexed is a wrapper around another [Tensor] that provides a
// indexed view onto the Tensor provided by an [Int] tensor with
// index coordinates into the source tensor. The innermost dimension
// size of the indexes is equal to the number of dimensions in
// the source tensor, and the remaining outer dimensions provide the
// shape for the [Indexed] tensor view.
// To produce a new concrete [Values] that has raw data actually
// organized according to the indexed order (i.e., the copy function
// of numpy), call [Indexed.AsValues].
type Indexed struct { //types:add

	// Tensor that we are an indexed view onto.
	Tensor Tensor

	// Indexes is the list of indexes into the source tensor,
	// with the innermost dimension size equal to the number of
	// dimensions in the source tensor, and the remaining outer
	// dimensions providing the shape for the [Indexed] tensor.
	Indexes *Int
}

// NewIndexed returns a new [Indexed] view of given tensor,
// with optional tensor of indexes into the source tensor.
func NewIndexed(tsr Tensor, idx ...*Int) *Indexed {
	ix := &Indexed{Tensor: tsr}
	if len(idx) == 1 {
		ix.Indexes = idx[0]
	}
	ix.ValidIndexes()
	return ix
}

// AsIndexed returns the tensor as a [Indexed] view.
// If it already is one, then it is returned, otherwise it is wrapped.
func AsIndexed(tsr Tensor) *Indexed {
	if ix, ok := tsr.(*Indexed); ok {
		return ix
	}
	return NewIndexed(tsr)
}

// SetTensor sets as indexes into given tensor with sequential initial indexes.
func (ix *Indexed) SetTensor(tsr Tensor) {
	ix.Tensor = tsr
	ix.ValidIndexes()
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

// todo: do this:

// ValidIndexes ensures that [Indexed.Indexes] are valid,
// removing any out-of-range values and setting the view to nil (full sequential)
// for any dimension with no indexes (which is an invalid condition).
// Call this when any structural changes are made to underlying Tensor.
func (ix *Indexed) ValidIndexes() {
	nd := ix.Tensor.NumDims()
	ix.Indexes = slicesx.SetLength(ix.Indexes, nd)
	for d := range nd {
		ni := len(ix.Indexes[d])
		if ni == 0 { // invalid
			ix.Indexes[d] = nil // full
			continue
		}
		ds := ix.Tensor.DimSize(d)
		ix := ix.Indexes[d]
		for i := ni - 1; i >= 0; i-- {
			if ix[i] >= ds {
				ix = append(ix[:i], ix[i+1:]...)
			}
		}
		ix.Indexes[d] = ix
	}
}

// Label satisfies the core.Labeler interface for a summary description of the tensor.
func (ix *Indexed) Label() string {
	return label(ix.Metadata().Name(), ix.Shape())
}

// String satisfies the fmt.Stringer interface for string of tensor data.
func (ix *Indexed) String() string { return sprint(ix, 0) }

func (ix *Indexed) Metadata() *metadata.Data { return ix.Tensor.Metadata() }

func (ix *Indexed) IsString() bool { return ix.Tensor.IsString() }

func (ix *Indexed) DataType() reflect.Kind { return ix.Tensor.DataType() }

// For each dimension, we return the effective shape sizes using
// the current number of indexes per dimension.
func (ix *Indexed) ShapeSizes() []int {
	si := slices.Clone(ix.Indexes.ShapeSizes())
	return si[:len(si)-1] // exclude last dim
}

// Shape() returns a [Shape] representation of the tensor shape
// (dimension sizes). If we have Indexes, this is the effective
// shape using the current number of indexes per dimension.
func (ix *Indexed) Shape() *Shape {
	return NewShape(ix.ShapeSizes()...)
}

// Len returns the total number of elements in our view of the tensor.
func (ix *Indexed) Len() int { return ix.Shape().Len() }

// NumDims returns the total number of dimensions.
func (ix *Indexed) NumDims() int { return ix.Indexes.NumDims() - 1 }

// DimSize returns the effective view size of given dimension.
func (ix *Indexed) DimSize(dim int) int {
	return ix.Indexes.DimSize(dim)
}

// todo:

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

///////////////////////////////////////////////
// Indexed access

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
