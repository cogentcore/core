// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package giv

import (
	"cogentcore.org/core/gi"
)

// TreeTableView combines a [TreeView] and [TableView].
type TreeTableView struct {
	*gi.Frame

	// Tree is the tree view component of the tree table view.
	Tree *TreeView

	// Table is the table view component of the tree table view.
	Table *TableView
}
