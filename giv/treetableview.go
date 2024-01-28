// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package giv

import (
	"image/color"

	"cogentcore.org/core/gi"
)

// TreeTableView combines a [TreeView] and [TableView].
type TreeTableView struct {
	gi.Frame

	// Tree is the tree view component of the tree table view.
	Tree *TreeView `set:"-"`

	// Table is the table view component of the tree table view.
	Table *TableView `set:"-"`
}

func (tt *TreeTableView) ConfigWidget() {
	if tt.HasChildren() {
		return
	}

	updt := tt.UpdateStart()

	sp := gi.NewSplits(tt)
	tt.Tree = NewTreeView(sp)
	tt.Table = NewTableView(sp).SetSlice(&[]color.RGBA{})

	tt.UpdateEndLayout(updt)
}
