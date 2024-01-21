// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Based on https://github.com/laher/mergefs
// Copyright (c) 2021 Amir Laher under the MIT License

// Package mergefs provides support for merging multiple filesystems together.
package mergefs

import (
	"errors"
	"io/fs"
	"os"
)

// Merge merges the given filesystems together,
func Merge(filesystems ...fs.FS) fs.FS {
	return MergedFS{filesystems: filesystems}
}

// MergedFS combines filesystems. Each filesystem can serve different paths.
// The first FS takes precedence
type MergedFS struct {
	filesystems []fs.FS
}

// Open opens the named file.
func (mfs MergedFS) Open(name string) (fs.File, error) {
	for _, fs := range mfs.filesystems {
		file, err := fs.Open(name)
		if err == nil { // TODO should we return early when it's not an os.ErrNotExist? Should we offer options to decide this behaviour?
			return file, nil
		}
	}
	return nil, os.ErrNotExist
}

// ReadDir reads from the directory, and produces a DirEntry array of different
// directories.
//
// It iterates through all different filesystems that exist in the mfs MergeFS
// filesystem slice and it identifies overlapping directories that exist in different
// filesystems
func (mfs MergedFS) ReadDir(name string) ([]fs.DirEntry, error) {
	dirsMap := make(map[string]fs.DirEntry)
	notExistCount := 0
	for _, filesystem := range mfs.filesystems {
		dir, err := fs.ReadDir(filesystem, name)
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				notExistCount++
				continue
			}
			return nil, err
		}
		for _, v := range dir {
			if _, ok := dirsMap[v.Name()]; !ok {
				dirsMap[v.Name()] = v
			}
		}
		continue
	}
	if len(mfs.filesystems) == notExistCount {
		return nil, fs.ErrNotExist
	}
	dirs := make([]fs.DirEntry, 0, len(dirsMap))

	for _, value := range dirsMap {
		dirs = append(dirs, value)
	}

	return dirs, nil
}
