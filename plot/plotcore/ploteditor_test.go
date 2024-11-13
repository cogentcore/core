// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package plotcore

import (
	"testing"

	"cogentcore.org/core/core"
	"cogentcore.org/core/plot"
	"cogentcore.org/core/plot/plots"
	"cogentcore.org/core/tensor"
	"cogentcore.org/core/tensor/table"
)

type data struct {
	City       string
	Population float32
	Area       float32
}

func TestTablePlotEditor(t *testing.T) {
	b := core.NewBody()

	epc := table.New("epc")
	epc.OpenCSV("testdata/ra25epoch.tsv", tensor.Tab)

	pl := NewPlotEditor(b)
	pst := func(s *plot.Style) {
		s.Plot.Title = "RA25 Epoch Train"
		s.Plot.PointsOn = plot.On
	}
	perr := epc.Column("PctErr")
	plot.SetStylersTo(perr, plot.Stylers{pst, func(s *plot.Style) {
		s.On = true
		s.Role = plot.Y
	}})
	pl.SetTable(epc)
	b.AddTopBar(func(bar *core.Frame) {
		core.NewToolbar(bar).Maker(pl.MakeToolbar)
	})
	b.AssertRender(t, "table")
}

func TestSlicePlotEditor(t *testing.T) {
	dt := []data{
		{"Davis", 62000, 500},
		{"Boulder", 85000, 800},
	}

	b := core.NewBody()
	pl := NewPlotEditor(b)
	pst := func(s *plot.Style) {
		s.Plot.Title = "Test Data"
		s.Plot.PointsOn = plot.On
	}
	onst := func(s *plot.Style) {
		pst(s)
		s.Plotter = plots.BarType
		s.On = true
		s.Role = plot.Y
	}
	pl.SetSlice(dt, pst, onst)
	b.AddTopBar(func(bar *core.Frame) {
		core.NewToolbar(bar).Maker(pl.MakeToolbar)
	})

	b.AssertRender(t, "slice")
}
