// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"image"
	"testing"
	"time"

	"cogentcore.org/core/base/fileinfo/mimedata"
	"cogentcore.org/core/events"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// menuButton returns the button of the given menu scene with the given
// text, or nil.
func menuButton(m *Scene, text string) *Button {
	for cw := range m.AllWidgets() {
		if bt, ok := cw.(*Button); ok && bt.Text == text {
			return bt
		}
	}
	return nil
}

// TestListPasteMenu tests the menu for pasting copied elements of a list:
// assigning, and inserting before and after an element.
func TestListPasteMenu(t *testing.T) {
	b := NewBody()
	sl := []string{"a", "b", "c"}
	ls := NewList(b).SetSlice(&sl)
	RenderImage(b, Locked(b, func() {
		ls.SelectIndexEvent(2, events.SelectOne)
		md := ls.CopySelectToMime()
		require.Len(t, md, 1)
		done := 0
		menu := func() *Scene {
			m := NewScene()
			ls.makePasteMenu(m, image.Point{}, md, 0, events.DropCopy, func() { done++ })
			return m
		}
		menuButton(menu(), "Insert before").Send(events.Click)
		assert.Equal(t, []string{"c", "a", "b", "c"}, sl)
		menuButton(menu(), "Insert after").Send(events.Click)
		assert.Equal(t, []string{"c", "c", "a", "b", "c"}, sl)
		menuButton(menu(), "Assign to").Send(events.Click)
		assert.Equal(t, "c", sl[0])
		assert.Equal(t, 3, done)
		assert.NotNil(t, menuButton(menu(), "Cancel"))

		m := NewScene()
		ls.makePasteMenu(m, image.Point{}, md, 0, events.DropMove, nil)
		assert.Nil(t, menuButton(m, "Assign to"), "no assigning for a move")

		// pasting from the clipboard shows the menu
		ls.SelectIndexEvent(1, events.SelectOne)
		ls.copyIndexes(false)
		ls.pasteIndex(0)
	}))
}

// TestListDragDrop tests dragging elements of a list and dropping them on
// another one, which moves them, as chosen in the drop menu.
func TestListDragDrop(t *testing.T) {
	b := NewBody()
	sl := []string{"a", "b", "c", "d"}
	ls := NewList(b).SetSlice(&sl)
	RenderImage(b, Locked(b, func() {
		ls.SelectIndexEvent(0, events.SelectOne)
		md := ls.CopySelectToMime()
		de := &events.DragDrop{DropMod: events.DropMove, Data: md, Source: ls.This}
		de.Typ = events.Drop
		w, ok := ls.rowFirstWidget(2) // the row of c
		require.True(t, ok)
		de.Where = w.Geom.TotalBBox.Min.Add(image.Pt(2, 2))
		idx, ok := ls.indexFromPos(de.Where.Y)
		require.True(t, ok)
		assert.Equal(t, 2, idx)

		ls.saveDraggedIndexes(idx)
		assert.Equal(t, []int{0}, ls.draggedIndexes, "before the drop, so not moved")
		m := NewScene()
		ls.makePasteMenu(m, image.Point{}, md, idx, de.DropMod, func() { ls.dropFinalize(de) })
		menuButton(m, "Insert after").Send(events.Click)
		assert.Equal(t, []string{"b", "c", "a", "d"}, sl, "moved after c")

		// a drop with no data does nothing
		ls.dragDrop(&events.DragDrop{})
		var empty mimedata.Mimes
		assert.Empty(t, ls.fromMimeData(empty))
	}))
}

