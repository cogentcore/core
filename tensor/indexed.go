// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package tensor

import (
	"cmp"
	"errors"
	"math"
	"math/rand"
	"slices"
	"sort"
	"strings"
)

// Indexed is an indexed wrapper around a tensor.Tensor that provides a
// specific view onto the Tensor defined by the set of indexes, which
// apply to the outermost row dimension (with default row-major indexing).
// This is the universal representation of a homogenous data type in the
// [tensor] package framework, from scalar to vector, matrix, and beyond,
// because it can efficiently represent any kind of element with sufficient
// flexibility to enable a full range of computations to be elegantly expressed.
// For example, sorting and filtering a tensor only requires
// updating the indexes while doing nothing to the Tensor itself.
// To produce a new [Tensor] that has its raw data actually organized according
// to the indexed order, call the [NewTensor] method.
// Use the [Set]FloatRowCell methods wherever possible, for the most efficient
// and natural indirection through the indexes.  The 1D methods on underlying
// tensor data do not indirect through the indexes and must be called directly
// on the [Tensor].
type Indexed struct { //types:add
	// Tensor that we are an indexed view onto.
	Tensor Tensor

	// Indexes are the indexes into Tensor rows.
	// Only set if order is different from default sequential order.
	// Use the Index() method for nil-aware logic.
	Indexes []int
}

// NewIndexed returns a new Indexed based on given tensor.
// If a list of indexes is passed, then our indexes are initialized
// as a copy of those.  This is used e.g. from a Indexed Table column.
// Otherwise it is initialized with default sequential indexes.
func NewIndexed(tsr Tensor, idxs ...[]int) *Indexed {
	ix := &Indexed{}
	if len(idxs) == 1 { // indexes were passed
		ix.Tensor = tsr
		ix.Indexes = slices.Clone(idxs[0])
	} else {
		ix.SetTensor(tsr)
	}
	return ix
}

// NewFloatScalar is a convenience method to quickly get an Indexed
// representation of a single float64 scalar value, for use in math routines etc.
func NewFloatScalar(val float64) *Indexed {
	return &Indexed{Tensor: NewNumberFromSlice([]float64{val})}
}

// SetTensor sets as indexes into given tensor with sequential initial indexes
func (ix *Indexed) SetTensor(tsr Tensor) {
	ix.Tensor = tsr
	ix.Sequential()
}

// SetShapeFrom sets our shape from given source, calling
// [Tensor.SetShape] with the shape params from source,
// and copying the indexes if present.
func (ix *Indexed) SetShapeFrom(src *Indexed) {
	ix.Tensor.SetShapeFrom(src.Tensor)
	if src.Indexes == nil {
		ix.Indexes = nil
	} else {
		ix.Indexes = slices.Clone(src.Indexes)
	}
}

// Index returns the actual index into underlying tensor row based on given
// index value.  If Indexes == nil, index is passed through.
func (ix *Indexed) Index(idx int) int {
	if ix.Indexes == nil {
		return idx
	}
	return ix.Indexes[idx]
}

// RowCellSize returns the size of the outermost Row shape dimension
// (via [Indexed.Rows] method), and the size of all the remaining
// inner dimensions (the "cell" size).
func (ix *Indexed) RowCellSize() (rows, cells int) {
	_, cells = ix.Tensor.RowCellSize()
	rows = ix.Rows()
	return
}

// RowCellIndex returns the direct Values index into underlying tensor
// based on given overall row * cell index.
func (ix *Indexed) RowCellIndex(idx int) (i1d, ri, ci int) {
	_, cells := ix.Tensor.RowCellSize()
	ri = idx / cells
	ci = idx % cells
	i1d = ix.Index(ri)*cells + ci
	return
}

// Rows returns the effective number of rows in this Indexed view,
// which is the length of the index list or number of outer
// rows dimension of tensor if no indexes.
func (ix *Indexed) Rows() int {
	if ix.Indexes == nil {
		return ix.Tensor.DimSize(0)
	}
	return len(ix.Indexes)
}

// Len returns the total number of elements in the tensor,
// taking into account the Indexes via [Rows],
// as Rows() * cell size.
func (ix *Indexed) Len() int {
	rows := ix.Rows()
	_, cells := ix.Tensor.RowCellSize()
	return cells * rows
}

// DeleteInvalid deletes all invalid indexes from the list.
// Call this if rows (could) have been deleted from tensor.
func (ix *Indexed) DeleteInvalid() {
	if ix.Tensor == nil || ix.Tensor.DimSize(0) <= 0 || ix.Indexes == nil {
		ix.Indexes = nil
		return
	}
	ni := ix.Rows()
	for i := ni - 1; i >= 0; i-- {
		if ix.Indexes[i] >= ix.Tensor.DimSize(0) {
			ix.Indexes = append(ix.Indexes[:i], ix.Indexes[i+1:]...)
		}
	}
}

