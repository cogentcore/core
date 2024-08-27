// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package split

//go:generate core generate

import (
	"log"
	"slices"

	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/tensor/table"
)

// All returns a single "split" with all of the rows in given view
// useful for leveraging the aggregation management functions in splits
func All(ix *table.IndexView) *table.Splits {
	spl := &table.Splits{}
	spl.Levels = []string{"All"}
	spl.New(ix.Table, []string{"All"}, ix.Indexes...)
	return spl
}

// GroupByIndex returns a new Splits set based on the groups of values
// across the given set of column indexes.
// Uses a stable sort on columns, so ordering of other dimensions is preserved.
func GroupByIndex(ix *table.IndexView, colIndexes []int) *table.Splits {
	nc := len(colIndexes)
	if nc == 0 || ix.Table == nil {
		return nil
	}
	if ix.Table.ColumnNames == nil {
		log.Println("split.GroupBy: Table does not have any column names -- will not work")
		return nil
	}
	spl := &table.Splits{}
	spl.Levels = make([]string, nc)
	for i, ci := range colIndexes {
		spl.Levels[i] = ix.Table.ColumnNames[ci]
	}
	srt := ix.Clone()
	srt.SortStableColumns(colIndexes, true) // important for consistency
	lstValues := make([]string, nc)
	curValues := make([]string, nc)
	var curIx *table.IndexView
	for _, rw := range srt.Indexes {
		diff := false
		for i, ci := range colIndexes {
			cl := ix.Table.Columns[ci]
			cv := cl.String1D(rw)
			curValues[i] = cv
			if cv != lstValues[i] {
				diff = true
			}
		}
		if diff || curIx == nil {
			curIx = spl.New(ix.Table, curValues, rw)
			copy(lstValues, curValues)
		} else {
			curIx.AddIndex(rw)
		}
	}
	if spl.Len() == 0 { // prevent crashing from subsequent ops: add an empty split
		spl.New(ix.Table, curValues) // no rows added here
	}
	return spl
}

// GroupBy returns a new Splits set based on the groups of values
// across the given set of column names.
// Uses a stable sort on columns, so ordering of other dimensions is preserved.
func GroupBy(ix *table.IndexView, columns ...string) *table.Splits {
	return GroupByIndex(ix, errors.Log1(ix.Table.ColumnIndexesByNames(columns...)))
}

// GroupByFunc returns a new Splits set based on the given function
// which returns value(s) to group on for each row of the table.
// The function should always return the same number of values -- if
// it doesn't behavior is undefined.
// Uses a stable sort on columns, so ordering of other dimensions is preserved.
func GroupByFunc(ix *table.IndexView, fun func(row int) []string) *table.Splits {
	if ix.Table == nil {
		return nil
	}

	// save function values
	funvals := make(map[int][]string, ix.Len())
	nv := 0 // number of valeus
	for _, rw := range ix.Indexes {
		sv := fun(rw)
		if nv == 0 {
			nv = len(sv)
		}
		funvals[rw] = slices.Clone(sv)
	}

	srt := ix.Clone()
	srt.SortStable(func(et *table.Table, i, j int) bool { // sort based on given function values
		fvi := funvals[i]
		fvj := funvals[j]
		for fi := 0; fi < nv; fi++ {
			if fvi[fi] < fvj[fi] {
				return true
			} else if fvi[fi] > fvj[fi] {
				return false
			}
		}
		return false
	})

	// now do our usual grouping operation
	spl := &table.Splits{}
	lstValues := make([]string, nv)
	var curIx *table.IndexView
	for _, rw := range srt.Indexes {
		curValues := funvals[rw]
		diff := (curIx == nil)
		if !diff {
			for fi := 0; fi < nv; fi++ {
				if lstValues[fi] != curValues[fi] {
					diff = true
					break
				}
			}
		}
		if diff {
			curIx = spl.New(ix.Table, curValues, rw)
			copy(lstValues, curValues)
		} else {
			curIx.AddIndex(rw)
		}
	}
	return spl
}
