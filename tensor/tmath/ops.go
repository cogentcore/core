// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package tmath

import (
	"cogentcore.org/core/tensor"
)

func init() {
	tensor.AddFunc("Add", Add, 1)
	tensor.AddFunc("Sub", Sub, 1)
	tensor.AddFunc("Mul", Mul, 1)
	tensor.AddFunc("Div", Div, 1)
}

// Add adds two tensors into output.
// If one is a scalar, it is added to all elements.
// If one is the same size as the Cell SubSpace of the other
// then the SubSpace is added to each row of the other.
// Otherwise an element-wise addition is performed
// for overlapping cells.
func Add(a, b, out tensor.Tensor) {
	if b.Len() == 1 {
		AddScalar(b.FloatRow(0), a, out)
		return
	}
	if a.Len() == 1 {
		AddScalar(a.FloatRow(0), b, out)
		return
	}
	arows, acells := a.RowCellSize()
	brows, bcells := b.RowCellSize()
	if brows*bcells == acells {
		AddSubSpace(a, b, out)
		return
	}
	if arows*acells == bcells {
		AddSubSpace(b, a, out)
		return
	}
	// just do element-wise
	tensor.SetShapeFrom(out, a)
	tensor.VectorizeThreaded(1, func(tsr ...tensor.Tensor) int {
		return tensor.NMinLen(2, tsr...)
	},
		func(idx int, tsr ...tensor.Tensor) {
			out.SetFloat1D(tsr[0].Float1D(idx)+tsr[1].Float1D(idx), idx)
		}, a, b, out)
}

// AddScalar adds a scalar to given tensor into output.
func AddScalar(scalar float64, a, out tensor.Tensor) {
	tensor.SetShapeFrom(out, a)
	tensor.VectorizeThreaded(1, tensor.NFirstLen, func(idx int, tsr ...tensor.Tensor) {
		out.SetFloat1D(a.Float1D(idx)+scalar, idx)
	}, a, out)
}

