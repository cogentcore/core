// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package table

//go:generate core generate

import (
	"errors"
	"fmt"
	"log/slog"
	"math"
	"reflect"
	"slices"
	"strings"

	"cogentcore.org/core/tensor"
)

// Table is a table of data, with columns of tensors,
// each with the same number of Rows (outer-most dimension).
type Table struct { //types:add

	// columns of data, as tensor.Tensor tensors
	Columns []tensor.Tensor `display:"no-inline"`

	// the names of the columns
	ColumnNames []string

	// number of rows, which is enforced to be the size of the outer-most dimension of the column tensors
	Rows int `edit:"-"`

	// the map of column names to column numbers
	ColumnNameMap map[string]int `display:"-"`

	// misc meta data for the table.  We use lower-case key names following the struct tag convention:  name = name of table; desc = description; read-only = gui is read-only; precision = n for precision to write out floats in csv.  For Column-specific data, we look for ColumnName: prefix, specifically ColumnName:desc = description of the column contents, which is shown as tooltip in the tensorcore.Table, and :width for width of a column
	MetaData map[string]string
}

func NewTable(name ...string) *Table {
	dt := &Table{}
	if len(name) > 0 {
		dt.SetMetaData("name", name[0])
	}
	return dt
}

// IsValidRow returns error if the row is invalid
func (dt *Table) IsValidRow(row int) error {
	if row < 0 || row >= dt.Rows {
		return fmt.Errorf("table.Table IsValidRow: row %d is out of valid range [0..%d]", row, dt.Rows)
	}
	return nil
}

// NumRows returns the number of rows
func (dt *Table) NumRows() int { return dt.Rows }

// NumColumns returns the number of columns
func (dt *Table) NumColumns() int { return len(dt.Columns) }

// Column returns the tensor at given column index
func (dt *Table) Column(i int) tensor.Tensor { return dt.Columns[i] }

// ColumnByName returns the tensor at given column name, with error message if not found.
// Returns nil if not found
func (dt *Table) ColumnByName(name string) (tensor.Tensor, error) {
	i, ok := dt.ColumnNameMap[name]
	if !ok {
		return nil, fmt.Errorf("table.Table ColumnByNameTry: column named: %v not found", name)
	}
	return dt.Columns[i], nil
}

// ColumnIndex returns the index of the given column name,
// along with an error if not found.
func (dt *Table) ColumnIndex(name string) (int, error) {
	i, ok := dt.ColumnNameMap[name]
	if !ok {
		return 0, fmt.Errorf("table.Table ColumnIndex: column named: %v not found", name)
	}
	return i, nil
}

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

// ColumnName returns the name of given column
func (dt *Table) ColumnName(i int) string {
	return dt.ColumnNames[i]
}

// UpdateColumnNameMap updates the column name map, returning an error
// if any of the column names are duplicates.
func (dt *Table) UpdateColumnNameMap() error {
	nc := dt.NumColumns()
	dt.ColumnNameMap = make(map[string]int, nc)
	var errs []error
	for i, nm := range dt.ColumnNames {
		if _, has := dt.ColumnNameMap[nm]; has {
			err := fmt.Errorf("table.Table duplicate column name: %s", nm)
			slog.Warn(err.Error())
			errs = append(errs, err)
		} else {
			dt.ColumnNameMap[nm] = i
		}
	}
	if len(errs) > 0 {
		return errors.Join(errs...)
	}
	return nil
}

// AddColumn adds a new column to the table, of given type and column name
// (which must be unique).  The cells of this column hold a single scalar value:
// see AddColumnTensor for n-dimensional cells.
func AddColumn[T string | bool | float32 | float64 | int | int32 | byte](dt *Table, name string) tensor.Tensor {
	rows := max(1, dt.Rows)
	tsr := tensor.New[T]([]int{rows}, "Row")
	dt.AddColumn(tsr, name)
	return tsr
}

