// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package metric

import (
	"cogentcore.org/core/math32/vecint"
	"cogentcore.org/core/tensor"
	"cogentcore.org/core/tensor/matrix"
)

// MatrixOut computes the rows x rows square distance / similarity matrix
// between the patterns for each row of the given higher dimensional input tensor,
// which must have at least 2 dimensions: the outermost rows,
// and within that, 1+dimensional patterns (cells). Use [tensor.NewRowCellsView]
// to organize data into the desired split between a 1D outermost Row dimension
// and the remaining Cells dimension.
// The metric function must have the [MetricFunc] signature.
// The results fill in the elements of the output matrix, which is symmetric,
// and only the lower triangular part is computed, with results copied
// to the upper triangular region, for maximum efficiency.
func MatrixOut(fun any, in tensor.Tensor, out tensor.Values) error {
	mfun, err := AsMetricFunc(fun)
	if err != nil {
		return err
	}
	rows, cells := in.Shape().RowCellSize()
	if rows == 0 || cells == 0 {
		return nil
	}
	out.SetShapeSizes(rows, rows)
	coords := matrix.TriLIndicies(rows)
	nc := coords.DimSize(0)
	// note: flops estimating 3 per item on average -- different for different metrics.
	tensor.VectorizeThreaded(cells*3, func(tsr ...tensor.Tensor) int { return nc },
		func(idx int, tsr ...tensor.Tensor) {
			cx := coords.Int(idx, 0)
			cy := coords.Int(idx, 1)
			sa := tensor.Cells1D(tsr[0], cx)
			sb := tensor.Cells1D(tsr[0], cy)
			mout := mfun(sa, sb)
			tsr[1].SetFloat(mout.Float1D(0), cx, cy)
		}, in, out)
	for idx := range nc { // copy to upper
		cx := coords.Int(idx, 0)
		cy := coords.Int(idx, 1)
		if cx == cy { // exclude diag
			continue
		}
		out.SetFloat(out.Float(cx, cy), cy, cx)
	}
	return nil
}

// Matrix computes the rows x rows square distance / similarity matrix
// between the patterns for each row of the given higher dimensional input tensor,
// which must have at least 2 dimensions: the outermost rows,
// and within that, 1+dimensional patterns (cells). Use [tensor.NewRowCellsView]
// to organize data into the desired split between a 1D outermost Row dimension
// and the remaining Cells dimension.
// The metric function must have the [MetricFunc] signature.
// The results fill in the elements of the output matrix, which is symmetric,
// and only the lower triangular part is computed, with results copied
// to the upper triangular region, for maximum efficiency.
func Matrix(fun any, in tensor.Tensor) tensor.Values {
	return tensor.CallOut1Gen1(MatrixOut, fun, in)
}

// CrossMatrixOut computes the distance / similarity matrix between
// two different sets of patterns in the two input tensors, where
// the patterns are in the sub-space cells of the tensors,
// which must have at least 2 dimensions: the outermost rows,
// and within that, 1+dimensional patterns that the given distance metric
// function is applied to, with the results filling in the cells of the output matrix.
// The metric function must have the [MetricFunc] signature.
// The rows of the output matrix are the rows of the first input tensor,
// and the columns of the output are the rows of the second input tensor.
func CrossMatrixOut(fun any, a, b tensor.Tensor, out tensor.Values) error {
	mfun, err := AsMetricFunc(fun)
	if err != nil {
		return err
	}
	arows, acells := a.Shape().RowCellSize()
	if arows == 0 || acells == 0 {
		return nil
	}
	brows, bcells := b.Shape().RowCellSize()
	if brows == 0 || bcells == 0 {
		return nil
	}
	out.SetShapeSizes(arows, brows)
	// note: flops estimating 3 per item on average -- different for different metrics.
	flops := min(acells, bcells) * 3
	nc := arows * brows
	tensor.VectorizeThreaded(flops, func(tsr ...tensor.Tensor) int { return nc },
		func(idx int, tsr ...tensor.Tensor) {
			ar := idx / brows
			br := idx % brows
			sa := tensor.Cells1D(tsr[0], ar)
			sb := tensor.Cells1D(tsr[1], br)
			mout := mfun(sa, sb)
			tsr[2].SetFloat(mout.Float1D(0), ar, br)
		}, a, b, out)
	return nil
}

