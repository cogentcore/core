// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"testing"

	"cogentcore.org/core/base/fileinfo"
	"cogentcore.org/core/base/fileinfo/mimedata"
	"cogentcore.org/core/events"
	"cogentcore.org/core/keymap"
	"github.com/stretchr/testify/assert"
)

// newTestTree makes a tree in a new body with this structure:
//
//	root
//	  a
//	    a1
//	    a2
//	  b
//	    b1
//	  c
func newTestTree() (*Body, *Tree) {
	b := NewBody()
	root := NewTree(b).SetText("root")
	a := NewTree(root).SetText("a")
	NewTree(a).SetText("a1")
	NewTree(a).SetText("a2")
	bb := NewTree(root).SetText("b")
	NewTree(bb).SetText("b1")
	NewTree(root).SetText("c")
	return b, root
}

// treeNode returns the node of the given tree with the given label.
func treeNode(root *Tree, label string) *Tree {
	var found *Tree
	root.WidgetWalkDown(func(cw Widget, cwb *WidgetBase) bool {
		if tr := AsTree(cw); tr != nil && tr.Label() == label {
			found = tr
			return false
		}
		return true
	})
	return found
}

// selectedLabels returns the labels of the selected nodes of the tree,
// in their selection order.
func selectedLabels(tr *Tree) []string {
	var labels []string
	for _, n := range tr.GetSelectedNodes() {
		labels = append(labels, n.AsCoreTree().Label())
	}
	return labels
}

// treeLabels returns the labels of all of the nodes of the tree, in order.
func treeLabels(root *Tree) []string {
	var labels []string
	root.WidgetWalkDown(func(cw Widget, cwb *WidgetBase) bool {
		if tr := AsTree(cw); tr != nil {
			labels = append(labels, tr.Label())
			return true
		}
		return false
	})
	return labels
}

// TestTreeKeyNavigation tests moving the selection with the arrow, page,
// home, and end keys, and opening and closing nodes with the keyboard.
func TestTreeKeyNavigation(t *testing.T) {
	b, root := newTestTree()
	sel := func() []string { return selectedLabels(root) }
	focused := func() *Tree { return AsTree(b.Scene.Events.focus) }
	key := func(kf keymap.Functions) {
		SendKeyFunction(focused(), kf)
	}
	AssertRender(t, b, "tree/key-navigation", Locked(b, func() {
		root.SelectEvent(events.SelectOne)
		root.SetFocus()
		assert.True(t, root.IsRootSelected())
		assert.True(t, root.HasSelection())

		key(keymap.MoveDown)
		assert.Equal(t, []string{"a"}, sel())
		key(keymap.MoveDown)
		assert.Equal(t, []string{"a1"}, sel())
		key(keymap.MoveDown)
		key(keymap.MoveDown)
		assert.Equal(t, []string{"b"}, sel(), "moves out of a to its next sibling")
		key(keymap.MoveUp)
		assert.Equal(t, []string{"a2"}, sel(), "moves up into the last child of a")

		key(keymap.End)
		assert.Equal(t, []string{"c"}, sel())
		key(keymap.Home)
		assert.Equal(t, []string{"root"}, sel())
		key(keymap.PageDown)
		assert.Equal(t, []string{"c"}, sel(), "page down stops at the last node")
		key(keymap.PageUp)
		assert.Equal(t, []string{"root"}, sel(), "page up stops at the root")

		a := treeNode(root, "a")
		a.SelectEvent(events.SelectOne)
		a.SetFocus()
		key(keymap.MoveLeft)
		assert.True(t, a.Closed, "left closes")
		key(keymap.MoveDown)
		assert.Equal(t, []string{"b"}, sel(), "skips the children of a closed node")
		key(keymap.MoveUp)
		assert.Equal(t, []string{"a"}, sel())
		key(keymap.MoveRight)
		assert.False(t, a.Closed, "right opens")
		key(keymap.Enter)
		assert.True(t, a.Closed, "enter toggles")
		key(keymap.Enter)
		assert.False(t, a.Closed)
	}))
}

// TestTreeSelectExtend tests extending the selection, selecting all, and
// cancelling the selection.
func TestTreeSelectExtend(t *testing.T) {
	b, root := newTestTree()
	sel := func() []string { return selectedLabels(root) }
	AssertRender(t, b, "tree/select-extend", Locked(b, func() {
		a1 := treeNode(root, "a1")
		a1.SelectEvent(events.SelectOne)
		treeNode(root, "b1").SelectEvent(events.ExtendContinuous)
		assert.ElementsMatch(t, []string{"a1", "a2", "b", "b1"}, sel())

		treeNode(root, "c").SelectEvent(events.ExtendOne)
		assert.Len(t, sel(), 5)
		treeNode(root, "c").SelectEvent(events.ExtendOne)
		assert.Len(t, sel(), 4, "extending one toggles a selected node")

		a1.SelectEvent(events.SelectOne)
		assert.Equal(t, []string{"a1"}, sel(), "selecting a selected node of many keeps only it")

		a1.SetFocus()
		SendKeyFunction(a1, keymap.SelectAll)
		assert.Len(t, sel(), 7)
		SendKeyFunction(a1, keymap.CancelSelect)
		assert.Empty(t, sel())
		assert.False(t, root.HasSelection())
	}))
}

