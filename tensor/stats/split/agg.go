// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package split

import (
	"fmt"

	"cogentcore.org/core/tensor/stats/agg"
	"cogentcore.org/core/tensor/table"
)

// AggIndex performs aggregation using given standard aggregation function across
// all splits, and returns the SplitAgg container of the results, which are also
// stored in the Splits.  Column is specified by index.
func AggIndex(spl *table.Splits, colIndex int, aggTyp agg.Aggs) *table.SplitAgg {
	ag := spl.AddAgg(agg.AggsName(aggTyp), colIndex)
	for _, sp := range spl.Splits {
		agv := agg.AggIndex(sp, colIndex, aggTyp)
		ag.Aggs = append(ag.Aggs, agv)
	}
	return ag
}

// Agg performs aggregation using given standard aggregation function across
// all splits, and returns the SplitAgg container of the results, which are also
// stored in the Splits.  Column is specified by name -- see Try for error msg version.
func Agg(spl *table.Splits, colNm string, aggTyp agg.Aggs) *table.SplitAgg {
	dt := spl.Table()
	if dt == nil {
		return nil
	}
	return AggIndex(spl, dt.ColumnIndex(colNm), aggTyp)
}

// AggTry performs aggregation using given standard aggregation function across
// all splits, and returns the SplitAgg container of the results, which are also
// stored in the Splits.  Column is specified by name -- returns error for bad column name.
func AggTry(spl *table.Splits, colNm string, aggTyp agg.Aggs) (*table.SplitAgg, error) {
	dt := spl.Table()
	if dt == nil {
		return nil, fmt.Errorf("split.AggTry: No splits to aggregate over")
	}
	colIndex, err := dt.ColumnIndexTry(colNm)
	if err != nil {
		return nil, err
	}
	return AggIndex(spl, colIndex, aggTyp), nil
}

// AggAllNumericCols performs aggregation using given standard aggregation function across
// all splits, for all number-valued columns in the table.
func AggAllNumericCols(spl *table.Splits, aggTyp agg.Aggs) {
	dt := spl.Table()
	for ci, cl := range dt.Columns {
		if cl.IsString() {
			continue
		}
		AggIndex(spl, ci, aggTyp)
	}
}

///////////////////////////////////////////////////
//   Desc

// DescIndex performs aggregation using standard aggregation functions across
// all splits, and stores results in the Splits.  Column is specified by index.
func DescIndex(spl *table.Splits, colIndex int) {
	dt := spl.Table()
	if dt == nil {
		return
	}
	col := dt.Columns[colIndex]
	allAggs := agg.DescAggs
	if col.NumDims() > 1 { // nd cannot do qiles
		allAggs = agg.DescAggsND
	}
	for _, ag := range allAggs {
		AggIndex(spl, colIndex, ag)
	}
}

// Desc performs aggregation using standard aggregation functions across
// all splits, and stores results in the Splits.
// Column is specified by name -- see Try for error msg version.
func Desc(spl *table.Splits, colNm string) {
	dt := spl.Table()
	if dt == nil {
		return
	}
	DescIndex(spl, dt.ColumnIndex(colNm))
}

// DescTry performs aggregation using standard aggregation functions across
// all splits, and stores results in the Splits.
// Column is specified by name -- returns error for bad column name.
func DescTry(spl *table.Splits, colNm string) error {
	dt := spl.Table()
	if dt == nil {
		return fmt.Errorf("split.DescTry: No splits to aggregate over")
	}
	colIndex, err := dt.ColumnIndexTry(colNm)
	if err != nil {
		return err
	}
	DescIndex(spl, colIndex)
	return nil
}
