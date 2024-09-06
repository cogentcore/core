// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package datafs

import (
	"bytes"
	"io/fs"
)

// File represents a data item for reading, as fs.File.
// All io functionality is handled by bytes.Reader
type File struct {
	bytes.Reader
	Data *Data
}

func (f *File) Stat() (fs.FileInfo, error) {
	return f.Data, nil
}

func (f *File) Close() error {
	f.Reader.Reset(f.Data.Bytes())
	return nil
}
