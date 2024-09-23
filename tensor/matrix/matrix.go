// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package matrix

import (
	"errors"

	"cogentcore.org/core/tensor"
	"gonum.org/v1/gonum/mat"
)

// Matrix provides a view of the given [tensor.Tensor] as a [gonum]
// [mat.Matrix] interface type.
type Matrix struct {
	Tensor tensor.Tensor
}

// NewMatrix returns given [tensor.Tensor] as a [gonum] [mat.Matrix].
// It returns an error if the tensor is not 2D.
func NewMatrix(tsr tensor.Tensor) (*Matrix, error) {
	if tsr.IsString() {
		err := errors.New("matrix.NewMatrix: tensor has string values; must be numeric")
		return nil, err
	}
	nd := tsr.NumDims()
	if nd != 2 {
		err := errors.New("matrix.NewMatrix: tensor is not 2D")
		return nil, err
	}
	return &Matrix{Tensor: tsr}, nil
}

// Dims is the gonum/mat.Matrix interface method for returning the
// dimension sizes of the 2D Matrix.  Assumes Row-major ordering.
func (mx *Matrix) Dims() (r, c int) {
	return mx.Tensor.DimSize(0), mx.Tensor.DimSize(1)
}

// At is the gonum/mat.Matrix interface method for returning 2D
// matrix element at given row, column index. Assumes Row-major ordering.
func (mx *Matrix) At(i, j int) float64 {
	return mx.Tensor.Float(i, j)
}

// T is the gonum/mat.Matrix transpose method.
// It performs an implicit transpose by returning the receiver inside a Transpose.
func (mx *Matrix) T() mat.Matrix {
	return mat.Transpose{mx}
}

/////////////////////////  Symmetric

// Symmetric provides a view of the given [tensor.Tensor] as a [gonum]
// [mat.Symmetric] matrix interface type.
type Symmetric struct {
	Matrix
}

// NewSymmetric returns given [tensor.Tensor] as a [gonum] [mat.Symmetric] matrix.
// It returns an error if the tensor is not 2D or not symmetric.
func NewSymmetric(tsr tensor.Tensor) (*Symmetric, error) {
	if tsr.IsString() {
		err := errors.New("matrix.NewSymmetric: tensor has string values; must be numeric")
		return nil, err
	}
	nd := tsr.NumDims()
	if nd != 2 {
		err := errors.New("matrix.NewSymmetric: tensor is not 2D")
		return nil, err
	}
	if tsr.DimSize(0) != tsr.DimSize(1) {
		err := errors.New("matrix.NewSymmetric: tensor is not symmetric")
		return nil, err
	}
	sy := &Symmetric{}
	sy.Tensor = tsr
	return sy, nil
}

// SymmetricDim is the gonum/mat.Matrix interface method for returning the
// dimensionality of a symmetric 2D Matrix.
func (sy *Symmetric) SymmetricDim() (r int) {
	return sy.Tensor.DimSize(0)
}

// CopyDense copies a gonum mat.Dense matrix into given Tensor
// using standard Float64 interface
func CopyDense(to tensor.Tensor, dm *mat.Dense) {
	nr, nc := dm.Dims()
	tensor.SetShapeSizesMustBeValues(to, nr, nc)
	idx := 0
	for ri := 0; ri < nr; ri++ {
		for ci := 0; ci < nc; ci++ {
			v := dm.At(ri, ci)
			to.SetFloat1D(v, idx)
			idx++
		}
	}
}
