// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"testing"

	"cogentcore.org/core/base/reflectx"
	"cogentcore.org/core/events"
	"cogentcore.org/core/events/key"
	"cogentcore.org/core/keymap"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestListKeyNavigation tests moving and extending the selection of a
// list with the keyboard.
func TestListKeyNavigation(t *testing.T) {
	b := NewBody()
	sl := []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19}
	ls := NewList(b).SetSlice(&sl)
	sel := func() []int { return ls.SelectedIndexesList(reflectx.Ascending) }
	AssertRender(t, b, "list/key-navigation", Locked(b, func() {
		ls.SelectIndexEvent(0, events.SelectOne)
		SendKeyFunction(ls, keymap.MoveDown)
		SendKeyFunction(ls, keymap.MoveDown)
		assert.Equal(t, 2, ls.SelectedIndex)
		SendKeyFunction(ls, keymap.MoveUp)
		assert.Equal(t, []int{1}, sel())

		SendKeyFunction(ls, keymap.PageDown)
		assert.Greater(t, ls.SelectedIndex, 2, "page down moves by the visible rows")
		pg := ls.SelectedIndex
		SendKeyFunction(ls, keymap.PageUp)
		assert.Less(t, ls.SelectedIndex, pg)

		ls.SelectIndexEvent(3, events.SelectOne)
		SendKey(ls, 0, key.CodeDownArrow, key.Shift)
		SendKey(ls, 0, key.CodeDownArrow, key.Shift)
		assert.Equal(t, []int{3, 4, 5}, sel(), "shift extends the selection")

		SendKeyFunction(ls, keymap.CancelSelect)
		assert.Empty(t, sel())
		SendKeyFunction(ls, keymap.SelectAll)
		assert.Len(t, sel(), len(sl))

		ls.SelectIndexEvent(0, events.SelectOne)
		SendKeyFunction(ls, keymap.SelectMode)
		assert.True(t, ls.SelectMode)
		SendKeyFunction(ls, keymap.MoveDown)
		assert.Equal(t, []int{0, 1}, sel(), "select mode extends the selection")
	}))
}

// TestListInsertDelete tests inserting and deleting elements of a list,
// with the selection following the change.
func TestListInsertDelete(t *testing.T) {
	b := NewBody()
	sl := []string{"a", "b", "c"}
	ls := NewList(b).SetSlice(&sl)
	changes := 0
	ls.OnChange(func(e events.Event) { changes++ })
	AssertRender(t, b, "list/insert-delete", Locked(b, func() {
		ls.SelectIndexEvent(1, events.SelectOne)
		SendKeyFunction(ls, keymap.InsertAfter)
		assert.Equal(t, []string{"a", "b", "", "c"}, sl)
		assert.Equal(t, 2, ls.SelectedIndex)

		SendKeyFunction(ls, keymap.Insert)
		assert.Equal(t, []string{"a", "b", "", "", "c"}, sl)

		ls.NewAt(-1)
		assert.Equal(t, 6, len(sl), "-1 appends")
		assert.Equal(t, 5, ls.SelectedIndex)

		ls.SelectIndexEvent(4, events.SelectOne)
		ls.DeleteAt(2)
		assert.Equal(t, []string{"a", "b", "", "c", ""}, sl)
		assert.Equal(t, []int{3}, ls.SelectedIndexesList(reflectx.Ascending), "the selection moves up")
		ls.DeleteAt(10)
		ls.DeleteAt(-1)
		assert.Len(t, sl, 5, "out of range deletes do nothing")
		assert.Greater(t, changes, 0)
	}))
}

// TestListArrayFixed tests that an array, which has a fixed size,
// cannot have elements inserted or deleted.
func TestListArrayFixed(t *testing.T) {
	b := NewBody()
	arr := [3]int{1, 2, 3}
	ls := NewList(b).SetSlice(&arr)
	AssertRender(t, b, "list/array", Locked(b, func() {
		ls.NewAt(0)
		ls.DeleteAt(0)
		assert.Equal(t, [3]int{1, 2, 3}, arr)
	}))
}

// TestListCopyPaste tests copying, cutting, duplicating, and pasting
// the elements of a list, through the clipboard.
func TestListCopyPaste(t *testing.T) {
	b := NewBody()
	sl := []string{"a", "b", "c", "d"}
	ls := NewList(b).SetSlice(&sl)
	AssertRender(t, b, "list/copy-paste", Locked(b, func() {
		ls.SelectIndexEvent(0, events.SelectOne)
		ls.SelectIndexEvent(2, events.ExtendOne)
		SendKeyFunction(ls, keymap.Copy)
		md := ls.Clipboard().Read([]string{ls.MimeDataType()})
		assert.Len(t, md, 2)

		ls.PasteAtIndex(md, 4)
		assert.Equal(t, []string{"a", "b", "c", "d", "a", "c"}, sl)
		ls.PasteAtIndex(md, 1)
		assert.Equal(t, []string{"a", "a", "c", "b", "c", "d", "a", "c"}, sl)
		ls.PasteAssign(md, 7)
		assert.Equal(t, "a", sl[7], "assigns the first element")

		ls.SelectIndexEvent(3, events.SelectOne)
		ls.SelectIndexEvent(5, events.ExtendOne)
		ls.cutIndexes()
		assert.Equal(t, []string{"a", "a", "c", "c", "a", "a"}, sl)
		md = ls.Clipboard().Read([]string{ls.MimeDataType()})
		assert.Equal(t, 2, len(md))

		ls.SelectIndexEvent(2, events.SelectOne)
		SendKeyFunction(ls, keymap.Duplicate)
		assert.Equal(t, []string{"a", "a", "c", "c", "c", "a", "a"}, sl)
	}))
}

// TestListSearch tests searching the elements of a list.
func TestListSearch(t *testing.T) {
	b := NewBody()
	sl := []string{"apple", "banana", "cherry", "grape"}
	ls := NewList(b).SetSlice(&sl)
	ls.SetReadOnly(true)
	AssertRender(t, b, "list/search", Locked(b, func() {
		res := SearchWidgets(b, "AP", IgnoreCase)
		assert.Equal(t, 2, res.Len(), "apple, grape")
		assert.Len(t, res, 1, "the list handles the search of all of its elements")
		r, i := res.AtIndex(1)
		assert.NotNil(t, r)
		assert.Equal(t, 1, i, "the local index in the matches of the list")
		r, i = res.AtIndex(5)
		assert.Nil(t, r)
		assert.Equal(t, -1, i)
		SearchHighlight(b, res)
		SearchSelect(res, 1, SelectSet)
		// highlights returns the highlights of the value widget of grape.
		highlights := func() int {
			n := -1
			ls.ListGrid.WidgetWalkDown(func(cw Widget, cwb *WidgetBase) bool {
				if tf, ok := cw.(*TextField); ok && string(tf.TextRunes()) == "grape" {
					n = len(tf.highlights)
				}
				return true
			})
			return n
		}
		require.Equal(t, 1, highlights())
		SearchHighlight(b, nil)
		assert.Equal(t, 0, highlights(), "reset")
	}))
}
