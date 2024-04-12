// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package paint

import (
	"image"
	"os"
	"slices"
	"testing"

	"cogentcore.org/core/colors"
	"cogentcore.org/core/colors/gradient"
	"cogentcore.org/core/iox/imagex"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/units"
	"github.com/stretchr/testify/assert"
)

func TestMain(m *testing.M) {
	FontLibrary.InitFontPaths(FontPaths...)
	os.Exit(m.Run())
}

// RunTest makes a rendering state, paint, and image with the given size, calls the given
// function, and then asserts the image using [imagex.Assert] with the given name.
func RunTest(t *testing.T, nm string, width int, height int, f func(pc *Context)) {
	pc := NewContext(width, height)
	pc.PushBounds(pc.Image.Rect)
	f(pc)
	imagex.Assert(t, pc.Image, nm)
}

func TestRender(t *testing.T) {
	RunTest(t, "render", 300, 300, func(pc *Context) {
		testimg, _, err := imagex.Open("test.png")
		assert.NoError(t, err)
		imgs := []image.Image{
			colors.C(colors.Blue),
			gradient.NewLinear().AddStop(colors.Orange, 0).AddStop(colors.Red, 1).SetTransform(math32.Rotate2D(90)),
			gradient.NewRadial().AddStop(colors.Green, 0).AddStop(colors.Blue, 0.6, 0.4).AddStop(colors.Purple, 0.9, 0.8),
			testimg,
		}

		bs := styles.Border{}
		bs.Color.Set(imgs...)
		bs.Width.Set(units.Dot(20), units.Dot(30), units.Dot(40), units.Dot(50))
		bs.ToDots(&pc.UnitContext)

		// first, draw a frame around the entire image
		// pc.StrokeStyle.Color = colors.C(blk)
		pc.FillStyle.Color = colors.C(colors.White)
		// pc.StrokeStyle.Width.SetDot(1) // use dots directly to render in literal pixels
		pc.DrawBorder(0, 0, 300, 300, bs)
		pc.FillStrokeClear() // actually render path that has been setup

		slices.Reverse(imgs)
		// next draw a rounded rectangle
		bs.Color.Set(imgs...)
		// bs.Width.Set(units.NewDot(10))
		bs.Radius.Set(units.Dot(0), units.Dot(30), units.Dot(10))
		pc.FillStyle.Color = colors.C(colors.Lightblue)
		pc.StrokeStyle.Width.Dot(10)
		bs.ToDots(&pc.UnitContext)
		pc.DrawBorder(60, 60, 150, 100, bs)
		pc.FillStrokeClear()

		tsty := &styles.Text{}
		tsty.Defaults()
		fsty := &styles.FontRender{}
		fsty.Defaults()
		fsty.Color = imgs[1]
		fsty.Background = imgs[2]

		tsty.Align = styles.Center

		txt := &Text{}
		txt.SetHTML("This is <a>HTML</a> <b>formatted</b> <i>text</i>", fsty, tsty, &pc.UnitContext, nil)

		tsz := txt.Layout(tsty, fsty, &pc.UnitContext, math32.V2(100, 40))
		if tsz.X != 100 || tsz.Y != 40 {
			t.Errorf("unexpected text size: %v", tsz)
		}

		txt.Render(pc, math32.V2(85, 80))
	})
}

