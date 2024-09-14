// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package table

//go:generate core generate

import (
	"fmt"
	"math"
	"reflect"
	"slices"

	"cogentcore.org/core/base/metadata"
	"cogentcore.org/core/tensor"
)

// Table is a table of Tensor columns aligned by a common outermost row dimension.
// Use the [Table.Column] (by name) and [Table.ColumnIndex] methods to obtain a 
// [tensor.Indexed] view of the column, using the shared [Table.Indexes] of the Table.
// Thus, a coordinated sorting and filtered view of the column data is automatically 
// available for any of the tensor package functions that use [tensor.Indexed] as the one
// common data representation for all operations.
type Table struct { //types:add
	// Columns has the list of column tensor data for this table.
	// Different tables can provide different indexed views onto the same Columns.
	Columns *Columns

	// Indexes are the indexes into Tensor rows, with nil = sequential.
	// Only set if order is different from default sequential order.
	// These indexes are shared into the `tensor.Indexed` Column values
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

// NewTable returns a new Table with its own (empty) set of Columns.
// Can pass an optional name which sets metadata.
func NewTable(name ...string) *Table {
	dt := &Table{}
	dt.Columns = NewColumns()
	if len(name) > 0 {
		dt.Meta.Set("name", name[0])
	}
	return dt
}

// NewTableView returns a new Table with its own Indexed view into the
// same underlying set of Column tensor data as the source table.
// Indexes are nil in the new Table, resulting in default full sequential view.
func NewTableView(src *Table) *Table {
	dt := &Table{Columns: src.Columns}
	dt.Meta.Copy(src.Meta)
	return dt
}

// IsValidRow returns error if the row is invalid, if error checking is needed.
func (dt *Table) IsValidRow(row int) error {
	if row < 0 || row >= dt.Rows() {
		return fmt.Errorf("table.Table IsValidRow: row %d is out of valid range [0..%d]", row, dt.Rows())
	}
	return nil
}

// NumRows returns the number of rows.
func (dt *Table) NumRows() int { return dt.Columns.Rows }

// NumColumns returns the number of columns.
func (dt *Table) NumColumns() int { return dt.Columns.Len() }

// Column returns the tensor with given column name, as a [tensor.Indexed]
// with the shared [Table.Indexes] from this table. It is best practice to
// access columns by name, and direct access through Columns does not
// provide the shared table-wide Indexes.
// Returns nil if not found.
func (dt *Table) Column(name string) *tensor.Indexed {
	cl := dt.Columns.ValueByKey(name)
	if cl == nil {
		return nil
	}
	return tensor.NewIndexed(cl, dt.Indexes)
}

// ColumnTry is a version of [Table.Column] that also returns an error
// if the column name is not found, for cases when error is needed.
func (dt *Table) ColumnTry(name string) (*tensor.Indexed, error) {
	cl := dt.Column(name)
	if cl != nil {
		return cl, nil
	}
	return nil, fmt.Errorf("table.Table: Column named %q not found", name)
}

// ColumnIndex returns the tensor at the given index, as a [tensor.Indexed]
// with the shared [Table.Indexes] from this table. It is best practice to
// access columns by name using [Table.Column] method instead.
// Direct access through Columns does not provide the shared table-wide Indexes.
// Returns nil if not found.
func (dt *Table) ColumnIndex(idx int) *tensor.Indexed {
	cl := dt.Columns.Values[idx]
	return tensor.NewIndexed(cl, dt.Indexes)
}

/*
// ColumnIndexesByNames returns the indexes of the given column names.
// idxs have -1 if name not found.
func (dt *Table) ColumnIndexesByNames(names ...string) ([]int, error) {
	nc := len(names)
	if nc == 0 {
		return nil, nil
	}
	var errs []error
	cidx := make([]int, nc)
	for i, cn := range names {
		var err error
		cidx[i], err = dt.ColumnIndex(cn)
		if err != nil {
			errs = append(errs, err)
		}
	}
	return cidx, errors.Join(errs...)
}
*/

// ColumnName returns the name of given column
func (dt *Table) ColumnName(i int) string {
	return dt.Columns.Keys[i]
}

// AddColumn adds a new column to the table, of given type and column name
// (which must be unique).  The cells of this column hold a single scalar value:
// see [Table.AddColumnTensor] for n-dimensional cells.
func AddColumn[T tensor.DataTypes](dt *Table, name string) tensor.Tensor {
	rows := max(1, dt.Columns.Rows)
	tsr := tensor.New[T](rows)
	tsr.SetNames("Row")
	dt.AddColumn(name, tsr)
	return tsr
}

// InsertColumn inserts a new column to the table, of given type and column name
// (which must be unique), at given index.
// The cells of this column hold a single scalar value.
func InsertColumn[T tensor.DataTypes](dt *Table, name string, idx int) tensor.Tensor {
	rows := max(1, dt.Columns.Rows)
	tsr := tensor.New[T](rows)
	tsr.SetNames("Row")
	dt.InsertColumn(idx, name, tsr)
	return tsr
}

// AddTensorColumn adds a new n-dimensional column to the table, of given type, column name
// (which must be unique), and dimensionality of each _cell_.
// An outermost Row dimension will be added to this dimensionality to create
// the tensor column.
func AddTensorColumn[T tensor.DataTypes](dt *Table, name string, cellSizes ...int) tensor.Tensor {
	rows := max(1, dt.Columns.Rows)
	sz := append([]int{rows}, cellSizes...)
	tsr := tensor.New[T](sz...)
	tsr.SetNames("Row")
	dt.AddColumn(name, tsr)
	return tsr
}

// AddColumn adds the given tensor as a column to the table,
// returning an error and not adding if the name is not unique.
// Automatically adjusts the shape to fit the current number of rows.
func (dt *Table) AddColumn(name string, tsr tensor.Tensor) error {
	return dt.Columns.AddColumn(name, tsr)
}

// InsertColumn inserts the given tensor as a column to the table at given index,
// returning an error and not adding if the name is not unique.
// Automatically adjusts the shape to fit the current number of rows.
func (dt *Table) InsertColumn(idx int, name string, tsr tensor.Tensor) error {
	return dt.Columns.InsertColumn(idx, name, tsr)
}

// AddColumnOfType adds a new scalar column to the table, of given reflect type,
// column name (which must be unique),
// The cells of this column hold a single (scalar) value of given type.
// Supported types are string, bool (for [tensor.Bits]), float32, float64, int, int32, and byte.
func (dt *Table) AddColumnOfType(name string, typ reflect.Kind) tensor.Tensor {
	rows := max(1, dt.Columns.Rows)
	tsr := tensor.NewOfType(typ, rows)
	tsr.SetNames("Row")
	dt.AddColumn(name, tsr)
	return tsr
}

// AddTensorColumnOfType adds a new n-dimensional column to the table, of given reflect type,
// column name (which must be unique), and dimensionality of each _cell_.
// An outermost Row dimension will be added to this dimensionality to create
// the tensor column.
// Supported types are string, bool (for [tensor.Bits]), float32, float64, int, int32, and byte.
func (dt *Table) AddTensorColumnOfType(name string, typ reflect.Kind, cellSizes ...int) tensor.Tensor {
	rows := max(1, dt.Columns.Rows)
	sz := append([]int{rows}, cellSizes...)
	tsr := tensor.NewOfType(typ, sz...)
	tsr.SetNames("Row")
	dt.AddColumn(name, tsr)
	return tsr
}

// AddStringColumn adds a new String column with given name.
// The cells of this column hold a single string value.
func (dt *Table) AddStringColumn(name string) *tensor.String {
	return AddColumn[string](dt, name).(*tensor.String)
}

// AddFloat64Column adds a new float64 column with given name.
// The cells of this column hold a single scalar value.
func (dt *Table) AddFloat64Column(name string) *tensor.Float64 {
	return AddColumn[float64](dt, name).(*tensor.Float64)
}

// AddFloat64TensorColumn adds a new n-dimensional float64 column with given name
// and dimensionality of each _cell_.
// An outermost Row dimension will be added to this dimensionality to create
// the tensor column.
func (dt *Table) AddFloat64TensorColumn(name string, cellSizes ...int) *tensor.Float64 {
	return AddTensorColumn[float64](dt, name, cellSizes...).(*tensor.Float64)
}

// AddFloat32Column adds a new float32 column with given name.
// The cells of this column hold a single scalar value.
func (dt *Table) AddFloat32Column(name string) *tensor.Float32 {
	return AddColumn[float32](dt, name).(*tensor.Float32)
}

// AddFloat32TensorColumn adds a new n-dimensional float32 column with given name
// and dimensionality of each _cell_.
// An outermost Row dimension will be added to this dimensionality to create
// the tensor column.
func (dt *Table) AddFloat32TensorColumn(name string, cellSizes ...int) *tensor.Float32 {
	return AddTensorColumn[float32](dt, name, cellSizes...).(*tensor.Float32)
}

// AddIntColumn adds a new int column with given name.
// The cells of this column hold a single scalar value.
func (dt *Table) AddIntColumn(name string) *tensor.Int {
	return AddColumn[int](dt, name).(*tensor.Int)
}

// AddIntTensorColumn adds a new n-dimensional int column with given name
// and dimensionality of each _cell_.
// An outermost Row dimension will be added to this dimensionality to create
// the tensor column.
func (dt *Table) AddIntTensorColumn(name string, cellSizes ...int) *tensor.Int {
	return AddTensorColumn[int](dt, name, cellSizes...).(*tensor.Int)
}

// DeleteColumnName deletes column of given name.
// returns false if not found.
func (dt *Table) DeleteColumnName(name string) bool {
	return dt.Columns.DeleteKey(name)
}

// DeleteColumnIndex deletes column within the index range [i:j].
func (dt *Table) DeleteColumnIndex(i, j int) {
	dt.Columns.DeleteIndex(i, j)
}

// DeleteAll deletes all columns, does full reset.
func (dt *Table) DeleteAll() {
	dt.Columns.Reset()
}

// AddRows adds n rows to end of underlying Table, and to the indexes in this view
func (dt *Table) AddRows(n int) *Table { //types:add
	dt.Columns.SetNumRows(dt.Columns.Rows + n)
	return dt
}

// InsertRows adds n rows to end of underlying Table, and to the indexes starting at
// given index in this view, providing an efficient insertion operation that only
// exists in the indexed view.  To create an in-memory ordering, use [Table.NewTable].
func (dt *Table) InsertRows(at, n int) *Table {
	dt.IndexesNeeded()
	strow := dt.Columns.Rows
	stidx := len(dt.Indexes)
	dt.Columns.SetNumRows(strow + n) // adds n indexes to end of list
	// move those indexes to at:at+n in index list
	dt.Indexes = append(dt.Indexes[:at], append(dt.Indexes[stidx:], dt.Indexes[at:]...)...)
}

// SetNumRows sets the number of rows in the table, across all columns
// if rows = 0 then effective number of rows in tensors is 1, as this dim cannot be 0
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
		dt.DeleteInvalid()
	}
	return dt
}

