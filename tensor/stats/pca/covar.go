// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package pca

import (
	"fmt"

	"cogentcore.org/core/tensor"
	"cogentcore.org/core/tensor/stats/metric"
	"cogentcore.org/core/tensor/table"
)

// CovarTableColumn generates a covariance matrix from given column name
// in given IndexView of an table.Table, and given metric function
// (typically Covariance or Correlation -- use Covar if vars have similar
// overall scaling, which is typical in neural network models, and use
// Correl if they are on very different scales -- Correl effectively rescales).
// A Covariance matrix computes the *row-wise* vector similarities for each
// pairwise combination of column cells -- i.e., the extent to which each
// cell co-varies in its value with each other cell across the rows of the table.
// This is the input to the PCA eigenvalue decomposition of the resulting
// covariance matrix.
func CovarTableColumn(cmat tensor.Tensor, ix *table.IndexView, column string, mfun metric.Func64) error {
	col, err := ix.Table.ColumnByName(column)
	if err != nil {
		return err
	}
	rows := ix.Len()
	nd := col.NumDims()
	if nd < 2 || rows == 0 {
		return fmt.Errorf("pca.CovarTableColumn: must have 2 or more dims and rows != 0")
	}
	ln := col.Len()
	sz := ln / col.DimSize(0) // size of cell

	cshp := []int{sz, sz}
	cmat.SetShape(cshp)

	av := make([]float64, rows)
	bv := make([]float64, rows)
	sdim := []int{0, 0}
	for ai := 0; ai < sz; ai++ {
		sdim[0] = ai
		TableColumnRowsVec(av, ix, col, ai)
		for bi := 0; bi <= ai; bi++ { // lower diag
			sdim[1] = bi
			TableColumnRowsVec(bv, ix, col, bi)
			cv := mfun(av, bv)
			cmat.SetFloat(sdim, cv)
		}
	}
	// now fill in upper diagonal with values from lower diagonal
	// note: assumes symmetric distance function
	fdim := []int{0, 0}
	for ai := 0; ai < sz; ai++ {
		sdim[0] = ai
		fdim[1] = ai
		for bi := ai + 1; bi < sz; bi++ { // upper diag
			fdim[0] = bi
			sdim[1] = bi
			cv := cmat.Float(fdim)
			cmat.SetFloat(sdim, cv)
		}
	}

	if nm, has := ix.Table.MetaData["name"]; has {
		cmat.SetMetaData("name", nm+"_"+column)
	} else {
		cmat.SetMetaData("name", column)
	}
	if ds, has := ix.Table.MetaData["desc"]; has {
		cmat.SetMetaData("desc", ds)
	}
	return nil
}

// CovarTensor generates a covariance matrix from given tensor.Tensor,
// where the outer-most dimension is rows, and all other dimensions within that
// are covaried against each other, using given metric function
// (typically Covariance or Correlation -- use Covar if vars have similar
// overall scaling, which is typical in neural network models, and use
// Correl if they are on very different scales -- Correl effectively rescales).
// A Covariance matrix computes the *row-wise* vector similarities for each
// pairwise combination of column cells -- i.e., the extent to which each
// cell co-varies in its value with each other cell across the rows of the table.
// This is the input to the PCA eigenvalue decomposition of the resulting
// covariance matrix.
func CovarTensor(cmat tensor.Tensor, tsr tensor.Tensor, mfun metric.Func64) error {
	rows := tsr.DimSize(0)
	nd := tsr.NumDims()
	if nd < 2 || rows == 0 {
		return fmt.Errorf("pca.CovarTensor: must have 2 or more dims and rows != 0")
	}
	ln := tsr.Len()
	sz := ln / rows

	cshp := []int{sz, sz}
	cmat.SetShape(cshp)

	av := make([]float64, rows)
	bv := make([]float64, rows)
	sdim := []int{0, 0}
	for ai := 0; ai < sz; ai++ {
		sdim[0] = ai
		TensorRowsVec(av, tsr, ai)
		for bi := 0; bi <= ai; bi++ { // lower diag
			sdim[1] = bi
			TensorRowsVec(bv, tsr, bi)
			cv := mfun(av, bv)
			cmat.SetFloat(sdim, cv)
		}
	}
	// now fill in upper diagonal with values from lower diagonal
	// note: assumes symmetric distance function
	fdim := []int{0, 0}
	for ai := 0; ai < sz; ai++ {
		sdim[0] = ai
		fdim[1] = ai
		for bi := ai + 1; bi < sz; bi++ { // upper diag
			fdim[0] = bi
			sdim[1] = bi
			cv := cmat.Float(fdim)
			cmat.SetFloat(sdim, cv)
		}
	}

	if nm, has := tsr.MetaData("name"); has {
		cmat.SetMetaData("name", nm+"Covar")
	} else {
		cmat.SetMetaData("name", "Covar")
	}
	if ds, has := tsr.MetaData("desc"); has {
		cmat.SetMetaData("desc", ds)
	}
	return nil
}

// TableColumnRowsVec extracts row-wise vector from given cell index into vec.
// vec must be of size ix.Len() -- number of rows
func TableColumnRowsVec(vec []float64, ix *table.IndexView, col tensor.Tensor, cidx int) {
	rows := ix.Len()
	ln := col.Len()
	sz := ln / col.DimSize(0) // size of cell
	for ri := 0; ri < rows; ri++ {
		coff := ix.Indexes[ri]*sz + cidx
		vec[ri] = col.Float1D(coff)
	}
}

// TensorRowsVec extracts row-wise vector from given cell index into vec.
// vec must be of size tsr.DimSize(0) -- number of rows
func TensorRowsVec(vec []float64, tsr tensor.Tensor, cidx int) {
	rows := tsr.DimSize(0)
	ln := tsr.Len()
	sz := ln / rows
	for ri := 0; ri < rows; ri++ {
		coff := ri*sz + cidx
		vec[ri] = tsr.Float1D(coff)
	}
}

// CovarTableColumnStd generates a covariance matrix from given column name
// in given IndexView of an table.Table, and given metric function
// (typically Covariance or Correlation -- use Covar if vars have similar
// overall scaling, which is typical in neural network models, and use
// Correl if they are on very different scales -- Correl effectively rescales).
// A Covariance matrix computes the *row-wise* vector similarities for each
// pairwise combination of column cells -- i.e., the extent to which each
// cell co-varies in its value with each other cell across the rows of the table.
// This is the input to the PCA eigenvalue decomposition of the resulting
// covariance matrix.
// This Std version is usable e.g., in Python where the func cannot be passed.
func CovarTableColumnStd(cmat tensor.Tensor, ix *table.IndexView, column string, met metric.StdMetrics) error {
	return CovarTableColumn(cmat, ix, column, metric.StdFunc64(met))
}

// CovarTensorStd generates a covariance matrix from given tensor.Tensor,
// where the outer-most dimension is rows, and all other dimensions within that
// are covaried against each other, using given metric function
// (typically Covariance or Correlation -- use Covar if vars have similar
// overall scaling, which is typical in neural network models, and use
// Correl if they are on very different scales -- Correl effectively rescales).
// A Covariance matrix computes the *row-wise* vector similarities for each
// pairwise combination of column cells -- i.e., the extent to which each
// cell co-varies in its value with each other cell across the rows of the table.
// This is the input to the PCA eigenvalue decomposition of the resulting
// covariance matrix.
// This Std version is usable e.g., in Python where the func cannot be passed.
func CovarTensorStd(cmat tensor.Tensor, tsr tensor.Tensor, met metric.StdMetrics) error {
	return CovarTensor(cmat, tsr, metric.StdFunc64(met))
}
