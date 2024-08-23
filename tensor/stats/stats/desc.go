// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package stats

import (
	"cogentcore.org/core/tensor/table"
)

// DescStats are all the standard stats
var DescStats = []Stats{Count, Mean, Std, Sem, Min, Max, Q1, Median, Q3}

// DescStatsND are all the standard stats for n-dimensional (n > 1) data -- cannot do quantiles
var DescStatsND = []Stats{Count, Mean, Std, Sem, Min, Max}

// DescAll returns a table of standard descriptive stats for
// all numeric columns in given table, operating over all non-Null, non-NaN elements
// in each column.
func DescAll(ix *table.IndexView) *table.Table {
	st := ix.Table
	nAgg := len(DescStats)
	dt := table.NewTable().SetNumRows(nAgg)
	dt.AddStringColumn("Stat")
	for ci := range st.Columns {
		col := st.Columns[ci]
		if col.IsString() {
			continue
		}
		dt.AddFloat64TensorColumn(st.ColumnNames[ci], col.Shape().Sizes[1:], col.Shape().Names[1:]...)
	}
	dtnm := dt.Columns[0]
	dtci := 1
	qs := []float64{.25, .5, .75}
	sq := len(DescStatsND)
	for ci := range st.Columns {
		col := st.Columns[ci]
		if col.IsString() {
			continue
		}
		_, csz := col.RowCellSize()
		dtst := dt.Columns[dtci]
		for i, styp := range DescStatsND {
			ag := StatIndex(ix, ci, styp)
			si := i * csz
			for j := 0; j < csz; j++ {
				dtst.SetFloat1D(si+j, ag[j])
			}
			if dtci == 1 {
				dtnm.SetString1D(i, styp.String())
			}
		}
		if col.NumDims() == 1 {
			qvs := QuantilesIndex(ix, ci, qs)
			for i, qv := range qvs {
				dtst.SetFloat1D(sq+i, qv)
				dtnm.SetString1D(sq+i, DescStats[sq+i].String())
			}
		}
		dtci++
	}
	return dt
}

// DescIndex returns a table of standard descriptive aggregates
// of non-Null, non-NaN elements in given IndexView indexed view of an
// table.Table, for given column index.
func DescIndex(ix *table.IndexView, colIndex int) *table.Table {
	st := ix.Table
	col := st.Columns[colIndex]
	stats := DescStats
	if col.NumDims() > 1 { // nd cannot do qiles
		stats = DescStatsND
	}
	nAgg := len(stats)
	dt := table.NewTable().SetNumRows(nAgg)
	dt.AddStringColumn("Stat")
	dt.AddFloat64TensorColumn(st.ColumnNames[colIndex], col.Shape().Sizes[1:], col.Shape().Names[1:]...)
	dtnm := dt.Columns[0]
	dtst := dt.Columns[1]
	_, csz := col.RowCellSize()
	for i, styp := range DescStatsND {
		ag := StatIndex(ix, colIndex, styp)
		si := i * csz
		for j := 0; j < csz; j++ {
			dtst.SetFloat1D(si+j, ag[j])
		}
		dtnm.SetString1D(i, styp.String())
	}
	if col.NumDims() == 1 {
		sq := len(DescStatsND)
		qs := []float64{.25, .5, .75}
		qvs := QuantilesIndex(ix, colIndex, qs)
		for i, qv := range qvs {
			dtst.SetFloat1D(sq+i, qv)
			dtnm.SetString1D(sq+i, DescStats[sq+i].String())
		}
	}
	return dt
}

// DescColumn returns a table of standard descriptive stats
// of non-NaN elements in given IndexView indexed view of an
// table.Table, for given column name.
// If name not found, returns error message.
// Return value is size of each column cell -- 1 for scalar 1D columns
// and N for higher-dimensional columns.
func DescColumn(ix *table.IndexView, column string) (*table.Table, error) {
	colIndex, err := ix.Table.ColumnIndex(column)
	if err != nil {
		return nil, err
	}
	return DescIndex(ix, colIndex), nil
}
