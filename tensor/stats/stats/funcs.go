// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package stats

import (
	"math"

	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/tensor"
)

// StatsFunc is the function signature for a stats function that
// returns a new output vector. This can be less efficient for repeated
// computations where the output can be re-used: see [StatsOutFunc].
// But this version can be directly chained with other function calls.
// Function is computed over the outermost row dimension and the
// output is the shape of the remaining inner cells (a scalar for 1D inputs).
// Use [tensor.As1D], [tensor.NewRowCellsView], [tensor.Cells1D] etc
// to reshape and reslice the data as needed.
// All stats functions skip over NaN's, as a missing value.
// Stats functions cannot be computed in parallel,
// e.g., using VectorizeThreaded or GPU, due to shared writing
// to the same output values.  Special implementations are required
// if that is needed.
type StatsFunc = func(in tensor.Tensor) tensor.Values

// StatsOutFunc is the function signature for a stats function,
// that takes output values as final argument. See [StatsFunc]
// This version is for computationally demanding cases and saves
// reallocation of output.
type StatsOutFunc = func(in tensor.Tensor, out tensor.Values) error

// CountOut64 computes the count of non-NaN tensor values,
// and returns the Float64 output values for subsequent use.
func CountOut64(in tensor.Tensor, out tensor.Values) (tensor.Tensor, error) {
	return VectorizeOut64(NFunc, func(idx int, tsr ...tensor.Tensor) {
		VecFunc(idx, tsr[0], tsr[1], 0, func(val, agg float64) float64 {
			return agg + 1
		})
	}, in, out)
}

// Count computes the count of non-NaN tensor values.
// See [StatsFunc] for general information.
func Count(in tensor.Tensor) tensor.Values {
	out := tensor.NewOfType(in.DataType())
	errors.Log1(CountOut64(in, out))
	return out
}

// CountOut computes the count of non-NaN tensor values.
// See [StatsOutFunc] for general information.
func CountOut(in tensor.Tensor, out tensor.Values) error {
	_, err := CountOut64(in, out)
	return err
}

// SumOut64 computes the sum of tensor values,
// and returns the Float64 output values for subsequent use.
func SumOut64(in tensor.Tensor, out tensor.Values) (tensor.Tensor, error) {
	return VectorizeOut64(NFunc, func(idx int, tsr ...tensor.Tensor) {
		VecFunc(idx, tsr[0], tsr[1], 0, func(val, agg float64) float64 {
			return agg + val
		})
	}, in, out)
}

// SumOut computes the sum of tensor values.
// See [StatsOutFunc] for general information.
func SumOut(in tensor.Tensor, out tensor.Values) error {
	_, err := SumOut64(in, out)
	return err
}

// Sum computes the sum of tensor values.
// See [StatsFunc] for general information.
func Sum(in tensor.Tensor) tensor.Values {
	out := tensor.NewOfType(in.DataType())
	errors.Log1(SumOut64(in, out))
	return out
}

// NormL1 computes the sum of absolute-value-of tensor values.
// This is also known as the L1 norm.
// See [StatsFunc] for general information.
func NormL1(in tensor.Tensor) tensor.Values {
	return tensor.CallOut1(NormL1Out, in)
}

// NormL1Out computes the sum of absolute-value-of tensor values.
// This is also known as the L1 norm.
// See [StatsFunc] for general information.
func NormL1Out(in tensor.Tensor, out tensor.Values) error {
	_, err := VectorizeOut64(NFunc, func(idx int, tsr ...tensor.Tensor) {
		VecFunc(idx, tsr[0], tsr[1], 0, func(val, agg float64) float64 {
			return agg + math.Abs(val)
		})
	}, in, out)
	return err
}

// Prod computes the product of tensor values.
// See [StatsFunc] for general information.
func Prod(in tensor.Tensor) tensor.Values {
	return tensor.CallOut1(ProdOut, in)
}

// ProdOut computes the product of tensor values.
// See [StatsOutFunc] for general information.
func ProdOut(in tensor.Tensor, out tensor.Values) error {
	_, err := VectorizeOut64(NFunc, func(idx int, tsr ...tensor.Tensor) {
		VecFunc(idx, tsr[0], tsr[1], 1, func(val, agg float64) float64 {
			return agg * val
		})
	}, in, out)
	return err
}

// Min computes the min of tensor values.
// See [StatsFunc] for general information.
func Min(in tensor.Tensor) tensor.Values {
	return tensor.CallOut1(MinOut, in)
}

