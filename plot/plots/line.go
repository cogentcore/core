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
	LineStyle plot.LineStyle

	// FillColor is the color to fill the area below the plot.
	// Use nil to disable the filling. This is the default.
	FillColor color.Color

	// if true, draw lines that connect points with a negative X-axis direction;
	// otherwise there is a break in the line.
	// default is false, so that repeated series of data across the X axis
	// are plotted separately.
	NegativeXDraw bool
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

// NewLinePoints returns both a Line and a
// Scatter plot for the given point data.
func NewLinePoints(xys XYer) (*Line, *Scatter, error) {
	sc, err := NewScatter(xys)
	if err != nil {
		return nil, nil, err
	}
	ln := &Line{XYs: sc.XYs}
	ln.LineStyle.Defaults()
	return ln, sc, nil
}

// Plot draws the Line, implementing the plot.Plotter interface.
func (pts *Line) Plot(plt *plot.Plot) {
	pc := plt.Paint

	ps := PlotXYs(plt, pts.XYs)
	np := len(ps)
	pts.PXYs = ps

	if pts.FillColor != nil {
		pc.FillStyle.Color = colors.C(pts.FillColor)
		minY := plt.PY(plt.Y.Min)
		prev := XY{X: ps[0].X, Y: minY}
		pc.MoveTo(prev.X, prev.Y)
		for i := range ps {
			pt := ps[i]
			switch pts.StepStyle {
			case NoStep:
				if pt.X < prev.X {
					pc.LineTo(prev.X, minY)
					pc.ClosePath()
					pc.MoveTo(pt.X, minY)
				}
				pc.LineTo(pt.X, pt.Y)
			case PreStep:
				if i == 0 {
					continue
				}
				if pt.X < prev.X {
					pc.LineTo(prev.X, minY)
					pc.ClosePath()
					pc.MoveTo(pt.X, minY)
				} else {
					pc.LineTo(prev.X, pt.Y)
				}
				pc.LineTo(pt.X, pt.Y)
			case MidStep:
				if pt.X < prev.X {
					pc.LineTo(prev.X, minY)
					pc.ClosePath()
					pc.MoveTo(pt.X, minY)
				} else {
					pc.LineTo(0.5*(prev.X+pt.X), prev.Y)
					pc.LineTo(0.5*(prev.X+pt.X), pt.Y)
				}
				pc.LineTo(pt.X, pt.Y)
			case PostStep:
				if pt.X < prev.X {
					pc.LineTo(prev.X, minY)
					pc.ClosePath()
					pc.MoveTo(pt.X, minY)
				} else {
					pc.LineTo(pt.X, prev.Y)
				}
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
		if pts.StepStyle != NoStep {
			if pt.X >= prev.X {
				switch pts.StepStyle {
				case PreStep:
					pc.LineTo(prev.X, pt.Y)
				case MidStep:
					pc.LineTo(0.5*(prev.X+pt.X), prev.Y)
					pc.LineTo(0.5*(prev.X+pt.X), pt.Y)
				case PostStep:
					pc.LineTo(pt.X, prev.Y)
				}
			} else {
				pc.MoveTo(pt.X, pt.Y)
			}
		}
		if !pts.NegativeXDraw && pt.X < prev.X {
			pc.MoveTo(pt.X, pt.Y)
		} else {
			pc.LineTo(pt.X, pt.Y)
		}
		prev = pt
	}
	pc.Stroke()
}

// DataRange returns the minimum and maximum
// x and y values, implementing the plot.DataRanger interface.
func (pts *XYs) DataRange() (xmin, xmax, ymin, ymax float32) {
	return XYRange(pts)
}

// Thumbnail returns the thumbnail for the LineTo, implementing the plot.Thumbnailer interface.
func (pts *Line) Thumbnail(plt *plot.Plot) {
	pc := plt.Paint
	ptb := pc.Bounds
	midY := 0.5 * float32(ptb.Min.Y+ptb.Max.Y)

	if pts.FillColor != nil {
		var topY float32
		if pts.LineStyle.Width.Value == 0 {
			topY = float32(ptb.Min.Y)
		} else {
			topY = midY
		}
		pc.MoveTo(float32(ptb.Min.X), float32(ptb.Max.Y))
		pc.LineTo(float32(ptb.Min.X), topY)
		pc.LineTo(float32(ptb.Max.X), topY)
		pc.LineTo(float32(ptb.Max.X), float32(ptb.Max.Y))
		pc.ClosePath()
		pc.Fill()
	}

	if pts.LineStyle.SetStroke(plt) {
		pc.MoveTo(float32(ptb.Min.X), midY)
		pc.LineTo(float32(ptb.Max.X), midY)
		pc.Stroke()
	}
}
