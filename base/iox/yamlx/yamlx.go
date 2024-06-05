// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package yamlx

import (
	"io"
	"io/fs"

	"cogentcore.org/core/base/iox"
	"gopkg.in/yaml.v3"
)

// Open reads the given object from the given filename using YAML encoding
func Open(v any, filename string) error {
	return iox.Open(v, filename, iox.NewDecoderFunc(yaml.NewDecoder))
}

// OpenFiles reads the given object from the given filenames using YAML encoding
func OpenFiles(v any, filenames ...string) error {
	return iox.OpenFiles(v, filenames, iox.NewDecoderFunc(yaml.NewDecoder))
}

// OpenFS reads the given object from the given filename using YAML encoding,
// using the given [fs.FS] filesystem (e.g., for embed files)
func OpenFS(v any, fsys fs.FS, filename string) error {
	return iox.OpenFS(v, fsys, filename, iox.NewDecoderFunc(yaml.NewDecoder))
}

// OpenFilesFS reads the given object from the given filenames using YAML encoding,
// using the given [fs.FS] filesystem (e.g., for embed files)
func OpenFilesFS(v any, fsys fs.FS, filenames ...string) error {
	return iox.OpenFilesFS(v, fsys, filenames, iox.NewDecoderFunc(yaml.NewDecoder))
}

// Read reads the given object from the given reader,
// using YAML encoding
func Read(v any, reader io.Reader) error {
	return iox.Read(v, reader, iox.NewDecoderFunc(yaml.NewDecoder))
}

// ReadBytes reads the given object from the given bytes,
// using YAML encoding
func ReadBytes(v any, data []byte) error {
	return iox.ReadBytes(v, data, iox.NewDecoderFunc(yaml.NewDecoder))
}

// Save writes the given object to the given filename using YAML encoding
func Save(v any, filename string) error {
	return iox.Save(v, filename, iox.NewEncoderFunc(yaml.NewEncoder))
}

// Write writes the given object using YAML encoding
func Write(v any, writer io.Writer) error {
	return iox.Write(v, writer, iox.NewEncoderFunc(yaml.NewEncoder))
}

// WriteBytes writes the given object, returning bytes of the encoding,
// using YAML encoding
func WriteBytes(v any) ([]byte, error) {
	return iox.WriteBytes(v, iox.NewEncoderFunc(yaml.NewEncoder))
}
