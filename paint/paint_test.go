// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package paint_test

import (
	"os"
	"testing"

	"cogentcore.org/core/base/iox/imagex"
	"cogentcore.org/core/colors"
	"cogentcore.org/core/math32"
	. "cogentcore.org/core/paint"
	"cogentcore.org/core/paint/renderers/rasterx"
)

func TestMain(m *testing.M) {
	FontLibrary.InitFontPaths(FontPaths...)
	NewDefaultImageRenderer = rasterx.New
	os.Exit(m.Run())
}

// RunTest makes a rendering state, paint, and image with the given size, calls the given
// function, and then asserts the image using [imagex.Assert] with the given name.
func RunTest(t *testing.T, nm string, width int, height int, f func(pc *Painter)) {
	pc := NewPainter(width, height)
	// pc.PushBounds(pc.Image.Rect)
	f(pc)
	imagex.Assert(t, pc.Image, nm)
}

/*
func TestRender(t *testing.T) {
	RunTest(t, "render", 300, 300, func(pc *Painter) {
		testimg, _, err := imagex.Open("test.png")
		assert.NoError(t, err)
		linear := gradient.NewLinear()
		linear.AddStop(colors.Orange, 0).AddStop(colors.Red, 1).SetTransform(math32.Rotate2D(90))
		radial := gradient.NewRadial()
		radial.AddStop(colors.Green, 0).AddStop(colors.Blue, 0.6, 0.4).AddStop(colors.Purple, 0.9, 0.8)

		imgs := []image.Image{colors.Uniform(colors.Blue), linear, radial, testimg}

		bs := styles.Border{}
		bs.Color.Set(imgs...)
		bs.Width.Set(units.Dot(20), units.Dot(30), units.Dot(40), units.Dot(50))
		bs.ToDots(&pc.UnitPaint)

		// first, draw a frame around the entire image
		// pc.Stroke.Color = colors.C(blk)
		pc.Fill.Color = colors.Uniform(colors.White)
		// pc.Stroke.Width.SetDot(1) // use dots directly to render in literal pixels
		pc.DrawBorder(0, 0, 300, 300, bs)
		pc.PathDone() // actually render path that has been setup

		slices.Reverse(imgs)
		// next draw a rounded rectangle
		bs.Color.Set(imgs...)
		// bs.Width.Set(units.NewDot(10))
		bs.Radius.Set(units.Dot(0), units.Dot(30), units.Dot(10))
		pc.Fill.Color = colors.Uniform(colors.Lightblue)
		pc.Stroke.Width.Dot(10)
		bs.ToDots(&pc.UnitPaint)
		pc.DrawBorder(60, 60, 150, 100, bs)
		pc.PathDone()

		tsty := &styles.Text{}
		tsty.Defaults()
		fsty := &styles.FontRender{}
		fsty.Defaults()
		fsty.Color = imgs[1]
		fsty.Background = imgs[2]

		tsty.Align = styles.Center

		txt := &Text{}
		txt.SetHTML("This is <a>HTML</a> <b>formatted</b> <i>text</i>", fsty, tsty, &pc.UnitPaint, nil)

		tsz := txt.Layout(tsty, fsty, &pc.UnitPaint, math32.Vec2(100, 60))
		if tsz.X != 100 || tsz.Y != 60 {
			t.Errorf("unexpected text size: %v", tsz)
		}

		txt.Render(pc, math32.Vec2(85, 80))
	})
}
*/

