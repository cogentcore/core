// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package ki

import (
	"bytes"
	"encoding/gob"
	"errors"
	"fmt"
	"strconv"
	"sync/atomic"
	"unsafe"

	"log"
	"reflect"
	"strings"

	"github.com/goki/ki/bitflag"
	"github.com/goki/ki/kit"
	"github.com/jinzhu/copier"
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
	Nm        string    `copy:"-" label:"Name" desc:"Ki.Name() user-supplied name of this node -- can be empty or non-unique"`
	Flag      int64     `tableview:"-" copy:"-" json:"-" xml:"-" max-width:"80" height:"3" desc:"bit flags for internal node state"`
	Props     Props     `tableview:"-" xml:"-" copy:"-" label:"Properties" desc:"Ki.Properties() property map for arbitrary extensible properties, including style properties"`
	Par       Ki        `tableview:"-" copy:"-" json:"-" xml:"-" label:"Parent" view:"-" desc:"Ki.Parent() parent of this node -- set automatically when this node is added as a child of parent"`
	Kids      Slice     `tableview:"-" copy:"-" label:"Children" desc:"Ki.Children() list of children of this node -- all are set to have this node as their parent -- can reorder etc but generally use Ki Node methods to Add / Delete to ensure proper usage"`
	NodeSig   Signal    `copy:"-" json:"-" xml:"-" view:"-" desc:"Ki.NodeSignal() signal for node structure / state changes -- emits NodeSignals signals -- can also extend to custom signals (see signal.go) but in general better to create a new Signal instead"`
	Ths       Ki        `copy:"-" json:"-" xml:"-" view:"-" desc:"we need a pointer to ourselves as a Ki, which can always be used to extract the true underlying type of object when Node is embedded in other structs -- function receivers do not have this ability so this is necessary.  This is set to nil when deleted.  Typically use This() convenience accessor which protects against concurrent access."`
	index     int       `copy:"-" json:"-" xml:"-" view:"-" desc:"last value of our index -- used as a starting point for finding us in our parent next time -- is not guaranteed to be accurate!  use Index() method"`
	depth     int       `copy:"-" json:"-" xml:"-" view:"-" desc:"optional depth parameter of this node -- only valid during specific contexts, not generally -- e.g., used in FuncDownBreadthFirst function"`
	fieldOffs []uintptr `copy:"-" json:"-" xml:"-" view:"-" desc:"cached version of the field offsets relative to base Node address -- used in generic field access."`
}

// must register all new types so type names can be looked up by name -- also props
// EnumType:Flags registers KiT_Flags as type for Flags field for GUI views
// Nodes can also use type properties e.g., StructViewFields key with props
// inside that to set view properties for types, e.g., to hide or show
// some of these base Node flags.
var KiT_Node = kit.Types.AddType(&Node{}, Props{"EnumType:Flag": KiT_Flags})

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

// Embed returns the embedded struct of given type from this node (or nil
// if it does not embed that type, or the type is not a Ki type -- see
// kit.Embed for a generic interface{} version.
func (n *Node) Embed(t reflect.Type) Ki {
	if n == nil {
		return nil
	}
	es := kit.Embed(n.This(), t)
	if es != nil {
		k, ok := es.(Ki)
		if ok {
			return k
		}
		log.Printf("ki.Embed on: %v embedded struct is not a Ki type -- use kit.Embed for a more general version\n", n.Path())
		return nil
	}
	return nil
}

// BaseIface returns the base interface type for all elements
// within this tree.  Use reflect.TypeOf((*<interface_type>)(nil)).Elem().
// Used e.g., for determining what types of children
// can be created (see kit.EmbedImplements for test method)
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
	if idx >= 0 {
		n.index = idx
	}
	return idx, ok
}

