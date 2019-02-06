// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Ki is the base element of GoKi Trees
// Ki = Tree in Japanese, and "Key" in English

package ki

import (
	"bytes"
	"encoding/gob"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strconv"
	"sync"
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
	Nm       string `copy:"-" label:"Name" desc:"Ki.Name() user-supplied name of this node -- can be empty or non-unique"`
	UniqueNm string `copy:"-" label:"UniqueName" desc:"Ki.UniqueName() automatically-updated version of Name that is guaranteed to be unique within the slice of Children within one Node -- used e.g., for saving Unique Paths in Ptr pointers"`
	Flag     int64  `copy:"-" json:"-" xml:"-" view:"-" desc:"bit flags for internal node state"`
	Props    Props  `xml:"-" copy:"-" label:"Properties" desc:"Ki.Properties() property map for arbitrary extensible properties, including style properties"`
	Par      Ki     `copy:"-" json:"-" xml:"-" label:"Parent" view:"-" desc:"Ki.Parent() parent of this node -- set automatically when this node is added as a child of parent"`
	Kids     Slice  `copy:"-" label:"Children" desc:"Ki.Children() list of children of this node -- all are set to have this node as their parent -- can reorder etc but generally use Ki Node methods to Add / Delete to ensure proper usage"`
	NodeSig  Signal `copy:"-" json:"-" xml:"-" desc:"Ki.NodeSignal() signal for node structure / state changes -- emits NodeSignals signals -- can also extend to custom signals (see signal.go) but in general better to create a new Signal instead"`
	Ths      Ki     `copy:"-" json:"-" xml:"-" view:"-" desc:"we need a pointer to ourselves as a Ki, which can always be used to extract the true underlying type of object when Node is embedded in other structs -- function receivers do not have this ability so this is necessary.  This is set to nil when deleted.  Typically use This() convenience accessor which protects against concurrent access."`
	index    int    `desc:"last value of our index -- used as a starting point for finding us in our parent next time -- is not guaranteed to be accurate!  use Index() method"`
}

// must register all new types so type names can be looked up by name -- also props
var KiT_Node = kit.Types.AddType(&Node{}, nil)

//////////////////////////////////////////////////////////////////////////
//  fmt.Stringer

// String implements the fmt.stringer interface -- returns the PathUnique of the node
func (n Node) String() string {
	return n.PathUnique()
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

// Init initializes the node -- automatically called during Add/Insert
// Child -- sets the This pointer for this node as a Ki interface (pass
// pointer to node as this arg) -- Go cannot always access the true
// underlying type for structs using embedded Ki objects (when these objs
// are receivers to methods) so we need a This interface pointer that
// guarantees access to the Ki interface in a way that always reveals the
// underlying type (e.g., in reflect calls).  Calls Init on Ki fields
// within struct, sets their names to the field name, and sets us as their
// parent.
func (n *Node) Init(this Ki) {
	kitype := KiType
	n.ClearFlagMask(int64(UpdateFlagsMask))
	if n.Ths != this {
		n.Ths = this
		// we need to call this directly instead of FuncFields because we need the field name
		FlatFieldsValueFunc(this, func(stru interface{}, typ reflect.Type, field reflect.StructField, fieldVal reflect.Value) bool {
			if fieldVal.Kind() == reflect.Struct && kit.EmbedImplements(field.Type, kitype) {
				fk := kit.PtrValue(fieldVal).Interface().(Ki)
				if fk != nil {
					fk.SetFlag(int(IsField))
					fk.InitName(fk, field.Name)
					fk.SetParent(this)
				}
			}
			return true
		})
	}
}

// InitName initializes this node and set its name -- used for root nodes
// which don't otherwise have their This pointer set (otherwise typically
// happens in Add, Insert Child).
func (n *Node) InitName(k Ki, name string) {
	n.Init(k)
	n.SetName(name)
}

// ThisCheck checks that the This pointer is set and issues a warning to
// log if not -- returns error if not set -- called when nodes are added
// and inserted.
func (n *Node) ThisCheck() error {
	if n.This() == nil {
		err := fmt.Errorf("Ki Node %v ThisCheck: node has null 'this' pointer -- must call Init or InitName on root nodes!", n.PathUnique())
		log.Print(err)
		return err
	}
	return nil
}

// Type returns the underlying struct type of this node
// (reflect.TypeOf(This).Elem()).
func (n *Node) Type() reflect.Type {
	return reflect.TypeOf(n.This()).Elem()
}

// TypeEmbeds tests whether this node is of the given type, or it embeds
// that type at any level of anonymous embedding -- use Embed to get the
// embedded struct of that type from this node.
func (n *Node) TypeEmbeds(t reflect.Type) bool {
	return kit.TypeEmbeds(n.Type(), t)
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
		log.Printf("ki.Embed on: %v embedded struct is not a Ki type -- use kit.Embed for a more general version\n", n.PathUnique())
		return nil
	}
	return nil
}

// BaseIface returns the 	base interface type for all elements
// within this tree.  Use reflect.TypeOf((*<interface_type>)(nil)).Elem().
// Used e.g., for determining what types of children
// can be created (see kit.EmbedImplements for test method)
func (n *Node) BaseIface() reflect.Type {
	return KiType
}

// Name returns the user-defined name of the object (Node.Nm), for finding
// elements, generating paths, IO, etc -- allows generic GUI / Text / Path
// / etc representation of Trees.
func (n *Node) Name() string {
	return n.Nm
}

// UniqueName returns a name that is guaranteed to be non-empty and unique
// within the children of this node (Node.UniqueNm), but starts with Name
// or parents name if Name is empty -- important for generating unique
// paths to definitively locate a given node in the tree (see PathUnique,
// FindPathUnique).
func (n *Node) UniqueName() string {
	return n.UniqueNm
}

// SetName sets the name of this node, and its unique name based on this
// name, such that all names are unique within list of siblings of this
// node (somewhat expensive but important, unless you definitely know that
// the names are unique -- see SetNameRaw).  Does nothing if name is
// already set to that value -- returns false in that case.  Does NOT
// wrap in UpdateStart / End.
func (n *Node) SetName(name string) bool {
	if n.Nm == name {
		return false
	}
	n.Nm = name
	n.SetUniqueName(name)
	if n.Par != nil {
		n.Par.UniquifyNames()
	}
	return true
}

// SetNameRaw just sets the name and doesn't update the unique name --
// only use if also/ setting unique names in some other way that is
// guaranteed to be unique.
func (n *Node) SetNameRaw(name string) {
	n.Nm = name
}

// SetUniqueName sets the unique name of this node based on given name
// string -- does not do any further testing that the name is indeed
// unique -- should generally only be used by UniquifyNames.
func (n *Node) SetUniqueName(name string) {
	n.UniqueNm = strings.Replace(strings.Replace(name, ".", "_", -1), "/", "_", -1)
}

// UniquifyPreserveNameLimit is the number of children below which a more
// expensive approach is taken to uniquify the names to guarantee unique
// paths, which preserves the original name wherever possible -- formatting of
// index assumes this limit is less than 1000
var UniquifyPreserveNameLimit = 100

