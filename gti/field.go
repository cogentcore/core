// Copyright (c) 2023, The Goki Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gti

import "reflect"

// Field represents a field or embed in a struct.
type Field struct {

	// Name is the name of the field (eg: Icon)
	Name string

	// Doc has all of the comment documentation
	// info as one string with directives removed.
	Doc string
}

func (f Field) GoString() string { return StructGoString(f) }

// GetField recursively attempts to extract the [gti.Field]
// with the given name from the given struct [reflect.Value],
// by searching through all of the embeds if it can not find
// it directly in the struct.
func GetField(val reflect.Value, field string) *Field {
	typ := TypeByName(TypeName(val.Type()))
	// if we are not in the gti registry, there is nothing that we can do
	if typ == nil {
		return nil
	}
	for _, f := range typ.Fields {
		if f.Name == field {
			// we have successfully gotten the field
			return &f
		}
	}
	// otherwise, we go through all of the embeds and call
	// GetField recursively on them
	for _, e := range typ.Embeds {
		rf := val.FieldByName(e.Name)
		f := GetField(rf, field)
		// we have successfully gotten the field
		if f != nil {
			return f
		}
	}
	return nil
}
