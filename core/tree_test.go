// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"testing"
)

func TestTree(t *testing.T) {
	b := NewBody()

	fr := NewFrame()
	NewButton(fr)
	NewText(fr)
	NewButton(NewFrame(fr))

	NewTree(b).SyncTree(fr)
	b.AssertRender(t, "tree/basic")
}

func TestTreeReadOnly(t *testing.T) {
	b := NewBody()

	fr := NewFrame()
	NewButton(fr)
	NewText(fr)
	NewButton(NewFrame(fr))

	NewTree(b).SyncTree(fr).SetReadOnly(true)
	b.AssertRender(t, "tree/read-only")
}
