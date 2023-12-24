// Copyright (c) 2023, The Goki Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Based on https://github.com/hack-pad/hackpad
// Licensed under the Apache 2.0 License

//go:build js

// Package jsfs provides a Node.js style filesystem API in Go that can be used to allow os functions to work on wasm.
package fs

import (
	"context"
	"os"
	"sync"
	"sync/atomic"
	"syscall/js"
	"time"

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
	Mu          sync.Mutex
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
	delete(f.Files, uint64(args[0].Int())) // TODO
	return nil, nil
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
	s, err := fl.Stat()
	if err != nil {
		return nil, err
	}
	return JSStat(s), nil
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
	s, err := hackpadfs.LstatOrStat(f.FS, args[0].String())
	if err != nil {
		return nil, err
	}
	return JSStat(s), nil
}

func (f *FS) Mkdir(args []js.Value) (any, error) {
	return nil, hackpadfs.Mkdir(f.FS, args[0].String(), hackpadfs.FileMode(args[1].Int()))
}

func (f *FS) MkdirAll(args []js.Value) (any, error) {
	return nil, hackpadfs.MkdirAll(f.FS, args[0].String(), hackpadfs.FileMode(args[1].Int()))
}

func (f *FS) Open(args []js.Value) (any, error) {
	f.Mu.Lock()
	defer f.Mu.Unlock()

	fid := atomic.AddUint64((*uint64)(&f.PreviousFID), 1) - 1
	fl, err := f.NewFile(args[0].String(), args[1].Int(), hackpadfs.FileMode(args[2].Int()))
	if err != nil {
		return nil, err
	}
	f.Files[fid] = fl

	return fid, nil
}

func (f *FS) NewFile(absPath string, flags int, mode os.FileMode) (hackpadfs.File, error) {
	switch absPath {
	case "dev/null":
		return NewNullFile("dev/null"), nil
	case "dev/stdin":
		return NewNullFile("dev/stdin"), nil // TODO: can this be mocked?
	case "dev/stdout":
		return Stdout, nil
	case "dev/stderr":
		return Stderr, nil
	}
	return hackpadfs.OpenFile(f.FS, absPath, flags, mode)
}

func (f *FS) Readdir(args []js.Value) (any, error) {
	des, err := hackpadfs.ReadDir(f.FS, args[0].String())
	if err != nil {
		return nil, err
	}
	names := make([]any, len(des))
	for i, de := range des {
		names[i] = de.Name()
	}
	return names, nil
}

func (f *FS) Readlink(args []js.Value) (any, error) {
	return nil, nil // TODO
}

func (f *FS) Rename(args []js.Value) (any, error) {
	return nil, hackpadfs.Rename(f.FS, args[0].String(), args[1].String())
}

func (f *FS) Rmdir(args []js.Value) (any, error) {
	info, err := f.Stat(args)
	if err != nil {
		return nil, err
	}
	if !js.ValueOf(info).Call("isDirectory").Bool() {
		return nil, ErrNotDir
	}
	return nil, hackpadfs.Remove(f.FS, args[0].String())
}

func (f *FS) Stat(args []js.Value) (any, error) {
	s, err := hackpadfs.Stat(f.FS, args[0].String())
	if err != nil {
		return nil, err
	}
	return JSStat(s), nil
}

func (f *FS) Symlink(args []js.Value) (any, error) {
	return nil, nil // TODO
}

func (f *FS) Unlink(args []js.Value) (any, error) {
	info, err := f.Stat(args)
	if err != nil {
		return nil, err
	}
	if js.ValueOf(info).Call("isDirectory").Bool() {
		return nil, os.ErrPermission
	}
	return nil, hackpadfs.Remove(f.FS, args[0].String())
}

func (f *FS) Utimes(args []js.Value) (any, error) {
	path := args[0].String()
	atime := time.Unix(int64(args[1].Int()), 0)
	mtime := time.Unix(int64(args[2].Int()), 0)

	return nil, hackpadfs.Chtimes(f.FS, path, atime, mtime)
}

func (f *FS) Truncate(args []js.Value) (any, error) {
	return nil, nil // TODO
}