// Sequential sets Indexes to nil, resulting in sequential row-wise access into tensor.
func (ix *Indexed) Sequential() { //types:add
	ix.Indexes = nil
}

// IndexesNeeded is called prior to an operation that needs actual indexes,
// e.g., Sort, Filter.  If Indexes == nil, they are set to all rows, otherwise
// current indexes are left as is. Use Sequential, then IndexesNeeded to ensure
// all rows are represented.
func (ix *Indexed) IndexesNeeded() { //types:add
	if ix.Tensor == nil || ix.Tensor.DimSize(0) <= 0 {
		ix.Indexes = nil
		return
	}
	if ix.Indexes != nil {
		return
	}
	ix.Indexes = make([]int, ix.Tensor.DimSize(0))
	for i := range ix.Indexes {
		ix.Indexes[i] = i
	}
}

// ExcludeMissing1D deletes indexes for a 1D tensor (only) where
// the values are missing, as indicated by NaN.
func (ix *Indexed) ExcludeMissing1D() { //types:add
	if ix.Tensor == nil || ix.Tensor.DimSize(0) <= 0 {
		ix.Indexes = nil
		return
	}
	if ix.Tensor.NumDims() > 1 {
		return
	}
	ix.IndexesNeeded()
	ni := ix.Rows()
	for i := ni - 1; i >= 0; i-- {
		if math.IsNaN(ix.Tensor.Float1D(ix.Indexes[i])) {
			ix.Indexes = append(ix.Indexes[:i], ix.Indexes[i+1:]...)
		}
	}
}

// Permuted sets indexes to a permuted order.  If indexes already exist
// then existing list of indexes is permuted, otherwise a new set of
// permuted indexes are generated
func (ix *Indexed) Permuted() {
	if ix.Tensor == nil || ix.Tensor.DimSize(0) <= 0 {
		ix.Indexes = nil
		return
	}
	if ix.Indexes == nil {
		ix.Indexes = rand.Perm(ix.Tensor.DimSize(0))
	} else {
		rand.Shuffle(len(ix.Indexes), func(i, j int) {
			ix.Indexes[i], ix.Indexes[j] = ix.Indexes[j], ix.Indexes[i]
		})
	}
}

// AddIndex adds a new index to the list
func (ix *Indexed) AddIndex(idx int) {
	ix.Indexes = append(ix.Indexes, idx)
}

const (
	// Ascending specifies an ascending sort direction for tensor Sort routines
	Ascending = true

	// Descending specifies a descending sort direction for tensor Sort routines
	Descending = false
)

// SortFunc sorts the indexes into 1D Tensor using given compare function.
// Returns an error if called on a higher-dimensional tensor.
// The compare function operates directly on row numbers into the Tensor
// as these row numbers have already been projected through the indexes.
// cmp(a, b) should return a negative number when a < b, a positive
// number when a > b and zero when a == b.
func (ix *Indexed) SortFunc(cmp func(tsr Tensor, i, j int) int) error {
	if ix.Tensor.NumDims() > 1 {
		return errors.New("tensor Sorting is only for 1D tensors")
	}
	ix.IndexesNeeded()
	slices.SortFunc(ix.Indexes, func(a, b int) int {
		return cmp(ix.Tensor, ix.Indexes[a], ix.Indexes[b])
	})
	return nil
}

// SortIndexes sorts the indexes into our Tensor directly in
// numerical order, producing the native ordering, while preserving
// any filtering that might have occurred.
func (ix *Indexed) SortIndexes() {
	if ix.Indexes == nil {
		return
	}
	sort.Ints(ix.Indexes)
}

// Sort compare function for string values.
func CompareStrings(a, b string, ascending bool) int {
	cmp := strings.Compare(a, b)
	if !ascending {
		cmp = -cmp
	}
	return cmp
}

func CompareNumbers(a, b float64, ascending bool) int {
	cmp := cmp.Compare(a, b)
	if !ascending {
		cmp = -cmp
	}
	return cmp
}

// Sort does default alpha or numeric sort of 1D tensor based on data type.
// Returns an error if called on a higher-dimensional tensor.
func (ix *Indexed) Sort(ascending bool) error {
	if ix.Tensor.IsString() {
		ix.SortFunc(func(tsr Tensor, i, j int) int {
			return CompareStrings(tsr.String1D(i), tsr.String1D(j), ascending)
		})
	} else {
		ix.SortFunc(func(tsr Tensor, i, j int) int {
			return CompareNumbers(tsr.Float1D(i), tsr.Float1D(j), ascending)
		})
	}
	return nil
}

