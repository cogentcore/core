// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package stats

import (
	"math"

	"cogentcore.org/core/tensor"
)

// NFunc is the nfun for stats functions, return the length of the
// first tensor, and initializing the second one to hold the output
// as the SubSpace of the first tensor.
func NFunc(tsr ...*tensor.Indexed) int {
	if len(tsr) != 2 {
		return 0
	}
	n := tsr[0].Len()
	sh := tensor.Shape{}
	sh.CopyShape(tsr[0].Tensor.Shape())
	sh.Sizes[0] = 1
	tsr[1].Tensor.SetShape(sh.Sizes, sh.Names...)
	tsr[1].Indexes = []int{0}
	return n
}

// StatFunc is a helper function for stats functions, dealing with iterating over
// the Cell subspace per row and initializing the aggregation values for first index.
// It also skips over NaN missing values.
func StatFunc(idx int, in, out *tensor.Indexed, ini float64, fun func(val, agg float64) float64) {
	nsub := out.Len()
	for i := 0; i < nsub; i++ {
		if idx == 0 {
			out.Tensor.SetFloat1D(i, ini)
		}
		val := in.Tensor.FloatRowCell(in.Indexes[idx], i)
		if math.IsNaN(val) {
			continue
		}
		out.Tensor.SetFloat1D(i, fun(val, out.Tensor.Float1D(i)))
	}
}

// SumFunc is a Vectorize function for computing the sum
func SumFunc(idx int, tsr ...*tensor.Indexed) {
	StatFunc(idx, tsr[0], tsr[1], 0, func(val, agg float64) float64 {
		return agg + val
	})
}

// CountFunc is a Vectorize function for computing the count
func CountFunc(idx int, tsr ...*tensor.Indexed) {
	StatFunc(idx, tsr[0], tsr[1], 0, func(val, agg float64) float64 {
		return agg + 1
	})
}

// ProdFunc is a Vectorize function for computing the product
func ProdFunc(idx int, tsr ...*tensor.Indexed) {
	StatFunc(idx, tsr[0], tsr[1], 1, func(val, agg float64) float64 {
		return agg * val
	})
}

// MinFunc is a Vectorize function for computing the min
func MinFunc(idx int, tsr ...*tensor.Indexed) {
	StatFunc(idx, tsr[0], tsr[1], math.MaxFloat64, func(val, agg float64) float64 {
		return math.Min(agg, val)
	})
}

// MaxFunc is a Vectorize function for computing the min
func MaxFunc(idx int, tsr ...*tensor.Indexed) {
	StatFunc(idx, tsr[0], tsr[1], -math.MaxFloat64, func(val, agg float64) float64 {
		return math.Max(agg, val)
	})
}

// MinAbsFunc is a Vectorize function for computing the min of abs values
func MinAbsFunc(idx int, tsr ...*tensor.Indexed) {
	StatFunc(idx, tsr[0], tsr[1], math.MaxFloat64, func(val, agg float64) float64 {
		return math.Min(agg, math.Abs(val))
	})
}

// MaxAbsFunc is a Vectorize function for computing the max of abs values
func MaxAbsFunc(idx int, tsr ...*tensor.Indexed) {
	StatFunc(idx, tsr[0], tsr[1], -math.MaxFloat64, func(val, agg float64) float64 {
		return math.Max(agg, math.Abs(val))
	})
}

// L1NormFunc is an StatFunc that computes the L1 norm: sum of absolute values
// use 0 as initial value.
// func L1NormFunc(idx int, val float64, agg float64) float64 {
// 	return agg + math.Abs(val)
// }

// Note: SumSq is not numerically stable for large N in simple func form.
