// Copyright (c) 2023, The Goki Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build js

package fs

import (
	"context"
	"syscall/js"

	"github.com/hack-pad/hackpadfs"
	"github.com/hack-pad/hackpadfs/indexeddb"
)

// FS represents a filesystem that implements the Node.js fs API.
// It is backed by an IndexedDB-based storage mechanism.
type FS struct {
	*indexeddb.FS

	PreviousFID uint64
	Files       map[uint64]hackpadfs.File
}

// NewFS returns a new [FS]. Most code should use [Config] instead.
func NewFS() (*FS, error) {
	ifs, err := indexeddb.NewFS(context.TODO(), "fs", indexeddb.Options{})
	if err != nil {
		return nil, err
	}
	f := &FS{
		FS:    ifs,
		Files: map[uint64]hackpadfs.File{},
	}
	return f, nil
}

func (f *FS) Chmod(args []js.Value) (any, error) {
	return nil, f.FS.Chmod(args[0].String(), hackpadfs.FileMode(args[1].Int()))
}

func (f *FS) Chown(args []js.Value) (any, error) {
	return nil, hackpadfs.Chown(f.FS, args[0].String(), args[1].Int(), args[2].Int())
}

func (f *FS) Close(args []js.Value) (any, error) {
	return nil, nil // TODO
}

func (f *FS) Fchmod(args []js.Value) (any, error) {
	fd := uint64(args[0].Int())
	fl := f.Files[fd]
	if fl == nil {
		return nil, ErrBadFileNumber(fd)
	}
	return nil, hackpadfs.ChmodFile(fl, hackpadfs.FileMode(args[1].Int())) // TODO
}

func (f *FS) Fchown(args []js.Value) (any, error) {
	return f.Chown(args) // TODO
}

func (f *FS) Fstat(args []js.Value) (any, error) {
	return f.Stat(args) // TODO
}

func (f *FS) Fsync(args []js.Value) (any, error) {
	return nil, nil // TODO
}

func (f *FS) Ftruncate(args []js.Value) (any, error) {
	return f.Stat(args) // TODO
}

func (f *FS) Stat(args []js.Value) (any, error) {
	return f.FS.Stat(args[0].String())
}

// func (fs *FS) Truncate(args []js.Value) (any, error) {
// 	return hackpadfs.TruncateFile(args[0].String())
// }
