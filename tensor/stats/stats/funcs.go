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
// Critically, the stat is always computed over the outer row dimension,
// so each cell in a higher-dimensional output reflects the _row-wise_
// stat for that cell across the different rows.  To compute a stat
// on the [tensor.SubSpace] cells themselves, must call on a
// [tensor.New1DViewOf] the sub space.
// All stats functions skip over NaN's, as a missing value.
// Stats functions cannot be computed in parallel,
// e.g., using VectorizeThreaded or GPU, due to shared writing
// to the same output values.  Special implementations are required
// if that is needed.
type StatsFunc func(in, out *tensor.Indexed)

// CountFuncOut64 computes the count of non-NaN tensor values,
// and returns the Float64 output values for subsequent use.
func CountFuncOut64(in, out *tensor.Indexed) *tensor.Indexed {
	return VectorizeOut64(NFunc, func(idx int, tsr ...*tensor.Indexed) {
		VecFunc(idx, tsr[0], tsr[1], 0, func(val, agg float64) float64 {
			return agg + 1
		})
	}, in, out)
}

// CountFunc computes the count of non-NaN tensor values.
// See [StatsFunc] for general information.
func CountFunc(in, out *tensor.Indexed) {
	CountFuncOut64(in, out)
}

// SumFuncOut64 computes the sum of tensor values,
// and returns the Float64 output values for subsequent use.
func SumFuncOut64(in, out *tensor.Indexed) *tensor.Indexed {
	return VectorizeOut64(NFunc, func(idx int, tsr ...*tensor.Indexed) {
		VecFunc(idx, tsr[0], tsr[1], 0, func(val, agg float64) float64 {
			return agg + val
		})
	}, in, out)
}

// SumFunc computes the sum of tensor values.
// See [StatsFunc] for general information.
func SumFunc(in, out *tensor.Indexed) {
	SumFuncOut64(in, out)
}

// SumAbsFunc computes the sum of absolute-value-of tensor values.
// This is also known as the L1 norm.
// See [StatsFunc] for general information.
func SumAbsFunc(in, out *tensor.Indexed) {
	VectorizeOut64(NFunc, func(idx int, tsr ...*tensor.Indexed) {
		VecFunc(idx, tsr[0], tsr[1], 0, func(val, agg float64) float64 {
			return agg + math.Abs(val)
		})
	}, in, out)
}

// ProdFunc computes the product of tensor values.
// See [StatsFunc] for general information.
func ProdFunc(in, out *tensor.Indexed) {
	VectorizeOut64(NFunc, func(idx int, tsr ...*tensor.Indexed) {
		VecFunc(idx, tsr[0], tsr[1], 1, func(val, agg float64) float64 {
			return agg * val
		})
	}, in, out)
}

// MinFunc computes the min of tensor values.
// See [StatsFunc] for general information.
func MinFunc(in, out *tensor.Indexed) {
	VectorizeOut64(NFunc, func(idx int, tsr ...*tensor.Indexed) {
		VecFunc(idx, tsr[0], tsr[1], math.MaxFloat64, func(val, agg float64) float64 {
			return math.Min(agg, val)
		})
	}, in, out)
}

// MaxFunc computes the max of tensor values.
// See [StatsFunc] for general information.
func MaxFunc(in, out *tensor.Indexed) {
	VectorizeOut64(NFunc, func(idx int, tsr ...*tensor.Indexed) {
		VecFunc(idx, tsr[0], tsr[1], -math.MaxFloat64, func(val, agg float64) float64 {
			return math.Max(agg, val)
		})
	}, in, out)
}

// MinAbsFunc computes the min of absolute-value-of tensor values.
// See [StatsFunc] for general information.
func MinAbsFunc(in, out *tensor.Indexed) {
	VectorizeOut64(NFunc, func(idx int, tsr ...*tensor.Indexed) {
		VecFunc(idx, tsr[0], tsr[1], math.MaxFloat64, func(val, agg float64) float64 {
			return math.Min(agg, math.Abs(val))
		})
	}, in, out)
}

// MaxAbsFunc computes the max of absolute-value-of tensor values.
// See [StatsFunc] for general information.
func MaxAbsFunc(in, out *tensor.Indexed) {
	VectorizeOut64(NFunc, func(idx int, tsr ...*tensor.Indexed) {
		VecFunc(idx, tsr[0], tsr[1], -math.MaxFloat64, func(val, agg float64) float64 {
			return math.Max(agg, math.Abs(val))
		})
	}, in, out)
}

// MeanFuncOut64 computes the mean of tensor values,
// and returns the Float64 output values for subsequent use.
func MeanFuncOut64(in, out *tensor.Indexed) (mean64, count64 *tensor.Indexed) {
	sum64 := SumFuncOut64(in, out)
	count := tensor.NewIndexed(out.Tensor.Clone())
	count64 = CountFuncOut64(in, count)
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

// SumSqDevFuncOut64 computes the sum of squared mean deviates of tensor values,
// and returns the Float64 output values for subsequent use.
func SumSqDevFuncOut64(in, out *tensor.Indexed) (ssd64, mean64, count64 *tensor.Indexed) {
	mean64, count64 = MeanFuncOut64(in, out)
	ssd64 = VectorizeOut64(NFunc, func(idx int, tsr ...*tensor.Indexed) {
		Vec2inFunc(idx, tsr[0], tsr[1], tsr[2], 0, func(val1, val2, agg float64) float64 {
			dv := val1 - val2
			return agg + dv*dv
		})
	}, in, mean64, out)
	return
}

// VarFuncOut64 computes the sample variance of tensor values,
// and returns the Float64 output values for subsequent use.
func VarFuncOut64(in, out *tensor.Indexed) (var64, mean64, count64 *tensor.Indexed) {
	var64, mean64, count64 = SumSqDevFuncOut64(in, out)
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
// and returns the Float64 output values for subsequent use.
func VarPopFuncOut64(in, out *tensor.Indexed) (var64, mean64, count64 *tensor.Indexed) {
	var64, mean64, count64 = SumSqDevFuncOut64(in, out)
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

// SumSqScaleFuncOut64 is a helper for sum-of-squares, returning scale and ss
// factors aggregated separately for better numerical stability, per BLAS.
// Returns the Float64 output values for subsequent use.
func SumSqScaleFuncOut64(in, out *tensor.Indexed) (scale64, ss64 *tensor.Indexed) {
	scale64, ss64 = Vectorize2Out64(NFunc, func(idx int, tsr ...*tensor.Indexed) {
		Vec2outFunc(idx, tsr[0], tsr[1], tsr[2], 0, 1, func(val, scale, ss float64) (float64, float64) {
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
	}, in, out)
	return
}

// SumSqFuncOut64 computes the sum of squares of tensor values,
// and returns the Float64 output values for subsequent use.
func SumSqFuncOut64(in, out *tensor.Indexed) *tensor.Indexed {
	scale64, ss64 := SumSqScaleFuncOut64(in, out)
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
	scale64, ss64 := SumSqScaleFuncOut64(in, out)
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
