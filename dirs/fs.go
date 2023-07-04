// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package dirs

import (
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
