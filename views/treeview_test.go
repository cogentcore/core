// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package views

import (
	"testing"

	"cogentcore.org/core/core"
	"cogentcore.org/core/tree"
)

func TestTreeView(t *testing.T) {
	b := core.NewBody()

	fr := tree.NewRoot[*core.Frame]("frame")
	core.NewButton(fr)
	core.NewLabel(fr)
	core.NewButton(core.NewLayout(fr))

	NewTreeView(b).SyncTree(fr)
	b.AssertRender(t, "treeview/basic")
}
