// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package datafs

import (
	"bytes"
	"errors"
	"io/fs"
	"path"
	"slices"
	"sort"
	"time"
	"unsafe"

	"cogentcore.org/core/base/fsx"
	"golang.org/x/exp/maps"
)

// fs.go contains all the io/fs interface implementations

// Open opens the given data Value within this datafs filesystem.
func (d *Data) Open(name string) (fs.File, error) {
	if !fs.ValidPath(name) {
		return nil, &fs.PathError{Op: "open", Path: name, Err: errors.New("invalid name")}
	}
	dir, file := path.Split(name)
	sd, err := d.DirAtPath(dir)
	if err != nil {
		return nil, err
	}
	fm := sd.filemap()
	itm, ok := fm[file]
	if !ok {
		if dir == "" && (file == d.name || file == ".") {
			return &DirFile{File: File{Reader: *bytes.NewReader(d.Bytes()), Data: d}}, nil
		}
		return nil, &fs.PathError{Op: "open", Path: name, Err: errors.New("file not found")}
	}
	if itm.IsDir() {
		return &DirFile{File: File{Reader: *bytes.NewReader(itm.Bytes()), Data: itm}}, nil
	}
	return &File{Reader: *bytes.NewReader(itm.Bytes()), Data: itm}, nil
}

// Stat returns a FileInfo describing the file.
// If there is an error, it should be of type *PathError.
func (d *Data) Stat(name string) (fs.FileInfo, error) {
	if !fs.ValidPath(name) {
		return nil, &fs.PathError{Op: "open", Path: name, Err: errors.New("invalid name")}
	}
	dir, file := path.Split(name)
	sd, err := d.DirAtPath(dir)
	if err != nil {
		return nil, err
	}
	fm := sd.filemap()
	itm, ok := fm[file]
	if !ok {
		if dir == "" && (file == d.name || file == ".") {
			return d, nil
		}
		return nil, &fs.PathError{Op: "stat", Path: name, Err: errors.New("file not found")}
	}
	return itm, nil
}

// Sub returns a data FS corresponding to the subtree rooted at dir.
func (d *Data) Sub(dir string) (fs.FS, error) {
	if err := d.mustDir("sub", dir); err != nil {
		return nil, err
	}
	if !fs.ValidPath(dir) {
		return nil, &fs.PathError{Op: "sub", Path: dir, Err: errors.New("invalid name")}
	}
	if dir == "." || dir == "" || dir == d.name {
		return d, nil
	}
	cd := dir
	cur := d
	root, rest := fsx.SplitRootPathFS(dir)
	if root == "." || root == d.name {
		cd = rest
	}
	for {
		if cd == "." || cd == "" {
			return cur, nil
		}
		root, rest := fsx.SplitRootPathFS(cd)
		if root == "." && rest == "" {
			return cur, nil
		}
		cd = rest
		fm := cur.filemap()
		sd, ok := fm[root]
		if !ok {
			return nil, &fs.PathError{Op: "sub", Path: dir, Err: errors.New("directory not found")}
		}
		if !sd.IsDir() {
			return nil, &fs.PathError{Op: "sub", Path: dir, Err: errors.New("is not a directory")}
		}
		cur = sd
	}
}

// ReadDir returns the contents of the given directory within this filesystem.
// Use "." (or "") to refer to the current directory.
func (d *Data) ReadDir(dir string) ([]fs.DirEntry, error) {
	sd, err := d.DirAtPath(dir)
	if err != nil {
		return nil, err
	}
	fm := sd.filemap()
	names := maps.Keys(fm)
	sort.Strings(names)
	ents := make([]fs.DirEntry, len(names))
	for i, nm := range names {
		ents[i] = fm[nm]
	}
	return ents, nil
}

// ReadFile reads the named file and returns its contents.
// A successful call returns a nil error, not io.EOF.
// (Because ReadFile reads the whole file, the expected EOF
// from the final Read is not treated as an error to be reported.)
//
// The caller is permitted to modify the returned byte slice.
// This method should return a copy of the underlying data.
func (d *Data) ReadFile(name string) ([]byte, error) {
	if err := d.mustDir("readFile", name); err != nil {
		return nil, err
	}
	if !fs.ValidPath(name) {
		return nil, &fs.PathError{Op: "readFile", Path: name, Err: errors.New("invalid name")}
	}
	dir, file := path.Split(name)
	sd, err := d.DirAtPath(dir)
	if err != nil {
		return nil, err
	}
	fm := sd.filemap()
	itm, ok := fm[file]
	if !ok {
		return nil, &fs.PathError{Op: "readFile", Path: name, Err: errors.New("file not found")}
	}
	if itm.IsDir() {
		return nil, &fs.PathError{Op: "readFile", Path: name, Err: errors.New("Value is a directory")}
	}
	return slices.Clone(itm.Bytes()), nil
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
