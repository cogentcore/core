// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package reflectx

import (
	"reflect"
)

// These are a set of consistently named functions for navigating pointer
// types and values within the reflect system.

// NonPointerType returns a non-pointer version of the given type.
func NonPointerType(typ reflect.Type) reflect.Type {
	if typ == nil {
		return typ
	}
	for typ.Kind() == reflect.Pointer {
		typ = typ.Elem()
	}
	return typ
}

// PointerType returns the pointer version of the given type
// if it is not already a pointer type.
func PointerType(typ reflect.Type) reflect.Type {
	if typ == nil {
		return typ
	}
	if typ.Kind() != reflect.Pointer {
		typ = reflect.PointerTo(typ)
	}
	return typ
}

// OnePointerType returns a type that is exactly one pointer away
// from a non-pointer type.
func OnePointerType(typ reflect.Type) reflect.Type {
	if typ == nil {
		return typ
	}
	if typ.Kind() != reflect.Pointer {
		typ = reflect.PointerTo(typ)
	} else {
		for typ.Elem().Kind() == reflect.Pointer {
			typ = typ.Elem()
		}
	}
	return typ
}

// NonPointerValue returns a non-pointer version of the given value.
func NonPointerValue(v reflect.Value) reflect.Value {
	for v.Kind() == reflect.Pointer {
		v = v.Elem()
	}
	return v
}

// PointerValue returns a pointer to the given value if it is not already
// a pointer.
func PointerValue(v reflect.Value) reflect.Value {
	if v.Kind() == reflect.Pointer {
		return v
	}
	if v.CanAddr() {
		return v.Addr()
	}
	pv := reflect.New(v.Type())
	pv.Elem().Set(v)
	return pv
}

// OnePointerValue returns a value that is exactly one pointer away
// from a non-pointer value.
func OnePointerValue(v reflect.Value) reflect.Value {
	if v.Kind() != reflect.Pointer {
		if v.CanAddr() {
			return v.Addr()
		}
		pv := reflect.New(v.Type())
		pv.Elem().Set(v)
		return pv
	} else {
		for v.Elem().Kind() == reflect.Pointer {
			v = v.Elem()
		}
	}
	return v
}

// OnePointerUnderlyingValue returns a value that is exactly one pointer
// away from a non-pointer value. It also goes through any interfaces to
// find the actual underlying value.
func OnePointerUnderlyingValue(v reflect.Value) reflect.Value {
	npv := NonPointerValue(v)
	if !npv.IsValid() {
		return v
	}
	if npv.IsZero() {
		return OnePointerValue(npv)
	}
	for npv.Type().Kind() == reflect.Interface || npv.Type().Kind() == reflect.Pointer {
		npv = npv.Elem()
	}
	return OnePointerValue(npv)
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
