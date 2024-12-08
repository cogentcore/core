// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package plots

import (
	"fmt"
	"image"
	"math"
	"math/rand"
	"os"
	"slices"
	"strconv"
	"testing"

	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/base/iox/imagex"
	"cogentcore.org/core/colors"
	"cogentcore.org/core/paint"
	"cogentcore.org/core/plot"
	"cogentcore.org/core/tensor"
	"cogentcore.org/core/tensor/table"
	"github.com/stretchr/testify/assert"
	"golang.org/x/exp/maps"
)

func ExampleLine() {
	xd, yd := make(plot.Values, 21), make(plot.Values, 21)
	for i := range xd {
		xd[i] = float64(i * 5)
		yd[i] = 50.0 + 40*math.Sin((float64(i)/8)*math.Pi)
	}
	data := plot.Data{plot.X: xd, plot.Y: yd}
	plt := plot.New()
	plt.Add(NewLine(data).Styler(func(s *plot.Style) {
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
		s.Line.Color = colors.Uniform(colors.Red)
		s.Point.Color = colors.Uniform(colors.Blue)
		s.Range.SetMin(0).SetMax(100)
	}))
	plt.Draw()
	imagex.Save(plt.Pixels, "testdata/ex_line_plot.png")
	// Output:
}

func ExampleStylerMetadata() {
	tx, ty := tensor.NewFloat64(21), tensor.NewFloat64(21)
	for i := range tx.DimSize(0) {
		tx.SetFloat1D(float64(i*5), i)
		ty.SetFloat1D(50.0+40*math.Sin((float64(i)/8)*math.Pi), i)
	}
	// attach stylers to the Y axis data: that is where plotter looks for it
	plot.SetStylersTo(ty, plot.Stylers{func(s *plot.Style) {
		s.Plot.Title = "Test Line"
		s.Plot.XAxis.Label = "X Axis"
		s.Plot.YAxisLabel = "Y Axis"
		s.Plot.Scale = 2
		s.Plot.XAxis.Range.SetMax(105)
		s.Plot.SetLinesOn(plot.On).SetPointsOn(plot.On)
		s.Line.Color = colors.Uniform(colors.Red)
		s.Point.Color = colors.Uniform(colors.Blue)
		s.Range.SetMin(0).SetMax(100)
	}})

	// somewhere else in the code:

	plt := plot.New()
	// NewLine automatically gets stylers from ty tensor metadata
	plt.Add(NewLine(plot.Data{plot.X: tx, plot.Y: ty}))
	plt.Draw()
	imagex.Save(plt.Pixels, "testdata/ex_styler_metadata.png")
	// Output:
}

func ExampleTable() {
	rand.Seed(1)
	n := 21
	tx, ty, th := tensor.NewFloat64(n), tensor.NewFloat64(n), tensor.NewFloat64(n)
	lbls := tensor.NewString(n)
	for i := range n {
		tx.SetFloat1D(float64(i*5), i)
		ty.SetFloat1D(50.0+40*math.Sin((float64(i)/8)*math.Pi), i)
		th.SetFloat1D(5*rand.Float64(), i)
		lbls.SetString1D(strconv.Itoa(i), i)
	}
	genst := func(s *plot.Style) {
		s.Plot.Title = "Test Table"
		s.Plot.XAxis.Label = "X Axis"
		s.Plot.YAxisLabel = "Y Axis"
		s.Plot.Scale = 2
		s.Plot.SetLinesOn(plot.On).SetPointsOn(plot.Off)
	}
	plot.SetStylersTo(ty, plot.Stylers{genst, func(s *plot.Style) {
		s.On = true
		s.Plotter = "XY"
		s.Role = plot.Y
		s.Line.Color = colors.Uniform(colors.Red)
		s.Range.SetMin(0).SetMax(100)
	}})
	// others get basic styling
	plot.SetStylersTo(tx, plot.Stylers{func(s *plot.Style) {
		s.Role = plot.X
	}})
	plot.SetStylersTo(th, plot.Stylers{func(s *plot.Style) {
		s.On = true
		s.Plotter = "YErrorBars"
		s.Role = plot.High
	}})
	plot.SetStylersTo(lbls, plot.Stylers{func(s *plot.Style) {
		s.On = true
		s.Plotter = "Labels"
		s.Role = plot.Label
		s.Text.Offset.X.Dp(6)
		s.Text.Offset.Y.Dp(-6)
	}})
	dt := table.New("Test Table") // todo: use Name by default for plot.
	dt.AddColumn("X", tx)
	dt.AddColumn("Y", ty)
	dt.AddColumn("High", th)
	dt.AddColumn("Labels", lbls)

	plt := errors.Log1(plot.NewTablePlot(dt))
	plt.Resize(image.Point{640, 480})
	plt.Draw()
	imagex.Save(plt.Pixels, "testdata/ex_table.png")
	// Output:
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
	return plot.Data{plot.X: xd, plot.Y: yd}
}

func sinDataXY() plot.Data {
	xd, yd := make(plot.Values, 21), make(plot.Values, 21)
	for i := range xd {
		xd[i] = float64(i * 5)
		yd[i] = float64(50) + 40*math.Sin((float64(i)/8)*math.Pi)
	}
	return plot.Data{plot.X: xd, plot.Y: yd}
}

