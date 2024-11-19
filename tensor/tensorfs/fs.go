// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package tensorfs

import (
	"bytes"
	"errors"
	"io/fs"
	"slices"
	"time"

	"cogentcore.org/core/base/fileinfo"
	"cogentcore.org/core/base/fsx"
)

// fs.go contains all the io/fs interface implementations, and other fs functionality.

// Open opens the given node at given path within this tensorfs filesystem.
func (nd *Node) Open(name string) (fs.File, error) {
	itm, err := nd.NodeAtPath(name)
	if err != nil {
		return nil, err
	}
	if itm.IsDir() {
		return &DirFile{File: File{Reader: *bytes.NewReader(itm.Bytes()), Node: itm}}, nil
	}
	return &File{Reader: *bytes.NewReader(itm.Bytes()), Node: itm}, nil
}

// Stat returns a FileInfo describing the file.
// If there is an error, it should be of type *PathError.
func (nd *Node) Stat(name string) (fs.FileInfo, error) {
	return nd.NodeAtPath(name)
}

// Sub returns a data FS corresponding to the subtree rooted at dir.
func (nd *Node) Sub(dir string) (fs.FS, error) {
	if err := nd.mustDir("Sub", dir); err != nil {
		return nil, err
	}
	if !fs.ValidPath(dir) {
		return nil, &fs.PathError{Op: "Sub", Path: dir, Err: errors.New("invalid name")}
	}
	if dir == "." || dir == "" || dir == nd.name {
		return nd, nil
	}
	cd := dir
	cur := nd
	root, rest := fsx.SplitRootPathFS(dir)
	if root == "." || root == nd.name {
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
		sd, ok := cur.Dir.AtTry(root)
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
func (nd *Node) ReadDir(dir string) ([]fs.DirEntry, error) {
	sd, err := nd.DirAtPath(dir)
	if err != nil {
		return nil, err
	}
	names := sd.dirNamesAlpha()
	ents := make([]fs.DirEntry, len(names))
	for i, nm := range names {
		ents[i] = sd.Dir.At(nm)
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
func (nd *Node) ReadFile(name string) ([]byte, error) {
	itm, err := nd.NodeAtPath(name)
	if err != nil {
		return nil, err
	}
	if itm.IsDir() {
		return nil, &fs.PathError{Op: "ReadFile", Path: name, Err: errors.New("Node is a directory")}
	}
	return slices.Clone(itm.Bytes()), nil
}

//////// FileInfo interface:

func (nd *Node) Name() string { return nd.name }

// Size returns the size of known data Values, or it uses
// the Sizer interface, otherwise returns 0.
func (nd *Node) Size() int64 {
	if nd.Tensor == nil {
		return 0
	}
	return nd.Tensor.AsValues().Sizeof()
}

func (nd *Node) IsDir() bool {
	return nd.Dir != nil
}

func (nd *Node) ModTime() time.Time {
	return nd.modTime
}

func (nd *Node) Mode() fs.FileMode {
	if nd.IsDir() {
		return 0755 | fs.ModeDir
	}
	return 0444
}

// Sys returns the Dir or Value
func (nd *Node) Sys() any {
	if nd.Tensor != nil {
		return nd.Tensor
	}
	return nd.Dir
}

//////// DirEntry interface

func (nd *Node) Type() fs.FileMode {
	return nd.Mode().Type()
}

func (nd *Node) Info() (fs.FileInfo, error) {
	return nd, nil
}

//////// Misc

func (nd *Node) KnownFileInfo() fileinfo.Known {
	if nd.Tensor == nil {
		return fileinfo.Unknown
	}
	tsr := nd.Tensor
	if tsr.Len() > 1 {
		return fileinfo.Tensor
	}
	// scalars by type
	if tsr.IsString() {
		return fileinfo.String
	}
	return fileinfo.Number
}

// Bytes returns the byte-wise representation of the data Value.
// This is the actual underlying data, so make a copy if it can be
// unintentionally modified or retained more than for immediate use.
func (nd *Node) Bytes() []byte {
	if nd.Tensor == nil || nd.Tensor.NumDims() == 0 || nd.Tensor.Len() == 0 {
		return nil
	}
	return nd.Tensor.AsValues().Bytes()
}
