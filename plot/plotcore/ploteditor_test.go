// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package plotcore

import (
	"testing"

	"cogentcore.org/core/core"
	"cogentcore.org/core/tensor/table"
)

type Data struct {
	City       string
	Population float32
	Area       float32
}

func TestTablePlotEditor(t *testing.T) {
	b := core.NewBody()

	epc := table.NewTable("epc")
	epc.OpenCSV("testdata/ra25epoch.tsv", table.Tab)

	pl := NewPlotEditor(b)
	pl.Options.Title = "RA25 Epoch Train"
	pl.Options.XAxis = "Epoch"
	// pl.Options.Scale = 2
	pl.Options.Points = true
	pl.SetTable(epc)
	pl.ColumnOptions("UnitErr").On = true
	b.AddAppBar(pl.MakeToolbar)
	b.AssertRender(t, "table")
}

func TestSlicePlotEditor(t *testing.T) {
	data := []Data{
		{"Davis", 62000, 500},
		{"Boulder", 85000, 800},
	}

	b := core.NewBody()

	pl := NewPlotEditor(b)
	pl.Options.Title = "Slice Data"
	pl.Options.Points = true
	pl.SetSlice(data)
	b.AddAppBar(pl.MakeToolbar)

	b.AssertRender(t, "slice")
}
