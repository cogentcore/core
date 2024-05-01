// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package norm

import (
	"cogentcore.org/core/tensor"
	"cogentcore.org/core/tensor/stats/stats"
)

///////////////////////////////////////////
//  DivNorm

// TensorDivNorm does divisive normalization by given norm function
// computed on the first ndim dims of the tensor, where 0 = all values,
// 1 = norm each of the sub-dimensions under the first outer-most dimension etc.
// ndim must be < NumDims() if not 0.
func TensorDivNorm(tsr tensor.Tensor, ndim int, nfunc32 Func32, nfunc64 Func64) {
	if ndim == 0 {
		FloatFunc(tsr, nfunc32, nfunc64, 0, 0, DivNorm32, DivNorm64)
	}
	if ndim >= tsr.NumDims() {
		panic("norm.TensorSubNorm32: number of dims must be < NumDims()")
	}
	sln := 1
	ln := tsr.Len()
	for i := 0; i < ndim; i++ {
		sln *= tsr.Shape().DimSize(i)
	}
	dln := ln / sln
	for sl := 0; sl < sln; sl++ {
		st := sl * dln
		FloatFunc(tsr, nfunc32, nfunc64, st, dln, DivNorm32, DivNorm64)
	}
}

///////////////////////////////////////////
//  SubNorm

// TensorSubNorm does subtractive normalization by given norm function
// computed on the first ndim dims of the tensor, where 0 = all values,
// 1 = norm each of the sub-dimensions under the first outer-most dimension etc.
// ndim must be < NumDims() if not 0 (panics).
func TensorSubNorm(tsr tensor.Tensor, ndim int, nfunc32 Func32, nfunc64 Func64) {
	if ndim == 0 {
		FloatFunc(tsr, nfunc32, nfunc64, 0, 0, SubNorm32, SubNorm64)
	}
	if ndim >= tsr.NumDims() {
		panic("norm.TensorSubNorm32: number of dims must be < NumDims()")
	}
	sln := 1
	ln := tsr.Len()
	for i := 0; i < ndim; i++ {
		sln *= tsr.Shape().DimSize(i)
	}
	dln := ln / sln
	for sl := 0; sl < sln; sl++ {
		st := sl * dln
		FloatFunc(tsr, nfunc32, nfunc64, st, dln, SubNorm32, SubNorm64)
	}
}

// TensorZScore subtracts the mean and divides by the standard deviation
// computed on the first ndim dims of the tensor, where 0 = all values,
// 1 = norm each of the sub-dimensions under the first outer-most dimension etc.
// ndim must be < NumDims() if not 0 (panics).
// must be a float32 or float64 tensor
func TensorZScore(tsr tensor.Tensor, ndim int) {
	TensorSubNorm(tsr, ndim, stats.Mean32, stats.Mean64)
	TensorDivNorm(tsr, ndim, stats.Std32, stats.Std64)
}

// TensorUnit subtracts the min and divides by the max, so that values are in 0-1 unit range
// computed on the first ndim dims of the tensor, where 0 = all values,
// 1 = norm each of the sub-dimensions under the first outer-most dimension etc.
// ndim must be < NumDims() if not 0 (panics).
// must be a float32 or float64 tensor
func TensorUnit(tsr tensor.Tensor, ndim int) {
	TensorSubNorm(tsr, ndim, stats.Min32, stats.Min64)
	TensorDivNorm(tsr, ndim, stats.Max32, stats.Max64)
}
