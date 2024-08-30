// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package pca

import (
	"fmt"

	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/tensor"
	"cogentcore.org/core/tensor/stats/metric"
	"cogentcore.org/core/tensor/table"
	"gonum.org/v1/gonum/mat"
)

// SVD computes the eigenvalue decomposition of a square similarity matrix,
// typically generated using the correlation metric.
type SVD struct {

	// type of SVD to run: SVDNone is the most efficient if you only need the values which are always computed.  Otherwise, SVDThin is the next most efficient for getting approximate vectors
	Kind mat.SVDKind

	// condition value -- minimum normalized eigenvalue to return in values
	Cond float64 `default:"0.01"`

	// the rank (count) of singular values greater than Cond
	Rank int

	// the covariance matrix computed on original data, which is then eigen-factored
	Covar tensor.Tensor `display:"no-inline"`

	// the eigenvectors, in same size as Covar - each eigenvector is a column in this 2D square matrix, ordered *lowest* to *highest* across the columns -- i.e., maximum eigenvector is the last column
	Vectors tensor.Tensor `display:"no-inline"`

	// the eigenvalues, ordered *lowest* to *highest*
	Values []float64 `display:"no-inline"`
}

func (svd *SVD) Init() {
	svd.Kind = mat.SVDNone
	svd.Cond = 0.01
	svd.Covar = &tensor.Float64{}
	svd.Vectors = &tensor.Float64{}
	svd.Values = nil
}

// TableColumn is a convenience method that computes a covariance matrix
// on given column of table and then performs the SVD on the resulting matrix.
// If no error occurs, the results can be read out from Vectors and Values
// or used in Projection methods.
// mfun is metric function, typically Covariance or Correlation -- use Covar
// if vars have similar overall scaling, which is typical in neural network models,
// and use Correl if they are on very different scales -- Correl effectively rescales).
// A Covariance matrix computes the *row-wise* vector similarities for each
// pairwise combination of column cells -- i.e., the extent to which each
// cell co-varies in its value with each other cell across the rows of the table.
// This is the input to the SVD eigenvalue decomposition of the resulting
// covariance matrix, which extracts the eigenvectors as directions with maximal
// variance in this matrix.
func (svd *SVD) TableColumn(ix *table.IndexView, column string, mfun metric.Func64) error {
	if svd.Covar == nil {
		svd.Init()
	}
	err := CovarTableColumn(svd.Covar, ix, column, mfun)
	if err != nil {
		return err
	}
	return svd.SVD()
}

// Tensor is a convenience method that computes a covariance matrix
// on given tensor and then performs the SVD on the resulting matrix.
// If no error occurs, the results can be read out from Vectors and Values
// or used in Projection methods.
// mfun is metric function, typically Covariance or Correlation -- use Covar
// if vars have similar overall scaling, which is typical in neural network models,
// and use Correl if they are on very different scales -- Correl effectively rescales).
// A Covariance matrix computes the *row-wise* vector similarities for each
// pairwise combination of column cells -- i.e., the extent to which each
// cell co-varies in its value with each other cell across the rows of the table.
// This is the input to the SVD eigenvalue decomposition of the resulting
// covariance matrix, which extracts the eigenvectors as directions with maximal
// variance in this matrix.
func (svd *SVD) Tensor(tsr tensor.Tensor, mfun metric.Func64) error {
	if svd.Covar == nil {
		svd.Init()
	}
	err := CovarTensor(svd.Covar, tsr, mfun)
	if err != nil {
		return err
	}
	return svd.SVD()
}

// TableColumnStd is a convenience method that computes a covariance matrix
// on given column of table and then performs the SVD on the resulting matrix.
// If no error occurs, the results can be read out from Vectors and Values
// or used in Projection methods.
// mfun is a Std metric function, typically Covariance or Correlation -- use Covar
// if vars have similar overall scaling, which is typical in neural network models,
// and use Correl if they are on very different scales -- Correl effectively rescales).
// A Covariance matrix computes the *row-wise* vector similarities for each
// pairwise combination of column cells -- i.e., the extent to which each
// cell co-varies in its value with each other cell across the rows of the table.
// This is the input to the SVD eigenvalue decomposition of the resulting
// covariance matrix, which extracts the eigenvectors as directions with maximal
// variance in this matrix.
// This Std version is usable e.g., in Python where the func cannot be passed.
func (svd *SVD) TableColumnStd(ix *table.IndexView, column string, met metric.StdMetrics) error {
	return svd.TableColumn(ix, column, metric.StdFunc64(met))
}

