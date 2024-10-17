// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package table

//go:generate core generate

import (
	"fmt"
	"reflect"
	"slices"

	"cogentcore.org/core/base/metadata"
	"cogentcore.org/core/tensor"
)

// Table is a table of Tensor columns aligned by a common outermost row dimension.
// Use the [Table.Column] (by name) and [Table.ColumnIndex] methods to obtain a
// [tensor.Rows] view of the column, using the shared [Table.Indexes] of the Table.
// Thus, a coordinated sorting and filtered view of the column data is automatically
// available for any of the tensor package functions that use [tensor.Tensor] as the one
// common data representation for all operations.
// Tensor Columns are always raw value types and support SubSpace operations on cells.
type Table struct { //types:add
	// Columns has the list of column tensor data for this table.
	// Different tables can provide different indexed views onto the same Columns.
	Columns *Columns

	// Indexes are the indexes into Tensor rows, with nil = sequential.
	// Only set if order is different from default sequential order.
	// These indexes are shared into the `tensor.Rows` Column values
	// to provide a coordinated indexed view into the underlying data.
	Indexes []int

	// Meta is misc metadata for the table. Use lower-case key names
	// following the struct tag convention:
	//	- name string = name of table
	//	- doc string = documentation, description
	//	- read-only bool = gui is read-only
	//	- precision int = n for precision to write out floats in csv.
	Meta metadata.Data
}

// New returns a new Table with its own (empty) set of Columns.
// Can pass an optional name which calls metadata SetName.
func New(name ...string) *Table {
	dt := &Table{}
	dt.Columns = NewColumns()
	if len(name) > 0 {
		dt.Meta.SetName(name[0])
	}
	return dt
}

// NewView returns a new Table with its own Rows view into the
// same underlying set of Column tensor data as the source table.
// Indexes are copied from the existing table -- use Sequential
// to reset to full sequential view.
func NewView(src *Table) *Table {
	dt := &Table{Columns: src.Columns}
	if src.Indexes != nil {
		dt.Indexes = slices.Clone(src.Indexes)
	}
	dt.Meta.Copy(src.Meta)
	return dt
}

// IsValidRow returns error if the row is invalid, if error checking is needed.
func (dt *Table) IsValidRow(row int) error {
	if row < 0 || row >= dt.NumRows() {
		return fmt.Errorf("table.Table IsValidRow: row %d is out of valid range [0..%d]", row, dt.NumRows())
	}
	return nil
}

// NumColumns returns the number of columns.
func (dt *Table) NumColumns() int { return dt.Columns.Len() }

// Column returns the tensor with given column name, as a [tensor.Rows]
// with the shared [Table.Indexes] from this table. It is best practice to
// access columns by name, and direct access through [Table.Columns] does not
// provide the shared table-wide Indexes.
// Returns nil if not found.
func (dt *Table) Column(name string) *tensor.Rows {
	cl := dt.Columns.At(name)
	if cl == nil {
		return nil
	}
	return tensor.NewRows(cl, dt.Indexes...)
}

// ColumnTry is a version of [Table.Column] that also returns an error
// if the column name is not found, for cases when error is needed.
func (dt *Table) ColumnTry(name string) (*tensor.Rows, error) {
	cl := dt.Column(name)
	if cl != nil {
		return cl, nil
	}
	return nil, fmt.Errorf("table.Table: Column named %q not found", name)
}

// ColumnIndex returns the tensor at the given column index,
// as a [tensor.Rows] with the shared [Table.Indexes] from this table.
// It is best practice to instead access columns by name using [Table.Column].
// Direct access through [Table.Columns} does not provide the shared table-wide Indexes.
// Will panic if out of range.
func (dt *Table) ColumnByIndex(idx int) *tensor.Rows {
	cl := dt.Columns.Values[idx]
	return tensor.NewRows(cl, dt.Indexes...)
}

// ColumnList returns a list of tensors with given column names,
// as [tensor.Rows] with the shared [Table.Indexes] from this table.
func (dt *Table) ColumnList(names ...string) []tensor.Tensor {
	list := make([]tensor.Tensor, 0, len(names))
	for _, nm := range names {
		cl := dt.Column(nm)
		if cl != nil {
			list = append(list, cl)
		}
	}
	return list
}

