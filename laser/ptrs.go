// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package laser

import (
	"reflect"
)

// These are a set of consistently-named functions for navigating pointer
// types and values within the reflect system

/////////////////////////////////////////////////
//  reflect.Type versions

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

// PtrType returns the pointer type for given type, if given type is not already a Ptr
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

/////////////////////////////////////////////////
//  reflect.Value versions

// NonPtrValue returns the non-pointer underlying value
func NonPtrValue(v reflect.Value) reflect.Value {
	for v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	return v
}

// PtrValue returns the pointer version (Addr()) of the underlying value if
// the value is not already a Ptr
func PtrValue(v reflect.Value) reflect.Value {
	if v.Kind() != reflect.Ptr {
		if v.CanAddr() {
			return v.Addr()
		}
		pv := reflect.New(v.Type())
		pv.Elem().Set(v)
		return pv
	}
	return v
}

// OnePtrValue returns a value that is exactly one pointer away
// from a non-pointer type
func OnePtrValue(v reflect.Value) reflect.Value {
	if v.Kind() != reflect.Ptr {
		if v.CanAddr() {
			return v.Addr()
		}
		if v.IsValid() {
			pv := reflect.New(v.Type())
			pv.Elem().Set(v)
			return pv
		}
	} else {
		for v.Elem().Kind() == reflect.Ptr {
			v = v.Elem()
		}
	}
	return v
}

// OnePtrUnderlyingValue returns a value that is exactly one pointer away
// from a non-pointer type, and also goes through an interface to find the
// actual underlying type behind the interface.
func OnePtrUnderlyingValue(v reflect.Value) reflect.Value {
	npv := NonPtrValue(v)
	if !npv.IsValid() {
		return OnePtrValue(npv)
	}
	for npv.Type().Kind() == reflect.Interface || npv.Type().Kind() == reflect.Pointer {
		npv = npv.Elem()
	}
	return OnePtrValue(npv)
}

// UnhideAnyValue returns a reflect.Value for any of the Make* functions
// that is actually assignable -- even though these functions return a pointer
// to the new object, it is somehow hidden behind an interface{} and this
// magic code, posted by someone somewhere that I cannot now find again,
// un-hides it..
func UnhideAnyValue(v reflect.Value) reflect.Value {
	vn := reflect.ValueOf(v.Interface())
	typ := vn.Type()
	ptr := reflect.New(typ)
	ptr.Elem().Set(vn)
	return ptr
}

/////////////////////////////////////////////////
//  interface{} versions

// NonPtrInterface returns the non-pointer value of an interface
func NonPtrInterface(el any) any {
	v := reflect.ValueOf(el)
	for v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	return v.Interface()
}

// PtrInterface returns the pointer value of an interface, if it is possible to get one through Addr()
func PtrInterface(el any) any {
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
func OnePtrInterface(el any) any {
	return OnePtrValue(reflect.ValueOf(el)).Interface()
}
