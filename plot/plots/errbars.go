// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package plots

import (
	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/plot"
)

//////////////////////////////////////////////////
// 	XErrorer

// XErrorer provides an interface for a list of Low, High error bar values.
// This is used in addition to an XYer interface, if implemented.
type XErrorer interface {
	// XError returns Low, High error values for X data.
	XError(i int) (low, high float32)
}

// Errors is a slice of low and high error values.
type Errors []struct{ Low, High float32 }

// XErrors implements the XErrorer interface.
type XErrors Errors

func (xe XErrors) XError(i int) (low, high float32) {
	return xe[i].Low, xe[i].High
}

// YErrorer provides an interface for YError method.
// This is used in addition to an XYer interface, if implemented.
type YErrorer interface {
	// YError returns two error values for Y data.
	YError(i int) (float32, float32)
}

// YErrors implements the YErrorer interface.
type YErrors Errors

func (ye YErrors) YError(i int) (float32, float32) {
	return ye[i].Low, ye[i].High
}

// YErrorBars implements the plot.Plotter, plot.DataRanger,
// and plot.GlyphBoxer interfaces, drawing vertical error
// bars, denoting error in Y values.
type YErrorBars struct {
	// XYs is a copy of the points for this line.
	plot.XYs

	// YErrors is a copy of the Y errors for each point.
	YErrors

	// PXYs is the actual pixel plotting coordinates for each XY value,
	// representing the high, center value of the error bar.
	PXYs plot.XYs

	// Style is the style for plotting.
	Style plot.Style

	stylers plot.Stylers
}

func (eb *YErrorBars) Defaults() {
	eb.Style.Defaults()
}

// NewYErrorBars returns a new YErrorBars plotter, or an error on failure.
// The error values from the YErrorer interface are interpreted as relative
// to the corresponding Y value. The errors for a given Y value are computed
// by taking the absolute value of the error returned by the YErrorer
// and subtracting the first and adding the second to the Y value.
func NewYErrorBars(yerrs interface {
	plot.XYer
	YErrorer
}) *YErrorBars {

	errs := make(YErrors, yerrs.Len())
	for i := range errs {
		errs[i].Low, errs[i].High = yerrs.YError(i)
		if err := plot.CheckFloats(errs[i].Low, errs[i].High); errors.Log(err) != nil {
			return nil
		}
	}
	xys, err := plot.CopyXYs(yerrs)
	if errors.Log(err) != nil {
		return nil
	}

	eb := &YErrorBars{
		XYs:     xys,
		YErrors: errs,
	}
	eb.Defaults()
	return eb
}

// Styler adds a style function to set style parameters.
func (eb *YErrorBars) Styler(f func(s *plot.Style)) *YErrorBars {
	eb.stylers.Add(f)
	return eb
}

func (eb *YErrorBars) ApplyStyle(ps *plot.PlotStyle) {
	ps.SetElementStyle(&eb.Style)
	eb.stylers.Run(&eb.Style)
}

func (eb *YErrorBars) Stylers() *plot.Stylers { return &eb.stylers }

func (eb *YErrorBars) XYData() (data plot.XYer, pixels plot.XYer) {
	data = eb.XYs
	pixels = eb.PXYs
	return
}

// Plot implements the Plotter interface, drawing labels.
func (eb *YErrorBars) Plot(plt *plot.Plot) {
	pc := plt.Paint
	uc := &pc.UnitContext

	eb.Style.Width.Cap.ToDots(uc)
	cw := 0.5 * eb.Style.Width.Cap.Dots
	nv := len(eb.YErrors)
	eb.PXYs = make(plot.XYs, nv)
	eb.Style.Line.SetStroke(plt)
	for i, err := range eb.YErrors {
		x := plt.PX(eb.XYs[i].X)
		ylow := plt.PY(eb.XYs[i].Y - math32.Abs(err.Low))
		yhigh := plt.PY(eb.XYs[i].Y + math32.Abs(err.High))

		eb.PXYs[i].X = x
		eb.PXYs[i].Y = yhigh

		pc.MoveTo(x, ylow)
		pc.LineTo(x, yhigh)

		pc.MoveTo(x-cw, ylow)
		pc.LineTo(x+cw, ylow)

		pc.MoveTo(x-cw, yhigh)
		pc.LineTo(x+cw, yhigh)
		pc.Stroke()
	}
}

