// Copyright (c) 2020, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"goki.dev/gi/v2/gi"
	"goki.dev/gi/v2/gimain"
)

func main() {
	gimain.Main(mainrun)
}

func mainrun() {
	width := 1024
	height := 768
	win := gi.NewMainRenderWin("gogi-align", "Align Test RenderWin", width, height)

	vp := win.WinScene()
	updt := vp.UpdateStart()

	mfr := win.SetMainFrame()

	//	horizontal alignment only works within a vertical layout!!
	vlay := gi.NewFrame(mfr, "vlay", gi.LayoutVert)
	vlay.SetStretchMaxWidth()

	lbl := gi.NewLabel(vlay, "l", "left aligned (default)") // nothing interesting about left..

	// center

	lbl = gi.NewLabel(vlay, "c", "horizontal-align=center")
	lbl.SetProp("horizontal-align", "center")

	// note: this has no effect because text-align is relative to the label container,
	// which is auto-sized based on text size, so..
	lbl = gi.NewLabel(vlay, "tc", "text-align=center (no effect)")
	lbl.SetProp("text-align", "center")

	lbl = gi.NewLabel(vlay, "tcm", "text-align=center\nnow\nhas effect\nbecause of multi-line")
	lbl.SetProp("text-align", "center")

	lbl = gi.NewLabel(vlay, "tcs", "text-align=center, StretchMaxWidth")
	lbl.SetProp("text-align", "center")
	lbl.SetStretchMaxWidth()

	lbl = gi.NewLabel(vlay, "tcf", "text-align=center, width=50em")
	lbl.SetProp("text-align", "center")
	lbl.SetProp("width", "50em")
	lbl.SetProp("border-width", 1)

	lbl = gi.NewLabel(vlay, "ctcf", "h-align=center, text-align=center, width=50em")
	lbl.SetProp("horizontal-align", "center")
	lbl.SetProp("text-align", "center")
	lbl.SetProp("width", "50em")
	lbl.SetProp("border-width", 1)

	// right

	lbl = gi.NewLabel(vlay, "r", "horizontal-align=right")
	lbl.SetProp("horizontal-align", "right")

	// note: this has no effect because text-align is relative to the label container,
	// which is auto-sized based on text size, so..
	lbl = gi.NewLabel(vlay, "tr", "text-align=right (no effect)")
	lbl.SetProp("text-align", "right")

	lbl = gi.NewLabel(vlay, "trm", "text-align=right\nnow\nhas effect\nbecause of multi-line")
	lbl.SetProp("text-align", "right")

	lbl = gi.NewLabel(vlay, "trs", "text-align=right, StretchMaxWidth")
	lbl.SetProp("text-align", "right")
	lbl.SetStretchMaxWidth()

	lbl = gi.NewLabel(vlay, "trf", "text-align=right, width=50em")
	lbl.SetProp("text-align", "right")
	lbl.SetProp("width", "50em")
	lbl.SetProp("border-width", 1)

	lbl = gi.NewLabel(vlay, "ctrf", "h-align=right, text-align=right, width=50em")
	lbl.SetProp("horizontal-align", "right")
	lbl.SetProp("text-align", "right")
	lbl.SetProp("width", "50em")
	lbl.SetProp("border-width", 1)

	// justify

	lbl = gi.NewLabel(vlay, "tjm", "text-align=justify\nnow it\nhas effect\nbecause of multi-line\n(not yet impl)")
	lbl.SetProp("text-align", "justify")

	//	vertical alignment only works within a horizontallayout!!
	hlay := gi.NewFrame(mfr, "hlay", gi.LayoutHoriz)
	// hlay.SetStretchMaxWidth()

	lbl = gi.NewLabel(hlay, "t", "top aligned (default)") // nothing interesting about top..

	// middle

	lbl = gi.NewLabel(hlay, "c", "vertical-align=middle")
	lbl.SetProp("vertical-align", "middle")

	lbl = gi.NewLabel(hlay, "tcs", "text-vertical-align=middle, StretchMaxHeight")
	lbl.SetProp("text-vertical-align", "middle")
	lbl.SetStretchMaxHeight()

	lbl = gi.NewLabel(hlay, "tcf", "text-vertical-align=middle, height=20em")
	lbl.SetProp("text-vertical-align", "middle")
	lbl.SetProp("height", "20em")
	lbl.SetProp("border-height", 1)

	// bottom

	lbl = gi.NewLabel(hlay, "r", "vertical-align=bottom")
	lbl.SetProp("vertical-align", "bottom")

	lbl = gi.NewLabel(hlay, "trs", "text-vertical-align=bottom, StretchMaxHeight")
	lbl.SetProp("text-vertical-align", "bottom")
	lbl.SetStretchMaxHeight()

	lbl = gi.NewLabel(hlay, "trf", "text-vertical-align=bottom, height=20em")
	lbl.SetProp("text-vertical-align", "bottom")
	lbl.SetProp("height", "20em")
	lbl.SetProp("border-width", 1)

	// main menu
	appnm := gi.AppName()
	mmen := win.MainMenu
	mmen.ConfigMenus([]string{appnm, "Edit", "RenderWin"})

	amen := win.MainMenu.ChildByName(appnm, 0).(*gi.Action)
	amen.Menu = make(gi.Menu, 0, 10)
	amen.Menu.AddAppMenu(win)

	emen := win.MainMenu.ChildByName("Edit", 1).(*gi.Action)
	emen.Menu = make(gi.Menu, 0, 10)
	emen.Menu.AddCopyCutPaste(win)

	win.SetCloseCleanFunc(func(w *gi.RenderWin) {
		go gi.Quit() // once main window is closed, quit
	})

	win.MainMenuUpdated()
	vp.UpdateEndNoSig(updt)
	win.StartEventLoop()
}
