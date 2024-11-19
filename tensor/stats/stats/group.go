// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package stats

import (
	"strconv"
	"strings"

	"cogentcore.org/core/base/metadata"
	"cogentcore.org/core/tensor"
	"cogentcore.org/core/tensor/table"
	"cogentcore.org/core/tensor/tensorfs"
)

// Groups generates indexes for each unique value in each of the given tensors.
// One can then use the resulting indexes for the [tensor.Rows] indexes to
// perform computations restricted to grouped subsets of data, as in the
// [GroupStats] function. See [GroupCombined] for function that makes a
// "Combined" Group that has a unique group for each _combination_ of
// the separate, independent groups created by this function.
// It creates subdirectories in a "Groups" directory within given [tensorfs],
// for each tensor passed in here, using the metadata Name property for
// names (index if empty).
// Within each subdirectory there are int tensors for each unique 1D
// row-wise value of elements in the input tensor, named as the string
// representation of the value, where the int tensor contains a list of
// row-wise indexes corresponding to the source rows having that value.
// Note that these indexes are directly in terms of the underlying [Tensor] data
// rows, indirected through any existing indexes on the inputs, so that
// the results can be used directly as Indexes into the corresponding tensor data.
// Uses a stable sort on columns, so ordering of other dimensions is preserved.
func Groups(dir *tensorfs.Node, tsrs ...tensor.Tensor) error {
	gd := dir.RecycleDir("Groups")
	makeIdxs := func(dir *tensorfs.Node, srt *tensor.Rows, val string, start, r int) {
		n := r - start
		it := tensorfs.Value[int](dir, val, n)
		for j := range n {
			it.SetIntRow(srt.Indexes[start+j], j, 0) // key to indirect through sort indexes
		}
	}

	for i, tsr := range tsrs {
		nr := tsr.DimSize(0)
		if nr == 0 {
			continue
		}
		nm := metadata.Name(tsr)
		if nm == "" {
			nm = strconv.Itoa(i)
		}
		td, _ := gd.Mkdir(nm)
		srt := tensor.AsRows(tsr).CloneIndexes()
		srt.SortStable(tensor.Ascending)
		start := 0
		if tsr.IsString() {
			lastVal := srt.StringRow(0, 0)
			for r := range nr {
				v := srt.StringRow(r, 0)
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
			lastVal := srt.FloatRow(0, 0)
			for r := range nr {
				v := srt.FloatRow(r, 0)
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
	return nil
}

// TableGroups runs [Groups] on the given columns from given [table.Table].
func TableGroups(dir *tensorfs.Node, dt *table.Table, columns ...string) error {
	dv := table.NewView(dt)
	// important for consistency across columns, to do full outer product sort first.
	dv.SortColumns(tensor.Ascending, tensor.StableSort, columns...)
	return Groups(dir, dv.ColumnList(columns...)...)
}

// GroupAll copies all indexes from the first given tensor,
// into an "All/All" tensor in the given [tensorfs], which can then
// be used with [GroupStats] to generate summary statistics across
// all the data. See [Groups] for more general documentation.
func GroupAll(dir *tensorfs.Node, tsrs ...tensor.Tensor) error {
	gd := dir.RecycleDir("Groups")
	tsr := tensor.AsRows(tsrs[0])
	nr := tsr.NumRows()
	if nr == 0 {
		return nil
	}
	td, _ := gd.Mkdir("All")
	it := tensorfs.Value[int](td, "All", nr)
	for j := range nr {
		it.SetIntRow(tsr.RowIndex(j), j, 0) // key to indirect through any existing indexes
	}
	return nil
}

// todo: GroupCombined

// GroupStats computes the given stats function on the unique grouped indexes
// produced by the [Groups] function, in the given [tensorfs] directory,
// applied to each of the tensors passed here.
// It creates a "Stats" subdirectory in given directory, with
// subdirectories with the name of each value tensor (if it does not
// yet exist), and then creates a subdirectory within that
// for the statistic name. Within that statistic directory, it creates
// a String tensor with the unique values of each source [Groups] tensor,
// and a aligned Float64 tensor with the statistics results for each such
// unique group value. See the README.md file for a diagram of the results.
func GroupStats(dir *tensorfs.Node, stat Stats, tsrs ...tensor.Tensor) error {
	gd := dir.RecycleDir("Groups")
	sd := dir.RecycleDir("Stats")
	stnm := StripPackage(stat.String())
	groups, _ := gd.Nodes()
	for _, gp := range groups {
		gpnm := gp.Name()
		ggd := gd.RecycleDir(gpnm)
		vals := ggd.ValuesFunc(nil)
		nv := len(vals)
		if nv == 0 {
			continue
		}
		sgd := sd.RecycleDir(gpnm)
		gv := sgd.Node(gpnm)
		if gv == nil {
			gtsr := tensorfs.Value[string](sgd, gpnm, nv)
			for i, v := range vals {
				gtsr.SetStringRow(metadata.Name(v), i, 0)
			}
		}
		for _, tsr := range tsrs {
			vd := sgd.RecycleDir(metadata.Name(tsr))
			sv := tensorfs.Value[float64](vd, stnm, nv)
			for i, v := range vals {
				idx := tensor.AsIntSlice(v)
				sg := tensor.NewRows(tsr.AsValues(), idx...)
				stout := stat.Call(sg)
				sv.SetFloatRow(stout.Float1D(0), i, 0)
			}
		}
	}
	return nil
}

// TableGroupStats runs [GroupStats] using standard [Stats]
// on the given columns from given [table.Table].
func TableGroupStats(dir *tensorfs.Node, stat Stats, dt *table.Table, columns ...string) error {
	return GroupStats(dir, stat, dt.ColumnList(columns...)...)
}

// GroupDescribe runs standard descriptive statistics on given tensor data
// using [GroupStats] function, with [DescriptiveStats] list of stats.
func GroupDescribe(dir *tensorfs.Node, tsrs ...tensor.Tensor) error {
	for _, st := range DescriptiveStats {
		err := GroupStats(dir, st, tsrs...)
		if err != nil {
			return err
		}
	}
	return nil
}

// TableGroupDescribe runs [GroupDescribe] on the given columns from given [table.Table].
func TableGroupDescribe(dir *tensorfs.Node, dt *table.Table, columns ...string) error {
	return GroupDescribe(dir, dt.ColumnList(columns...)...)
}

// GroupStatsAsTable returns the results from [GroupStats] in given directory
// as a [table.Table], using [tensorfs.DirTable] function.
func GroupStatsAsTable(dir *tensorfs.Node) *table.Table {
	return tensorfs.DirTable(dir.Node("Stats"), nil)
}

// GroupStatsAsTableNoStatName returns the results from [GroupStats]
// in given directory as a [table.Table], using [tensorfs.DirTable] function.
// Column names are updated to not include the stat name, if there is only
// one statistic such that the resulting name will still be unique.
// Otherwise, column names are Value/Stat.
func GroupStatsAsTableNoStatName(dir *tensorfs.Node) *table.Table {
	dt := tensorfs.DirTable(dir.Node("Stats"), nil)
	cols := make(map[string]string)
	for _, nm := range dt.Columns.Keys {
		vn := nm
		si := strings.Index(nm, "/")
		if si > 0 {
			vn = nm[:si]
		}
		if _, exists := cols[vn]; exists {
			continue
		}
		cols[vn] = nm
	}
	for k, v := range cols {
		ci := dt.Columns.IndexByKey(v)
		dt.Columns.RenameIndex(ci, k)
	}
	return dt
}
