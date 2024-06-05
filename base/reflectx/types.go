// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package reflectx

import (
	"path"
	"reflect"
)

// LongTypeName returns the long, full package-path qualified type name.
// This is guaranteed to be unique and used for internal storage of
// several maps to avoid any conflicts.  It is also very quick to compute.
func LongTypeName(typ reflect.Type) string {
	nptyp := NonPointerType(typ)
	nm := nptyp.Name()
	if nm != "" {
		p := nptyp.PkgPath()
		if p != "" {
			return p + "." + nm
		}
		return nm
	}
	return typ.String()
}

// ShortTypeName returns the short version of a package-qualified type name
// which just has the last element of the path.  This is what is used in
// standard Go programming, and is is used for the key to lookup reflect.Type
// names -- i.e., this is what you should save in a JSON file.
// The potential naming conflict is worth the brevity, and typically a given
// file will only contain mutually compatible, non-conflicting types.
// This is cached in ShortNames because the path.Base computation is apparently
// a bit slow.
func ShortTypeName(typ reflect.Type) string {
	nptyp := NonPointerType(typ)
	nm := nptyp.Name()
	if nm != "" {
		p := nptyp.PkgPath()
		if p != "" {
			return path.Base(p) + "." + nm
		}
		return nm
	}
	return typ.String()
}

// CloneToType creates a new pointer to the given type
// and uses [SetRobust] to copy an existing value
// (of potentially another type) to it.
func CloneToType(typ reflect.Type, val any) reflect.Value {
	vn := reflect.New(typ)
	evi := vn.Interface()
	SetRobust(evi, val)
	return vn
}
