// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package ki

import (
	"errors"
	"fmt"
	"log/slog"
	"strconv"

	"log"
	"reflect"
	"strings"

	"goki.dev/enums"
	"goki.dev/gti"
)

// The Node implements the Ki interface and provides the core functionality
// for the GoKi tree -- use the Node as an embedded struct or as a struct
// field -- the embedded version supports full JSON save / load.
//
// The desc: key for fields is used by the GoGi GUI viewer for help / tooltip
// info -- add these to all your derived struct's fields.  See relevant docs
// for other such tags controlling a wide range of GUI and other functionality
// -- Ki makes extensive use of such tags.
type Node struct {

	// Ki.Name() user-supplied name of this node -- can be empty or non-unique
	Nm string `copy:"-" label:"Name" desc:"Ki.Name() user-supplied name of this node -- can be empty or non-unique"`

	// [tableview: -] bit flags for internal node state -- can extend this using enums package
	Flags Flags `tableview:"-" copy:"-" json:"-" xml:"-" max-width:"80" height:"3" desc:"bit flags for internal node state -- can extend this using enums package"`

	// [tableview: -] Ki.Properties() property map for arbitrary extensible properties, including style properties
	Props Props `tableview:"-" xml:"-" copy:"-" label:"Properties" desc:"Ki.Properties() property map for arbitrary extensible properties, including style properties"`

	// [view: -] [tableview: -] Ki.Parent() parent of this node -- set automatically when this node is added as a child of parent
	Par Ki `tableview:"-" copy:"-" json:"-" xml:"-" label:"Parent" view:"-" desc:"Ki.Parent() parent of this node -- set automatically when this node is added as a child of parent"`

	// [tableview: -] Ki.Children() list of children of this node -- all are set to have this node as their parent -- can reorder etc but generally use Ki Node methods to Add / Delete to ensure proper usage
	Kids Slice `tableview:"-" copy:"-" label:"Children" desc:"Ki.Children() list of children of this node -- all are set to have this node as their parent -- can reorder etc but generally use Ki Node methods to Add / Delete to ensure proper usage"`

	// [view: -] Ki.NodeSignal() signal for node structure / state changes -- emits NodeSignals signals -- can also extend to custom signals (see signal.go) but in general better to create a new Signal instead
	NodeSig Signal `copy:"-" json:"-" xml:"-" view:"-" desc:"Ki.NodeSignal() signal for node structure / state changes -- emits NodeSignals signals -- can also extend to custom signals (see signal.go) but in general better to create a new Signal instead"`

	// [view: -] we need a pointer to ourselves as a Ki, which can always be used to extract the true underlying type of object when Node is embedded in other structs -- function receivers do not have this ability so this is necessary.  This is set to nil when deleted.  Typically use This() convenience accessor which protects against concurrent access.
	Ths Ki `copy:"-" json:"-" xml:"-" view:"-" desc:"we need a pointer to ourselves as a Ki, which can always be used to extract the true underlying type of object when Node is embedded in other structs -- function receivers do not have this ability so this is necessary.  This is set to nil when deleted.  Typically use This() convenience accessor which protects against concurrent access."`

	// [view: -] last value of our index -- used as a starting point for finding us in our parent next time -- is not guaranteed to be accurate!  use IndexInParent() method
	index int `copy:"-" json:"-" xml:"-" view:"-" desc:"last value of our index -- used as a starting point for finding us in our parent next time -- is not guaranteed to be accurate!  use IndexInParent() method"`

	// [view: -] optional depth parameter of this node -- only valid during specific contexts, not generally -- e.g., used in FuncDownBreadthFirst function
	depth int `copy:"-" json:"-" xml:"-" view:"-" desc:"optional depth parameter of this node -- only valid during specific contexts, not generally -- e.g., used in FuncDownBreadthFirst function"`

	// [view: -] cached version of the field offsets relative to base Node address -- used in generic field access.
	fieldOffs []uintptr `copy:"-" json:"-" xml:"-" view:"-" desc:"cached version of the field offsets relative to base Node address -- used in generic field access."`
}

// check implementation of [Ki] interface
var _ = Ki(&Node{})

// EnumTypeFlag is a [Props] property name that
// indicates what enum type to use as the type for
// the flags field in GUI views. Its value should be
// of the type [reflect.Type]
const EnumTypeFlag string = "EnumType:Flag"

//////////////////////////////////////////////////////////////////////////
//  fmt.Stringer

// String implements the fmt.stringer interface -- returns the Path of the node
func (n *Node) String() string {
	return n.This().Path()
}

//////////////////////////////////////////////////////////////////////////
//  Basic Ki fields

// This returns the Ki interface that guarantees access to the Ki
// interface in a way that always reveals the underlying type
// (e.g., in reflect calls).  Returns nil if node is nil,
// has been destroyed, or is improperly constructed.
func (n *Node) This() Ki {
	if n == nil || n.IsDestroyed() {
		return nil
	}
	return n.Ths
}

// AsNode returns the *ki.Node base type for this node.
func (n *Node) AsNode() *Node {
	return n
}

// InitName initializes this node to given actual object as a Ki interface
// and sets its name.  The names should be unique among children of a node.
// This is needed for root nodes -- automatically done for other nodes
// when they are added to the Ki tree.
// Even though this is a method and gets the method receiver, it needs
// an "external" version of itself passed as the first arg, from which
// the proper Ki interface pointer will be obtained.  This is the only
// way to get virtual functional calling to work within the Go framework.
func (n *Node) InitName(k Ki, name string) {
	InitNode(k)
	n.SetName(name)
}

// BaseIface returns the base interface type for all elements
// within this tree.  Use reflect.TypeOf((*<interface_type>)(nil)).Elem().
// Used e.g., for determining what types of children
// can be created.
func (n *Node) BaseIface() reflect.Type {
	return KiType
}

