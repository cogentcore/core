// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"goki.dev/gi/v2/gi"
	"goki.dev/gi/v2/gimain"
	"goki.dev/gi/v2/gist"
	"goki.dev/gi/v2/giv"
)

func main() {
	gimain.Main(mainrun)
}

func mainrun() {
	width := 1024
	height := 768

	win := gi.NewMainWindow("gogi-tabview-test", "GoGi TabView Test", width, height)

	vp := win.WinViewport2D()
	updt := vp.UpdateStart()

	mfr := win.SetMainFrame()

	tv := gi.AddNewTabView(mfr, "tv")
	tv.NewTabButton = true

	lbl1 := tv.AddNewTab(gi.TypeLabel, "This is Label1").(*gi.Label)
	lbl1.SetText("this is the contents of the first tab")
	lbl1.SetProp("white-space", gist.WhiteSpaceNormal) // wrap

	lbl2 := tv.AddNewTab(gi.TypeLabel, "And this Label2").(*gi.Label)
	lbl2.SetText("this is the contents of the second tab")
	lbl2.SetProp("white-space", gist.WhiteSpaceNormal) // wrap

	tv1i, tv1ly := tv.AddNewTabLayout(giv.TypeTextView, "TextView1")
	tv1ly.SetStretchMax()
	tv1 := tv1i.(*giv.TextView)
	tb1 := &giv.TextBuf{}
	tb1.InitName(tb1, "tb1")
	tv1.SetBuf(tb1)
	tb1.SetText([]byte("TextView1 text"))

	tv.SelectTabIndex(0)

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
