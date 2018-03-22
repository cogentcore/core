// Copyright (c) 2018, Randall C. O'Reilly. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"github.com/rcoreilly/goki/gi"
	_ "github.com/rcoreilly/goki/gi/init"
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

	vlay := vpfill.AddNewChildNamed(gi.KiT_Layout, "collay").(*gi.Layout)
	vlay.Layout = gi.LayoutCol

	rlay := vlay.AddNewChildNamed(gi.KiT_Layout, "rowlay1").(*gi.Layout)
	rlay.Layout = gi.LayoutRow

	rlay.SetProp("align-vert", "top")
	// rlay.SetProp("align-horiz", "hjustify")
	rlay.SetProp("align-horiz", "left")
	rlay.SetProp("margin", 4.0)
	rlay.SetProp("max-width", -1) // always stretch width

	for i, sz := range frsz {
		nm := fmt.Sprintf("fr%v", i)
		fr := rlay.AddNewChildNamed(gi.KiT_Frame, nm).(*gi.Frame)
		fr.SetProp("width", sz.X)
		fr.SetProp("height", sz.Y)
		fr.SetProp("align-vert", "inherit")
		// fr.SetProp("align-horiz", "inherit")
		fr.SetProp("margin", "inherit")
		if i != 2 {
			fr.SetProp("max-width", -1) // spacer
		}
	}

	rlay2 := vlay.AddNewChildNamed(gi.KiT_Layout, "rowlay2").(*gi.Layout)
	rlay2.Layout = gi.LayoutRow
	rlay2.SetProp("text-align", "center")
	rlay2.SetProp("max-width", -1) // always stretch width

	rlay2.SetProp("align-vert", "vcenter")
	// rlay2.SetProp("align-horiz", "hjustify")
	rlay2.SetProp("align-horiz", "left")
	rlay2.SetProp("margin", 4.0)

	for i, sz := range frsz {
		nm := fmt.Sprintf("fr%v", i)
		fr := rlay2.AddNewChildNamed(gi.KiT_Frame, nm).(*gi.Frame)
		fr.SetProp("width", sz.X)
		fr.SetProp("height", sz.Y)
		fr.SetProp("align-vert", "inherit")
		// fr.SetProp("align-horiz", "inherit")
		fr.SetProp("margin", "inherit")
		// fr.SetProp("max-width", -1) // spacer
	}

	rlay3 := vlay.AddNewChildNamed(gi.KiT_Layout, "rowlay3").(*gi.Layout)
	rlay3.Layout = gi.LayoutRow
	rlay3.SetProp("text-align", "center")
	// rlay3.SetProp("max-width", -1) // always stretch width

	rlay3.SetProp("align-vert", "bottom")
	rlay3.SetProp("align-horiz", "hjustify")
	// rlay3.SetProp("align-horiz", "left")
	rlay3.SetProp("margin", 4.0)

	for i, sz := range frsz {
		nm := fmt.Sprintf("fr%v", i)
		fr := rlay3.AddNewChildNamed(gi.KiT_Frame, nm).(*gi.Frame)
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
