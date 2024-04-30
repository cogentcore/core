// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package stats

import (
	"math"

	"cogentcore.org/core/math32"
)

// Stat32 returns statistic according to given Stats type applied
// to all non-NaN elements in given slice of float32
func Stat32(a []float32, stat Stats) float32 {
	switch stat {
	case Count:
		return Count32(a)
	case Sum:
		return Sum32(a)
	case Prod:
		return Prod32(a)
	case Min:
		return Min32(a)
	case Max:
		return Max32(a)
	case MinAbs:
		return MinAbs32(a)
	case MaxAbs:
		return MaxAbs32(a)
	case Mean:
		return Mean32(a)
	case Var:
		return Var32(a)
	case Std:
		return Std32(a)
	case Sem:
		return Sem32(a)
	case L1Norm:
		return L1Norm32(a)
	case SumSq:
		return SumSq32(a)
	case L2Norm:
		return L2Norm32(a)
	case VarPop:
		return VarPop32(a)
	case StdPop:
		return StdPop32(a)
	case SemPop:
		return SemPop32(a)
		// case Median:
		// 	return Median32(a)
		// case Q1:
		// 	return Q132(a)
		// case Q3:
		// 	return Q332(a)
	}
	return 0
}

// Stat64 returns statistic according to given Stats type applied
// to all non-NaN elements in given slice of float64
func Stat64(a []float64, stat Stats) float64 {
	switch stat {
	case Count:
		return Count64(a)
	case Sum:
		return Sum64(a)
	case Prod:
		return Prod64(a)
	case Min:
		return Min64(a)
	case Max:
		return Max64(a)
	case MinAbs:
		return MinAbs64(a)
	case MaxAbs:
		return MaxAbs64(a)
	case Mean:
		return Mean64(a)
	case Var:
		return Var64(a)
	case Std:
		return Std64(a)
	case Sem:
		return Sem64(a)
	case L1Norm:
		return L1Norm64(a)
	case SumSq:
		return SumSq64(a)
	case L2Norm:
		return L2Norm64(a)
	case VarPop:
		return VarPop64(a)
	case StdPop:
		return StdPop64(a)
	case SemPop:
		return SemPop64(a)
		// case Median:
		// 	return Median64(a)
		// case Q1:
		// 	return Q164(a)
		// case Q3:
		// 	return Q364(a)
	}
	return 0
}

///////////////////////////////////////////
//  Count

// Count32 computes the number of non-NaN vector values.
// Skips NaN's
func Count32(a []float32) float32 {
	n := 0
	for _, av := range a {
		if math32.IsNaN(av) {
			continue
		}
		n++
	}
	return float32(n)
}

// Count64 computes the number of non-NaN vector values.
// Skips NaN's
func Count64(a []float64) float64 {
	n := 0
	for _, av := range a {
		if math.IsNaN(av) {
			continue
		}
		n++
	}
	return float64(n)
}

///////////////////////////////////////////
//  Sum

// Sum32 computes the sum of vector values.
// Skips NaN's
func Sum32(a []float32) float32 {
	s := float32(0)
	for _, av := range a {
		if math32.IsNaN(av) {
			continue
		}
		s += av
	}
	return s
}

// Sum64 computes the sum of vector values.
// Skips NaN's
func Sum64(a []float64) float64 {
	s := float64(0)
	for _, av := range a {
		if math.IsNaN(av) {
			continue
		}
		s += av
	}
	return s
}

///////////////////////////////////////////
//  Prod

// Prod32 computes the product of vector values.
// Skips NaN's
func Prod32(a []float32) float32 {
	s := float32(1)
	for _, av := range a {
		if math32.IsNaN(av) {
			continue
		}
		s *= av
	}
	return s
}

// Prod64 computes the product of vector values.
// Skips NaN's
func Prod64(a []float64) float64 {
	s := float64(1)
	for _, av := range a {
		if math.IsNaN(av) {
			continue
		}
		s *= av
	}
	return s
}

///////////////////////////////////////////
//  Min

