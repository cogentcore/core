// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package table

import (
	"math/rand"
	"slices"
	"sort"

	"cogentcore.org/core/tensor"
)

// RowIndex returns the actual index into underlying tensor row based on given
// index value.  If Indexes == nil, index is passed through.
func (dt *Table) RowIndex(idx int) int {
	if dt.Indexes == nil {
		return idx
	}
	return dt.Indexes[idx]
}

// NumRows returns the number of rows, which is the number of Indexes if present,
// else actual number of [Columns.Rows].
func (dt *Table) NumRows() int {
	if dt.Indexes == nil {
		return dt.Columns.Rows
	}
	return len(dt.Indexes)
}

// Sequential sets Indexes to nil, resulting in sequential row-wise access into tensor.
func (dt *Table) Sequential() { //types:add
	dt.Indexes = nil
}

// IndexesNeeded is called prior to an operation that needs actual indexes,
// e.g., Sort, Filter.  If Indexes == nil, they are set to all rows, otherwise
// current indexes are left as is. Use Sequential, then IndexesNeeded to ensure
// all rows are represented.
func (dt *Table) IndexesNeeded() {
	if dt.Indexes != nil {
		return
	}
	dt.Indexes = make([]int, dt.Columns.Rows)
	for i := range dt.Indexes {
		dt.Indexes[i] = i
	}
}

// IndexesFromTensor copies Indexes from the given [tensor.Rows] tensor,
// including if they are nil. This allows column-specific Sort, Filter and
// other such methods to be applied to the entire table.
func (dt *Table) IndexesFromTensor(ix *tensor.Rows) {
	dt.Indexes = ix.Indexes
}

// ValidIndexes deletes all invalid indexes from the list.
// Call this if rows (could) have been deleted from table.
func (dt *Table) ValidIndexes() {
	if dt.Columns.Rows <= 0 || dt.Indexes == nil {
		dt.Indexes = nil
		return
	}
	ni := dt.NumRows()
	for i := ni - 1; i >= 0; i-- {
		if dt.Indexes[i] >= dt.Columns.Rows {
			dt.Indexes = append(dt.Indexes[:i], dt.Indexes[i+1:]...)
		}
	}
}

// Permuted sets indexes to a permuted order -- if indexes already exist
// then existing list of indexes is permuted, otherwise a new set of
// permuted indexes are generated
func (dt *Table) Permuted() {
	if dt.Columns.Rows <= 0 {
		dt.Indexes = nil
		return
	}
	if dt.Indexes == nil {
		dt.Indexes = rand.Perm(dt.Columns.Rows)
	} else {
		rand.Shuffle(len(dt.Indexes), func(i, j int) {
			dt.Indexes[i], dt.Indexes[j] = dt.Indexes[j], dt.Indexes[i]
		})
	}
}

// SortColumn sorts the indexes into our Table according to values in
// given column, using either ascending or descending order,
// (use [tensor.Ascending] or [tensor.Descending] for self-documentation).
// Uses first cell of higher dimensional data.
// Returns error if column name not found.
func (dt *Table) SortColumn(columnName string, ascending bool) error { //types:add
	dt.IndexesNeeded()
	cl, err := dt.ColumnTry(columnName)
	if err != nil {
		return err
	}
	cl.Sort(ascending)
	dt.IndexesFromTensor(cl)
	return nil
}

// SortFunc sorts the indexes into our Table using given compare function.
// The compare function operates directly on row numbers into the Table
// as these row numbers have already been projected through the indexes.
// cmp(a, b) should return a negative number when a < b, a positive
// number when a > b and zero when a == b.
func (dt *Table) SortFunc(cmp func(dt *Table, i, j int) int) {
	dt.IndexesNeeded()
	slices.SortFunc(dt.Indexes, func(a, b int) int {
		return cmp(dt, a, b) // key point: these are already indirected through indexes!!
	})
}

// SortStableFunc stably sorts the indexes into our Table using given compare function.
// The compare function operates directly on row numbers into the Table
// as these row numbers have already been projected through the indexes.
// cmp(a, b) should return a negative number when a < b, a positive
// number when a > b and zero when a == b.
// It is *essential* that it always returns 0 when the two are equal
// for the stable function to actually work.
func (dt *Table) SortStableFunc(cmp func(dt *Table, i, j int) int) {
	dt.IndexesNeeded()
	slices.SortStableFunc(dt.Indexes, func(a, b int) int {
		return cmp(dt, a, b) // key point: these are already indirected through indexes!!
	})
}

