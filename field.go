// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gti

import (
	"reflect"

	"goki.dev/ordmap"
)

// Field represents a field or embed in a struct,
// or an argument or return value of a function.
type Field struct {
	// Name is the name of the field (eg: Icon)
	Name string

	// Type has the fully-package-path-qualified name of the type,
	// which can be used to look up the type in the Types registry
	// (eg: goki.dev/gi/v2/gi.Button)
	Type string

	// LocalType is the shorter, local name of the type from the
	// perspective of the code where this field is declared
	// (eg: gi.Button or Button, depending on where this is)
	LocalType string

	// Doc has all of the comment documentation
	// info as one string with directives removed.
	Doc string

	// Directives has the parsed comment directives
	Directives Directives

	// Tag, if this field is part of a struct, contains the struct
	// tag for it.
	Tag reflect.StructTag
}

// Fields represents a set of multiple [Field] objects
type Fields = ordmap.Map[string, *Field]

// GetField recursively attempts to extract the [gti.Field]
// with the given name from the given struct [gti.Type],
// by searching through all of the embeds if it can not find
// it directly in the struct.
func GetField(typ *Type, field string) *Field {
	f := typ.Fields.ValByKey(field)
	// we have successfully gotten the field
	if f != nil {
		return f
	}
	// otherwise, we go through all of the embeds and call GetField recursively on them
	for _, kv := range typ.Embeds.Order {
		e := kv.Val
		etyp := TypeByName(e.Type)
		// we can't do anything if we have an un-added type
		if etyp == nil {
			return nil
		}
		f := GetField(etyp, field)
		// we have successfully gotten the field
		if f != nil {
			return f
		}
	}
	return nil
}
