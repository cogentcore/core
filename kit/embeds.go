// Copyright (c) 2018, Randall C. O'Reilly. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package kit

// github.com/rcoreilly/goki/ki/kit

import (
	"log"
	"reflect"
)

// This file contains helpful functions for dealing with embedded structs, in
// the reflect system, including basic functions for guaranteed access to
// pointer and non-pointer types and values

// the non-pointer underlying type
func NonPtrType(typ reflect.Type) reflect.Type {
	for typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}
	return typ
}

// the pointer to underlying type
func PtrType(typ reflect.Type) reflect.Type {
	for typ.Kind() != reflect.Ptr {
		typ = reflect.PtrTo(typ)
	}
	return typ
}

// the non-pointer underlying value
func NonPtrValue(v reflect.Value) reflect.Value {
	for v.Kind() == reflect.Ptr { // todo: could also look for slice here
		v = v.Elem()
	}
	return v
}

// the pointer version (Addr()) of the underlying value
func PtrValue(v reflect.Value) reflect.Value {
	for v.Kind() != reflect.Ptr { // todo: could also look for slice here
		v = v.Addr()
	}
	return v
}

// the non-pointer value of an interface
func NonPtrInterface(el interface{}) interface{} {
	v := reflect.ValueOf(el)
	for v.Kind() == reflect.Ptr { // todo: could also look for slice here
		v = v.Elem()
	}
	return v.Interface()
}

// call a function on all the the primary fields of a given struct type,
// including those on anonymous embedded structs that this struct has, passing
// the current (embedded) type and StructField -- effectively flattens the
// reflect field list -- if fun returns false then iteration stops -- overall
// rval is false if iteration was stopped or there was an error (logged), true
// otherwise
func FlatFieldsTypeFun(typ reflect.Type, fun func(typ reflect.Type, field reflect.StructField) bool) bool {
	typ = NonPtrType(typ)
	if typ.Kind() != reflect.Struct {
		log.Printf("kit.FlatFieldsTypeFun: Must call on a struct type, not: %v\n", typ)
		return false
	}
	rval := true
	for i := 0; i < typ.NumField(); i++ {
		f := typ.Field(i)
		if f.Type.Kind() == reflect.Struct && f.Anonymous {
			rval = FlatFieldsTypeFun(f.Type, fun) // no err here
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

// call a function on all the the primary fields of a given struct value (must
// pass a pointer to the struct) including those on anonymous embedded structs
// that this struct has, passing the current (embedded) type and StructField
// -- effectively flattens the reflect field list
func FlatFieldsValueFun(stru interface{}, fun func(stru interface{}, typ reflect.Type, field reflect.StructField, fieldVal reflect.Value) bool) bool {
	vv := reflect.ValueOf(stru)
	if stru == nil || vv.Kind() != reflect.Ptr {
		log.Printf("kit.FlatFieldsValueFun: must pass a non-nil pointer to the struct: %v\n", stru)
		return false
	}
	v := NonPtrValue(vv)
	typ := v.Type()
	rval := true
	for i := 0; i < typ.NumField(); i++ {
		f := typ.Field(i)
		vf := v.Field(i)
		if !vf.CanInterface() {
			continue
		}
		vfi := vf.Interface()
		if vfi == nil || vfi == stru {
			continue
		}
		if f.Type.Kind() == reflect.Struct && f.Anonymous {
			// key to take addr here so next level is addressable
			rval = FlatFieldsValueFun(PtrValue(vf).Interface(), fun)
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

// a slice list of all the StructField type information for fields of given type and any embedded types -- returns nil on error (logged)
func FlatFields(typ reflect.Type) []reflect.StructField {
	ff := make([]reflect.StructField, 0)
	falseErr := FlatFieldsTypeFun(typ, func(typ reflect.Type, field reflect.StructField) bool {
		ff = append(ff, field)
		return true
	})
	if falseErr == false {
		return nil
	}
	return ff
}

// a slice list of all the field reflect.Value's for fields of given struct (must pass a pointer to the struct) and any of its embedded structs -- returns nil on error (logged)
func FlatFieldVals(stru interface{}) []reflect.Value {
	ff := make([]reflect.Value, 0)
	falseErr := FlatFieldsValueFun(stru, func(stru interface{}, typ reflect.Type, field reflect.StructField, fieldVal reflect.Value) bool {
		ff = append(ff, fieldVal)
		return true
	})
	if falseErr == false {
		return nil
	}
	return ff
}

// a slice list of all the field interface{} values *as pointers to the field value* (i.e., calling Addr() on the Field Value) for fields of given struct (must pass a pointer to the struct) and any of its embedded structs -- returns nil on error (logged)
func FlatFieldInterfaces(stru interface{}) []interface{} {
	ff := make([]interface{}, 0)
	falseErr := FlatFieldsValueFun(stru, func(stru interface{}, typ reflect.Type, field reflect.StructField, fieldVal reflect.Value) bool {
		ff = append(ff, PtrValue(fieldVal).Interface())
		return true
	})
	if falseErr == false {
		return nil
	}
	return ff
}

// find field in type or embedded structs within type, by name -- native function already does flat version, so this is just for reference and consistency
func FlatFieldByName(typ reflect.Type, nm string) (reflect.StructField, bool) {
	return typ.FieldByName(nm)
}

// find field in object and embedded objects, by name, returning reflect.Value of field -- native version of Value function already does flat find, so this just provides a convenient wrapper
func FlatFieldValueByName(stru interface{}, nm string) reflect.Value {
	vv := reflect.ValueOf(stru)
	if stru == nil || vv.Kind() != reflect.Ptr {
		log.Printf("kit.FlatFieldsValueFun: must pass a non-nil pointer to the struct: %v\n", stru)
		return reflect.Value{}
	}
	v := NonPtrValue(vv)
	return v.FieldByName(nm)
}

// find field in object and embedded objects, by name, returning interface{} to pointer of field, or nil if not found
func FlatFieldInterfaceByName(stru interface{}, nm string) interface{} {
	ff := FlatFieldValueByName(stru, nm)
	if !ff.IsValid() {
		return nil
	}
	return PtrValue(ff).Interface()
}

// checks if given type embeds another type, at any level of recursive embedding
func TypeEmbeds(typ, embed reflect.Type) bool {
	typ = NonPtrType(typ)
	embed = NonPtrType(embed)
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

// return the embedded struct of given type within given struct
func EmbededStruct(stru interface{}, embed reflect.Type) interface{} {
	v := NonPtrValue(reflect.ValueOf(stru))
	typ := v.Type()
	for i := 0; i < typ.NumField(); i++ {
		f := typ.Field(i)
		if f.Type.Kind() == reflect.Struct && f.Anonymous { // anon only avail on StructField fm typ
			vf := v.Field(i)
			vfpi := PtrValue(vf).Interface()
			if f.Type == embed {
				return vfpi
			}
			rv := EmbededStruct(vfpi, embed)
			if rv != nil {
				return rv
			}
		}
	}
	return nil
}

// checks if given type implements given interface, or it embeds a type that does so
func EmbeddedTypeImplements(typ, iface reflect.Type) bool {
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
