// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"testing"
)

func TestList(t *testing.T) {
	b := NewBody()
	NewList(b).SetSlice(&[]int{1, 3, 5})
	AssertRender(t, b, "list/basic")
}

func TestListScroll(t *testing.T) {
	b := NewBody()
	NewList(b).SetSlice(&[]int{1, 3, 5, 8, 10, 20, 50})
	AssertRender(t, b, "list/scroll")
}

func TestListStrings(t *testing.T) {
	b := NewBody()
	NewList(b).SetSlice(&[]string{"a", "b", "c", "d", "e", "f", "g"})
	AssertRender(t, b, "list/strings")
}

func TestListReadOnly(t *testing.T) {
	b := NewBody()
	NewList(b).SetSlice(&[]int{1, 3, 5}).SetReadOnly(true)
	AssertRender(t, b, "list/read-only")
}
