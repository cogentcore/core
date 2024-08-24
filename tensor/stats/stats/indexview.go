// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package stats

import (
	"math"

	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/tensor/table"
)

// Every IndexView Stats method in this file follows one of these signatures:

// IndexViewFuncIndex is a stats function operating on IndexView, taking a column index arg
type IndexViewFuncIndex func(ix *table.IndexView, colIndex int) []float64

// IndexViewFuncColumn is a stats function operating on IndexView, taking a column name arg
type IndexViewFuncColumn func(ix *table.IndexView, column string) []float64

// StatIndex returns IndexView statistic according to given Stats type applied
// to all non-NaN elements in given IndexView indexed view of
// an table.Table, for given column index.
// Return value(s) is size of column cell: 1 for scalar 1D columns
// and N for higher-dimensional columns.
func StatIndex(ix *table.IndexView, colIndex int, stat Stats) []float64 {
	switch stat {
	case Count:
		return CountIndex(ix, colIndex)
	case Sum:
		return SumIndex(ix, colIndex)
	case Prod:
		return ProdIndex(ix, colIndex)
	case Min:
		return MinIndex(ix, colIndex)
	case Max:
		return MaxIndex(ix, colIndex)
	case MinAbs:
		return MinAbsIndex(ix, colIndex)
	case MaxAbs:
		return MaxAbsIndex(ix, colIndex)
	case Mean:
		return MeanIndex(ix, colIndex)
	case Var:
		return VarIndex(ix, colIndex)
	case Std:
		return StdIndex(ix, colIndex)
	case Sem:
		return SemIndex(ix, colIndex)
	case L1Norm:
		return L1NormIndex(ix, colIndex)
	case SumSq:
		return SumSqIndex(ix, colIndex)
	case L2Norm:
		return L2NormIndex(ix, colIndex)
	case VarPop:
		return VarPopIndex(ix, colIndex)
	case StdPop:
		return StdPopIndex(ix, colIndex)
	case SemPop:
		return SemPopIndex(ix, colIndex)
	case Median:
		return MedianIndex(ix, colIndex)
	case Q1:
		return Q1Index(ix, colIndex)
	case Q3:
		return Q3Index(ix, colIndex)
	}
	return nil
}

// StatColumn returns IndexView statistic according to given Stats type applied
// to all non-NaN elements in given IndexView indexed view of
// an table.Table, for given column name.
// If name not found, returns error message.
// Return value(s) is size of column cell: 1 for scalar 1D columns
// and N for higher-dimensional columns.
func StatColumn(ix *table.IndexView, column string, stat Stats) ([]float64, error) {
	colIndex, err := ix.Table.ColumnIndex(column)
	if err != nil {
		return nil, err
	}
	rv := StatIndex(ix, colIndex, stat)
	return rv, nil
}

// StatIndexFunc applies given StatFunc function to each element in the given column,
// using float64 conversions of the values.  ini is the initial value for the agg variable.
// Operates independently over each cell on n-dimensional columns and returns the result as a slice
// of values per cell.
func StatIndexFunc(ix *table.IndexView, colIndex int, ini float64, fun StatFunc) []float64 {
	cl := ix.Table.Columns[colIndex]
	_, csz := cl.RowCellSize()

	ag := make([]float64, csz)
	for i := range ag {
		ag[i] = ini
	}
	if csz == 1 {
		for _, srw := range ix.Indexes {
			val := cl.Float1D(srw)
			if !math.IsNaN(val) {
				ag[0] = fun(srw, val, ag[0])
			}
		}
	} else {
		for _, srw := range ix.Indexes {
			si := srw * csz
			for j := range ag {
				val := cl.Float1D(si + j)
				if !math.IsNaN(val) {
					ag[j] = fun(si+j, val, ag[j])
				}
			}
		}
	}
	return ag
}

///////////////////////////////////////////////////
//   Count

// CountIndex returns the count of non-NaN elements in given
// IndexView indexed view of an table.Table, for given column index.
// Return value is size of each column cell -- 1 for scalar 1D columns
// and N for higher-dimensional columns.
func CountIndex(ix *table.IndexView, colIndex int) []float64 {
	return StatIndexFunc(ix, colIndex, 0, CountFunc)
}

// CountColumn returns the count of non-NaN elements in given
// IndexView indexed view of an table.Table, for given column name.
// If name not found, nil is returned.
// Return value is size of each column cell -- 1 for scalar 1D columns
// and N for higher-dimensional columns.
func CountColumn(ix *table.IndexView, column string) []float64 {
	colIndex := errors.Log1(ix.Table.ColumnIndex(column))
	if colIndex == -1 {
		return nil
	}
	return CountIndex(ix, colIndex)
}

