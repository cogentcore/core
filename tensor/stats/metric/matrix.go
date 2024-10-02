// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package metric

import (
	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/math32/vecint"
	"cogentcore.org/core/tensor"
	"cogentcore.org/core/tensor/matrix"
	"cogentcore.org/core/tensor/stats/stats"
	"cogentcore.org/core/tensor/tmath"
	"gonum.org/v1/gonum/mat"
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

// PCA performs the eigen decomposition of the given CovarianceMatrix,
// using principal components analysis (PCA), which is slower than [SVD].
// The eigenvectors are same size as Covar. Each eigenvector is a column
// in this 2D square matrix, ordered *lowest* to *highest* across the columns,
// i.e., maximum eigenvector is the last column.
// The eigenvalues are the size of one row, ordered *lowest* to *highest*.
// Note that PCA produces results in the *opposite* order of [SVD].
func PCA(covar tensor.Tensor, eigenvecs, vals tensor.Values) error {
	n := covar.DimSize(0)
	cv, err := matrix.NewSymmetric(tensor.AsFloat64(covar))
	if err != nil {
		return err
	}
	eigenvecs.SetShapeSizes(n, n)
	vals.SetShapeSizes(n)
	var eig mat.EigenSym
	ok := eig.Factorize(cv, true)
	if !ok {
		return errors.New("gonum mat.EigenSym Factorize failed")
	}
	var ev mat.Dense
	eig.VectorsTo(&ev)
	matrix.CopyFromDense(eigenvecs, &ev)
	fv := tensor.AsFloat64(vals)
	eig.Values(fv.Values)
	if fv != vals {
		vals.(tensor.Values).CopyFrom(fv)
	}
	return nil
}

// SVD performs the eigen decomposition of the given CovarianceMatrix,
// using singular value decomposition (SVD), which is faster than [PCA].
// The eigenvectors are same size as Covar. Each eigenvector is a column
// in this 2D square matrix, ordered *highest* to *lowest* across the columns,
// i.e., maximum eigenvector is the last column.
// The eigenvalues are the size of one row, ordered *highest* to *lowest*.
// Note that SVD produces results in the *opposite* order of [PCA].
func SVD(covar tensor.Tensor, eigenvecs, vals tensor.Values) error {
	n := covar.DimSize(0)
	cv, err := matrix.NewSymmetric(tensor.AsFloat64(covar))
	if err != nil {
		return err
	}
	eigenvecs.SetShapeSizes(n, n)
	vals.SetShapeSizes(n)
	var eig mat.SVD
	ok := eig.Factorize(cv, mat.SVDFull) // todo: benchmark different versions
	if !ok {
		return errors.New("gonum mat.SVD Factorize failed")
	}
	var ev mat.Dense
	eig.UTo(&ev)
	matrix.CopyFromDense(eigenvecs, &ev)
	fv := tensor.AsFloat64(vals)
	eig.Values(fv.Values)
	if fv != vals {
		vals.(tensor.Values).CopyFrom(fv)
	}
	return nil
}

// ProjectOnMatrixColumnOut is a convenience function for projecting given vector
// of values along a specific column (2nd dimension) of the given 2D matrix,
// specified by the scalar colindex, putting results into out.
// If the vec is more than 1 dimensional, then it is treated as rows x cells,
// and each row of cells is projected through the matrix column, producing a
// 1D output with the number of rows.  Otherwise a single number is produced.
// This is typically done with results from SVD or PCA.
func ProjectOnMatrixColumnOut(mtx, vec, colindex tensor.Tensor, out tensor.Values) error {
	ci := int(colindex.Float1D(0))
	col := tensor.As1D(tensor.Reslice(mtx, tensor.Slice{}, ci))
	// fmt.Println(mtx.String(), col.String())
	rows, cells := vec.Shape().RowCellSize()
	if rows > 0 && cells > 0 {
		msum := tensor.NewFloat64Scalar(0)
		out.SetShapeSizes(rows)
		mout := tensor.NewFloat64(cells)
		for i := range rows {
			err := tmath.MulOut(tensor.Cells1D(vec, i), col, mout)
			if err != nil {
				return err
			}
			stats.SumOut(mout, msum)
			out.SetFloat1D(msum.Float1D(0), i)
		}
	} else {
		mout := tensor.NewFloat64(1)
		tmath.MulOut(vec, col, mout)
		stats.SumOut(mout, out)
	}
	return nil
}

// ProjectOnMatrixColumn is a convenience function for projecting given vector
// of values along a specific column (2nd dimension) of the given 2D matrix,
// specified by the scalar colindex, putting results into out.
// If the vec is more than 1 dimensional, then it is treated as rows x cells,
// and each row of cells is projected through the matrix column, producing a
// 1D output with the number of rows.  Otherwise a single number is produced.
// This is typically done with results from SVD or PCA.
func ProjectOnMatrixColumn(mtx, vec, colindex tensor.Tensor) tensor.Values {
	out := tensor.NewOfType(vec.DataType())
	errors.Log(ProjectOnMatrixColumnOut(mtx, vec, colindex, out))
	return out
}
