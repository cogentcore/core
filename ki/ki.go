// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package ki

import (
	"io"
	"log"
	"reflect"

	"github.com/goki/ki/kit"
)

// The Ki interface provides the core functionality for a GoKi tree.
// Each Ki is a node in the tree and can have child nodes, and no cycles
// are allowed (i.e., each node can only appear once in the tree).
// All the usual methods are included for accessing and managing Children,
// and efficiently traversing the tree and calling functions on the nodes.
// In addition, Ki nodes can have Fields that are also Ki nodes that
// are included in all the automatic tree traversal methods -- they are
// effectively named fixed children that are automatically present.
//
// In general, the names of the children of a given node should all be unique.
// The following functions defined in ki package can be used:
// UniqueNameCheck(node) to check for unique names on node if uncertain.
// UniqueNameCheckAll(node) to check entire tree under given node.
// UniquifyNames(node) to add a suffix to name to ensure uniqueness.
// UniquifyNamesAll(node) to to uniquify all names in entire tree.
//
// Use function MoveChild to move a node between trees or within a tree --
// otherwise nodes are typically created and deleted but not moved.
//
// The Ki interface is designed to support virtual method calling in Go
// and is only intended to be implemented once, by the ki.Node type
// (as opposed to interfaces that are used for hiding multiple different
// implementations of a common concept).  Thus, all of the fields in ki.Node
// are exported (have captital names), to be accessed directly in types
// that embed and extend the ki.Node. The Ki interface has the "formal" name
// (e.g., Children) while the Node has the "nickname" (e.g., Kids).  See the
// Naming Conventions on the GoKi Wiki for more details.
//
// Each Node stores the Ki interface version of itself, as This() / Ths
// which enables full virtual function calling by calling the method
// on that interface instead of directly on the receiver Node itself.
// This requires proper initialization via Init method of the Ki interface.
//
// Ki nodes also support the following core functionality:
//   - UpdateStart() / UpdateEnd() to wrap around tree updating code, which then
//     automatically triggers update signals at the highest level of the
//     affected tree, resulting in efficient updating logic for arbitrary
//     nested tree modifications.
//   - Signal framework for sending messages such as the Update signals, used
//     extensively in the GoGi GUI framework for sending event messages etc.
//   - ConfigChildren system for minimally updating children to fit a given
//     Name & Type template.
//   - Automatic JSON I/O of entire tree including type information.
type Ki interface {
	// InitName initializes this node to given actual object as a Ki interface
	// and sets its name.  The names should be unique among children of a node.
	// This is needed for root nodes -- automatically done for other nodes
	// when they are added to the Ki tree.
	// Even though this is a method and gets the method receiver, it needs
	// an "external" version of itself passed as the first arg, from which
	// the proper Ki interface pointer will be obtained.  This is the only
	// way to get virtual functional calling to work within the Go framework.
	InitName(this Ki, name string)

	// This returns the Ki interface that guarantees access to the Ki
	// interface in a way that always reveals the underlying type
	// (e.g., in reflect calls).  Returns nil if node is nil,
	// has been destroyed, or is improperly constructed.
	This() Ki

	// AsNode returns the *ki.Node base type for this node.
	AsNode() *Node

	// Embed returns the embedded struct of given type from this node (or nil
	// if it does not embed that type, or the type is not a Ki type -- see
	// kit.Embed for a generic interface{} version.
	Embed(t reflect.Type) Ki

	// BaseIface returns the base interface type for all elements
	// within this tree.  Use reflect.TypeOf((*<interface_type>)(nil)).Elem().
	// Used e.g., for determining what types of children
	// can be created (see kit.EmbedImplements for test method)
	BaseIface() reflect.Type

	// Name returns the user-defined name of the object (Node.Nm),
	// for finding elements, generating paths, IO, etc.
	Name() string

	// SetName sets the name of this node.
	// Names should generally be unique across children of each node.
	// See Unique* functions to check / fix.
	// If node requires non-unique names, add a separate Label field.
	// Does NOT wrap in UpdateStart / End.
	SetName(name string)

	//////////////////////////////////////////////////////////////////////////
	//  Parents

	// Parent returns the parent of this Ki (Node.Par) -- Ki has strict
	// one-parent, no-cycles structure -- see SetParent.
	Parent() Ki

	// IndexInParent returns our index within our parent object -- caches the
	// last value and uses that for an optimized search so subsequent calls
	// are typically quite fast.  Returns false if we don't have a parent.
	IndexInParent() (int, bool)

	// ParentLevel finds a given potential parent node recursively up the
	// hierarchy, returning level above current node that the parent was
	// found, and -1 if not found.
	ParentLevel(par Ki) int

	// ParentByName finds first parent recursively up hierarchy that matches
	// given name.  Returns nil if not found.
	ParentByName(name string) Ki

	// ParentByNameTry finds first parent recursively up hierarchy that matches
	// given name -- Try version returns error on failure.
	ParentByNameTry(name string) (Ki, error)

	// ParentByType finds parent recursively up hierarchy, by type, and
	// returns nil if not found. If embeds is true, then it looks for any
	// type that embeds the given type at any level of anonymous embedding.
	ParentByType(t reflect.Type, embeds bool) Ki

	// ParentByTypeTry finds parent recursively up hierarchy, by type, and
	// returns error if not found. If embeds is true, then it looks for any
	// type that embeds the given type at any level of anonymous embedding.
	ParentByTypeTry(t reflect.Type, embeds bool) (Ki, error)

	//////////////////////////////////////////////////////////////////////////
	//  Children

	// HasChildren tests whether this node has children (i.e., non-terminal).
	HasChildren() bool

	// NumChildren returns the number of children
	NumChildren() int

	// Children returns a pointer to the slice of children (Node.Kids) -- use
	// methods on ki.Slice for further ways to access (ByName, ByType, etc).
	// Slice can be modified, deleted directly (e.g., sort, reorder) but Add
	// method on parent node should be used to ensure proper init.
	Children() *Slice

	// Child returns the child at given index -- will panic if index is invalid.
	// See methods on ki.Slice for more ways to access.
	Child(idx int) Ki

	// ChildTry returns the child at given index -- Try version returns
	// error if index is invalid.
	// See methods on ki.Slice for more ways to access.
	ChildTry(idx int) (Ki, error)

	// ChildByName returns first element that has given name, nil if not found.
	// startIdx arg allows for optimized bidirectional find if you have
	// an idea where it might be -- can be key speedup for large lists -- pass
	// [ki.StartMiddle] to start in the middle (good default).
	ChildByName(name string, startIdx int) Ki

	// ChildByNameTry returns first element that has given name -- Try version
	// returns error message if not found.
	// startIdx arg allows for optimized bidirectional find if you have
	// an idea where it might be -- can be key speedup for large lists -- pass
	// [ki.StartMiddle] to start in the middle (good default).
	ChildByNameTry(name string, startIdx int) (Ki, error)

	// ChildByType returns first element that has given type, nil if not found.
	// If embeds is true, then it looks for any type that embeds the given type
	// at any level of anonymous embedding.
	// startIdx arg allows for optimized bidirectional find if you have
	// an idea where it might be -- can be key speedup for large lists -- pass
	// [ki.StartMiddle] to start in the middle (good default).
	ChildByType(t reflect.Type, embeds bool, startIdx int) Ki

	// ChildByTypeTry returns first element that has given name -- Try version
	// returns error message if not found.
	// If embeds is true, then it looks for any type that embeds the given type
	// at any level of anonymous embedding.
	// startIdx arg allows for optimized bidirectional find if you have
	// an idea where it might be -- can be key speedup for large lists -- pass
	// [ki.StartMiddle] to start in the middle (good default).
	ChildByTypeTry(t reflect.Type, embeds bool, startIdx int) (Ki, error)

	//////////////////////////////////////////////////////////////////////////
	//  Paths

	// Path returns path to this node from the tree root, using node Names
	// separated by / and fields by .
	// Node names escape any existing / and . characters to \\ and \,
	// Path is only valid when child names are unique (see Unique* functions)
	Path() string

	// PathFrom returns path to this node from given parent node, using
	// node Names separated by / and fields by .
	// Node names escape any existing / and . characters to \\ and \,
	// Path is only valid when child names are unique (see Unique* functions)
	PathFrom(par Ki) string

	// FindPath returns Ki object at given path, starting from this node
	// (e.g., the root).  If this node is not the root, then the path
	// to this node is subtracted from the start of the path if present there.
	// FindPath only works correctly when names are unique.
	// Path has node Names separated by / and fields by .
	// Node names escape any existing / and . characters to \\ and \,
	// There is also support for [idx] index-based access for any given path
	// element, for cases when indexes are more useful than names.
	// Returns nil if not found.
	FindPath(path string) Ki

	// FindPathTry returns Ki object at given path, starting from this node
	// (e.g., the root).  If this node is not the root, then the path
	// to this node is subtracted from the start of the path if present there.
	// FindPath only works correctly when names are unique.
	// Path has node Names separated by / and fields by .
	// Node names escape any existing / and . characters to \\ and \,
	// There is also support for [idx] index-based access for any given path
	// element, for cases when indexes are more useful than names.
	// Returns error if not found.
	FindPathTry(path string) (Ki, error)

	//////////////////////////////////////////////////////////////////////////
	//  Adding, Inserting Children

	// AddChild adds given child at end of children list.
	// The kid node is assumed to not be on another tree (see MoveToParent)
	// and the existing name should be unique among children.
	// No UpdateStart / End wrapping is done: do that externally as needed.
	// Can also call SetFlag(ki.ChildAdded) if notification is needed.
	AddChild(kid Ki) error

	// AddNewChild creates a new child of given type and
	// add at end of children list.
	// The name should be unique among children.
	// No UpdateStart / End wrapping is done: do that externally as needed.
	// Can also call SetFlag(ki.ChildAdded) if notification is needed.
	AddNewChild(typ reflect.Type, name string) Ki

	// SetChild sets child at given index to be the given item -- if name is
	// non-empty then it sets the name of the child as well -- just calls Init
	// (or InitName) on the child, and SetParent.
	// Names should be unique among children.
	// No UpdateStart / End wrapping is done: do that externally as needed.
	// Can also call SetFlag(ki.ChildAdded) if notification is needed.
	SetChild(kid Ki, idx int, name string) error

	// InsertChild adds given child at position in children list.
	// The kid node is assumed to not be on another tree (see MoveToParent)
	// and the existing name should be unique among children.
	// No UpdateStart / End wrapping is done: do that externally as needed.
	// Can also call SetFlag(ki.ChildAdded) if notification is needed.
	InsertChild(kid Ki, at int) error

	// InsertNewChild creates a new child of given type and
	// add at position in children list.
	// The name should be unique among children.
	// No UpdateStart / End wrapping is done: do that externally as needed.
	// Can also call SetFlag(ki.ChildAdded) if notification is needed.
	InsertNewChild(typ reflect.Type, at int, name string) Ki

	// SetNChildren ensures that there are exactly n children, deleting any
	// extra, and creating any new ones, using AddNewChild with given type and
	// naming according to nameStubX where X is the index of the child.
	//
	// IMPORTANT: returns whether any modifications were made (mods) AND if
	// that is true, the result from the corresponding UpdateStart call --
	// UpdateEnd is NOT called, allowing for further subsequent updates before
	// you call UpdateEnd(updt)
	//
	// Note that this does not ensure existing children are of given type, or
	// change their names, or call UniquifyNames -- use ConfigChildren for
	// those cases -- this function is for simpler cases where a parent uses
	// this function consistently to manage children all of the same type.
	SetNChildren(n int, typ reflect.Type, nameStub string) (mods, updt bool)

	// ConfigChildren configures children according to given list of
	// type-and-name's -- attempts to have minimal impact relative to existing
	// items that fit the type and name constraints (they are moved into the
	// corresponding positions), and any extra children are removed, and new
	// ones added, to match the specified config.
	// It is important that names are unique!
	//
	// IMPORTANT: returns whether any modifications were made (mods) AND if
	// that is true, the result from the corresponding UpdateStart call --
	// UpdateEnd is NOT called, allowing for further subsequent updates before
	// you call UpdateEnd(updt).
	ConfigChildren(config kit.TypeAndNameList) (mods, updt bool)

	//////////////////////////////////////////////////////////////////////////
	//  Deleting Children

	// DeleteChildAtIndex deletes child at given index (returns error for
	// invalid index).
	// Wraps delete in UpdateStart / End and sets ChildDeleted flag.
	DeleteChildAtIndex(idx int, destroy bool) error

	// DeleteChild deletes child node, returning error if not found in
	// Children.
	// Wraps delete in UpdateStart / End and sets ChildDeleted flag.
	DeleteChild(child Ki, destroy bool) error

	// DeleteChildByName deletes child node by name -- returns child, error
	// if not found.
	// Wraps delete in UpdateStart / End and sets ChildDeleted flag.
	DeleteChildByName(name string, destroy bool) (Ki, error)

	// DeleteChildren deletes all children nodes -- destroy will add removed
	// children to deleted list, to be destroyed later -- otherwise children
	// remain intact but parent is nil -- could be inserted elsewhere, but you
	// better have kept a slice of them before calling this.
	DeleteChildren(destroy bool)

	// Delete deletes this node from its parent children list -- destroy will
	// add removed child to deleted list, to be destroyed later -- otherwise
	// child remains intact but parent is nil -- could be inserted elsewhere.
	Delete(destroy bool)

	// Destroy calls DisconnectAll to cut all signal connections,
	// and remove all children and their childrens-children, etc.
	Destroy()

	//////////////////////////////////////////////////////////////////////////
	//  Flags

	// Flag returns an atomically safe copy of the bit flags for this node --
	// can use bitflag package to check lags.
	// See Flags type for standard values used in Ki Node --
	// can be extended from FlagsN up to 64 bit capacity.
	// Note that we must always use atomic access as *some* things need to be atomic,
	// and with bits, that means that *all* access needs to be atomic,
	// as you cannot atomically update just a single bit.
	Flags() int64

	// HasFlag checks if flag is set
	// using atomic, safe for concurrent access
	HasFlag(flag int) bool

	// SetFlag sets the given flag(s)
	// using atomic, safe for concurrent access
	SetFlag(flag ...int)

	// SetFlagState sets the given flag(s) to given state
	// using atomic, safe for concurrent access
	SetFlagState(on bool, flag ...int)

	// SetFlagMask sets the given flags as a mask
	// using atomic, safe for concurrent access
	SetFlagMask(mask int64)

	// ClearFlag clears the given flag(s)
	// using atomic, safe for concurrent access
	ClearFlag(flag ...int)

	// ClearFlagMask clears the given flags as a bitmask
	// using atomic, safe for concurrent access
	ClearFlagMask(mask int64)

	// IsField checks if this is a field on a parent struct (via IsField
	// Flag), as opposed to a child in Children -- Ki nodes can be added as
	// fields to structs and they are automatically parented and named with
	// field name during Init function -- essentially they function as fixed
	// children of the parent struct, and are automatically included in
	// FuncDown* traversals, etc -- see also FunFields.
	IsField() bool

	// IsUpdating checks if node is currently updating.
	IsUpdating() bool

	// OnlySelfUpdate checks if this node only applies UpdateStart / End logic
	// to itself, not its children (which is the default) (via Flag of same
	// name) -- useful for a parent node that has a different function than
	// its children.
	OnlySelfUpdate() bool

	// SetChildAdded sets the ChildAdded flag -- set when notification is needed
	// for Add, Insert methods
	SetChildAdded()

	// SetValUpdated sets the ValUpdated flag -- set when notification is needed
	// for modifying a value (field, prop, etc)
	SetValUpdated()

	// IsDeleted checks if this node has just been deleted (within last update
	// cycle), indicated by the NodeDeleted flag which is set when the node is
	// deleted, and is cleared at next UpdateStart call.
	IsDeleted() bool

	// IsDestroyed checks if this node has been destroyed -- the NodeDestroyed
	// flag is set at start of Destroy function -- the Signal Emit process
	// checks for destroyed receiver nodes and removes connections to them
	// automatically -- other places where pointers to potentially destroyed
	// nodes may linger should also check this flag and reset those pointers.
	IsDestroyed() bool

	//////////////////////////////////////////////////////////////////////////
	//  Property interface with inheritance -- nodes can inherit props from parents

	// Properties (Node.Props) tell the GoGi GUI or other frameworks operating
	// on Trees about special features of each node -- functions below support
	// inheritance up Tree -- see kit convert.go for robust convenience
	// methods for converting interface{} values to standard types.
	Properties() *Props

	// SetProp sets given property key to value val -- initializes property
	// map if nil.
	SetProp(key string, val any)

	// Prop returns property value for key that is known to exist.
	// Returns nil if it actually doesn't -- this version allows
	// direct conversion of return.  See PropTry for version with
	// error message if uncertain if property exists.
	Prop(key string) any

	// PropTry returns property value for key.  Returns error message
	// if property with that key does not exist.
	PropTry(key string) (any, error)

	// PropInherit gets property value from key with options for inheriting
	// property from parents and / or type-level properties.  If inherit, then
	// checks all parents.  If typ then checks property on type as well
	// (registered via KiT type registry).  Returns false if not set anywhere.
	PropInherit(key string, inherit, typ bool) (any, bool)

	// DeleteProp deletes property key on this node.
	DeleteProp(key string)

	// PropTag returns the name to look for in type properties, for types
	// that are valid options for values that can be set in Props.  For example
	// in GoGi, it is "style-props" which is then set for all types that can
	// be used in a style (colors, enum options, etc)
	PropTag() string

	//////////////////////////////////////////////////////////////////////////
	//  Tree walking and Paths
	//   note: always put function args last -- looks better for inline functions

	// FuncFields calls function on all Ki fields within this node.
	FuncFields(level int, data any, fun Func)

	// FuncUp calls function on given node and all the way up to its parents,
	// and so on -- sequentially all in current go routine (generally
	// necessary for going up, which is typically quite fast anyway) -- level
	// is incremented after each step (starts at 0, goes up), and passed to
	// function -- returns false if fun aborts with false, else true.
	FuncUp(level int, data any, fun Func) bool

	// FuncUpParent calls function on parent of node and all the way up to its
	// parents, and so on -- sequentially all in current go routine (generally
	// necessary for going up, which is typically quite fast anyway) -- level
	// is incremented after each step (starts at 0, goes up), and passed to
	// function -- returns false if fun aborts with false, else true.
	FuncUpParent(level int, data any, fun Func) bool

	// FuncDownMeFirst calls function on this node (MeFirst) and then iterates
	// in a depth-first manner over all the children, including Ki Node fields,
	// which are processed first before children.
	// This uses node state information to manage the traversal and is very fast,
	// but can only be called by one thread at a time -- use a Mutex if there is
	// a chance of multiple threads running at the same time.
	// Function calls are sequential all in current go routine.
	// The level var tracks overall depth in the tree.
	// If fun returns false then any further traversal of that branch of the tree is
	// aborted, but other branches continue -- i.e., if fun on current node
	// returns false, children are not processed further.
	FuncDownMeFirst(level int, data any, fun Func)

	// FuncDownMeLast iterates in a depth-first manner over the children, calling
	// doChildTestFunc on each node to test if processing should proceed (if it returns
	// false then that branch of the tree is not further processed), and then
	// calls given fun function after all of a node's children (including fields)
	// have been iterated over ("Me Last").
	// This uses node state information to manage the traversal and is very fast,
	// but can only be called by one thread at a time -- use a Mutex if there is
	// a chance of multiple threads running at the same time.
	// Function calls are sequential all in current go routine.
	// The level var tracks overall depth in the tree.
	FuncDownMeLast(level int, data any, doChildTestFunc Func, fun Func)

	// FuncDownBreadthFirst calls function on all children in breadth-first order
	// using the standard queue strategy.  This depends on and updates the
	// Depth parameter of the node.  If fun returns false then any further
	// traversal of that branch of the tree is aborted, but other branches continue.
	FuncDownBreadthFirst(level int, data any, fun Func)

	//////////////////////////////////////////////////////////////////////////
	//  State update signaling -- automatically consolidates all changes across
	//   levels so there is only one update at end (optionally per node or only
	//   at highest level)
	//   All modification starts with UpdateStart() and ends with UpdateEnd()

	// NodeSignal returns the main signal for this node that is used for
	// update, child signals.
	NodeSignal() *Signal

	// UpdateStart should be called when starting to modify the tree (state or
	// structure) -- returns whether this node was first to set the Updating
	// flag (if so, all children have their Updating flag set -- pass the
	// result to UpdateEnd -- automatically determines the highest level
	// updated, within the normal top-down updating sequence -- can be called
	// multiple times at multiple levels -- it is essential to ensure that all
	// such Start's have an End!  Usage:
	//
	//   updt := n.UpdateStart()
	//   ... code
	//   n.UpdateEnd(updt)
	// or
	//   updt := n.UpdateStart()
	//   defer n.UpdateEnd(updt)
	//   ... code
	UpdateStart() bool

	// UpdateEnd should be called when done updating after an UpdateStart, and
	// passed the result of the UpdateStart call -- if this is true, the
	// NodeSignalUpdated signal will be emitted and the Updating flag will be
	// cleared, and DestroyDeleted called -- otherwise it is a no-op.
	UpdateEnd(updt bool)

	// UpdateEndNoSig is just like UpdateEnd except it does not emit a
	// NodeSignalUpdated signal -- use this for situations where updating is
	// already known to be in progress and the signal would be redundant.
	UpdateEndNoSig(updt bool)

	// UpdateSig just emits a NodeSignalUpdated if the Updating flag is not
	// set -- use this to trigger an update of a given node when there aren't
	// any structural changes and you don't need to prevent any lower-level
	// updates -- much more efficient than a pair of UpdateStart /
	// UpdateEnd's.  Returns true if an update signal was sent.
	UpdateSig() bool

	// Disconnect disconnects this node, by calling DisconnectAll() on
	// any Signal fields.  Any Node that adds a Signal must define an
	// updated version of this method that calls its embedded parent's
	// version and then calls DisconnectAll() on its Signal fields.
	Disconnect()

	// DisconnectAll disconnects all the way from me down the tree.
	DisconnectAll()

	//////////////////////////////////////////////////////////////////////////
	//  Field Value setting with notification

	// SetField sets given field name to given value, using very robust
	// conversion routines to e.g., convert from strings to numbers, and
	// vice-versa, automatically.  Returns error if not successfully set.
	// wrapped in UpdateStart / End and sets the ValUpdated flag.
	SetField(field string, val any) error

	//////////////////////////////////////////////////////////////////////////
	//  Deep Copy of Trees

	// CopyFrom another Ki node.  It is essential that source has Unique names!
	// The Ki copy function recreates the entire tree in the copy, duplicating
	// children etc, copying Props too.  It is very efficient by
	// using the ConfigChildren method which attempts to preserve any existing
	// nodes in the destination if they have the same name and type -- so a
	// copy from a source to a target that only differ minimally will be
	// minimally destructive.  Only copies to same types are supported.
	// Signal connections are NOT copied.  No other Ki pointers are copied,
	// and the field tag copy:"-" can be added for any other fields that
	// should not be copied (unexported, lower-case fields are not copyable).
	CopyFrom(frm Ki) error

	// Clone creates and returns a deep copy of the tree from this node down.
	// Any pointers within the cloned tree will correctly point within the new
	// cloned tree (see Copy info).
	Clone() Ki

	// CopyFieldsFrom is the base-level copy method that any copy-intensive
	// nodes should implement directly to explicitly copy relevant fields
	// that should be copied, avoiding any internal pointers etc.
	// This is the performance bottleneck in copying -- the Node version
	// uses generic GenCopyFieldsFrom method using reflection etc
	// which is very slow.  It can be ~10x faster overall to use custom
	// method that explicitly copies each field.  When doing so, you
	// must explicitly call the CopyFieldsFrom method on any embedded
	// Ki types that you inherit from, and, critically, NONE of those
	// can rely on the generic Node-level version.  Furthermore, if the
	// actual end type itself does not define a custom version of this method
	// then the generic one will be called for everything.
	CopyFieldsFrom(frm any)

	//////////////////////////////////////////////////////////////////////////
	//  IO: for JSON and XML formats -- see also Slice
	//  see https://github.com/goki/ki/wiki/Naming for IO naming conventions

	// WriteJSON writes the tree to an io.Writer, using MarshalJSON -- also
	// saves a critical starting record that allows file to be loaded de-novo
	// and recreate the proper root type for the tree.
	WriteJSON(writer io.Writer, indent bool) error

	// SaveJSON saves the tree to a JSON-encoded file, using WriteJSON.
	SaveJSON(filename string) error

	// ReadJSON reads and unmarshals tree starting at this node, from a
	// JSON-encoded byte stream via io.Reader.  First element in the stream
	// must be of same type as this node -- see ReadNewJSON function to
	// construct a new tree.  Uses ConfigureChildren to minimize changes from
	// current tree relative to loading one -- wraps UnmarshalJSON and calls
	// UnmarshalPost to recover pointers from paths.
	ReadJSON(reader io.Reader) error

	// OpenJSON opens file over this tree from a JSON-encoded file -- see
	// ReadJSON for details, and OpenNewJSON for opening an entirely new tree.
	OpenJSON(filename string) error

	// WriteXML writes the tree to an XML-encoded byte string over io.Writer
	// using MarshalXML.
	WriteXML(writer io.Writer, indent bool) error

	// ReadXML reads the tree from an XML-encoded byte string over io.Reader, calls
	// UnmarshalPost to recover pointers from paths.
	ReadXML(reader io.Reader) error
}

