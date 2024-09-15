// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package datafs

import (
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

// NewValue returns a new Data value as an [tensor.Indexed] [tensor.Tensor]
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

func (d *Data) KnownFileInfo() fileinfo.Known {
	if d.Data == nil {
		return fileinfo.Unknown
	}
	tsr := d.Data.Tensor
	if tsr.Len() > 1 {
		return fileinfo.Tensor
	}
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
