// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package giv

import (
	"testing"

	"cogentcore.org/core/gi"
	"cogentcore.org/core/tree"
)

func TestTreeView(t *testing.T) {
	b := gi.NewBody()

	fr := tree.NewRoot[*gi.Frame]("frame")
	gi.NewButton(fr)
	gi.NewLabel(fr)
	gi.NewButton(gi.NewLayout(fr))

	NewTreeView(b).SyncTree(fr)
	b.AssertRender(t, "treeview/basic")
}
