// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package tensor

import (
	"log"
	"math/rand"
	"reflect"
	"slices"
	"sort"
	"strings"

	"cogentcore.org/core/base/metadata"
	"cogentcore.org/core/base/slicesx"
	"gonum.org/v1/gonum/mat"
)

// Slice extracts a subset of values from the given tensor into the
// output tensor, according to the provided ranges.
// Dimensions beyond the ranges specified are automatically included.
// Unlike the [Tensor.SubSlice] function, the values extracted here are
// copies of the original, not a slice pointer into them,
// which is necessary to allow discontinuous ranges to be extracted.
// Use the [SliceSet] function to copy sliced values back to the original.
func Slice(tsr, out *Sliced, ranges ...Range) error {
	sizes := slices.Clone(tsr.Tensor.ShapeInts())
	sizes[0] = tsr.NumRows() // takes into account indexes
	nsz, err := SliceSize(sizes, ranges...)
	if err != nil {
		return err
	}
	ndim := len(nsz)
	out.Tensor.SetShapeInts(nsz...)
	out.Sequential()
	nl := out.Len()
	oc := make([]int, ndim) // orig coords
	nr := len(ranges)
	for ni := range nl {
		nc := out.Tensor.Shape().IndexFrom1D(ni)
		for i := range ndim {
			c := nc[i]
			if i < nr {
				r := ranges[i]
				oc[i] = r.Start + c*r.IncrActual()
			} else {
				oc[i] = c
			}
		}
		oc[0] = tsr.RowIndex(oc[0])
		oi := tsr.Tensor.Shape().IndexTo1D(oc...)
		if out.Tensor.IsString() {
			out.Tensor.SetString1D(tsr.Tensor.String1D(oi), ni)
		} else {
			out.SetFloat1D(tsr.Float1D(oi), ni)
		}
	}
	return nil
}

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

// NewSliced returns a new [Sliced] view of given tensor,
// with optional list of indexes (none / nil = sequential).
func NewSliced(tsr Tensor, idxs ...[]int) *Sliced {
	sl := &Sliced{Tensor: tsr, Indexes: idxs}
	sl.ValidIndexes()
	return sl
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
	return sl.Tensor.Label()
}

// Metadata returns the metadata for this tensor, which can be used
// to encode plotting options, etc.
func (sl *Sliced) Metadata() *metadata.Data { return sl.Tensor.Metadata() }

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

// SetShapeInts sets the shape of the underlying wrapped tensor
// to the given sizes per dimension, and resets our indexes
// which are now invalid.
func (sl *Sliced) SetShapeInts(sizes ...int) {
	sl.Tensor.SetShapeInts(sizes...)
	sl.Sequential()
}

