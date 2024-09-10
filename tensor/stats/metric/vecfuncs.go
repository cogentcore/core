// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package metric

import (
	"math"
	"slices"

	"cogentcore.org/core/tensor"
)

// VectorizeOut64 is a version of the [tensor.Vectorize] function
// for metrics, which makes a Float64 output tensor for aggregating
// and computing values, and then copies the results back to the
// original output.  This allows metrics functions to operate directly
// on integer valued inputs and produce sensible results.
// It automatically calls NFunc for the nfun function,
// and returns the Float64 output tensor for further processing as needed.
// It uses the _last_ tensor as the output, allowing for multiple inputs.
func VectorizeOut64(fun func(idx int, tsr ...*tensor.Indexed), tsr ...*tensor.Indexed) *tensor.Indexed {
	n := NFunc(tsr...)
	if n <= 0 {
		return nil
	}
	nt := len(tsr)
	out := tsr[nt-1]
	o64 := tensor.NewIndexed(tensor.NewFloat64(out.Tensor.Shape().Sizes))
	etsr := slices.Clone(tsr)
	etsr[nt-1] = o64
	for idx := range n {
		fun(idx, etsr...)
	}
	nsub := out.Tensor.Len()
	for i := range nsub {
		out.Tensor.SetFloat1D(i, o64.Tensor.Float1D(i))
	}
	return o64
}

// Vectorize2Out64 is a version of the [tensor.Vectorize] function
// for metrics, which makes two Float64 output tensors for aggregating
// and computing values, returning them for final computation.
// It automatically calls NFunc for the nfun function.
func Vectorize2Out64(fun func(idx int, tsr ...*tensor.Indexed), tsr ...*tensor.Indexed) (out1, out2 *tensor.Indexed) {
	n := NFunc(tsr...)
	if n <= 0 {
		return nil, nil
	}
	nt := len(tsr)
	out := tsr[nt-1]
	out1 = tensor.NewIndexed(tensor.NewFloat64(out.Tensor.Shape().Sizes))
	out2 = tensor.NewIndexed(tensor.NewFloat64(out.Tensor.Shape().Sizes))
	for idx := range n {
		fun(idx, tsr[0], tsr[1], out1, out2)
	}
	return out1, out2
}

// Vectorize3Out64 is a version of the [tensor.Vectorize] function
// for metrics, which makes three Float64 output tensors for aggregating
// and computing values, returning them for final computation.
// It automatically calls NFunc for the nfun function.
func Vectorize3Out64(fun func(idx int, tsr ...*tensor.Indexed), tsr ...*tensor.Indexed) (out1, out2, out3 *tensor.Indexed) {
	n := NFunc(tsr...)
	if n <= 0 {
		return nil, nil, nil
	}
	nt := len(tsr)
	out := tsr[nt-1]
	out1 = tensor.NewIndexed(tensor.NewFloat64(out.Tensor.Shape().Sizes))
	out2 = tensor.NewIndexed(tensor.NewFloat64(out.Tensor.Shape().Sizes))
	out3 = tensor.NewIndexed(tensor.NewFloat64(out.Tensor.Shape().Sizes))
	tsrs := tsr[:nt-1]
	tsrs = append(tsrs, out1, out2, out3)
	for idx := range n {
		fun(idx, tsrs...)
	}
	return out1, out2, out3
}

// OutShape returns the output shape based on given input
// tensor, with outer row dim = 1.
func OutShape(ish *tensor.Shape) *tensor.Shape {
	osh := &tensor.Shape{}
	osh.CopyShape(ish)
	osh.Sizes[0] = 1
	return osh
}

// NFunc is the nfun for metrics functions, returning the min length across the
// two input tensors, and initializing the _last_ one to hold the output
// with the first, row dimension set to 1.
func NFunc(tsr ...*tensor.Indexed) int {
	nt := len(tsr)
	if nt < 3 {
		return 0
	}
	a, b, out := tsr[0], tsr[1], tsr[nt-1]
	na, nb := a.Len(), b.Len()
	osh := OutShape(a.Tensor.Shape())
	out.Tensor.SetShape(osh.Sizes, osh.Names...)
	out.Indexes = []int{0}
	return min(na, nb)
}

// MetricVecFunc is a helper function for metrics functions, dealing with iterating over
// the Cell subspace per row and initializing the aggregation values for first index.
// It also skips over NaN missing values.
func MetricVecFunc(idx int, a, b, out *tensor.Indexed, ini float64, fun func(a, b, agg float64) float64) {
	nsub := out.Tensor.Len()
	for i := range nsub {
		if idx == 0 {
			out.Tensor.SetFloat1D(i, ini)
		}
		av := a.FloatRowCell(idx, i)
		if math.IsNaN(av) {
			continue
		}
		bv := b.FloatRowCell(idx, i)
		if math.IsNaN(bv) {
			continue
		}
		out.Tensor.SetFloat1D(i, fun(av, bv, out.Tensor.Float1D(i)))
	}
}

