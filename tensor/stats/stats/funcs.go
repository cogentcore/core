// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package stats

import "cogentcore.org/core/tensor"

// StatsFunc is the function signature for a stats function,
// where the output has the same shape as the input but with
// an outer-most row dimension size of 1, and contains
// the stat value(s) for the "cells" in higher-dimensional tensors,
// and a single scalar value for a 1D input tensor.
// All stats functions skip over NaN's, as a not-present value.
// Stats functions cannot be computed in parallel,
// e.g., using VectorizeThreaded or GPU, due to shared writing
// to the same output values.  Special implementations are required
// if that is needed.
type StatsFunc func(in, out *tensor.Indexed)

// CountFunc computes the count of non-NaN tensor values.
// See [StatsFunc] for general information.
func CountFunc(in, out *tensor.Indexed) {
	tensor.Vectorize(NFunc, CountVecFunc, in, out)
}

// SumFunc computes the sum of tensor values.
// See [StatsFunc] for general information.
func SumFunc(in, out *tensor.Indexed) {
	tensor.Vectorize(NFunc, SumVecFunc, in, out)
}

// ProdFunc computes the product of tensor values.
// See [StatsFunc] for general information.
func ProdFunc(in, out *tensor.Indexed) {
	tensor.Vectorize(NFunc, ProdVecFunc, in, out)
}

// MinFunc computes the min of tensor values.
// See [StatsFunc] for general information.
func MinFunc(in, out *tensor.Indexed) {
	tensor.Vectorize(NFunc, MinVecFunc, in, out)
}

// MaxFunc computes the max of tensor values.
// See [StatsFunc] for general information.
func MaxFunc(in, out *tensor.Indexed) {
	tensor.Vectorize(NFunc, MaxVecFunc, in, out)
}

// MinAbsFunc computes the min of absolute-value-of tensor values.
// See [StatsFunc] for general information.
func MinAbsFunc(in, out *tensor.Indexed) {
	tensor.Vectorize(NFunc, MinAbsVecFunc, in, out)
}

// MaxAbsFunc computes the max of absolute-value-of tensor values.
// See [StatsFunc] for general information.
func MaxAbsFunc(in, out *tensor.Indexed) {
	tensor.Vectorize(NFunc, MaxAbsVecFunc, in, out)
}

