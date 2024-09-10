// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package datafs

import (
	"cogentcore.org/core/tensor"
	"cogentcore.org/core/tensor/stats/stats"
	"cogentcore.org/core/tensor/table"
)

// StdStats are the standard descriptive stats computed in StdStatsData,
// for one-dimensional tensors.  For higher-dimensional cases, the last 3
// quartile-based ones are excluded because they are not compatible.
var StdStats = []Stats{stats.Count, stats.Mean, stats.Std, stats.Sem, stats.Min, stats.Max, stats.Q1, stats.Median, stats.Q3}

// StdStatsData adds standard descriptive statistics for given tensor
// to the given [datafs] directory.
// This is an easy way to provide a comprehensive description of data.
// Stats are in StdStats list: Count, Mean, Std, Sem, Min, Max, Q1, Median, Q3
func StdStatsData(dir *datafs.Data, tsr *tensor.Indexed) {
	/*
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
			// ag := StatIndex(ix, colIndex, styp)
			ag := 0.0
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
	*/
	return ix.Table // dt
}

// DescAll returns a table of standard descriptive stats for
// all numeric columns in given table, operating over all non-Null, non-NaN elements
// in each column.
func DescAll(ix *table.Indexed) *table.Table {
	/*
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
				ag := 0.0
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
	*/
	return ix.Table // dt
}
