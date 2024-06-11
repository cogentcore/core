// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package tree

import (
	"fmt"
	"reflect"
	"slices"
	"strconv"
	"sync/atomic"

	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/types"
)

// admin.go has infrastructure code outside of the [Node] interface.

// New returns a new node of the given the type with the given optional parent.
// If the name is unspecified, it defaults to the ID (kebab-case) name of
// the type, plus the [Node.NumLifetimeChildren] of the parent.
func New[T Node](parent ...Node) T {
	if len(parent) == 0 {
		return newRoot[T]()
	}
	var n T
	return parent[0].AsTree().NewChild(n.NodeType()).(T)
}

// newRoot returns a new initialized node of the given type without a parent.
func newRoot[T Node]() T {
	var n T
	typ := n.NodeType()
	n = NewOfType(typ).(T)
	initNode(n)
	n.AsTree().SetName(n.NodeType().IDName)
	return n
}

// initNode initializes the node.
func initNode(n Node) {
	nb := n.AsTree()
	if nb.Ths != n {
		nb.Ths = n
		nb.Ths.Init()
	}
}

// checkThis checks that [Node.This] is non-nil.
// It returns and logs an error otherwise.
func checkThis(n Node) error {
	if n.AsTree().This() != nil {
		return nil
	}
	return errors.Log(fmt.Errorf("tree.Node %q has nil tree.NodeBase.This; you must use tree.New or New* so that the node is initialized", n.AsTree().Path()))
}

// SetParent sets the parent of the given node to the given parent node.
// This is only for nodes with no existing parent; see [MoveToParent] to
// move nodes that already have a parent. It does not add the node to the
// parent's list of children; see [Node.AddChild] for a version that does.
// It automatically gets the [Node.This] of the parent.
func SetParent(child Node, parent Node) {
	n := child.AsTree()
	n.Par = parent.AsTree().This()
	setUniqueName(n, false)
	child.AsTree().This().OnAdd()
	n.WalkUpParent(func(pn Node) bool {
		pn.AsTree().This().OnChildAdded(child)
		return Continue
	})
}

// MoveToParent removes the given node from its current parent
// and adds it as a child of the given new parent.
// The old and new parents can be in different trees (or not).
func MoveToParent(child Node, parent Node) {
	oldParent := child.AsTree().Parent()
	if oldParent != nil {
		idx := IndexOf(oldParent.AsTree().Children, child)
		if idx >= 0 {
			oldParent.AsTree().Children = slices.Delete(oldParent.AsTree().Children, idx, idx+1)
		}
	}
	parent.AsTree().AddChild(child)
}

// InsertNewChild is a generic helper function for [NodeBase.InsertNewChild].
func InsertNewChild[T Node](parent Node, at int) T {
	var n T
	return parent.AsTree().InsertNewChild(n.NodeType(), at).(T)
}

// ParentByType is a generic helper function for [NodeBase.ParentByType].
func ParentByType[T Node](n Node, embeds bool) T {
	var nt T
	v, _ := n.AsTree().ParentByType(nt.NodeType(), embeds).(T)
	return v
}

// ChildByType is a generic helper function for [NodeBase.ChildByType].
func ChildByType[T Node](k Node, embeds bool, startIndex ...int) T {
	var n T
	v, _ := k.AsTree().ChildByType(n.NodeType(), embeds, startIndex...).(T)
	return v
}

// IsRoot returns whether the given node is the root node in its tree.
func IsRoot(n Node) bool {
	return n.AsTree().Parent() == nil
}

// Root returns the root node of the given node's tree.
func Root(n Node) Node {
	if IsRoot(n) {
		return n
	}
	return Root(n.AsTree().Parent())
}

// nodeType is the [reflect.Type] of [Node].
var nodeType = reflect.TypeFor[Node]()

// IsNode returns whether the given type or a pointer to it
// implements the [Node] interface.
func IsNode(typ reflect.Type) bool {
	if typ == nil {
		return false
	}
	return typ.Implements(nodeType) || reflect.PointerTo(typ).Implements(nodeType)
}

// NewOfType returns a new instance of the given [Node] type.
func NewOfType(typ *types.Type) Node {
	return typ.Instance.(Node).New()
}

// SetUniqueName sets the name of the node to be unique, using
// the number of lifetime children of the parent node as a unique
// identifier. If the node already has a name, it adds the unique id
// to it. Otherwise, it uses the type name of the node plus the unique id.
func SetUniqueName(n Node) {
	setUniqueName(n, true)
}

// setUniqueName is the implementation of [SetUniqueName] that takes whether
// to add the unique id to the name even if it is already set.
func setUniqueName(n Node, addIfSet bool) {
	nb := n.AsTree()
	pn := nb.Parent()
	if pn == nil {
		return
	}
	c := atomic.AddUint64(&pn.AsTree().numLifetimeChildren, 1)
	id := "-" + strconv.FormatUint(c-1, 10) // must subtract 1 so we start at 0
	if nb.Name() == "" {
		// must get This for accurate NodeType
		nb.SetName(nb.This().NodeType().IDName + id)
	} else if addIfSet {
		nb.SetName(nb.Name() + id)
	}
}
