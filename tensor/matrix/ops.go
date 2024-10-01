// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package matrix

import (
	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/tensor"
	"gonum.org/v1/gonum/mat"
)

// CallOut2 calls an Out function with 2 input args. All matrix functions
// require *tensor.Float64 outputs.
func CallOut2(fun func(a, b tensor.Tensor, out *tensor.Float64) error, a, b tensor.Tensor) tensor.Values {
	out := tensor.NewFloat64()
	errors.Log(fun(a, b, out))
	return out
}

// Mul performs matrix multiplication, using the following rules based
// on the shapes of the relevant tensors. If the tensor shapes are not
// suitable, an error is logged (see [MulOut] for a version returning the error).
//   - If both arguments are 2-D they are multiplied like conventional matrices.
//   - If either argument is N-D, N > 2, it is treated as a stack of matrices
//     residing in the last two indexes and broadcast accordingly.
//   - If the first argument is 1-D, it is promoted to a matrix by prepending
//     a 1 to its dimensions. After matrix multiplication the prepended 1 is removed.
//   - If the second argument is 1-D, it is promoted to a matrix by appending
//     a 1 to its dimensions. After matrix multiplication the appended 1 is removed.
func Mul(a, b tensor.Tensor) tensor.Tensor {
	return CallOut2(MulOut, a, b)
}

// MulOut performs matrix multiplication, into the given output tensor,
// using the following rules based
// on the shapes of the relevant tensors. If the tensor shapes are not
// suitable, an error is returned.
//   - If both arguments are 2-D they are multiplied like conventional matrices.
//     The result has shape a.Rows, b.Columns.
//   - If either argument is N-D, N > 2, it is treated as a stack of matrices
//     residing in the last two indexes and broadcast accordingly. Both cannot
//     be > 2 dimensional, unless their outer dimension size is 1 or the same.
//   - If the first argument is 1-D, it is promoted to a matrix by prepending
//     a 1 to its dimensions. After matrix multiplication the prepended 1 is removed.
//   - If the second argument is 1-D, it is promoted to a matrix by appending
//     a 1 to its dimensions. After matrix multiplication the appended 1 is removed.
func MulOut(a, b tensor.Tensor, out *tensor.Float64) error {
	if err := StringCheck(a); err != nil {
		return err
	}
	if err := StringCheck(b); err != nil {
		return err
	}
	na := a.NumDims()
	nb := b.NumDims()
	ea := a
	eb := b
	if na == 1 {
		ea = tensor.Reshape(a, 1, a.DimSize(0))
		na = 2
	}
	if nb == 1 {
		eb = tensor.Reshape(b, b.DimSize(0), 1)
		nb = 2
	}
	if na > 2 {
		asz := tensor.SplitAtInnerDims(a, 2)
		if asz[0] == 1 {
			ea = tensor.Reshape(a, asz[1:]...)
			na = 2
		}
	}
	if nb > 2 {
		bsz := tensor.SplitAtInnerDims(b, 2)
		if bsz[0] == 1 {
			eb = tensor.Reshape(b, bsz[1:]...)
			nb = 2
		}
	}
	switch {
	case na == nb && na == 2:
		ma, _ := NewMatrix(ea)
		mb, _ := NewMatrix(eb)
		out.SetShapeSizes(ea.DimSize(0), eb.DimSize(1))
		do, _ := NewDense(out)
		do.Mul(ma, mb)
	case na > 2 && nb == 2:
		mb, _ := NewMatrix(eb)
		nr := ea.DimSize(0)
		out.SetShapeSizes(nr, ea.DimSize(1), eb.DimSize(1))
		for r := range nr {
			sa := tensor.Reslice(ea, r, tensor.FullAxis, tensor.FullAxis)
			ma, _ := NewMatrix(sa)
			do, _ := NewDense(out.RowTensor(r).(*tensor.Float64))
			do.Mul(ma, mb)
		}
	case nb > 2 && na == 2:
		ma, _ := NewMatrix(ea)
		nr := eb.DimSize(0)
		out.SetShapeSizes(nr, ea.DimSize(0), eb.DimSize(2))
		for r := range nr {
			sb := tensor.Reslice(eb, r, tensor.FullAxis, tensor.FullAxis)
			mb, _ := NewMatrix(sb)
			do, _ := NewDense(out.RowTensor(r).(*tensor.Float64))
			do.Mul(ma, mb)
		}
	case na > 2 && nb > 2:
		if ea.DimSize(0) != eb.DimSize(0) {
			return errors.New("matrix.Mul: a and b input matricies are > 2 dimensional; must have same outer dimension sizes")
		}
		nr := ea.DimSize(0)
		out.SetShapeSizes(nr, ea.DimSize(1), eb.DimSize(2))
		for r := range nr {
			sa := tensor.Reslice(ea, r, tensor.FullAxis, tensor.FullAxis)
			ma, _ := NewMatrix(sa)
			sb := tensor.Reslice(eb, r, tensor.FullAxis, tensor.FullAxis)
			mb, _ := NewMatrix(sb)
			do, _ := NewDense(out.RowTensor(r).(*tensor.Float64))
			do.Mul(ma, mb)
		}
	default:
		return errors.New("matrix.Mul: input dimensions do not align")
	}
	return nil
}

// Det returns the determinant of the given tensor.
// For a 2D matrix [[a, b], [c, d]] it this is ad - bc.
// See also [LogDet] for a version that is more numerically
// stable for large matricies.
func Det(a tensor.Tensor) tensor.Tensor {
	m, err := NewMatrix(a)
	if errors.Log(err) != nil {
		return tensor.NewFloat64Scalar(0)
	}
	return tensor.NewFloat64Scalar(mat.Det(m))
}

// LogDet returns the determinant of the given tensor,
// as the log and sign of the value, which is more
// numerically stable. The return is a 1D vector of length 2,
// with the first value being the log, and the second the sign.
func LogDet(a tensor.Tensor) tensor.Tensor {
	m, err := NewMatrix(a)
	if errors.Log(err) != nil {
		return tensor.NewFloat64Scalar(0)
	}
	l, s := mat.LogDet(m)
	return tensor.NewFloat64FromValues(l, s)
}
