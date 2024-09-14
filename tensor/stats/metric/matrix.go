// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package metric

import (
	"log/slog"

	"cogentcore.org/core/math32/vecint"
	"cogentcore.org/core/tensor"
	"cogentcore.org/core/tensor/stats/stats"
	"cogentcore.org/core/tensor/tmath"
	"gonum.org/v1/gonum/mat"
)

func init() {
	tensor.AddFunc("metric.Matrix", Matrix, 1, tensor.StringFirstArg)
	tensor.AddFunc("metric.CrossMatrix", CrossMatrix, 1, tensor.StringFirstArg)
	tensor.AddFunc("metric.CovarMatrix", CovarMatrix, 1, tensor.StringFirstArg)
}

// Matrix computes the rows x rows square distance / similarity matrix
// between the patterns for each row of the given higher dimensional input tensor,
// which must have at least 2 dimensions: the outermost rows,
// and within that, 1+dimensional patterns (cells).
// The metric function registered in tensor Funcs can be passed as Metrics.FuncName().
// The results fill in the elements of the output matrix, which is symmetric,
// and only the lower triangular part is computed, with results copied
// to the upper triangular region, for maximum efficiency.
func Matrix(funcName string, in, out *tensor.Indexed) {
	rows, cells := in.RowCellSize()
	if rows == 0 || cells == 0 {
		return
	}
	out.Tensor.SetShape(rows, rows)
	mout := tensor.NewFloat64Scalar(0.0)
	coords := TriangularLIndicies(rows)
	nc := len(coords)
	// note: flops estimating 3 per item on average -- different for different metrics.
	tensor.VectorizeThreaded(cells*3, func(tsr ...*tensor.Indexed) int { return nc },
		func(idx int, tsr ...*tensor.Indexed) {
			c := coords[idx]
			sa := tsr[0].Cells1D(c.X)
			sb := tsr[0].Cells1D(c.Y)
			tensor.Call(funcName, sa, sb, mout)
			tsr[1].SetFloat(mout.Float1D(0), c.X, c.Y)
		}, in, out)
	for _, c := range coords { // copy to upper
		if c.X == c.Y { // exclude diag
			continue
		}
		out.Tensor.SetFloat(out.Tensor.Float(c.X, c.Y), c.Y, c.X)
	}
}

// CrossMatrix computes the distance / similarity matrix between
// two different sets of patterns in the two input tensors, where
// the patterns are in the sub-space cells of the tensors,
// which must have at least 2 dimensions: the outermost rows,
// and within that, 1+dimensional patterns that the given distance metric
// function is applied to, with the results filling in the cells of the output matrix.
// The metric function registered in tensor Funcs can be passed as Metrics.FuncName().
// The rows of the output matrix are the rows of the first input tensor,
// and the columns of the output are the rows of the second input tensor.
func CrossMatrix(funcName string, a, b, out *tensor.Indexed) {
	arows, acells := a.RowCellSize()
	if arows == 0 || acells == 0 {
		return
	}
	brows, bcells := b.RowCellSize()
	if brows == 0 || bcells == 0 {
		return
	}
	out.Tensor.SetShape(arows, brows)
	mout := tensor.NewFloat64Scalar(0.0)
	// note: flops estimating 3 per item on average -- different for different metrics.
	flops := min(acells, bcells) * 3
	nc := arows * brows
	tensor.VectorizeThreaded(flops, func(tsr ...*tensor.Indexed) int { return nc },
		func(idx int, tsr ...*tensor.Indexed) {
			ar := idx / brows
			br := idx % brows
			sa := tsr[0].Cells1D(ar)
			sb := tsr[1].Cells1D(br)
			tensor.Call(funcName, sa, sb, mout)
			tsr[2].SetFloat(mout.Float1D(0), ar, br)
		}, a, b, out)
}

