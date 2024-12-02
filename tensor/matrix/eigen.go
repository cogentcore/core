// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package matrix

import (
	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/tensor"
	"cogentcore.org/core/tensor/stats/stats"
	"cogentcore.org/core/tensor/tmath"
	"gonum.org/v1/gonum/mat"
)

// Eig performs the eigen decomposition of the given square matrix,
// which is not symmetric. See EigSym for a symmetric square matrix.
// In this non-symmetric case, the results are typically complex valued,
// so the outputs are complex tensors. TODO: need complex support!
// The vectors are same size as the input. Each vector is a column
// in this 2D square matrix, ordered *lowest* to *highest* across the columns,
// i.e., maximum vector is the last column.
// The values are the size of one row, ordered *lowest* to *highest*.
// If the input tensor is > 2D, it is treated as a list of 2D matricies,
// and parallel threading is used where beneficial.
func Eig(a tensor.Tensor) (vecs, vals *tensor.Float64) {
	vecs = tensor.NewFloat64()
	vals = tensor.NewFloat64()
	errors.Log(EigOut(a, vecs, vals))
	return
}

// EigOut performs the eigen decomposition of the given square matrix,
// which is not symmetric. See EigSym for a symmetric square matrix.
// In this non-symmetric case, the results are typically complex valued,
// so the outputs are complex tensors. TODO: need complex support!
// The vectors are same size as the input. Each vector is a column
// in this 2D square matrix, ordered *lowest* to *highest* across the columns,
// i.e., maximum vector is the last column.
// The values are the size of one row, ordered *lowest* to *highest*.
// If the input tensor is > 2D, it is treated as a list of 2D matricies,
// and parallel threading is used where beneficial.
func EigOut(a tensor.Tensor, vecs, vals *tensor.Float64) error {
	if err := StringCheck(a); err != nil {
		return err
	}
	na := a.NumDims()
	if na == 1 {
		return mat.ErrShape
	}
	var asz []int
	ea := a
	if na > 2 {
		asz = tensor.SplitAtInnerDims(a, 2)
		if asz[0] == 1 {
			ea = tensor.Reshape(a, asz[1:]...)
			na = 2
		}
	}
	if na == 2 {
		if a.DimSize(0) != a.DimSize(1) {
			return mat.ErrShape
		}
		ma, _ := NewMatrix(a)
		vecs.SetShapeSizes(a.DimSize(0), a.DimSize(1))
		vals.SetShapeSizes(a.DimSize(0))
		do, _ := NewDense(vecs)
		var eig mat.Eigen
		ok := eig.Factorize(ma, mat.EigenRight)
		if !ok {
			return errors.New("gonum mat.Eigen Factorize failed")
		}
		_ = do
		// eig.VectorsTo(do) // todo: requires complex!
		// eig.Values(vals.Values)
		return nil
	}
	ea = tensor.Reshape(a, asz...)
	if ea.DimSize(1) != ea.DimSize(2) {
		return mat.ErrShape
	}
	nr := ea.DimSize(0)
	sz := ea.DimSize(1)
	vecs.SetShapeSizes(nr, sz, sz)
	vals.SetShapeSizes(nr, sz)
	var errs []error
	tensor.VectorizeThreaded(ea.DimSize(1)*ea.DimSize(2)*1000,
		func(tsr ...tensor.Tensor) int { return nr },
		func(r int, tsr ...tensor.Tensor) {
			sa := tensor.Reslice(ea, r, tensor.FullAxis, tensor.FullAxis)
			ma, _ := NewMatrix(sa)
			do, _ := NewDense(vecs.RowTensor(r).(*tensor.Float64))
			var eig mat.Eigen
			ok := eig.Factorize(ma, mat.EigenRight)
			if !ok {
				errs = append(errs, errors.New("gonum mat.Eigen Factorize failed"))
			}
			_ = do
			// eig.VectorsTo(do) // todo: requires complex!
			// eig.Values(vals.Values[r*sz : (r+1)*sz])
		})
	return errors.Join(errs...)
}

// EigSym performs the eigen decomposition of the given symmetric square matrix,
// which produces real-valued results. When input is the [metric.CovarianceMatrix],
// this is known as Principal Components Analysis (PCA).
// The vectors are same size as the input. Each vector is a column
// in this 2D square matrix, ordered *lowest* to *highest* across the columns,
// i.e., maximum vector is the last column.
// The values are the size of one row, ordered *lowest* to *highest*.
// Note that Eig produces results in the *opposite* order of [SVD] (which is much faster).
// If the input tensor is > 2D, it is treated as a list of 2D matricies,
// and parallel threading is used where beneficial.
func EigSym(a tensor.Tensor) (vecs, vals *tensor.Float64) {
	vecs = tensor.NewFloat64()
	vals = tensor.NewFloat64()
	errors.Log(EigSymOut(a, vecs, vals))
	return
}

