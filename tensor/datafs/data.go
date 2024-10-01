// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package datafs

import (
	"reflect"
	"time"

	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/base/fileinfo"
	"cogentcore.org/core/tensor"
	"cogentcore.org/core/tensor/table"
)

// Data is a single item of data, the "file" or "directory" in the data filesystem.
// Data is represented using the [tensor] package universal data type: the [tensor.Indexed]
// [tensor.Tensor], which can represent everything from a single scalar value up to
// n-dimensional collections of patterns, in a range of data types.
// Directories have an ordered map of items.
type Data struct {
	// Parent is the parent data directory.
	Parent *Data

	// name is the name of this item.  it is not a path.
	name string

	// modTime tracks time added to directory, used for ordering.
	modTime time.Time

	// Data is the data value for "files" / "leaves" in the FS,
	// represented using the universal [tensor] data type of
	// [tensor.Indexed], which can represent anything from a scalar
	// to n-dimensional data, in a range of data types.
	Data *tensor.Indexed

	// Dir is for directory nodes, with all the items in the directory.
	Dir *Dir

	// DirTable is a summary [table.Table] with columns comprised of
	// Value items in the directory, which can be used for plotting or other
	// operations.
	DirTable *table.Table
}

// newData returns a new Data item in given directory Data item,
// which can be nil. If not a directory, or the name is not unique,
// an error will be generated.
// The modTime is set to now. The name must be unique within parent.
func newData(dir *Data, name string) (*Data, error) {
	d := &Data{Parent: dir, name: name, modTime: time.Now()}
	var err error
	if dir != nil {
		err = dir.Add(d)
	}
	return d, err
}

// NewScalar returns new scalar Data value(s) (as a [tensor.Indexed])
// of given data type, in given directory.
// The names must be unique in the directory.
// Returns the first item created, for immediate use of one value.
func NewScalar[T tensor.DataTypes](dir *Data, names ...string) *tensor.Indexed {
	var first *tensor.Indexed
	for _, nm := range names {
		tsr := tensor.New[T](1)
		tsr.Metadata().SetName(nm)
		d, err := newData(dir, nm)
		if errors.Log(err) != nil {
			return nil
		}
		d.Data = tensor.NewIndexed(tsr)
		if first == nil {
			first = d.Data
		}
	}
	return first
}

// NewValue returns a new Data value as a [tensor.Indexed] [tensor.Tensor]
// of given data type and shape sizes, in given directory Data item.
// The name must be unique in the directory.
func NewValue[T tensor.DataTypes](dir *Data, name string, sizes ...int) *tensor.Indexed {
	tsr := tensor.New[T](sizes...)
	tsr.Metadata().SetName(name)
	d, err := newData(dir, name)
	if errors.Log(err) != nil {
		return nil
	}
	d.Data = tensor.NewIndexed(tsr)
	return d.Data
}

// NewOfType returns a new Data value as a [tensor.Indexed]
// of given reflect.Kind type and shape sizes per dimension, in given directory Data item.
// Supported types are string, bool (for [Bits]), float32, float64, int, int32, and byte.
// The name must be unique in the directory.
func (d *Data) NewOfType(name string, typ reflect.Kind, sizes ...int) *tensor.Indexed {
	tsr := tensor.NewOfType(typ, sizes...)
	tsr.Metadata().SetName(name)
	nd, err := newData(d, name)
	if errors.Log(err) != nil {
		return nil
	}
	nd.Data = tensor.NewIndexed(tsr)
	return nd.Data
}

func (d *Data) KnownFileInfo() fileinfo.Known {
	if d.Data == nil {
		return fileinfo.Unknown
	}
	tsr := d.Data.Tensor
	if tsr.Len() > 1 {
		return fileinfo.Tensor
	}
	// scalars by type
	if tsr.IsString() {
		return fileinfo.String
	}
	return fileinfo.Number
}

// Bytes returns the byte-wise representation of the data Value.
// This is the actual underlying data, so make a copy if it can be
// unintentionally modified or retained more than for immediate use.
func (d *Data) Bytes() []byte {
	if d.Data == nil {
		return nil
	}
	return d.Data.Tensor.Bytes()
}

// AsString returns data as scalar string.
func (d *Data) AsString() string {
	if d.Data == nil {
		return ""
	}
	return d.Data.StringRow(0)
}

// SetString sets scalar data value from given string.
func (d *Data) SetString(v string) {
	if d.Data == nil {
		return
	}
	d.Data.SetStringRow(v, 0)
}

// AsFloat64 returns data as a scalar float64 (first element of tensor).
func (d *Data) AsFloat64() float64 {
	if d.Data == nil {
		return 0
	}
	return d.Data.FloatRow(0)
}

// SetFloat64 sets scalar data value from given float64.
func (d *Data) SetFloat64(v float64) {
	if d.Data == nil {
		return
	}
	d.Data.SetFloatRow(v, 0)
}

// AsFloat32 returns data as a scalar float32 (first element of tensor).
func (d *Data) AsFloat32() float32 {
	return float32(d.AsFloat64())
}

// SetFloat32 sets scalar data value from given float32.
func (d *Data) SetFloat32(v float32) {
	d.SetFloat64(float64(v))
}

// AsInt returns data as a scalar int (first element of tensor).
func (d *Data) AsInt() int {
	if d.Data == nil {
		return 0
	}
	return d.Data.IntRow(0)
}

// SetInt sets scalar data value from given int.
func (d *Data) SetInt(v int) {
	if d.Data == nil {
		return
	}
	d.Data.SetIntRow(v, 0)
}
