// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package tensor

import (
	"cmp"
	"log"
	"math"
	"math/rand"
	"reflect"
	"slices"
	"sort"
	"strings"

	"cogentcore.org/core/base/metadata"
	"gonum.org/v1/gonum/mat"
)

// Rows is an indexed wrapper around another [Tensor] that provides a
// specific view onto the Tensor defined by the set of [Rows.Indexes],
// which apply to the outermost row dimension (with default row-major indexing).
// Sorting and filtering a tensor only requires updating the indexes while
// leaving the underlying Tensor alone.
// To produce a new [Tensor] that has its raw data actually organized according
// to the indexed order (i.e., the copy function of numpy), call [Rows.NewTensor].
// Use the [Set]FloatRow[Cell] methods wherever possible, for the most efficient
// and natural indirection through the indexes.
type Rows struct { //types:add

	// Tensor that we are an indexed view onto.
	Tensor Tensor

	// Indexes are the indexes into Tensor rows, with nil = sequential.
	// Only set if order is different from default sequential order.
	// Use the Index() method for nil-aware logic.
	Indexes []int
}

// NewRows returns a new [Rows] view of given tensor,
// with optional list of indexes (none / nil = sequential).
func NewRows(tsr Tensor, idxs ...int) *Rows {
	ix := &Rows{Tensor: tsr, Indexes: slices.Clone(idxs)}
	return ix
}

// AsRows returns the tensor as an [Rows[] view.
// If it already is one, then it is returned, otherwise it is wrapped.
func AsRows(tsr Tensor) *Rows {
	if ix, ok := tsr.(*Rows); ok {
		return ix
	}
	return NewRows(tsr)
}

// SetTensor sets as indexes into given tensor with sequential initial indexes.
func (ix *Rows) SetTensor(tsr Tensor) {
	ix.Tensor = tsr
	ix.Sequential()
}

// RowIndex returns the actual index into underlying tensor row based on given
// index value.  If Indexes == nil, index is passed through.
func (ix *Rows) RowIndex(idx int) int {
	if ix.Indexes == nil {
		return idx
	}
	return ix.Indexes[idx]
}

// NumRows returns the effective number of rows in this Rows view,
// which is the length of the index list or number of outer
// rows dimension of tensor if no indexes (full sequential view).
func (ix *Rows) NumRows() int {
	if ix.Indexes == nil {
		return ix.Tensor.DimSize(0)
	}
	return len(ix.Indexes)
}

// String satisfies the fmt.Stringer interface for string of tensor data.
func (ix *Rows) String() string {
	return sprint(ix.Tensor, 0) // todo: no need
}

// Label satisfies the core.Labeler interface for a summary description of the tensor.
func (ix *Rows) Label() string {
	return ix.Tensor.Label()
}

// Metadata returns the metadata for this tensor, which can be used
// to encode plotting options, etc.
func (ix *Rows) Metadata() *metadata.Data { return ix.Tensor.Metadata() }

// If we have Indexes, this is the effective shape sizes using
// the current number of indexes as the outermost row dimension size.
func (ix *Rows) ShapeInts() []int {
	if ix.Indexes == nil || ix.Tensor.NumDims() == 0 {
		return ix.Tensor.ShapeInts()
	}
	sh := slices.Clone(ix.Tensor.ShapeInts())
	sh[0] = len(ix.Indexes)
	return sh
}

func (ix *Rows) ShapeSizes() Tensor {
	if ix.Indexes == nil {
		return ix.Tensor.ShapeSizes()
	}
	return NewIntFromSlice(ix.ShapeInts()...)
}

// Shape() returns a [Shape] representation of the tensor shape
// (dimension sizes). If we have Indexes, this is the effective
// shape using the current number of indexes as the outermost row dimension size.
func (ix *Rows) Shape() *Shape {
	if ix.Indexes == nil {
		return ix.Tensor.Shape()
	}
	return NewShape(ix.ShapeInts()...)
}

