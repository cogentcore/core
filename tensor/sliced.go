// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package tensor

import (
	"math/rand"
	"reflect"
	"slices"
	"sort"

	"cogentcore.org/core/base/metadata"
	"cogentcore.org/core/base/reflectx"
	"cogentcore.org/core/base/slicesx"
)

// Sliced is a fully indexed wrapper around another [Tensor] that provides a
// re-sliced view onto the Tensor defined by the set of [SlicedIndexes],
// for each dimension (must have at least 1 per dimension).
// Thus, every dimension can be transformed in arbitrary ways relative
// to the original tensor. There is some additional cost for every
// access operation associated with the additional indexed indirection.
// See also [Indexed] for a version that only indexes the outermost row dimension,
// which is much more efficient for this common use-case.
// To produce a new [Tensor] that has its raw data actually organized according
// to the indexed order (i.e., the copy function of numpy), call [Sliced.NewTensor].
type Sliced struct { //types:add

	// Tensor that we are an indexed view onto.
	Tensor Tensor

	// Indexes are the indexes for each dimension, with dimensions as the outer
	// slice (enforced to be the same length as the NumDims of the source Tensor),
	// and a list of dimension index values (within range of DimSize(d)).
	// A nil list of indexes automatically provides a full, sequential view of that
	// dimension.
	Indexes [][]int
}

// NewSlicedIndexes returns a new [Sliced] view of given tensor,
// with optional list of indexes for each dimension (none / nil = sequential).
func NewSlicedIndexes(tsr Tensor, idxs ...[]int) *Sliced {
	sl := &Sliced{Tensor: tsr, Indexes: idxs}
	sl.ValidIndexes()
	return sl
}

// NewSliced returns a new [Sliced] view of given tensor,
// with given slices for each dimension (none / nil = sequential).
func NewSliced(tsr Tensor, sls ...Slice) *Sliced {
	ns := len(sls)
	if ns == 0 {
		return NewSlicedIndexes(tsr)
	}
	ns = min(ns, tsr.NumDims())
	ixs := make([][]int, ns)
	for d := range ns {
		sl := sls[d]
		ixs[d] = sl.IntSlice(tsr.DimSize(d))
	}
	return NewSlicedIndexes(tsr, ixs...)
}

// AsSliced returns the tensor as a [Sliced] view.
// If it already is one, then it is returned, otherwise it is wrapped.
func AsSliced(tsr Tensor) *Sliced {
	if sl, ok := tsr.(*Sliced); ok {
		return sl
	}
	return NewSliced(tsr)
}

// SetTensor sets as indexes into given tensor with sequential initial indexes.
func (sl *Sliced) SetTensor(tsr Tensor) {
	sl.Tensor = tsr
	sl.Sequential()
}

// SliceIndex returns the actual index into underlying tensor dimension
// based on given index value.
func (sl *Sliced) SliceIndex(dim, idx int) int {
	ix := sl.Indexes[dim]
	if ix == nil {
		return idx
	}
	return ix[idx]
}

// SliceIndexes returns the actual indexes into underlying tensor
// based on given list of indexes.
func (sl *Sliced) SliceIndexes(i ...int) []int {
	ix := slices.Clone(i)
	for d, idx := range i {
		ix[d] = sl.SliceIndex(d, idx)
	}
	return ix
}

// IndexFrom1D returns the full indexes into source tensor based on the
// given 1d index.
func (sl *Sliced) IndexFrom1D(oned int) []int {
	sh := sl.Shape()
	oix := sh.IndexFrom1D(oned) // full indexes in our coords
	return sl.SliceIndexes(oix...)
}

// ValidIndexes ensures that [Sliced.Indexes] are valid,
// removing any out-of-range values and setting the view to nil (full sequential)
// for any dimension with no indexes (which is an invalid condition).
// Call this when any structural changes are made to underlying Tensor.
func (sl *Sliced) ValidIndexes() {
	nd := sl.Tensor.NumDims()
	sl.Indexes = slicesx.SetLength(sl.Indexes, nd)
	for d := range nd {
		ni := len(sl.Indexes[d])
		if ni == 0 { // invalid
			sl.Indexes[d] = nil // full
			continue
		}
		ds := sl.Tensor.DimSize(d)
		ix := sl.Indexes[d]
		for i := ni - 1; i >= 0; i-- {
			if ix[i] >= ds {
				ix = append(ix[:i], ix[i+1:]...)
			}
		}
		sl.Indexes[d] = ix
	}
}

// Sequential sets all Indexes to nil, resulting in full sequential access into tensor.
func (sl *Sliced) Sequential() { //types:add
	nd := sl.Tensor.NumDims()
	sl.Indexes = slicesx.SetLength(sl.Indexes, nd)
	for d := range nd {
		sl.Indexes[d] = nil
	}
}

