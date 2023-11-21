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
	"goki.dev/goosi/events"
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

	gi.SetAppName("treeview")
	gi.SetAppAbout(`This is a demo of the treeview in the <b>GoGi</b> graphical interface system, within the <b>GoKi</b> tree framework.  See <a href="https://github.com/goki">GoKi on GitHub</a>
<p>Full Drag-and-Drop, Copy / Cut / Paste, and Keyboard Navigation is supported.</p>`)

	b := gi.NewBody().SetTitle("TreeView Test")

	// gi.DefaultTopAppBar = nil

	split := gi.NewSplits(b, "split")
	split.Dim = mat32.X

	tvfr := gi.NewFrame(split, "tvfr")
	svfr := gi.NewFrame(split, "svfr")
	split.SetSplits(.3, .7)

	svfr.Style(func(s *styles.Style) {
		s.Direction = styles.Column
		// s.Border.Color.Set(colors.Black)
		// s.Border.Width.Set(units.Dp(2))
		s.Grow.Set(1, 0)
	})

	tvfr.Style(func(s *styles.Style) {
		s.Direction = styles.Column
		// s.Border.Color.Set(colors.Black)
		// s.Border.Width.Set(units.Dp(2))
		s.Overflow.Y = styles.OverflowAuto
	})

	tv := giv.NewTreeView(tvfr, "tv")
	tv.RootView = tv

	// depth := 3 // 1 = small tree for testing
	depth := 10 // big tree
	MakeTree(tv, 0, depth, 5)

	nleaves := tv.RootSetViewIdx()
	fmt.Println("N leaves:", nleaves)

	sv := giv.NewStructView(svfr, "sv")
	sv.Style(func(s *styles.Style) {
		s.Grow.Set(1, 1)
	})
	sv.SetStruct(tv)

	tv.OnSelect(func(e events.Event) {
		if len(tv.SelectedNodes) > 0 {
			sv.SetStruct(tv.SelectedNodes[0])
		}
	})

	b.NewWindow().Run().Wait()
}
