// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package table

import (
	"fmt"
	"reflect"

	"cogentcore.org/core/base/reflectx"
)

// NewSliceTable returns a new Table with data from the given slice
// of structs.
func NewSliceTable(st any) (*Table, error) {
	npv := reflectx.NonPointerValue(reflect.ValueOf(st))
	if npv.Kind() != reflect.Slice {
		return nil, fmt.Errorf("NewSliceTable: not a slice")
	}
	eltyp := reflectx.NonPointerType(npv.Type().Elem())
	if eltyp.Kind() != reflect.Struct {
		return nil, fmt.Errorf("NewSliceTable: element type is not a struct")
	}
	dt := New()

	for i := 0; i < eltyp.NumField(); i++ {
		f := eltyp.Field(i)
		kind := f.Type.Kind()
		if !reflectx.KindIsBasic(kind) {
			continue
		}
		dt.AddColumnOfType(f.Name, kind)
	}
	UpdateSliceTable(st, dt)
	return dt, nil
}

// UpdateSliceTable updates given Table with data from the given slice
// of structs, which must be the same type as used to configure the table
func UpdateSliceTable(st any, dt *Table) {
	npv := reflectx.NonPointerValue(reflect.ValueOf(st))
	eltyp := reflectx.NonPointerType(npv.Type().Elem())

	nr := npv.Len()
	dt.SetNumRows(nr)
	for ri := 0; ri < nr; ri++ {
		for i := 0; i < eltyp.NumField(); i++ {
			f := eltyp.Field(i)
			kind := f.Type.Kind()
			if !reflectx.KindIsBasic(kind) {
				continue
			}
			val := npv.Index(ri).Field(i).Interface()
			cl := dt.Column(f.Name)
			if kind == reflect.String {
				cl.SetStringRow(val.(string), ri)
			} else {
				fv, _ := reflectx.ToFloat(val)
				cl.SetFloatRow(fv, ri)
			}
		}
	}
}
