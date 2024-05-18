// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package types

import (
	"fmt"
	"reflect"
	"regexp"
	"strings"
)

// Type represents a type.
type Type struct {
	// Name is the fully package-path-qualified name of the type (eg: cogentcore.org/core/core.Button)
	Name string

	// IDName is the short, package-unqualified, kebab-case name of the type that is suitable
	// for use in an ID (eg: button)
	IDName string

	// Doc has all of the comment documentation
	// info as one string with directives removed.
	Doc string

	// Directives has the parsed comment directives
	Directives []Directive

	// Methods are available for all types
	Methods []Method

	// Embedded fields for struct types
	Embeds []Field

	// Fields for struct types
	Fields []Field

	// Instance is an optional instance of the type
	Instance any

	// ID is the unique type ID number
	ID uint64

	// All embedded fields (including nested ones) for struct types;
	// not set by typegen -- HasEmbed automatically compiles it as needed.
	// Key is the ID of the type.
	AllEmbeds map[uint64]*Type
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

// ReflectType returns the [reflect.Type] for this type, using the Instance
func (tp *Type) ReflectType() reflect.Type {
	if tp.Instance == nil {
		return nil
	}
	return reflect.TypeOf(tp.Instance).Elem()
}

// HasEmbed returns true if this type has the given type
// at any level of embedding depth, including if this type is
// the given type.  The first time called it will Compile
// a map of all embedded types so subsequent calls are very fast.
func (tp *Type) HasEmbed(typ *Type) bool {
	if tp.AllEmbeds == nil {
		tp.CompileEmbeds()
		if tp.AllEmbeds == nil {
			return typ == tp
		}
	}
	if tp == typ {
		return true
	}
	_, has := tp.AllEmbeds[typ.ID]
	return has
}

func (tp *Type) CompileEmbeds() {
	if len(tp.Embeds) == 0 {
		return
	}
	rt := tp.ReflectType()
	if rt == nil {
		return
	}
	tp.AllEmbeds = make(map[uint64]*Type)
	for _, em := range tp.Embeds {
		enm := em.Name
		if idx := strings.LastIndex(enm, "."); idx >= 0 {
			enm = enm[idx+1:]
		}
		etf, has := rt.FieldByName(enm)
		if !has {
			continue
		}
		etft := TypeName(etf.Type)
		et := TypeByName(etft)
		if et == nil {
			continue
		}
		tp.AllEmbeds[et.ID] = et
		et.CompileEmbeds()
		if et.AllEmbeds == nil {
			continue
		}
		for id, ct := range et.AllEmbeds {
			tp.AllEmbeds[id] = ct
		}
	}
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

// need to get rid of quotes around instance
var typeInstanceRegexp = regexp.MustCompile(`Instance: "(.+)"`)

func (tp Type) GoString() string {
	res := StructGoString(tp)
	if tp.Instance == nil {
		return res
	}
	return typeInstanceRegexp.ReplaceAllString(res, "Instance: $1")
}
