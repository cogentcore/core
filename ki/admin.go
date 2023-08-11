// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package ki

import (
	"fmt"
	"log"
	"reflect"
	"strings"
	"sync"
	"unsafe"

	"github.com/goki/ki/kit"
)

// admin has infrastructure level code, outside of ki interface

// InitNode initializes the node -- automatically called during Add/Insert
// Child -- sets the This pointer for this node as a Ki interface (pass
// pointer to node as this arg) -- Go cannot always access the true
// underlying type for structs using embedded Ki objects (when these objs
// are receivers to methods) so we need a This interface pointer that
// guarantees access to the Ki interface in a way that always reveals the
// underlying type (e.g., in reflect calls).  Calls Init on Ki fields
// within struct, sets their names to the field name, and sets us as their
// parent.
func InitNode(this Ki) {
	n := this.AsNode()
	this.ClearFlagMask(int64(UpdateFlagsMask))
	if n.Ths != this {
		n.Ths = this
		n.Ths.OnInit()
		KiInitKiFields(this)
	}
}

// ThisCheck checks that the This pointer is set and issues a warning to
// log if not -- returns error if not set -- called when nodes are added
// and inserted.
func ThisCheck(k Ki) error {
	if k.This() == nil {
		err := fmt.Errorf("Ki Node %v ThisCheck: node has null 'this' pointer -- must call Init or InitName on root nodes!", k.Name())
		log.Print(err)
		return err
	}
	return nil
}

// SetParent just sets parent of node (and inherits update count from
// parent, to keep consistent).
// Assumes not already in a tree or anything.
func SetParent(kid Ki, parent Ki) {
	n := kid.AsNode()
	n.Par = parent
	kid.OnAdd()
	n.FuncUpParent(0, nil, func(k Ki, level int, data any) bool {
		k.OnChildAdded(kid)
		return Continue
	})
	if parent != nil && !parent.OnlySelfUpdate() {
		parup := parent.IsUpdating()
		n.FuncDownMeFirst(0, nil, func(k Ki, level int, d any) bool {
			k.SetFlagState(parup, int(Updating))
			return true
		})
	}
}

// MoveToParent deletes given node from its current parent and adds it as a child
// of given new parent.  Parents could be in different trees or not.
func MoveToParent(kid Ki, parent Ki) {
	oldPar := kid.Parent()
	if oldPar != nil {
		SetParent(kid, nil)
		oldPar.DeleteChild(kid, false)
	}
	parent.AddChild(kid)
}

// IsRoot tests if this node is the root node -- checks Parent = nil.
func IsRoot(k Ki) bool {
	if k.This() == nil || k.Parent() == nil || k.Parent().This() == nil {
		return true
	}
	return false
}

// Root returns the root node of given ki node in tree (the node with a nil parent).
func Root(k Ki) Ki {
	if IsRoot(k) {
		return k.This()
	}
	return Root(k.Parent())
}

// Depth returns the current depth of the node.
// This is only valid in a given context, not a stable
// property of the node (e.g., used in FuncDownBreadthFirst).
func Depth(kn Ki) int {
	return kn.AsNode().depth
}

// SetDepth sets the current depth of the node to given value.
func SetDepth(kn Ki, depth int) {
	kn.AsNode().depth = depth
}

// UpdateReset resets Updating flag for this node and all children -- in
// case they are out-of-sync due to more complex tree maninpulations --
// only call at a known point of non-updating.
func UpdateReset(kn Ki) {
	if kn.OnlySelfUpdate() {
		kn.ClearFlag(int(Updating))
	} else {
		kn.FuncDownMeFirst(0, nil, func(k Ki, level int, d any) bool {
			k.ClearFlag(int(Updating))
			return true
		})
	}
}

//////////////////////////////////////////////////////////////////
// Fields

// KiInitKiFields calls Init on all Ki fields
// within the struct, sets their names to the field name,
// and sets us as their parent.
func KiInitKiFields(this Ki) {
	n := this.AsNode()
	if KiHasKiFields(n) {
		fnms := KiFieldNames(n)
		val := reflect.ValueOf(this).Elem()
		for _, fnm := range fnms {
			fldval := val.FieldByName(fnm)
			fk := kit.PtrValue(fldval).Interface().(Ki)
			fk.SetFlag(int(IsField))
			fk.InitName(fk, fnm)
			SetParent(fk, this)
		}
	}
}

// FieldRoot returns the field root object for this node -- the node that
// owns the branch of the tree rooted in one of its fields -- i.e., the
// first non-Field parent node after the first Field parent node -- can be
// nil if no such thing exists for this node.
func FieldRoot(kn Ki) Ki {
	var root Ki
	gotField := false
	kn.FuncUpParent(0, kn, func(k Ki, level int, d any) bool {
		if !gotField {
			if k.IsField() {
				gotField = true
			}
			return Continue
		}
		if !k.IsField() {
			root = k
			return Break
		}
		return Continue
	})
	return root
}

