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
	"math"

	"cogentcore.org/core/math32"
	"cogentcore.org/core/math32/minmax"
	"cogentcore.org/core/plot"
)

// BarType is be used for specifying the type name.
const BarType = "Bar"

func init() {
	plot.RegisterPlotter(BarType, "A Bar presents ordinally-organized data with rectangular bars with lengths proportional to the data values, and an optional error bar at the top of the bar using the High data role.", []plot.Roles{plot.Y}, []plot.Roles{plot.High}, func(data plot.Data) plot.Plotter {
		return NewBar(data)
	})
}

// A Bar presents ordinally-organized data with rectangular bars
// with lengths proportional to the data values, and an optional
// error bar ("handle") at the top of the bar using the High data role.
//
// Bars are plotted centered at integer multiples of Stride plus Start offset.
// Full data range also includes Pad value to extend range beyond edge bar centers.
// Bar Width is in data units, e.g., should be <= Stride.
// Defaults provide a unit-spaced plot.
type Bar struct {
	// copies of data
	Y, Err plot.Values

	// actual plotting X, Y values in data coordinates, taking into account stacking etc.
	X, Yp plot.Values

	// PX, PY are the actual pixel plotting coordinates for each XY value.
	PX, PY []float32

	// Style has the properties used to render the bars.
	Style plot.Style

	// Horizontal dictates whether the bars should be in the vertical
	// (default) or horizontal direction. If Horizontal is true, all
	// X locations and distances referred to here will actually be Y
	// locations and distances.
	Horizontal bool

	// stackedOn is the bar chart upon which this bar chart is stacked.
	StackedOn *Bar

	stylers plot.Stylers
}

// NewBar returns a new bar plotter with a single bar for each value.
// The bars heights correspond to the values and their x locations correspond
// to the index of their value in the Valuer.
// Optional error-bar values can be provided using the High data role.
// Styler functions are obtained from the Y metadata if present.
func NewBar(data plot.Data) *Bar {
	if data.CheckLengths() != nil {
		return nil
	}
	bc := &Bar{}
	bc.Y = plot.MustCopyRole(data, plot.Y)
	if bc.Y == nil {
		return nil
	}
	bc.stylers = plot.GetStylersFromData(data, plot.Y)
	bc.Err = plot.CopyRole(data, plot.High)
	bc.Defaults()
	return bc
}

func (bc *Bar) Defaults() {
	bc.Style.Defaults()
}

func (bc *Bar) Styler(f func(s *plot.Style)) *Bar {
	bc.stylers.Add(f)
	return bc
}

func (bc *Bar) ApplyStyle(ps *plot.PlotStyle) {
	ps.SetElementStyle(&bc.Style)
	bc.stylers.Run(&bc.Style)
}

func (bc *Bar) Stylers() *plot.Stylers { return &bc.stylers }

func (bc *Bar) Data() (data plot.Data, pixX, pixY []float32) {
	pixX = bc.PX
	pixY = bc.PY
	data = plot.Data{}
	data[plot.X] = bc.X
	data[plot.Y] = bc.Y
	if bc.Err != nil {
		data[plot.High] = bc.Err
	}
	return
}

// BarHeight returns the maximum y value of the
// ith bar, taking into account any bars upon
// which it is stacked.
func (bc *Bar) BarHeight(i int) float64 {
	ht := float64(0.0)
	if bc == nil {
		return 0
	}
	if i >= 0 && i < len(bc.Y) {
		ht += bc.Y[i]
	}
	if bc.StackedOn != nil {
		ht += bc.StackedOn.BarHeight(i)
	}
	return ht
}

// StackOn stacks a bar chart on top of another,
// and sets the bar positioning options to that of the
// chart upon which it is being stacked.
func (bc *Bar) StackOn(on *Bar) {
	bc.Style.Width = on.Style.Width
	bc.StackedOn = on
}

