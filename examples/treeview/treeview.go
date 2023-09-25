// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"image"

	"goki.dev/gi/v2/gi"
	"goki.dev/gi/v2/gimain"
	"goki.dev/gi/v2/giv"
	"goki.dev/girl/units"
	"goki.dev/ki/v2"
	"goki.dev/mat32/v2"
)

// todo: enum field, etc

// A node for testing
type TestNodeA struct {
	ki.Node

	// a string
	StrField string `desc:"a string"`

	// [min: 5] [max: 25] [step: 5] an int
	IntField int `min:"5" max:"25" step:"5" desc:"an int"`

	// [min: -1] [max: 1] [step: .25] a float
	FloatField float64 `min:"-1" max:"1" step:".25" desc:"a float"`

	// a bool
	BoolField bool `desc:"a bool"`

	// a vec
	Vec mat32.Vec2 `desc:"a vec"`

	// rect
	Rect image.Rectangle `desc:"rect"`
}

// B node for testing
type TestNodeB struct {
	ki.Node

	// a string
	StrField string `desc:"a string"`

	// [min: 5] [max: 25] [step: 5] an int
	IntField int `min:"5" max:"25" step:"5" desc:"an int"`

	// [min: -1] [max: 1] [step: .25] a float
	FloatField float64 `min:"-1" max:"1" step:".25" desc:"a float"`

	// a bool
	BoolField bool `desc:"a bool"`

	// a vec
	Vec mat32.Vec2 `desc:"a vec"`

	// rect
	Rect image.Rectangle `desc:"rect"`

	// a sub-object
	SubObj TestNodeA `desc:"a sub-object"`
}

func main() {
	gimain.Main(mainrun)
}

func mainrun() {
	// a source tree to view
	srctree := TestNodeB{}
	srctree.InitName(&srctree, "par1")
	// child1 :=
	srctree.NewChild(TypeTestNodeB, "child1")
	child2 := srctree.NewChild(TypeTestNodeB, "child2")
	// child3 :=
	srctree.NewChild(TypeTestNodeB, "child3")
	// schild2 :=
	child2.NewChild(TypeTestNodeB, "subchild1")

	srctree.SetProp("test1", "string val")
	srctree.SetProp("test2", 3.14)
	srctree.SetProp("test3", false)

	// turn this on to see a trace of the rendering
	// gi.RenderTrace = true
	// gi.LayoutTrace = true

	gi.SetAppName("treeview")
	gi.SetAppAbout(`This is a demo of the treeview in the <b>GoGi</b> graphical interface system, within the <b>GoKi</b> tree framework.  See <a href="https://github.com/goki">GoKi on GitHub</a>
<p>Full Drag-and-Drop, Copy / Cut / Paste, and Keyboard Navigation is supported.</p>`)

	width := 1024
	height := 768
	win := gi.NewMainRenderWin("gogi-treeview-test", "TreeView Test", width, height)

	vp := win.WinScene()
	updt := vp.UpdateStart()

	mfr := win.SetMainFrame()

	trow := gi.NewLayout(mfr, "trow", gi.LayoutHoriz)
	trow.SetProp("horizontal-align", "center")
	trow.SetProp("margin", 2.0) // raw numbers = px = 96 dpi pixels
	trow.SetStretchMaxWidth()

	spc := gi.NewSpace(mfr, "spc1")
	spc.SetFixedHeight(units.Em(2))

	gi.NewStretch(trow, "str1")
	lab1 := gi.NewLabel(trow, "lab1", "This is a test of the TreeView and StructView reflect-ive GUI")
	lab1.SetStretchMaxWidth()
	lab1.SetProp("text-align", "center")
	gi.NewStretch(trow, "str2")

	split := gi.NewSplitView(mfr, "split")
	split.Dim = mat32.X

	tvfr := gi.NewFrame(split, "tvfr", gi.LayoutHoriz)
	svfr := gi.NewFrame(split, "svfr", gi.LayoutHoriz)
	split.SetSplits(.3, .7)

	tv := giv.NewTreeView(tvfr, "tv")
	tv.SetRootNode(&srctree)

	sv := giv.NewStructView(svfr, "sv")
	sv.SetStretchMaxWidth()
	sv.SetStretchMaxHeight()
	sv.SetStruct(&srctree)

	tv.TreeViewSig.Connect(sv.This(), func(recv, send ki.Ki, sig int64, data any) {
		if data == nil {
			return
		}
		// tvr, _ := send.Embed(giv.TypeTreeView).(*gi.TreeView) // root is sender
		tvn, _ := data.(ki.Ki).Embed(giv.TypeTreeView).(*giv.TreeView)
		svr, _ := recv.Embed(giv.TypeStructView).(*giv.StructView)
		if sig == int64(giv.TreeViewSelected) {
			svr.SetStruct(tvn.SrcNode)
		}
	})

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
