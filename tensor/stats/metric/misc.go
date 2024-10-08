// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package metric

import (
	"math"

	"cogentcore.org/core/tensor"
)

// ClosestRow returns the closest fit between probe pattern and patterns in
// a "vocabulary" tensor with outermost row dimension, using given metric
// function, which must fit the MetricFunc signature.
// The metric *must have the Increasing property*, i.e., larger = further.
// Output is a 1D tensor with 2 elements: the row index and metric value for that row.
// Note: this does _not_ use any existing Indexes for the probe,
// but does for the vocab, and the returned index is the logical index
// into any existing Indexes.
func ClosestRow(fun any, probe, vocab tensor.Tensor) tensor.Values {
	return tensor.CallOut2Gen1(ClosestRowOut, fun, probe, vocab)
}

// ClosestRowOut returns the closest fit between probe pattern and patterns in
// a "vocabulary" tensor with outermost row dimension, using given metric
// function, which must fit the MetricFunc signature.
// The metric *must have the Increasing property*, i.e., larger = further.
// Output is a 1D tensor with 2 elements: the row index and metric value for that row.
// Note: this does _not_ use any existing Indexes for the probe,
// but does for the vocab, and the returned index is the logical index
// into any existing Indexes.
func ClosestRowOut(fun any, probe, vocab tensor.Tensor, out tensor.Values) error {
	out.SetShapeSizes(2)
	mfun, err := AsMetricFunc(fun)
	if err != nil {
		return err
	}
	rows, _ := vocab.Shape().RowCellSize()
	mi := -1
	mind := math.MaxFloat64
	pr1d := tensor.As1D(probe)
	for ri := range rows {
		sub := tensor.Cells1D(vocab, ri)
		mout := mfun(pr1d, sub)
		d := mout.Float1D(0)
		if d < mind {
			mi = ri
			mind = d
		}
	}
	out.SetFloat1D(float64(mi), 0)
	out.SetFloat1D(mind, 1)
	return nil
}
