// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package tensor

import (
	"cmp"
	"math"
	"math/rand"
	"reflect"
	"slices"
	"sort"
	"strings"

	"cogentcore.org/core/base/metadata"
)

// Rows is a row-indexed wrapper view around a [Values] [Tensor] that allows
// arbitrary row-wise ordering and filtering according to the [Rows.Indexes].
// Sorting and filtering a tensor along this outermost row dimension only
// requires updating the indexes while leaving the underlying Tensor alone.
// Unlike the more general [Sliced] view, Rows maintains memory contiguity
// for the inner dimensions ("cells") within each row, and supports the [RowMajor]
// interface, with the [Set]FloatRow[Cell] methods providing efficient access.
// Use [Rows.AsValues] to obtain a concrete [Values] representation with the
// current row sorting.
type Rows struct { //types:add

	// Tensor source that we are an indexed view onto.
	// Note that this must be a concrete [Values] tensor, to enable efficient
	// [RowMajor] access and subspace functions.
	Tensor Values

	// Indexes are the indexes into Tensor rows, with nil = sequential.
	// Only set if order is different from default sequential order.
	// Use the [Rows.RowIndex] method for nil-aware logic.
	Indexes []int
}

// NewRows returns a new [Rows] view of given tensor,
// with optional list of indexes (none / nil = sequential).
func NewRows(tsr Values, idxs ...int) *Rows {
	rw := &Rows{Tensor: tsr, Indexes: slices.Clone(idxs)}
	return rw
}

// AsRows returns the tensor as a [Rows] view.
// If it already is one, then it is returned, otherwise
// a new Rows is created to wrap around the given tensor, which is
// enforced to be a [Values] tensor either because it already is one,
// or by calling [Tensor.AsValues] on it.
func AsRows(tsr Tensor) *Rows {
	if rw, ok := tsr.(*Rows); ok {
		return rw
	}
	return NewRows(tsr.AsValues())
}

// SetTensor sets as indexes into given [Values] tensor with sequential initial indexes.
func (rw *Rows) SetTensor(tsr Values) {
	rw.Tensor = tsr
	rw.Sequential()
}

func (rw *Rows) IsString() bool { return rw.Tensor.IsString() }

func (rw *Rows) DataType() reflect.Kind { return rw.Tensor.DataType() }

// RowIndex returns the actual index into underlying tensor row based on given
// index value.  If Indexes == nil, index is passed through.
func (rw *Rows) RowIndex(idx int) int {
	if rw.Indexes == nil {
		return idx
	}
	return rw.Indexes[idx]
}

// NumRows returns the effective number of rows in this Rows view,
// which is the length of the index list or number of outer
// rows dimension of tensor if no indexes (full sequential view).
func (rw *Rows) NumRows() int {
	if rw.Indexes == nil {
		return rw.Tensor.DimSize(0)
	}
	return len(rw.Indexes)
}

func (rw *Rows) String() string           { return Sprintf("", rw.Tensor, 0) }
func (rw *Rows) Label() string            { return rw.Tensor.Label() }
func (rw *Rows) Metadata() *metadata.Data { return rw.Tensor.Metadata() }
func (rw *Rows) NumDims() int             { return rw.Tensor.NumDims() }

// If we have Indexes, this is the effective shape sizes using
// the current number of indexes as the outermost row dimension size.
func (rw *Rows) ShapeSizes() []int {
	if rw.Indexes == nil || rw.Tensor.NumDims() == 0 {
		return rw.Tensor.ShapeSizes()
	}
	sh := slices.Clone(rw.Tensor.ShapeSizes())
	sh[0] = len(rw.Indexes)
	return sh
}

// Shape() returns a [Shape] representation of the tensor shape
// (dimension sizes). If we have Indexes, this is the effective
// shape using the current number of indexes as the outermost row dimension size.
func (rw *Rows) Shape() *Shape {
	if rw.Indexes == nil {
		return rw.Tensor.Shape()
	}
	return NewShape(rw.ShapeSizes()...)
}