// UniquifyNames makes sure that the names are unique -- the "deluxe" version
// preserves the regular User-given name but is relatively expensive (creates
// a map), so is only used below a certain size (UniquifyPreserveNameLimit =
// 100), above which the index is appended, guaranteeing uniqueness at the
// cost of making paths longer and less user-friendly
func (n *Node) UniquifyNames() {
	// pr := prof.Start("ki.Node.UniquifyNames")
	// defer pr.End()

	sz := len(n.Kids)
	if sz > UniquifyPreserveNameLimit {
		sfmt := "%v_%05d"
		switch {
		case sz > 9999999:
			sfmt = "%v_%10d"
		case sz > 999999:
			sfmt = "%v_%07d"
		case sz > 99999:
			sfmt = "%v_%06d"
		}
		for i, child := range n.Kids {
			child.SetUniqueName(fmt.Sprintf(sfmt, child.Name(), i))
		}
		return
	}
	nmap := make(map[string]int, sz)
	for i, child := range n.Kids {
		if len(child.UniqueName()) == 0 {
			if n.Par != nil {
				child.SetUniqueName(fmt.Sprintf("%v_%03d", n.Par.UniqueName(), i))
			} else {
				child.SetUniqueName(fmt.Sprintf("c%03d", i))
			}
		}
		if _, taken := nmap[child.UniqueName()]; taken {
			child.SetUniqueName(fmt.Sprintf("%v_%03d", child.UniqueName(), i))
		} else {
			nmap[child.UniqueName()] = i
		}
	}
}

//////////////////////////////////////////////////////////////////////////
//  Parents

// Parent returns the parent of this Ki (Node.Par) -- Ki has strict
// one-parent, no-cycles structure -- see SetParent.
func (n *Node) Parent() Ki {
	return n.Par
}

// SetParent just sets parent of node (and inherits update count from
// parent, to keep consistent) -- does NOT remove from existing parent --
// use Add / Insert / Delete Child functions properly move or delete nodes.
func (n *Node) SetParent(parent Ki) {
	n.Par = parent
	if parent != nil && !parent.OnlySelfUpdate() {
		parup := parent.IsUpdating()
		n.FuncDownMeFirst(0, nil, func(k Ki, level int, d interface{}) bool {
			k.SetFlagState(parup, int(Updating))
			return true
		})
	}
}

// IsRoot tests if this node is the root node -- checks Parent = nil.
func (n *Node) IsRoot() bool {
	if n.This() == nil || n.Par == nil || n.Par.This() == nil {
		return true
	}
	return false
}

// Root returns the root object of this tree (the node with a nil parent).
func (n *Node) Root() Ki {
	if n.IsRoot() {
		return n.This()
	}
	return n.Par.Root()
}

// FieldRoot returns the field root object for this node -- the node that
// owns the branch of the tree rooted in one of its fields -- i.e., the
// first non-Field parent node after the first Field parent node -- can be
// nil if no such thing exists for this node.
func (n *Node) FieldRoot() Ki {
	var root Ki
	gotField := false
	n.FuncUpParent(0, n, func(k Ki, level int, d interface{}) bool {
		if !gotField {
			if k.IsField() {
				gotField = true
			}
			return true
		} else {
			if !k.IsField() {
				root = k
				return false
			}
		}
		return true
	})
	return root
}

// IndexInParent returns our index within our parent object -- caches the
// last value and uses that for an optimized search so subsequent calls
// are typically quite fast.  Returns false if we don't have a parent.
func (n *Node) IndexInParent() (int, bool) {
	if n.Par == nil {
		return -1, false
	}
	var ok bool
	n.index, ok = n.Par.Children().IndexOf(n.This(), n.index) // very fast if index is close..
	return n.index, ok
}

// ParentLevel finds a given potential parent node recursively up the
// hierarchy, returning level above current node that the parent was
// found, and -1 if not found.
func (n *Node) ParentLevel(par Ki) int {
	parLev := -1
	n.FuncUpParent(0, n, func(k Ki, level int, d interface{}) bool {
		if k == par {
			parLev = level
			return false
		}
		return true
	})
	return parLev
}

// HasParent checks if given node is a parent of this one (i.e.,
// ParentLevel(par) != -1).
func (n *Node) HasParent(par Ki) bool {
	return n.ParentLevel(par) != -1
}

