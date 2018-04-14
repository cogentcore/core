// Copyright (c) 2018, Randall C. O'Reilly. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package kit provides various reflect type functions for GoKi system, including:
//
// * kit.TypeRegistry (types.go) for associating string names with
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
// * kit.EnumRegistry (enums.go) that registers constant int iota (aka enum) types, and
// provides general conversion utilities to / from string, int64, general
// properties associated with enum types, and deals with bit flags
//
// * kit.Type (type.go) struct provides JSON and XML Marshal / Unmarshal functions for
// saving / loading reflect.Type using registrered type names.
//
// * convert.go: robust interface{}-based type conversion routines that are
// useful in more lax user-interface contexts where "common sense" conversions
// between strings, numbers etc are useful
//
// * embeds.go: various functions for managing embedded struct types, e.g.,
// determining if a given type embeds another type (directly or indirectly),
// and iterating over fields to flatten the otherwise nested nature of the
// field encoding in embedded types.
package kit

// github.com/rcoreilly/goki/ki/kit

import (
	// "fmt"
	// "log"
	"log"
	"reflect"
)

// TypeRegistry is a map from type name (package path + "." + type name) to
// reflect.Type -- need to explicitly register each new type by calling
// AddType in the process of creating a new global variable, as in:
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

// the full package-qualified type name -- this is what is used for encoding
// type names in the registry
func FullTypeName(typ reflect.Type) string {
	return typ.PkgPath() + "." + typ.Name()
}

// AddType adds a given type to the registry -- requires an empty object to grab type info from -- must be passed as a pointer to ensure that it is an addressable, settable type -- also optional properties that can be associated with the type and accessible e.g. for view-specific properties etc
func (tr *TypeRegistry) AddType(obj interface{}, props map[string]interface{}) reflect.Type {
	if tr.Types == nil {
		tr.Types = make(map[string]reflect.Type)
		tr.Props = make(map[string]map[string]interface{})
	}

	typ := reflect.TypeOf(obj).Elem()
	tn := FullTypeName(typ)
	tr.Types[tn] = typ
	// fmt.Printf("added type: %v\n", tn)
	if props != nil {
		// fmt.Printf("added props: %v\n", tn)
		tr.Props[tn] = props
	}
	return typ
}

// Type finds a type based on its name (package path + "." + type name) --
// returns nil if not found
func (tr *TypeRegistry) Type(typeName string) reflect.Type {
	if typ, ok := tr.Types[typeName]; ok {
		return typ
	}
	return nil
}

// Properties returns properties for this type -- optionally makes props map
// if not already made -- can use this to register properties for types that
// are not registered
func (tr *TypeRegistry) Properties(typeName string, makeNew bool) map[string]interface{} {
	tp, ok := tr.Props[typeName]
	if !ok {
		if !makeNew {
			return nil
		}
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

// AllImplementersOf returns a list of all registered types that implement the given interface type at any level of embedding -- must pass a type constructed like this: reflect.TypeOf((*gi.Node2D)(nil)).Elem()
func (tr *TypeRegistry) AllImplementersOf(iface reflect.Type) []reflect.Type {
	if iface.Kind() != reflect.Interface {
		log.Printf("kit.TypeRegistry AllImplementersOf -- type is not an interface: %v\n", iface)
		return nil
	}
	tl := make([]reflect.Type, 0)
	for _, typ := range tr.Types {
		if EmbeddedTypeImplements(typ, iface) {
			tl = append(tl, typ)
		}
	}
	return tl
}

// AllEmbedsOf returns a list of all registered types that embed (inherit from in C++ terminology) the given type -- inclusive determines whether the type itself is included in list
func (tr *TypeRegistry) AllEmbedsOf(embed reflect.Type, inclusive bool) []reflect.Type {
	tl := make([]reflect.Type, 0)
	for _, typ := range tr.Types {
		if !inclusive && typ == embed {
			continue
		}
		if TypeEmbeds(typ, embed) {
			tl = append(tl, typ)
		}
	}
	return tl
}
