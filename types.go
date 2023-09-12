// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gti

import (
	"fmt"
	"log/slog"
	"reflect"
	"sync/atomic"
)

var (
	// Types records all types (i.e., a type registry)
	// key is long type name: package_url.Type, e.g., goki.dev/gi/v2/gi.Button
	Types = map[string]*Type{}

	// TypeIDCounter is an atomically incremented uint64 used
	// for assigning new [Type.ID] numbers
	TypeIDCounter uint64
)

// TypeByName returns a Type by name (package_url.Type, e.g., goki.dev/gi/v2/gi.Button),
func TypeByName(nm string) *Type {
	tp, ok := Types[nm]
	if !ok {
		return nil
	}
	return tp
}

// TypeByNameTry returns a Type by name (package_url.Type, e.g., goki.dev/gi/v2/gi.Button),
// or error if not found
func TypeByNameTry(nm string) (*Type, error) {
	tp, ok := Types[nm]
	if !ok {
		return nil, fmt.Errorf("type %q not found", nm)
	}
	return tp, nil
}

// TypeByValue returns the [Type] of the given value
func TypeByValue(val any) *Type {
	return TypeByName(TypeNameObj(val))
}

// TypeByValueTry returns the [Type] of the given value,
// or an error if it is not found
func TypeByValueTry(val any) (*Type, error) {
	return TypeByNameTry(TypeNameObj(val))
}

// AddType adds a constructed [Type] to the registry
// and returns it. This sets the ID.
func AddType(typ *Type) *Type {
	if _, has := Types[typ.Name]; has {
		slog.Debug("gti.AddType: Type already exists", "Type.Name", typ.Name)
		return typ
	}
	typ.ID = atomic.AddUint64(&TypeIDCounter, 1)
	Types[typ.Name] = typ
	return typ
}

// example constructor:
// var TypeMyType = gti.AddType(&gti.Type{
// 	Name: "goki.dev/ki/v2.MyType",
// 	Comment: `my type is awesome`,
// 	Directives: gti.Directives{},
// 	Methods: ordmap.Make(...),
// 	Embeds: ordmap.Make(...),
// 	Fields: ordmap.Make(...),
// 	// optional instance
// 	Instance: &MyType{},
// })

// TypeName returns the long, full package-path qualified type name.
// This is guaranteed to be unique and used for the Types registry.
func TypeName(typ reflect.Type) string {
	return typ.PkgPath() + "." + typ.Name()
}

// TypeNameObj returns the long, full package-path qualified type name
// from given object.  Automatically finds the non-pointer base type.
// This is guaranteed to be unique and used for the Types registry.
func TypeNameObj(v any) string {
	typ := reflect.TypeOf(v)
	for typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}
	return TypeName(typ)
}

// ShortTypeName returns the short version of a package-qualified type name
// which just has the last element of the path.  This is what is used in
// standard Go programming, and is is used for the key to lookup reflect.Type
// names -- i.e., this is what you should save in a JSON file.
// The potential naming conflict is worth the brevity, and typically a given
// file will only contain mutually-compatible, non-conflicting types.
// This is cached in ShortNames because the path.Base computation is apparently
// a bit slow.
// func ShortTypeName(typ reflect.Type) string {
// 	return path.Base(typ.PkgPath()) + "." + typ.Name()
// }

/*
// TypeFor returns the [Type] for the given
// type. It returns nil if the type is not found
// in the [TypeRegistry].
func TypeFor[T any]() *Type {
	for _, typ := range TypeRegistry {
		if _, ok := typ.Instance.(T); ok {
			return typ
		}
	}
	return nil
}
*/
