// Copyright (c) 2018, Randall C. O'Reilly. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

/*
	Package Ki provides the base element of GoKi Trees: Ki = Tree in Japanese, and "Key" in English -- powerful tree structures supporting scenegraphs, programs, parsing, etc.

	The Node struct that implements the Ki interface, which
	can be used as an embedded type (or a struct field) in other structs to provide
	core tree functionality, including:
		* Parent / Child Tree structure -- each Node can ONLY have one parent
		* Paths for locating Nodes within the hierarchy -- key for many use-cases, including IO for pointers
		* Apply a function across nodes up or down a tree -- very flexible for tree walking
		* Generalized I/O -- can Save and Load the Tree as JSON, XML, etc -- including pointers which are saved using paths and automatically cached-out after loading
		* Event sending and receiving between Nodes (simlar to Qt Signals / Slots)
		* Robust updating state -- wrap updates in UpdateStart / End, and signals are blocked until the final end, at which point an update signal is sent -- works across levels
		* Properties (as a string-keyed map) with property inheritance -- css anyone!?
*/
package ki

import (
	"reflect"
)

// flags for basic Ki status -- not using yet..
// type KiFlags int32

// const (
// 	KiFlagsEmpty KiFlags = 0
// 	KiDirty      KiFlags = 1 << iota
// 	KiDeleted    KiFlags = 1 << iota
// )

