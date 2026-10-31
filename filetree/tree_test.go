// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package filetree

import (
	"testing"

	"cogentcore.org/core/core"
)

func TestTree(t *testing.T) {
	b := core.NewBody()
	// testdata/events is a copy of the files of the events package at one
	// point (their contents matter for generated files, which are shown
	// differently), so that the image does not change when they change.
	NewTree(b).OpenPath("testdata/events")
	b.AssertRender(t, "basic")
}
