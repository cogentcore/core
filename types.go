// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package laser

import (
	"path"
	"reflect"
)

// LongTypeName returns the long, full package-path qualified type name.
// This is guaranteed to be unique and used for internal storage of
// several maps to avoid any conflicts.  It is also very quick to compute.
func LongTypeName(typ reflect.Type) string {
	return typ.PkgPath() + "." + typ.Name()
}

// ShortTypeName returns the short version of a package-qualified type name
// which just has the last element of the path.  This is what is used in
// standard Go programming, and is is used for the key to lookup reflect.Type
// names -- i.e., this is what you should save in a JSON file.
// The potential naming conflict is worth the brevity, and typically a given
// file will only contain mutually-compatible, non-conflicting types.
// This is cached in ShortNames because the path.Base computation is apparently
// a bit slow.
func ShortTypeName(typ reflect.Type) string {
	return path.Base(typ.PkgPath()) + "." + typ.Name()
}

// TypeFor returns the [reflect.Type] that represents the type argument T.
// It is a copy of [reflect.TypeFor], which will likely be added in Go 1.22
// (see https://github.com/golang/go/issues/60088)
func TypeFor[T any]() reflect.Type {
	return reflect.TypeOf((*T)(nil)).Elem()
}

// CloneToType creates a new object of given type, and uses SetRobust to copy
// an existing value (of perhaps another type) into it
func CloneToType(typ reflect.Type, val any) reflect.Value {
	vn := reflect.New(typ)
	evi := vn.Interface()
	SetRobust(evi, val)
	return vn
}

// MakeOfType creates a new object of given type with appropriate magic foo to
// make it usable
func MakeOfType(typ reflect.Type) reflect.Value {
	if NonPtrType(typ).Kind() == reflect.Map {
		return MakeMap(typ)
	} else if NonPtrType(typ).Kind() == reflect.Slice {
		return MakeSlice(typ, 0, 0)
	}
	vn := reflect.New(typ)
	return vn
}
