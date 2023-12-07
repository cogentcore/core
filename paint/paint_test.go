// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package paint

import (
	"os"
	"testing"

	"goki.dev/colors"
	"goki.dev/girl/styles"
	"goki.dev/girl/units"
	"goki.dev/grows/images"
	"goki.dev/mat32/v2"
)

func TestMain(m *testing.M) {
	FontLibrary.InitFontPaths(FontPaths...)
	os.Exit(m.Run())
}

// RunTest makes a rendering state, paint, and image with the given size, calls the given
// function, and then asserts the image using [images.Assert] with the given name.
func RunTest(t *testing.T, nm string, width int, height int, f func(pc *Context)) {
	pc := NewContext(width, height)
	pc.PushBounds(pc.Image.Rect)
	pc.Lock()

	f(pc)

	pc.Unlock()

	images.Assert(t, pc.Image, nm)
}

func TestRender(t *testing.T) {
	RunTest(t, "render", 300, 300, func(pc *Context) {
		bs := styles.Border{}
		bs.Color.Set(colors.Red, colors.Blue, colors.Green, colors.Orange)
		bs.Width.Set(units.Dot(20), units.Dot(30), units.Dot(40), units.Dot(50))
		bs.ToDots(&pc.UnContext)

		// first, draw a frame around the entire image
		// pc.StrokeStyle.SetColor(blk)
		pc.FillStyle.SetColor(colors.White)
		// pc.StrokeStyle.Width.SetDot(1) // use dots directly to render in literal pixels
		pc.DrawBorder(0, 0, 300, 300, bs)
		pc.FillStrokeClear() // actually render path that has been setup

		// next draw a rounded rectangle
		bs.Color.Set(colors.Purple, colors.Green, colors.Red, colors.Blue)
		// bs.Width.Set(units.NewDot(10))
		bs.Radius.Set(units.Dot(0), units.Dot(30), units.Dot(10))
		pc.FillStyle.SetColor(colors.Lightblue)
		pc.StrokeStyle.Width.Dot(10)
		bs.ToDots(&pc.UnContext)
		pc.DrawBorder(60, 60, 150, 100, bs)
		pc.FillStrokeClear()

		tsty := &styles.Text{}
		tsty.Defaults()
		fsty := &styles.FontRender{}
		fsty.Defaults()

		tsty.Align = styles.Center

		txt := &Text{}
		txt.SetHTML("This is <a>HTML</a> <b>formatted</b> <i>text</i>", fsty, tsty, &pc.UnContext, nil)

		// the last, size arg provides constraints for layout to fit within -- uses width mainly
		tsz := txt.LayoutStdLR(tsty, fsty, &pc.UnContext, mat32.Vec2{100, 40})
		if tsz.X != 100 || tsz.Y != 40 {
			t.Errorf("unexpected text size: %v", tsz)
		}

		txt.Render(pc, mat32.Vec2{85, 80})
	})
}

func TestPaintPath(t *testing.T) {
	test := func(nm string, f func(pc *Context)) {
		RunTest(t, nm, 300, 300, func(pc *Context) {
			pc.FillBox(mat32.Vec2{}, mat32.Vec2{300, 300}, colors.SolidFull(colors.White))
			f(pc)
			pc.FillStrokeClear()
		})
	}
	test("line_to", func(pc *Context) {
		pc.MoveTo(100, 200)
		pc.LineTo(200, 100)
	})
	test("quadratic_to", func(pc *Context) {
		pc.MoveTo(100, 200)
		pc.QuadraticTo(120, 140, 200, 100)
	})
	test("cubic_to", func(pc *Context) {
		pc.MoveTo(100, 200)
		pc.CubicTo(130, 110, 160, 180, 200, 100)
	})
	test("close_path", func(pc *Context) {
		pc.MoveTo(100, 200)
		pc.MoveTo(200, 100)
		pc.ClosePath()
	})
	test("clear_path", func(pc *Context) {
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
	test("fill_box_color", func(pc *Context) {
		pc.FillBoxColor(mat32.Vec2{10, 100}, mat32.Vec2{200, 100}, colors.Green)
	})
	test("fill_box_solid", func(pc *Context) {
		pc.FillBox(mat32.Vec2{10, 100}, mat32.Vec2{200, 100}, colors.SolidFull(colors.Blue))
	})
	test("fill_box_linear_gradient", func(pc *Context) {
		g := colors.LinearGradient().AddStop(colors.Red, 0, 1).AddStop(colors.Yellow, 0.3, 0.5).AddStop(colors.Green, 1, 1)
		pc.FillBox(mat32.Vec2{10, 100}, mat32.Vec2{200, 100}, colors.GradientFull(g))
	})
	test("fill_box_radial_gradient", func(pc *Context) {
		g := colors.RadialGradient().AddStop(colors.Green, 0, 0.5).AddStop(colors.Blue, 0.6, 1).AddStop(colors.Purple, 1, 0.3)
		pc.FillBox(mat32.Vec2{10, 100}, mat32.Vec2{200, 100}, colors.GradientFull(g))
	})
	test("blur_box", func(pc *Context) {
		pc.FillBoxColor(mat32.Vec2{10, 100}, mat32.Vec2{200, 100}, colors.Green)
		pc.BlurBox(mat32.Vec2{0, 50}, mat32.Vec2{300, 200}, 10)
	})

	test("fill", func(pc *Context) {
		pc.FillStyle.SetColor(colors.Purple)
		pc.StrokeStyle.SetColor(colors.Orange)
		pc.DrawRectangle(50, 25, 150, 200)
		pc.Fill()
	})
	test("stroke", func(pc *Context) {
		pc.FillStyle.SetColor(colors.Purple)
		pc.StrokeStyle.SetColor(colors.Orange)
		pc.DrawRectangle(50, 25, 150, 200)
		pc.Stroke()
	})

	// testing whether nil values turn off stroking/filling with FillStrokeClear
	test("fill_stroke_clear_fill", func(pc *Context) {
		pc.FillStyle.SetColor(colors.Purple)
		pc.StrokeStyle.SetColor(nil)
		pc.DrawRectangle(50, 25, 150, 200)
		pc.FillStrokeClear()
	})
	test("fill_stroke_clear_stroke", func(pc *Context) {
		pc.FillStyle.SetColor(nil)
		pc.StrokeStyle.SetColor(colors.Orange)
		pc.DrawRectangle(50, 25, 150, 200)
		pc.FillStrokeClear()
	})
}
