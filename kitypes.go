// Copyright (c) 2018, Randall C. O'Reilly. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package ki

import (
	// "fmt"
	"reflect"
)

// TypeRegistry is a map from type name to reflect.Type -- need to explicitly register each new type by calling AddType in the process of creating a new global variable, as in:
// 	var KtTypeName = KiTypes.AddType(&TypeName{})
// 	where TypeName is the name of the type
type TypeRegistry struct {
	Types map[string]reflect.Type
}

// KiTypes is master registry of types that embed Ki Nodes
var KiTypes TypeRegistry

// AddType adds a given type to the registry -- requires an empty object to grab type info from
func (tr *TypeRegistry) AddType(obj interface{}) reflect.Type {
	if tr.Types == nil {
		tr.Types = make(map[string]reflect.Type)
	}

	typ := reflect.TypeOf(obj).Elem()
	tr.Types[typ.Name()] = typ
	// fmt.Printf("added type: %v\n", typ.Name())
	return typ
}

// GetType finds a type based on its name -- returns nil if not found
func (tr *TypeRegistry) GetType(name string) reflect.Type {
	return tr.Types[name]
}
