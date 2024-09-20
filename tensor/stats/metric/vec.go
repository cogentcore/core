// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package metric

import (
	"math"
	"slices"

	"cogentcore.org/core/tensor"
)

// Vectorize3Out64 is a version of the [tensor.Vectorize] function
// for metrics, which makes three Float64 output tensors for aggregating
// and computing values, returning them for final computation.
func Vectorize3Out64(nfunc func(tsr ...*tensor.Indexed) int, fun func(idx int, tsr ...*tensor.Indexed), tsr ...*tensor.Indexed) (out1, out2, out3 *tensor.Indexed) {
	n := nfunc(tsr...)
	if n <= 0 {
		return nil, nil, nil
	}
	nt := len(tsr)
	out := tsr[nt-1]
	osz := out.Tensor.ShapeInts()
	out1 = tensor.NewFloat64Indexed(osz...)
	out2 = tensor.NewFloat64Indexed(osz...)
	out3 = tensor.NewFloat64Indexed(osz...)
	tsrs := slices.Clone(tsr[:nt-1])
	tsrs = append(tsrs, out1, out2, out3)
	for idx := range n {
		fun(idx, tsrs...)
	}
	return out1, out2, out3
}

// OutShape returns the output shape based on given input
// shape ints, with outer row dim = 1.
func OutShape(ish ...int) []int {
	ish[0] = 1
	return ish
}

// NFunc is the nfun for metrics functions, returning the min number of rows across the
// two input tensors, and initializing the _last_ one to hold the output
// with the first, row dimension set to 1.
func NFunc(tsr ...*tensor.Indexed) int {
	nt := len(tsr)
	if nt < 3 {
		return 0
	}
	a, b, out := tsr[0], tsr[1], tsr[nt-1]
	na, nb := a.NumRows(), b.NumRows()
	out.Tensor.SetShape(OutShape(a.Tensor.ShapeInts()...))
	out.Indexes = nil
	return min(na, nb)
}

// VecFunc is a helper function for metrics functions, dealing with iterating over
// the Cell subspace per row and initializing the aggregation values for first index.
// It also skips over NaN missing values.
func VecFunc(idx int, a, b, out *tensor.Indexed, ini float64, fun func(a, b, agg float64) float64) {
	nsub := out.Tensor.Len()
	for i := range nsub {
		if idx == 0 {
			out.SetFloat1D(ini, i)
		}
		av := a.FloatRowCell(idx, i)
		if math.IsNaN(av) {
			continue
		}
		bv := b.FloatRowCell(idx, i)
		if math.IsNaN(bv) {
			continue
		}
		out.SetFloat1D(fun(av, bv, out.Float1D(i)), i)
	}
}

// VecSSFunc is a helper function for metric functions, dealing with iterating over
// the Cell subspace per row and initializing the aggregation values for first index.
// This version does sum-of-squares integration over 2 output vectors,
// It also skips over NaN missing values.
func VecSSFunc(idx int, a, b, out1, out2 *tensor.Indexed, ini1, ini2 float64, fun func(a, b float64) float64) {
	nsub := out2.Tensor.Len()
	for i := range nsub {
		if idx == 0 {
			out1.SetFloat1D(ini1, i)
			out2.SetFloat1D(ini2, i)
		}
		av := a.FloatRowCell(idx, i)
		if math.IsNaN(av) {
			continue
		}
		bv := b.FloatRowCell(idx, i)
		if math.IsNaN(bv) {
			continue
		}
		scale, ss := out1.Float1D(i), out2.Float1D(i)
		d := fun(av, bv)
		if d == 0 {
			continue
		}
		absxi := math.Abs(d)
		if scale < absxi {
			ss = 1 + ss*(scale/absxi)*(scale/absxi)
			scale = absxi
		} else {
			ss = ss + (absxi/scale)*(absxi/scale)
		}
		out1.SetFloat1D(scale, i)
		out2.SetFloat1D(ss, i)
	}
}

// Vec2inFunc is a helper function for stats functions, dealing with iterating over
// the Cell subspace per row and initializing the aggregation values for first index.
// This version has 2 input vectors, the second input being the output of another stat
// e.g., the mean. It also skips over NaN missing values.
func Vec2inFunc(idx int, a, b, a2, b2, out *tensor.Indexed, ini float64, fun func(a, b, a2, b2, agg float64) float64) {
	nsub := out.Tensor.Len()
	for i := range nsub {
		if idx == 0 {
			out.SetFloat1D(ini, i)
		}
		av := a.FloatRowCell(idx, i)
		if math.IsNaN(av) {
			continue
		}
		bv := b.FloatRowCell(idx, i)
		if math.IsNaN(bv) {
			continue
		}
		av2 := a2.Float1D(i)
		bv2 := b2.Float1D(i)
		out.SetFloat1D(fun(av, bv, av2, bv2, out.Float1D(i)), i)
	}
}

// Vec2in3outFunc is a helper function for stats functions, dealing with iterating over
// the Cell subspace per row and initializing the aggregation values for first index.
// This version has 2 input, 3 output vectors. The second input being the output of another stat
// e.g., the mean. It also skips over NaN missing values.
func Vec2in3outFunc(idx int, a, b, a2, b2, out1, out2, out3 *tensor.Indexed, ini float64, fun func(a, b, a2, b2, out1, out2, out3 float64) (float64, float64, float64)) {
	nsub := out1.Tensor.Len()
	for i := range nsub {
		if idx == 0 {
			out1.SetFloat1D(ini, i)
			out2.SetFloat1D(ini, i)
			out3.SetFloat1D(ini, i)
		}
		av := a.FloatRowCell(idx, i)
		if math.IsNaN(av) {
			continue
		}
		bv := b.FloatRowCell(idx, i)
		if math.IsNaN(bv) {
			continue
		}
		av2 := a2.Float1D(i)
		bv2 := b2.Float1D(i)
		o1 := out1.Float1D(i)
		o2 := out2.Float1D(i)
		o3 := out3.Float1D(i)
		o1, o2, o3 = fun(av, bv, av2, bv2, o1, o2, o3)
		out1.SetFloat1D(o1, i)
		out2.SetFloat1D(o2, i)
		out3.SetFloat1D(o3, i)
	}
}

// Vec3outFunc is a helper function for stats functions, dealing with iterating over
// the Cell subspace per row and initializing the aggregation values for first index.
// This version has 3 output vectors. It also skips over NaN missing values.
func Vec3outFunc(idx int, a, b, out1, out2, out3 *tensor.Indexed, ini float64, fun func(a, b, out1, out2, out3 float64) (float64, float64, float64)) {
	nsub := out1.Tensor.Len()
	for i := range nsub {
		if idx == 0 {
			out1.SetFloat1D(ini, i)
			out2.SetFloat1D(ini, i)
			out3.SetFloat1D(ini, i)
		}
		av := a.FloatRowCell(idx, i)
		if math.IsNaN(av) {
			continue
		}
		bv := b.FloatRowCell(idx, i)
		if math.IsNaN(bv) {
			continue
		}
		o1 := out1.Float1D(i)
		o2 := out2.Float1D(i)
		o3 := out3.Float1D(i)
		o1, o2, o3 = fun(av, bv, o1, o2, o3)
		out1.SetFloat1D(o1, i)
		out2.SetFloat1D(o2, i)
		out3.SetFloat1D(o3, i)
	}
}
