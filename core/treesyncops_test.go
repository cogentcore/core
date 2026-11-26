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
	"cogentcore.org/core/tree"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newSyncTestTree returns a new sync tree in a new body, of a source
// frame with a button, a text, and a frame with a button.
func newSyncTestTree() (*Body, *Frame, *Tree) {
	b := NewBody()
	src := NewFrame()
	NewButton(src).SetName("bt")
	NewText(src).SetName("tx")
	sub := NewFrame(src).SetName("sub")
	NewButton(sub).SetName("sbt")
	st := NewTree(b).SyncTree(src)
	return b, src, st
}

// childNames returns the names of the children of the given node.
func treeChildNames(n tree.Node) []string {
	var names []string
	for _, c := range n.AsTree().Children {
		names = append(names, c.AsTree().Name)
	}
	return names
}

// TestTreeSyncFind tests finding the tree nodes of source nodes, and
// the source nodes of the selected tree nodes.
func TestTreeSyncFind(t *testing.T) {
	b, src, st := newSyncTestTree()
	AssertRender(t, b, "tree/sync-find", Locked(b, func() {
		sbt := src.Child(2).AsTree().Child(0)
		tn := st.FindSyncNode(sbt)
		require.NotNil(t, tn)
		assert.Equal(t, "sbt", tn.Label())
		assert.Nil(t, st.FindSyncNode(NewButton()), "not in the tree")

		tn.SelectEvent(events.SelectOne)
		st.FindSyncNode(src.Child(0)).SelectEvent(events.ExtendOne)
		assert.ElementsMatch(t, []tree.Node{sbt, src.Child(0)}, st.selectedSyncNodes())
	}))
}

// TestTreeSyncEdit tests deleting, duplicating, and cutting nodes of a
// sync tree, which edit the source tree.
func TestTreeSyncEdit(t *testing.T) {
	b, src, st := newSyncTestTree()
	changes := 0
	st.OnChange(func(e events.Event) { changes++ })
	AssertRender(t, b, "tree/sync-edit", Locked(b, func() {
		st.FindSyncNode(src.Child(1)).DeleteNode()
		assert.Equal(t, []string{"bt", "sub"}, treeChildNames(src))
		assert.Equal(t, 2, st.NumChildren(), "resynced")

		bt := st.FindSyncNode(src.Child(0))
		bt.SelectEvent(events.SelectOne)
		bt.SetFocus()
		SendKeyFunction(bt, keymap.Duplicate)
		assert.Equal(t, []string{"bt", "bt_Copy", "sub"}, treeChildNames(src))

		st.FindSyncNode(src.Child(1)).SelectEvent(events.SelectOne)
		st.FindSyncNode(src.Child(2)).SelectEvent(events.ExtendOne)
		st.FindSyncNode(src.Child(1)).Cut()
		assert.Equal(t, []string{"bt"}, treeChildNames(src))
		md := st.Clipboard().Read([]string{fileinfo.DataJSON})
		assert.Len(t, md, 4, "a path and json for each node")

		st.FindSyncNode(src.Child(0)).SelectEvent(events.SelectOne)
		st.FindSyncNode(src.Child(0)).DeleteSelected()
		assert.Empty(t, treeChildNames(src))
		assert.Greater(t, changes, 3)
	}))
}

// TestTreeSyncPaste tests pasting copied nodes into a sync tree, which
// edits the source tree.
func TestTreeSyncPaste(t *testing.T) {
	b, src, st := newSyncTestTree()
	AssertRender(t, b, "tree/sync-paste", Locked(b, func() {
		bt := st.FindSyncNode(src.Child(0))
		bt.SelectEvent(events.SelectOne)
		bt.Copy()
		md := st.Clipboard().Read([]string{fileinfo.DataJSON})
		require.Len(t, md, 2)

		st.FindSyncNode(src.Child(1)).PasteAfter(md, events.DropCopy)
		assert.Len(t, src.Children, 4)
		assert.IsType(t, &Button{}, src.Child(2))
		assert.NotEqual(t, "bt", src.Child(2).AsTree().Name, "a unique name")

		st.FindSyncNode(src.Child(0)).PasteBefore(md, events.DropCopy)
		assert.Len(t, src.Children, 5)
		assert.Equal(t, 0, st.FindSyncNode(src.Child(0)).SyncNode.AsTree().IndexInParent())

		sub := src.ChildByName("sub", 0)
		st.FindSyncNode(sub).PasteChildren(md, events.DropCopy)
		assert.Len(t, sub.AsTree().Children, 2)
		assert.IsType(t, &Button{}, sub.AsTree().Child(1))

		tx := st.FindSyncNode(src.ChildByName("tx", 0))
		tx.PasteAssign(md)
		assert.IsType(t, &Text{}, tx.SyncNode, "assigning keeps the type of the node")
		assert.Equal(t, st.NumChildren(), len(src.Children), "resynced")
	}))
}

// TestTreeMoveKeepsName tests that moving a node within its parent by
// drag-and-drop keeps its name, in a plain and a sync tree, and that moving
// a copy of it still gives a unique name.
func TestTreeMoveKeepsName(t *testing.T) {
	// move drags a node and drops it after another one, deleting the source
	move := func(from, to *Tree, mod events.DropMods) {
		var md mimedata.Mimes
		from.MimeData(&md)
		to.PasteAfter(md, mod)
		if mod == events.DropMove {
			from.DropDeleteSource(&events.DragDrop{Data: md, DropMod: mod})
		}
	}
	_, root, kids := newDnDTree()
	for i, k := range kids {
		k.SetName([]string{"a", "b", "c"}[i])
	}
	move(kids[0], kids[2], events.DropMove)
	assert.Equal(t, []string{"b", "c", "a"}, treeChildNames(root))
	move(AsTree(root.Child(0)), AsTree(root.Child(2)), events.DropCopy)
	assert.Equal(t, "b", root.Child(0).AsTree().Name)
	assert.NotEqual(t, "b", root.Child(3).AsTree().Name, "a copy has a unique name")

	_, src, st := newSyncTestTree()
	move(st.FindSyncNode(src.Child(0)), st.FindSyncNode(src.Child(2)), events.DropMove)
	assert.Equal(t, []string{"tx", "sub", "bt"}, treeChildNames(src))
	assert.Equal(t, st.NumChildren(), len(src.Children), "resynced")
}
