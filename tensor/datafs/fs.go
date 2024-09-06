// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package datafs

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"path"
	"slices"
	"sort"

	"cogentcore.org/core/base/fsx"
	"golang.org/x/exp/maps"
)

// NewDir returns a new datafs directory with given name.
// if parent != nil and a directory, this dir is added to it.
// if name is empty, then it is set to "/", the root directory.
// Names must be unique within a directory.
func NewDir(name string, parent ...*Data) (*Data, error) {
	if name == "" {
		name = "/"
	}
	var par *Data
	if len(parent) == 1 {
		par = parent[0]
	}
	d, err := NewData(par, name)
	d.Value = make(map[string]*Data)
	return d, err
}

// Items returns data items in given directory by name.
// error reports any items not found, or if not a directory.
func (d *Data) Items(names ...string) ([]*Data, error) {
	if err := d.mustDir("items", names[0]); err != nil {
		return nil, err
	}
	fm := d.filemap()
	var errs []error
	var its []*Data
	for _, nm := range names {
		dt := fm[nm]
		if dt != nil {
			its = append(its, dt)
		} else {
			err := fmt.Errorf("datafs Dir %q item not found: %q", d.Path(), nm)
			errs = append(errs, err)
		}
	}
	return its, errors.Join(errs...)
}

// ItemsFunc returns data items in given directory
// filtered by given function, in alpha order.
func (d *Data) ItemsFunc(fun func(item *Data) bool) []*Data {
	if err := d.mustDir("items-func", ""); err != nil {
		return nil
	}
	fm := d.filemap()
	names := d.DirNamesAlpha()
	var its []*Data
	for _, nm := range names {
		dt := fm[nm]
		if !fun(dt) {
			continue
		}
		its = append(its, dt)
	}
	return its
}

// ItemsAddedFunc returns data items in given directory
// filtered by given function, in added order.
func (d *Data) ItemsAddedFunc(fun func(item *Data) bool) []*Data {
	if err := d.mustDir("items-added-func", ""); err != nil {
		return nil
	}
	fm := d.filemap()
	names := d.DirNamesAdded()
	var its []*Data
	for _, nm := range names {
		dt := fm[nm]
		if !fun(dt) {
			continue
		}
		its = append(its, dt)
	}
	return its
}

// Path returns the full path to this data item
func (d *Data) Path() string {
	pt := d.name
	cur := d.parent
	loops := make(map[*Data]struct{})
	for {
		if cur == nil {
			return pt
		}
		if _, ok := loops[cur]; ok {
			return pt
		}
		pt = path.Join(cur.name, pt)
		loops[cur] = struct{}{}
		cur = cur.parent
	}
}

// filemap returns the Value as map[string]*Data, or nil if not a dir
func (d *Data) filemap() map[string]*Data {
	fm, ok := d.Value.(map[string]*Data)
	if !ok {
		return nil
	}
	return fm
}

// DirNamesAlpha returns the names of items in the directory
// sorted alphabetically.  Data must be dir by this point.
func (d *Data) DirNamesAlpha() []string {
	fm := d.filemap()
	names := maps.Keys(fm)
	sort.Strings(names)
	return names
}

// DirNamesAdded returns the names of items in the directory
// sorted by order added (modTime).  Data must be dir by this point.
func (d *Data) DirNamesAdded() []string {
	fm := d.filemap()
	names := maps.Keys(fm)
	slices.SortFunc(names, func(a, b string) int {
		ad := fm[a]
		bd := fm[b]
		if ad.ModTime().After(bd.ModTime()) {
			return -1
		}
		if bd.ModTime().After(ad.ModTime()) {
			return 1
		}
		return 0
	})
	return names
}

// mustDir returns an error for given operation and path
// if this data item is not a directory.
func (d *Data) mustDir(op, path string) error {
	if !d.IsDir() {
		return &fs.PathError{Op: "open", Path: path, Err: errors.New("datafs item is not a directory")}
	}
	return nil
}

// Add adds an item to this directory data item.
// The only errors are if this item is not a directory,
// or the name already exists.
// Names must be unique within a directory.
func (d *Data) Add(it *Data) error {
	if err := d.mustDir("add", it.name); err != nil {
		return err
	}
	fm := d.filemap()
	_, ok := fm[it.name]
	if ok {
		return &fs.PathError{Op: "add", Path: it.name, Err: errors.New("data item already exists; names must be unique")}
	}
	fm[it.name] = it
	return nil
}

// Open opens the given data Value within this datafs filesystem.
func (d *Data) Open(name string) (fs.File, error) {
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
	if err := d.mustDir("sub", dir); err != nil {
		return nil, err
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
	if err := d.mustDir("readFile", name); err != nil {
		return nil, err
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
		return nil, &fs.PathError{Op: "readFile", Path: name, Err: errors.New("Value is a directory")}
	}
	return slices.Clone(itm.Bytes()), nil
}

// Mkdir creates a new directory with the specified name.
// The only error is if this item is not a directory.
func (d *Data) Mkdir(name string) (*Data, error) {
	if err := d.mustDir("mkdir", name); err != nil {
		return nil, err
	}
	return NewDir(name, d)
}
