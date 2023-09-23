// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"strings"

	"goki.dev/colors"
	"goki.dev/gi/v2/gi"
	"goki.dev/gi/v2/gimain"
	"goki.dev/girl/units"
)

func main() {
	gimain.Main(mainrun)
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

	vp := win.WinScene()
	updt := vp.UpdateStart()

	mfr := win.SetMainFrame()

	row1 := gi.NewLayout(mfr, "row1", gi.LayoutHoriz)
	row1.SetProp("margin", units.Px(2))
	row1.SetStretchMaxWidth()

	spc := gi.NewSpace(mfr, "spc1")
	spc.SetFixedHeight(units.Em(2))

	gi.NewStretch(row1, "str1")
	lab1 := gi.NewLabel(row1, "lab1", "These are all of the GoGi Icons")
	lab1.SetProp("max-width", -1)
	lab1.SetProp("text-align", "center")
	gi.NewStretch(row1, "str2")

	grid := gi.NewFrame(mfr, "grid", gi.LayoutGrid)
	grid.Stripes = gi.RowStripes
	grid.SetProp("columns", nColumns)
	grid.SetProp("horizontal-align", "center")
	grid.SetProp("margin", units.Px(1))
	// grid.SetProp("spacing", units.Px(1))
	grid.SetStretchMaxWidth()
	grid.SetStretchMaxHeight()

	il := gi.TheIconMgr.IconList(true)

	for _, icnm := range il {
		icnms := string(icnm)
		if icnm.IsNil() || strings.HasSuffix(icnms, "-fill") {
			continue
		}
		vb := gi.NewLayout(grid, "vb", gi.LayoutVert)
		vb.SetProp("max-width", "19vw")
		vb.SetProp("overflow", "hidden")
		gi.NewLabel(vb, "lab1", icnms)

		smico := gi.NewIcon(vb, icnms, icnm)
		smico.SetMinPrefWidth(units.Px(24))
		smico.SetMinPrefHeight(units.Px(24))
		smico.SetProp("background-color", colors.Transparent)
		smico.SetProp("fill", gi.ColorScheme.OnBackground)
		smico.SetProp("stroke", gi.ColorScheme.OnBackground)
		// smico.SetProp("horizontal-align", gi.AlignLeft)

		// ico := gi.NewIcon(vb, icnms+"_big", icnms)
		// ico.SetMinPrefWidth(units.Px(100))
		// ico.SetMinPrefHeight(units.Px(100))
		// ico.SetProp("background-color", colors.Transparent)
		// ico.SetProp("fill", "#88F")
		// ico.SetProp("stroke", "black")
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