// TensorStd is a convenience method that computes a covariance matrix
// on given tensor and then performs the SVD on the resulting matrix.
// If no error occurs, the results can be read out from Vectors and Values
// or used in Projection methods.
// mfun is Std metric function, typically Covariance or Correlation -- use Covar
// if vars have similar overall scaling, which is typical in neural network models,
// and use Correl if they are on very different scales -- Correl effectively rescales).
// A Covariance matrix computes the *row-wise* vector similarities for each
// pairwise combination of column cells -- i.e., the extent to which each
// cell co-varies in its value with each other cell across the rows of the table.
// This is the input to the SVD eigenvalue decomposition of the resulting
// covariance matrix, which extracts the eigenvectors as directions with maximal
// variance in this matrix.
// This Std version is usable e.g., in Python where the func cannot be passed.
func (svd *SVD) TensorStd(tsr tensor.Tensor, met metric.StdMetrics) error {
	return svd.Tensor(tsr, metric.StdFunc64(met))
}

// SVD performs the eigen decomposition of the existing Covar matrix.
// Vectors and Values fields contain the results.
func (svd *SVD) SVD() error {
	if svd.Covar == nil || svd.Covar.NumDims() != 2 {
		return fmt.Errorf("svd.SVD: Covar matrix is nil or not 2D")
	}
	var eig mat.SVD
	// note: MUST be a Float64 otherwise doesn't have Symmetric function
	ok := eig.Factorize(svd.Covar, svd.Kind)
	if !ok {
		return fmt.Errorf("gonum SVD Factorize failed")
	}
	if svd.Kind > mat.SVDNone {
		if svd.Vectors == nil {
			svd.Vectors = &tensor.Float64{}
		}
		var ev mat.Dense
		eig.UTo(&ev)
		tensor.CopyDense(svd.Vectors, &ev)
	}
	nr := svd.Covar.DimSize(0)
	if len(svd.Values) != nr {
		svd.Values = make([]float64, nr)
	}
	eig.Values(svd.Values)
	svd.Rank = eig.Rank(svd.Cond)
	return nil
}

// ProjectColumn projects values from the given column of given table (via IndexView)
// onto the idx'th eigenvector (0 = largest eigenvalue, 1 = next, etc).
// Must have already called SVD() method.
func (svd *SVD) ProjectColumn(vals *[]float64, ix *table.IndexView, column string, idx int) error {
	col, err := ix.Table.ColumnByName(column)
	if err != nil {
		return err
	}
	if svd.Vectors == nil || svd.Vectors.Len() == 0 {
		return fmt.Errorf("SVD.ProjectColumn Vectors are nil: must call SVD first, with Kind = mat.SVDFull so that the vectors are returned")
	}
	nr := svd.Vectors.DimSize(0)
	if idx >= nr {
		return fmt.Errorf("SVD.ProjectColumn eigenvector index > rank of matrix")
	}
	cvec := make([]float64, nr)
	// eidx := nr - 1 - idx // eigens in reverse order
	vec := svd.Vectors.(*tensor.Float64)
	for ri := 0; ri < nr; ri++ {
		cvec[ri] = vec.Value([]int{ri, idx}) // vecs are in columns, reverse magnitude order
	}
	rows := ix.Len()
	if len(*vals) != rows {
		*vals = make([]float64, rows)
	}
	ln := col.Len()
	sz := ln / col.DimSize(0) // size of cell
	if sz != nr {
		return fmt.Errorf("SVD.ProjectColumn column cell size != svd eigenvectors")
	}
	rdim := []int{0}
	for row := 0; row < rows; row++ {
		sum := 0.0
		rdim[0] = ix.Indexes[row]
		rt := col.SubSpace(rdim)
		for ci := 0; ci < sz; ci++ {
			sum += cvec[ci] * rt.Float1D(ci)
		}
		(*vals)[row] = sum
	}
	return nil
}

// ProjectColumnToTable projects values from the given column of given table (via IndexView)
// onto the given set of eigenvectors (idxs, 0 = largest eigenvalue, 1 = next, etc),
// and stores results along with labels from column labNm into results table.
// Must have already called SVD() method.
func (svd *SVD) ProjectColumnToTable(projections *table.Table, ix *table.IndexView, column, labNm string, idxs []int) error {
	_, err := ix.Table.ColumnByName(column)
	if errors.Log(err) != nil {
		return err
	}
	if svd.Vectors == nil {
		return fmt.Errorf("SVD.ProjectColumn Vectors are nil -- must call SVD first")
	}
	rows := ix.Len()
	projections.DeleteAll()
	pcolSt := 0
	if labNm != "" {
		projections.AddStringColumn(labNm)
		pcolSt = 1
	}
	for _, idx := range idxs {
		projections.AddFloat64Column(fmt.Sprintf("Projection%v", idx))
	}
	projections.SetNumRows(rows)

	for ii, idx := range idxs {
		pcol := projections.Columns[pcolSt+ii].(*tensor.Float64)
		svd.ProjectColumn(&pcol.Values, ix, column, idx)
	}

	if labNm != "" {
		lcol, err := ix.Table.ColumnByName(labNm)
		if errors.Log(err) == nil {
			plcol := projections.Columns[0]
			for row := 0; row < rows; row++ {
				plcol.SetString1D(row, lcol.String1D(row))
			}
		}
	}
	return nil
}
