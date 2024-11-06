// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package plots

import (
	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/plot"
	"cogentcore.org/core/styles/units"
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

	// Line is the style used to draw the error bars.
	Line plot.LineStyle

	// CapWidth is the width of the caps drawn at the top of each error bar.
	CapWidth units.Value

	stylers plot.Stylers
}

func (eb *YErrorBars) Defaults() {
	eb.Line.Defaults()
	eb.CapWidth.Dp(10)
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
func (e *YErrorBars) Styler(f func(s *YErrorBars)) *YErrorBars {
	e.stylers.Add(func(p plot.Plotter) { f(p.(*YErrorBars)) })
	return e
}

func (e *YErrorBars) ApplyStyle() { e.stylers.Run(e) }

func (e *YErrorBars) XYData() (data plot.XYer, pixels plot.XYer) {
	data = e.XYs
	pixels = e.PXYs
	return
}

// Plot implements the Plotter interface, drawing labels.
func (e *YErrorBars) Plot(plt *plot.Plot) {
	pc := plt.Paint
	uc := &pc.UnitContext

	e.CapWidth.ToDots(uc)
	cw := 0.5 * e.CapWidth.Dots
	nv := len(e.YErrors)
	e.PXYs = make(plot.XYs, nv)
	e.Line.SetStroke(plt)
	for i, err := range e.YErrors {
		x := plt.PX(e.XYs[i].X)
		ylow := plt.PY(e.XYs[i].Y - math32.Abs(err.Low))
		yhigh := plt.PY(e.XYs[i].Y + math32.Abs(err.High))

		e.PXYs[i].X = x
		e.PXYs[i].Y = yhigh

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
func (e *YErrorBars) DataRange(plt *plot.Plot) (xmin, xmax, ymin, ymax float32) {
	xmin, xmax = plot.Range(plot.XValues{e})
	ymin = math32.Inf(1)
	ymax = math32.Inf(-1)
	for i, err := range e.YErrors {
		y := e.XYs[i].Y
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

	// Line is the style used to draw the error bars.
	Line plot.LineStyle

	// CapWidth is the width of the caps drawn at the top
	// of each error bar.
	CapWidth units.Value

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
	eb.Line.Defaults()
	eb.CapWidth.Dp(10)
}

// Styler adds a style function to set style parameters.
func (e *XErrorBars) Styler(f func(s *XErrorBars)) *XErrorBars {
	e.stylers.Add(func(p plot.Plotter) { f(p.(*XErrorBars)) })
	return e
}

func (e *XErrorBars) ApplyStyle() { e.stylers.Run(e) }

func (e *XErrorBars) XYData() (data plot.XYer, pixels plot.XYer) {
	data = e.XYs
	pixels = e.PXYs
	return
}

// Plot implements the Plotter interface, drawing labels.
func (e *XErrorBars) Plot(plt *plot.Plot) {
	pc := plt.Paint
	uc := &pc.UnitContext

	e.CapWidth.ToDots(uc)
	cw := 0.5 * e.CapWidth.Dots

	nv := len(e.XErrors)
	e.PXYs = make(plot.XYs, nv)
	e.Line.SetStroke(plt)
	for i, err := range e.XErrors {
		y := plt.PY(e.XYs[i].Y)
		xlow := plt.PX(e.XYs[i].X - math32.Abs(err.Low))
		xhigh := plt.PX(e.XYs[i].X + math32.Abs(err.High))

		e.PXYs[i].X = xhigh
		e.PXYs[i].Y = y

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
func (e *XErrorBars) DataRange(plt *plot.Plot) (xmin, xmax, ymin, ymax float32) {
	ymin, ymax = plot.Range(plot.YValues{e})
	xmin = math32.Inf(1)
	xmax = math32.Inf(-1)
	for i, err := range e.XErrors {
		x := e.XYs[i].X
		xlow := x - math32.Abs(err.Low)
		xhigh := x + math32.Abs(err.High)
		xmin = math32.Min(math32.Min(math32.Min(xmin, x), xlow), xhigh)
		xmax = math32.Max(math32.Max(math32.Max(xmax, x), xlow), xhigh)
	}
	return
}
