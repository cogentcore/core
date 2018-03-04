// Copyright (c) 2018, Randall C. O'Reilly. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Ki is the base element of GoKi Trees
// Ki = Tree in Japanese, and "Key" in English
package ki

import (
	"bytes"
	"fmt"
	"reflect"
)

// KiType provides JSON marshal / unmarshal with encoding of underlying type name
type KiType struct {
	t reflect.Type
}

// this saves type information for each object in a slice, and the unmarshal uses it to create
// proper object types
func (k KiType) MarshalJSON() ([]byte, error) {
	if k.t == nil {
		b := []byte("null")
		return b, nil
	}
	nm := "\"" + k.t.Name() + "\""
	b := []byte(nm)
	return b, nil
}

func (k *KiType) UnmarshalJSON(b []byte) error {
	if bytes.Equal(b, []byte("null")) {
		k.t = nil
		return nil
	}
	tn := string(bytes.Trim(bytes.TrimSpace(b), "\""))
	// fmt.Printf("making type: %v", tn)
	typ := KiTypes.GetType(tn)
	if typ == nil {
		return fmt.Errorf("KiType UnmarshalJSON: KiTypes type name not found: %v", tn)
	}
	k.t = typ
	return nil
}
