// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package kit

import (
	"fmt"
	"log"
	"reflect"
	"strings"
)

// SetFromDefaultTags sets values of fields in given struct based on
// `def:` default value field tags.
func SetFromDefaultTags(obj any) error {
	if IfaceIsNil(obj) {
		return nil
	}
	ov := reflect.ValueOf(obj)
	if ov.Kind() == reflect.Pointer && ov.IsNil() {
		return nil
	}
	val := NonPtrValue(ov)
	typ := val.Type()
	var err error
	for i := 0; i < typ.NumField(); i++ {
		f := typ.Field(i)
		fv := val.Field(i)
		def, ok := f.Tag.Lookup("def")
		if NonPtrType(f.Type).Kind() == reflect.Struct && (!ok || def == "") {
			SetFromDefaultTags(PtrValue(fv).Interface())
			continue
		}
		if !ok || def == "" {
			continue
		}
		if def[0] == '{' || def[0] == '[' { // complex type
			def = strings.ReplaceAll(def, `'`, `"`) // allow single quote to work as double quote for JSON format
		} else {
			def = strings.Split(def, ",")[0]
			if strings.Contains(def, ":") { // don't do ranges
				continue
			}
		}
		ok = SetRobust(PtrValue(fv).Interface(), def) // overkill but whatever
		if !ok {
			err = fmt.Errorf("SetFromDefaultTags: was not able to set field: %s in object of type: %s from val: %s", f.Name, typ.Name(), def)
			log.Println(err)
		}
	}
	return err
}
