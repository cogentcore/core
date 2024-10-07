// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package databrowser

//go:generate core generate

import (
	"io/fs"
	"os"
	"path/filepath"
	"slices"

	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/base/fsx"
	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/filetree"
	"cogentcore.org/core/goal/interpreter"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/tree"
	"cogentcore.org/core/types"
	"golang.org/x/exp/maps"
)

// TheBrowser is the current browser,
// which is valid immediately after NewBrowserWindow
// where it is used to get a local variable for subsequent use.
var TheBrowser *Browser

// Browser is a data browser, for browsing data either on an OS filesystem
// or as a datafs virtual data filesystem.
// It supports the automatic loading of [goal] scripts as toolbar actions to
// perform pre-programmed tasks on the data, to create app-like functionality.
// Scripts are ordered alphabetically and any leading #- prefix is automatically
// removed from the label, so you can use numbers to specify a custom order.
type Browser struct {
	core.Frame

	// FS is the filesystem, if browsing an FS
	FS fs.FS

	// DataRoot is the path to the root of the data to browse.
	DataRoot string

	// StartDir is the starting directory, where the app was originally started.
	StartDir string

	// ScriptsDir is the directory containing scripts for toolbar actions.
	// It defaults to DataRoot/dbscripts
	ScriptsDir string

	// Scripts
	Scripts map[string]string `set:"-"`

	// Interpreter is the interpreter to use for running Browser scripts
	Interpreter *interpreter.Interpreter `set:"-"`

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
	br.InitInterp()

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
	startDir, _ := os.Getwd()
	startDir = errors.Log1(filepath.Abs(startDir))
	b := core.NewBody("Cogent Data Browser: " + fsx.DirAndFile(startDir))
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
	br.SetScriptsDir(filepath.Join(ddr, "dbscripts"))
	TheBrowser = br
	br.Interpreter.Eval("br := databrowser.TheBrowser") // grab it
	br.UpdateScripts()
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

func (br *Browser) GetDataRoot() string {
	return br.DataRoot
}

func (br *Browser) MakeToolbar(p *tree.Plan) {
	tree.Add(p, func(w *core.FuncButton) {
		w.SetFunc(br.UpdateFiles).SetText("").SetIcon(icons.Refresh).SetShortcut("Command+U")
	})
	tree.Add(p, func(w *core.FuncButton) {
		w.SetFunc(br.UpdateScripts).SetText("").SetIcon(icons.Code)
	})
	scr := maps.Keys(br.Scripts)
	slices.Sort(scr)
	for _, s := range scr {
		lbl := TrimOrderPrefix(s)
		tree.AddAt(p, lbl, func(w *core.Button) {
			w.SetText(lbl).SetIcon(icons.RunCircle).
				OnClick(func(e events.Event) {
					br.RunScript(s)
				})
			sc := br.Scripts[s]
			tt := FirstComment(sc)
			if tt == "" {
				tt = "Run Script (add a comment to top of script to provide more useful info here)"
			}
			w.SetTooltip(tt)
		})
	}
}
