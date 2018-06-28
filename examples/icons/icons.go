// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"image/color"

	"github.com/goki/gi"
	"github.com/goki/gi/oswin"
	"github.com/goki/gi/oswin/driver"
	"github.com/goki/gi/units"
	"github.com/goki/ki"
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

	win := gi.NewWindow2D("GoGi Icons Window", width, height, true)
	vp := win.WinViewport2D()
	updt := vp.UpdateStart()

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
	lab1.Text = "These are all the GoGi Icons, in a small and large size"
	lab1.SetProp("max-width", -1)
	lab1.SetProp("text-align", "center")
	row1.AddNewChild(gi.KiT_Stretch, "str2")

	grid := vlay.AddNewChild(gi.KiT_Layout, "grid").(*gi.Layout)
	grid.Lay = gi.LayoutGrid
	grid.SetProp("columns", 4)
	grid.SetProp("align-vert", "center")
	grid.SetProp("align-horiz", "center")
	grid.SetProp("margin", 2.0)
	grid.SetStretchMaxWidth()
	grid.SetStretchMaxHeight()

	il := gi.IconListSorted(*gi.CurIconSet)

	for _, icnm := range il {
		if icnm == "none" {
			continue
		}
		vb := grid.AddNewChild(gi.KiT_Layout, "vb").(*gi.Layout)
		vb.Lay = gi.LayoutCol

		smico := vb.AddNewChild(gi.KiT_Icon, icnm).(*gi.Icon)
		smico.InitFromName(icnm)
		smico.SetMinPrefWidth(units.NewValue(20, units.Px))
		smico.SetMinPrefHeight(units.NewValue(20, units.Px))
		smico.SetProp("background-color", color.Transparent)
		smico.SetProp("fill", "#88F")
		smico.SetProp("stroke", "black")
		smico.SetProp("align-horiz", gi.AlignCenter)

		ico := vb.AddNewChild(gi.KiT_Icon, icnm).(*gi.Icon)
		ico.InitFromName(icnm)
		ico.SetMinPrefWidth(units.NewValue(100, units.Px))
		ico.SetMinPrefHeight(units.NewValue(100, units.Px))
		ico.SetProp("background-color", color.Transparent)
		ico.SetProp("fill", "#88F")
		ico.SetProp("stroke", "black")
		ico.SetProp("align-horiz", gi.AlignCenter)
		nmlbl := vb.AddNewChild(gi.KiT_Label, "lab1").(*gi.Label)
		nmlbl.Text = icnm
	}

	vp.UpdateEndNoSig(updt)

	win.StartEventLoop()
}
