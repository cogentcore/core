// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package tmath

import (
	"cogentcore.org/core/tensor"
	"cogentcore.org/core/tensor/stats/stats"
)

// ZScore computes Z-normalized values into given output tensor,
// subtracting the Mean and dividing by the standard deviation.
func ZScore(a, out *tensor.Indexed) {
	osh := stats.OutShape(a.Tensor.Shape())
	mout := tensor.NewIndexed(tensor.NewFloat64(osh.Sizes))
	std, mean, _ := stats.StdFuncOut64(a, mout)
	out.SetShapeFrom(a)
	Sub(a, mean, out)
	Div(out, std, out) // for use out as in too
}

// UnitNorm computes unit normalized values into given output tensor,
// subtracting the Min value and dividing by the Max of the remaining numbers.
func UnitNorm(a, out *tensor.Indexed) {
	osh := stats.OutShape(a.Tensor.Shape())
	mout := tensor.NewIndexed(tensor.NewFloat64(osh.Sizes))
	stats.MinFunc(a, mout)
	out.SetShapeFrom(a)
	Sub(a, mout, out)
	stats.MaxFunc(out, mout)
	Div(out, mout, out) // for use out as in too
}

/*

///////////////////////////////////////////
//  Thresh


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

*/