// Name returns the user-defined name of the object (Node.Nm),
// for finding elements, generating paths, IO, etc.
func (n *Node) Name() string {
	return n.Nm
}

// SetName sets the name of this node.
// Names should generally be unique across children of each node.
// See Unique* functions to check / fix.
// If node requires non-unique names, add a separate Label field.
// Does NOT wrap in UpdateStart / End.
func (n *Node) SetName(name string) {
	n.Nm = name
}

// OnInit is a placeholder implementation of
// [Ki.OnInit] that does nothing.
func (n *Node) OnInit() {}

// OnAdd is a placeholder implementation of
// [Ki.OnAdd] that does nothing.
func (n *Node) OnAdd() {}

// OnChildAdded is a placeholder implementation of
// [Ki.OnChildAdded] that does nothing.
func (n *Node) OnChildAdded(child Ki) {}

// todo: generate me!
var NodeType = &gti.Type{}

// Type returns the gti Type record for this Ki node.
// This is auto-generated by the gtigen generator for Ki types.
func (n *Node) Type() *gti.Type {
	return NodeType
}

// New returns a new token of this Ki node.
// InitName _must_ still be called on this new token.
// This is auto-generated by the gtigen generator for Ki types.
func (n *Node) New() any {
	return &Node{}
}

//////////////////////////////////////////////////////////////////////////
//  Parents

// Parent returns the parent of this Ki (Node.Par) -- Ki has strict
// one-parent, no-cycles structure -- see SetParent.
func (n *Node) Parent() Ki {
	return n.Par
}

// IndexInParent returns our index within our parent object -- caches the
// last value and uses that for an optimized search so subsequent calls
// are typically quite fast.  Returns false if we don't have a parent.
func (n *Node) IndexInParent() (int, bool) {
	if n.Par == nil {
		return -1, false
	}
	idx, ok := n.Par.Children().IndexOf(n.This(), n.index) // very fast if index is close..
	if ok {
		n.index = idx
	}
	return idx, ok
}

// ParentLevel finds a given potential parent node recursively up the
// hierarchy, returning level above current node that the parent was
// found, and -1 if not found.
func (n *Node) ParentLevel(par Ki) int {
	parLev := -1
	n.FuncUpParent(0, n, func(k Ki, level int, d any) bool {
		if k == par {
			parLev = level
			return Break
		}
		return Continue
	})
	return parLev
}

// ParentByName finds first parent recursively up hierarchy that matches
// given name -- returns nil if not found.
func (n *Node) ParentByName(name string) Ki {
	if IsRoot(n) {
		return nil
	}
	if n.Par.Name() == name {
		return n.Par
	}
	return n.Par.ParentByName(name)
}

// ParentByNameTry finds first parent recursively up hierarchy that matches
// given name -- returns error if not found.
func (n *Node) ParentByNameTry(name string) (Ki, error) {
	par := n.ParentByName(name)
	if par != nil {
		return par, nil
	}
	return nil, fmt.Errorf("ki %v: Parent name: %v not found", n.Nm, name)
}

/*

todo: these are not available generically, but are used in several
places in gi -- need to write a Gi-specific version for different
types using the embed logic.

// ParentByType finds parent recursively up hierarchy, by type, and
// returns nil if not found. If embeds is true, then it looks for any
// type that embeds the given type at any level of anonymous embedding.
func (n *Node) ParentByType(t *gti.Type, embeds bool) Ki {
	if IsRoot(n) {
		return nil
	}
	if embeds {
		if TypeEmbeds(n.Par, t) {
			return n.Par
		}
	} else {
		if n.Par.Type() == t {
			return n.Par
		}
	}
	return n.Par.ParentByType(t, embeds)
}

// ParentByTypeTry finds parent recursively up hierarchy, by type, and
// returns error if not found. If embeds is true, then it looks for any
// type that embeds the given type at any level of anonymous embedding.
func (n *Node) ParentByTypeTry(t *gti.Type, embeds bool) (Ki, error) {
	par := n.ParentByType(t, embeds)
	if par != nil {
		return par, nil
	}
	return nil, fmt.Errorf("ki %v: Parent of type: %v not found", n.Nm, t)
}
*/

//////////////////////////////////////////////////////////////////////////
//  Children

// HasChildren tests whether this node has children (i.e., non-terminal).
func (n *Node) HasChildren() bool {
	return len(n.Kids) > 0
}

// NumChildren returns the number of children of this node.
func (n *Node) NumChildren() int {
	return len(n.Kids)
}

// Children returns a pointer to the slice of children (Node.Kids) -- use
// methods on ki.Slice for further ways to access (ByName, ByType, etc).
// Slice can be modified directly (e.g., sort, reorder) but Add* / Delete*
// methods on parent node should be used to ensure proper tracking.
func (n *Node) Children() *Slice {
	return &n.Kids
}

// IsValidIndex returns error if given index is not valid for accessing children
// nil otherwise.
func (n *Node) IsValidIndex(idx int) error {
	sz := len(n.Kids)
	if idx >= 0 && idx < sz {
		return nil
	}
	return fmt.Errorf("ki %v: invalid index: %v -- len = %v", n.Nm, idx, sz)
}

// Child returns the child at given index -- will panic if index is invalid.
// See methods on ki.Slice for more ways to access.
func (n *Node) Child(idx int) Ki {
	return n.Kids[idx]
}

// ChildTry returns the child at given index.  Try version returns error if index is invalid.
// See methods on ki.Slice for more ways to acces.
func (n *Node) ChildTry(idx int) (Ki, error) {
	if err := n.IsValidIndex(idx); err != nil {
		return nil, err
	}
	return n.Kids[idx], nil
}

// ChildByName returns first element that has given name, nil if not found.
// startIdx arg allows for optimized bidirectional find if you have
// an idea where it might be -- can be key speedup for large lists -- pass
// [ki.StartMiddle] to start in the middle (good default).
func (n *Node) ChildByName(name string, startIdx int) Ki {
	return n.Kids.ElemByName(name, startIdx)
}

