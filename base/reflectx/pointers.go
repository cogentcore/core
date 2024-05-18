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
		// slog.Error("reflectx.OnePointerValue: cannot take address of value", "value", v)
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

// Underlying returns the actual underlying version of the given value,
// going through any pointers and interfaces.
func Underlying(v reflect.Value) reflect.Value {
	return UnderlyingPointer(v).Elem()
}

// UnderlyingPointer returns a pointer to the actual underlying version of the
// given value, going through any pointers and interfaces.
func UnderlyingPointer(v reflect.Value) reflect.Value {
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

// NewFrom returns a value that is guaranteed to be a pointer to the [Underlying] version of
// the given value. If that value is not addressable, it makes a new fake pointer that points
// to a copy of the value, not the actual value. This should only be used you do not need the
// pointer to actually point to the original value.
func NewFrom(v reflect.Value) reflect.Value {
	u := Underlying(v)
	if u.CanAddr() {
		return u.Addr()
	}
	p := reflect.New(u.Type())
	p.Elem().Set(u)
	return p
}
