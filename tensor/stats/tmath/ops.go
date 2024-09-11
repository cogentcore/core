// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package tmath

import (
	"cogentcore.org/core/tensor"
)

// Add adds two tensors into output.
// If one is a scalar, it is added to all elements.
// If one is the same size as the Cell SubSpace of the other
// then the SubSpace is added to each row of the other.
// Otherwise an element-wise addition is performed
// for overlapping cells.
func Add(a, b, out *tensor.Indexed) {
	if b.Len() == 1 {
		AddScalar(b.FloatRowCell(0, 0), a, out)
		return
	}
	if a.Len() == 1 {
		AddScalar(a.FloatRowCell(0, 0), b, out)
		return
	}
	arows, acells := a.Tensor.RowCellSize()
	brows, bcells := b.Tensor.RowCellSize()
	if brows*bcells == acells {
		AddSubSpace(a, b, out)
		return
	}
	if arows*acells == bcells {
		AddSubSpace(b, a, out)
		return
	}
	// just do element-wise
	out.SetShapeFrom(a)
	tensor.VectorizeThreaded(1, func(tsr ...*tensor.Indexed) int {
		return tensor.NMinLen(2, tsr...)
	},
		func(idx int, tsr ...*tensor.Indexed) {
			ia, _, _ := tsr[0].RowCellIndex(idx)
			ib, _, _ := tsr[1].RowCellIndex(idx)
			out.Tensor.SetFloat1D(ia, tsr[0].Tensor.Float1D(ia)+tsr[1].Tensor.Float1D(ib))
		}, a, b, out)
}

// AddScalar adds a scalar to given tensor into output.
func AddScalar(scalar float64, a, out *tensor.Indexed) {
	out.SetShapeFrom(a)
	tensor.VectorizeThreaded(1, tensor.NFirstLen, func(idx int, tsr ...*tensor.Indexed) {
		ia, _, _ := a.RowCellIndex(idx)
		out.Tensor.SetFloat1D(ia, a.Tensor.Float1D(ia)+scalar)
	}, a, out)
}

// AddSubSpace adds the subspace tensor to each row in the given tensor,
// into the output tensor.
func AddSubSpace(a, sub, out *tensor.Indexed) {
	out.SetShapeFrom(a)
	tensor.VectorizeThreaded(1, tensor.NFirstLen, func(idx int, tsr ...*tensor.Indexed) {
		ai, _, ci := a.RowCellIndex(idx)
		si, _, _ := sub.RowCellIndex(ci)
		out.Tensor.SetFloat1D(ai, a.Tensor.Float1D(ai)+sub.Tensor.Float1D(si))
	}, a, sub, out)
}
