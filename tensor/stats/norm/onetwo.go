// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package norm

import (
	"math"

	"cogentcore.org/core/math32"
)

///////////////////////////////////////////
//  L1

// L132 computes the sum of absolute values (L1 Norm).
// Skips NaN's
func L132(a []float32) float32 {
	ss := float32(0)
	for _, av := range a {
		if math32.IsNaN(av) {
			continue
		}
		ss += math32.Abs(av)
	}
	return ss
}

// L164 computes the sum of absolute values (L1 Norm).
// Skips NaN's
func L164(a []float64) float64 {
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

// SumSquares32 computes the sum-of-squares of vector.
// Skips NaN's.  Uses optimized algorithm from BLAS that avoids numerical overflow.
func SumSquares32(a []float32) float32 {
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

// SumSquares64 computes the sum-of-squares of vector.
// Skips NaN's. Uses optimized algorithm from BLAS that avoids numerical overflow.
func SumSquares64(a []float64) float64 {
	n := len(a)
	if n < 2 {
		if n == 1 {
			return math.Abs(a[0])
		}
		return 0
	}
	var (
		scale      float64 = 0
		sumSquares float64 = 1
	)
	for _, v := range a {
		if v == 0 || math.IsNaN(v) {
			continue
		}
		absxi := math.Abs(v)
		if scale < absxi {
			sumSquares = 1 + sumSquares*(scale/absxi)*(scale/absxi)
			scale = absxi
		} else {
			sumSquares = sumSquares + (absxi/scale)*(absxi/scale)
		}
	}
	if math.IsInf(scale, 1) {
		return math.Inf(1)
	}
	return scale * scale * sumSquares
}

///////////////////////////////////////////
//  L2

// L232 computes the square-root of sum-of-squares of vector, i.e., the L2 norm.
// Skips NaN's. Uses optimized algorithm from BLAS that avoids numerical overflow.
func L232(a []float32) float32 {
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
	return scale * math32.Sqrt(sumSquares)
}

// L264 computes the square-root of sum-of-squares of vector, i.e., the L2 norm.
// Skips NaN's.  Uses optimized algorithm from BLAS that avoids numerical overflow.
func L264(a []float64) float64 {
	n := len(a)
	if n < 2 {
		if n == 1 {
			return math.Abs(a[0])
		}
		return 0
	}
	var (
		scale      float64 = 0
		sumSquares float64 = 1
	)
	for _, v := range a {
		if v == 0 || math.IsNaN(v) {
			continue
		}
		absxi := math.Abs(v)
		if scale < absxi {
			sumSquares = 1 + sumSquares*(scale/absxi)*(scale/absxi)
			scale = absxi
		} else {
			sumSquares = sumSquares + (absxi/scale)*(absxi/scale)
		}
	}
	if math.IsInf(scale, 1) {
		return math.Inf(1)
	}
	return scale * math.Sqrt(sumSquares)
}
