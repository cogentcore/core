// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package stats

import (
	"math"

	"cogentcore.org/core/tensor"
)

// StatsFunc is the function signature for a stats function,
// where the output has the same shape as the input but with
// the outer-most row dimension size of 1, and contains
// the stat value(s) for the "cells" in higher-dimensional tensors,
// and a single scalar value for a 1D input tensor.
// All stats functions skip over NaN's, as a missing value.
// Stats functions cannot be computed in parallel,
// e.g., using VectorizeThreaded or GPU, due to shared writing
// to the same output values.  Special implementations are required
// if that is needed.
type StatsFunc func(in, out *tensor.Indexed)

// CountVecFunc is a Vectorize function for computing the count.
func CountVecFunc(idx int, tsr ...*tensor.Indexed) {
	StatVecFunc(idx, tsr[0], tsr[1], 0, func(val, agg float64) float64 {
		return agg + 1
	})
}

// CountFunc computes the count of non-NaN tensor values.
// See [StatsFunc] for general information.
func CountFunc(in, out *tensor.Indexed) {
	VectorizeOut64(CountVecFunc, in, out)
}

// SumVecFunc is a Vectorize function for computing the sum.
func SumVecFunc(idx int, tsr ...*tensor.Indexed) {
	StatVecFunc(idx, tsr[0], tsr[1], 0, func(val, agg float64) float64 {
		return agg + val
	})
}

// SumFunc computes the sum of tensor values.
// See [StatsFunc] for general information.
func SumFunc(in, out *tensor.Indexed) {
	VectorizeOut64(SumVecFunc, in, out)
}

// SumAbsVecFunc is a Vectorize function for computing the sum of abs values.
// This is also known as the L1 norm.
func SumAbsVecFunc(idx int, tsr ...*tensor.Indexed) {
	StatVecFunc(idx, tsr[0], tsr[1], 0, func(val, agg float64) float64 {
		return agg + math.Abs(val)
	})
}

// SumAbsFunc computes the sum of absolute-value-of tensor values.
// This is also known as the L1 norm.
// See [StatsFunc] for general information.
func SumAbsFunc(in, out *tensor.Indexed) {
	VectorizeOut64(SumAbsVecFunc, in, out)
}

// ProdVecFunc is a Vectorize function for computing the product.
func ProdVecFunc(idx int, tsr ...*tensor.Indexed) {
	StatVecFunc(idx, tsr[0], tsr[1], 1, func(val, agg float64) float64 {
		return agg * val
	})
}

// ProdFunc computes the product of tensor values.
// See [StatsFunc] for general information.
func ProdFunc(in, out *tensor.Indexed) {
	VectorizeOut64(ProdVecFunc, in, out)
}

// MinVecFunc is a Vectorize function for computing the min.
func MinVecFunc(idx int, tsr ...*tensor.Indexed) {
	StatVecFunc(idx, tsr[0], tsr[1], math.MaxFloat64, func(val, agg float64) float64 {
		return math.Min(agg, val)
	})
}

// MinFunc computes the min of tensor values.
// See [StatsFunc] for general information.
func MinFunc(in, out *tensor.Indexed) {
	VectorizeOut64(MinVecFunc, in, out)
}

// MaxVecFunc is a Vectorize function for computing the max.
func MaxVecFunc(idx int, tsr ...*tensor.Indexed) {
	StatVecFunc(idx, tsr[0], tsr[1], -math.MaxFloat64, func(val, agg float64) float64 {
		return math.Max(agg, val)
	})
}

// MaxFunc computes the max of tensor values.
// See [StatsFunc] for general information.
func MaxFunc(in, out *tensor.Indexed) {
	VectorizeOut64(MaxVecFunc, in, out)
}

// MinAbsVecFunc is a Vectorize function for computing the min of abs.
func MinAbsVecFunc(idx int, tsr ...*tensor.Indexed) {
	StatVecFunc(idx, tsr[0], tsr[1], math.MaxFloat64, func(val, agg float64) float64 {
		return math.Min(agg, math.Abs(val))
	})
}

// MinAbsFunc computes the min of absolute-value-of tensor values.
// See [StatsFunc] for general information.
func MinAbsFunc(in, out *tensor.Indexed) {
	VectorizeOut64(MinAbsVecFunc, in, out)
}