// SortStableFunc stably sorts the indexes of 1D Tensor using given compare function.
// The compare function operates directly on row numbers into the Tensor
// as these row numbers have already been projected through the indexes.
// cmp(a, b) should return a negative number when a < b, a positive
// number when a > b and zero when a == b.
// It is *essential* that it always returns 0 when the two are equal
// for the stable function to actually work.
func (ix *Indexed) SortStableFunc(cmp func(tsr Tensor, i, j int) int) error {
	if ix.Tensor.NumDims() > 1 {
		return errors.New("tensor Sorting is only for 1D tensors")
	}
	ix.IndexesNeeded()
	slices.SortStableFunc(ix.Indexes, func(a, b int) int {
		return cmp(ix.Tensor, ix.Indexes[a], ix.Indexes[b])
	})
	return nil
}

// SortStable does default alpha or numeric stable sort
// of 1D tensor based on data type.
// Returns an error if called on a higher-dimensional tensor.
func (ix *Indexed) SortStable(ascending bool) error {
	if ix.Tensor.IsString() {
		ix.SortStableFunc(func(tsr Tensor, i, j int) int {
			return CompareStrings(tsr.String1D(i), tsr.String1D(j), ascending)
		})
	} else {
		ix.SortStableFunc(func(tsr Tensor, i, j int) int {
			return CompareNumbers(tsr.Float1D(i), tsr.Float1D(j), ascending)
		})
	}
	return nil
}

// FilterFunc is a function used for filtering that returns
// true if Tensor row should be included in the current filtered
// view of the tensor, and false if it should be removed.
type FilterFunc func(tsr Tensor, row int) bool

// Filter filters the indexes into our Tensor using given Filter function.
// The Filter function operates directly on row numbers into the Tensor
// as these row numbers have already been projected through the indexes.
func (ix *Indexed) Filter(filterer func(tsr Tensor, row int) bool) {
	ix.IndexesNeeded()
	sz := len(ix.Indexes)
	for i := sz - 1; i >= 0; i-- { // always go in reverse for filtering
		if !filterer(ix.Tensor, ix.Indexes[i]) { // delete
			ix.Indexes = append(ix.Indexes[:i], ix.Indexes[i+1:]...)
		}
	}
}

// NewTensor returns a new tensor with column data organized according to
// the Indexes.  If Indexes are nil, a clone of the current tensor is returned
// but this function is only sensible if there is an indexed view in place.
func (ix *Indexed) NewTensor() Tensor {
	nt := ix.Tensor.Clone()
	if ix.Indexes == nil {
		return nt
	}
	rows := len(ix.Indexes)
	nt.SetNumRows(rows)
	_, cells := ix.Tensor.RowCellSize()
	str := ix.Tensor.IsString()
	for r := range rows {
		for c := range cells {
			if str {
				nt.SetStringRowCell(ix.StringRowCell(r, c), r, c)
			} else {
				nt.SetFloatRowCell(ix.FloatRowCell(r, c), r, c)
			}
		}
	}
	return nt
}

// Clone returns a copy of the current Indexed view with a cloned copy of
// the underlying Tensor and copy of the indexes.
func (ix *Indexed) Clone() *Indexed {
	nix := &Indexed{}
	nix.Tensor = ix.Tensor.Clone()
	nix.CopyIndexes(ix)
	return nix
}

// CloneIndexes returns a copy of the current Indexed view with new indexes,
// with a pointer to the same underlying Tensor as the source.
func (ix *Indexed) CloneIndexes() *Indexed {
	nix := &Indexed{}
	nix.Tensor = ix.Tensor
	nix.CopyIndexes(ix)
	return nix
}

// CopyIndexes copies indexes from other Indexed view.
func (ix *Indexed) CopyIndexes(oix *Indexed) {
	if oix.Indexes == nil {
		ix.Indexes = nil
	} else {
		ix.Indexes = slices.Clone(oix.Indexes)
	}
}

// AddRows adds n rows to end of underlying Tensor, and to the indexes in this view
func (ix *Indexed) AddRows(n int) { //types:add
	stidx := ix.Tensor.DimSize(0)
	ix.Tensor.SetNumRows(stidx + n)
	if ix.Indexes != nil {
		for i := stidx; i < stidx+n; i++ {
			ix.Indexes = append(ix.Indexes, i)
		}
	}
}