// MetricVecSSFunc is a helper function for metric functions, dealing with iterating over
// the Cell subspace per row and initializing the aggregation values for first index.
// This version does sum-of-squares integration over 2 output vectors,
// It also skips over NaN missing values.
func MetricVecSSFunc(idx int, a, b, out1, out2 *tensor.Indexed, ini1, ini2 float64, fun func(a, b float64) float64) {
	nsub := out2.Tensor.Len()
	for i := range nsub {
		if idx == 0 {
			out1.Tensor.SetFloat1D(i, ini1)
			out2.Tensor.SetFloat1D(i, ini2)
		}
		av := a.FloatRowCell(idx, i)
		if math.IsNaN(av) {
			continue
		}
		bv := b.FloatRowCell(idx, i)
		if math.IsNaN(bv) {
			continue
		}
		scale, ss := out1.Tensor.Float1D(i), out2.Tensor.Float1D(i)
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
		out1.Tensor.SetFloat1D(i, scale)
		out2.Tensor.SetFloat1D(i, ss)
	}
}

// MetricVec2inFunc is a helper function for stats functions, dealing with iterating over
// the Cell subspace per row and initializing the aggregation values for first index.
// This version has 2 input vectors, the second input being the output of another stat
// e.g., the mean. It also skips over NaN missing values.
func MetricVec2inFunc(idx int, a, b, a2, b2, out *tensor.Indexed, ini float64, fun func(a, b, a2, b2, agg float64) float64) {
	nsub := out.Tensor.Len()
	for i := range nsub {
		if idx == 0 {
			out.Tensor.SetFloat1D(i, ini)
		}
		av := a.FloatRowCell(idx, i)
		if math.IsNaN(av) {
			continue
		}
		bv := b.FloatRowCell(idx, i)
		if math.IsNaN(bv) {
			continue
		}
		av2 := a2.Tensor.Float1D(i)
		bv2 := b2.Tensor.Float1D(i)
		out.Tensor.SetFloat1D(i, fun(av, bv, av2, bv2, out.Tensor.Float1D(i)))
	}
}

// MetricVec2in3outFunc is a helper function for stats functions, dealing with iterating over
// the Cell subspace per row and initializing the aggregation values for first index.
// This version has 2 input, 3 output vectors. The second input being the output of another stat
// e.g., the mean. It also skips over NaN missing values.
func MetricVec2in3outFunc(idx int, a, b, a2, b2, out1, out2, out3 *tensor.Indexed, ini float64, fun func(a, b, a2, b2, out1, out2, out3 float64) (float64, float64, float64)) {
	nsub := out1.Tensor.Len()
	for i := range nsub {
		if idx == 0 {
			out1.Tensor.SetFloat1D(i, ini)
			out2.Tensor.SetFloat1D(i, ini)
			out3.Tensor.SetFloat1D(i, ini)
		}
		av := a.FloatRowCell(idx, i)
		if math.IsNaN(av) {
			continue
		}
		bv := b.FloatRowCell(idx, i)
		if math.IsNaN(bv) {
			continue
		}
		av2 := a2.Tensor.Float1D(i)
		bv2 := b2.Tensor.Float1D(i)
		o1 := out1.Tensor.Float1D(i)
		o2 := out2.Tensor.Float1D(i)
		o3 := out3.Tensor.Float1D(i)
		o1, o2, o3 = fun(av, bv, av2, bv2, o1, o2, o3)
		out1.Tensor.SetFloat1D(i, o1)
		out2.Tensor.SetFloat1D(i, o2)
		out3.Tensor.SetFloat1D(i, o3)
	}
}

// MetricVec3outFunc is a helper function for stats functions, dealing with iterating over
// the Cell subspace per row and initializing the aggregation values for first index.
// This version has 3 output vectors. It also skips over NaN missing values.
func MetricVec3outFunc(idx int, a, b, out1, out2, out3 *tensor.Indexed, ini float64, fun func(a, b, out1, out2, out3 float64) (float64, float64, float64)) {
	nsub := out1.Tensor.Len()
	for i := range nsub {
		if idx == 0 {
			out1.Tensor.SetFloat1D(i, ini)
			out2.Tensor.SetFloat1D(i, ini)
			out3.Tensor.SetFloat1D(i, ini)
		}
		av := a.FloatRowCell(idx, i)
		if math.IsNaN(av) {
			continue
		}
		bv := b.FloatRowCell(idx, i)
		if math.IsNaN(bv) {
			continue
		}
		o1 := out1.Tensor.Float1D(i)
		o2 := out2.Tensor.Float1D(i)
		o3 := out3.Tensor.Float1D(i)
		o1, o2, o3 = fun(av, bv, o1, o2, o3)
		out1.Tensor.SetFloat1D(i, o1)
		out2.Tensor.SetFloat1D(i, o2)
		out3.Tensor.SetFloat1D(i, o3)
	}
}
