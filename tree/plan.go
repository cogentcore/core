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
	"cogentcore.org/core/base/profile"
)

// Plan represents a plan for how the children of a [Node] should be configured.
// A Plan instance is passed to [NodeBase.Makers], which are responsible for
// configuring it. To add a child item to a plan, use [Add], [AddAt], or [AddNew].
// To add a child item maker to a [Node], use [AddChild] or [AddChildAt]. To extend
// an existing child item, use [AddInit] or [AddChildInit].
type Plan struct {

	// Children are the [PlanItem]s for the children.
	Children []*PlanItem

	// EnforceEmpty is whether an empty plan results in the removal
	// of all children of the parent. If there are [NodeBase.Makers]
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

// Updater adds a new function to [NodeBase.Updaters.Normal], which are called in sequential
// descending (reverse) order in [NodeBase.RunUpdaters] to update the node.
func (nb *NodeBase) Updater(updater func()) {
	nb.Updaters.Normal = append(nb.Updaters.Normal, updater)
}

// FirstUpdater adds a new function to [NodeBase.Updaters.First], which are called in sequential
// descending (reverse) order in [NodeBase.RunUpdaters] to update the node.
func (nb *NodeBase) FirstUpdater(updater func()) {
	nb.Updaters.First = append(nb.Updaters.First, updater)
}

// FinalUpdater adds a new function to [NodeBase.Updaters.Final], which are called in sequential
// descending (reverse) order in [NodeBase.RunUpdaters] to update the node.
func (nb *NodeBase) FinalUpdater(updater func()) {
	nb.Updaters.Final = append(nb.Updaters.Final, updater)
}

// Maker adds a new function to [NodeBase.Makers.Normal], which are called in sequential
// ascending order in [NodeBase.Make] to make the plan for how the node's children
// should be configured.
func (nb *NodeBase) Maker(maker func(p *Plan)) {
	nb.Makers.Normal = append(nb.Makers.Normal, maker)
}

// FirstMaker adds a new function to [NodeBase.Makers.First], which are called in sequential
// ascending order in [NodeBase.Make] to make the plan for how the node's children
// should be configured.
func (nb *NodeBase) FirstMaker(maker func(p *Plan)) {
	nb.Makers.First = append(nb.Makers.First, maker)
}

// FinalMaker adds a new function to [NodeBase.Makers.Final], which are called in sequential
// ascending order in [NodeBase.Make] to make the plan for how the node's children
// should be configured.
func (nb *NodeBase) FinalMaker(maker func(p *Plan)) {
	nb.Makers.Final = append(nb.Makers.Final, maker)
}

// Make makes a plan for how the node's children should be structured.
// It does this by running [NodeBase.Makers] in sequential ascending order.
func (nb *NodeBase) Make(p *Plan) {
	// only enforce empty if makers exist
	if len(nb.Makers.First) > 0 || len(nb.Makers.Normal) > 0 || len(nb.Makers.Final) > 0 {
		p.EnforceEmpty = true
	}
	nb.Makers.Do(func(makers *[]func(p *Plan)) {
		for _, maker := range *makers {
			maker(p)
		}
	})
}

// UpdateFromMake updates the node using [NodeBase.Make].
func (nb *NodeBase) UpdateFromMake() {
	p := &Plan{}
	nb.Make(p)
	p.Update(nb)
}

// RunUpdaters runs the [NodeBase.Updaters] in sequential descending (reverse) order.
// It is called in [cogentcore.org/core/core.WidgetBase.UpdateWidget] and other places
// such as in xyz to update the node.
func (nb *NodeBase) RunUpdaters() {
	nb.Updaters.Do(func(updaters *[]func()) {
		for i := len(*updaters) - 1; i >= 0; i-- {
			(*updaters)[i]()
		}
	})
}

// Add adds a new [PlanItem] to the given [Plan] for a [Node] with
// the given function to initialize the node. The node is
// guaranteed to be added to its parent before the init function
// is called. The name of the node is automatically generated based
// on the file and line number of the calling function.
func Add[T NodeValue](p *Plan, init func(w *T)) { //yaegi:add
	AddAt(p, AutoPlanName(2), init)
}

// AutoPlanName returns the dir-filename of [runtime.Caller](level),
// with all / . replaced to -, which is suitable as a unique name
// for a [PlanItem.Name].
func AutoPlanName(level int) string {
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
func AddAt[T NodeValue](p *Plan, name string, init func(w *T)) { //yaegi:add
	p.Add(name, func() Node {
		return any(New[T]()).(Node)
	}, func(n Node) {
		init(any(n).(*T))
	})
}

// AddNew adds a new [PlanItem] to the given [Plan] for a [Node] with
// the given name, function for constructing the node, and function
// for initializing the node. The node is guaranteed to be added
// to its parent before the init function is called.
// It should only be called instead of [Add] and [AddAt] when the node
// must be made new, like when using [cogentcore.org/core/core.NewValue].
func AddNew[T Node](p *Plan, name string, new func() T, init func(w T)) { //yaegi:add
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
func AddInit[T NodeValue](p *Plan, name string, init func(w *T)) { //yaegi:add
	for _, child := range p.Children {
		if child.Name == name {
			child.Init = append(child.Init, func(n Node) {
				init(any(n).(*T))
			})
			return
		}
	}
	slog.Error("AddInit: child not found", "name", name)
}

// AddChild adds a new [NodeBase.Maker] to the the given parent [Node] that
// adds a [PlanItem] with the given init function using [Add]. In other words,
// this adds a maker that will add a child to the given parent.
func AddChild[T NodeValue](parent Node, init func(w *T)) { //yaegi:add
	name := AutoPlanName(2) // must get here to get correct name
	parent.AsTree().Maker(func(p *Plan) {
		AddAt(p, name, init)
	})
}

// AddChildAt adds a new [NodeBase.Maker] to the the given parent [Node] that
// adds a [PlanItem] with the given name and init function using [AddAt]. In other
// words, this adds a maker that will add a child to the given parent.
func AddChildAt[T NodeValue](parent Node, name string, init func(w *T)) { //yaegi:add
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
func AddChildInit[T NodeValue](parent Node, name string, init func(w *T)) { //yaegi:add
	parent.AsTree().Maker(func(p *Plan) {
		AddInit(p, name, init)
	})
}

// Add adds a new [PlanItem] with the given name and functions to the [Plan].
// It should typically not be called by end-user code; see the generic
// [Add], [AddAt], [AddNew], [AddChild], [AddChildAt], [AddInit], and [AddChildInit]
// functions instead.
func (p *Plan) Add(name string, new func() Node, init func(w Node)) {
	p.Children = append(p.Children, &PlanItem{Name: name, New: new, Init: []func(n Node){init}})
}

// Update updates the children of the given [Node] in accordance with the [Plan].
func (p *Plan) Update(n Node) {
	if len(p.Children) == 0 && !p.EnforceEmpty {
		return
	}
	pr := profile.Start("plan.Update")
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
	pr.End()
}
