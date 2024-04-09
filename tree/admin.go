// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package tree

import (
	"fmt"
	"log/slog"
	"reflect"
	"strconv"
	"sync/atomic"

	"cogentcore.org/core/gti"
)

// admin has infrastructure level code, outside of Node interface

// InitNode initializes the node -- automatically called during Add/Insert
// Child -- sets the This pointer for this node as a Ki interface (pass
// pointer to node as this arg) -- Go cannot always access the true
// underlying type for structs using embedded Ki objects (when these objs
// are receivers to methods) so we need a This interface pointer that
// guarantees access to the Ki interface in a way that always reveals the
// underlying type (e.g., in reflect calls).  Calls Init on Ki fields
// within struct, sets their names to the field name, and sets us as their
// parent.
func InitNode(this Node) {
	n := this.AsTreeNode()
	if n.Ths != this {
		n.Ths = this
		n.Ths.OnInit()
	}
}

// ThisCheck checks that the This pointer is set and issues a warning to
// log if not -- returns error if not set -- called when nodes are added
// and inserted.
func ThisCheck(k Node) error {
	if k.This() == nil {
		err := fmt.Errorf("tree.NodeBase %q ThisCheck: node has null 'this' pointer; must call Init or InitName on root nodes", k.Path())
		slog.Error(err.Error())
		return err
	}
	return nil
}

// SetParent just sets parent of node (and inherits update count from
// parent, to keep consistent).
// Assumes not already in a tree or anything.
func SetParent(kid Node, parent Node) {
	n := kid.AsTreeNode()
	n.Par = parent
	if parent != nil {
		pn := parent.AsTreeNode()
		c := atomic.AddUint64(&pn.NumLifetimeKids, 1)
		if kid.Name() == "" {
			kid.SetName(kid.NodeType().IDName + "-" + strconv.FormatUint(c-1, 10)) // must subtract 1 so we start at 0
		}
	}
	kid.This().OnAdd()
	n.WalkUpParent(func(k Node) bool {
		k.This().OnChildAdded(kid)
		return Continue
	})
}

// MoveToParent deletes given node from its current parent and adds it as a child
// of given new parent.  Parents could be in different trees or not.
func MoveToParent(kid Node, parent Node) {
	// TODO(kai/ki): implement MoveToParent
	// oldPar := kid.Parent()
	// if oldPar != nil {
	// 	oldPar.DeleteChild(kid, false)
	// }
	// parent.AddChild(kid)
}

// New adds a new child of the given the type
// with the given name to the given parent.
// If the name is unspecified, it defaults to the
// ID (kebab-case) name of the type, plus the
// [Node.NumLifetimeChildren] of its parent.
// It is a helper function that calls [Node.NewChild].
func New[T Node](parent Node, name ...string) T {
	var n T
	return parent.NewChild(n.NodeType(), name...).(T)
}

// NewRoot returns a new root node of the given the type
// with the given name. If the name is unspecified, it
// defaults to the ID (kebab-case) name of the type.
// It is a helper function that calls [Node.InitName].
func NewRoot[T Node](name ...string) T {
	var n T
	n = n.New().(T)
	n.InitName(n, name...)
	return n
}

// InsertNewChild is a generic helper function for [Node.InsertNewChild]
func InsertNewChild[T Node](parent Node, at int, name ...string) T {
	var n T
	return parent.InsertNewChild(n.NodeType(), at, name...).(T)
}

// ParentByType is a generic helper function for [Node.ParentByType]
func ParentByType[T Node](k Node, embeds bool) T {
	var n T
	v, _ := k.ParentByType(n.NodeType(), embeds).(T)
	return v
}

// ChildByType is a generic helper function for [Node.ChildByType]
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
func NewOfType(typ *gti.Type) Node {
	return typ.Instance.(Node).New()
}
