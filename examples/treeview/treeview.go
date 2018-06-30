// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"image"

	"github.com/goki/gi"
	"github.com/goki/gi/oswin"
	"github.com/goki/gi/oswin/driver"
	"github.com/goki/gi/units"
	"github.com/goki/ki"
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
	driver.Main(func(app oswin.App) {
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

	width := 1024
	height := 768
	win := gi.NewWindow2D("gogi-treeview-test", "TreeView Test", width, height, true)

	vp := win.WinViewport2D()
	updt := vp.UpdateStart()
	vp.Fill = true

	vlay := vp.AddNewChild(gi.KiT_Frame, "vlay").(*gi.Frame)
	vlay.Lay = gi.LayoutCol

	trow := vlay.AddNewChild(gi.KiT_Layout, "trow").(*gi.Layout)
	trow.Lay = gi.LayoutRow
	trow.SetProp("align-horiz", "center")
	trow.SetProp("margin", 2.0) // raw numbers = px = 96 dpi pixels
	trow.SetStretchMaxWidth()

	spc := vlay.AddNewChild(gi.KiT_Space, "spc1").(*gi.Space)
	spc.SetFixedHeight(units.NewValue(2.0, units.Em))

	trow.AddNewChild(gi.KiT_Stretch, "str1")
	lab1 := trow.AddNewChild(gi.KiT_Label, "lab1").(*gi.Label)
	lab1.Text = "This is a test of the TreeView and StructView reflect-ive GUI"
	lab1.SetProp("max-width", -1)
	lab1.SetProp("text-align", "center")
	trow.AddNewChild(gi.KiT_Stretch, "str2")

	split := vlay.AddNewChild(gi.KiT_SplitView, "split").(*gi.SplitView)
	split.Dim = gi.X

	tvfr := split.AddNewChild(gi.KiT_Frame, "tvfr").(*gi.Frame)
	svfr := split.AddNewChild(gi.KiT_Frame, "svfr").(*gi.Frame)
	split.SetSplits(.3, .7)

	tv := tvfr.AddNewChild(gi.KiT_TreeView, "tv").(*gi.TreeView)
	tv.SetRootNode(&srctree)

	sv := svfr.AddNewChild(gi.KiT_StructView, "sv").(*gi.StructView)
	sv.SetStretchMaxWidth()
	sv.SetStretchMaxHeight()
	sv.SetStruct(&srctree, nil)

	tv.TreeViewSig.Connect(sv.This, func(recv, send ki.Ki, sig int64, data interface{}) {
		if data == nil {
			return
		}
		// tvr, _ := send.EmbeddedStruct(gi.KiT_TreeView).(*gi.TreeView) // root is sender
		tvn, _ := data.(ki.Ki).EmbeddedStruct(gi.KiT_TreeView).(*gi.TreeView)
		svr, _ := recv.EmbeddedStruct(gi.KiT_StructView).(*gi.StructView)
		if sig == int64(gi.TreeViewSelected) {
			svr.SetStruct(tvn.SrcNode.Ptr, nil)
		}
	})

	vp.UpdateEndNoSig(updt)

	win.StartEventLoop()
}
