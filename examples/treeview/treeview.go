// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"math/rand"

	"goki.dev/gi/v2/gi"
	"goki.dev/gi/v2/gimain"
	"goki.dev/gi/v2/giv"
	"goki.dev/girl/styles"
	"goki.dev/mat32/v2"
)

func main() { gimain.Run(app) }

func MakeTree(tv *giv.TreeView, iter, maxIter, maxKids int) {
	if iter > maxIter {
		return
	}
	n := rand.Intn(maxKids)
	if iter == 0 {
		n = maxKids
	}
	iter++
	parnm := tv.Name() + "_"
	tv.SetNChildren(n, giv.TreeViewType, parnm+"ch")
	for j := 0; j < n; j++ {
		kt := tv.Child(j).(*giv.TreeView)
		kt.RootView = tv.RootView
		MakeTree(kt, iter, maxIter, maxKids)
	}
}

func app() {
	// turn this on to see a trace of the rendering
	// gi.RenderTrace = true
	// gi.LayoutTrace = true

	// goosi.ZoomFactor = 2

	gi.SetAppName("treeview")
	gi.SetAppAbout(`This is a demo of the treeview in the <b>GoGi</b> graphical interface system, within the <b>GoKi</b> tree framework.  See <a href="https://github.com/goki">GoKi on GitHub</a>
<p>Full Drag-and-Drop, Copy / Cut / Paste, and Keyboard Navigation is supported.</p>`)

	sc := gi.NewScene("treeview-test").SetTitle("TreeView Test")

	split := gi.NewSplits(sc, "split")
	split.Dim = mat32.X

	tvfr := gi.NewFrame(split, "tvfr").SetLayout(gi.LayoutHoriz)
	svfr := gi.NewFrame(split, "svfr").SetLayout(gi.LayoutHoriz)
	split.SetSplits(.3, .7)

	tvfr.Style(func(s *styles.Style) {
		s.SetStretchMax()
	})

	tv := giv.NewTreeView(tvfr, "tv")
	tv.RootView = tv

	depth := 2 // 1 = small tree for testing
	// depth := 10 // big tree
	MakeTree(tv, 0, depth, 5)

	nleaves := tv.SetViewIdx()
	fmt.Println("N leaves:", nleaves)

	_ = svfr
	// sv := giv.NewStructView(svfr, "sv")
	// sv.SetStretchMaxWidth()
	// sv.SetStretchMaxHeight()
	// sv.SetStruct(tv)

	// tv.TreeViewSig.Connect(sv.This(), func(recv, send ki.Ki, sig int64, data any) {
	// 	if data == nil {
	// 		return
	// 	}
	// 	//t vr, _ := send.Embed(giv.TypeTreeView).(*gi.TreeView) // root is sender
	// 	tvn, _ := data.(ki.Ki).Embed(giv.TypeTreeView).(*giv.TreeView)
	// 	svr, _ := recv.Embed(giv.TypeStructView).(*giv.StructView)
	// 	if sig == int64(giv.TreeViewSelected) {
	// 		svr.SetStruct(tvn.SrcNode)
	// 	}
	// })

	gi.NewWindow(sc).Run().Wait()
}
