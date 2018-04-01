// Copyright (c) 2018, Randall C. O'Reilly. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package kit

// github.com/rcoreilly/goki/ki/kit

import (
	// "fmt"
	// "log"
	"reflect"
)

// This file contains helpful functions for dealing with embedded structs, in
// the reflect system

// get the non-pointer underlying type
func NonPtrType(typ reflect.Type) reflect.Type {
	for typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}
	return typ
}

// get the non-pointer underlying value
func NonPtrValue(v reflect.Value) reflect.Value {
	for v.Kind() == reflect.Ptr { // todo: could also look for slice here
		v = v.Elem()
	}
	return v
}

// call a function on all the the primary fields of a given struct type,
// including those on anonymous embedded structs that this struct has, passing
// the current (embedded) type and StructField -- effectively flattens the
// reflect field list
func FlatFieldsTypeFun(typ reflect.Type, fun func(typ reflect.Type, field reflect.StructField)) {
	typ = NonPtrType(typ)
	for i := 0; i < typ.NumField(); i++ {
		f := typ.Field(i)
		if f.Type.Kind() == reflect.Struct && f.Anonymous {
			FlatFieldsTypeFun(f.Type, fun)
		} else {
			fun(typ, f)
		}
	}
}

// call a function on all the the primary fields of a given struct value
// including those on anonymous embedded structs that this struct has, passing
// the current (embedded) type and StructField -- effectively flattens the
// reflect field list
func FlatFieldsValueFun(stru interface{}, fun func(stru interface{}, typ reflect.Type, field reflect.StructField, fieldVal reflect.Value)) {
	v := NonPtrValue(reflect.ValueOf(stru))
	typ := v.Type()
	for i := 0; i < typ.NumField(); i++ {
		f := typ.Field(i)
		vf := v.Field(i)
		vfi := vf.Interface() // todo: check for interfaceablity etc
		if vfi == nil || vfi == stru {
			continue
		}
		if f.Type.Kind() == reflect.Struct && f.Anonymous {
			FlatFieldsValueFun(vf.Addr().Interface(), fun) // key to take addr here so next level is addressable
		} else {
			fun(vfi, typ, f, vf)
		}
	}
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
			if f.Type == embed {
				return vf.Interface()
			}
			rv := EmbededStruct(vf.Addr().Interface(), embed)
			if rv != nil {
				return rv
			}
		}
	}
	return nil
}
