// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"github.com/goki/gi"
	"github.com/goki/gi/oswin"
	"github.com/goki/gi/oswin/driver"
	_ "github.com/goki/gi/svg"
	"github.com/goki/ki"
)

func main() {
	driver.Main(func(app oswin.App) {
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
	vp.Fill = true

	vlay := vp.AddNewChild(gi.KiT_Frame, "vlay").(*gi.Frame)
	vlay.Lay = gi.LayoutCol

	tv1 := vlay.AddNewChild(gi.KiT_TabView, "tv1").(*gi.TabView)
	tv1.SetSrcNode(&srctree)

	for i, sk := range srctree.Kids {
		tf := tv1.TabFrameAtIndex(i)
		lbl := tf.AddNewChild(gi.KiT_Label, "tst").(*gi.Label)
		lbl.Text = sk.UniqueName()
		// note: these were set by default -- could override
		// tf.SetProp("max-width", -1.0) // stretch flex
		// tf.SetProp("max-height", -1.0)
	}

	vp.UpdateEndNoSig(updt)

	win.StartEventLoop()
}
