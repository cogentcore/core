// Copyright (c) 2026, Cogent Core. All rights reserved.
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

// treeTableMenu returns the context menu of the tree table with the given
// row selected.
func treeTableMenu(tt *TreeTable, idx int) *Scene {
	tt.SelectIndexEvent(idx, events.SelectOne)
	m := NewScene()
	tt.contextMenu(m, image.Point{})
	return m
}

// TestTreeTableMenuActions tests the actions of the context menu of an
// editable tree table and of a read-only one, moving items, pasting, and
// assigning pasted items.
func TestTreeTableMenuActions(t *testing.T) {
	b := NewBody()
	tasks := testTasks()
	tt := NewTreeTable(b).SetItems(&tasks)
	RenderImage(b, Locked(b, func() {
		click := func(idx int, text string) {
			t.Helper()
			bt := ButtonByText(treeTableMenu(tt, idx), text)
			require.NotNil(t, bt, text)
			bt.Send(events.Click)
		}
		assert.Equal(t, []string{"Work", "  Email", "  Report", "    Outline", "Home", "  Dishes"}, rowNames(tt))
		click(1, "Move down") // Email
		assert.Equal(t, []string{"Work", "  Report", "    Outline", "  Email", "Home", "  Dishes"}, rowNames(tt))
		tt.MoveItemUp(3) // Email
		assert.Equal(t, []string{"Work", "  Email", "  Report", "    Outline", "Home", "  Dishes"}, rowNames(tt))
		click(2, "Move up") // Report
		assert.Equal(t, []string{"Work", "  Report", "    Outline", "  Email", "Home", "  Dishes"}, rowNames(tt))

		click(rowIndex(rowNames(tt), "Home"), "Add child")
		assert.Len(t, tasks[1].Children, 2)
		click(0, "Insert after") // Work
		assert.Len(t, tasks, 3)
		click(0, "Insert before")
		assert.Len(t, tasks, 4)
		click(0, "Delete")
		assert.Len(t, tasks, 3)

		idx := rowIndex(rowNames(tt), "  Email")
		click(idx, "Indent") // into Report, its previous sibling
		assert.Contains(t, rowNames(tt), "    Email")
		idx = rowIndex(rowNames(tt), "    Email")
		click(idx, "Outdent")
		assert.Contains(t, rowNames(tt), "  Email")

		// copying and pasting
		click(0, "Copy") // (which resets the selection)
		md := tt.Clipboard().Read([]string{tt.MimeDataType()})
		require.NotEmpty(t, md)
		home := rowIndex(rowNames(tt), "Home")
		tt.PasteAssign(md, home)
		works := 0
		for _, tk := range tasks {
			if tk.Name == "Work" {
				works++
			}
		}
		assert.Equal(t, 2, works, "Home is assigned the copy of Work")
		n := len(tasks)
		m := NewScene()
		tt.makeTreePasteMenu(m, md, -1, events.DropCopy, nil)
		ButtonByText(m, "Insert at end").Send(events.Click)
		assert.Len(t, tasks, n+1)
		assert.Nil(t, ButtonByText(m, "Assign to"), "not for an index out of range")
		assert.Equal(t, &tt.Items, tt.WidgetValue())

		tt.SetReadOnly(true)
		m = treeTableMenu(tt, 0)
		assert.NotNil(t, ButtonByText(m, "View"))
		assert.Nil(t, ButtonByText(m, "Delete"), "read-only")
	}))
}

// rowIndex returns the index of the given name in the row names, or -1.
func rowIndex(names []string, name string) int {
	for i, n := range names {
		if n == name {
			return i
		}
	}
	return -1
}
