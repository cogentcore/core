// Copyright (c) 2018, Randall C. O'Reilly. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package kit provides a TypeRegistry for associating string names with
// reflect.Type values, to allow dynamic marshaling of structs, and also
// bidirectional string conversion of const int iota (enum) types.  It is used
// by the GoKi ki system, hence the kit (ki types) name.
//
// To register a new type, add:
//
// var KiT_TypeName = kit.Types.AddType(&TypeName{}, [props|nil])
//
// where the props is a map[string]interface{} of optional properties that can
// be associated with the type -- this is used in the GoGi graphical interface
// system for example to color objects of different types using the
// background-color property.  KiT_TypeName variable can be conveniently used
// wherever a reflect.Type of that type is needed.
//
// The kit.Type struct provides JSON and XML Marshal / Unmarshal functions for
// saving / loading reflect.Type using registrered type names.
//
// Also provided are robust interface{}-based type conversion routines in
// convert.go that are useful in more lax user-interface contexts where
// "common sense" conversions between strings, numbers etc are useful
package kit

// github.com/rcoreilly/goki/ki/kit

import (
	// "fmt"
	// "log"
	"reflect"
)

// TypeRegistry is a map from type name to reflect.Type -- need to explicitly
// register each new type by calling AddType in the process of creating a new
// global variable, as in:
//
// var KiT_TypeName = ki.Types.AddType(&TypeName{}, [props|nil])
//
// where TypeName is the name of the type -- note that it is ESSENTIAL to pass a pointer
// so that the type is considered addressable, even after we get Elem() of it.
//
// props is a map[string]interface{} of optional properties that can be
// associated with the type -- this is used in the GoGi graphical interface
// system for example to color objects of different types using the
// background-color property.
type TypeRegistry struct {
	// to get a type from its name
	Types map[string]reflect.Type
	// type properties -- nodes can get default properties from their types and then optionally override them with their own settings
	Props map[string]map[string]interface{}
}

// Types is master registry of types that embed Ki Nodes
var Types TypeRegistry

// AddType adds a given type to the registry -- requires an empty object to grab type info from -- must be passed as a pointer to ensure that it is an addressable, settable type -- also optional properties that can be associated with the type and accessible e.g. for view-specific properties etc
func (tr *TypeRegistry) AddType(obj interface{}, props map[string]interface{}) reflect.Type {
	if tr.Types == nil {
		tr.Types = make(map[string]reflect.Type)
		tr.Props = make(map[string]map[string]interface{})
	}

	typ := reflect.TypeOf(obj).Elem()
	tn := typ.Name()
	tr.Types[tn] = typ
	// fmt.Printf("added type: %v\n", tn)
	if props != nil {
		// fmt.Printf("added props: %v\n", tn)
		tr.Props[tn] = props
	}
	return typ
}

// FindType finds a type based on its name -- returns nil if not found
func (tr *TypeRegistry) FindType(name string) reflect.Type {
	return tr.Types[name]
}

// Properties returns properties for this type -- makes props map if not already made
func (tr *TypeRegistry) Properties(typeName string) map[string]interface{} {
	tp, ok := tr.Props[typeName]
	if !ok {
		tp = make(map[string]interface{})
		tr.Props[typeName] = tp
	}
	return tp
}

// Prop safely finds a type property from type name and property key -- nil if not found
func (tr *TypeRegistry) Prop(typeName, propKey string) interface{} {
	tp, ok := tr.Props[typeName]
	if !ok {
		// fmt.Printf("no props for type: %v\n", typeName)
		return nil
	}
	p, ok := tp[propKey]
	if !ok {
		// fmt.Printf("no props for key: %v\n", propKey)
		return nil
	}
	return p
}
