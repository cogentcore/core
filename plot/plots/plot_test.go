// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package plots

import (
	"image"
	"os"
	"testing"

	"cogentcore.org/core/colors"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/paint"
	"cogentcore.org/core/plot"
)

func TestMain(m *testing.M) {
	paint.FontLibrary.InitFontPaths(paint.FontPaths...)
	os.Exit(m.Run())
}

func TestPlot(t *testing.T) {
	pt := plot.New()
	pt.Title.Text = "Test Plot"
	pt.X.Min = 0
	pt.X.Max = 100
	pt.X.Label.Text = "X Axis"
	pt.Y.Min = 0
	pt.Y.Max = 100
	pt.Y.Label.Text = "Y Axis"

	data := make(XYs, 21)
	for i := range data {
		data[i].X = float32(i * 5)
		data[i].Y = float32(50) + 60*math32.Sin((float32(i)/8)*math32.Pi)
	}

	// data[8].Y = math32.NaN()

	l1, err := NewLine(data)
	if err != nil {
		t.Error(err.Error())
	}
	// l1.StepStyle = PostStep
	l1.FillColor = colors.Yellow
	pt.Add(l1)

	pt.Resize(image.Point{640, 480})
	pt.Draw()
	pt.SaveImage("test1.png")
}
