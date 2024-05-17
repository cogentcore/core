// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package config provides an efficent mechanism for updating a slice
// to contain a target list of elements, generating minimal edits to
// modify the current slice contents to match the target.
// The mechanism depends on the use of unique name string identifiers
// to determine whether an element is currently configured correctly.
// These could be algorithmically generated hash strings or any other
// such unique identifier.
package config

import (
	"log/slog"
	"slices"

	"cogentcore.org/core/base/findfast"
	"cogentcore.org/core/base/namer"
)

// Config ensures that the elements of the slice contain
// the desired elements in a specific order, specified by unique
// element names, with n = total number of items in the target slice.
// If a new item is needed then newEl is called to create it,
// for given name at given index position.
// Returns the updated slice and true if any changes were made.
func Config[T namer.Namer](s []T, n int, name func(i int) string, newEl func(name string, i int) T) (r []T, mods bool) {
	// first make a map for looking up the indexes of the target names
	names := make([]string, n)
	nmap := make(map[string]int, n)
	smap := make(map[string]int, n)
	for i := range n {
		nm := name(i)
		names[i] = nm
		if _, has := nmap[nm]; has {
			slog.Error("config.Config: duplicate name", "name", nm)
		}
		nmap[nm] = i
	}
	// first remove anything we don't want
	r = s
	rn := len(r)
	for i := rn - 1; i >= 0; i-- {
		nm := r[i].Name()
		if _, ok := nmap[nm]; !ok {
			mods = true
			// fmt.Println("delete at:", i, "bad name:", nm)
			r = slices.Delete(r, i, i+1)
		}
		smap[nm] = i
	}
	// next add and move items as needed -- in order so guaranteed
	for i, tn := range names {
		ci := findfast.FindName(r, tn, smap[tn])
		if ci < 0 { // item not currently on the list
			mods = true
			ne := newEl(tn, i)
			r = slices.Insert(r, i, ne)
			// fmt.Println("new item needed at:", i, "name:", tn)
		} else { // on the list -- is it in the right place?
			if ci != i {
				mods = true
				e := r[ci]
				r = slices.Delete(r, ci, ci+1)
				r = slices.Insert(r, i, e)
				// fmt.Println("moved item:", tn, "from:", ci, "to:", i)
			}
		}
	}
	return
}