// CovarMatrix generates the cells x cells square covariance matrix
// for all per-row cells of the given higher dimensional input tensor,
// which must have at least 2 dimensions: the outermost rows,
// and within that, 1+dimensional patterns (cells).
// Each value in the resulting matrix represents the extent to which the
// value of a given cell covaries across the rows of the tensor with the
// value of another cell.
// Uses the given metric function, typically [Covariance] or [Correlation],
// which must be registered in tensor Funcs, and can be passed as Metrics.FuncName().
// Use Covariance if vars have similar overall scaling,
// which is typical in neural network models, and use
// Correlation if they are on very different scales, because it effectively rescales).
// The resulting matrix can be used as the input to PCA or SVD eigenvalue decomposition.
func CovarMatrix(funcName string, in, out *tensor.Indexed) {
	rows, cells := in.RowCellSize()
	if rows == 0 || cells == 0 {
		return
	}

	flatsz := []int{in.Tensor.DimSize(0), cells}
	flatvw := in.Tensor.View()
	flatvw.SetShape(flatsz...)
	flatix := tensor.NewIndexed(flatvw)
	flatix.Indexes = in.Indexes

	mout := tensor.NewFloat64Scalar(0.0)
	out.Tensor.SetShape(cells, cells)
	av := tensor.NewFloat64Indexed(rows)
	bv := tensor.NewFloat64Indexed(rows)
	curCoords := vecint.Vector2i{-1, -1}

	coords := TriangularLIndicies(cells)
	nc := len(coords)
	// note: flops estimating 3 per item on average -- different for different metrics.
	tensor.VectorizeThreaded(rows*3, func(tsr ...*tensor.Indexed) int { return nc },
		func(idx int, tsr ...*tensor.Indexed) {
			c := coords[idx]
			if c.X != curCoords.X {
				tensor.Slice(tsr[0], av, tensor.Range{}, tensor.Range{Start: c.X, End: c.X + 1})
				curCoords.X = c.X
			}
			if c.Y != curCoords.Y {
				tensor.Slice(tsr[0], bv, tensor.Range{}, tensor.Range{Start: c.Y, End: c.Y + 1})
				curCoords.Y = c.Y
			}
			tensor.Call(funcName, av, bv, mout)
			tsr[1].SetFloat(mout.Float1D(0), c.X, c.Y)
		}, flatix, out)
	for _, c := range coords { // copy to upper
		if c.X == c.Y { // exclude diag
			continue
		}
		out.Tensor.SetFloat(out.Tensor.Float(c.X, c.Y), c.Y, c.X)
	}
}

// PCA performs the eigen decomposition of the given CovarMatrix,
// using principal components analysis (PCA), which is slower than [SVD].
// The eigenvectors are same size as Covar. Each eigenvector is a column
// in this 2D square matrix, ordered *lowest* to *highest* across the columns,
// i.e., maximum eigenvector is the last column.
// The eigenvalues are the size of one row, ordered *lowest* to *highest*.
// Note that PCA produces results in the *opposite* order of [SVD].
func PCA(covar, eigenvecs, vals *tensor.Indexed) {
	n := covar.Tensor.DimSize(0)
	cv := tensor.AsFloat64(covar.Tensor)
	eigenvecs.Tensor.SetShape(n, n)
	eigenvecs.Sequential()
	vals.Tensor.SetShape(n)
	vals.Sequential()
	var eig mat.EigenSym
	ok := eig.Factorize(cv, true)
	if !ok {
		slog.Error("gonum mat.EigenSym Factorize failed")
		return
	}
	var ev mat.Dense
	eig.VectorsTo(&ev)
	tensor.CopyDense(eigenvecs.Tensor, &ev)
	fv := tensor.AsFloat64(vals.Tensor)
	eig.Values(fv.Values)
	if fv != vals.Tensor {
		vals.Tensor.CopyFrom(fv)
	}
}

