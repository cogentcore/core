// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package datafs

import (
	"errors"
	"reflect"
	"time"
	"unsafe"

	"cogentcore.org/core/base/fileinfo"
	"cogentcore.org/core/base/metadata"
	"cogentcore.org/core/base/reflectx"
	"cogentcore.org/core/tensor"
	"cogentcore.org/core/tensor/table"
)

// Data is a single item of data, the "file" or "directory" in the data filesystem.
type Data struct {
	// Parent is the parent data directory
	Parent *Data

	// name is the name of this item.  it is not a path.
	name string

	// modTime tracks time added to directory, used for ordering.
	modTime time.Time

	// Meta has metadata, including standardized support for
	// plotting options, compute functions.
	Meta metadata.Data

	// Value is the underlying value of data;
	// is a map[string]*Data for directories.
	Value any
}

// NewData returns a new Data item in given directory Data item,
// which can be nil. If not a directory, an error will be generated.
// The modTime is automatically set to now, and can be used for sorting
// by order created. The name must be unique within parent.
func NewData(dir *Data, name string) (*Data, error) {
	d := &Data{Parent: dir, name: name, modTime: time.Now()}
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
		d, err := NewData(dir, nm)
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
// The name must be unique in the directory.
func NewTensor[T string | bool | float32 | float64 | int | int32 | byte](dir *Data, name string, sizes []int, names ...string) (tensor.Tensor, error) {
	tsr := tensor.New[T](sizes, names...)
	d, err := NewData(dir, name)
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
		d, err := NewData(dir, nm)
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
// Data Access

// IsNumeric returns true if the [DataType] is a basic scalar
// numerical value, e.g., float32, int, etc.
func (d *Data) IsNumeric() bool {
	return reflectx.KindIsNumber(d.DataType())
}

// DataType returns the type of the data elements in the tensor.
// Bool is returned for the Bits tensor type.
func (d *Data) DataType() reflect.Kind {
	if d.Value == nil {
		return reflect.Invalid
	}
	return reflect.TypeOf(d.Value).Kind()
}

func (d *Data) KnownFileInfo() fileinfo.Known {
	if tsr := d.AsTensor(); tsr != nil {
		return fileinfo.Tensor
	}
	kind := d.DataType()
	if reflectx.KindIsNumber(kind) {
		return fileinfo.Number
	}
	if kind == reflect.String {
		return fileinfo.String
	}
	return fileinfo.Unknown
}

// AsTensor returns the data as a tensor if it is one, else nil.
func (d *Data) AsTensor() tensor.Tensor {
	tsr, _ := d.Value.(tensor.Tensor)
	return tsr
}

// AsTable returns the data as a table if it is one, else nil.
func (d *Data) AsTable() *table.Table {
	dt, _ := d.Value.(*table.Table)
	return dt
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
	if dt := d.AsTable(); dt != nil {
		return 0, false
	}
	v, err := reflectx.ToFloat(d.Value)
	if err != nil {
		return 0, false
	}
	return v, true
}

// SetFloat64 sets data from given float64 if it is a scalar value
// that can be so set.  Returns false if not.
func (d *Data) SetFloat64(v float64) bool {
	// fast path for actual floats
	if _, ok := d.Value.(float64); ok {
		d.Value = v
		return true
	}
	if _, ok := d.Value.(float32); ok {
		d.Value = float32(v)
		return true
	}
	if tsr := d.AsTensor(); tsr != nil {
		return false
	}
	if dt := d.AsTable(); dt != nil {
		return false
	}
	err := reflectx.SetRobust(&d.Value, v)
	if err != nil {
		return false
	}
	return true
}

// AsFloat32 returns data as a float32 if it is a scalar value
// that can be so converted.  Returns false if not.
func (d *Data) AsFloat32() (float32, bool) {
	v, ok := d.AsFloat64()
	return float32(v), ok
}

// SetFloat32 sets data from given float32 if it is a scalar value
// that can be so set.  Returns false if not.
func (d *Data) SetFloat32(v float32) bool {
	return d.SetFloat64(float64(v))
}

// AsString returns data as a string if it is a scalar value
// that can be so converted.  Returns false if not.
func (d *Data) AsString() (string, bool) {
	// fast path for actual strings
	if s, ok := d.Value.(string); ok {
		return s, true
	}
	if tsr := d.AsTensor(); tsr != nil {
		return "", false
	}
	if dt := d.AsTable(); dt != nil {
		return "", false
	}
	s := reflectx.ToString(d.Value)
	return s, true
}

// SetString sets data from given string if it is a scalar value
// that can be so set.  Returns false if not.
func (d *Data) SetString(v string) bool {
	// fast path for actual strings
	if _, ok := d.Value.(string); ok {
		d.Value = v
		return true
	}
	if tsr := d.AsTensor(); tsr != nil {
		return false
	}
	if dt := d.AsTable(); dt != nil {
		return false
	}
	err := reflectx.SetRobust(&d.Value, v)
	if err != nil {
		return false
	}
	return true
}

// AsInt returns data as a int if it is a scalar value
// that can be so converted.  Returns false if not.
func (d *Data) AsInt() (int, bool) {
	// fast path for actual ints
	if f, ok := d.Value.(int); ok {
		return f, true
	}
	if tsr := d.AsTensor(); tsr != nil {
		return 0, false
	}
	if dt := d.AsTable(); dt != nil {
		return 0, false
	}
	v, err := reflectx.ToInt(d.Value)
	if err != nil {
		return 0, false
	}
	return int(v), true
}

// SetInt sets data from given int if it is a scalar value
// that can be so set.  Returns false if not.
func (d *Data) SetInt(v int) bool {
	// fast path for actual ints
	if _, ok := d.Value.(int); ok {
		d.Value = v
		return true
	}
	if tsr := d.AsTensor(); tsr != nil {
		return false
	}
	if dt := d.AsTable(); dt != nil {
		return false
	}
	err := reflectx.SetRobust(&d.Value, v)
	if err != nil {
		return false
	}
	return true
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