// ChildByNameTry returns first element that has given name, error if not found.
// startIdx arg allows for optimized bidirectional find if you have
// an idea where it might be -- can be key speedup for large lists -- pass
// [ki.StartMiddle] to start in the middle (good default).
func (n *Node) ChildByNameTry(name string, startIdx int) (Ki, error) {
	idx, ok := n.Kids.IndexByName(name, startIdx)
	if !ok {
		return nil, fmt.Errorf("ki %v: child named: %v not found", n.Nm, name)
	}
	return n.Kids[idx], nil
}

/*

todo: gti

// ChildByType returns first element that has given type, nil if not found.
// If embeds is true, then it looks for any type that embeds the given type
// at any level of anonymous embedding.
// startIdx arg allows for optimized bidirectional find if you have
// an idea where it might be -- can be key speedup for large lists -- pass
// [ki.StartMiddle] to start in the middle (good default).
func (n *Node) ChildByType(t *gti.Type, embeds bool, startIdx int) Ki {
	return n.Kids.ElemByType(t, embeds, startIdx)
}

// ChildByTypeTry returns first element that has given name -- Try version
// returns error message if not found.
// If embeds is true, then it looks for any type that embeds the given type
// at any level of anonymous embedding.
// startIdx arg allows for optimized bidirectional find if you have
// an idea where it might be -- can be key speedup for large lists -- pass
// [ki.StartMiddle] to start in the middle (good default).
func (n *Node) ChildByTypeTry(t *gti.Type, embeds bool, startIdx int) (Ki, error) {
	idx, ok := n.Kids.IndexByType(t, embeds, startIdx)
	if !ok {
		return nil, fmt.Errorf("ki %v: child of type: %t not found", n.Nm, t)
	}
	return n.Kids[idx], nil
}
*/

//////////////////////////////////////////////////////////////////////////
//  Paths

// EscapePathName returns a name that replaces any path delimiter symbols
// . or / with \, and \\ escaped versions.
func EscapePathName(name string) string {
	return strings.Replace(strings.Replace(name, ".", `\,`, -1), "/", `\\`, -1)
}

// UnescapePathName returns a name that replaces any escaped path delimiter symbols
// \, or \\ with . and / unescaped versions.
func UnescapePathName(name string) string {
	return strings.Replace(strings.Replace(name, `\,`, ".", -1), `\\`, "/", -1)
}

// Path returns path to this node from the tree root, using node Names
// separated by / and fields by .
// Node names escape any existing / and . characters to \\ and \,
// Path is only valid when child names are unique (see Unique* functions)
func (n *Node) Path() string {
	if n.Par != nil {
		if n.IsField() {
			return n.Par.Path() + "." + EscapePathName(n.Nm)
		}
		return n.Par.Path() + "/" + EscapePathName(n.Nm)
	}
	return "/" + EscapePathName(n.Nm)
}

// PathFrom returns path to this node from given parent node, using
// node Names separated by / and fields by .
// Node names escape any existing / and . characters to \\ and \,
// Path is only valid for finding items when child names are unique
// (see Unique* functions)
func (n *Node) PathFrom(par Ki) string {
	if n.Par != nil {
		ppath := ""
		if n.Par == par {
			ppath = "/" + EscapePathName(par.Name())
		} else {
			ppath = n.Par.PathFrom(par)
		}
		if n.IsField() {
			return ppath + "." + EscapePathName(n.Nm)
		}
		return ppath + "/" + EscapePathName(n.Nm)
	}
	return "/" + n.Nm
}

// find the child on the path
func findPathChild(k Ki, child string) (int, bool) {
	if child[0] == '[' && child[len(child)-1] == ']' {
		idx, err := strconv.Atoi(child[1 : len(child)-1])
		if err != nil {
			return idx, false
		}
		if idx < 0 { // from end
			idx = len(*k.Children()) + idx
		}
		if k.Children().IsValidIndex(idx) != nil {
			return idx, false
		}
		return idx, true
	}
	return k.Children().IndexByName(child, 0)
}

// FindPath returns Ki object at given path, starting from this node
// (e.g., the root).  If this node is not the root, then the path
// to this node is subtracted from the start of the path if present there.
// FindPath only works correctly when names are unique.
// Path has node Names separated by / and fields by .
// Node names escape any existing / and . characters to \\ and \,
// There is also support for [idx] index-based access for any given path
// element, for cases when indexes are more useful than names.
// Returns nil if not found.
func (n *Node) FindPath(path string) Ki {
	if n.Par != nil { // we are not root..
		myp := n.Path()
		path = strings.TrimPrefix(path, myp)
	}
	curn := Ki(n)
	pels := strings.Split(strings.Trim(strings.TrimSpace(path), "\""), "/")
	for i, pe := range pels {
		if len(pe) == 0 {
			continue
		}
		if i <= 1 && curn.Name() == UnescapePathName(pe) {
			continue
		}
		if strings.Contains(pe, ".") { // has fields
			fels := strings.Split(pe, ".")
			// find the child first, then the fields
			idx, ok := findPathChild(curn, UnescapePathName(fels[0]))
			if !ok {
				return nil
			}
			curn = (*(curn.Children()))[idx]
			for i := 1; i < len(fels); i++ {
				fe := UnescapePathName(fels[i])
				fk, err := curn.FieldByName(fe)
				if err != nil {
					slog.Debug("ki.FindPath: %v", err)
					return nil
				}
				curn = fk
			}
		} else {
			idx, ok := findPathChild(curn, UnescapePathName(pe))
			if !ok {
				return nil
			}
			curn = (*(curn.Children()))[idx]
		}
	}
	return curn
}

