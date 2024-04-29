// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package plots

import (
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