// EigSymOut performs the eigen decomposition of the given symmetric square matrix,
// which produces real-valued results. When input is the [metric.CovarianceMatrix],
// this is known as Principal Components Analysis (PCA).
// The vectors are same size as the input. Each vector is a column
// in this 2D square matrix, ordered *lowest* to *highest* across the columns,
// i.e., maximum vector is the last column.
// The values are the size of one row, ordered *lowest* to *highest*.
// Note that Eig produces results in the *opposite* order of [SVD] (which is much faster).
// If the input tensor is > 2D, it is treated as a list of 2D matricies,
// and parallel threading is used where beneficial.
func EigSymOut(a tensor.Tensor, vecs, vals *tensor.Float64) error {
	if err := StringCheck(a); err != nil {
		return err
	}
	na := a.NumDims()
	if na == 1 {
		return mat.ErrShape
	}
	var asz []int
	ea := a
	if na > 2 {
		asz = tensor.SplitAtInnerDims(a, 2)
		if asz[0] == 1 {
			ea = tensor.Reshape(a, asz[1:]...)
			na = 2
		}
	}
	if na == 2 {
		if a.DimSize(0) != a.DimSize(1) {
			return mat.ErrShape
		}
		ma, _ := NewSymmetric(a)
		vecs.SetShapeSizes(a.DimSize(0), a.DimSize(1))
		vals.SetShapeSizes(a.DimSize(0))
		do, _ := NewDense(vecs)
		var eig mat.EigenSym
		ok := eig.Factorize(ma, true)
		if !ok {
			return errors.New("gonum mat.EigenSym Factorize failed")
		}
		eig.VectorsTo(do)
		eig.Values(vals.Values)
		return nil
	}
	ea = tensor.Reshape(a, asz...)
	if ea.DimSize(1) != ea.DimSize(2) {
		return mat.ErrShape
	}
	nr := ea.DimSize(0)
	sz := ea.DimSize(1)
	vecs.SetShapeSizes(nr, sz, sz)
	vals.SetShapeSizes(nr, sz)
	var errs []error
	tensor.VectorizeThreaded(ea.DimSize(1)*ea.DimSize(2)*1000,
		func(tsr ...tensor.Tensor) int { return nr },
		func(r int, tsr ...tensor.Tensor) {
			sa := tensor.Reslice(ea, r, tensor.FullAxis, tensor.FullAxis)
			ma, _ := NewSymmetric(sa)
			do, _ := NewDense(vecs.RowTensor(r).(*tensor.Float64))
			var eig mat.EigenSym
			ok := eig.Factorize(ma, true)
			if !ok {
				errs = append(errs, errors.New("gonum mat.Eigen Factorize failed"))
			}
			eig.VectorsTo(do)
			eig.Values(vals.Values[r*sz : (r+1)*sz])
		})
	return errors.Join(errs...)
}

// SVD performs the singular value decomposition of the given symmetric square matrix,
// which produces real-valued results, and is generally much faster than [EigSym],
// while producing the same results.
// The vectors are same size as the input. Each vector is a column
// in this 2D square matrix, ordered *highest* to *lowest* across the columns,
// i.e., maximum vector is the first column.
// The values are the size of one row ordered in alignment with the vectors.
// Note that SVD produces results in the *opposite* order of [EigSym].
// If the input tensor is > 2D, it is treated as a list of 2D matricies,
// and parallel threading is used where beneficial.
func SVD(a tensor.Tensor) (vecs, vals *tensor.Float64) {
	vecs = tensor.NewFloat64()
	vals = tensor.NewFloat64()
	errors.Log(SVDOut(a, vecs, vals))
	return
}

// SVDOut performs the singular value decomposition of the given symmetric square matrix,
// which produces real-valued results, and is generally much faster than [EigSym],
// while producing the same results.
// The vectors are same size as the input. Each vector is a column
// in this 2D square matrix, ordered *highest* to *lowest* across the columns,
// i.e., maximum vector is the first column.
// The values are the size of one row ordered in alignment with the vectors.
// Note that SVD produces results in the *opposite* order of [EigSym].
// If the input tensor is > 2D, it is treated as a list of 2D matricies,
// and parallel threading is used where beneficial.
func SVDOut(a tensor.Tensor, vecs, vals *tensor.Float64) error {
	if err := StringCheck(a); err != nil {
		return err
	}
	na := a.NumDims()
	if na == 1 {
		return mat.ErrShape
	}
	var asz []int
	ea := a
	if na > 2 {
		asz = tensor.SplitAtInnerDims(a, 2)
		if asz[0] == 1 {
			ea = tensor.Reshape(a, asz[1:]...)
			na = 2
		}
	}
	if na == 2 {
		if a.DimSize(0) != a.DimSize(1) {
			return mat.ErrShape
		}
		ma, _ := NewSymmetric(a)
		vecs.SetShapeSizes(a.DimSize(0), a.DimSize(1))
		vals.SetShapeSizes(a.DimSize(0))
		do, _ := NewDense(vecs)
		var eig mat.SVD
		ok := eig.Factorize(ma, mat.SVDFull)
		if !ok {
			return errors.New("gonum mat.SVD Factorize failed")
		}
		eig.UTo(do)
		eig.Values(vals.Values)
		return nil
	}
	ea = tensor.Reshape(a, asz...)
	if ea.DimSize(1) != ea.DimSize(2) {
		return mat.ErrShape
	}
	nr := ea.DimSize(0)
	sz := ea.DimSize(1)
	vecs.SetShapeSizes(nr, sz, sz)
	vals.SetShapeSizes(nr, sz)
	var errs []error
	tensor.VectorizeThreaded(ea.DimSize(1)*ea.DimSize(2)*1000,
		func(tsr ...tensor.Tensor) int { return nr },
		func(r int, tsr ...tensor.Tensor) {
			sa := tensor.Reslice(ea, r, tensor.FullAxis, tensor.FullAxis)
			ma, _ := NewSymmetric(sa)
			do, _ := NewDense(vecs.RowTensor(r).(*tensor.Float64))
			var eig mat.SVD
			ok := eig.Factorize(ma, mat.SVDFull)
			if !ok {
				errs = append(errs, errors.New("gonum mat.SVD Factorize failed"))
			}
			eig.UTo(do)
			eig.Values(vals.Values[r*sz : (r+1)*sz])
		})
	return errors.Join(errs...)
}

