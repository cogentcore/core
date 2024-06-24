// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package plan provides an efficient mechanism for updating a slice
// to contain a target list of elements, generating minimal edits to
// modify the current slice contents to match the target.
// The mechanism depends on the use of unique name string identifiers
// to determine whether an element is currently configured correctly.
// These could be algorithmically generated hash strings or any other
// such unique identifier.
package plan

import (
	"slices"

	"cogentcore.org/core/base/slicesx"
)

// Namer is an interface that types can implement to specify their name in a plan context.
type Namer interface {

	// PlanName returns the name of the object in a plan context.
	PlanName() string
}

// Update ensures that the elements of the given slice contain
// the elements according to the plan specified by the given arguments.
// The argument n specifies the total number of items in the target plan.
// The elements have unique names specified by the given name function.
// If a new item is needed, the given new function is called to create it
// for the given name at the given index position. After a new element is
// created, it is added to the slice, and if the given optional init function
// is non-nil, it is called with the new element and its index. If the
// given destroy function is not-nil, then it is called on any element
// that is being deleted from the slice. Update returns whether any changes
// were made. The given slice must be a pointer so that it can be modified
// live, which is required for init functions to run when the slice is
// correctly updated to the current state.
func Update[T Namer](s *[]T, n int, name func(i int) string, new func(name string, i int) T, init func(e T, i int), destroy func(e T)) bool {
	changed := false
	// first make a map for looking up the indexes of the target names
	names := make([]string, n)
	nmap := make(map[string]int, n)
	smap := make(map[string]int, n)
	for i := range n {
		nm := name(i)
		names[i] = nm
		if _, has := nmap[nm]; has {
			panic("plan.Update: duplicate name: " + nm) // no way to recover
		}
		nmap[nm] = i
	}
	// first remove anything we don't want
	sn := len(*s)
	for i := sn - 1; i >= 0; i-- {
		nm := (*s)[i].PlanName()
		if _, ok := nmap[nm]; !ok {
			changed = true
			if destroy != nil {
				destroy((*s)[i])
			}
			*s = slices.Delete(*s, i, i+1)
		}
		smap[nm] = i
	}
	// next add and move items as needed; in order so guaranteed
	for i, tn := range names {
		ci := slicesx.Search(*s, func(e T) bool { return e.PlanName() == tn }, smap[tn])
		if ci < 0 { // item not currently on the list
			changed = true
			ne := new(tn, i)
			*s = slices.Insert(*s, i, ne)
			if init != nil {
				init(ne, i)
			}
		} else { // on the list; is it in the right place?
			if ci != i {
				changed = true
				e := (*s)[ci]
				*s = slices.Delete(*s, ci, ci+1)
				*s = slices.Insert(*s, i, e)
			}
		}
	}
	return changed
}
