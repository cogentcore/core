// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package reflectx

import (
	"reflect"
)

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
	if !v.IsValid() {
		return v
	}
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
	if !v.IsValid() {
		return v
	}
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
	if !v.IsValid() {
		return v
	}
	for v.Type().Kind() == reflect.Interface || v.Type().Kind() == reflect.Pointer {
		v = v.Elem()
		if !v.IsValid() {
			return v
		}
	}
	return v
}

// UnderlyingPointer returns a pointer to the actual underlying version of the
// given value, going through any pointers and interfaces.
func UnderlyingPointer(v reflect.Value) reflect.Value {
	if !v.IsValid() {
		return v
	}
	uv := Underlying(v)
	if !uv.IsValid() {
		return v
	}
	return OnePointerValue(uv)
}

// NonNilNew has the same overall behavior as [reflect.New] except that
// it traverses through any pointers such that a new zero non-pointer value
// will be created in the end, so any pointers in the original type will not
// be nil. For example, in pseudo-code, NonNilNew(**int) will return
// &(&(&(0))).
func NonNilNew(typ reflect.Type) reflect.Value {
	n := 0
	for typ.Kind() == reflect.Pointer {
		n++
		typ = typ.Elem()
	}
	v := reflect.New(typ)
	for range n {
		pv := reflect.New(v.Type())
		pv.Elem().Set(v)
		v = pv
	}
	return v
}
