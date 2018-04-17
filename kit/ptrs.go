// Copyright (c) 2018, Randall C. O'Reilly. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package kit

// github.com/rcoreilly/goki/ki/kit

import (
	"reflect"
)

// These are a set of consistently-named functions for navigating pointer
// types and values within the reflect system

// NonPtrType returns the non-pointer underlying type
func NonPtrType(typ reflect.Type) reflect.Type {
	if typ == nil {
		return typ
	}
	for typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}
	return typ
}

// PtrType returns the pointer to underlying type
func PtrType(typ reflect.Type) reflect.Type {
	if typ == nil {
		return typ
	}
	if typ.Kind() != reflect.Ptr {
		typ = reflect.PtrTo(typ)
	}
	return typ
}

// OnePtrType returns a type that is exactly one pointer away from a non-pointer type
func OnePtrType(typ reflect.Type) reflect.Type {
	if typ == nil {
		return typ
	}
	if typ.Kind() != reflect.Ptr {
		typ = reflect.PtrTo(typ)
	} else {
		for typ.Elem().Kind() == reflect.Ptr {
			typ = typ.Elem()
		}
	}
	return typ
}

// NonPtrValue returns the non-pointer underlying value
func NonPtrValue(v reflect.Value) reflect.Value {
	for v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	return v
}

// PtrValue returns the pointer version (Addr()) of the underlying value
func PtrValue(v reflect.Value) reflect.Value {
	if v.Kind() != reflect.Ptr {
		v = v.Addr()
	}
	return v
}

// OnePtrValue returns a value that is exactly one pointer away from a non-pointer type
func OnePtrValue(v reflect.Value) reflect.Value {
	if v.Kind() != reflect.Ptr {
		v = v.Addr()
	} else {
		for v.Elem().Kind() == reflect.Ptr {
			v = v.Elem()
		}
	}
	return v
}

// NonPtrInterface returns the non-pointer value of an interface
func NonPtrInterface(el interface{}) interface{} {
	v := reflect.ValueOf(el)
	for v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	return v.Interface()
}

// PtrInterface returns the pointer value of an interface, if it is possible to get one through Addr()
func PtrInterface(el interface{}) interface{} {
	v := reflect.ValueOf(el)
	if v.Kind() == reflect.Ptr {
		return el
	}
	if v.CanAddr() {
		return v.Addr().Interface()
	}
	return el
}

// OnePtrInterface returns the pointer value of an interface, if it is possible to get one through Addr()
func OnePtrInterface(el interface{}) interface{} {
	return OnePtrValue(reflect.ValueOf(el)).Interface()
}
