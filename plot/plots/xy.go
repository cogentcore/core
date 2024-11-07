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
	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/plot"
	"cogentcore.org/core/tensor"
)

// XY draws lines between and / or points for XY data values,
// based on Style properties.
type XY struct {
	// XYs is a copy of the points for this line.
	plot.XYs

	// PXYs is the actual pixel plotting coordinates for each XY value.
	PXYs plot.XYs

	// Style is the style for plotting.
	Style plot.Style

	stylers plot.Stylers
}

// NewLine returns an XY plot drawing Lines by default.
func NewLine(xys plot.XYer) *XY {
	data, err := plot.CopyXYs(xys)
	if errors.Log(err) != nil {
		return nil
	}
	ln := &XY{XYs: data}
	ln.Defaults()
	ln.Style.Line.On = plot.On
	ln.Style.Point.On = plot.Off
	return ln
}

// NewLineTensor returns an XY plot drawing Lines,
// using two tensors for X, Y values.
func NewLineTensor(x, y tensor.Tensor) *XY {
	return NewLine(plot.TensorXYs{X: x, Y: y})
}

// NewScatter returns an XY scatter plot drawing Points by default.
func NewScatter(xys plot.XYer) *XY {
	data, err := plot.CopyXYs(xys)
	if errors.Log(err) != nil {
		return nil
	}
	ln := &XY{XYs: data}
	ln.Defaults()
	ln.Style.Line.On = plot.Off
	ln.Style.Point.On = plot.On
	return ln
}

// NewScatterTensor returns an XY scatter plot drawing Points by default,
// using two tensors for X, Y values.
func NewScatterTensor(x, y tensor.Tensor) *XY {
	return NewScatter(plot.TensorXYs{X: x, Y: y})
}

func (ln *XY) Defaults() {
	ln.Style.Defaults()
}

// Styler adds a style function to set style parameters.
func (ln *XY) Styler(f func(s *plot.Style)) *XY {
	ln.stylers.Add(f)
	return ln
}

func (ln *XY) ApplyStyle() { ln.stylers.Run(&ln.Style) }

func (ln *XY) XYData() (data plot.XYer, pixels plot.XYer) {
	data = ln.XYs
	pixels = ln.PXYs
	return
}

// Plot does the drawing, implementing the plot.Plotter interface.
func (ln *XY) Plot(plt *plot.Plot) {
	pc := plt.Paint

	ps := plot.PlotXYs(plt, ln.XYs)
	np := len(ps)
	ln.PXYs = ps

	if ln.Style.Line.Fill != nil {
		pc.FillStyle.Color = ln.Style.Line.Fill
		minY := plt.PY(plt.Y.Min)
		prev := math32.Vec2(ps[0].X, minY)
		pc.MoveTo(prev.X, prev.Y)
		for i := range ps {
			pt := ps[i]
			switch ln.Style.Line.Step {
			case plot.NoStep:
				if pt.X < prev.X {
					pc.LineTo(prev.X, minY)
					pc.ClosePath()
					pc.MoveTo(pt.X, minY)
				}
				pc.LineTo(pt.X, pt.Y)
			case plot.PreStep:
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
			case plot.MidStep:
				if pt.X < prev.X {
					pc.LineTo(prev.X, minY)
					pc.ClosePath()
					pc.MoveTo(pt.X, minY)
				} else {
					pc.LineTo(0.5*(prev.X+pt.X), prev.Y)
					pc.LineTo(0.5*(prev.X+pt.X), pt.Y)
				}
				pc.LineTo(pt.X, pt.Y)
			case plot.PostStep:
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

	if ln.Style.Line.SetStroke(plt) {
		prev := ps[0]
		pc.MoveTo(prev.X, prev.Y)
		for i := 1; i < np; i++ {
			pt := ps[i]
			if ln.Style.Line.Step != plot.NoStep {
				if pt.X >= prev.X {
					switch ln.Style.Line.Step {
					case plot.PreStep:
						pc.LineTo(prev.X, pt.Y)
					case plot.MidStep:
						pc.LineTo(0.5*(prev.X+pt.X), prev.Y)
						pc.LineTo(0.5*(prev.X+pt.X), pt.Y)
					case plot.PostStep:
						pc.LineTo(pt.X, prev.Y)
					}
				} else {
					pc.MoveTo(pt.X, pt.Y)
				}
			}
			if !ln.Style.Line.NegativeX && pt.X < prev.X {
				pc.MoveTo(pt.X, pt.Y)
			} else {
				pc.LineTo(pt.X, pt.Y)
			}
			prev = pt
		}
		pc.Stroke()
	}
	if ln.Style.Point.SetStroke(plt) {
		for i := range ps {
			pt := ps[i]
			ln.Style.Point.DrawShape(pc, math32.Vec2(pt.X, pt.Y))
		}
	}
	pc.FillStyle.Color = nil
}

// DataRange returns the minimum and maximum
// x and y values, implementing the plot.DataRanger interface.
func (ln *XY) DataRange(plt *plot.Plot) (xmin, xmax, ymin, ymax float32) {
	return plot.XYRange(ln)
}

// Thumbnail returns the thumbnail, implementing the plot.Thumbnailer interface.
func (ln *XY) Thumbnail(plt *plot.Plot) {
	pc := plt.Paint
	ptb := pc.Bounds
	midY := 0.5 * float32(ptb.Min.Y+ptb.Max.Y)

	if ln.Style.Line.Fill != nil {
		tb := ptb
		if ln.Style.Line.Width.Value > 0 {
			tb.Min.Y = int(midY)
		}
		pc.FillBox(math32.FromPoint(tb.Min), math32.FromPoint(tb.Size()), ln.Style.Line.Fill)
	}

	if ln.Style.Line.SetStroke(plt) {
		pc.MoveTo(float32(ptb.Min.X), midY)
		pc.LineTo(float32(ptb.Max.X), midY)
		pc.Stroke()
	}

	if ln.Style.Point.SetStroke(plt) {
		midX := 0.5 * float32(ptb.Min.X+ptb.Max.X)
		ln.Style.Point.DrawShape(pc, math32.Vec2(midX, midY))
	}
	pc.FillStyle.Color = nil
}
