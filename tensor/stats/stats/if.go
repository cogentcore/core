// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package stats

import (
	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/tensor/table"
)

// IfFunc is used for the *If aggregators -- counted if it returns true
type IfFunc func(idx int, val float64) bool

///////////////////////////////////////////////////
//   CountIf

// CountIfIndex returns the count of true return values for given IfFunc on
// non-NaN elements in given IndexView indexed view of an
// table.Table, for given column index.
// Return value(s) is size of column cell: 1 for scalar 1D columns
// and N for higher-dimensional columns.
func CountIfIndex(ix *table.IndexView, colIndex int, iffun IfFunc) []float64 {
	return StatIndexFunc(ix, colIndex, 0, func(idx int, val float64, agg float64) float64 {
		if iffun(idx, val) {
			return agg + 1
		}
		return agg
	})
}

// CountIfColumn returns the count of true return values for given IfFunc on
// non-NaN elements in given IndexView indexed view of an
// table.Table, for given column name.
// If name not found, nil is returned.
// Return value(s) is size of column cell: 1 for scalar 1D columns
// and N for higher-dimensional columns.
func CountIfColumn(ix *table.IndexView, column string, iffun IfFunc) []float64 {
	colIndex := errors.Log1(ix.Table.ColumnIndex(column))
	if colIndex == -1 {
		return nil
	}
	return CountIfIndex(ix, colIndex, iffun)
}

///////////////////////////////////////////////////
//   PropIf

// PropIfIndex returns the proportion (0-1) of true return values for given IfFunc on
// non-Null, non-NaN elements in given IndexView indexed view of an
// table.Table, for given column index.
// Return value(s) is size of column cell: 1 for scalar 1D columns
// and N for higher-dimensional columns.
func PropIfIndex(ix *table.IndexView, colIndex int, iffun IfFunc) []float64 {
	cnt := CountIndex(ix, colIndex)
	if cnt == nil {
		return nil
	}
	pif := CountIfIndex(ix, colIndex, iffun)
	for i := range pif {
		if cnt[i] > 0 {
			pif[i] /= cnt[i]
		}
	}
	return pif
}

// PropIfColumn returns the proportion (0-1) of true return values for given IfFunc on
// non-NaN elements in given IndexView indexed view of an
// table.Table, for given column name.
// If name not found, nil is returned -- use Try version for error message.
// Return value(s) is size of column cell: 1 for scalar 1D columns
// and N for higher-dimensional columns.
func PropIfColumn(ix *table.IndexView, column string, iffun IfFunc) []float64 {
	colIndex := errors.Log1(ix.Table.ColumnIndex(column))
	if colIndex == -1 {
		return nil
	}
	return PropIfIndex(ix, colIndex, iffun)
}

///////////////////////////////////////////////////
//   PctIf

// PctIfIndex returns the percentage (0-100) of true return values for given IfFunc on
// non-Null, non-NaN elements in given IndexView indexed view of an
// table.Table, for given column index.
// Return value(s) is size of column cell: 1 for scalar 1D columns
// and N for higher-dimensional columns.
func PctIfIndex(ix *table.IndexView, colIndex int, iffun IfFunc) []float64 {
	cnt := CountIndex(ix, colIndex)
	if cnt == nil {
		return nil
	}
	pif := CountIfIndex(ix, colIndex, iffun)
	for i := range pif {
		if cnt[i] > 0 {
			pif[i] = 100.0 * (pif[i] / cnt[i])
		}
	}
	return pif
}

// PctIfColumn returns the percentage (0-100) of true return values for given IfFunc on
// non-Null, non-NaN elements in given IndexView indexed view of an
// table.Table, for given column name.
// If name not found, nil is returned -- use Try version for error message.
// Return value(s) is size of column cell: 1 for scalar 1D columns
// and N for higher-dimensional columns.
func PctIfColumn(ix *table.IndexView, column string, iffun IfFunc) []float64 {
	colIndex := errors.Log1(ix.Table.ColumnIndex(column))
	if colIndex == -1 {
		return nil
	}
	return PctIfIndex(ix, colIndex, iffun)
}
