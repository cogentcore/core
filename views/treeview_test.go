// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package views

import (
	"testing"

	"cogentcore.org/core/core"
)

func TestTree(t *testing.T) {
	b := core.NewBody()

	fr := core.NewFrame()
	core.NewButton(fr)
	core.NewText(fr)
	core.NewButton(core.NewFrame(fr))

	NewTree(b).SyncTree(fr)
	b.AssertRender(t, "tree/basic")
}

func TestTreeReadOnly(t *testing.T) {
	b := core.NewBody()

	fr := core.NewFrame()
	core.NewButton(fr)
	core.NewText(fr)
	core.NewButton(core.NewFrame(fr))

	NewTree(b).SyncTree(fr).SetReadOnly(true)
	b.AssertRender(t, "tree/read-only")
}
