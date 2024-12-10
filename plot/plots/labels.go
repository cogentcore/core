// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package plots

import (
	"image"

	"cogentcore.org/core/math32"
	"cogentcore.org/core/math32/minmax"
	"cogentcore.org/core/plot"
)

// LabelsType is be used for specifying the type name.
const LabelsType = "Labels"

func init() {
	plot.RegisterPlotter(LabelsType, "draws text labels at specified X, Y points.", []plot.Roles{plot.X, plot.Y, plot.Label}, []plot.Roles{}, func(data plot.Data) plot.Plotter {
		return NewLabels(data)
	})
}

// Labels draws text labels at specified X, Y points.
type Labels struct {
	// copies of data for this line
	X, Y   plot.Values
	Labels plot.Labels

	// PX, PY are the actual pixel plotting coordinates for each XY value.
	PX, PY []float32

	// Style is the style of the label text.
	Style plot.Style

	// plot size and number of TextStyle when styles last generated -- don't regen
	styleSize image.Point
	stylers   plot.Stylers
	ystylers  plot.Stylers
}

// NewLabels returns a new Labels using defaults
// Styler functions are obtained from the Label metadata if present.
func NewLabels(data plot.Data) *Labels {
	if data.CheckLengths() != nil {
		return nil
	}
	lb := &Labels{}
	lb.X = plot.MustCopyRole(data, plot.X)
	lb.Y = plot.MustCopyRole(data, plot.Y)
	if lb.X == nil || lb.Y == nil {
		return nil
	}
	ld := data[plot.Label]
	if ld == nil {
		return nil
	}
	lb.Labels = make(plot.Labels, lb.X.Len())
	for i := range ld.Len() {
		lb.Labels[i] = ld.String1D(i)
	}

	lb.stylers = plot.GetStylersFromData(data, plot.Label)
	lb.ystylers = plot.GetStylersFromData(data, plot.Y)
	lb.Defaults()
	return lb
}

func (lb *Labels) Defaults() {
	lb.Style.Defaults()
}

// Styler adds a style function to set style parameters.
func (lb *Labels) Styler(f func(s *plot.Style)) *Labels {
	lb.stylers.Add(f)
	return lb
}

func (lb *Labels) ApplyStyle(ps *plot.PlotStyle) {
	ps.SetElementStyle(&lb.Style)
	yst := &plot.Style{}
	lb.ystylers.Run(yst)
	lb.Style.Range = yst.Range // get range from y
	lb.stylers.Run(&lb.Style)  // can still override here
}

func (lb *Labels) Stylers() *plot.Stylers { return &lb.stylers }

func (lb *Labels) Data() (data plot.Data, pixX, pixY []float32) {
	pixX = lb.PX
	pixY = lb.PY
	data = plot.Data{}
	data[plot.X] = lb.X
	data[plot.Y] = lb.Y
	data[plot.Label] = lb.Labels
	return
}

// Plot implements the Plotter interface, drawing labels.
func (lb *Labels) Plot(plt *plot.Plot) {
	pc := plt.Paint
	uc := &pc.UnitContext
	lb.PX = plot.PlotX(plt, lb.X)
	lb.PY = plot.PlotY(plt, lb.Y)
	st := &lb.Style.Text
	st.Offset.ToDots(uc)
	var ltxt plot.Text
	ltxt.Defaults()
	ltxt.Style = *st
	ltxt.ToDots(uc)
	for i, label := range lb.Labels {
		if label == "" {
			continue
		}
		ltxt.Text = label
		ltxt.Config(plt)
		tht := ltxt.PaintText.BBox.Size().Y
		ltxt.Draw(plt, math32.Vec2(lb.PX[i]+st.Offset.X.Dots, lb.PY[i]+st.Offset.Y.Dots-tht))
	}
}

// UpdateRange updates the given ranges.
func (lb *Labels) UpdateRange(plt *plot.Plot, xr, yr, zr *minmax.F64) {
	// todo: include point sizes!
	plot.Range(lb.X, xr)
	plot.RangeClamp(lb.Y, yr, &lb.Style.Range)
	pxToData := math32.FromPoint(plt.Size)
	pxToData.X = float32(xr.Range()) / pxToData.X
	pxToData.Y = float32(yr.Range()) / pxToData.Y
	st := &lb.Style.Text
	var ltxt plot.Text
	ltxt.Style = *st
	for i, label := range lb.Labels {
		if label == "" {
			continue
		}
		ltxt.Text = label
		ltxt.Config(plt)
		tht := pxToData.Y * ltxt.PaintText.BBox.Size().Y
		twd := 1.1 * pxToData.X * ltxt.PaintText.BBox.Size().X
		x := lb.X[i]
		y := lb.Y[i]
		maxx := x + float64(pxToData.X*st.Offset.X.Dots+twd)
		maxy := y + float64(pxToData.Y*st.Offset.Y.Dots+tht) // y is up here
		xr.FitInRange(minmax.F64{x, maxx})
		yr.FitInRange(minmax.F64{y, maxy})
	}
}