// IndexesNeeded is called prior to an operation that needs actual indexes,
// on given dimension.  If Indexes == nil, they are set to all items, otherwise
// current indexes are left as is. Use Sequential, then IndexesNeeded to ensure
// all dimension indexes are represented.
func (sl *Sliced) IndexesNeeded(d int) {
	ix := sl.Indexes[d]
	if ix != nil {
		return
	}
	ix = make([]int, sl.Tensor.DimSize(d))
	for i := range ix {
		ix[i] = i
	}
	sl.Indexes[d] = ix
}

// Label satisfies the core.Labeler interface for a summary description of the tensor.
func (sl *Sliced) Label() string {
	return label(sl.Metadata().Name(), sl.Shape())
}

// String satisfies the fmt.Stringer interface for string of tensor data.
func (sl *Sliced) String() string {
	return sprint(sl, 0)
}

// Metadata returns the metadata for this tensor, which can be used
// to encode plotting options, etc.
func (sl *Sliced) Metadata() *metadata.Data { return sl.Tensor.Metadata() }

func (sl *Sliced) IsString() bool {
	return sl.Tensor.IsString()
}

func (sl *Sliced) DataType() reflect.Kind {
	return sl.Tensor.DataType()
}

// For each dimension, we return the effective shape sizes using
// the current number of indexes per dimension.
func (sl *Sliced) ShapeInts() []int {
	nd := sl.Tensor.NumDims()
	if nd == 0 {
		return sl.Tensor.ShapeInts()
	}
	sh := slices.Clone(sl.Tensor.ShapeInts())
	for d := range nd {
		if sl.Indexes[d] != nil {
			sh[d] = len(sl.Indexes[d])
		}
	}
	return sh
}

func (sl *Sliced) ShapeSizes() Tensor {
	return NewIntFromSlice(sl.ShapeInts()...)
}

// Shape() returns a [Shape] representation of the tensor shape
// (dimension sizes). If we have Indexes, this is the effective
// shape using the current number of indexes per dimension.
func (sl *Sliced) Shape() *Shape {
	return NewShape(sl.ShapeInts()...)
}

// Len returns the total number of elements in our view of the tensor.
func (sl *Sliced) Len() int {
	return sl.Shape().Len()
}

// NumDims returns the total number of dimensions.
func (sl *Sliced) NumDims() int { return sl.Tensor.NumDims() }

// DimSize returns the effective view size of given dimension.
func (sl *Sliced) DimSize(dim int) int {
	if sl.Indexes[dim] != nil {
		return len(sl.Indexes[dim])
	}
	return sl.Tensor.DimSize(dim)
}

// AsValues returns a copy of this tensor as raw [Values].
// This "renders" the Sliced view into a fully contiguous
// and optimized memory representation of that view, which will be faster
// to access for further processing, and enables all the additional
// functionality provided by the [Values] interface.
func (sl *Sliced) AsValues() Values {
	dt := sl.Tensor.DataType()
	vt := NewOfType(dt, sl.ShapeInts()...)
	n := sl.Len()
	switch {
	case sl.Tensor.IsString():
		for i := range n {
			vt.SetString1D(sl.String1D(i), i)
		}
	case reflectx.KindIsFloat(dt):
		for i := range n {
			vt.SetFloat1D(sl.Float1D(i), i)
		}
	default:
		for i := range n {
			vt.SetInt1D(sl.Int1D(i), i)
		}
	}
	return vt
}

// // CloneIndexes returns a copy of the current Sliced view with new indexes,
// // with a pointer to the same underlying Tensor as the source.
// func (sl *Sliced) CloneIndexes() *Sliced {
// 	nix := &Sliced{}
// 	nix.Tensor = sl.Tensor
// 	nix.CopyIndexes(sl)
// 	return nix
// }
//
// // CopyIndexes copies indexes from other Sliced view.
// func (sl *Sliced) CopyIndexes(oix *Sliced) {
// 	if oix.Indexes == nil {
// 		sl.Indexes = nil
// 	} else {
// 		sl.Indexes = slices.Clone(oix.Indexes)
// 	}
// }

///////////////////////////////////////////////
// Sliced access

/////////////////////  Floats

// Float returns the value of given index as a float64.
// The indexes are indirected through the [Sliced.Indexes].
func (sl *Sliced) Float(i ...int) float64 {
	return sl.Tensor.Float(sl.SliceIndexes(i...)...)
}

// SetFloat sets the value of given index as a float64
// The indexes are indirected through the [Sliced.Indexes].
func (sl *Sliced) SetFloat(val float64, i ...int) {
	sl.Tensor.SetFloat(val, sl.SliceIndexes(i...)...)
}

// Float1D is somewhat expensive if indexes are set, because it needs to convert
// the flat index back into a full n-dimensional index and then use that api.
func (sl *Sliced) Float1D(i int) float64 {
	return sl.Tensor.Float(sl.IndexFrom1D(i)...)
}

// SetFloat1D is somewhat expensive if indexes are set, because it needs to convert
// the flat index back into a full n-dimensional index and then use that api.
func (sl *Sliced) SetFloat1D(val float64, i int) {
	sl.Tensor.SetFloat(val, sl.IndexFrom1D(i)...)
}

/////////////////////  Strings

