// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package datafs

import (
	"io/fs"
	"reflect"
	"time"

	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/base/fileinfo"
	"cogentcore.org/core/tensor"
	"cogentcore.org/core/tensor/table"
)

// Data is a single item of data, the "file" or "directory" in the data filesystem.
// Data is represented using the [tensor] package universal data type: the
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
	// [tensor.Tensor], which can represent anything from a scalar
	// to n-dimensional data, in a range of data types.
	Data tensor.Tensor

	// Dir is for directory nodes, with all the items in the directory.
	Dir *Dir

	// DirTable is a summary [table.Table] with columns comprised of
	// Value items in the directory, which can be used for plotting or other
	// operations.
	DirTable *table.Table
}

// newData returns a new Data item in given directory Data item,
// which can be nil. If dir is not a directory, returns nil and an error.
// If an item already exists in dir with that name, that item is returned
// with an [fs.ErrExist] error, and the caller can decide how to proceed.
// The modTime is set to now. The name must be unique within parent.
func newData(dir *Data, name string) (*Data, error) {
	if dir == nil {
		return &Data{name: name, modTime: time.Now()}, nil
	}
	if err := dir.mustDir("newData", name); err != nil {
		return nil, err
	}
	if ex, ok := dir.Dir.AtTry(name); ok {
		return ex, fs.ErrExist
	}
	d := &Data{Parent: dir, name: name, modTime: time.Now()}
	dir.Dir.Add(name, d)
	return d, nil
}

// Value returns a Data value as a [tensor.Tensor]
// of given data type and shape sizes, in given directory Data item.
// If it already exists, it is returned, else a new one is made.
func Value[T tensor.DataTypes](dir *Data, name string, sizes ...int) tensor.Values {
	it := dir.Item(name)
	if it != nil {
		return it.Data.(tensor.Values)
	}
	tsr := tensor.New[T](sizes...)
	tsr.Metadata().SetName(name)
	d, err := newData(dir, name)
	if errors.Log(err) != nil {
		return nil
	}
	d.Data = tsr
	return tsr
}

// Scalar returns a scalar Data value (as a [tensor.Tensor])
// of given data type, in given directory and name.
// If it already exists, it is returned, else a new one is made.
func Scalar[T tensor.DataTypes](dir *Data, name string) tensor.Values {
	return Value[T](dir, name, 1)
}

// NewScalars makes new scalar Data value(s) (as a [tensor.Tensor])
// of given data type, in given directory.
// The names must be unique in the directory (existing items are recycled).
func NewScalars[T tensor.DataTypes](dir *Data, names ...string) {
	for _, nm := range names {
		Scalar[T](dir, nm)
	}
}

// NewOfType returns a new Data value as a [tensor.Tensor]
// of given reflect.Kind type and shape sizes per dimension, in given directory Data item.
// Supported types are string, bool (for [Bool]), float32, float64, int, int32, and byte.
// If an item with that name already exists, then it is returned.
func (d *Data) NewOfType(name string, typ reflect.Kind, sizes ...int) tensor.Values {
	it := d.Item(name)
	if it != nil {
		return it.Data.(tensor.Values)
	}
	tsr := tensor.NewOfType(typ, sizes...)
	tsr.Metadata().SetName(name)
	nd, err := newData(d, name)
	if errors.Log(err) != nil {
		return nil
	}
	nd.Data = tsr
	return tsr
}

// NewData creates a new Data node for given tensor with given name.
// If the name already exists, that item is returned with [fs.ErrExists] error.
func (d *Data) NewData(tsr tensor.Tensor, name string) (*Data, error) {
	nd, err := newData(d, name)
	if err != nil {
		return nd, err
	}
	nd.Data = tsr
	return nd, nil
}

func (d *Data) KnownFileInfo() fileinfo.Known {
	if d.Data == nil {
		return fileinfo.Unknown
	}
	tsr := d.Data
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
	if d.Data == nil || d.Data.NumDims() == 0 || d.Data.Len() == 0 {
		return nil
	}
	return d.Data.AsValues().Bytes()
}

// AsString returns data as scalar string.
func (d *Data) AsString() string {
	if d.Data == nil || d.Data.NumDims() == 0 || d.Data.Len() == 0 {
		return ""
	}
	return d.Data.String1D(0)
}

// SetString sets scalar data value from given string.
func (d *Data) SetString(v string) {
	if d.Data == nil || d.Data.NumDims() == 0 || d.Data.Len() == 0 {
		return
	}
	d.Data.SetString1D(v, 0)
}

// AsFloat64 returns data as a scalar float64 (first element of tensor).
func (d *Data) AsFloat64() float64 {
	if d.Data == nil || d.Data.NumDims() == 0 || d.Data.Len() == 0 {
		return 0
	}
	return d.Data.Float1D(0)
}

// SetFloat64 sets scalar data value from given float64.
func (d *Data) SetFloat64(v float64) {
	if d.Data == nil || d.Data.NumDims() == 0 || d.Data.Len() == 0 {
		return
	}
	d.Data.SetFloat1D(v, 0)
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
	if d.Data == nil || d.Data.NumDims() == 0 || d.Data.Len() == 0 {
		return 0
	}
	return d.Data.Int1D(0)
}

// SetInt sets scalar data value from given int.
func (d *Data) SetInt(v int) {
	if d.Data == nil || d.Data.NumDims() == 0 || d.Data.Len() == 0 {
		return
	}
	d.Data.SetInt1D(v, 0)
}
