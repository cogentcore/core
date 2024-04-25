// Copyright (c) 2024, The Cogent Core Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package table

//go:generate core generate

import (
	"fmt"
	"log/slog"
	"math"
	"strings"

	"cogentcore.org/core/tensor"
)

// table.Table is a table of data, with columns of tensors,
// each with the same number of Rows (outer-most dimension).
type Table struct { //types:add

	// columns of data, as tensor.Tensor tensors
	Cols []tensor.Tensor `view:"no-inline"`

	// the names of the columns
	ColNames []string

	// number of rows, which is enforced to be the size of the outer-most dimension of the column tensors
	Rows int `edit:"-"`

	// the map of column names to column numbers
	ColNameMap map[string]int `view:"-"`

	// misc meta data for the table.  We use lower-case key names following the struct tag convention:  name = name of table; desc = description; read-only = gui is read-only; precision = n for precision to write out floats in csv.  For Column-specific data, we look for ColName: prefix, specifically ColName:desc = description of the column contents, which is shown as tooltip in the etview.TableView, and :width for width of a column
	MetaData map[string]string
}

// NumRows returns the number of rows (arrow / dframe api)
func (dt *Table) NumRows() int {
	return dt.Rows
}

// IsValidRow returns true if the row is valid
func (dt *Table) IsValidRow(row int) bool {
	if row < 0 || row >= dt.Rows {
		return false
	}
	return true
}

// NumCols returns the number of columns (arrow / dframe api)
func (dt *Table) NumCols() int {
	return len(dt.Cols)
}

// Col returns the tensor at given column index
func (dt *Table) Col(i int) tensor.Tensor {
	return dt.Cols[i]
}

// ColByName returns the tensor at given column name without any error messages.
// Returns nil if not found
func (dt *Table) ColByName(name string) tensor.Tensor {
	i, ok := dt.ColNameMap[name]
	if !ok {
		return nil
	}
	return dt.Cols[i]
}

// ColIndex returns the index of the given column name.
// returns -1 if name not found -- see Try version for error message.
func (dt *Table) ColIndex(name string) int {
	i, ok := dt.ColNameMap[name]
	if !ok {
		return -1
	}
	return i
}

// ColIndexTry returns the index of the given column name,
// along with an error if not found.
func (dt *Table) ColIndexTry(name string) (int, error) {
	i, ok := dt.ColNameMap[name]
	if !ok {
		return 0, fmt.Errorf("table.Table ColIndex: column named: %v not found", name)
	}
	return i, nil
}

// ColIndexesByNames returns the indexes of the given column names.
// idxs have -1 if name not found -- see Try version for error message.
func (dt *Table) ColIndexesByNames(names []string) []int {
	nc := len(names)
	if nc == 0 {
		return nil
	}
	cidx := make([]int, nc)
	for i, cn := range names {
		cidx[i] = dt.ColIndex(cn)
	}
	return cidx
}

// ColName returns the name of given column
func (dt *Table) ColName(i int) string {
	return dt.ColNames[i]
}

// UpdateColNameMap updates the column name map
func (dt *Table) UpdateColNameMap() {
	nc := dt.NumCols()
	dt.ColNameMap = make(map[string]int, nc)
	for i, nm := range dt.ColNames {
		if _, has := dt.ColNameMap[nm]; has {
			slog.Warn("table.Table duplicate column name", "name", nm)
		} else {
			dt.ColNameMap[nm] = i
		}
	}
}

// AddCol adds the given tensor as a column to the table.
// returns error if it is not a RowMajor organized tensor, and automatically
// adjusts the shape to fit the current number of rows.
func (dt *Table) AddCol(tsr tensor.Tensor, name string) error {
	dt.Cols = append(dt.Cols, tsr)
	dt.ColNames = append(dt.ColNames, name)
	dt.UpdateColNameMap()
	rows := max(1, dt.Rows)
	tsr.SetNumRows(rows)
	return nil
}

