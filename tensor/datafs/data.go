// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package datafs

import (
	"io/fs"
	"time"
	"unsafe"

	"cogentcore.org/core/base/reflectx"
	"cogentcore.org/core/tensor"
	"cogentcore.org/core/tensor/table"
)

// Data is a single item of data, the "file" in the data filesystem.
// A directory has the Item = map[string]*Data
type Data struct {
	// name is the name of this item.  it is not a path.
	name string

	// modTime tracks time added to directory, used for ordering.
	modTime time.Time

	// Meta has arbitrary metadata.
	Meta MetaData

	// Item is the underlying item of data; is a map[string]*Data for directories.
	Item any
}

// NewData returns a new Data item in given parent directory Data item,
// which can be nil and is only used if actually a directory.
// The modTime is automatically set to now, and can be used for sorting
// by order created.
func NewData(name string, parent ...*Data) *Data {
	d := &Data{name: name, modTime: time.Now()}
	if len(parent) == 1 && parent[0] != nil && parent[0].IsDir() {
		parent[0].Add(d)
	}
	return d
}

// New returns a new data item representing given value.
// Returns a pointer to the value represented by this data item.
func New[T any](name string, parent ...*Data) T {
	var v T
	NewData(name, parent...).Set(v)
	return v
}

// NewTensor returns a new Tensor of given data type
// in given parent directory Data item,  which can be nil and is only
// used if actually a directory.
func NewTensor[T string | bool | float32 | float64 | int | int32 | byte](name string, parent *Data, sizes []int, names ...string) tensor.Tensor {
	tsr := tensor.New[T](sizes, names...)
	NewData(name, parent).Set(tsr)
	return tsr
}

// NewTable returns a new table.Table
func NewTable(name string, parent ...*Data) *table.Table {
	t := table.NewTable(name)
	NewData(name, parent...).Set(t)
	return t
}

// Set sets the Item for this data element
func (d *Data) Set(val any) *Data {
	d.Item = val
	return d
}

///////////////////////////////
// FileInfo interface:

// Sizer is an interface to allow an arbitrary data Item
// to report its size in bytes.  Size is automatically computed for
// known basic data Items supported by datafs directly.
type Sizer interface {
	Sizeof() int64
}

func (d *Data) Name() string { return d.name }

// Size returns the size of known data Items, or it uses
// the Sizer interface, otherwise returns 0.
func (d *Data) Size() int64 {
	if szr, ok := d.Item.(Sizer); ok { // tensor implements Sizer
		return szr.Sizeof()
	}
	switch x := d.Item.(type) {
	case float32, int32, uint32:
		return 4
	case float64, int64:
		return 8
	case int:
		return int64(unsafe.Sizeof(x))
	case complex64:
		return 16
	case complex128:
		return 32
	}
	return 0
}

func (d *Data) IsDir() bool {
	_, ok := d.Item.(map[string]*Data)
	return ok
}

func (d *Data) ModTime() time.Time {
	return d.modTime
}

func (d *Data) Mode() fs.FileMode {
	if d.IsDir() {
		return 0755 | fs.ModeDir
	}
	return 0444
}

// Sys returns the metadata for Item
func (d *Data) Sys() any { return d.Meta }

///////////////////////////////
// DirEntry interface

func (d *Data) Type() fs.FileMode {
	return d.Mode().Type()
}

func (d *Data) Info() (fs.FileInfo, error) {
	return d, nil
}

///////////////////////////////
// Data Access

// AsTensor returns the data as a tensor if it is one, else nil.
func (d *Data) AsTensor() tensor.Tensor {
	tsr, _ := d.Item.(tensor.Tensor)
	return tsr
}

// AsFloat64 returns data as a float64 if it is a scalar value
// that can be so converted.  Returns false if not.
func (d *Data) AsFloat64() (float64, bool) {
	// fast path for actual floats
	if f, ok := d.Item.(float64); ok {
		return f, true
	}
	if f, ok := d.Item.(float32); ok {
		return float64(f), true
	}
	if tsr := d.AsTensor(); tsr != nil {
		return 0, false
	}
	v, err := reflectx.ToFloat(d.Item)
	if err != nil {
		return 0, false
	}
	return v, true
}

// Bytes returns the byte-wise representation of the data Item.
// This is the actual underlying data, so make a copy if it can be
// unintentionally modified or retained more than for immediate use.
func (d *Data) Bytes() []byte {
	if tsr := d.AsTensor(); tsr != nil {
		return tsr.Bytes()
	}
	size := d.Size()
	switch x := d.Item.(type) {
	// todo: other things here?
	default:
		return unsafe.Slice((*byte)(unsafe.Pointer(&x)), size)
	}
}
