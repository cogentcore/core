// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package girl

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"log"
	"os"
	"strings"
	"testing"

	"goki.dev/colors"
	"goki.dev/gi/v2/gist"
	"goki.dev/gi/v2/units"
	"goki.dev/mat32/v2"
)

// TestPrefs are needed for setting gist.ThePrefs, for any text-based
// rendering, as it relies on these prefs
type TestPrefs struct {

	// font family name
	FontFamily string `desc:"font family name"`

	// default font / pen color
	Font color.RGBA `desc:"default font / pen color"`

	// default background color
	Background color.RGBA `desc:"default background color"`

	// color for shadows -- should generally be a darker shade of the background color
	Shadow color.RGBA `desc:"color for shadows -- should generally be a darker shade of the background color"`

	// default border color, for button, frame borders, etc
	Border color.RGBA `desc:"default border color, for button, frame borders, etc"`

	// default main color for controls: buttons, etc
	Control color.RGBA `desc:"default main color for controls: buttons, etc"`

	// color for icons or other solidly-colored, small elements
	Icon color.RGBA `desc:"color for icons or other solidly-colored, small elements"`

	// color for selected elements
	Select color.RGBA `desc:"color for selected elements"`

	// color for highlight background
	Highlight color.RGBA `desc:"color for highlight background"`

	// color for links in text etc
	Link color.RGBA `desc:"color for links in text etc"`
}

func (pf *TestPrefs) Defaults() {
	pf.FontFamily = "Go"
	pf.Font = colors.Black
	pf.Border = colors.MustFromHex("#666")
	pf.Background = colors.White
	pf.Shadow = colors.MustFromString("darken-10", &pf.Background)
	pf.Control = colors.MustFromHex("#F8F8F8")
	pf.Icon = colors.MustFromString("highlight-30", pf.Control)
	pf.Select = colors.MustFromHex("#CFC")
	pf.Highlight = colors.MustFromHex("#FFA")
	pf.Link = colors.MustFromHex("#00F")
}

// PrefColor returns preference color of given name (case insensitive)
// std names are: font, background, shadow, border, control, icon, select, highlight, link
func (pf *TestPrefs) PrefColor(clrName string) *color.RGBA {
	lc := strings.Replace(strings.ToLower(clrName), "-", "", -1)
	switch lc {
	case "font":
		return &pf.Font
	case "background":
		return &pf.Background
	case "shadow":
		return &pf.Shadow
	case "border":
		return &pf.Border
	case "control":
		return &pf.Control
	case "icon":
		return &pf.Icon
	case "select":
		return &pf.Select
	case "highlight":
		return &pf.Highlight
	case "link":
		return &pf.Link
	}
	log.Printf("Preference color %v (simplified to: %v) not found\n", clrName, lc)
	return nil
}

// PrefFontFamily returns the default FontFamily
func (pf *TestPrefs) PrefFontFamily() string {
	return pf.FontFamily
}

func TestRender(t *testing.T) {
	prefs := &TestPrefs{}
	prefs.Defaults()
	gist.ThePrefs = prefs

	// in GoGi, oswin.TheApp gives you default font paths per-platform.
	// here, we usually auto-test on linux so leave that one in place
	// but if interactively testing e.g., on mac then use that..

	// mac:
	// FontLibrary.InitFontPaths("/System/Library/Fonts", "/Library/Fonts")
	// linux:
	FontLibrary.InitFontPaths("/usr/share/fonts/truetype")
	// windows
	// FontLibrary.InitFontPaths("C:\\Windows\\Fonts")

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

	bs := gist.Border{}
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
	tsty := &gist.Text{}
	tsty.Defaults()
	fsty := &gist.FontRender{}
	fsty.Defaults()

	// experiment!
	tsty.Align = gist.AlignCenter

	txt := &Text{}
	txt.SetHTML("This is <b>HTML</b> formatted <i>text</i>", fsty, tsty, &pc.UnContext, nil)

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
