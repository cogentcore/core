// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package tree

import (
	"fmt"
	"reflect"
	"strconv"
	"sync/atomic"

	"cogentcore.org/core/gox/errors"
	"cogentcore.org/core/types"
)

// admin.go has infrastructure code outside of the Node interface.

// initNode initializes the node. See [Node.InitName] for more information
// on what it does.
func initNode(this Node) {
	n := this.AsTreeNode()
	if n.Ths != this {
		n.Ths = this
		n.Ths.OnInit()
	}
}

// checkThis checks that [Node.This] is non-nil.
// It returns and logs an error otherwise.
func checkThis(n Node) error {
	if n.This() != nil {
		return nil
	}
	return errors.Log(fmt.Errorf("tree.Node %q has nil Node.This; you must use NewRoot or call Node.InitName on root nodes", n.Path()))
}

// SetParent sets the parent of the given node to the given parent node.
// This is only for nodes with no existing parent; see [MoveToParent] to
// move nodes that already have a parent. It does not add the node to the
// parent's list of children; see [Node.AddChild] for a version that does.
func SetParent(child Node, parent Node) {
	n := child.AsTreeNode()
	n.Par = parent
	if parent != nil {
		pn := parent.AsTreeNode()
		c := atomic.AddUint64(&pn.numLifetimeChildren, 1)
		if child.Name() == "" {
			child.SetName(child.NodeType().IDName + "-" + strconv.FormatUint(c-1, 10)) // must subtract 1 so we start at 0
		}
	}
	child.This().OnAdd()
	n.WalkUpParent(func(k Node) bool {
		k.This().OnChildAdded(child)
		return Continue
	})
}

// MoveToParent removes the given node from its current parent
// and adds it as a child of the given new parent.
// The old and new parents can be in different trees (or not).
func MoveToParent(child Node, parent Node) {
	oldParent := child.Parent()
	if oldParent != nil {
		idx, ok := oldParent.Children().IndexOf(child)
		if ok {
			oldParent.Children().DeleteAtIndex(idx)
		}
	}
	parent.AddChild(child)
}

// New adds a new child of the given the type
// with the given name to the given parent.
// If the name is unspecified, it defaults to the
// ID (kebab-case) name of the type, plus the
// [Node.NumLifetimeChildren] of its parent.
// It is a generic helper function that calls [Node.NewChild].
func New[T Node](parent Node, name ...string) T {
	var n T
	return parent.NewChild(n.NodeType(), name...).(T)
}

// NewRoot returns a new root node of the given the type
// with the given name. If the name is unspecified, it
// defaults to the ID (kebab-case) name of the type.
// It is a generic helper function that calls [Node.InitName].
func NewRoot[T Node](name ...string) T {
	var n T
	n = n.New().(T)
	n.InitName(n, name...)
	return n
}

// InsertNewChild is a generic helper function for [Node.InsertNewChild].
func InsertNewChild[T Node](parent Node, at int, name ...string) T {
	var n T
	return parent.InsertNewChild(n.NodeType(), at, name...).(T)
}

// ParentByType is a generic helper function for [Node.ParentByType].
func ParentByType[T Node](k Node, embeds bool) T {
	var n T
	v, _ := k.ParentByType(n.NodeType(), embeds).(T)
	return v
}

// ChildByType is a generic helper function for [Node.ChildByType].
func ChildByType[T Node](k Node, embeds bool, startIndex ...int) T {
	var n T
	v, _ := k.ChildByType(n.NodeType(), embeds, startIndex...).(T)
	return v
}

// IsRoot tests whether the given node is the root node in its tree.
func IsRoot(n Node) bool {
	return n.This() == nil || n.Parent() == nil || n.Parent().This() == nil
}

// Root returns the root node of the given node's tree.
func Root(n Node) Node {
	if IsRoot(n) {
		return n.This()
	}
	return Root(n.Parent())
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
// identifier.  If the node already has a name, it adds this, otherwise
// it uses the type name of the node plus the unique id.
func SetUniqueName(n Node) {
	pn := n.Parent()
	if pn == nil {
		return
	}
	c := pn.AsTreeNode().numLifetimeChildren
	id := "-" + strconv.FormatUint(c-1, 10) // must subtract 1 so we start at 0
	if n.Name() == "" {
		n.SetName(n.NodeType().IDName + id)
	} else {
		n.SetName(n.Name() + id)
	}
}
