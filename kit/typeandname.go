// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package kit

import (
	"fmt"
	"reflect"
	"strings"
)

// a type and a name -- useful for specifying configurations of children in Ki
// nodes, and various other use-cases
type TypeAndName struct {
	Type reflect.Type
	Name string
}

// list of type-and-names -- can be created from a string spec
type TypeAndNameList []TypeAndName

// construct a type-and-name list from a list of type name pairs, space separated -- can include any json-like { } , [ ] formatting which is all stripped away and just the pairs of names are used
func (t *TypeAndNameList) SetFromString(str string) error {
	str = strings.Replace(str, ",", " ", -1)
	str = strings.Replace(str, "{", " ", -1)
	str = strings.Replace(str, "}", " ", -1)
	str = strings.Replace(str, "[", " ", -1)
	str = strings.Replace(str, "]", " ", -1)
	ds := strings.Fields(str) // split by whitespace
	sz := len(ds)
	for i := 0; i < sz; i += 2 {
		tn := ds[i]
		nm := ds[i+1]
		typ := Types.Type(tn)
		if typ == nil {
			return fmt.Errorf("TypeAndNameList SetFromString: Types type name not found: %v", tn)
		}
		(*t) = append(*t, TypeAndName{typ, nm})
	}
	return nil
}

func (t *TypeAndNameList) Add(typ reflect.Type, nm string) {
	(*t) = append(*t, TypeAndName{typ, nm})
}