// Len returns the total number of elements in the tensor,
// taking into account the Indexes via [Rows],
// as NumRows() * cell size.
func (rw *Rows) Len() int {
	rows := rw.NumRows()
	_, cells := rw.Tensor.Shape().RowCellSize()
	return cells * rows
}

// DimSize returns size of given dimension, returning NumRows()
// for first dimension.
func (rw *Rows) DimSize(dim int) int {
	if dim == 0 {
		return rw.NumRows()
	}
	return rw.Tensor.DimSize(dim)
}

// RowCellSize returns the size of the outermost Row shape dimension
// (via [Rows.NumRows] method), and the size of all the remaining
// inner dimensions (the "cell" size).
func (rw *Rows) RowCellSize() (rows, cells int) {
	_, cells = rw.Tensor.Shape().RowCellSize()
	rows = rw.NumRows()
	return
}

// ValidIndexes deletes all invalid indexes from the list.
// Call this if rows (could) have been deleted from tensor.
func (rw *Rows) ValidIndexes() {
	if rw.Tensor.DimSize(0) <= 0 || rw.Indexes == nil {
		rw.Indexes = nil
		return
	}
	ni := rw.NumRows()
	for i := ni - 1; i >= 0; i-- {
		if rw.Indexes[i] >= rw.Tensor.DimSize(0) {
			rw.Indexes = append(rw.Indexes[:i], rw.Indexes[i+1:]...)
		}
	}
}

// Sequential sets Indexes to nil, resulting in sequential row-wise access into tensor.
func (rw *Rows) Sequential() { //types:add
	rw.Indexes = nil
}

// IndexesNeeded is called prior to an operation that needs actual indexes,
// e.g., Sort, Filter.  If Indexes == nil, they are set to all rows, otherwise
// current indexes are left as is. Use Sequential, then IndexesNeeded to ensure
// all rows are represented.
func (rw *Rows) IndexesNeeded() {
	if rw.Tensor.DimSize(0) <= 0 {
		rw.Indexes = nil
		return
	}
	if rw.Indexes != nil {
		return
	}
	rw.Indexes = make([]int, rw.Tensor.DimSize(0))
	for i := range rw.Indexes {
		rw.Indexes[i] = i
	}
}

// ExcludeMissing deletes indexes where the values are missing, as indicated by NaN.
// Uses first cell of higher dimensional data.
func (rw *Rows) ExcludeMissing() { //types:add
	if rw.Tensor.DimSize(0) <= 0 {
		rw.Indexes = nil
		return
	}
	rw.IndexesNeeded()
	ni := rw.NumRows()
	for i := ni - 1; i >= 0; i-- {
		if math.IsNaN(rw.Tensor.FloatRow(rw.Indexes[i], 0)) {
			rw.Indexes = append(rw.Indexes[:i], rw.Indexes[i+1:]...)
		}
	}
}

// Permuted sets indexes to a permuted order.  If indexes already exist
// then existing list of indexes is permuted, otherwise a new set of
// permuted indexes are generated
func (rw *Rows) Permuted() {
	if rw.Tensor.DimSize(0) <= 0 {
		rw.Indexes = nil
		return
	}
	if rw.Indexes == nil {
		rw.Indexes = rand.Perm(rw.Tensor.DimSize(0))
	} else {
		rand.Shuffle(len(rw.Indexes), func(i, j int) {
			rw.Indexes[i], rw.Indexes[j] = rw.Indexes[j], rw.Indexes[i]
		})
	}
}

const (
	// Ascending specifies an ascending sort direction for tensor Sort routines
	Ascending = true

	// Descending specifies a descending sort direction for tensor Sort routines
	Descending = false

	// StableSort specifies using stable, original order-preserving sort, which is slower.
	StableSort = true

	// Unstable specifies using faster but unstable sorting.
	UnstableSort = false
)

