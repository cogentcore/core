// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

/*
Package tsragg provides aggregation functions (Sum, Mean, etc) that
operate directly on tensor.Tensor data.  See also agg package that
operates on an IndexView of an table.Table column.
*/
package tsragg

//go:generate core generate

import (
	"math"

	"cogentcore.org/core/tensor"
	"cogentcore.org/core/tensor/stats/agg"
)

// AggFunc is an aggregation function that incrementally updates agg value
// from each element in the tensor in turn -- returns new agg value that
// will be passed into next item as agg
type AggFunc func(idx int, val float64, agg float64) float64

// Agg applies given aggregation function to each element in the tensor
// (automatically skips IsNull and NaN elements), using float64 conversions of the values.
// init is the initial value for the agg variable. returns final aggregate value
func Agg(tsr tensor.Tensor, ini float64, fun AggFunc) float64 {
	ln := tsr.Len()
	ag := ini
	for j := 0; j < ln; j++ {
		val := tsr.Float1D(j)
		if !tsr.IsNull1D(j) && !math.IsNaN(val) {
			ag = fun(j, val, ag)
		}
	}
	return ag
}

// Count returns the count of non-Null, non-NaN elements in given Tensor.
func Count(tsr tensor.Tensor) float64 {
	return Agg(tsr, 0, agg.CountFunc)
}

// Sum returns the sum of non-Null, non-NaN elements in given Tensor.
func Sum(tsr tensor.Tensor) float64 {
	return Agg(tsr, 0, agg.SumFunc)
}

// Prod returns the product of non-Null, non-NaN elements in given Tensor.
func Prod(tsr tensor.Tensor) float64 {
	return Agg(tsr, 1, agg.ProdFunc)
}

// Max returns the maximum of non-Null, non-NaN elements in given Tensor.
func Max(tsr tensor.Tensor) float64 {
	return Agg(tsr, -math.MaxFloat64, agg.MaxFunc)
}

// Min returns the minimum of non-Null, non-NaN elements in given Tensor.
func Min(tsr tensor.Tensor) float64 {
	return Agg(tsr, math.MaxFloat64, agg.MinFunc)
}

// Mean returns the mean of non-Null, non-NaN elements in given Tensor.
func Mean(tsr tensor.Tensor) float64 {
	cnt := Count(tsr)
	if cnt == 0 {
		return 0
	}
	return Sum(tsr) / cnt
}

// Var returns the sample variance of non-Null, non-NaN elements in given Tensor.
func Var(tsr tensor.Tensor) float64 {
	cnt := Count(tsr)
	if cnt < 2 {
		return 0
	}
	mean := Sum(tsr) / cnt
	vr := Agg(tsr, 0, func(idx int, val float64, agg float64) float64 {
		dv := val - mean
		return agg + dv*dv
	})
	return vr / (cnt - 1)
}

// Std returns the sample standard deviation of non-Null, non-NaN elements in given Tensor.
func Std(tsr tensor.Tensor) float64 {
	return math.Sqrt(Var(tsr))
}

// Sem returns the sample standard error of the mean of non-Null, non-NaN elements in given Tensor.
func Sem(tsr tensor.Tensor) float64 {
	cnt := Count(tsr)
	if cnt < 2 {
		return 0
	}
	return Std(tsr) / math.Sqrt(cnt)
}

// VarPop returns the population variance of non-Null, non-NaN elements in given Tensor.
func VarPop(tsr tensor.Tensor) float64 {
	cnt := Count(tsr)
	if cnt < 2 {
		return 0
	}
	mean := Sum(tsr) / cnt
	vr := Agg(tsr, 0, func(idx int, val float64, agg float64) float64 {
		dv := val - mean
		return agg + dv*dv
	})
	return vr / cnt
}

// StdPop returns the population standard deviation of non-Null, non-NaN elements in given Tensor.
func StdPop(tsr tensor.Tensor) float64 {
	return math.Sqrt(VarPop(tsr))
}

// SemPop returns the population standard error of the mean of non-Null, non-NaN elements in given Tensor.
func SemPop(tsr tensor.Tensor) float64 {
	cnt := Count(tsr)
	if cnt < 2 {
		return 0
	}
	return StdPop(tsr) / math.Sqrt(cnt)
}

// SumSq returns the sum-of-squares of non-Null, non-NaN elements in given Tensor.
func SumSq(tsr tensor.Tensor) float64 {
	return Agg(tsr, 0, agg.SumSqFunc)
}