// KiFieldOffs returns the uintptr offsets for Ki fields of this Node.
// Cached for fast access, but use HasKiFields for even faster checking.
func KiFieldOffs(n *Node) []uintptr {
	if n.fieldOffs != nil {
		return n.fieldOffs
	}
	// we store the offsets for the fields in type properties
	tprops := *kit.Types.Properties(Type(n.This()), true) // true = makeNew
	if foff, ok := kit.TypeProp(tprops, "__FieldOffs"); ok {
		n.fieldOffs = foff.([]uintptr)
		return n.fieldOffs
	}
	foff, _ := KiFieldsInit(n)
	return foff
}

// KiHasKiFields returns true if this node has Ki Node fields that are
// included in recursive descent traversal of the tree.  This is very
// efficient compared to accessing the field information on the type
// so it should be checked first -- caches the info on the node in flags.
func KiHasKiFields(n *Node) bool {
	if n.HasFlag(int(HasKiFields)) {
		return true
	}
	if n.HasFlag(int(HasNoKiFields)) {
		return false
	}
	foffs := KiFieldOffs(n)
	if len(foffs) == 0 {
		n.SetFlag(int(HasNoKiFields))
		return false
	}
	n.SetFlag(int(HasKiFields))
	return true
}

// NumKiFields returns the number of Ki Node fields on this node.
// This calls HasKiFields first so it is also efficient.
func NumKiFields(n *Node) int {
	if !KiHasKiFields(n) {
		return 0
	}
	foffs := KiFieldOffs(n)
	return len(foffs)
}

// KiField returns the Ki Node field at given index, from KiFieldOffs list.
// Returns nil if index is out of range.  This is generally used for
// generic traversal methods and thus does not have a Try version.
func KiField(n *Node, idx int) Ki {
	if !KiHasKiFields(n) {
		return nil
	}
	foffs := KiFieldOffs(n)
	if idx >= len(foffs) || idx < 0 {
		return nil
	}
	fn := (*Node)(unsafe.Pointer(uintptr(unsafe.Pointer(n)) + foffs[idx]))
	return fn.This()
}

// KiFieldByName returns field Ki element by name -- returns false if not found.
func KiFieldByName(n *Node, name string) Ki {
	if !KiHasKiFields(n) {
		return nil
	}
	foffs := KiFieldOffs(n)
	op := uintptr(unsafe.Pointer(n))
	for _, fo := range foffs {
		fn := (*Node)(unsafe.Pointer(op + fo))
		if fn.Nm == name {
			return fn.This()
		}
	}
	return nil
}

// KiFieldNames returns the field names for Ki fields of this Node. Cached for fast access.
func KiFieldNames(n *Node) []string {
	// we store the offsets for the fields in type properties
	tprops := *kit.Types.Properties(Type(n.This()), true) // true = makeNew
	if fnms, ok := kit.TypeProp(tprops, "__FieldNames"); ok {
		return fnms.([]string)
	}
	_, fnm := KiFieldsInit(n)
	return fnm
}

// KiFieldsInit initializes cached data about the KiFields in this node
// offsets and names -- returns them
func KiFieldsInit(n *Node) (foff []uintptr, fnm []string) {
	foff = make([]uintptr, 0)
	fnm = make([]string, 0)
	kitype := KiType
	FlatFieldsValueFunc(n.This(), func(stru any, typ reflect.Type, field reflect.StructField, fieldVal reflect.Value) bool {
		if fieldVal.Kind() == reflect.Struct && kit.EmbedImplements(field.Type, kitype) {
			foff = append(foff, field.Offset)
			fnm = append(fnm, field.Name)
		}
		return true
	})
	tprops := *kit.Types.Properties(Type(n), true) // true = makeNew
	kit.SetTypeProp(tprops, "__FieldOffs", foff)
	n.fieldOffs = foff
	kit.SetTypeProp(tprops, "__FieldNames", fnm)
	return
}

// FieldByName returns field value by name (can be any type of field --
// see KiFieldByName for Ki fields) -- returns nil if not found.
func FieldByName(kn Ki, field string) any {
	return kit.FlatFieldInterfaceByName(kn.This(), field)
}

// FieldByNameTry returns field value by name (can be any type of field --
// see KiFieldByName for Ki fields) -- returns error if not found.
func FieldByNameTry(kn Ki, field string) (any, error) {
	fld := FieldByName(kn, field)
	if fld != nil {
		return fld, nil
	}
	return nil, fmt.Errorf("ki %v: field named: %v not found", kn.Path(), field)
}

// FieldTag returns given field tag for that field, or empty string if not set.
func FieldTag(kn Ki, field, tag string) string {
	return kit.FlatFieldTag(Type(kn.This()), field, tag)
}

//////////////////////////////////////////////////
//  Unique Names