// ColumnName returns the name of given column.
func (dt *Table) ColumnName(i int) string {
	return dt.Columns.Keys[i]
}

// ColumnIndex returns the index for given column name.
func (dt *Table) ColumnIndex(name string) int {
	return dt.Columns.IndexByKey(name)
}

// ColumnIndexList returns a list of indexes to columns of given names.
func (dt *Table) ColumnIndexList(names ...string) []int {
	list := make([]int, 0, len(names))
	for _, nm := range names {
		ci := dt.ColumnIndex(nm)
		if ci >= 0 {
			list = append(list, ci)
		}
	}
	return list
}

// AddColumn adds a new column to the table, of given type and column name
// (which must be unique). If no cellSizes are specified, it holds scalar values,
// otherwise the cells are n-dimensional tensors of given size.
func AddColumn[T tensor.DataTypes](dt *Table, name string, cellSizes ...int) tensor.Tensor {
	rows := dt.Columns.Rows
	sz := append([]int{rows}, cellSizes...)
	tsr := tensor.New[T](sz...)
	// tsr.SetNames("Row")
	dt.AddColumn(name, tsr)
	return tsr
}

// InsertColumn inserts a new column to the table, of given type and column name
// (which must be unique), at given index.
// If no cellSizes are specified, it holds scalar values,
// otherwise the cells are n-dimensional tensors of given size.
func InsertColumn[T tensor.DataTypes](dt *Table, name string, idx int, cellSizes ...int) tensor.Tensor {
	rows := dt.Columns.Rows
	sz := append([]int{rows}, cellSizes...)
	tsr := tensor.New[T](sz...)
	// tsr.SetNames("Row")
	dt.InsertColumn(idx, name, tsr)
	return tsr
}

// AddColumn adds the given [tensor.Values] as a column to the table,
// returning an error and not adding if the name is not unique.
// Automatically adjusts the shape to fit the current number of rows.
func (dt *Table) AddColumn(name string, tsr tensor.Values) error {
	return dt.Columns.AddColumn(name, tsr)
}

// InsertColumn inserts the given [tensor.Values] as a column to the table at given index,
// returning an error and not adding if the name is not unique.
// Automatically adjusts the shape to fit the current number of rows.
func (dt *Table) InsertColumn(idx int, name string, tsr tensor.Values) error {
	return dt.Columns.InsertColumn(idx, name, tsr)
}

// AddColumnOfType adds a new scalar column to the table, of given reflect type,
// column name (which must be unique),
// If no cellSizes are specified, it holds scalar values,
// otherwise the cells are n-dimensional tensors of given size.
// Supported types include string, bool (for [tensor.Bool]), float32, float64, int, int32, and byte.
func (dt *Table) AddColumnOfType(name string, typ reflect.Kind, cellSizes ...int) tensor.Tensor {
	rows := dt.Columns.Rows
	sz := append([]int{rows}, cellSizes...)
	tsr := tensor.NewOfType(typ, sz...)
	// tsr.SetNames("Row")
	dt.AddColumn(name, tsr)
	return tsr
}

// AddStringColumn adds a new String column with given name.
// If no cellSizes are specified, it holds scalar values,
// otherwise the cells are n-dimensional tensors of given size.
func (dt *Table) AddStringColumn(name string, cellSizes ...int) *tensor.String {
	return AddColumn[string](dt, name, cellSizes...).(*tensor.String)
}

// AddFloat64Column adds a new float64 column with given name.
// If no cellSizes are specified, it holds scalar values,
// otherwise the cells are n-dimensional tensors of given size.
func (dt *Table) AddFloat64Column(name string, cellSizes ...int) *tensor.Float64 {
	return AddColumn[float64](dt, name, cellSizes...).(*tensor.Float64)
}

