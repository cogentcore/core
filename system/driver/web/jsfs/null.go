// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Based on https://github.com/hack-pad/hackpad
// Licensed under the Apache 2.0 License

//go:build js

package jsfs

import (
	"io"
	"os"
	"time"

	"github.com/hack-pad/hackpadfs"
)

type NullFile struct {
	Nm string
}

func NewNullFile(name string) hackpadfs.File {
	return NullFile{Nm: name}
}

func (f NullFile) Close() error                                   { return nil }
func (f NullFile) Read(p []byte) (n int, err error)               { return 0, io.EOF }
func (f NullFile) ReadAt(p []byte, off int64) (n int, err error)  { return 0, io.EOF }
func (f NullFile) Seek(offset int64, whence int) (int64, error)   { return 0, nil }
func (f NullFile) Write(p []byte) (n int, err error)              { return len(p), nil }
func (f NullFile) WriteAt(p []byte, off int64) (n int, err error) { return len(p), nil }
func (f NullFile) Stat() (os.FileInfo, error)                     { return NullStat(f), nil }
func (f NullFile) Truncate(size int64) error                      { return nil }

type NullStat struct {
	Nm string
}

func (s NullStat) Name() string       { return s.Nm }
func (s NullStat) Size() int64        { return 0 }
func (s NullStat) Mode() os.FileMode  { return 0 }
func (s NullStat) ModTime() time.Time { return time.Time{} }
func (s NullStat) IsDir() bool        { return false }
func (s NullStat) Sys() interface{}   { return nil }
