// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package tensorfs

import (
	"errors"
	"io/fs"
	"time"

	"cogentcore.org/core/tensor"
)

const (
	// Preserve is used for Overwrite flag, indicating to not overwrite and preserve existing.
	Preserve = false

	// Overwrite is used for Overwrite flag, indicating to overwrite existing.
	Overwrite = true
)

// CopyFromValue copies value from given source node, cloning it.
func (d *Node) CopyFromValue(frd *Node) {
	d.modTime = time.Now()
	d.Tensor = tensor.Clone(frd.Tensor)
}

// Clone returns a copy of this node, recursively cloning directory nodes
// if it is a directory.
func (nd *Node) Clone() *Node {
	if !nd.IsDir() {
		cp, _ := newNode(nil, nd.name)
		cp.Tensor = tensor.Clone(nd.Tensor)
		return cp
	}
	nodes, _ := nd.Nodes()
	cp, _ := NewDir(nd.name)
	for _, it := range nodes {
		cp.Add(it.Clone())
	}
	return cp
}

// Copy copies node(s) from given paths to given path or directory.
// if there are multiple from nodes, then to must be a directory.
// must be called on a directory node.
func (dir *Node) Copy(overwrite bool, to string, from ...string) error {
	if err := dir.mustDir("Copy", to); err != nil {
		return err
	}
	switch {
	case to == "":
		return &fs.PathError{Op: "Copy", Path: to, Err: errors.New("to location is empty")}
	case len(from) == 0:
		return &fs.PathError{Op: "Copy", Path: to, Err: errors.New("no from sources specified")}
	}
	// todo: check for to conflict first here..
	tod, _ := dir.NodeAtPath(to)
	var errs []error
	if len(from) > 1 && tod != nil && !tod.IsDir() {
		return &fs.PathError{Op: "Copy", Path: to, Err: errors.New("multiple source nodes requires destination to be a directory")}
	}
	targd := dir
	targf := to
	if tod != nil && tod.IsDir() {
		targd = tod
		targf = ""
	}
	for _, fr := range from {
		opstr := fr + " -> " + to
		frd, err := dir.NodeAtPath(fr)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		if targf == "" {
			if trg, ok := targd.Dir.AtTry(frd.name); ok { // target exists
				switch {
				case trg.IsDir() && frd.IsDir():
					// todo: copy all nodes from frd into trg
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
		}
		nw := frd.Clone()
		if targf != "" {
			nw.name = targf
		}
		targd.Add(nw)
	}
	return errors.Join(errs...)
}
