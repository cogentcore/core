// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"image"
	"testing"

	"cogentcore.org/core/events"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

// TestListEventsBeforeGrid tests that the mouse events of a list before
// its grid is made, as before it is first rendered, do nothing, instead
// of crashing.
func TestListEventsBeforeGrid(t *testing.T) {
	b := NewBody()
	sl := []string{"a", "b", "c"}
	ls := NewList(b).SetSlice(&sl)
	require.Nil(t, ls.ListGrid, "not made yet")
	for _, typ := range []events.Types{events.MouseMove, events.MouseDrag, events.DoubleClick} {
		assert.NotPanics(t, func() {
			ls.HandleEvent(events.NewMouse(typ, events.Left, image.Pt(5, 5), 0))
		}, typ.String())
	}
}
