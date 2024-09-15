// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package table

import (
	"math/rand"
	"slices"
	"sort"
	"strings"

	"cogentcore.org/core/tensor"
)

// Index returns the actual index into underlying tensor row based on given
// index value.  If Indexes == nil, index is passed through.
func (dt *Table) Index(idx int) int {
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
func (dt *Table) IndexesNeeded() { //types:add
	if dt.Indexes != nil {
		return
	}
	dt.Indexes = make([]int, dt.Columns.Rows)
	for i := range dt.Indexes {
		dt.Indexes[i] = i
	}
}

// DeleteInvalid deletes all invalid indexes from the list.
// Call this if rows (could) have been deleted from table.
func (dt *Table) DeleteInvalid() {
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

// SortIndexes sorts the indexes into our Table directly in
// numerical order, producing the native ordering, while preserving
// any filtering that might have occurred.
func (dt *Table) SortIndexes() {
	if dt.Indexes == nil {
		return
	}
	sort.Ints(dt.Indexes)
}

// SortColumns sorts the indexes into our Table according to values in
// given column names, using either ascending or descending order,
// and optionally using a stable sort.
// Only valid for 1-dimensional columns.
// Returns error if column name not found.
func (dt *Table) SortColumns(ascending, stable bool, columns ...string) {
	nc := len(columns)
	cis := make([]int, 0, nc)
	for _, cn := range columns {
		ci := dt.Columns.IndexByKey(cn)
		if ci >= 0 {
			cis = append(cis, ci)
		}
	}
	dt.SortColumnIndexes(ascending, stable, cis...)
}

// SortColumnIndexes sorts the indexes into our Table according to values in
// given list of column indexes, using either ascending or descending order for
// all of the columns.  Only valid for 1-dimensional columns.
func (dt *Table) SortColumnIndexes(ascending, stable bool, colIndexes ...int) {
	dt.IndexesNeeded()
	sf := dt.SortFunc
	if stable {
		sf = dt.SortStableFunc
	}
	sf(func(dt *Table, i, j int) int {
		for _, ci := range colIndexes {
			cl := dt.ColumnIndex(ci).Tensor
			if cl.IsString() {
				v := tensor.CompareAscending(cl.StringRowCell(i, 0), cl.StringRowCell(j, 0), ascending)
				if v != 0 {
					return v
				}
			} else {
				v := tensor.CompareAscending(cl.FloatRowCell(i, 0), cl.FloatRowCell(j, 0), ascending)
				if v != 0 {
					return v
				}
			}
		}
		return 0
	})
}

/////////////////////////////////////////////////////////////////////////
//  Stable sorts -- sometimes essential..

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

// SortStableColumn sorts the indexes into our Table according to values in
// given column name, using either ascending or descending order.
// Only valid for 1-dimensional columns.
// Returns error if column name not found.
func (dt *Table) SortStableColumn(column string, ascending bool) error {
	dt.IndexesNeeded()
	cl, err := dt.ColumnTry(column) // has our indexes
	if err != nil {
		return err
	}
	return cl.SortStable(ascending)
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

// FilterColumn filters the indexes into our Table according to values in
// given column name, using string representation of column values.
// Includes rows with matching values unless exclude is set.
// If contains, only checks if row contains string; if ignoreCase, ignores case.
// Use named args for greater clarity.
// Only valid for 1-dimensional columns.
// Returns error if column name not found.
func (dt *Table) FilterColumn(column string, str string, exclude, contains, ignoreCase bool) error { //types:add
	col, err := dt.ColumnTry(column)
	if err != nil {
		return err
	}
	lowstr := strings.ToLower(str)
	dt.Filter(func(dt *Table, row int) bool {
		val := col.StringRowCell(row, 0)
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
	return nil
}

// NewTable returns a new table with column data organized according to
// the indexes.  If Indexes are nil, a clone of the current tensor is returned
// but this function is only sensible if there is an indexed view in place.
func (dt *Table) NewTable() *Table {
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
		_, csz := cl.RowCellSize()
		for i, srw := range dt.Indexes {
			cl.CopyCellsFrom(scl, i*csz, srw*csz, csz)
		}
	}
	return nt
}

// DeleteRows deletes n rows of Indexes starting at given index in the list of indexes.
// This does not affect the underlying tensor data; To create an actual in-memory
// ordering with rows deleted, use [Table.NewTable].
func (dt *Table) DeleteRows(at, n int) {
	dt.IndexesNeeded()
	dt.Indexes = append(dt.Indexes[:at], dt.Indexes[at+n:]...)
}

// Named arg values for Contains, IgnoreCase
const (
	// Contains means the string only needs to contain the target string (see Equals)
	Contains bool = true
	// Equals means the string must equal the target string (see Contains)
	Equals = false
	// IgnoreCase means that differences in case are ignored in comparing strings
	IgnoreCase = true
	// UseCase means that case matters when comparing strings
	UseCase = false
)

// RowsByString returns the list of row _indexes_ (not necessarily underlying row numbers,
// if Indexes are in place) whose row in the table has given string value in given column name.
// The results can be used as row indexes to Indexed tensor column data.
// If contains, only checks if row contains string; if ignoreCase, ignores case.
// Use the named const args [Contains], [Equals], [IgnoreCase], [UseCase] for greater clarity.
func (dt *Table) RowsByString(colname string, str string, contains, ignoreCase bool) []int {
	col := dt.Column(colname)
	if col == nil {
		return nil
	}
	lowstr := strings.ToLower(str)
	var indexes []int
	rows := dt.NumRows()
	for idx := range rows {
		srw := dt.Index(idx)
		val := col.Tensor.StringRowCell(srw, 0)
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
		if has {
			indexes = append(indexes, idx)
		}
	}
	return indexes
}

// Swap switches the indexes for i and j
func (dt *Table) Swap(i, j int) {
	dt.Indexes[i], dt.Indexes[j] = dt.Indexes[j], dt.Indexes[i]
}
