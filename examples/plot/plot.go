// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"embed"

	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/core"
	"cogentcore.org/core/plot/plotcore"
	"cogentcore.org/core/tensor/table"
)

//go:embed *.tsv
var tsv embed.FS

type Data struct {
	City       string
	Population float32
	Area       float32
}

func main() {
	b := core.NewBody("Plot Example")

	if true {
		epc := table.NewTable("epc")
		epc.OpenFS(tsv, "ra25epoch.tsv", table.Tab)

		pl := plotcore.NewPlotEditor(b)
		pl.Options.Title = "RA25 Epoch Train"
		pl.Options.XAxisColumn = "Epoch"
		// pl.Options.Scale = 2
		pl.Options.Points = true
		pl.SetTable(epc)
		pl.ColumnOptions("UnitErr").On = true
		b.AddAppBar(pl.MakeToolbar)
	} else {
		data := []Data{
			{"Davis", 62000, 500},
			{"Boulder", 85000, 800},
		}

		dt := errors.Log1(table.NewSliceTable(data))

		pl := plotcore.NewPlotEditor(b)
		pl.Options.Title = "Slice Data"
		pl.Options.XAxisColumn = "City"
		pl.Options.Points = true
		pl.SetTable(dt)
		pl.ColumnOptions("Population").On = true
		b.AddAppBar(pl.MakeToolbar)
	}

	b.RunMainWindow()
}
