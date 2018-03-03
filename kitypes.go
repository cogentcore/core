// Copyright (c) 2018, Randall C. O'Reilly. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Ki is the base element of GoKi Trees
// Ki = Tree in Japanese, and "Key" in English
package ki

import (
	"fmt"
	"reflect"
)

// map from type name to reflect.Type -- need to explicitly register each new type

type TypeRegistry struct {
	types map[string]reflect.Type
}

// this is master registry of Ki types
var KiTypes TypeRegistry

func (tr *TypeRegistry) AddType(obj interface{}) reflect.Type {
	if tr.types == nil {
		tr.types = make(map[string]reflect.Type)
	}

	typ := reflect.TypeOf(obj).Elem()
	tr.types[typ.Name()] = typ
	fmt.Printf("added type: %v\n", typ.Name())
	return typ
}

func (tr *TypeRegistry) GetType(name string) reflect.Type {
	return tr.types[name]
}
