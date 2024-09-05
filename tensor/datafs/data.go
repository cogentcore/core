// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package datafs

import (
	"io/fs"
	"time"
	"unsafe"
)

// data is a single item of data, the "file" in the datafs.
// a subdirectory has the item = map[string]*data
type data struct {
	// name is the name of this item.  it is not a path.
	name string

	// modTime tracks time added to directory, used for ordering.
	modTime time.Time

	// item is the underlying item of data; is a map[string]*data for directories.
	item any

	// meta has arbitrary metadata.
	meta MetaData
}

///////////////////////////////
// File interface:

func (d data) Stat() (fs.FileInfo, error) {
	return &d, nil
}

func (d data) Read([]byte) (int, error) {
	return 0, nil // todo
}

func (d data) Close() error {
	return nil
}

///////////////////////////////
// FileInfo interface:

// Sizer is an interface to allow an arbitrary data item
// to report its size in bytes.  Size is automatically computed for
// known basic data items supported by datafs directly.
type Sizer interface {
	Sizeof() int64
}

func (d data) Name() string { return d.name }

// Size returns the size of known data items, or it uses
// the Sizer interface, otherwise returns 0.
func (d data) Size() int64 {
	if szr, ok := d.item.(Sizer); ok { // tensor implements Sizer
		return szr.Sizeof()
	}
	switch x := d.item.(type) {
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
	default:
		return 0
	}
}

func (d data) IsDir() bool {
	_, ok := d.item.(map[string]*data)
	return ok
}

func (d data) ModTime() time.Time {
	return d.modTime
}

func (d data) Mode() fs.FileMode {
	if d.IsDir() {
		return 0755 | fs.ModeDir
	}
	return 0444
}

// Sys returns the metadata for item
func (d data) Sys() any { return d.meta }

///////////////////////////////
// DirEntry interface

func (d data) Type() fs.FileMode {
	return d.Mode().Type()
}

func (d data) Info() (fs.FileInfo, error) {
	return d, nil
}
