// Copyright (c) 2018, Randall C. O'Reilly. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package ki

import (
	"reflect"

	"github.com/rcoreilly/goki/ki/atomctr"
	"github.com/rcoreilly/goki/ki/kit"
)

// bit flags for efficient core state of nodes -- see bitflag package for
// using these ordinal values to manipulate bit flag field
type Flags int32

const (
	// this node is a field in its parent node, not a child in children
	IsField Flags = iota
	// following flags record what happened to a given node since the last
	// Update signal -- these are cleared at first UpdateStart and valid after
	// UpdateEnd -- these should be coordinated with NodeSignals in signal.go

	// node was added to new parent
	NodeAdded
	// node was copied from other node
	NodeCopied
	// node was moved in the tree, or to a new tree
	NodeMoved
	// this node has been deleted
	NodeDeleted
	// this node has been destroyed -- do not trigger any more update signals on it
	NodeDestroyed
	// one or more new children were added to the node
	ChildAdded
	// one or more children were moved within the node
	ChildMoved
	// one or more children were deleted from the node
	ChildDeleted
	// all children were deleted
	ChildrenDeleted
	// total number of flags used by base Ki Node -- can extend from here up to 64 bits
	FlagsN

	// Mask for all the update flags -- destroyed is excluded b/c otherwise it would get cleared
	UpdateFlagsMask = (1 << uint32(NodeAdded)) | (1 << uint32(NodeCopied)) | (1 << uint32(NodeMoved)) | (1 << uint32(NodeDeleted)) | (1 << uint32(ChildAdded)) | (1 << uint32(ChildMoved)) | (1 << uint32(ChildDeleted)) | (1 << uint32(ChildrenDeleted))
)

//go:generate stringer -type=Flags

var KiT_Flags = kit.Enums.AddEnum(FlagsN, true, nil) // true = bitflags

