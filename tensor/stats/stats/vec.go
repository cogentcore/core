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
func VectorizeOut64(nfunc func(tsr ...*tensor.Indexed) int, fun func(idx int, tsr ...*tensor.Indexed), tsr ...*tensor.Indexed) *tensor.Indexed {
	n := nfunc(tsr...)
	if n <= 0 {
		return nil
	}
	nt := len(tsr)
	out := tsr[nt-1]
	osz := out.Tensor.Shape().Sizes
	o64 := tensor.NewIndexed(tensor.NewFloat64(osz...))
	etsr := slices.Clone(tsr)
	etsr[nt-1] = o64
	for idx := range n {
		fun(idx, etsr...)
	}
	nsub := out.Tensor.Len()
	for i := range nsub {
		out.SetFloat1D(o64.Float1D(i), i)
	}
	return o64
}

// Vectorize2Out64 is a version of the [tensor.Vectorize] function
// for stats, which makes two Float64 output tensors for aggregating
// and computing values, returning them for final computation.
func Vectorize2Out64(nfunc func(tsr ...*tensor.Indexed) int, fun func(idx int, tsr ...*tensor.Indexed), tsr ...*tensor.Indexed) (out1, out2 *tensor.Indexed) {
	n := nfunc(tsr...)
	if n <= 0 {
		return nil, nil
	}
	nt := len(tsr)
	out := tsr[nt-1]
	osz := out.Tensor.Shape().Sizes
	out1 = tensor.NewIndexed(tensor.NewFloat64(osz...))
	out2 = tensor.NewIndexed(tensor.NewFloat64(osz...))
	tsrs := slices.Clone(tsr[:nt-1])
	tsrs = append(tsrs, out1, out2)
	for idx := range n {
		fun(idx, tsrs...)
	}
	return out1, out2
}

// OutShape returns the output shape based on given input
// tensor, with outer row dim = 1.
func OutShape(ish *tensor.Shape) *tensor.Shape {
	osh := &tensor.Shape{}
	osh.CopyShape(ish)
	osh.Sizes[0] = 1
	return osh
}

// NFunc is the nfun for stats functions, returning number of rows of the
// first tensor, and initializing the _last_ one to hold the output
// with the first, row dimension set to 1.
func NFunc(tsr ...*tensor.Indexed) int {
	nt := len(tsr)
	if nt < 2 {
		return 0
	}
	in, out := tsr[0], tsr[nt-1]
	n := in.Rows()
	osh := OutShape(in.Tensor.Shape())
	out.Tensor.SetShape(osh.Sizes...)
	out.Tensor.SetNames(osh.Names...)
	out.Indexes = nil
	return n
}

// VecFunc is a helper function for stats functions, dealing with iterating over
// the Cell subspace per row and initializing the aggregation values for first index.
// It also skips over NaN missing values.
func VecFunc(idx int, in, out *tensor.Indexed, ini float64, fun func(val, agg float64) float64) {
	nsub := out.Tensor.Len()
	for i := range nsub {
		if idx == 0 {
			out.SetFloat1D(ini, i)
		}
		val := in.FloatRowCell(idx, i)
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
func Vec2inFunc(idx int, in1, in2, out *tensor.Indexed, ini float64, fun func(val1, val2, agg float64) float64) {
	nsub := out.Tensor.Len()
	for i := range nsub {
		if idx == 0 {
			out.SetFloat1D(ini, i)
		}
		val1 := in1.FloatRowCell(idx, i)
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
func Vec2outFunc(idx int, in, out1, out2 *tensor.Indexed, ini1, ini2 float64, fun func(val, agg1, agg2 float64) (float64, float64)) {
	nsub := out2.Tensor.Len()
	for i := range nsub {
		if idx == 0 {
			out1.SetFloat1D(ini1, i)
			out2.SetFloat1D(ini2, i)
		}
		val := in.FloatRowCell(idx, i)
		if math.IsNaN(val) {
			continue
		}
		ag1, ag2 := out1.Float1D(i), out2.Float1D(i)
		ag1, ag2 = fun(val, ag1, ag2)
		out1.SetFloat1D(ag1, i)
		out2.SetFloat1D(ag2, i)
	}
}