// SortFunc sorts the row-wise indexes using given compare function.
// The compare function operates directly on row numbers into the Tensor
// as these row numbers have already been projected through the indexes.
// cmp(a, b) should return a negative number when a < b, a positive
// number when a > b and zero when a == b.
func (rw *Rows) SortFunc(cmp func(tsr Values, i, j int) int) {
	rw.IndexesNeeded()
	slices.SortFunc(rw.Indexes, func(a, b int) int {
		return cmp(rw.Tensor, a, b) // key point: these are already indirected through indexes!!
	})
}

// SortIndexes sorts the indexes into our Tensor directly in
// numerical order, producing the native ordering, while preserving
// any filtering that might have occurred.
func (rw *Rows) SortIndexes() {
	if rw.Indexes == nil {
		return
	}
	sort.Ints(rw.Indexes)
}

// CompareAscending is a sort compare function that reverses direction
// based on the ascending bool.
func CompareAscending[T cmp.Ordered](a, b T, ascending bool) int {
	if ascending {
		return cmp.Compare(a, b)
	}
	return cmp.Compare(b, a)
}

// Sort does default alpha or numeric sort of row-wise data.
// Uses first cell of higher dimensional data.
func (rw *Rows) Sort(ascending bool) {
	if rw.Tensor.IsString() {
		rw.SortFunc(func(tsr Values, i, j int) int {
			return CompareAscending(tsr.StringRow(i, 0), tsr.StringRow(j, 0), ascending)
		})
	} else {
		rw.SortFunc(func(tsr Values, i, j int) int {
			return CompareAscending(tsr.FloatRow(i, 0), tsr.FloatRow(j, 0), ascending)
		})
	}
}

// SortStableFunc stably sorts the row-wise indexes using given compare function.
// The compare function operates directly on row numbers into the Tensor
// as these row numbers have already been projected through the indexes.
// cmp(a, b) should return a negative number when a < b, a positive
// number when a > b and zero when a == b.
// It is *essential* that it always returns 0 when the two are equal
// for the stable function to actually work.
func (rw *Rows) SortStableFunc(cmp func(tsr Values, i, j int) int) {
	rw.IndexesNeeded()
	slices.SortStableFunc(rw.Indexes, func(a, b int) int {
		return cmp(rw.Tensor, a, b) // key point: these are already indirected through indexes!!
	})
}

// SortStable does stable default alpha or numeric sort.
// Uses first cell of higher dimensional data.
func (rw *Rows) SortStable(ascending bool) {
	if rw.Tensor.IsString() {
		rw.SortStableFunc(func(tsr Values, i, j int) int {
			return CompareAscending(tsr.StringRow(i, 0), tsr.StringRow(j, 0), ascending)
		})
	} else {
		rw.SortStableFunc(func(tsr Values, i, j int) int {
			return CompareAscending(tsr.FloatRow(i, 0), tsr.FloatRow(j, 0), ascending)
		})
	}
}

// FilterFunc is a function used for filtering that returns
// true if Tensor row should be included in the current filtered
// view of the tensor, and false if it should be removed.
type FilterFunc func(tsr Values, row int) bool

// Filter filters the indexes using given Filter function.
// The Filter function operates directly on row numbers into the Tensor
// as these row numbers have already been projected through the indexes.
func (rw *Rows) Filter(filterer func(tsr Values, row int) bool) {
	rw.IndexesNeeded()
	sz := len(rw.Indexes)
	for i := sz - 1; i >= 0; i-- { // always go in reverse for filtering
		if !filterer(rw.Tensor, rw.Indexes[i]) { // delete
			rw.Indexes = append(rw.Indexes[:i], rw.Indexes[i+1:]...)
		}
	}
}

// FilterOptions are options to a Filter function
// determining how the string filter value is used for matching.
type FilterOptions struct { //types:add

	// Exclude means to exclude matches,
	// with the default (false) being to include
	Exclude bool

	// Contains means the string only needs to contain the target string,
	// with the default (false) requiring a complete match to entire string.
	Contains bool

	// IgnoreCase means that differences in case are ignored in comparing strings,
	// with the default (false) using case.
	IgnoreCase bool
}

