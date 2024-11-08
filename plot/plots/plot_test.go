// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package plots

import (
	"fmt"
	"image"
	"os"
	"testing"

	"cogentcore.org/core/base/iox/imagex"
	"cogentcore.org/core/colors"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/paint"
	"cogentcore.org/core/plot"
)

func ExampleLine() {
	data := make(plot.XYs, 42)
	for i := range data {
		x := float32(i % 21)
		data[i].X = x * 5
		if i < 21 {
			data[i].Y = float32(50) + 40*math32.Sin((x/8)*math32.Pi)
		} else {
			data[i].Y = float32(50) + 40*math32.Cos((x/8)*math32.Pi)
		}
	}

	plt := plot.New()
	plt.Add(NewLine(data).Styler(func(s *plot.Style) {
		s.Line.Color = colors.Uniform(colors.Red)
		s.Line.Width.Pt(2)
	}))
	plt.Draw()
	imagex.Save(plt.Pixels, "testdata/ex_line_plot.png")
	// Output:
}

func TestMain(m *testing.M) {
	paint.FontLibrary.InitFontPaths(paint.FontPaths...)
	os.Exit(m.Run())
}

func TestLine(t *testing.T) {
	plt := plot.New()
	plt.Title.Text = "Test Line"
	plt.X.Min = 0
	plt.X.Max = 100
	plt.X.Label.Text = "X Axis"
	plt.Y.Min = 0
	plt.Y.Max = 100
	plt.Y.Label.Text = "Y Axis"

	// note: making two overlapping series
	data := make(plot.XYs, 42)
	for i := range data {
		x := float32(i % 21)
		data[i].X = x * 5
		if i < 21 {
			data[i].Y = float32(50) + 40*math32.Sin((x/8)*math32.Pi)
		} else {
			data[i].Y = float32(50) + 40*math32.Cos((x/8)*math32.Pi)
		}
	}

	l1 := NewLine(data)
	if l1 == nil {
		t.Error("bad data")
	}
	plt.Add(l1)
	plt.Legend.Add("Sine", l1)
	plt.Legend.Add("Cos", l1)

	plt.Resize(image.Point{640, 480})
	plt.Draw()
	imagex.Assert(t, plt.Pixels, "line.png")

	l1.Style.Line.Fill = colors.Uniform(colors.Yellow)
	plt.Draw()
	imagex.Assert(t, plt.Pixels, "line-fill.png")

	l1.Style.Line.Step = plot.PreStep
	plt.Draw()
	imagex.Assert(t, plt.Pixels, "line-prestep.png")

	l1.Style.Line.Step = plot.MidStep
	plt.Draw()
	imagex.Assert(t, plt.Pixels, "line-midstep.png")

	l1.Style.Line.Step = plot.PostStep
	plt.Draw()
	imagex.Assert(t, plt.Pixels, "line-poststep.png")

	l1.Style.Line.Step = plot.NoStep
	l1.Style.Line.Fill = nil
	l1.Style.Line.NegativeX = true
	plt.Draw()
	imagex.Assert(t, plt.Pixels, "line-negx.png")
}

func TestScatter(t *testing.T) {
	plt := plot.New()
	plt.Title.Text = "Test Scatter"
	plt.X.Min = 0
	plt.X.Max = 100
	plt.X.Label.Text = "X Axis"
	plt.Y.Min = 0
	plt.Y.Max = 100
	plt.Y.Label.Text = "Y Axis"

	data := make(plot.XYs, 21)
	for i := range data {
		data[i].X = float32(i * 5)
		data[i].Y = float32(50) + 40*math32.Sin((float32(i)/8)*math32.Pi)
	}

	l1 := NewScatter(data)
	if l1 == nil {
		t.Error("bad data")
	}
	plt.Add(l1)

	plt.Resize(image.Point{640, 480})

	shs := plot.ShapesValues()
	for _, sh := range shs {
		l1.Style.Point.Shape = sh
		plt.Draw()
		imagex.Assert(t, plt.Pixels, "scatter-"+sh.String()+".png")
	}
}

func TestLabels(t *testing.T) {
	plt := plot.New()
	plt.Title.Text = "Test Labels"
	plt.X.Label.Text = "X Axis"
	plt.Y.Label.Text = "Y Axis"

	// note: making two overlapping series
	data := make(plot.XYs, 12)
	labels := make([]string, 12)
	for i := range data {
		x := float32(i % 21)
		data[i].X = x * 5
		data[i].Y = float32(50) + 40*math32.Sin((x/8)*math32.Pi)
		labels[i] = fmt.Sprintf("%7.4g", data[i].Y)
	}

	l1 := NewLine(data)
	if l1 == nil {
		t.Error("bad data")
	}
	l1.Style.Point.On = plot.On
	plt.Add(l1)
	plt.Legend.Add("Sine", l1)

	l2 := NewLabels(XYLabels{XYs: data, Labels: labels})
	if l2 == nil {
		t.Error("bad data")
	}
	l2.Style.Offset.X.Dp(6)
	l2.Style.Offset.Y.Dp(-6)
	plt.Add(l2)

	plt.Resize(image.Point{640, 480})
	plt.Draw()
	imagex.Assert(t, plt.Pixels, "labels.png")
}

