// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package table

import (
	"errors"
	"fmt"
	"reflect"
	"strings"

	"cogentcore.org/core/tensor"
)

// InsertKeyColumns returns a copy of the given Table with new columns
// having given values, inserted at the start, used as legend keys etc.
// args must be in pairs: column name, value.  All rows get the same value.
func (dt *Table) InsertKeyColumns(args ...string) *Table {
	n := len(args)
	if n%2 != 0 {
		fmt.Println("InsertKeyColumns requires even number of args as column name, value pairs")
		return dt
	}
	c := dt.Clone()
	nc := n / 2
	for j := range nc {
		colNm := args[2*j]
		val := args[2*j+1]
		col := tensor.NewString([]int{c.Rows})
		c.InsertColumn(col, colNm, 0)
		for i := range col.Values {
			col.Values[i] = val
		}
	}
	return c
}

// ConfigFromTable configures the columns of this table according to the
// values in the first two columns of given format table, conventionally named
// Name, Type (but names are not used), which must be of the string type.
func (dt *Table) ConfigFromTable(ft *Table) error {
	nmcol := ft.Columns[0]
	tycol := ft.Columns[1]
	var errs []error
	for i := range ft.Rows {
		name := nmcol.String1D(i)
		typ := strings.ToLower(tycol.String1D(i))
		kind := reflect.Float64
		switch typ {
		case "string":
			kind = reflect.String
		case "bool":
			kind = reflect.Bool
		case "float32":
			kind = reflect.Float32
		case "float64":
			kind = reflect.Float64
		case "int":
			kind = reflect.Int
		case "int32":
			kind = reflect.Int32
		case "byte", "uint8":
			kind = reflect.Uint8
		default:
			err := fmt.Errorf("ConfigFromTable: type string %q not recognized", typ)
			errs = append(errs, err)
		}
		dt.AddColumnOfType(kind, name)
	}
	return errors.Join(errs...)
}
