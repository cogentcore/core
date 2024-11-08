// Copyright (c) 2019, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// This is copied and modified directly from gonum to add better error-bar
// plotting for bar plots, along with multiple groups.

// Copyright ©2015 The Gonum Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package plots

import (
	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/plot"
	"cogentcore.org/core/tensor"
)

// A BarChart presents ordinally-organized data with rectangular bars
// with lengths proportional to the data values, and an optional
// error bar ("handle") at the top of the bar using given error value
// (single value, like a standard deviation etc, not drawn below the bar).
//
// Bars are plotted centered at integer multiples of Stride plus Start offset.
// Full data range also includes Pad value to extend range beyond edge bar centers.
// Bar Width is in data units, e.g., should be <= Stride.
// Defaults provide a unit-spaced plot.
type BarChart struct {
	// Values are the plotted values
	Values plot.Values

	// YErrors is a copy of the Y errors for each point.
	Errors plot.Values

	// XYs is the actual pixel plotting coordinates for each value.
	XYs plot.XYs

	// PXYs is the actual pixel plotting coordinates for each value.
	PXYs plot.XYs

	// Style has the properties used to render the bars.
	Style plot.Style

	// Horizontal dictates whether the bars should be in the vertical
	// (default) or horizontal direction. If Horizontal is true, all
	// X locations and distances referred to here will actually be Y
	// locations and distances.
	Horizontal bool

	// stackedOn is the bar chart upon which this bar chart is stacked.
	StackedOn *BarChart

	stylers plot.Stylers
}

// NewBarChart returns a new bar chart with a single bar for each value.
// The bars heights correspond to the values and their x locations correspond
// to the index of their value in the Valuer.
// Optional error-bar values can be provided.
func NewBarChart(vs, ers plot.Valuer) *BarChart {
	values, err := plot.CopyValues(vs)
	if errors.Log(err) != nil {
		return nil
	}
	var errs plot.Values
	if ers != nil {
		errs, err = plot.CopyValues(ers)
		if errors.Log(err) != nil {
			return nil
		}
	}
	bc := &BarChart{
		Values: values,
		Errors: errs,
	}
	bc.Defaults()
	return bc
}

// NewBarChartTensor returns a new bar chart with a single bar for each value.
// The bars heights correspond to the values and their x locations correspond
// to the index of their value in the Valuer.
// Optional error-bar values can be provided.
func NewBarChartTensor(vs, ers tensor.Tensor) *BarChart {
	vt := plot.TensorValues{vs}
	if ers == nil {
		return NewBarChart(vt, nil)
	}
	return NewBarChart(vt, plot.TensorValues{ers})
}

func (bc *BarChart) Defaults() {
	bc.Style.Defaults()
}

func (bc *BarChart) Styler(f func(s *plot.Style)) *BarChart {
	bc.stylers.Add(f)
	return bc
}

func (bc *BarChart) ApplyStyle(ps *plot.PlotStyle) {
	ps.SetElementStyle(&bc.Style)
	bc.stylers.Run(&bc.Style)
}

func (bc *BarChart) Stylers() *plot.Stylers { return &bc.stylers }

func (bc *BarChart) XYData() (data plot.XYer, pixels plot.XYer) {
	data = bc.XYs
	pixels = bc.PXYs
	return
}

// BarHeight returns the maximum y value of the
// ith bar, taking into account any bars upon
// which it is stacked.
func (bc *BarChart) BarHeight(i int) float32 {
	ht := float32(0.0)
	if bc == nil {
		return 0
	}
	if i >= 0 && i < len(bc.Values) {
		ht += bc.Values[i]
	}
	if bc.StackedOn != nil {
		ht += bc.StackedOn.BarHeight(i)
	}
	return ht
}

// StackOn stacks a bar chart on top of another,
// and sets the bar positioning options to that of the
// chart upon which it is being stacked.
func (bc *BarChart) StackOn(on *BarChart) {
	bc.Style.Width = on.Style.Width
	bc.StackedOn = on
}

