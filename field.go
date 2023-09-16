// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package grease

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/iancoleman/strcase"
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
	addFieldsImpl(obj, "", false, allFields, map[string]*Field{}, cmd)
}

// addFieldsImpl is the underlying implementation of [AddFields].
// usedNames is a map keyed by used kebab-case names with values
// of their associated fields, used to track naming conflicts.
func addFieldsImpl(obj any, path string, nest bool, allFields *Fields, usedNames map[string]*Field, cmd string) {
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
			addFieldsImpl(laser.PtrValue(fv).Interface(), nwPath, nwNest, allFields, usedNames, cmd)
			continue
		}
		name := f.Name
		if path != "" {
			name = path + "." + name
		}
		names := []string{strcase.ToKebab(f.Name)}
		greasetag, ok := f.Tag.Lookup("grease")
		if ok {
			names = strings.Split(greasetag, ",")
			if len(names) == 0 {
				fmt.Println("warning: programmer error: expected at least one name in grease struct tag, but got none")
			}
		}

		nf := &Field{
			Field: f,
			Value: pval,
			Name:  name,
			Names: names,
			Nest:  nest,
		}
		for i, name := range names {
			if of, has := usedNames[name]; has {
				// nest field tag values for new and other
				nfns := nf.Field.Tag.Get("nest")
				ofns := of.Field.Tag.Get("nest")

				// whether new and other get to have non-nested version
				nfn := nfns == "-" || nfns == "false"
				ofn := ofns == "-" || ofns == "false"

				if nfn && ofn {
					fmt.Printf("warning: programmer error: %q specified on two config fields (%q and %q) with the same name (%q); keep %q on the field you want to be able to access without nesting (eg: -target instead of -build-target) and remove it from the other one", `nest:"-"`, of.Name, nf.Name, name, `nest:"-"`)
				} else if !nfn && !ofn {
					// neither one gets it, so we replace both with fully qualified name
					names[i] = strcase.ToKebab(nf.Name)
					for i, on := range of.Names {
						if on == name {
							of.Names[i] = strcase.ToKebab(of.Name)
						}
					}
				} else if nfn && !ofn {
					// we get it, so we keep ours as is and replace them with fully qualified name
					for i, on := range of.Names {
						if on == name {
							of.Names[i] = strcase.ToKebab(of.Name)
						}
					}
					// we also need to update the comparison point to us
					usedNames[name] = nf
				} else if !nfn && ofn {
					// they get it, so we replace ours with fully qualified name
					names[i] = strcase.ToKebab(nf.Name)
				}
			} else {
				// if no conflict, we have it
				usedNames[name] = nf
			}
		}
		allFields.Add(name, nf)
	}
}