// FindPathTry returns Ki object at given path, starting from this node
// (e.g., the root).  If this node is not the root, then the path
// to this node is subtracted from the start of the path if present there.
// FindPath only works correctly when names are unique.
// Path has node Names separated by / and fields by .
// Node names escape any existing / and . characters to \\ and \,
// There is also support for [idx] index-based access for any given path
// element, for cases when indexes are more useful than names.
// Returns error if not found.
func (n *Node) FindPathTry(path string) (Ki, error) {
	fk := n.This().FindPath(path)
	if fk != nil {
		return fk, nil
	}
	return nil, fmt.Errorf("ki %v: element at path: %v not found", n.Nm, path)
}

func (n *Node) FieldByName(field string) (Ki, error) {
	return nil, errors.New("ki.FieldByName: no Ki fields defined for this node")
}

//////////////////////////////////////////////////////////////////////////
//  Adding, Inserting Children

// AddChild adds given child at end of children list.
// The kid node is assumed to not be on another tree (see MoveToParent)
// and the existing name should be unique among children.
// No UpdateStart / End wrapping is done: do that externally as needed.
// Can also call SetFlag(ki.ChildAdded) if notification is needed.
func (n *Node) AddChild(kid Ki) error {
	if err := ThisCheck(n); err != nil {
		return err
	}
	InitNode(kid)
	n.Kids = append(n.Kids, kid)
	SetParent(kid, n.This()) // key to set new parent before deleting: indicates move instead of delete
	return nil
}

// NewChild creates a new child of given type and
// add at end of children list.
// The name should be unique among children.
// No UpdateStart / End wrapping is done: do that externally as needed.
// Can also call SetChildAdded() if notification is needed.
func (n *Node) NewChild(typ *gti.Type, name string) Ki {
	if err := ThisCheck(n); err != nil {
		return nil
	}
	kid := NewOfType(typ)
	InitNode(kid)
	n.Kids = append(n.Kids, kid)
	kid.SetName(name)
	SetParent(kid, n.This())
	return kid
}

// SetChild sets child at given index to be the given item -- if name is
// non-empty then it sets the name of the child as well -- just calls Init
// (or InitName) on the child, and SetParent.
// Names should be unique among children.
// No UpdateStart / End wrapping is done: do that externally as needed.
// Can also call SetChildAdded() if notification is needed.
func (n *Node) SetChild(kid Ki, idx int, name string) error {
	if err := n.Kids.IsValidIndex(idx); err != nil {
		return err
	}
	if name != "" {
		kid.InitName(kid, name)
	} else {
		InitNode(kid)
	}
	n.Kids[idx] = kid
	SetParent(kid, n.This())
	return nil
}

// InsertChild adds given child at position in children list.
// The kid node is assumed to not be on another tree (see MoveToParent)
// and the existing name should be unique among children.
// No UpdateStart / End wrapping is done: do that externally as needed.
// Can also call SetChildAdded() if notification is needed.
func (n *Node) InsertChild(kid Ki, at int) error {
	if err := ThisCheck(n); err != nil {
		return err
	}
	InitNode(kid)
	n.Kids.Insert(kid, at)
	SetParent(kid, n.This())
	return nil
}

// InsertNewChild creates a new child of given type and
// add at position in children list.
// The name should be unique among children.
// No UpdateStart / End wrapping is done: do that externally as needed.
// Can also call SetChildAdded() if notification is needed.
func (n *Node) InsertNewChild(typ *gti.Type, at int, name string) Ki {
	if err := ThisCheck(n); err != nil {
		return nil
	}
	kid := NewOfType(typ)
	InitNode(kid)
	n.Kids.Insert(kid, at)
	kid.SetName(name)
	SetParent(kid, n.This())
	return kid
}

// SetNChildren ensures that there are exactly n children, deleting any
// extra, and creating any new ones, using NewChild with given type and
// naming according to nameStubX where X is the index of the child.
//
// IMPORTANT: returns whether any modifications were made (mods) AND if
// that is true, the result from the corresponding UpdateStart call --
// UpdateEnd is NOT called, allowing for further subsequent updates before
// you call UpdateEnd(updt)
//
// Note that this does not ensure existing children are of given type, or
// change their names -- use ConfigChildren for those cases.
// This function is for simpler cases where a parent uses this function
// consistently to manage children all of the same type.
func (n *Node) SetNChildren(trgn int, typ *gti.Type, nameStub string) (mods, updt bool) {
	mods, updt = false, false
	sz := len(n.Kids)
	if trgn == sz {
		return
	}
	for sz > trgn {
		if !mods {
			mods = true
			updt = n.UpdateStart()
		}
		sz--
		n.DeleteChildAtIndex(sz, true)
	}
	for sz < trgn {
		if !mods {
			mods = true
			updt = n.UpdateStart()
		}
		nm := fmt.Sprintf("%s%d", nameStub, sz)
		n.InsertNewChild(typ, sz, nm)
		sz++
	}
	return
}

// ConfigChildren configures children according to given list of
// type-and-name's -- attempts to have minimal impact relative to existing
// items that fit the type and name constraints (they are moved into the
// corresponding positions), and any extra children are removed, and new
// ones added, to match the specified config.  If uniqNm, then names
// represent UniqueNames (this results in Name == UniqueName for created
// children).
//
// IMPORTANT: returns whether any modifications were made (mods) AND if
// that is true, the result from the corresponding UpdateStart call --
// UpdateEnd is NOT called, allowing for further subsequent updates before
// you call UpdateEnd(updt).
func (n *Node) ConfigChildren(config TypeAndNameList) (mods, updt bool) {
	return n.Kids.Config(n.This(), config)
}

//////////////////////////////////////////////////////////////////////////
//  Deleting Children

