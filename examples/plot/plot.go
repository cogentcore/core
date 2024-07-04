// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"embed"

	"cogentcore.org/core/core"
	"cogentcore.org/core/plot/plotcore"
	"cogentcore.org/core/tensor/table"
)

//go:embed *.tsv
var tsv embed.FS

func main() {
	b := core.NewBody("Plot Example")

	epc := table.NewTable("epc")
	epc.OpenFS(tsv, "ra25epoch.tsv", table.Tab)

	pl := plotcore.NewPlotEditor(b)
	pl.Options.Title = "RA25 Epoch Train"
	pl.Options.XAxisColumn = "Epoch"
	pl.Options.Points = true
	pl.SetTable(epc)
	pl.ColumnOptions("UnitErr").On = true
	b.AddAppBar(pl.MakeToolbar)

	b.RunMainWindow()
}
