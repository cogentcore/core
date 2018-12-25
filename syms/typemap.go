// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package syms

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

// TypeKind is used for initialization of builtin typemaps
type TypeKind struct {
	Name string
	Kind Kinds
}