// StringValue returns the value of given index as a string.
// The first index value is indirected through the indexes.
func (sl *Sliced) StringValue(i ...int) string {
	return sl.Tensor.StringValue(sl.SliceIndexes(i...)...)
}

// SetString sets the value of given index as a string
// The first index value is indirected through the [Sliced.Indexes].
func (sl *Sliced) SetString(val string, i ...int) {
	sl.Tensor.SetString(val, sl.SliceIndexes(i...)...)
}

// String1D is somewhat expensive if indexes are set, because it needs to convert
// the flat index back into a full n-dimensional index and then use that api.
func (sl *Sliced) String1D(i int) string {
	return sl.Tensor.StringValue(sl.IndexFrom1D(i)...)
}

// SetString1D is somewhat expensive if indexes are set, because it needs to convert
// the flat index back into a full n-dimensional index and then use that api.
func (sl *Sliced) SetString1D(val string, i int) {
	sl.Tensor.SetString(val, sl.IndexFrom1D(i)...)
}

/////////////////////  Ints

// Int returns the value of given index as an int.
// The first index value is indirected through the indexes.
func (sl *Sliced) Int(i ...int) int {
	return sl.Tensor.Int(sl.SliceIndexes(i...)...)
}

// SetInt sets the value of given index as an int
// The first index value is indirected through the [Sliced.Indexes].
func (sl *Sliced) SetInt(val int, i ...int) {
	sl.Tensor.SetInt(val, sl.SliceIndexes(i...)...)
}

// Int1D is somewhat expensive if indexes are set, because it needs to convert
// the flat index back into a full n-dimensional index and then use that api.
func (sl *Sliced) Int1D(i int) int {
	return sl.Tensor.Int(sl.IndexFrom1D(i)...)
}

// SetInt1D is somewhat expensive if indexes are set, because it needs to convert
// the flat index back into a full n-dimensional index and then use that api.
func (sl *Sliced) SetInt1D(val int, i int) {
	sl.Tensor.SetInt(val, sl.IndexFrom1D(i)...)
}

// Permuted sets indexes in given dimension to a permuted order.
// If indexes already exist then existing list of indexes is permuted,
// otherwise a new set of permuted indexes are generated
func (sl *Sliced) Permuted(dim int) {
	ix := sl.Indexes[dim]
	if ix == nil {
		ix = rand.Perm(sl.Tensor.DimSize(dim))
	} else {
		rand.Shuffle(len(ix), func(i, j int) {
			ix[i], ix[j] = ix[j], ix[i]
		})
	}
	sl.Indexes[dim] = ix
}

// SortFunc sorts the indexes along given dimension using given compare function.
// The compare function operates directly on indexes into the Tensor
// as these row numbers have already been projected through the indexes.
// cmp(a, b) should return a negative number when a < b, a positive
// number when a > b and zero when a == b.
func (sl *Sliced) SortFunc(dim int, cmp func(tsr Tensor, dim, i, j int) int) {
	sl.IndexesNeeded(dim)
	ix := sl.Indexes[dim]
	slices.SortFunc(ix, func(a, b int) int {
		return cmp(sl.Tensor, dim, a, b) // key point: these are already indirected through indexes!!
	})
	sl.Indexes[dim] = ix
}

// SortIndexes sorts the indexes along given dimension directly in
// numerical order, producing the native ordering, while preserving
// any filtering that might have occurred.
func (sl *Sliced) SortIndexes(dim int) {
	ix := sl.Indexes[dim]
	if ix == nil {
		return
	}
	sort.Ints(ix)
	sl.Indexes[dim] = ix
}

// SortStableFunc stably sorts along given dimension using given compare function.
// The compare function operates directly on row numbers into the Tensor
// as these row numbers have already been projected through the indexes.
// cmp(a, b) should return a negative number when a < b, a positive
// number when a > b and zero when a == b.
// It is *essential* that it always returns 0 when the two are equal
// for the stable function to actually work.
func (sl *Sliced) SortStableFunc(dim int, cmp func(tsr Tensor, dim, i, j int) int) {
	sl.IndexesNeeded(dim)
	ix := sl.Indexes[dim]
	slices.SortStableFunc(ix, func(a, b int) int {
		return cmp(sl.Tensor, dim, a, b) // key point: these are already indirected through indexes!!
	})
	sl.Indexes[dim] = ix
}

// Filter filters the indexes using given Filter function.
// The Filter function operates directly on row numbers into the Tensor
// as these row numbers have already been projected through the indexes.
func (sl *Sliced) Filter(dim int, filterer func(tsr Tensor, dim, idx int) bool) {
	sl.IndexesNeeded(dim)
	ix := sl.Indexes[dim]
	sz := len(ix)
	for i := sz - 1; i >= 0; i-- { // always go in reverse for filtering
		if !filterer(sl, dim, ix[i]) { // delete
			ix = append(ix[:i], ix[i+1:]...)
		}
	}
	sl.Indexes[dim] = ix
}

// check for interface impl
var _ Tensor = (*Sliced)(nil)
