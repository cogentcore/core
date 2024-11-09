// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package plots

import (
	"fmt"
	"image"
	"math"
	"os"
	"testing"

	"cogentcore.org/core/base/iox/imagex"
	"cogentcore.org/core/colors"
	"cogentcore.org/core/paint"
	"cogentcore.org/core/plot"
)

func ExampleLine() {
	// data := make(plot.XYs, 42)
	// for i := range data {
	// 	x := float32(i % 21)
	// 	data[i].X = x * 5
	// 	if i < 21 {
	// 		data[i].Y = float32(50) + 40*math32.Sin((x/8)*math32.Pi)
	// 	} else {
	// 		data[i].Y = float32(50) + 40*math32.Cos((x/8)*math32.Pi)
	// 	}
	// }
	//
	// plt := plot.New()
	// plt.Add(NewLine(data).Styler(func(s *plot.Style) {
	// 	s.Line.Color = colors.Uniform(colors.Red)
	// 	s.Line.Width.Pt(2)
	// }))
	// plt.Draw()
	// imagex.Save(plt.Pixels, "testdata/ex_line_plot.png")
	// // Output:
}

func TestMain(m *testing.M) {
	paint.FontLibrary.InitFontPaths(paint.FontPaths...)
	os.Exit(m.Run())
}

// sinCosWrapData returns overlapping sin / cos curves in one sequence.
func sinCosWrapData() plot.Data {
	xd, yd := make(plot.Values, 42), make(plot.Values, 42)
	for i := range xd {
		x := float64(i % 21)
		xd[i] = x * 5
		if i < 21 {
			yd[i] = float64(50) + 40*math.Sin((x/8)*math.Pi)
		} else {
			yd[i] = float64(50) + 40*math.Cos((x/8)*math.Pi)
		}
	}
	data := plot.Data{}
	data[plot.X] = xd
	data[plot.Y] = yd
	return data
}

func sinDataXY() plot.Data {
	xd, yd := make(plot.Values, 21), make(plot.Values, 21)
	for i := range xd {
		xd[i] = float64(i * 5)
		xd[i] = float64(50) + 40*math.Sin((float64(i)/8)*math.Pi)
	}
	data := plot.Data{}
	data[plot.X] = xd
	data[plot.Y] = yd
	return data
}

func sinData() plot.Data {
	yd := make(plot.Values, 21)
	for i := range yd {
		x := float64(i % 21)
		yd[i] = float64(50) + 40*math.Sin((x/8)*math.Pi)
	}
	data := plot.Data{}
	data[plot.Y] = yd
	return data
}

func cosData() plot.Data {
	yd := make(plot.Values, 21)
	for i := range yd {
		x := float64(i % 21)
		yd[i] = float64(50) + 40*math.Cos((x/8)*math.Pi)
	}
	data := plot.Data{}
	data[plot.Y] = yd
	return data
}

