// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package stats

import (
	"strconv"

	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/tensor"
	"cogentcore.org/core/tensor/datafs"
	"cogentcore.org/core/tensor/table"
)

// DescriptiveStats are the standard descriptive stats used in Describe function.
// Cannot apply the final 3 sort-based stats to higher-dimensional data.
var DescriptiveStats = []Stats{StatCount, StatMean, StatStd, StatSem, StatMin, StatMax, StatQ1, StatMedian, StatQ3}

// Describe adds standard descriptive statistics for given tensor
// to the given [datafs] directory, adding a directory for each tensor
// and result tensor stats for each result.
// This is an easy way to provide a comprehensive description of data.
// The [DescriptiveStats] list is: [Count], [Mean], [Std], [Sem],
// [Min], [Max], [Q1], [Median], [Q3]
func Describe(dir *datafs.Data, tsrs ...tensor.Tensor) {
	dd, err := dir.RecycleDir("Describe")
	if errors.Log(err) != nil {
		return
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
		td, _ := dd.Mkdir(nm)
		for _, st := range DescriptiveStats {
			stnm := st.String()
			sv := datafs.NewValue[float64](td, stnm, 1)
			stout := st.Call(tsr)
			sv.CopyFrom(stout)
		}
	}
}

// DescribeTable runs [Describe] on given columns in table.
func DescribeTable(dir *datafs.Data, dt *table.Table, columns ...string) {
	Describe(dir, dt.ColumnList(columns...)...)
}

// DescribeTableAll runs [Describe] on all numeric columns in given table.
func DescribeTableAll(dir *datafs.Data, dt *table.Table) {
	var cols []string
	for i, cl := range dt.Columns.Values {
		if !cl.IsString() {
			cols = append(cols, dt.ColumnName(i))
		}
	}
	Describe(dir, dt.ColumnList(cols...)...)
}
