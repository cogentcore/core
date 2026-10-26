// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package basesymbols

import (
	"archive/zip"
	"bytes"
	_ "embed"
	"io/fs"
)

//go:generate go run genstdsrc.go

// stdsrcZip has the sources of the generic standard library packages, such
// as slices, which interpreted code imports from source, since they cannot
// be bound as values (see genstdsrc.go).
//
//go:embed stdsrc.zip
var stdsrcZip []byte

// StdSource returns the file system of the sources of [stdsrcZip], by
// package import path, which an interpreter without the standard library
// sources of mvm (which the mvmnosrc build tag leaves out to keep apps
// small) is given with SetStdlibFS.
func StdSource() fs.FS {
	r, err := zip.NewReader(bytes.NewReader(stdsrcZip), int64(len(stdsrcZip)))
	if err != nil {
		panic(err)
	}
	return r
}
