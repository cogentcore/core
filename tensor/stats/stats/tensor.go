// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package stats

import (
	"math"

	"cogentcore.org/core/tensor"
)

// StatTensor returns Tensor statistic according to given Stats type applied
// to all non-NaN elements in given Tensor
func StatTensor(tsr tensor.Tensor, stat Stats) float64 {
	switch stat {
	case Count:
		return CountTensor(tsr)
	case Sum:
		return SumTensor(tsr)
	case Prod:
		return ProdTensor(tsr)
	case Min:
		return MinTensor(tsr)
	case Max:
		return MaxTensor(tsr)
	case MinAbs:
		return MinAbsTensor(tsr)
	case MaxAbs:
		return MaxAbsTensor(tsr)
	case Mean:
		return MeanTensor(tsr)
	case Var:
		return VarTensor(tsr)
	case Std:
		return StdTensor(tsr)
	case Sem:
		return SemTensor(tsr)
	case L1Norm:
		return L1NormTensor(tsr)
	case SumSq:
		return SumSqTensor(tsr)
	case L2Norm:
		return L2NormTensor(tsr)
	case VarPop:
		return VarPopTensor(tsr)
	case StdPop:
		return StdPopTensor(tsr)
	case SemPop:
		return SemPopTensor(tsr)
		// case Median:
		// 	return MedianTensor(tsr)
		// case Q1:
		// 	return Q1Tensor(tsr)
		// case Q3:
		// 	return Q3Tensor(tsr)
	}
	return 0
}

// TensorStat applies given StatFunc function to each element in the tensor
// (automatically skips NaN elements), using float64 conversions of the values.
// ini is the initial value for the agg variable. returns final aggregate value
func TensorStat(tsr tensor.Tensor, ini float64, fun StatFunc) float64 {
	ln := tsr.Len()
	agg := ini
	for j := 0; j < ln; j++ {
		val := tsr.Float1D(j)
		if !math.IsNaN(val) {
			agg = fun(j, val, agg)
		}
	}
	return agg
}

// CountTensor returns the count of non-NaN elements in given Tensor.
func CountTensor(tsr tensor.Tensor) float64 {
	return TensorStat(tsr, 0, CountFunc)
}

// SumTensor returns the sum of non-NaN elements in given Tensor.
func SumTensor(tsr tensor.Tensor) float64 {
	return TensorStat(tsr, 0, SumFunc)
}

// ProdTensor returns the product of non-NaN elements in given Tensor.
func ProdTensor(tsr tensor.Tensor) float64 {
	return TensorStat(tsr, 1, ProdFunc)
}

// MinTensor returns the minimum of non-NaN elements in given Tensor.
func MinTensor(tsr tensor.Tensor) float64 {
	return TensorStat(tsr, math.MaxFloat64, MinFunc)
}

// MaxTensor returns the maximum of non-NaN elements in given Tensor.
func MaxTensor(tsr tensor.Tensor) float64 {
	return TensorStat(tsr, -math.MaxFloat64, MaxFunc)
}

// MinAbsTensor returns the minimum of non-NaN elements in given Tensor.
func MinAbsTensor(tsr tensor.Tensor) float64 {
	return TensorStat(tsr, math.MaxFloat64, MinAbsFunc)
}

// MaxAbsTensor returns the maximum of non-NaN elements in given Tensor.
func MaxAbsTensor(tsr tensor.Tensor) float64 {
	return TensorStat(tsr, -math.MaxFloat64, MaxAbsFunc)
}

// MeanTensor returns the mean of non-NaN elements in given Tensor.
func MeanTensor(tsr tensor.Tensor) float64 {
	cnt := CountTensor(tsr)
	if cnt == 0 {
		return 0
	}
	return SumTensor(tsr) / cnt
}

// VarTensor returns the sample variance of non-NaN elements in given Tensor.
func VarTensor(tsr tensor.Tensor) float64 {
	cnt := CountTensor(tsr)
	if cnt < 2 {
		return 0
	}
	mean := SumTensor(tsr) / cnt
	vr := TensorStat(tsr, 0, func(idx int, val float64, agg float64) float64 {
		dv := val - mean
		return agg + dv*dv
	})
	return vr / (cnt - 1)
}

// StdTensor returns the sample standard deviation of non-NaN elements in given Tensor.
func StdTensor(tsr tensor.Tensor) float64 {
	return math.Sqrt(VarTensor(tsr))
}

// SemTensor returns the sample standard error of the mean of non-NaN elements in given Tensor.
func SemTensor(tsr tensor.Tensor) float64 {
	cnt := CountTensor(tsr)
	if cnt < 2 {
		return 0
	}
	return StdTensor(tsr) / math.Sqrt(cnt)
}

// L1NormTensor returns the L1 norm: sum of absolute values of non-NaN elements in given Tensor.
func L1NormTensor(tsr tensor.Tensor) float64 {
	return TensorStat(tsr, 0, L1NormFunc)
}

// SumSqTensor returns the sum-of-squares of non-NaN elements in given Tensor.
func SumSqTensor(tsr tensor.Tensor) float64 {
	n := tsr.Len()
	if n < 2 {
		if n == 1 {
			return math.Abs(tsr.Float1D(0))
		}
		return 0
	}
	var (
		scale float64 = 0
		ss    float64 = 1
	)
	for j := 0; j < n; j++ {
		v := tsr.Float1D(j)
		if v == 0 || math.IsNaN(v) {
			continue
		}
		absxi := math.Abs(v)
		if scale < absxi {
			ss = 1 + ss*(scale/absxi)*(scale/absxi)
			scale = absxi
		} else {
			ss = ss + (absxi/scale)*(absxi/scale)
		}
	}
	if math.IsInf(scale, 1) {
		return math.Inf(1)
	}
	return scale * scale * ss
}

// L2NormTensor returns the L2 norm: square root of sum-of-squared values of non-NaN elements in given Tensor.
func L2NormTensor(tsr tensor.Tensor) float64 {
	return math.Sqrt(SumSqTensor(tsr))
}

// VarPopTensor returns the population variance of non-NaN elements in given Tensor.
func VarPopTensor(tsr tensor.Tensor) float64 {
	cnt := CountTensor(tsr)
	if cnt < 2 {
		return 0
	}
	mean := SumTensor(tsr) / cnt
	vr := TensorStat(tsr, 0, func(idx int, val float64, agg float64) float64 {
		dv := val - mean
		return agg + dv*dv
	})
	return vr / cnt
}

// StdPopTensor returns the population standard deviation of non-NaN elements in given Tensor.
func StdPopTensor(tsr tensor.Tensor) float64 {
	return math.Sqrt(VarPopTensor(tsr))
}

// SemPopTensor returns the population standard error of the mean of non-NaN elements in given Tensor.
func SemPopTensor(tsr tensor.Tensor) float64 {
	cnt := CountTensor(tsr)
	if cnt < 2 {
		return 0
	}
	return StdPopTensor(tsr) / math.Sqrt(cnt)
}
