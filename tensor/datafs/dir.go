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

	"golang.org/x/exp/maps"
)

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
	d, err := NewData(par, name)
	d.Value = make(map[string]*Data)
	return d, err
}

// Item returns data item in given directory by name.
// This is for fast access and direct usage of known
// items, and it will crash if item is not found or
// this data is not a directory.
func (d *Data) Item(name string) *Data {
	fm := d.filemap()
	return fm[name]
}

// Items returns data items in given directory by name.
// error reports any items not found, or if not a directory.
func (d *Data) Items(names ...string) ([]*Data, error) {
	if err := d.mustDir("Items", ""); err != nil {
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
// If func is nil, all items are returned.
// Any directories within this directory are returned,
// unless specifically filtered.
func (d *Data) ItemsFunc(fun func(item *Data) bool) []*Data {
	if err := d.mustDir("ItemsFunc", ""); err != nil {
		return nil
	}
	fm := d.filemap()
	names := d.DirNamesAlpha()
	var its []*Data
	for _, nm := range names {
		dt := fm[nm]
		if fun != nil && !fun(dt) {
			continue
		}
		its = append(its, dt)
	}
	return its
}

// ItemsByTimeFunc returns data items in given directory
// filtered by given function, in time order (i.e., order added).
// If func is nil, all items are returned.
// Any directories within this directory are returned,
// unless specifically filtered.
func (d *Data) ItemsByTimeFunc(fun func(item *Data) bool) []*Data {
	if err := d.mustDir("ItemsByTimeFunc", ""); err != nil {
		return nil
	}
	fm := d.filemap()
	names := d.DirNamesByTime()
	var its []*Data
	for _, nm := range names {
		dt := fm[nm]
		if fun != nil && !fun(dt) {
			continue
		}
		its = append(its, dt)
	}
	return its
}

// FlatItemsFunc returns all "leaf" (non directory) data items
// in given directory, recursively descending into directories
// to return a flat list of the entire subtree,
// filtered by given function, in alpha order.  The function can
// filter out directories to prune the tree.
// If func is nil, all items are returned.
func (d *Data) FlatItemsFunc(fun func(item *Data) bool) []*Data {
	if err := d.mustDir("FlatItemsFunc", ""); err != nil {
		return nil
	}
	fm := d.filemap()
	names := d.DirNamesAlpha()
	var its []*Data
	for _, nm := range names {
		dt := fm[nm]
		if fun != nil && !fun(dt) {
			continue
		}
		if dt.IsDir() {
			subs := dt.FlatItemsFunc(fun)
			its = append(its, subs...)
		} else {
			its = append(its, dt)
		}
	}
	return its
}

// FlatItemsByTimeFunc returns all "leaf" (non directory) data items
// in given directory, recursively descending into directories
// to return a flat list of the entire subtree,
// filtered by given function, in time order (i.e., order added).
// The function can filter out directories to prune the tree.
// If func is nil, all items are returned.
func (d *Data) FlatItemsByTimeFunc(fun func(item *Data) bool) []*Data {
	if err := d.mustDir("FlatItemsByTimeFunc", ""); err != nil {
		return nil
	}
	fm := d.filemap()
	names := d.DirNamesByTime()
	var its []*Data
	for _, nm := range names {
		dt := fm[nm]
		if fun != nil && !fun(dt) {
			continue
		}
		if dt.IsDir() {
			subs := dt.FlatItemsByTimeFunc(fun)
			its = append(its, subs...)
		} else {
			its = append(its, dt)
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

// DirNamesByTime returns the names of items in the directory
// sorted by modTime (order added).  Data must be dir by this point.
func (d *Data) DirNamesByTime() []string {
	fm := d.filemap()
	names := maps.Keys(fm)
	slices.SortFunc(names, func(a, b string) int {
		return fm[a].ModTime().Compare(fm[b].ModTime())
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
	fm := d.filemap()
	_, ok := fm[it.name]
	if ok {
		return &fs.PathError{Op: "add", Path: it.name, Err: errors.New("data item already exists; names must be unique")}
	}
	fm[it.name] = it
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