// InsertColumn inserts a new column to the table, of given type and column name
// (which must be unique), at given index.
// The cells of this column hold a single scalar value.
func InsertColumn[T string | bool | float32 | float64 | int | int32 | byte](dt *Table, name string, idx int) tensor.Tensor {
	rows := max(1, dt.Rows)
	tsr := tensor.New[T]([]int{rows}, "Row")
	dt.InsertColumn(tsr, name, idx)
	return tsr
}

// AddTensorColumn adds a new n-dimensional column to the table, of given type, column name
// (which must be unique), and dimensionality of each _cell_.
// An outer-most Row dimension will be added to this dimensionality to create
// the tensor column.
func AddTensorColumn[T string | bool | float32 | float64 | int | int32 | byte](dt *Table, name string, cellSizes []int, dimNames ...string) tensor.Tensor {
	rows := max(1, dt.Rows)
	sz := append([]int{rows}, cellSizes...)
	nms := append([]string{"Row"}, dimNames...)
	tsr := tensor.New[T](sz, nms...)
	dt.AddColumn(tsr, name)
	return tsr
}

// AddColumn adds the given tensor as a column to the table,
// returning an error and not adding if the name is not unique.
// Automatically adjusts the shape to fit the current number of rows.
func (dt *Table) AddColumn(tsr tensor.Tensor, name string) error {
	dt.ColumnNames = append(dt.ColumnNames, name)
	err := dt.UpdateColumnNameMap()
	if err != nil {
		dt.ColumnNames = dt.ColumnNames[:len(dt.ColumnNames)-1]
		return err
	}
	dt.Columns = append(dt.Columns, tsr)
	rows := max(1, dt.Rows)
	tsr.SetNumRows(rows)
	return nil
}

// InsertColumn inserts the given tensor as a column to the table at given index,
// returning an error and not adding if the name is not unique.
// Automatically adjusts the shape to fit the current number of rows.
func (dt *Table) InsertColumn(tsr tensor.Tensor, name string, idx int) error {
	if _, has := dt.ColumnNameMap[name]; has {
		err := fmt.Errorf("table.Table duplicate column name: %s", name)
		slog.Warn(err.Error())
		return err
	}
	dt.ColumnNames = slices.Insert(dt.ColumnNames, idx, name)
	dt.UpdateColumnNameMap()
	dt.Columns = slices.Insert(dt.Columns, idx, tsr)
	rows := max(1, dt.Rows)
	tsr.SetNumRows(rows)
	return nil
}

// AddColumnOfType adds a new scalar column to the table, of given reflect type,
// column name (which must be unique),
// The cells of this column hold a single (scalar) value of given type.
// Supported types are string, bool (for [tensor.Bits]), float32, float64, int, int32, and byte.
func (dt *Table) AddColumnOfType(typ reflect.Kind, name string) tensor.Tensor {
	rows := max(1, dt.Rows)
	tsr := tensor.NewOfType(typ, []int{rows}, "Row")
	dt.AddColumn(tsr, name)
	return tsr
}