// DeleteChildAtIndex deletes child at given index (returns error for
// invalid index).
// Wraps delete in UpdateStart / End and sets ChildDeleted flag.
func (n *Node) DeleteChildAtIndex(idx int, destroy bool) error {
	child, err := n.ChildTry(idx)
	if err != nil {
		return err
	}
	updt := n.UpdateStart()
	n.SetFlag(true, ChildDeleted)
	if child.Parent() == n.This() {
		// only deleting if we are still parent -- change parent first to
		// signal move delete is always sent live to affected node without
		// update blocking note: children of child etc will not send a signal
		// at this point -- only later at destroy -- up to this parent to
		// manage all that
		child.SetFlag(true, NodeDeleted)
		child.NodeSignal().Emit(child, int64(NodeSignalDeleting), nil)
		SetParent(child, nil)
	}
	n.Kids.DeleteAtIndex(idx)
	if destroy {
		DelMgr.Add(child)
	}
	UpdateReset(child) // it won't get the UpdateEnd from us anymore -- init fresh in any case
	n.UpdateEnd(updt)
	return nil
}

// DeleteChild deletes child node, returning error if not found in
// Children.
// Wraps delete in UpdateStart / End and sets ChildDeleted flag.
func (n *Node) DeleteChild(child Ki, destroy bool) error {
	if child == nil {
		return errors.New("ki DeleteChild: child is nil")
	}
	idx, ok := n.Kids.IndexOf(child, 0)
	if !ok {
		return fmt.Errorf("ki %v: child: %v not found", n.Nm, child.Path())
	}
	return n.DeleteChildAtIndex(idx, destroy)
}

// DeleteChildByName deletes child node by name -- returns child, error
// if not found.
// Wraps delete in UpdateStart / End and sets ChildDeleted flag.
func (n *Node) DeleteChildByName(name string, destroy bool) (Ki, error) {
	idx, ok := n.Kids.IndexByName(name, 0)
	if !ok {
		return nil, fmt.Errorf("ki %v: child named: %v not found", n.Nm, name)
	}
	child := n.Kids[idx]
	return child, n.DeleteChildAtIndex(idx, destroy)
}

// DeleteChildren deletes all children nodes -- destroy will add removed
// children to deleted list, to be destroyed later -- otherwise children
// remain intact but parent is nil -- could be inserted elsewhere, but you
// better have kept a slice of them before calling this.
func (n *Node) DeleteChildren(destroy bool) {
	updt := n.UpdateStart()
	n.SetFlag(true, ChildrenDeleted)
	kids := n.Kids
	n.Kids = n.Kids[:0] // preserves capacity of list
	for _, child := range kids {
		if child == nil {
			continue
		}
		child.SetFlag(true, NodeDeleted)
		child.NodeSignal().Emit(child, int64(NodeSignalDeleting), nil)
		SetParent(child, nil)
		UpdateReset(child)
	}
	if destroy {
		DelMgr.Add(kids...)
	}
	n.UpdateEnd(updt)
}

// Delete deletes this node from its parent children list -- destroy will
// add removed child to deleted list, to be destroyed later -- otherwise
// child remains intact but parent is nil -- could be inserted elsewhere.
func (n *Node) Delete(destroy bool) {
	if n.Par == nil {
		if destroy {
			n.This().Destroy()
		}
	} else {
		n.Par.DeleteChild(n.This(), destroy)
	}
}

// Destroy calls DisconnectAll to cut all pointers and signal connections,
// and remove all children and their childrens-children, etc.
func (n *Node) Destroy() {
	// fmt.Printf("Destroying: %v %T %p Kids: %v\n", n.Nm, n.This(), n.This(), len(n.Kids))
	if n.This() == nil { // already dead!
		return
	}
	n.DisconnectAll()
	n.DeleteChildren(true) // first delete all my children
	// and destroy all my fields
	//
	// n.FuncFields(0, nil, func(k Ki, level int, d any) bool {
	// 	k.Destroy()
	// 	return true
	// })
	DelMgr.DestroyDeleted() // then destroy all those kids
	n.SetFlag(true, NodeDestroyed)
	n.Ths = nil // last gasp: lose our own sense of self..
	// note: above is thread-safe because This() accessor checks Destroyed
}

//////////////////////////////////////////////////////////////////////////
//  Flags

// HasFlag checks if flag is set
// using atomic, safe for concurrent access
func (n *Node) HasFlag(f enums.BitFlag) bool {
	return n.Flags.HasFlag(f)
}

// SetFlag sets the given flag(s) to given state
// using atomic, safe for concurrent access
func (n *Node) SetFlag(on bool, f ...enums.BitFlag) {
	n.Flags.SetFlag(on, f...)
}

// IsField checks if this is a field on a parent struct (via IsField
// Flag), as opposed to a child in Children -- Ki nodes can be added as
// fields to structs and they are automatically parented and named with
// field name during Init function -- essentially they function as fixed
// children of the parent struct, and are automatically included in
// FuncDown* traversals, etc -- see also FunFields.
func (n *Node) IsField() bool {
	return n.Flags.HasFlag(IsField)
}

// IsUpdating checks if node is currently updating.
func (n *Node) IsUpdating() bool {
	return n.Flags.HasFlag(Updating)
}

// OnlySelfUpdate checks if this node only applies UpdateStart / End logic
// to itself, not its children (which is the default) (via Flag of same
// name) -- useful for a parent node that has a different function than
// its children.
func (n *Node) OnlySelfUpdate() bool {
	return n.Flags.HasFlag(OnlySelfUpdate)
}

// SetChildAdded sets the ChildAdded flag -- set when notification is needed
// for Add, Insert methods
func (n *Node) SetChildAdded() {
	n.SetFlag(true, ChildAdded)
}

// SetValUpdated sets the ValUpdated flag -- set when notification is needed
// for modifying a value (field, prop, etc)
func (n *Node) SetValUpdated() {
	n.SetFlag(true, ValUpdated)
}

// IsDeleted checks if this node has just been deleted (within last update
// cycle), indicated by the NodeDeleted flag which is set when the node is
// deleted, and is cleared at next UpdateStart call.
func (n *Node) IsDeleted() bool {
	return n.Flags.HasFlag(NodeDeleted)
}

