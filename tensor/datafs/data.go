// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package datafs

import (
	"errors"
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
	// parent is the parent data directory
	parent *Data

	// name is the name of this item.  it is not a path.
	name string

	// modTime tracks time added to directory, used for ordering.
	modTime time.Time

	// Meta has arbitrary metadata.
	Meta Metadata

	// Value is the underlying value of data;
	// is a map[string]*Data for directories.
	Value any
}

// newData returns a new Data item in given directory Data item,
// which can be nil. If not a directory, an error will be generated.
// The modTime is automatically set to now, and can be used for sorting
// by order created.  name must be unique within parent.
func newData(dir *Data, name string) (*Data, error) {
	d := &Data{parent: dir, name: name, modTime: time.Now()}
	var err error
	if dir != nil {
		err = dir.Add(d)
	}
	return d, err
}

// New adds new data item(s) of given basic type to given directory,
// with given name(s) (at least one is required).
// Values are initialized to zero value for type.
// All names must be unique in the directory.
// Returns the first item created, for immediate use of one value.
func New[T any](dir *Data, names ...string) (*Data, error) {
	if len(names) == 0 {
		err := errors.New("datafs.New requires at least 1 name")
		return nil, err
	}
	var r *Data
	var errs []error
	for _, nm := range names {
		var v T
		d, err := newData(dir, nm)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		d.Value = v
		if r == nil {
			r = d
		}
	}
	return r, errors.Join(errs...)
}

// NewTensor returns a new Tensor of given data type, shape sizes,
// and optional dimension names, in given directory Data item.
// name must be unique in the directory.
func NewTensor[T string | bool | float32 | float64 | int | int32 | byte](dir *Data, name string, sizes []int, names ...string) (tensor.Tensor, error) {
	tsr := tensor.New[T](sizes, names...)
	d, err := newData(dir, name)
	d.Value = tsr
	return tsr, err
}

// NewTable makes new table.Table(s) in given directory,
// for given name(s) (at least one is required).
// All names must be unique in the directory.
// Returns the first table created, for immediate use of one item.
func NewTable(dir *Data, names ...string) (*table.Table, error) {
	if len(names) == 0 {
		err := errors.New("datafs.New requires at least 1 name")
		return nil, err
	}
	var r *table.Table
	var errs []error
	for _, nm := range names {
		t := table.NewTable(nm)
		d, err := newData(dir, nm)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		d.Value = t
		if r == nil {
			r = t
		}
	}
	return r, errors.Join(errs...)
}

///////////////////////////////
// FileInfo interface:

// Sizer is an interface to allow an arbitrary data Value
// to report its size in bytes.  Size is automatically computed for
// known basic data Values supported by datafs directly.
type Sizer interface {
	Sizeof() int64
}

func (d *Data) Name() string { return d.name }

// Size returns the size of known data Values, or it uses
// the Sizer interface, otherwise returns 0.
func (d *Data) Size() int64 {
	if szr, ok := d.Value.(Sizer); ok { // tensor implements Sizer
		return szr.Sizeof()
	}
	switch x := d.Value.(type) {
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
	_, ok := d.Value.(map[string]*Data)
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

// Sys returns the metadata for Value
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
	tsr, _ := d.Value.(tensor.Tensor)
	return tsr
}

// AsFloat64 returns data as a float64 if it is a scalar value
// that can be so converted.  Returns false if not.
func (d *Data) AsFloat64() (float64, bool) {
	// fast path for actual floats
	if f, ok := d.Value.(float64); ok {
		return f, true
	}
	if f, ok := d.Value.(float32); ok {
		return float64(f), true
	}
	if tsr := d.AsTensor(); tsr != nil {
		return 0, false
	}
	v, err := reflectx.ToFloat(d.Value)
	if err != nil {
		return 0, false
	}
	return v, true
}

// Bytes returns the byte-wise representation of the data Value.
// This is the actual underlying data, so make a copy if it can be
// unintentionally modified or retained more than for immediate use.
func (d *Data) Bytes() []byte {
	if tsr := d.AsTensor(); tsr != nil {
		return tsr.Bytes()
	}
	size := d.Size()
	switch x := d.Value.(type) {
	// todo: other things here?
	default:
		return unsafe.Slice((*byte)(unsafe.Pointer(&x)), size)
	}
}
