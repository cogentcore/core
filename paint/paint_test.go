// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package paint

import (
	"image"
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
func RunTest(t *testing.T, nm string, sz image.Point, f func(rs *State, pc *Paint)) {
	szrec := image.Rectangle{Max: sz}
	img := image.NewRGBA(szrec)

	rs := &State{}
	pc := &Paint{}

	pc.Defaults()            // zeros are not good defaults for paint
	pc.SetUnitContextExt(sz) // initialize units

	rs.Init(sz.X, sz.Y, img)
	rs.PushBounds(szrec)
	rs.Lock()

	f(rs, pc)

	rs.Unlock()

	images.Assert(t, img, nm)
}

func TestRender(t *testing.T) {
	RunTest(t, "render", image.Pt(320, 240), func(rs *State, pc *Paint) {
		bs := styles.Border{}
		bs.Color.Set(colors.Red, colors.Blue, colors.Green, colors.Orange)
		bs.Width.Set(units.Dot(20), units.Dot(30), units.Dot(40), units.Dot(50))
		bs.ToDots(&pc.UnContext)

		// first, draw a frame around the entire image
		// pc.StrokeStyle.SetColor(blk)
		pc.FillStyle.SetColor(colors.White)
		// pc.StrokeStyle.Width.SetDot(1) // use dots directly to render in literal pixels
		pc.DrawBorder(rs, 0, 0, 320, 240, bs)
		pc.FillStrokeClear(rs) // actually render path that has been setup

		// next draw a rounded rectangle
		bs.Color.Set(colors.Purple, colors.Green, colors.Red, colors.Blue)
		// bs.Width.Set(units.NewDot(10))
		bs.Radius.Set(units.Dot(0), units.Dot(30), units.Dot(10))
		pc.FillStyle.SetColor(colors.Lightblue)
		pc.StrokeStyle.Width.Dot(10)
		bs.ToDots(&pc.UnContext)
		pc.DrawBorder(rs, 60, 60, 150, 100, bs)
		pc.FillStrokeClear(rs)

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

		txt.Render(rs, mat32.Vec2{85, 80})
	})
}
