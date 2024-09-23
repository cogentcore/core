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
// the outermost row dimension size of 1, and contains
// the stat value(s) for the "cells" in higher-dimensional tensors,
// and a single scalar value for a 1D input tensor.
// Critically, the stat is always computed over the outer row dimension,
// so each cell in a higher-dimensional output reflects the _row-wise_
// stat for that cell across the different rows. Use [tensor.NewRowCellsView],
// [tensor.Cells1D], and [tensor.As1D] to reshape and reslice the data as needed.
// All stats functions skip over NaN's, as a missing value.
// Stats functions cannot be computed in parallel,
// e.g., using VectorizeThreaded or GPU, due to shared writing
// to the same output values.  Special implementations are required
// if that is needed.
type StatsFunc func(in, out tensor.Tensor) error

// CountOut64 computes the count of non-NaN tensor values,
// and returns the Float64 output values for subsequent use.
func CountOut64(in, out tensor.Tensor) (tensor.Tensor, error) {
	return VectorizeOut64(NFunc, func(idx int, tsr ...tensor.Tensor) {
		VecFunc(idx, tsr[0], tsr[1], 0, func(val, agg float64) float64 {
			return agg + 1
		})
	}, in, out)
}

// CountFunc computes the count of non-NaN tensor values.
// See [StatsFunc] for general information.
func CountFunc(in, out tensor.Tensor) error {
	_, err := CountOut64(in, out)
	return err
}

// SumOut64 computes the sum of tensor values,
// and returns the Float64 output values for subsequent use.
func SumOut64(in, out tensor.Tensor) (tensor.Tensor, error) {
	return VectorizeOut64(NFunc, func(idx int, tsr ...tensor.Tensor) {
		VecFunc(idx, tsr[0], tsr[1], 0, func(val, agg float64) float64 {
			return agg + val
		})
	}, in, out)
}

// SumFunc computes the sum of tensor values.
// See [StatsFunc] for general information.
func SumFunc(in, out tensor.Tensor) error {
	_, err := SumOut64(in, out)
	return err
}

// SumAbsFunc computes the sum of absolute-value-of tensor values.
// This is also known as the L1 norm.
// See [StatsFunc] for general information.
func SumAbsFunc(in, out tensor.Tensor) error {
	_, err := VectorizeOut64(NFunc, func(idx int, tsr ...tensor.Tensor) {
		VecFunc(idx, tsr[0], tsr[1], 0, func(val, agg float64) float64 {
			return agg + math.Abs(val)
		})
	}, in, out)
	return err
}

// ProdFunc computes the product of tensor values.
// See [StatsFunc] for general information.
func ProdFunc(in, out tensor.Tensor) error {
	_, err := VectorizeOut64(NFunc, func(idx int, tsr ...tensor.Tensor) {
		VecFunc(idx, tsr[0], tsr[1], 1, func(val, agg float64) float64 {
			return agg * val
		})
	}, in, out)
	return err
}

// MinFunc computes the min of tensor values.
// See [StatsFunc] for general information.
func MinFunc(in, out tensor.Tensor) error {
	_, err := VectorizeOut64(NFunc, func(idx int, tsr ...tensor.Tensor) {
		VecFunc(idx, tsr[0], tsr[1], math.MaxFloat64, func(val, agg float64) float64 {
			return math.Min(agg, val)
		})
	}, in, out)
	return err
}

// MaxFunc computes the max of tensor values.
// See [StatsFunc] for general information.
func MaxFunc(in, out tensor.Tensor) error {
	_, err := VectorizeOut64(NFunc, func(idx int, tsr ...tensor.Tensor) {
		VecFunc(idx, tsr[0], tsr[1], -math.MaxFloat64, func(val, agg float64) float64 {
			return math.Max(agg, val)
		})
	}, in, out)
	return err
}

// MinAbsFunc computes the min of absolute-value-of tensor values.
// See [StatsFunc] for general information.
func MinAbsFunc(in, out tensor.Tensor) error {
	_, err := VectorizeOut64(NFunc, func(idx int, tsr ...tensor.Tensor) {
		VecFunc(idx, tsr[0], tsr[1], math.MaxFloat64, func(val, agg float64) float64 {
			return math.Min(agg, math.Abs(val))
		})
	}, in, out)
	return err
}