// MinOut computes the min of tensor values.
// See [StatsOutFunc] for general information.
func MinOut(in tensor.Tensor, out tensor.Values) error {
	_, err := VectorizeOut64(NFunc, func(idx int, tsr ...tensor.Tensor) {
		VecFunc(idx, tsr[0], tsr[1], math.MaxFloat64, func(val, agg float64) float64 {
			return math.Min(agg, val)
		})
	}, in, out)
	return err
}

// Max computes the max of tensor values.
// See [StatsFunc] for general information.
func Max(in tensor.Tensor) tensor.Values {
	return tensor.CallOut1(MaxOut, in)
}

// MaxOut computes the max of tensor values.
// See [StatsOutFunc] for general information.
func MaxOut(in tensor.Tensor, out tensor.Values) error {
	_, err := VectorizeOut64(NFunc, func(idx int, tsr ...tensor.Tensor) {
		VecFunc(idx, tsr[0], tsr[1], -math.MaxFloat64, func(val, agg float64) float64 {
			return math.Max(agg, val)
		})
	}, in, out)
	return err
}

// MinAbs computes the min of absolute-value-of tensor values.
// See [StatsFunc] for general information.
func MinAbs(in tensor.Tensor) tensor.Values {
	return tensor.CallOut1(MinAbsOut, in)
}

// MinAbsOut computes the min of absolute-value-of tensor values.
// See [StatsOutFunc] for general information.
func MinAbsOut(in tensor.Tensor, out tensor.Values) error {
	_, err := VectorizeOut64(NFunc, func(idx int, tsr ...tensor.Tensor) {
		VecFunc(idx, tsr[0], tsr[1], math.MaxFloat64, func(val, agg float64) float64 {
			return math.Min(agg, math.Abs(val))
		})
	}, in, out)
	return err
}

// MaxAbs computes the max of absolute-value-of tensor values.
// See [StatsFunc] for general information.
func MaxAbs(in tensor.Tensor) tensor.Values {
	return tensor.CallOut1(MaxAbsOut, in)
}

// MaxAbsOut computes the max of absolute-value-of tensor values.
// See [StatsOutFunc] for general information.
func MaxAbsOut(in tensor.Tensor, out tensor.Values) error {
	_, err := VectorizeOut64(NFunc, func(idx int, tsr ...tensor.Tensor) {
		VecFunc(idx, tsr[0], tsr[1], -math.MaxFloat64, func(val, agg float64) float64 {
			return math.Max(agg, math.Abs(val))
		})
	}, in, out)
	return err
}