// Plot implements the plot.Plotter interface.
func (bc *Bar) Plot(plt *plot.Plot) {
	pc := plt.Paint
	bc.Style.Line.SetStroke(plt)
	pc.FillStyle.Color = bc.Style.Line.Fill
	bw := bc.Style.Width

	nv := len(bc.Y)
	bc.X = make(plot.Values, nv)
	bc.Yp = make(plot.Values, nv)
	bc.PX = make([]float32, nv)
	bc.PY = make([]float32, nv)

	hw := 0.5 * bw.Width
	ew := bw.Width / 3
	for i, ht := range bc.Y {
		cat := bw.Offset + float64(i)*bw.Stride
		var bottom float64
		var catVal, catMin, catMax, valMin, valMax float32
		var box math32.Box2
		if bc.Horizontal {
			catVal = plt.PY(cat)
			catMin = plt.PY(cat - hw)
			catMax = plt.PY(cat + hw)
			bottom = bc.StackedOn.BarHeight(i) // nil safe
			valMin = plt.PX(bottom)
			valMax = plt.PX(bottom + ht)
			bc.X[i] = bottom + ht
			bc.Yp[i] = cat
			bc.PX[i] = valMax
			bc.PY[i] = catVal
			box.Min.Set(valMin, catMin)
			box.Max.Set(valMax, catMax)
		} else {
			catVal = plt.PX(cat)
			catMin = plt.PX(cat - hw)
			catMax = plt.PX(cat + hw)
			bottom = bc.StackedOn.BarHeight(i) // nil safe
			valMin = plt.PY(bottom)
			valMax = plt.PY(bottom + ht)
			bc.X[i] = cat
			bc.Yp[i] = bottom + ht
			bc.PX[i] = catVal
			bc.PY[i] = valMax
			box.Min.Set(catMin, valMin)
			box.Max.Set(catMax, valMax)
		}

		pc.DrawRectangle(box.Min.X, box.Min.Y, box.Size().X, box.Size().Y)
		pc.FillStrokeClear()

		if i < len(bc.Err) {
			errval := bc.Err[i]
			if bc.Horizontal {
				eVal := plt.PX(bottom + ht + math.Abs(errval))
				pc.MoveTo(valMax, catVal)
				pc.LineTo(eVal, catVal)
				pc.MoveTo(eVal, plt.PY(cat-ew))
				pc.LineTo(eVal, plt.PY(cat+ew))
			} else {
				eVal := plt.PY(bottom + ht + math.Abs(errval))
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

// UpdateRange updates the given ranges.
func (bc *Bar) UpdateRange(plt *plot.Plot, xr, yr, zr *minmax.F64) {
	bw := bc.Style.Width
	catMin := bw.Offset - bw.Pad
	catMax := bw.Offset + float64(len(bc.Y)-1)*bw.Stride + bw.Pad

	for i, val := range bc.Y {
		valBot := bc.StackedOn.BarHeight(i)
		valTop := valBot + val
		if i < len(bc.Err) {
			valTop += math.Abs(bc.Err[i])
		}
		if bc.Horizontal {
			xr.FitValInRange(valBot)
			xr.FitValInRange(valTop)
		} else {
			yr.FitValInRange(valBot)
			yr.FitValInRange(valTop)
		}
	}
	if bc.Horizontal {
		xr.Min, xr.Max = bc.Style.Range.Clamp(xr.Min, xr.Max)
		yr.FitInRange(minmax.F64{catMin, catMax})
	} else {
		yr.Min, yr.Max = bc.Style.Range.Clamp(yr.Min, yr.Max)
		xr.FitInRange(minmax.F64{catMin, catMax})
	}
}

// Thumbnail fulfills the plot.Thumbnailer interface.
func (bc *Bar) Thumbnail(plt *plot.Plot) {
	pc := plt.Paint
	bc.Style.Line.SetStroke(plt)
	pc.FillStyle.Color = bc.Style.Line.Fill
	ptb := pc.Bounds
	pc.DrawRectangle(float32(ptb.Min.X), float32(ptb.Min.Y), float32(ptb.Size().X), float32(ptb.Size().Y))
	pc.FillStrokeClear()
	pc.FillStyle.Color = nil
}