// see node.go for struct implementing this interface

// IMPORTANT: all types should initialize entry in package kit Types Registry

// var KiT_TypeName = kit.Types.AddType(&TypeName{})

// Func is a function to call on ki objects walking the tree -- return Break
// = false means don't continue processing this branch of the tree, but other
// branches can continue.  return Continue = true continues down the tree.
type Func func(k Ki, level int, data any) bool

// KiType is a Ki reflect.Type, suitable for checking for Type.Implements.
var KiType = reflect.TypeOf((*Ki)(nil)).Elem()

// IsKi returns true if the given type implements the Ki interface at any
// level of embedded structure.
func IsKi(typ reflect.Type) bool {
	if typ == nil {
		return false
	}
	return kit.EmbedImplements(typ, KiType)
}

// NewOfType makes a new Ki struct of given type -- must be a Ki type -- will
// return nil if not.
func NewOfType(typ reflect.Type) Ki {
	nkid := reflect.New(typ).Interface()
	kid, ok := nkid.(Ki)
	if !ok {
		log.Printf("ki.NewOfType: type %v cannot be converted into a Ki interface type\n", typ.String())
		return nil
	}
	return kid
}

// Type returns the underlying struct type of given node
func Type(k Ki) reflect.Type {
	return reflect.TypeOf(k.This()).Elem()
}

// TypeEmbeds tests whether this node is of the given type, or it embeds
// that type at any level of anonymous embedding -- use Embed to get the
// embedded struct of that type from this node.
func TypeEmbeds(k Ki, t reflect.Type) bool {
	return kit.TypeEmbeds(Type(k), t)
}