/*
The Ki interface provides the core functionality for the GoKi tree -- insipred by Qt QObject in specific and every other Tree everywhere in general.

NOTE: The inability to have a field and a method of the same name makes it so you either have to use private fields in a struct that implements this interface (lowercase) or we need to use Ki prefix for basic items here so your fields can be more normal looking.  Assuming more regular access to fields of the struct than those in the interface.

Other key issues with the Ki design / Go:
* All interfaces are implicitly pointers: this is why you have to pass args with & address of
*/
type Ki interface {
	// unfortunately, Go cannot always access the true underlying type for structs using embedded Ki objects (when these objs are receivers to methods) so we need a this pointer that guarantees access to the Ki interface in a way that always reveals the underlying type (e.g., in reflect calls)
	ThisKi() Ki

	// Set the this -- done automatically in AddChild and InsertChild methods
	SetThis(ki Ki)

	// check that the this pointer is set and issue a warning to log if not -- returns error if not set
	ThisCheck() error

	// Ki has strict one-parent, no-cycles structure -- see SetParent
	KiParent() Ki

	// get child at index, does range checking to avoid slice panic
	KiChild(idx int) (Ki, error)

	// get list of children -- can modify directly (e.g., sort, reorder) but add / remove should use existing methods to ensure proper tracking
	KiChildren() KiSlice

	// These allow generic GUI / Text / Path / etc representation of Trees
	// The user-defined name of the object, for finding elements, generating paths, io, etc
	KiName() string

	// A name that is guaranteed to be non-empty and unique within the children of this node, but starts with KiName or parents name if KiName is empty -- important for generating unique paths
	KiUniqueName() string

	// sets the name of this node, and its unique name based on this name, such that all names are unique within list of siblings of this node
	SetName(name string)

	// sets the unique name of this node -- should generally only be used by UniquifyNames
	SetUniqueName(name string)

	// ensure all my children have unique, non-empty names -- duplicates are named sequentially _1, _2 etc, and empty names
	UniquifyNames()

	// Properties tell GUI or other frameworks operating on Trees about special features of each node -- functions below support inheritance up Tree
	KiProperties() map[string]interface{}

	//////////////////////////////////////////////////////////////////////////
	//  Property interface with inheritance -- nodes can inherit props from parents

	// Set given property key to value val -- initializes property map if nil
	SetProp(key string, val interface{})

	// Get property value from key, safely -- if inherit, then check all parents too
	Prop(key string, inherit bool) interface{}

	// Get property value from key, safely -- if inherit, then check all parents too -- as a bool -- false if not set or false -- error if key exists but is of a different type
	PropBool(key string, inherit bool) (bool, error)

	// Get property value from key, safely -- if inherit, then check all parents too -- as an int -- 0 if not set -- error if key exists but is of a different type
	PropInt(key string, inherit bool) (int, error)

	// Get property value from key, safely -- if inherit, then check all parents too -- as a float -- 0 if not set -- error if key exists but is of a different type
	PropFloat64(key string, inherit bool) (float64, error)

	// Get property value from key, safely -- if inherit, then check all parents too -- as a string -- "" if not set -- error if key exists but is of a different type
	PropString(key string, inherit bool) (string, error)

	// Delete property key, safely
	DeleteProp(key string)

	//////////////////////////////////////////////////////////////////////////
	//  Parent / Child Functionality

	// set parent of node -- if parent is already set, then removes from that parent first -- nodes can ONLY have one parent -- only for true Tree structures, not DAG's or other such graphs that do not enforce a strict single-parent relationship
	SetParent(parent Ki)

	// mark this node as the root node of the tree by adding "root" = true to Properties and calling SetThis which is why the arg is required -- essential to run this -- e.g., for properly triggering updates of pointers after JSON Unmarshal for example
	SetRoot(this Ki)

	// test if this node is the root node -- checks Properties for "root" bool = true
	IsRoot() bool

	// get the root object of this tree
	Root() Ki

	// set the ChildType to create using *NewChild routines, and for the gui -- ensures that it is a Ki type, and errors if not
	SetChildType(t reflect.Type) error

	// emit SignalChildAdded on NodeSignal -- only if not currently updating
	EmitChildAddedSignal(kid Ki)

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

	// find index of child -- start_idx arg allows for optimized bidirectional find if you have an idea where it might be -- can be key speedup for large lists
	FindChildIndex(kid Ki, start_idx int) int

	// find index of child from name -- start_idx arg allows for optimized bidirectional find if you have an idea where it might be -- can be key speedup for large lists
	FindChildIndexByName(name string, start_idx int) int

	// find index of child from unique name -- start_idx arg allows for optimized bidirectional find if you have an idea where it might be -- can be key speedup for large lists
	FindChildIndexByUniqueName(name string, start_idx int) int

	// find index of child by type -- start_idx arg allows for optimized bidirectional find if you have an idea where it might be -- can be key speedup for large lists
	FindChildIndexByType(T reflect.Type, start_idx int) int

	// find child from name -- start_idx arg allows for optimized find if you have an idea where it might be -- can be key speedup for large lists
	FindChildByName(name string, start_idx int) Ki

	// emit SignalChildDeleted on NodeSignal -- only if not currently updating
	EmitChildDeletedSignal(kid Ki)

	// Delete child at index -- destroy will add removed child to deleted list, to be destroyed later -- otherwise child remains intact but parent is nil -- could be inserted elsewhere
	DeleteChildAtIndex(idx int, destroy bool)

	// Delete child node -- destroy will add removed child to deleted list, to be destroyed later -- otherwise child remains intact but parent is nil -- could be inserted elsewhere
	DeleteChild(child Ki, destroy bool)

	// Delete child node by name -- returns child -- destroy will add removed child to deleted list, to be destroyed later -- otherwise child remains intact but parent is nil -- could be inserted elsewhere
	DeleteChildByName(name string, destroy bool) Ki

	// emit SignalChildrenDeleted on NodeSignal -- only if not currently updating
	EmitChildrenDeletedSignal()

	// Delete all children nodes -- destroy will add removed children to deleted list, to be destroyed later -- otherwise children remain intact but parent is nil -- could be inserted elsewhere, but you better have kept a slice of them before calling this
	DeleteChildren(destroy bool)

	// second-pass actually delete all previously-removed children: causes them to remove all their children and then destroy them
	DestroyDeleted()

	// remove all children and their childrens-children, etc
	DestroyKi()

	// is this a terminal node in the tree?  i.e., has no children
	IsLeaf() bool

	// does this node have children (i.e., non-terminal)
	HasChildren() bool

	//////////////////////////////////////////////////////////////////////////
	//  Tree walking and Paths
	//   note: always put functions last -- looks better for inline functions

	// call function on given node and all the way up to its parents, and so on -- sequentially all in current go routine (generally necessary for going up, which is typicaly quite fast anyway)
	FunUp(data interface{}, fun KiFun)

	// call function on given node and all the way down to its children, and so on -- sequentially all in current go routine -- does depth-first "natural" order: calls on node, then calls FunDown on all children
	FunDown(data interface{}, fun KiFun)

	// call function on children in a breadth-first manner -- calls on all the children, and then calls FunDownBreadthFirst on all the children -- does NOT call on first node where method is first called!
	FunDownBreadthFirst(data interface{}, fun KiFun)

	// concurrent go function on given node and all the way down to its children, and so on -- does not wait for completion of the go routines -- returns immediately
	GoFunDown(data interface{}, fun KiFun)

	// concurrent go function on given node and all the way down to its children, and so on -- does wait for the completion of the go routines before returning
	GoFunDownWait(data interface{}, fun KiFun)

	// report path to this node, all the way up to top-level parent
	Path() string

	// report path to this node using unique names, all the way up to top-level parent
	PathUnique() string

	// find Ki object at given unique path
	FindPathUnique(path string) Ki

	//////////////////////////////////////////////////////////////////////////
	//  State update signaling -- automatically consolidates all changes across
	//   levels so there is only one update at end (optionally per node or only
	//   at highest level)
	//   All modification starts with UpdateStart() and ends with UpdateEnd()

	// the main signal for this node that is used for update, child signals
	NodeSignal() *Signal

	// the update counter for this node -- uses atomic counter for thread safety
	UpdateCtr() *AtomCtr

	// call this when starting to modify the tree (state or structure) -- increments an atomic update counter and automatically calls start update on all children -- can be called multiple times at multiple levels
	UpdateStart()

	// call this when done updating -- decrements update counter and emits SignalNodeUpdated when counter goes to 0 -- if updtAll then always signal, else only if parent is not updating (i.e., this is the highest-level node that finished updating)
	UpdateEnd(updtAll bool)

	//////////////////////////////////////////////////////////////////////////
	//  IO: Marshal / Unmarshal support -- see also KiSlice, KiPtr

	// save the tree to a JSON-encoded byte string -- wraps MarshalJSON
	SaveJSON(indent bool) ([]byte, error)

	// load the tree from a JSON-encoded byte string -- wraps UnmarshalJSON and calls UnmarshalPost
	LoadJSON(b []byte) error

	// walk the tree down from current node and call FindPtrFromPath on all KiPtr fields found -- must be called after UnmarshalJSON to recover pointers after entire structure is in place -- see UnmarshalPost
	SetKiPtrsFmPaths()

	// walk the tree down from current node and call SetParent on all children -- needed after JSON Unmarshal, etc
	ParentAllChildren()

	// this must be called after an Unmarshal -- calls SetKiPtrsFmPaths and ParentAllChildren -- due to inability to reflect into receiver types, cannot do it automatically unfortunately
	UnmarshalPost()
}

// see node.go for struct implementing this interface

// IMPORTANT: all types must initialize entry in KiTypes Registry:
// var KtTypeName = KiTypes.AddType(&TypeName{})

// function to call on ki objects walking the tree -- bool rval = false means stop traversing
type KiFun func(node Ki, data interface{}) bool