func sinData() plot.Data {
	yd := make(plot.Values, 21)
	for i := range yd {
		x := float64(i % 21)
		yd[i] = float64(50) + 40*math.Sin((x/8)*math.Pi)
	}
	return plot.Data{plot.Y: yd}
}

func cosData() plot.Data {
	yd := make(plot.Values, 21)
	for i := range yd {
		x := float64(i % 21)
		yd[i] = float64(50) + 40*math.Cos((x/8)*math.Pi)
	}
	return plot.Data{plot.Y: yd}
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
	l2.Style.Text.Offset.X.Dp(6)
	l2.Style.Text.Offset.Y.Dp(-6)
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

	plt.Add(l2)
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

	stf := func(s *plot.Style) {
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
	}

	plt := plot.New()
	l1 := NewLine(data).Styler(stf)
	plt.Add(l1)
	plt.Legend.Add("Sine", l1) // todo: auto-add!
	plt.Legend.Add("Cos", l1)

	plt.Resize(image.Point{640, 480})
	plt.Draw()
	imagex.Assert(t, plt.Pixels, "style_line_point.png")

	plt = plot.New()
	tdy := tensor.NewFloat64FromValues(data[plot.Y].(plot.Values)...)
	plot.SetStylersTo(tdy, plot.Stylers{stf}) // set metadata for tensor
	tdx := tensor.NewFloat64FromValues(data[plot.X].(plot.Values)...)
	// NewLine auto-grabs from Y metadata
	l1 = NewLine(plot.Data{plot.X: tdx, plot.Y: tdy})
	plt.Add(l1)
	plt.Legend.Add("Sine", l1) // todo: auto-add!
	plt.Legend.Add("Cos", l1)
	plt.Resize(image.Point{640, 480})
	plt.Draw()
	imagex.Assert(t, plt.Pixels, "style_line_point_auto.png")
}

// todo: move into statplot and test everything

func TestTable(t *testing.T) {
	rand.Seed(1)
	n := 21
	tx, ty := tensor.NewFloat64(n), tensor.NewFloat64(n)
	tl, th := tensor.NewFloat64(n), tensor.NewFloat64(n)
	ts, tc := tensor.NewFloat64(n), tensor.NewFloat64(n)
	lbls := tensor.NewString(n)
	for i := range n {
		tx.SetFloat1D(float64(i*5), i)
		ty.SetFloat1D(50.0+40*math.Sin((float64(i)/8)*math.Pi), i)
		tl.SetFloat1D(5*rand.Float64(), i)
		th.SetFloat1D(5*rand.Float64(), i)
		ts.SetFloat1D(1+5*rand.Float64(), i)
		tc.SetFloat1D(float64(i), i)
		lbls.SetString1D(strconv.Itoa(i), i)
	}
	ptyps := maps.Keys(plot.Plotters)
	slices.Sort(ptyps)
	for _, ttyp := range ptyps {
		// attach stylers to the Y axis data: that is where plotter looks for it
		genst := func(s *plot.Style) {
			s.Plot.Title = "Test " + ttyp
			s.Plot.XAxis.Label = "X Axis"
			s.Plot.YAxisLabel = "Y Axis"
			s.Plotter = plot.PlotterName(ttyp)
			s.Plot.Scale = 2
			s.Plot.SetLinesOn(plot.On).SetPointsOn(plot.On)
			s.Line.Color = colors.Uniform(colors.Red)
			s.Point.Color = colors.Uniform(colors.Blue)
			s.Range.SetMin(0).SetMax(100)
		}
		plot.SetStylersTo(ty, plot.Stylers{genst, func(s *plot.Style) {
			s.On = true
			s.Role = plot.Y
			s.Group = "Y"
		}})
		// others get basic styling
		plot.SetStylersTo(tx, plot.Stylers{func(s *plot.Style) {
			s.Role = plot.X
			s.Group = "Y"
		}})
		plot.SetStylersTo(tl, plot.Stylers{func(s *plot.Style) {
			s.Role = plot.Low
			s.Group = "Y"
		}})
		plot.SetStylersTo(th, plot.Stylers{genst, func(s *plot.Style) {
			s.On = true
			s.Role = plot.High
			s.Group = "Y"
		}})
		plot.SetStylersTo(ts, plot.Stylers{func(s *plot.Style) {
			s.Role = plot.Size
			s.Group = "Y"
		}})
		plot.SetStylersTo(tc, plot.Stylers{func(s *plot.Style) {
			s.Role = plot.Color
			s.Group = "Y"
		}})
		plot.SetStylersTo(lbls, plot.Stylers{genst, func(s *plot.Style) {
			s.On = true
			s.Role = plot.Label
			s.Group = "Y"
		}})
		dt := table.New("Test Table") // todo: use Name by default for plot.
		dt.AddColumn("X", tx)
		dt.AddColumn("Y", ty)
		dt.AddColumn("Low", tl)
		dt.AddColumn("High", th)
		dt.AddColumn("Size", ts)
		dt.AddColumn("Color", tc)
		dt.AddColumn("Labels", lbls)

		plt, err := plot.NewTablePlot(dt)
		assert.NoError(t, err)
		plt.Resize(image.Point{640, 480})
		plt.Draw()
		fnm := "table_" + ttyp + ".png"
		imagex.Assert(t, plt.Pixels, fnm)
	}
}
