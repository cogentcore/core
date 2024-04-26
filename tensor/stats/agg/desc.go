// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package agg

import (
	"cogentcore.org/core/tensor/table"
)

// DescAggs are all the standard aggregates
var DescAggs = []Aggs{AggCount, AggMean, AggStd, AggSem, AggMin, AggQ1, AggMedian, AggQ3, AggMax}

// DescAggsND are all the standard aggregates for n-dimensional (n > 1) data -- cannot do quantiles
var DescAggsND = []Aggs{AggCount, AggMean, AggStd, AggSem, AggMin, AggMax}

// DescAll returns a table of standard descriptive aggregates for
// all numeric columns in given table, operating over all non-Null, non-NaN elements
// in each column.
func DescAll(ix *table.IndexView) *table.Table {
	st := ix.Table
	nonQs := []Aggs{AggCount, AggMean, AggStd, AggSem, AggMin, AggMax} // everything else done wth quantiles
	allAggs := []Aggs{AggCount, AggMean, AggStd, AggSem, AggMin, AggMax, AggQ1, AggMedian, AggQ3}
	nAgg := len(allAggs)
	dt := table.NewTable(nAgg)
	dt.AddStringColumn("Agg")
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
	sq := len(nonQs)
	for ci := range st.Columns {
		col := st.Columns[ci]
		if col.IsString() {
			continue
		}
		_, csz := col.RowCellSize()
		dtst := dt.Columns[dtci]
		for i, agtyp := range nonQs {
			ag := AggIndex(ix, ci, agtyp)
			si := i * csz
			for j := 0; j < csz; j++ {
				dtst.SetFloat1D(si+j, ag[j])
			}
			if dtci == 1 {
				dtnm.SetString1D(i, AggsName(agtyp))
			}
		}
		if col.NumDims() == 1 {
			qvs := QuantilesIndex(ix, ci, qs)
			for i, qv := range qvs {
				dtst.SetFloat1D(sq+i, qv)
				dtnm.SetString1D(sq+i, AggsName(allAggs[sq+i]))
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
	nonQs := []Aggs{AggCount, AggMean, AggStd, AggSem} // everything else done wth quantiles
	allAggs := []Aggs{AggCount, AggMean, AggStd, AggSem, AggMin, AggQ1, AggMedian, AggQ3, AggMax}
	nAgg := len(allAggs)
	if col.NumDims() > 1 { // nd cannot do qiles
		nonQs = append(nonQs, []Aggs{AggMin, AggMax}...)
		allAggs = nonQs
		nAgg += 2
	}
	dt := table.NewTable(nAgg)
	dt.AddStringColumn("Agg")
	dt.AddFloat64TensorColumn(st.ColumnNames[colIndex], col.Shape().Sizes[1:], col.Shape().Names[1:]...)
	dtnm := dt.Columns[0]
	dtst := dt.Columns[1]
	_, csz := col.RowCellSize()
	for i, agtyp := range nonQs {
		ag := AggIndex(ix, colIndex, agtyp)
		si := i * csz
		for j := 0; j < csz; j++ {
			dtst.SetFloat1D(si+j, ag[j])
		}
		dtnm.SetString1D(i, AggsName(agtyp))
	}
	if col.NumDims() == 1 {
		qs := []float64{0, .25, .5, .75, 1}
		qvs := QuantilesIndex(ix, colIndex, qs)
		sq := len(nonQs)
		for i, qv := range qvs {
			dtst.SetFloat1D(sq+i, qv)
			dtnm.SetString1D(sq+i, AggsName(allAggs[sq+i]))
		}
	}
	return dt
}

// Desc returns a table of standard descriptive aggregates
// of non-Null, non-NaN elements in given IndexView indexed view of an
// table.Table, for given column name.
// If name not found, nil is returned -- use Try version for error message.
func Desc(ix *table.IndexView, colNm string) *table.Table {
	colIndex := ix.Table.ColumnIndex(colNm)
	if colIndex == -1 {
		return nil
	}
	return DescIndex(ix, colIndex)
}

// Desc returns a table of standard descriptive aggregate aggs
// of non-Null, non-NaN elements in given IndexView indexed view of an
// table.Table, for given column name.
// If name not found, returns error message.
// Return value is size of each column cell -- 1 for scalar 1D columns
// and N for higher-dimensional columns.
func DescTry(ix *table.IndexView, colNm string) (*table.Table, error) {
	colIndex, err := ix.Table.ColumnIndexTry(colNm)
	if err != nil {
		return nil, err
	}
	return DescIndex(ix, colIndex), nil
}
