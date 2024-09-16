// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package datafs

import (
	"bytes"
	"errors"
	"io/fs"
	"slices"
	"time"

	"cogentcore.org/core/base/fsx"
)

// fs.go contains all the io/fs interface implementations

// Open opens the given data Value within this datafs filesystem.
func (d *Data) Open(name string) (fs.File, error) {
	itm, err := d.ItemAtPath(name)
	if err != nil {
		return nil, err
	}
	if itm.IsDir() {
		return &DirFile{File: File{Reader: *bytes.NewReader(itm.Bytes()), Data: itm}}, nil
	}
	return &File{Reader: *bytes.NewReader(itm.Bytes()), Data: itm}, nil
}

// Stat returns a FileInfo describing the file.
// If there is an error, it should be of type *PathError.
func (d *Data) Stat(name string) (fs.FileInfo, error) {
	return d.ItemAtPath(name)
}

// Sub returns a data FS corresponding to the subtree rooted at dir.
func (d *Data) Sub(dir string) (fs.FS, error) {
	if err := d.mustDir("Sub", dir); err != nil {
		return nil, err
	}
	if !fs.ValidPath(dir) {
		return nil, &fs.PathError{Op: "Sub", Path: dir, Err: errors.New("invalid name")}
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
		sd, ok := cur.Dir.ValueByKeyTry(root)
		if !ok {
			return nil, &fs.PathError{Op: "Sub", Path: dir, Err: errors.New("directory not found")}
		}
		if !sd.IsDir() {
			return nil, &fs.PathError{Op: "Sub", Path: dir, Err: errors.New("is not a directory")}
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
	names := sd.DirNamesAlpha()
	ents := make([]fs.DirEntry, len(names))
	for i, nm := range names {
		ents[i] = sd.Dir.ValueByKey(nm)
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
	itm, err := d.ItemAtPath(name)
	if err != nil {
		return nil, err
	}
	if itm.IsDir() {
		return nil, &fs.PathError{Op: "ReadFile", Path: name, Err: errors.New("Item is a directory")}
	}
	return slices.Clone(itm.Bytes()), nil
}

///////////////////////////////
// FileInfo interface:

func (d *Data) Name() string { return d.name }

// Size returns the size of known data Values, or it uses
// the Sizer interface, otherwise returns 0.
func (d *Data) Size() int64 {
	if d.Data == nil {
		return 0
	}
	return d.Data.Tensor.Sizeof()
}

func (d *Data) IsDir() bool {
	return d.Dir != nil
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

// Sys returns the Dir or Value
func (d *Data) Sys() any {
	if d.Data != nil {
		return d.Data
	}
	return d.Dir
}

///////////////////////////////
// DirEntry interface

func (d *Data) Type() fs.FileMode {
	return d.Mode().Type()
}

func (d *Data) Info() (fs.FileInfo, error) {
	return d, nil
}
