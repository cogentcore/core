// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"embed"
	"log/slog"

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
		pl.Params.Title = "RA25 Epoch Train"
		pl.Params.XAxisColumn = "Epoch"
		// pl.Params.Scale = 2
		pl.Params.Points = true
		pl.SetTable(epc)
		pl.ColumnParams("UnitErr").On = true
		b.AddAppBar(pl.MakeToolbar)
	} else {
		data := []Data{
			{"Davis", 62000, 500},
			{"Boulder", 85000, 800},
		}

		dt, err := table.NewSliceTable(data)
		if err != nil {
			slog.Error(err.Error())
		}

		pl := plotcore.NewPlotEditor(b)
		pl.Params.Title = "Slice Data"
		pl.Params.XAxisColumn = "City"
		pl.Params.Points = true
		pl.SetTable(dt)
		pl.ColumnParams("Population").On = true
		b.AddAppBar(pl.MakeToolbar)
	}

	b.RunMainWindow()
}
