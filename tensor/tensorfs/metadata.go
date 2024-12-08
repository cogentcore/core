// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package tensorfs

import (
	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/tensor"
)

// This file provides standardized metadata options for frequent
// use cases, using codified key names to eliminate typos.

// SetMetaItems sets given metadata for Value items in given directory
// with given names.  Returns error for any items not found.
func (d *Node) SetMetaItems(key string, value any, names ...string) error {
	tsrs, err := d.Values(names...)
	for _, tsr := range tsrs {
		tsr.Metadata().Set(key, value)
	}
	return err
}

// CalcAll calls function set by [Node.SetCalcFunc] for all items
// in this directory and all of its subdirectories.
// Calls Calc on items from ValuesFunc(nil)
func (d *Node) CalcAll() error {
	var errs []error
	items := d.ValuesFunc(nil)
	for _, it := range items {
		err := tensor.Calc(it)
		if err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}
