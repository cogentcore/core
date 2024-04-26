// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package simat

import (
	"fmt"

	"cogentcore.org/core/tensor"
	"cogentcore.org/core/tensor/stats/metric"
)

// Tensor computes a similarity / distance matrix on tensor
// using given metric function.   Outer-most dimension ("rows") is
// used as "indexical" dimension and all other dimensions within that
// are compared.
// Results go in smat which is ensured to have proper square 2D shape
// (rows * rows).
func Tensor(smat tensor.Tensor, a tensor.Tensor, mfun metric.Func64) error {
	rows := a.DimSize(0)
	nd := a.NumDims()
	if nd < 2 || rows == 0 {
		return fmt.Errorf("simat.Tensor: must have 2 or more dims and rows != 0")
	}
	ln := a.Len()
	sz := ln / rows

	sshp := []int{rows, rows}
	smat.SetShape(sshp)

	av := make([]float64, sz)
	bv := make([]float64, sz)
	ardim := []int{0}
	brdim := []int{0}
	sdim := []int{0, 0}
	for ai := 0; ai < rows; ai++ {
		ardim[0] = ai
		sdim[0] = ai
		ar := a.SubSpace(ardim)
		ar.Floats(&av)
		for bi := 0; bi <= ai; bi++ { // lower diag
			brdim[0] = bi
			sdim[1] = bi
			br := a.SubSpace(brdim)
			br.Floats(&bv)
			sv := mfun(av, bv)
			smat.SetFloat(sdim, sv)
		}
	}
	// now fill in upper diagonal with values from lower diagonal
	// note: assumes symmetric distance function
	fdim := []int{0, 0}
	for ai := 0; ai < rows; ai++ {
		sdim[0] = ai
		fdim[1] = ai
		for bi := ai + 1; bi < rows; bi++ { // upper diag
			fdim[0] = bi
			sdim[1] = bi
			sv := smat.Float(fdim)
			smat.SetFloat(sdim, sv)
		}
	}
	return nil
}

// Tensors computes a similarity / distance matrix on two tensors
// using given metric function.   Outer-most dimension ("rows") is
// used as "indexical" dimension and all other dimensions within that
// are compared.  Resulting reduced 2D shape of two tensors must be
// the same (returns error if not).
// Rows of smat = a, cols = b
func Tensors(smat tensor.Tensor, a, b tensor.Tensor, mfun metric.Func64) error {
	arows := a.DimSize(0)
	and := a.NumDims()
	brows := b.DimSize(0)
	bnd := b.NumDims()
	if and < 2 || bnd < 2 || arows == 0 || brows == 0 {
		return fmt.Errorf("simat.Tensors: must have 2 or more dims and rows != 0")
	}
	alen := a.Len()
	asz := alen / arows
	blen := b.Len()
	bsz := blen / brows
	if asz != bsz {
		return fmt.Errorf("simat.Tensors: size of inner dimensions must be same")
	}

	sshp := []int{arows, brows}
	smat.SetShape(sshp, "a", "b")

	av := make([]float64, asz)
	bv := make([]float64, bsz)
	ardim := []int{0}
	brdim := []int{0}
	sdim := []int{0, 0}
	for ai := 0; ai < arows; ai++ {
		ardim[0] = ai
		sdim[0] = ai
		ar := a.SubSpace(ardim)
		ar.Floats(&av)
		for bi := 0; bi < brows; bi++ {
			brdim[0] = bi
			sdim[1] = bi
			br := b.SubSpace(brdim)
			br.Floats(&bv)
			sv := mfun(av, bv)
			smat.SetFloat(sdim, sv)
		}
	}
	return nil
}

// TensorStd computes a similarity / distance matrix on tensor
// using given Std metric function.   Outer-most dimension ("rows") is
// used as "indexical" dimension and all other dimensions within that
// are compared.
// Results go in smat which is ensured to have proper square 2D shape
// (rows * rows).
// This Std version is usable e.g., in Python where the func cannot be passed.
func TensorStd(smat tensor.Tensor, a tensor.Tensor, met metric.StdMetrics) error {
	return Tensor(smat, a, metric.StdFunc64(met))
}

// TensorsStd computes a similarity / distance matrix on two tensors
// using given Std metric function.   Outer-most dimension ("rows") is
// used as "indexical" dimension and all other dimensions within that
// are compared.  Resulting reduced 2D shape of two tensors must be
// the same (returns error if not).
// Rows of smat = a, cols = b
// This Std version is usable e.g., in Python where the func cannot be passed.
func TensorsStd(smat tensor.Tensor, a, b tensor.Tensor, met metric.StdMetrics) error {
	return Tensors(smat, a, b, metric.StdFunc64(met))
}
