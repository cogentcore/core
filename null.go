// Copyright (c) 2023, The Goki Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build js

package fs

import (
	"io"
	"os"
	"time"

	"github.com/hack-pad/hackpadfs"
)

type NullFile struct {
	name string
}

func NewNullFile(name string) hackpadfs.File {
	return NullFile{name: name}
}

func (f NullFile) Close() error                                   { return nil }
func (f NullFile) Read(p []byte) (n int, err error)               { return 0, io.EOF }
func (f NullFile) ReadAt(p []byte, off int64) (n int, err error)  { return 0, io.EOF }
func (f NullFile) Seek(offset int64, whence int) (int64, error)   { return 0, nil }
func (f NullFile) Write(p []byte) (n int, err error)              { return len(p), nil }
func (f NullFile) WriteAt(p []byte, off int64) (n int, err error) { return len(p), nil }
func (f NullFile) Stat() (os.FileInfo, error)                     { return NullStat{f}, nil }
func (f NullFile) Truncate(size int64) error                      { return nil }

type NullStat struct {
	F NullFile
}

func (s NullStat) Name() string       { return s.F.name }
func (s NullStat) Size() int64        { return 0 }
func (s NullStat) Mode() os.FileMode  { return 0 }
func (s NullStat) ModTime() time.Time { return time.Time{} }
func (s NullStat) IsDir() bool        { return false }
func (s NullStat) Sys() interface{}   { return nil }
