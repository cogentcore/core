// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package grease

import (
	"fmt"
	"os"
	"reflect"
	"strings"

	"slices"

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
}

// Fields is a simple type alias for an ordered map of [Field] objects.
type Fields = ordmap.Map[string, *Field]

// AddAllFields, when passed as the command to [AddFields], indicates
// to add all fields, regardless of their command association.
const AddAllFields = "*"

// AddFields adds to the given fields map all of the fields of the given
// object, in the context of the given command name. A value of [AddAllFields]
// for cmd indicates to add all fields, regardless of their command association.
func AddFields(obj any, allFields *Fields, cmd string) {
	addFieldsImpl(obj, "", allFields, map[string]*Field{}, cmd)
}

// addFieldsImpl is the underlying implementation of [AddFields].
// usedNames is a map keyed by used kebab-case names with values
// of their associated fields, used to track naming conflicts.
func addFieldsImpl(obj any, path string, allFields *Fields, usedNames map[string]*Field, cmd string) {
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
		if ok && cmdtag != cmd && cmd != AddAllFields { // if we are associated with a different command, skip
			continue
		}
		if laser.NonPtrType(f.Type).Kind() == reflect.Struct {
			nwPath := f.Name
			if path != "" {
				nwPath = path + "." + nwPath
			}
			addFieldsImpl(laser.PtrValue(fv).Interface(), nwPath, allFields, usedNames, cmd)
			continue
		}
		// we add both unqualified and fully-qualified names
		name := f.Name
		names := []string{name}
		if path != "" {
			name = path + "." + name
			names = append(names, name)
		}

		greasetag, ok := f.Tag.Lookup("grease")
		if ok {
			names = strings.Split(greasetag, ",")
			if len(names) == 0 {
				fmt.Println(ErrorColor("warning: programmer error:") + " expected at least one name in grease struct tag, but got none")
			}
		}

		nf := &Field{
			Field: f,
			Value: pval,
			Name:  name,
			Names: names,
		}
		for i, name := range names {
			name := strcase.ToCamel(name)        // everybody is in camel for naming conflict check
			if of, has := usedNames[name]; has { // we have a conflict

				// if we have a naming conflict between two fields with the same base
				// (in the same parent struct), then there is no nesting and they have
				// been directly given conflicting names, so there is a simple programmer error
				nbase := ""
				nli := strings.LastIndex(nf.Name, ".")
				if nli >= 0 {
					nbase = nf.Name[:nli]
				}
				obase := ""
				oli := strings.LastIndex(of.Name, ".")
				if oli >= 0 {
					obase = of.Name[:oli]
				}
				if nbase == obase {
					fmt.Printf(ErrorColor("programmer error:")+" fields %q and %q were both assigned the same name (%q)\n", of.Name, nf.Name, name)
					os.Exit(1)
				}

				// if that isn't the case, they are in different parent structs and
				// it is a nesting problem, so we use the nest tags to resolve the conflict.
				// the basic rule is that whoever specifies the nest:"-" tag gets to
				// be non-nested, and if no one specifies it, everyone is nested.
				// if both want to be non-nested, that is a programmer error.

				// nest field tag values for new and other
				nfns := nf.Field.Tag.Get("nest")
				ofns := of.Field.Tag.Get("nest")

				// whether new and other get to have non-nested version
				nfn := nfns == "-" || nfns == "false"
				ofn := ofns == "-" || ofns == "false"

				if nfn && ofn {
					fmt.Printf(ErrorColor("programmer error:")+" %s specified on two config fields (%q and %q) with the same name (%q); keep %s on the field you want to be able to access without nesting (eg: with %q instead of %q) and remove it from the other one\n", CmdColor(`nest:"-"`), of.Name, nf.Name, name, CmdColor(`nest:"-"`), "-"+name, "-"+strcase.ToKebab(nf.Name))
					os.Exit(1)
				} else if !nfn && !ofn {
					// neither one gets it, so we replace both with fully qualified name
					applyShortestUniqueName(nf, i, usedNames)
					for i, on := range of.Names {
						if on == name {
							applyShortestUniqueName(of, i, usedNames)
						}
					}
				} else if nfn && !ofn {
					// we get it, so we keep ours as is and replace them with fully qualified name
					for i, on := range of.Names {
						if on == name {
							applyShortestUniqueName(of, i, usedNames)
						}
					}
					// we also need to update the field for our name to us
					usedNames[name] = nf
				} else if !nfn && ofn {
					// they get it, so we replace ours with fully qualified name
					applyShortestUniqueName(nf, i, usedNames)
				}
			} else {
				// if no conflict, we get the name
				usedNames[name] = nf
			}
		}
		allFields.Add(name, nf)
	}
}

// applyShortestUniqueName uses [shortestUniqueName] to apply the shortest
// unique name for the given field, in the context of the given
// used names, at the given index.
func applyShortestUniqueName(field *Field, idx int, usedNames map[string]*Field) {
	nm := shortestUniqueName(field.Name, usedNames)
	// if we already have this name, we don't need to add it, so we just delete this entry
	if slices.Contains(field.Names, nm) {
		field.Names = slices.Delete(field.Names, idx, idx+1)
	} else {
		field.Names[idx] = nm
		usedNames[nm] = field
	}
}

// shortestUniqueName returns the shortest unique camel-case name for
// the given fully-qualified nest name of a field, using the given
// map of used names. It works backwards, so, for example, if given "A.B.C.D",
// it would check "D", then "C.D", then "B.C.D", and finally "A.B.C.D".
func shortestUniqueName(name string, usedNames map[string]*Field) string {
	strs := strings.Split(name, ".")
	cur := ""
	for i := len(strs) - 1; i >= 0; i-- {
		if cur == "" {
			cur = strs[i]
		} else {
			cur = strs[i] + "." + cur
		}
		if _, has := usedNames[cur]; !has {
			return cur
		}
	}
	return cur // TODO: this should never happen, but if it does, we might want to print an error
}