func TestBarChart(t *testing.T) {
	plt := plot.New()
	plt.Title.Text = "Test Bar Chart"
	plt.X.Label.Text = "X Axis"
	plt.Y.Min = 0
	plt.Y.Max = 100
	plt.Y.Label.Text = "Y Axis"

	data := make(plot.Values, 21)
	for i := range data {
		x := float32(i % 21)
		data[i] = float32(50) + 40*math32.Sin((x/8)*math32.Pi)
	}

	cos := make(plot.Values, 21)
	for i := range data {
		x := float32(i % 21)
		cos[i] = float32(50) + 40*math32.Cos((x/8)*math32.Pi)
	}

	l1 := NewBarChart(data, nil)
	if l1 == nil {
		t.Error("bad data")
	}
	l1.Style.Line.Fill = colors.Uniform(colors.Red)
	plt.Add(l1)
	plt.Legend.Add("Sine", l1)

	plt.Resize(image.Point{640, 480})
	plt.Draw()
	imagex.Assert(t, plt.Pixels, "bar.png")

	l2 := NewBarChart(cos, nil)
	if l2 == nil {
		t.Error("bad data")
	}
	l2.Style.Line.Fill = colors.Uniform(colors.Blue)
	plt.Legend.Add("Cosine", l2)

	l1.Style.Width.Stride = 2
	l2.Style.Width.Stride = 2
	l2.Style.Width.Offset = 2

	plt.Add(l2) // note: range updated when added!
	plt.Draw()
	imagex.Assert(t, plt.Pixels, "bar-cos.png")
}

func TestBarChartErr(t *testing.T) {
	plt := plot.New()
	plt.Title.Text = "Test Bar Chart Errors"
	plt.X.Label.Text = "X Axis"
	plt.Y.Min = 0
	plt.Y.Max = 100
	plt.Y.Label.Text = "Y Axis"

	data := make(plot.Values, 21)
	for i := range data {
		x := float32(i % 21)
		data[i] = float32(50) + 40*math32.Sin((x/8)*math32.Pi)
	}

	cos := make(plot.Values, 21)
	for i := range data {
		x := float32(i % 21)
		cos[i] = float32(5) + 4*math32.Cos((x/8)*math32.Pi)
	}

	l1 := NewBarChart(data, cos)
	if l1 == nil {
		t.Error("bad data")
	}
	l1.Style.Line.Fill = colors.Uniform(colors.Red)
	plt.Add(l1)
	plt.Legend.Add("Sine", l1)

	plt.Resize(image.Point{640, 480})
	plt.Draw()
	imagex.Assert(t, plt.Pixels, "bar-err.png")

	l1.Horizontal = true
	plt.UpdateRange()
	plt.X.Min = 0
	plt.X.Max = 100
	plt.Draw()
	imagex.Assert(t, plt.Pixels, "bar-err-horiz.png")
}

func TestBarChartStack(t *testing.T) {
	plt := plot.New()
	plt.Title.Text = "Test Bar Chart Stacked"
	plt.X.Label.Text = "X Axis"
	plt.Y.Min = 0
	plt.Y.Max = 100
	plt.Y.Label.Text = "Y Axis"

	data := make(plot.Values, 21)
	for i := range data {
		x := float32(i % 21)
		data[i] = float32(50) + 40*math32.Sin((x/8)*math32.Pi)
	}

	cos := make(plot.Values, 21)
	for i := range data {
		x := float32(i % 21)
		cos[i] = float32(5) + 4*math32.Cos((x/8)*math32.Pi)
	}

	l1 := NewBarChart(data, nil)
	if l1 == nil {
		t.Error("bad data")
	}
	l1.Style.Line.Fill = colors.Uniform(colors.Red)
	plt.Add(l1)
	plt.Legend.Add("Sine", l1)

	l2 := NewBarChart(cos, nil)
	if l2 == nil {
		t.Error("bad data")
	}
	l2.Style.Line.Fill = colors.Uniform(colors.Blue)
	l2.StackedOn = l1
	plt.Add(l2)
	plt.Legend.Add("Cos", l2)

	plt.Resize(image.Point{640, 480})
	plt.Draw()
	imagex.Assert(t, plt.Pixels, "bar-stacked.png")
}

type XYErr struct {
	plot.XYs
	YErrors
}

func TestErrBar(t *testing.T) {
	plt := plot.New()
	plt.Title.Text = "Test Line Errors"
	plt.X.Label.Text = "X Axis"
	plt.Y.Min = 0
	plt.Y.Max = 100
	plt.Y.Label.Text = "Y Axis"

	data := make(plot.XYs, 21)
	for i := range data {
		x := float32(i % 21)
		data[i].X = x * 5
		data[i].Y = float32(50) + 40*math32.Sin((x/8)*math32.Pi)
	}

	yerr := make(YErrors, 21)
	for i := range yerr {
		x := float32(i % 21)
		yerr[i].High = float32(5) + 4*math32.Cos((x/8)*math32.Pi)
		yerr[i].Low = -yerr[i].High
	}

	xyerr := XYErr{XYs: data, YErrors: yerr}

	l1 := NewLine(data)
	if l1 == nil {
		t.Error("bad data")
	}
	plt.Add(l1)
	plt.Legend.Add("Sine", l1)

	l2 := NewYErrorBars(xyerr)
	if l2 == nil {
		t.Error("bad data")
	}
	plt.Add(l2)

	plt.Resize(image.Point{640, 480})
	plt.Draw()
	imagex.Assert(t, plt.Pixels, "errbar.png")
}
