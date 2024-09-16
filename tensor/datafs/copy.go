// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package datafs

import (
	"errors"
	"fmt"
	"io/fs"
	"time"
)

const (
	// Preserve is used for Overwrite flag, indicating to not overwrite and preserve existing.
	Preserve = false

	// Overwrite is used for Overwrite flag, indicating to overwrite existing.
	Overwrite = true
)

// CopyFromValue copies value from given source data node, cloning it.
func (d *Data) CopyFromValue(frd *Data) {
	d.modTime = time.Now()
	d.Data = frd.Data.Clone()
}

// Clone returns a copy of this data item, recursively cloning directory items
// if it is a directory.
func (d *Data) Clone() *Data {
	if !d.IsDir() {
		cp, _ := newData(nil, d.name)
		cp.Data = d.Data.Clone()
		return cp
	}
	items := d.ItemsFunc(nil)
	cp, _ := NewDir(d.name)
	for _, it := range items {
		cp.Add(it.Clone())
	}
	return cp
}

// Copy copies item(s) from given paths to given path or directory.
// if there are multiple from items, then to must be a directory.
// must be called on a directory node.
func (d *Data) Copy(overwrite bool, to string, from ...string) error {
	if err := d.mustDir("Copy", to); err != nil {
		return err
	}
	switch {
	case to == "":
		return &fs.PathError{Op: "Copy", Path: to, Err: errors.New("to location is empty")}
	case len(from) == 0:
		return &fs.PathError{Op: "Copy", Path: to, Err: errors.New("no from sources specified")}
	}
	tod, _ := d.ItemAtPath(to)
	var errs []error
	if len(from) > 1 && tod != nil && !tod.IsDir() {
		return &fs.PathError{Op: "Copy", Path: to, Err: errors.New("multiple source items requires destination to be a directory")}
	}
	targd := d
	targf := to
	if tod != nil && tod.IsDir() {
		targd = tod
		targf = ""
	}
	for _, fr := range from {
		opstr := fr + " -> " + to
		frd, err := d.ItemAtPath(fr)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		if trg, ok := targd.Dir.ValueByKeyTry(frd.name); ok { // target exists
			switch {
			case trg.IsDir() && frd.IsDir():
				// todo: copy all items from frd into trg
			case trg.IsDir(): // frd is not
				errs = append(errs, &fs.PathError{Op: "Copy", Path: opstr, Err: errors.New("cannot copy from Value onto directory of same name")})
			case frd.IsDir(): // trg is not
				errs = append(errs, &fs.PathError{Op: "Copy", Path: opstr, Err: errors.New("cannot copy from Directory onto Value of same name")})
			default: // both nodes
				if overwrite { // todo: interactive!?
					trg.CopyFromValue(frd)
				}
			}
			continue
		}
		nw := frd.Clone()
		if targf != "" {
			nw.name = targf
		}
		fmt.Println("adding new:", nw.name, nw.String())
		targd.Add(nw)
	}
	return errors.Join(errs...)
}