// FilterString filters the indexes using string values compared to given
// string. Includes rows with matching values unless the Exclude option is set.
// If Contains option is set, it only checks if row contains string;
// if IgnoreCase, ignores case, otherwise filtering is case sensitive.
// Uses first cell of higher dimensional data.
func (rw *Rows) FilterString(str string, opts FilterOptions) { //types:add
	lowstr := strings.ToLower(str)
	rw.Filter(func(tsr Values, row int) bool {
		val := tsr.StringRow(row, 0)
		has := false
		switch {
		case opts.Contains && opts.IgnoreCase:
			has = strings.Contains(strings.ToLower(val), lowstr)
		case opts.Contains:
			has = strings.Contains(val, str)
		case opts.IgnoreCase:
			has = strings.EqualFold(val, str)
		default:
			has = (val == str)
		}
		if opts.Exclude {
			return !has
		}
		return has
	})
}

// AsValues returns this tensor as raw [Values].
// If the row [Rows.Indexes] are nil, then the wrapped Values tensor
// is returned.  Otherwise, it "renders" the Rows view into a fully contiguous
// and optimized memory representation of that view, which will be faster
// to access for further processing, and enables all the additional
// functionality provided by the [Values] interface.
func (rw *Rows) AsValues() Values {
	if rw.Indexes == nil {
		return rw.Tensor
	}
	vt := NewOfType(rw.Tensor.DataType(), rw.ShapeSizes()...)
	rows := rw.NumRows()
	for r := range rows {
		vt.SetRowTensor(rw.RowTensor(r), r)
	}
	return vt
}

// CloneIndexes returns a copy of the current Rows view with new indexes,
// with a pointer to the same underlying Tensor as the source.
func (rw *Rows) CloneIndexes() *Rows {
	nix := &Rows{}
	nix.Tensor = rw.Tensor
	nix.CopyIndexes(rw)
	return nix
}

// CopyIndexes copies indexes from other Rows view.
func (rw *Rows) CopyIndexes(oix *Rows) {
	if oix.Indexes == nil {
		rw.Indexes = nil
	} else {
		rw.Indexes = slices.Clone(oix.Indexes)
	}
}

// AddRows adds n rows to end of underlying Tensor, and to the indexes in this view
func (rw *Rows) AddRows(n int) { //types:add
	stidx := rw.Tensor.DimSize(0)
	rw.Tensor.SetNumRows(stidx + n)
	if rw.Indexes != nil {
		for i := stidx; i < stidx+n; i++ {
			rw.Indexes = append(rw.Indexes, i)
		}
	}
}

// InsertRows adds n rows to end of underlying Tensor, and to the indexes starting at
// given index in this view
func (rw *Rows) InsertRows(at, n int) {
	stidx := rw.Tensor.DimSize(0)
	rw.IndexesNeeded()
	rw.Tensor.SetNumRows(stidx + n)
	nw := make([]int, n, n+len(rw.Indexes)-at)
	for i := 0; i < n; i++ {
		nw[i] = stidx + i
	}
	rw.Indexes = append(rw.Indexes[:at], append(nw, rw.Indexes[at:]...)...)
}

// DeleteRows deletes n rows of indexes starting at given index in the list of indexes
func (rw *Rows) DeleteRows(at, n int) {
	rw.IndexesNeeded()
	rw.Indexes = append(rw.Indexes[:at], rw.Indexes[at+n:]...)
}

// Swap switches the indexes for i and j
func (rw *Rows) Swap(i, j int) {
	if rw.Indexes == nil {
		return
	}
	rw.Indexes[i], rw.Indexes[j] = rw.Indexes[j], rw.Indexes[i]
}

///////////////////////////////////////////////
// Rows access

/////////////////////  Floats

// Float returns the value of given index as a float64.
// The first index value is indirected through the indexes.
func (rw *Rows) Float(i ...int) float64 {
	if rw.Indexes == nil {
		return rw.Tensor.Float(i...)
	}
	ic := slices.Clone(i)
	ic[0] = rw.Indexes[ic[0]]
	return rw.Tensor.Float(ic...)
}