// IsDestroyed checks if this node has been destroyed -- the NodeDestroyed
// flag is set at start of Destroy function -- the Signal Emit process
// checks for destroyed receiver nodes and removes connections to them
// automatically -- other places where pointers to potentially destroyed
// nodes may linger should also check this flag and reset those pointers.
func (n *Node) IsDestroyed() bool {
	return n.Flags.HasFlag(NodeDestroyed)
}

// ClearUpdateFlags resets all structure and value update related flags:
// ChildAdded, ChildDeleted, ChildrenDeleted, NodeDeleted, ValUpdated
// automatically called on StartUpdate to reset any old state.
func (n *Node) ClearUpdateFlags() {
	n.SetFlag(false, ChildAdded, ChildDeleted, ChildrenDeleted, NodeDeleted, ValUpdated)
}

//////////////////////////////////////////////////////////////////////////
//  Property interface with inheritance -- nodes can inherit props from parents

// Properties (Node.Props) tell the GoGi GUI or other frameworks operating
// on Trees about special features of each node -- functions below support
// inheritance up Tree.
func (n *Node) Properties() *Props {
	return &n.Props
}

// SetProp sets given property key to value val.
// initializes property map if nil.
func (n *Node) SetProp(key string, val any) {
	if n.Props == nil {
		n.Props = make(Props)
	}
	n.Props[key] = val
}

// SetProps sets a whole set of properties
func (n *Node) SetProps(props Props) {
	if n.Props == nil {
		n.Props = make(Props, len(props))
	}
	for key, val := range props {
		n.Props[key] = val
	}
}

// Prop returns property value for key that is known to exist.
// Returns nil if it actually doesn't -- this version allows
// direct conversion of return.  See PropTry for version with
// error message if uncertain if property exists.
func (n *Node) Prop(key string) any {
	return n.Props[key]
}

// PropTry returns property value for key.  Returns error message
// if property with that key does not exist.
func (n *Node) PropTry(key string) (any, error) {
	v, ok := n.Props[key]
	if !ok {
		return v, fmt.Errorf("ki.PropTry, could not find property with key %v on node %v", key, n.Nm)
	}
	return v, nil
}

// PropInherit gets property value from key with options for inheriting
// property from parents and / or type-level properties.  If inherit, then
// checks all parents.  If typ then checks property on type as well
// (registered via KiT type registry).  Returns false if not set anywhere.
func (n *Node) PropInherit(key string, inherit, typ bool) (any, bool) {
	// pr := prof.Start("PropInherit")
	// defer pr.End()
	v, ok := n.Props[key]
	if ok {
		return v, ok
	}
	if inherit && n.Par != nil {
		v, ok = n.Par.PropInherit(key, inherit, typ)
		if ok {
			return v, ok
		}
	}
	// todo: use gti type registry here instead
	// if typ {
	// 	return kit.Types.Prop(Type(n.This()), key)
	// }
	return nil, false
}

// DeleteProp deletes property key on this node.
func (n *Node) DeleteProp(key string) {
	if n.Props == nil {
		return
	}
	delete(n.Props, key)
}

// PropTag returns the name to look for in type properties, for types
// that are valid options for values that can be set in Props.  For example
// in GoGi, it is "style-props" which is then set for all types that can
// be used in a style (colors, enum options, etc)
func (n *Node) PropTag() string {
	return ""
}

//////////////////////////////////////////////////////////////////////////
//  Tree walking and state updating

// FuncUp calls function on given node and all the way up to its parents,
// and so on -- sequentially all in current go routine (generally
// necessary for going up, which is typically quite fast anyway) -- level
// is incremented after each step (starts at 0, goes up), and passed to
// function -- returns false if fun aborts with false, else true.
func (n *Node) FuncUp(level int, data any, fun Func) bool {
	cur := n.This()
	for {
		if !fun(cur, level, data) { // false return means stop
			return false
		}
		level++
		par := cur.Parent()
		if par == nil || par == cur { // prevent loops
			return true
		}
		cur = par
	}
	return true
}

// FuncUpParent calls function on parent of node and all the way up to its
// parents, and so on -- sequentially all in current go routine (generally
// necessary for going up, which is typically quite fast anyway) -- level
// is incremented after each step (starts at 0, goes up), and passed to
// function -- returns false if fun aborts with false, else true.
func (n *Node) FuncUpParent(level int, data any, fun Func) bool {
	if IsRoot(n) {
		return true
	}
	cur := n.Parent()
	for {
		if !fun(cur, level, data) { // false return means stop
			return false
		}
		level++
		par := cur.Parent()
		if par == nil || par == cur { // prevent loops
			return true
		}
		cur = par
	}
}

////////////////////////////////////////////////////////////////////////
// FuncDown -- Traversal records

// TravMap is a map for recording the traversal of nodes
type TravMap map[Ki]int

// Start is called at start of traversal
func (tm TravMap) Start(k Ki) {
	tm[k] = -1
}

// End deletes node once done at end of traversal
func (tm TravMap) End(k Ki) {
	delete(tm, k)
}

// Set updates traversal state
func (tm TravMap) Set(k Ki, curChild int) {
	tm[k] = curChild
}

// Get retrieves current traversal state
func (tm TravMap) Get(k Ki) int {
	return tm[k]
}

// strategy -- same as used in TreeView:
// https://stackoverflow.com/questions/5278580/non-recursive-depth-first-search-algorithm

