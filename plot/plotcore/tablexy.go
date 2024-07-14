// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package plotcore

import (
	"errors"
	"log"

	"cogentcore.org/core/math32"
	"cogentcore.org/core/math32/minmax"
	"cogentcore.org/core/plot"
	"cogentcore.org/core/plot/plots"
	"cogentcore.org/core/tensor/table"
)

// tableXY selects two columns from a [table.Table] data table to plot in a [plot.Plot],
// satisfying the [plot.XYer], [plot.Valuer], [plot.Labeler], and [plots.YErrorer] interfaces.
// For Tensor-valued cells, Index's specify tensor cell.
// Also satisfies the plot/plots.Labeler interface for labels attached to a line, and
// plot/plots.YErrorer for error bars.
type tableXY struct {

	// the index view of data table to plot from
	table *table.IndexView

	// the indexes of the tensor columns to use for the X and Y data, respectively
	xColumn, yColumn int

	// numer of elements in each row of data -- 1 for scalar, > 1 for multi-dimensional
	xRowSize, yRowSize int

	// the indexes of the element within each tensor cell if cells are n-dimensional, respectively
	xIndex, yIndex int

	// the column to use for returning a label using Label interface -- for string cols
	labelColumn int

	// the column to use for returning errorbars (+/- given value) -- if YColumn is tensor then this must also be a tensor and given YIndex used
	errColumn int

	// range constraints on Y values
	yRange minmax.Range32
}

var _ plot.XYer = &tableXY{}
var _ plot.Valuer = &tableXY{}
var _ plot.Labeler = &tableXY{}
var _ plots.YErrorer = &tableXY{}

// newTableXY returns a new XY plot view onto the given IndexView of table.Table (makes a copy),
// from given column indexes, and tensor indexes within each cell.
// Column indexes are enforced to be valid, with an error message if they are not.
func newTableXY(dt *table.IndexView, xcol, xtsrIndex, ycol, ytsrIndex int, yrng minmax.Range32) (*tableXY, error) {
	txy := &tableXY{table: dt.Clone(), xColumn: xcol, yColumn: ycol, xIndex: xtsrIndex, yIndex: ytsrIndex, yRange: yrng}
	return txy, txy.validate()
}

// newTableXYName returns a new XY plot view onto the given IndexView of table.Table (makes a copy),
// from given column name and tensor indexes within each cell.
// Column indexes are enforced to be valid, with an error message if they are not.
func newTableXYName(dt *table.IndexView, xi, xtsrIndex int, ycol string, ytsrIndex int, yrng minmax.Range32) (*tableXY, error) {
	yi, err := dt.Table.ColumnIndexTry(ycol)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	txy := &tableXY{table: dt.Clone(), xColumn: xi, yColumn: yi, xIndex: xtsrIndex, yIndex: ytsrIndex, yRange: yrng}
	return txy, txy.validate()
}

// validate returns error message if column indexes are invalid, else nil
// it also sets column indexes to 0 so nothing crashes.
func (txy *tableXY) validate() error {
	if txy.table == nil {
		return errors.New("eplot.TableXY table is nil")
	}
	nc := txy.table.Table.NumColumns()
	if txy.xColumn >= nc || txy.xColumn < 0 {
		txy.xColumn = 0
		return errors.New("eplot.TableXY XColumn index invalid -- reset to 0")
	}
	if txy.yColumn >= nc || txy.yColumn < 0 {
		txy.yColumn = 0
		return errors.New("eplot.TableXY YColumn index invalid -- reset to 0")
	}
	xc := txy.table.Table.Columns[txy.xColumn]
	yc := txy.table.Table.Columns[txy.yColumn]
	if xc.NumDims() > 1 {
		_, txy.xRowSize = xc.RowCellSize()
		// note: index already validated
	}
	if yc.NumDims() > 1 {
		_, txy.yRowSize = yc.RowCellSize()
		if txy.yIndex >= txy.yRowSize || txy.yIndex < 0 {
			txy.yIndex = 0
			return errors.New("eplot.TableXY Y TensorIndex invalid -- reset to 0")
		}
	}
	txy.filterValues()
	return nil
}

// filterValues removes items with NaN values, and out of Y range
func (txy *tableXY) filterValues() {
	txy.table.Filter(func(et *table.Table, row int) bool {
		xv := txy.tRowXValue(row)
		yv := txy.tRowValue(row)
		if math32.IsNaN(yv) || math32.IsNaN(xv) {
			return false
		}
		if txy.yRange.FixMin && yv < txy.yRange.Min {
			return false
		}
		if txy.yRange.FixMax && yv > txy.yRange.Max {
			return false
		}
		return true
	})
}