///////////////////////////////////////////////////
//   Sum

// SumIndex returns the sum of non-NaN elements in given
// IndexView indexed view of an table.Table, for given column index.
// Return value is size of each column cell -- 1 for scalar 1D columns
// and N for higher-dimensional columns.
func SumIndex(ix *table.IndexView, colIndex int) []float64 {
	return StatIndexFunc(ix, colIndex, 0, SumFunc)
}

// SumColumn returns the sum of non-NaN elements in given
// IndexView indexed view of an table.Table, for given column name.
// If name not found, nil is returned.
// Return value is size of each column cell -- 1 for scalar 1D columns
// and N for higher-dimensional columns.
func SumColumn(ix *table.IndexView, column string) []float64 {
	colIndex := errors.Log1(ix.Table.ColumnIndex(column))
	if colIndex == -1 {
		return nil
	}
	return SumIndex(ix, colIndex)
}

///////////////////////////////////////////////////
//   Prod

// ProdIndex returns the product of non-NaN elements in given
// IndexView indexed view of an table.Table, for given column index.
// Return value is size of each column cell -- 1 for scalar 1D columns
// and N for higher-dimensional columns.
func ProdIndex(ix *table.IndexView, colIndex int) []float64 {
	return StatIndexFunc(ix, colIndex, 1, ProdFunc)
}

// ProdColumn returns the product of non-NaN elements in given
// IndexView indexed view of an table.Table, for given column name.
// If name not found, nil is returned.
// Return value is size of each column cell -- 1 for scalar 1D columns
// and N for higher-dimensional columns.
func ProdColumn(ix *table.IndexView, column string) []float64 {
	colIndex := errors.Log1(ix.Table.ColumnIndex(column))
	if colIndex == -1 {
		return nil
	}
	return ProdIndex(ix, colIndex)
}

///////////////////////////////////////////////////
//   Min

// MinIndex returns the minimum of non-NaN elements in given
// IndexView indexed view of an table.Table, for given column index.
// Return value is size of each column cell -- 1 for scalar 1D columns
// and N for higher-dimensional columns.
func MinIndex(ix *table.IndexView, colIndex int) []float64 {
	return StatIndexFunc(ix, colIndex, math.MaxFloat64, MinFunc)
}

// MinColumn returns the minimum of non-NaN elements in given
// IndexView indexed view of an table.Table, for given column name.
// If name not found, nil is returned.
// Return value is size of each column cell -- 1 for scalar 1D columns
// and N for higher-dimensional columns.
func MinColumn(ix *table.IndexView, column string) []float64 {
	colIndex := errors.Log1(ix.Table.ColumnIndex(column))
	if colIndex == -1 {
		return nil
	}
	return MinIndex(ix, colIndex)
}

///////////////////////////////////////////////////
//   Max

// MaxIndex returns the maximum of non-NaN elements in given
// IndexView indexed view of an table.Table, for given column index.
// Return value is size of each column cell -- 1 for scalar 1D columns
// and N for higher-dimensional columns.
func MaxIndex(ix *table.IndexView, colIndex int) []float64 {
	return StatIndexFunc(ix, colIndex, -math.MaxFloat64, MaxFunc)
}

// MaxColumn returns the maximum of non-NaN elements in given
// IndexView indexed view of an table.Table, for given column name.
// If name not found, nil is returned.
// Return value is size of each column cell -- 1 for scalar 1D columns
// and N for higher-dimensional columns.
func MaxColumn(ix *table.IndexView, column string) []float64 {
	colIndex := errors.Log1(ix.Table.ColumnIndex(column))
	if colIndex == -1 {
		return nil
	}
	return MaxIndex(ix, colIndex)
}

///////////////////////////////////////////////////
//   MinAbs

// MinAbsIndex returns the minimum of abs of non-NaN elements in given
// IndexView indexed view of an table.Table, for given column index.
// Return value is size of each column cell -- 1 for scalar 1D columns
// and N for higher-dimensional columns.
func MinAbsIndex(ix *table.IndexView, colIndex int) []float64 {
	return StatIndexFunc(ix, colIndex, math.MaxFloat64, MinAbsFunc)
}

// MinAbsColumn returns the minimum of abs of non-NaN elements in given
// IndexView indexed view of an table.Table, for given column name.
// If name not found, nil is returned.
// Return value is size of each column cell -- 1 for scalar 1D columns
// and N for higher-dimensional columns.
func MinAbsColumn(ix *table.IndexView, column string) []float64 {
	colIndex := errors.Log1(ix.Table.ColumnIndex(column))
	if colIndex == -1 {
		return nil
	}
	return MinAbsIndex(ix, colIndex)
}

