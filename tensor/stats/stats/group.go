// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package stats

import (
	"strconv"
	"strings"

	"cogentcore.org/core/tensor"
	"cogentcore.org/core/tensor/datafs"
	"cogentcore.org/core/tensor/table"
)

// note: we cannot register these functions because they take vararg tensors!!
//
// func init() {
// 	tensor.AddFunc("stats.Groups", Groups, 0, tensor.AnyFirstArg)
// 	tensor.AddFunc("stats.GroupAll", GroupAll, 0, tensor.AnyFirstArg)
// }

// Groups generates indexes for each unique value in each of the given tensors.
// One can then use the resulting indexes for the [tensor.Rows] indexes to
// perform computations restricted to grouped subsets of data, as in the
// [GroupStats] function. See [GroupCombined] for function that makes a
// "Combined" Group that has a unique group for each _combination_ of
// the separate, independent groups created by this function.
// It creates subdirectories in a "Groups" directory within given [datafs],
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
func Groups(dir *datafs.Data, tsrs ...tensor.Tensor) error {
	gd, err := dir.RecycleDir("Groups")
	if err != nil {
		return err
	}
	makeIdxs := func(dir *datafs.Data, srt *tensor.Rows, val string, start, r int) {
		n := r - start
		it := datafs.NewValue[int](dir, val, n)
		for j := range n {
			it.SetIntRow(srt.Indexes[start+j], j) // key to indirect through sort indexes
		}
	}

	for i, tsr := range tsrs {
		nr := tsr.DimSize(0)
		if nr == 0 {
			continue
		}
		nm := tsr.Metadata().Name()
		if nm == "" {
			nm = strconv.Itoa(i)
		}
		td, _ := gd.Mkdir(nm)
		srt := tensor.AsRows(tsr).CloneIndexes()
		srt.SortStable(tensor.Ascending)
		start := 0
		if tsr.IsString() {
			lastVal := srt.StringRow(0)
			for r := range nr {
				v := srt.StringRow(r)
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
			lastVal := srt.FloatRow(0)
			for r := range nr {
				v := srt.FloatRow(r)
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
func TableGroups(dir *datafs.Data, dt *table.Table, columns ...string) error {
	dv := table.NewView(dt)
	// important for consistency across columns, to do full outer product sort first.
	dv.SortColumns(tensor.Ascending, tensor.StableSort, columns...)
	return Groups(dir, dv.ColumnList(columns...)...)
}

// GroupAll copies all indexes from the first given tensor,
// into an "All/All" tensor in the given [datafs], which can then
// be used with [GroupStats] to generate summary statistics across
// all the data. See [Groups] for more general documentation.
func GroupAll(dir *datafs.Data, tsrs ...tensor.Tensor) error {
	gd, err := dir.RecycleDir("Groups")
	if err != nil {
		return err
	}
	tsr := tensor.AsRows(tsrs[0])
	nr := tsr.NumRows()
	if nr == 0 {
		return nil
	}
	td, _ := gd.Mkdir("All")
	it := datafs.NewValue[int](td, "All", nr)
	for j := range nr {
		it.SetIntRow(tsr.RowIndex(j), j) // key to indirect through any existing indexes
	}
	return nil
}

// todo: GroupCombined

// note: we have to pass stat as a string here because we need the name
// to record the results in the datafs, and we can't get the name directly.
// also we need _2_ anys, and varargs!

// GroupStats computes the given stats function on the unique grouped indexes
// produced by the [Groups] function, in the given [datafs] directory,
// applied to each of the tensors passed here.
// It creates a "Stats" subdirectory in given directory, with
// subdirectories with the name of each value tensor (if it does not
// yet exist), and then creates a subdirectory within that
// for the statistic name.  Within that statistic directory, it creates
// a String tensor with the unique values of each source [Groups] tensor,
// and a aligned Float64 tensor with the statistics results for each such
// unique group value. See the README.md file for a diagram of the results.
func GroupStats(dir *datafs.Data, stat string, tsrs ...tensor.Tensor) error {
	gd, err := dir.RecycleDir("Groups")
	if err != nil {
		return err
	}
	sd, err := dir.RecycleDir("Stats")
	if err != nil {
		return err
	}
	stnm := StripPackage(stat)
	spl := strings.Split(stat, ".")
	if len(spl) == 2 {
		stnm = spl[1]
	}
	stout := tensor.NewFloat64Scalar(0)
	groups := gd.ItemsFunc(nil)
	for _, gp := range groups {
		gpnm := gp.Name()
		ggd, _ := gd.RecycleDir(gpnm)
		vals := ggd.ValuesFunc(nil)
		nv := len(vals)
		if nv == 0 {
			continue
		}
		sgd, _ := sd.RecycleDir(gpnm)
		gv := sgd.Item(gpnm)
		if gv == nil {
			gtsr := datafs.NewValue[string](sgd, gpnm, nv)
			for i, v := range vals {
				gtsr.SetStringRow(v.Metadata().Name(), i)
			}
		}
		for _, tsr := range tsrs {
			vd, _ := sgd.RecycleDir(tsr.Metadata().Name())
			sv := datafs.NewValue[float64](vd, stnm, nv)
			for i, v := range vals {
				idx := tensor.AsIntSlice(v)
				sg := tensor.NewRows(tsr.AsValues(), idx...)
				tensor.Call(stat, sg, stout)
				sv.SetFloatRow(stout.Float1D(0), i)
			}
		}
	}
	return nil
}

// TableGroupStats runs [GroupStats] using standard [Stats]
// on the given columns from given [table.Table].
func TableGroupStats(dir *datafs.Data, stat Stats, dt *table.Table, columns ...string) error {
	return GroupStats(dir, stat.FuncName(), dt.ColumnList(columns...)...)
}

// GroupDescribe runs standard descriptive statistics on given tensor data
// using [GroupStats] function, with [DescriptiveStats] list of stats.
func GroupDescribe(dir *datafs.Data, tsrs ...tensor.Tensor) error {
	for _, st := range DescriptiveStats {
		err := GroupStats(dir, st.FuncName(), tsrs...)
		if err != nil {
			return err
		}
	}
	return nil
}

// TableGroupDescribe runs [GroupDescribe] on the given columns from given [table.Table].
func TableGroupDescribe(dir *datafs.Data, dt *table.Table, columns ...string) error {
	return GroupDescribe(dir, dt.ColumnList(columns...)...)
}
