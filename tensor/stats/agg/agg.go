// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package agg

import (
	"math"

	"cogentcore.org/core/tensor/table"
)

// Every standard Agg method in this file follows one of these signatures:

// IndexViewAggFuncIndex is an aggregation function operating on IndexView, taking a column index arg
type IndexViewAggFuncIndex func(ix *table.IndexView, colIndex int) []float64

// IndexViewAggFunc is an aggregation function operating on IndexView, taking a column name arg
type IndexViewAggFunc func(ix *table.IndexView, column string) []float64

// IndexViewAggFuncTry is an aggregation function operating on IndexView, taking a column name arg,
// returning an error message
type IndexViewAggFuncTry func(ix *table.IndexView, colIndex int) ([]float64, error)

///////////////////////////////////////////////////
//   Count

// CountIndex returns the count of non-Null, non-NaN elements in given
// IndexView indexed view of an table.Table, for given column index.
// Return value is size of each column cell -- 1 for scalar 1D columns
// and N for higher-dimensional columns.
func CountIndex(ix *table.IndexView, colIndex int) []float64 {
	return ix.AggColumn(colIndex, 0, CountFunc)
}

// Count returns the count of non-Null, non-NaN elements in given
// IndexView indexed view of an table.Table, for given column name.
// If name not found, nil is returned -- use Try version for error message.
// Return value is size of each column cell -- 1 for scalar 1D columns
// and N for higher-dimensional columns.
func Count(ix *table.IndexView, column string) []float64 {
	colIndex := ix.Table.ColumnIndex(column)
	if colIndex == -1 {
		return nil
	}
	return CountIndex(ix, colIndex)
}

///////////////////////////////////////////////////
//   Sum

// SumIndex returns the sum of non-Null, non-NaN elements in given
// IndexView indexed view of an table.Table, for given column index.
// Return value is size of each column cell -- 1 for scalar 1D columns
// and N for higher-dimensional columns.
func SumIndex(ix *table.IndexView, colIndex int) []float64 {
	return ix.AggColumn(colIndex, 0, SumFunc)
}

// Sum returns the sum of non-Null, non-NaN elements in given
// IndexView indexed view of an table.Table, for given column name.
// If name not found, nil is returned -- use Try version for error message.
// Return value is size of each column cell -- 1 for scalar 1D columns
// and N for higher-dimensional columns.
func Sum(ix *table.IndexView, column string) []float64 {
	colIndex := ix.Table.ColumnIndex(column)
	if colIndex == -1 {
		return nil
	}
	return SumIndex(ix, colIndex)
}

///////////////////////////////////////////////////
//   Prod

// ProdIndex returns the product of non-Null, non-NaN elements in given
// IndexView indexed view of an table.Table, for given column index.
// Return value is size of each column cell -- 1 for scalar 1D columns
// and N for higher-dimensional columns.
func ProdIndex(ix *table.IndexView, colIndex int) []float64 {
	return ix.AggColumn(colIndex, 1, ProdFunc)
}

// Prod returns the product of non-Null, non-NaN elements in given
// IndexView indexed view of an table.Table, for given column name.
// If name not found, nil is returned -- use Try version for error message.
// Return value is size of each column cell -- 1 for scalar 1D columns
// and N for higher-dimensional columns.
func Prod(ix *table.IndexView, column string) []float64 {
	colIndex := ix.Table.ColumnIndex(column)
	if colIndex == -1 {
		return nil
	}
	return ProdIndex(ix, colIndex)
}

///////////////////////////////////////////////////
//   Max

// MaxIndex returns the maximum of non-Null, non-NaN elements in given
// IndexView indexed view of an table.Table, for given column index.
// Return value is size of each column cell -- 1 for scalar 1D columns
// and N for higher-dimensional columns.
func MaxIndex(ix *table.IndexView, colIndex int) []float64 {
	return ix.AggColumn(colIndex, -math.MaxFloat64, MaxFunc)
}

// Max returns the maximum of non-Null, non-NaN elements in given
// IndexView indexed view of an table.Table, for given column name.
// If name not found, nil is returned -- use Try version for error message.
// Return value is size of each column cell -- 1 for scalar 1D columns
// and N for higher-dimensional columns.
func Max(ix *table.IndexView, column string) []float64 {
	colIndex := ix.Table.ColumnIndex(column)
	if colIndex == -1 {
		return nil
	}
	return MaxIndex(ix, colIndex)
}

///////////////////////////////////////////////////
//   Min

// MinIndex returns the minimum of non-Null, non-NaN elements in given
// IndexView indexed view of an table.Table, for given column index.
// Return value is size of each column cell -- 1 for scalar 1D columns
// and N for higher-dimensional columns.
func MinIndex(ix *table.IndexView, colIndex int) []float64 {
	return ix.AggColumn(colIndex, math.MaxFloat64, MinFunc)
}