// FuncDownMeFirst calls function on this node (MeFirst) and then iterates
// in a depth-first manner over all the children.
// The node traversal is non-recursive and uses locally-allocated state -- safe
// for concurrent calling (modulo conflict management in function call itself).
// Function calls are sequential all in current go routine.
// The level var tracks overall depth in the tree.
// If fun returns false then any further traversal of that branch of the tree is
// aborted, but other branches continue -- i.e., if fun on current node
// returns false, children are not processed further.
func (n *Node) FuncDownMeFirst(level int, data any, fun Func) {
	if n.This() == nil || n.IsDeleted() {
		return
	}
	tm := TravMap{} // not significantly faster to pre-allocate larger size
	start := n.This()
	cur := start
	tm.Start(cur)
outer:
	for {
		if cur.This() != nil && !cur.IsDeleted() && fun(cur, level, data) { // false return means stop
			level++ // this is the descent branch
			if cur.HasChildren() {
				tm.Set(cur, 0) // 0 for no fields
				nxt := cur.Child(0)
				if nxt != nil && nxt.This() != nil && !nxt.IsDeleted() {
					cur = nxt.This()
					tm.Start(cur)
					continue
				}
			}
		} else {
			tm.Set(cur, cur.NumChildren())
			level++ // we will pop back up out of this next
		}
		// if we get here, we're in the ascent branch -- move to the right and then up
		for {
			curChild := tm.Get(cur)
			if (curChild + 1) < cur.NumChildren() {
				curChild++
				tm.Set(cur, curChild)
				nxt := cur.Child(curChild)
				if nxt != nil && nxt.This() != nil && !nxt.IsDeleted() {
					cur = nxt.This()
					tm.Start(cur)
					continue outer
				}
				continue
			}
			tm.End(cur)
			// couldn't go right, move up..
			if cur == start {
				break outer // done!
			}
			level--
			par := cur.Parent()
			if par == nil || par == cur { // shouldn't happen, but does..
				// fmt.Printf("nil / cur parent %v\n", par)
				break outer
			}
			cur = par
		}
	}
}

// FuncDownMeLast iterates in a depth-first manner over the children, calling
// doChildTestFunc on each node to test if processing should proceed (if it returns
// false then that branch of the tree is not further processed), and then
// calls given fun function after all of a node's children.
// have been iterated over ("Me Last").
// The node traversal is non-recursive and uses locally-allocated state -- safe
// for concurrent calling (modulo conflict management in function call itself).
// Function calls are sequential all in current go routine.
// The level var tracks overall depth in the tree.
func (n *Node) FuncDownMeLast(level int, data any, doChildTestFunc Func, fun Func) {
	if n.This() == nil || n.IsDeleted() {
		return
	}
	tm := TravMap{} // not significantly faster to pre-allocate larger size
	start := n.This()
	cur := start
	tm.Start(cur)
outer:
	for {
		if cur.This() != nil && !cur.IsDeleted() && doChildTestFunc(cur, level, data) { // false return means stop
			level++ // this is the descent branch
			if cur.HasChildren() {
				tm.Set(cur, 0) // 0 for no fields
				nxt := cur.Child(0)
				if nxt != nil && nxt.This() != nil && !nxt.IsDeleted() {
					cur = nxt.This()
					tm.Set(cur, -1)
					continue
				}
			}
		} else {
			tm.Set(cur, cur.NumChildren())
			level++ // we will pop back up out of this next
		}
		// if we get here, we're in the ascent branch -- move to the right and then up
		for {
			curChild := tm.Get(cur)
			if (curChild + 1) < cur.NumChildren() {
				curChild++
				tm.Set(cur, curChild)
				nxt := cur.Child(curChild)
				if nxt != nil && nxt.This() != nil && !nxt.IsDeleted() {
					cur = nxt.This()
					tm.Start(cur)
					continue outer
				}
				continue
			}
			level--
			fun(cur, level, data) // now we call the function, last..
			// couldn't go right, move up..
			tm.End(cur)
			if cur == start {
				break outer // done!
			}
			par := cur.Parent()
			if par == nil || par == cur { // shouldn't happen
				break outer
			}
			cur = par
		}
	}
}

// Note: it does not appear that there is a good recursive BFS search strategy
// https://herringtondarkholme.github.io/2014/02/17/generator/
// https://stackoverflow.com/questions/2549541/performing-breadth-first-search-recursively/2549825#2549825

// FuncDownBreadthFirst calls function on all children in breadth-first order
// using the standard queue strategy.  This depends on and updates the
// Depth parameter of the node.  If fun returns false then any further
// traversal of that branch of the tree is aborted, but other branches continue.
func (n *Node) FuncDownBreadthFirst(level int, data any, fun Func) {
	start := n.This()

	SetDepth(start, level)
	queue := make([]Ki, 1)
	queue[0] = start

	for {
		if len(queue) == 0 {
			break
		}
		cur := queue[0]
		depth := Depth(cur)
		queue = queue[1:]

		if cur.This() != nil && !cur.IsDeleted() && fun(cur, depth, data) { // false return means don't proceed
			for _, k := range *cur.Children() {
				if k != nil && k.This() != nil && !k.IsDeleted() {
					SetDepth(k, depth+1)
					queue = append(queue, k)
				}
			}
		}
	}
}

//////////////////////////////////////////////////////////////////////////
//  State update signaling -- automatically consolidates all changes across
//   levels so there is only one update at highest level of modification
//   All modification starts with UpdateStart() and ends with UpdateEnd()

// after an UpdateEnd, DestroyDeleted is called

// NodeSignal returns the main signal for this node that is used for
// update, child signals.
func (n *Node) NodeSignal() *Signal {
	return &n.NodeSig
}

// UpdateStart should be called when starting to modify the tree (state or
// structure) -- returns whether this node was first to set the Updating
// flag (if so, all children have their Updating flag set -- pass the
// result to UpdateEnd -- automatically determines the highest level
// updated, within the normal top-down updating sequence -- can be called
// multiple times at multiple levels -- it is essential to ensure that all
// such Start's have an End!  Usage:
//
//	updt := n.UpdateStart()
//	... code
//	n.UpdateEnd(updt)
//
// or
//
//	updt := n.UpdateStart()
//	defer n.UpdateEnd(updt)
//	... code
func (n *Node) UpdateStart() bool {
	if n.IsUpdating() || n.IsDestroyed() {
		return false
	}
	if n.OnlySelfUpdate() {
		n.SetFlag(true, Updating)
	} else {
		// pr := prof.Start("ki.Node.UpdateStart")
		n.FuncDownMeFirst(0, nil, func(k Ki, level int, d any) bool {
			if !k.IsUpdating() {
				k.ClearUpdateFlags()
				k.SetFlag(true, Updating)
				return Continue
			}
			return Break // bail -- already updating
		})
		// pr.End()
	}
	return true
}