// CrossMatrix computes the distance / similarity matrix between
// two different sets of patterns in the two input tensors, where
// the patterns are in the sub-space cells of the tensors,
// which must have at least 2 dimensions: the outermost rows,
// and within that, 1+dimensional patterns that the given distance metric
// function is applied to, with the results filling in the cells of the output matrix.
// The metric function must have the [MetricFunc] signature.
// The rows of the output matrix are the rows of the first input tensor,
// and the columns of the output are the rows of the second input tensor.
func CrossMatrix(fun any, a, b tensor.Tensor) tensor.Values {
	return tensor.CallOut2Gen1(CrossMatrixOut, fun, a, b)
}

// CovarianceMatrixOut generates the cells x cells square covariance matrix
// for all per-row cells of the given higher dimensional input tensor,
// which must have at least 2 dimensions: the outermost rows,
// and within that, 1+dimensional patterns (cells).
// Each value in the resulting matrix represents the extent to which the
// value of a given cell covaries across the rows of the tensor with the
// value of another cell.
// Uses the given metric function, typically [Covariance] or [Correlation],
// The metric function must have the [MetricFunc] signature.
// Use Covariance if vars have similar overall scaling,
// which is typical in neural network models, and use
// Correlation if they are on very different scales, because it effectively rescales).
// The resulting matrix can be used as the input to PCA or SVD eigenvalue decomposition.
func CovarianceMatrixOut(fun any, in tensor.Tensor, out tensor.Values) error {
	mfun, err := AsMetricFunc(fun)
	if err != nil {
		return err
	}
	rows, cells := in.Shape().RowCellSize()
	if rows == 0 || cells == 0 {
		return nil
	}
	out.SetShapeSizes(cells, cells)
	flatvw := tensor.NewReshaped(in, rows, cells)

	var av, bv tensor.Tensor
	curCoords := vecint.Vector2i{-1, -1}

	coords := matrix.TriLIndicies(cells)
	nc := coords.DimSize(0)
	// note: flops estimating 3 per item on average -- different for different metrics.
	tensor.VectorizeThreaded(rows*3, func(tsr ...tensor.Tensor) int { return nc },
		func(idx int, tsr ...tensor.Tensor) {
			cx := coords.Int(idx, 0)
			cy := coords.Int(idx, 1)
			if cx != curCoords.X {
				av = tensor.Reslice(tsr[0], tensor.FullAxis, cx)
				curCoords.X = cx
			}
			if cy != curCoords.Y {
				bv = tensor.Reslice(tsr[0], tensor.FullAxis, cy)
				curCoords.Y = cy
			}
			mout := mfun(av, bv)
			tsr[1].SetFloat(mout.Float1D(0), cx, cy)
		}, flatvw, out)
	for idx := range nc { // copy to upper
		cx := coords.Int(idx, 0)
		cy := coords.Int(idx, 1)
		if cx == cy { // exclude diag
			continue
		}
		out.SetFloat(out.Float(cx, cy), cy, cx)
	}
	return nil
}

// CovarianceMatrix generates the cells x cells square covariance matrix
// for all per-row cells of the given higher dimensional input tensor,
// which must have at least 2 dimensions: the outermost rows,
// and within that, 1+dimensional patterns (cells).
// Each value in the resulting matrix represents the extent to which the
// value of a given cell covaries across the rows of the tensor with the
// value of another cell.
// Uses the given metric function, typically [Covariance] or [Correlation],
// The metric function must have the [MetricFunc] signature.
// Use Covariance if vars have similar overall scaling,
// which is typical in neural network models, and use
// Correlation if they are on very different scales, because it effectively rescales).
// The resulting matrix can be used as the input to PCA or SVD eigenvalue decomposition.
func CovarianceMatrix(fun any, in tensor.Tensor) tensor.Values {
	return tensor.CallOut1Gen1(CovarianceMatrixOut, fun, in)
}
