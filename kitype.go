// Copyright (c) 2018, Randall C. O'Reilly. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package ki

import (
	"bytes"
	"fmt"
	"reflect"
)

// KiType provides JSON marshal / unmarshal with encoding of underlying type name
type KiType struct {
	T reflect.Type
}

// MarshalJSON saves only the type name
func (k KiType) MarshalJSON() ([]byte, error) {
	if k.T == nil {
		b := []byte("null")
		return b, nil
	}
	nm := "\"" + k.T.Name() + "\""
	b := []byte(nm)
	return b, nil
}

// UnmarshalJSON loads the type name and looks it up in the KiTypes registry of type names
func (k *KiType) UnmarshalJSON(b []byte) error {
	if bytes.Equal(b, []byte("null")) {
		k.T = nil
		return nil
	}
	tn := string(bytes.Trim(bytes.TrimSpace(b), "\""))
	// fmt.Printf("loading type: %v", tn)
	typ := KiTypes.GetType(tn)
	if typ == nil {
		return fmt.Errorf("KiType UnmarshalJSON: KiTypes type name not found: %v", tn)
	}
	k.T = typ
	return nil
}
