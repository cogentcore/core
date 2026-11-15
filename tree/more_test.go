// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package tree_test

import (
	"reflect"
	"testing"

	. "cogentcore.org/core/tree"
	"cogentcore.org/core/tree/testdata"
	"cogentcore.org/core/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// embedType returns the type of [testdata.NodeEmbed].
func embedType() *types.Type { return testdata.NewNodeEmbed().NodeType() }

// newTestTree returns a root with children a and b, and b with a child c.
func newTestTree() (root *NodeBase, a, b, c *testdata.NodeEmbed) {
	root = NewNodeBase()
	root.SetName("root")
	a = NewOfType(embedType()).(*testdata.NodeEmbed)
	a.SetName("a")
	root.AddChild(a)
	b = testdata.NewNodeEmbed(root)
	b.SetName("b")
	c = testdata.NewNodeEmbed(b)
	c.SetName("c")
	return
}

// TestFindPathIndexes tests finding nodes by paths with indexes, from the
// start and the end, and that invalid and out of range indexes find nothing.
func TestFindPathIndexes(t *testing.T) {
	root, a, b, c := newTestTree()
	assert.Equal(t, Node(a), root.FindPath("[0]"))
	assert.Equal(t, Node(b), root.FindPath("[-1]"))
	assert.Equal(t, Node(c), root.FindPath("b/[0]"))
	assert.Equal(t, Node(c), root.FindPath(`"[1]/c"`))
	assert.Nil(t, root.FindPath("[x]"), "not an index")
	assert.Nil(t, root.FindPath("[2]"), "out of range")
	assert.Nil(t, root.FindPath("[-3]"), "out of range from the end")
	assert.Nil(t, root.FindPath("b/nope"))
}

// TestParentsChildren tests finding parents and children, and the root.
func TestParentsChildren(t *testing.T) {
	root, a, b, c := newTestTree()
	assert.Equal(t, Node(b), ParentByType[*testdata.NodeEmbed](c))
	assert.Nil(t, ParentByType[*testdata.NodeEmbed](b), "the root is a NodeBase")
	assert.Equal(t, Node(root), ParentByType[*NodeBase](c))
	assert.Nil(t, ParentByType[*NodeBase](root))
	assert.Equal(t, Node(b), c.ParentByName("b"))
	assert.Equal(t, Node(root), c.ParentByName("root"))
	assert.Nil(t, c.ParentByName("x"))
	assert.Equal(t, 0, c.ParentLevel(b), "the direct parent")
	assert.Equal(t, 1, c.ParentLevel(root))
	assert.Equal(t, -1, c.ParentLevel(a))
	assert.Equal(t, Node(b), root.ChildByName("b"))
	assert.Nil(t, root.ChildByName("x"))
	assert.Nil(t, root.Child(-1))
	assert.Equal(t, Node(root), Root(c))
	assert.Equal(t, Node(root), Root(root))
	assert.Equal(t, Node(a), ChildByType[*testdata.NodeEmbed](root), "the first one")
	assert.Equal(t, Node(b), ChildByType[*testdata.NodeEmbed](root, 1))
	assert.Nil(t, ChildByType[*NodeBase](root))

	assert.True(t, IsNode(reflect.TypeFor[testdata.NodeEmbed]()))
	assert.True(t, IsNode(reflect.TypeFor[*NodeBase]()))
	assert.False(t, IsNode(reflect.TypeFor[int]()))
	assert.False(t, IsNode(nil))
	assert.False(t, IsNil(a))
	assert.True(t, IsNil(nil))
	a.Destroy()
	assert.True(t, IsNil(a), "destroyed")
}

// TestMoveAndDelete tests moving a node to another parent, renaming
// duplicates, and deleting nodes and properties.
func TestMoveAndDelete(t *testing.T) {
	root, a, b, c := newTestTree()
	MoveToParent(c, a)
	assert.Equal(t, Node(a), c.Parent)
	assert.Equal(t, 0, b.NumChildren())
	assert.Equal(t, "/root/a/c", c.Path())

	d := NewNodeBase()
	d.SetName("c")
	assert.True(t, SetUniqueNameIfDuplicate(a, d))
	assert.NotEqual(t, "c", d.Name)
	assert.Nil(t, d.Parent, "the parent is restored")
	e := NewNodeBase()
	e.SetName("e")
	assert.False(t, SetUniqueNameIfDuplicate(a, e))
	assert.Equal(t, "e", e.Name)

	assert.False(t, root.DeleteChild(nil))
	assert.False(t, root.DeleteChild(c), "not a child of the root")
	c.Delete()
	assert.Equal(t, 0, a.NumChildren())
	assert.True(t, IsNil(c))
	alone := NewNodeBase()
	alone.Delete()
	assert.True(t, IsNil(alone), "a root destroys itself")

	b.DeleteProperty("x") // no properties
	b.SetProperty("x", 1)
	b.DeleteProperty("x")
	assert.Nil(t, b.Property("x"))
	root.DeleteChildren()
	assert.Equal(t, 0, root.NumChildren())
	assert.True(t, IsNil(a))
}

// TestCopyClone tests copying the fields, properties, and children of a
// node, and cloning a tree.
func TestCopyClone(t *testing.T) {
	_, _, b, c := newTestTree()
	b.Mbr1 = "one"
	b.Mbr2 = 2
	c.Mbr1 = "child"
	b.SetProperty("p", "v")

	cl := b.Clone().(*testdata.NodeEmbed)
	assert.Equal(t, "b", cl.Name)
	assert.Equal(t, "one", cl.Mbr1)
	assert.Equal(t, 2, cl.Mbr2)
	assert.Equal(t, "v", cl.Property("p"))
	require.Equal(t, 1, cl.NumChildren())
	cc := cl.Child(0).(*testdata.NodeEmbed)
	assert.Equal(t, "child", cc.Mbr1)
	assert.Equal(t, Node(cl), cc.Parent)
	assert.NotSame(t, c, cc)

	// copying a node without children removes them
	cl.CopyFrom(NewOfType(embedType()))
	assert.Equal(t, 0, cl.NumChildren())
	cl.CopyFrom(nil) // logs an error
	cl.Mbr1 = "kept"
	cl.CopyFieldsFrom(NewNodeBase()) // a different type: logs an error
	assert.Equal(t, "kept", cl.Mbr1, "not copied from another type")
}

// TestPlanUpdaters tests the tiers of updaters and makers, and the
// functions that add plan items.
func TestPlanUpdaters(t *testing.T) {
	root := NewNodeBase()
	var order []string
	root.Updater(func() { order = append(order, "normal") })
	root.FirstUpdater(func() { order = append(order, "first") })
	root.FinalUpdater(func() { order = append(order, "final") })
	root.RunUpdaters()
	assert.Equal(t, []string{"first", "normal", "final"}, order)

	inits := 0
	root.FirstMaker(func(p *Plan) {
		AddNew(p, "new", func() *testdata.NodeEmbed { return testdata.NewNodeEmbed() }, func(w *testdata.NodeEmbed) { w.Mbr2 = 5 })
	})
	root.FinalMaker(func(p *Plan) {
		AddInit(p, "new", func(w *testdata.NodeEmbed) { inits++ })
		AddInit(p, "missing", func(w *testdata.NodeEmbed) {}) // logs an error
	})
	AddChildAt(root, "at", func(w *testdata.NodeEmbed) { w.Mbr1 = "at" })
	AddChildInit(root, "at", func(w *testdata.NodeEmbed) { w.Mbr2 = 7 })
	root.UpdateFromMake()
	require.Equal(t, 2, root.NumChildren())
	nw := root.ChildByName("new").(*testdata.NodeEmbed)
	assert.Equal(t, 5, nw.Mbr2)
	assert.Equal(t, 1, inits)
	at := root.ChildByName("at").(*testdata.NodeEmbed)
	assert.Equal(t, "at", at.Mbr1)
	assert.Equal(t, 7, at.Mbr2)

	var tp TypePlan
	tp.Add(embedType(), "x")
	assert.True(t, Update(root, tp))
	assert.Equal(t, "x", root.Child(0).AsTree().Name)

	root.NodeWalkDown(func(n Node) bool { return Continue }) // a placeholder
	root.Init()
	root.OnAdd()
}