// DataRange implements the plot.DataRanger interface.
func (eb *YErrorBars) DataRange(plt *plot.Plot) (xmin, xmax, ymin, ymax float32) {
	xmin, xmax = plot.Range(plot.XValues{eb})
	ymin = math32.Inf(1)
	ymax = math32.Inf(-1)
	for i, err := range eb.YErrors {
		y := eb.XYs[i].Y
		ylow := y - math32.Abs(err.Low)
		yhigh := y + math32.Abs(err.High)
		ymin = math32.Min(math32.Min(math32.Min(ymin, y), ylow), yhigh)
		ymax = math32.Max(math32.Max(math32.Max(ymax, y), ylow), yhigh)
	}
	return
}

// XErrorBars implements the plot.Plotter, plot.DataRanger,
// and plot.GlyphBoxer interfaces, drawing horizontal error
// bars, denoting error in Y values.
type XErrorBars struct {
	// XYs is a copy of the points for this line.
	plot.XYs

	// XErrors is a copy of the X errors for each point.
	XErrors

	// PXYs is the actual pixel plotting coordinates for each XY value,
	// representing the high, center value of the error bar.
	PXYs plot.XYs

	// Style is the style for plotting.
	Style plot.Style

	stylers plot.Stylers
}

// Returns a new XErrorBars plotter, or an error on failure. The error values
// from the XErrorer interface are interpreted as relative to the corresponding
// X value. The errors for a given X value are computed by taking the absolute
// value of the error returned by the XErrorer and subtracting the first and
// adding the second to the X value.
func NewXErrorBars(xerrs interface {
	plot.XYer
	XErrorer
}) *XErrorBars {

	errs := make(XErrors, xerrs.Len())
	for i := range errs {
		errs[i].Low, errs[i].High = xerrs.XError(i)
		if err := plot.CheckFloats(errs[i].Low, errs[i].High); errors.Log(err) != nil {
			return nil
		}
	}
	xys, err := plot.CopyXYs(xerrs)
	if errors.Log(err) != nil {
		return nil
	}

	eb := &XErrorBars{
		XYs:     xys,
		XErrors: errs,
	}
	eb.Defaults()
	return eb
}

func (eb *XErrorBars) Defaults() {
	eb.Style.Defaults()
}

// Styler adds a style function to set style parameters.
func (eb *XErrorBars) Styler(f func(s *plot.Style)) *XErrorBars {
	eb.stylers.Add(f)
	return eb
}

func (eb *XErrorBars) ApplyStyle(ps *plot.PlotStyle) {
	ps.SetElementStyle(&eb.Style)
	eb.stylers.Run(&eb.Style)
}

func (eb *XErrorBars) Stylers() *plot.Stylers { return &eb.stylers }

// Plot implements the Plotter interface, drawing labels.
func (eb *XErrorBars) Plot(plt *plot.Plot) {
	pc := plt.Paint
	uc := &pc.UnitContext

	eb.Style.Width.Cap.ToDots(uc)
	cw := 0.5 * eb.Style.Width.Cap.Dots

	nv := len(eb.XErrors)
	eb.PXYs = make(plot.XYs, nv)
	eb.Style.Line.SetStroke(plt)
	for i, err := range eb.XErrors {
		y := plt.PY(eb.XYs[i].Y)
		xlow := plt.PX(eb.XYs[i].X - math32.Abs(err.Low))
		xhigh := plt.PX(eb.XYs[i].X + math32.Abs(err.High))

		eb.PXYs[i].X = xhigh
		eb.PXYs[i].Y = y

		pc.MoveTo(xlow, y)
		pc.LineTo(xhigh, y)

		pc.MoveTo(xlow, y-cw)
		pc.LineTo(xlow, y+cw)

		pc.MoveTo(xhigh, y-cw)
		pc.LineTo(xhigh, y+cw)
		pc.Stroke()
	}
}

// DataRange implements the plot.DataRanger interface.
func (eb *XErrorBars) DataRange(plt *plot.Plot) (xmin, xmax, ymin, ymax float32) {
	ymin, ymax = plot.Range(plot.YValues{eb})
	xmin = math32.Inf(1)
	xmax = math32.Inf(-1)
	for i, err := range eb.XErrors {
		x := eb.XYs[i].X
		xlow := x - math32.Abs(err.Low)
		xhigh := x + math32.Abs(err.High)
		xmin = math32.Min(math32.Min(math32.Min(xmin, x), xlow), xhigh)
		xmax = math32.Max(math32.Max(math32.Max(xmax, x), xlow), xhigh)
	}
	return
}
