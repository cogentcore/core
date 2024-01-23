// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"math/rand"

	"cogentcore.org/core/events"
	"cogentcore.org/core/gi"
	"cogentcore.org/core/giv"
	"cogentcore.org/core/styles"
)

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
		MakeTree(kt, iter, maxIter, maxKids)
	}
}

func main() {
	b := gi.NewAppBody("Cogent Core Tree View Demo")
	b.App().About = `This is a demo of the treeview in the <b>Cogent Core</b> graphical interface system, within the <b>Goki</b> tree framework.  See <a href="https://github.com/goki">Cogent Core on GitHub</a>
<p>Full Drag-and-Drop, Copy / Cut / Paste, and Keyboard Navigation is supported.</p>`

	splits := gi.NewSplits(b)

	tvfr := gi.NewFrame(splits)
	svfr := gi.NewFrame(splits)
	splits.SetSplits(.3, .7)

	svfr.Style(func(s *styles.Style) {
		s.Direction = styles.Column
		// s.Border.Color.Set(colors.Black)
		// s.Border.Width.Set(units.Dp(2))
		s.Grow.Set(1, 1)
	})

	tvfr.Style(func(s *styles.Style) {
		s.Direction = styles.Column
		// s.Border.Color.Set(colors.Black)
		// s.Border.Width.Set(units.Dp(2))
		s.Overflow.Y = styles.OverflowAuto
	})

	tv := giv.NewTreeView(tvfr)

	depth := 3 // 1 = small tree for testing
	// depth := 10 // big tree
	MakeTree(tv, 0, depth, 5)

	nleaves := tv.RootSetViewIdx()
	fmt.Println("N leaves:", nleaves)

	sv := giv.NewStructView(svfr)
	sv.Style(func(s *styles.Style) {
		s.Grow.Set(1, 1)
	})
	sv.SetStruct(tv)

	tv.OnSelect(func(e events.Event) {
		if len(tv.SelectedNodes) > 0 {
			sv.SetStruct(tv.SelectedNodes[0])
		}
	})

	b.StartMainWindow()
}
