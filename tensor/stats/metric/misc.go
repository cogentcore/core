// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package metric

import (
	"math"

	"cogentcore.org/core/tensor"
)

func init() {
	tensor.AddFunc("ClosestRow", ClosestRow, 1, tensor.StringFirstArg)
}

// ClosestRow returns the closest fit between probe pattern and patterns in
// a "vocabulary" tensor with outermost row dimension, using given metric
// function registered in tensor Funcs (e.g., use String() method on Metrics enum).
// The metric *must have the Increasing property*, i.e., larger = further.
// Output is a 1D tensor with 2 elements: the row index and metric value for that row.
// Note: this does _not_ use any existing Indexes for the probe,
// but does for the vocab, and the returned index is the logical index
// into any existing Indexes.
func ClosestRow(funcName string, probe, vocab, out tensor.Tensor) {
	rows, _ := vocab.RowCellSize()
	mi := -1
	mout := tensor.NewFloat64Scalar(0.0)
	mind := math.MaxFloat64
	pr1d := tensor.New1DViewOf(probe)
	for ri := range rows {
		sub := tensor.Cells1D(vocab, ri)
		tensor.Call(funcName, pr1d, sub, mout)
		d := mout.Float1D(0)
		if d < mind {
			mi = ri
			mind = d
		}
	}
	out.SetShapeInts(2)
	out.SetFloat1D(float64(mi), 0)
	out.SetFloat1D(mind, 1)
}