// SetShapeInts sets our shape to given sizes.
// If we do not have indexes, or the row-wise shape dimension
// in the new shape is the same as current, then we set the shape
// of the wrapped Tensor accordingly.
// This allows reshaping of inner dimensions while preserving indexes,
// e.g., for computational routines that use a 1D cell view.
// Otherwise, we reset the indexes and then set the wrapped shape,
// because our current indexes are now invalidated.
func (ix *Rows) SetShapeInts(sizes ...int) {
	if ix.Indexes == nil || ix.Tensor.NumDims() == 0 {
		ix.Tensor.SetShapeInts(sizes...)
		return
	}
	sh := ix.Tensor.ShapeInts()
	if sizes[0] == sh[0] { // keep our indexes
		ix.Tensor.SetShapeInts(sizes...)
		return
	}
	ix.Indexes = nil // now invalid
	ix.Tensor.SetShapeInts(sizes...)
}

// SetNumRows sets the number of rows (outermost dimension) in a RowMajor organized tensor.
// This invalidates the indexes.
func (ix *Rows) SetNumRows(rows int) {
	ix.Sequential()
	ix.Tensor.SetNumRows(rows)
}

// SetShape sets our shape to given sizes.
// See [Rows.SetShapeInts] for details.
func (ix *Rows) SetShape(sizes Tensor) {
	ix.SetShapeInts(AsIntSlice(sizes)...)
}

// Len returns the total number of elements in the tensor,
// taking into account the Indexes via [Rows],
// as NumRows() * cell size.
func (ix *Rows) Len() int {
	rows := ix.NumRows()
	_, cells := ix.Tensor.RowCellSize()
	return cells * rows
}

// NumDims returns the total number of dimensions.
func (ix *Rows) NumDims() int { return ix.Tensor.NumDims() }

// DimSize returns size of given dimension, returning NumRows()
// for first dimension.
func (ix *Rows) DimSize(dim int) int {
	if dim == 0 {
		return ix.NumRows()
	}
	return ix.Tensor.DimSize(dim)
}

// RowCellSize returns the size of the outermost Row shape dimension
// (via [Rows.NumRows] method), and the size of all the remaining
// inner dimensions (the "cell" size).
func (ix *Rows) RowCellSize() (rows, cells int) {
	_, cells = ix.Tensor.RowCellSize()
	rows = ix.NumRows()
	return
}

// ValidIndexes deletes all invalid indexes from the list.
// Call this if rows (could) have been deleted from tensor.
func (ix *Rows) ValidIndexes() {
	if ix.Tensor.DimSize(0) <= 0 || ix.Indexes == nil {
		ix.Indexes = nil
		return
	}
	ni := ix.NumRows()
	for i := ni - 1; i >= 0; i-- {
		if ix.Indexes[i] >= ix.Tensor.DimSize(0) {
			ix.Indexes = append(ix.Indexes[:i], ix.Indexes[i+1:]...)
		}
	}
}

// Sequential sets Indexes to nil, resulting in sequential row-wise access into tensor.
func (ix *Rows) Sequential() { //types:add
	ix.Indexes = nil
}

