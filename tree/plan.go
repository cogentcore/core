// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package tree

import (
	"log/slog"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"cogentcore.org/core/base/plan"
)

// Plan represents a plan for how the children of a [Node] should be configured.
// A Plan instance is passed to [NodeBase.Makers], which are responsible for
// configuring it. To add a child item to a plan, use [Add], [AddAt], or [AddNew].
// To add a child item maker to a [Node], use [AddChild] or [AddChildAt]. To extend
// an existing child item, use [AddInit] or [AddChildInit].
type Plan struct {

	// Children are the [PlanItem]s for the children.
	Children []*PlanItem

	// Parent is the parent [Node] that the Children are being added to.
	// It can be useful to access and update this item in some cases,
	// for example in MakeToolbar functions to also add an OverflowMenu
	// to the toolbar.
	Parent Node

	// EnforceEmpty is whether an empty plan results in the removal
	// of any children on the [Plan.Parent]. If there are [NodeBase.Makers]
	// defined then this is true by default; otherwise it is false.
	EnforceEmpty bool
}

// PlanItem represents a plan for how a child [Node] should be constructed and initialized.
// See [Plan] for more information.
type PlanItem struct {

	// Name is the name of the planned node.
	Name string

	// New returns a new node of the correct type for this child.
	New func() Node

	// Init is a slice of functions that are called once in sequential ascending order
	// after [PlanItem.New] to initialize the node for the first time.
	Init []func(n Node)
}

// Updater adds a new function to [NodeBase.Updaters], which are called in sequential
// descending (reverse) order in [NodeBase.RunUpdaters] to update the node.
func (nb *NodeBase) Updater(updater func()) {
	nb.Updaters = append(nb.Updaters, updater)
}

// Maker adds a new function to [NodeBase.Makers], which are called in sequential
// ascending order in [NodeBase.Make] to make the plan for how the node's children
// should be configured.
func (nb *NodeBase) Maker(maker func(p *Plan)) {
	nb.Makers = append(nb.Makers, maker)
}

// Make makes a plan for how the node's children should be structured.
// It does this by running [NodeBase.Makers] in sequential ascending order.
func (nb *NodeBase) Make(p *Plan) {
	if len(nb.Makers) > 0 { // only enforce empty if makers exist
		p.EnforceEmpty = true
	}
	for _, maker := range nb.Makers {
		maker(p)
	}
}

// UpdateFromMake updates the node using [NodeBase.Make].
func (nb *NodeBase) UpdateFromMake() {
	p := Plan{Parent: nb.This}
	nb.Make(&p)
	p.Update(nb)
}

// RunUpdaters runs the [NodeBase.Updaters] in sequential descending (reverse) order.
// It is called in [cogentcore.org/core/core.WidgetBase.UpdateWidget] and other places
// such as in xyz to update the node.
func (nb *NodeBase) RunUpdaters() {
	for i := len(nb.Updaters) - 1; i >= 0; i-- {
		nb.Updaters[i]()
	}
}

// Add adds a new [PlanItem] to the given [Plan] for a [Node] with
// the given function to initialize the node. The node is
// guaranteed to be added to its parent before the init function
// is called. The name of the node is automatically generated based
// on the file and line number of the calling function.
func Add[T Node](p *Plan, init func(n T)) {
	AddAt(p, autoPlanName(2), init)
}

// autoPlanName returns the dir-filename of [runtime.Caller](level),
// with all / . replaced to -, which is suitable as a unique name
// for a [PlanItem.Name].
func autoPlanName(level int) string {
	_, file, line, _ := runtime.Caller(level)
	name := filepath.Base(file)
	dir := filepath.Base(filepath.Dir(file))
	path := dir + "-" + name
	path = strings.ReplaceAll(strings.ReplaceAll(path, "/", "-"), ".", "-") + "-" + strconv.Itoa(line)
	return path
}

