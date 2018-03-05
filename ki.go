// Copyright (c) 2018, Randall C. O'Reilly. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Ki is the base element of GoKi Trees
// Ki = Tree in Japanese, and "Key" in English
package ki

import (
	"reflect"
)

/*
The Ki provides the core functionality for the GoKi Tree functionality -- insipred by Qt QObject in specific and every other Tree everywhere in general -- provides core functionality:
* Parent / Child Tree structure -- each Node can ONLY have one parent
* Paths for locating Nodes within the hierarchy -- key for many use-cases, including IO for pointers
* Generalized I/O -- can Save and Load the Tree as JSON, XML, etc
* Event sending and receiving between Nodes (simlar to Qt Signals / Slots)

NOTE: The inability to have a field and a method of the same name makes it so you either have to use private fields in a struct that implements this interface (lowercase) or we need to use Ki prefix here so your fields can be more normal looking.  Assuming more regular access to fields of the struct than those in the interface.
*/

// function to call on ki objects walking the tree -- bool rval = false means stop traversing
type KiFun func(node Ki, data interface{}) bool

// flags for basic Ki status
type KiFlags int32

const (
	KiFlagsEmpty KiFlags = 0
	KiDirty      KiFlags = 1 << iota
	KiDeleted    KiFlags = 1 << iota
)

// counter flag for things that can be called redundantly
type KiCtr int64

type Ki interface {
	KiParent() Ki
	// get child at index, does range checking to avoid slice panic
	KiChild(idx int) (Ki, error)
	// get list of children -- this is a new temporary list of Ki's so any changes to it have no effect on structure -- use Ki methods to add / remove children -- underlying struct will have its own actual list -- can use (specific version of) InterfaceToStructPtr to recover underlying struct from each Ki child
	KiChildren() []Ki

	// These allow generic GUI / Text / Path / etc representation of Trees
	// The user-defined name of the object, for finding elements, generating paths, io, etc
	KiName() string
	// A name that is guaranteed to be non-empty and unique within the children of this node -- important for generating unique paths
	KiUniqueName() string
	// Properties tell GUI or other frameworks operating on Trees about special features of each node -- functions below support inheritance up Tree
	KiProperties() map[string]interface{}

	// sets the name of this node, and its unique name based on this name, such that all names are unique within list of siblings of this node
	SetName(name string)

	// sets the unique name of this node -- should generally only be used by UniquifyNames
	SetUniqueName(name string)

	// ensure all my children have unique, non-empty names -- duplicates are named sequentially _1, _2 etc, and empty names
	UniquifyNames()

	// set parent of node -- if parent is already set, then removes from that parent first -- nodes can ONLY have one parent -- only for true Tree structures, not DAG's or other such graphs that do not enforce a strict single-parent relationship
	SetParent(parent Ki)

	// set the ChildType to create using *NewChild routines, and for the gui -- ensures that it is a Ki type
	SetChildType(t reflect.Type) error

	// emit SignalChildAdded on NodeSignal
	EmitAddChildSignal(kid Ki)

	// add a new child at end of children list
	AddChild(kid Ki)

	// add a new child at given position in children list
	InsertChild(kid Ki, at int)

	// add a new child at end of children list, and give it a name -- important to set name after adding, to ensure that UniqueNames are indeed unique
	AddChildNamed(kid Ki, name string)

	// add a new child at given position in children list, and give it a name -- important to set name after adding, to ensure that UniqueNames are indeed unique
	InsertChildNamed(kid Ki, at int, name string)

	// create a new child of ChildType
	MakeNewChild() Ki

	// create a new child of ChildType and add at end of children list
	AddNewChild() Ki

	// create a new child of ChildType and add at given position in children list
	InsertNewChild(at int) Ki

	// create a new child of ChildType and add at end of children list, and give it a name
	AddNewChildNamed(name string) Ki

	// create a new child of ChildType and add at given position in children list, and give it a name
	InsertNewChildNamed(at int, name string) Ki

	// find index of child -- start_idx arg allows for optimized find if you have an idea where it might be -- can be key speedup for large lists
	FindChildIndex(kid Ki, start_idx int) int

	// find index of child from name -- start_idx arg allows for optimized find if you have an idea where it might be -- can be key speedup for large lists
	FindChildNameIndex(name string, start_idx int) int

	// find index of child from unique name -- start_idx arg allows for optimized find if you have an idea where it might be -- can be key speedup for large lists
	FindChildUniqueNameIndex(name string, start_idx int) int

	// find child from name -- start_idx arg allows for optimized find if you have an idea where it might be -- can be key speedup for large lists
	FindChildName(name string, start_idx int) Ki

	// Remove child at index -- destroy will add removed child to deleted list, to be destroyed later -- otherwise child remains intact but parent is nil -- could be inserted elsewhere
	RemoveChildIndex(idx int, destroy bool)

	// Remove child node -- destroy will add removed child to deleted list, to be destroyed later -- otherwise child remains intact but parent is nil -- could be inserted elsewhere
	RemoveChild(child Ki, destroy bool)

	// Remove child node by name -- returns child -- destroy will add removed child to deleted list, to be destroyed later -- otherwise child remains intact but parent is nil -- could be inserted elsewhere
	RemoveChildName(name string, destroy bool) Ki

	// Remove all children nodes -- destroy will add removed children to deleted list, to be destroyed later -- otherwise children remain intact but parent is nil -- could be inserted elsewhere, but you better have kept a slice of them before calling this
	RemoveAllChildren(destroy bool)

	// second-pass actually delete all previously-removed children: causes them to remove all their children and then destroy them
	DestroyDeleted()

	// remove all children and their childrens-children, etc
	DestroyKi()

	// is this a terminal node in the tree?  i.e., has no children
	IsLeaf() bool

	// is this the top of the tree (i.e., parent is nil)
	IsTop() bool

	// does this node have children (i.e., non-terminal)
	HasChildren() bool

	// report path to this node, all the way up to top-level parent
	Path() string

	// report path to this node using unique names, all the way up to top-level parent
	PathUnique() string

	// find Ki object at given unique path
	FindPathUnique(path string) Ki

	// call function on given node and all the way up to its parents, and so on..
	FunUp(fun KiFun, data interface{})

	// call function on given node and all the way down to its children, and so on..
	FunDown(fun KiFun, data interface{})

	// concurrent go function on given node and all the way down to its children, and so on..
	GoFunDown(fun KiFun, data interface{})

	// the main signal for this node that is used for update, child signals
	NodeSignal() *Signal

	// the update counter for this node
	UpdateCtr() *KiCtr

	// call this when starting to modify the tree (state or structure) -- increments an atomic update counter and automatically calls start update on all children -- can be called multiple times at multiple levels
	UpdateStart()

	// call this when done updating -- decrements update counter and emits SignalNodeUpdated when counter goes to 0 -- if updtall then always signal, else only if parent is not updating (i.e., this is the highest-level node that finished updating)
	UpdateEnd(updtall bool)
}

// see node.go for struct implementing this interface

type Kier interface {
	Ki() Ki
}

// IMPORTANT: all types must define Kier and initialize entry in KiTypes Registry:
// func (t *TypeName) Ki() Ki { return t }
// var KtTypeName = KiTypes.AddType(&TypeName{})