// AddFloat32Column adds a new float32 column with given name.
// If no cellSizes are specified, it holds scalar values,
// otherwise the cells are n-dimensional tensors of given size.
func (dt *Table) AddFloat32Column(name string, cellSizes ...int) *tensor.Float32 {
	return AddColumn[float32](dt, name, cellSizes...).(*tensor.Float32)
}

// AddIntColumn adds a new int column with given name.
// If no cellSizes are specified, it holds scalar values,
// otherwise the cells are n-dimensional tensors of given size.
func (dt *Table) AddIntColumn(name string, cellSizes ...int) *tensor.Int {
	return AddColumn[int](dt, name, cellSizes...).(*tensor.Int)
}

// DeleteColumnName deletes column of given name.
// returns false if not found.
func (dt *Table) DeleteColumnName(name string) bool {
	return dt.Columns.DeleteByKey(name)
}

// DeleteColumnIndex deletes column within the index range [i:j].
func (dt *Table) DeleteColumnByIndex(i, j int) {
	dt.Columns.DeleteByIndex(i, j)
}

// DeleteAll deletes all columns, does full reset.
func (dt *Table) DeleteAll() {
	dt.Indexes = nil
	dt.Columns.Reset()
}

// AddRows adds n rows to end of underlying Table, and to the indexes in this view.
func (dt *Table) AddRows(n int) *Table { //types:add
	return dt.SetNumRows(dt.Columns.Rows + n)
}

// InsertRows adds n rows to end of underlying Table, and to the indexes starting at
// given index in this view, providing an efficient insertion operation that only
// exists in the indexed view. To create an in-memory ordering, use [Table.New].
func (dt *Table) InsertRows(at, n int) *Table {
	dt.IndexesNeeded()
	strow := dt.Columns.Rows
	stidx := len(dt.Indexes)
	dt.SetNumRows(strow + n) // adds n indexes to end of list
	// move those indexes to at:at+n in index list
	dt.Indexes = append(dt.Indexes[:at], append(dt.Indexes[stidx:], dt.Indexes[at:]...)...)
	dt.Indexes = dt.Indexes[:strow+n]
	return dt
}

// SetNumRows sets the number of rows in the table, across all columns.
// If rows = 0 then effective number of rows in tensors is 1, as this dim cannot be 0.
// If indexes are in place and rows are added, indexes for the new rows are added.
func (dt *Table) SetNumRows(rows int) *Table { //types:add
	strow := dt.Columns.Rows
	dt.Columns.SetNumRows(rows)
	if dt.Indexes == nil {
		return dt
	}
	if rows > strow {
		for i := range rows - strow {
			dt.Indexes = append(dt.Indexes, strow+i)
		}
	} else {
		dt.ValidIndexes()
	}
	return dt
}

// SetNumRowsToMax gets the current max number of rows across all the column tensors,
// and sets the number of rows to that. This will automatically pad shorter columns
// so they all have the same number of rows. If a table has columns that are not fully
// under its own control, they can change size, so this reestablishes
// a common row dimension.
func (dt *Table) SetNumRowsToMax() {
	var maxRow int
	for _, tsr := range dt.Columns.Values {
		maxRow = max(maxRow, tsr.DimSize(0))
	}
	dt.SetNumRows(maxRow)
}

// note: no really clean definition of CopyFrom -- no point of re-using existing
// table -- just clone it.

// Clone returns a complete copy of this table, including cloning
// the underlying Columns tensors, and the current [Table.Indexes].
// See also [Table.New] to flatten the current indexes.
func (dt *Table) Clone() *Table {
	cp := &Table{}
	cp.Columns = dt.Columns.Clone()
	cp.Meta.Copy(dt.Meta)
	if dt.Indexes != nil {
		cp.Indexes = slices.Clone(dt.Indexes)
	}
	return cp
}

// AppendRows appends shared columns in both tables with input table rows.
func (dt *Table) AppendRows(dt2 *Table) {
	strow := dt.Columns.Rows
	n := dt2.Columns.Rows
	dt.Columns.AppendRows(dt2.Columns)
	if dt.Indexes == nil {
		return
	}
	for i := range n {
		dt.Indexes = append(dt.Indexes, strow+i)
	}
}
