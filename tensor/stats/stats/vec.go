// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package stats

import (
	"math"
	"slices"

	"cogentcore.org/core/tensor"
)

// VectorizeOut64 is a version of the [tensor.Vectorize] function
// for stats, which makes a Float64 output tensor for aggregating
// and computing values, and then copies the results back to the
// original output.  This allows stats functions to operate directly
// on integer valued inputs and produce sensible results.
// and returns the Float64 output tensor for further processing as needed.
// It uses the _last_ tensor as the output, allowing for multiple inputs,
// as in the case of VarVecFun.
func VectorizeOut64(nfunc func(tsr ...tensor.Tensor) int, fun func(idx int, tsr ...tensor.Tensor), tsr ...tensor.Tensor) (tensor.Tensor, error) {
	n := nfunc(tsr...)
	if n <= 0 {
		return nil, nil
	}
	nt := len(tsr)
	osz := tensor.CellsSize(tsr[0].ShapeSizes())
	out := tsr[nt-1]
	if err := tensor.SetShapeSizesMustBeValues(out, osz...); err != nil {
		return nil, err
	}
	o64 := tensor.NewFloat64(osz...)
	etsr := slices.Clone(tsr)
	etsr[nt-1] = o64
	for idx := range n {
		fun(idx, etsr...)
	}
	nsub := out.Len()
	for i := range nsub {
		out.SetFloat1D(o64.Float1D(i), i)
	}
	return o64, nil
}

// Vectorize2Out64 is a version of the [tensor.Vectorize] function
// for stats, which makes two Float64 output tensors for aggregating
// and computing values, returning them for final computation.
func Vectorize2Out64(nfunc func(tsr ...tensor.Tensor) int, fun func(idx int, tsr ...tensor.Tensor), tsr ...tensor.Tensor) (out1, out2 tensor.Tensor, err error) {
	n := nfunc(tsr...)
	if n <= 0 {
		return nil, nil, nil
	}
	nt := len(tsr)
	osz := tensor.CellsSize(tsr[0].ShapeSizes())
	out := tsr[nt-1]
	if err = tensor.SetShapeSizesMustBeValues(out, osz...); err != nil {
		return
	}
	out1 = tensor.NewFloat64(osz...)
	out2 = tensor.NewFloat64(osz...)
	tsrs := slices.Clone(tsr[:nt-1])
	tsrs = append(tsrs, out1, out2)
	for idx := range n {
		fun(idx, tsrs...)
	}
	return out1, out2, err
}

// NFunc is the nfun for stats functions, returning number of rows of the
// first tensor
func NFunc(tsr ...tensor.Tensor) int {
	nt := len(tsr)
	if nt < 2 {
		return 0
	}
	return tsr[0].DimSize(0)
}

// VecFunc is a helper function for stats functions, dealing with iterating over
// the Cell subspace per row and initializing the aggregation values for first index.
// It also skips over NaN missing values.
func VecFunc(idx int, in, out tensor.Tensor, ini float64, fun func(val, agg float64) float64) {
	nsub := out.Len()
	si := idx * nsub // 1D start of sub
	for i := range nsub {
		if idx == 0 {
			out.SetFloat1D(ini, i)
		}
		val := in.Float1D(si + i)
		if math.IsNaN(val) {
			continue
		}
		out.SetFloat1D(fun(val, out.Float1D(i)), i)
	}
}

// Vec2inFunc is a helper function for stats functions, dealing with iterating over
// the Cell subspace per row and initializing the aggregation values for first index.
// This version has 2 input vectors, the second input being the output of another stat
// e.g., the mean for Var.
// It also skips over NaN missing values.
func Vec2inFunc(idx int, in1, in2, out tensor.Tensor, ini float64, fun func(val1, val2, agg float64) float64) {
	nsub := out.Len()
	si := idx * nsub // 1D start of sub
	for i := range nsub {
		if idx == 0 {
			out.SetFloat1D(ini, i)
		}
		val1 := in1.Float1D(si + i)
		if math.IsNaN(val1) {
			continue
		}
		val2 := in2.Float1D(i) // output = not nan
		out.SetFloat1D(fun(val1, val2, out.Float1D(i)), i)
	}
}

// Vec2outFunc is a helper function for stats functions, dealing with iterating over
// the Cell subspace per row and initializing the aggregation values for first index.
// This version has 2 output vectors, for separate integration of scale sum squared
// It also skips over NaN missing values.
func Vec2outFunc(idx int, in, out1, out2 tensor.Tensor, ini1, ini2 float64, fun func(val, agg1, agg2 float64) (float64, float64)) {
	nsub := out2.Len()
	si := idx * nsub // 1D start of sub
	for i := range nsub {
		if idx == 0 {
			out1.SetFloat1D(ini1, i)
			out2.SetFloat1D(ini2, i)
		}
		val := in.Float1D(si + i)
		if math.IsNaN(val) {
			continue
		}
		ag1, ag2 := out1.Float1D(i), out2.Float1D(i)
		ag1, ag2 = fun(val, ag1, ag2)
		out1.SetFloat1D(ag1, i)
		out2.SetFloat1D(ag2, i)
	}
}