// MeanOut64 computes the mean of tensor values,
// and returns the Float64 output values for subsequent use.
func MeanOut64(in tensor.Tensor, out tensor.Values) (mean64, count64 tensor.Tensor, err error) {
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

// Mean computes the mean of tensor values.
// See [StatsFunc] for general information.
func Mean(in tensor.Tensor) tensor.Values {
	out := tensor.NewOfType(in.DataType())
	errors.Log2(MeanOut64(in, out))
	return out
}

// MeanOut computes the mean of tensor values.
// See [StatsOutFunc] for general information.
func MeanOut(in tensor.Tensor, out tensor.Values) error {
	_, _, err := MeanOut64(in, out)
	return err
}

// SumSqDevOut64 computes the sum of squared mean deviates of tensor values,
// and returns the Float64 output values for subsequent use.
func SumSqDevOut64(in tensor.Tensor, out tensor.Values) (ssd64, mean64, count64 tensor.Tensor, err error) {
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
func VarOut64(in tensor.Tensor, out tensor.Values) (var64, mean64, count64 tensor.Tensor, err error) {
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

// Var computes the sample variance of tensor values.
// Squared deviations from mean, divided by n-1. See also [VarPopFunc].
// See [StatsFunc] for general information.
func Var(in tensor.Tensor) tensor.Values {
	out := tensor.NewOfType(in.DataType())
	_, _, _, err := VarOut64(in, out)
	errors.Log(err)
	return out
}

// VarOut computes the sample variance of tensor values.
// Squared deviations from mean, divided by n-1. See also [VarPopFunc].
// See [StatsOutFunc] for general information.
func VarOut(in tensor.Tensor, out tensor.Values) error {
	_, _, _, err := VarOut64(in, out)
	return err
}

// StdOut64 computes the sample standard deviation of tensor values.
// and returns the Float64 output values for subsequent use.
func StdOut64(in tensor.Tensor, out tensor.Values) (std64, mean64, count64 tensor.Tensor, err error) {
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

// Std computes the sample standard deviation of tensor values.
// Sqrt of variance from [VarFunc]. See also [StdPopFunc].
// See [StatsFunc] for general information.
func Std(in tensor.Tensor) tensor.Values {
	out := tensor.NewOfType(in.DataType())
	_, _, _, err := StdOut64(in, out)
	errors.Log(err)
	return out
}

// StdOut computes the sample standard deviation of tensor values.
// Sqrt of variance from [VarFunc]. See also [StdPopFunc].
// See [StatsOutFunc] for general information.
func StdOut(in tensor.Tensor, out tensor.Values) error {
	_, _, _, err := StdOut64(in, out)
	return err
}

// Sem computes the sample standard error of the mean of tensor values.
// Standard deviation [StdFunc] / sqrt(n). See also [SemPopFunc].
// See [StatsFunc] for general information.
func Sem(in tensor.Tensor) tensor.Values {
	out := tensor.NewOfType(in.DataType())
	errors.Log(SemOut(in, out))
	return out
}

// SemOut computes the sample standard error of the mean of tensor values.
// Standard deviation [StdFunc] / sqrt(n). See also [SemPopFunc].
// See [StatsOutFunc] for general information.
func SemOut(in tensor.Tensor, out tensor.Values) error {
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
func VarPopOut64(in tensor.Tensor, out tensor.Values) (var64, mean64, count64 tensor.Tensor, err error) {
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

// VarPop computes the population variance of tensor values.
// Squared deviations from mean, divided by n. See also [VarFunc].
// See [StatsFunc] for general information.
func VarPop(in tensor.Tensor) tensor.Values {
	out := tensor.NewOfType(in.DataType())
	_, _, _, err := VarPopOut64(in, out)
	errors.Log(err)
	return out
}

// VarPopOut computes the population variance of tensor values.
// Squared deviations from mean, divided by n. See also [VarFunc].
// See [StatsOutFunc] for general information.
func VarPopOut(in tensor.Tensor, out tensor.Values) error {
	_, _, _, err := VarPopOut64(in, out)
	return err
}

// StdPop computes the population standard deviation of tensor values.
// Sqrt of variance from [VarPopFunc]. See also [StdFunc].
// See [StatsFunc] for general information.
func StdPop(in tensor.Tensor) tensor.Values {
	return tensor.CallOut1(StdPopOut, in)
}

// StdPopOut computes the population standard deviation of tensor values.
// Sqrt of variance from [VarPopFunc]. See also [StdFunc].
// See [StatsOutFunc] for general information.
func StdPopOut(in tensor.Tensor, out tensor.Values) error {
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

// SemPop computes the population standard error of the mean of tensor values.
// Standard deviation [StdPopFunc] / sqrt(n). See also [SemFunc].
// See [StatsFunc] for general information.
func SemPop(in tensor.Tensor) tensor.Values {
	return tensor.CallOut1(SemPopOut, in)
}

// SemPopOut computes the population standard error of the mean of tensor values.
// Standard deviation [StdPopFunc] / sqrt(n). See also [SemFunc].
// See [StatsOutFunc] for general information.
func SemPopOut(in tensor.Tensor, out tensor.Values) error {
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
func SumSqScaleOut64(in tensor.Tensor, out tensor.Values) (scale64, ss64 tensor.Tensor, err error) {
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
func SumSqOut64(in tensor.Tensor, out tensor.Values) (tensor.Tensor, error) {
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

// SumSq computes the sum of squares of tensor values,
// See [StatsFunc] for general information.
func SumSq(in tensor.Tensor) tensor.Values {
	out := tensor.NewOfType(in.DataType())
	errors.Log1(SumSqOut64(in, out))
	return out
}

// SumSqOut computes the sum of squares of tensor values,
// See [StatsOutFunc] for general information.
func SumSqOut(in tensor.Tensor, out tensor.Values) error {
	_, err := SumSqOut64(in, out)
	return err
}

// NormL2Out64 computes the square root of the sum of squares of tensor values,
// known as the L2 norm, and returns the Float64 output values for
// use in subsequent computations.
func NormL2Out64(in tensor.Tensor, out tensor.Values) (tensor.Tensor, error) {
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

// NormL2 computes the square root of the sum of squares of tensor values,
// known as the L2 norm.
// See [StatsFunc] for general information.
func NormL2(in tensor.Tensor) tensor.Values {
	out := tensor.NewOfType(in.DataType())
	errors.Log1(NormL2Out64(in, out))
	return out
}

// NormL2Out computes the square root of the sum of squares of tensor values,
// known as the L2 norm.
// See [StatsOutFunc] for general information.
func NormL2Out(in tensor.Tensor, out tensor.Values) error {
	_, err := NormL2Out64(in, out)
	return err
}
