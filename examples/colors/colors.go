// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"github.com/goki/gi/gi"
	"github.com/goki/gi/gimain"
	"github.com/goki/gi/gist"
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

	gi.SetAppName("colors")
	gi.SetAppAbout(`This is a demo of the color space functions in the <b>GoGi</b> graphical interface system, within the <b>GoKi</b> tree framework.  See <a href="https://github.com/goki">GoKi on GitHub</a>`)

	win := gi.NewMainWindow("gogi-colors-test", "GoGi Colors Test", width, height)

	vp := win.WinViewport2D()
	updt := vp.UpdateStart()

	mfr := win.SetMainFrame()

	row1 := gi.AddNewLayout(mfr, "row1", gi.LayoutHoriz)
	row1.SetProp("vertical-align", gist.AlignMiddle)
	row1.SetProp("horizontal-align", "center")
	row1.SetProp("margin", 2.0) // raw numbers = px = 96 dpi pixels
	row1.SetStretchMaxWidth()

	spc := gi.AddNewSpace(mfr, "spc1")
	spc.SetFixedHeight(units.NewEm(2))

	gi.AddNewStretch(row1, "str1")
	lab1 := gi.AddNewLabel(row1, "lab1", "These are tests of the various GoGi Color functions")
	lab1.SetProp("max-width", -1)
	lab1.SetProp("text-align", "center")
	gi.AddNewStretch(row1, "str2")

	grid := gi.AddNewLayout(mfr, "grid", gi.LayoutGrid)
	grid.SetProp("columns", 11)
	grid.SetProp("vertical-align", "center")
	grid.SetProp("horizontal-align", "center")
	grid.SetProp("margin", 2.0)
	grid.SetStretchMaxWidth()
	grid.SetStretchMaxHeight()

	// first test the HSL color scheme
	var hues = [...]float32{0, 60, 120, 180, 240, 300}
	sat := float32(1.0)

	for _, hu := range hues {
		for lt := float32(0.0); lt <= 1.01; lt += 0.1 {
			fr := gi.AddNewFrame(grid, "fr", gi.LayoutHoriz)
			fr.SetProp("background-color", gist.HSLA{hu, sat, lt, 1.0})
			fr.SetProp("max-width", -1)
			fr.SetProp("max-height", -1)
		}
	}
	// try again with alpha
	for _, hu := range hues {
		for lt := float32(0.0); lt <= 1.01; lt += 0.1 {
			fr := gi.AddNewFrame(grid, "fr", gi.LayoutHoriz)
			fr.SetProp("background-color", gist.HSLA{hu, sat, lt, 0.5})
			fr.SetProp("max-width", -1)
			fr.SetProp("max-height", -1)
		}
	}
	// then sats
	lt := float32(0.5)
	for _, hu := range hues {
		for sat := float32(0.0); sat <= 1.01; sat += 0.1 {
			fr := gi.AddNewFrame(grid, "fr", gi.LayoutHoriz)
			fr.SetProp("background-color", gist.HSLA{hu, sat, lt, 1.0})
			fr.SetProp("max-width", -1)
			fr.SetProp("max-height", -1)
		}
	}
	// then doing it with colors -- tests the "there and back again" round trip..
	for _, hu := range hues {
		clr := gist.Color{}
		clr.SetHSLA(hu, 1.0, 0.2, 1)
		for lt := float32(0.0); lt <= 100.01; lt += 10 {
			fr := gi.AddNewFrame(grid, "fr", gi.LayoutHoriz)
			fr.SetProp("background-color", clr.Lighter(lt))
			fr.SetProp("max-width", -1)
			fr.SetProp("max-height", -1)
		}
	}

	// main menu
	appnm := gi.AppName()
	mmen := win.MainMenu
	mmen.ConfigMenus([]string{appnm, "Edit", "Window"})

	amen := win.MainMenu.ChildByName(appnm, 0).(*gi.Action)
	amen.Menu = make(gi.Menu, 0, 10)
	amen.Menu.AddAppMenu(win)

	// note: Command in shortcuts is automatically translated into Control for
	// Linux, Windows or Meta for MacOS

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
