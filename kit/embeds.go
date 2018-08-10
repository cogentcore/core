// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package kit

import (
	"log"
	"reflect"
)

// This file contains helpful functions for dealing with embedded structs, in
// the reflect system

// FlatFieldsTypeFunc calls a function on all the primary fields of a given
// struct type, including those on anonymous embedded structs that this struct
// has, passing the current (embedded) type and StructField -- effectively
// flattens the reflect field list -- if fun returns false then iteration
// stops -- overall rval is false if iteration was stopped or there was an
// error (logged), true otherwise
func FlatFieldsTypeFunc(typ reflect.Type, fun func(typ reflect.Type, field reflect.StructField) bool) bool {
	typ = NonPtrType(typ)
	if typ.Kind() != reflect.Struct {
		log.Printf("kit.FlatFieldsTypeFunc: Must call on a struct type, not: %v\n", typ)
		return false
	}
	rval := true
	for i := 0; i < typ.NumField(); i++ {
		f := typ.Field(i)
		if f.Type.Kind() == reflect.Struct && f.Anonymous {
			rval = FlatFieldsTypeFunc(f.Type, fun) // no err here
			if !rval {
				break
			}
		} else {
			rval = fun(typ, f)
			if !rval {
				break
			}
		}
	}
	return rval
}

// AllFieldsTypeFunc calls a function on all the fields of a given struct type,
// including those on *any* embedded structs that this struct has -- if fun
// returns false then iteration stops -- overall rval is false if iteration
// was stopped or there was an error (logged), true otherwise.
func AllFieldsTypeFunc(typ reflect.Type, fun func(typ reflect.Type, field reflect.StructField) bool) bool {
	typ = NonPtrType(typ)
	if typ.Kind() != reflect.Struct {
		log.Printf("kit.AllFieldsTypeFunc: Must call on a struct type, not: %v\n", typ)
		return false
	}
	rval := true
	for i := 0; i < typ.NumField(); i++ {
		f := typ.Field(i)
		if f.Type.Kind() == reflect.Struct {
			rval = AllFieldsTypeFunc(f.Type, fun) // no err here
			if !rval {
				break
			}
		} else {
			rval = fun(typ, f)
			if !rval {
				break
			}
		}
	}
	return rval
}

// FlatFieldsValueFunc calls a function on all the primary fields of a
// given struct value (must pass a pointer to the struct) including those on
// anonymous embedded structs that this struct has, passing the current
// (embedded) type and StructField -- effectively flattens the reflect field
// list
func FlatFieldsValueFunc(stru interface{}, fun func(stru interface{}, typ reflect.Type, field reflect.StructField, fieldVal reflect.Value) bool) bool {
	vv := reflect.ValueOf(stru)
	if stru == nil || vv.Kind() != reflect.Ptr {
		log.Printf("kit.FlatFieldsValueFunc: must pass a non-nil pointer to the struct: %v\n", stru)
		return false
	}
	v := NonPtrValue(vv)
	if !v.IsValid() {
		return true
	}
	typ := v.Type()
	if typ.Kind() != reflect.Struct {
		log.Printf("kit.FlatFieldsValueFunc: non-pointer type is not a struct: %v\n", typ.String())
		return false
	}
	rval := true
	for i := 0; i < typ.NumField(); i++ {
		f := typ.Field(i)
		vf := v.Field(i)
		if !vf.CanInterface() {
			continue
		}
		vfi := vf.Interface()
		if vfi == stru {
			continue
		}
		if f.Type.Kind() == reflect.Struct && f.Anonymous {
			// key to take addr here so next level is addressable
			rval = FlatFieldsValueFunc(PtrValue(vf).Interface(), fun)
			if !rval {
				break
			}
		} else {
			rval = fun(vfi, typ, f, vf)
			if !rval {
				break
			}
		}
	}
	return rval
}

// FlatFields returns a slice list of all the StructField type information for
// fields of given type and any embedded types -- returns nil on error
// (logged)
func FlatFields(typ reflect.Type) []reflect.StructField {
	ff := make([]reflect.StructField, 0)
	falseErr := FlatFieldsTypeFunc(typ, func(typ reflect.Type, field reflect.StructField) bool {
		ff = append(ff, field)
		return true
	})
	if falseErr == false {
		return nil
	}
	return ff
}

// AllFields returns a slice list of all the StructField type information for
// all elemental fields of given type and all embedded types -- returns nil on
// error (logged)
func AllFields(typ reflect.Type) []reflect.StructField {
	ff := make([]reflect.StructField, 0)
	falseErr := AllFieldsTypeFunc(typ, func(typ reflect.Type, field reflect.StructField) bool {
		ff = append(ff, field)
		return true
	})
	if falseErr == false {
		return nil
	}
	return ff
}

// AllFieldsN returns number of elemental fields in given type
func AllFieldsN(typ reflect.Type) int {
	n := 0
	falseErr := AllFieldsTypeFunc(typ, func(typ reflect.Type, field reflect.StructField) bool {
		n++
		return true
	})
	if falseErr == false {
		return 0
	}
	return n
}