// Min returns the minimum of non-Null, non-NaN elements in given
// IndexView indexed view of an table.Table, for given column name.
// If name not found, nil is returned -- use Try version for error message.
// Return value is size of each column cell -- 1 for scalar 1D columns
// and N for higher-dimensional columns.
func Min(ix *table.IndexView, column string) []float64 {
	colIndex := ix.Table.ColumnIndex(column)
	if colIndex == -1 {
		return nil
	}
	return MinIndex(ix, colIndex)
}

///////////////////////////////////////////////////
//   Mean

// MeanIndex returns the mean of non-Null, non-NaN elements in given
// IndexView indexed view of an table.Table, for given column index.
// Return value is size of each column cell -- 1 for scalar 1D columns
// and N for higher-dimensional columns.
func MeanIndex(ix *table.IndexView, colIndex int) []float64 {
	cnt := CountIndex(ix, colIndex)
	if cnt == nil {
		return nil
	}
	mean := SumIndex(ix, colIndex)
	for i := range mean {
		if cnt[i] > 0 {
			mean[i] /= cnt[i]
		}
	}
	return mean
}

// Mean returns the mean of non-Null, non-NaN elements in given
// IndexView indexed view of an table.Table, for given column name.
// If name not found, nil is returned -- use Try version for error message.
// Return value is size of each column cell -- 1 for scalar 1D columns
// and N for higher-dimensional columns.
func Mean(ix *table.IndexView, column string) []float64 {
	colIndex := ix.Table.ColumnIndex(column)
	if colIndex == -1 {
		return nil
	}
	return MeanIndex(ix, colIndex)
}

///////////////////////////////////////////////////
//   Var

// VarIndex returns the sample variance of non-Null, non-NaN elements in given
// IndexView indexed view of an table.Table, for given column index.
// Sample variance is normalized by 1/(n-1) -- see VarPop version for 1/n normalization.
// Return value is size of each column cell -- 1 for scalar 1D columns
// and N for higher-dimensional columns.
func VarIndex(ix *table.IndexView, colIndex int) []float64 {
	cnt := CountIndex(ix, colIndex)
	if cnt == nil {
		return nil
	}
	mean := SumIndex(ix, colIndex)
	for i := range mean {
		if cnt[i] > 0 {
			mean[i] /= cnt[i]
		}
	}
	col := ix.Table.Columns[colIndex]
	_, csz := col.RowCellSize()
	vr := ix.AggColumn(colIndex, 0, func(idx int, val float64, agg float64) float64 {
		cidx := idx % csz
		dv := val - mean[cidx]
		return agg + dv*dv
	})
	for i := range vr {
		if cnt[i] > 1 {
			vr[i] /= (cnt[i] - 1)
		}
	}
	return vr
}

// Var returns the sample variance of non-Null, non-NaN elements in given
// IndexView indexed view of an table.Table, for given column name.
// Sample variance is normalized by 1/(n-1) -- see VarPop version for 1/n normalization.
// If name not found, nil is returned -- use Try version for error message.
// Return value is size of each column cell -- 1 for scalar 1D columns
// and N for higher-dimensional columns.
func Var(ix *table.IndexView, column string) []float64 {
	colIndex := ix.Table.ColumnIndex(column)
	if colIndex == -1 {
		return nil
	}
	return VarIndex(ix, colIndex)
}

///////////////////////////////////////////////////
//   Std

// StdIndex returns the sample std deviation of non-Null, non-NaN elements in given
// IndexView indexed view of an table.Table, for given column index.
// Sample std deviation is normalized by 1/(n-1) -- see StdPop version for 1/n normalization.
// Return value is size of each column cell -- 1 for scalar 1D columns
// and N for higher-dimensional columns.
func StdIndex(ix *table.IndexView, colIndex int) []float64 {
	std := VarIndex(ix, colIndex)
	for i := range std {
		std[i] = math.Sqrt(std[i])
	}
	return std
}

// Std returns the sample std deviation of non-Null, non-NaN elements in given
// IndexView indexed view of an table.Table, for given column name.
// Sample std deviation is normalized by 1/(n-1) -- see StdPop version for 1/n normalization.
// If name not found, nil is returned -- use Try version for error message.
// Return value is size of each column cell -- 1 for scalar 1D columns
// and N for higher-dimensional columns.
func Std(ix *table.IndexView, column string) []float64 {
	colIndex := ix.Table.ColumnIndex(column)
	if colIndex == -1 {
		return nil
	}
	return StdIndex(ix, colIndex)
}

