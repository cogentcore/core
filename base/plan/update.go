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
	"log/slog"
	"slices"

	"cogentcore.org/core/base/slicesx"
)

// Namer is an interface that types can implement to specify their name in a plan context.
type Namer interface {

	// PlanName returns the name of the object in a plan context.
	PlanName() string
}

// Update ensures that the elements of the slice contain
// the elements according to the plan, specified by unique
// element names, with n = total number of items in the target slice.
// If a new item is needed then new is called to create it,
// for given name at given index position.
// if destroy is not-nil, then it is called on any element
// that is being deleted from the slice.
// It returns the updated slice and whether any changes were made.
func Update[T Namer](s []T, n int, name func(i int) string, new func(name string, i int) T, destroy func(e T)) (r []T, mods bool) {
	// first make a map for looking up the indexes of the target names
	names := make([]string, n)
	nmap := make(map[string]int, n)
	smap := make(map[string]int, n)
	for i := range n {
		nm := name(i)
		names[i] = nm
		if _, has := nmap[nm]; has {
			slog.Error("plan.Build: duplicate name", "name", nm)
		}
		nmap[nm] = i
	}
	// first remove anything we don't want
	r = s
	rn := len(r)
	for i := rn - 1; i >= 0; i-- {
		nm := r[i].PlanName()
		if _, ok := nmap[nm]; !ok {
			mods = true
			if destroy != nil {
				destroy(r[i])
			}
			r = slices.Delete(r, i, i+1)
		}
		smap[nm] = i
	}
	// next add and move items as needed; in order so guaranteed
	for i, tn := range names {
		ci := slicesx.Search(r, func(e T) bool { return e.PlanName() == tn }, smap[tn])
		if ci < 0 { // item not currently on the list
			mods = true
			ne := new(tn, i)
			r = slices.Insert(r, i, ne)
		} else { // on the list; is it in the right place?
			if ci != i {
				mods = true
				e := r[ci]
				r = slices.Delete(r, ci, ci+1)
				r = slices.Insert(r, i, e)
			}
		}
	}
	return
}
