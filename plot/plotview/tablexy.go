// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package plotview

import (
	"errors"
	"log"

	"cogentcore.org/core/math32"
	"cogentcore.org/core/math32/minmax"
	"cogentcore.org/core/tensor/table"
)

// TableXY selects two columns from a table.Table data table to plot in a gonum plot,
// satisfying the plotter.XYer and .Valuer interfaces (for bar charts).
// For Tensor-valued cells, Index's specify tensor cell.
// Also satisfies the plotter.Labeler interface for labels attached to a line, and
// plotter.YErrorer for error bars.
type TableXY struct {

	// the index view of data table to plot from
	Table *table.IndexView

	// the indexes of the tensor columns to use for the X and Y data, respectively
	XColumn, YColumn int

	// numer of elements in each row of data -- 1 for scalar, > 1 for multi-dimensional
	XRowSz, YRowSz int

	// the indexes of the element within each tensor cell if cells are n-dimensional, respectively
	XIndex, YIndex int

	// the column to use for returning a label using Label interface -- for string cols
	LabelColumn int

	// the column to use for returning errorbars (+/- given value) -- if YColumn is tensor then this must also be a tensor and given YIndex used
	ErrColumn int

	// range constraints on Y values
	YRange minmax.Range32
}

// NewTableXY returns a new XY plot view onto the given IndexView of table.Table (makes a copy),
// from given column indexes, and tensor indexes within each cell.
// Column indexes are enforced to be valid, with an error message if they are not.
func NewTableXY(dt *table.IndexView, xcol, xtsrIndex, ycol, ytsrIndex int, yrng minmax.Range32) (*TableXY, error) {
	txy := &TableXY{Table: dt.Clone(), XColumn: xcol, YColumn: ycol, XIndex: xtsrIndex, YIndex: ytsrIndex, YRange: yrng}
	return txy, txy.Validate()
}

// NewTableXYName returns a new XY plot view onto the given IndexView of table.Table (makes a copy),
// from given column name and tensor indexes within each cell.
// Column indexes are enforced to be valid, with an error message if they are not.
func NewTableXYName(dt *table.IndexView, xi, xtsrIndex int, ycol string, ytsrIndex int, yrng minmax.Range32) (*TableXY, error) {
	yi, err := dt.Table.ColumnIndexTry(ycol)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	txy := &TableXY{Table: dt.Clone(), XColumn: xi, YColumn: yi, XIndex: xtsrIndex, YIndex: ytsrIndex, YRange: yrng}
	return txy, txy.Validate()
}

// Validate returns error message if column indexes are invalid, else nil
// it also sets column indexes to 0 so nothing crashes.
func (txy *TableXY) Validate() error {
	if txy.Table == nil {
		return errors.New("eplot.TableXY table is nil")
	}
	nc := txy.Table.Table.NumColumns()
	if txy.XColumn >= nc || txy.XColumn < 0 {
		txy.XColumn = 0
		return errors.New("eplot.TableXY XColumn index invalid -- reset to 0")
	}
	if txy.YColumn >= nc || txy.YColumn < 0 {
		txy.YColumn = 0
		return errors.New("eplot.TableXY YColumn index invalid -- reset to 0")
	}
	xc := txy.Table.Table.Columns[txy.XColumn]
	yc := txy.Table.Table.Columns[txy.YColumn]
	if xc.NumDims() > 1 {
		_, txy.XRowSz = xc.RowCellSize()
		// note: index already validated
	}
	if yc.NumDims() > 1 {
		_, txy.YRowSz = yc.RowCellSize()
		if txy.YIndex >= txy.YRowSz || txy.YIndex < 0 {
			txy.YIndex = 0
			return errors.New("eplot.TableXY Y TensorIndex invalid -- reset to 0")
		}
	}
	txy.FilterValues()
	return nil
}

// FilterValues removes items with NaN values, and out of Y range
func (txy *TableXY) FilterValues() {
	txy.Table.Filter(func(et *table.Table, row int) bool {
		xv := txy.TRowXValue(row)
		yv := txy.TRowValue(row)
		if math32.IsNaN(yv) || math32.IsNaN(xv) {
			return false
		}
		if txy.YRange.FixMin && yv < txy.YRange.Min {
			return false
		}
		if txy.YRange.FixMax && yv > txy.YRange.Max {
			return false
		}
		return true
	})
}

