// Copyright (c) 2023, The Goki Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gti

// Field represents a field or embed in a struct,
// or an argument or return value of a function.
type Field struct {
	// Name is the name of the field (eg: Icon)
	Name string

	// Type has the fully-package-path-qualified name of the type,
	// which can be used to look up the type in the Types registry
	// (eg: goki.dev/gi.Button)
	Type string

	// LocalType is the shorter, local name of the type from the
	// perspective of the code where this field is declared
	// (eg: gi.Button or Button, depending on where this is)
	LocalType string

	// Doc, if this is a struct field, has all of the comment documentation
	// info as one string with directives removed.
	Doc string
}

// Fields represents a set of multiple [Field] objects
type Fields = []Field

// GetField recursively attempts to extract the [gti.Field]
// with the given name from the given struct [gti.Type],
// by searching through all of the embeds if it can not find
// it directly in the struct.
func GetField(typ *Type, field string) *Field {
	for _, f := range typ.Fields {
		if f.Name == field {
			// we have successfully gotten the field
			return &f
		}
	}
	// otherwise, we go through all of the embeds and call GetField recursively on them
	for _, e := range typ.Embeds {
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
