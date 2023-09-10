// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gti

import (
	"fmt"
)

// TypeRegistry provides a way to look up types from string long names
// (package_url.Type, e.g., goki.dev/gi/v2/gi.Button)
var TypeRegistry = map[string]*Type{}

// TypeByName returns a Type by name (package.Type, e,g. gi.Button), or error if not found
func TypeByName(nm string) (*Type, error) {
	tp, ok := TypeRegistry[nm]
	if !ok {
		return nil, fmt.Errorf("type %q not found", nm)
	}
	return tp, nil
}

// ShortTypeName returns the short version of a package-qualified type name
// which just has the last element of the path.  This is what is used in
// standard Go programming, and is is used for the key to lookup reflect.Type
// names -- i.e., this is what you should save in a JSON file.
// The potential naming conflict is worth the brevity, and typically a given
// file will only contain mutually-compatible, non-conflicting types.
// This is cached in ShortNames because the path.Base computation is apparently
// a bit slow.
// func ShortTypeName(typ reflect.Type) string {
// 	return path.Base(typ.PkgPath()) + "." + typ.Name()
// }

/*
// TypeFor returns the [Type] for the given
// type. It returns nil if the type is not found
// in the [TypeRegistry].
func TypeFor[T any]() *Type {
	for _, typ := range TypeRegistry {
		if _, ok := typ.Instance.(T); ok {
			return typ
		}
	}
	return nil
}
*/
