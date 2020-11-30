// Copyright (c) 2018, The GoKi Authors. All rights reserved.
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
	win := gi.NewMainWindow("gogi-basic", "Basic Test Window", width, height)

	vp := win.WinViewport2D()
	updt := vp.UpdateStart()

	mfr := win.SetMainFrame()

	rlay := gi.AddNewFrame(mfr, "rowlay", gi.LayoutHoriz)
	rlay.SetProp("text-align", "center")
	rlay.SetStretchMaxWidth()
	lbl := gi.AddNewLabel(rlay, "label1", "This is test text")
	lbl.SetProp("text-align", "center")
	lbl.SetProp("border-width", 1)
	lbl.SetStretchMaxWidth()

	// edit1 := gi.AddNewTextField(rlay, "edit1")
	// button1 := gi.AddNewButton(rlay, "button1")
	// button2 := gi.AddNewButton(rlay, "button2")
	// slider1 := gi.AddNewSlider(rlay, "slider1")
	// spin1 := gi.AddNewSpinBox(rlay, "spin1")

	// edit1.SetText("Edit this text")
	// edit1.SetProp("min-width", "20em")
	// button1.Text = "Button 1"
	// button2.Text = "Button 2"
	// slider1.Dim = gi.X
	// slider1.SetProp("width", "20em")
	// slider1.SetValue(0.5)
	// spin1.SetValue(0.0)

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