// MaxAbsFunc computes the max of absolute-value-of tensor values.
// See [StatsFunc] for general information.
func MaxAbsFunc(in, out tensor.Tensor) error {
	_, err := VectorizeOut64(NFunc, func(idx int, tsr ...tensor.Tensor) {
		VecFunc(idx, tsr[0], tsr[1], -math.MaxFloat64, func(val, agg float64) float64 {
			return math.Max(agg, math.Abs(val))
		})
	}, in, out)
	return err
}

// MeanOut64 computes the mean of tensor values,
// and returns the Float64 output values for subsequent use.
func MeanOut64(in, out tensor.Tensor) (mean64, count64 tensor.Tensor, err error) {
	sum64, err := SumOut64(in, out)
	if err != nil {
		return
	}
	count := out.AsValues().Clone()
	count64, _ = CountOut64(in, count) // if sum works, this works
	nsub := out.Len()
	for i := range nsub {
		c := count64.Float1D(i)
		if c == 0 {
			continue
		}
		sum64.SetFloat1D(sum64.Float1D(i)/c, i)
		out.SetFloat1D(sum64.Float1D(i), i)
	}
	return sum64, count64, err
}

// MeanFunc computes the mean of tensor values.
// See [StatsFunc] for general information.
func MeanFunc(in, out tensor.Tensor) error {
	_, _, err := MeanOut64(in, out)
	return err
}

// SumSqDevOut64 computes the sum of squared mean deviates of tensor values,
// and returns the Float64 output values for subsequent use.
func SumSqDevOut64(in, out tensor.Tensor) (ssd64, mean64, count64 tensor.Tensor, err error) {
	mean64, count64, err = MeanOut64(in, out)
	if err != nil {
		return
	}
	ssd64, err = VectorizeOut64(NFunc, func(idx int, tsr ...tensor.Tensor) {
		Vec2inFunc(idx, tsr[0], tsr[1], tsr[2], 0, func(val1, val2, agg float64) float64 {
			dv := val1 - val2
			return agg + dv*dv
		})
	}, in, mean64, out)
	return
}

// VarOut64 computes the sample variance of tensor values,
// and returns the Float64 output values for subsequent use.
func VarOut64(in, out tensor.Tensor) (var64, mean64, count64 tensor.Tensor, err error) {
	var64, mean64, count64, err = SumSqDevOut64(in, out)
	if err != nil {
		return
	}
	nsub := out.Len()
	for i := range nsub {
		c := count64.Float1D(i)
		if c < 2 {
			continue
		}
		vr := var64.Float1D(i) / (c - 1)
		var64.SetFloat1D(vr, i)
		out.SetFloat1D(vr, i)
	}
	return
}

// VarFunc computes the sample variance of tensor values.
// Squared deviations from mean, divided by n-1. See also [VarPopFunc].
// See [StatsFunc] for general information.
func VarFunc(in, out tensor.Tensor) error {
	_, _, _, err := VarOut64(in, out)
	return err
}

// StdOut64 computes the sample standard deviation of tensor values.
// and returns the Float64 output values for subsequent use.
func StdOut64(in, out tensor.Tensor) (std64, mean64, count64 tensor.Tensor, err error) {
	std64, mean64, count64, err = VarOut64(in, out)
	if err != nil {
		return
	}
	nsub := out.Len()
	for i := range nsub {
		std := math.Sqrt(std64.Float1D(i))
		std64.SetFloat1D(std, i)
		out.SetFloat1D(std, i)
	}
	return
}

// StdFunc computes the sample standard deviation of tensor values.
// Sqrt of variance from [VarFunc]. See also [StdPopFunc].
// See [StatsFunc] for general information.
func StdFunc(in, out tensor.Tensor) error {
	_, _, _, err := StdOut64(in, out)
	return err
}

// SemFunc computes the sample standard error of the mean of tensor values.
// Standard deviation [StdFunc] / sqrt(n). See also [SemPopFunc].
// See [StatsFunc] for general information.
func SemFunc(in, out tensor.Tensor) error {
	var64, _, count64, err := VarOut64(in, out)
	if err != nil {
		return err
	}
	nsub := out.Len()
	for i := range nsub {
		c := count64.Float1D(i)
		if c < 2 {
			out.SetFloat1D(math.Sqrt(var64.Float1D(i)), i)
		} else {
			out.SetFloat1D(math.Sqrt(var64.Float1D(i))/math.Sqrt(c), i)
		}
	}
	return nil
}