///////////////////////////////////////////////////
//   MaxAbs

// MaxAbsIndex returns the maximum of abs of non-NaN elements in given
// IndexView indexed view of an table.Table, for given column index.
// Return value is size of each column cell -- 1 for scalar 1D columns
// and N for higher-dimensional columns.
func MaxAbsIndex(ix *table.IndexView, colIndex int) []float64 {
	return StatIndexFunc(ix, colIndex, -math.MaxFloat64, MaxAbsFunc)
}

// MaxAbsColumn returns the maximum of abs of non-NaN elements in given
// IndexView indexed view of an table.Table, for given column name.
// If name not found, nil is returned.
// Return value is size of each column cell -- 1 for scalar 1D columns
// and N for higher-dimensional columns.
func MaxAbsColumn(ix *table.IndexView, column string) []float64 {
	colIndex := errors.Log1(ix.Table.ColumnIndex(column))
	if colIndex == -1 {
		return nil
	}
	return MaxAbsIndex(ix, colIndex)
}

///////////////////////////////////////////////////
//   Mean

// MeanIndex returns the mean of non-NaN elements in given
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

// MeanColumn returns the mean of non-NaN elements in given
// IndexView indexed view of an table.Table, for given column name.
// If name not found, nil is returned.
// Return value is size of each column cell -- 1 for scalar 1D columns
// and N for higher-dimensional columns.
func MeanColumn(ix *table.IndexView, column string) []float64 {
	colIndex := errors.Log1(ix.Table.ColumnIndex(column))
	if colIndex == -1 {
		return nil
	}
	return MeanIndex(ix, colIndex)
}

///////////////////////////////////////////////////
//   Var

