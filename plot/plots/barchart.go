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
	"image/color"

	"cogentcore.org/core/colors"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/plot"
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

	// Offset is offset added to each X axis value relative to the
	// Stride computed value (X = offset + index * Stride)
	// Defaults to 1.
	Offset float32

	// Stride is distance between bars. Defaults to 1.
	Stride float32

	// Width is the width of the bars, which should be less than
	// the Stride to prevent bar overlap.
	// Defaults to .8
	Width float32

	// Pad is additional space at start / end of data range, to keep bars from
	// overflowing ends.  This amount is subtracted from Offset
	// and added to (len(Values)-1)*Stride -- no other accommodation for bar
	// width is provided, so that should be built into this value as well.
	// Defaults to 1.
	Pad float32

	// Color is the fill color of the bars.
	Color color.Color

	// LineStyle is the style of the line connecting the points.
	// Use zero width to disable lines.
	LineStyle plot.LineStyle

	// Horizontal dictates whether the bars should be in the vertical
	// (default) or horizontal direction. If Horizontal is true, all
	// X locations and distances referred to here will actually be Y
	// locations and distances.
	Horizontal bool

	// stackedOn is the bar chart upon which this bar chart is stacked.
	StackedOn *BarChart
}

// NewBarChart returns a new bar chart with a single bar for each value.
// The bars heights correspond to the values and their x locations correspond
// to the index of their value in the Valuer.  Optional error-bar values can be
// provided.
func NewBarChart(vs, ers plot.Valuer) (*BarChart, error) {
	values, err := plot.CopyValues(vs)
	if err != nil {
		return nil, err
	}
	var errs plot.Values
	if ers != nil {
		errs, err = plot.CopyValues(ers)
		if err != nil {
			return nil, err
		}
	}
	b := &BarChart{
		Values: values,
		Errors: errs,
	}
	b.Defaults()
	return b, nil
}

func (b *BarChart) Defaults() {
	b.Offset = 1
	b.Stride = 1
	b.Width = .8
	b.Pad = 1
	b.Color = colors.ToUniform(colors.Scheme.OnSurface)
	b.LineStyle.Defaults()
}

func (b *BarChart) XYData() (data plot.XYer, pixels plot.XYer) {
	data = b.XYs
	pixels = b.PXYs
	return
}

// BarHeight returns the maximum y value of the
// ith bar, taking into account any bars upon
// which it is stacked.
func (b *BarChart) BarHeight(i int) float32 {
	ht := float32(0.0)
	if b == nil {
		return 0
	}
	if i >= 0 && i < len(b.Values) {
		ht += b.Values[i]
	}
	if b.StackedOn != nil {
		ht += b.StackedOn.BarHeight(i)
	}
	return ht
}

// StackOn stacks a bar chart on top of another,
// and sets the bar positioning params to that of the
// chart upon which it is being stacked.
func (b *BarChart) StackOn(on *BarChart) {
	b.Offset = on.Offset
	b.Stride = on.Stride
	b.Pad = on.Pad
	b.StackedOn = on
}

// Plot implements the plot.Plotter interface.
func (b *BarChart) Plot(plt *plot.Plot) {
	pc := plt.Paint
	if b.Color != nil {
		pc.FillStyle.Color = colors.Uniform(b.Color)
	} else {
		pc.FillStyle.Color = nil
	}
	b.LineStyle.SetStroke(plt)

	nv := len(b.Values)
	b.XYs = make(plot.XYs, nv)
	b.PXYs = make(plot.XYs, nv)

	hw := 0.5 * b.Width
	ew := b.Width / 3
	for i, ht := range b.Values {
		cat := b.Offset + float32(i)*b.Stride
		var bottom, catVal, catMin, catMax, valMin, valMax float32
		var box math32.Box2
		if b.Horizontal {
			catVal = plt.PY(cat)
			catMin = plt.PY(cat - hw)
			catMax = plt.PY(cat + hw)
			bottom = b.StackedOn.BarHeight(i) // nil safe
			valMin = plt.PX(bottom)
			valMax = plt.PX(bottom + ht)
			b.XYs[i] = math32.Vec2(bottom+ht, cat)
			b.PXYs[i] = math32.Vec2(valMax, catVal)
			box.Min.Set(valMin, catMin)
			box.Max.Set(valMax, catMax)
		} else {
			catVal = plt.PX(cat)
			catMin = plt.PX(cat - hw)
			catMax = plt.PX(cat + hw)
			bottom = b.StackedOn.BarHeight(i) // nil safe
			valMin = plt.PY(bottom)
			valMax = plt.PY(bottom + ht)
			b.XYs[i] = math32.Vec2(cat, bottom+ht)
			b.PXYs[i] = math32.Vec2(catVal, valMax)
			box.Min.Set(catMin, valMin)
			box.Max.Set(catMax, valMax)
		}

		pc.DrawRectangle(box.Min.X, box.Min.Y, box.Size().X, box.Size().Y)
		pc.FillStrokeClear()

		if i < len(b.Errors) {
			errval := b.Errors[i]
			if b.Horizontal {
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
}

// DataRange implements the plot.DataRanger interface.
func (b *BarChart) DataRange() (xmin, xmax, ymin, ymax float32) {
	catMin := b.Offset - b.Pad
	catMax := b.Offset + float32(len(b.Values)-1)*b.Stride + b.Pad

	valMin := math32.Inf(1)
	valMax := math32.Inf(-1)
	for i, val := range b.Values {
		valBot := b.StackedOn.BarHeight(i)
		valTop := valBot + val
		if i < len(b.Errors) {
			valTop += math32.Abs(b.Errors[i])
		}
		valMin = math32.Min(valMin, math32.Min(valBot, valTop))
		valMax = math32.Max(valMax, math32.Max(valBot, valTop))
	}
	if !b.Horizontal {
		return catMin, catMax, valMin, valMax
	}
	return valMin, valMax, catMin, catMax
}

// Thumbnail fulfills the plot.Thumbnailer interface.
func (b *BarChart) Thumbnail(plt *plot.Plot) {
	pc := plt.Paint
	if b.Color != nil {
		pc.FillStyle.Color = colors.Uniform(b.Color)
	} else {
		pc.FillStyle.Color = nil
	}
	b.LineStyle.SetStroke(plt)
	ptb := pc.Bounds
	pc.DrawRectangle(float32(ptb.Min.X), float32(ptb.Min.Y), float32(ptb.Size().X), float32(ptb.Size().Y))
	pc.FillStrokeClear()
}
