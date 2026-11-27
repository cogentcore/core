// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"testing"

	"cogentcore.org/core/events"
	"cogentcore.org/core/keymap"
	"cogentcore.org/core/tree"
	"github.com/stretchr/testify/assert"
)

// embedTree is a type that embeds [Tree], whose This is not the *Tree.
type embedTree struct {
	Tree
}

// TestTreeUnselectEmbedded checks that unselecting a node of a type that
// embeds [Tree] removes it from the selected nodes.
func TestTreeUnselectEmbedded(t *testing.T) {
	b := NewBody()
	root := tree.New[embedTree](b)
	root.SetText("root")
	for _, lb := range []string{"a", "b", "c"} {
		tree.New[embedTree](root).SetText(lb)
	}
	rt := &root.Tree
	RenderImage(b, Locked(b, func() {
		a, c := treeNode(rt, "a"), treeNode(rt, "c")
		a.SelectEvent(events.SelectOne)
		assert.Equal(t, []string{"a"}, selectedLabels(rt))
		a.Unselect()
		assert.Empty(t, selectedLabels(rt), "Unselect removes the node")

		a.SelectEvent(events.SelectOne)
		c.SelectEvent(events.ExtendOne)
		assert.Equal(t, []string{"a", "c"}, selectedLabels(rt))
		a.SelectEvent(events.ExtendOne) // toggles a off
		assert.Equal(t, []string{"c"}, selectedLabels(rt))
		c.UnselectEvent()
		assert.Empty(t, selectedLabels(rt))

		rt.SelectEvent(events.SelectOne)
		assert.True(t, rt.IsRootSelected())
		assert.Nil(t, rt.moveUp(events.SelectOne), "cannot move up from the root")
		assert.Nil(t, rt.moveDownSibling(events.SelectOne))
	}))
}

// TestTreeKeySelectEvents checks that keyboard moves send the select event
// on the newly selected node (as well as on the root).
func TestTreeKeySelectEvents(t *testing.T) {
	b, root := newTestTree()
	var got []string
	rootEvents := 0
	root.WidgetWalkDown(func(cw Widget, cwb *WidgetBase) bool {
		tr := AsTree(cw)
		if tr == nil {
			return false
		}
		if tr == root {
			tr.OnSelect(func(e events.Event) { rootEvents++ })
			return true
		}
		tr.OnSelect(func(e events.Event) { got = append(got, tr.Label()) })
		return true
	})
	focused := func() *Tree { return AsTree(b.Scene.Events.focus) }
	key := func(kf keymap.Functions) {
		got = nil
		rootEvents = 0
		SendKeyFunction(focused(), kf)
	}
	RenderImage(b, Locked(b, func() {
		root.SelectEvent(events.SelectOne)
		root.SetFocus()

		key(keymap.MoveDown)
		assert.Equal(t, []string{"a"}, got)
		assert.Equal(t, 1, rootEvents)
		key(keymap.MoveDown)
		assert.Equal(t, []string{"a1"}, got)
		key(keymap.MoveUp)
		assert.Equal(t, []string{"a"}, got)
		assert.Equal(t, 1, rootEvents)
		key(keymap.End)
		assert.Equal(t, []string{"c"}, got)
		key(keymap.PageUp)
		assert.Equal(t, []string{"root"}, selectedLabels(root))
		assert.Empty(t, got, "root gets only its own event")
		assert.Equal(t, 1, rootEvents)
		key(keymap.PageDown)
		assert.Equal(t, []string{"c"}, got)
		key(keymap.Home)
		assert.Empty(t, got)
		assert.Equal(t, 1, rootEvents)
	}))
}
