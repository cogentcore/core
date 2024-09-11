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

////////////////////////////////////////////////////////////
// 	Sub

// Sub subtracts two tensors into output.
// If one is a scalar, it is subtracted from all elements.
// If one is the same size as the Cell SubSpace of the other
// then the SubSpace is subtracted from each row of the other.
// Otherwise an element-wise subtractition is performed
// for overlapping cells.
func Sub(a, b, out *tensor.Indexed) {
	if b.Len() == 1 {
		SubScalar(1, b.FloatRowCell(0, 0), a, out)
		return
	}
	if a.Len() == 1 {
		SubScalar(-1, a.FloatRowCell(0, 0), b, out)
		return
	}
	arows, acells := a.Tensor.RowCellSize()
	brows, bcells := b.Tensor.RowCellSize()
	if brows*bcells == acells {
		SubSubSpace(1, a, b, out)
		return
	}
	if arows*acells == bcells {
		SubSubSpace(-1, b, a, out)
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
			out.Tensor.SetFloat1D(ia, tsr[0].Tensor.Float1D(ia)-tsr[1].Tensor.Float1D(ib))
		}, a, b, out)
}

// SubScalar subtracts a scalar from given tensor into output.
// sign determines which way the subtraction goes: 1 = a-scalar, -1 = scalar-a
func SubScalar(sign float64, scalar float64, a, out *tensor.Indexed) {
	out.SetShapeFrom(a)
	tensor.VectorizeThreaded(1, tensor.NFirstLen, func(idx int, tsr ...*tensor.Indexed) {
		ia, _, _ := a.RowCellIndex(idx)
		out.Tensor.SetFloat1D(ia, sign*(a.Tensor.Float1D(ia)-scalar))
	}, a, out)
}

// SubSubSpace subtracts the subspace tensor to each row in the given tensor,
// into the output tensor.
// sign determines which way the subtraction goes: 1 = a-sub, -1 = sub-a
func SubSubSpace(sign float64, a, sub, out *tensor.Indexed) {
	out.SetShapeFrom(a)
	tensor.VectorizeThreaded(1, tensor.NFirstLen, func(idx int, tsr ...*tensor.Indexed) {
		ai, _, ci := a.RowCellIndex(idx)
		si, _, _ := sub.RowCellIndex(ci)
		out.Tensor.SetFloat1D(ai, sign*(a.Tensor.Float1D(ai)-sub.Tensor.Float1D(si)))
	}, a, sub, out)
}

////////////////////////////////////////////////////////////
// 	Mul

// Mul does element-wise multiplication between two tensors into output.
// If one is a scalar, it multiplies all elements.
// If one is the same size as the Cell SubSpace of the other
// then the SubSpace multiplies each row of the other.
// Otherwise an element-wise multiplication is performed
// for overlapping cells.
func Mul(a, b, out *tensor.Indexed) {
	if b.Len() == 1 {
		MulScalar(b.FloatRowCell(0, 0), a, out)
		return
	}
	if a.Len() == 1 {
		MulScalar(a.FloatRowCell(0, 0), b, out)
		return
	}
	arows, acells := a.Tensor.RowCellSize()
	brows, bcells := b.Tensor.RowCellSize()
	if brows*bcells == acells {
		MulSubSpace(a, b, out)
		return
	}
	if arows*acells == bcells {
		MulSubSpace(b, a, out)
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
			out.Tensor.SetFloat1D(ia, tsr[0].Tensor.Float1D(ia)*tsr[1].Tensor.Float1D(ib))
		}, a, b, out)
}

// MulScalar multiplies a scalar to given tensor into output.
func MulScalar(scalar float64, a, out *tensor.Indexed) {
	out.SetShapeFrom(a)
	tensor.VectorizeThreaded(1, tensor.NFirstLen, func(idx int, tsr ...*tensor.Indexed) {
		ia, _, _ := a.RowCellIndex(idx)
		out.Tensor.SetFloat1D(ia, a.Tensor.Float1D(ia)*scalar)
	}, a, out)
}

