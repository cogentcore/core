// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package agg

//go:generate core generate

import (
	"fmt"
	"strings"

	"cogentcore.org/core/tensor/table"
)

// Aggs is a list of different standard aggregation functions, which can be used
// to choose an aggregation function
type Aggs int32 //enums:enum

const (
	// Count of number of elements
	AggCount Aggs = iota

	// Sum of elements
	AggSum

	// Product of elements
	AggProd

	// Min minimum value
	AggMin

	// Max maximum value
	AggMax

	// Mean mean value
	AggMean

	// Var sample variance (squared diffs from mean, divided by n-1)
	AggVar

	// Std sample standard deviation (sqrt of Var)
	AggStd

	// Sem sample standard error of the mean (Std divided by sqrt(n))
	AggSem

	// VarPop population variance (squared diffs from mean, divided by n)
	AggVarPop

	// StdPop population standard deviation (sqrt of VarPop)
	AggStdPop

	// SemPop population standard error of the mean (StdPop divided by sqrt(n))
	AggSemPop

	// Median middle value in sorted ordering
	AggMedian

	// Q1 first quartile = 25%ile value = .25 quantile value
	AggQ1

	// Q3 third quartile = 75%ile value = .75 quantile value
	AggQ3

	// SumSq sum of squares
	AggSumSq
)

// AggsName returns the name of the Aggs varaible without the Agg prefix..
func AggsName(ag Aggs) string {
	return strings.TrimPrefix(ag.String(), "Agg")
}

// AggIndex returns aggregate according to given agg type applied
// to all non-Null, non-NaN elements in given IndexView indexed view of
// an table.Table, for given column index.
// valid names are: Count, Sum, Var, Std, Sem, VarPop, StdPop, SemPop,
// Min, Max, SumSq, 25%, 1Q, Median, 50%, 2Q, 75%, 3Q (case insensitive)
// Return value is size of each column cell -- 1 for scalar 1D columns
// and N for higher-dimensional columns.
func AggIndex(ix *table.IndexView, colIndex int, ag Aggs) []float64 {
	switch ag {
	case AggCount:
		return CountIndex(ix, colIndex)
	case AggSum:
		return SumIndex(ix, colIndex)
	case AggProd:
		return ProdIndex(ix, colIndex)
	case AggMin:
		return MinIndex(ix, colIndex)
	case AggMax:
		return MaxIndex(ix, colIndex)
	case AggMean:
		return MeanIndex(ix, colIndex)
	case AggVar:
		return VarIndex(ix, colIndex)
	case AggStd:
		return StdIndex(ix, colIndex)
	case AggSem:
		return SemIndex(ix, colIndex)
	case AggVarPop:
		return VarPopIndex(ix, colIndex)
	case AggStdPop:
		return StdPopIndex(ix, colIndex)
	case AggSemPop:
		return SemPopIndex(ix, colIndex)
	case AggQ1:
		return Q1Index(ix, colIndex)
	case AggMedian:
		return MedianIndex(ix, colIndex)
	case AggQ3:
		return Q3Index(ix, colIndex)
	case AggSumSq:
		return SumSqIndex(ix, colIndex)
	}
	return nil
}

// Agg returns aggregate according to given agg type applied
// to all non-Null, non-NaN elements in given IndexView indexed view of
// an table.Table, for given column name.
// valid names are: Count, Sum, Var, Std, Sem, VarPop, StdPop, SemPop,
// Min, Max, SumSq (case insensitive)
// If name not found, nil is returned -- use Try version for error message.
// Return value is size of each column cell -- 1 for scalar 1D columns
// and N for higher-dimensional columns.
func Agg(ix *table.IndexView, colNm string, ag Aggs) []float64 {
	colIndex := ix.Table.ColumnIndex(colNm)
	if colIndex == -1 {
		return nil
	}
	return AggIndex(ix, colIndex, ag)
}

// AggTry returns aggregate according to given agg type applied
// to all non-Null, non-NaN elements in given IndexView indexed view of
// an table.Table, for given column name.
// valid names are: Count, Sum, Var, Std, Sem, VarPop, StdPop, SemPop,
// Min, Max, SumSq (case insensitive)
// If col name not found, returns error message.
// Return value is size of each column cell -- 1 for scalar 1D columns
// and N for higher-dimensional columns.
func AggTry(ix *table.IndexView, colNm string, ag Aggs) ([]float64, error) {
	colIndex, err := ix.Table.ColumnIndexTry(colNm)
	if err != nil {
		return nil, err
	}
	rv := AggIndex(ix, colIndex, ag)
	if rv == nil {
		return nil, fmt.Errorf("table agg.AggTry: agg type: %v not recognized", ag)
	}
	return rv, nil
}
