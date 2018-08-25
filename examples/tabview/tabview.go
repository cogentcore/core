// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"github.com/goki/gi"
	"github.com/goki/gi/gimain"
	"github.com/goki/gi/giv"
	"github.com/goki/gi/oswin"
	"github.com/goki/ki"
)

func main() {
	gimain.Main(func() {
		mainrun()
	})
}

func mainrun() {
	// a source tree to view
	srctree := ki.Node{}
	srctree.InitName(&srctree, "par1")
	// child1 :=
	srctree.AddNewChild(nil, "child1")
	child2 := srctree.AddNewChild(nil, "child2")
	// child3 :=
	srctree.AddNewChild(nil, "child3")
	// schild2 :=
	child2.AddNewChild(nil, "subchild1")

	width := 1024
	height := 768

	win := gi.NewWindow2D("gogi-tabview-test", "GoGi TabView Test", width, height, true) // pixel sizes

	vp := win.WinViewport2D()
	updt := vp.UpdateStart()

	mfr := win.SetMainFrame()

	tv1 := mfr.AddNewChild(giv.KiT_TabView, "tv1").(*giv.TabView)
	tv1.SetSrcNode(&srctree)

	for i, sk := range srctree.Kids {
		tf := tv1.TabFrameAtIndex(i)
		lbl := tf.AddNewChild(gi.KiT_Label, "tst").(*gi.Label)
		lbl.Text = sk.UniqueName()
		// note: these were set by default -- could override
		// tf.SetProp("max-width", -1.0) // stretch flex
		// tf.SetProp("max-height", -1.0)
	}

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