// MulSubSpace multiplies the subspace tensor to each row in the given tensor,
// into the output tensor.
func MulSubSpace(a, sub, out *tensor.Indexed) {
	out.SetShapeFrom(a)
	tensor.VectorizeThreaded(1, tensor.NFirstLen, func(idx int, tsr ...*tensor.Indexed) {
		ai, _, ci := a.RowCellIndex(idx)
		si, _, _ := sub.RowCellIndex(ci)
		out.Tensor.SetFloat1D(ai, a.Tensor.Float1D(ai)*sub.Tensor.Float1D(si))
	}, a, sub, out)
}

////////////////////////////////////////////////////////////
// 	Div

// Div does element-wise division between two tensors into output.
// If one is a scalar, it divides all elements.
// If one is the same size as the Cell SubSpace of the other
// then the SubSpace divides each row of the other.
// Otherwise an element-wise division is performed
// for overlapping cells.
func Div(a, b, out *tensor.Indexed) {
	if b.Len() == 1 {
		DivScalar(b.FloatRowCell(0, 0), a, out)
		return
	}
	if a.Len() == 1 {
		DivScalarInv(a.FloatRowCell(0, 0), b, out)
		return
	}
	arows, acells := a.Tensor.RowCellSize()
	brows, bcells := b.Tensor.RowCellSize()
	if brows*bcells == acells {
		DivSubSpace(a, b, out)
		return
	}
	if arows*acells == bcells {
		DivSubSpaceInv(b, a, out)
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
			out.Tensor.SetFloat1D(ia, tsr[0].Tensor.Float1D(ia)/tsr[1].Tensor.Float1D(ib))
		}, a, b, out)
}

// DivScalar divides given tensor elements by scalar into output.
func DivScalar(scalar float64, a, out *tensor.Indexed) {
	out.SetShapeFrom(a)
	tensor.VectorizeThreaded(1, tensor.NFirstLen, func(idx int, tsr ...*tensor.Indexed) {
		ia, _, _ := a.RowCellIndex(idx)
		out.Tensor.SetFloat1D(ia, a.Tensor.Float1D(ia)/scalar)
	}, a, out)
}

// DivScalarInv divides scalar by given tensor elements into output
// (inverse of [DivScalar]).
func DivScalarInv(scalar float64, a, out *tensor.Indexed) {
	out.SetShapeFrom(a)
	tensor.VectorizeThreaded(1, tensor.NFirstLen, func(idx int, tsr ...*tensor.Indexed) {
		ia, _, _ := a.RowCellIndex(idx)
		out.Tensor.SetFloat1D(ia, scalar/a.Tensor.Float1D(ia))
	}, a, out)
}

// DivSubSpace divides each row of the given tensor by the subspace tensor elements,
// into the output tensor.
func DivSubSpace(a, sub, out *tensor.Indexed) {
	out.SetShapeFrom(a)
	tensor.VectorizeThreaded(1, tensor.NFirstLen, func(idx int, tsr ...*tensor.Indexed) {
		ai, _, ci := a.RowCellIndex(idx)
		si, _, _ := sub.RowCellIndex(ci)
		out.Tensor.SetFloat1D(ai, a.Tensor.Float1D(ai)/sub.Tensor.Float1D(si))
	}, a, sub, out)
}

// DivSubSpaceInv divides the subspace tensor by each row of the given tensor,
// into the output tensor (inverse of [DivSubSpace])
func DivSubSpaceInv(a, sub, out *tensor.Indexed) {
	out.SetShapeFrom(a)
	tensor.VectorizeThreaded(1, tensor.NFirstLen, func(idx int, tsr ...*tensor.Indexed) {
		ai, _, ci := a.RowCellIndex(idx)
		si, _, _ := sub.RowCellIndex(ci)
		out.Tensor.SetFloat1D(ai, sub.Tensor.Float1D(si)/a.Tensor.Float1D(ai))
	}, a, sub, out)
}