// Plot implements the plot.Plotter interface.
func (bc *BarChart) Plot(plt *plot.Plot) {
	pc := plt.Paint
	bc.Style.Line.SetStroke(plt)
	pc.FillStyle.Color = bc.Style.Line.Fill
	bw := bc.Style.Width

	nv := len(bc.Values)
	bc.XYs = make(plot.XYs, nv)
	bc.PXYs = make(plot.XYs, nv)

	hw := 0.5 * bw.Width
	ew := bw.Width / 3
	for i, ht := range bc.Values {
		cat := bw.Offset + float32(i)*bw.Stride
		var bottom, catVal, catMin, catMax, valMin, valMax float32
		var box math32.Box2
		if bc.Horizontal {
			catVal = plt.PY(cat)
			catMin = plt.PY(cat - hw)
			catMax = plt.PY(cat + hw)
			bottom = bc.StackedOn.BarHeight(i) // nil safe
			valMin = plt.PX(bottom)
			valMax = plt.PX(bottom + ht)
			bc.XYs[i] = math32.Vec2(bottom+ht, cat)
			bc.PXYs[i] = math32.Vec2(valMax, catVal)
			box.Min.Set(valMin, catMin)
			box.Max.Set(valMax, catMax)
		} else {
			catVal = plt.PX(cat)
			catMin = plt.PX(cat - hw)
			catMax = plt.PX(cat + hw)
			bottom = bc.StackedOn.BarHeight(i) // nil safe
			valMin = plt.PY(bottom)
			valMax = plt.PY(bottom + ht)
			bc.XYs[i] = math32.Vec2(cat, bottom+ht)
			bc.PXYs[i] = math32.Vec2(catVal, valMax)
			box.Min.Set(catMin, valMin)
			box.Max.Set(catMax, valMax)
		}

		pc.DrawRectangle(box.Min.X, box.Min.Y, box.Size().X, box.Size().Y)
		pc.FillStrokeClear()

		if i < len(bc.Errors) {
			errval := bc.Errors[i]
			if bc.Horizontal {
				eVal := plt.PX(bottom + ht + math32.Abs(errval))
				pc.MoveTo(valMax, catVal)
				pc.LineTo(eVal, catVal)
				pc.MoveTo(eVal, plt.PY(cat-ew))
				pc.LineTo(eVal, plt.PY(cat+ew))
			} else {
				eVal := plt.PY(bottom + ht + math32.Abs(errval))
				pc.MoveTo(catVal, valMax)
				pc.LineTo(catVal, eVal)
				pc.MoveTo(plt.PX(cat-ew), eVal)
				pc.LineTo(plt.PX(cat+ew), eVal)
			}
			pc.Stroke()
		}
	}
	pc.FillStyle.Color = nil
}

// DataRange implements the plot.DataRanger interface.
func (bc *BarChart) DataRange(plt *plot.Plot) (xmin, xmax, ymin, ymax float32) {
	bw := bc.Style.Width
	catMin := bw.Offset - bw.Pad
	catMax := bw.Offset + float32(len(bc.Values)-1)*bw.Stride + bw.Pad

	valMin := math32.Inf(1)
	valMax := math32.Inf(-1)
	for i, val := range bc.Values {
		valBot := bc.StackedOn.BarHeight(i)
		valTop := valBot + val
		if i < len(bc.Errors) {
			valTop += math32.Abs(bc.Errors[i])
		}
		valMin = math32.Min(valMin, math32.Min(valBot, valTop))
		valMax = math32.Max(valMax, math32.Max(valBot, valTop))
	}
	if !bc.Horizontal {
		return catMin, catMax, valMin, valMax
	}
	return valMin, valMax, catMin, catMax
}

// Thumbnail fulfills the plot.Thumbnailer interface.
func (bc *BarChart) Thumbnail(plt *plot.Plot) {
	pc := plt.Paint
	bc.Style.Line.SetStroke(plt)
	pc.FillStyle.Color = bc.Style.Line.Fill
	ptb := pc.Bounds
	pc.DrawRectangle(float32(ptb.Min.X), float32(ptb.Min.Y), float32(ptb.Size().X), float32(ptb.Size().Y))
	pc.FillStrokeClear()
	pc.FillStyle.Color = nil
}