// SVD performs the eigen decomposition of the given CovarMatrix,
// using singular value decomposition (SVD), which is faster than [PCA].
// The eigenvectors are same size as Covar. Each eigenvector is a column
// in this 2D square matrix, ordered *highest* to *lowest* across the columns,
// i.e., maximum eigenvector is the last column.
// The eigenvalues are the size of one row, ordered *highest* to *lowest*.
// Note that SVD produces results in the *opposite* order of [PCA].
func SVD(covar, eigenvecs, vals *tensor.Indexed) {
	n := covar.Tensor.DimSize(0)
	cv := tensor.AsFloat64(covar.Tensor)
	eigenvecs.Tensor.SetShape(n, n)
	eigenvecs.Sequential()
	vals.Tensor.SetShape(n)
	vals.Sequential()
	var eig mat.SVD
	ok := eig.Factorize(cv, mat.SVDFull) // todo: benchmark different versions
	if !ok {
		slog.Error("gonum mat.SVD Factorize failed")
		return
	}
	var ev mat.Dense
	eig.UTo(&ev)
	tensor.CopyDense(eigenvecs.Tensor, &ev)
	fv := tensor.AsFloat64(vals.Tensor)
	eig.Values(fv.Values)
	if fv != vals.Tensor {
		vals.Tensor.CopyFrom(fv)
	}
}

// ProjectOnMatrixColumn is a convenience function for projecting given vector
// of values along a specific column (2nd dimension) of the given 2D matrix,
// specified by the scalar colindex, putting results into out.
// If the vec is more than 1 dimensional, then it is treated as rows x cells,
// and each row of cells is projected through the matrix column, producing a
// 1D output with the number of rows.  Otherwise a single number is produced.
// This is typically done with results from SVD or PCA.
func ProjectOnMatrixColumn(mtx, vec, colindex, out *tensor.Indexed) {
	ci := int(colindex.Float1D(0))
	col := tensor.NewFloat64Indexed()
	tensor.Slice(mtx, col, tensor.Range{}, tensor.Range{Start: ci, End: ci + 1})
	// fmt.Println(mtx.Tensor.String(), col.Tensor.String())
	rows, cells := vec.RowCellSize()
	mout := tensor.NewFloat64Indexed()
	if rows > 0 && cells > 0 {
		msum := tensor.NewFloat64Scalar(0)
		out.Tensor.SetShape(rows)
		out.Sequential()
		for i := range rows {
			tmath.Mul(vec.Cells1D(i), col, mout)
			stats.SumFunc(mout, msum)
			// fmt.Println(vec.Cells1D(i).Tensor.String(), mout.Tensor.String(), msum.Tensor.String())
			out.SetFloat1D(msum.Float1D(0), i)
		}
	} else {
		tmath.Mul(vec, col, mout)
		stats.SumFunc(mout, out)
	}
}

////////////////////////////////////////////
// 	Triangular square matrix functions

// TODO: move these somewhere more appropriate

// note: this answer gives an index into the upper triangular
// https://math.stackexchange.com/questions/2134011/conversion-of-upper-triangle-linear-index-from-index-on-symmetrical-array
// return TriangularN(n) - ((n-c)*(n-c-1))/2 + r - c - 1 <- this works for lower excluding diag
// return (n * (n - 1) / 2) - ((n-r)*(n-r-1))/2 + c <- this works for upper including diag
// but I wasn't able to get an equation for r, c back from index, for this "including diagonal"
// https://stackoverflow.com/questions/27086195/linear-index-upper-triangular-matrix?rq=3
// python just iterates manually and returns a list
// https://github.com/numpy/numpy/blob/v2.1.0/numpy/lib/_twodim_base_impl.py#L902-L985

// TriangularN returns the number of elements in the triangular region
// of a square matrix of given size, where the triangle includes the
// n elements along the diagonal.
func TriangularN(n int) int {
	return n + (n*(n-1))/2
}

// TriangularLIndicies returns the list of r, c indexes (as X, Y coordinates)
// for the lower triangular portion of a square matrix of size n,
// including the diagonal.
func TriangularLIndicies(n int) []vecint.Vector2i {
	trin := TriangularN(n)
	coords := make([]vecint.Vector2i, trin)
	i := 0
	for r := range n {
		for c := range n {
			if c <= r {
				coords[i] = vecint.Vector2i{X: r, Y: c}
				i++
			}
		}
	}
	return coords
}
