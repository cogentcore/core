// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package plot

import (
	"image"
	"os"
	"testing"

	"cogentcore.org/core/base/iox/imagex"
	"cogentcore.org/core/paint"
)

func TestMain(m *testing.M) {
	paint.FontLibrary.InitFontPaths(paint.FontPaths...)
	os.Exit(m.Run())
}

func TestPlot(t *testing.T) {
	pt := New()
	pt.Title.Text = "Test Plot"
	pt.X.Min = 0
	pt.X.Max = 100
	pt.X.Label.Text = "X Axis"
	pt.Y.Min = 0
	pt.Y.Max = 100
	pt.Y.Label.Text = "Y Axis"

	pt.Resize(image.Point{640, 480})
	pt.Draw()
	imagex.Assert(t, pt.Pixels, "plot.png")
}
