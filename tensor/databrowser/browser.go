// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package databrowser

//go:generate core generate

import (
	"io/fs"
	"path/filepath"

	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/base/fsx"
	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/filetree"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/tree"
	"cogentcore.org/core/types"
)

// Browser is a data browser, for browsing data either on an os filesystem
// or as a datafs virtual data filesystem.
type Browser struct {
	core.Frame

	// FS is the filesystem, if browsing an FS
	FS fs.FS

	// DataRoot is the path to the root of the data to browse
	DataRoot string

	toolbar *core.Toolbar
	splits  *core.Splits
	files   *filetree.Tree
	tabs    *core.Tabs
}

// Init initializes with the data and script directories
func (br *Browser) Init() {
	br.Frame.Init()
	br.Styler(func(s *styles.Style) {
		s.Grow.Set(1, 1)
	})

	br.OnShow(func(e events.Event) {
		br.UpdateFiles()
	})

	tree.AddChildAt(br, "splits", func(w *core.Splits) {
		br.splits = w
		w.SetSplits(.15, .85)
		tree.AddChildAt(w, "fileframe", func(w *core.Frame) {
			w.Styler(func(s *styles.Style) {
				s.Direction = styles.Column
				s.Overflow.Set(styles.OverflowAuto)
				s.Grow.Set(1, 1)
			})
			tree.AddChildAt(w, "filetree", func(w *filetree.Tree) {
				br.files = w
				w.FileNodeType = types.For[FileNode]()
				// w.OnSelect(func(e events.Event) {
				// 	e.SetHandled()
				// 	sels := w.SelectedViews()
				// 	if sels != nil {
				// 		br.FileNodeSelected(sn)
				// 	}
				// })
			})
		})
		tree.AddChildAt(w, "tabs", func(w *core.Tabs) {
			br.tabs = w
			w.Type = core.FunctionalTabs
		})
	})
}

// NewBrowserWindow opens a new data Browser for given
// file system (nil for os files) and data directory.
func NewBrowserWindow(fsys fs.FS, dataDir string) *Browser {
	b := core.NewBody("Cogent Data Browser: " + fsx.DirAndFile(dataDir))
	br := NewBrowser(b)
	br.FS = fsys
	ddr := dataDir
	if fsys == nil {
		ddr = errors.Log1(filepath.Abs(dataDir))
	}
	b.AddTopBar(func(bar *core.Frame) {
		tb := core.NewToolbar(bar)
		br.toolbar = tb
		tb.Maker(br.MakeToolbar)
	})
	br.SetDataRoot(ddr)
	b.RunWindow()
	return br
}

// ParentBrowser returns the Browser parent of given node
func ParentBrowser(tn tree.Node) *Browser {
	var res *Browser
	tn.AsTree().WalkUp(func(n tree.Node) bool {
		if c, ok := n.(*Browser); ok {
			res = c
			return false
		}
		return true
	})
	return res
}

// UpdateFiles Updates the files list.
func (br *Browser) UpdateFiles() { //types:add
	files := br.files
	if br.FS != nil {
		files.SortByModTime = true
		files.OpenPathFS(br.FS, br.DataRoot)
	} else {
		files.OpenPath(br.DataRoot)
	}
	br.Update()
}

func (br *Browser) MakeToolbar(p *tree.Plan) {
	tree.Add(p, func(w *core.FuncButton) {
		w.SetFunc(br.UpdateFiles).SetText("").SetIcon(icons.Refresh).SetShortcut("Command+U")
	})
}
