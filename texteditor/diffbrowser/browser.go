// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package diffbrowser

//go:generate core generate

import (
	"cogentcore.org/core/base/dirs"
	"cogentcore.org/core/base/stringsx"
	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/texteditor"
)

// Browser is a diff browser, for browsing a set of paired files
// for viewing differences between them, organized into a tree
// structure, e.g., reflecting their source in a filesystem.
type Browser struct {
	core.Frame

	// starting paths for the files being compared
	PathA, PathB string
}

func (br *Browser) OnInit() {
	br.Frame.OnInit()
	br.Style(func(s *styles.Style) {
		s.Grow.Set(1, 1)
	})
	br.OnShow(func(e events.Event) {
		br.OpenFiles()
	})

	core.AddChildAt(br, "splits", func(w *core.Splits) {
		w.SetSplits(.15, .85)
		core.AddChildAt(w, "treeframe", func(w *core.Frame) {
			w.Style(func(s *styles.Style) {
				s.Direction = styles.Column
				s.Overflow.Set(styles.OverflowAuto)
				s.Grow.Set(1, 1)
			})
			core.AddChildAt(w, "tree", func(w *Node) {})
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
	br.BuildTree() // must have tree
	// b.AddAppBar(br.MakeToolbar)
	return br, b
}

func (br *Browser) Splits() *core.Splits {
	return br.FindPath("splits").(*core.Splits)
}

func (br *Browser) Tree() *Node {
	sp := br.Splits()
	return sp.Child(0).Child(0).(*Node)
}

func (br *Browser) Tabs() *core.Tabs {
	return br.FindPath("splits/tabs").(*core.Tabs)
}

// OpenFiles Updates the tree based on files
func (br *Browser) OpenFiles() { //types:add
	tv := br.Tree()
	if tv == nil {
		return
	}
	tv.Open()
}

func (br *Browser) MakeToolbar(p *core.Plan) {
	// core.Add(p, func(w *views.FuncButton) {
	// 	w.SetFunc(br.OpenFiles).SetText("").SetIcon(icons.Refresh).SetShortcut("Command+U")
	// })
}

// ViewDiff views diff for given file Node, returning a texteditor.DiffView
func (br *Browser) ViewDiff(fn *Node) *texteditor.DiffView {
	df := dirs.DirAndFile(fn.FileA)
	tabs := br.Tabs()
	tab := tabs.RecycleTab(df, true)
	if tab.HasChildren() {
		dv := tab.Child(0).(*texteditor.DiffView)
		return dv
	}
	dv := texteditor.NewDiffView(tab)
	dv.SetFileA(fn.FileA).SetFileB(fn.FileB).SetRevA(fn.RevA).SetRevB(fn.RevB)
	dv.DiffStrings(stringsx.SplitLines(fn.TextA), stringsx.SplitLines(fn.TextB))
	br.Update()
	return dv
}
