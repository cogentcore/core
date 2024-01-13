// Copyright (c) 2023, The Goki Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gti

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
	// TODO(kai)
	// // otherwise, we go through all of the embeds and call GetField recursively on them
	// for _, e := range typ.Embeds {
	// 	etyp := TypeByName(e.Type)
	// 	// we can't do anything if we have an un-added type
	// 	if etyp == nil {
	// 		return nil
	// 	}
	// 	f := GetField(etyp, field)
	// 	// we have successfully gotten the field
	// 	if f != nil {
	// 		return f
	// 	}
	// }
	return nil
}
