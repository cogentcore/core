// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"github.com/goki/gi/gi"
	"github.com/goki/gi/gimain"
	"github.com/goki/gi/oswin"
)

func main() {
	gimain.Main(func() {
		mainrun()
	})
}

func mainrun() {
	width := 1024
	height := 768
	win := gi.NewWindow2D("gogi-basic", "Basic Test Window", width, height, true)

	vp := win.WinViewport2D()
	updt := vp.UpdateStart()

	mfr := win.SetMainFrame()

	rlay := mfr.AddNewChild(gi.KiT_Layout, "rowlay").(*gi.Layout)
	rlay.Lay = gi.LayoutHoriz
	rlay.SetProp("text-align", "center")
	label1 := rlay.AddNewChild(gi.KiT_Label, "label1").(*gi.Label)
	// edit1 := rlay.AddNewChild(gi.KiT_TextField, "edit1").(*gi.TextField)
	// button1 := rlay.AddNewChild(gi.KiT_Button, "button1").(*gi.Button)
	// button2 := rlay.AddNewChild(gi.KiT_Button, "button2").(*gi.Button)
	// slider1 := rlay.AddNewChild(gi.KiT_Slider, "slider1").(*gi.Slider)
	// spin1 := rlay.AddNewChild(gi.KiT_SpinBox, "spin1").(*gi.SpinBox)

	label1.Text = "This is test text"
	// edit1.SetText("Edit this text")
	// edit1.SetProp("min-width", "20em")
	// button1.Text = "Button 1"
	// button2.Text = "Button 2"
	// slider1.Dim = gi.X
	// slider1.SetProp("width", "20em")
	// slider1.SetValue(0.5)
	// spin1.SetValue(0.0)

	// main menu
	appnm := oswin.TheApp.Name()
	mmen := win.MainMenu
	mmen.ConfigMenus([]string{appnm, "Edit", "Window"})

	amen := win.MainMenu.KnownChildByName(appnm, 0).(*gi.Action)
	amen.Menu = make(gi.Menu, 0, 10)
	amen.Menu.AddAppMenu(win)

	emen := win.MainMenu.KnownChildByName("Edit", 1).(*gi.Action)
	emen.Menu = make(gi.Menu, 0, 10)
	emen.Menu.AddCopyCutPaste(win)

	win.OSWin.SetCloseCleanFunc(func(w oswin.Window) {
		go oswin.TheApp.Quit() // once main window is closed, quit
	})

	win.MainMenuUpdated()

	vp.UpdateEndNoSig(updt)

	win.StartEventLoop()
}
