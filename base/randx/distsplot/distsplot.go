// Copyright (c) 2020, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Command distsplot plots histograms of random distributions
package main

//go:generate core generate -add-types

import (
	"strconv"

	"cogentcore.org/core/base/randx"
	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/math32/minmax"
	"cogentcore.org/core/plot/plotview"
	"cogentcore.org/core/tensor"
	"cogentcore.org/core/tensor/stats/histogram"
	"cogentcore.org/core/tensor/table"
	"cogentcore.org/core/views"
)

func main() {
	TheSim.Config()
	TheSim.ConfigGUI()
}

// LogPrec is precision for saving float values in logs
const LogPrec = 4

// Sim holds the params, table, etc
type Sim struct {

	// random params
	Dist randx.RandParams

	// number of samples
	NSamp int

	// number of bins in the histogram
	NBins int

	// range for histogram
	Range minmax.F64

	// table for raw data
	Table *table.Table `view:"no-inline"`

	// histogram of data
	Hist *table.Table `view:"no-inline"`

	// the plot
	Plot *plotview.PlotView `view:"-"`
}

// TheSim is the overall state for this simulation
var TheSim Sim

// Config configures all the elements using the standard functions
func (ss *Sim) Config() {
	ss.Dist.Defaults()
	ss.Dist.Dist = randx.Gaussian
	ss.Dist.Mean = 0.5
	ss.Dist.Var = 0.15
	ss.NSamp = 1000000
	ss.NBins = 100
	ss.Range.Set(0, 1)
	ss.Update()
	ss.Table = &table.Table{}
	ss.Hist = &table.Table{}
	ss.ConfigTable(ss.Table)
	ss.Run()
}

// Update updates computed values
func (ss *Sim) Update() {
}

// Run generates the data and plots a histogram of results
func (ss *Sim) Run() {
	ss.Update()
	dt := ss.Table

	dt.SetNumRows(ss.NSamp)
	for vi := 0; vi < ss.NSamp; vi++ {
		vl := ss.Dist.Gen()
		dt.SetFloat("Val", vi, float64(vl))
	}

	histogram.F64Table(ss.Hist, dt.Columns[0].(*tensor.Float64).Values, ss.NBins, ss.Range.Min, ss.Range.Max)
	if ss.Plot != nil {
		ss.Plot.UpdatePlot()
	}
}

func (ss *Sim) ConfigTable(dt *table.Table) {
	dt.SetMetaData("name", "Data")
	dt.SetMetaData("read-only", "true")
	dt.SetMetaData("precision", strconv.Itoa(LogPrec))

	dt.AddFloat64Column("Val")
}

func (ss *Sim) ConfigPlot(plt *plotview.PlotView, dt *table.Table) *plotview.PlotView {
	plt.Params.Title = "Rand Dist Histogram"
	plt.Params.XAxisColumn = "Value"
	plt.Params.Type = plotview.Bar
	plt.Params.XAxisRotation = 45
	plt.SetTable(dt)
	// order of params: on, fixMin, min, fixMax, max
	plt.SetColParams("Value", plotview.Off, plotview.FloatMin, 0, plotview.FloatMax, 0)
	plt.SetColParams("Count", plotview.On, plotview.FixMin, 0, plotview.FloatMax, 0)
	return plt
}

// ConfigGUI configures the Cogent Core GUI interface for this simulation.
func (ss *Sim) ConfigGUI() *core.Body {
	b := core.NewBody("distplot")

	split := core.NewSplits(b)

	sv := views.NewStructView(split)
	sv.SetStruct(ss)

	tv := core.NewTabs(split)

	pt := tv.NewTab("Histogram")
	plt := plotview.NewPlotView(pt)
	ss.Plot = ss.ConfigPlot(plt, ss.Hist)

	split.SetSplits(.3, .7)

	b.AddAppBar(func(c *core.Config) {
		core.Configure(c, "", func(w *core.Button) {
			w.SetText("Run").SetIcon(icons.Update).
				SetTooltip("Generate data and plot histogram.").
				OnClick(func(e events.Event) {
					ss.Run()
				})
		})
		core.Configure(c, "", func(w *core.Button) {
			w.SetText("README").SetIcon(icons.FileMarkdown).
				SetTooltip("Opens your browser on the README file that contains instructions for how to run this model.").
				OnClick(func(e events.Event) {
					core.TheApp.OpenURL("https://github.com/cogentcore/core/blob/main/base/randx/distplot/README.md")
				})
		})
	})
	b.RunMainWindow()
	return b
}