///////////////////////////////////////////////////
//   Sem

// SemIndex returns the sample standard error of the mean of non-Null, non-NaN elements in given
// IndexView indexed view of an table.Table, for given column index.
// Sample sem is normalized by 1/(n-1) -- see SemPop version for 1/n normalization.
// Return value is size of each column cell -- 1 for scalar 1D columns
// and N for higher-dimensional columns.
func SemIndex(ix *table.IndexView, colIndex int) []float64 {
	cnt := CountIndex(ix, colIndex)
	if cnt == nil {
		return nil
	}
	sem := StdIndex(ix, colIndex)
	for i := range sem {
		if cnt[i] > 0 {
			sem[i] /= math.Sqrt(cnt[i])
		}
	}
	return sem
}

// Sem returns the sample standard error of the mean of non-Null, non-NaN elements in given
// IndexView indexed view of an table.Table, for given column name.
// Sample sem is normalized by 1/(n-1) -- see SemPop version for 1/n normalization.
// If name not found, nil is returned -- use Try version for error message.
// Return value is size of each column cell -- 1 for scalar 1D columns
// and N for higher-dimensional columns.
func Sem(ix *table.IndexView, column string) []float64 {
	colIndex := ix.Table.ColumnIndex(column)
	if colIndex == -1 {
		return nil
	}
	return SemIndex(ix, colIndex)
}

///////////////////////////////////////////////////
//   VarPop

// VarPopIndex returns the population variance of non-Null, non-NaN elements in given
// IndexView indexed view of an table.Table, for given column index.
// population variance is normalized by 1/n -- see Var version for 1/(n-1) sample normalization.
// Return value is size of each column cell -- 1 for scalar 1D columns
// and N for higher-dimensional columns.
func VarPopIndex(ix *table.IndexView, colIndex int) []float64 {
	cnt := CountIndex(ix, colIndex)
	if cnt == nil {
		return nil
	}
	mean := SumIndex(ix, colIndex)
	for i := range mean {
		if cnt[i] > 0 {
			mean[i] /= cnt[i]
		}
	}
	col := ix.Table.Columns[colIndex]
	_, csz := col.RowCellSize()
	vr := ix.AggColumn(colIndex, 0, func(idx int, val float64, agg float64) float64 {
		cidx := idx % csz
		dv := val - mean[cidx]
		return agg + dv*dv
	})
	for i := range vr {
		if cnt[i] > 0 {
			vr[i] /= cnt[i]
		}
	}
	return vr
}

// VarPop returns the population variance of non-Null, non-NaN elements in given
// IndexView indexed view of an table.Table, for given column name.
// population variance is normalized by 1/n -- see Var version for 1/(n-1) sample normalization.
// If name not found, nil is returned -- use Try version for error message.
// Return value is size of each column cell -- 1 for scalar 1D columns
// and N for higher-dimensional columns.
func VarPop(ix *table.IndexView, column string) []float64 {
	colIndex := ix.Table.ColumnIndex(column)
	if colIndex == -1 {
		return nil
	}
	return VarPopIndex(ix, colIndex)
}

///////////////////////////////////////////////////
//   StdPop

// StdPopIndex returns the population std deviation of non-Null, non-NaN elements in given
// IndexView indexed view of an table.Table, for given column index.
// population std dev is normalized by 1/n -- see Var version for 1/(n-1) sample normalization.
// Return value is size of each column cell -- 1 for scalar 1D columns
// and N for higher-dimensional columns.
func StdPopIndex(ix *table.IndexView, colIndex int) []float64 {
	std := VarPopIndex(ix, colIndex)
	for i := range std {
		std[i] = math.Sqrt(std[i])
	}
	return std
}

// StdPop returns the population std deviation of non-Null, non-NaN elements in given
// IndexView indexed view of an table.Table, for given column name.
// population std dev is normalized by 1/n -- see Var version for 1/(n-1) sample normalization.
// If name not found, nil is returned -- use Try version for error message.
// Return value is size of each column cell -- 1 for scalar 1D columns
// and N for higher-dimensional columns.
func StdPop(ix *table.IndexView, column string) []float64 {
	colIndex := ix.Table.ColumnIndex(column)
	if colIndex == -1 {
		return nil
	}
	return StdPopIndex(ix, colIndex)
}

///////////////////////////////////////////////////
//   SemPop

// SemPopIndex returns the population standard error of the mean of non-Null, non-NaN elements in given
// IndexView indexed view of an table.Table, for given column index.
// population sem is normalized by 1/n -- see Var version for 1/(n-1) sample normalization.
// Return value is size of each column cell -- 1 for scalar 1D columns
// and N for higher-dimensional columns.
func SemPopIndex(ix *table.IndexView, colIndex int) []float64 {
	cnt := CountIndex(ix, colIndex)
	if cnt == nil {
		return nil
	}
	sem := StdPopIndex(ix, colIndex)
	for i := range sem {
		if cnt[i] > 0 {
			sem[i] /= math.Sqrt(cnt[i])
		}
	}
	return sem
}