// MaxAbsVecFunc is a Vectorize function for computing the max of abs.
func MaxAbsVecFunc(idx int, tsr ...*tensor.Indexed) {
	StatVecFunc(idx, tsr[0], tsr[1], -math.MaxFloat64, func(val, agg float64) float64 {
		return math.Max(agg, math.Abs(val))
	})
}

// MaxAbsFunc computes the max of absolute-value-of tensor values.
// See [StatsFunc] for general information.
func MaxAbsFunc(in, out *tensor.Indexed) {
	VectorizeOut64(MaxAbsVecFunc, in, out)
}

// MeanFuncOut64 computes the mean of tensor values,
// and returns the Float64 count and mean output values for
// use in subsequent computations.
func MeanFuncOut64(in, out *tensor.Indexed) (mean64, count64 *tensor.Indexed) {
	sum64 := VectorizeOut64(SumVecFunc, in, out)
	count := tensor.NewIndexed(out.Tensor.Clone())
	count64 = VectorizeOut64(CountVecFunc, in, count)
	nsub := out.Tensor.Len()
	for i := range nsub {
		c := count64.Tensor.Float1D(i)
		if c == 0 {
			continue
		}
		sum64.Tensor.SetFloat1D(i, sum64.Tensor.Float1D(i)/c)
		out.Tensor.SetFloat1D(i, sum64.Tensor.Float1D(i))
	}
	return sum64, count64
}

// MeanFunc computes the mean of tensor values.
// See [StatsFunc] for general information.
func MeanFunc(in, out *tensor.Indexed) {
	MeanFuncOut64(in, out)
}

// VarVecFunc is a Vectorize function for computing the variance,
// using 3 tensors: in, mean, out.
func VarVecFunc(idx int, tsr ...*tensor.Indexed) {
	StatVec2inFunc(idx, tsr[0], tsr[1], tsr[2], 0, func(val1, val2, agg float64) float64 {
		dv := val1 - val2
		return agg + dv*dv
	})
}

// VarFuncOut64 computes the sample variance of tensor values.
// and returns the Float64 output values for
// use in subsequent computations.
func VarFuncOut64(in, out *tensor.Indexed) (var64, mean64, count64 *tensor.Indexed) {
	mean64, count64 = MeanFuncOut64(in, out)
	var64 = VectorizeOut64(VarVecFunc, in, mean64, out)
	nsub := out.Tensor.Len()
	for i := range nsub {
		c := count64.Tensor.Float1D(i)
		if c < 2 {
			continue
		}
		var64.Tensor.SetFloat1D(i, var64.Tensor.Float1D(i)/(c-1))
		out.Tensor.SetFloat1D(i, var64.Tensor.Float1D(i))
	}
	return
}

// VarFunc computes the sample variance of tensor values.
// Squared deviations from mean, divided by n-1. See also [VarPopFunc].
// See [StatsFunc] for general information.
func VarFunc(in, out *tensor.Indexed) {
	VarFuncOut64(in, out)
}

// StdFunc computes the sample standard deviation of tensor values.
// Sqrt of variance from [VarFunc]. See also [StdPopFunc].
// See [StatsFunc] for general information.
func StdFunc(in, out *tensor.Indexed) {
	var64, _, _ := VarFuncOut64(in, out)
	nsub := out.Tensor.Len()
	for i := range nsub {
		out.Tensor.SetFloat1D(i, math.Sqrt(var64.Tensor.Float1D(i)))
	}
}

// SemFunc computes the sample standard error of the mean of tensor values.
// Standard deviation [StdFunc] / sqrt(n). See also [SemPopFunc].
// See [StatsFunc] for general information.
func SemFunc(in, out *tensor.Indexed) {
	var64, _, count64 := VarFuncOut64(in, out)
	nsub := out.Tensor.Len()
	for i := range nsub {
		c := count64.Tensor.Float1D(i)
		if c < 2 {
			out.Tensor.SetFloat1D(i, math.Sqrt(var64.Tensor.Float1D(i)))
		} else {
			out.Tensor.SetFloat1D(i, math.Sqrt(var64.Tensor.Float1D(i))/math.Sqrt(c))
		}
	}
}

// VarPopFuncOut64 computes the population variance of tensor values.
// and returns the Float64 output values for
// use in subsequent computations.
func VarPopFuncOut64(in, out *tensor.Indexed) (var64, mean64, count64 *tensor.Indexed) {
	mean64, count64 = MeanFuncOut64(in, out)
	var64 = VectorizeOut64(VarVecFunc, in, mean64, out)
	nsub := out.Tensor.Len()
	for i := range nsub {
		c := count64.Tensor.Float1D(i)
		if c == 0 {
			continue
		}
		var64.Tensor.SetFloat1D(i, var64.Tensor.Float1D(i)/c)
		out.Tensor.SetFloat1D(i, var64.Tensor.Float1D(i))
	}
	return
}

