// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package types

import (
	"fmt"
	"reflect"
	"strings"
)

// Type represents a type.
type Type struct {
	// Name is the fully package-path-qualified name of the type
	// (eg: cogentcore.org/core/core.Button).
	Name string

	// IDName is the short, package-unqualified, kebab-case name of
	// the type that is suitable for use in an ID (eg: button).
	IDName string

	// Doc has all of the comment documentation
	// info as one string with directives removed.
	Doc string

	// Directives has the parsed comment directives.
	Directives []Directive

	// Methods of the type, which are available for all types.
	Methods []Method

	// Embedded fields of struct types.
	Embeds []Field

	// Fields of struct types.
	Fields []Field

	// Instance is an instance of a non-nil pointer to the type,
	// which is set by [For] and other external functions such that
	// a [Type] can be used to make new instances of the type by
	// reflection. It is not set by typegen.
	Instance any

	// ID is the unique type ID number set by [AddType].
	ID uint64
}

func (tp *Type) String() string {
	return tp.Name
}

// ShortName returns the short name of the type (package.Type)
func (tp *Type) ShortName() string {
	li := strings.LastIndex(tp.Name, "/")
	return tp.Name[li+1:]
}

func (tp *Type) Label() string {
	return tp.ShortName()
}

// ReflectType returns the [reflect.Type] for this type, using [Type.Instance].
func (tp *Type) ReflectType() reflect.Type {
	if tp.Instance == nil {
		return nil
	}
	return reflect.TypeOf(tp.Instance).Elem()
}

// StructGoString creates a GoString for the given struct,
// omitting any zero values.
func StructGoString(str any) string {
	s := reflect.ValueOf(str)
	typ := s.Type()
	strs := []string{}
	for i := 0; i < s.NumField(); i++ {
		f := s.Field(i)
		if f.IsZero() {
			continue
		}
		nm := typ.Field(i).Name
		strs = append(strs, fmt.Sprintf("%s: %#v", nm, f))

	}
	return "{" + strings.Join(strs, ", ") + "}"
}

func (tp Type) GoString() string { return StructGoString(tp) }
