// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package tensor

import (
	"cmp"
	"math"
	"math/rand"
	"slices"
	"sort"
	"strings"

	"cogentcore.org/core/base/metadata"
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

	// Indexes are the indexes into Tensor rows, with nil = sequential.
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

// NewFloat64Indexed is a convenience method to quickly get an Indexed
// representation of [Float64] tensor of given shape, for use in math routines etc.
func NewFloat64Indexed(sizes ...int) *Indexed {
	return &Indexed{Tensor: NewFloat64(sizes...)}
}

// NewFloat64Scalar is a convenience method to quickly get an Indexed
// representation of a single float64 scalar value, for use in math routines etc.
func NewFloat64Scalar(val float64) *Indexed {
	return &Indexed{Tensor: NewNumberFromSlice(val)}
}

// NewIntScalar is a convenience method to quickly get an Indexed
// representation of a single int scalar value, for use in math routines etc.
func NewIntScalar(val int) *Indexed {
	return &Indexed{Tensor: NewNumberFromSlice(val)}
}

// NewStringScalar is a convenience method to quickly get an Indexed
// representation of a single string scalar value, for use in math routines etc.
func NewStringScalar(val string) *Indexed {
	return &Indexed{Tensor: NewStringTensorFromSlice(val)}
}

// NewFloat64FromSlice returns a new 1-dimensional tensor of given value type
// initialized directly from the given slice values, which are not copied.
// The resulting Tensor thus "wraps" the given values.
func NewFloat64FromSlice(vals ...float64) *Indexed {
	return &Indexed{Tensor: NewNumberFromSlice(vals...)}
}

// NewIntFromSlice returns a new 1-dimensional tensor of given value type
// initialized directly from the given slice values, which are not copied.
// The resulting Tensor thus "wraps" the given values.
func NewIntFromSlice(vals ...int) *Indexed {
	return &Indexed{Tensor: NewNumberFromSlice(vals...)}
}

// NewStringFromSlice returns a new 1-dimensional tensor of given value type
// initialized directly from the given slice values, which are not copied.
// The resulting Tensor thus "wraps" the given values.
func NewStringFromSlice(vals ...string) *Indexed {
	return &Indexed{Tensor: NewStringTensorFromSlice(vals...)}
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

// String satisfies the fmt.Stringer interface for string of tensor data.
func (ix *Indexed) String() string {
	return stringIndexed(ix.Tensor, 0, ix.Indexes)
}

// Label satisfies the core.Labeler interface for a summary description of the tensor.
func (ix *Indexed) Label() string {
	return ix.Tensor.Label()
}

// note: goal transpiling needs all expressions to work directly on Indexed
// so we need wrappers for everything.

// Shape returns a pointer to the shape that fully parametrizes the tensor shape.
func (ix *Indexed) Shape() *Shape { return ix.Tensor.Shape() }

// Metadata returns the metadata for this tensor, which can be used
// to encode plotting options, etc.
func (ix *Indexed) Metadata() *metadata.Data { return ix.Tensor.Metadata() }

// NumDims returns the total number of dimensions.
func (ix *Indexed) NumDims() int { return ix.Tensor.NumDims() }

// DimSize returns size of given dimension.
func (ix *Indexed) DimSize(dim int) int { return ix.Tensor.DimSize(dim) }

// RowCellSize returns the size of the outermost Row shape dimension
// (via [Indexed.Rows] method), and the size of all the remaining
// inner dimensions (the "cell" size).
func (ix *Indexed) RowCellSize() (rows, cells int) {
	_, cells = ix.Tensor.RowCellSize()
	rows = ix.NumRows()
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

// NumRows returns the effective number of rows in this Indexed view,
// which is the length of the index list or number of outer
// rows dimension of tensor if no indexes (full sequential view).
func (ix *Indexed) NumRows() int {
	if ix.Indexes == nil {
		return ix.Tensor.DimSize(0)
	}
	return len(ix.Indexes)
}

// Len returns the total number of elements in the tensor,
// taking into account the Indexes via [Rows],
// as Rows() * cell size.
func (ix *Indexed) Len() int {
	rows := ix.NumRows()
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
	ni := ix.NumRows()
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

// ExcludeMissing deletes indexes where the values are missing, as indicated by NaN.
// Uses first cell of higher dimensional data.
func (ix *Indexed) ExcludeMissing() { //types:add
	if ix.Tensor == nil || ix.Tensor.DimSize(0) <= 0 {
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
func (ix *Indexed) SortFunc(cmp func(tsr Tensor, i, j int) int) {
	ix.IndexesNeeded()
	slices.SortFunc(ix.Indexes, func(a, b int) int {
		return cmp(ix.Tensor, a, b) // key point: these are already indirected through indexes!!
	})
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

func CompareAscending[T cmp.Ordered](a, b T, ascending bool) int {
	if ascending {
		return cmp.Compare(a, b)
	}
	return cmp.Compare(b, a)
}

// Sort does default alpha or numeric sort of row-wise data.
// Uses first cell of higher dimensional data.
func (ix *Indexed) Sort(ascending bool) {
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
func (ix *Indexed) SortStableFunc(cmp func(tsr Tensor, i, j int) int) {
	ix.IndexesNeeded()
	slices.SortStableFunc(ix.Indexes, func(a, b int) int {
		return cmp(ix.Tensor, a, b) // key point: these are already indirected through indexes!!
	})
}

// SortStable does stable default alpha or numeric sort.
// Uses first cell of higher dimensional data.
func (ix *Indexed) SortStable(ascending bool) {
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
func (ix *Indexed) Filter(filterer func(tsr Tensor, row int) bool) {
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
func (ix *Indexed) FilterString(str string, exclude, contains, ignoreCase bool) { //types:add
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
	ix.IndexesNeeded()
	ix.Tensor.SetNumRows(stidx + n)
	nw := make([]int, n, n+len(ix.Indexes)-at)
	for i := 0; i < n; i++ {
		nw[i] = stidx + i
	}
	ix.Indexes = append(ix.Indexes[:at], append(nw, ix.Indexes[at:]...)...)
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

/////////////////////  Floats

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
// The first index value is indirected through the [Indexed.Indexes].
func (ix *Indexed) SetFloat(val float64, i ...int) {
	if ix.Indexes == nil {
		ix.Tensor.SetFloat(val, i...)
		return
	}
	ic := slices.Clone(i)
	ic[0] = ix.Indexes[ic[0]]
	ix.Tensor.SetFloat(val, ic...)
}

// FloatRow returns the value at given row (outermost dimension).
// Row is indirected through the [Indexed.Indexes].
// It is a convenience wrapper for FloatRowCell(row, 0), providing robust
// operations on 1D and higher-dimensional data (which nevertheless should
// generally be processed separately in ways that treat it properly).
func (ix *Indexed) FloatRow(row int) float64 {
	return ix.Tensor.FloatRowCell(ix.Index(row), 0)
}

// SetFloatRow sets the value at given row (outermost dimension).
// Row is indirected through the [Indexed.Indexes].
// It is a convenience wrapper for SetFloatRowCell(row, 0), providing robust
// operations on 1D and higher-dimensional data (which nevertheless should
// generally be processed separately in ways that treat it properly).
func (ix *Indexed) SetFloatRow(val float64, row int) {
	ix.Tensor.SetFloatRowCell(val, ix.Index(row), 0)
}

// FloatRowCell returns the value at given row and cell,
// where row is outermost dim, and cell is 1D index into remaining inner dims.
// Row is indirected through the [Indexed.Indexes].
// This is the preferred interface for all Indexed operations.
func (ix *Indexed) FloatRowCell(row, cell int) float64 {
	return ix.Tensor.FloatRowCell(ix.Index(row), cell)
}

// SetFloatRowCell sets the value at given row and cell,
// where row is outermost dim, and cell is 1D index into remaining inner dims.
// Row is indirected through the [Indexed.Indexes].
// This is the preferred interface for all Indexed operations.
func (ix *Indexed) SetFloatRowCell(val float64, row, cell int) {
	ix.Tensor.SetFloatRowCell(val, ix.Index(row), cell)
}

// Float1D returns the value of given 1-dimensional index (0-Len()-1) as a float64.
// This is just a convenience pass-through to the Tensor, and does _not_ use
// the [Indexed.Indexes].
func (ix *Indexed) Float1D(i int) float64 {
	return ix.Tensor.Float1D(i)
}

// SetFloat1D sets the value of given 1-dimensional index (0-Len()-1) as a float64.
// This is just a convenience pass-through to the Tensor, and does _not_ use
// the [Indexed.Indexes].
func (ix *Indexed) SetFloat1D(val float64, i int) {
	ix.Tensor.SetFloat1D(val, i)
}

/////////////////////  Strings

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
// The first index value is indirected through the [Indexed.Indexes].
func (ix *Indexed) SetString(val string, i ...int) {
	if ix.Indexes == nil {
		ix.Tensor.SetString(val, i...)
	}
	ic := slices.Clone(i)
	ic[0] = ix.Indexes[ic[0]]
	ix.Tensor.SetString(val, ic...)
}

// StringRow returns the value at given row (outermost dimension).
// Row is indirected through the [Indexed.Indexes].
// It is a convenience wrapper for StringRowCell(row, 0), providing robust
// operations on 1D and higher-dimensional data (which nevertheless should
// generally be processed separately in ways that treat it properly).
func (ix *Indexed) StringRow(row int) string {
	return ix.Tensor.StringRowCell(ix.Index(row), 0)
}

// SetStringRow sets the value at given row (outermost dimension).
// Row is indirected through the [Indexed.Indexes].
// It is a convenience wrapper for SetStringRowCell(row, 0), providing robust
// operations on 1D and higher-dimensional data (which nevertheless should
// generally be processed separately in ways that treat it properly).
func (ix *Indexed) SetStringRow(val string, row int) {
	ix.Tensor.SetStringRowCell(val, ix.Index(row), 0)
}

// StringRowCell returns the value at given row and cell,
// where row is outermost dim, and cell is 1D index into remaining inner dims.
// Row is indirected through the [Indexed.Indexes].
// This is the preferred interface for all Indexed operations.
func (ix *Indexed) StringRowCell(row, cell int) string {
	return ix.Tensor.StringRowCell(ix.Index(row), cell)
}

// SetStringRowCell sets the value at given row and cell,
// where row is outermost dim, and cell is 1D index into remaining inner dims.
// Row is indirected through the [Indexed.Indexes].
// This is the preferred interface for all Indexed operations.
func (ix *Indexed) SetStringRowCell(val string, row, cell int) {
	ix.Tensor.SetStringRowCell(val, ix.Index(row), cell)
}

// String1D returns the value of given 1-dimensional index (0-Len()-1) as a string.
// This is just a convenience pass-through to the Tensor, and does _not_ use
// the [Indexed.Indexes].
func (ix *Indexed) String1D(i int) string {
	return ix.Tensor.String1D(i)
}

// SetString1D sets the value of given 1-dimensional index (0-Len()-1) as a string.
// This is just a convenience pass-through to the Tensor, and does _not_ use
// the [Indexed.Indexes].
func (ix *Indexed) SetString1D(val string, i int) {
	ix.Tensor.SetString1D(val, i)
}

/////////////////////  Ints

// Int returns the value of given index as an int.
// The first index value is indirected through the indexes.
func (ix *Indexed) Int(i ...int) int {
	if ix.Indexes == nil {
		return ix.Tensor.Int(i...)
	}
	ic := slices.Clone(i)
	ic[0] = ix.Indexes[ic[0]]
	return ix.Tensor.Int(ic...)
}

// SetInt sets the value of given index as an int
// The first index value is indirected through the [Indexed.Indexes].
func (ix *Indexed) SetInt(val int, i ...int) {
	if ix.Indexes == nil {
		ix.Tensor.SetInt(val, i...)
		return
	}
	ic := slices.Clone(i)
	ic[0] = ix.Indexes[ic[0]]
	ix.Tensor.SetInt(val, ic...)
}

// IntRow returns the value at given row (outermost dimension).
// Row is indirected through the [Indexed.Indexes].
// It is a convenience wrapper for IntRowCell(row, 0), providing robust
// operations on 1D and higher-dimensional data (which nevertheless should
// generally be processed separately in ways that treat it properly).
func (ix *Indexed) IntRow(row int) int {
	return ix.Tensor.IntRowCell(ix.Index(row), 0)
}

// SetIntRow sets the value at given row (outermost dimension).
// Row is indirected through the [Indexed.Indexes].
// It is a convenience wrapper for SetIntRowCell(row, 0), providing robust
// operations on 1D and higher-dimensional data (which nevertheless should
// generally be processed separately in ways that treat it properly).
func (ix *Indexed) SetIntRow(val int, row int) {
	ix.Tensor.SetIntRowCell(val, ix.Index(row), 0)
}

// IntRowCell returns the value at given row and cell,
// where row is outermost dim, and cell is 1D index into remaining inner dims.
// Row is indirected through the [Indexed.Indexes].
// This is the preferred interface for all Indexed operations.
func (ix *Indexed) IntRowCell(row, cell int) int {
	return ix.Tensor.IntRowCell(ix.Index(row), cell)
}

// SetIntRowCell sets the value at given row and cell,
// where row is outermost dim, and cell is 1D index into remaining inner dims.
// Row is indirected through the [Indexed.Indexes].
// This is the preferred interface for all Indexed operations.
func (ix *Indexed) SetIntRowCell(val int, row, cell int) {
	ix.Tensor.SetIntRowCell(val, ix.Index(row), cell)
}

// Int1D returns the value of given 1-dimensional index (0-Len()-1) as a int.
// This is just a convenience pass-through to the Tensor, and does _not_ use
// the [Indexed.Indexes].
func (ix *Indexed) Int1D(i int) int {
	return ix.Tensor.Int1D(i)
}

// SetInt1D sets the value of given 1-dimensional index (0-Len()-1) as a int.
// This is just a convenience pass-through to the Tensor, and does _not_ use
// the [Indexed.Indexes].
func (ix *Indexed) SetInt1D(val int, i int) {
	ix.Tensor.SetInt1D(val, i)
}

/////////////////////  SubSpaces

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

// Cells1D returns a flat 1D [tensor.Indexed] view of the cells for given row
// index (indirected through our Indexes).  This is useful for passing to
// other functions e.g., in stats or metrics that process a 1D tensor.
func (ix *Indexed) Cells1D(row int) *Indexed {
	return NewIndexed(New1DViewOf(ix.SubSpace(row)))
}

// RowTensor is a convenience version of [Indexed.SubSpace] to return the
// SubSpace for the outermost row dimension, indirected through the indexes.
func (ix *Indexed) RowTensor(row int) Tensor {
	return ix.Tensor.RowTensor(ix.Index(row))
}

// SetRowTensor sets the values of the SubSpace at given row to given values,
// with row indirected through the indexes.
func (ix *Indexed) SetRowTensor(val Tensor, row int) {
	ix.Tensor.SetRowTensor(val, ix.Index(row))
}