// SortColumns sorts the indexes into our Table according to values in
// given column names, using either ascending or descending order,
// (use [tensor.Ascending] or [tensor.Descending] for self-documentation,
// and optionally using a stable sort.
// Uses first cell of higher dimensional data.
func (dt *Table) SortColumns(ascending, stable bool, columns ...string) { //types:add
	dt.SortColumnIndexes(ascending, stable, dt.ColumnIndexList(columns...)...)
}

// SortColumnIndexes sorts the indexes into our Table according to values in
// given list of column indexes, using either ascending or descending order for
// all of the columns. Uses first cell of higher dimensional data.
func (dt *Table) SortColumnIndexes(ascending, stable bool, colIndexes ...int) {
	dt.IndexesNeeded()
	sf := dt.SortFunc
	if stable {
		sf = dt.SortStableFunc
	}
	sf(func(dt *Table, i, j int) int {
		for _, ci := range colIndexes {
			cl := dt.ColumnByIndex(ci).Tensor
			if cl.IsString() {
				v := tensor.CompareAscending(cl.StringRow(i, 0), cl.StringRow(j, 0), ascending)
				if v != 0 {
					return v
				}
			} else {
				v := tensor.CompareAscending(cl.FloatRow(i, 0), cl.FloatRow(j, 0), ascending)
				if v != 0 {
					return v
				}
			}
		}
		return 0
	})
}

// SortIndexes sorts the indexes into our Table directly in
// numerical order, producing the native ordering, while preserving
// any filtering that might have occurred.
func (dt *Table) SortIndexes() {
	if dt.Indexes == nil {
		return
	}
	sort.Ints(dt.Indexes)
}

// FilterFunc is a function used for filtering that returns
// true if Table row should be included in the current filtered
// view of the table, and false if it should be removed.
type FilterFunc func(dt *Table, row int) bool

// Filter filters the indexes into our Table using given Filter function.
// The Filter function operates directly on row numbers into the Table
// as these row numbers have already been projected through the indexes.
func (dt *Table) Filter(filterer func(dt *Table, row int) bool) {
	dt.IndexesNeeded()
	sz := len(dt.Indexes)
	for i := sz - 1; i >= 0; i-- { // always go in reverse for filtering
		if !filterer(dt, dt.Indexes[i]) { // delete
			dt.Indexes = append(dt.Indexes[:i], dt.Indexes[i+1:]...)
		}
	}
}

// FilterString filters the indexes using string values in column compared to given
// string. Includes rows with matching values unless the Exclude option is set.
// If Contains option is set, it only checks if row contains string;
// if IgnoreCase, ignores case, otherwise filtering is case sensitive.
// Uses first cell from higher dimensions.
// Returns error if column name not found.
func (dt *Table) FilterString(columnName string, str string, opts tensor.FilterOptions) error { //types:add
	dt.IndexesNeeded()
	cl, err := dt.ColumnTry(columnName)
	if err != nil {
		return err
	}
	cl.FilterString(str, opts)
	dt.IndexesFromTensor(cl)
	return nil
}

// New returns a new table with column data organized according to
// the indexes.  If Indexes are nil, a clone of the current tensor is returned
// but this function is only sensible if there is an indexed view in place.
func (dt *Table) New() *Table {
	if dt.Indexes == nil {
		return dt.Clone()
	}
	rows := len(dt.Indexes)
	nt := dt.Clone()
	nt.Indexes = nil
	nt.SetNumRows(rows)
	if rows == 0 {
		return nt
	}
	for ci, cl := range nt.Columns.Values {
		scl := dt.Columns.Values[ci]
		_, csz := cl.Shape().RowCellSize()
		for i, srw := range dt.Indexes {
			cl.CopyCellsFrom(scl, i*csz, srw*csz, csz)
		}
	}
	return nt
}

// DeleteRows deletes n rows of Indexes starting at given index in the list of indexes.
// This does not affect the underlying tensor data; To create an actual in-memory
// ordering with rows deleted, use [Table.New].
func (dt *Table) DeleteRows(at, n int) {
	dt.IndexesNeeded()
	dt.Indexes = append(dt.Indexes[:at], dt.Indexes[at+n:]...)
}

// Swap switches the indexes for i and j
func (dt *Table) Swap(i, j int) {
	dt.Indexes[i], dt.Indexes[j] = dt.Indexes[j], dt.Indexes[i]
}
