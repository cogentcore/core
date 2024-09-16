// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package split

import (
	"cogentcore.org/core/tensor"
	"cogentcore.org/core/tensor/datafs"
	"cogentcore.org/core/tensor/table"
)

func TableStats(dir *datafs.Data, dt *table.Table, columns ...string) {
	dv := table.NewView(dt)
	// important for consistency across columns, to do full outer product sort first.
	dv.SortColumns(tensor.Ascending, tensor.Stable, columns...)
	Groups(dir, dv.ColumnList(columns...)...)
}

// Stats computes the given stats function on the unique grouped values of the
// first tensor passed (the Group tensor), for each of the additional
// tensors passed (the Value tensors).
// It creates a subdirectory in given directory with the name of each value tensor,
// (if it does not yet exist), and then creates a subdirectory within that
// for the statistic name.  Within that statistic directory, it creates
// a String tensor with the unique values of the Group tensor, and a
// Float64 tensor with the statistics results for each such unique group value.
// It calls the Groups function on the Group tensor first.
func Stats(dir *datafs.Data, stat string, tsrs ...*tensor.Indexed) {

}

/*

// AggIndex performs aggregation using given standard statistic (e.g., Mean) across
// all splits, and returns the SplitAgg container of the results, which are also
// stored in the Splits.  Column is specified by index.
func AggIndex(spl *table.Splits, colIndex int, stat stats.Stats) *table.SplitAgg {
	ag := spl.AddAgg(stat.String(), colIndex)
	for _, sp := range spl.Splits {
		agv := stats.StatIndex(sp, colIndex, stat)
		ag.Aggs = append(ag.Aggs, agv)
	}
	return ag
}

// AggColumn performs aggregation using given standard statistic (e.g., Mean) across
// all splits, and returns the SplitAgg container of the results, which are also
// stored in the Splits.  Column is specified by name; returns error for bad column name.
func AggColumn(spl *table.Splits, column string, stat stats.Stats) (*table.SplitAgg, error) {
	dt := spl.Table()
	if dt == nil {
		return nil, fmt.Errorf("split.AggTry: No splits to aggregate over")
	}
	colIndex, err := dt.ColumnByIndex(column)
	if err != nil {
		return nil, err
	}
	return AggIndex(spl, colIndex, stat), nil
}

// AggAllNumericColumns performs aggregation using given standard aggregation function across
// all splits, for all number-valued columns in the table.
func AggAllNumericColumns(spl *table.Splits, stat stats.Stats) {
	dt := spl.Table()
	for ci, cl := range dt.Columns {
		if cl.IsString() {
			continue
		}
		AggIndex(spl, ci, stat)
	}
}

///////////////////////////////////////////////////
//   Desc

// DescIndex performs aggregation using standard statistics across
// all splits, and stores results in the Splits.  Column is specified by index.
func DescIndex(spl *table.Splits, colIndex int) {
	dt := spl.Table()
	if dt == nil {
		return
	}
	col := dt.Columns[colIndex]
	sts := stats.DescStats
	if col.NumDims() > 1 { // nd cannot do qiles
		sts = stats.DescStatsND
	}
	for _, st := range sts {
		AggIndex(spl, colIndex, st)
	}
}

// DescColumn performs aggregation using standard statistics across
// all splits, and stores results in the Splits.
// Column is specified by name; returns error for bad column name.
func DescColumn(spl *table.Splits, column string) error {
	dt := spl.Table()
	if dt == nil {
		return fmt.Errorf("split.DescTry: No splits to aggregate over")
	}
	colIndex, err := dt.ColumnByIndex(column)
	if err != nil {
		return err
	}
	DescIndex(spl, colIndex)
	return nil
}
*/
