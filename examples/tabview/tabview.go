// Copyright (c) 2018, Randall C. O'Reilly. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	// "fmt"
	"github.com/rcoreilly/goki/gi"
	_ "github.com/rcoreilly/goki/gi/init"
	"github.com/rcoreilly/goki/ki"
	// "math/rand"
	"reflect"
	// "runtime"
	// "sync"
	// "time"
	// "image"
	// "image/draw"
)

func main() {
	go mainrun()
	gi.RunBackendEventLoop() // this needs to run in main loop
}

func mainrun() {
	// a source tree to view
	srctree := ki.Node{}
	srctree.SetThisName(&srctree, "par1")
	srctree.SetChildType(reflect.TypeOf(srctree))
	// child1 :=
	srctree.AddNewChildNamed(nil, "child1")
	child2 := srctree.AddNewChildNamed(nil, "child2")
	// child3 :=
	srctree.AddNewChildNamed(nil, "child3")

	child2.SetChildType(reflect.TypeOf(srctree))
	// schild2 :=
	child2.AddNewChildNamed(nil, "subchild1")

	width := 800
	height := 800
	win := gi.NewWindow2D("test window", width, height)
	win.UpdateStart()

	vp := win.WinViewport2D()

	vpfill := vp.AddNewChildNamed(gi.KiT_Viewport2DFill, "vpfill").(*gi.Viewport2DFill)
	vpfill.SetProp("fill", "#FFF")

	vlay := vpfill.AddNewChildNamed(gi.KiT_Layout, "vlay").(*gi.Layout)
	vlay.Lay = gi.LayoutCol

	tv1 := vpfill.AddNewChildNamed(gi.KiT_TabWidget, "tv1").(*gi.TabWidget)
	tv1.SetSrcNode(&srctree)

	win.UpdateEnd()

	win.StartEventLoop()
}
