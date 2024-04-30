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

func TestMain(m *testing.M) {
	paint.FontLibrary.InitFontPaths(paint.FontPaths...)
	os.Exit(m.Run())
}

func TestLine(t *testing.T) {
	pt := plot.New()
	pt.Title.Text = "Test Line"
	pt.X.Min = 0
	pt.X.Max = 100
	pt.X.Label.Text = "X Axis"
	pt.Y.Min = 0
	pt.Y.Max = 100
	pt.Y.Label.Text = "Y Axis"

	// note: making two overlapping series
	data := make(XYs, 42)
	for i := range data {
		x := float32(i % 21)
		data[i].X = x * 5
		if i < 21 {
			data[i].Y = float32(50) + 40*math32.Sin((x/8)*math32.Pi)
		} else {
			data[i].Y = float32(50) + 40*math32.Cos((x/8)*math32.Pi)
		}
	}

	l1, err := NewLine(data)
	if err != nil {
		t.Error(err.Error())
	}
	pt.Add(l1)
	pt.Legend.Add("Sine", l1)
	pt.Legend.Add("Cos", l1)

	pt.Resize(image.Point{640, 480})
	pt.Draw()
	imagex.Assert(t, pt.Pixels, "line.png")

	l1.FillColor = colors.Yellow
	pt.Draw()
	imagex.Assert(t, pt.Pixels, "line_fill.png")

	l1.StepStyle = PreStep
	pt.Draw()
	imagex.Assert(t, pt.Pixels, "line_prestep.png")

	l1.StepStyle = MidStep
	pt.Draw()
	imagex.Assert(t, pt.Pixels, "line_midstep.png")

	l1.StepStyle = PostStep
	pt.Draw()
	imagex.Assert(t, pt.Pixels, "line_poststep.png")

	l1.StepStyle = NoStep
	l1.FillColor = nil
	l1.NegativeXDraw = true
	pt.Draw()
	imagex.Assert(t, pt.Pixels, "line_negx.png")

}

func TestScatter(t *testing.T) {
	pt := plot.New()
	pt.Title.Text = "Test Scatter"
	pt.X.Min = 0
	pt.X.Max = 100
	pt.X.Label.Text = "X Axis"
	pt.Y.Min = 0
	pt.Y.Max = 100
	pt.Y.Label.Text = "Y Axis"

	data := make(XYs, 21)
	for i := range data {
		data[i].X = float32(i * 5)
		data[i].Y = float32(50) + 40*math32.Sin((float32(i)/8)*math32.Pi)
	}

	l1, err := NewScatter(data)
	if err != nil {
		t.Error(err.Error())
	}
	pt.Add(l1)

	pt.Resize(image.Point{640, 480})

	shs := ShapesValues()
	for _, sh := range shs {
		l1.PointShape = sh
		pt.Draw()
		imagex.Assert(t, pt.Pixels, "scatter_"+sh.String()+".png")
	}
}

func TestLabels(t *testing.T) {
	pt := plot.New()
	pt.Title.Text = "Test Labels"
	pt.X.Label.Text = "X Axis"
	pt.Y.Label.Text = "Y Axis"

	// note: making two overlapping series
	data := make(XYs, 12)
	labels := make([]string, 12)
	for i := range data {
		x := float32(i % 21)
		data[i].X = x * 5
		data[i].Y = float32(50) + 40*math32.Sin((x/8)*math32.Pi)
		labels[i] = fmt.Sprintf("%7.4g", data[i].Y)
	}

	l1, sc, err := NewLinePoints(data)
	if err != nil {
		t.Error(err.Error())
	}
	pt.Add(l1)
	pt.Add(sc)
	pt.Legend.Add("Sine", l1, sc)

	l2, err := NewLabels(XYLabels{XYs: data, Labels: labels})
	if err != nil {
		t.Error(err.Error())
	}
	l2.Offset.X.Dp(6)
	l2.Offset.Y.Dp(-20)
	pt.Add(l2)

	pt.Resize(image.Point{640, 480})
	pt.Draw()
	imagex.Assert(t, pt.Pixels, "labels.png")
}

