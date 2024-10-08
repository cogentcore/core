// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"testing"
)

func TestInlineList(t *testing.T) {
	b := NewBody()
	NewInlineList(b).SetSlice(&[]int{1, 3, 5})
	b.AssertRender(t, "inline-list/basic")
}

func TestInlineListUpdate(t *testing.T) {
	b := NewBody()
	il := NewInlineList(b).SetSlice(&[]int{1, 2, 3})
	il.Update()
	il.SetSlice(&[]int{4, 5, 6})
	il.Update()
	b.AssertRender(t, "inline-list/update")
}
