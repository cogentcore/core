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
	"github.com/pkg/errors"
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

// GetFile fetches the file specified by the file descriptor that is the first of the given arguments.
func (f *FS) GetFile(args []js.Value) (hackpadfs.File, error) {
	fd := uint64(args[0].Int())
	fl := f.Files[fd]
	if fl == nil {
		return nil, ErrBadFileNumber(fd)
	}
	return fl, nil
}

func (f *FS) Chmod(args []js.Value) (any, error) {
	return nil, hackpadfs.Chmod(f.FS, args[0].String(), hackpadfs.FileMode(args[1].Int()))
}

func (f *FS) Chown(args []js.Value) (any, error) {
	return nil, hackpadfs.Chown(f.FS, args[0].String(), args[1].Int(), args[2].Int())
}

func (f *FS) Close(args []js.Value) (any, error) {
	return nil, nil // TODO
}

func (f *FS) Fchmod(args []js.Value) (any, error) {
	fl, err := f.GetFile(args)
	if err != nil {
		return nil, err
	}
	return nil, hackpadfs.ChmodFile(fl, hackpadfs.FileMode(args[1].Int()))
}

func (f *FS) Fchown(args []js.Value) (any, error) {
	fl, err := f.GetFile(args)
	if err != nil {
		return nil, err
	}
	return nil, hackpadfs.ChownFile(fl, args[1].Int(), args[2].Int())
}

func (f *FS) Fstat(args []js.Value) (any, error) {
	fl, err := f.GetFile(args)
	if err != nil {
		return nil, err
	}
	return fl.Stat()
}

func (f *FS) Fsync(args []js.Value) (any, error) {
	fl, err := f.GetFile(args)
	if err != nil {
		return nil, err
	}
	err = hackpadfs.SyncFile(fl)
	if errors.Is(err, hackpadfs.ErrNotImplemented) {
		err = nil // not all FS implement Sync(), so fall back to a no-op
	}
	return nil, err
}

func (f *FS) Ftruncate(args []js.Value) (any, error) {
	fl, err := f.GetFile(args)
	if err != nil {
		return nil, err
	}
	return nil, hackpadfs.TruncateFile(fl, int64(args[1].Int()))
}

func (f *FS) Lchown(args []js.Value) (any, error) {
	return nil, hackpadfs.Chown(f.FS, args[0].String(), args[1].Int(), args[2].Int()) // TODO
}

func (f *FS) Link(args []js.Value) (any, error) {
	return nil, nil // TODO
}

func (f *FS) Lstat(args []js.Value) (any, error) {
	return hackpadfs.LstatOrStat(f.FS, args[0].String())
}

func (f *FS) Mkdir(args []js.Value) (any, error) {
	return nil, hackpadfs.Mkdir(f.FS, args[0].String(), hackpadfs.FileMode(args[1].Int()))
}

func (f *FS) MkdirAll(args []js.Value) (any, error) {
	return nil, hackpadfs.MkdirAll(f.FS, args[0].String(), hackpadfs.FileMode(args[1].Int()))
}

func (f *FS) Stat(args []js.Value) (any, error) {
	return hackpadfs.Stat(f.FS, args[0].String())
}

func (f *FS) Truncate(args []js.Value) (any, error) {
	return nil, nil // TODO
}