// VarIndex returns the sample variance of non-NaN elements in given
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
	vr := StatIndexFunc(ix, colIndex, 0, func(idx int, val float64, agg float64) float64 {
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

// VarColumn returns the sample variance of non-NaN elements in given
// IndexView indexed view of an table.Table, for given column name.
// Sample variance is normalized by 1/(n-1) -- see VarPop version for 1/n normalization.
// If name not found, nil is returned.
// Return value is size of each column cell -- 1 for scalar 1D columns
// and N for higher-dimensional columns.
func VarColumn(ix *table.IndexView, column string) []float64 {
	colIndex := errors.Log1(ix.Table.ColumnIndex(column))
	if colIndex == -1 {
		return nil
	}
	return VarIndex(ix, colIndex)
}

///////////////////////////////////////////////////
//   Std

// StdIndex returns the sample std deviation of non-NaN elements in given
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

// StdColumn returns the sample std deviation of non-NaN elements in given
// IndexView indexed view of an table.Table, for given column name.
// Sample std deviation is normalized by 1/(n-1) -- see StdPop version for 1/n normalization.
// If name not found, nil is returned.
// Return value is size of each column cell -- 1 for scalar 1D columns
// and N for higher-dimensional columns.
func StdColumn(ix *table.IndexView, column string) []float64 {
	colIndex := errors.Log1(ix.Table.ColumnIndex(column))
	if colIndex == -1 {
		return nil
	}
	return StdIndex(ix, colIndex)
}

///////////////////////////////////////////////////
//   Sem

// SemIndex returns the sample standard error of the mean of non-NaN elements in given
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

// SemColumn returns the sample standard error of the mean of non-NaN elements in given
// IndexView indexed view of an table.Table, for given column name.
// Sample sem is normalized by 1/(n-1) -- see SemPop version for 1/n normalization.
// If name not found, nil is returned.
// Return value is size of each column cell -- 1 for scalar 1D columns
// and N for higher-dimensional columns.
func SemColumn(ix *table.IndexView, column string) []float64 {
	colIndex := errors.Log1(ix.Table.ColumnIndex(column))
	if colIndex == -1 {
		return nil
	}
	return SemIndex(ix, colIndex)
}

///////////////////////////////////////////////////
//   L1Norm

// L1NormIndex returns the L1 norm (sum abs values) of non-NaN elements in given
// IndexView indexed view of an table.Table, for given column index.
// Return value is size of each column cell -- 1 for scalar 1D columns
// and N for higher-dimensional columns.
func L1NormIndex(ix *table.IndexView, colIndex int) []float64 {
	return StatIndexFunc(ix, colIndex, 0, L1NormFunc)
}

// L1NormColumn returns the L1 norm (sum abs values) of non-NaN elements in given
// IndexView indexed view of an table.Table, for given column name.
// If name not found, nil is returned.
// Return value is size of each column cell -- 1 for scalar 1D columns
// and N for higher-dimensional columns.
func L1NormColumn(ix *table.IndexView, column string) []float64 {
	colIndex := errors.Log1(ix.Table.ColumnIndex(column))
	if colIndex == -1 {
		return nil
	}
	return L1NormIndex(ix, colIndex)
}

///////////////////////////////////////////////////
//   SumSq

// SumSqIndex returns the sum-of-squares of non-NaN elements in given
// IndexView indexed view of an table.Table, for given column index.
// Return value is size of each column cell -- 1 for scalar 1D columns
// and N for higher-dimensional columns.
func SumSqIndex(ix *table.IndexView, colIndex int) []float64 {
	cl := ix.Table.Columns[colIndex]
	_, csz := cl.RowCellSize()

	scale := make([]float64, csz)
	ss := make([]float64, csz)
	for i := range ss {
		ss[i] = 1
	}
	n := len(ix.Indexes)
	if csz == 1 {
		if n < 2 {
			if n == 1 {
				ss[0] = math.Abs(cl.Float1D(ix.Indexes[0]))
				return ss
			}
			return scale // all 0s
		}
		for _, srw := range ix.Indexes {
			v := cl.Float1D(srw)
			absxi := math.Abs(v)
			if scale[0] < absxi {
				ss[0] = 1 + ss[0]*(scale[0]/absxi)*(scale[0]/absxi)
				scale[0] = absxi
			} else {
				ss[0] = ss[0] + (absxi/scale[0])*(absxi/scale[0])
			}
		}
		if math.IsInf(scale[0], 1) {
			ss[0] = math.Inf(1)
		} else {
			ss[0] = scale[0] * scale[0] * ss[0]
		}
	} else {
		if n < 2 {
			if n == 1 {
				si := csz * ix.Indexes[0]
				for j := range csz {
					ss[j] = math.Abs(cl.Float1D(si + j))
				}
				return ss
			}
			return scale // all 0s
		}
		for _, srw := range ix.Indexes {
			si := srw * csz
			for j := range ss {
				v := cl.Float1D(si + j)
				absxi := math.Abs(v)
				if scale[j] < absxi {
					ss[j] = 1 + ss[j]*(scale[j]/absxi)*(scale[j]/absxi)
					scale[j] = absxi
				} else {
					ss[j] = ss[j] + (absxi/scale[j])*(absxi/scale[j])
				}
			}
		}
		for j := range ss {
			if math.IsInf(scale[j], 1) {
				ss[j] = math.Inf(1)
			} else {
				ss[j] = scale[j] * scale[j] * ss[j]
			}
		}
	}
	return ss
}

// SumSqColumn returns the sum-of-squares of non-NaN elements in given
// IndexView indexed view of an table.Table, for given column name.
// If name not found, nil is returned.
// Return value is size of each column cell -- 1 for scalar 1D columns
// and N for higher-dimensional columns.
func SumSqColumn(ix *table.IndexView, column string) []float64 {
	colIndex := errors.Log1(ix.Table.ColumnIndex(column))
	if colIndex == -1 {
		return nil
	}
	return SumSqIndex(ix, colIndex)
}

///////////////////////////////////////////////////
//   L2Norm

// L2NormIndex returns the L2 norm (square root of sum-of-squares) of non-NaN elements in given
// IndexView indexed view of an table.Table, for given column index.
// Return value is size of each column cell -- 1 for scalar 1D columns
// and N for higher-dimensional columns.
func L2NormIndex(ix *table.IndexView, colIndex int) []float64 {
	ss := SumSqIndex(ix, colIndex)
	for i := range ss {
		ss[i] = math.Sqrt(ss[i])
	}
	return ss
}

// L2NormColumn returns the L2 norm (square root of sum-of-squares) of non-NaN elements in given
// IndexView indexed view of an table.Table, for given column name.
// If name not found, nil is returned.
// Return value is size of each column cell -- 1 for scalar 1D columns
// and N for higher-dimensional columns.
func L2NormColumn(ix *table.IndexView, column string) []float64 {
	colIndex := errors.Log1(ix.Table.ColumnIndex(column))
	if colIndex == -1 {
		return nil
	}
	return L2NormIndex(ix, colIndex)
}

///////////////////////////////////////////////////
//   VarPop

// VarPopIndex returns the population variance of non-NaN elements in given
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
	vr := StatIndexFunc(ix, colIndex, 0, func(idx int, val float64, agg float64) float64 {
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

// VarPopColumn returns the population variance of non-NaN elements in given
// IndexView indexed view of an table.Table, for given column name.
// population variance is normalized by 1/n -- see Var version for 1/(n-1) sample normalization.
// If name not found, nil is returned.
// Return value is size of each column cell -- 1 for scalar 1D columns
// and N for higher-dimensional columns.
func VarPopColumn(ix *table.IndexView, column string) []float64 {
	colIndex := errors.Log1(ix.Table.ColumnIndex(column))
	if colIndex == -1 {
		return nil
	}
	return VarPopIndex(ix, colIndex)
}

///////////////////////////////////////////////////
//   StdPop

// StdPopIndex returns the population std deviation of non-NaN elements in given
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

// StdPopColumn returns the population std deviation of non-NaN elements in given
// IndexView indexed view of an table.Table, for given column name.
// population std dev is normalized by 1/n -- see Var version for 1/(n-1) sample normalization.
// If name not found, nil is returned.
// Return value is size of each column cell -- 1 for scalar 1D columns
// and N for higher-dimensional columns.
func StdPopColumn(ix *table.IndexView, column string) []float64 {
	colIndex := errors.Log1(ix.Table.ColumnIndex(column))
	if colIndex == -1 {
		return nil
	}
	return StdPopIndex(ix, colIndex)
}

///////////////////////////////////////////////////
//   SemPop

// SemPopIndex returns the population standard error of the mean of non-NaN elements in given
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

// SemPopColumn returns the standard error of the mean of non-NaN elements in given
// IndexView indexed view of an table.Table, for given column name.
// population sem is normalized by 1/n -- see Var version for 1/(n-1) sample normalization.
// If name not found, nil is returned.
// Return value is size of each column cell -- 1 for scalar 1D columns
// and N for higher-dimensional columns.
func SemPopColumn(ix *table.IndexView, column string) []float64 {
	colIndex := errors.Log1(ix.Table.ColumnIndex(column))
	if colIndex == -1 {
		return nil
	}
	return SemPopIndex(ix, colIndex)
}

///////////////////////////////////////////////////
//   Median

// MedianIndex returns the median of non-NaN elements in given
// IndexView indexed view of an table.Table, for given column index.
// Return value is size of each column cell -- 1 for scalar 1D columns
// and N for higher-dimensional columns.
func MedianIndex(ix *table.IndexView, colIndex int) []float64 {
	return QuantilesIndex(ix, colIndex, []float64{.5})
}

// MedianColumn returns the median of non-NaN elements in given
// IndexView indexed view of an table.Table, for given column name.
// If name not found, nil is returned.
// Return value is size of each column cell -- 1 for scalar 1D columns
// and N for higher-dimensional columns.
func MedianColumn(ix *table.IndexView, column string) []float64 {
	colIndex := errors.Log1(ix.Table.ColumnIndex(column))
	if colIndex == -1 {
		return nil
	}
	return MedianIndex(ix, colIndex)
}

///////////////////////////////////////////////////
//   Q1

// Q1Index returns the first quartile of non-NaN elements in given
// IndexView indexed view of an table.Table, for given column index.
// Return value is size of each column cell -- 1 for scalar 1D columns
// and N for higher-dimensional columns.
func Q1Index(ix *table.IndexView, colIndex int) []float64 {
	return QuantilesIndex(ix, colIndex, []float64{.25})
}

// Q1Column returns the first quartile of non-NaN elements in given
// IndexView indexed view of an table.Table, for given column name.
// If name not found, nil is returned.
// Return value is size of each column cell -- 1 for scalar 1D columns
// and N for higher-dimensional columns.
func Q1Column(ix *table.IndexView, column string) []float64 {
	colIndex := errors.Log1(ix.Table.ColumnIndex(column))
	if colIndex == -1 {
		return nil
	}
	return Q1Index(ix, colIndex)
}

///////////////////////////////////////////////////
//   Q3

// Q3Index returns the third quartile of non-NaN elements in given
// IndexView indexed view of an table.Table, for given column index.
// Return value is size of each column cell -- 1 for scalar 1D columns
// and N for higher-dimensional columns.
func Q3Index(ix *table.IndexView, colIndex int) []float64 {
	return QuantilesIndex(ix, colIndex, []float64{.75})
}

// Q3Column returns the third quartile of non-NaN elements in given
// IndexView indexed view of an table.Table, for given column name.
// If name not found, nil is returned.
// Return value is size of each column cell -- 1 for scalar 1D columns
// and N for higher-dimensional columns.
func Q3Column(ix *table.IndexView, column string) []float64 {
	colIndex := errors.Log1(ix.Table.ColumnIndex(column))
	if colIndex == -1 {
		return nil
	}
	return Q3Index(ix, colIndex)
}
