// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gti

import (
	"fmt"
	"log/slog"
	"reflect"
	"sort"
	"sync/atomic"

	"goki.dev/glop/sentence"
	"goki.dev/laser"
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

// TypeByReflectType returns the [Type] of the given reflect type
func TypeByReflectType(typ reflect.Type) *Type {
	return TypeByName(TypeName(typ))
}

// TypeByReflectTypeTry returns the [Type] of the given reflect type,
// or an error if it is not found
func TypeByReflectTypeTry(typ reflect.Type) (*Type, error) {
	return TypeByNameTry(TypeName(typ))
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

// TypeName returns the long, full package-path qualified type name.
// This is guaranteed to be unique and used for the Types registry.
func TypeName(typ reflect.Type) string {
	return laser.LongTypeName(typ)
}

// TypeNameObj returns the long, full package-path qualified type name
// from given object.  Automatically finds the non-pointer base type.
// This is guaranteed to be unique and used for the Types registry.
func TypeNameObj(v any) string {
	typ := laser.NonPtrType(reflect.TypeOf(v))
	return TypeName(typ)
}

// AllEmbeddersOf returns all registered types that embed the given type.
// List is sorted in alpha order by fully package-path-qualified Name.
func AllEmbeddersOf(typ *Type) []*Type {
	var typs []*Type
	for _, t := range Types {
		if t.HasEmbed(typ) {
			typs = append(typs, t)
		}
	}
	sort.Slice(typs, func(i, j int) bool {
		return typs[i].Name < typs[j].Name
	})
	return typs
}

// GetDoc gets the documentation for the given value with the given owner value, field, and label.
// The value, owner value, and field may be nil/invalid. The owner value, if valid, is the value that
// contains the value (the parent struct, map, slice, or array). The field, if non-nil,
// is the struct field that the value represents. GetDoc uses the given label to format
// the documentation with [sentence.Doc] before returning it.
func GetDoc(val, owner reflect.Value, field *reflect.StructField, label string) (string, bool) {
	// if we are not part of a struct, we just get the documentation for our type
	if field == nil || !owner.IsValid() {
		if !val.IsValid() {
			return "", false
		}
		rtyp := laser.NonPtrType(val.Type())
		typ := TypeByName(TypeName(rtyp))
		if typ == nil {
			return "", false
		}
		return sentence.Doc(typ.Doc, rtyp.Name(), label), true
	}

	// otherwise, we get our field documentation in our parent
	otyp := TypeByReflectType(owner.Type())
	if otyp != nil {
		f := GetField(otyp, field.Name)
		if f == nil {
			return "", false
		}
		return sentence.Doc(f.Doc, field.Name, label), true
	}
	// if we aren't in gti, we fall back on struct tag
	desc, ok := field.Tag.Lookup("desc")
	if !ok {
		return "", false
	}
	return sentence.Doc(desc, field.Name, label), true
}
