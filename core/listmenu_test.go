// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"image"
	"testing"

	"cogentcore.org/core/events"
	"cogentcore.org/core/keymap"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// listMenu returns the context menu of the given list.
func listMenu(lb *ListBase) *Scene {
	m := NewScene()
	lb.contextMenu(m, image.Point{})
	return m
}

// TestListContextMenu tests the actions of the context menu of an
// editable list and of a read-only one.
func TestListContextMenu(t *testing.T) {
	b := NewBody()
	sl := []string{"a", "b", "c"}
	ls := NewList(b).SetSlice(&sl)
	AssertRender(t, b, "list/context-menu", Locked(b, func() {
		click := func(text string) {
			t.Helper()
			bt := ButtonByText(listMenu(ls), text)
			require.NotNil(t, bt, text)
			bt.Send(events.Click)
		}
		ls.SelectIndexEvent(1, events.SelectOne)
		click("Add row")
		assert.Equal(t, []string{"a", "b", "", "c"}, sl)
		assert.Equal(t, 2, ls.SelectedIndex, "the new row")
		click("Delete row")
		assert.Equal(t, []string{"a", "b", "c"}, sl)

		ls.SelectIndexEvent(0, events.SelectOne)
		click("Duplicate")
		assert.Equal(t, []string{"a", "a", "b", "c"}, sl)
		ls.SelectIndexEvent(1, events.SelectOne)
		click("Cut")
		assert.Equal(t, []string{"a", "b", "c"}, sl)
		ls.SelectIndexEvent(2, events.SelectOne)
		click("Copy")
		click("Paste") // shows the paste menu
		click("Toggle indexes")
		assert.True(t, ls.ShowIndexes)

		ls.SetReadOnly(true)
		m := listMenu(ls)
		assert.Nil(t, ButtonByText(m, "Add row"), "read-only")
		ButtonByText(m, "Copy").Send(events.Click)
		ButtonByText(m, "Toggle indexes").Send(events.Click)
		assert.False(t, ls.ShowIndexes)
	}))
}

// TestListBindSelect tests a read-only selection list: the bound index,
// navigating with keys, the initially selected value, and toggling the
// selection of a row.
func TestListBindSelect(t *testing.T) {
	b := NewBody()
	sl := []string{"a", "b", "c", "d", "e", "f", "g", "h"}
	ls := NewList(b).SetSlice(&sl)
	ls.SetSelectedValue("c")
	sel := -1
	changes := 0
	ls.BindSelect(&sel)
	ls.OnChange(func(e events.Event) { changes++ })
	assert.True(t, ls.IsReadOnly())
	AssertRender(t, b, "list/bind-select", Locked(b, func() {
		assert.Equal(t, 2, ls.SelectedIndex, "the selected value")
		assert.Nil(t, ls.SelectedValue)

		SendKeyFunction(ls, keymap.MoveDown)
		assert.Equal(t, 3, sel)
		SendKeyFunction(ls, keymap.MoveUp)
		assert.Equal(t, 2, sel)
		SendKeyFunction(ls, keymap.PageDown)
		assert.Greater(t, sel, 2)
		pg := sel
		SendKeyFunction(ls, keymap.PageUp)
		assert.Less(t, sel, pg)
		ls.SelectIndexEvent(0, events.SelectOne)
		SendKeyFunction(ls, keymap.MoveUp)
		assert.Equal(t, 0, sel, "at the start")
		assert.Positive(t, changes)
		// accepting sends a double click, which is ignored without a
		// position on a row
		SendKeyFunction(ls, keymap.Accept)

		ls.ReadOnlyMultiSelect = true
		SendKeyFunction(ls, keymap.MoveDown) // with the editable key navigation
		assert.Equal(t, 1, sel)
		ls.SelectIndexEvent(0, events.SelectOne)
		ls.SelectIndexEvent(1, events.ExtendOne)
		assert.Equal(t, []int{0, 1}, ls.SelectedIndexesList(false))
		ls.SelectIndexEvent(1, events.ExtendOne)
		assert.Equal(t, []int{0}, ls.SelectedIndexesList(false), "toggled off")
		ls.SelectIndexEvent(0, events.Unselect)
		assert.Empty(t, ls.SelectedIndexesList(false))
		assert.Nil(t, ls.TextRunes())
		assert.Equal(t, &ls.Slice, ls.WidgetValue())
	}))
	idx, ok := sliceIndexByValue(&sl, "z")
	assert.False(t, ok)
	assert.Equal(t, -1, idx)
}