// VarPopFunc computes the population variance of tensor values.
// Squared deviations from mean, divided by n. See also [VarFunc].
// See [StatsFunc] for general information.
func VarPopFunc(in, out *tensor.Indexed) {
	VarPopFuncOut64(in, out)
}

// StdPopFunc computes the population standard deviation of tensor values.
// Sqrt of variance from [VarPopFunc]. See also [StdFunc].
// See [StatsFunc] for general information.
func StdPopFunc(in, out *tensor.Indexed) {
	var64, _, _ := VarPopFuncOut64(in, out)
	nsub := out.Tensor.Len()
	for i := range nsub {
		out.Tensor.SetFloat1D(i, math.Sqrt(var64.Tensor.Float1D(i)))
	}
}

// SemPopFunc computes the population standard error of the mean of tensor values.
// Standard deviation [StdPopFunc] / sqrt(n). See also [SemFunc].
// See [StatsFunc] for general information.
func SemPopFunc(in, out *tensor.Indexed) {
	var64, _, count64 := VarPopFuncOut64(in, out)
	nsub := out.Tensor.Len()
	for i := range nsub {
		c := count64.Tensor.Float1D(i)
		if c < 2 {
			out.Tensor.SetFloat1D(i, math.Sqrt(var64.Tensor.Float1D(i)))
		} else {
			out.Tensor.SetFloat1D(i, math.Sqrt(var64.Tensor.Float1D(i))/math.Sqrt(c))
		}
	}
}

// SumSqVecFunc is a Vectorize function for computing the sum of squares,
// using 2 separate aggregation tensors.
func SumSqVecFunc(idx int, tsr ...*tensor.Indexed) {
	StatVec2outFunc(idx, tsr[0], tsr[1], tsr[2], 0, 1, func(val, scale, ss float64) (float64, float64) {
		if val == 0 {
			return scale, ss
		}
		absxi := math.Abs(val)
		if scale < absxi {
			ss = 1 + ss*(scale/absxi)*(scale/absxi)
			scale = absxi
		} else {
			ss = ss + (absxi/scale)*(absxi/scale)
		}
		return scale, ss
	})
}

// SumSqFuncOut64 computes the sum of squares of tensor values,
// and returns the Float64 output values for
// use in subsequent computations.
func SumSqFuncOut64(in, out *tensor.Indexed) *tensor.Indexed {
	scale64, ss64 := Vectorize2Out64(SumSqVecFunc, in, out)
	nsub := out.Tensor.Len()
	for i := range nsub {
		scale := scale64.Tensor.Float1D(i)
		ss := ss64.Tensor.Float1D(i)
		v := 0.0
		if math.IsInf(scale, 1) {
			v = math.Inf(1)
		} else {
			v = scale * scale * ss
		}
		scale64.Tensor.SetFloat1D(i, v)
		out.Tensor.SetFloat1D(i, v)
	}
	return scale64
}

// SumSqFunc computes the sum of squares of tensor values,
// See [StatsFunc] for general information.
func SumSqFunc(in, out *tensor.Indexed) {
	SumSqFuncOut64(in, out)
}

// L2NormFuncOut64 computes the square root of the sum of squares of tensor values,
// known as the L2 norm, and returns the Float64 output values for
// use in subsequent computations.
func L2NormFuncOut64(in, out *tensor.Indexed) *tensor.Indexed {
	scale64, ss64 := Vectorize2Out64(SumSqVecFunc, in, out)
	nsub := out.Tensor.Len()
	for i := range nsub {
		scale := scale64.Tensor.Float1D(i)
		ss := ss64.Tensor.Float1D(i)
		v := 0.0
		if math.IsInf(scale, 1) {
			v = math.Inf(1)
		} else {
			v = scale * math.Sqrt(ss)
		}
		scale64.Tensor.SetFloat1D(i, v)
		out.Tensor.SetFloat1D(i, v)
	}
	return scale64
}

// L2NormFunc computes the square root of the sum of squares of tensor values,
// known as the L2 norm.
// See [StatsFunc] for general information.
func L2NormFunc(in, out *tensor.Indexed) {
	L2NormFuncOut64(in, out)
}
