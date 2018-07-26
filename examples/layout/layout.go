// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"fmt"

	"github.com/goki/gi"
	"github.com/goki/gi/oswin"
	"github.com/goki/gi/oswin/driver"
	_ "github.com/goki/gi/svg"
	"github.com/goki/gi/units"
)

func main() {
	driver.Main(func(app oswin.App) {
		mainrun()
	})
}

func mainrun() {
	width := 1024
	height := 768

	gi.Layout2DTrace = true

	frsz := [5]gi.Vec2D{
		{20, 100},
		{80, 20},
		{60, 80},
		{40, 120},
		{150, 100},
	}
	// frsz := [4]gi.Vec2D{
	// 	{100, 100},
	// 	{100, 100},
	// 	{100, 100},
	// 	{100, 100},
	// }

	win := gi.NewWindow2D("gogi-layout-test", "GoGi Layout Test", width, height, true)

	vp := win.WinViewport2D()
	updt := vp.UpdateStart()
	vp.Fill = true

	vlay := vp.AddNewChild(gi.KiT_Frame, "vlay").(*gi.Frame)
	vlay.Lay = gi.LayoutCol

	row1 := vlay.AddNewChild(gi.KiT_Layout, "row1").(*gi.Layout)
	row1.Lay = gi.LayoutRow

	row1.SetProp("align-vert", "top")
	// row1.SetProp("align-horiz", "justify")
	row1.SetProp("align-horiz", "left")
	row1.SetProp("margin", 4.0)
	row1.SetProp("max-width", -1) // always stretch width

	for i, sz := range frsz {
		nm := fmt.Sprintf("fr%v", i)
		fr := row1.AddNewChild(gi.KiT_Frame, nm).(*gi.Frame)
		fr.SetProp("width", sz.X)
		fr.SetProp("height", sz.Y)
		fr.SetProp("align-vert", "inherit")
		// fr.SetProp("align-horiz", "inherit")
		fr.SetProp("margin", "inherit")
		if i == 2 {
			fr.SetFixedWidth(units.NewValue(20, units.Em))
			spc := row1.AddNewChild(gi.KiT_Space, "spc").(*gi.Space)
			spc.SetFixedWidth(units.NewValue(4, units.Em))
		} else {
			fr.SetProp("max-width", -1) // spacer
		}
	}

	row2 := vlay.AddNewChild(gi.KiT_Layout, "row2").(*gi.Layout)
	row2.Lay = gi.LayoutRow
	row2.SetProp("text-align", "center")
	row2.SetProp("max-width", -1) // always stretch width

	row2.SetProp("align-vert", "center")
	// row2.SetProp("align-horiz", "justify")
	row2.SetProp("align-horiz", "left")
	row2.SetProp("margin", 4.0)

	for i, sz := range frsz {
		nm := fmt.Sprintf("fr%v", i)
		fr := row2.AddNewChild(gi.KiT_Frame, nm).(*gi.Frame)
		fr.SetProp("width", sz.X)
		fr.SetProp("height", sz.Y)
		fr.SetProp("align-vert", "inherit")
		// fr.SetProp("align-horiz", "inherit")
		fr.SetProp("margin", "inherit")
		// if i == 2 {
		// 	row2.AddNewChild(gi.KiT_Stretch, "str")
		// }
	}

	row3 := vlay.AddNewChild(gi.KiT_Layout, "row3").(*gi.Layout)
	row3.Lay = gi.LayoutRow
	row3.SetProp("text-align", "center")
	// row3.SetProp("max-width", -1) // always stretch width

	row3.SetProp("align-vert", "bottom")
	row3.SetProp("align-horiz", "justify")
	// row3.SetProp("align-horiz", "left")
	row3.SetProp("margin", 4.0)

	for i, sz := range frsz {
		nm := fmt.Sprintf("fr%v", i)
		fr := row3.AddNewChild(gi.KiT_Frame, nm).(*gi.Frame)
		fr.SetProp("width", sz.X)
		fr.SetProp("height", sz.Y)
		fr.SetProp("min-height", sz.Y)
		fr.SetProp("align-vert", "inherit")
		// fr.SetProp("align-horiz", "inherit")
		fr.SetProp("margin", "inherit")
		// fr.SetProp("max-width", -1) // spacer
	}

	row4 := vlay.AddNewChild(gi.KiT_Layout, "row4").(*gi.Layout)
	row4.Lay = gi.LayoutGrid
	row4.SetProp("columns", 2)
	// row4.SetProp("max-width", -1)

	row4.SetProp("align-vert", "top")
	// row4.SetProp("align-horiz", "justify")
	row4.SetProp("align-horiz", "left")
	row4.SetProp("margin", 6.0)

	for i, sz := range frsz {
		nm := fmt.Sprintf("fr%v", i)
		fr := row4.AddNewChild(gi.KiT_Frame, nm).(*gi.Frame)
		fr.SetProp("width", sz.X)
		fr.SetProp("height", sz.Y)
		// fr.SetProp("min-height", sz.Y)
		fr.SetProp("align-vert", "inherit")
		fr.SetProp("align-horiz", "inherit")
		fr.SetProp("margin", 2.0)
		// fr.SetProp("max-width", -1) // spacer
	}

	vp.UpdateEndNoSig(updt)

	win.StartEventLoop()
}
