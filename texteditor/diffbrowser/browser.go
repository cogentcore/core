// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package diffbrowser

//go:generate core generate

import (
	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/views"
)

// Browser is a diff browser, for browsing a set of paired files
// for viewing differences between them, organized into a tree
// structure, e.g., reflecting their source in a filesystem.
type Browser struct {
	core.Frame

	// starting paths for the files being compared
	PathA, PathB string

	// Files is the source tree of files
	Files *Node
}

// OnInit initializes the browser
func (br *Browser) OnInit() {
	br.Frame.OnInit()
	br.Style(func(s *styles.Style) {
		s.Grow.Set(1, 1)
	})
	br.OnShow(func(e events.Event) {
		br.UpdateFiles()
	})

	core.AddChildAt(br, "splits", func(w *core.Splits) {
		w.SetSplits(.15, .85)
		core.AddChildAt(w, "treeframe", func(w *core.Frame) {
			w.Style(func(s *styles.Style) {
				s.Direction = styles.Column
				s.Overflow.Set(styles.OverflowAuto)
				s.Grow.Set(1, 1)
			})
			core.AddChildAt(w, "tree", func(w *views.TreeView) {
				w.OpenDepth = 4
				// w.OnSelect(func(e events.Event) {
				// 	e.SetHandled()
				// 	sels := w.SelectedViews()
				// 	if sels != nil {
				// 		br.FileNodeSelected(sn)
				// 	}
				// })
			})
		})
		core.AddChildAt(w, "tabs", func(w *core.Tabs) {
			w.Type = core.FunctionalTabs
		})
	})
}

// NewBrowserWindow opens a new diff Browser in a new window
func NewBrowserWindow() (*Browser, *core.Body) {
	b := core.NewBody("Diff Browser")
	br := NewBrowser(b)
	b.AddAppBar(br.MakeToolbar)
	return br, b
}

func (br *Browser) Splits() *core.Splits {
	return br.FindPath("splits").(*core.Splits)
}

func (br *Browser) Tree() *views.TreeView {
	sp := br.Splits()
	return sp.Child(0).Child(0).(*views.TreeView)
}

func (br *Browser) Tabs() *core.Tabs {
	return br.FindPath("splits/tabs").(*core.Tabs)
}

// UpdateFiles Updates the tree based on files
func (br *Browser) UpdateFiles() { //types:add
	if br.Files == nil {
		return
	}
	tr := br.Tree()
	if tr == nil {
		return
	}
	tr.SyncTree(br.Files)
	br.Update()
	tr.Open()
}

func (br *Browser) MakeToolbar(p *core.Plan) {
	core.Add(p, func(w *views.FuncButton) {
		w.SetFunc(br.UpdateFiles).SetText("").SetIcon(icons.Refresh).SetShortcut("Command+U")
	})
}
