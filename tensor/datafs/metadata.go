// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package datafs

import (
	"fmt"

	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/plot/plotcore"
	"golang.org/x/exp/maps"
)

// Metadata can be attached to any data item
type Metadata map[string]any

func (md *Metadata) init() {
	if *md == nil {
		*md = make(map[string]any)
	}
}

// GetMetadata gets metadata value of given type.
// returns non-nil error if not present or item is a different type.
func GetMetadata[T any](md Metadata, key string) (T, error) {
	var z T
	x, ok := md[key]
	if !ok {
		return z, fmt.Errorf("key %q not found in metadata", key)
	}
	v, ok := x.(T)
	if !ok {
		return z, fmt.Errorf("key %q has a different type than expected %t: is %t", key, z, x)
	}
	return v, nil
}

// SetMetadata sets the given metadata for this item.
func (d *Data) SetMetadata(key string, value any) {
	d.Meta.init()
	d.Meta[key] = value
}

// CopyMetadata does a shallow copy of metadata from source metadata
// to this data item.
func (d *Data) CopyMetadata(md Metadata) {
	d.Meta.init()
	maps.Copy(d.Meta, md)
}

// SetMetadataItems sets given metadata for items in given directory
// with given names.  Returns error for any items not found.
func (d *Data) SetMetadataItems(key string, value any, names ...string) error {
	its, err := d.Items(names...)
	for _, it := range its {
		it.SetMetadata(key, value)
	}
	return err
}

// PlotColumnZeroOne returns plot options with a fixed 0-1 range
func PlotColumnZeroOne() *plotcore.ColumnOptions {
	opts := &plotcore.ColumnOptions{}
	opts.Range.SetMin(0)
	opts.Range.SetMax(1)
	return opts
}

func (d *Data) SetPlotColumnOptions(opts *plotcore.ColumnOptions, name ...string) error {
	return d.SetMetadataItems("PlotColumnOptions", opts)
}

func (md Metadata) PlotColumnOptions() *plotcore.ColumnOptions {
	return errors.Log1(GetMetadata[*plotcore.ColumnOptions](md, "PlotColumnOptions"))
}