// SetFloat sets the value of given index as a float64
// The first index value is indirected through the [Rows.Indexes].
func (rw *Rows) SetFloat(val float64, i ...int) {
	if rw.Indexes == nil {
		rw.Tensor.SetFloat(val, i...)
		return
	}
	ic := slices.Clone(i)
	ic[0] = rw.Indexes[ic[0]]
	rw.Tensor.SetFloat(val, ic...)
}

// FloatRow returns the value at given row and cell,
// where row is outermost dim, and cell is 1D index into remaining inner dims.
// Row is indirected through the [Rows.Indexes].
// This is the preferred interface for all Rows operations.
func (rw *Rows) FloatRow(row, cell int) float64 {
	return rw.Tensor.FloatRow(rw.RowIndex(row), cell)
}

// SetFloatRow sets the value at given row and cell,
// where row is outermost dim, and cell is 1D index into remaining inner dims.
// Row is indirected through the [Rows.Indexes].
// This is the preferred interface for all Rows operations.
func (rw *Rows) SetFloatRow(val float64, row, cell int) {
	rw.Tensor.SetFloatRow(val, rw.RowIndex(row), cell)
}

// Float1D is somewhat expensive if indexes are set, because it needs to convert
// the flat index back into a full n-dimensional index and then use that api.
func (rw *Rows) Float1D(i int) float64 {
	if rw.Indexes == nil {
		return rw.Tensor.Float1D(i)
	}
	return rw.Float(rw.Tensor.Shape().IndexFrom1D(i)...)
}

// SetFloat1D is somewhat expensive if indexes are set, because it needs to convert
// the flat index back into a full n-dimensional index and then use that api.
func (rw *Rows) SetFloat1D(val float64, i int) {
	if rw.Indexes == nil {
		rw.Tensor.SetFloat1D(val, i)
	}
	rw.SetFloat(val, rw.Tensor.Shape().IndexFrom1D(i)...)
}

/////////////////////  Strings

// StringValue returns the value of given index as a string.
// The first index value is indirected through the indexes.
func (rw *Rows) StringValue(i ...int) string {
	if rw.Indexes == nil {
		return rw.Tensor.StringValue(i...)
	}
	ic := slices.Clone(i)
	ic[0] = rw.Indexes[ic[0]]
	return rw.Tensor.StringValue(ic...)
}

// SetString sets the value of given index as a string
// The first index value is indirected through the [Rows.Indexes].
func (rw *Rows) SetString(val string, i ...int) {
	if rw.Indexes == nil {
		rw.Tensor.SetString(val, i...)
	}
	ic := slices.Clone(i)
	ic[0] = rw.Indexes[ic[0]]
	rw.Tensor.SetString(val, ic...)
}

// StringRow returns the value at given row and cell,
// where row is outermost dim, and cell is 1D index into remaining inner dims.
// Row is indirected through the [Rows.Indexes].
// This is the preferred interface for all Rows operations.
func (rw *Rows) StringRow(row, cell int) string {
	return rw.Tensor.StringRow(rw.RowIndex(row), cell)
}

// SetStringRow sets the value at given row and cell,
// where row is outermost dim, and cell is 1D index into remaining inner dims.
// Row is indirected through the [Rows.Indexes].
// This is the preferred interface for all Rows operations.
func (rw *Rows) SetStringRow(val string, row, cell int) {
	rw.Tensor.SetStringRow(val, rw.RowIndex(row), cell)
}

// String1D is somewhat expensive if indexes are set, because it needs to convert
// the flat index back into a full n-dimensional index and then use that api.
func (rw *Rows) String1D(i int) string {
	if rw.Indexes == nil {
		return rw.Tensor.String1D(i)
	}
	return rw.StringValue(rw.Tensor.Shape().IndexFrom1D(i)...)
}

