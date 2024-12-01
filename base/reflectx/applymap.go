// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package reflectx

import (
	"fmt"
	"log"
	"reflect"

	"cogentcore.org/core/base/errors"
)

// SetFieldsFromMap sets given map[string]any values to fields of given object,
// where the map keys are field paths (with . delimiters for sub-field paths).
// The value can be any appropriate type that applies to the given field.
// It prints a message if a parameter fails to be set, and returns an error.
func SetFieldsFromMap(obj any, vals map[string]any) error {
	objv := reflect.ValueOf(obj)
	npv := NonPointerValue(objv)
	if npv.Kind() == reflect.Map {
		err := CopyMapRobust(obj, vals)
		if err != nil {
			log.Println(err)
			return err
		}
	}
	var errs []error
	for k, v := range vals {
		fld, err := FieldAtPath(objv, k)
		if err != nil {
			errs = append(errs, err)
		}
		err = SetRobust(fld.Interface(), v)
		if err != nil {
			err = fmt.Errorf("SetFieldsFromMap: was not able to apply value: %v to field: %s", v, k)
			log.Println(err)
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}
