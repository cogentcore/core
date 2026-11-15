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

// newDnDTree returns a tree of root with children a, b, and c.
func newDnDTree() (*Body, *Tree, []*Tree) {
	b := NewBody()
	root := NewTree(b).SetText("root")
	var kids []*Tree
	for _, s := range []string{"a", "b", "c"} {
		kids = append(kids, NewTree(root).SetText(s))
	}
	return b, root, kids
}

// treeTexts returns the texts of the children of the tree.
func treeTexts(tr *Tree) []string {
	var ts []string
	for _, k := range tr.Children {
		if kt := AsTree(k); kt != nil {
			ts = append(ts, kt.Text)
		}
	}
	return ts
}

// rowCenter returns the window position of the middle of the row of a tree
// node (its parts), not including its children.
func rowCenter(tr *Tree) image.Point {
	bb := tr.Parts.Geom.TotalBBox
	return bb.Min.Add(bb.Max).Div(2)
}

// popupButton returns the button with the given text in a popup of the
// window of the body, or nil.
func popupButton(b *Body, text string) *Button {
	for _, sc := range Popups(b) {
		if bt := ButtonByText(sc, text); bt != nil {
			return bt
		}
	}
	return nil
}

// TestTreeDragDrop tests dragging a node of a tree with the mouse and
// dropping a copy of it after another one, chosen in the drop menu.
func TestTreeDragDrop(t *testing.T) {
	b, root, kids := newDnDTree()
	var from, to image.Point
	AssertRender(t, b, "tree/drag-drop", Locked(b, func() {
		from, to = rowCenter(kids[0]), rowCenter(kids[2])
	}), func() {
		DragMouse(b, from, to)
	}, Locked(b, func() {
		bt := popupButton(b, "Insert After")
		require.NotNil(t, bt, "the drop menu")
		bt.Send(events.Click)
	}), Locked(b, func() {
		assert.Equal(t, []string{"a", "b", "c", "a"}, treeTexts(root), "copied after c (dragging copies by default)")
	}))
}

// TestTreePasteClipboard tests pasting a copied node through the paste menu,
// and the context menus of editable and read-only trees.
func TestTreePasteClipboard(t *testing.T) {
	b, root, kids := newDnDTree()
	AssertRender(t, b, "tree/paste-menu", Locked(b, func() {
		m := NewScene()
		kids[0].contextMenu(m, image.Point{})
		assert.NotNil(t, ButtonByText(m, "Add child"))
		assert.NotNil(t, ButtonByText(m, "Delete"))

		kids[0].SelectEvent(events.SelectOne)
		kids[0].Copy()
		kids[2].Paste()
	}), Locked(b, func() {
		bt := popupButton(b, "Add to Children")
		require.NotNil(t, bt, "the paste menu")
		bt.Send(events.Click)
	}), Locked(b, func() {
		assert.Equal(t, []string{"a"}, treeTexts(kids[2]), "pasted as a child of c")
		assert.Equal(t, []string{"a", "b", "c"}, treeTexts(root))

		root.SetReadOnly(true)
		m := NewScene()
		kids[1].contextMenu(m, image.Point{})
		assert.Nil(t, ButtonByText(m, "Delete"), "read-only")
		assert.NotNil(t, ButtonByText(m, "View"))
		assert.Nil(t, kids[1].TextRunes())
	}))
}
