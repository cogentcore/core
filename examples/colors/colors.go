// Copyright (c) 2018, Randall C. O'Reilly. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"github.com/rcoreilly/goki/gi"
	"github.com/rcoreilly/goki/gi/oswin"
	"github.com/rcoreilly/goki/gi/oswin/driver"
	"github.com/rcoreilly/goki/gi/units"
	"github.com/rcoreilly/goki/ki"
)

func main() {
	driver.Main(func(app oswin.App) {
		mainrun()
	})
}

func mainrun() {
	width := 1024
	height := 768

	// turn this on to see a trace of the rendering
	// gi.Update2DTrace = true
	// gi.Render2DTrace = true
	// gi.Layout2DTrace = true

	rec := ki.Node{}          // receiver for events
	rec.InitName(&rec, "rec") // this is essential for root objects not owned by other Ki tree nodes

	win := gi.NewWindow2D("GoGi Colors Window", width, height, true)
	vp := win.WinViewport2D()
	updt := vp.UpdateStart()

	vp.SetProp("background-color", "#FFF")
	vp.Fill = true

	vlay := vp.AddNewChild(gi.KiT_Frame, "vlay").(*gi.Frame)
	vlay.Lay = gi.LayoutCol

	row1 := vlay.AddNewChild(gi.KiT_Layout, "row1").(*gi.Layout)
	row1.Lay = gi.LayoutRow
	row1.SetProp("align-vert", gi.AlignMiddle)
	row1.SetProp("align-horiz", "center")
	row1.SetProp("margin", 2.0) // raw numbers = px = 96 dpi pixels
	row1.SetStretchMaxWidth()

	spc := vlay.AddNewChild(gi.KiT_Space, "spc1").(*gi.Space)
	spc.SetFixedHeight(units.NewValue(2.0, units.Em))

	row1.AddNewChild(gi.KiT_Stretch, "str1")
	lab1 := row1.AddNewChild(gi.KiT_Label, "lab1").(*gi.Label)
	lab1.Text = "These are tests of the various GoGi Color functions"
	lab1.SetProp("max-width", -1)
	lab1.SetProp("text-align", "center")
	row1.AddNewChild(gi.KiT_Stretch, "str2")

	grid := vlay.AddNewChild(gi.KiT_Layout, "grid").(*gi.Layout)
	grid.Lay = gi.LayoutGrid
	grid.SetProp("columns", 11)
	grid.SetProp("align-vert", "center")
	grid.SetProp("align-horiz", "center")
	grid.SetProp("margin", 2.0)
	grid.SetStretchMaxWidth()
	grid.SetStretchMaxHeight()

	// first test the HSL color scheme
	var hues = [...]float32{0, 60, 120, 180, 240, 300}
	sat := float32(1.0)

	for _, hu := range hues {
		for lt := float32(0.0); lt <= 1.01; lt += 0.1 {
			fr := grid.AddNewChild(gi.KiT_Frame, "fr").(*gi.Frame)
			fr.SetProp("background-color", gi.HSLAf32{hu, sat, lt, 1.0})
			fr.SetProp("max-width", -1)
			fr.SetProp("max-height", -1)
		}
	}
	// try again with alpha
	for _, hu := range hues {
		for lt := float32(0.0); lt <= 1.01; lt += 0.1 {
			fr := grid.AddNewChild(gi.KiT_Frame, "fr").(*gi.Frame)
			fr.SetProp("background-color", gi.HSLAf32{hu, sat, lt, 0.5})
			fr.SetProp("max-width", -1)
			fr.SetProp("max-height", -1)
		}
	}
	// then sats
	lt := float32(0.5)
	for _, hu := range hues {
		for sat := float32(0.0); sat <= 1.01; sat += 0.1 {
			fr := grid.AddNewChild(gi.KiT_Frame, "fr").(*gi.Frame)
			fr.SetProp("background-color", gi.HSLAf32{hu, sat, lt, 1.0})
			fr.SetProp("max-width", -1)
			fr.SetProp("max-height", -1)
		}
	}
	// then doing it with colors -- tests the "there and back again" round trip..
	for _, hu := range hues {
		clr := gi.Color{}
		clr.SetHSLA(hu, 1.0, 0.5, 1.0)
		for lt := float32(0.0); lt <= 2.01; lt += 0.2 {
			fr := grid.AddNewChild(gi.KiT_Frame, "fr").(*gi.Frame)
			fr.SetProp("background-color", clr.Lighter(lt))
			fr.SetProp("max-width", -1)
			fr.SetProp("max-height", -1)
		}
	}

	vp.UpdateEndNoSig(updt)

	win.StartEventLoop()
}
