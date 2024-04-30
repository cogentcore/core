// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package stats

import "math"

// These are standard StatFunc functions that can operate on tensor.Tensor
// or table.Table, using float64 values

// StatFunc is an statistic function that incrementally updates agg
// aggregation value from each element in the tensor in turn.
// Returns new agg value that will be passed into next item as agg.
type StatFunc func(idx int, val float64, agg float64) float64

// CountFunc is an StatFunc that computes number of elements (non-Null, non-NaN)
// Use 0 as initial value.
func CountFunc(idx int, val float64, agg float64) float64 {
	return agg + 1
}

// SumFunc is an StatFunc that computes a sum aggregate.
// use 0 as initial value.
func SumFunc(idx int, val float64, agg float64) float64 {
	return agg + val
}

// Prodfunc is an StatFunc that computes a product aggregate.
// use 1 as initial value.
func ProdFunc(idx int, val float64, agg float64) float64 {
	return agg * val
}

// MinFunc is an StatFunc that computes a min aggregate.
// use math.MaxFloat64 for initial agg value.
func MinFunc(idx int, val float64, agg float64) float64 {
	return math.Min(agg, val)
}

// MaxFunc is an StatFunc that computes a max aggregate.
// use -math.MaxFloat64 for initial agg value.
func MaxFunc(idx int, val float64, agg float64) float64 {
	return math.Max(agg, val)
}

// MinAbsFunc is an StatFunc that computes a min aggregate.
// use math.MaxFloat64 for initial agg value.
func MinAbsFunc(idx int, val float64, agg float64) float64 {
	return math.Min(agg, math.Abs(val))
}

// MaxAbsFunc is an StatFunc that computes a max aggregate.
// use -math.MaxFloat64 for initial agg value.
func MaxAbsFunc(idx int, val float64, agg float64) float64 {
	return math.Max(agg, math.Abs(val))
}

// L1NormFunc is an StatFunc that computes the L1 norm: sum of absolute values
// use 0 as initial value.
func L1NormFunc(idx int, val float64, agg float64) float64 {
	return agg + math.Abs(val)
}

// Note: SumSq is not numerically stable for large N in simple func form.