func TestPaintPath(t *testing.T) {
	test := func(nm string, f func(pc *Painter)) {
		RunTest(t, nm, 300, 300, func(pc *Painter) {
			pc.FillBox(math32.Vector2{}, math32.Vec2(300, 300), colors.Uniform(colors.White))
			f(pc)
			pc.Stroke.Color = colors.Uniform(colors.Blue)
			pc.Fill.Color = colors.Uniform(colors.Yellow)
			pc.Stroke.Width.Dot(3)
			pc.PathDone()
			pc.RenderDone()
		})
	}
	test("line-to", func(pc *Painter) {
		pc.MoveTo(100, 200)
		pc.LineTo(200, 100)
	})
	// test("quadratic-to", func(pc *Painter) {
	// 	pc.MoveTo(100, 200)
	// 	pc.QuadTo(120, 140, 200, 100)
	// })
	// test("cubic-to", func(pc *Painter) {
	// 	pc.MoveTo(100, 200)
	// 	pc.CubeTo(130, 110, 160, 180, 200, 100)
	// })
	// test("close-path", func(pc *Painter) {
	// 	pc.MoveTo(100, 200)
	// 	pc.LineTo(200, 100)
	// 	pc.LineTo(250, 150)
	// 	pc.Close()
	// })
	// test("clear-path", func(pc *Painter) {
	// 	pc.MoveTo(100, 200)
	// 	pc.MoveTo(200, 100)
	// 	pc.Clear()
	// })
}

/*
func TestPaintFill(t *testing.T) {
	test := func(nm string, f func(pc *Painter)) {
		RunTest(t, nm, 300, 300, func(pc *Painter) {
			f(pc)
		})
	}
	test("fill-box-color", func(pc *Painter) {
		pc.FillBox(math32.Vec2(10, 100), math32.Vec2(200, 100), colors.Uniform(colors.Green))
	})
	test("fill-box-solid", func(pc *Painter) {
		pc.FillBox(math32.Vec2(10, 100), math32.Vec2(200, 100), colors.Uniform(colors.Blue))
	})
	test("fill-box-linear-gradient-black-white", func(pc *Painter) {
		g := gradient.NewLinear().AddStop(colors.Black, 0).AddStop(colors.White, 1)
		pc.FillBox(math32.Vec2(10, 100), math32.Vec2(200, 100), g)
	})
	test("fill-box-linear-gradient-red-green", func(pc *Painter) {
		g := gradient.NewLinear().AddStop(colors.Red, 0).AddStop(colors.Limegreen, 1)
		pc.FillBox(math32.Vec2(10, 100), math32.Vec2(200, 100), g)
	})
	test("fill-box-linear-gradient-red-yellow-green", func(pc *Painter) {
		g := gradient.NewLinear().AddStop(colors.Red, 0).AddStop(colors.Yellow, 0.3).AddStop(colors.Green, 1)
		pc.FillBox(math32.Vec2(10, 100), math32.Vec2(200, 100), g)
	})
	test("fill-box-radial-gradient", func(pc *Painter) {
		g := gradient.NewRadial().AddStop(colors.ApplyOpacity(colors.Green, 0.5), 0).AddStop(colors.Blue, 0.6).
			AddStop(colors.ApplyOpacity(colors.Purple, 0.3), 1)
		pc.FillBox(math32.Vec2(10, 100), math32.Vec2(200, 100), g)
	})
	test("blur-box", func(pc *Painter) {
		pc.FillBox(math32.Vec2(10, 100), math32.Vec2(200, 100), colors.Uniform(colors.Green))
		pc.BlurBox(math32.Vec2(0, 50), math32.Vec2(300, 200), 10)
	})

	test("fill", func(pc *Painter) {
		pc.Fill.Color = colors.Uniform(colors.Purple)
		pc.Stroke.Color = colors.Uniform(colors.Orange)
		pc.DrawRectangle(50, 25, 150, 200)
		pc.PathDone()
	})
	test("stroke", func(pc *Painter) {
		pc.Fill.Color = colors.Uniform(colors.Purple)
		pc.Stroke.Color = colors.Uniform(colors.Orange)
		pc.DrawRectangle(50, 25, 150, 200)
		pc.PathDone()
	})

	// testing whether nil values turn off stroking/filling with FillStrokeClear
	test("fill-stroke-clear-fill", func(pc *Painter) {
		pc.Fill.Color = colors.Uniform(colors.Purple)
		pc.Stroke.Color = nil
		pc.DrawRectangle(50, 25, 150, 200)
		pc.PathDone()
	})
	test("fill-stroke-clear-stroke", func(pc *Painter) {
		pc.Fill.Color = nil
		pc.Stroke.Color = colors.Uniform(colors.Orange)
		pc.DrawRectangle(50, 25, 150, 200)
		pc.PathDone()
	})
}
*/
