// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package datafs

import (
	"errors"
	"io/fs"
	"path"
	"sort"

	"cogentcore.org/core/base/fsx"
	"golang.org/x/exp/maps"
)

// NewFS returns a new datafs filesystem with given directory name.
// if name is empty, then it is set to "/", the root directory.
func NewFS(dir string) fs.FS {
	if dir == "" {
		dir = "/"
	}
	d := &data{name: dir}
	d.item = make(map[string]*data)
	return d
}

// filemap returns the item as map[string]data, or nil if not a dir
func (d data) filemap() map[string]*data {
	fm, ok := d.item.(map[string]*data)
	if !ok {
		return nil
	}
	return fm
}

// Open opens the given data item within this datafs filesystem.
func (d data) Open(name string) (fs.File, error) {
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
	return itm, nil
}

// Sub returns a data FS corresponding to the subtree rooted at dir.
func (d data) Sub(dir string) (fs.FS, error) {
	if !d.IsDir() {
		return nil, &fs.PathError{Op: "sub", Path: dir, Err: errors.New("this datafs item is not a directory")}
	}
	if !fs.ValidPath(dir) {
		return nil, &fs.PathError{Op: "sub", Path: dir, Err: errors.New("invalid name")}
	}
	cd := dir
	cur := &d
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
func (d data) ReadDir(dir string) ([]fs.DirEntry, error) {
	dir = path.Clean(dir)
	sdf, err := d.Sub(dir) // this ensures that d is a dir
	if err != nil {
		return nil, err
	}
	sd := sdf.(*data)
	fm := sd.filemap()
	names := maps.Keys(fm)
	sort.Strings(names)
	ents := make([]fs.DirEntry, len(names))
	for i, nm := range names {
		ents[i] = fm[nm]
	}
	return ents, nil
}
