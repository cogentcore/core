// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package datafs

import (
	"errors"
	"fmt"
	"io/fs"
	"path"
	"slices"
	"sort"

	"cogentcore.org/core/base/keylist"
	"cogentcore.org/core/tensor"
)

// Dir is a map of directory entry names to Data nodes.
// It retains the order that items were added in, which is
// the natural order items are processed in.
type Dir = keylist.List[string, *Data]

// NewDir returns a new datafs directory with given name.
// if parent != nil and a directory, this dir is added to it.
// if name is empty, then it is set to "root", the root directory.
// Note that "/" is not allowed for the root directory in Go [fs].
// Names must be unique within a directory.
func NewDir(name string, parent ...*Data) (*Data, error) {
	if name == "" {
		name = "root"
	}
	var par *Data
	if len(parent) == 1 {
		par = parent[0]
	}
	d, err := newData(par, name)
	d.Dir = &Dir{}
	return d, err
}

// Item returns data item in given directory by name.
// This is for fast access and direct usage of known
// items, and it will panic if item is not found or
// this data is not a directory.
func (d *Data) Item(name string) *Data {
	return d.Dir.ValueByKey(name)
}

// Value returns the [tensor.Indexed] Value for given item
// within this directory. This will panic if item is not
// found, and will return nil if it is not a Value
// (i.e., it is a directory).
func (d *Data) Value(name string) *tensor.Indexed {
	return d.Dir.ValueByKey(name).Value
}

// Items returns data items in given directory by name.
// error reports any items not found, or if not a directory.
func (d *Data) Items(names ...string) ([]*Data, error) {
	if err := d.mustDir("Items", ""); err != nil {
		return nil, err
	}
	var errs []error
	var its []*Data
	for _, nm := range names {
		dt := d.Dir.ValueByKey(nm)
		if dt != nil {
			its = append(its, dt)
		} else {
			err := fmt.Errorf("datafs Dir %q item not found: %q", d.Path(), nm)
			errs = append(errs, err)
		}
	}
	return its, errors.Join(errs...)
}

// Values returns Value items (tensors) in given directory by name.
// error reports any items not found, or if not a directory.
func (d *Data) Values(names ...string) ([]*tensor.Indexed, error) {
	if err := d.mustDir("Values", ""); err != nil {
		return nil, err
	}
	var errs []error
	var its []*tensor.Indexed
	for _, nm := range names {
		it := d.Dir.ValueByKey(nm)
		if it != nil && it.Value != nil {
			its = append(its, it.Value)
		} else {
			err := fmt.Errorf("datafs Dir %q item not found: %q", d.Path(), nm)
			errs = append(errs, err)
		}
	}
	return its, errors.Join(errs...)
}

// ItemsFunc returns data items in given directory
// filtered by given function, in directory order (e.g., order added).
// If func is nil, all items are returned.
// Any directories within this directory are returned,
// unless specifically filtered.
func (d *Data) ItemsFunc(fun func(item *Data) bool) []*Data {
	if err := d.mustDir("ItemsFunc", ""); err != nil {
		return nil
	}
	var its []*Data
	for _, it := range d.Dir.Values {
		if fun != nil && !fun(it) {
			continue
		}
		its = append(its, it)
	}
	return its
}

// ValuesFunc returns Value items (tensors) in given directory
// filtered by given function, in directory order (e.g., order added).
// If func is nil, all values are returned.
func (d *Data) ValuesFunc(fun func(item *Data) bool) []*tensor.Indexed {
	if err := d.mustDir("ItemsFunc", ""); err != nil {
		return nil
	}
	var its []*tensor.Indexed
	for _, it := range d.Dir.Values {
		if it.Value == nil {
			continue
		}
		if fun != nil && !fun(it) {
			continue
		}
		its = append(its, it.Value)
	}
	return its
}

// ItemsAlphaFunc returns data items in given directory
// filtered by given function, in alphabetical order.
// If func is nil, all items are returned.
// Any directories within this directory are returned,
// unless specifically filtered.
func (d *Data) ItemsAlphaFunc(fun func(item *Data) bool) []*Data {
	if err := d.mustDir("ItemsAlphaFunc", ""); err != nil {
		return nil
	}
	names := d.DirNamesAlpha()
	var its []*Data
	for _, nm := range names {
		it := d.Dir.ValueByKey(nm)
		if fun != nil && !fun(it) {
			continue
		}
		its = append(its, it)
	}
	return its
}

