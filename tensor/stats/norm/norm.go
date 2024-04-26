// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package norm

//go:generate core generate

import (
	"math"
	"reflect"

	"cogentcore.org/core/math32"
	"cogentcore.org/core/tensor"
)

func FloatFunc(tsr tensor.Tensor, nfunc32 Func32, nfunc64 Func64, stIdx, nIdx int, ffunc32 func(a []float32, fun Func32), ffunc64 func(a []float64, fun Func64)) {
	switch tsr.DataType() {
	case reflect.Float32:
		vals := tsr.(*tensor.Number[float32]).Values
		if nIdx > 0 {
			vals = vals[stIdx : stIdx+nIdx]
		}
		ffunc32(vals, nfunc32)
	case reflect.Float64:
		vals := tsr.(*tensor.Number[float64]).Values
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
	SubNorm32(a, Mean32)
	DivNorm32(a, Std32)
}

// ZScore64 subtracts the mean and divides by the standard deviation
func ZScore64(a []float64) {
	SubNorm64(a, Mean64)
	DivNorm64(a, Std64)
}

///////////////////////////////////////////
//  Unit

// Unit32 subtracts the min and divides by the max, so that values are in 0-1 unit range
func Unit32(a []float32) {
	SubNorm32(a, Min32)
	DivNorm32(a, Max32)
}

// Unit64 subtracts the min and divides by the max, so that values are in 0-1 unit range
func Unit64(a []float64) {
	SubNorm64(a, Min64)
	DivNorm64(a, Max64)
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

// StdNorms are standard norm functions, including stats
type StdNorms int32 //enums:enum

const (
	L1 StdNorms = iota
	L2
	SumSquares
	N
	Sum
	Mean
	Var
	Std
	Max
	MaxAbs
	Min
	MinAbs
)

// StdFunc32 returns a standard norm function as specified
func StdFunc32(std StdNorms) Func32 {
	switch std {
	case L1:
		return L132
	case L2:
		return L232
	case SumSquares:
		return SumSquares32
	case N:
		return N32
	case Sum:
		return Sum32
	case Mean:
		return Mean32
	case Var:
		return Var32
	case Std:
		return Std32
	case Max:
		return Max32
	case MaxAbs:
		return MaxAbs32
	case Min:
		return Min32
	case MinAbs:
		return MinAbs32
	}
	return nil
}

// StdFunc64 returns a standard norm function as specified
func StdFunc64(std StdNorms) Func64 {
	switch std {
	case L1:
		return L164
	case L2:
		return L264
	case SumSquares:
		return SumSquares64
	case N:
		return N64
	case Sum:
		return Sum64
	case Mean:
		return Mean64
	case Var:
		return Var64
	case Std:
		return Std64
	case Max:
		return Max64
	case MaxAbs:
		return MaxAbs64
	case Min:
		return Min64
	case MinAbs:
		return MinAbs64
	}
	return nil
}
