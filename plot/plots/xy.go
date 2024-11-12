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
	"cogentcore.org/core/math32"
	"cogentcore.org/core/math32/minmax"
	"cogentcore.org/core/plot"
)

// XYType is be used for specifying the type name.
const XYType = "XY"

func init() {
	plot.RegisterPlotter(XYType, "draws lines between and / or points for X,Y data values, using optional Size and Color data for the points, for a bubble plot.", []plot.Roles{plot.X, plot.Y}, []plot.Roles{plot.Size, plot.Color}, func(data plot.Data) plot.Plotter {
		return NewXY(data)
	})
}

// XY draws lines between and / or points for XY data values.
type XY struct {
	// copies of data for this line
	X, Y, Color, Size plot.Values

	// PX, PY are the actual pixel plotting coordinates for each XY value.
	PX, PY []float32

	// Style is the style for plotting.
	Style plot.Style

	stylers plot.Stylers
}

// NewXY returns an XY plotter for given X, Y data.
// data can also include Color and / or Size for the points.
// Styler functions are obtained from the Y metadata if present.
func NewXY(data plot.Data) *XY {
	if data.CheckLengths() != nil {
		return nil
	}
	ln := &XY{}
	ln.X = plot.MustCopyRole(data, plot.X)
	ln.Y = plot.MustCopyRole(data, plot.Y)
	if ln.X == nil || ln.Y == nil {
		return nil
	}
	ln.stylers = plot.GetStylersFromData(data, plot.Y)
	ln.Color = plot.CopyRole(data, plot.Color)
	ln.Size = plot.CopyRole(data, plot.Size)
	ln.Defaults()
	return ln
}

// NewLine returns an XY plot drawing Lines by default.
func NewLine(data plot.Data) *XY {
	ln := NewXY(data)
	if ln == nil {
		return ln
	}
	ln.Style.Line.On = plot.On
	ln.Style.Point.On = plot.Off
	return ln
}

// NewScatter returns an XY scatter plot drawing Points by default.
func NewScatter(data plot.Data) *XY {
	ln := NewXY(data)
	if ln == nil {
		return ln
	}
	ln.Style.Line.On = plot.Off
	ln.Style.Point.On = plot.On
	return ln
}

func (ln *XY) Defaults() {
	ln.Style.Defaults()
}

// Styler adds a style function to set style parameters.
func (ln *XY) Styler(f func(s *plot.Style)) *XY {
	ln.stylers.Add(f)
	return ln
}

func (ln *XY) Stylers() *plot.Stylers { return &ln.stylers }

func (ln *XY) ApplyStyle(ps *plot.PlotStyle) {
	ps.SetElementStyle(&ln.Style)
	ln.stylers.Run(&ln.Style)
}

func (ln *XY) Data() (data plot.Data, pixX, pixY []float32) {
	pixX = ln.PX
	pixY = ln.PY
	data = plot.Data{}
	data[plot.X] = ln.X
	data[plot.Y] = ln.Y
	if ln.Size != nil {
		data[plot.Size] = ln.Size
	}
	if ln.Color != nil {
		data[plot.Color] = ln.Color
	}
	return
}

// Plot does the drawing, implementing the plot.Plotter interface.
func (ln *XY) Plot(plt *plot.Plot) {
	pc := plt.Paint
	ln.PX = plot.PlotX(plt, ln.X)
	ln.PY = plot.PlotY(plt, ln.Y)
	np := len(ln.PX)

	if ln.Style.Line.Fill != nil {
		pc.FillStyle.Color = ln.Style.Line.Fill
		minY := plt.PY(plt.Y.Range.Min)
		prevX := ln.PX[0]
		prevY := minY
		pc.MoveTo(prevX, prevY)
		for i, ptx := range ln.PX {
			pty := ln.PY[i]
			switch ln.Style.Line.Step {
			case plot.NoStep:
				if ptx < prevX {
					pc.LineTo(prevX, minY)
					pc.ClosePath()
					pc.MoveTo(ptx, minY)
				}
				pc.LineTo(ptx, pty)
			case plot.PreStep:
				if i == 0 {
					continue
				}
				if ptx < prevX {
					pc.LineTo(prevX, minY)
					pc.ClosePath()
					pc.MoveTo(ptx, minY)
				} else {
					pc.LineTo(prevX, pty)
				}
				pc.LineTo(ptx, pty)
			case plot.MidStep:
				if ptx < prevX {
					pc.LineTo(prevX, minY)
					pc.ClosePath()
					pc.MoveTo(ptx, minY)
				} else {
					pc.LineTo(0.5*(prevX+ptx), prevY)
					pc.LineTo(0.5*(prevX+ptx), pty)
				}
				pc.LineTo(ptx, pty)
			case plot.PostStep:
				if ptx < prevX {
					pc.LineTo(prevX, minY)
					pc.ClosePath()
					pc.MoveTo(ptx, minY)
				} else {
					pc.LineTo(ptx, prevY)
				}
				pc.LineTo(ptx, pty)
			}
			prevX, prevY = ptx, pty
		}
		pc.LineTo(prevX, minY)
		pc.ClosePath()
		pc.Fill()
	}
	pc.FillStyle.Color = nil

	if ln.Style.Line.SetStroke(plt) {
		prevX, prevY := ln.PX[0], ln.PY[0]
		pc.MoveTo(prevX, prevY)
		for i := 1; i < np; i++ {
			ptx, pty := ln.PX[i], ln.PY[i]
			if ln.Style.Line.Step != plot.NoStep {
				if ptx >= prevX {
					switch ln.Style.Line.Step {
					case plot.PreStep:
						pc.LineTo(prevX, pty)
					case plot.MidStep:
						pc.LineTo(0.5*(prevX+ptx), prevY)
						pc.LineTo(0.5*(prevX+ptx), pty)
					case plot.PostStep:
						pc.LineTo(ptx, prevY)
					}
				} else {
					pc.MoveTo(ptx, pty)
				}
			}
			if !ln.Style.Line.NegativeX && ptx < prevX {
				pc.MoveTo(ptx, pty)
			} else {
				pc.LineTo(ptx, pty)
			}
			prevX, prevY = ptx, pty
		}
		pc.Stroke()
	}
	if ln.Style.Point.SetStroke(plt) {
		for i, ptx := range ln.PX {
			pty := ln.PY[i]
			ln.Style.Point.DrawShape(pc, math32.Vec2(ptx, pty))
		}
	}
	pc.FillStyle.Color = nil
}

// UpdateRange updates the given ranges.
func (ln *XY) UpdateRange(plt *plot.Plot, xr, yr, zr *minmax.F64) {
	// todo: include point sizes!
	plot.Range(ln.X, xr)
	plot.RangeClamp(ln.Y, yr, &ln.Style.Range)
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
