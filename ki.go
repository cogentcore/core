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
	// the UpdateStart / End logic only applies to this node in isolation, not to its children -- useful for a parent node that has a different functional role than its children
	OnlySelfUpdate
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
	// a field was updated
	FieldUpdated
	// a property was set
	PropUpdated
	// total number of flags used by base Ki Node -- can extend from here up to 64 bits
	FlagsN

	// Mask for node updates
	NodeUpdateFlagsMask = (1 << uint32(NodeAdded)) | (1 << uint32(NodeCopied)) | (1 << uint32(NodeMoved))

	// Mask for child updates
	ChildUpdateFlagsMask = (1 << uint32(ChildAdded)) | (1 << uint32(ChildMoved)) | (1 << uint32(ChildDeleted)) | (1 << uint32(ChildrenDeleted))

	// Mask for structural changes update flags
	StruUpdateFlagsMask = NodeUpdateFlagsMask | ChildUpdateFlagsMask | (1 << uint32(NodeDeleted))

	// Mask for non-structural, value-only changes update flags
	ValUpdateFlagsMask = (1 << uint32(FieldUpdated)) | (1 << uint32(PropUpdated))

	// Mask for all the update flags -- destroyed is excluded b/c otherwise it would get cleared
	UpdateFlagsMask = StruUpdateFlagsMask | ValUpdateFlagsMask
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
	// Init initializes the node -- automatically called during Add/Insert Child -- sets the This pointer for this node as a Ki interface (pass pointer to node as this arg) -- Go cannot always access the true underlying type for structs using embedded Ki objects (when these objs are receivers to methods) so we need a This interface pointer that guarantees access to the Ki interface in a way that always reveals the underlying type (e.g., in reflect calls).  Calls Init on Ki fields within struct, sets their names to the field name, and sets us as their parent.
	Init(this Ki)

	// InitName initializes this node and set its name -- used for root nodes which don't otherwise have their This pointer set (typically happens in Add, Insert Child)
	InitName(this Ki, name string)

	// ThisCheck checks that the This pointer is set and issues a warning to log if not -- returns error if not set -- called when nodes are added and inserted
	ThisCheck() error

	// Type returns the underlying struct type of this node (reflect.TypeOf(This).Elem())
	Type() reflect.Type

	// TypeEmbeds tests whether this node is of the given type, or it embeds that type at any level of anonymous embedding -- use EmbeddedStruct to get the embedded struct of that type from this node
	TypeEmbeds(t reflect.Type) bool

	// EmbeddedStruct returns the embedded struct of given type from this node (or nil if it does not embed that type, or the type is not a Ki type -- see kit.EmbeddedStruct for a generic interface{} version
	EmbeddedStruct(t reflect.Type) Ki

	// Parent returns the parent of this Ki (Node.Par) -- Ki has strict one-parent, no-cycles structure -- see SetParent
	Parent() Ki

	// HasParent returns true if this node has a parent at any level above it that is the given node
	HasParent(par Ki) bool

	// Child returns the child at given index -- supports negative indexes to access from end of slice (-1 = last child, etc) -- returns nil if no children or index is invalid -- use IsValidIndex when unsure -- use Children()[idx] for fast direct access when index is known to be valid
	Child(idx int) Ki

	// Children returns the slice of children (Node.Kids) -- this can be modified directly (e.g., sort, reorder) but Add* / Delete* functions should be used to ensure proper tracking
	Children() Slice

	// IsValidIndex checks whether the given index is a valid index into children, within range of 0..len-1 -- see ki.Slice.ValidIndex for version that transforms negative numbers into indicies from end of slice, and has explicit error messages
	IsValidIndex(idx int) bool

	// Name returns the user-defined name of the object (Node.Nm), for finding elements, generating paths, IO, etc -- allows generic GUI / Text / Path / etc representation of Trees
	Name() string

	// UniqueName returns a name that is guaranteed to be non-empty and unique within the children of this node (Node.UniqueNm), but starts with Name or parents name if Name is empty -- important for generating unique paths to definitively locate a given node in the tree (see PathUnique, FindPathUnique)
	UniqueName() string

	// SetName sets the name of this node, and its unique name based on this name, such that all names are unique within list of siblings of this node (somewhat expensive but important, unless you definitely know that the names are unique)
	SetName(name string)

	// SetNameRaw just sets the name and doesn't update the unique name -- only use if also/ setting unique names in some other way that is guaranteed to be unique
	SetNameRaw(name string)

	// SetUniqueName sets the unique name of this node based on given name string -- does not do any further testing that the name is indeed unique -- should generally only be used by UniquifyNames
	SetUniqueName(name string)

	// UniquifyNames ensures all of my children have unique, non-empty names -- duplicates are named sequentially _1, _2 etc, and empty names get a name based on my name or my type name
	UniquifyNames()

	//////////////////////////////////////////////////////////////////////////
	//  Flags

	// Flag returns the bit flags for this node -- use bitflag package to manipulate flags -- see Flags type for standard values used in Ki Node -- can be extended from FlagsN up to 64 bit capacity
	Flags() *int64

	// IsField checks if this is a field on a parent struct (via IsField Flag), as opposed to a child in Children -- Ki nodes can be added as fields to structs and they are automatically parented and named with field name during Init function -- essentially they function as fixed children of the parent struct, and are automatically included in FuncDown* traversals, etc -- see also FunFields
	IsField() bool

	// OnlySelfUpdate checks if this node only applies UpdateStart / End logic to itself, not its children (which is the default) (via Flag of same name) -- useful for a parent node that has a different function than its children
	OnlySelfUpdate() bool

	// SetOnlySelfUpdate sets the OnlySelfUpdate flag -- see OnlySelfUpdate method and flag
	SetOnlySelfUpdate()

	// IsDeleted checks if this node has just been deleted (within last update cycle), indicated by the NodeDeleted flag which is set when the node is deleted, and is cleared at next UpdateStart call
	IsDeleted() bool

	// IsDestroyed checks if this node has been destroyed -- the NodeDestroyed flag is set at start of Destroy function -- the Signal Emit process checks for destroyed receiver nodes and removes connections to them automatically -- other places where pointers to potentially destroyed nodes may linger should also check this flag and reset those pointers
	IsDestroyed() bool

	//////////////////////////////////////////////////////////////////////////
	//  Property interface with inheritance -- nodes can inherit props from parents

	// Properties (Node.Props) tell the GoGi GUI or other frameworks operating on Trees about special features of each node -- functions below support inheritance up Tree -- see kit convert.go for robust convenience methods for converting interface{} values to standard types
	Properties() map[string]interface{}

	// SetProp sets given property key to value val -- initializes property map if nil
	SetProp(key string, val interface{})

	// SetPropUpdate sets given property key to value val, with update notification (wrapped in UpdateStart/End) so other nodes receiving update signals from this node can update to reflect these changes
	SetPropUpdate(key string, val interface{})

	// Prop gets property value from key -- if inherit, then checks all parents too -- if typ then checks property on type as well -- returns nil if not set
	Prop(key string, inherit, typ bool) interface{}

	// DeleteProp deletes property key, safely
	DeleteProp(key string)

	// DeleteAllProps deletes all properties on this node -- just makes a new Props map -- can specify the capacity of the new map (0 is ok -- always grows automatically anyway)
	DeleteAllProps(cap int)

	// CopyPropsFrom copies our properties from another node -- if deep then does a deep copy -- otherwise copied map just points to same values in the original map (and we don't reset our map first -- call DeleteAllProps to do that -- deep copy uses gob encode / decode -- usually not needed)
	CopyPropsFrom(from Ki, deep bool) error

	//////////////////////////////////////////////////////////////////////////
	//  Parent / Child Functionality

	// SetParent just sets parent of node (and inherits update count from parent, to keep consistent) -- does NOT remove from existing parent -- use Add / Insert / Delete Child functions properly move or delete nodes
	SetParent(parent Ki)

	// IsRoot tests if this node is the root node -- checks Parent = nil
	IsRoot() bool

	// Root returns the root object of this tree (the node with a nil parent)
	Root() Ki

	// FieldRoot returns the field root object for this node -- the node that owns the branch of the tree rooted in one of its fields -- the first non-Field parent node after the first Field parent node -- can be nil if no such thing exists for this node
	FieldRoot() Ki

	// HasChildren tests whether this node has children (i.e., non-terminal)
	HasChildren() bool

	// Index returns our index within our parent object -- caches the last value and uses that for an optimized search so subsequent calls are typically quite fast -- returns -1 if we don't have a parent
	Index() int

	// SetChildType sets the ChildType used as a default type for creating new children, and for the gui -- ensures that the type is a Ki type, and errors if not
	SetChildType(t reflect.Type) error

	// AddChild adds a new child at end of children list -- if child is in an existing tree, it is removed from that parent, and a NodeMoved signal is emitted for the child
	AddChild(kid Ki) error

	// InsertChild adds a new child at given position in children list -- if child is in an existing tree, it is removed from that parent, and a NodeMoved signal is emitted for the child
	InsertChild(kid Ki, at int) error

	// AddChildNamed adds a new child at end of children list, and gives it a name -- important to set name after adding, to ensure that UniqueNames are indeed unique
	AddChildNamed(kid Ki, name string) error

	// InsertChildNamed adds a new child at given position in children list, and gives it a name -- important to set name after adding, to ensure that UniqueNames are indeed unique
	InsertChildNamed(kid Ki, at int, name string) error

	// InsertChildNamedUnique adds a child at given position in children list, and give it a name, using SetNameRaw and SetUniqueName for the name -- only when names are known to be unique (faster)
	InsertChildNamedUnique(kid Ki, at int, name string) error

	// MakeNew creates a new child of given type -- if nil, uses ChildType, else uses the same type as this struct
	MakeNew(typ reflect.Type) Ki

	// AddNewChild creates a new child of given type -- if nil, uses ChildType, else type of this struct -- and add at end of children list
	AddNewChild(typ reflect.Type) Ki

	// InsertNewChild creates a new child of given type -- if nil, uses ChildType, else type of this struct -- and add at given position in children list
	InsertNewChild(typ reflect.Type, at int) Ki

	// AddNewChildNamed creates a new child of given type -- if nil, uses ChildType, else type of this struct -- and add at end of children list, and give it a name
	AddNewChildNamed(typ reflect.Type, name string) Ki

	// InsertNewChildNamed creates a new child of given type -- if nil, uses ChildType, else type of this struct -- and add at given position in children list, and give it a name
	InsertNewChildNamed(typ reflect.Type, at int, name string) Ki

	// InsertNewChildNamedUnique adds a new child at given position in children list, and gives it a name, using SetNameRaw and SetUniqueName for the name -- only when names are known to be unique (faster)
	InsertNewChildNamedUnique(typ reflect.Type, at int, name string) Ki

	// MoveChild moves child from one position to another in the list of children (see also corresponding Slice method)
	MoveChild(from, to int) error

	// ConfigChildren configures children according to given list of
	// type-and-name's -- attempts to have minimal impact relative to existing
	// items that fit the type and name constraints (they are moved into the
	// corresponding positions), and any extra children are removed, and new
	// ones added, to match the specified config.  If uniqNm, then names
	// represent UniqueNames (this results in Name == UniqueName for created
	// children).  Returns whether any changes were made
	ConfigChildren(config kit.TypeAndNameList, uniqNm bool) bool

	// ChildIndexByFunc returns index of child based on match function (true for match, false for not) -- startIdx arg allows for optimized bidirectional search if you have an idea where it might be -- can be key speedup for large lists
	ChildIndexByFunc(startIdx int, match func(ki Ki) bool) int

	// ChildIndex returns index of child -- startIdx arg allows for optimized bidirectional search if you have an idea where it might be -- can be key speedup for large lists
	ChildIndex(kid Ki, startIdx int) int

	// ChildIndexByName returns index of child from name -- startIdx arg allows for optimized bidirectional search if you have an idea where it might be -- can be key speedup for large lists
	ChildIndexByName(name string, startIdx int) int

	// ChildIndexByUniqueName returns index of child from unique name -- startIdx arg allows for optimized bidirectional search if you have an idea where it might be -- can be key speedup for large lists
	ChildIndexByUniqueName(name string, startIdx int) int

	// ChildIndexByType returns index of child by type -- if embeds is true, then it looks for any type that embeds the given type at any level of anonymous embedding -- startIdx arg allows for optimized bidirectional search if you have an idea where it might be -- can be key speedup for large lists
	ChildIndexByType(t reflect.Type, embeds bool, startIdx int) int

	// ChildByName returns child from name -- startIdx arg allows for optimized search if you have an idea where it might be -- can be key speedup for large lists
	ChildByName(name string, startIdx int) Ki

	// ChildByType returns child from type (any of types given) -- returns nil if not found -- if embeds is true, then it looks for any type that embeds the given type at any level of anonymous embedding -- startIdx arg allows for optimized bidirectional search if you have an idea where it might be -- can be key speedup for large lists
	ChildByType(t reflect.Type, embeds bool, startIdx int) Ki

	// ParentByName returns parent by name -- returns nil if not found
	ParentByName(name string) Ki

	// ParentByType returns parent by type (any of types given) -- returns nil if not found -- if embeds is true, then it looks for any type that embeds the given type at any level of anonymous embedding
	ParentByType(t reflect.Type, embeds bool) Ki

	// KiFieldByName returns field Ki element by name -- returns nil if not found
	KiFieldByName(name string) Ki

	// DeleteChildAtIndex deletes child at given index -- if child's parent = this node, then will call SetParent(nil), so to transfer to another list, set new parent first -- destroy will add removed child to deleted list, to be destroyed later -- otherwise child remains intact but parent is nil -- could be inserted elsewhere
	DeleteChildAtIndex(idx int, destroy bool)

	// DeletChild deletes child node -- if child's parent = this node, then will call SetParent(nil), so to transfer to another list, set new parent first -- destroy will add removed child to deleted list, to be destroyed later -- otherwise child remains intact but parent is nil -- could be inserted elsewhere
	DeleteChild(child Ki, destroy bool)

	// DeleteChildByName deletes child node by name -- returns child -- if child's parent = this node, then will call SetParent(nil), so to transfer to another list, set new parent first -- destroy will add removed child to deleted list, to be destroyed later -- otherwise child remains intact but parent is nil -- could be inserted elsewhere
	DeleteChildByName(name string, destroy bool) Ki

	// DeleteChildren deletes all children nodes -- destroy will add removed children to deleted list, to be destroyed later -- otherwise children remain intact but parent is nil -- could be inserted elsewhere, but you better have kept a slice of them before calling this
	DeleteChildren(destroy bool)

	// Delete deletes this node from its parent children list-- destroy will add removed child to deleted list, to be destroyed later -- otherwise child remains intact but parent is nil -- could be inserted elsewhere
	Delete(destroy bool)

	// DestroyDeleted is a second-pass that destroys all previously-removed children: causes them to remove all their children, cuts all their pointers and signal connections
	DestroyDeleted()

	// DestroyAllDeleted recursively calls DestroyDeleted on all nodes under this one -- called automatically when UpdateEnd reaches 0 Updating count and the Update signal is sent
	DestroyAllDeleted()

	// Destroy calls DisconnectAll to cut all pointers and signal connections, and remove all children and their childrens-children, etc
	Destroy()

	//////////////////////////////////////////////////////////////////////////
	//  Tree walking and Paths
	//   note: always put functions last -- looks better for inline functions

	// FuncFields calls function on all Ki fields within this node
	FuncFields(level int, data interface{}, fun Func)

	// GoFuncFields calls concurrent goroutine function on all Ki fields within this node
	GoFuncFields(level int, data interface{}, fun Func)

	// FuncUp calls function on given node and all the way up to its parents, and so on -- sequentially all in current go routine (generally necessary for going up, which is typicaly quite fast anyway) -- level is incremented after each step (starts at 0, goes up), and passed to function -- returns false if fun aborts with false, else true
	FuncUp(level int, data interface{}, fun Func) bool

	// FuncUpParent calls function on parent of node and all the way up to its parents, and so on -- sequentially all in current go routine (generally necessary for going up, which is typicaly quite fast anyway) -- level is incremented after each step (starts at 0, goes up), and passed to function -- returns false if fun aborts with false, else true
	FuncUpParent(level int, data interface{}, fun Func) bool

	// FuncDownMeFirst calls function on this node (MeFirst) and then call FuncDownMeFirst on all the children -- sequentially all in current go routine -- level var is incremented before calling children -- if fun returns false then any further traversal of that branch of the tree is aborted, but other branches continue -- i.e., if fun on current node returns false, then returns false and children are not processed further -- this is the fastest, most natural form of traversal
	FuncDownMeFirst(level int, data interface{}, fun Func) bool

	// FuncDownDepthFirst calls FuncDownDepthFirst on all children, then calls function on this node -- sequentially all in current go routine -- level var is incremented before calling children -- runs doChildTestFunc on each child first to determine if it should process that child, and if that returns true, then it calls FuncDownDepthFirst on that child
	FuncDownDepthFirst(level int, data interface{}, doChildTestFunc Func, fun Func)

	// FuncDownBreadthFirst calls function on all children, then calls FuncDownBreadthFirst on all the children -- does NOT call on first node where this method is first called, due to nature of recursive logic -- level var is incremented before calling chlidren -- if fun returns false then any further traversal of that branch of the tree is aborted, but other branches can continue
	FuncDownBreadthFirst(level int, data interface{}, fun Func)

	// GoFuncDown calls concurrent goroutine function on given node and all the way down to its children, and so on -- does not wait for completion of the go routines -- returns immediately
	GoFuncDown(level int, data interface{}, fun Func)

	// GoFuncDownWait calls concurrent goroutine function on given node and all the way down to its children, and so on -- does wait for the completion of the go routines before returning
	GoFuncDownWait(level int, data interface{}, fun Func)

	// Path returns path to this node from Root(), using regular user-given Name's (may be empty or non-unique), with nodes separated by / and fields by . -- only use for informational purposes
	Path() string

	// PathUnique returns path to this node from Root(), using unique names, with nodes separated by / and fields by . -- suitable for reliably finding this node
	PathUnique() string

	// PathFrom returns path to this node from given parent node, using regular user-given Name's (may be empty or non-unique), with nodes separated by / and fields by . -- only use for informational purposes
	PathFrom(par Ki) string

	// PathFromUnique returns path to this node from given parent node, using unique names, with nodes separated by / and fields by . -- suitable for reliably finding this node
	PathFromUnique(par Ki) string

	// FindPathUnique returns Ki object at given unique path, starting from this node (e.g., Root()) -- returns nil if not found
	FindPathUnique(path string) Ki

	//////////////////////////////////////////////////////////////////////////
	//  State update signaling -- automatically consolidates all changes across
	//   levels so there is only one update at end (optionally per node or only
	//   at highest level)
	//   All modification starts with UpdateStart() and ends with UpdateEnd()

	// NodeSignal returns the main signal for this node that is used for update, child signals
	NodeSignal() *Signal

	// UpdateCtr returns the update counter for this node -- uses atomic counter for thread safety
	UpdateCtr() *atomctr.Ctr

	// UpdateStart should be called when starting to modify the tree (state or structure) -- increments an atomic update counter and automatically calls start update on all children -- can be called multiple times at multiple levels -- it is essential to ensure that all such Start's have an End!
	UpdateStart()

	// UpdateEnd should be called when done updating after an UpdateStart -- decrements update counter and emits NodeSignalUpdated when counter goes to 0, only if parent is not current updating (i.e., this is the highest-level node that finished updating) -- see also UpdateEndAll
	UpdateEnd()

	// UpdateEndAll is an alternative to UpdateEnd when done updating -- decrements update counter and emits NodeSignalUpdated when counter goes to 0 for ALL nodes that might have updated, even if my parent node is still updating -- this is less typically used
	UpdateEndAll()

	// UpdateReset resets update counters to 0 -- in case they are out-of-sync due to more
	// complex tree maninpulations -- only call at a known point of
	// non-updating..
	UpdateReset()

	// Disconnect disconnects node -- reset all ptrs to nil, and DisconnectAll() signals
	// -- e.g., for freeing up all connnections so node can be destroyed and
	// making GC easier
	Disconnect()

	// DisconnectAll disconnects all the way from me down the tree
	DisconnectAll()

	//////////////////////////////////////////////////////////////////////////
	//  Field Value setting with notification

	// SetField sets given field name to given value, using very robust conversion routines to e.g., convert from strings to numbers, and vice-versa, automatically -- returns true if successfully set -- wrapped in UpdateStart / End and sets the FieldUpdated flag
	SetField(field string, val interface{}) bool

	// SetFieldDown sets given field name to given value, all the way down the tree from me -- wrapped in UpdateStart / End
	SetFieldDown(field string, val interface{})

	// SetFieldUp sets given field name to given value, all the way up the tree from me -- wrapped in UpdateStart / End
	SetFieldUp(field string, val interface{})

	// FieldByName returns field value by name (can be any type of field -- see KiFieldByName for Ki fields) -- returns nil if not found
	FieldByName(field string) interface{}

	//////////////////////////////////////////////////////////////////////////
	//  Deep Copy of Trees

	// CopyFrom another Ki node.  The Ki copy function recreates the entire
	// tree in the copy, duplicating children etc.  It is very efficient by
	// using the ConfigChildren method which attempts to preserve any existing
	// nodes in the destination if they have the same name and type -- so a
	// copy from a source to a target that only differ minimally will be
	// minimally destructive.  Only copies to same types are supported.
	// Pointers (Ptr) are copied by saving the current UniquePath and then
	// SetPtrsFmPaths is called -- no other Ki point.  Signal connections are
	// NOT copied (todo: revisit).  No other Ki pointers are copied, and the
	// field tag copy:"-" can be added for any other fields that should not be
	// copied (unexported, lower-case fields are not copyable).
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

	// Clone creates and returns a deep copy of the tree from this node down.
	// Any pointers within the cloned tree will correctly point within the new
	// cloned tree (see Copy info)
	Clone() Ki

	// CopyFromRaw performs a raw copy that just does the deep copy of the bits and doesn't do anything with pointers
	CopyFromRaw(from Ki) error

	// GetPtrPaths gets all Ptr path strings -- walks the tree down from current node and calls GetPath on all Ptr fields -- this is called prior to copying / moving
	GetPtrPaths()

	// UpdatePtrPaths updates Ptr paths, replacing any occurrence of oldPath with
	// newPath, optionally only at the start of the path (typically true) --
	// for all nodes down from this one
	UpdatePtrPaths(oldPath, newPath string, startOnly bool)

	// SetPtrsFmPaths walks the tree down from current node and calls PtrFromPath on all Ptr fields found -- called after Copy, Unmarshal* to recover pointers after entire structure is in place -- see UnmarshalPost
	SetPtrsFmPaths()

	//////////////////////////////////////////////////////////////////////////
	//  IO: Marshal / Unmarshal support -- see also Slice, Ptr

	// SaveJSON saves the tree to a JSON-encoded byte string -- wraps MarshalJSON
	SaveJSON(indent bool) ([]byte, error)

	// LoadJSON loads the tree from a JSON-encoded byte string -- wraps UnmarshalJSON and calls UnmarshalPost to recover pointers from paths
	LoadJSON(b []byte) error

	// SaveXML saves the tree to an XML-encoded byte string
	SaveXML(indent bool) ([]byte, error)

	// LoadXML loads the tree from an XML-encoded byte string, calls UnmarshalPost to recover pointers from paths
	LoadXML(b []byte) error

	// ParentAllChildren walks the tree down from current node and call SetParent on all children -- needed after an Unmarshal
	ParentAllChildren()

	// UnmarshallPost must be called after an Unmarshal -- calls SetPtrsFmPaths and ParentAllChildren
	UnmarshalPost()
}

// see node.go for struct implementing this interface

// IMPORTANT: all types must initialize entry in package kit Types Registry:
// var KiT_TypeName = kit.Types.AddType(&TypeName{})

// function to call on ki objects walking the tree -- return bool = false means don't continue processing this branch of the tree, but other branches can continue
type Func func(ki Ki, level int, data interface{}) bool

// a Ki reflect.Type, suitable for checking for Type.Implements
func KiType() reflect.Type {
	return reflect.TypeOf((*Ki)(nil)).Elem()
}
