// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package ki

import (
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"sync/atomic"
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
	n := this.AsKi()
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
		err := fmt.Errorf("ki.Node %q ThisCheck: node has null 'this' pointer; must call Init or InitName on root nodes", k.Path())
		slog.Error(err.Error())
		return err
	}
	return nil
}

// SetParent just sets parent of node (and inherits update count from
// parent, to keep consistent).
// Assumes not already in a tree or anything.
func SetParent(kid Ki, parent Ki) {
	n := kid.AsKi()
	n.Par = parent
	if parent != nil {
		pn := parent.AsKi()
		c := atomic.AddUint64(&pn.NumLifetimeKids, 1)
		if kid.Name() == "" {
			kid.SetName(kid.KiType().IDName + "-" + strconv.FormatUint(c-1, 10)) // must subtract 1 so we start at 0
		}
	}
	kid.This().OnAdd()
	n.WalkUpParent(func(k Ki) bool {
		k.This().OnChildAdded(kid)
		return Continue
	})
}

// MoveToParent deletes given node from its current parent and adds it as a child
// of given new parent.  Parents could be in different trees or not.
func MoveToParent(kid Ki, parent Ki) {
	// TODO(kai/ki): implement MoveToParent
	// oldPar := kid.Parent()
	// if oldPar != nil {
	// 	oldPar.DeleteChild(kid, false)
	// }
	// parent.AddChild(kid)
}

// New adds a new child of the given the type
// with the given name to the given parent.
// If the name is unspecified, it defaults to the
// ID (kebab-case) name of the type, plus the
// [Ki.NumLifetimeChildren] of its parent.
// It is a helper function that calls [Ki.NewChild].
func New[T Ki](par Ki, name ...string) T {
	var n T
	return par.NewChild(n.KiType(), name...).(T)
}

// NewRoot returns a new root node of the given the type
// with the given name. If the name is unspecified, it
// defaults to the ID (kebab-case) name of the type.
// It is a helper function that calls [Ki.InitName].
func NewRoot[T Ki](name ...string) T {
	var n T
	n = n.New().(T)
	n.InitName(n, name...)
	return n
}

// InsertNewChild is a generic helper function for [Ki.InsertNewChild]
func InsertNewChild[T Ki](par Ki, at int, name ...string) T {
	var n T
	return par.InsertNewChild(n.KiType(), at, name...).(T)
}

// ParentByType is a generic helper function for [Ki.ParentByType]
func ParentByType[T Ki](k Ki, embeds bool) T {
	var n T
	v, _ := k.ParentByType(n.KiType(), embeds).(T)
	return v
}

// ChildByType is a generic helper function for [Ki.ChildByType]
func ChildByType[T Ki](k Ki, embeds bool, startIdx ...int) T {
	var n T
	v, _ := k.ChildByType(n.KiType(), embeds, startIdx...).(T)
	return v
}

// IsRoot tests if this node is the root node -- checks Parent = nil.
func IsRoot(k Ki) bool {
	return k.This() == nil || k.Parent() == nil || k.Parent().This() == nil
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
	return kn.AsKi().depth
}

// SetDepth sets the current depth of the node to given value.
func SetDepth(kn Ki, depth int) {
	kn.AsKi().depth = depth
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
