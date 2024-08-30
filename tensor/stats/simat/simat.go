// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package simat

import (
	"fmt"

	"cogentcore.org/core/tensor"
	"cogentcore.org/core/tensor/stats/metric"
	"cogentcore.org/core/tensor/table"
)

// SimMat is a similarity / distance matrix with additional row and column
// labels for display purposes.
type SimMat struct {

	// the similarity / distance matrix (typically an tensor.Float64)
	Mat tensor.Tensor

	// labels for the rows -- blank rows trigger generation of grouping lines
	Rows []string

	// labels for the columns -- blank columns trigger generation of grouping lines
	Columns []string
}

// NewSimMat returns a new SimMat similarity matrix
func NewSimMat() *SimMat {
	return &SimMat{}
}

// Init initializes SimMat with default Matrix and nil rows, cols
func (smat *SimMat) Init() {
	smat.Mat = &tensor.Float64{}
	smat.Mat.SetMetaData("grid-fill", "1") // best for sim mats -- can override later if need to
	smat.Rows = nil
	smat.Columns = nil
}

// TableColumnStd generates a similarity / distance matrix from given column name
// in given IndexView of an table.Table, and given standard metric function.
// if labNm is not empty, uses given column name for labels, which if blankRepeat
// is true are filtered so that any sequentially repeated labels are blank.
// This Std version is usable e.g., in Python where the func cannot be passed.
func (smat *SimMat) TableColumnStd(ix *table.IndexView, column, labNm string, blankRepeat bool, met metric.StdMetrics) error {
	return smat.TableColumn(ix, column, labNm, blankRepeat, metric.StdFunc64(met))
}

// TableColumn generates a similarity / distance matrix from given column name
// in given IndexView of an table.Table, and given metric function.
// if labNm is not empty, uses given column name for labels, which if blankRepeat
// is true are filtered so that any sequentially repeated labels are blank.
func (smat *SimMat) TableColumn(ix *table.IndexView, column, labNm string, blankRepeat bool, mfun metric.Func64) error {
	col, err := ix.Table.ColumnByName(column)
	if err != nil {
		return err
	}
	smat.Init()
	sm := smat.Mat

	rows := ix.Len()
	nd := col.NumDims()
	if nd < 2 || rows == 0 {
		return fmt.Errorf("simat.Tensor: must have 2 or more dims and rows != 0")
	}
	ln := col.Len()
	sz := ln / col.DimSize(0) // size of cell

	sshp := []int{rows, rows}
	sm.SetShape(sshp)

	av := make([]float64, sz)
	bv := make([]float64, sz)
	ardim := []int{0}
	brdim := []int{0}
	sdim := []int{0, 0}
	for ai := 0; ai < rows; ai++ {
		ardim[0] = ix.Indexes[ai]
		sdim[0] = ai
		ar := col.SubSpace(ardim)
		ar.Floats(&av)
		for bi := 0; bi <= ai; bi++ { // lower diag
			brdim[0] = ix.Indexes[bi]
			sdim[1] = bi
			br := col.SubSpace(brdim)
			br.Floats(&bv)
			sv := mfun(av, bv)
			sm.SetFloat(sdim, sv)
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
			sv := sm.Float(fdim)
			sm.SetFloat(sdim, sv)
		}
	}

	if nm, has := ix.Table.MetaData["name"]; has {
		sm.SetMetaData("name", nm+"_"+column)
	} else {
		sm.SetMetaData("name", column)
	}
	if ds, has := ix.Table.MetaData["desc"]; has {
		sm.SetMetaData("desc", ds)
	}

	if labNm == "" {
		return nil
	}
	lc, err := ix.Table.ColumnByName(labNm)
	if err != nil {
		return err
	}
	smat.Rows = make([]string, rows)
	last := ""
	for r := 0; r < rows; r++ {
		lbl := lc.String1D(ix.Indexes[r])
		if blankRepeat && lbl == last {
			continue
		}
		smat.Rows[r] = lbl
		last = lbl
	}
	smat.Columns = smat.Rows // identical
	return nil
}

// BlankRepeat returns string slice with any sequentially repeated strings blanked out
func BlankRepeat(str []string) []string {
	sz := len(str)
	br := make([]string, sz)
	last := ""
	for r, s := range str {
		if s == last {
			continue
		}
		br[r] = s
		last = s
	}
	return br
}
