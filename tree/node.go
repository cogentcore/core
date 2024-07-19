// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package tree provides a powerful and extensible tree system,
// centered on the core [Node] interface.
package tree

//go:generate core generate
//go:generate core generate ./testdata

import (
	"cogentcore.org/core/base/plan"
)

// Node is an interface that all tree nodes satisfy. The core functionality
// of a tree node is defined on [NodeBase], and all higher-level tree types
// must embed it. This interface only contains the tree functionality that
// higher-level tree types may need to override. You can call [Node.AsTree]
// to get the [NodeBase] of a Node and access the core tree functionality.
// All values that implement [Node] are pointer values; see [NodeValue]
// for an interface for non-pointer values.
type Node interface {

	// AsTree returns the [NodeBase] of this Node. Most core
	// tree functionality is implemented on [NodeBase].
	AsTree() *NodeBase

	// Init is called when the node is first initialized.
	// It is called before the node is added to the tree,
	// so it will not have any parents or siblings.
	// It will be called only once in the lifetime of the node.
	// It does nothing by default, but it can be implemented
	// by higher-level types that want to do something.
	// It is the main place that initialization steps should
	// be done, like adding Stylers, Makers, and event handlers
	// to widgets in Cogent Core.
	Init()

	// OnAdd is called when the node is added to a parent.
	// It will be called only once in the lifetime of the node,
	// unless the node is moved. It will not be called on root
	// nodes, as they are never added to a parent.
	// It does nothing by default, but it can be implemented
	// by higher-level types that want to do something.
	OnAdd()

	// Destroy recursively deletes and destroys the node, all of its children,
	// and all of its children's children, etc. Node types can implement this
	// to do additional necessary destruction; if they do, they should call
	// [NodeBase.Destroy] at the end of their implementation.
	Destroy()

	// NodeWalkDown is a method that nodes can implement to traverse additional nodes
	// like widget parts during [NodeBase.WalkDown]. It is called with the function passed
	// to [Node.WalkDown] after the function is called with the node itself.
	NodeWalkDown(fun func(n Node) bool)

	// CopyFieldsFrom copies the fields of the node from the given node.
	// By default, it is [NodeBase.CopyFieldsFrom], which automatically does
	// a deep copy of all of the fields of the node that do not a have a
	// `copier:"-"` struct tag. Node types should only implement a custom
	// CopyFieldsFrom method when they have fields that need special copying
	// logic that can not be automatically handled. All custom CopyFieldsFrom
	// methods should call [NodeBase.CopyFieldsFrom] first and then only do manual
	// handling of specific fields that can not be automatically copied. See
	// [cogentcore.org/core/core.WidgetBase.CopyFieldsFrom] for an example of a
	// custom CopyFieldsFrom method.
	CopyFieldsFrom(from Node)

	// This is necessary for tree planning to work.
	plan.Namer
}

// NodeValue is an interface that all non-pointer tree nodes satisfy.
// Pointer tree nodes satisfy [Node], not NodeValue. NodeValue and [Node]
// are mutually exclusive; a [Node] cannot be a NodeValue and vice versa.
// However, a pointer to a NodeValue type is guaranteed to be a [Node],
// and a non-pointer version of a [Node] type is guaranteed to be a NodeValue.
type NodeValue interface {

	// NodeValue should only be implemented by [NodeBase],
	// and it should not be called. It must be exported due
	// to a nuance with the way that [reflect.StructOf] works,
	// which results in panics with embedded structs that have
	// unexported non-pointer methods.
	NodeValue()
}

// NodeValue implements [NodeValue]. It should not be called.
func (nb NodeBase) NodeValue() {}
