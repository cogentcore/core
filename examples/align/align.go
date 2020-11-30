// Copyright (c) 2020, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"github.com/goki/gi/gi"
	"github.com/goki/gi/gimain"
)

func main() {
	gimain.Main(func() {
		mainrun()
	})
}

func mainrun() {
	width := 1024
	height := 768
	win := gi.NewMainWindow("gogi-align", "Align Test Window", width, height)

	vp := win.WinViewport2D()
	updt := vp.UpdateStart()

	mfr := win.SetMainFrame()

	//	horizontal alignment only works within a vertical layout!!
	vlay := gi.AddNewFrame(mfr, "vlay", gi.LayoutVert)
	vlay.SetStretchMaxWidth()

	lbl := gi.AddNewLabel(vlay, "l", "left aligned (default)") // nothing interesting about left..

	// center

	lbl = gi.AddNewLabel(vlay, "c", "horizontal-align=center")
	lbl.SetProp("horizontal-align", "center")

	// note: this has no effect because text-align is relative to the label container,
	// which is auto-sized based on text size, so..
	lbl = gi.AddNewLabel(vlay, "tc", "text-align=center (no effect)")
	lbl.SetProp("text-align", "center")

	lbl = gi.AddNewLabel(vlay, "tcm", "text-align=center\nnow\nhas effect\nbecause of multi-line")
	lbl.SetProp("text-align", "center")

	lbl = gi.AddNewLabel(vlay, "tcs", "text-align=center, StretchMaxWidth")
	lbl.SetProp("text-align", "center")
	lbl.SetStretchMaxWidth()

	lbl = gi.AddNewLabel(vlay, "tcf", "text-align=center, width=50em")
	lbl.SetProp("text-align", "center")
	lbl.SetProp("width", "50em")
	lbl.SetProp("border-width", 1)

	lbl = gi.AddNewLabel(vlay, "ctcf", "h-align=center, text-align=center, width=50em")
	lbl.SetProp("horizontal-align", "center")
	lbl.SetProp("text-align", "center")
	lbl.SetProp("width", "50em")
	lbl.SetProp("border-width", 1)

	// right

	lbl = gi.AddNewLabel(vlay, "r", "horizontal-align=right")
	lbl.SetProp("horizontal-align", "right")

	// note: this has no effect because text-align is relative to the label container,
	// which is auto-sized based on text size, so..
	lbl = gi.AddNewLabel(vlay, "tr", "text-align=right (no effect)")
	lbl.SetProp("text-align", "right")

	lbl = gi.AddNewLabel(vlay, "trm", "text-align=right\nnow\nhas effect\nbecause of multi-line")
	lbl.SetProp("text-align", "right")

	lbl = gi.AddNewLabel(vlay, "trs", "text-align=right, StretchMaxWidth")
	lbl.SetProp("text-align", "right")
	lbl.SetStretchMaxWidth()

	lbl = gi.AddNewLabel(vlay, "trf", "text-align=right, width=50em")
	lbl.SetProp("text-align", "right")
	lbl.SetProp("width", "50em")
	lbl.SetProp("border-width", 1)

	lbl = gi.AddNewLabel(vlay, "ctrf", "h-align=right, text-align=right, width=50em")
	lbl.SetProp("horizontal-align", "right")
	lbl.SetProp("text-align", "right")
	lbl.SetProp("width", "50em")
	lbl.SetProp("border-width", 1)

	// justify

	lbl = gi.AddNewLabel(vlay, "tjm", "text-align=justify\nnow it\nhas effect\nbecause of multi-line\n(not yet impl)")
	lbl.SetProp("text-align", "justify")

	//	vertical alignment only works within a horizontallayout!!
	hlay := gi.AddNewFrame(mfr, "hlay", gi.LayoutHoriz)
	// hlay.SetStretchMaxWidth()

	lbl = gi.AddNewLabel(hlay, "t", "top aligned (default)") // nothing interesting about top..

	// middle

	lbl = gi.AddNewLabel(hlay, "c", "vertical-align=middle")
	lbl.SetProp("vertical-align", "middle")

	lbl = gi.AddNewLabel(hlay, "tcs", "text-vertical-align=middle, StretchMaxHeight")
	lbl.SetProp("text-vertical-align", "middle")
	lbl.SetStretchMaxHeight()

	lbl = gi.AddNewLabel(hlay, "tcf", "text-vertical-align=middle, height=20em")
	lbl.SetProp("text-vertical-align", "middle")
	lbl.SetProp("height", "20em")
	lbl.SetProp("border-height", 1)

	// bottom

	lbl = gi.AddNewLabel(hlay, "r", "vertical-align=bottom")
	lbl.SetProp("vertical-align", "bottom")

	lbl = gi.AddNewLabel(hlay, "trs", "text-vertical-align=bottom, StretchMaxHeight")
	lbl.SetProp("text-vertical-align", "bottom")
	lbl.SetStretchMaxHeight()

	lbl = gi.AddNewLabel(hlay, "trf", "text-vertical-align=bottom, height=20em")
	lbl.SetProp("text-vertical-align", "bottom")
	lbl.SetProp("height", "20em")
	lbl.SetProp("border-width", 1)

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