// SemPop returns the standard error of the mean of non-Null, non-NaN elements in given
// IndexView indexed view of an table.Table, for given column name.
// population sem is normalized by 1/n -- see Var version for 1/(n-1) sample normalization.
// If name not found, nil is returned -- use Try version for error message.
// Return value is size of each column cell -- 1 for scalar 1D columns
// and N for higher-dimensional columns.
func SemPop(ix *table.IndexView, column string) []float64 {
	colIndex := ix.Table.ColumnIndex(column)
	if colIndex == -1 {
		return nil
	}
	return SemPopIndex(ix, colIndex)
}

///////////////////////////////////////////////////
//   SumSq

// SumSqIndex returns the sum-of-squares of non-Null, non-NaN elements in given
// IndexView indexed view of an table.Table, for given column index.
// Return value is size of each column cell -- 1 for scalar 1D columns
// and N for higher-dimensional columns.
func SumSqIndex(ix *table.IndexView, colIndex int) []float64 {
	return ix.AggColumn(colIndex, 0, SumSqFunc)
}

// SumSq returns the sum-of-squares of non-Null, non-NaN elements in given
// IndexView indexed view of an table.Table, for given column name.
// If name not found, nil is returned -- use Try version for error message.
// Return value is size of each column cell -- 1 for scalar 1D columns
// and N for higher-dimensional columns.
func SumSq(ix *table.IndexView, column string) []float64 {
	colIndex := ix.Table.ColumnIndex(column)
	if colIndex == -1 {
		return nil
	}
	return SumSqIndex(ix, colIndex)
}

///////////////////////////////////////////////////
//   Median

// MedianIndex returns the median of non-Null, non-NaN elements in given
// IndexView indexed view of an table.Table, for given column index.
// Return value is size of each column cell -- 1 for scalar 1D columns
// and N for higher-dimensional columns.
func MedianIndex(ix *table.IndexView, colIndex int) []float64 {
	return QuantilesIndex(ix, colIndex, []float64{.5})
}

// Median returns the median of non-Null, non-NaN elements in given
// IndexView indexed view of an table.Table, for given column name.
// If name not found, nil is returned -- use Try version for error message.
// Return value is size of each column cell -- 1 for scalar 1D columns
// and N for higher-dimensional columns.
func Median(ix *table.IndexView, column string) []float64 {
	colIndex := ix.Table.ColumnIndex(column)
	if colIndex == -1 {
		return nil
	}
	return MedianIndex(ix, colIndex)
}

///////////////////////////////////////////////////
//   Q1

// Q1Index returns the first quartile of non-Null, non-NaN elements in given
// IndexView indexed view of an table.Table, for given column index.
// Return value is size of each column cell -- 1 for scalar 1D columns
// and N for higher-dimensional columns.
func Q1Index(ix *table.IndexView, colIndex int) []float64 {
	return QuantilesIndex(ix, colIndex, []float64{.25})
}

// Q1 returns the first quartile of non-Null, non-NaN elements in given
// IndexView indexed view of an table.Table, for given column name.
// If name not found, nil is returned -- use Try version for error message.
// Return value is size of each column cell -- 1 for scalar 1D columns
// and N for higher-dimensional columns.
func Q1(ix *table.IndexView, column string) []float64 {
	colIndex := ix.Table.ColumnIndex(column)
	if colIndex == -1 {
		return nil
	}
	return Q1Index(ix, colIndex)
}

///////////////////////////////////////////////////
//   Q3

// Q3Index returns the third quartile of non-Null, non-NaN elements in given
// IndexView indexed view of an table.Table, for given column index.
// Return value is size of each column cell -- 1 for scalar 1D columns
// and N for higher-dimensional columns.
func Q3Index(ix *table.IndexView, colIndex int) []float64 {
	return QuantilesIndex(ix, colIndex, []float64{.75})
}

// Q3 returns the third quartile of non-Null, non-NaN elements in given
// IndexView indexed view of an table.Table, for given column name.
// If name not found, nil is returned -- use Try version for error message.
// Return value is size of each column cell -- 1 for scalar 1D columns
// and N for higher-dimensional columns.
func Q3(ix *table.IndexView, column string) []float64 {
	colIndex := ix.Table.ColumnIndex(column)
	if colIndex == -1 {
		return nil
	}
	return Q3Index(ix, colIndex)
}
