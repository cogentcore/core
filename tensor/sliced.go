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

// Sliced provides a re-sliced view onto another "source" [Tensor],
// defined by a set of [Sliced.Indexes] for each dimension (must have
// at least 1 index per dimension to avoid a null view).
// Thus, each dimension can be transformed in arbitrary ways relative
// to the original tensor (filtered subsets, reversals, sorting, etc).
// This view is not memory-contiguous and does not support the [RowMajor]
// interface or efficient access to inner-dimensional subspaces.
// A new Sliced view defaults to a full transparent view of the source tensor.
// There is additional cost for every access operation associated with the
// indexed indirection, and access is always via the full n-dimensional indexes.
// See also [Rows] for a version that only indexes the outermost row dimension,
// which is much more efficient for this common use-case, and does support [RowMajor].
// To produce a new concrete [Values] that has raw data actually organized according
// to the indexed order (i.e., the copy function of numpy), call [Sliced.AsValues].
type Sliced struct { //types:add

	// Tensor source that we are an indexed view onto.
	Tensor Tensor

	// Indexes are the indexes for each dimension, with dimensions as the outer
	// slice (enforced to be the same length as the NumDims of the source Tensor),
	// and a list of dimension index values (within range of DimSize(d)).
	// A nil list of indexes for a dimension automatically provides a full,
	// sequential view of that dimension.
	Indexes [][]int
}

// NewSlicedIndexes returns a new [Sliced] view of given tensor,
// with optional list of indexes for each dimension (none / nil = sequential).
// Any dimensions without indexes default to nil = full sequential view.
func NewSlicedIndexes(tsr Tensor, idxs ...[]int) *Sliced {
	sl := &Sliced{Tensor: tsr, Indexes: idxs}
	sl.ValidIndexes()
	return sl
}

// NewSliced returns a new [Sliced] view of given tensor,
// with given slices for each dimension (none / nil = sequential).
// Any dimensions without indexes default to nil = full sequential view.
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
// If it already is one, then it is returned, otherwise it is wrapped
// in a new Sliced, with default full sequential ("transparent") view.
func AsSliced(tsr Tensor) *Sliced {
	if sl, ok := tsr.(*Sliced); ok {
		return sl
	}
	return NewSliced(tsr)
}

// SetTensor sets tensor as source for this view, and initializes a full
// transparent view onto source (calls [Sliced.Sequential]).
func (sl *Sliced) SetTensor(tsr Tensor) {
	sl.Tensor = tsr
	sl.Sequential()
}

// SourceIndex returns the actual index into source tensor dimension
// based on given index value.
func (sl *Sliced) SourceIndex(dim, idx int) int {
	ix := sl.Indexes[dim]
	if ix == nil {
		return idx
	}
	return ix[idx]
}

// SourceIndexes returns the actual n-dimensional indexes into source tensor
// based on given list of indexes based on the Sliced view shape.
func (sl *Sliced) SourceIndexes(i ...int) []int {
	ix := slices.Clone(i)
	for d, idx := range i {
		ix[d] = sl.SourceIndex(d, idx)
	}
	return ix
}

