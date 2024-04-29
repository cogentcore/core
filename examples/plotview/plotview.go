// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"embed"

	"cogentcore.org/core/core"
	"cogentcore.org/core/plot/plotview"
	"cogentcore.org/core/tensor/table"
)

//go:embed *.tsv
var tsv embed.FS

func main() {
	b := core.NewBody("plotview")

	epc := table.NewTable(0, "epc")
	epc.OpenFS(tsv, "ra25epoch.tsv", table.Tab)

	pl := plotview.NewPlotView(b)
	pl.SetTable(epc)
	pl.Params.Title = "RA25 Epoch Train"
	pl.Params.XAxisCol = "Epoch"
	pl.ColumnParams("UnitErr").On = true

	b.AddAppBar(pl.ConfigToolbar)

	b.RunMainWindow()
}