// ParentByName finds first parent recursively up hierarchy that matches
// given name -- returns nil if not found.
func (n *Node) ParentByName(name string) Ki {
	if n.IsRoot() {
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
	return nil, fmt.Errorf("ki %v: Parent name: %v not found", n.PathUnique(), name)
}

// ParentByType finds parent recursively up hierarchy, by type, and
// returns nil if not found. If embeds is true, then it looks for any
// type that embeds the given type at any level of anonymous embedding.
func (n *Node) ParentByType(t reflect.Type, embeds bool) Ki {
	if n.IsRoot() {
		return nil
	}
	if embeds {
		if n.Par.TypeEmbeds(t) {
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
func (n *Node) ParentByTypeTry(t reflect.Type, embeds bool) (Ki, error) {
	par := n.ParentByType(t, embeds)
	if par != nil {
		return par, nil
	}
	return nil, fmt.Errorf("ki %v: Parent of type: %v not found", n.PathUnique(), t)
}

// KiFieldByName returns field Ki element by name -- returns false if not found.
func (n *Node) KiFieldByName(name string) Ki {
	v := reflect.ValueOf(n.This()).Elem()
	f := v.FieldByName(name)
	if !f.IsValid() {
		return nil
	}
	if !kit.EmbedImplements(f.Type(), KiType) {
		return nil
	}
	return kit.PtrValue(f).Interface().(Ki)
}

// KiFieldByNameTry returns field Ki element by name -- returns error if not found.
func (n *Node) KiFieldByNameTry(name string) (Ki, error) {
	fld := n.KiFieldByName(name)
	if fld != nil {
		return fld, nil
	}
	return nil, fmt.Errorf("ki %v: Ki Field named: %v not found", n.PathUnique(), name)
}

//////////////////////////////////////////////////////////////////////////
//  Children

// HasChildren tests whether this node has children (i.e., non-terminal).
func (n *Node) HasChildren() bool {
	return len(n.Kids) > 0
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
	return fmt.Errorf("ki %v: invalid index: %v -- len = %v", n.PathUnique(), idx, sz)
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
		return nil, fmt.Errorf("ki %v: child named: %v not found", n.PathUnique(), name)
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
		return nil, fmt.Errorf("ki %v: child of type: %t not found", n.PathUnique(), t)
	}
	return n.Kids[idx], nil
}

//////////////////////////////////////////////////////////////////////////
//  Paths

// Path returns path to this node from Root(), using regular user-given
// Name's (may be empty or non-unique), with nodes separated by / and
// fields by . -- only use for informational purposes.
func (n *Node) Path() string {
	if n.Par != nil {
		if n.IsField() {
			return n.Par.Path() + "." + n.Nm
		} else {
			return n.Par.Path() + "/" + n.Nm
		}
	}
	return "/" + n.Nm
}

// PathUnique returns path to this node from Root(), using unique names,
// with nodes separated by / and fields by . -- suitable for reliably
// finding this node.
func (n *Node) PathUnique() string {
	if n.Par != nil {
		if n.IsField() {
			return n.Par.PathUnique() + "." + n.UniqueNm
		} else {
			return n.Par.PathUnique() + "/" + n.UniqueNm
		}
	}
	return "/" + n.UniqueNm
}

// PathFrom returns path to this node from given parent node, using
// regular user-given Name's (may be empty or non-unique), with nodes
// separated by / and fields by . -- only use for informational purposes.
func (n *Node) PathFrom(par Ki) string {
	if n.Par != nil && n.Par != par {
		if n.IsField() {
			return n.Par.PathFrom(par) + "." + n.Nm
		} else {
			return n.Par.PathFrom(par) + "/" + n.Nm
		}
	}
	return "/" + n.Nm
}

// PathFromUnique returns path to this node from given parent node, using
// unique names, with nodes separated by / and fields by . -- suitable for
// reliably finding this node.
func (n *Node) PathFromUnique(par Ki) string {
	if n.Par != nil && n.Par != par {
		if n.IsField() {
			return n.Par.PathFromUnique(par) + "." + n.UniqueNm
		} else {
			return n.Par.PathFromUnique(par) + "/" + n.UniqueNm
		}
	}
	return "/" + n.UniqueNm
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
	return k.Children().IndexByUniqueName(child, 0)
}

// FindPathUnique returns Ki object at given unique path, starting from
// this node (e.g., Root()) -- if this node is not the root, then the path
// to this node is subtracted from the start of the path if present there.
// There is also support for [idx] index-based access for any given path
// element, for cases when indexes are more useful than names.
// Returns nil if not found.
func (n *Node) FindPathUnique(path string) Ki {
	if n.Par != nil { // we are not root..
		myp := n.PathUnique()
		path = strings.TrimPrefix(path, myp)
	}
	curn := Ki(n)
	pels := strings.Split(strings.Trim(strings.TrimSpace(path), "\""), "/")
	for i, pe := range pels {
		if len(pe) == 0 {
			continue
		}
		if i <= 1 && curn.UniqueName() == pe {
			continue
		}
		if strings.Contains(pe, ".") { // has fields
			fels := strings.Split(pe, ".")
			// find the child first, then the fields
			idx, ok := findPathChild(curn, fels[0])
			if !ok {
				return nil
			}
			curn = (*(curn.Children()))[idx]
			for i := 1; i < len(fels); i++ {
				fe := fels[i]
				fk := curn.KiFieldByName(fe)
				if fk == nil {
					return nil
				}
				curn = fk
			}
		} else {
			idx, ok := findPathChild(curn, pe)
			if !ok {
				return nil
			}
			curn = (*(curn.Children()))[idx]
		}
	}
	return curn
}

// FindPathUniqueTry returns Ki object at given unique path, starting from
// this node (e.g., Root()) -- if this node is not the root, then the path
// to this node is subtracted from the start of the path if present there.
// There is also support for [idx] index-based access for any given path
// element, for cases when indexes are more useful than names.
// Returns error if not found.
func (n *Node) FindPathUniqueTry(path string) (Ki, error) {
	fk := n.FindPathUnique(path)
	if fk != nil {
		return fk, nil
	}
	return nil, fmt.Errorf("ki %v: element at path: %v not found", n.PathUnique(), path)
}

//////////////////////////////////////////////////////////////////////////
//  Adding, Inserting Children

// SetChildType sets the ChildType used as a default type for creating new
// children -- as a property called ChildType --ensures that the type is a
// Ki type, and errors if not.
func (n *Node) SetChildType(t reflect.Type) error {
	if !reflect.PtrTo(t).Implements(reflect.TypeOf((*Ki)(nil)).Elem()) {
		err := fmt.Errorf("Ki Node %v SetChildType: type does not implement the Ki interface -- must -- type passed is: %v", n.PathUnique(), t.Name())
		log.Print(err)
		return err
	}
	n.SetProp("ChildType", t)
	return nil
}

// AddChildCheck checks if it is safe to add child -- it cannot be a parent of us -- prevent loops!
func (n *Node) AddChildCheck(kid Ki) error {
	var err error
	n.FuncUp(0, n, func(k Ki, level int, d interface{}) bool {
		if k == kid {
			err = fmt.Errorf("ki.Node Attempt to add child to node %v that is my own parent -- no cycles permitted", (d.(Ki)).PathUnique())
			log.Println(err)
			return false
		}
		return true
	})
	return err
}

// after adding child -- signals etc
func (n *Node) addChildImplPost(kid Ki) {
	oldPar := kid.Parent()
	kid.SetParent(n.This()) // key to set new parent before deleting: indicates move instead of delete
	if oldPar != nil {
		oldPar.DeleteChild(kid, false)
		kid.SetFlag(int(ChildMoved))
	} else {
		kid.SetFlag(int(ChildAdded))
	}
}

// addChildImpl does the internal mechanics of adding a child
func (n *Node) addChildImpl(kid Ki) error {
	if err := n.ThisCheck(); err != nil {
		return err
	}
	if err := n.AddChildCheck(kid); err != nil {
		return err
	}
	kid.Init(kid)
	n.Kids = append(n.Kids, kid)
	n.addChildImplPost(kid)
	return nil
}

// insertChildImpl does the internal mechanics of inserting a child
func (n *Node) insertChildImpl(kid Ki, at int) error {
	if err := n.ThisCheck(); err != nil {
		return err
	}
	if err := n.AddChildCheck(kid); err != nil {
		return err
	}
	kid.Init(kid)
	n.Kids.Insert(kid, at)
	n.addChildImplPost(kid)
	return nil
}

// AddChild adds a new child at end of children list -- if child is in an
// existing tree, it is removed from that parent, and a NodeMoved signal
// is emitted for the child -- UniquifyNames is called after adding to
// ensure name is unique (assumed to already have a name).
func (n *Node) AddChild(kid Ki) error {
	updt := n.UpdateStart()
	err := n.addChildImpl(kid)
	if err == nil {
		n.SetFlag(int(ChildAdded))
		if kid.UniqueName() == "" {
			kid.SetUniqueName(kid.Name())
		}
		n.UniquifyNames()
	}
	n.UpdateEnd(updt)
	return err
}

// AddChildFast adds a new child at end of children list in the fastest
// way possible -- assumes InitName has already been run, and doesn't
// ensure names are unique, or run other checks, including if child
// already has a parent.
func (n *Node) AddChildFast(kid Ki) {
	updt := n.UpdateStart()
	n.Kids = append(n.Kids, kid)
	kid.SetParent(n.This())
	kid.SetFlag(int(ChildAdded))
	n.SetFlag(int(ChildAdded))
	n.UpdateEnd(updt)
}

// InsertChild adds a new child at given position in children list -- if
// child is in an existing tree, it is removed from that parent, and a
// NodeMoved signal is emitted for the child -- UniquifyNames is called
// after adding to ensure name is unique (assumed to already have a name).
func (n *Node) InsertChild(kid Ki, at int) error {
	updt := n.UpdateStart()
	err := n.insertChildImpl(kid, at)
	if err == nil {
		n.SetFlag(int(ChildAdded))
		if kid.UniqueName() == "" {
			kid.SetUniqueName(kid.Name())
		}
		n.UniquifyNames()
	}
	n.UpdateEnd(updt)
	return err
}

// NewOfType creates a new child of given type -- if nil, uses ChildType,
// else uses the same type as this struct.
func (n *Node) NewOfType(typ reflect.Type) Ki {
	if err := n.ThisCheck(); err != nil {
		return nil
	}
	if typ == nil {
		ct, ok := n.PropInherit("ChildType", false, true) // no inherit but yes from type
		if ok {
			if ctt, ok := ct.(reflect.Type); ok {
				typ = ctt
			}
		}
	}
	if typ == nil {
		typ = n.Type() // make us by default
	}
	nkid := reflect.New(typ).Interface()
	kid, _ := nkid.(Ki)
	return kid
}

// AddNewChild creates a new child of given type -- if nil, uses
// ChildType, else type of this struct -- and add at end of children list
// -- assigns name (can be empty) and enforces UniqueName.
func (n *Node) AddNewChild(typ reflect.Type, name string) Ki {
	updt := n.UpdateStart()
	kid := n.NewOfType(typ)
	err := n.addChildImpl(kid)
	if err == nil {
		kid.SetName(name)
		n.SetFlag(int(ChildAdded))
	}
	n.UpdateEnd(updt)
	return kid
}

// InsertNewChild creates a new child of given type -- if nil, uses
// ChildType, else type of this struct -- and add at given position in
// children list -- assigns name (can be empty) and enforces UniqueName.
func (n *Node) InsertNewChild(typ reflect.Type, at int, name string) Ki {
	updt := n.UpdateStart()
	kid := n.NewOfType(typ)
	err := n.insertChildImpl(kid, at)
	if err == nil {
		kid.SetName(name)
		n.SetFlag(int(ChildAdded))
	}
	n.UpdateEnd(updt)
	return kid
}

// InsertNewChildUnique adds a new child at given position in children
// list, and gives it a name, using SetNameRaw and SetUniqueName for the
// name -- only when names are known to be unique (faster).
func (n *Node) InsertNewChildUnique(typ reflect.Type, at int, name string) Ki {
	updt := n.UpdateStart()
	kid := n.NewOfType(typ)
	err := n.insertChildImpl(kid, at)
	if err == nil {
		kid.SetNameRaw(name)
		kid.SetUniqueName(name)
		n.SetFlag(int(ChildAdded))
	}
	n.UpdateEnd(updt)
	return kid
}

// SetChild sets child at given index to be the given item -- if name is
// non-empty then it sets the name of the child as well -- just calls Init
// (or InitName) on the child, and SetParent -- does NOT uniquify the
// names -- this is for high-volume child creation -- call UniquifyNames
// afterward if needed, but better to ensure that names are unique up front.
func (n *Node) SetChild(kid Ki, idx int, name string) error {
	if err := n.Kids.IsValidIndex(idx); err != nil {
		return err
	}
	if name != "" {
		kid.InitName(kid, name)
	} else {
		kid.Init(kid)
	}
	n.Kids[idx] = kid
	kid.SetParent(n.This())
	return nil
}

// MoveChild moves child from one position to another in the list of
// children (see also corresponding Slice method, which does not
// signal, like this one does).  Returns error if either index is invalid.
func (n *Node) MoveChild(from, to int) error {
	updt := n.UpdateStart()
	err := n.Kids.Move(from, to)
	if err == nil {
		n.SetFlag(int(ChildMoved))
	}
	n.UpdateEnd(updt)
	return err
}

// SwapChildren swaps children between positions (see also corresponding
// Slice method which does not signal like this one does).  Returns error if
// either index is invalid.
func (n *Node) SwapChildren(i, j int) error {
	updt := n.UpdateStart()
	err := n.Kids.Swap(i, j)
	if err == nil {
		n.SetFlag(int(ChildMoved))
	}
	n.UpdateEnd(updt)
	return err
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
// change their names, or call UniquifyNames -- use ConfigChildren for
// those cases -- this function is for simpler cases where a parent uses
// this function consistently to manage children all of the same type.
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
		nm := fmt.Sprintf("%v%v", nameStub, sz)
		n.InsertNewChildUnique(typ, sz, nm)
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
func (n *Node) ConfigChildren(config kit.TypeAndNameList, uniqNm bool) (mods, updt bool) {
	return n.Kids.Config(n.This(), config, uniqNm)
}

//////////////////////////////////////////////////////////////////////////
//  Deleting Children

// DeleteChildAtIndex deletes child at given index (returns error for
// invalid index) -- if child's parent = this node, then will call
// SetParent(nil), so to transfer to another list, set new parent first --
// destroy will add removed child to deleted list, to be destroyed later
// -- otherwise child remains intact but parent is nil -- could be
// inserted elsewhere.
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
		child.SetParent(nil)
	}
	n.Kids.DeleteAtIndex(idx)
	if destroy {
		DelMgr.Add(child)
	}
	child.UpdateReset() // it won't get the UpdateEnd from us anymore -- init fresh in any case
	n.UpdateEnd(updt)
	return nil
}

// DeleteChild deletes child node, returning error if not found in
// Children.  If child's parent = this node, then will call
// SetParent(nil), so to transfer to another list, set new parent
// first. See DeleteChildAtIndex for destroy info.
func (n *Node) DeleteChild(child Ki, destroy bool) error {
	if child == nil {
		return errors.New("ki DeleteChild: child is nil")
	}
	idx, ok := n.Kids.IndexOf(child, 0)
	if !ok {
		return fmt.Errorf("ki %v: child: %v not found", n.PathUnique(), child.PathUnique())
	}
	return n.DeleteChildAtIndex(idx, destroy)
}

// DeleteChildByName deletes child node by name -- returns child, error
// if not found -- if child's parent = this node, then will call
// SetParent(nil), so to transfer to another list, set new parent first.
// See DeleteChildAtIndex for destroy info.
func (n *Node) DeleteChildByName(name string, destroy bool) (Ki, error) {
	idx, ok := n.Kids.IndexByName(name, 0)
	if !ok {
		return nil, fmt.Errorf("ki %v: child named: %v not found", n.PathUnique(), name)
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
		child.SetFlag(int(NodeDeleted))
		child.NodeSignal().Emit(child, int64(NodeSignalDeleting), nil)
		child.SetParent(nil)
		child.UpdateReset()
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
			n.Destroy()
		}
	} else {
		n.Par.DeleteChild(n.This(), destroy)
	}
}

// Destroy calls DisconnectAll to cut all pointers and signal connections,
// and remove all children and their childrens-children, etc.
func (n *Node) Destroy() {
	// fmt.Printf("Destroying: %v %T %p Kids: %v\n", n.PathUnique(), n.This(), n.This(), len(n.Kids))
	if n.This() == nil { // already dead!
		return
	}
	n.NodeSig.Emit(n.This(), int64(NodeSignalDestroying), nil)
	n.DisconnectAll()
	n.DeleteChildren(true) // first delete all my children
	// and destroy all my fields
	n.FuncFields(0, nil, func(k Ki, level int, d interface{}) bool {
		k.Destroy()
		return true
	})
	DelMgr.DestroyDeleted() // then destroy all those kids
	// extra step to delete all the slices and maps -- super friendly to GC :)
	// note: not safe at this point!
	// FlatFieldsValueFunc(n.This(), func(stru interface{}, typ reflect.Type, field reflect.StructField, fieldVal reflect.Value) bool {
	// 	if fieldVal.Kind() == reflect.Slice || fieldVal.Kind() == reflect.Map {
	// 		fieldVal.Set(reflect.Zero(fieldVal.Type())) // set to nil
	// 	}
	// 	return true
	// })
	n.SetFlag(int(NodeDestroyed))
	n.Ths = nil // last gasp: lose our own sense of self..
	// note: above is thread-safe because This() accessor checks Destroyed
}

//////////////////////////////////////////////////////////////////////////
//  Flags

// Flag returns an atomically safe copy of the bit flags for this node --
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

// HasAnyFlag checks if *any* of a set of flags is set (logical OR)
// using atomic, safe for concurrent access
func (n *Node) HasAnyFlag(flag ...int) bool {
	return bitflag.HasAnyAtomic(&n.Flag, flag...)
}

// HasAllFlags checks if *all* of a set of flags is set (logical AND)
// using atomic, safe for concurrent access
func (n *Node) HasAllFlags(flag ...int) bool {
	return bitflag.HasAllAtomic(&n.Flag, flag...)
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

// SetProp sets given property key to value val -- initializes property
// map if nil.
func (n *Node) SetProp(key string, val interface{}) {
	if n.Props == nil {
		n.Props = make(Props)
	}
	n.Props[key] = val
}

// SetProps sets a whole set of properties, and optionally sets the
// updated flag and triggers an UpdateSig.
func (n *Node) SetProps(props Props, update bool) {
	if n.Props == nil {
		n.Props = make(Props)
	}
	for key, val := range props {
		n.Props[key] = val
	}
	if update {
		n.SetFlag(int(PropUpdated))
		n.UpdateSig()
	}
}

// SetPropUpdate sets given property key to value val, with update
// notification (sets PropUpdated and emits UpdateSig) so other nodes
// receiving update signals from this node can update to reflect these
// changes.
func (n *Node) SetPropUpdate(key string, val interface{}) {
	n.SetFlag(int(PropUpdated))
	n.SetProp(key, val)
	n.UpdateSig()
}

// SetPropChildren sets given property key to value val for all Children.
func (n *Node) SetPropChildren(key string, val interface{}) {
	for _, k := range n.Kids {
		k.SetProp(key, val)
	}
}

// Prop gets property value from key.
func (n *Node) Prop(key string) (interface{}, bool) {
	v, ok := n.Props[key]
	return v, ok
}

// KnownProp gets property value from key that is known to exist --
// returns nil if it actually doesn't -- less cumbersome for conversions.
func (n *Node) KnownProp(key string) interface{} {
	return n.Props[key]
}

// PropInherit gets property value from key with options for inheriting
// property from parents and / or type-level properties.  If inherit, then
// checks all parents.  If typ then checks property on type as well
// (registered via KiT type registry).  Returns false if not set anywhere.
func (n *Node) PropInherit(key string, inherit, typ bool) (interface{}, bool) {
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
		return kit.Types.Prop(n.Type(), key)
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

// DeleteAllProps deletes all properties on this node -- just makes a new
// Props map -- can specify the capacity of the new map (0 means set to
// nil instead of making a new one -- most efficient if potentially no
// properties will be set).
func (n *Node) DeleteAllProps(cap int) {
	if n.Props != nil {
		if cap == 0 {
			n.Props = nil
		} else {
			n.Props = make(Props, cap)
		}
	}
}

func init() {
	gob.Register(Props{})
}

// CopyPropsFrom copies our properties from another node -- if deep then
// does a deep copy -- otherwise copied map just points to same values in
// the original map (and we don't reset our map first -- call
// DeleteAllProps to do that -- deep copy uses gob encode / decode --
// usually not needed).
func (n *Node) CopyPropsFrom(from Ki, deep bool) error {
	if *(from.Properties()) == nil {
		return nil
	}
	if n.Props == nil {
		n.Props = make(Props)
	}
	fmP := *(from.Properties())
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
	} else {
		for k, v := range fmP {
			n.Props[k] = v
		}
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

// Fields returns the uintptr offsets for Ki fields of this Node -- cached for fast
// access
func (n *Node) Fields() []uintptr {
	// we store the offsets for the fields in type properties
	tprops := *kit.Types.Properties(n.Type(), true) // true = makeNew
	pnm := "__FieldOffs"
	if foff, ok := kit.TypeProp(tprops, pnm); ok {
		return foff.([]uintptr)
	}
	foff := make([]uintptr, 0)
	kitype := KiType
	FlatFieldsValueFunc(n.This(), func(stru interface{}, typ reflect.Type, field reflect.StructField, fieldVal reflect.Value) bool {
		if fieldVal.Kind() == reflect.Struct && kit.EmbedImplements(field.Type, kitype) {
			foff = append(foff, field.Offset)
		}
		return true
	})
	kit.SetTypeProp(tprops, pnm, foff)
	return foff
}

// FlatFieldsValueFunc is the Node version of this function from kit/embeds.go
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
	op := reflect.ValueOf(n.This()).Pointer()
	foffs := n.Fields()
	for _, fo := range foffs {
		fn := (*Node)(unsafe.Pointer(op + fo))
		fun(fn.This(), level, data)
	}
}

// GoFuncFields calls concurrent goroutine function on all Ki fields
// within this node.
func (n *Node) GoFuncFields(level int, data interface{}, fun Func) {
	if n.This() == nil {
		return
	}
	op := reflect.ValueOf(n.This()).Pointer()
	foffs := n.Fields()
	for _, fo := range foffs {
		fn := (*Node)(unsafe.Pointer(op + fo))
		go fun(fn.This(), level, data)
	}
}

// FuncUp calls function on given node and all the way up to its parents,
// and so on -- sequentially all in current go routine (generally
// necessary for going up, which is typicaly quite fast anyway) -- level
// is incremented after each step (starts at 0, goes up), and passed to
// function -- returns false if fun aborts with false, else true.
func (n *Node) FuncUp(level int, data interface{}, fun Func) bool {
	if !fun(n.This(), level, data) { // false return means stop
		return false
	}
	level++
	if n.Parent() != nil && n.Parent() != n.This() { // prevent loops
		return n.Parent().FuncUp(level, data, fun)
	}
	return true
}

// FuncUpParent calls function on parent of node and all the way up to its
// parents, and so on -- sequentially all in current go routine (generally
// necessary for going up, which is typicaly quite fast anyway) -- level
// is incremented after each step (starts at 0, goes up), and passed to
// function -- returns false if fun aborts with false, else true.
func (n *Node) FuncUpParent(level int, data interface{}, fun Func) bool {
	if n.IsRoot() {
		return true
	}
	if !fun(n.Parent(), level, data) { // false return means stop
		return false
	}
	level++
	return n.Parent().FuncUpParent(level, data, fun)
}

// FuncDownMeFirst calls function on this node (MeFirst) and then call
// FuncDownMeFirst on all the children -- sequentially all in current go
// routine -- level var is incremented before calling children -- if fun
// returns false then any further traversal of that branch of the tree is
// aborted, but other branches continue -- i.e., if fun on current node
// returns false, then returns false and children are not processed
// further -- this is the fastest, most natural form of traversal.
func (n *Node) FuncDownMeFirst(level int, data interface{}, fun Func) bool {
	if n.This() == nil {
		return false
	}
	if !fun(n.This(), level, data) { // false return means stop
		return false
	}
	level++
	n.FuncFields(level, data, func(k Ki, level int, d interface{}) bool {
		k.FuncDownMeFirst(level, data, fun)
		return true
	})
	for _, child := range *n.Children() {
		child.FuncDownMeFirst(level, data, fun) // don't care about their return values
	}
	return true
}

// FuncDownDepthFirst calls FuncDownDepthFirst on all children, then calls
// function on this node -- sequentially all in current go routine --
// level var is incremented before calling children -- runs
// doChildTestFunc on each child first to determine if it should process
// that child, and if that returns true, then it calls FuncDownDepthFirst
// on that child.
func (n *Node) FuncDownDepthFirst(level int, data interface{}, doChildTestFunc Func, fun Func) {
	if n.This() == nil {
		return
	}
	level++
	for _, child := range *n.Children() {
		if child.This() != nil {
			if doChildTestFunc(child.This(), level, data) { // test if we should run on this child
				child.FuncDownDepthFirst(level, data, doChildTestFunc, fun)
			}
		}
	}
	n.FuncFields(level, data, func(k Ki, level int, d interface{}) bool {
		if k.This() != nil {
			if doChildTestFunc(k, level, data) { // test if we should run on this child
				k.FuncDownDepthFirst(level, data, doChildTestFunc, fun)
			}
			fun(k, level, data)
		}
		return true
	})
	level--
	fun(n.This(), level, data) // can't use the return value at this point
}

// FuncDownBreadthFirst calls function on all children, then calls
// FuncDownBreadthFirst on all the children -- does NOT call on first node
// where this method is first called, due to nature of recursive logic.
// This is NOT a strict breadth-first search algorithm (which is computationally
// expensive) but rather just a breadth-first variant of the recursive top-down
// traversal, that first traverses all the children of a node before recursing.
// The level var is incremented before calling children -- if fun returns
// false then any further traversal of that branch of the tree is aborted,
// but other branches can continue.
func (n *Node) FuncDownBreadthFirst(level int, data interface{}, fun Func) {
	if n.This() == nil {
		return
	}
	dontMap := make(map[int]struct{}) // map of who NOT to process further -- default is false for map so reverse
	level++
	for i, child := range *n.Children() {
		if child.This() == nil || !fun(child, level, data) {
			// false return means stop
			dontMap[i] = struct{}{}
		} else {
			child.FuncFields(level+1, data, func(k Ki, level int, d interface{}) bool {
				k.FuncDownBreadthFirst(level+1, data, fun)
				fun(k, level+1, data)
				return true
			})
		}
	}
	for i, child := range *n.Children() {
		if _, has := dontMap[i]; has {
			continue
		}
		child.FuncDownBreadthFirst(level, data, fun)
	}
}

// GoFuncDown calls concurrent goroutine function on given node and all
// the way down to its children, and so on -- does not wait for completion
// of the go routines -- returns immediately.
func (n *Node) GoFuncDown(level int, data interface{}, fun Func) {
	if n.This() == nil {
		return
	}
	go fun(n.This(), level, data)
	level++
	n.GoFuncFields(level, data, fun)
	for _, child := range *n.Children() {
		child.GoFuncDown(level, data, fun)
	}
}

// func (n *Node) GoFuncDownWait(level int, data interface{}, fun Func) {
// if n.This() == nil {
// 	return
// }
// 	// todo: use channel or something to wait
// 	go fun(n.This(), level, data)
// 	level++
// 	n.GoFuncFields(level, data, fun)
// 	for _, child := range *n.Children() {
// 		child.GoFuncDown(level, data, fun)
// 	}
// }

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
		n.FuncDownMeFirst(0, nil, func(k Ki, level int, d interface{}) bool {
			if !k.IsUpdating() {
				k.ClearFlagMask(int64(UpdateFlagsMask))
				k.SetFlag(int(Updating))
				return true // keep going down
			} else {
				return false // bail -- already updating
			}
		})
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
	if n.HasAnyFlag(int(ChildDeleted), int(ChildrenDeleted)) {
		DelMgr.DestroyDeleted()
	}
	if n.OnlySelfUpdate() {
		n.ClearFlag(int(Updating))
		n.NodeSignal().Emit(n.This(), int64(NodeSignalUpdated), n.Flags())
	} else {
		n.FuncDownMeFirst(0, nil, func(k Ki, level int, d interface{}) bool {
			k.ClearFlag(int(Updating)) // todo: could check first and break here but good to ensure all clear
			return true
		})
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
	if n.HasAnyFlag(int(ChildDeleted), int(ChildrenDeleted)) {
		DelMgr.DestroyDeleted()
	}
	if n.OnlySelfUpdate() {
		n.ClearFlag(int(Updating))
		// n.NodeSignal().Emit(n.This(), int64(NodeSignalUpdated), n.Flags())
	} else {
		n.FuncDownMeFirst(0, nil, func(k Ki, level int, d interface{}) bool {
			k.ClearFlag(int(Updating)) // todo: could check first and break here but good to ensure all clear
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

// UpdateReset resets Updating flag for this node and all children -- in
// case they are out-of-sync due to more complex tree maninpulations --
// only call at a known point of non-updating.
func (n *Node) UpdateReset() {
	if n.OnlySelfUpdate() {
		n.ClearFlag(int(Updating))
	} else {
		n.FuncDownMeFirst(0, nil, func(k Ki, level int, d interface{}) bool {
			k.ClearFlag(int(Updating))
			return true
		})
	}
}

// Disconnect disconnects node -- reset all ptrs to nil, and
// DisconnectAll() signals -- e.g., for freeing up all connnections so
// node can be destroyed and making GC easier.
func (n *Node) Disconnect() {
	n.NodeSig.DisconnectAll()
	FlatFieldsValueFunc(n.This(), func(stru interface{}, typ reflect.Type, field reflect.StructField, fieldVal reflect.Value) bool {
		switch {
		case fieldVal.Kind() == reflect.Interface:
			if field.Name != "This" { // reserve that for last step in Destroy
				fieldVal.Set(reflect.Zero(fieldVal.Type())) // set to nil
			}
		case fieldVal.Kind() == reflect.Ptr:
			fieldVal.Set(reflect.Zero(fieldVal.Type())) // set to nil
		case fieldVal.Type() == KiT_Signal:
			if fs, ok := kit.PtrValue(fieldVal).Interface().(*Signal); ok {
				// fmt.Printf("ki.Node: %v Type: %T Disconnecting signal field: %v\n", n.Name(), n.This(), field.Name)
				fs.DisconnectAll()
			}
		case fieldVal.Type() == KiT_Ptr:
			if pt, ok := kit.PtrValue(fieldVal).Interface().(*Ptr); ok {
				pt.Reset()
			}
		}
		return true
	})
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
// wrapped in UpdateStart / End and sets the FieldUpdated flag.
func (n *Node) SetField(field string, val interface{}) error {
	fv := kit.FlatFieldValueByName(n.This(), field)
	if !fv.IsValid() {
		return fmt.Errorf("ki.SetField, could not find field %v on node %v", field, n.PathUnique())
	}
	updt := n.UpdateStart()
	var err error
	if field == "Nm" {
		n.SetName(kit.ToString(val))
		n.SetFlag(int(FieldUpdated))
	} else {
		if kit.SetRobust(kit.PtrValue(fv).Interface(), val) {
			n.SetFlag(int(FieldUpdated))
		} else {
			err = fmt.Errorf("ki.SetField, SetRobust failed to set field %v on node %v to value: %v", field, n.PathUnique(), val)
		}
	}
	n.UpdateEnd(updt)
	return err
}

// SetFieldDown sets given field name to given value, all the way down the
// tree from me -- wrapped in UpdateStart / End.
func (n *Node) SetFieldDown(field string, val interface{}) {
	updt := n.UpdateStart()
	n.FuncDownMeFirst(0, nil, func(k Ki, level int, d interface{}) bool {
		k.SetField(field, val)
		return true
	})
	n.UpdateEnd(updt)
}

// SetFieldUp sets given field name to given value, all the way up the
// tree from me -- wrapped in UpdateStart / End.
func (n *Node) SetFieldUp(field string, val interface{}) {
	updt := n.UpdateStart()
	n.FuncUp(0, nil, func(k Ki, level int, d interface{}) bool {
		k.SetField(field, val)
		return true
	})
	n.UpdateEnd(updt)
}

// FieldByName returns field value by name (can be any type of field --
// see KiFieldByName for Ki fields) -- returns nil if not found.
func (n *Node) FieldByName(field string) interface{} {
	return kit.FlatFieldInterfaceByName(n.This(), field)
}

// FieldByNameTry returns field value by name (can be any type of field --
// see KiFieldByName for Ki fields) -- returns error if not found.
func (n *Node) FieldByNameTry(field string) (interface{}, error) {
	fld := n.FieldByName(field)
	if fld != nil {
		return fld, nil
	}
	return nil, fmt.Errorf("ki %v: field named: %v not found", n.PathUnique(), field)
}

// FieldTag returns given field tag for that field, or empty string if not
// set.
func (n *Node) FieldTag(field, tag string) string {
	return kit.FlatFieldTag(n.Type(), field, tag)
}

//////////////////////////////////////////////////////////////////////////
//  Deep Copy / Clone

// note: we use the copy from direction as the receiver is modifed whereas the
// from is not and assignment is typically in same direction

// CopyFrom another Ki node.  The Ki copy function recreates the entire
// tree in the copy, duplicating children etc.  It is very efficient by
// using the ConfigChildren method which attempts to preserve any existing
// nodes in the destination if they have the same name and type -- so a
// copy from a source to a target that only differ minimally will be
// minimally destructive.  Only copies to same types are supported.
// Pointers (Ptr) are copied by saving the current UniquePath and then
// SetPtrsFmPaths is called.  Signal connections are NOT copied.  No other
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
// UpdtPtrPaths, then SetPtrsFmPaths.
func (n *Node) CopyFrom(from Ki) error {
	if from == nil {
		err := fmt.Errorf("ki.Node CopyFrom into %v -- null 'from' source", n.PathUnique())
		log.Println(err)
		return err
	}
	mypath := n.PathUnique()
	fmpath := from.PathUnique()
	if n.Type() != from.Type() {
		err := fmt.Errorf("ki.Node Copy to %v from %v -- must have same types, but %v != %v", mypath, fmpath, n.Type().Name(), from.Type().Name())
		log.Println(err)
		return err
	}
	updt := n.UpdateStart()
	n.SetFlag(int(NodeCopied))
	sameTree := (n.Root() == from.Root())
	from.GetPtrPaths()
	err := n.CopyFromRaw(from)
	// DelMgr.DestroyDeleted() // in case we deleted some kiddos
	if err != nil {
		n.UpdateEnd(updt)
		return err
	}
	if sameTree {
		n.UpdatePtrPaths(fmpath, mypath, true)
	}
	n.SetPtrsFmPaths()
	n.UpdateEnd(updt)
	return nil
}

// Clone creates and returns a deep copy of the tree from this node down.
// Any pointers within the cloned tree will correctly point within the new
// cloned tree (see Copy info).
func (n *Node) Clone() Ki {
	nki := n.NewOfType(n.Type())
	nki.InitName(nki, n.Nm)
	nki.CopyFrom(n.This())
	return nki
}

// CopyMakeChildrenFrom uses ConfigChildren to recreate source children
func (n *Node) CopyMakeChildrenFrom(from Ki) {
	sz := len(*from.Children())
	if sz > 0 {
		cfg := make(kit.TypeAndNameList, sz)
		for i, kid := range *from.Children() {
			cfg[i].Type = kid.Type()
			cfg[i].Name = kid.UniqueName() // use unique so guaranteed to have something
		}
		mods, updt := n.ConfigChildren(cfg, true) // use unique names -- this means name = uniquname
		for i, kid := range *from.Children() {
			mkid := n.Kids[i]
			mkid.SetNameRaw(kid.Name()) // restore orig user-names
		}
		if mods {
			n.UpdateEnd(updt)
		}
	} else {
		n.DeleteChildren(true)
	}
}

// CopyFieldsFrom copies from primary fields of source object,
// recursively following anonymous embedded structs
func (n *Node) CopyFieldsFrom(to interface{}, from interface{}) {
	kitype := KiType
	tv := kit.NonPtrValue(reflect.ValueOf(to))
	sv := kit.NonPtrValue(reflect.ValueOf(from))
	typ := tv.Type()
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
			n.CopyFieldsFrom(tfpi, sfpi)
		} else {
			switch {
			case sf.Kind() == reflect.Struct && kit.EmbedImplements(sf.Type(), kitype):
				sfk := sfpi.(Ki)
				tfk := tfpi.(Ki)
				if tfk != nil && sfk != nil {
					tfk.CopyFrom(sfk)
				}
			case f.Type == KiT_Signal: // todo: don't copy signals by default
			case sf.Type().AssignableTo(tf.Type()):
				tf.Set(sf)
				// kit.PtrValue(tf).Set(sf)
			default:
				// use copier https://github.com/jinzhu/copier which handles as much as possible..
				copier.Copy(tfpi, sfpi)
			}
		}

	}
}

// CopyFromRaw performs a raw copy that just does the deep copy of the
// bits and doesn't do anything with pointers.
func (n *Node) CopyFromRaw(from Ki) error {
	n.CopyMakeChildrenFrom(from)
	n.DeleteAllProps(len(*from.Properties())) // start off fresh, allocated to size of from
	n.CopyPropsFrom(from, false)              // use shallow props copy by default
	n.CopyFieldsFrom(n.This(), from)
	for i, kid := range n.Kids {
		fmk := (*(from.Children()))[i]
		kid.CopyFromRaw(fmk)
	}
	return nil
}

// GetPtrPaths gets all Ptr path strings -- walks the tree down from
// current node and calls GetPath on all Ptr fields -- this is called
// prior to copying / moving.
func (n *Node) GetPtrPaths() {
	root := n.This()
	n.FuncDownMeFirst(0, root, func(k Ki, level int, d interface{}) bool {
		FlatFieldsValueFunc(k, func(stru interface{}, typ reflect.Type, field reflect.StructField, fieldVal reflect.Value) bool {
			if fieldVal.CanInterface() {
				vfi := kit.PtrValue(fieldVal).Interface()
				switch vfv := vfi.(type) {
				case *Ptr:
					vfv.GetPath()
					// case *Signal:
					// 	vfv.GetPaths()
				}
			}
			return true
		})
		return true
	})
}

// SetPtrsFmPaths walks the tree down from current node and calls
// PtrFromPath on all Ptr fields found -- called after Copy, Unmarshal* to
// recover pointers after entire structure is in place -- see
// UnmarshalPost.
func (n *Node) SetPtrsFmPaths() {
	root := n.Root()
	n.FuncDownMeFirst(0, root, func(k Ki, level int, d interface{}) bool {
		FlatFieldsValueFunc(k, func(stru interface{}, typ reflect.Type, field reflect.StructField, fieldVal reflect.Value) bool {
			if fieldVal.CanInterface() {
				vfi := kit.PtrValue(fieldVal).Interface()
				switch vfv := vfi.(type) {
				case *Ptr:
					if err := vfv.PtrFmPath(root); err != nil {
						log.Println(err)
					}
				}
			}
			return true
		})
		return true
	})
}

// UpdatePtrPaths updates Ptr paths, replacing any occurrence of oldPath with
// newPath, optionally only at the start of the path (typically true) --
// for all nodes down from this one.
func (n *Node) UpdatePtrPaths(oldPath, newPath string, startOnly bool) {
	root := n.Root()
	n.FuncDownMeFirst(0, root, func(k Ki, level int, d interface{}) bool {
		FlatFieldsValueFunc(k, func(stru interface{}, typ reflect.Type, field reflect.StructField, fieldVal reflect.Value) bool {
			if fieldVal.CanInterface() {
				vfi := kit.PtrValue(fieldVal).Interface()
				switch vfv := vfi.(type) {
				case *Ptr:
					vfv.UpdatePath(oldPath, newPath, startOnly)
				}
			}
			return true
		})
		return true
	})
}

//////////////////////////////////////////////////////////////////////////
//  IO Marshal / Unmarshal support -- mostly in Slice

// see https://github.com/goki/ki/wiki/Naming for IO naming conventions

// Note: it is unfortunate that [Un]MarshalJSON uses byte[] instead of
// io.Reader / Writer..

// JSONTypePrefix is the first thing output in a ki tree JSON output file,
// specifying the type of the root node of the ki tree -- this info appears
// all on one { } bracketed line at the start of the file, and can also be
// used to identify the file as a ki tree JSON file
var JSONTypePrefix = []byte("{\"ki.RootType\": ")

// JSONTypeSuffix is just the } and \n at the end of the prefix line
var JSONTypeSuffix = []byte("}\n")

// WriteJSON writes the tree to an io.Writer, using MarshalJSON -- also
// saves a critical starting record that allows file to be loaded de-novo
// and recreate the proper root type for the tree.
func (n *Node) WriteJSON(writer io.Writer, indent bool) error {
	err := n.ThisCheck()
	if err != nil {
		return err
	}
	var b []byte
	if indent {
		b, err = json.MarshalIndent(n.This(), "", "  ")
	} else {
		b, err = json.Marshal(n.This())
	}
	if err != nil {
		log.Println(err)
		return err
	}
	knm := kit.FullTypeName(n.Type())
	tstr := string(JSONTypePrefix) + fmt.Sprintf("\"%v\"}\n", knm)
	nwb := make([]byte, len(b)+len(tstr))
	copy(nwb, []byte(tstr))
	copy(nwb[len(tstr):], b) // is there a way to avoid this?
	_, err = writer.Write(nwb)
	if err != nil {
		log.Println(err)
		return err
	}
	return nil
}

// SaveJSON saves the tree to a JSON-encoded file, using WriteJSON.
func (n *Node) SaveJSON(filename string) error {
	fp, err := os.Create(filename)
	defer fp.Close()
	if err != nil {
		log.Println(err)
		return err
	}
	err = n.WriteJSON(fp, true) // use indent by default
	if err != nil {
		log.Println(err)
	}
	return err
}

// ReadJSON reads and unmarshals tree starting at this node, from a
// JSON-encoded byte stream via io.Reader.  First element in the stream
// must be of same type as this node -- see ReadNewJSON function to
// construct a new tree.  Uses ConfigureChildren to minimize changes from
// current tree relative to loading one -- wraps UnmarshalJSON and calls
// UnmarshalPost to recover pointers from paths.
func (n *Node) ReadJSON(reader io.Reader) error {
	err := n.ThisCheck()
	if err != nil {
		log.Println(err)
		return err
	}
	b, err := ioutil.ReadAll(reader)
	if err != nil {
		log.Println(err)
		return err
	}
	updt := n.UpdateStart()
	stidx := 0
	if bytes.HasPrefix(b, JSONTypePrefix) { // skip type
		stidx = bytes.Index(b, JSONTypeSuffix) + len(JSONTypeSuffix)
	}
	err = json.Unmarshal(b[stidx:], n.This()) // key use of this!
	if err == nil {
		n.UnmarshalPost()
	}
	n.SetFlag(int(ChildAdded)) // this might not be set..
	n.UpdateEnd(updt)
	return err
}

// OpenJSON opens file over this tree from a JSON-encoded file -- see
// ReadJSON for details, and OpenNewJSON for opening an entirely new tree.
func (n *Node) OpenJSON(filename string) error {
	fp, err := os.Open(filename)
	defer fp.Close()
	if err != nil {
		log.Println(err)
		return err
	}
	return n.ReadJSON(fp)
}

// ReadNewJSON reads a new Ki tree from a JSON-encoded byte string, using type
// information at start of file to create an object of the proper type
func ReadNewJSON(reader io.Reader) (Ki, error) {
	b, err := ioutil.ReadAll(reader)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	if bytes.HasPrefix(b, JSONTypePrefix) {
		stidx := len(JSONTypePrefix) + 1
		eidx := bytes.Index(b, JSONTypeSuffix)
		bodyidx := eidx + len(JSONTypeSuffix)
		tn := string(bytes.Trim(bytes.TrimSpace(b[stidx:eidx]), "\""))
		typ := kit.Types.Type(tn)
		if typ == nil {
			return nil, fmt.Errorf("ki.OpenNewJSON: kit.Types type name not found: %v", tn)
		}
		root := NewOfType(typ)
		root.Init(root)

		updt := root.UpdateStart()
		err = json.Unmarshal(b[bodyidx:], root)
		if err == nil {
			root.UnmarshalPost()
		}
		root.SetFlag(int(ChildAdded)) // this might not be set..
		root.UpdateEnd(updt)
		return root, nil
	} else {
		return nil, fmt.Errorf("ki.OpenNewJSON -- type prefix not found at start of file -- must be there to identify type of root node of tree")
	}
}

// OpenNewJSON opens a new Ki tree from a JSON-encoded file, using type
// information at start of file to create an object of the proper type
func OpenNewJSON(filename string) (Ki, error) {
	fp, err := os.Open(filename)
	defer fp.Close()
	if err != nil {
		log.Println(err)
		return nil, err
	}
	return ReadNewJSON(fp)
}

// WriteXML writes the tree to an XML-encoded byte string over io.Writer
// using MarshalXML.
func (n *Node) WriteXML(writer io.Writer, indent bool) error {
	err := n.ThisCheck()
	if err != nil {
		log.Println(err)
		return err
	}
	var b []byte
	if indent {
		b, err = xml.MarshalIndent(n.This(), "", "  ")
	} else {
		b, err = xml.Marshal(n.This())
	}
	if err != nil {
		log.Println(err)
		return err
	}
	_, err = writer.Write(b)
	if err != nil {
		log.Println(err)
		return err
	}
	return nil
}

// ReadXML reads the tree from an XML-encoded byte string over io.Reader, calls
// UnmarshalPost to recover pointers from paths.
func (n *Node) ReadXML(reader io.Reader) error {
	var err error
	if err = n.ThisCheck(); err != nil {
		log.Println(err)
		return err
	}
	b, err := ioutil.ReadAll(reader)
	if err != nil {
		log.Println(err)
		return err
	}
	updt := n.UpdateStart()
	err = xml.Unmarshal(b, n.This()) // key use of this!
	if err == nil {
		n.UnmarshalPost()
	}
	n.UpdateEnd(updt)
	return nil
}

// ParentAllChildren walks the tree down from current node and call
// SetParent on all children -- needed after an Unmarshal.
func (n *Node) ParentAllChildren() {
	n.FuncDownMeFirst(0, nil, func(k Ki, level int, d interface{}) bool {
		for _, child := range *k.Children() {
			if child != nil {
				child.SetParent(k)
			} else {
				return false
			}
		}
		return true
	})
}

// UnmarshalPost must be called after an Unmarshal -- calls
// SetPtrsFmPaths and ParentAllChildren.
func (n *Node) UnmarshalPost() {
	n.ParentAllChildren()
	n.SetPtrsFmPaths()
}

// Deleted manages all the deleted Ki elements, that are destined to then be
// destroyed, without having an additional pointer on the Ki object
type Deleted struct {
	Dels []Ki
	Mu   sync.Mutex
}

// DelMgr is the manager of all deleted items
var DelMgr = Deleted{}

// Add the Ki elements to the deleted list
func (dm *Deleted) Add(kis ...Ki) {
	dm.Mu.Lock()
	if dm.Dels == nil {
		dm.Dels = make([]Ki, 0, 1000)
	}
	dm.Dels = append(dm.Dels, kis...)
	dm.Mu.Unlock()
}

func (dm *Deleted) DestroyDeleted() {
	dm.Mu.Lock()
	curdels := make([]Ki, len(dm.Dels))
	copy(curdels, dm.Dels)
	dm.Dels = dm.Dels[:0]
	dm.Mu.Unlock()
	for _, k := range curdels {
		k.Destroy() // destroy will add to the dels so we need to do this outside of lock
	}
}