/*
The Ki interface provides the core functionality for the GoKi tree -- insipred by Qt QObject in specific and every other Tree everywhere in general.

NOTE: The inability to have a field and a method of the same name makes it so you either have to use private fields in a struct that implements this interface (lowercase) or we have to use different names in the struct vs. interface.  We want to export and use the direct fields, which are easy to use, so we have different synonyms.

Other key issues with the Ki design / Go:
* All interfaces are implicitly pointers: this is why you have to pass args with & address of
*/
type Ki interface {
	// Initialize the node -- automatically called during Add/Insert Child -- sets the This pointer for this node as a Ki interface (pass pointer to node as this arg) -- Go cannot always access the true underlying type for structs using embedded Ki objects (when these objs are receivers to methods) so we need a This interface pointer that guarantees access to the Ki interface in a way that always reveals the underlying type (e.g., in reflect calls).  Calls Init on Ki fields within struct, sets their names to the field name, and sets us as their parent.
	Init(this Ki)

	// init this node and set its name -- used for root nodes which don't otherwise have their This pointer set (typically happens in Add, Insert Child)
	InitName(this Ki, name string)

	// check that the this pointer is set and issue a warning to log if not -- returns error if not set
	ThisCheck() error

	// Type returns the underlying struct type of this node (reflect.TypeOf(This).Elem())
	Type() reflect.Type

	// IsType tests whether This underlying struct object is of the given type(s) -- Go does not support a notion of inheritance, so this must be an exact match to the type
	IsType(t ...reflect.Type) bool

	// Parent (Node.Par) of this Ki -- Ki has strict one-parent, no-cycles structure -- see SetParent
	Parent() Ki

	// child at index -- supports negative indexes to access from end of slice -- only errors if there are no children in list
	Child(idx int) (Ki, error)

	// list of children (Node.Kids) -- can modify directly (e.g., sort, reorder) but add / remove should use existing methods to ensure proper tracking
	Children() Slice

	// Name (Node.Nm) is a user-defined name of the object, for finding elements, generating paths, IO, etc -- allows generic GUI / Text / Path / etc representation of Trees
	Name() string

	// A name (Node.UniqueNm) that is guaranteed to be non-empty and unique within the children of this node, but starts with Name or parents name if Name is empty -- important for generating unique paths
	UniqueName() string

	// sets the name of this node, and its unique name based on this name, such that all names are unique within list of siblings of this node (somewhat expensive but important, unless you definitely know that the names are unique)
	SetName(name string)

	// just set the name and don't update the unique name -- only use if also
	// setting unique names in some other way
	SetNameRaw(name string)

	// sets the unique name of this node -- should generally only be used by UniquifyNames
	SetUniqueName(name string)

	// ensure all my children have unique, non-empty names -- duplicates are named sequentially _1, _2 etc, and empty names
	UniquifyNames()

	//////////////////////////////////////////////////////////////////////////
	//  Flags

	// the flags for this node -- use bitflag package to manipulate flags
	Flags() *int64

	// is this a field on a parent struct, as opposed to a child?
	IsField() bool

	// has this node just been deleted (within last update cycle?)
	IsDeleted() bool

	// is this node destroyed?
	IsDestroyed() bool

	//////////////////////////////////////////////////////////////////////////
	//  Property interface with inheritance -- nodes can inherit props from parents

	// Properties (Node.Props) tell GUI or other frameworks operating on Trees about special features of each node -- functions below support inheritance up Tree -- see type.go for convenience methods for converting interface{} to standard types
	Properties() map[string]interface{}

	// Set given property key to value val -- initializes property map if nil
	SetProp(key string, val interface{})

	// Get property value from key -- if inherit, then check all parents too -- if typ then check property on type as well
	Prop(key string, inherit, typ bool) interface{}

	// Delete property key, safely
	DeleteProp(key string)

	// Delete all properties on this node -- just makes a new Props map -- can specify the capacity of the new map (0 is ok -- always grows automatically anyway)
	DeleteAllProps(cap int)

	// copy our properties from another node -- if deep then does a deep copy -- otherwise copied map just points to same values in the original map (and we don't reset our map first -- call DeleteAllProps to do that -- deep copy uses gob encode / decode -- usually not needed
	CopyPropsFrom(from Ki, deep bool) error

	//////////////////////////////////////////////////////////////////////////
	//  Parent / Child Functionality

	// just sets parent of node (and inherits update count from it, to keep consistent) -- does NOT remove from existing parent -- use Add / Insert / Delete Child functions properly move or delete nodes
	SetParent(parent Ki)

	// test if this node is the root node -- checks Parent = nil
	IsRoot() bool

	// get the root object of this tree
	Root() Ki

	// does this node have children (i.e., non-terminal)
	HasChildren() bool

	// set the ChildType to create using *NewChild routines, and for the gui -- ensures that it is a Ki type, and errors if not
	SetChildType(t reflect.Type) error

	// add a new child at end of children list -- if child is in an existing tree, it is removed from that parent, and a NodeMoved signal is emitted for the child
	AddChild(kid Ki) error

	// add a new child at given position in children list -- if child is in an existing tree, it is removed from that parent, and a NodeMoved signal is emitted for the child
	InsertChild(kid Ki, at int) error

	// add a new child at end of children list, and give it a name -- important to set name after adding, to ensure that UniqueNames are indeed unique
	AddChildNamed(kid Ki, name string) error

	// add a new child at given position in children list, and give it a name -- important to set name after adding, to ensure that UniqueNames are indeed unique
	InsertChildNamed(kid Ki, at int, name string) error

	// add a child at given position in children list, and give it a name, using SetNameRaw and SetUniqueName for the name -- only when names are known to be unique (faster)
	InsertChildNamedUnique(kid Ki, at int, name string) error

	// create a new child of given type -- if nil, uses ChildType, then This type
	MakeNew(typ reflect.Type) Ki

	// create a new child of given type -- if nil, uses ChildType, then This type -- and add at end of children list
	AddNewChild(typ reflect.Type) Ki

	// create a new child of given type -- if nil, uses ChildType, then This type -- and add at given position in children list
	InsertNewChild(typ reflect.Type, at int) Ki

	// create a new child of given type -- if nil, uses ChildType, then This type -- and add at end of children list, and give it a name
	AddNewChildNamed(typ reflect.Type, name string) Ki

	// create a new child of given type -- if nil, uses ChildType, then This type -- and add at given position in children list, and give it a name
	InsertNewChildNamed(typ reflect.Type, at int, name string) Ki

	// add a new child at given position in children list, and give it a name, using SetNameRaw and SetUniqueName for the name -- only when names are known to be unique (faster)
	InsertNewChildNamedUnique(typ reflect.Type, at int, name string) Ki

	// move child from one position to another in the list of children (see also Slice method)
	MoveChild(from, to int) error

	// configure children according to given list of type-and-name's --
	// attempts to have minimal impact relative to existing items that fit the
	// type and name constraints (they are moved into the corresponding
	// positions), and any extra children are removed, and new ones added, to
	// match the specified config.  If uniqNm, then names represent
	// UniqueNames (this results in Name == UniqueName for created children)
	ConfigChildren(config kit.TypeAndNameList, uniqNm bool)

	// find index of child based on match function (true for find, false for not) -- start_idx arg allows for optimized bidirectional find if you have an idea where it might be -- can be key speedup for large lists
	FindChildIndexByFun(start_idx int, match func(ki Ki) bool) int

	// find index of child -- start_idx arg allows for optimized bidirectional find if you have an idea where it might be -- can be key speedup for large lists
	FindChildIndex(kid Ki, start_idx int) int

	// find index of child from name -- start_idx arg allows for optimized bidirectional find if you have an idea where it might be -- can be key speedup for large lists
	FindChildIndexByName(name string, start_idx int) int

	// find index of child from unique name -- start_idx arg allows for optimized bidirectional find if you have an idea where it might be -- can be key speedup for large lists
	FindChildIndexByUniqueName(name string, start_idx int) int

	// find index of child by type (any of types given)
	FindChildIndexByType(t ...reflect.Type) int

	// find child from name -- start_idx arg allows for optimized find if you have an idea where it might be -- can be key speedup for large lists
	FindChildByName(name string, start_idx int) Ki

	// find child from type (any of types given) -- returns nil if not found
	FindChildByType(t ...reflect.Type) Ki

	// find parent by name -- returns nil if not found
	FindParentByName(name string) Ki

	// find parent by type (any of types given) -- returns nil if not found
	FindParentByType(t ...reflect.Type) Ki

	// find field Ki element by name -- returns nil if not found
	FindFieldByName(name string) Ki

	// Delete child at index -- if child's parent = this node, then will call SetParent(nil), so to transfer to another list, set new parent first -- destroy will add removed child to deleted list, to be destroyed later -- otherwise child remains intact but parent is nil -- could be inserted elsewhere
	DeleteChildAtIndex(idx int, destroy bool)

	// Delete child node -- if child's parent = this node, then will call SetParent(nil), so to transfer to another list, set new parent first -- destroy will add removed child to deleted list, to be destroyed later -- otherwise child remains intact but parent is nil -- could be inserted elsewhere
	DeleteChild(child Ki, destroy bool)

	// Delete child node by name -- returns child -- if child's parent = this node, then will call SetParent(nil), so to transfer to another list, set new parent first -- destroy will add removed child to deleted list, to be destroyed later -- otherwise child remains intact but parent is nil -- could be inserted elsewhere
	DeleteChildByName(name string, destroy bool) Ki

	// Delete all children nodes -- destroy will add removed children to deleted list, to be destroyed later -- otherwise children remain intact but parent is nil -- could be inserted elsewhere, but you better have kept a slice of them before calling this
	DeleteChildren(destroy bool)

	// Delete this node from its parent children list-- destroy will add removed child to deleted list, to be destroyed later -- otherwise child remains intact but parent is nil -- could be inserted elsewhere
	DeleteMe(destroy bool)

	// second-pass actually delete all previously-removed children: causes them to remove all their children and then destroy them
	DestroyDeleted()

	// recursively call DestroyDeleted on all nodes under this one -- called automatically when UpdateEnd reaches 0 Updating count and the Update signal is sent
	DestroyAllDeleted()

	// call DisconnectAll and remove all children and their childrens-children, etc
	Destroy()

	//////////////////////////////////////////////////////////////////////////
	//  Tree walking and Paths
	//   note: always put functions last -- looks better for inline functions

	// call function on all Ki fields within this node
	FunFields(level int, data interface{}, fun Fun)

	// concurrent go function call function on all Ki fields within this node
	GoFunFields(level int, data interface{}, fun Fun)

	// call function on given node and all the way up to its parents, and so on -- sequentially all in current go routine (generally necessary for going up, which is typicaly quite fast anyway) -- level is incremented after each step (starts at 0, goes up), and passed to function -- returns false if fun aborts with false, else true
	FunUp(level int, data interface{}, fun Fun) bool

	// call function on parent of node and all the way up to its parents, and so on -- sequentially all in current go routine (generally necessary for going up, which is typicaly quite fast anyway) -- level is incremented after each step (starts at 0, goes up), and passed to function -- returns false if fun aborts with false, else true
	FunUpParent(level int, data interface{}, fun Fun) bool

	// call fun on this node (MeFirst) and then call FunDownMeFirst on all the children -- sequentially all in current go routine -- level var is incremented before calling children -- if fun returns false then any further traversal of that branch of the tree is aborted, but other branches continue -- i.e., if fun on current node returns false, then returns false and children are not processed further -- this is the fastest, most natural form of traversal
	FunDownMeFirst(level int, data interface{}, fun Fun) bool

	// call FunDownDepthFirst on all children, then call fun on this node -- sequentially all in current go routine -- level var is incremented before calling children -- runs doChildTestFun on each child first to determine if it should process that child, and if that returns true, then it calls FunDownDepthFirst on that child
	FunDownDepthFirst(level int, data interface{}, doChildTestFun Fun, fun Fun)

	// call fun on all children, then call FunDownBreadthFirst on all the children -- does NOT call on first node where method is first called -- level var is incremented before calling chlidren -- if fun returns false then any further traversal of that branch of the tree is aborted, but other branches can continue
	FunDownBreadthFirst(level int, data interface{}, fun Fun)

	// concurrent go function on given node and all the way down to its children, and so on -- does not wait for completion of the go routines -- returns immediately
	GoFunDown(level int, data interface{}, fun Fun)

	// concurrent go function on given node and all the way down to its children, and so on -- does wait for the completion of the go routines before returning
	GoFunDownWait(level int, data interface{}, fun Fun)

	// call fun on previous node in the tree (previous child in my siblings, then parent, and so on)
	FunPrev(level int, data interface{}, fun Fun) bool

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
	UpdateCtr() *atomctr.Ctr

	// call this when starting to modify the tree (state or structure) -- increments an atomic update counter and automatically calls start update on all children -- can be called multiple times at multiple levels
	UpdateStart()

	// call this when done updating -- decrements update counter and emits NodeSignalUpdated when counter goes to 0, only if parent is not current updating (i.e., this is the highest-level node that finished updating) -- see also UpdateEndAll
	UpdateEnd()

	// call this when done updating -- decrements update counter and emits NodeSignalUpdated when counter goes to 0 for ALL nodes that might have updated, even if my parent node is still updating -- this is less typically used
	UpdateEndAll()

	// reset update counters to 0 -- in case they are out-of-sync due to more
	// complex tree maninpulations -- only call at a known point of
	// non-updating..
	UpdateReset()

	// disconnect node -- reset all ptrs to nil, and DisconnectAll() signals
	// -- e.g., for freeing up all connnections so node can be destroyed and
	// making GC easier
	Disconnect()

	// disconnect all the way from me down the tree
	DisconnectAll()

	//////////////////////////////////////////////////////////////////////////
	//  Deep Copy of Trees

	// The Ki copy function recreates the entire tree in the copy, duplicating
	// children etc.  It is very efficient by using the ConfigChildren method
	// which attempts to preserve any existing nodes in the destination if
	// they have the same name and type -- so a copy from a source to a target
	// that only differ minimally will be minimally destructive.  Only copies
	// to same types are supported.  Pointers (Ptr) are copied by saving the
	// current UniquePath and then SetPtrsFmPaths is called -- no other Ki
	// point.  Signal connections are NOT copied (todo: revisit)a.  No other
	// Ki pointers are copied, and the field tag copy:"-" can be added for any
	// other fields that should not be copied (unexported, lower-case fields
	// are not copyable).
	//
	// When nodes are copied from one place to another within the same overall
	// tree, paths are updated so that pointers to items within the copied
	// sub-tree are updated to the new location there (i.e., the path to the
	// old loation is replaced with that of the new destination location),
	// whereas paths outside of the copied location are not changed and point
	// as before.  See also MoveTo function for moving nodes to other parts of
	// the tree.  Sequence of functions is: GetPtrPaths on from, CopyFromRaw,
	// UpdtPtrPaths, then SetPtrsFmPaths
	CopyFrom(from Ki) error

	// clone (create and return a deep copy) of the tree from this node down.
	// Any pointers within the cloned tree will correctly point within the new
	// cloned tree (see Copy info)
	Clone() Ki

	// raw copy that just does the deep copy and doesn't do anything with pointers
	CopyFromRaw(from Ki) error

	// get all Ptr paths -- walk the tree down from current node and call GetPath on all Ptr fields -- this is called prior to copying / moving
	GetPtrPaths()

	// update Ptr (and Signal) paths replacing any occurrence of oldPath with
	// newPath, optionally only at the start of the path (typically true) --
	// for all nodes down from this one
	UpdatePtrPaths(oldPath, newPath string, startOnly bool)

	// walk the tree down from current node and call FindPtrFromPath on all Ptr fields found -- called after Copy, Unmarshal* to recover pointers after entire structure is in place -- see UnmarshalPost
	SetPtrsFmPaths()

	//////////////////////////////////////////////////////////////////////////
	//  IO: Marshal / Unmarshal support -- see also Slice, Ptr

	// save the tree to a JSON-encoded byte string -- wraps MarshalJSON
	SaveJSON(indent bool) ([]byte, error)

	// load the tree from a JSON-encoded byte string -- wraps UnmarshalJSON and calls UnmarshalPost
	LoadJSON(b []byte) error

	// save the tree to an XML-encoded byte string
	SaveXML(indent bool) ([]byte, error)

	// load the tree from an XML-encoded byte string
	LoadXML(b []byte) error

	// walk the tree down from current node and call SetParent on all children -- needed after JSON Unmarshal, etc
	ParentAllChildren()

	// this must be called after an Unmarshal -- calls SetPtrsFmPaths and ParentAllChildren -- due to inability to reflect into receiver types, cannot do it automatically unfortunately
	UnmarshalPost()
}

// see node.go for struct implementing this interface

// IMPORTANT: all types must initialize entry in package kit Types Registry:
// var KiT_TypeName = kit.Types.AddType(&TypeName{})

// function to call on ki objects walking the tree -- return bool = false means don't continue processing this branch of the tree, but other branches can continue
type Fun func(ki Ki, level int, data interface{}) bool

// a Ki reflect.Type, suitable for checking for Type.Implements
func KiType() reflect.Type {
	return reflect.TypeOf((*Ki)(nil)).Elem()
}