// TestTreeOpenCloseAll tests opening and closing all of the nodes of a
// tree, and opening the parents of a node.
func TestTreeOpenCloseAll(t *testing.T) {
	b, root := newTestTree()
	closed := func() []string {
		var labels []string
		root.WidgetWalkDown(func(cw Widget, cwb *WidgetBase) bool {
			if tr := AsTree(cw); tr != nil {
				if tr.Closed && tr.HasChildren() {
					labels = append(labels, tr.Label())
				}
				return true
			}
			return false
		})
		return labels
	}
	AssertRender(t, b, "tree/close-all", Locked(b, func() {
		root.CloseAll()
		assert.Equal(t, []string{"root", "a", "b"}, closed(), "(leaves are not counted)")
		treeNode(root, "a1").OpenParents()
		assert.Equal(t, []string{"b"}, closed())
		root.OpenAll()
		assert.Empty(t, closed())
		treeNode(root, "b").ToggleClose()
		assert.Equal(t, []string{"b"}, closed())
		root.Close()
		assert.Equal(t, []string{"root", "b"}, closed())
	}))
}

// TestTreeCopyPaste tests copying nodes to the clipboard and pasting them
// before, after, and as children of other nodes, and assigning them.
func TestTreeCopyPaste(t *testing.T) {
	b, root := newTestTree()
	AssertRender(t, b, "tree/copy-paste", Locked(b, func() {
		a := treeNode(root, "a")
		a.SelectEvent(events.SelectOne)
		a.Copy()
		md := a.Clipboard().Read([]string{fileinfo.DataJSON})
		assert.Len(t, md, 2, "a path and the json of the node")

		c := treeNode(root, "c")
		c.PasteAfter(md, events.DropCopy)
		pasted := AsTree(root.Child(3))
		assert.Equal(t, []string{"root", "a", "a1", "a2", "b", "b1", "c", "a", "a1", "a2"}, treeLabels(root), "the pasted node keeps its label")
		assert.Equal(t, []string{"a"}, selectedLabels(root), "the pasted node is selected")
		assert.NotEqual(t, a.Name, pasted.Name, "the pasted node gets a unique name")

		treeNode(root, "b").PasteBefore(md, events.DropCopy)
		assert.Equal(t, 5, root.NumChildren())
		assert.Equal(t, "a", AsTree(root.Child(1)).Label(), "a copy of a")

		b1 := treeNode(root, "b1")
		b1.PasteChildren(md, events.DropCopy)
		assert.Equal(t, 1, b1.NumChildren())
		assert.Equal(t, []string{"b1", "a", "a1", "a2"}, treeLabels(b1))

		c.PasteAssign(md)
		assert.Equal(t, 2, c.NumChildren(), "assigned the children of a")
	}))
}

// TestTreeCutDelete tests cutting and deleting the selected nodes,
// with the keyboard, which the root does not allow.
func TestTreeCutDelete(t *testing.T) {
	b, root := newTestTree()
	AssertRender(t, b, "tree/cut-delete", Locked(b, func() {
		changes := 0
		root.OnChange(func(e events.Event) { changes++ })

		a := treeNode(root, "a")
		a.SelectEvent(events.SelectOne)
		treeNode(root, "c").SelectEvent(events.ExtendOne)
		a.SetFocus()
		SendKeyFunction(a, keymap.Cut)
		assert.Equal(t, []string{"root", "b", "b1"}, treeLabels(root))
		assert.Equal(t, 1, changes)
		md := root.Clipboard().Read([]string{fileinfo.DataJSON})
		assert.Len(t, md, 4, "both cut nodes are on the clipboard")

		b1 := treeNode(root, "b1")
		b1.SelectEvent(events.SelectOne)
		b1.SetFocus()
		SendKeyFunction(b1, keymap.Delete)
		assert.Equal(t, []string{"root", "b"}, treeLabels(root))
		assert.Equal(t, 2, changes)

		root.SelectEvent(events.SelectOne)
		root.Cut()
		root.DeleteSelected()
		assert.Equal(t, []string{"root", "b"}, treeLabels(root), "the root is not cut or deleted")
	}))
}

// TestTreeReadOnlyKeys tests that the editing keys do nothing in a
// read-only tree, while copy still works.
func TestTreeReadOnlyKeys(t *testing.T) {
	b, root := newTestTree()
	root.SetReadOnly(true)
	AssertRender(t, b, "tree/read-only-keys", Locked(b, func() {
		a := treeNode(root, "a")
		a.SelectEvent(events.SelectOne)
		a.SetFocus()
		a.Clipboard().Write(mimedata.NewText("none"))
		for _, kf := range []keymap.Functions{keymap.Delete, keymap.Cut, keymap.Duplicate, keymap.InsertAfter} {
			SendKeyFunction(a, kf)
		}
		assert.Len(t, treeLabels(root), 7)
		SendKeyFunction(a, keymap.Copy)
		md := a.Clipboard().Read([]string{fileinfo.DataJSON})
		assert.Len(t, md, 2)
	}))
}

// TestTreeSearch tests searching the labels of the nodes of a tree,
// highlighting the matches, and selecting one in a closed node.
func TestTreeSearch(t *testing.T) {
	b, root := newTestTree()
	AssertRender(t, b, "tree/search", Locked(b, func() {
		treeNode(root, "a").Close()
		res := SearchWidgets(b, "A", IgnoreCase)
		assert.Equal(t, 3, res.Len(), "a, a1, a2")
		assert.Len(t, res, 1, "the tree handles the search of all of its nodes")
		assert.Empty(t, SearchWidgets(b, "A", UseCase))
		SearchHighlight(b, res)
		SearchSelect(res, 2, SelectSet)
		assert.False(t, treeNode(root, "a").Closed, "selecting a match opens its parents")
	}))
}