// SetString1D is somewhat expensive if indexes are set, because it needs to convert
// the flat index back into a full n-dimensional index and then use that api.
func (rw *Rows) SetString1D(val string, i int) {
	if rw.Indexes == nil {
		rw.Tensor.SetString1D(val, i)
	}
	rw.SetString(val, rw.Tensor.Shape().IndexFrom1D(i)...)
}

/////////////////////  Ints

// Int returns the value of given index as an int.
// The first index value is indirected through the indexes.
func (rw *Rows) Int(i ...int) int {
	if rw.Indexes == nil {
		return rw.Tensor.Int(i...)
	}
	ic := slices.Clone(i)
	ic[0] = rw.Indexes[ic[0]]
	return rw.Tensor.Int(ic...)
}

// SetInt sets the value of given index as an int
// The first index value is indirected through the [Rows.Indexes].
func (rw *Rows) SetInt(val int, i ...int) {
	if rw.Indexes == nil {
		rw.Tensor.SetInt(val, i...)
		return
	}
	ic := slices.Clone(i)
	ic[0] = rw.Indexes[ic[0]]
	rw.Tensor.SetInt(val, ic...)
}

// IntRow returns the value at given row and cell,
// where row is outermost dim, and cell is 1D index into remaining inner dims.
// Row is indirected through the [Rows.Indexes].
// This is the preferred interface for all Rows operations.
func (rw *Rows) IntRow(row, cell int) int {
	return rw.Tensor.IntRow(rw.RowIndex(row), cell)
}

// SetIntRow sets the value at given row and cell,
// where row is outermost dim, and cell is 1D index into remaining inner dims.
// Row is indirected through the [Rows.Indexes].
// This is the preferred interface for all Rows operations.
func (rw *Rows) SetIntRow(val int, row, cell int) {
	rw.Tensor.SetIntRow(val, rw.RowIndex(row), cell)
}

// Int1D is somewhat expensive if indexes are set, because it needs to convert
// the flat index back into a full n-dimensional index and then use that api.
func (rw *Rows) Int1D(i int) int {
	if rw.Indexes == nil {
		return rw.Tensor.Int1D(i)
	}
	return rw.Int(rw.Tensor.Shape().IndexFrom1D(i)...)
}

// SetInt1D is somewhat expensive if indexes are set, because it needs to convert
// the flat index back into a full n-dimensional index and then use that api.
func (rw *Rows) SetInt1D(val int, i int) {
	if rw.Indexes == nil {
		rw.Tensor.SetInt1D(val, i)
	}
	rw.SetInt(val, rw.Tensor.Shape().IndexFrom1D(i)...)
}

/////////////////////  SubSpaces

// SubSpace returns a new tensor with innermost subspace at given
// offset(s) in outermost dimension(s) (len(offs) < NumDims).
// The new tensor points to the values of the this tensor (i.e., modifications
// will affect both), as its Values slice is a view onto the original (which
// is why only inner-most contiguous supsaces are supported).
// Use Clone() method to separate the two.
// Rows version does indexed indirection of the outermost row dimension
// of the offsets.
func (rw *Rows) SubSpace(offs ...int) Values {
	if len(offs) == 0 {
		return nil
	}
	offs[0] = rw.RowIndex(offs[0])
	return rw.Tensor.SubSpace(offs...)
}

// RowTensor is a convenience version of [Rows.SubSpace] to return the
// SubSpace for the outermost row dimension, indirected through the indexes.
func (rw *Rows) RowTensor(row int) Values {
	return rw.Tensor.RowTensor(rw.RowIndex(row))
}

// SetRowTensor sets the values of the SubSpace at given row to given values,
// with row indirected through the indexes.
func (rw *Rows) SetRowTensor(val Values, row int) {
	rw.Tensor.SetRowTensor(val, rw.RowIndex(row))
}

// AppendRow adds a row and sets values to given values.
func (rw *Rows) AppendRow(val Values) {
	nrow := rw.Tensor.DimSize(0)
	rw.AddRows(1)
	rw.Tensor.SetRowTensor(val, nrow)
}

// check for interface impl
var _ RowMajor = (*Rows)(nil)
