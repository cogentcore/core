// Copyright (c) 2018, Randall C. O'Reilly. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"github.com/rcoreilly/goki/gi"
	"github.com/rcoreilly/goki/gi/oswin"
	_ "github.com/rcoreilly/goki/gi/oswin/init"
	"github.com/rcoreilly/goki/ki"
)

// todo: enum field, etc

// A node for testing
type TestNode struct {
	ki.Node
	StrField   string  `desc:"a string"`
	IntField   int     `desc:"an int"`
	FloatField float64 `desc:"a float"`
	BoolField  bool    `desc:"a bool"`
}

func main() {
	go mainrun()
	oswin.RunBackendEventLoop() // this needs to run in main loop
}

func mainrun() {
	// a source tree to view
	srctree := TestNode{}
	srctree.InitName(&srctree, "par1")
	// child1 :=
	srctree.AddNewChild(nil, "child1")
	child2 := srctree.AddNewChild(nil, "child2")
	// child3 :=
	srctree.AddNewChild(nil, "child3")
	// schild2 :=
	child2.AddNewChild(nil, "subchild1")

	// turn this on to see a trace of the rendering
	// gi.Render2DTrace = true
	// gi.Layout2DTrace = true

	width := 800
	height := 800
	win := gi.NewWindow2D("TreeView Window", width, height)
	win.UpdateStart()

	vp := win.WinViewport2D()
	vp.SetProp("background-color", "#FFF")
	vp.Fill = true

	vlay := vp.AddNewChild(gi.KiT_Frame, "vlay").(*gi.Frame)
	vlay.Lay = gi.LayoutCol

	row1 := vlay.AddNewChild(gi.KiT_Layout, "row1").(*gi.Layout)
	row1.Lay = gi.LayoutRow
	row1.SetProp("margin", 2.0) // raw numbers = px = 96 dpi pixels
	// row1.SetStretchMaxWidth()

	tv1 := row1.AddNewChild(gi.KiT_TreeView, "tv1").(*gi.TreeView)
	tv1.SetSrcNode(&srctree)

	sv1 := row1.AddNewChild(gi.KiT_StructView, "sv1").(*gi.StructView)
	sv1.SetStruct(&srctree)
	sv1.SetProp("horiz-align", gi.AlignLeft)

	tv1.TreeViewSig.Connect(sv1.This, func(recv, send ki.Ki, sig int64, data interface{}) {
		if data == nil {
			return
		}
		// tvr, _ := send.EmbeddedStruct(gi.KiT_TreeView).(*gi.TreeView) // root is sender
		tvn, _ := data.(ki.Ki).EmbeddedStruct(gi.KiT_TreeView).(*gi.TreeView)
		svr, _ := recv.EmbeddedStruct(gi.KiT_StructView).(*gi.StructView)
		if sig == int64(gi.NodeSelected) {
			svr.SetStruct(tvn.SrcNode.Ptr)
		}
	})

	win.UpdateEnd()

	win.StartEventLoop()
}
