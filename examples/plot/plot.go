// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"embed"

	"cogentcore.org/core/core"
	"cogentcore.org/core/plot"
	"cogentcore.org/core/plot/plotcore"
	"cogentcore.org/core/tensor"
	"cogentcore.org/core/tensor/table"
)

//go:embed *.tsv
var tsv embed.FS

func main() {
	b := core.NewBody("Plot Example")

	epc := table.New("epc")
	epc.OpenFS(tsv, "ra25epoch.tsv", tensor.Tab)
	pst := func(s *plot.Style) {
		s.Plot.Title = "RA25 Epoch Train"
	}
	perr := epc.Column("PctErr")
	plot.SetStylersTo(perr, plot.Stylers{pst, func(s *plot.Style) {
		s.On = true
		s.Role = plot.Y
	}})

	pl := plotcore.NewPlotEditor(b)
	pl.SetTable(epc)
	b.AddTopBar(func(bar *core.Frame) {
		core.NewToolbar(bar).Maker(pl.MakeToolbar)
	})

	b.RunMainWindow()
}