// SourceIndexesFrom1D returns the n-dimensional indexes into source tensor
// based on the given 1D index based on the Sliced view shape.
func (sl *Sliced) SourceIndexesFrom1D(oned int) []int {
	sh := sl.Shape()
	oix := sh.IndexFrom1D(oned) // full indexes in our coords
	return sl.SourceIndexes(oix...)
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

func (sl *Sliced) Label() string            { return label(sl.Metadata().Name(), sl.Shape()) }
func (sl *Sliced) String() string           { return sprint(sl, 0) }
func (sl *Sliced) Metadata() *metadata.Data { return sl.Tensor.Metadata() }
func (sl *Sliced) IsString() bool           { return sl.Tensor.IsString() }
func (sl *Sliced) DataType() reflect.Kind   { return sl.Tensor.DataType() }
func (sl *Sliced) Shape() *Shape            { return NewShape(sl.ShapeSizes()...) }
func (sl *Sliced) Len() int                 { return sl.Shape().Len() }
func (sl *Sliced) NumDims() int             { return sl.Tensor.NumDims() }

// For each dimension, we return the effective shape sizes using
// the current number of indexes per dimension.
func (sl *Sliced) ShapeSizes() []int {
	nd := sl.Tensor.NumDims()
	if nd == 0 {
		return sl.Tensor.ShapeSizes()
	}
	sh := slices.Clone(sl.Tensor.ShapeSizes())
	for d := range nd {
		if sl.Indexes[d] != nil {
			sh[d] = len(sl.Indexes[d])
		}
	}
	return sh
}

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
	vt := NewOfType(dt, sl.ShapeSizes()...)
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

/////////////////////  Floats

// Float returns the value of given index as a float64.
// The indexes are indirected through the [Sliced.Indexes].
func (sl *Sliced) Float(i ...int) float64 {
	return sl.Tensor.Float(sl.SourceIndexes(i...)...)
}

// SetFloat sets the value of given index as a float64
// The indexes are indirected through the [Sliced.Indexes].
func (sl *Sliced) SetFloat(val float64, i ...int) {
	sl.Tensor.SetFloat(val, sl.SourceIndexes(i...)...)
}

// Float1D is somewhat expensive if indexes are set, because it needs to convert
// the flat index back into a full n-dimensional index and then use that api.
func (sl *Sliced) Float1D(i int) float64 {
	return sl.Tensor.Float(sl.SourceIndexesFrom1D(i)...)
}

// SetFloat1D is somewhat expensive if indexes are set, because it needs to convert
// the flat index back into a full n-dimensional index and then use that api.
func (sl *Sliced) SetFloat1D(val float64, i int) {
	sl.Tensor.SetFloat(val, sl.SourceIndexesFrom1D(i)...)
}

/////////////////////  Strings

// StringValue returns the value of given index as a string.
// The indexes are indirected through the [Sliced.Indexes].
func (sl *Sliced) StringValue(i ...int) string {
	return sl.Tensor.StringValue(sl.SourceIndexes(i...)...)
}

// SetString sets the value of given index as a string
// The indexes are indirected through the [Sliced.Indexes].
func (sl *Sliced) SetString(val string, i ...int) {
	sl.Tensor.SetString(val, sl.SourceIndexes(i...)...)
}

// String1D is somewhat expensive if indexes are set, because it needs to convert
// the flat index back into a full n-dimensional index and then use that api.
func (sl *Sliced) String1D(i int) string {
	return sl.Tensor.StringValue(sl.SourceIndexesFrom1D(i)...)
}

// SetString1D is somewhat expensive if indexes are set, because it needs to convert
// the flat index back into a full n-dimensional index and then use that api.
func (sl *Sliced) SetString1D(val string, i int) {
	sl.Tensor.SetString(val, sl.SourceIndexesFrom1D(i)...)
}

/////////////////////  Ints

// Int returns the value of given index as an int.
// The indexes are indirected through the [Sliced.Indexes].
func (sl *Sliced) Int(i ...int) int {
	return sl.Tensor.Int(sl.SourceIndexes(i...)...)
}

// SetInt sets the value of given index as an int
// The indexes are indirected through the [Sliced.Indexes].
func (sl *Sliced) SetInt(val int, i ...int) {
	sl.Tensor.SetInt(val, sl.SourceIndexes(i...)...)
}

// Int1D is somewhat expensive if indexes are set, because it needs to convert
// the flat index back into a full n-dimensional index and then use that api.
func (sl *Sliced) Int1D(i int) int {
	return sl.Tensor.Int(sl.SourceIndexesFrom1D(i)...)
}

// SetInt1D is somewhat expensive if indexes are set, because it needs to convert
// the flat index back into a full n-dimensional index and then use that api.
func (sl *Sliced) SetInt1D(val int, i int) {
	sl.Tensor.SetInt(val, sl.SourceIndexesFrom1D(i)...)
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

// Filter filters the indexes using the given Filter function
// for setting the indexes for given dimension, and index into the
// source data.
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
