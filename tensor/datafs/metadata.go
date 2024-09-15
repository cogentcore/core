// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package datafs

import (
	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/base/metadata"
)

// This file provides standardized metadata options for frequent
// use cases, using codified key names to eliminate typos.

// SetMetaItems sets given metadata for Value items in given directory
// with given names.  Returns error for any items not found.
func (d *Data) SetMetaItems(key string, value any, names ...string) error {
	tsrs, err := d.Value(names...)
	for _, tsr := range tsrs {
		tsr.Tensor.Meta.Set(key, value)
	}
	return err
}

// SetCalcFunc sets a function to compute an updated Value for this Value item.
// Function is stored as CalcFunc in Metadata.  Can be called by [Data.Calc] method.
func (d *Data) SetCalcFunc(fun func() error) {
	if d.Value == nil {
		return
	}
	d.Value.Tensor.Meta.Set("CalcFunc", fun)
}

// Calc calls function set by [Data.SetCalcFunc] to compute an updated Value
// for this data item. Returns an error if func not set, or any error from func itself.
// Function is stored as CalcFunc in Metadata.
func (d *Data) Calc() error {
	if d.Value == nil {
		return
	}
	fun, err := metadata.Get[func() error](d.Value.Tensor.Meta, "CalcFunc")
	if err != nil {
		return err
	}
	return fun()
}

// CalcAll calls function set by [Data.SetCalcFunc] for all items
// in this directory and all of its subdirectories.
// Calls Calc on items from FlatValuesFunc(nil)
func (d *Data) CalcAll() error {
	var errs []error
	items := d.FlatValuesFunc(nil)
	for _, it := range items {
		err := it.Calc()
		if err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}
