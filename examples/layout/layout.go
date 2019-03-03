// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"fmt"

	"github.com/goki/gi/gi"
	"github.com/goki/gi/gimain"
	"github.com/goki/gi/units"
)

func main() {
	gimain.Main(func() {
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

	gi.SetAppName("layout")
	gi.SetAppAbout(`This is a demo of the layout functions in the <b>GoGi</b> graphical interface system, within the <b>GoKi</b> tree framework.  See <a href="https://github.com/goki">GoKi on GitHub</a>`)

	win := gi.NewWindow2D("gogi-layout-test", "GoGi Layout Test", width, height, true)

	vp := win.WinViewport2D()
	updt := vp.UpdateStart()

	mfr := win.SetMainFrame()

	row1 := mfr.AddNewChild(gi.KiT_Layout, "row1").(*gi.Layout)
	row1.Lay = gi.LayoutHoriz

	row1.SetProp("vertical-align", "top")
	// row1.SetProp("horizontal-align", "justify")
	row1.SetProp("horizontal-align", "left")
	row1.SetProp("margin", 4.0)
	row1.SetProp("max-width", -1) // always stretch width
	row1.SetProp("spacing", 6.0)

	for i, sz := range frsz {
		nm := fmt.Sprintf("fr%v", i)
		fr := row1.AddNewChild(gi.KiT_Frame, nm).(*gi.Frame)
		fr.SetProp("width", sz.X)
		fr.SetProp("height", sz.Y)
		fr.SetProp("vertical-align", "inherit")
		// fr.SetProp("horizontal-align", "inherit")
		fr.SetProp("margin", "inherit")
		if i == 2 {
			fr.SetFixedWidth(units.NewValue(20, units.Em))
			spc := row1.AddNewChild(gi.KiT_Space, "spc").(*gi.Space)
			spc.SetFixedWidth(units.NewValue(4, units.Em))
		} else {
			fr.SetProp("max-width", -1) // spacer
		}
	}

	row2 := mfr.AddNewChild(gi.KiT_Layout, "row2").(*gi.Layout)
	row2.Lay = gi.LayoutHoriz
	row2.SetProp("text-align", "center")
	row2.SetProp("max-width", -1) // always stretch width

	row2.SetProp("vertical-align", "center")
	// row2.SetProp("horizontal-align", "justify")
	row2.SetProp("horizontal-align", "left")
	row2.SetProp("margin", 4.0)
	row2.SetProp("spacing", 6.0)

	for i, sz := range frsz {
		nm := fmt.Sprintf("fr%v", i)
		fr := row2.AddNewChild(gi.KiT_Frame, nm).(*gi.Frame)
		fr.SetProp("width", sz.X)
		fr.SetProp("height", sz.Y)
		fr.SetProp("vertical-align", "inherit")
		// fr.SetProp("horizontal-align", "inherit")
		fr.SetProp("margin", "inherit")
		// if i == 2 {
		// 	row2.AddNewChild(gi.KiT_Stretch, "str")
		// }
	}

	row3 := mfr.AddNewChild(gi.KiT_Layout, "row3").(*gi.Layout)
	row3.Lay = gi.LayoutHoriz
	row3.SetProp("text-align", "center")
	// row3.SetProp("max-width", -1) // always stretch width

	row3.SetProp("vertical-align", "bottom")
	row3.SetProp("horizontal-align", "justify")
	// row3.SetProp("horizontal-align", "left")
	row3.SetProp("margin", 4.0)
	row3.SetProp("spacing", 6.0)

	for i, sz := range frsz {
		nm := fmt.Sprintf("fr%v", i)
		fr := row3.AddNewChild(gi.KiT_Frame, nm).(*gi.Frame)
		fr.SetProp("width", sz.X)
		fr.SetProp("height", sz.Y)
		fr.SetProp("min-height", sz.Y)
		fr.SetProp("vertical-align", "inherit")
		// fr.SetProp("horizontal-align", "inherit")
		fr.SetProp("margin", "inherit")
		// fr.SetProp("max-width", -1) // spacer
	}

	row4 := mfr.AddNewChild(gi.KiT_Layout, "row4").(*gi.Layout)
	row4.Lay = gi.LayoutGrid
	row4.SetProp("columns", 2)
	// row4.SetProp("max-width", -1)

	row4.SetProp("vertical-align", "top")
	// row4.SetProp("horizontal-align", "justify")
	row4.SetProp("horizontal-align", "left")
	row4.SetProp("margin", 6.0)

	for i, sz := range frsz {
		nm := fmt.Sprintf("fr%v", i)
		fr := row4.AddNewChild(gi.KiT_Frame, nm).(*gi.Frame)
		fr.SetProp("width", sz.X)
		fr.SetProp("height", sz.Y)
		// fr.SetProp("min-height", sz.Y)
		fr.SetProp("vertical-align", "inherit")
		fr.SetProp("horizontal-align", "inherit")
		fr.SetProp("margin", 2.0)
		// fr.SetProp("max-width", -1) // spacer
	}

	// main menu
	appnm := gi.AppName()
	mmen := win.MainMenu
	mmen.ConfigMenus([]string{appnm, "Edit", "Window"})

	amen := win.MainMenu.ChildByName(appnm, 0).(*gi.Action)
	amen.Menu = make(gi.Menu, 0, 10)
	amen.Menu.AddAppMenu(win)

	emen := win.MainMenu.ChildByName("Edit", 1).(*gi.Action)
	emen.Menu = make(gi.Menu, 0, 10)
	emen.Menu.AddCopyCutPaste(win)

	win.SetCloseCleanFunc(func(w *gi.Window) {
		go gi.Quit() // once main window is closed, quit
	})

	win.MainMenuUpdated()

	vp.UpdateEndNoSig(updt)

	win.StartEventLoop()
}
