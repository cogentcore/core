// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package views

import (
	"testing"

	"cogentcore.org/core/core"
)

func TestSliceView(t *testing.T) {
	b := core.NewBody()
	NewSliceView(b).SetSlice(&[]int{1, 3, 5})
	b.AssertRender(t, "slice-view/basic")
}

func TestSliceViewScroll(t *testing.T) {
	b := core.NewBody()
	NewSliceView(b).SetSlice(&[]int{1, 3, 5, 8, 10, 20, 50})
	b.AssertRender(t, "slice-view/scroll")
}

func TestSliceViewStrings(t *testing.T) {
	b := core.NewBody()
	NewSliceView(b).SetSlice(&[]string{"a", "b", "c", "d", "e", "f", "g"})
	b.AssertRender(t, "slice-view/strings")
}

func TestSliceViewReadOnly(t *testing.T) {
	b := core.NewBody()
	NewSliceView(b).SetSlice(&[]int{1, 3, 5}).SetReadOnly(true)
	b.AssertRender(t, "slice-view/read-only")
}
