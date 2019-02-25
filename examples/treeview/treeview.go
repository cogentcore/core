// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"image"

	"github.com/goki/gi/gi"
	"github.com/goki/gi/gimain"
	"github.com/goki/gi/giv"
	"github.com/goki/gi/oswin"
	"github.com/goki/gi/units"
	"github.com/goki/ki/ki"
	"github.com/goki/ki/kit"
)

// todo: enum field, etc

// A node for testing
type TestNodeA struct {
	ki.Node
	StrField   string          `desc:"a string"`
	IntField   int             `min:"5" max:"25" step:"5" desc:"an int"`
	FloatField float64         `min:"-1" max:"1" step:".25" desc:"a float"`
	BoolField  bool            `desc:"a bool"`
	Vec        gi.Vec2D        `desc:"a vec"`
	Rect       image.Rectangle `desc:"rect"`
}

var KiT_TestNodeA = kit.Types.AddType(&TestNodeA{}, nil)

// B node for testing
type TestNodeB struct {
	ki.Node
	StrField   string          `desc:"a string"`
	IntField   int             `min:"5" max:"25" step:"5" desc:"an int"`
	FloatField float64         `min:"-1" max:"1" step:".25" desc:"a float"`
	BoolField  bool            `desc:"a bool"`
	Vec        gi.Vec2D        `desc:"a vec"`
	Rect       image.Rectangle `desc:"rect"`
	SubObj     TestNodeA       `desc:"a sub-object"`
}

var KiT_TestNodeB = kit.Types.AddType(&TestNodeB{}, nil)

func main() {
	gimain.Main(func() {
		mainrun()
	})
}

func mainrun() {
	// a source tree to view
	srctree := TestNodeB{}
	srctree.InitName(&srctree, "par1")
	// child1 :=
	srctree.AddNewChild(nil, "child1")
	child2 := srctree.AddNewChild(nil, "child2")
	// child3 :=
	srctree.AddNewChild(nil, "child3")
	// schild2 :=
	child2.AddNewChild(nil, "subchild1")

	srctree.SetProp("test1", "string val")
	srctree.SetProp("test2", 3.14)
	srctree.SetProp("test3", false)

	// turn this on to see a trace of the rendering
	// gi.Render2DTrace = true
	// gi.Layout2DTrace = true

	oswin.TheApp.SetName("treeview")
	oswin.TheApp.SetAbout(`This is a demo of the treeview in the <b>GoGi</b> graphical interface system, within the <b>GoKi</b> tree framework.  See <a href="https://github.com/goki">GoKi on GitHub</a>
<p>Full Drag-and-Drop, Copy / Cut / Paste, and Keyboard Navigation is supported.</p>`)

	width := 1024
	height := 768
	win := gi.NewWindow2D("gogi-treeview-test", "TreeView Test", width, height, true)

	vp := win.WinViewport2D()
	updt := vp.UpdateStart()

	mfr := win.SetMainFrame()

	trow := mfr.AddNewChild(gi.KiT_Layout, "trow").(*gi.Layout)
	trow.Lay = gi.LayoutHoriz
	trow.SetProp("horizontal-align", "center")
	trow.SetProp("margin", 2.0) // raw numbers = px = 96 dpi pixels
	trow.SetStretchMaxWidth()

	spc := mfr.AddNewChild(gi.KiT_Space, "spc1").(*gi.Space)
	spc.SetFixedHeight(units.NewValue(2.0, units.Em))

	trow.AddNewChild(gi.KiT_Stretch, "str1")
	lab1 := trow.AddNewChild(gi.KiT_Label, "lab1").(*gi.Label)
	lab1.Text = "This is a test of the TreeView and StructView reflect-ive GUI"
	lab1.SetStretchMaxWidth()
	lab1.SetProp("text-align", "center")
	trow.AddNewChild(gi.KiT_Stretch, "str2")

	split := mfr.AddNewChild(gi.KiT_SplitView, "split").(*gi.SplitView)
	split.Dim = gi.X

	tvfr := split.AddNewChild(gi.KiT_Frame, "tvfr").(*gi.Frame)
	svfr := split.AddNewChild(gi.KiT_Frame, "svfr").(*gi.Frame)
	split.SetSplits(.3, .7)

	tv := tvfr.AddNewChild(giv.KiT_TreeView, "tv").(*giv.TreeView)
	tv.SetRootNode(&srctree)

	sv := svfr.AddNewChild(giv.KiT_StructView, "sv").(*giv.StructView)
	sv.SetStretchMaxWidth()
	sv.SetStretchMaxHeight()
	sv.SetStruct(&srctree, nil)

	tv.TreeViewSig.Connect(sv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
		if data == nil {
			return
		}
		// tvr, _ := send.Embed(giv.KiT_TreeView).(*gi.TreeView) // root is sender
		tvn, _ := data.(ki.Ki).Embed(giv.KiT_TreeView).(*giv.TreeView)
		svr, _ := recv.Embed(giv.KiT_StructView).(*giv.StructView)
		if sig == int64(giv.TreeViewSelected) {
			svr.SetStruct(tvn.SrcNode.Ptr, nil)
		}
	})

	// main menu
	appnm := oswin.TheApp.Name()
	mmen := win.MainMenu
	mmen.ConfigMenus([]string{appnm, "Edit", "Window"})

	amen := win.MainMenu.ChildByName(appnm, 0).(*gi.Action)
	amen.Menu = make(gi.Menu, 0, 10)
	amen.Menu.AddAppMenu(win)

	emen := win.MainMenu.ChildByName("Edit", 1).(*gi.Action)
	emen.Menu = make(gi.Menu, 0, 10)
	emen.Menu.AddCopyCutPaste(win)

	win.OSWin.SetCloseCleanFunc(func(w oswin.Window) {
		go oswin.TheApp.Quit() // once main window is closed, quit
	})

	win.MainMenuUpdated()

	vp.UpdateEndNoSig(updt)

	win.StartEventLoop()
}