// InsertRows adds n rows to end of underlying Tensor, and to the indexes starting at
// given index in this view
func (ix *Indexed) InsertRows(at, n int) {
	stidx := ix.Tensor.DimSize(0)
	ix.Tensor.SetNumRows(stidx + n)
	if ix.Indexes != nil {
		nw := make([]int, n, n+len(ix.Indexes)-at)
		for i := 0; i < n; i++ {
			nw[i] = stidx + i
		}
		ix.Indexes = append(ix.Indexes[:at], append(nw, ix.Indexes[at:]...)...)
	}
}

// DeleteRows deletes n rows of indexes starting at given index in the list of indexes
func (ix *Indexed) DeleteRows(at, n int) {
	ix.IndexesNeeded()
	ix.Indexes = append(ix.Indexes[:at], ix.Indexes[at+n:]...)
}

// Swap switches the indexes for i and j
func (ix *Indexed) Swap(i, j int) {
	if ix.Indexes == nil {
		return
	}
	ix.Indexes[i], ix.Indexes[j] = ix.Indexes[j], ix.Indexes[i]
}

///////////////////////////////////////////////
// Indexed access

// Float returns the value of given index as a float64.
// The first index value is indirected through the indexes.
func (ix *Indexed) Float(i ...int) float64 {
	if ix.Indexes == nil {
		return ix.Tensor.Float(i...)
	}
	ic := slices.Clone(i)
	ic[0] = ix.Indexes[ic[0]]
	return ix.Tensor.Float(ic...)
}

// SetFloat sets the value of given index as a float64
// The first index value is indirected through the [Indexes].
func (ix *Indexed) SetFloat(val float64, i ...int) {
	if ix.Indexes == nil {
		ix.Tensor.SetFloat(val, i...)
		return
	}
	ic := slices.Clone(i)
	ic[0] = ix.Indexes[ic[0]]
	ix.Tensor.SetFloat(val, ic...)
}

// FloatRowCell returns the value at given row and cell,
// where row is outermost dim, and cell is 1D index into remaining inner dims.
// Row is indirected through the [Indexes].
// This is the preferred interface for all Indexed operations.
func (ix *Indexed) FloatRowCell(row, cell int) float64 {
	return ix.Tensor.FloatRowCell(ix.Index(row), cell)
}

// SetFloatRowCell sets the value at given row and cell,
// where row is outermost dim, and cell is 1D index into remaining inner dims.
// Row is indirected through the [Indexes].
// This is the preferred interface for all Indexed operations.
func (ix *Indexed) SetFloatRowCell(val float64, row, cell int) {
	ix.Tensor.SetFloatRowCell(val, ix.Index(row), cell)
}

// StringValue returns the value of given index as a string.
// The first index value is indirected through the indexes.
func (ix *Indexed) StringValue(i ...int) string {
	if ix.Indexes == nil {
		return ix.Tensor.StringValue(i...)
	}
	ic := slices.Clone(i)
	ic[0] = ix.Indexes[ic[0]]
	return ix.Tensor.StringValue(ic...)
}

// SetString sets the value of given index as a string
// The first index value is indirected through the [Indexes].
func (ix *Indexed) SetString(val string, i ...int) {
	if ix.Indexes == nil {
		ix.Tensor.SetString(val, i...)
	}
	ic := slices.Clone(i)
	ic[0] = ix.Indexes[ic[0]]
	ix.Tensor.SetString(val, ic...)
}

// StringRowCell returns the value at given row and cell,
// where row is outermost dim, and cell is 1D index into remaining inner dims.
// Row is indirected through the [Indexes].
// This is the preferred interface for all Indexed operations.
func (ix *Indexed) StringRowCell(row, cell int) string {
	return ix.Tensor.StringRowCell(ix.Index(row), cell)
}

// SetStringRowCell sets the value at given row and cell,
// where row is outermost dim, and cell is 1D index into remaining inner dims.
// Row is indirected through the [Indexes].
// This is the preferred interface for all Indexed operations.
func (ix *Indexed) SetStringRowCell(val string, row, cell int) {
	ix.Tensor.SetStringRowCell(val, ix.Index(row), cell)
}

// SubSpace returns a new tensor with innermost subspace at given
// offset(s) in outermost dimension(s) (len(offs) < NumDims).
// The new tensor points to the values of the this tensor (i.e., modifications
// will affect both), as its Values slice is a view onto the original (which
// is why only inner-most contiguous supsaces are supported).
// Use Clone() method to separate the two.
// Indexed version does indexed indirection of the outermost row dimension
// of the offsets.
func (ix *Indexed) SubSpace(offs ...int) Tensor {
	if len(offs) == 0 {
		return nil
	}
	offs[0] = ix.Index(offs[0])
	return ix.Tensor.SubSpace(offs...)
}
