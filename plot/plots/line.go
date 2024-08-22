// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Adapted from github.com/gonum/plot:
// Copyright ©2015 The Gonum Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package plots

//go:generate core generate

import (
	"image"

	"cogentcore.org/core/math32"
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
	plot.XYs

	// PXYs is the actual pixel plotting coordinates for each XY value.
	PXYs plot.XYs

	// StepStyle is the kind of the step line.
	StepStyle StepKind

	// LineStyle is the style of the line connecting the points.
	// Use zero width to disable lines.
	LineStyle plot.LineStyle

	// Fill is the color to fill the area below the plot.
	// Use nil to disable filling, which is the default.
	Fill image.Image

	// if true, draw lines that connect points with a negative X-axis direction;
	// otherwise there is a break in the line.
	// default is false, so that repeated series of data across the X axis
	// are plotted separately.
	NegativeXDraw bool
}

// NewLine returns a Line that uses the default line style and
// does not draw glyphs.
func NewLine(xys plot.XYer) (*Line, error) {
	data, err := plot.CopyXYs(xys)
	if err != nil {
		return nil, err
	}
	ln := &Line{XYs: data}
	ln.Defaults()
	return ln, nil
}

// NewLinePoints returns both a Line and a
// Scatter plot for the given point data.
func NewLinePoints(xys plot.XYer) (*Line, *Scatter, error) {
	sc, err := NewScatter(xys)
	if err != nil {
		return nil, nil, err
	}
	ln := &Line{XYs: sc.XYs}
	ln.Defaults()
	return ln, sc, nil
}

func (pts *Line) Defaults() {
	pts.LineStyle.Defaults()
}

func (pts *Line) XYData() (data plot.XYer, pixels plot.XYer) {
	data = pts.XYs
	pixels = pts.PXYs
	return
}

// Plot draws the Line, implementing the plot.Plotter interface.
func (pts *Line) Plot(plt *plot.Plot) {
	pc := plt.Paint

	ps := plot.PlotXYs(plt, pts.XYs)
	np := len(ps)
	pts.PXYs = ps

	if pts.Fill != nil {
		pc.FillStyle.Color = pts.Fill
		minY := plt.PY(plt.Y.Min)
		prev := math32.Vec2(ps[0].X, minY)
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
func (pts *Line) DataRange(plt *plot.Plot) (xmin, xmax, ymin, ymax float32) {
	return plot.XYRange(pts)
}

// Thumbnail returns the thumbnail for the LineTo, implementing the plot.Thumbnailer interface.
func (pts *Line) Thumbnail(plt *plot.Plot) {
	pc := plt.Paint
	ptb := pc.Bounds
	midY := 0.5 * float32(ptb.Min.Y+ptb.Max.Y)

	if pts.Fill != nil {
		tb := ptb
		if pts.LineStyle.Width.Value > 0 {
			tb.Min.Y = int(midY)
		}
		pc.FillBox(math32.FromPoint(tb.Min), math32.FromPoint(tb.Size()), pts.Fill)
	}

	if pts.LineStyle.SetStroke(plt) {
		pc.MoveTo(float32(ptb.Min.X), midY)
		pc.LineTo(float32(ptb.Max.X), midY)
		pc.Stroke()
	}
}
