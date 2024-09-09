// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package datafs

import (
	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/base/fsx"
	"cogentcore.org/core/base/metadata"
	"cogentcore.org/core/plot/plotcore"
	"cogentcore.org/core/tensor/table"
)

// This file provides standardized metadata options for frequent
// use cases, using codified key names to eliminate typos.

// SetMetaItems sets given metadata for items in given directory
// with given names.  Returns error for any items not found.
func (d *Data) SetMetaItems(key string, value any, names ...string) error {
	its, err := d.Items(names...)
	for _, it := range its {
		it.Meta.Set(key, value)
	}
	return err
}

// PlotColumnZeroOne returns plot options with a fixed 0-1 range
func PlotColumnZeroOne() *plotcore.ColumnOptions {
	opts := &plotcore.ColumnOptions{}
	opts.Range.SetMin(0)
	opts.Range.SetMax(1)
	return opts
}

// SetPlotColumnOptions sets given plotting options for named items
// within this directory (stored in Metadata).
func (d *Data) SetPlotColumnOptions(opts *plotcore.ColumnOptions, names ...string) error {
	return d.SetMetaItems("PlotColumnOptions", opts, names...)
}

// PlotColumnOptions returns plotting options if they have been set, else nil.
func (d *Data) PlotColumnOptions() *plotcore.ColumnOptions {
	return errors.Ignore1(metadata.Get[*plotcore.ColumnOptions](d.Meta, "PlotColumnOptions"))
}

// SetCalcFunc sets a function to compute an updated Value for this data item.
// Function is stored as CalcFunc in Metadata.  Can be called by [Data.Calc] method.
func (d *Data) SetCalcFunc(fun func() error) {
	d.Meta.Set("CalcFunc", fun)
}

// Calc calls function set by [Data.SetCalcFunc] to compute an updated Value
// for this data item. Returns an error if func not set, or any error from func itself.
// Function is stored as CalcFunc in Metadata.
func (d *Data) Calc() error {
	fun, err := metadata.Get[func() error](d.Meta, "CalcFunc")
	if err != nil {
		return err
	}
	return fun()
}

// CalcAll calls function set by [Data.SetCalcFunc] for all items
// in this directory and all of its subdirectories.
// Calls Calc on items from FlatItemsByTimeFunc(nil)
func (d *Data) CalcAll() error {
	var errs []error
	items := d.FlatItemsByTimeFunc(nil)
	for _, it := range items {
		err := it.Calc()
		if err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

// DirTable returns a table.Table for this directory item, with columns
// as the Tensor elements in the directory and any subdirectories,
// from FlatItemsByTimeFunc using given filter function.
// This is a convenient mechanism for creating a plot of all the data
// in a given directory.
// If such was previously constructed, it is returned from "DirTable"
// Metadata key where the table is stored.
// Row count is updated to current max row.
// Delete that key to reconstruct if items have changed.
func (d *Data) DirTable(fun func(item *Data) bool) *table.Table {
	dt, err := metadata.Get[*table.Table](d.Meta, "DirTable")
	if err == nil {
		var maxRow int
		for _, tsr := range dt.Columns {
			maxRow = max(maxRow, tsr.DimSize(0))
		}
		dt.Rows = maxRow
		return dt
	}
	items := d.FlatItemsByTimeFunc(fun)
	dt = table.NewTable(fsx.DirAndFile(string(d.Path())))
	for _, it := range items {
		tsr := it.AsTensor()
		if tsr == nil {
			continue
		}
		if dt.Rows == 0 {
			dt.Rows = tsr.DimSize(0)
		}
		nm := it.Name()
		if it.Parent != d {
			nm = fsx.DirAndFile(string(it.Path()))
		}
		dt.AddColumn(tsr, nm)
	}
	d.Meta.Set("DirTable", dt)
	return dt
}