// SVDValues performs the singular value decomposition of the given
// symmetric square matrix, which produces real-valued results,
// and is generally much faster than [EigSym], while producing the same results.
// This version only generates eigenvalues, not vectors: see [SVD].
// The values are the size of one row ordered highest to lowest,
// which is the opposite of [EigSym].
// If the input tensor is > 2D, it is treated as a list of 2D matricies,
// and parallel threading is used where beneficial.
func SVDValues(a tensor.Tensor) *tensor.Float64 {
	vals := tensor.NewFloat64()
	errors.Log(SVDValuesOut(a, vals))
	return vals
}

// SVDValuesOut performs the singular value decomposition of the given
// symmetric square matrix, which produces real-valued results,
// and is generally much faster than [EigSym], while producing the same results.
// This version only generates eigenvalues, not vectors: see [SVDOut].
// The values are the size of one row ordered highest to lowest,
// which is the opposite of [EigSym].
// If the input tensor is > 2D, it is treated as a list of 2D matricies,
// and parallel threading is used where beneficial.
func SVDValuesOut(a tensor.Tensor, vals *tensor.Float64) error {
	if err := StringCheck(a); err != nil {
		return err
	}
	na := a.NumDims()
	if na == 1 {
		return mat.ErrShape
	}
	var asz []int
	ea := a
	if na > 2 {
		asz = tensor.SplitAtInnerDims(a, 2)
		if asz[0] == 1 {
			ea = tensor.Reshape(a, asz[1:]...)
			na = 2
		}
	}
	if na == 2 {
		if a.DimSize(0) != a.DimSize(1) {
			return mat.ErrShape
		}
		ma, _ := NewSymmetric(a)
		vals.SetShapeSizes(a.DimSize(0))
		var eig mat.SVD
		ok := eig.Factorize(ma, mat.SVDNone)
		if !ok {
			return errors.New("gonum mat.SVD Factorize failed")
		}
		eig.Values(vals.Values)
		return nil
	}
	ea = tensor.Reshape(a, asz...)
	if ea.DimSize(1) != ea.DimSize(2) {
		return mat.ErrShape
	}
	nr := ea.DimSize(0)
	sz := ea.DimSize(1)
	vals.SetShapeSizes(nr, sz)
	var errs []error
	tensor.VectorizeThreaded(ea.DimSize(1)*ea.DimSize(2)*1000,
		func(tsr ...tensor.Tensor) int { return nr },
		func(r int, tsr ...tensor.Tensor) {
			sa := tensor.Reslice(ea, r, tensor.FullAxis, tensor.FullAxis)
			ma, _ := NewSymmetric(sa)
			var eig mat.SVD
			ok := eig.Factorize(ma, mat.SVDNone)
			if !ok {
				errs = append(errs, errors.New("gonum mat.SVD Factorize failed"))
			}
			eig.Values(vals.Values[r*sz : (r+1)*sz])
		})
	return errors.Join(errs...)
}

// ProjectOnMatrixColumn is a convenience function for projecting given vector
// of values along a specific column (2nd dimension) of the given 2D matrix,
// specified by the scalar colindex, putting results into out.
// If the vec is more than 1 dimensional, then it is treated as rows x cells,
// and each row of cells is projected through the matrix column, producing a
// 1D output with the number of rows.  Otherwise a single number is produced.
// This is typically done with results from SVD or EigSym (PCA).
func ProjectOnMatrixColumn(mtx, vec, colindex tensor.Tensor) tensor.Values {
	out := tensor.NewOfType(vec.DataType())
	errors.Log(ProjectOnMatrixColumnOut(mtx, vec, colindex, out))
	return out
}

// ProjectOnMatrixColumnOut is a convenience function for projecting given vector
// of values along a specific column (2nd dimension) of the given 2D matrix,
// specified by the scalar colindex, putting results into out.
// If the vec is more than 1 dimensional, then it is treated as rows x cells,
// and each row of cells is projected through the matrix column, producing a
// 1D output with the number of rows.  Otherwise a single number is produced.
// This is typically done with results from SVD or EigSym (PCA).
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
