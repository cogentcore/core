// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package tomlx

import (
	"errors"
	"io"
	"io/fs"

	"cogentcore.org/core/base/fsx"
	"cogentcore.org/core/base/iox"
	"github.com/pelletier/go-toml/v2"
)

// NewDecoder returns a new [iox.Decoder]
func NewDecoder(r io.Reader) iox.Decoder { return toml.NewDecoder(r) }

// Open reads the given object from the given filename using TOML encoding
func Open(v any, filename string) error {
	return iox.Open(v, filename, NewDecoder)
}

// OpenFiles reads the given object from the given filenames using TOML encoding
func OpenFiles(v any, filenames ...string) error {
	return iox.OpenFiles(v, filenames, NewDecoder)
}

// OpenFS reads the given object from the given filename using TOML encoding,
// using the given [fs.FS] filesystem (e.g., for embed files)
func OpenFS(v any, fsys fs.FS, filename string) error {
	return iox.OpenFS(v, fsys, filename, NewDecoder)
}

// OpenFilesFS reads the given object from the given filenames using TOML encoding,
// using the given [fs.FS] filesystem (e.g., for embed files)
func OpenFilesFS(v any, fsys fs.FS, filenames ...string) error {
	return iox.OpenFilesFS(v, fsys, filenames, NewDecoder)
}

// Read reads the given object from the given reader,
// using TOML encoding
func Read(v any, reader io.Reader) error {
	return iox.Read(v, reader, NewDecoder)
}

// ReadBytes reads the given object from the given bytes,
// using TOML encoding
func ReadBytes(v any, data []byte) error {
	return iox.ReadBytes(v, data, NewDecoder)
}

// NewEncoder returns a new [iox.Encoder]
func NewEncoder(w io.Writer) iox.Encoder {
	return toml.NewEncoder(w).SetIndentTables(true).SetArraysMultiline(true)
}

// Save writes the given object to the given filename using TOML encoding
func Save(v any, filename string) error {
	return iox.Save(v, filename, NewEncoder)
}

// Write writes the given object using TOML encoding
func Write(v any, writer io.Writer) error {
	return iox.Write(v, writer, NewEncoder)
}

// WriteBytes writes the given object, returning bytes of the encoding,
// using TOML encoding
func WriteBytes(v any) ([]byte, error) {
	return iox.WriteBytes(v, NewEncoder)
}

// OpenFromPaths reads the given object from the given TOML file,
// looking on paths for the file.
func OpenFromPaths(v any, file string, paths ...string) error {
	filenames := fsx.FindFilesOnPaths(paths, file)
	if len(filenames) == 0 {
		return errors.New("OpenFromPaths: no files found")
	}
	return Open(v, filenames[0])
}
