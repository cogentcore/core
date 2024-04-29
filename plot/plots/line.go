// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Adapted from github.com/gonum/plot:
// Copyright ©2015 The Gonum Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package plots

import (
	"image/color"

	"cogentcore.org/core/colors"
	"cogentcore.org/core/plot"
)

// StepKind specifies a form of a connection of two consecutive points.
type StepKind int32 //enums:enum

const (
	// NoStep connects two points by simple line
	NoStep StepKind = iota

	// PreStep connects two points by following lines: vertical, horizontal.
	PreStep

	// MidStep connects two points by following lines: horizontal, vertical, horizontal.
	// Vertical line is placed in the middle of the interval.
	MidStep

	// PostStep connects two points by following lines: horizontal, vertical.
	PostStep
)

// Line implements the Plotter interface, drawing a line using XYer data.
type Line struct {
	// XYs is a copy of the points for this line.
	XYs

	// PXYs is the actual plotting coordinates for each XY value.
	PXYs XYs

	// StepStyle is the kind of the step line.
	StepStyle StepKind

	// LineStyle is the style of the line connecting the points.
	// Use zero width to disable lines.
	plot.LineStyle

	// FillColor is the color to fill the area below the plot.
	// Use nil to disable the filling. This is the default.
	FillColor color.Color
}

// NewLine returns a Line that uses the default line style and
// does not draw glyphs.
func NewLine(xys XYer) (*Line, error) {
	data, err := CopyXYs(xys)
	if err != nil {
		return nil, err
	}
	ln := &Line{XYs: data}
	ln.LineStyle.Defaults()
	return ln, nil
}

/*
// NewLinePoints returns both a Line and a
// Points for the given point data.
func NewLinePoints(xys XYer) (*Line, *Scatter, error) {
	s, err := NewScatter(xys)
	if err != nil {
		return nil, nil, err
	}
	ln := &Line{XYs: s.XYs}
	ln.Defaults()
	return ln, s, nil
}
*/

// Plot draws the Line, implementing the plot.Plotter interface.
func (pts *Line) Plot(plt *plot.Plot) {
	pc := plt.Paint

	ps := PlotXYs(plt, pts.XYs)
	np := len(ps)
	pts.PXYs = ps

	if pts.FillColor != nil {
		pc.FillStyle.Color = colors.C(pts.FillColor)
		minY := plt.PY(plt.Y.Min)
		pc.MoveTo(ps[0].X, minY)
		prev := XY{X: 0, Y: minY}
		for i := range ps {
			pt := ps[i]
			switch pts.StepStyle {
			case NoStep:
				pc.LineTo(pt.X, pt.Y)
			case PreStep:
				if i == 0 {
					continue
				}
				pc.LineTo(prev.X, pt.Y)
				pc.LineTo(pt.X, pt.Y)
			case MidStep:
				pc.LineTo(0.5*(prev.X+pt.X), prev.Y)
				pc.LineTo(0.5*(prev.X+pt.X), pt.Y)
				pc.LineTo(pt.X, pt.Y)
			case PostStep:
				pc.LineTo(pt.X, prev.Y)
				pc.LineTo(pt.X, pt.Y)
			}
			prev = pt
		}
		pc.LineTo(prev.X, minY)
		pc.ClosePath()
		pc.Fill()
	}
	pc.FillStyle.Color = nil

	if !pts.LineStyle.SetStroke(plt) {
		return
	}
	prev := ps[0]
	pc.MoveTo(prev.X, prev.Y)
	for i := 1; i < np; i++ {
		pt := ps[i]
		switch pts.StepStyle {
		case PreStep:
			pc.LineTo(prev.X, pt.Y)
		case MidStep:
			pc.LineTo(0.5*(prev.X+pt.X), prev.Y)
			pc.LineTo(0.5*(prev.X+pt.X), pt.Y)
		case PostStep:
			pc.LineTo(pt.X, prev.Y)
		}
		pc.LineTo(pt.X, pt.Y)
		prev = pt
	}
	pc.Stroke()
}

// DataRange returns the minimum and maximum
// x and y values, implementing the plot.DataRanger interface.
func (pts *XYs) DataRange() (xmin, xmax, ymin, ymax float32) {
	return XYRange(pts)
}

/*
// Thumbnail returns the thumbnail for the LineTo, implementing the plot.Thumbnailer interface.
func (pts *LineTo) Thumbnail(c *draw.Canvas) {
	if pts.FillColor != nil {
		var topY vg.Length
		if pts.LineToStyle.Width == 0 {
			topY = c.Max.Y
		} else {
			topY = (c.Min.Y + c.Max.Y) / 2
		}
		points := []vg.Point{
			{X: c.Min.X, Y: c.Min.Y},
			{X: c.Min.X, Y: topY},
			{X: c.Max.X, Y: topY},
			{X: c.Max.X, Y: c.Min.Y},
		}
		poly := c.ClipPolygonY(points)
		c.FillPolygon(pts.FillColor, poly)
	}

	if pts.LineToStyle.Width != 0 {
		y := c.Center().Y
		c.StrokeLineTo2(pts.LineToStyle, c.Min.X, y, c.Max.X, y)
	}
}
*/
