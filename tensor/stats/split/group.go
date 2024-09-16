// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package split

//go:generate core generate

import (
	"strconv"

	"cogentcore.org/core/tensor"
	"cogentcore.org/core/tensor/datafs"
	"cogentcore.org/core/tensor/table"
)

// All returns a single "split" with all of the rows in given view
// useful for leveraging the aggregation management functions in splits
// func All(ix *table.Table) *table.Splits {
// 	spl := &table.Splits{}
// 	spl.Levels = []string{"All"}
// 	spl.New(ix.Table, []string{"All"}, ix.Indexes...)
// 	return spl
// }

// TableGroups does [Groups] on given columns from table.
func TableGroups(dir *datafs.Data, dt *table.Table, columns ...string) {
	dv := table.NewView(dt)
	// important for consistency across columns, to do full outer product sort first.
	dv.SortColumns(tensor.Ascending, tensor.Stable, columns...)
	Groups(dir, dv.ColumnList(columns...)...)
}

// Groups generates indexes for each unique value in each of the given tensors.
// One can then use the resulting indexes for the [tensor.Indexed] indexes to
// perform computations restricted to grouped subsets of data, as in the
// [Stats] function.
// It creates subdirectories in given [datafs] for each tensor
// passed in here, using the metadata Name property for names (index if empty).
// Within each subdirectory there are int value tensors for each unique 1D
// row-wise value of elements in the input tensor, named as the string
// representation of the value, where the int tensor contains a list of
// row-wise indexes corresponding to the source rows having that value.
// Note that these indexes are directly in terms of the underlying [Tensor] data
// rows, indirected through any existing indexes on the inputs, so that
// the results can be used directly as indexes into the corresponding tensor data.
// Uses a stable sort on columns, so ordering of other dimensions is preserved.
func Groups(dir *datafs.Data, tsrs ...*tensor.Indexed) {

	makeIdxs := func(dir *datafs.Data, srt *tensor.Indexed, val string, start, r int) {
		n := r - start
		it := datafs.NewValue[int](dir, val, n)
		for j := range n {
			it.SetIntRowCell(srt.Indexes[start+j], j, 0) // key to indirect through sort indexes
		}
	}

	for i, tsr := range tsrs {
		nr := tsr.NumRows()
		if nr == 0 {
			continue
		}
		nm := tsr.Tensor.Metadata().GetName()
		if nm == "" {
			nm = strconv.Itoa(i)
		}
		td, _ := dir.Mkdir(nm)
		srt := tsr.CloneIndexes()
		srt.SortStable(tensor.Ascending)
		start := 0
		if tsr.Tensor.IsString() {
			lastVal := srt.StringRowCell(0, 0)
			for r := range nr {
				v := srt.StringRowCell(r, 0)
				if v != lastVal {
					makeIdxs(td, srt, lastVal, start, r)
					start = r
					lastVal = v
				}
			}
			if start != nr-1 {
				makeIdxs(td, srt, lastVal, start, nr)
			}
		} else {
			lastVal := srt.FloatRowCell(0, 0)
			for r := range nr {
				v := srt.FloatRowCell(r, 0)
				if v != lastVal {
					makeIdxs(td, srt, tensor.Float64ToString(lastVal), start, r)
					start = r
					lastVal = v
				}
			}
			if start != nr-1 {
				makeIdxs(td, srt, tensor.Float64ToString(lastVal), start, nr)
			}
		}
	}
}

// todo: make an outer-product function?