// DeleteColName deletes column of given name.
// returns false if not found.
func (dt *Table) DeleteColName(name string) bool {
	ci := dt.ColIndex(name)
	if ci < 0 {
		return false
	}
	dt.DeleteColIndex(ci)
	return true
}

// DeleteColIndex deletes column of given index
func (dt *Table) DeleteColIndex(idx int) {
	dt.Cols = append(dt.Cols[:idx], dt.Cols[idx+1:]...)
	dt.ColNames = append(dt.ColNames[:idx], dt.ColNames[idx+1:]...)
	dt.UpdateColNameMap()
}

// DeleteAll deletes all columns -- full reset
func (dt *Table) DeleteAll() {
	dt.Cols = nil
	dt.ColNames = nil
	dt.Rows = 0
	dt.ColNameMap = nil
}

// AddRows adds n rows to each of the columns
func (dt *Table) AddRows(n int) { //types:add
	dt.SetNumRows(dt.Rows + n)
}

// SetNumRows sets the number of rows in the table, across all columns
// if rows = 0 then effective number of rows in tensors is 1, as this dim cannot be 0
func (dt *Table) SetNumRows(rows int) { //types:add
	dt.Rows = rows // can be 0
	rows = max(1, rows)
	for _, tsr := range dt.Cols {
		tsr.SetNumRows(rows)
	}
}

// SetFromSchema configures table from given Schema.
// The actual tensor number of rows is enforced to be > 0, because we
// cannot have a null dimension in tensor shape.
// does not preserve any existing columns / data.
func (dt *Table) SetFromSchema(sc Schema, rows int) {
	nc := len(sc)
	dt.Cols = make([]tensor.Tensor, nc)
	dt.ColNames = make([]string, nc)
	dt.Rows = rows // can be 0
	rows = max(1, rows)
	for i := range dt.Cols {
		cl := &sc[i]
		dt.ColNames[i] = cl.Name
		sh := append([]int{rows}, cl.CellShape...)
		dn := append([]string{"row"}, cl.DimNames...)
		_, _ = dn, sh
		// tsr := tensor.New(cl.Type, sh, nil, dn)
		// dt.Cols[i] = tsr
	}
	dt.UpdateColNameMap()
}

func NewTable(name string) *Table {
	et := &Table{}
	et.SetMetaData("name", name)
	return et
}

// New returns a new Table constructed from given Schema.
// The actual tensor number of rows is enforced to be > 0, because we
// cannot have a null dimension in tensor shape
func New(sc Schema, rows int) *Table {
	dt := &Table{}
	dt.SetFromSchema(sc, rows)
	return dt
}

// Schema returns the Schema (column properties) for this table
func (dt *Table) Schema() Schema {
	nc := dt.NumCols()
	sc := make(Schema, nc)
	for i := range dt.Cols {
		cl := &sc[i]
		tsr := dt.Cols[i]
		cl.Name = dt.ColNames[i]
		// cl.Type = tsr.DataType()
		cl.CellShape = tsr.Shape().Sizes[1:]
		cl.DimNames = tsr.Shape().Names[1:]
	}
	return sc
}

// note: no really clean definition of CopyFrom -- no point of re-using existing
// table -- just clone it.

// Clone returns a complete copy of this table
func (dt *Table) Clone() *Table {
	sc := dt.Schema()
	cp := New(sc, dt.Rows)
	for i, cl := range dt.Cols {
		ccl := cp.Cols[i]
		ccl.CopyFrom(cl)
	}
	cp.CopyMetaDataFrom(dt)
	return cp
}

// AppendRows appends shared columns in both tables with input table rows
func (dt *Table) AppendRows(dt2 *Table) {
	shared := false
	strow := dt.NumRows()
	for iCol := range dt.Cols {
		colName := dt.ColName(iCol)
		if dt2.ColIndex(colName) != -1 {
			if !shared {
				shared = true
				dt.AddRows(dt2.NumRows())
			}
			for iRow := 0; iRow < dt2.NumRows(); iRow++ {
				dt.CopyCell(colName, iRow+strow, dt2, colName, iRow)
			}
		}
	}
}