// ParentLevel finds a given potential parent node recursively up the
// hierarchy, returning level above current node that the parent was
// found, and -1 if not found.
func (n *Node) ParentLevel(par Ki) int {
	parLev := -1
	n.FuncUpParent(0, n, func(k Ki, level int, d interface{}) bool {
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

// ParentByType finds parent recursively up hierarchy, by type, and
// returns nil if not found. If embeds is true, then it looks for any
// type that embeds the given type at any level of anonymous embedding.
func (n *Node) ParentByType(t reflect.Type, embeds bool) Ki {
	if IsRoot(n) {
		return nil
	}
	if embeds {
		if TypeEmbeds(n.Par, t) {
			return n.Par
		}
	} else {
		if Type(n.Par) == t {
			return n.Par
		}
	}
	return n.Par.ParentByType(t, embeds)
}

// ParentByTypeTry finds parent recursively up hierarchy, by type, and
// returns error if not found. If embeds is true, then it looks for any
// type that embeds the given type at any level of anonymous embedding.
func (n *Node) ParentByTypeTry(t reflect.Type, embeds bool) (Ki, error) {
	par := n.ParentByType(t, embeds)
	if par != nil {
		return par, nil
	}
	return nil, fmt.Errorf("ki %v: Parent of type: %v not found", n.Nm, t)
}

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
// -1 to start in the middle (good default).
func (n *Node) ChildByName(name string, startIdx int) Ki {
	return n.Kids.ElemByName(name, startIdx)
}

// ChildByNameTry returns first element that has given name, error if not found.
// startIdx arg allows for optimized bidirectional find if you have
// an idea where it might be -- can be key speedup for large lists -- pass
// -1 to start in the middle (good default).
func (n *Node) ChildByNameTry(name string, startIdx int) (Ki, error) {
	idx, ok := n.Kids.IndexByName(name, startIdx)
	if !ok {
		return nil, fmt.Errorf("ki %v: child named: %v not found", n.Nm, name)
	}
	return n.Kids[idx], nil
}

// ChildByType returns first element that has given type, nil if not found.
// If embeds is true, then it looks for any type that embeds the given type
// at any level of anonymous embedding.
// startIdx arg allows for optimized bidirectional find if you have
// an idea where it might be -- can be key speedup for large lists -- pass
// -1 to start in the middle (good default).
func (n *Node) ChildByType(t reflect.Type, embeds bool, startIdx int) Ki {
	return n.Kids.ElemByType(t, embeds, startIdx)
}

// ChildByTypeTry returns first element that has given name -- Try version
// returns error message if not found.
// If embeds is true, then it looks for any type that embeds the given type
// at any level of anonymous embedding.
// startIdx arg allows for optimized bidirectional find if you have
// an idea where it might be -- can be key speedup for large lists -- pass
// -1 to start in the middle (good default).
func (n *Node) ChildByTypeTry(t reflect.Type, embeds bool, startIdx int) (Ki, error) {
	idx, ok := n.Kids.IndexByType(t, embeds, startIdx)
	if !ok {
		return nil, fmt.Errorf("ki %v: child of type: %t not found", n.Nm, t)
	}
	return n.Kids[idx], nil
}

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
				fk := KiFieldByName(curn.AsNode(), fe)
				if fk == nil {
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

// AddNewChild creates a new child of given type and
// add at end of children list.
// The name should be unique among children.
// No UpdateStart / End wrapping is done: do that externally as needed.
// Can also call SetChildAdded() if notification is needed.
func (n *Node) AddNewChild(typ reflect.Type, name string) Ki {
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
func (n *Node) InsertNewChild(typ reflect.Type, at int, name string) Ki {
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
// extra, and creating any new ones, using AddNewChild with given type and
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
func (n *Node) SetNChildren(trgn int, typ reflect.Type, nameStub string) (mods, updt bool) {
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
func (n *Node) ConfigChildren(config kit.TypeAndNameList) (mods, updt bool) {
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
	n.SetFlag(int(ChildDeleted))
	if child.Parent() == n.This() {
		// only deleting if we are still parent -- change parent first to
		// signal move delete is always sent live to affected node without
		// update blocking note: children of child etc will not send a signal
		// at this point -- only later at destroy -- up to this parent to
		// manage all that
		child.SetFlag(int(NodeDeleted))
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
	n.SetFlag(int(ChildrenDeleted))
	for _, child := range n.Kids {
		if child == nil {
			continue
		}
		child.SetFlag(int(NodeDeleted))
		child.NodeSignal().Emit(child, int64(NodeSignalDeleting), nil)
		SetParent(child, nil)
		UpdateReset(child)
	}
	if destroy {
		DelMgr.Add(n.Kids...)
	}
	n.Kids = n.Kids[:0] // preserves capacity of list
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
	n.FuncFields(0, nil, func(k Ki, level int, d interface{}) bool {
		k.Destroy()
		return true
	})
	DelMgr.DestroyDeleted() // then destroy all those kids
	n.SetFlag(int(NodeDestroyed))
	n.Ths = nil // last gasp: lose our own sense of self..
	// note: above is thread-safe because This() accessor checks Destroyed
}

//////////////////////////////////////////////////////////////////////////
//  Flags

// Flags returns an atomically safe copy of the bit flags for this node --
// can use bitflag package to check lags.
// See Flags type for standard values used in Ki Node --
// can be extended from FlagsN up to 64 bit capacity.
// Note that we must always use atomic access as *some* things need to be atomic,
// and with bits, that means that *all* access needs to be atomic,
// as you cannot atomically update just a single bit.
func (n *Node) Flags() int64 {
	return atomic.LoadInt64(&n.Flag)
}

// HasFlag checks if flag is set
// using atomic, safe for concurrent access
func (n *Node) HasFlag(flag int) bool {
	return bitflag.HasAtomic(&n.Flag, flag)
}

// SetFlag sets the given flag(s)
// using atomic, safe for concurrent access
func (n *Node) SetFlag(flag ...int) {
	bitflag.SetAtomic(&n.Flag, flag...)
}

// SetFlagState sets the given flag(s) to given state
// using atomic, safe for concurrent access
func (n *Node) SetFlagState(on bool, flag ...int) {
	bitflag.SetStateAtomic(&n.Flag, on, flag...)
}

// SetFlagMask sets the given flags as a mask
// using atomic, safe for concurrent access
func (n *Node) SetFlagMask(mask int64) {
	bitflag.SetMaskAtomic(&n.Flag, mask)
}

// ClearFlag clears the given flag(s)
// using atomic, safe for concurrent access
func (n *Node) ClearFlag(flag ...int) {
	bitflag.ClearAtomic(&n.Flag, flag...)
}

// ClearFlagMask clears the given flags as a bitmask
// using atomic, safe for concurrent access
func (n *Node) ClearFlagMask(mask int64) {
	bitflag.ClearMaskAtomic(&n.Flag, mask)
}

// IsField checks if this is a field on a parent struct (via IsField
// Flag), as opposed to a child in Children -- Ki nodes can be added as
// fields to structs and they are automatically parented and named with
// field name during Init function -- essentially they function as fixed
// children of the parent struct, and are automatically included in
// FuncDown* traversals, etc -- see also FunFields.
func (n *Node) IsField() bool {
	return bitflag.HasAtomic(&n.Flag, int(IsField))
}

// IsUpdating checks if node is currently updating.
func (n *Node) IsUpdating() bool {
	return bitflag.HasAtomic(&n.Flag, int(Updating))
}

// OnlySelfUpdate checks if this node only applies UpdateStart / End logic
// to itself, not its children (which is the default) (via Flag of same
// name) -- useful for a parent node that has a different function than
// its children.
func (n *Node) OnlySelfUpdate() bool {
	return bitflag.HasAtomic(&n.Flag, int(OnlySelfUpdate))
}

// SetOnlySelfUpdate sets the OnlySelfUpdate flag -- see OnlySelfUpdate
// method and flag.
func (n *Node) SetOnlySelfUpdate() {
	n.SetFlag(int(OnlySelfUpdate))
}

// SetChildAdded sets the ChildAdded flag -- set when notification is needed
// for Add, Insert methods
func (n *Node) SetChildAdded() {
	n.SetFlag(int(ChildAdded))
}

// SetValUpdated sets the ValUpdated flag -- set when notification is needed
// for modifying a value (field, prop, etc)
func (n *Node) SetValUpdated() {
	n.SetFlag(int(ValUpdated))
}

// IsDeleted checks if this node has just been deleted (within last update
// cycle), indicated by the NodeDeleted flag which is set when the node is
// deleted, and is cleared at next UpdateStart call.
func (n *Node) IsDeleted() bool {
	return bitflag.HasAtomic(&n.Flag, int(NodeDeleted))
}

// IsDestroyed checks if this node has been destroyed -- the NodeDestroyed
// flag is set at start of Destroy function -- the Signal Emit process
// checks for destroyed receiver nodes and removes connections to them
// automatically -- other places where pointers to potentially destroyed
// nodes may linger should also check this flag and reset those pointers.
func (n *Node) IsDestroyed() bool {
	return bitflag.HasAtomic(&n.Flag, int(NodeDestroyed))
}

//////////////////////////////////////////////////////////////////////////
//  Property interface with inheritance -- nodes can inherit props from parents

// Properties (Node.Props) tell the GoGi GUI or other frameworks operating
// on Trees about special features of each node -- functions below support
// inheritance up Tree -- see kit convert.go for robust convenience
// methods for converting interface{} values to standard types.
func (n *Node) Properties() *Props {
	return &n.Props
}

// SetProp sets given property key to value val.
// initializes property map if nil.
func (n *Node) SetProp(key string, val interface{}) {
	if n.Props == nil {
		n.Props = make(Props)
	}
	n.Props[key] = val
}

// SetPropStr sets given property key to value val as a string (e.g., for python wrapper)
// Initializes property map if nil.
func (n *Node) SetPropStr(key string, val string) {
	n.SetProp(key, val)
}

// SetPropInt sets given property key to value val as an int (e.g., for python wrapper)
// Initializes property map if nil.
func (n *Node) SetPropInt(key string, val int) {
	n.SetProp(key, val)
}

// SetPropFloat64 sets given property key to value val as a float64 (e.g., for python wrapper)
// Initializes property map if nil.
func (n *Node) SetPropFloat64(key string, val float64) {
	n.SetProp(key, val)
}

// SetSubProps sets given property key to sub-Props value (e.g., for python wrapper)
// Initializes property map if nil.
func (n *Node) SetSubProps(key string, val Props) {
	n.SetProp(key, val)
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
func (n *Node) Prop(key string) interface{} {
	return n.Props[key]
}

// PropTry returns property value for key.  Returns error message
// if property with that key does not exist.
func (n *Node) PropTry(key string) (interface{}, error) {
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
func (n *Node) PropInherit(key string, inherit, typ bool) (interface{}, bool) {
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
	if typ {
		return kit.Types.Prop(Type(n.This()), key)
	}
	return nil, false
}

// DeleteProp deletes property key on this node.
func (n *Node) DeleteProp(key string) {
	if n.Props == nil {
		return
	}
	delete(n.Props, key)
}

func init() {
	gob.Register(Props{})
}

// CopyPropsFrom copies our properties from another node -- if deep then
// does a deep copy -- otherwise copied map just points to same values in
// the original map (and we don't reset our map first -- call
// DeleteAllProps to do that -- deep copy uses gob encode / decode --
// usually not needed).
func (n *Node) CopyPropsFrom(frm Ki, deep bool) error {
	if *(frm.Properties()) == nil {
		return nil
	}
	// pr := prof.Start("CopyPropsFrom")
	// defer pr.End()
	if n.Props == nil {
		n.Props = make(Props)
	}
	fmP := *(frm.Properties())
	if deep {
		// code from https://gist.github.com/soroushjp/0ec92102641ddfc3ad5515ca76405f4d
		var buf bytes.Buffer
		enc := gob.NewEncoder(&buf)
		dec := gob.NewDecoder(&buf)
		err := enc.Encode(fmP)
		if err != nil {
			return err
		}
		err = dec.Decode(&n.Props)
		if err != nil {
			return err
		}
		return nil
	}
	for k, v := range fmP {
		n.Props[k] = v
	}
	return nil
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

// FlatFieldsValueFunc is the Node version of this function from kit/embeds.go
// it is very slow and should be avoided at all costs!
func FlatFieldsValueFunc(stru interface{}, fun func(stru interface{}, typ reflect.Type, field reflect.StructField, fieldVal reflect.Value) bool) bool {
	v := kit.NonPtrValue(reflect.ValueOf(stru))
	typ := v.Type()
	if typ == nil || typ == KiT_Node { // this is only diff from embeds.go version -- prevent processing of any Node fields
		return true
	}
	rval := true
	for i := 0; i < typ.NumField(); i++ {
		f := typ.Field(i)
		vf := v.Field(i)
		if !vf.CanInterface() {
			continue
		}
		vfi := vf.Interface() // todo: check for interfaceablity etc
		if vfi == nil || vfi == stru {
			continue
		}
		if f.Type.Kind() == reflect.Struct && f.Anonymous && kit.PtrType(f.Type) != KiT_Node {
			rval = FlatFieldsValueFunc(kit.PtrValue(vf).Interface(), fun)
			if !rval {
				break
			}
		} else {
			rval = fun(vfi, typ, f, vf)
			if !rval {
				break
			}
		}
	}
	return rval
}

// FuncFields calls function on all Ki fields within this node.
func (n *Node) FuncFields(level int, data interface{}, fun Func) {
	if n.This() == nil {
		return
	}
	op := uintptr(unsafe.Pointer(n))
	foffs := KiFieldOffs(n)
	for _, fo := range foffs {
		fn := (*Node)(unsafe.Pointer(op + fo))
		fun(fn.This(), level, data)
	}
}

// FuncUp calls function on given node and all the way up to its parents,
// and so on -- sequentially all in current go routine (generally
// necessary for going up, which is typically quite fast anyway) -- level
// is incremented after each step (starts at 0, goes up), and passed to
// function -- returns false if fun aborts with false, else true.
func (n *Node) FuncUp(level int, data interface{}, fun Func) bool {
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
func (n *Node) FuncUpParent(level int, data interface{}, fun Func) bool {
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

// TravIdxs are tree traversal indexes
type TravIdxs struct {
	Field int `desc:"current index of field: -1 for start"`
	Child int `desc:"current index of children: -1 for start"`
}

// TravMap is a map for recording the traversal of nodes
type TravMap map[Ki]TravIdxs

// Start is called at start of traversal
func (tm TravMap) Start(k Ki) {
	tm[k] = TravIdxs{-1, -1}
}

// End deletes node once done at end of traversal
func (tm TravMap) End(k Ki) {
	delete(tm, k)
}

// Set updates traversal state
func (tm TravMap) Set(k Ki, curField, curChild int) {
	tm[k] = TravIdxs{curField, curChild}
}

// Get retrieves current traversal state
func (tm TravMap) Get(k Ki) (curField, curChild int) {
	tr := tm[k]
	return tr.Field, tr.Child
}

// strategy -- same as used in TreeView:
// https://stackoverflow.com/questions/5278580/non-recursive-depth-first-search-algorithm

// FuncDownMeFirst calls function on this node (MeFirst) and then iterates
// in a depth-first manner over all the children, including Ki Node fields,
// which are processed first before children.
// The node traversal is non-recursive and uses locally-allocated state -- safe
// for concurrent calling (modulo conflict management in function call itself).
// Function calls are sequential all in current go routine.
// The level var tracks overall depth in the tree.
// If fun returns false then any further traversal of that branch of the tree is
// aborted, but other branches continue -- i.e., if fun on current node
// returns false, children are not processed further.
func (n *Node) FuncDownMeFirst(level int, data interface{}, fun Func) {
	if n.This() == nil {
		return
	}
	tm := TravMap{} // not significantly faster to pre-allocate larger size
	start := n.This()
	cur := start
	tm.Start(cur)
outer:
	for {
		if cur.This() != nil && fun(cur, level, data) { // false return means stop
			level++ // this is the descent branch
			if KiHasKiFields(cur.AsNode()) {
				tm.Set(cur, 0, -1)
				nxt := KiField(cur.AsNode(), 0).This()
				if nxt != nil {
					cur = nxt
					tm.Start(cur)
					continue
				}
			}
			if cur.HasChildren() {
				tm.Set(cur, 0, 0) // 0 for no fields
				nxt := cur.Child(0)
				if nxt != nil && nxt.This() != nil {
					cur = nxt.This()
					tm.Start(cur)
					continue
				}
			}
		} else {
			tm.Set(cur, NumKiFields(cur.AsNode()), cur.NumChildren())
			level++ // we will pop back up out of this next
		}
		// if we get here, we're in the ascent branch -- move to the right and then up
		for {
			curField, curChild := tm.Get(cur)
			if KiHasKiFields(cur.AsNode()) {
				if (curField + 1) < NumKiFields(cur.AsNode()) {
					curField++
					tm.Set(cur, curField, curChild)
					nxt := KiField(cur.AsNode(), curField).This()
					if nxt != nil {
						cur = nxt
						tm.Start(cur)
						continue outer
					}
					continue
				}
			}
			if (curChild + 1) < cur.NumChildren() {
				curChild++
				tm.Set(cur, curField, curChild)
				nxt := cur.Child(curChild)
				if nxt != nil && nxt.This() != nil {
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
// calls given fun function after all of a node's children (including fields)
// have been iterated over ("Me Last").
// The node traversal is non-recursive and uses locally-allocated state -- safe
// for concurrent calling (modulo conflict management in function call itself).
// Function calls are sequential all in current go routine.
// The level var tracks overall depth in the tree.
func (n *Node) FuncDownMeLast(level int, data interface{}, doChildTestFunc Func, fun Func) {
	if n.This() == nil {
		return
	}
	tm := TravMap{} // not significantly faster to pre-allocate larger size
	start := n.This()
	cur := start
	tm.Start(cur)
outer:
	for {
		if cur.This() != nil && doChildTestFunc(cur, level, data) { // false return means stop
			level++ // this is the descent branch
			if KiHasKiFields(cur.AsNode()) {
				tm.Set(cur, 0, -1)
				nxt := KiField(cur.AsNode(), 0).This()
				if nxt != nil {
					cur = nxt
					tm.Set(cur, -1, -1)
					continue
				}
			}
			if cur.HasChildren() {
				tm.Set(cur, 0, 0) // 0 for no fields
				nxt := cur.Child(0)
				if nxt != nil && nxt.This() != nil {
					cur = nxt.This()
					tm.Set(cur, -1, -1)
					continue
				}
			}
		} else {
			tm.Set(cur, NumKiFields(cur.AsNode()), cur.NumChildren())
			level++ // we will pop back up out of this next
		}
		// if we get here, we're in the ascent branch -- move to the right and then up
		for {
			curField, curChild := tm.Get(cur)
			if KiHasKiFields(cur.AsNode()) {
				if (curField + 1) < NumKiFields(cur.AsNode()) {
					curField++
					tm.Set(cur, curField, curChild)
					nxt := KiField(cur.AsNode(), curField).This()
					if nxt != nil {
						cur = nxt
						tm.Set(cur, -1, -1)
						continue outer
					}
					continue
				}
			}
			if (curChild + 1) < cur.NumChildren() {
				curChild++
				tm.Set(cur, curField, curChild)
				nxt := cur.Child(curChild)
				if nxt != nil && nxt.This() != nil {
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
func (n *Node) FuncDownBreadthFirst(level int, data interface{}, fun Func) {
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

		if cur.This() != nil && fun(cur, depth, data) { // false return means don't proceed
			if KiHasKiFields(cur.AsNode()) {
				cur.FuncFields(depth+1, data, func(k Ki, level int, d interface{}) bool {
					SetDepth(k, level)
					queue = append(queue, k)
					return true
				})
			}
			for _, k := range *cur.Children() {
				if k != nil && k.This() != nil {
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
//   updt := n.UpdateStart()
//   ... code
//   n.UpdateEnd(updt)
// or
//   updt := n.UpdateStart()
//   defer n.UpdateEnd(updt)
//   ... code
func (n *Node) UpdateStart() bool {
	if n.IsUpdating() || n.IsDestroyed() {
		return false
	}
	if n.OnlySelfUpdate() {
		n.SetFlag(int(Updating))
	} else {
		// pr := prof.Start("ki.Node.UpdateStart")
		n.FuncDownMeFirst(0, nil, func(k Ki, level int, d interface{}) bool {
			if !k.IsUpdating() {
				k.ClearFlagMask(int64(UpdateFlagsMask))
				k.SetFlag(int(Updating))
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
	if bitflag.HasAnyAtomic(&n.Flag, int(ChildDeleted), int(ChildrenDeleted)) {
		DelMgr.DestroyDeleted()
	}
	if n.OnlySelfUpdate() {
		n.ClearFlag(int(Updating))
		n.NodeSignal().Emit(n.This(), int64(NodeSignalUpdated), n.Flags())
	} else {
		// pr := prof.Start("ki.Node.UpdateEnd")
		n.FuncDownMeFirst(0, nil, func(k Ki, level int, d interface{}) bool {
			k.ClearFlag(int(Updating)) // note: could check first and break here but good to ensure all clear
			return true
		})
		// pr.End()
		n.NodeSignal().Emit(n.This(), int64(NodeSignalUpdated), n.Flags())
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
	if bitflag.HasAnyAtomic(&n.Flag, int(ChildDeleted), int(ChildrenDeleted)) {
		DelMgr.DestroyDeleted()
	}
	if n.OnlySelfUpdate() {
		n.ClearFlag(int(Updating))
		// n.NodeSignal().Emit(n.This(), int64(NodeSignalUpdated), n.Flags())
	} else {
		n.FuncDownMeFirst(0, nil, func(k Ki, level int, d interface{}) bool {
			k.ClearFlag(int(Updating)) // note: could check first and break here but good to ensure all clear
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
	n.NodeSignal().Emit(n.This(), int64(NodeSignalUpdated), n.Flags())
	return true
}

// Disconnect disconnects this node, by calling DisconnectAll() on
// any Signal fields.  Any Node that adds a Signal must define an
// updated version of this method that calls its embedded parent's
// version and then calls DisconnectAll() on its Signal fields.
func (n *Node) Disconnect() {
	n.NodeSig.DisconnectAll()
}

// DisconnectAll disconnects all the way from me down the tree.
func (n *Node) DisconnectAll() {
	n.FuncDownMeFirst(0, nil, func(k Ki, level int, d interface{}) bool {
		k.Disconnect()
		return true
	})
}

//////////////////////////////////////////////////////////////////////////
//  Field Value setting with notification

// SetField sets given field name to given value, using very robust
// conversion routines to e.g., convert from strings to numbers, and
// vice-versa, automatically.  Returns error if not successfully set.
// wrapped in UpdateStart / End and sets the ValUpdated flag.
func (n *Node) SetField(field string, val interface{}) error {
	fv := kit.FlatFieldValueByName(n.This(), field)
	if !fv.IsValid() {
		return fmt.Errorf("ki.SetField, could not find field %v on node %v", field, n.Nm)
	}
	updt := n.UpdateStart()
	var err error
	if field == "Nm" {
		n.SetName(kit.ToString(val))
		n.SetValUpdated()
	} else {
		if kit.SetRobust(kit.PtrValue(fv).Interface(), val) {
			n.SetValUpdated()
		} else {
			err = fmt.Errorf("ki.SetField, SetRobust failed to set field %v on node %v to value: %v", field, n.Nm, val)
		}
	}
	n.UpdateEnd(updt)
	return err
}

//////////////////////////////////////////////////////////////////////////
//  Deep Copy / Clone

// note: we use the copy from direction as the receiver is modifed whereas the
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
	if Type(n.This()) != Type(frm.This()) {
		err := fmt.Errorf("ki.Node Copy to %v from %v -- must have same types, but %v != %v", n.Path(), frm.Path(), Type(n.This()).Name(), Type(frm.This()).Name())
		log.Println(err)
		return err
	}
	updt := n.UpdateStart()
	defer n.UpdateEnd(updt)
	err := CopyFromRaw(n.This(), frm)
	return err
}

// Clone creates and returns a deep copy of the tree from this node down.
// Any pointers within the cloned tree will correctly point within the new
// cloned tree (see Copy info).
func (n *Node) Clone() Ki {
	nki := NewOfType(Type(n.This()))
	nki.InitName(nki, n.Nm)
	nki.CopyFrom(n.This())
	return nki
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
func (n *Node) CopyFieldsFrom(frm interface{}) {
	GenCopyFieldsFrom(n.This(), frm)
}

// GenCopyFieldsFrom is a general-purpose copy of primary fields
// of source object, recursively following anonymous embedded structs
func GenCopyFieldsFrom(to interface{}, frm interface{}) {
	// pr := prof.Start("GenCopyFieldsFrom")
	// defer pr.End()
	kitype := KiType
	tv := kit.NonPtrValue(reflect.ValueOf(to))
	sv := kit.NonPtrValue(reflect.ValueOf(frm))
	typ := tv.Type()
	if kit.ShortTypeName(typ) == "ki.Node" {
		return // nothing to copy for base node!
	}
	for i := 0; i < typ.NumField(); i++ {
		f := typ.Field(i)
		tf := tv.Field(i)
		if !tf.CanInterface() {
			continue
		}
		ctag := f.Tag.Get("copy")
		if ctag == "-" {
			continue
		}
		sf := sv.Field(i)
		tfpi := kit.PtrValue(tf).Interface()
		sfpi := kit.PtrValue(sf).Interface()
		if f.Type.Kind() == reflect.Struct && f.Anonymous {
			// the generic version cannot ever go back to the node-specific
			// because the n.This() is ALWAYS the final type, not the intermediate
			// embedded ones
			GenCopyFieldsFrom(tfpi, sfpi)
		} else {
			switch {
			case sf.Kind() == reflect.Struct && kit.EmbedImplements(sf.Type(), kitype):
				sfk := sfpi.(Ki)
				tfk := tfpi.(Ki)
				if tfk != nil && sfk != nil {
					tfk.CopyFrom(sfk)
				}
			case f.Type == KiT_Signal: // note: don't copy signals by default
			case sf.Type().AssignableTo(tf.Type()):
				tf.Set(sf)
				// kit.PtrValue(tf).Set(sf)
			default:
				// use copier https://github.com/jinzhu/copier which handles as much as possible..
				// pr := prof.Start("Copier")
				copier.Copy(tfpi, sfpi)
				// pr.End()
			}
		}

	}
}