// AddTensorColumnOfType adds a new n-dimensional column to the table, of given reflect type,
// column name (which must be unique), and dimensionality of each _cell_.
// An outer-most Row dimension will be added to this dimensionality to create
// the tensor column.
// Supported types are string, bool (for [tensor.Bits]), float32, float64, int, int32, and byte.
func (dt *Table) AddTensorColumnOfType(typ reflect.Kind, name string, cellSizes []int, dimNames ...string) tensor.Tensor {
	rows := max(1, dt.Rows)
	sz := append([]int{rows}, cellSizes...)
	nms := append([]string{"Row"}, dimNames...)
	tsr := tensor.NewOfType(typ, sz, nms...)
	dt.AddColumn(tsr, name)
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
// An outer-most Row dimension will be added to this dimensionality to create
// the tensor column.
func (dt *Table) AddFloat64TensorColumn(name string, cellSizes []int, dimNames ...string) *tensor.Float64 {
	return AddTensorColumn[float64](dt, name, cellSizes, dimNames...).(*tensor.Float64)
}

// AddFloat32Column adds a new float32 column with given name.
// The cells of this column hold a single scalar value.
func (dt *Table) AddFloat32Column(name string) *tensor.Float32 {
	return AddColumn[float32](dt, name).(*tensor.Float32)
}

// AddFloat32TensorColumn adds a new n-dimensional float32 column with given name
// and dimensionality of each _cell_.
// An outer-most Row dimension will be added to this dimensionality to create
// the tensor column.
func (dt *Table) AddFloat32TensorColumn(name string, cellSizes []int, dimNames ...string) *tensor.Float32 {
	return AddTensorColumn[float32](dt, name, cellSizes, dimNames...).(*tensor.Float32)
}

// AddIntColumn adds a new int column with given name.
// The cells of this column hold a single scalar value.
func (dt *Table) AddIntColumn(name string) *tensor.Int {
	return AddColumn[int](dt, name).(*tensor.Int)
}

// AddIntTensorColumn adds a new n-dimensional int column with given name
// and dimensionality of each _cell_.
// An outer-most Row dimension will be added to this dimensionality to create
// the tensor column.
func (dt *Table) AddIntTensorColumn(name string, cellSizes []int, dimNames ...string) *tensor.Int {
	return AddTensorColumn[int](dt, name, cellSizes, dimNames...).(*tensor.Int)
}

// DeleteColumnName deletes column of given name.
// returns error if not found.
func (dt *Table) DeleteColumnName(name string) error {
	ci, err := dt.ColumnIndex(name)
	if err != nil {
		return err
	}
	dt.DeleteColumnIndex(ci)
	return nil
}

// DeleteColumnIndex deletes column of given index
func (dt *Table) DeleteColumnIndex(idx int) {
	dt.Columns = append(dt.Columns[:idx], dt.Columns[idx+1:]...)
	dt.ColumnNames = append(dt.ColumnNames[:idx], dt.ColumnNames[idx+1:]...)
	dt.UpdateColumnNameMap()
}

// DeleteAll deletes all columns -- full reset
func (dt *Table) DeleteAll() {
	dt.Columns = nil
	dt.ColumnNames = nil
	dt.Rows = 0
	dt.ColumnNameMap = nil
}

// AddRows adds n rows to each of the columns
func (dt *Table) AddRows(n int) { //types:add
	dt.SetNumRows(dt.Rows + n)
}

// SetNumRows sets the number of rows in the table, across all columns
// if rows = 0 then effective number of rows in tensors is 1, as this dim cannot be 0
func (dt *Table) SetNumRows(rows int) *Table { //types:add
	dt.Rows = rows // can be 0
	rows = max(1, rows)
	for _, tsr := range dt.Columns {
		tsr.SetNumRows(rows)
	}
	return dt
}

// note: no really clean definition of CopyFrom -- no point of re-using existing
// table -- just clone it.

// Clone returns a complete copy of this table
func (dt *Table) Clone() *Table {
	cp := NewTable().SetNumRows(dt.Rows)
	cp.CopyMetaDataFrom(dt)
	for i, cl := range dt.Columns {
		cp.AddColumn(cl.Clone(), dt.ColumnNames[i])
	}
	return cp
}

// AppendRows appends shared columns in both tables with input table rows
func (dt *Table) AppendRows(dt2 *Table) {
	shared := false
	strow := dt.Rows
	for iCol := range dt.Columns {
		colName := dt.ColumnName(iCol)
		_, err := dt2.ColumnIndex(colName)
		if err != nil {
			continue
		}
		if !shared {
			shared = true
			dt.AddRows(dt2.Rows)
		}
		for iRow := 0; iRow < dt2.Rows; iRow++ {
			dt.CopyCell(colName, iRow+strow, dt2, colName, iRow)
		}
	}
}

// SetMetaData sets given meta-data key to given value, safely creating the
// map if not yet initialized.  Standard Keys are:
// * name -- name of table
// * desc -- description of table
// * read-only  -- makes gui read-only (inactive edits) for tensorcore.Table
// * ColumnName:* -- prefix for all column-specific meta-data
//   - desc -- description of column
func (dt *Table) SetMetaData(key, val string) {
	if dt.MetaData == nil {
		dt.MetaData = make(map[string]string)
	}
	dt.MetaData[key] = val
}

// CopyMetaDataFrom copies meta data from other table
func (dt *Table) CopyMetaDataFrom(cp *Table) {
	nm := len(cp.MetaData)
	if nm == 0 {
		return
	}
	if dt.MetaData == nil {
		dt.MetaData = make(map[string]string, nm)
	}
	for k, v := range cp.MetaData {
		dt.MetaData[k] = v
	}
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

// RowsByStringIndex returns the list of rows that have given
// string value in given column index.
// if contains, only checks if row contains string; if ignoreCase, ignores case.
// Use named args for greater clarity.
func (dt *Table) RowsByStringIndex(column int, str string, contains, ignoreCase bool) []int {
	col := dt.Columns[column]
	lowstr := strings.ToLower(str)
	var idxs []int
	for i := 0; i < dt.Rows; i++ {
		val := col.String1D(i)
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
			idxs = append(idxs, i)
		}
	}
	return idxs
}

// RowsByString returns the list of rows that have given
// string value in given column name.  returns nil & error if name invalid.
// if contains, only checks if row contains string; if ignoreCase, ignores case.
// Use named args for greater clarity.
func (dt *Table) RowsByString(column string, str string, contains, ignoreCase bool) ([]int, error) {
	ci, err := dt.ColumnIndex(column)
	if err != nil {
		return nil, err
	}
	return dt.RowsByStringIndex(ci, str, contains, ignoreCase), nil
}

//////////////////////////////////////////////////////////////////////////////////////
//  Cell convenience access methods

// FloatIndex returns the float64 value of cell at given column, row index
// for columns that have 1-dimensional tensors.
// Returns NaN if column is not a 1-dimensional tensor or row not valid.
func (dt *Table) FloatIndex(column, row int) float64 {
	if dt.IsValidRow(row) != nil {
		return math.NaN()
	}
	ct := dt.Columns[column]
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
	ct, err := dt.ColumnByName(column)
	if err != nil {
		return math.NaN()
	}
	if ct.NumDims() != 1 {
		return math.NaN()
	}
	return ct.Float1D(row)
}

// StringIndex returns the string value of cell at given column, row index
// for columns that have 1-dimensional tensors.
// Returns "" if column is not a 1-dimensional tensor or row not valid.
func (dt *Table) StringIndex(column, row int) string {
	if dt.IsValidRow(row) != nil {
		return ""
	}
	ct := dt.Columns[column]
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
	ct, err := dt.ColumnByName(column)
	if err != nil {
		return ""
	}
	if ct.NumDims() != 1 {
		return ""
	}
	return ct.String1D(row)
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
	ct := dt.Columns[column]
	if ct.NumDims() == 1 {
		return nil
	}
	return ct.SubSpace([]int{row})
}

// Tensor returns the tensor SubSpace for given column (by name), row index
// for columns that have higher-dimensional tensors so each row is
// represented by an n-1 dimensional tensor, with the outer dimension
// being the row number.  Returns nil on any error.
func (dt *Table) Tensor(column string, row int) tensor.Tensor {
	if dt.IsValidRow(row) != nil {
		return nil
	}
	ct, err := dt.ColumnByName(column)
	if err != nil {
		return nil
	}
	if ct.NumDims() == 1 {
		return nil
	}
	return ct.SubSpace([]int{row})
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
	ct, err := dt.ColumnByName(column)
	if err != nil {
		return math.NaN()
	}
	if ct.NumDims() == 1 {
		return math.NaN()
	}
	_, sz := ct.RowCellSize()
	if idx >= sz || idx < 0 {
		return math.NaN()
	}
	off := row*sz + idx
	return ct.Float1D(off)
}

/////////////////////////////////////////////////////////////////////////////////////
//  Set

// SetFloatIndex sets the float64 value of cell at given column, row index
// for columns that have 1-dimensional tensors.
func (dt *Table) SetFloatIndex(column, row int, val float64) error {
	if err := dt.IsValidRow(row); err != nil {
		return err
	}
	ct := dt.Columns[column]
	if ct.NumDims() != 1 {
		return fmt.Errorf("table.Table SetFloatIndex: Column %d is a tensor, must use SetTensorFloat1D", column)
	}
	ct.SetFloat1D(row, val)
	return nil
}

// SetFloat sets the float64 value of cell at given column (by name), row index
// for columns that have 1-dimensional tensors.
func (dt *Table) SetFloat(column string, row int, val float64) error {
	if err := dt.IsValidRow(row); err != nil {
		return err
	}
	ct, err := dt.ColumnByName(column)
	if err != nil {
		return err
	}
	if ct.NumDims() != 1 {
		return fmt.Errorf("table.Table SetFloat: Column %s is a tensor, must use SetTensorFloat1D", column)
	}
	ct.SetFloat1D(row, val)
	return nil
}

// SetStringIndex sets the string value of cell at given column, row index
// for columns that have 1-dimensional tensors.  Returns true if set.
func (dt *Table) SetStringIndex(column, row int, val string) error {
	if err := dt.IsValidRow(row); err != nil {
		return err
	}
	ct := dt.Columns[column]
	if ct.NumDims() != 1 {
		return fmt.Errorf("table.Table SetStringIndex: Column %d is a tensor, must use SetTensorFloat1D", column)
	}
	ct.SetString1D(row, val)
	return nil
}

// SetString sets the string value of cell at given column (by name), row index
// for columns that have 1-dimensional tensors.  Returns true if set.
func (dt *Table) SetString(column string, row int, val string) error {
	if err := dt.IsValidRow(row); err != nil {
		return err
	}
	ct, err := dt.ColumnByName(column)
	if err != nil {
		return err
	}
	if ct.NumDims() != 1 {
		return fmt.Errorf("table.Table SetString: Column %s is a tensor, must use SetTensorFloat1D", column)
	}
	ct.SetString1D(row, val)
	return nil
}

// SetTensorIndex sets the tensor value of cell at given column, row index
// for columns that have n-dimensional tensors.  Returns true if set.
func (dt *Table) SetTensorIndex(column, row int, val tensor.Tensor) error {
	if err := dt.IsValidRow(row); err != nil {
		return err
	}
	ct := dt.Columns[column]
	_, csz := ct.RowCellSize()
	st := row * csz
	sz := min(csz, val.Len())
	if ct.IsString() {
		for j := 0; j < sz; j++ {
			ct.SetString1D(st+j, val.String1D(j))
		}
	} else {
		for j := 0; j < sz; j++ {
			ct.SetFloat1D(st+j, val.Float1D(j))
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
	ci, err := dt.ColumnIndex(column)
	if err != nil {
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
	ct, err := dt.ColumnByName(column)
	if err != nil {
		return err
	}
	_, sz := ct.RowCellSize()
	if idx >= sz || idx < 0 {
		return fmt.Errorf("table.Table IsValidRow: index %d is out of valid range [0..%d]", idx, sz)
	}
	off := row*sz + idx
	ct.SetFloat1D(off, val)
	return nil
}

//////////////////////////////////////////////////////////////////////////////////////
//  Copy Cell

// CopyCell copies into cell at given column, row from cell in other table.
// It is robust to differences in type; uses destination cell type.
// Returns error if column names are invalid.
func (dt *Table) CopyCell(column string, row int, cpt *Table, cpColNm string, cpRow int) error {
	ct, err := dt.ColumnByName(column)
	if err != nil {
		return err
	}
	cpct, err := cpt.ColumnByName(cpColNm)
	if err != nil {
		return err
	}
	_, sz := ct.RowCellSize()
	if sz == 1 {
		if ct.IsString() {
			ct.SetString1D(row, cpct.String1D(cpRow))
			return nil
		}
		ct.SetFloat1D(row, cpct.Float1D(cpRow))
		return nil
	}
	_, cpsz := cpct.RowCellSize()
	st := row * sz
	cst := cpRow * cpsz
	msz := min(sz, cpsz)
	if ct.IsString() {
		for j := 0; j < msz; j++ {
			ct.SetString1D(st+j, cpct.String1D(cst+j))
		}
	} else {
		for j := 0; j < msz; j++ {
			ct.SetFloat1D(st+j, cpct.Float1D(cst+j))
		}
	}
	return nil
}
