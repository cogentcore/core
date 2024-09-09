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
// It automatically calls NFunc for the nfun function,
// and returns the Float64 output tensor for further processing as needed.
// It uses the _last_ tensor as the output, allowing for multiple inputs,
// as in the case of VarVecFun.
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
// for stats, which makes two Float64 output tensors for aggregating
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
		fun(idx, tsr[0], out1, out2)
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

// NFunc is the nfun for stats functions, returning the length of the
// first tensor, and initializing the _last_ one to hold the output
// with the first, row dimension set to 1.
func NFunc(tsr ...*tensor.Indexed) int {
	nt := len(tsr)
	if nt < 2 {
		return 0
	}
	in, out := tsr[0], tsr[nt-1]
	n := in.Len()
	osh := OutShape(in.Tensor.Shape())
	out.Tensor.SetShape(osh.Sizes, osh.Names...)
	out.Indexes = []int{0}
	return n
}

// StatVecFunc is a helper function for stats functions, dealing with iterating over
// the Cell subspace per row and initializing the aggregation values for first index.
// It also skips over NaN missing values.
func StatVecFunc(idx int, in, out *tensor.Indexed, ini float64, fun func(val, agg float64) float64) {
	nsub := out.Tensor.Len()
	for i := range nsub {
		if idx == 0 {
			out.Tensor.SetFloat1D(i, ini)
		}
		val := in.FloatRowCell(idx, i)
		if math.IsNaN(val) {
			continue
		}
		out.Tensor.SetFloat1D(i, fun(val, out.Tensor.Float1D(i)))
	}
}

// SumVecFunc is a Vectorize function for computing the sum.
func SumVecFunc(idx int, tsr ...*tensor.Indexed) {
	StatVecFunc(idx, tsr[0], tsr[1], 0, func(val, agg float64) float64 {
		return agg + val
	})
}

// SumAbsVecFunc is a Vectorize function for computing the sum of abs values.
// This is also known as the L1 norm.
func SumAbsVecFunc(idx int, tsr ...*tensor.Indexed) {
	StatVecFunc(idx, tsr[0], tsr[1], 0, func(val, agg float64) float64 {
		return agg + math.Abs(val)
	})
}

// CountVecFunc is a Vectorize function for computing the count.
func CountVecFunc(idx int, tsr ...*tensor.Indexed) {
	StatVecFunc(idx, tsr[0], tsr[1], 0, func(val, agg float64) float64 {
		return agg + 1
	})
}

// ProdVecFunc is a Vectorize function for computing the product.
func ProdVecFunc(idx int, tsr ...*tensor.Indexed) {
	StatVecFunc(idx, tsr[0], tsr[1], 1, func(val, agg float64) float64 {
		return agg * val
	})
}

// MinVecFunc is a Vectorize function for computing the min.
func MinVecFunc(idx int, tsr ...*tensor.Indexed) {
	StatVecFunc(idx, tsr[0], tsr[1], math.MaxFloat64, func(val, agg float64) float64 {
		return math.Min(agg, val)
	})
}

// MaxVecFunc is a Vectorize function for computing the max.
func MaxVecFunc(idx int, tsr ...*tensor.Indexed) {
	StatVecFunc(idx, tsr[0], tsr[1], -math.MaxFloat64, func(val, agg float64) float64 {
		return math.Max(agg, val)
	})
}

// MinAbsVecFunc is a Vectorize function for computing the min of abs.
func MinAbsVecFunc(idx int, tsr ...*tensor.Indexed) {
	StatVecFunc(idx, tsr[0], tsr[1], math.MaxFloat64, func(val, agg float64) float64 {
		return math.Min(agg, math.Abs(val))
	})
}

// MaxAbsVecFunc is a Vectorize function for computing the max of abs.
func MaxAbsVecFunc(idx int, tsr ...*tensor.Indexed) {
	StatVecFunc(idx, tsr[0], tsr[1], -math.MaxFloat64, func(val, agg float64) float64 {
		return math.Max(agg, math.Abs(val))
	})
}

/////////////////////////////////////////////////////\
//		Two input Tensors

// StatVec2inFunc is a helper function for stats functions, dealing with iterating over
// the Cell subspace per row and initializing the aggregation values for first index.
// This version has 2 input vectors, the second input being the output of another stat
// e.g., the mean for Var.
// It also skips over NaN missing values.
func StatVec2inFunc(idx int, in1, in2, out *tensor.Indexed, ini float64, fun func(val1, val2, agg float64) float64) {
	nsub := out.Tensor.Len()
	for i := range nsub {
		if idx == 0 {
			out.Tensor.SetFloat1D(i, ini)
		}
		val1 := in1.FloatRowCell(idx, i)
		if math.IsNaN(val1) {
			continue
		}
		val2 := in2.Tensor.Float1D(i)
		out.Tensor.SetFloat1D(i, fun(val1, val2, out.Tensor.Float1D(i)))
	}
}

// VarVecFunc is a Vectorize function for computing the variance,
// using 3 tensors: in, mean, out.
func VarVecFunc(idx int, tsr ...*tensor.Indexed) {
	StatVec2inFunc(idx, tsr[0], tsr[1], tsr[2], 0, func(val1, val2, agg float64) float64 {
		dv := val1 - val2
		return agg + dv*dv
	})
}

/////////////////////////////////////////////////////\
//		Two output Tensors

// StatVec2outFunc is a helper function for stats functions, dealing with iterating over
// the Cell subspace per row and initializing the aggregation values for first index.
// This version has 2 output vectors, for separate integration of scale sum squared
// It also skips over NaN missing values.
func StatVec2outFunc(idx int, in, out1, out2 *tensor.Indexed, ini1, ini2 float64, fun func(val, agg1, agg2 float64) (float64, float64)) {
	nsub := out2.Tensor.Len()
	for i := range nsub {
		if idx == 0 {
			out1.Tensor.SetFloat1D(i, ini1)
			out2.Tensor.SetFloat1D(i, ini2)
		}
		val := in.FloatRowCell(idx, i)
		if math.IsNaN(val) {
			continue
		}
		ag1, ag2 := out1.Tensor.Float1D(i), out2.Tensor.Float1D(i)
		ag1, ag2 = fun(val, ag1, ag2)
		out1.Tensor.SetFloat1D(i, ag1)
		out2.Tensor.SetFloat1D(i, ag2)
	}
}

// SumSqVecFunc is a Vectorize function for computing the sum of squares,
// using 2 separate aggregation tensors.
func SumSqVecFunc(idx int, tsr ...*tensor.Indexed) {
	StatVec2outFunc(idx, tsr[0], tsr[1], tsr[2], 0, 1, func(val, scale, ss float64) (float64, float64) {
		if val == 0 {
			return scale, ss
		}
		absxi := math.Abs(val)
		if scale < absxi {
			ss = 1 + ss*(scale/absxi)*(scale/absxi)
			scale = absxi
		} else {
			ss = ss + (absxi/scale)*(absxi/scale)
		}
		return scale, ss
	})
}