// AddSubSpace adds the subspace tensor to each row in the given tensor,
// into the output tensor.
func AddSubSpace(a, sub, out tensor.Tensor) {
	_, acells := a.RowCellSize()
	tensor.SetShapeFrom(out, a)
	tensor.VectorizeThreaded(1, tensor.NFirstLen, func(idx int, tsr ...tensor.Tensor) {
		si := idx % acells
		out.SetFloat1D(a.Float1D(idx)+sub.Float1D(si), idx)
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
func Sub(a, b, out tensor.Tensor) {
	if b.Len() == 1 {
		SubScalar(1, b.FloatRow(0), a, out)
		return
	}
	if a.Len() == 1 {
		SubScalar(-1, a.FloatRow(0), b, out)
		return
	}
	arows, acells := a.RowCellSize()
	brows, bcells := b.RowCellSize()
	if brows*bcells == acells {
		SubSubSpace(1, a, b, out)
		return
	}
	if arows*acells == bcells {
		SubSubSpace(-1, b, a, out)
		return
	}
	// just do element-wise
	tensor.SetShapeFrom(out, a)
	tensor.VectorizeThreaded(1, func(tsr ...tensor.Tensor) int {
		return tensor.NMinLen(2, tsr...)
	},
		func(idx int, tsr ...tensor.Tensor) {
			out.SetFloat1D(tsr[0].Float1D(idx)-tsr[1].Float1D(idx), idx)
		}, a, b, out)
}

// SubScalar subtracts a scalar from given tensor into output.
// sign determines which way the subtraction goes: 1 = a-scalar, -1 = scalar-a
func SubScalar(sign float64, scalar float64, a, out tensor.Tensor) {
	tensor.SetShapeFrom(out, a)
	tensor.VectorizeThreaded(1, tensor.NFirstLen, func(idx int, tsr ...tensor.Tensor) {
		out.SetFloat1D(sign*(a.Float1D(idx)-scalar), idx)
	}, a, out)
}

// SubSubSpace subtracts the subspace tensor to each row in the given tensor,
// into the output tensor.
// sign determines which way the subtraction goes: 1 = a-sub, -1 = sub-a
func SubSubSpace(sign float64, a, sub, out tensor.Tensor) {
	_, acells := a.RowCellSize()
	tensor.SetShapeFrom(out, a)
	tensor.VectorizeThreaded(1, tensor.NFirstLen, func(idx int, tsr ...tensor.Tensor) {
		si := idx % acells
		out.SetFloat1D(sign*(a.Float1D(idx)-sub.Float1D(si)), idx)
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
func Mul(a, b, out tensor.Tensor) {
	if b.Len() == 1 {
		MulScalar(b.FloatRow(0), a, out)
		return
	}
	if a.Len() == 1 {
		MulScalar(a.FloatRow(0), b, out)
		return
	}
	arows, acells := a.RowCellSize()
	brows, bcells := b.RowCellSize()
	if brows*bcells == acells {
		MulSubSpace(a, b, out)
		return
	}
	if arows*acells == bcells {
		MulSubSpace(b, a, out)
		return
	}
	// just do element-wise
	tensor.SetShapeFrom(out, a)
	tensor.VectorizeThreaded(1, func(tsr ...tensor.Tensor) int {
		return tensor.NMinLen(2, tsr...)
	},
		func(idx int, tsr ...tensor.Tensor) {
			out.SetFloat1D(tsr[0].Float1D(idx)*tsr[1].Float1D(idx), idx)
		}, a, b, out)
}

// MulScalar multiplies a scalar to given tensor into output.
func MulScalar(scalar float64, a, out tensor.Tensor) {
	tensor.SetShapeFrom(out, a)
	tensor.VectorizeThreaded(1, tensor.NFirstLen, func(idx int, tsr ...tensor.Tensor) {
		out.SetFloat1D(a.Float1D(idx)*scalar, idx)
	}, a, out)
}

// MulSubSpace multiplies the subspace tensor to each row in the given tensor,
// into the output tensor.
func MulSubSpace(a, sub, out tensor.Tensor) {
	_, acells := a.RowCellSize()
	tensor.SetShapeFrom(out, a)
	tensor.VectorizeThreaded(1, tensor.NFirstLen, func(idx int, tsr ...tensor.Tensor) {
		si := idx % acells
		out.SetFloat1D(a.Float1D(idx)*sub.Float1D(si), idx)
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
func Div(a, b, out tensor.Tensor) {
	if b.Len() == 1 {
		DivScalar(b.FloatRow(0), a, out)
		return
	}
	if a.Len() == 1 {
		DivScalarInv(a.FloatRow(0), b, out)
		return
	}
	arows, acells := a.RowCellSize()
	brows, bcells := b.RowCellSize()
	if brows*bcells == acells {
		DivSubSpace(a, b, out)
		return
	}
	if arows*acells == bcells {
		DivSubSpaceInv(b, a, out)
		return
	}
	// just do element-wise
	tensor.SetShapeFrom(out, a)
	tensor.VectorizeThreaded(1, func(tsr ...tensor.Tensor) int {
		return tensor.NMinLen(2, tsr...)
	},
		func(idx int, tsr ...tensor.Tensor) {
			out.SetFloat1D(tsr[0].Float1D(idx)/tsr[1].Float1D(idx), idx)
		}, a, b, out)
}

// DivScalar divides given tensor elements by scalar into output.
func DivScalar(scalar float64, a, out tensor.Tensor) {
	tensor.SetShapeFrom(out, a)
	tensor.VectorizeThreaded(1, tensor.NFirstLen, func(idx int, tsr ...tensor.Tensor) {
		out.SetFloat1D(a.Float1D(idx)/scalar, idx)
	}, a, out)
}

// DivScalarInv divides scalar by given tensor elements into output
// (inverse of [DivScalar]).
func DivScalarInv(scalar float64, a, out tensor.Tensor) {
	tensor.SetShapeFrom(out, a)
	tensor.VectorizeThreaded(1, tensor.NFirstLen, func(idx int, tsr ...tensor.Tensor) {
		out.SetFloat1D(scalar/a.Float1D(idx), idx)
	}, a, out)
}

// DivSubSpace divides each row of the given tensor by the subspace tensor elements,
// into the output tensor.
func DivSubSpace(a, sub, out tensor.Tensor) {
	_, acells := a.RowCellSize()
	tensor.SetShapeFrom(out, a)
	tensor.VectorizeThreaded(1, tensor.NFirstLen, func(idx int, tsr ...tensor.Tensor) {
		si := idx % acells
		out.SetFloat1D(a.Float1D(idx)/sub.Float1D(si), idx)
	}, a, sub, out)
}

// DivSubSpaceInv divides the subspace tensor by each row of the given tensor,
// into the output tensor (inverse of [DivSubSpace])
func DivSubSpaceInv(a, sub, out tensor.Tensor) {
	_, acells := a.RowCellSize()
	tensor.SetShapeFrom(out, a)
	tensor.VectorizeThreaded(1, tensor.NFirstLen, func(idx int, tsr ...tensor.Tensor) {
		si := idx % acells
		out.SetFloat1D(sub.Float1D(si)/a.Float1D(idx), idx)
	}, a, sub, out)
}