// SetMetaData sets given meta-data key to given value, safely creating the
// map if not yet initialized.  Standard Keys are:
// * name -- name of table
// * desc -- description of table
// * read-only  -- makes gui read-only (inactive edits) for etview.TableView
// * ColName:* -- prefix for all column-specific meta-data
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
func (dt *Table) RowsByStringIndex(colIndex int, str string, contains, ignoreCase bool) []int {
	col := dt.Cols[colIndex]
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
// string value in given column name.  returns nil if name invalid -- see also Try.
// if contains, only checks if row contains string; if ignoreCase, ignores case.
// Use named args for greater clarity.
func (dt *Table) RowsByString(colNm string, str string, contains, ignoreCase bool) []int {
	ci := dt.ColIndex(colNm)
	if ci < 0 {
		return nil
	}
	return dt.RowsByStringIndex(ci, str, contains, ignoreCase)
}

//////////////////////////////////////////////////////////////////////////////////////
//  Cell convenience access methods

// FloatIndex returns the float64 value of cell at given column, row index
// for columns that have 1-dimensional tensors.
// Returns NaN if column is not a 1-dimensional tensor or row not valid.
func (dt *Table) FloatIndex(col, row int) float64 {
	if !dt.IsValidRow(row) {
		return math.NaN()
	}
	ct := dt.Cols[col]
	if ct.NumDims() != 1 {
		return math.NaN()
	}
	return ct.Float1D(row)
}

// Float returns the float64 value of cell at given column (by name), row index
// for columns that have 1-dimensional tensors.
// Returns NaN if column is not a 1-dimensional tensor or col name not found, or row not valid.
func (dt *Table) Float(colNm string, row int) float64 {
	if !dt.IsValidRow(row) {
		return math.NaN()
	}
	ct := dt.ColByName(colNm)
	if ct == nil {
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
func (dt *Table) StringIndex(col, row int) string {
	if !dt.IsValidRow(row) {
		return ""
	}
	ct := dt.Cols[col]
	if ct.NumDims() != 1 {
		return ""
	}
	return ct.String1D(row)
}

// StringValue returns the string value of cell at given column (by name), row index
// for columns that have 1-dimensional tensors.
// Returns "" if column is not a 1-dimensional tensor or row not valid.
func (dt *Table) StringValue(colNm string, row int) string {
	if !dt.IsValidRow(row) {
		return ""
	}
	ct := dt.ColByName(colNm)
	if ct == nil {
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
func (dt *Table) TensorIndex(col, row int) tensor.Tensor {
	if !dt.IsValidRow(row) {
		return nil
	}
	ct := dt.Cols[col]
	if ct.NumDims() == 1 {
		return nil
	}
	return ct.SubSpace([]int{row})
}

// Tensor returns the tensor SubSpace for given column (by name), row index
// for columns that have higher-dimensional tensors so each row is
// represented by an n-1 dimensional tensor, with the outer dimension
// being the row number.  Returns nil on any error -- see Try version for
// error returns.
func (dt *Table) Tensor(colNm string, row int) tensor.Tensor {
	if !dt.IsValidRow(row) {
		return nil
	}
	ct := dt.ColByName(colNm)
	if ct == nil {
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
// being the row number.  Returns 0 on any error -- see Try version for
// error returns.
func (dt *Table) TensorFloat1D(colNm string, row int, idx int) float64 {
	if !dt.IsValidRow(row) {
		return 0
	}
	ct := dt.ColByName(colNm)
	if ct == nil {
		return 0
	}
	if ct.NumDims() == 1 {
		return 0
	}
	_, sz := ct.Shape().RowCellSize()
	if idx >= sz || idx < 0 {
		return 0
	}
	off := row*sz + idx
	return ct.Float1D(off)
}

/////////////////////////////////////////////////////////////////////////////////////
//  Set

// SetFloatIndex sets the float64 value of cell at given column, row index
// for columns that have 1-dimensional tensors.  Returns true if set.
func (dt *Table) SetFloatIndex(col, row int, val float64) bool {
	if !dt.IsValidRow(row) {
		return false
	}
	ct := dt.Cols[col]
	if ct.NumDims() != 1 {
		return false
	}
	ct.SetFloat1D(row, val)
	return true
}

// SetFloat sets the float64 value of cell at given column (by name), row index
// for columns that have 1-dimensional tensors.
func (dt *Table) SetFloat(colNm string, row int, val float64) bool {
	if !dt.IsValidRow(row) {
		return false
	}
	ct := dt.ColByName(colNm)
	if ct == nil {
		return false
	}
	if ct.NumDims() != 1 {
		return false
	}
	ct.SetFloat1D(row, val)
	return true
}

// SetStringIndex sets the string value of cell at given column, row index
// for columns that have 1-dimensional tensors.  Returns true if set.
func (dt *Table) SetStringIndex(col, row int, val string) bool {
	if !dt.IsValidRow(row) {
		return false
	}
	ct := dt.Cols[col]
	if ct.NumDims() != 1 {
		return false
	}
	ct.SetString1D(row, val)
	return true
}

// SetString sets the string value of cell at given column (by name), row index
// for columns that have 1-dimensional tensors.  Returns true if set.
func (dt *Table) SetString(colNm string, row int, val string) bool {
	if !dt.IsValidRow(row) {
		return false
	}
	ct := dt.ColByName(colNm)
	if ct == nil {
		return false
	}
	if ct.NumDims() != 1 {
		return false
	}
	ct.SetString1D(row, val)
	return true
}

// SetTensorIndex sets the tensor value of cell at given column, row index
// for columns that have n-dimensional tensors.  Returns true if set.
func (dt *Table) SetTensorIndex(col, row int, val tensor.Tensor) bool {
	if !dt.IsValidRow(row) {
		return false
	}
	ct := dt.Cols[col]
	_, csz := ct.Shape().RowCellSize()
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
	return true
}

// SetTensor sets the tensor value of cell at given column (by name), row index
// for columns that have n-dimensional tensors.  Returns true if set.
func (dt *Table) SetTensor(colNm string, row int, val tensor.Tensor) bool {
	if !dt.IsValidRow(row) {
		return false
	}
	ci := dt.ColIndex(colNm)
	if ci < 0 {
		return false
	}
	return dt.SetTensorIndex(ci, row, val)
}

// SetTensorFloat1D sets the tensor cell's float cell value at given 1D index within cell,
// at given column (by name), row index for columns that have n-dimensional tensors.
// Returns true if set.
func (dt *Table) SetTensorFloat1D(colNm string, row int, idx int, val float64) bool {
	if !dt.IsValidRow(row) {
		return false
	}
	ct := dt.ColByName(colNm)
	if ct == nil {
		return false
	}
	_, sz := ct.Shape().RowCellSize()
	if idx >= sz || idx < 0 {
		return false
	}
	off := row*sz + idx
	ct.SetFloat1D(off, val)
	return true
}

//////////////////////////////////////////////////////////////////////////////////////
//  Copy Cell

// CopyCell copies into cell at given col, row from cell in other table.
// It is robust to differences in type -- uses destination cell type.
// Returns error if column names are invalid.
func (dt *Table) CopyCell(colNm string, row int, cpt *Table, cpColNm string, cpRow int) bool {
	ct := dt.ColByName(colNm)
	if ct != nil {
		return false
	}
	cpct := cpt.ColByName(cpColNm)
	if cpct != nil {
		return false
	}
	_, sz := ct.Shape().RowCellSize()
	if sz == 1 {
		if ct.IsString() {
			ct.SetString1D(row, cpct.String1D(cpRow))
		} else {
			ct.SetFloat1D(row, cpct.Float1D(cpRow))
		}
	} else {
		_, cpsz := cpct.Shape().RowCellSize()
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
	}
	return true
}
