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

	"cogentcore.org/core/base/errors"
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

	// Line is the style of the line connecting the points.
	// Use zero width to disable lines.
	Line plot.LineStyle

	// Fill is the color to fill the area below the plot.
	// Use nil to disable filling, which is the default.
	Fill image.Image

	// if true, draw lines that connect points with a negative X-axis direction;
	// otherwise there is a break in the line.
	// default is false, so that repeated series of data across the X axis
	// are plotted separately.
	NegativeXDraw bool

	stylers plot.Stylers
}

// NewLine returns a Line that uses the default line style and
// does not draw glyphs.
func NewLine(xys plot.XYer) *Line {
	data, err := plot.CopyXYs(xys)
	if errors.Log(err) != nil {
		return nil
	}
	ln := &Line{XYs: data}
	ln.Defaults()
	return ln
}

// NewLinePoints returns both a Line and a
// Scatter plot for the given point data.
func NewLinePoints(xys plot.XYer) (*Line, *Scatter) {
	sc := NewScatter(xys)
	ln := &Line{XYs: sc.XYs}
	ln.Defaults()
	return ln, sc
}

func (ln *Line) Defaults() {
	ln.Line.Defaults()
}

// Styler adds a style function to set style parameters.
func (ln *Line) Styler(f func(s *Line)) *Line {
	ln.stylers.Add(func(p plot.Plotter) { f(p.(*Line)) })
	return ln
}

func (ln *Line) ApplyStyle() { ln.stylers.Run(ln) }

func (ln *Line) XYData() (data plot.XYer, pixels plot.XYer) {
	data = ln.XYs
	pixels = ln.PXYs
	return
}

// Plot draws the Line, implementing the plot.Plotter interface.
func (ln *Line) Plot(plt *plot.Plot) {
	pc := plt.Paint

	ps := plot.PlotXYs(plt, ln.XYs)
	np := len(ps)
	ln.PXYs = ps

	if ln.Fill != nil {
		pc.FillStyle.Color = ln.Fill
		minY := plt.PY(plt.Y.Min)
		prev := math32.Vec2(ps[0].X, minY)
		pc.MoveTo(prev.X, prev.Y)
		for i := range ps {
			pt := ps[i]
			switch ln.StepStyle {
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

	if !ln.Line.SetStroke(plt) {
		return
	}
	prev := ps[0]
	pc.MoveTo(prev.X, prev.Y)
	for i := 1; i < np; i++ {
		pt := ps[i]
		if ln.StepStyle != NoStep {
			if pt.X >= prev.X {
				switch ln.StepStyle {
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
		if !ln.NegativeXDraw && pt.X < prev.X {
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
func (ln *Line) DataRange(plt *plot.Plot) (xmin, xmax, ymin, ymax float32) {
	return plot.XYRange(ln)
}

// Thumbnail returns the thumbnail for the LineTo, implementing the plot.Thumbnailer interface.
func (ln *Line) Thumbnail(plt *plot.Plot) {
	pc := plt.Paint
	ptb := pc.Bounds
	midY := 0.5 * float32(ptb.Min.Y+ptb.Max.Y)

	if ln.Fill != nil {
		tb := ptb
		if ln.Line.Width.Value > 0 {
			tb.Min.Y = int(midY)
		}
		pc.FillBox(math32.FromPoint(tb.Min), math32.FromPoint(tb.Size()), ln.Fill)
	}

	if ln.Line.SetStroke(plt) {
		pc.MoveTo(float32(ptb.Min.X), midY)
		pc.LineTo(float32(ptb.Max.X), midY)
		pc.Stroke()
	}
}
