// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package norm

//go:generate core generate

import (
	"math"

	"cogentcore.org/core/math32"
	"cogentcore.org/core/tensor"
	"cogentcore.org/core/tensor/stats/stats"
)

// FloatFunc applies given functions to float tensor data, which is either Float32 or Float64
func FloatFunc(tsr tensor.Tensor, nfunc32 Func32, nfunc64 Func64, stIdx, nIdx int, ffunc32 func(a []float32, fun Func32), ffunc64 func(a []float64, fun Func64)) {
	switch tt := tsr.(type) {
	case *tensor.Float32:
		vals := tt.Values
		if nIdx > 0 {
			vals = vals[stIdx : stIdx+nIdx]
		}
		ffunc32(vals, nfunc32)
	case *tensor.Float64:
		vals := tt.Values
		if nIdx > 0 {
			vals = vals[stIdx : stIdx+nIdx]
		}
		ffunc64(vals, nfunc64)
	default:
		FloatOnlyError()
	}
}

///////////////////////////////////////////
//  DivNorm

// DivNorm32 does divisive normalization by given norm function
// i.e., it divides each element by the norm value computed from nfunc.
func DivNorm32(a []float32, nfunc Func32) {
	nv := nfunc(a)
	if nv != 0 {
		MultVector32(a, 1/nv)
	}
}

// DivNorm64 does divisive normalization by given norm function
// i.e., it divides each element by the norm value computed from nfunc.
func DivNorm64(a []float64, nfunc Func64) {
	nv := nfunc(a)
	if nv != 0 {
		MultVec64(a, 1/nv)
	}
}

///////////////////////////////////////////
//  SubNorm

// SubNorm32 does subtractive normalization by given norm function
// i.e., it subtracts norm computed by given function from each element.
func SubNorm32(a []float32, nfunc Func32) {
	nv := nfunc(a)
	AddVector32(a, -nv)
}

// SubNorm64 does subtractive normalization by given norm function
// i.e., it subtracts norm computed by given function from each element.
func SubNorm64(a []float64, nfunc Func64) {
	nv := nfunc(a)
	AddVec64(a, -nv)
}

///////////////////////////////////////////
//  ZScore

// ZScore32 subtracts the mean and divides by the standard deviation
func ZScore32(a []float32) {
	SubNorm32(a, stats.Mean32)
	DivNorm32(a, stats.Std32)
}

// ZScore64 subtracts the mean and divides by the standard deviation
func ZScore64(a []float64) {
	SubNorm64(a, stats.Mean64)
	DivNorm64(a, stats.Std64)
}

///////////////////////////////////////////
//  Unit

// Unit32 subtracts the min and divides by the max, so that values are in 0-1 unit range
func Unit32(a []float32) {
	SubNorm32(a, stats.Min32)
	DivNorm32(a, stats.Max32)
}

// Unit64 subtracts the min and divides by the max, so that values are in 0-1 unit range
func Unit64(a []float64) {
	SubNorm64(a, stats.Min64)
	DivNorm64(a, stats.Max64)
}

///////////////////////////////////////////
//  MultVec

// MultVector32 multiplies vector elements by scalar
func MultVector32(a []float32, val float32) {
	for i, av := range a {
		if math32.IsNaN(av) {
			continue
		}
		a[i] *= val
	}
}

// MultVec64 multiplies vector elements by scalar
func MultVec64(a []float64, val float64) {
	for i, av := range a {
		if math.IsNaN(av) {
			continue
		}
		a[i] *= val
	}
}

///////////////////////////////////////////
//  AddVec

// AddVector32 adds scalar to vector
func AddVector32(a []float32, val float32) {
	for i, av := range a {
		if math32.IsNaN(av) {
			continue
		}
		a[i] += val
	}
}

// AddVec64 adds scalar to vector
func AddVec64(a []float64, val float64) {
	for i, av := range a {
		if math.IsNaN(av) {
			continue
		}
		a[i] += val
	}
}

///////////////////////////////////////////
//  Thresh

// Thresh32 thresholds the values of the vector -- anything above the high threshold is set
// to the high value, and everything below the low threshold is set to the low value.
func Thresh32(a []float32, hi bool, hiThr float32, lo bool, loThr float32) {
	for i, av := range a {
		if math32.IsNaN(av) {
			continue
		}
		if hi && av > hiThr {
			a[i] = hiThr
		}
		if lo && av < loThr {
			a[i] = loThr
		}
	}
}

// Thresh64 thresholds the values of the vector -- anything above the high threshold is set
// to the high value, and everything below the low threshold is set to the low value.
func Thresh64(a []float64, hi bool, hiThr float64, lo bool, loThr float64) {
	for i, av := range a {
		if math.IsNaN(av) {
			continue
		}
		if hi && av > hiThr {
			a[i] = hiThr
		}
		if lo && av < loThr {
			a[i] = loThr
		}
	}
}

///////////////////////////////////////////
//  Binarize

// Binarize32 turns vector into binary-valued, by setting anything >= the threshold
// to the high value, and everything below to the low value.
func Binarize32(a []float32, thr, hiVal, loVal float32) {
	for i, av := range a {
		if math32.IsNaN(av) {
			continue
		}
		if av >= thr {
			a[i] = hiVal
		} else {
			a[i] = loVal
		}
	}
}

// Binarize64 turns vector into binary-valued, by setting anything >= the threshold
// to the high value, and everything below to the low value.
func Binarize64(a []float64, thr, hiVal, loVal float64) {
	for i, av := range a {
		if math.IsNaN(av) {
			continue
		}
		if av >= thr {
			a[i] = hiVal
		} else {
			a[i] = loVal
		}
	}
}

// Func32 is a norm function operating on slice of float32 numbers
type Func32 func(a []float32) float32

// Func64 is a norm function operating on slices of float64 numbers
type Func64 func(a []float64) float64
