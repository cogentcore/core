// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package grease

import (
	"fmt"
	"reflect"
	"strings"

	"goki.dev/laser"
	"goki.dev/ordmap"
)

// Field represents a struct field in a configuration object.
// It is passed around in flag parsing functions, but it should
// not typically be used by end-user code going through the
// standard Run/Config/SetFromArgs API.
type Field struct {
	// Field is the reflect struct field object for this field
	Field reflect.StructField
	// Value is the reflect value of the settable pointer to this field
	Value reflect.Value
	// Name is the fully qualified, nested name of this field (eg: A.B.C).
	// It is as it appears in code, and is NOT transformed something like kebab-case.
	Name string
	// Names contains all of the possible end-user names for this field as a flag.
	// It defaults to the name of the field, but custom names can be specified via
	// the grease struct tag.
	Names []string
	// Nest is whether, if true, a nested version of this field should be the only
	// way to access it (eg: A.B.C), or, if false, this field should be accessible
	// through its non-nested version (eg: C).
	Nest bool
}

// Fields is a simple type alias for an ordered map of [Field] objects.
type Fields = ordmap.Map[string, *Field]

// AddFields adds to the given fields map all of the fields of the given
// object, in the context of the given command name.
func AddFields(obj any, allFields *Fields, cmd string) {
	addFieldsImpl(obj, "", false, allFields, cmd)
}

// addFieldsImpl is the underlying implementation of [AddFields].
func addFieldsImpl(obj any, path string, nest bool, allFields *Fields, cmd string) {
	if laser.AnyIsNil(obj) {
		return
	}
	ov := reflect.ValueOf(obj)
	if ov.Kind() == reflect.Pointer && ov.IsNil() {
		return
	}
	val := laser.NonPtrValue(ov)
	typ := val.Type()
	for i := 0; i < typ.NumField(); i++ {
		f := typ.Field(i)
		fv := val.Field(i)
		pval := laser.PtrValue(fv)
		cmdtag, ok := f.Tag.Lookup("cmd")
		if ok && cmdtag != cmd { // if we are associated with a different command, skip
			continue
		}
		if laser.NonPtrType(f.Type).Kind() == reflect.Struct {
			nwPath := f.Name
			if path != "" {
				nwPath = path + "." + nwPath
			}
			nwNest := nest
			if !nwNest {
				neststr, ok := f.Tag.Lookup("nest")
				if ok && (neststr == "+" || neststr == "true") {
					nwNest = true
				}
			}
			addFieldsImpl(laser.PtrValue(fv).Interface(), nwPath, nwNest, allFields, cmd)
			continue
		}
		name := f.Name
		if path != "" {
			name = path + "." + name
		}
		names := []string{f.Name}
		greasetag, ok := f.Tag.Lookup("grease")
		if ok {
			names = strings.Split(greasetag, ",")
			if len(names) == 0 {
				fmt.Println("warning: programmer error: expected at least one name in grease struct tag, but got none")
			}
		}
		allFields.Add(name, &Field{
			Field: f,
			Value: pval,
			Name:  name,
			Names: names,
			Nest:  nest,
		})

	}
}