// UpdateEnd should be called when done updating after an UpdateStart, and
// passed the result of the UpdateStart call -- if this is true, the
// NodeSignalUpdated signal will be emitted and the Updating flag will be
// cleared, and DestroyDeleted called -- otherwise it is a no-op.
func (n *Node) UpdateEnd(updt bool) {
	if !updt {
		return
	}
	if n.IsDestroyed() || n.IsDeleted() {
		return
	}
	if n.HasFlag(ChildDeleted) || n.HasFlag(ChildrenDeleted) {
		DelMgr.DestroyDeleted()
	}
	if n.OnlySelfUpdate() {
		n.SetFlag(false, Updating)
		n.NodeSignal().Emit(n.This(), int64(NodeSignalUpdated), n.Flags)
	} else {
		// pr := prof.Start("ki.Node.UpdateEnd")
		n.FuncDownMeFirst(0, nil, func(k Ki, level int, d any) bool {
			k.SetFlag(false, Updating) // note: could check first and break here but good to ensure all clear
			return true
		})
		// pr.End()
		n.NodeSignal().Emit(n.This(), int64(NodeSignalUpdated), n.Flags)
	}
}

// UpdateEndNoSig is just like UpdateEnd except it does not emit a
// NodeSignalUpdated signal -- use this for situations where updating is
// already known to be in progress and the signal would be redundant.
func (n *Node) UpdateEndNoSig(updt bool) {
	if !updt {
		return
	}
	if n.IsDestroyed() || n.IsDeleted() {
		return
	}
	if n.HasFlag(ChildDeleted) || n.HasFlag(ChildrenDeleted) {
		DelMgr.DestroyDeleted()
	}
	if n.OnlySelfUpdate() {
		n.SetFlag(false, Updating)
		// n.NodeSignal().Emit(n.This(), int64(NodeSignalUpdated), n.Flags())
	} else {
		n.FuncDownMeFirst(0, nil, func(k Ki, level int, d any) bool {
			k.SetFlag(false, Updating) // note: could check first and break here but good to ensure all clear
			return true
		})
		// n.NodeSignal().Emit(n.This(), int64(NodeSignalUpdated), n.Flags())
	}
}

// UpdateSig just emits a NodeSignalUpdated if the Updating flag is not
// set -- use this to trigger an update of a given node when there aren't
// any structural changes and you don't need to prevent any lower-level
// updates -- much more efficient than a pair of UpdateStart /
// UpdateEnd's.  Returns true if an update signal was sent.
func (n *Node) UpdateSig() bool {
	if n.IsUpdating() || n.IsDestroyed() {
		return false
	}
	n.NodeSignal().Emit(n.This(), int64(NodeSignalUpdated), n.Flags)
	return true
}

// Disconnect disconnects this node, by calling DisconnectAll() on
// any Signal or Ki fields.  Any Node that adds a Signal or a Ki Field
// must define an updated version of this method that calls its
// embedded parent's version and then calls DisconnectAll() on
// its Signal and Ki fields.
func (n *Node) Disconnect() {
	n.NodeSig.DisconnectAll()
}

// DisconnectAll disconnects all the way from me down the tree.
func (n *Node) DisconnectAll() {
	n.FuncDownMeFirst(0, nil, func(k Ki, level int, d any) bool {
		k.Disconnect()
		return true
	})
}

//////////////////////////////////////////////////////////////////////////
//  Field Value setting with notification

// note: SetField is in laser -- just call UpdateSig if err == nil to get updating

//////////////////////////////////////////////////////////////////////////
//  Deep Copy / Clone

// note: we use the copy from direction as the receiver is modified whereas the
// from is not and assignment is typically in same direction

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
func (n *Node) CopyFrom(frm Ki) error {
	if frm == nil {
		err := fmt.Errorf("ki.Node CopyFrom into %v -- null 'from' source", n.Path())
		log.Println(err)
		return err
	}
	// todo: see if we want this
	// if Type(n.This()) != Type(frm.This()) {
	// 	err := fmt.Errorf("ki.Node Copy to %v from %v -- must have same types, but %v != %v", n.Path(), frm.Path(), Type(n.This()).Name(), Type(frm.This()).Name())
	// 	log.Println(err)
	// 	return err
	// }
	updt := n.UpdateStart()
	defer n.UpdateEnd(updt)
	err := CopyFromRaw(n.This(), frm)
	return err
}

// Clone creates and returns a deep copy of the tree from this node down.
// Any pointers within the cloned tree will correctly point within the new
// cloned tree (see Copy info).
func (n *Node) Clone() Ki {
	// todo: gti
	// nki := NewOfType(Type(n.This()))
	// nki.InitName(nki, n.Nm)
	// nki.CopyFrom(n.This())
	// return nki
	return nil
}

// CopyFromRaw performs a raw copy that just does the deep copy of the
// bits and doesn't do anything with pointers.
func CopyFromRaw(kn, frm Ki) error {
	kn.Children().ConfigCopy(kn.This(), *frm.Children())
	n := kn.AsNode()
	fmp := *frm.Properties()
	n.Props = make(Props, len(fmp))
	n.Props.CopyFrom(fmp, DeepCopy)

	kn.This().CopyFieldsFrom(frm)
	for i, kid := range *kn.Children() {
		fmk := frm.Child(i)
		CopyFromRaw(kid, fmk)
	}
	return nil
}

// CopyFieldsFrom copies from primary fields of source object,
// recursively following anonymous embedded structs
func (n *Node) CopyFieldsFrom(frm any) {

}
