// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package stats

import (
	"cogentcore.org/core/math32"
	"cogentcore.org/core/tensor"
	"cogentcore.org/core/tensor/tmath"
)

// ZScore computes Z-normalized values into given output tensor,
// subtracting the Mean and dividing by the standard deviation.
func ZScore(a tensor.Tensor) tensor.Values {
	return tensor.CallOut1(ZScoreOut, a)
}

// ZScore computes Z-normalized values into given output tensor,
// subtracting the Mean and dividing by the standard deviation.
func ZScoreOut(a tensor.Tensor, out tensor.Values) error {
	mout := tensor.NewFloat64()
	std, mean, _, err := StdOut64(a, mout)
	if err != nil {
		return err
	}
	tmath.SubOut(a, mean, out)
	tmath.DivOut(out, std, out)
	return nil
}

// UnitNorm computes unit normalized values into given output tensor,
// subtracting the Min value and dividing by the Max of the remaining numbers.
func UnitNorm(a tensor.Tensor) tensor.Values {
	return tensor.CallOut1(UnitNormOut, a)
}

// UnitNormOut computes unit normalized values into given output tensor,
// subtracting the Min value and dividing by the Max of the remaining numbers.
func UnitNormOut(a tensor.Tensor, out tensor.Values) error {
	mout := tensor.NewFloat64()
	err := MinOut(a, mout)
	if err != nil {
		return err
	}
	tmath.SubOut(a, mout, out)
	MaxOut(out, mout)
	tmath.DivOut(out, mout, out)
	return nil
}

// Clamp ensures that all values are within min, max limits, clamping
// values to those bounds if they exceed them.  min and max args are
// treated as scalars (first value used).
func Clamp(in, minv, maxv tensor.Tensor) tensor.Values {
	return tensor.CallOut3(ClampOut, in, minv, minv)
}

// ClampOut ensures that all values are within min, max limits, clamping
// values to those bounds if they exceed them.  min and max args are
// treated as scalars (first value used).
func ClampOut(in, minv, maxv tensor.Tensor, out tensor.Values) error {
	tensor.SetShapeFrom(out, in)
	mn := minv.Float1D(0)
	mx := maxv.Float1D(0)
	tensor.VectorizeThreaded(1, tensor.NFirstLen, func(idx int, tsr ...tensor.Tensor) {
		tsr[1].SetFloat1D(math32.Clamp(tsr[0].Float1D(idx), mn, mx), idx)
	}, in, out)
	return nil
}

// Binarize results in a binary-valued output by setting
// values >= the threshold to 1, else 0.  threshold is
// treated as a scalar (first value used).
func Binarize(in, threshold tensor.Tensor) tensor.Values {
	return tensor.CallOut2(BinarizeOut, in, threshold)
}

// BinarizeOut results in a binary-valued output by setting
// values >= the threshold to 1, else 0.  threshold is
// treated as a scalar (first value used).
func BinarizeOut(in, threshold tensor.Tensor, out tensor.Values) error {
	tensor.SetShapeFrom(out, in)
	thr := threshold.Float1D(0)
	tensor.VectorizeThreaded(1, tensor.NFirstLen, func(idx int, tsr ...tensor.Tensor) {
		v := tsr[0].Float1D(idx)
		if v >= thr {
			v = 1
		} else {
			v = 0
		}
		tsr[1].SetFloat1D(v, idx)
	}, in, out)
	return nil
}
