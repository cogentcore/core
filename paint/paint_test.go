// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package paint

import (
	"fmt"
	"image"
	"image/png"
	"os"
	"testing"

	"goki.dev/colors"
	"goki.dev/girl/styles"
	"goki.dev/girl/units"
	"goki.dev/mat32/v2"
)

func TestRender(t *testing.T) {
	FontLibrary.InitFontPaths(FontPaths...)

	imgsz := image.Point{320, 240}
	szrec := image.Rectangle{Max: imgsz}
	img := image.NewRGBA(szrec)

	rs := &State{}
	pc := &Paint{}

	pc.Defaults()               // zeros are not good defaults for paint
	pc.SetUnitContextExt(imgsz) // initialize units

	rs.Init(imgsz.X, imgsz.Y, img)
	rs.PushBounds(szrec)
	rs.Lock()

	bs := styles.Border{}
	bs.Color.Set(colors.Red, colors.Blue, colors.Green, colors.Orange)
	bs.Width.Set(units.Dot(20), units.Dot(30), units.Dot(40), units.Dot(50))
	bs.ToDots(&pc.UnContext)

	// first, draw a frame around the entire image
	// pc.StrokeStyle.SetColor(blk)
	pc.FillStyle.SetColor(colors.White)
	// pc.StrokeStyle.Width.SetDot(1) // use dots directly to render in literal pixels
	pc.DrawBorder(rs, 0, 0, float32(imgsz.X), float32(imgsz.Y), bs)
	pc.FillStrokeClear(rs) // actually render path that has been setup

	// next draw a rounded rectangle
	bs.Color.Set(colors.Purple, colors.Green, colors.Red, colors.Blue)
	// bs.Width.Set(units.NewDot(10))
	bs.Radius.Set(units.Dot(0), units.Dot(30), units.Dot(10))
	pc.FillStyle.SetColor(colors.Lightblue)
	pc.StrokeStyle.Width.SetDot(10)
	bs.ToDots(&pc.UnContext)
	pc.DrawBorder(rs, 60, 60, 150, 100, bs)
	pc.FillStrokeClear(rs)

	// // use units-based styling instead of dots:
	// bs.Color.Set(blu, grn, blk)
	// bs.Width.Set(units.NewPct(5), units.NewPct(7), units.NewPct(3), units.NewPct(15))
	// bs.Width.ToDots(&pc.UnContext)
	// bs.ToDots(&pc.UnContext)
	// // pc.StrokeStyle.SetColor(blu)
	// // pc.StrokeStyle.Width.SetPct(2) // percent of total image (width)
	// // pc.ToDots() // convert pct -> dots based on units context
	// // fmt.Printf("pct dots: %g\n", pc.StrokeStyle.Width.Dots) // 6.4
	// pc.DrawChangingRoundedRectangle(rs, 100, 100, 150, 100, bs)
	// pc.FillStrokeClear(rs)

	// Text rendering
	tsty := &styles.Text{}
	tsty.Defaults()
	fsty := &styles.FontRender{}
	fsty.Defaults()

	// experiment!
	tsty.Align = styles.AlignCenter

	txt := &Text{}
	txt.SetHTML("This is <a>HTML</a> <b>formatted</b> <i>text</i>", fsty, tsty, &pc.UnContext, nil)

	// the last, size arg provides constraints for layout to fit within -- uses width mainly
	tsz := txt.LayoutStdLR(tsty, fsty, &pc.UnContext, mat32.Vec2{100, 40})
	fmt.Printf("text size: %v\n", tsz)

	txt.Render(rs, mat32.Vec2{85, 80})

	rs.Unlock()

	file, err := os.Create("test.png")
	if err != nil {
		t.Error(err)
	}
	defer file.Close()
	png.Encode(file, img)
}
