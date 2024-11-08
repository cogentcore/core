// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package plots

import (
	"image"

	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/plot"
)

// Labels implements the Plotter interface,
// drawing a set of labels at specified points.
type Labels struct {
	// XYs is a copy of the points for labels
	plot.XYs

	// PXYs is the actual pixel plotting coordinates for each XY value.
	PXYs plot.XYs

	// Labels is the set of labels corresponding to each point.
	Labels []string

	// Style is the style of the label text.
	Style plot.TextStyle

	// plot size and number of TextStyle when styles last generated -- don't regen
	styleSize image.Point
	stylers   plot.Stylers
}

// NewLabels returns a new Labels using defaults
func NewLabels(d XYLabeler) *Labels {
	xys, err := plot.CopyXYs(d)
	if errors.Log(err) != nil {
		return nil
	}

	if d.Len() != len(xys) {
		errors.Log(errors.New("plotter: number of points does not match the number of labels"))
		return nil
	}

	strs := make([]string, d.Len())
	for i := range strs {
		strs[i] = d.Label(i)
	}

	lb := &Labels{XYs: xys, Labels: strs}
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
	st := lb.stylers.NewStyle(ps)
	lb.Style = st.Text
}

func (lb *Labels) Stylers() *plot.Stylers { return &lb.stylers }

func (lb *Labels) XYData() (data plot.XYer, pixels plot.XYer) {
	data = lb.XYs
	pixels = lb.PXYs
	return
}

// Plot implements the Plotter interface, drawing labels.
func (lb *Labels) Plot(plt *plot.Plot) {
	ps := plot.PlotXYs(plt, lb.XYs)
	pc := plt.Paint
	uc := &pc.UnitContext
	lb.Style.Offset.ToDots(uc)
	lb.Style.ToDots(uc)
	var ltxt plot.Text
	ltxt.Style = lb.Style
	for i, label := range lb.Labels {
		if label == "" {
			continue
		}
		ltxt.Text = label
		ltxt.Config(plt)
		tht := ltxt.PaintText.BBox.Size().Y
		ltxt.Draw(plt, math32.Vec2(ps[i].X+lb.Style.Offset.X.Dots, ps[i].Y+lb.Style.Offset.Y.Dots-tht))
	}
}

// DataRange returns the minimum and maximum X and Y values
func (lb *Labels) DataRange(plt *plot.Plot) (xmin, xmax, ymin, ymax float32) {
	xmin, xmax, ymin, ymax = plot.XYRange(lb) // first get basic numerical range
	pxToData := math32.FromPoint(plt.Size)
	pxToData.X = (xmax - xmin) / pxToData.X
	pxToData.Y = (ymax - ymin) / pxToData.Y
	var ltxt plot.Text
	ltxt.Style = lb.Style
	for i, label := range lb.Labels {
		if label == "" {
			continue
		}
		ltxt.Text = label
		ltxt.Config(plt)
		tht := pxToData.Y * ltxt.PaintText.BBox.Size().Y
		twd := 1.1 * pxToData.X * ltxt.PaintText.BBox.Size().X
		x, y := lb.XY(i)
		minx := x
		maxx := x + pxToData.X*lb.Style.Offset.X.Dots + twd
		miny := y
		maxy := y + pxToData.Y*lb.Style.Offset.Y.Dots + tht // y is up here
		xmin = min(xmin, minx)
		xmax = max(xmax, maxx)
		ymin = min(ymin, miny)
		ymax = max(ymax, maxy)
	}
	return
}

// XYLabeler combines the [plot.XYer] and [plot.Labeler] types.
type XYLabeler interface {
	plot.XYer
	plot.Labeler
}

// XYLabels holds XY data with labels.
// The ith label corresponds to the ith XY.
type XYLabels struct {
	plot.XYs
	Labels []string
}

// Label returns the label for point index i.
func (lb XYLabels) Label(i int) string {
	return lb.Labels[i]
}

var _ XYLabeler = (*XYLabels)(nil)
