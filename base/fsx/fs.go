// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package fsx

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"cogentcore.org/core/base/errors"
)

// Sub returns [fs.Sub] with any error automatically logged
// for cases where the directory is hardcoded and there is
// no chance of error.
func Sub(fsys fs.FS, dir string) fs.FS {
	return errors.Log1(fs.Sub(fsys, dir))
}

// DirFS returns the directory part of given file path as an os.DirFS
// and the filename as a string.  These can then be used to access the file
// using the FS-based interface, consistent with embed and other use-cases.
func DirFS(fpath string) (fs.FS, string, error) {
	fabs, err := filepath.Abs(fpath)
	if err != nil {
		return nil, "", err
	}
	dir, fname := filepath.Split(fabs)
	dfs := os.DirFS(dir)
	return dfs, fname, nil
}

// FileExistsFS checks whether given file exists, returning true if so,
// false if not, and error if there is an error in accessing the file.
func FileExistsFS(fsys fs.FS, filePath string) (bool, error) {
	if fsys, ok := fsys.(fs.StatFS); ok {
		fileInfo, err := fsys.Stat(filePath)
		if err == nil {
			return !fileInfo.IsDir(), nil
		}
		if errors.Is(err, fs.ErrNotExist) {
			return false, nil
		}
		return false, err
	}
	fp, err := fsys.Open(filePath)
	if err == nil {
		fp.Close()
		return true, nil
	}
	if errors.Is(err, fs.ErrNotExist) {
		return false, nil
	}
	return false, err
}

// SplitRootPathFS returns a split of the given FS path (only / path separators)
// into the root element and everything after that point.
// Examples:
//   - "/a/b/c" returns "/", "a/b/c"
//   - "a/b/c" returns "a", "b/c" (note removal of intervening "/")
//   - "a" returns "a", ""
//   - "a/" returns "a", "" (note removal of trailing "/")
func SplitRootPathFS(path string) (root, rest string) {
	pi := strings.IndexByte(path, '/')
	if pi < 0 {
		return path, ""
	}
	if pi == 0 {
		return "/", path[1:]
	}
	if pi < len(path)-1 {
		return path[:pi], path[pi+1:]
	}
	return path[:pi], ""
}
