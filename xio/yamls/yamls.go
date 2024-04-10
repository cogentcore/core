// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package yamls

import (
	"io"
	"io/fs"

	"cogentcore.org/core/xio"
	"gopkg.in/yaml.v3"
)

// Open reads the given object from the given filename using YAML encoding
func Open(v any, filename string) error {
	return xio.Open(v, filename, xio.NewDecoderFunc(yaml.NewDecoder))
}

// OpenFiles reads the given object from the given filenames using YAML encoding
func OpenFiles(v any, filenames []string) error {
	return xio.OpenFiles(v, filenames, xio.NewDecoderFunc(yaml.NewDecoder))
}

// OpenFS reads the given object from the given filename using YAML encoding,
// using the given [fs.FS] filesystem (e.g., for embed files)
func OpenFS(v any, fsys fs.FS, filename string) error {
	return xio.OpenFS(v, fsys, filename, xio.NewDecoderFunc(yaml.NewDecoder))
}

// OpenFilesFS reads the given object from the given filenames using YAML encoding,
// using the given [fs.FS] filesystem (e.g., for embed files)
func OpenFilesFS(v any, fsys fs.FS, filenames []string) error {
	return xio.OpenFilesFS(v, fsys, filenames, xio.NewDecoderFunc(yaml.NewDecoder))
}

// Read reads the given object from the given reader,
// using YAML encoding
func Read(v any, reader io.Reader) error {
	return xio.Read(v, reader, xio.NewDecoderFunc(yaml.NewDecoder))
}

// ReadBytes reads the given object from the given bytes,
// using YAML encoding
func ReadBytes(v any, data []byte) error {
	return xio.ReadBytes(v, data, xio.NewDecoderFunc(yaml.NewDecoder))
}

// Save writes the given object to the given filename using YAML encoding
func Save(v any, filename string) error {
	return xio.Save(v, filename, xio.NewEncoderFunc(yaml.NewEncoder))
}

// Write writes the given object using YAML encoding
func Write(v any, writer io.Writer) error {
	return xio.Write(v, writer, xio.NewEncoderFunc(yaml.NewEncoder))
}

// WriteBytes writes the given object, returning bytes of the encoding,
// using YAML encoding
func WriteBytes(v any) ([]byte, error) {
	return xio.WriteBytes(v, xio.NewEncoderFunc(yaml.NewEncoder))
}
