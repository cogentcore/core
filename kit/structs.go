// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package kit

import (
	"fmt"
	"log"
	"reflect"
)

// SetFromDefaultTags sets values of fields in given struct based on
// `def:` default value field tags.
func SetFromDefaultTags(obj any) error {
	typ := NonPtrType(reflect.TypeOf(obj))
	val := NonPtrValue(reflect.ValueOf(obj))
	var err error
	for i := 0; i < typ.NumField(); i++ {
		f := typ.Field(i)
		fv := val.Field(i)
		if NonPtrType(f.Type).Kind() == reflect.Struct {
			SetFromDefaultTags(PtrValue(fv).Interface())
			continue
		}
		def, ok := f.Tag.Lookup("def")
		if !ok || def == "" {
			continue
		}
		ok = SetRobust(PtrValue(fv).Interface(), def) // overkill but whatever
		if !ok {
			err = fmt.Errorf("SetFromDefaultTags: was not able to set field: %s in object of type: %s from val: %s", f.Name, typ.Name(), def)
			log.Println(err)
		}
	}
	return err
}
