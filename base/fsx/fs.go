// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package fsx

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
)

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