// FlatValuesFunc returns all Value items (tensor) in given directory,
// recursively descending into directories to return a flat list of
// the entire subtree, filtered by given function, in directory order
// (e.g., order added).
// The function can filter out directories to prune the tree.
// If func is nil, all Value items are returned.
func (d *Data) FlatValuesFunc(fun func(item *Data) bool) []*tensor.Indexed {
	if err := d.mustDir("FlatValuesFunc", ""); err != nil {
		return nil
	}
	var its []*tensor.Indexed
	for _, it := range d.Dir.Values {
		if fun != nil && !fun(it) {
			continue
		}
		if it.IsDir() {
			subs := it.FlatValuesFunc(fun)
			its = append(its, subs...)
		} else {
			its = append(its, it.Value)
		}
	}
	return its
}

// FlatValuesAlphaFunc returns all Value items (tensors) in given directory,
// recursively descending into directories to return a flat list of
// the entire subtree, filtered by given function, in alphabetical order.
// The function can filter out directories to prune the tree.
// If func is nil, all items are returned.
func (d *Data) FlatValuesAlphaFunc(fun func(item *Data) bool) []*tensor.Indexed {
	if err := d.mustDir("FlatValuesFunc", ""); err != nil {
		return nil
	}
	names := d.DirNamesAlpha()
	var its []*tensor.Indexed
	for _, nm := range names {
		it := d.Dir.ValueByKey(nm)
		if fun != nil && !fun(it) {
			continue
		}
		if it.IsDir() {
			subs := it.FlatValuesAlphaFunc(fun)
			its = append(its, subs...)
		} else {
			its = append(its, it.Value)
		}
	}
	return its
}

// DirAtPath returns directory at given relative path
// from this starting dir.
func (d *Data) DirAtPath(dir string) (*Data, error) {
	var err error
	dir = path.Clean(dir)
	sdf, err := d.Sub(dir) // this ensures that d is a dir
	if err != nil {
		return nil, err
	}
	return sdf.(*Data), nil
}

// Path returns the full path to this data item
func (d *Data) Path() string {
	pt := d.name
	cur := d.Parent
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
		cur = cur.Parent
	}
}

// DirNamesAlpha returns the names of items in the directory
// sorted alphabetically.  Data must be dir by this point.
func (d *Data) DirNamesAlpha() []string {
	names := slices.Clone(d.Dir.Keys)
	sort.Strings(names)
	return names
}

// DirNamesByTime returns the names of items in the directory
// sorted by modTime. Data must be dir by this point.
func (d *Data) DirNamesByTime() []string {
	names := slices.Clone(d.Dir.Keys)
	slices.SortFunc(names, func(a, b string) int {
		return d.Dir.ValueByKey(a).ModTime().Compare(d.Dir.ValueByKey(b).ModTime())
	})
	return names
}

// mustDir returns an error for given operation and path
// if this data item is not a directory.
func (d *Data) mustDir(op, path string) error {
	if !d.IsDir() {
		return &fs.PathError{Op: op, Path: path, Err: errors.New("datafs item is not a directory")}
	}
	return nil
}

// Add adds an item to this directory data item.
// The only errors are if this item is not a directory,
// or the name already exists.
// Names must be unique within a directory.
func (d *Data) Add(it *Data) error {
	if err := d.mustDir("Add", it.name); err != nil {
		return err
	}
	err := d.Dir.Add(name, it)
	if err != nil {
		return &fs.PathError{Op: "Add", Path: it.name, Err: errors.New("data item already exists; names must be unique")}
	}
	return nil
}

// Mkdir creates a new directory with the specified name.
// The only error is if this item is not a directory.
func (d *Data) Mkdir(name string) (*Data, error) {
	if err := d.mustDir("Mkdir", name); err != nil {
		return nil, err
	}
	return NewDir(name, d)
}
