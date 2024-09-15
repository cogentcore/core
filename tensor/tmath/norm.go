// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package tmath

import (
	"cogentcore.org/core/math32"
	"cogentcore.org/core/tensor"
	"cogentcore.org/core/tensor/stats/stats"
)

// ZScore computes Z-normalized values into given output tensor,
// subtracting the Mean and dividing by the standard deviation.
func ZScore(a, out *tensor.Indexed) {
	mout := tensor.NewIndexed(tensor.NewFloat64())
	std, mean, _ := stats.StdFuncOut64(a, mout)
	Sub(a, mean, out)
	Div(out, std, out)
}

// UnitNorm computes unit normalized values into given output tensor,
// subtracting the Min value and dividing by the Max of the remaining numbers.
func UnitNorm(a, out *tensor.Indexed) {
	mout := tensor.NewIndexed(tensor.NewFloat64())
	stats.MinFunc(a, mout)
	Sub(a, mout, out)
	stats.MaxFunc(out, mout)
	Div(out, mout, out)
}

// Clamp ensures that all values are within min, max limits, clamping
// values to those bounds if they exceed them.  min and max args are
// treated as scalars (first value used).
func Clamp(in, minv, maxv, out *tensor.Indexed) {
	out.SetShapeFrom(in)
	mn := minv.Float1D(0)
	mx := maxv.Float1D(0)
	tensor.VectorizeThreaded(1, tensor.NFirstLen, func(idx int, tsr ...*tensor.Indexed) {
		i, _, _ := tsr[0].RowCellIndex(idx)
		tsr[1].SetFloat1D(math32.Clamp64(tsr[0].Float1D(i), mn, mx), i)
	}, in, out)
}

// Binarize results in a binary-valued output by setting
// values >= the threshold to 1, else 0.  threshold is
// treated as a scalar (first value used).
func Binarize(in, threshold, out *tensor.Indexed) {
	out.SetShapeFrom(in)
	thr := threshold.Float1D(0)
	tensor.VectorizeThreaded(1, tensor.NFirstLen, func(idx int, tsr ...*tensor.Indexed) {
		i, _, _ := tsr[0].RowCellIndex(idx)
		v := tsr[0].Float1D(i)
		if v >= thr {
			v = 1
		} else {
			v = 0
		}
		tsr[1].SetFloat1D(v, i)
	}, in, out)
}