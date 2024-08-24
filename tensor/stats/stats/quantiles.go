// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package stats

import (
	"math"

	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/tensor/table"
)

// QuantilesIndex returns the given quantile(s) of non-NaN elements in given
// IndexView indexed view of an table.Table, for given column index.
// Column must be a 1d Column -- returns nil for n-dimensional columns.
// qs are 0-1 values, 0 = min, 1 = max, .5 = median, etc.  Uses linear interpolation.
// Because this requires a sort, it is more efficient to get as many quantiles
// as needed in one pass.
func QuantilesIndex(ix *table.IndexView, colIndex int, qs []float64) []float64 {
	nq := len(qs)
	if nq == 0 {
		return nil
	}
	col := ix.Table.Columns[colIndex]
	if col.NumDims() > 1 { // only valid for 1D
		return nil
	}
	rvs := make([]float64, nq)
	six := ix.Clone()                                // leave original indexes intact
	six.Filter(func(et *table.Table, row int) bool { // get rid of NaNs in this column
		if math.IsNaN(col.Float1D(row)) {
			return false
		}
		return true
	})
	six.SortColumn(colIndex, true)
	sz := len(six.Indexes) - 1 // length of our own index list
	fsz := float64(sz)
	for i, q := range qs {
		val := 0.0
		qi := q * fsz
		lwi := math.Floor(qi)
		lwii := int(lwi)
		if lwii >= sz {
			val = col.Float1D(six.Indexes[sz])
		} else if lwii < 0 {
			val = col.Float1D(six.Indexes[0])
		} else {
			phi := qi - lwi
			lwv := col.Float1D(six.Indexes[lwii])
			hiv := col.Float1D(six.Indexes[lwii+1])
			val = (1-phi)*lwv + phi*hiv
		}
		rvs[i] = val
	}
	return rvs
}

// Quantiles returns the given quantile(s) of non-Null, non-NaN elements in given
// IndexView indexed view of an table.Table, for given column name.
// If name not found, nil is returned -- use Try version for error message.
// Column must be a 1d Column -- returns nil for n-dimensional columns.
// qs are 0-1 values, 0 = min, 1 = max, .5 = median, etc.  Uses linear interpolation.
// Because this requires a sort, it is more efficient to get as many quantiles
// as needed in one pass.
func Quantiles(ix *table.IndexView, column string, qs []float64) []float64 {
	colIndex := errors.Log1(ix.Table.ColumnIndex(column))
	if colIndex == -1 {
		return nil
	}
	return QuantilesIndex(ix, colIndex, qs)
}
