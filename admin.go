// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package ki

import (
	"fmt"
	"log"
	"strings"
	"sync"

	"goki.dev/gti"
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
	this.ClearUpdateFlags()
	if n.Ths != this {
		n.Ths = this
		n.Ths.OnInit()
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
	kid.This().OnAdd()
	n.WalkUpParent(func(k Ki) bool {
		k.This().OnChildAdded(kid)
		return Continue
	})
	if parent != nil {
		parup := parent.Is(Updating)
		n.WalkPre(func(k Ki) bool {
			k.SetFlag(parup, Updating)
			return Continue
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

// DeleteFromParent calls OnChildDeleting on all parents of given node
// then calls OnDelete on the node, and finally sets the Parent to nil.
// Call this *before* deleting the child.
func DeleteFromParent(kid Ki) {
	if kid.Parent() == nil {
		return
	}
	n := kid.AsNode()
	n.WalkUpParent(func(k Ki) bool {
		k.This().OnChildDeleting(kid)
		return Continue
	})
	kid.SetFlag(true, Deleted)
	kid.This().OnDelete()
	SetParent(kid, nil)
}

// DeletingChildren calls OnChildrenDeleting on given node
// and all parents thereof.
// Call this *before* deleting the children.
func DeletingChildren(k Ki) {
	k.This().OnChildrenDeleting()
	n := k.AsNode()
	n.WalkUpParent(func(k Ki) bool {
		k.This().OnChildrenDeleting()
		return Continue
	})
}

// New adds a new child of the given the type
// with the given name to the given parent.
// It is a helper function that calls [Ki.NewChild].
func New[T Ki](par Ki, name string) T {
	return par.NewChild(gti.TypeByValue((*T)(nil)), name).(T)
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
// property of the node (e.g., used in WalkBreadth).
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
	kn.WalkPre(func(k Ki) bool {
		k.SetFlag(false, Updating)
		return true
	})
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
	kn.WalkPre(func(k Ki) bool {
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
	kn.WalkPre(func(k Ki) bool {
		UniquifyNames(k)
		return Continue
	})
}

//////////////////////////////////////////////////////////////////////////////
//  Deletion manager

// DeletedKi manages all the deleted Ki elements, that are destined to then be
// destroyed, without having an additional pointer on the Ki object
type DeletedKi struct {
	Dels []Ki
	Mu   sync.Mutex
}

// DelMgr is the manager of all deleted items
var DelMgr = DeletedKi{}

// Add the Ki elements to the deleted list
func (dm *DeletedKi) Add(kis ...Ki) {
	dm.Mu.Lock()
	if dm.Dels == nil {
		dm.Dels = make([]Ki, 0)
	}
	dm.Dels = append(dm.Dels, kis...)
	dm.Mu.Unlock()
}

// DestroyDeleted destroys any deleted items in list
func (dm *DeletedKi) DestroyDeleted() {
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
