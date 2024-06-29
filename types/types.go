// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package types provides type information for Go types, methods,
// and functions.
package types

import (
	"cmp"
	"reflect"
	"slices"
	"strings"
	"sync/atomic"

	"cogentcore.org/core/base/reflectx"
)

var (
	// Types is a type registry, initialized to contain all builtin types. New types
	// can be added with [AddType]. The key is the long type name: package/path.Type,
	// e.g., cogentcore.org/core/core.Button.
	Types = map[string]*Type{}

	// typeIDCounter is an atomically incremented uint64 used
	// for assigning new [Type.ID] numbers.
	typeIDCounter uint64
)

func init() {
	addBuiltin[bool]("bool")
	addBuiltin[complex64]("complex64")
	addBuiltin[complex128]("complex128")
	addBuiltin[float32]("float32")
	addBuiltin[float64]("float64")
	addBuiltin[int]("int")
	addBuiltin[int64]("int8")
	addBuiltin[int16]("int16")
	addBuiltin[int32]("int32")
	addBuiltin[int64]("int64")
	addBuiltin[string]("string")
	addBuiltin[uint]("uint")
	addBuiltin[uint8]("uint8")
	addBuiltin[uint16]("uint16")
	addBuiltin[uint32]("uint32")
	addBuiltin[uint64]("uint64")
	addBuiltin[uint64]("uintptr")
}

// addBuiltin adds the given builtin type with the given name to the type registry.
func addBuiltin[T any](name string) {
	var v T
	AddType(&Type{Name: name, IDName: name, Instance: v})
}

// TypeByName returns a Type by name (package/path.Type, e.g., cogentcore.org/core/core.Button),
func TypeByName(name string) *Type {
	return Types[name]
}

// TypeByValue returns the [Type] of the given value
func TypeByValue(v any) *Type {
	return TypeByName(TypeNameValue(v))
}

// TypeByReflectType returns the [Type] of the given reflect type
func TypeByReflectType(typ reflect.Type) *Type {
	return TypeByName(TypeName(typ))
}

// For returns the [Type] of the generic type parameter,
// setting its [Type.Instance] to a new(T) if it is nil.
func For[T any]() *Type {
	var v T
	t := TypeByValue(v)
	if t != nil && t.Instance == nil {
		t.Instance = new(T)
	}
	return t
}

// AddType adds a constructed [Type] to the registry
// and returns it. This sets the ID.
func AddType(typ *Type) *Type {
	typ.ID = atomic.AddUint64(&typeIDCounter, 1)
	Types[typ.Name] = typ
	return typ
}

// TypeName returns the long, full package-path qualified type name.
// This is guaranteed to be unique and used for the Types registry.
func TypeName(typ reflect.Type) string {
	return reflectx.LongTypeName(typ)
}

// TypeNameValue returns the long, full package-path qualified type name
// of the given Go value. Automatically finds the non-pointer base type.
// This is guaranteed to be unique and used for the Types registry.
func TypeNameValue(v any) string {
	typ := reflectx.Underlying(reflect.ValueOf(v)).Type()
	return TypeName(typ)
}

// BuiltinTypes returns all of the builtin types in the type registry.
func BuiltinTypes() []*Type {
	res := []*Type{}
	for _, t := range Types {
		if !strings.Contains(t.Name, ".") {
			res = append(res, t)
		}
	}
	slices.SortFunc(res, func(a, b *Type) int {
		return cmp.Compare(a.Name, b.Name)
	})
	return res
}

// GetDoc gets the documentation for the given value with the given parent struct, field, and label.
// The value, parent value, and field may be nil/invalid. GetDoc uses the given label to format
// the documentation with [FormatDoc] before returning it.
func GetDoc(value, parent reflect.Value, field reflect.StructField, label string) (string, bool) {
	// if we are not part of a struct, we just get the documentation for our type
	if !parent.IsValid() {
		if !value.IsValid() {
			return "", false
		}
		rtyp := reflectx.NonPointerType(value.Type())
		typ := TypeByName(TypeName(rtyp))
		if typ == nil {
			return "", false
		}
		return FormatDoc(typ.Doc, rtyp.Name(), label), true
	}

	// otherwise, we get our field documentation in our parent
	f := GetField(parent, field.Name)
	if f != nil {
		return FormatDoc(f.Doc, field.Name, label), true
	}
	// if we aren't in the type registry, we fall back on struct tag
	doc, ok := field.Tag.Lookup("doc")
	if !ok {
		return "", false
	}
	return FormatDoc(doc, field.Name, label), true
}

// FormatDoc formats the given Go documentation string for an identifier with the given
// CamelCase name and intended label. It replaces the name with the label and cleans
// up trailing punctuation.
func FormatDoc(doc, name, label string) string {
	doc = strings.ReplaceAll(doc, name, label)

	// if we only have one period, get rid of it if it is at the end
	if strings.Count(doc, ".") == 1 {
		doc = strings.TrimSuffix(doc, ".")
	}
	return doc
}
