// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package plots

import "cogentcore.org/core/plot"

// Table is an interface for tabular data for plotting,
// with columns of values.
type Table interface {
	// number of columns of data
	NumColumns() int

	//	name of given column
	ColumnName(i int) string

	//	number of rows of data
	NumRows() int

	// Data returns the data value at given column and row
	Data(column, row int) float32
}

// TableXYer is an interface for providing XY access to Table data
type TableXYer struct {
	Table Table

	// the indexes of the tensor columns to use for the X and Y data, respectively
	XColumn, YColumn int
}

func (dt *TableXYer) Len() int {
	return dt.Table.NumRows()
}

func (dt *TableXYer) XY(i int) (x, y float32) {
	return dt.Table.Data(dt.XColumn, i), dt.Table.Data(dt.YColumn, i)
}

// AddTableLine adds Line with given x, y columns from given tabular data
func AddTableLine(plt *plot.Plot, tab Table, xcolumn, ycolumn int) (*Line, error) {
	txy := &TableXYer{Table: tab, XColumn: xcolumn, YColumn: ycolumn}
	ln, err := NewLine(txy)
	if err != nil {
		return nil, err
	}
	plt.Add(ln)
	return ln, nil
}

// AddTableLinePoints adds Line w/ Points with given x, y columns from given tabular data
func AddTableLinePoints(plt *plot.Plot, tab Table, xcolumn, ycolumn int) (*Line, *Scatter, error) {
	txy := &TableXYer{Table: tab, XColumn: xcolumn, YColumn: ycolumn}
	ln, sc, err := NewLinePoints(txy)
	if err != nil {
		return nil, nil, err
	}
	plt.Add(ln)
	plt.Add(sc)
	return ln, sc, nil
}
