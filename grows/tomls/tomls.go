// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package tomls

import (
	"fmt"
	"io"
	"io/fs"

	"cogentcore.org/core/glop/dirs"
	"cogentcore.org/core/grows"
	"github.com/pelletier/go-toml/v2"
)

// NewDecoder returns a new [grows.Decoder]
func NewDecoder(r io.Reader) grows.Decoder { return toml.NewDecoder(r) }

// Open reads the given object from the given filename using TOML encoding
func Open(v any, filename string) error {
	return grows.Open(v, filename, NewDecoder)
}

// OpenFiles reads the given object from the given filenames using TOML encoding
func OpenFiles(v any, filenames []string) error {
	return grows.OpenFiles(v, filenames, NewDecoder)
}

// OpenFS reads the given object from the given filename using TOML encoding,
// using the given [fs.FS] filesystem (e.g., for embed files)
func OpenFS(v any, fsys fs.FS, filename string) error {
	return grows.OpenFS(v, fsys, filename, NewDecoder)
}

// OpenFilesFS reads the given object from the given filenames using TOML encoding,
// using the given [fs.FS] filesystem (e.g., for embed files)
func OpenFilesFS(v any, fsys fs.FS, filenames []string) error {
	return grows.OpenFilesFS(v, fsys, filenames, NewDecoder)
}

// Read reads the given object from the given reader,
// using TOML encoding
func Read(v any, reader io.Reader) error {
	return grows.Read(v, reader, NewDecoder)
}

// ReadBytes reads the given object from the given bytes,
// using TOML encoding
func ReadBytes(v any, data []byte) error {
	return grows.ReadBytes(v, data, NewDecoder)
}

// NewEncoder returns a new [grows.Encoder]
func NewEncoder(w io.Writer) grows.Encoder {
	return toml.NewEncoder(w).SetIndentTables(true).SetArraysMultiline(true)
}

// Save writes the given object to the given filename using TOML encoding
func Save(v any, filename string) error {
	return grows.Save(v, filename, NewEncoder)
}

// Write writes the given object using TOML encoding
func Write(v any, writer io.Writer) error {
	return grows.Write(v, writer, NewEncoder)
}

// WriteBytes writes the given object, returning bytes of the encoding,
// using TOML encoding
func WriteBytes(v any) ([]byte, error) {
	return grows.WriteBytes(v, NewEncoder)
}

// OpenFromPaths reads the given object from the given TOML file,
// looking on paths for the file.
func OpenFromPaths(v any, file string, paths []string) error {
	filenames := dirs.FindFilesOnPaths(paths, file)
	if len(filenames) == 0 {
		return fmt.Errorf("OpenFromPaths: no files found")
	}
	return Open(v, filenames[0])
}
