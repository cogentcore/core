// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"image/color"

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
	nColumns := 5

	gi.SetAppName("icons")
	gi.SetAppAbout(`This is a demo of the icons in the <b>GoGi</b> graphical interface system, within the <b>GoKi</b> tree framework.  See <a href="https://github.com/goki">GoKi on GitHub</a>`)

	// note: can add a path to view other icon sets
	// svg.CurIconSet.OpenIconsFromPath("/Users/oreilly/github/inkscape/share/icons/multicolor/symbolic/actions")

	win := gi.NewMainWindow("gogi-icons-demo", "GoGi Icons", width, height)

	vp := win.WinViewport2D()
	updt := vp.UpdateStart()

	mfr := win.SetMainFrame()

	row1 := gi.AddNewLayout(mfr, "row1", gi.LayoutHoriz)
	row1.SetProp("margin", 2.0) // raw numbers = px = 96 dpi pixels
	row1.SetStretchMaxWidth()

	spc := gi.AddNewSpace(mfr, "spc1")
	spc.SetFixedHeight(units.NewEm(2))

	gi.AddNewStretch(row1, "str1")
	lab1 := gi.AddNewLabel(row1, "lab1", "These are all the GoGi Icons, in a small and large size")
	lab1.SetProp("max-width", -1)
	lab1.SetProp("text-align", "center")
	gi.AddNewStretch(row1, "str2")

	grid := gi.AddNewFrame(mfr, "grid", gi.LayoutGrid)
	grid.Stripes = gi.RowStripes
	grid.SetProp("columns", nColumns)
	grid.SetProp("horizontal-align", "center")
	grid.SetProp("margin", 2.0)
	grid.SetProp("spacing", 6.0)
	grid.SetStretchMaxWidth()
	grid.SetStretchMaxHeight()

	il := gi.TheIconMgr.IconList(true)

	for _, icnm := range il {
		icnms := string(icnm)
		if icnms == "none" {
			continue
		}
		vb := gi.AddNewLayout(grid, "vb", gi.LayoutVert)
		gi.AddNewLabel(vb, "lab1", icnms)

		smico := gi.AddNewIcon(vb, icnms, icnms)
		smico.SetMinPrefWidth(units.NewPx(20))
		smico.SetMinPrefHeight(units.NewPx(20))
		smico.SetProp("background-color", color.Transparent)
		smico.SetProp("fill", "#88F")
		smico.SetProp("stroke", "black")
		// smico.SetProp("horizontal-align", gi.AlignLeft)

		ico := gi.AddNewIcon(vb, icnms+"_big", icnms)
		ico.SetMinPrefWidth(units.NewPx(100))
		ico.SetMinPrefHeight(units.NewPx(100))
		ico.SetProp("background-color", color.Transparent)
		ico.SetProp("fill", "#88F")
		ico.SetProp("stroke", "black")
		// ico.SetProp("horizontal-align", gi.AlignLeft)
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
