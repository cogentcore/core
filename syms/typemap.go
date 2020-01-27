// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package syms

import (
	"io"
	"sort"
	"strings"
)

// TypeMap is a map of types for quick looking up by name
type TypeMap map[string]*Type

// Alloc ensures that map is made
func (tm *TypeMap) Alloc() {
	if *tm == nil {
		*tm = make(TypeMap)
	}
}

// Add adds a type to the map, handling allocation for nil maps
func (tm *TypeMap) Add(ty *Type) {
	tm.Alloc()
	(*tm)[ty.Name] = ty
}

// CopyFrom copies all the types from given source map into this one
func (tm *TypeMap) CopyFrom(src TypeMap) {
	tm.Alloc()
	for nm, sty := range src {
		_, has := (*tm)[nm]
		if !has {
			(*tm)[nm] = sty
			continue
		}
		// todo: any merging?
	}
}

// Clone returns deep copy of this type map -- types are Clone() copies.
// returns nil if this map is empty
func (tm *TypeMap) Clone() TypeMap {
	sz := len(*tm)
	if sz == 0 {
		return nil
	}
	ntm := make(TypeMap, sz)
	for nm, sty := range *tm {
		ntm[nm] = sty.Clone()
	}
	return ntm
}

// Names returns a slice of the names in this map, optionally sorted
func (tm *TypeMap) Names(sorted bool) []string {
	nms := make([]string, len(*tm))
	idx := 0
	for _, ty := range *tm {
		nms[idx] = ty.Name
		idx++
	}
	if sorted {
		sort.StringSlice(nms).Sort()
	}
	return nms
}

// KindNames returns a slice of the kind:names in this map, optionally sorted
func (tm *TypeMap) KindNames(sorted bool) []string {
	nms := make([]string, len(*tm))
	idx := 0
	for _, ty := range *tm {
		nms[idx] = ty.Kind.String() + ":" + ty.Name
		idx++
	}
	if sorted {
		sort.StringSlice(nms).Sort()
	}
	return nms
}

// WriteDoc writes basic doc info, sorted by kind and name
func (tm *TypeMap) WriteDoc(out io.Writer, depth int) {
	nms := tm.KindNames(true)
	for _, nm := range nms {
		ci := strings.Index(nm, ":")
		ty := (*tm)[nm[ci+1:]]
		ty.WriteDoc(out, depth)
	}
}

// TypeKindSize is used for initialization of builtin typemaps
type TypeKindSize struct {
	Name string
	Kind Kinds
	Size int
}