// UniqueNameCheck checks if all the children names are unique or not.
// returns true if all names are unique; false if not
// if not unique, call UniquifyNames or take other steps to ensure uniqueness.
func UniqueNameCheck(k Ki) bool {
	kk := *k.Children()
	sz := len(kk)
	nmap := make(map[string]struct{}, sz)
	for _, child := range kk {
		if child == nil {
			continue
		}
		nm := child.Name()
		_, hasnm := nmap[nm]
		if hasnm {
			return false
		}
		nmap[nm] = struct{}{}
	}
	return true
}

// UniqueNameCheckAll checks entire tree from given node,
// if all the children names are unique or not.
// returns true if all names are unique; false if not
// if not unique, call UniquifyNames or take other steps to ensure uniqueness.
func UniqueNameCheckAll(kn Ki) bool {
	allunq := true
	kn.FuncDownMeFirst(0, nil, func(k Ki, level int, d any) bool {
		unq := UniqueNameCheck(k)
		if !unq {
			allunq = false
			return Break
		}
		return Continue
	})
	return allunq
}

// UniquifyIndexAbove is the number of children above which UniquifyNamesAddIndex
// is called -- that is much faster for large numbers of children.
// Must be < 1000
var UniquifyIndexAbove = 1000

// UniquifyNamesAddIndex makes sure that the names are unique by automatically
// adding a suffix with index number, separated by underbar.
// Empty names get the parent name as a prefix.
// if there is an existing underbar, then whatever is after it is replaced with
// the unique index, ensuring that multiple calls are safe!
func UniquifyNamesAddIndex(kn Ki) {
	kk := *kn.Children()
	sz := len(kk)
	sfmt := "%s_%05d"
	switch {
	case sz > 9999999:
		sfmt = "%s_%10d"
	case sz > 999999:
		sfmt = "%s_%07d"
	case sz > 99999:
		sfmt = "%s_%06d"
	}
	parnm := "c"
	if kn.Parent() != nil {
		parnm = kn.Parent().Name()
	}
	for i, child := range kk {
		if child == nil {
			continue
		}
		nm := child.Name()
		if nm == "" {
			child.SetName(fmt.Sprintf(sfmt, parnm, i))
		} else {
			ubi := strings.LastIndex(nm, "_")
			if ubi > 0 {
				nm = nm[ubi+1:]
			}
			child.SetName(fmt.Sprintf(sfmt, nm, i))
		}
	}
}

// UniquifyNames makes sure that the names are unique.
// If number of children >= UniquifyIndexAbove, then UniquifyNamesAddIndex
// is called, for faster performance.
// Otherwise, existing names are preserved if they are unique, and only
// duplicates are renamed.  This is a bit slower.
func UniquifyNames(kn Ki) {
	kk := *kn.Children()
	sz := len(kk)
	if sz >= UniquifyIndexAbove {
		UniquifyNamesAddIndex(kn)
		return
	}
	parnm := "c"
	if kn.Parent() != nil {
		parnm = kn.Parent().Name()
	}
	nmap := make(map[string]struct{}, sz)
	for i, child := range kk {
		if child == nil {
			continue
		}
		nm := child.Name()
		if nm == "" {
			nm = fmt.Sprintf("%s_%03d", parnm, i)
			child.SetName(nm)
		} else {
			_, hasnm := nmap[nm]
			if hasnm {
				ubi := strings.LastIndex(nm, "_")
				if ubi > 0 {
					nm = nm[ubi+1:]
				}
				nm = fmt.Sprintf("%s_%03d", nm, i)
				child.SetName(nm)
			}
		}
		nmap[nm] = struct{}{}
	}
}

// UniquifyNamesAll makes sure that the names are unique for entire tree
// If number of children >= UniquifyIndexAbove, then UniquifyNamesAddIndex
// is called, for faster performance.
// Otherwise, existing names are preserved if they are unique, and only
// duplicates are renamed.  This is a bit slower.
func UniquifyNamesAll(kn Ki) {
	kn.FuncDownMeFirst(0, nil, func(k Ki, level int, d any) bool {
		UniquifyNames(k)
		return Continue
	})
}

//////////////////////////////////////////////////////////////////////////////
//  Deletion manager

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
		dm.Dels = make([]Ki, 0)
	}
	dm.Dels = append(dm.Dels, kis...)
	dm.Mu.Unlock()
}

// DestroyDeleted destroys any deleted items in list
func (dm *Deleted) DestroyDeleted() {
	// pr := prof.Start("ki.DestroyDeleted")
	// defer pr.End()
	dm.Mu.Lock()
	curdels := dm.Dels
	dm.Dels = make([]Ki, 0)
	dm.Mu.Unlock()
	for _, k := range curdels {
		if k == nil {
			continue
		}
		k.Destroy() // destroy will add to the dels so we need to do this outside of lock
	}
}