func TestPaintPath(t *testing.T) {
	test := func(nm string, f func(pc *Context)) {
		RunTest(t, nm, 300, 300, func(pc *Context) {
			pc.FillBox(math32.Vector2{}, math32.V2(300, 300), colors.C(colors.White))
			f(pc)
			pc.StrokeStyle.Color = colors.C(colors.Blue)
			pc.FillStyle.Color = colors.C(colors.Yellow)
			pc.StrokeStyle.Width.Dot(3)
			pc.FillStrokeClear()
		})
	}
	test("line-to", func(pc *Context) {
		pc.MoveTo(100, 200)
		pc.LineTo(200, 100)
	})
	test("quadratic-to", func(pc *Context) {
		pc.MoveTo(100, 200)
		pc.QuadraticTo(120, 140, 200, 100)
	})
	test("cubic-to", func(pc *Context) {
		pc.MoveTo(100, 200)
		pc.CubicTo(130, 110, 160, 180, 200, 100)
	})
	test("close-path", func(pc *Context) {
		pc.MoveTo(100, 200)
		pc.LineTo(200, 100)
		pc.LineTo(250, 150)
		pc.ClosePath()
	})
	test("clear-path", func(pc *Context) {
		pc.MoveTo(100, 200)
		pc.MoveTo(200, 100)
		pc.ClearPath()
	})
}

func TestPaintFill(t *testing.T) {
	test := func(nm string, f func(pc *Context)) {
		RunTest(t, nm, 300, 300, func(pc *Context) {
			f(pc)
		})
	}
	test("fill-box-color", func(pc *Context) {
		pc.FillBox(math32.V2(10, 100), math32.V2(200, 100), colors.C(colors.Green))
	})
	test("fill-box-solid", func(pc *Context) {
		pc.FillBox(math32.V2(10, 100), math32.V2(200, 100), colors.C(colors.Blue))
	})
	test("fill-box-linear-gradient-black-white", func(pc *Context) {
		g := gradient.NewLinear().AddStop(colors.Black, 0).AddStop(colors.White, 1)
		pc.FillBox(math32.V2(10, 100), math32.V2(200, 100), g)
	})
	test("fill-box-linear-gradient-red-green", func(pc *Context) {
		g := gradient.NewLinear().AddStop(colors.Red, 0).AddStop(colors.Limegreen, 1)
		pc.FillBox(math32.V2(10, 100), math32.V2(200, 100), g)
	})
	test("fill-box-linear-gradient-red-yellow-green", func(pc *Context) {
		g := gradient.NewLinear().AddStop(colors.Red, 0).AddStop(colors.Yellow, 0.3).AddStop(colors.Green, 1)
		pc.FillBox(math32.V2(10, 100), math32.V2(200, 100), g)
	})
	test("fill-box-radial-gradient", func(pc *Context) {
		g := gradient.NewRadial().AddStop(colors.ApplyOpacity(colors.Green, 0.5), 0).AddStop(colors.Blue, 0.6).
			AddStop(colors.ApplyOpacity(colors.Purple, 0.3), 1)
		pc.FillBox(math32.V2(10, 100), math32.V2(200, 100), g)
	})
	test("blur-box", func(pc *Context) {
		pc.FillBox(math32.V2(10, 100), math32.V2(200, 100), colors.C(colors.Green))
		pc.BlurBox(math32.V2(0, 50), math32.V2(300, 200), 10)
	})

	test("fill", func(pc *Context) {
		pc.FillStyle.Color = colors.C(colors.Purple)
		pc.StrokeStyle.Color = colors.C(colors.Orange)
		pc.DrawRectangle(50, 25, 150, 200)
		pc.Fill()
	})
	test("stroke", func(pc *Context) {
		pc.FillStyle.Color = colors.C(colors.Purple)
		pc.StrokeStyle.Color = colors.C(colors.Orange)
		pc.DrawRectangle(50, 25, 150, 200)
		pc.Stroke()
	})

	// testing whether nil values turn off stroking/filling with FillStrokeClear
	test("fill-stroke-clear-fill", func(pc *Context) {
		pc.FillStyle.Color = colors.C(colors.Purple)
		pc.StrokeStyle.Color = nil
		pc.DrawRectangle(50, 25, 150, 200)
		pc.FillStrokeClear()
	})
	test("fill-stroke-clear-stroke", func(pc *Context) {
		pc.FillStyle.Color = nil
		pc.StrokeStyle.Color = colors.C(colors.Orange)
		pc.DrawRectangle(50, 25, 150, 200)
		pc.FillStrokeClear()
	})
}
