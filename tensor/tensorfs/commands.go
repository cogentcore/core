// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package tensorfs

import (
	"fmt"
	"io/fs"
	"path"
	"strings"

	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/tensor"
)

var (
	// CurDir is the current working directory.
	CurDir *Data

	// CurRoot is the current root tensorfs system.
	// A default root tensorfs is created at startup.
	CurRoot *Data
)

func init() {
	CurRoot, _ = NewDir("data")
	CurDir = CurRoot
}

// Record saves given tensor to current directory with given name.
func Record(tsr tensor.Tensor, name string) error {
	_, err := CurDir.NewData(tsr, name)
	return err // todo: could prompt about conficts, or always overwrite existing?
}

// Chdir changes the current working tensorfs directory to the named directory.
func Chdir(dir string) error {
	if CurDir == nil {
		CurDir = CurRoot
	}
	if dir == "" {
		CurDir = CurRoot
		return nil
	}
	ndir, err := CurDir.DirAtPath(dir)
	if err != nil {
		return err
	}
	CurDir = ndir
	return nil
}

// Mkdir creates a new directory with the specified name
// in the current directory.
func Mkdir(dir string) *Data {
	if CurDir == nil {
		CurDir = CurRoot
	}
	if dir == "" {
		err := &fs.PathError{Op: "Mkdir", Path: dir, Err: errors.New("path must not be empty")}
		errors.Log(err)
		return nil
	}
	return errors.Log1(CurDir.Mkdir(dir))
}

// List lists files using arguments (options and path) from the current directory.
func List(opts ...string) error {
	if CurDir == nil {
		CurDir = CurRoot
	}

	long := false
	recursive := false
	if len(opts) > 0 && len(opts[0]) > 0 && opts[0][0] == '-' {
		op := opts[0]
		if strings.Contains(op, "l") {
			long = true
		}
		if strings.Contains(op, "r") {
			recursive = true
		}
		opts = opts[1:]
	}
	dir := CurDir
	if len(opts) > 0 {
		nd, err := CurDir.DirAtPath(opts[0])
		if err == nil {
			dir = nd
		}
	}
	ls := dir.List(long, recursive)
	fmt.Println(ls)
	return nil
}

// Get returns the data item as a tensor at given path
// relative to the current working directory.
// This is the direct pointer to the data item, so changes
// to it will change the data item. Clone the data to make
// a new copy disconnected from the original.
func Get(name string) tensor.Tensor {
	if CurDir == nil {
		CurDir = CurRoot
	}
	if name == "" {
		err := &fs.PathError{Op: "Get", Path: name, Err: errors.New("name must not be empty")}
		errors.Log(err)
		return nil
	}
	d, err := CurDir.ItemAtPath(name)
	if errors.Log(err) != nil {
		return nil
	}
	if d.IsDir() {
		err := &fs.PathError{Op: "Get", Path: name, Err: errors.New("item is a directory, not a data item")}
		errors.Log(err)
		return nil
	}
	return d.Data
}

// Set sets tensor data item to given data item name or path
// relative to the current working directory.
// If the item already exists, its previous data tensor is
// updated to the given one; if it doesn't, then a new data
// item is created.
func Set(name string, tsr tensor.Tensor) error {
	if CurDir == nil {
		CurDir = CurRoot
	}
	if name == "" {
		err := &fs.PathError{Op: "Set", Path: name, Err: errors.New("name must not be empty")}
		return errors.Log(err)
	}
	itm, err := CurDir.ItemAtPath(name)
	if err == nil {
		if itm.IsDir() {
			err := &fs.PathError{Op: "Set", Path: name, Err: errors.New("existing item is a directory, not a data item")}
			return errors.Log(err)
		}
		itm.Data = tsr
		return nil
	}
	cd := CurDir
	dir, name := path.Split(name)
	if dir != "" {
		d, err := CurDir.DirAtPath(dir)
		if err != nil {
			return errors.Log(err)
		}
		cd = d
	}
	_, err = cd.NewData(tsr, name)
	return errors.Log(err)
}
