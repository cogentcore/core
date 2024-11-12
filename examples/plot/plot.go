// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"embed"

	"cogentcore.org/core/core"
	"cogentcore.org/core/plot"
	"cogentcore.org/core/plot/plotcore"
	_ "cogentcore.org/core/plot/plots"
	"cogentcore.org/core/tensor"
	"cogentcore.org/core/tensor/table"
)

//go:embed *.tsv
var tsv embed.FS

func main() {
	b := core.NewBody("Plot Example")

	epc := table.New("epc")
	epc.OpenFS(tsv, "ra25epoch.tsv", tensor.Tab)
	epcc := epc.Column("Epoch")
	plot.SetStylersTo(epcc, plot.Stylers{func(s *plot.Style) {
		s.Role = plot.X
	}})
	perr := epc.Column("PctErr")
	plot.SetStylersTo(perr, plot.Stylers{func(s *plot.Style) {
		s.On = true
		s.Role = plot.Y
	}})

	pl := plotcore.NewPlotEditor(b)
	// pl.Options.Title = "RA25 Epoch Train"
	// pl.Options.XAxis = "Epoch"
	// pl.Options.Points = true
	// pl.ColumnOptions("UnitErr").On = true
	pl.SetTable(epc)
	b.AddTopBar(func(bar *core.Frame) {
		core.NewToolbar(bar).Maker(pl.MakeToolbar)
	})

	b.RunMainWindow()
}