// Len returns the number of rows in the view of table
func (txy *TableXY) Len() int {
	if txy.Table == nil || txy.Table.Table == nil {
		return 0
	}
	return txy.Table.Len()
}

// TRowValue returns the y value at given true table row in table view
func (txy *TableXY) TRowValue(row int) float32 {
	yc := txy.Table.Table.Columns[txy.YColumn]
	y := float32(0.0)
	switch {
	case yc.IsString():
		y = float32(row)
	case yc.NumDims() > 1:
		_, sz := yc.RowCellSize()
		if txy.YIndex < sz && txy.YIndex >= 0 {
			y = float32(yc.FloatRowCell(row, txy.YIndex))
		}
	default:
		y = float32(yc.Float1D(row))
	}
	return y
}

// Value returns the y value at given row in table view
func (txy *TableXY) Value(row int) float32 {
	if txy.Table == nil || txy.Table.Table == nil || row >= txy.Table.Len() {
		return 0
	}
	trow := txy.Table.Indexes[row] // true table row
	yc := txy.Table.Table.Columns[txy.YColumn]
	y := float32(0.0)
	switch {
	case yc.IsString():
		y = float32(row)
	case yc.NumDims() > 1:
		_, sz := yc.RowCellSize()
		if txy.YIndex < sz && txy.YIndex >= 0 {
			y = float32(yc.FloatRowCell(trow, txy.YIndex))
		}
	default:
		y = float32(yc.Float1D(trow))
	}
	return y
}

// TRowXValue returns an x value at given actual row in table
func (txy *TableXY) TRowXValue(row int) float32 {
	if txy.Table == nil || txy.Table.Table == nil {
		return 0
	}
	xc := txy.Table.Table.Columns[txy.XColumn]
	x := float32(0.0)
	switch {
	case xc.IsString():
		x = float32(row)
	case xc.NumDims() > 1:
		_, sz := xc.RowCellSize()
		if txy.XIndex < sz && txy.XIndex >= 0 {
			x = float32(xc.FloatRowCell(row, txy.XIndex))
		}
	default:
		x = float32(xc.Float1D(row))
	}
	return x
}

// XValue returns an x value at given row in table view
func (txy *TableXY) XValue(row int) float32 {
	if txy.Table == nil || txy.Table.Table == nil || row >= txy.Table.Len() {
		return 0
	}
	trow := txy.Table.Indexes[row] // true table row
	xc := txy.Table.Table.Columns[txy.XColumn]
	x := float32(0.0)
	switch {
	case xc.IsString():
		x = float32(row)
	case xc.NumDims() > 1:
		_, sz := xc.RowCellSize()
		if txy.XIndex < sz && txy.XIndex >= 0 {
			x = float32(xc.FloatRowCell(trow, txy.XIndex))
		}
	default:
		x = float32(xc.Float1D(trow))
	}
	return x
}

// XY returns an x, y pair at given row in table
func (txy *TableXY) XY(row int) (x, y float32) {
	if txy.Table == nil || txy.Table.Table == nil {
		return 0, 0
	}
	x = txy.XValue(row)
	y = txy.Value(row)
	return
}

// Label returns a label for given row in table, using plotter.Labeler interface
func (txy *TableXY) Label(row int) string {
	if txy.Table == nil || txy.Table.Table == nil || row >= txy.Table.Len() {
		return ""
	}
	trow := txy.Table.Indexes[row] // true table row
	return txy.Table.Table.Columns[txy.LabelColumn].String1D(trow)
}

// YError returns a error bars using ploter.YErrorer interface
func (txy *TableXY) YError(row int) (float32, float32) {
	if txy.Table == nil || txy.Table.Table == nil || row >= txy.Table.Len() {
		return 0, 0
	}
	trow := txy.Table.Indexes[row] // true table row
	ec := txy.Table.Table.Columns[txy.ErrColumn]
	eval := float32(0.0)
	switch {
	case ec.IsString():
		eval = float32(row)
	case ec.NumDims() > 1:
		_, sz := ec.RowCellSize()
		if txy.YIndex < sz && txy.YIndex >= 0 {
			eval = float32(ec.FloatRowCell(trow, txy.YIndex))
		}
	default:
		eval = float32(ec.Float1D(trow))
	}
	return -eval, eval
}