// TestTreePasteMenu tests the menu for pasting copied tree nodes, and
// dropping a dragged node, which moves it.
func TestTreePasteMenu(t *testing.T) {
	b, root := newTestTree()
	RenderImage(b, Locked(b, func() {
		a := treeNode(root, "a")
		a.SelectEvent(events.SelectOne)
		var md mimedata.Mimes
		a.MimeData(&md)
		menu := func(tr *Tree, fun func()) *Scene {
			m := NewScene()
			tr.makePasteMenu(m, image.Point{}, md, fun)
			return m
		}
		c := treeNode(root, "c")
		menuButton(menu(c, nil), "Insert After").Send(events.Click)
		assert.Equal(t, 4, root.NumChildren())
		menuButton(menu(c, nil), "Insert Before").Send(events.Click)
		assert.Equal(t, 5, root.NumChildren())
		menuButton(menu(c, nil), "Add to Children").Send(events.Click)
		assert.Equal(t, 1, c.NumChildren())
		b1 := treeNode(root, "b1")
		menuButton(menu(b1, nil), "Assign To").Send(events.Click)
		assert.Equal(t, 2, b1.NumChildren(), "the children of a")
		assert.Nil(t, menuButton(menu(root, nil), "Insert Before"), "not for the root")

		// a drop of a moved node deletes its source after the paste
		n := root.NumChildren()
		bb := treeNode(root, "b")
		de := &events.DragDrop{DropMod: events.DropMove, Data: md, Source: a.This}
		de.Typ = events.Drop
		menuButton(menu(bb, func() { bb.DropFinalize(de) }), "Insert After").Send(events.Click)
		assert.Equal(t, n, root.NumChildren(), "moved: added and deleted")
		assert.NotContains(t, root.Children, a.This, "the source is deleted")

		c.SelectEvent(events.SelectOne)
		var cmd mimedata.Mimes
		c.MimeData(&cmd)
		cde := &events.DragDrop{DropMod: events.DropCopy, Data: cmd, Source: c.This}
		cde.Typ = events.Drop
		bb.DragDrop(cde) // shows the menu
		bb.dropExternal(cmd, events.DropCopy)
	}))
}

// dropMenuAfterDrag drags from the value of row 0 to row 2 of the given
// list, with mouse events in its window, and returns whether the drop menu
// appeared.
func dropMenuAfterDrag(t *testing.T, b *Body, lb *ListBase) bool {
	prev := TimingSettings.DragStartTime
	TimingSettings.DragStartTime = 0
	defer func() { TimingSettings.DragStartTime = prev }()
	got := false
	RenderImage(b, func() {
		b.AsyncLock()
		nw, idx := lb.This.(Lister).RowWidgetNs()
		from := lb.ListGrid.Child(idx).(Widget).AsWidget().Geom.TotalBBox.Min.Add(image.Pt(3, 3))
		to := lb.ListGrid.Child(2*nw + idx).(Widget).AsWidget().Geom.TotalBBox.Min.Add(image.Pt(3, 3))
		b.AsyncUnlock()
		ev := b.Scene.RenderWindow().SystemWindow.Events()
		ev.MouseButton(events.MouseDown, events.Left, from, 0)
		time.Sleep(20 * time.Millisecond)
		for i := 1; i <= 10; i++ {
			ev.MouseMove(from.Add(to.Sub(from).Mul(i).Div(10)))
			time.Sleep(10 * time.Millisecond)
		}
		ev.MouseButton(events.MouseUp, events.Left, to, 0)
	}, Locked(b, func() {
		for _, st := range b.Scene.Stage.popups.stack.Values {
			if menuButton(st.Scene, "Insert after") != nil {
				got = true
			}
		}
	}))
	return got
}

// TestListRowDrag tests that dragging a row of a list or a table by its
// value (not its index, which is not shown by default) drops it, with a
// menu (#1553).
func TestListRowDrag(t *testing.T) {
	b := NewBody()
	sl := []string{"a", "b", "c", "d"}
	assert.True(t, dropMenuAfterDrag(t, b, NewList(b).SetSlice(&sl)), "list")
	b = NewBody()
	langs := []language{{"Go", 10}, {"Python", 5}, {"C", 3}, {"D", 1}}
	tb := NewTable(b).SetSlice(&langs)
	assert.True(t, dropMenuAfterDrag(t, b, &tb.ListBase), "table")
}