func TestBarChart(t *testing.T) {
	pt := plot.New()
	pt.Title.Text = "Test Bar Chart"
	pt.X.Label.Text = "X Axis"
	pt.Y.Min = 0
	pt.Y.Max = 100
	pt.Y.Label.Text = "Y Axis"

	data := make(Values, 21)
	for i := range data {
		x := float32(i % 21)
		data[i] = float32(50) + 40*math32.Sin((x/8)*math32.Pi)
	}

	cos := make(Values, 21)
	for i := range data {
		x := float32(i % 21)
		cos[i] = float32(50) + 40*math32.Cos((x/8)*math32.Pi)
	}

	l1, err := NewBarChart(data, nil)
	if err != nil {
		t.Error(err.Error())
	}
	l1.Color = colors.Red
	pt.Add(l1)
	pt.Legend.Add("Sine", l1)

	pt.Resize(image.Point{640, 480})
	pt.Draw()
	imagex.Assert(t, pt.Pixels, "bar.png")

	l2, err := NewBarChart(cos, nil)
	if err != nil {
		t.Error(err.Error())
	}
	l2.Color = colors.Blue
	pt.Legend.Add("Cosine", l2)

	l1.Stride = 2
	l2.Stride = 2
	l2.Offset = 2

	pt.Add(l2) // note: range updated when added!
	pt.Draw()
	imagex.Assert(t, pt.Pixels, "bar_cos.png")
}

func TestBarChartErr(t *testing.T) {
	pt := plot.New()
	pt.Title.Text = "Test Bar Chart Errors"
	pt.X.Label.Text = "X Axis"
	pt.Y.Min = 0
	pt.Y.Max = 100
	pt.Y.Label.Text = "Y Axis"

	data := make(Values, 21)
	for i := range data {
		x := float32(i % 21)
		data[i] = float32(50) + 40*math32.Sin((x/8)*math32.Pi)
	}

	cos := make(Values, 21)
	for i := range data {
		x := float32(i % 21)
		cos[i] = float32(5) + 4*math32.Cos((x/8)*math32.Pi)
	}

	l1, err := NewBarChart(data, cos)
	if err != nil {
		t.Error(err.Error())
	}
	l1.Color = colors.Red
	pt.Add(l1)
	pt.Legend.Add("Sine", l1)

	pt.Resize(image.Point{640, 480})
	pt.Draw()
	imagex.Assert(t, pt.Pixels, "bar_err.png")

	l1.Horizontal = true
	pt.UpdateRange()
	pt.X.Min = 0
	pt.X.Max = 100
	pt.Draw()
	imagex.Assert(t, pt.Pixels, "bar_err_horiz.png")
}

func TestBarChartStack(t *testing.T) {
	pt := plot.New()
	pt.Title.Text = "Test Bar Chart Stacked"
	pt.X.Label.Text = "X Axis"
	pt.Y.Min = 0
	pt.Y.Max = 100
	pt.Y.Label.Text = "Y Axis"

	data := make(Values, 21)
	for i := range data {
		x := float32(i % 21)
		data[i] = float32(50) + 40*math32.Sin((x/8)*math32.Pi)
	}

	cos := make(Values, 21)
	for i := range data {
		x := float32(i % 21)
		cos[i] = float32(5) + 4*math32.Cos((x/8)*math32.Pi)
	}

	l1, err := NewBarChart(data, nil)
	if err != nil {
		t.Error(err.Error())
	}
	l1.Color = colors.Red
	pt.Add(l1)
	pt.Legend.Add("Sine", l1)

	l2, err := NewBarChart(cos, nil)
	if err != nil {
		t.Error(err.Error())
	}
	l2.Color = colors.Blue
	l2.StackedOn = l1
	pt.Add(l2)
	pt.Legend.Add("Cos", l2)

	pt.Resize(image.Point{640, 480})
	pt.Draw()
	imagex.Assert(t, pt.Pixels, "bar_stacked.png")
}