// Min32 computes the max over vector values.
// Skips NaN's
func Min32(a []float32) float32 {
	m := float32(math.MaxFloat32)
	for _, av := range a {
		if math32.IsNaN(av) {
			continue
		}
		m = math32.Min(m, av)
	}
	return m
}

// MinIndex32 computes the min over vector values, and returns index of min as well
// Skips NaN's
func MinIndex32(a []float32) (float32, int) {
	m := float32(math.MaxFloat32)
	mi := -1
	for i, av := range a {
		if math32.IsNaN(av) {
			continue
		}
		if av < m {
			m = av
			mi = i
		}
	}
	return m, mi
}

// Min64 computes the max over vector values.
// Skips NaN's
func Min64(a []float64) float64 {
	m := float64(math.MaxFloat64)
	for _, av := range a {
		if math.IsNaN(av) {
			continue
		}
		m = math.Min(m, av)
	}
	return m
}

// MinIndex64 computes the min over vector values, and returns index of min as well
// Skips NaN's
func MinIndex64(a []float64) (float64, int) {
	m := float64(math.MaxFloat64)
	mi := -1
	for i, av := range a {
		if math.IsNaN(av) {
			continue
		}
		if av < m {
			m = av
			mi = i
		}
	}
	return m, mi
}

///////////////////////////////////////////
//  Max

// Max32 computes the max over vector values.
// Skips NaN's
func Max32(a []float32) float32 {
	m := float32(-math.MaxFloat32)
	for _, av := range a {
		if math32.IsNaN(av) {
			continue
		}
		m = math32.Max(m, av)
	}
	return m
}

// MaxIndex32 computes the max over vector values, and returns index of max as well
// Skips NaN's
func MaxIndex32(a []float32) (float32, int) {
	m := float32(-math.MaxFloat32)
	mi := -1
	for i, av := range a {
		if math32.IsNaN(av) {
			continue
		}
		if av > m {
			m = av
			mi = i
		}
	}
	return m, mi
}

// Max64 computes the max over vector values.
// Skips NaN's
func Max64(a []float64) float64 {
	m := float64(-math.MaxFloat64)
	for _, av := range a {
		if math.IsNaN(av) {
			continue
		}
		m = math.Max(m, av)
	}
	return m
}

// MaxIndex64 computes the max over vector values, and returns index of max as well
// Skips NaN's
func MaxIndex64(a []float64) (float64, int) {
	m := float64(-math.MaxFloat64)
	mi := -1
	for i, av := range a {
		if math.IsNaN(av) {
			continue
		}
		if av > m {
			m = av
			mi = i
		}
	}
	return m, mi
}

///////////////////////////////////////////
//  MinAbs

// MinAbs32 computes the max of absolute value over vector values.
// Skips NaN's
func MinAbs32(a []float32) float32 {
	m := float32(math.MaxFloat32)
	for _, av := range a {
		if math32.IsNaN(av) {
			continue
		}
		m = math32.Min(m, math32.Abs(av))
	}
	return m
}

// MinAbs64 computes the max over vector values.
// Skips NaN's
func MinAbs64(a []float64) float64 {
	m := float64(math.MaxFloat64)
	for _, av := range a {
		if math.IsNaN(av) {
			continue
		}
		m = math.Min(m, math.Abs(av))
	}
	return m
}

///////////////////////////////////////////
//  MaxAbs

// MaxAbs32 computes the max of absolute value over vector values.
// Skips NaN's
func MaxAbs32(a []float32) float32 {
	m := float32(0)
	for _, av := range a {
		if math32.IsNaN(av) {
			continue
		}
		m = math32.Max(m, math32.Abs(av))
	}
	return m
}

// MaxAbs64 computes the max over vector values.
// Skips NaN's
func MaxAbs64(a []float64) float64 {
	m := float64(0)
	for _, av := range a {
		if math.IsNaN(av) {
			continue
		}
		m = math.Max(m, math.Abs(av))
	}
	return m
}

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