// AddAt adds a new [PlanItem] to the given [Plan] for a [Node] with
// the given name and function to initialize the node. The node
// is guaranteed to be added to its parent before the init function
// is called.
func AddAt[T Node](p *Plan, name string, init func(n T)) {
	p.Add(name, func() Node {
		return New[T]()
	}, func(n Node) {
		init(n.(T))
	})
}

// AddNew adds a new [PlanItem] to the given [Plan] for a [Node] with
// the given name, function for constructing the node, and function
// for initializing the node. The node is guaranteed to be added
// to its parent before the init function is called.
// It should only be called instead of [Add] and [AddAt] when the node
// must be made new, like when using [cogentcore.org/core/core.NewValue].
func AddNew[T Node](p *Plan, name string, new func() T, init func(n T)) {
	p.Add(name, func() Node {
		return new()
	}, func(n Node) {
		init(n.(T))
	})
}

// AddInit adds a new function for initializing the [Node] with the given name
// in the given [Plan]. The node must already exist in the plan; this is for
// extending an existing [PlanItem], not adding a new one. The node is guaranteed to
// be added to its parent before the init function is called. The init functions are
// called in sequential ascending order.
func AddInit[T Node](p *Plan, name string, init func(n T)) {
	for _, child := range p.Children {
		if child.Name == name {
			child.Init = append(child.Init, func(n Node) {
				init(n.(T))
			})
			return
		}
	}
	slog.Error("AddInit: child not found", "name", name)
}

// AddChild adds a new [NodeBase.Maker] to the the given parent [Node] that
// adds a [PlanItem] with the given init function using [Add]. In other words,
// this adds a maker that will add a child to the given parent.
func AddChild[T Node](parent Node, init func(n T)) {
	name := autoPlanName(2) // must get here to get correct name
	parent.AsTree().Maker(func(p *Plan) {
		AddAt(p, name, init)
	})
}

// AddChildAt adds a new [NodeBase.Maker] to the the given parent [Node] that
// adds a [PlanItem] with the given name and init function using [AddAt]. In other
// words, this adds a maker that will add a child to the given parent.
func AddChildAt[T Node](parent Node, name string, init func(n T)) {
	parent.AsTree().Maker(func(p *Plan) {
		AddAt(p, name, init)
	})
}

// AddChildInit adds a new [NodeBase.Maker] to the the given parent [Node] that
// adds a new function for initializing the node with the given name
// in the given [Plan]. The node must already exist in the plan; this is for
// extending an existing [PlanItem], not adding a new one. The node is guaranteed
// to be added to its parent before the init function is called. The init functions are
// called in sequential ascending order.
func AddChildInit[T Node](parent Node, name string, init func(n T)) {
	parent.AsTree().Maker(func(p *Plan) {
		AddInit(p, name, init)
	})
}

// Add adds a new [PlanItem with the given name and functions to the [Plan].
// It should typically not be called by end-user code; see the generic
// [Add], [AddAt], [AddNew], [AddChild], [AddChildAt], [AddInit], and [AddChildInit]
// functions instead.
func (p *Plan) Add(name string, new func() Node, init func(n Node)) {
	p.Children = append(p.Children, &PlanItem{Name: name, New: new, Init: []func(n Node){init}})
}

// Update updates the children of the given [Node] in accordance with the [Plan].
func (p *Plan) Update(n Node) {
	if len(p.Children) == 0 && !p.EnforceEmpty {
		return
	}
	plan.Update(&n.AsTree().Children, len(p.Children),
		func(i int) string {
			return p.Children[i].Name
		}, func(name string, i int) Node {
			item := p.Children[i]
			child := item.New()
			child.AsTree().SetName(item.Name)
			return child
		}, func(child Node, i int) {
			SetParent(child, n)
			for _, f := range p.Children[i].Init {
				f(child)
			}
		}, func(child Node) {
			child.Destroy()
		},
	)
}