// SetShape sets our shape to given sizes.
// See [Sliced.SetShapeInts] for details.
func (sl *Sliced) SetShape(sizes Tensor) {
	sl.SetShapeInts(AsIntSlice(sizes)...)
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

// RowCellSize returns the size of the outermost Row shape dimension
// (via [Sliced.Rows] method), and the size of all the remaining
// inner dimensions (the "cell" size).
func (sl *Sliced) RowCellSize() (rows, cells int) {
	rows = sl.DimSize(0)
	nd := sl.Tensor.NumDims()
	if nd == 1 {
		cells = 1
	} else if rows > 0 {
		cells = sl.Len() / rows
	} else {
		ln := 1
		for d := 1; d < nd; d++ {
			ln *= sl.DimSize()
		}
		cells = ln
	}
	return
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

// Sort does default alpha or numeric sort along given dimension of data.
func (sl *Sliced) Sort(dim int, ascending bool) {
	if sl.Tensor.IsString() {
		sl.SortFunc(dim, func(tsr Tensor, dim, i, j int) int {
			return CompareAscending(tsr.StringRowCell(i, 0), tsr.StringRowCell(j, 0), ascending)
		})
	} else {
		sl.SortFunc(func(tsr Tensor, i, j int) int {
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
func (sl *Sliced) SortStableFunc(cmp func(tsr Tensor, i, j int) int) {
	sl.IndexesNeeded()
	slices.SortStableFunc(sl.Indexes, func(a, b int) int {
		return cmp(sl.Tensor, a, b) // key point: these are already indirected through indexes!!
	})
}

// SortStable does stable default alpha or numeric sort.
// Uses first cell of higher dimensional data.
func (sl *Sliced) SortStable(ascending bool) {
	if sl.Tensor.IsString() {
		sl.SortStableFunc(func(tsr Tensor, i, j int) int {
			return CompareAscending(tsr.StringRowCell(i, 0), tsr.StringRowCell(j, 0), ascending)
		})
	} else {
		sl.SortStableFunc(func(tsr Tensor, i, j int) int {
			return CompareAscending(tsr.FloatRowCell(i, 0), tsr.FloatRowCell(j, 0), ascending)
		})
	}
}

// Filter filters the indexes using given Filter function.
// The Filter function operates directly on row numbers into the Tensor
// as these row numbers have already been projected through the indexes.
func (sl *Sliced) Filter(filterer func(tsr Tensor, row int) bool) {
	sl.IndexesNeeded()
	sz := len(sl.Indexes)
	for i := sz - 1; i >= 0; i-- { // always go in reverse for filtering
		if !filterer(sl.Tensor, sl.Indexes[i]) { // delete
			sl.Indexes = append(sl.Indexes[:i], sl.Indexes[i+1:]...)
		}
	}
}

// FilterString filters the indexes using string values compared to given
// string. Includes rows with matching values unless exclude is set.
// If contains, only checks if row contains string; if ignoreCase, ignores case.
// Use the named const args [Include], [Exclude], [Contains], [Equals],
// [IgnoreCase], [UseCase] for greater clarity.
// Uses first cell of higher dimensional data.
func (sl *Sliced) FilterString(str string, exclude, contains, ignoreCase bool) { //types:add
	lowstr := strings.ToLower(str)
	sl.Filter(func(tsr Tensor, row int) bool {
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
func (sl *Sliced) NewTensor() Tensor {
	nt := sl.Tensor.Clone()
	if sl.Indexes == nil {
		return nt
	}
	rows := len(sl.Indexes)
	nt.SetNumRows(rows)
	_, cells := sl.Tensor.RowCellSize()
	str := sl.Tensor.IsString()
	for r := range rows {
		for c := range cells {
			if str {
				nt.SetStringRowCell(sl.StringRowCell(r, c), r, c)
			} else {
				nt.SetFloatRowCell(sl.FloatRowCell(r, c), r, c)
			}
		}
	}
	return nt
}

// Clone returns a copy of the current Sliced view with a cloned copy of
// the underlying Tensor and copy of the indexes.
func (sl *Sliced) Clone() Tensor {
	nix := &Sliced{}
	nix.Tensor = sl.Tensor.Clone()
	nix.CopyIndexes(sl)
	return nix
}

func (sl *Sliced) View() Tensor {
	nix := &Sliced{}
	nix.Tensor = sl.Tensor.View()
	nix.CopyIndexes(sl)
	return nix
}

// CloneIndexes returns a copy of the current Sliced view with new indexes,
// with a pointer to the same underlying Tensor as the source.
func (sl *Sliced) CloneIndexes() *Sliced {
	nix := &Sliced{}
	nix.Tensor = sl.Tensor
	nix.CopyIndexes(sl)
	return nix
}

// CopyIndexes copies indexes from other Sliced view.
func (sl *Sliced) CopyIndexes(oix *Sliced) {
	if oix.Indexes == nil {
		sl.Indexes = nil
	} else {
		sl.Indexes = slices.Clone(oix.Indexes)
	}
}

// AddRows adds n rows to end of underlying Tensor, and to the indexes in this view
func (sl *Sliced) AddRows(n int) { //types:add
	stidx := sl.Tensor.DimSize(0)
	sl.Tensor.SetNumRows(stidx + n)
	if sl.Indexes != nil {
		for i := stidx; i < stidx+n; i++ {
			sl.Indexes = append(sl.Indexes, i)
		}
	}
}

// InsertRows adds n rows to end of underlying Tensor, and to the indexes starting at
// given index in this view
func (sl *Sliced) InsertRows(at, n int) {
	stidx := sl.Tensor.DimSize(0)
	sl.IndexesNeeded()
	sl.Tensor.SetNumRows(stidx + n)
	nw := make([]int, n, n+len(sl.Indexes)-at)
	for i := 0; i < n; i++ {
		nw[i] = stidx + i
	}
	sl.Indexes = append(sl.Indexes[:at], append(nw, sl.Indexes[at:]...)...)
}

// DeleteRows deletes n rows of indexes starting at given index in the list of indexes
func (sl *Sliced) DeleteRows(at, n int) {
	sl.IndexesNeeded()
	sl.Indexes = append(sl.Indexes[:at], sl.Indexes[at+n:]...)
}

// Swap switches the indexes for i and j
func (sl *Sliced) Swap(i, j int) {
	if sl.Indexes == nil {
		return
	}
	sl.Indexes[i], sl.Indexes[j] = sl.Indexes[j], sl.Indexes[i]
}

///////////////////////////////////////////////
// Sliced access

/////////////////////  Floats

// Float returns the value of given index as a float64.
// The first index value is indirected through the indexes.
func (sl *Sliced) Float(i ...int) float64 {
	if sl.Indexes == nil {
		return sl.Tensor.Float(i...)
	}
	ic := slices.Clone(i)
	ic[0] = sl.Indexes[ic[0]]
	return sl.Tensor.Float(ic...)
}

// SetFloat sets the value of given index as a float64
// The first index value is indirected through the [Sliced.Indexes].
func (sl *Sliced) SetFloat(val float64, i ...int) {
	if sl.Indexes == nil {
		sl.Tensor.SetFloat(val, i...)
		return
	}
	ic := slices.Clone(i)
	ic[0] = sl.Indexes[ic[0]]
	sl.Tensor.SetFloat(val, ic...)
}

// FloatRowCell returns the value at given row and cell,
// where row is outermost dim, and cell is 1D index into remaining inner dims.
// Row is indirected through the [Sliced.Indexes].
// This is the preferred interface for all Sliced operations.
func (sl *Sliced) FloatRowCell(row, cell int) float64 {
	return sl.Tensor.FloatRowCell(sl.RowIndex(row), cell)
}

// SetFloatRowCell sets the value at given row and cell,
// where row is outermost dim, and cell is 1D index into remaining inner dims.
// Row is indirected through the [Sliced.Indexes].
// This is the preferred interface for all Sliced operations.
func (sl *Sliced) SetFloatRowCell(val float64, row, cell int) {
	sl.Tensor.SetFloatRowCell(val, sl.RowIndex(row), cell)
}

// Float1D is somewhat expensive if indexes are set, because it needs to convert
// the flat index back into a full n-dimensional index and then use that api.
func (sl *Sliced) Float1D(i int) float64 {
	if sl.Indexes == nil {
		return sl.Tensor.Float1D(i)
	}
	return sl.Float(sl.Tensor.Shape().IndexFrom1D(i)...)
}

// SetFloat1D is somewhat expensive if indexes are set, because it needs to convert
// the flat index back into a full n-dimensional index and then use that api.
func (sl *Sliced) SetFloat1D(val float64, i int) {
	if sl.Indexes == nil {
		sl.Tensor.SetFloat1D(val, i)
	}
	sl.SetFloat(val, sl.Tensor.Shape().IndexFrom1D(i)...)
}

func (sl *Sliced) FloatRow(row int) float64 {
	return sl.FloatRowCell(row, 0)
}

func (sl *Sliced) SetFloatRow(val float64, row int) {
	sl.SetFloatRowCell(val, row, 0)
}

/////////////////////  Strings

// StringValue returns the value of given index as a string.
// The first index value is indirected through the indexes.
func (sl *Sliced) StringValue(i ...int) string {
	if sl.Indexes == nil {
		return sl.Tensor.StringValue(i...)
	}
	ic := slices.Clone(i)
	ic[0] = sl.Indexes[ic[0]]
	return sl.Tensor.StringValue(ic...)
}

// SetString sets the value of given index as a string
// The first index value is indirected through the [Sliced.Indexes].
func (sl *Sliced) SetString(val string, i ...int) {
	if sl.Indexes == nil {
		sl.Tensor.SetString(val, i...)
	}
	ic := slices.Clone(i)
	ic[0] = sl.Indexes[ic[0]]
	sl.Tensor.SetString(val, ic...)
}

// StringRowCell returns the value at given row and cell,
// where row is outermost dim, and cell is 1D index into remaining inner dims.
// Row is indirected through the [Sliced.Indexes].
// This is the preferred interface for all Sliced operations.
func (sl *Sliced) StringRowCell(row, cell int) string {
	return sl.Tensor.StringRowCell(sl.RowIndex(row), cell)
}

// SetStringRowCell sets the value at given row and cell,
// where row is outermost dim, and cell is 1D index into remaining inner dims.
// Row is indirected through the [Sliced.Indexes].
// This is the preferred interface for all Sliced operations.
func (sl *Sliced) SetStringRowCell(val string, row, cell int) {
	sl.Tensor.SetStringRowCell(val, sl.RowIndex(row), cell)
}

// String1D is somewhat expensive if indexes are set, because it needs to convert
// the flat index back into a full n-dimensional index and then use that api.
func (sl *Sliced) String1D(i int) string {
	if sl.Indexes == nil {
		return sl.Tensor.String1D(i)
	}
	return sl.StringValue(sl.Tensor.Shape().IndexFrom1D(i)...)
}

// SetString1D is somewhat expensive if indexes are set, because it needs to convert
// the flat index back into a full n-dimensional index and then use that api.
func (sl *Sliced) SetString1D(val string, i int) {
	if sl.Indexes == nil {
		sl.Tensor.SetString1D(val, i)
	}
	sl.SetString(val, sl.Tensor.Shape().IndexFrom1D(i)...)
}

func (sl *Sliced) StringRow(row int) string {
	return sl.StringRowCell(row, 0)
}

func (sl *Sliced) SetStringRow(val string, row int) {
	sl.SetStringRowCell(val, row, 0)
}

/////////////////////  Ints

// Int returns the value of given index as an int.
// The first index value is indirected through the indexes.
func (sl *Sliced) Int(i ...int) int {
	if sl.Indexes == nil {
		return sl.Tensor.Int(i...)
	}
	ic := slices.Clone(i)
	ic[0] = sl.Indexes[ic[0]]
	return sl.Tensor.Int(ic...)
}

// SetInt sets the value of given index as an int
// The first index value is indirected through the [Sliced.Indexes].
func (sl *Sliced) SetInt(val int, i ...int) {
	if sl.Indexes == nil {
		sl.Tensor.SetInt(val, i...)
		return
	}
	ic := slices.Clone(i)
	ic[0] = sl.Indexes[ic[0]]
	sl.Tensor.SetInt(val, ic...)
}

// IntRowCell returns the value at given row and cell,
// where row is outermost dim, and cell is 1D index into remaining inner dims.
// Row is indirected through the [Sliced.Indexes].
// This is the preferred interface for all Sliced operations.
func (sl *Sliced) IntRowCell(row, cell int) int {
	return sl.Tensor.IntRowCell(sl.RowIndex(row), cell)
}

// SetIntRowCell sets the value at given row and cell,
// where row is outermost dim, and cell is 1D index into remaining inner dims.
// Row is indirected through the [Sliced.Indexes].
// This is the preferred interface for all Sliced operations.
func (sl *Sliced) SetIntRowCell(val int, row, cell int) {
	sl.Tensor.SetIntRowCell(val, sl.RowIndex(row), cell)
}

// Int1D is somewhat expensive if indexes are set, because it needs to convert
// the flat index back into a full n-dimensional index and then use that api.
func (sl *Sliced) Int1D(i int) int {
	if sl.Indexes == nil {
		return sl.Tensor.Int1D(i)
	}
	return sl.Int(sl.Tensor.Shape().IndexFrom1D(i)...)
}

// SetInt1D is somewhat expensive if indexes are set, because it needs to convert
// the flat index back into a full n-dimensional index and then use that api.
func (sl *Sliced) SetInt1D(val int, i int) {
	if sl.Indexes == nil {
		sl.Tensor.SetInt1D(val, i)
	}
	sl.SetInt(val, sl.Tensor.Shape().IndexFrom1D(i)...)
}

func (sl *Sliced) IntRow(row int) int {
	return sl.IntRowCell(row, 0)
}

func (sl *Sliced) SetIntRow(val int, row int) {
	sl.SetIntRowCell(val, row, 0)
}

/////////////////////  SubSpaces

// SubSpace returns a new tensor with innermost subspace at given
// offset(s) in outermost dimension(s) (len(offs) < NumDims).
// The new tensor points to the values of the this tensor (i.e., modifications
// will affect both), as its Values slice is a view onto the original (which
// is why only inner-most contiguous supsaces are supported).
// Use Clone() method to separate the two.
// Sliced version does indexed indirection of the outermost row dimension
// of the offsets.
func (sl *Sliced) SubSpace(offs ...int) Tensor {
	if len(offs) == 0 {
		return nil
	}
	offs[0] = sl.RowIndex(offs[0])
	return sl.Tensor.SubSpace(offs...)
}

// RowTensor is a convenience version of [Sliced.SubSpace] to return the
// SubSpace for the outermost row dimension, indirected through the indexes.
func (sl *Sliced) RowTensor(row int) Tensor {
	return sl.Tensor.RowTensor(sl.RowIndex(row))
}

// SetRowTensor sets the values of the SubSpace at given row to given values,
// with row indirected through the indexes.
func (sl *Sliced) SetRowTensor(val Tensor, row int) {
	sl.Tensor.SetRowTensor(val, sl.RowIndex(row))
}

// CopyFrom copies all values from other tensor into this tensor.
// Checks if source is an Sliced and copies indexes too,
// otherwise underlying tensor copies from and indexes are reset.
func (sl *Sliced) CopyFrom(from Tensor) {
	if fix, ok := from.(*Sliced); ok {
		sl.Tensor.CopyFrom(fix.Tensor)
		sl.CopyIndexes(fix)
		return
	}
	sl.Sequential()
	sl.Tensor.CopyFrom(from)
}

// AppendFrom appends all values from other tensor into this tensor.
// This invalidates the indexes which are reset.
func (sl *Sliced) AppendFrom(from Tensor) error {
	sl.Sequential()
	return sl.Tensor.AppendFrom(from)
}

// CopyCellsFrom copies given range of values from other tensor into this tensor,
// This invalidates the indexes which are reset.
func (sl *Sliced) CopyCellsFrom(from Tensor, to, start, n int) {
	sl.Sequential()
	sl.Tensor.CopyCellsFrom(from, to, start, n)
}

func (sl *Sliced) Sizeof() int64 {
	return sl.Tensor.Sizeof() // todo: could be out of sync with shape!
}

func (sl *Sliced) Bytes() []byte {
	return sl.Tensor.Bytes() // todo: could be out of sync with shape!
}

func (sl *Sliced) IsString() bool {
	return sl.Tensor.IsString()
}

func (sl *Sliced) DataType() reflect.Kind {
	return sl.Tensor.DataType()
}

func (sl *Sliced) Range() (min, max float64, minIndex, maxIndex int) {
	return sl.Tensor.Range()
}

func (sl *Sliced) SetZeros() {
	sl.Tensor.SetZeros()
}

//////////////////////////  gonum matrix api

// Dims is the gonum/mat.Matrix interface method for returning the dimensionality of the
// 2D Matrix.  Assumes Row-major ordering and logs an error if NumDims < 2.
func (sl *Sliced) Dims() (r, c int) {
	nd := sl.NumDims()
	if nd < 2 {
		log.Println("tensor Dims gonum Matrix call made on Tensor with dims < 2")
		return 0, 0
	}
	return sl.DimSize(nd - 2), sl.DimSize(nd - 1)
}

// Symmetric is the gonum/mat.Matrix interface method for returning the dimensionality of a symmetric
// 2D Matrix.
func (sl *Sliced) Symmetric() (r int) {
	nd := sl.NumDims()
	if nd < 2 {
		log.Println("tensor Symmetric gonum Matrix call made on Tensor with dims < 2")
		return 0
	}
	if sl.DimSize(nd-2) != sl.DimSize(nd-1) {
		log.Println("tensor Symmetric gonum Matrix call made on Tensor that is not symmetric")
		return 0
	}
	return sl.DimSize(nd - 1)
}

// SymmetricDim returns the number of rows/columns in the matrix.
func (sl *Sliced) SymmetricDim() int {
	nd := sl.NumDims()
	if nd < 2 {
		log.Println("tensor Symmetric gonum Matrix call made on Tensor with dims < 2")
		return 0
	}
	if sl.DimSize(nd-2) != sl.DimSize(nd-1) {
		log.Println("tensor Symmetric gonum Matrix call made on Tensor that is not symmetric")
		return 0
	}
	return sl.DimSize(nd - 1)
}

// At is the gonum/mat.Matrix interface method for returning 2D matrix element at given
// row, column index.  Assumes Row-major ordering and logs an error if NumDims < 2.
func (sl *Sliced) At(i, j int) float64 {
	nd := sl.NumDims()
	if nd < 2 {
		log.Println("tensor Dims gonum Matrix call made on Tensor with dims < 2")
		return 0
	} else if nd == 2 {
		return sl.Float(i, j)
	} else {
		nix := make([]int, nd)
		nix[nd-2] = i
		nix[nd-1] = j
		return sl.Float(nix...)
	}
}

// T is the gonum/mat.Matrix transpose method.
// It performs an implicit transpose by returning the receiver inside a Transpose.
func (sl *Sliced) T() mat.Matrix {
	return mat.Transpose{sl}
}

// check for interface impl
var _ Tensor = (*Sliced)(nil)
