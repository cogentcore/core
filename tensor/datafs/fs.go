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

	"cogentcore.org/core/base/fsx"
	"golang.org/x/exp/maps"
)

// NewDir returns a new datafs filesystem directory with given name.
// if parent != nil and a directory, this dir is added to it.
// if name is empty, then it is set to "/", the root directory.
func NewDir(dir string, parent ...*Data) *Data {
	if dir == "" {
		dir = "/"
	}
	d := NewData(dir, parent...)
	d.Item = make(map[string]*Data)
	return d
}

// filemap returns the Item as map[string]*Data, or nil if not a dir
func (d *Data) filemap() map[string]*Data {
	fm, ok := d.Item.(map[string]*Data)
	if !ok {
		return nil
	}
	return fm
}

// Add adds an item to this directory data item.
// The only errors are if this item is not a directory,
// or the name already exists.  Names must be unique within a directory.
func (d *Data) Add(it *Data) error {
	if !d.IsDir() {
		return &fs.PathError{Op: "add", Path: it.name, Err: errors.New("this datafs item is not a directory")}
	}
	fm := d.filemap()
	_, ok := fm[it.name]
	if ok {
		return &fs.PathError{Op: "add", Path: it.name, Err: errors.New("data item already exists; names must be unique")}
	}
	fm[it.name] = it
	return nil
}

// Open opens the given data Item within this datafs filesystem.
func (d *Data) Open(name string) (fs.File, error) {
	if !d.IsDir() {
		return nil, &fs.PathError{Op: "open", Path: name, Err: errors.New("this datafs item is not a directory")}
	}
	if !fs.ValidPath(name) {
		return nil, &fs.PathError{Op: "open", Path: name, Err: errors.New("invalid name")}
	}
	fm := d.filemap()
	itm, ok := fm[name]
	if !ok {
		return nil, &fs.PathError{Op: "open", Path: name, Err: errors.New("file not found")}
	}
	return &File{Reader: *bytes.NewReader(itm.Bytes()), Data: itm}, nil
}

// Sub returns a data FS corresponding to the subtree rooted at dir.
func (d *Data) Sub(dir string) (fs.FS, error) {
	if !d.IsDir() {
		return nil, &fs.PathError{Op: "sub", Path: dir, Err: errors.New("this datafs Item is not a directory")}
	}
	if !fs.ValidPath(dir) {
		return nil, &fs.PathError{Op: "sub", Path: dir, Err: errors.New("invalid name")}
	}
	cd := dir
	cur := d
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
	dir = path.Clean(dir)
	sdf, err := d.Sub(dir) // this ensures that d is a dir
	if err != nil {
		return nil, err
	}
	sd := sdf.(*Data)
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
	if !d.IsDir() {
		return nil, &fs.PathError{Op: "readFile", Path: name, Err: errors.New("this datafs Item is not a directory")}
	}
	if !fs.ValidPath(name) {
		return nil, &fs.PathError{Op: "readFile", Path: name, Err: errors.New("invalid name")}
	}
	fm := d.filemap()
	itm, ok := fm[name]
	if !ok {
		return nil, &fs.PathError{Op: "readFile", Path: name, Err: errors.New("file not found")}
	}
	if itm.IsDir() {
		return nil, &fs.PathError{Op: "readFile", Path: name, Err: errors.New("Item is a directory")}
	}
	return slices.Clone(itm.Bytes()), nil
}

// Mkdir creates a new directory with the specified name.
// The only error is if this item is not a directory.
func (d *Data) Mkdir(name string) (*Data, error) {
	if !d.IsDir() {
		return nil, &fs.PathError{Op: "readFile", Path: name, Err: errors.New("this datafs Item is not a directory")}
	}
	return NewDir(name, d), nil
}
