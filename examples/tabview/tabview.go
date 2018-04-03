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

	tv1 := vlay.AddNewChildNamed(gi.KiT_TabWidget, "tv1").(*gi.TabWidget)
	tv1.SetSrcNode(&srctree)

	for i, sk := range srctree.Kids {
		tf := tv1.TabFrameAtIndex(i)
		lbl := tf.AddNewChildNamed(gi.KiT_Label, "tst").(*gi.Label)
		lbl.Text = sk.UniqueName()
		// note: these were set by default -- could override
		// tf.SetProp("max-width", -1.0) // stretch flex
		// tf.SetProp("max-height", -1.0)
	}

	win.UpdateEnd()

	win.StartEventLoop()
}