func TestLine(t *testing.T) {
	data := sinCosWrapData()

	plt := plot.New()
	plt.Title.Text = "Test Line"
	plt.X.Range.Min = 0
	plt.X.Range.Max = 100
	plt.X.Label.Text = "X Axis"
	plt.Y.Range.Min = 0
	plt.Y.Range.Max = 100
	plt.Y.Label.Text = "Y Axis"

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
	data := sinDataXY()

	plt := plot.New()
	plt.Title.Text = "Test Scatter"
	plt.X.Range.Min = 0
	plt.X.Range.Max = 100
	plt.X.Label.Text = "X Axis"
	plt.Y.Range.Min = 0
	plt.Y.Range.Max = 100
	plt.Y.Label.Text = "Y Axis"

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
	xd, yd := make(plot.Values, 12), make(plot.Values, 12)
	labels := make(plot.Labels, 12)
	for i := range xd {
		x := float64(i % 21)
		xd[i] = x * 5
		yd[i] = float64(50) + 40*math.Sin((x/8)*math.Pi)
		labels[i] = fmt.Sprintf("%7.4g", yd[i])
	}
	data := plot.Data{}
	data[plot.X] = xd
	data[plot.Y] = yd
	data[plot.Label] = labels

	l1 := NewLine(data)
	if l1 == nil {
		t.Error("bad data")
	}
	l1.Style.Point.On = plot.On
	plt.Add(l1)
	plt.Legend.Add("Sine", l1)

	l2 := NewLabels(data)
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

func TestBar(t *testing.T) {
	plt := plot.New()
	plt.Title.Text = "Test Bar Chart"
	plt.X.Label.Text = "X Axis"
	plt.Y.Range.Min = 0
	plt.Y.Range.Max = 100
	plt.Y.Label.Text = "Y Axis"

	data := sinData()
	cos := cosData()

	l1 := NewBar(data)
	if l1 == nil {
		t.Error("bad data")
	}
	l1.Style.Line.Fill = colors.Uniform(colors.Red)
	plt.Add(l1)
	plt.Legend.Add("Sine", l1)

	plt.Resize(image.Point{640, 480})
	plt.Draw()
	imagex.Assert(t, plt.Pixels, "bar.png")

	l2 := NewBar(cos)
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

func TestBarErr(t *testing.T) {
	plt := plot.New()
	plt.Title.Text = "Test Bar Chart Errors"
	plt.X.Label.Text = "X Axis"
	plt.Y.Range.Min = 0
	plt.Y.Range.Max = 100
	plt.Y.Label.Text = "Y Axis"

	data := sinData()
	cos := cosData()
	data[plot.High] = cos[plot.Y]

	l1 := NewBar(data)
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
	plt.X.Range.Min = 0
	plt.X.Range.Max = 100
	plt.Draw()
	imagex.Assert(t, plt.Pixels, "bar-err-horiz.png")
}

func TestBarStack(t *testing.T) {
	plt := plot.New()
	plt.Title.Text = "Test Bar Chart Stacked"
	plt.X.Label.Text = "X Axis"
	plt.Y.Range.Min = 0
	plt.Y.Range.Max = 100
	plt.Y.Label.Text = "Y Axis"

	data := sinData()
	cos := cosData()

	l1 := NewBar(data)
	if l1 == nil {
		t.Error("bad data")
	}
	l1.Style.Line.Fill = colors.Uniform(colors.Red)
	plt.Add(l1)
	plt.Legend.Add("Sine", l1)

	l2 := NewBar(cos)
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

func TestErrBar(t *testing.T) {
	plt := plot.New()
	plt.Title.Text = "Test Line Errors"
	plt.X.Label.Text = "X Axis"
	plt.Y.Range.Min = 0
	plt.Y.Range.Max = 100
	plt.Y.Label.Text = "Y Axis"

	xd, yd := make(plot.Values, 21), make(plot.Values, 21)
	for i := range xd {
		x := float64(i % 21)
		xd[i] = x * 5
		yd[i] = float64(50) + 40*math.Sin((x/8)*math.Pi)
	}

	low, high := make(plot.Values, 21), make(plot.Values, 21)
	for i := range low {
		x := float64(i % 21)
		high[i] = float64(5) + 4*math.Cos((x/8)*math.Pi)
		low[i] = -high[i]
	}

	data := plot.Data{plot.X: xd, plot.Y: yd, plot.Low: low, plot.High: high}

	l1 := NewLine(data)
	if l1 == nil {
		t.Error("bad data")
	}
	plt.Add(l1)
	plt.Legend.Add("Sine", l1)

	l2 := NewYErrorBars(data)
	if l2 == nil {
		t.Error("bad data")
	}
	plt.Add(l2)

	plt.Resize(image.Point{640, 480})
	plt.Draw()
	imagex.Assert(t, plt.Pixels, "errbar.png")
}

func TestStyle(t *testing.T) {
	data := sinCosWrapData()

	plt := plot.New()
	l1 := NewLine(data).Styler(func(s *plot.Style) {
		s.Plot.Title = "Test Line"
		s.Plot.XAxis.Label = "X Axis"
		s.Plot.YAxisLabel = "Y Axis"
		s.Plot.XAxis.Range.SetMax(105)
		s.Plot.LineWidth.Pt(2)
		s.Plot.SetLinesOn(plot.On).SetPointsOn(plot.On)
		s.Plot.TitleStyle.Size.Dp(48)
		s.Plot.Legend.Position.Left = true
		s.Plot.Legend.Text.Size.Dp(24)
		s.Plot.Axis.Text.Size.Dp(32)
		s.Plot.Axis.TickText.Size.Dp(24)
		s.Plot.XAxis.Rotation = -45
		// s.Line.On = plot.Off
		s.Line.Color = colors.Uniform(colors.Red)
		s.Point.Color = colors.Uniform(colors.Blue)
		s.Range.SetMax(100)
	})
	plt.Add(l1)
	plt.Legend.Add("Sine", l1)
	plt.Legend.Add("Cos", l1)

	plt.Resize(image.Point{640, 480})
	plt.Draw()
	imagex.Assert(t, plt.Pixels, "style_line_point.png")
}
