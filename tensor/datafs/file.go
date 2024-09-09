// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package datafs

import (
	"bytes"
	"io"
	"io/fs"
)

// File represents a data item for reading, as an [fs.File].
// All io functionality is handled by [bytes.Reader].
type File struct {
	bytes.Reader
	Data       *Data
	dirEntries []fs.DirEntry
	dirsRead   int
}

func (f *File) Stat() (fs.FileInfo, error) {
	return f.Data, nil
}

func (f *File) Close() error {
	f.Reader.Reset(f.Data.Bytes())
	return nil
}

// DirFile represents a directory data item for reading, as fs.ReadDirFile.
type DirFile struct {
	File
	dirEntries []fs.DirEntry
	dirsRead   int
}

func (f *DirFile) ReadDir(n int) ([]fs.DirEntry, error) {
	if err := f.Data.mustDir("DirFile:ReadDir", ""); err != nil {
		return nil, err
	}
	if f.dirEntries == nil {
		f.dirEntries, _ = f.Data.ReadDir(".")
		f.dirsRead = 0
	}
	ne := len(f.dirEntries)
	if n <= 0 {
		if f.dirsRead >= ne {
			return nil, nil
		}
		re := f.dirEntries[f.dirsRead:]
		f.dirsRead = ne
		return re, nil
	}
	if f.dirsRead >= ne {
		return nil, io.EOF
	}
	mx := min(n+f.dirsRead, ne)
	re := f.dirEntries[f.dirsRead:mx]
	f.dirsRead = mx
	return re, nil
}

func (f *DirFile) Close() error {
	f.dirsRead = 0
	return nil
}