// note: no really clean definition of CopyFrom -- no point of re-using existing
// table -- just clone it.

// Clone returns a complete copy of this table, including cloning
// the underlying Columns tensors, and the current [Table.Indexes].
// See also [Table.NewTable] to flatten the current indexes.
func (dt *Table) Clone() *Table {
	cp := &Table{}
	cp.Columns = dt.Columns.Clone()
	cp.Meta.Copy(dt.Meta)
	if dt.Indexes != nil {
		cp.Indexes = slices.Clone(dt.Indexes)
	}
	return cp
}

// AppendRows appends shared columns in both tables with input table rows
func (dt *Table) AppendRows(dt2 *Table) {
	strow := dt.Columns.Rows
	n := dt2.Columns.Rows
	dt.Columns.AppendRows(dt2.Columns)
	if dt.Indexes == nil {
		return
	}
	if rows > strow {
		for i := range rows - strow {
			dt.Indexes = append(dt.Indexes, strow+i)
		}
}

///////////////////////////////////////////////////////////////////////////
//  Cell convenience access methods

// FloatIndex returns the float64 value of cell at given column, row index
// for columns that have 1-dimensional tensors.
// Returns NaN if column is not a 1-dimensional tensor or row not valid.
func (dt *Table) FloatIndex(column, row int) float64 {
	if dt.IsValidRow(row) != nil {
		return math.NaN()
	}
	ct := dt.Columns.Values[column]
	if ct.NumDims() != 1 {
		return math.NaN()
	}
	return ct.Float1D(row)
}

// Float returns the float64 value of cell at given column (by name),
// row index for columns that have 1-dimensional tensors.
// Returns NaN if column is not a 1-dimensional tensor
// or col name not found, or row not valid.
func (dt *Table) Float(column string, row int) float64 {
	if dt.IsValidRow(row) != nil {
		return math.NaN()
	}
	ct := dt.Column(column)
	if ct == nil {
		return math.NaN()
	}
	if ct.Tensor.NumDims() != 1 {
		return math.NaN()
	}
	return ct.Tensor.Float1D(row)
}

// StringIndex returns the string value of cell at given column, row index
// for columns that have 1-dimensional tensors.
// Returns "" if column is not a 1-dimensional tensor or row not valid.
func (dt *Table) StringIndex(column, row int) string {
	if dt.IsValidRow(row) != nil {
		return ""
	}
	ct := dt.Columns.Values[column]
	if ct.NumDims() != 1 {
		return ""
	}
	return ct.String1D(row)
}

// NOTE: String conflicts with [fmt.Stringer], so we have to use StringValue

// StringValue returns the string value of cell at given column (by name), row index
// for columns that have 1-dimensional tensors.
// Returns "" if column is not a 1-dimensional tensor or row not valid.
func (dt *Table) StringValue(column string, row int) string {
	if dt.IsValidRow(row) != nil {
		return ""
	}
	ct := dt.Column(column)
	if ct == nil {
		return ""
	}
	if ct.Tensor.NumDims() != 1 {
		return ""
	}
	return ct.Tensor.String1D(row)
}

// TensorIndex returns the tensor SubSpace for given column, row index
// for columns that have higher-dimensional tensors so each row is
// represented by an n-1 dimensional tensor, with the outer dimension
// being the row number.  Returns nil if column is a 1-dimensional
// tensor or there is any error from the tensor.Tensor.SubSpace call.
func (dt *Table) TensorIndex(column, row int) tensor.Tensor {
	if dt.IsValidRow(row) != nil {
		return nil
	}
	ct := dt.Columns.Values[column]
	if ct.NumDims() == 1 {
		return nil
	}
	return ct.SubSpace(row)
}

// Tensor returns the tensor SubSpace for given column (by name), row index
// for columns that have higher-dimensional tensors so each row is
// represented by an n-1 dimensional tensor, with the outer dimension
// being the row number.  Returns nil on any error.
func (dt *Table) Tensor(column string, row int) tensor.Tensor {
	if dt.IsValidRow(row) != nil {
		return nil
	}
	ct := dt.Column(column)
	if ct == nil {
		return nil
	}
	if ct.Tensor.NumDims() == 1 {
		return nil
	}
	return ct.SubSpace(row)
}

// TensorFloat1D returns the float value of a Tensor cell's cell at given
// 1D offset within cell, for given column (by name), row index
// for columns that have higher-dimensional tensors so each row is
// represented by an n-1 dimensional tensor, with the outer dimension
// being the row number.  Returns 0 on any error.
func (dt *Table) TensorFloat1D(column string, row int, idx int) float64 {
	if dt.IsValidRow(row) != nil {
		return math.NaN()
	}
	ct := dt.Column(column)
	if ct == nil {
		return math.NaN()
	}
	if ct.Tensor.NumDims() == 1 {
		return math.NaN()
	}
	_, sz := ct.RowCellSize()
	if idx >= sz || idx < 0 {
		return math.NaN()
	}
	off := row*sz + idx
	return ct.Tensor.Float1D(off)
}

/////////////////////////////////////////////////////////////////////////////////////
//  Set

// SetFloatIndex sets the float64 value of cell at given column, row index
// for columns that have 1-dimensional tensors.
func (dt *Table) SetFloatIndex(column, row int, val float64) error {
	if err := dt.IsValidRow(row); err != nil {
		return err
	}
	ct := dt.Columns.Values[column]
	if ct.NumDims() != 1 {
		return fmt.Errorf("table.Table SetFloatIndex: Column %d is a tensor, must use SetTensorFloat1D", column)
	}
	ct.SetFloat1D(val, row)
	return nil
}

// SetFloat sets the float64 value of cell at given column (by name), row index
// for columns that have 1-dimensional tensors.
func (dt *Table) SetFloat(column string, row int, val float64) error {
	if err := dt.IsValidRow(row); err != nil {
		return err
	}
	ct, err := dt.ColumnTry(column)
	if err != nil {
		return err
	}
	if ct.Tensor.NumDims() != 1 {
		return fmt.Errorf("table.Table SetFloat: Column %s is a tensor, must use SetTensorFloat1D", column)
	}
	ct.Tensor.SetFloat1D(val, row)
	return nil
}

// SetStringIndex sets the string value of cell at given column, row index
// for columns that have 1-dimensional tensors.  Returns true if set.
func (dt *Table) SetStringIndex(column, row int, val string) error {
	if err := dt.IsValidRow(row); err != nil {
		return err
	}
	ct := dt.Columns.Values[column]
	if ct.NumDims() != 1 {
		return fmt.Errorf("table.Table SetStringIndex: Column %d is a tensor, must use SetTensorFloat1D", column)
	}
	ct.SetString1D(val, row)
	return nil
}

// SetString sets the string value of cell at given column (by name), row index
// for columns that have 1-dimensional tensors.  Returns true if set.
func (dt *Table) SetString(column string, row int, val string) error {
	if err := dt.IsValidRow(row); err != nil {
		return err
	}
	ct, err := dt.ColumnTry(column)
	if err != nil {
		return err
	}
	if ct.Tensor.NumDims() != 1 {
		return fmt.Errorf("table.Table SetString: Column %s is a tensor, must use SetTensorFloat1D", column)
	}
	ct.Tensor.SetString1D(val, row)
	return nil
}

// SetTensorIndex sets the tensor value of cell at given column, row index
// for columns that have n-dimensional tensors.  Returns true if set.
func (dt *Table) SetTensorIndex(column, row int, val tensor.Tensor) error {
	if err := dt.IsValidRow(row); err != nil {
		return err
	}
	ct := dt.Columns.Values[column]
	_, csz := ct.RowCellSize()
	st := row * csz
	sz := min(csz, val.Len())
	if ct.IsString() {
		for j := 0; j < sz; j++ {
			ct.SetString1D(val.String1D(j), st+j)
		}
	} else {
		for j := 0; j < sz; j++ {
			ct.SetFloat1D(val.Float1D(j), st+j)
		}
	}
	return nil
}

// SetTensor sets the tensor value of cell at given column (by name), row index
// for columns that have n-dimensional tensors.  Returns true if set.
func (dt *Table) SetTensor(column string, row int, val tensor.Tensor) error {
	if err := dt.IsValidRow(row); err != nil {
		return err
	}
	ci := dt.Columns.IndexByKey(column)
	if ci == nil {
		return err
	}
	return dt.SetTensorIndex(ci, row, val)
}

// SetTensorFloat1D sets the tensor cell's float cell value at given 1D index within cell,
// at given column (by name), row index for columns that have n-dimensional tensors.
// Returns true if set.
func (dt *Table) SetTensorFloat1D(column string, row int, idx int, val float64) error {
	if err := dt.IsValidRow(row); err != nil {
		return err
	}
	ct, err := dt.ColumnTry(column)
	if err != nil {
		return err
	}
	_, sz := ct.RowCellSize()
	if idx >= sz || idx < 0 {
		return fmt.Errorf("table.Table IsValidRow: index %d is out of valid range [0..%d]", idx, sz)
	}
	off := row*sz + idx
	ct.Tensor.SetFloat1D(val, off)
	return nil
}

//////////////////////////////////////////////////////////////////////////////////////
//  Copy Cell

// CopyCell copies into cell at given column, row from cell in other table.
// It is robust to differences in type; uses destination cell type.
// Returns error if column names are invalid.
func (dt *Table) CopyCell(column string, row int, cpt *Table, cpColNm string, cpRow int) error {
	ct, err := dt.ColumnTry(column)
	if err != nil {
		return err
	}
	cpct, err := cpt.ColumnTry(cpColNm)
	if err != nil {
		return err
	}
	_, sz := ct.RowCellSize()
	if sz == 1 {
		if ct.Tensor.IsString() {
			ct.SetString1D(cpct.String1D(cpRow), row)
			return nil
		}
		ct.Tensor.SetFloat1D(cpct.Float1D(cpRow), row)
		return nil
	}
	_, cpsz := cpct.RowCellSize()
	st := row * sz
	cst := cpRow * cpsz
	msz := min(sz, cpsz)
	if ct.IsString() {
		for j := 0; j < msz; j++ {
			ct.SetString1D(cpct.String1D(cst+j), st+j)
		}
	} else {
		for j := 0; j < msz; j++ {
			ct.SetFloat1D(cpct.Float1D(cst+j), st+j)
		}
	}
	return nil
}
