// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package stats

import (
	"math"

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
func CountOut64(in tensor.Tensor, out tensor.Values) *tensor.Float64 {
	return VectorizeOut64(in, out, 0, func(val, agg float64) float64 {
		if math.IsNaN(val) {
			return agg
		}
		return agg + 1
	})
}

// Count computes the count of non-NaN tensor values.
// See [StatsFunc] for general information.
func Count(in tensor.Tensor) tensor.Values {
	return tensor.CallOut1(CountOut, in)
}

// CountOut computes the count of non-NaN tensor values.
// See [StatsOutFunc] for general information.
func CountOut(in tensor.Tensor, out tensor.Values) error {
	CountOut64(in, out)
	return nil
}

// SumOut64 computes the sum of tensor values,
// and returns the Float64 output values for subsequent use.
func SumOut64(in tensor.Tensor, out tensor.Values) *tensor.Float64 {
	return VectorizeOut64(in, out, 0, func(val, agg float64) float64 {
		if math.IsNaN(val) {
			return agg
		}
		return agg + val
	})
}

// SumOut computes the sum of tensor values.
// See [StatsOutFunc] for general information.
func SumOut(in tensor.Tensor, out tensor.Values) error {
	SumOut64(in, out)
	return nil
}

// Sum computes the sum of tensor values.
// See [StatsFunc] for general information.
func Sum(in tensor.Tensor) tensor.Values {
	out := tensor.NewOfType(in.DataType())
	SumOut64(in, out)
	return out
}

// L1Norm computes the sum of absolute-value-of tensor values.
// See [StatsFunc] for general information.
func L1Norm(in tensor.Tensor) tensor.Values {
	return tensor.CallOut1(L1NormOut, in)
}

// L1NormOut computes the sum of absolute-value-of tensor values.
// See [StatsFunc] for general information.
func L1NormOut(in tensor.Tensor, out tensor.Values) error {
	VectorizeOut64(in, out, 0, func(val, agg float64) float64 {
		if math.IsNaN(val) {
			return agg
		}
		return agg + math.Abs(val)
	})
	return nil
}

// Prod computes the product of tensor values.
// See [StatsFunc] for general information.
func Prod(in tensor.Tensor) tensor.Values {
	return tensor.CallOut1(ProdOut, in)
}

// ProdOut computes the product of tensor values.
// See [StatsOutFunc] for general information.
func ProdOut(in tensor.Tensor, out tensor.Values) error {
	VectorizeOut64(in, out, 1, func(val, agg float64) float64 {
		if math.IsNaN(val) {
			return agg
		}
		return agg * val
	})
	return nil
}

// Min computes the min of tensor values.
// See [StatsFunc] for general information.
func Min(in tensor.Tensor) tensor.Values {
	return tensor.CallOut1(MinOut, in)
}

// MinOut computes the min of tensor values.
// See [StatsOutFunc] for general information.
func MinOut(in tensor.Tensor, out tensor.Values) error {
	VectorizeOut64(in, out, math.MaxFloat64, func(val, agg float64) float64 {
		if math.IsNaN(val) {
			return agg
		}
		return math.Min(agg, val)
	})
	return nil
}

// Max computes the max of tensor values.
// See [StatsFunc] for general information.
func Max(in tensor.Tensor) tensor.Values {
	return tensor.CallOut1(MaxOut, in)
}

// MaxOut computes the max of tensor values.
// See [StatsOutFunc] for general information.
func MaxOut(in tensor.Tensor, out tensor.Values) error {
	VectorizeOut64(in, out, -math.MaxFloat64, func(val, agg float64) float64 {
		if math.IsNaN(val) {
			return agg
		}
		return math.Max(agg, val)
	})
	return nil
}

// MinAbs computes the min of absolute-value-of tensor values.
// See [StatsFunc] for general information.
func MinAbs(in tensor.Tensor) tensor.Values {
	return tensor.CallOut1(MinAbsOut, in)
}

// MinAbsOut computes the min of absolute-value-of tensor values.
// See [StatsOutFunc] for general information.
func MinAbsOut(in tensor.Tensor, out tensor.Values) error {
	VectorizeOut64(in, out, math.MaxFloat64, func(val, agg float64) float64 {
		if math.IsNaN(val) {
			return agg
		}
		return math.Min(agg, math.Abs(val))
	})
	return nil
}

// MaxAbs computes the max of absolute-value-of tensor values.
// See [StatsFunc] for general information.
func MaxAbs(in tensor.Tensor) tensor.Values {
	return tensor.CallOut1(MaxAbsOut, in)
}

// MaxAbsOut computes the max of absolute-value-of tensor values.
// See [StatsOutFunc] for general information.
func MaxAbsOut(in tensor.Tensor, out tensor.Values) error {
	VectorizeOut64(in, out, -math.MaxFloat64, func(val, agg float64) float64 {
		if math.IsNaN(val) {
			return agg
		}
		return math.Max(agg, math.Abs(val))
	})
	return nil
}

// MeanOut64 computes the mean of tensor values,
// and returns the Float64 output values for subsequent use.
func MeanOut64(in tensor.Tensor, out tensor.Values) (mean64, count64 *tensor.Float64) {
	var sum64 *tensor.Float64
	sum64, count64 = Vectorize2Out64(in, 0, 0, func(val, sum, count float64) (float64, float64) {
		if math.IsNaN(val) {
			return sum, count
		}
		count += 1
		sum += val
		return sum, count
	})
	osz := tensor.CellsSize(in.ShapeSizes())
	out.SetShapeSizes(osz...)
	nsub := out.Len()
	for i := range nsub {
		c := count64.Float1D(i)
		if c == 0 {
			continue
		}
		mean := sum64.Float1D(i) / c
		sum64.SetFloat1D(mean, i)
		out.SetFloat1D(mean, i)
	}
	return sum64, count64
}

// Mean computes the mean of tensor values.
// See [StatsFunc] for general information.
func Mean(in tensor.Tensor) tensor.Values {
	return tensor.CallOut1(MeanOut, in)
}

// MeanOut computes the mean of tensor values.
// See [StatsOutFunc] for general information.
func MeanOut(in tensor.Tensor, out tensor.Values) error {
	MeanOut64(in, out)
	return nil
}

// SumSqDevOut64 computes the sum of squared mean deviates of tensor values,
// and returns the Float64 output values for subsequent use.
func SumSqDevOut64(in tensor.Tensor, out tensor.Values) (ssd64, mean64, count64 *tensor.Float64) {
	mean64, count64 = MeanOut64(in, out)
	ssd64 = VectorizePreOut64(in, out, 0, mean64, func(val, mean, agg float64) float64 {
		if math.IsNaN(val) {
			return agg
		}
		dv := val - mean
		return agg + dv*dv
	})
	return
}

// VarOut64 computes the sample variance of tensor values,
// and returns the Float64 output values for subsequent use.
func VarOut64(in tensor.Tensor, out tensor.Values) (var64, mean64, count64 *tensor.Float64) {
	var64, mean64, count64 = SumSqDevOut64(in, out)
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
	return tensor.CallOut1(VarOut, in)
}

// VarOut computes the sample variance of tensor values.
// Squared deviations from mean, divided by n-1. See also [VarPopFunc].
// See [StatsOutFunc] for general information.
func VarOut(in tensor.Tensor, out tensor.Values) error {
	VarOut64(in, out)
	return nil
}

// StdOut64 computes the sample standard deviation of tensor values.
// and returns the Float64 output values for subsequent use.
func StdOut64(in tensor.Tensor, out tensor.Values) (std64, mean64, count64 *tensor.Float64) {
	std64, mean64, count64 = VarOut64(in, out)
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
	return tensor.CallOut1(StdOut, in)
}

// StdOut computes the sample standard deviation of tensor values.
// Sqrt of variance from [VarFunc]. See also [StdPopFunc].
// See [StatsOutFunc] for general information.
func StdOut(in tensor.Tensor, out tensor.Values) error {
	StdOut64(in, out)
	return nil
}

// Sem computes the sample standard error of the mean of tensor values.
// Standard deviation [StdFunc] / sqrt(n). See also [SemPopFunc].
// See [StatsFunc] for general information.
func Sem(in tensor.Tensor) tensor.Values {
	return tensor.CallOut1(SemOut, in)
}

// SemOut computes the sample standard error of the mean of tensor values.
// Standard deviation [StdFunc] / sqrt(n). See also [SemPopFunc].
// See [StatsOutFunc] for general information.
func SemOut(in tensor.Tensor, out tensor.Values) error {
	var64, _, count64 := VarOut64(in, out)
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
func VarPopOut64(in tensor.Tensor, out tensor.Values) (var64, mean64, count64 *tensor.Float64) {
	var64, mean64, count64 = SumSqDevOut64(in, out)
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
	return tensor.CallOut1(VarPopOut, in)
}

// VarPopOut computes the population variance of tensor values.
// Squared deviations from mean, divided by n. See also [VarFunc].
// See [StatsOutFunc] for general information.
func VarPopOut(in tensor.Tensor, out tensor.Values) error {
	VarPopOut64(in, out)
	return nil
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
	var64, _, _ := VarPopOut64(in, out)
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
	var64, _, count64 := VarPopOut64(in, out)
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
func SumSqScaleOut64(in tensor.Tensor) (scale64, ss64 *tensor.Float64) {
	scale64, ss64 = Vectorize2Out64(in, 0, 1, func(val, scale, ss float64) (float64, float64) {
		if math.IsNaN(val) || val == 0 {
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
	return
}

// SumSqOut64 computes the sum of squares of tensor values,
// and returns the Float64 output values for subsequent use.
func SumSqOut64(in tensor.Tensor, out tensor.Values) *tensor.Float64 {
	scale64, ss64 := SumSqScaleOut64(in)
	osz := tensor.CellsSize(in.ShapeSizes())
	out.SetShapeSizes(osz...)
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
	return scale64
}

// SumSq computes the sum of squares of tensor values,
// See [StatsFunc] for general information.
func SumSq(in tensor.Tensor) tensor.Values {
	return tensor.CallOut1(SumSqOut, in)
}

// SumSqOut computes the sum of squares of tensor values,
// See [StatsOutFunc] for general information.
func SumSqOut(in tensor.Tensor, out tensor.Values) error {
	SumSqOut64(in, out)
	return nil
}

// L2NormOut64 computes the square root of the sum of squares of tensor values,
// known as the L2 norm, and returns the Float64 output values for
// use in subsequent computations.
func L2NormOut64(in tensor.Tensor, out tensor.Values) *tensor.Float64 {
	scale64, ss64 := SumSqScaleOut64(in)
	osz := tensor.CellsSize(in.ShapeSizes())
	out.SetShapeSizes(osz...)
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
	return scale64
}

// L2Norm computes the square root of the sum of squares of tensor values,
// known as the L2 norm.
// See [StatsFunc] for general information.
func L2Norm(in tensor.Tensor) tensor.Values {
	return tensor.CallOut1(L2NormOut, in)
}

// L2NormOut computes the square root of the sum of squares of tensor values,
// known as the L2 norm.
// See [StatsOutFunc] for general information.
func L2NormOut(in tensor.Tensor, out tensor.Values) error {
	L2NormOut64(in, out)
	return nil
}

// First returns the first tensor value(s), as a stats function,
// for the starting point in a naturally-ordered set of data.
// See [StatsFunc] for general information.
func First(in tensor.Tensor) tensor.Values {
	return tensor.CallOut1(FirstOut, in)
}

// FirstOut returns the first tensor value(s), as a stats function,
// for the starting point in a naturally-ordered set of data.
// See [StatsOutFunc] for general information.
func FirstOut(in tensor.Tensor, out tensor.Values) error {
	rows, cells := in.Shape().RowCellSize()
	if cells == 1 {
		out.SetShapeSizes(1)
		if rows > 0 {
			out.SetFloat1D(in.Float1D(0), 0)
		}
		return nil
	}
	osz := tensor.CellsSize(in.ShapeSizes())
	out.SetShapeSizes(osz...)
	if rows == 0 {
		return nil
	}
	for i := range cells {
		out.SetFloat1D(in.Float1D(i), i)
	}
	return nil
}

// Final returns the final tensor value(s), as a stats function,
// for the ending point in a naturally-ordered set of data.
// See [StatsFunc] for general information.
func Final(in tensor.Tensor) tensor.Values {
	return tensor.CallOut1(FinalOut, in)
}

// FinalOut returns the first tensor value(s), as a stats function,
// for the ending point in a naturally-ordered set of data.
// See [StatsOutFunc] for general information.
func FinalOut(in tensor.Tensor, out tensor.Values) error {
	rows, cells := in.Shape().RowCellSize()
	if cells == 1 {
		out.SetShapeSizes(1)
		if rows > 0 {
			out.SetFloat1D(in.Float1D(rows-1), 0)
		}
		return nil
	}
	osz := tensor.CellsSize(in.ShapeSizes())
	out.SetShapeSizes(osz...)
	if rows == 0 {
		return nil
	}
	st := (rows - 1) * cells
	for i := range cells {
		out.SetFloat1D(in.Float1D(st+i), i)
	}
	return nil
}