/*
///////////////////////////////////////////
//  Mean

// Mean32 computes the mean of the vector (sum / N).
// Skips NaN's
func Mean32(a []float32) float32 {
	s := float32(0)
	n := 0
	for _, av := range a {
		if math32.IsNaN(av) {
			continue
		}
		s += av
		n++
	}
	if n > 0 {
		s /= float32(n)
	}
	return s
}

// Mean64 computes the mean of the vector (sum / N).
// Skips NaN's
func Mean64(a []float64) float64 {
	s := float64(0)
	n := 0
	for _, av := range a {
		if math.IsNaN(av) {
			continue
		}
		s += av
		n++
	}
	if n > 0 {
		s /= float64(n)
	}
	return s
}

///////////////////////////////////////////
//  Var

// Var32 returns the sample variance of non-NaN elements.
func Var32(a []float32) float32 {
	mean := Mean32(a)
	n := 0
	s := float32(0)
	for _, av := range a {
		if math32.IsNaN(av) {
			continue
		}
		dv := av - mean
		s += dv * dv
		n++
	}
	if n > 1 {
		s /= float32(n - 1)
	}
	return s
}

// Var64 returns the sample variance of non-NaN elements.
func Var64(a []float64) float64 {
	mean := Mean64(a)
	n := 0
	s := float64(0)
	for _, av := range a {
		if math.IsNaN(av) {
			continue
		}
		dv := av - mean
		s += dv * dv
		n++
	}
	if n > 1 {
		s /= float64(n - 1)
	}
	return s
}

///////////////////////////////////////////
//  Std

// Std32 returns the sample standard deviation of non-NaN elements in vector.
func Std32(a []float32) float32 {
	return math32.Sqrt(Var32(a))
}

// Std64 returns the sample standard deviation of non-NaN elements in vector.
func Std64(a []float64) float64 {
	return math.Sqrt(Var64(a))
}

///////////////////////////////////////////
//  Sem

// Sem32 returns the sample standard error of the mean of non-NaN elements in vector.
func Sem32(a []float32) float32 {
	cnt := Count32(a)
	if cnt < 2 {
		return 0
	}
	return Std32(a) / math32.Sqrt(cnt)
}

// Sem64 returns the sample standard error of the mean of non-NaN elements in vector.
func Sem64(a []float64) float64 {
	cnt := Count64(a)
	if cnt < 2 {
		return 0
	}
	return Std64(a) / math.Sqrt(cnt)
}

///////////////////////////////////////////
//  L1Norm

// L1Norm32 computes the sum of absolute values (L1 Norm).
// Skips NaN's
func L1Norm32(a []float32) float32 {
	ss := float32(0)
	for _, av := range a {
		if math32.IsNaN(av) {
			continue
		}
		ss += math32.Abs(av)
	}
	return ss
}

// L1Norm64 computes the sum of absolute values (L1 Norm).
// Skips NaN's
func L1Norm64(a []float64) float64 {
	ss := float64(0)
	for _, av := range a {
		if math.IsNaN(av) {
			continue
		}
		ss += math.Abs(av)
	}
	return ss
}

///////////////////////////////////////////
//  SumSquares

// SumSq32 computes the sum-of-squares of vector.
// Skips NaN's.  Uses optimized algorithm from BLAS that avoids numerical overflow.
func SumSq32(a []float32) float32 {
	n := len(a)
	if n < 2 {
		if n == 1 {
			return math32.Abs(a[0])
		}
		return 0
	}
	var (
		scale      float32 = 0
		sumSquares float32 = 1
	)
	for _, v := range a {
		if v == 0 || math32.IsNaN(v) {
			continue
		}
		absxi := math32.Abs(v)
		if scale < absxi {
			sumSquares = 1 + sumSquares*(scale/absxi)*(scale/absxi)
			scale = absxi
		} else {
			sumSquares = sumSquares + (absxi/scale)*(absxi/scale)
		}
	}
	if math32.IsInf(scale, 1) {
		return math32.Inf(1)
	}
	return scale * scale * sumSquares
}

// SumSq64 computes the sum-of-squares of vector.
// Skips NaN's. Uses optimized algorithm from BLAS that avoids numerical overflow.
func SumSq64(a []float64) float64 {
	n := len(a)
	if n < 2 {
		if n == 1 {
			return math.Abs(a[0])
		}
		return 0
	}
	var (
		scale float64 = 0
		ss    float64 = 1
	)
	for _, v := range a {
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

///////////////////////////////////////////
//  L2Norm

// L2Norm32 computes the square-root of sum-of-squares of vector, i.e., the L2 norm.
// Skips NaN's. Uses optimized algorithm from BLAS that avoids numerical overflow.
func L2Norm32(a []float32) float32 {
	n := len(a)
	if n < 2 {
		if n == 1 {
			return math32.Abs(a[0])
		}
		return 0
	}
	var (
		scale float32 = 0
		ss    float32 = 1
	)
	for _, v := range a {
		if v == 0 || math32.IsNaN(v) {
			continue
		}
		absxi := math32.Abs(v)
		if scale < absxi {
			ss = 1 + ss*(scale/absxi)*(scale/absxi)
			scale = absxi
		} else {
			ss = ss + (absxi/scale)*(absxi/scale)
		}
	}
	if math32.IsInf(scale, 1) {
		return math32.Inf(1)
	}
	return scale * math32.Sqrt(ss)
}

// L2Norm64 computes the square-root of sum-of-squares of vector, i.e., the L2 norm.
// Skips NaN's.  Uses optimized algorithm from BLAS that avoids numerical overflow.
func L2Norm64(a []float64) float64 {
	n := len(a)
	if n < 2 {
		if n == 1 {
			return math.Abs(a[0])
		}
		return 0
	}
	var (
		scale float64 = 0
		ss    float64 = 1
	)
	for _, v := range a {
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
	return scale * math.Sqrt(ss)
}

///////////////////////////////////////////
//  VarPop

// VarPop32 returns the population variance of non-NaN elements.
func VarPop32(a []float32) float32 {
	mean := Mean32(a)
	n := 0
	s := float32(0)
	for _, av := range a {
		if math32.IsNaN(av) {
			continue
		}
		dv := av - mean
		s += dv * dv
		n++
	}
	if n > 0 {
		s /= float32(n)
	}
	return s
}

// VarPop64 returns the population variance of non-NaN elements.
func VarPop64(a []float64) float64 {
	mean := Mean64(a)
	n := 0
	s := float64(0)
	for _, av := range a {
		if math.IsNaN(av) {
			continue
		}
		dv := av - mean
		s += dv * dv
		n++
	}
	if n > 0 {
		s /= float64(n)
	}
	return s
}

///////////////////////////////////////////
//  StdPop

// StdPop32 returns the population standard deviation of non-NaN elements in vector.
func StdPop32(a []float32) float32 {
	return math32.Sqrt(VarPop32(a))
}

// StdPop64 returns the population standard deviation of non-NaN elements in vector.
func StdPop64(a []float64) float64 {
	return math.Sqrt(VarPop64(a))
}

///////////////////////////////////////////
//  SemPop

// SemPop32 returns the population standard error of the mean of non-NaN elements in vector.
func SemPop32(a []float32) float32 {
	cnt := Count32(a)
	if cnt < 2 {
		return 0
	}
	return StdPop32(a) / math32.Sqrt(cnt)
}

// SemPop64 returns the population standard error of the mean of non-NaN elements in vector.
func SemPop64(a []float64) float64 {
	cnt := Count64(a)
	if cnt < 2 {
		return 0
	}
	return StdPop64(a) / math.Sqrt(cnt)
}

*/