// FlatFieldsVals returns a slice list of all the field reflect.Value's for
// fields of given struct (must pass a pointer to the struct) and any of its
// embedded structs -- returns nil on error (logged)
func FlatFieldVals(stru interface{}) []reflect.Value {
	ff := make([]reflect.Value, 0)
	falseErr := FlatFieldsValueFunc(stru, func(stru interface{}, typ reflect.Type, field reflect.StructField, fieldVal reflect.Value) bool {
		ff = append(ff, fieldVal)
		return true
	})
	if falseErr == false {
		return nil
	}
	return ff
}

// FlatFieldInterfaces returns a slice list of all the field interface{}
// values *as pointers to the field value* (i.e., calling Addr() on the Field
// Value) for fields of given struct (must pass a pointer to the struct) and
// any of its embedded structs -- returns nil on error (logged)
func FlatFieldInterfaces(stru interface{}) []interface{} {
	ff := make([]interface{}, 0)
	falseErr := FlatFieldsValueFunc(stru, func(stru interface{}, typ reflect.Type, field reflect.StructField, fieldVal reflect.Value) bool {
		ff = append(ff, PtrValue(fieldVal).Interface())
		return true
	})
	if falseErr == false {
		return nil
	}
	return ff
}

// FlatFieldByName returns field in type or embedded structs within type, by
// name -- native function already does flat version, so this is just for
// reference and consistency
func FlatFieldByName(typ reflect.Type, nm string) (reflect.StructField, bool) {
	return typ.FieldByName(nm)
}

// FlatFieldTag returns given tag value in field in type or embedded structs
// within type, by name -- empty string if not set or field not found
func FlatFieldTag(typ reflect.Type, nm, tag string) string {
	fld, ok := typ.FieldByName(nm)
	if !ok {
		return ""
	}
	return fld.Tag.Get(tag)
}

// FlatFieldValueByName finds field in object and embedded objects, by name,
// returning reflect.Value of field -- native version of Value function
// already does flat find, so this just provides a convenient wrapper
func FlatFieldValueByName(stru interface{}, nm string) reflect.Value {
	vv := reflect.ValueOf(stru)
	if stru == nil || vv.Kind() != reflect.Ptr {
		log.Printf("kit.FlatFieldsValueFunc: must pass a non-nil pointer to the struct: %v\n", stru)
		return reflect.Value{}
	}
	v := NonPtrValue(vv)
	return v.FieldByName(nm)
}

// FlatFieldInterfaceByName finds field in object and embedded objects, by
// name, returning interface{} to pointer of field, or nil if not found
func FlatFieldInterfaceByName(stru interface{}, nm string) interface{} {
	ff := FlatFieldValueByName(stru, nm)
	if !ff.IsValid() {
		return nil
	}
	return PtrValue(ff).Interface()
}

// TypeEmbeds checks if given type embeds another type, at any level of
// recursive embedding (including being the type itself)
func TypeEmbeds(typ, embed reflect.Type) bool {
	typ = NonPtrType(typ)
	embed = NonPtrType(embed)
	if typ == embed {
		return true
	}
	for i := 0; i < typ.NumField(); i++ {
		f := typ.Field(i)
		if f.Type.Kind() == reflect.Struct && f.Anonymous {
			// fmt.Printf("typ %v anon struct %v\n", typ.Name(), f.Name)
			if f.Type == embed {
				return true
			}
			return TypeEmbeds(f.Type, embed)
		}
	}
	return false
}

// Embed returns the embedded struct of given type within given struct
func Embed(stru interface{}, embed reflect.Type) interface{} {
	if IfaceIsNil(stru) {
		return nil
	}
	v := NonPtrValue(reflect.ValueOf(stru))
	typ := v.Type()
	if typ == embed {
		return PtrValue(v).Interface()
	}
	for i := 0; i < typ.NumField(); i++ {
		f := typ.Field(i)
		if f.Type.Kind() == reflect.Struct && f.Anonymous { // anon only avail on StructField fm typ
			vf := v.Field(i)
			vfpi := PtrValue(vf).Interface()
			if f.Type == embed {
				return vfpi
			}
			rv := Embed(vfpi, embed)
			if rv != nil {
				return rv
			}
		}
	}
	return nil
}

// EmbeddedTypeImplements checks if given type implements given interface, or
// it embeds a type that does so -- must pass a type constructed like this:
// reflect.TypeOf((*gi.Node2D)(nil)).Elem()
func EmbeddedTypeImplements(typ, iface reflect.Type) bool {
	if iface.Kind() != reflect.Interface {
		log.Printf("kit.TypeRegistry EmbeddedTypeImplements -- type is not an interface: %v\n", iface)
		return false
	}
	if typ.Implements(iface) {
		return true
	}
	if reflect.PtrTo(typ).Implements(iface) { // typically need the pointer type to impl
		return true
	}
	typ = NonPtrType(typ)
	if typ.Implements(iface) { // try it all possible ways..
		return true
	}
	if typ.Kind() != reflect.Struct {
		return false
	}
	for i := 0; i < typ.NumField(); i++ {
		f := typ.Field(i)
		if f.Type.Kind() == reflect.Struct && f.Anonymous {
			rv := EmbeddedTypeImplements(f.Type, iface)
			if rv {
				return true
			}
		}
	}
	return false
}