// IndexesNeeded is called prior to an operation that needs actual indexes,
// e.g., Sort, Filter.  If Indexes == nil, they are set to all rows, otherwise
// current indexes are left as is. Use Sequential, then IndexesNeeded to ensure
// all rows are represented.
func (ix *Rows) IndexesNeeded() {
	if ix.Tensor.DimSize(0) <= 0 {
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

// ExcludeMissing deletes indexes where the values are missing, as indicated by NaN.
// Uses first cell of higher dimensional data.
func (ix *Rows) ExcludeMissing() { //types:add
	if ix.Tensor.DimSize(0) <= 0 {
		ix.Indexes = nil
		return
	}
	ix.IndexesNeeded()
	ni := ix.NumRows()
	for i := ni - 1; i >= 0; i-- {
		if math.IsNaN(ix.Tensor.FloatRowCell(ix.Indexes[i], 0)) {
			ix.Indexes = append(ix.Indexes[:i], ix.Indexes[i+1:]...)
		}
	}
}

// Permuted sets indexes to a permuted order.  If indexes already exist
// then existing list of indexes is permuted, otherwise a new set of
// permuted indexes are generated
func (ix *Rows) Permuted() {
	if ix.Tensor.DimSize(0) <= 0 {
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

const (
	// Ascending specifies an ascending sort direction for tensor Sort routines
	Ascending = true

	// Descending specifies a descending sort direction for tensor Sort routines
	Descending = false

	//	Stable specifies using stable, original order-preserving sort, which is slower.
	Stable = true

	//	Unstable specifies using faster but unstable sorting.
	Unstable = false
)

// SortFunc sorts the row-wise indexes using given compare function.
// The compare function operates directly on row numbers into the Tensor
// as these row numbers have already been projected through the indexes.
// cmp(a, b) should return a negative number when a < b, a positive
// number when a > b and zero when a == b.
func (ix *Rows) SortFunc(cmp func(tsr Tensor, i, j int) int) {
	ix.IndexesNeeded()
	slices.SortFunc(ix.Indexes, func(a, b int) int {
		return cmp(ix.Tensor, a, b) // key point: these are already indirected through indexes!!
	})
}

// SortIndexes sorts the indexes into our Tensor directly in
// numerical order, producing the native ordering, while preserving
// any filtering that might have occurred.
func (ix *Rows) SortIndexes() {
	if ix.Indexes == nil {
		return
	}
	sort.Ints(ix.Indexes)
}

func CompareAscending[T cmp.Ordered](a, b T, ascending bool) int {
	if ascending {
		return cmp.Compare(a, b)
	}
	return cmp.Compare(b, a)
}

// Sort does default alpha or numeric sort of row-wise data.
// Uses first cell of higher dimensional data.
func (ix *Rows) Sort(ascending bool) {
	if ix.Tensor.IsString() {
		ix.SortFunc(func(tsr Tensor, i, j int) int {
			return CompareAscending(tsr.StringRowCell(i, 0), tsr.StringRowCell(j, 0), ascending)
		})
	} else {
		ix.SortFunc(func(tsr Tensor, i, j int) int {
			return CompareAscending(tsr.FloatRowCell(i, 0), tsr.FloatRowCell(j, 0), ascending)
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
func (ix *Rows) SortStableFunc(cmp func(tsr Tensor, i, j int) int) {
	ix.IndexesNeeded()
	slices.SortStableFunc(ix.Indexes, func(a, b int) int {
		return cmp(ix.Tensor, a, b) // key point: these are already indirected through indexes!!
	})
}

// SortStable does stable default alpha or numeric sort.
// Uses first cell of higher dimensional data.
func (ix *Rows) SortStable(ascending bool) {
	if ix.Tensor.IsString() {
		ix.SortStableFunc(func(tsr Tensor, i, j int) int {
			return CompareAscending(tsr.StringRowCell(i, 0), tsr.StringRowCell(j, 0), ascending)
		})
	} else {
		ix.SortStableFunc(func(tsr Tensor, i, j int) int {
			return CompareAscending(tsr.FloatRowCell(i, 0), tsr.FloatRowCell(j, 0), ascending)
		})
	}
}

// FilterFunc is a function used for filtering that returns
// true if Tensor row should be included in the current filtered
// view of the tensor, and false if it should be removed.
type FilterFunc func(tsr Tensor, row int) bool

// Filter filters the indexes using given Filter function.
// The Filter function operates directly on row numbers into the Tensor
// as these row numbers have already been projected through the indexes.
func (ix *Rows) Filter(filterer func(tsr Tensor, row int) bool) {
	ix.IndexesNeeded()
	sz := len(ix.Indexes)
	for i := sz - 1; i >= 0; i-- { // always go in reverse for filtering
		if !filterer(ix.Tensor, ix.Indexes[i]) { // delete
			ix.Indexes = append(ix.Indexes[:i], ix.Indexes[i+1:]...)
		}
	}
}

// Named arg values for FilterString
const (
	// Include means include matches
	Include = false
	// Exclude means exclude matches
	Exclude = true
	// Contains means the string only needs to contain the target string (see Equals)
	Contains = true
	// Equals means the string must equal the target string (see Contains)
	Equals = false
	// IgnoreCase means that differences in case are ignored in comparing strings
	IgnoreCase = true
	// UseCase means that case matters when comparing strings
	UseCase = false
)

// FilterString filters the indexes using string values compared to given
// string. Includes rows with matching values unless exclude is set.
// If contains, only checks if row contains string; if ignoreCase, ignores case.
// Use the named const args [Include], [Exclude], [Contains], [Equals],
// [IgnoreCase], [UseCase] for greater clarity.
// Uses first cell of higher dimensional data.
func (ix *Rows) FilterString(str string, exclude, contains, ignoreCase bool) { //types:add
	lowstr := strings.ToLower(str)
	ix.Filter(func(tsr Tensor, row int) bool {
		val := tsr.StringRowCell(row, 0)
		has := false
		switch {
		case contains && ignoreCase:
			has = strings.Contains(strings.ToLower(val), lowstr)
		case contains:
			has = strings.Contains(val, str)
		case ignoreCase:
			has = strings.EqualFold(val, str)
		default:
			has = (val == str)
		}
		if exclude {
			return !has
		}
		return has
	})
}

// NewTensor returns a new tensor with column data organized according to
// the Indexes.  If Indexes are nil, a clone of the current tensor is returned
// but this function is only sensible if there is an indexed view in place.
func (ix *Rows) NewTensor() Tensor {
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

// Clone returns a copy of the current Rows view with a cloned copy of
// the underlying Tensor and copy of the indexes.
func (ix *Rows) Clone() Tensor {
	nix := &Rows{}
	nix.Tensor = ix.Tensor.Clone()
	nix.CopyIndexes(ix)
	return nix
}

func (ix *Rows) View() Tensor {
	nix := &Rows{}
	nix.Tensor = ix.Tensor.View()
	nix.CopyIndexes(ix)
	return nix
}

// CloneIndexes returns a copy of the current Rows view with new indexes,
// with a pointer to the same underlying Tensor as the source.
func (ix *Rows) CloneIndexes() *Rows {
	nix := &Rows{}
	nix.Tensor = ix.Tensor
	nix.CopyIndexes(ix)
	return nix
}

// CopyIndexes copies indexes from other Rows view.
func (ix *Rows) CopyIndexes(oix *Rows) {
	if oix.Indexes == nil {
		ix.Indexes = nil
	} else {
		ix.Indexes = slices.Clone(oix.Indexes)
	}
}

// AddRows adds n rows to end of underlying Tensor, and to the indexes in this view
func (ix *Rows) AddRows(n int) { //types:add
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
func (ix *Rows) InsertRows(at, n int) {
	stidx := ix.Tensor.DimSize(0)
	ix.IndexesNeeded()
	ix.Tensor.SetNumRows(stidx + n)
	nw := make([]int, n, n+len(ix.Indexes)-at)
	for i := 0; i < n; i++ {
		nw[i] = stidx + i
	}
	ix.Indexes = append(ix.Indexes[:at], append(nw, ix.Indexes[at:]...)...)
}

// DeleteRows deletes n rows of indexes starting at given index in the list of indexes
func (ix *Rows) DeleteRows(at, n int) {
	ix.IndexesNeeded()
	ix.Indexes = append(ix.Indexes[:at], ix.Indexes[at+n:]...)
}

// Swap switches the indexes for i and j
func (ix *Rows) Swap(i, j int) {
	if ix.Indexes == nil {
		return
	}
	ix.Indexes[i], ix.Indexes[j] = ix.Indexes[j], ix.Indexes[i]
}

///////////////////////////////////////////////
// Rows access

/////////////////////  Floats

// Float returns the value of given index as a float64.
// The first index value is indirected through the indexes.
func (ix *Rows) Float(i ...int) float64 {
	if ix.Indexes == nil {
		return ix.Tensor.Float(i...)
	}
	ic := slices.Clone(i)
	ic[0] = ix.Indexes[ic[0]]
	return ix.Tensor.Float(ic...)
}

// SetFloat sets the value of given index as a float64
// The first index value is indirected through the [Rows.Indexes].
func (ix *Rows) SetFloat(val float64, i ...int) {
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
// Row is indirected through the [Rows.Indexes].
// This is the preferred interface for all Rows operations.
func (ix *Rows) FloatRowCell(row, cell int) float64 {
	return ix.Tensor.FloatRowCell(ix.RowIndex(row), cell)
}

// SetFloatRowCell sets the value at given row and cell,
// where row is outermost dim, and cell is 1D index into remaining inner dims.
// Row is indirected through the [Rows.Indexes].
// This is the preferred interface for all Rows operations.
func (ix *Rows) SetFloatRowCell(val float64, row, cell int) {
	ix.Tensor.SetFloatRowCell(val, ix.RowIndex(row), cell)
}

// Float1D is somewhat expensive if indexes are set, because it needs to convert
// the flat index back into a full n-dimensional index and then use that api.
func (ix *Rows) Float1D(i int) float64 {
	if ix.Indexes == nil {
		return ix.Tensor.Float1D(i)
	}
	return ix.Float(ix.Tensor.Shape().IndexFrom1D(i)...)
}

// SetFloat1D is somewhat expensive if indexes are set, because it needs to convert
// the flat index back into a full n-dimensional index and then use that api.
func (ix *Rows) SetFloat1D(val float64, i int) {
	if ix.Indexes == nil {
		ix.Tensor.SetFloat1D(val, i)
	}
	ix.SetFloat(val, ix.Tensor.Shape().IndexFrom1D(i)...)
}

func (ix *Rows) FloatRow(row int) float64 {
	return ix.FloatRowCell(row, 0)
}

func (ix *Rows) SetFloatRow(val float64, row int) {
	ix.SetFloatRowCell(val, row, 0)
}

/////////////////////  Strings

// StringValue returns the value of given index as a string.
// The first index value is indirected through the indexes.
func (ix *Rows) StringValue(i ...int) string {
	if ix.Indexes == nil {
		return ix.Tensor.StringValue(i...)
	}
	ic := slices.Clone(i)
	ic[0] = ix.Indexes[ic[0]]
	return ix.Tensor.StringValue(ic...)
}

// SetString sets the value of given index as a string
// The first index value is indirected through the [Rows.Indexes].
func (ix *Rows) SetString(val string, i ...int) {
	if ix.Indexes == nil {
		ix.Tensor.SetString(val, i...)
	}
	ic := slices.Clone(i)
	ic[0] = ix.Indexes[ic[0]]
	ix.Tensor.SetString(val, ic...)
}

// StringRowCell returns the value at given row and cell,
// where row is outermost dim, and cell is 1D index into remaining inner dims.
// Row is indirected through the [Rows.Indexes].
// This is the preferred interface for all Rows operations.
func (ix *Rows) StringRowCell(row, cell int) string {
	return ix.Tensor.StringRowCell(ix.RowIndex(row), cell)
}

// SetStringRowCell sets the value at given row and cell,
// where row is outermost dim, and cell is 1D index into remaining inner dims.
// Row is indirected through the [Rows.Indexes].
// This is the preferred interface for all Rows operations.
func (ix *Rows) SetStringRowCell(val string, row, cell int) {
	ix.Tensor.SetStringRowCell(val, ix.RowIndex(row), cell)
}

// String1D is somewhat expensive if indexes are set, because it needs to convert
// the flat index back into a full n-dimensional index and then use that api.
func (ix *Rows) String1D(i int) string {
	if ix.Indexes == nil {
		return ix.Tensor.String1D(i)
	}
	return ix.StringValue(ix.Tensor.Shape().IndexFrom1D(i)...)
}

// SetString1D is somewhat expensive if indexes are set, because it needs to convert
// the flat index back into a full n-dimensional index and then use that api.
func (ix *Rows) SetString1D(val string, i int) {
	if ix.Indexes == nil {
		ix.Tensor.SetString1D(val, i)
	}
	ix.SetString(val, ix.Tensor.Shape().IndexFrom1D(i)...)
}

func (ix *Rows) StringRow(row int) string {
	return ix.StringRowCell(row, 0)
}

func (ix *Rows) SetStringRow(val string, row int) {
	ix.SetStringRowCell(val, row, 0)
}

/////////////////////  Ints

// Int returns the value of given index as an int.
// The first index value is indirected through the indexes.
func (ix *Rows) Int(i ...int) int {
	if ix.Indexes == nil {
		return ix.Tensor.Int(i...)
	}
	ic := slices.Clone(i)
	ic[0] = ix.Indexes[ic[0]]
	return ix.Tensor.Int(ic...)
}

// SetInt sets the value of given index as an int
// The first index value is indirected through the [Rows.Indexes].
func (ix *Rows) SetInt(val int, i ...int) {
	if ix.Indexes == nil {
		ix.Tensor.SetInt(val, i...)
		return
	}
	ic := slices.Clone(i)
	ic[0] = ix.Indexes[ic[0]]
	ix.Tensor.SetInt(val, ic...)
}

// IntRowCell returns the value at given row and cell,
// where row is outermost dim, and cell is 1D index into remaining inner dims.
// Row is indirected through the [Rows.Indexes].
// This is the preferred interface for all Rows operations.
func (ix *Rows) IntRowCell(row, cell int) int {
	return ix.Tensor.IntRowCell(ix.RowIndex(row), cell)
}

// SetIntRowCell sets the value at given row and cell,
// where row is outermost dim, and cell is 1D index into remaining inner dims.
// Row is indirected through the [Rows.Indexes].
// This is the preferred interface for all Rows operations.
func (ix *Rows) SetIntRowCell(val int, row, cell int) {
	ix.Tensor.SetIntRowCell(val, ix.RowIndex(row), cell)
}

// Int1D is somewhat expensive if indexes are set, because it needs to convert
// the flat index back into a full n-dimensional index and then use that api.
func (ix *Rows) Int1D(i int) int {
	if ix.Indexes == nil {
		return ix.Tensor.Int1D(i)
	}
	return ix.Int(ix.Tensor.Shape().IndexFrom1D(i)...)
}

// SetInt1D is somewhat expensive if indexes are set, because it needs to convert
// the flat index back into a full n-dimensional index and then use that api.
func (ix *Rows) SetInt1D(val int, i int) {
	if ix.Indexes == nil {
		ix.Tensor.SetInt1D(val, i)
	}
	ix.SetInt(val, ix.Tensor.Shape().IndexFrom1D(i)...)
}

func (ix *Rows) IntRow(row int) int {
	return ix.IntRowCell(row, 0)
}

func (ix *Rows) SetIntRow(val int, row int) {
	ix.SetIntRowCell(val, row, 0)
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
func (ix *Rows) SubSpace(offs ...int) Tensor {
	if len(offs) == 0 {
		return nil
	}
	offs[0] = ix.RowIndex(offs[0])
	return ix.Tensor.SubSpace(offs...)
}

// RowTensor is a convenience version of [Rows.SubSpace] to return the
// SubSpace for the outermost row dimension, indirected through the indexes.
func (ix *Rows) RowTensor(row int) Tensor {
	return ix.Tensor.RowTensor(ix.RowIndex(row))
}

// SetRowTensor sets the values of the SubSpace at given row to given values,
// with row indirected through the indexes.
func (ix *Rows) SetRowTensor(val Tensor, row int) {
	ix.Tensor.SetRowTensor(val, ix.RowIndex(row))
}

// CopyFrom copies all values from other tensor into this tensor.
// Checks if source is an Rows and copies indexes too,
// otherwise underlying tensor copies from and indexes are reset.
func (ix *Rows) CopyFrom(from Tensor) {
	if fix, ok := from.(*Rows); ok {
		ix.Tensor.CopyFrom(fix.Tensor)
		ix.CopyIndexes(fix)
		return
	}
	ix.Sequential()
	ix.Tensor.CopyFrom(from)
}

// AppendFrom appends all values from other tensor into this tensor.
// This invalidates the indexes which are reset.
func (ix *Rows) AppendFrom(from Tensor) error {
	ix.Sequential()
	return ix.Tensor.AppendFrom(from)
}

// CopyCellsFrom copies given range of values from other tensor into this tensor,
// This invalidates the indexes which are reset.
func (ix *Rows) CopyCellsFrom(from Tensor, to, start, n int) {
	ix.Tensor.CopyCellsFrom(from, to, start, n)
}

func (ix *Rows) Sizeof() int64 {
	return ix.Tensor.Sizeof() // todo: could be out of sync with shape!
}

func (ix *Rows) Bytes() []byte {
	return ix.Tensor.Bytes() // todo: could be out of sync with shape!
}

func (ix *Rows) IsString() bool {
	return ix.Tensor.IsString()
}

func (ix *Rows) DataType() reflect.Kind {
	return ix.Tensor.DataType()
}

func (ix *Rows) Range() (min, max float64, minIndex, maxIndex int) {
	return ix.Tensor.Range()
}

func (ix *Rows) SetZeros() {
	ix.Tensor.SetZeros()
}

//////////////////////////  gonum matrix api

// Dims is the gonum/mat.Matrix interface method for returning the dimensionality of the
// 2D Matrix.  Assumes Row-major ordering and logs an error if NumDims < 2.
func (ix *Rows) Dims() (r, c int) {
	nd := ix.NumDims()
	if nd < 2 {
		log.Println("tensor Dims gonum Matrix call made on Tensor with dims < 2")
		return 0, 0
	}
	return ix.DimSize(nd - 2), ix.DimSize(nd - 1)
}

// Symmetric is the gonum/mat.Matrix interface method for returning the dimensionality of a symmetric
// 2D Matrix.
func (ix *Rows) Symmetric() (r int) {
	nd := ix.NumDims()
	if nd < 2 {
		log.Println("tensor Symmetric gonum Matrix call made on Tensor with dims < 2")
		return 0
	}
	if ix.DimSize(nd-2) != ix.DimSize(nd-1) {
		log.Println("tensor Symmetric gonum Matrix call made on Tensor that is not symmetric")
		return 0
	}
	return ix.DimSize(nd - 1)
}

// SymmetricDim returns the number of rows/columns in the matrix.
func (ix *Rows) SymmetricDim() int {
	nd := ix.NumDims()
	if nd < 2 {
		log.Println("tensor Symmetric gonum Matrix call made on Tensor with dims < 2")
		return 0
	}
	if ix.DimSize(nd-2) != ix.DimSize(nd-1) {
		log.Println("tensor Symmetric gonum Matrix call made on Tensor that is not symmetric")
		return 0
	}
	return ix.DimSize(nd - 1)
}

// At is the gonum/mat.Matrix interface method for returning 2D matrix element at given
// row, column index.  Assumes Row-major ordering and logs an error if NumDims < 2.
func (ix *Rows) At(i, j int) float64 {
	nd := ix.NumDims()
	if nd < 2 {
		log.Println("tensor Dims gonum Matrix call made on Tensor with dims < 2")
		return 0
	} else if nd == 2 {
		return ix.Float(i, j)
	} else {
		nix := make([]int, nd)
		nix[nd-2] = i
		nix[nd-1] = j
		return ix.Float(nix...)
	}
}

// T is the gonum/mat.Matrix transpose method.
// It performs an implicit transpose by returning the receiver inside a Transpose.
func (ix *Rows) T() mat.Matrix {
	return mat.Transpose{ix}
}

// check for interface impl
var _ Tensor = (*Rows)(nil)