// VarPopOut64 computes the population variance of tensor values.
// and returns the Float64 output values for subsequent use.
func VarPopOut64(in, out tensor.Tensor) (var64, mean64, count64 tensor.Tensor, err error) {
	var64, mean64, count64, err = SumSqDevOut64(in, out)
	if err != nil {
		return
	}
	nsub := out.Len()
	for i := range nsub {
		c := count64.Float1D(i)
		if c == 0 {
			continue
		}
		var64.SetFloat1D(var64.Float1D(i)/c, i)
		out.SetFloat1D(var64.Float1D(i), i)
	}
	return
}

// VarPopFunc computes the population variance of tensor values.
// Squared deviations from mean, divided by n. See also [VarFunc].
// See [StatsFunc] for general information.
func VarPopFunc(in, out tensor.Tensor) error {
	_, _, _, err := VarPopOut64(in, out)
	return err
}

// StdPopFunc computes the population standard deviation of tensor values.
// Sqrt of variance from [VarPopFunc]. See also [StdFunc].
// See [StatsFunc] for general information.
func StdPopFunc(in, out tensor.Tensor) error {
	var64, _, _, err := VarPopOut64(in, out)
	if err != nil {
		return err
	}
	nsub := out.Len()
	for i := range nsub {
		out.SetFloat1D(math.Sqrt(var64.Float1D(i)), i)
	}
	return nil
}

// SemPopFunc computes the population standard error of the mean of tensor values.
// Standard deviation [StdPopFunc] / sqrt(n). See also [SemFunc].
// See [StatsFunc] for general information.
func SemPopFunc(in, out tensor.Tensor) error {
	var64, _, count64, err := VarPopOut64(in, out)
	if err != nil {
		return err
	}
	nsub := out.Len()
	for i := range nsub {
		c := count64.Float1D(i)
		if c < 2 {
			out.SetFloat1D(math.Sqrt(var64.Float1D(i)), i)
		} else {
			out.SetFloat1D(math.Sqrt(var64.Float1D(i))/math.Sqrt(c), i)
		}
	}
	return nil
}

// SumSqScaleOut64 is a helper for sum-of-squares, returning scale and ss
// factors aggregated separately for better numerical stability, per BLAS.
// Returns the Float64 output values for subsequent use.
func SumSqScaleOut64(in, out tensor.Tensor) (scale64, ss64 tensor.Tensor, err error) {
	scale64, ss64, err = Vectorize2Out64(NFunc, func(idx int, tsr ...tensor.Tensor) {
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

// SumSqOut64 computes the sum of squares of tensor values,
// and returns the Float64 output values for subsequent use.
func SumSqOut64(in, out tensor.Tensor) (tensor.Tensor, error) {
	scale64, ss64, err := SumSqScaleOut64(in, out)
	if err != nil {
		return nil, err
	}
	nsub := out.Len()
	for i := range nsub {
		scale := scale64.Float1D(i)
		ss := ss64.Float1D(i)
		v := 0.0
		if math.IsInf(scale, 1) {
			v = math.Inf(1)
		} else {
			v = scale * scale * ss
		}
		scale64.SetFloat1D(v, i)
		out.SetFloat1D(v, i)
	}
	return scale64, nil
}

// SumSqFunc computes the sum of squares of tensor values,
// See [StatsFunc] for general information.
func SumSqFunc(in, out tensor.Tensor) error {
	_, err := SumSqOut64(in, out)
	return err
}

// L2NormOut64 computes the square root of the sum of squares of tensor values,
// known as the L2 norm, and returns the Float64 output values for
// use in subsequent computations.
func L2NormOut64(in, out tensor.Tensor) (tensor.Tensor, error) {
	scale64, ss64, err := SumSqScaleOut64(in, out)
	if err != nil {
		return nil, err
	}
	nsub := out.Len()
	for i := range nsub {
		scale := scale64.Float1D(i)
		ss := ss64.Float1D(i)
		v := 0.0
		if math.IsInf(scale, 1) {
			v = math.Inf(1)
		} else {
			v = scale * math.Sqrt(ss)
		}
		scale64.SetFloat1D(v, i)
		out.SetFloat1D(v, i)
	}
	return scale64, nil
}

// L2NormFunc computes the square root of the sum of squares of tensor values,
// known as the L2 norm.
// See [StatsFunc] for general information.
func L2NormFunc(in, out tensor.Tensor) error {
	_, err := L2NormOut64(in, out)
	return err
}
