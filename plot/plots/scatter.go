// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Adapted from github.com/gonum/plot:
// Copyright ©2015 The Gonum Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package plots

import (
	"cogentcore.org/core/math32"
	"cogentcore.org/core/plot"
	"cogentcore.org/core/styles/units"
)

// Scatter implements the Plotter interface, drawing
// a shape for each point.
type Scatter struct {
	// XYs is a copy of the points for this scatter.
	plot.XYs

	// PXYs is the actual plotting coordinates for each XY value.
	PXYs plot.XYs

	// size of shape to draw for each point
	PointSize units.Value

	// shape to draw for each point
	PointShape Shapes

	// LineStyle is the style of the line connecting the points.
	// Use zero width to disable lines.
	LineStyle plot.LineStyle
}

// NewScatter returns a Scatter that uses the
// default glyph style.
func NewScatter(xys plot.XYer) (*Scatter, error) {
	data, err := plot.CopyXYs(xys)
	if err != nil {
		return nil, err
	}
	sc := &Scatter{XYs: data}
	sc.LineStyle.Defaults()
	sc.PointSize.Pt(4)
	return sc, nil
}

func (pts *Scatter) XYData() (data plot.XYer, pixels plot.XYer) {
	data = pts.XYs
	pixels = pts.PXYs
	return
}

// Plot draws the Line, implementing the plot.Plotter interface.
func (pts *Scatter) Plot(plt *plot.Plot) {
	pc := plt.Paint
	if !pts.LineStyle.SetStroke(plt) {
		return
	}
	pts.PointSize.ToDots(&pc.UnitContext)
	pc.FillStyle.Color = pts.LineStyle.Color
	ps := plot.PlotXYs(plt, pts.XYs)
	for i := range ps {
		pt := ps[i]
		DrawShape(pc, math32.Vec2(pt.X, pt.Y), pts.PointSize.Dots, pts.PointShape)
	}
	pc.FillStyle.Color = nil
}

// DataRange returns the minimum and maximum
// x and y values, implementing the plot.DataRanger interface.
func (pts *Scatter) DataRange() (xmin, xmax, ymin, ymax float32) {
	return plot.XYRange(pts)
}

// Thumbnail the thumbnail for the Scatter,
// implementing the plot.Thumbnailer interface.
func (pts *Scatter) Thumbnail(plt *plot.Plot) {
	if !pts.LineStyle.SetStroke(plt) {
		return
	}
	pc := plt.Paint
	pts.PointSize.ToDots(&pc.UnitContext)
	pc.FillStyle.Color = pts.LineStyle.Color
	ptb := pc.Bounds
	midX := 0.5 * float32(ptb.Min.X+ptb.Max.X)
	midY := 0.5 * float32(ptb.Min.Y+ptb.Max.Y)

	DrawShape(pc, math32.Vec2(midX, midY), pts.PointSize.Dots, pts.PointShape)
}
