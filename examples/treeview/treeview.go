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

func main() {
	go mainrun()
	oswin.RunBackendEventLoop() // this needs to run in main loop
}

func mainrun() {
	// a source tree to view
	srctree := ki.Node{}
	srctree.InitName(&srctree, "par1")
	// child1 :=
	srctree.AddNewChildNamed(nil, "child1")
	child2 := srctree.AddNewChildNamed(nil, "child2")
	// child3 :=
	srctree.AddNewChildNamed(nil, "child3")
	// schild2 :=
	child2.AddNewChildNamed(nil, "subchild1")

	width := 800
	height := 800
	win := gi.NewWindow2D("test window", width, height)
	win.UpdateStart()

	vp := win.WinViewport2D()
	vp.SetProp("background-color", "#FFF")
	vp.Fill = true

	vlay := vp.AddNewChildNamed(gi.KiT_Frame, "vlay").(*gi.Frame)
	vlay.Lay = gi.LayoutCol

	row1 := vlay.AddNewChildNamed(gi.KiT_Layout, "row1").(*gi.Layout)
	row1.Lay = gi.LayoutRow
	row1.SetProp("margin", 2.0) // raw numbers = px = 96 dpi pixels
	// row1.SetStretchMaxWidth()

	tv1 := row1.AddNewChildNamed(gi.KiT_TreeView, "tv1").(*gi.TreeView)
	tv1.SetSrcNode(&srctree)

	win.UpdateEnd()

	win.StartEventLoop()
}
