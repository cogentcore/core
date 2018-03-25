// Copyright (c) 2018, Randall C. O'Reilly. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"github.com/rcoreilly/goki/gi"
	_ "github.com/rcoreilly/goki/gi/init"
	"github.com/rcoreilly/goki/gi/units"
	// "math/rand"
	// "reflect"
	// "runtime"
	// "sync"
	// "time"
	// "image"
	// "image/draw"
)

func main() {
	go mainrun()
	gi.RunBackendEventLoop() // this needs to run in main loop
}

func mainrun() {
	width := 800
	height := 800
	win := gi.NewWindow2D("test window", width, height)
	win.UpdateStart()

	frsz := [5]gi.Vec2D{
		{20, 100},
		{80, 20},
		{60, 80},
		{40, 120},
		{150, 100},
	}

	vp := win.WinViewport2D()

	vpfill := vp.AddNewChildNamed(gi.KiT_Viewport2DFill, "vpfill").(*gi.Viewport2DFill)
	vpfill.SetProp("fill", "#FFF")

	vlay := vpfill.AddNewChildNamed(gi.KiT_Frame, "vlay").(*gi.Frame)
	vlay.Lay = gi.LayoutCol

	row1 := vlay.AddNewChildNamed(gi.KiT_Layout, "row1").(*gi.Layout)
	row1.Lay = gi.LayoutRow

	row1.SetProp("align-vert", "top")
	// row1.SetProp("align-horiz", "justify")
	row1.SetProp("align-horiz", "left")
	row1.SetProp("margin", 4.0)
	row1.SetProp("max-width", -1) // always stretch width

	for i, sz := range frsz {
		nm := fmt.Sprintf("fr%v", i)
		fr := row1.AddNewChildNamed(gi.KiT_Frame, nm).(*gi.Frame)
		fr.SetProp("width", sz.X)
		fr.SetProp("height", sz.Y)
		fr.SetProp("align-vert", "inherit")
		// fr.SetProp("align-horiz", "inherit")
		fr.SetProp("margin", "inherit")
		if i == 2 {
			fr.SetFixedWidth(units.NewValue(20, units.Em))
			spc := row1.AddNewChildNamed(gi.KiT_Space, "spc").(*gi.Space)
			spc.SetFixedWidth(units.NewValue(4, units.Em))
		} else {
			fr.SetProp("max-width", -1) // spacer
		}
	}

	row2 := vlay.AddNewChildNamed(gi.KiT_Layout, "row2").(*gi.Layout)
	row2.Lay = gi.LayoutRow
	row2.SetProp("text-align", "center")
	row2.SetProp("max-width", -1) // always stretch width

	row2.SetProp("align-vert", "center")
	// row2.SetProp("align-horiz", "justify")
	row2.SetProp("align-horiz", "left")
	row2.SetProp("margin", 4.0)

	for i, sz := range frsz {
		nm := fmt.Sprintf("fr%v", i)
		fr := row2.AddNewChildNamed(gi.KiT_Frame, nm).(*gi.Frame)
		fr.SetProp("width", sz.X)
		fr.SetProp("height", sz.Y)
		fr.SetProp("align-vert", "inherit")
		// fr.SetProp("align-horiz", "inherit")
		fr.SetProp("margin", "inherit")
		if i == 2 {
			row2.AddNewChildNamed(gi.KiT_Stretch, "str")
		}
	}

	row3 := vlay.AddNewChildNamed(gi.KiT_Layout, "row3").(*gi.Layout)
	row3.Lay = gi.LayoutRow
	row3.SetProp("text-align", "center")
	// row3.SetProp("max-width", -1) // always stretch width

	row3.SetProp("align-vert", "bottom")
	row3.SetProp("align-horiz", "justify")
	// row3.SetProp("align-horiz", "left")
	row3.SetProp("margin", 4.0)

	for i, sz := range frsz {
		nm := fmt.Sprintf("fr%v", i)
		fr := row3.AddNewChildNamed(gi.KiT_Frame, nm).(*gi.Frame)
		fr.SetProp("width", sz.X)
		fr.SetProp("height", sz.Y)
		fr.SetProp("min-height", sz.Y)
		fr.SetProp("align-vert", "inherit")
		// fr.SetProp("align-horiz", "inherit")
		fr.SetProp("margin", "inherit")
		// fr.SetProp("max-width", -1) // spacer
	}

	win.UpdateEnd()

	win.StartEventLoop()
}