// Len returns the number of rows in the view of table
func (txy *tableXY) Len() int {
	if txy.table == nil || txy.table.Table == nil {
		return 0
	}
	return txy.table.Len()
}

// tRowValue returns the y value at given true table row in table
func (txy *tableXY) tRowValue(row int) float32 {
	yc := txy.table.Table.Columns[txy.yColumn]
	y := float32(0.0)
	switch {
	case yc.IsString():
		y = float32(row)
	case yc.NumDims() > 1:
		_, sz := yc.RowCellSize()
		if txy.yIndex < sz && txy.yIndex >= 0 {
			y = float32(yc.FloatRowCell(row, txy.yIndex))
		}
	default:
		y = float32(yc.Float1D(row))
	}
	return y
}

// Value returns the y value at given row in table
func (txy *tableXY) Value(row int) float32 {
	if txy.table == nil || txy.table.Table == nil || row >= txy.table.Len() {
		return 0
	}
	trow := txy.table.Indexes[row] // true table row
	yc := txy.table.Table.Columns[txy.yColumn]
	y := float32(0.0)
	switch {
	case yc.IsString():
		y = float32(row)
	case yc.NumDims() > 1:
		_, sz := yc.RowCellSize()
		if txy.yIndex < sz && txy.yIndex >= 0 {
			y = float32(yc.FloatRowCell(trow, txy.yIndex))
		}
	default:
		y = float32(yc.Float1D(trow))
	}
	return y
}

// tRowXValue returns an x value at given actual row in table
func (txy *tableXY) tRowXValue(row int) float32 {
	if txy.table == nil || txy.table.Table == nil {
		return 0
	}
	xc := txy.table.Table.Columns[txy.xColumn]
	x := float32(0.0)
	switch {
	case xc.IsString():
		x = float32(row)
	case xc.NumDims() > 1:
		_, sz := xc.RowCellSize()
		if txy.xIndex < sz && txy.xIndex >= 0 {
			x = float32(xc.FloatRowCell(row, txy.xIndex))
		}
	default:
		x = float32(xc.Float1D(row))
	}
	return x
}

// xValue returns an x value at given row in table
func (txy *tableXY) xValue(row int) float32 {
	if txy.table == nil || txy.table.Table == nil || row >= txy.table.Len() {
		return 0
	}
	trow := txy.table.Indexes[row] // true table row
	xc := txy.table.Table.Columns[txy.xColumn]
	x := float32(0.0)
	switch {
	case xc.IsString():
		x = float32(row)
	case xc.NumDims() > 1:
		_, sz := xc.RowCellSize()
		if txy.xIndex < sz && txy.xIndex >= 0 {
			x = float32(xc.FloatRowCell(trow, txy.xIndex))
		}
	default:
		x = float32(xc.Float1D(trow))
	}
	return x
}

// XY returns an x, y pair at given row in table
func (txy *tableXY) XY(row int) (x, y float32) {
	if txy.table == nil || txy.table.Table == nil {
		return 0, 0
	}
	x = txy.xValue(row)
	y = txy.Value(row)
	return
}

// Label returns a label for given row in table, implementing [plot.Labeler] interface
func (txy *tableXY) Label(row int) string {
	if txy.table == nil || txy.table.Table == nil || row >= txy.table.Len() {
		return ""
	}
	trow := txy.table.Indexes[row] // true table row
	return txy.table.Table.Columns[txy.labelColumn].String1D(trow)
}

// YError returns error bars, implementing [plots.YErrorer] interface.
func (txy *tableXY) YError(row int) (float32, float32) {
	if txy.table == nil || txy.table.Table == nil || row >= txy.table.Len() {
		return 0, 0
	}
	trow := txy.table.Indexes[row] // true table row
	ec := txy.table.Table.Columns[txy.errColumn]
	eval := float32(0.0)
	switch {
	case ec.IsString():
		eval = float32(row)
	case ec.NumDims() > 1:
		_, sz := ec.RowCellSize()
		if txy.yIndex < sz && txy.yIndex >= 0 {
			eval = float32(ec.FloatRowCell(trow, txy.yIndex))
		}
	default:
		eval = float32(ec.Float1D(trow))
	}
	return -eval, eval
}
