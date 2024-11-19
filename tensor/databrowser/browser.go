// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package databrowser

//go:generate core generate

import (
	"io/fs"
	"slices"

	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/goal/interpreter"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/tree"
	"golang.org/x/exp/maps"
)

// TheBrowser is the current browser,
// which is valid immediately after NewBrowserWindow
// where it is used to get a local variable for subsequent use.
var TheBrowser *Browser

// Browser holds all the elements of a data browser, for browsing data
// either on an OS filesystem or as a tensorfs virtual data filesystem.
// It supports the automatic loading of [goal] scripts as toolbar actions to
// perform pre-programmed tasks on the data, to create app-like functionality.
// Scripts are ordered alphabetically and any leading #- prefix is automatically
// removed from the label, so you can use numbers to specify a custom order.
// It is not a [core.Widget] itself, and is intended to be incorporated into
// a [core.Frame] widget, potentially along with other custom elements.
// See [Basic] for a basic implementation.
type Browser struct { //types:add -setters
	// FS is the filesystem, if browsing an FS.
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

	// Files is the [DataTree] tree browser of the tensorfs or files.
	Files *DataTree

	// Tabs is the [Tabber] element managing tabs of data views.
	Tabs Tabber

	// Toolbar is the top-level toolbar for the browser, if used.
	Toolbar *core.Toolbar

	// Splits is the overall [core.Splits] for the browser.
	Splits *core.Splits
}

// UpdateFiles Updates the files list.
func (br *Browser) UpdateFiles() { //types:add
	if br.Files == nil {
		return
	}
	files := br.Files
	if br.FS != nil {
		files.SortByModTime = true
		files.OpenPathFS(br.FS, br.DataRoot)
	} else {
		files.OpenPath(br.DataRoot)
	}
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
