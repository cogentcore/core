// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package yamls

import (
	"io"
	"io/fs"

	"goki.dev/grows"
	"gopkg.in/yaml.v3"
)

// Open reads object from the given filename using YAML encoding
func Open(v any, filename string) error {
	return grows.Open(v, filename, grows.NewDecoderFunc(yaml.NewDecoder))
}

// OpenFiles reads object from the given filenames using YAML encoding
func OpenFiles(v any, filenames []string) error {
	return grows.OpenFiles(v, filenames, grows.NewDecoderFunc(yaml.NewDecoder))
}

// OpenFS reads object from the given filename using YAML encoding,
// using the fs.FS filesystem (e.g., for embed files)
func OpenFS(v any, fsys fs.FS, filename string) error {
	return grows.OpenFS(v, fsys, filename, grows.NewDecoderFunc(yaml.NewDecoder))
}

// OpenFilesFS reads object from the given filenames using YAML encoding,
// using the fs.FS filesystem (e.g., for embed files)
func OpenFilesFS(v any, fsys fs.FS, filenames []string) error {
	return grows.OpenFilesFS(v, fsys, filenames, grows.NewDecoderFunc(yaml.NewDecoder))
}

// Read reads object encoding from the given reader,
// using YAML encoding
func Read(v any, reader io.Reader) error {
	return grows.Read(v, reader, grows.NewDecoderFunc(yaml.NewDecoder))
}

// ReadBytes reads object encoding from the given bytes,
// using YAML encoding
func ReadBytes(v any, data []byte) error {
	return grows.ReadBytes(v, data, grows.NewDecoderFunc(yaml.NewDecoder))
}

// Save writes object to the given filename using YAML encoding
func Save(v any, filename string) error {
	return grows.Save(v, filename, grows.NewEncoderFunc(yaml.NewEncoder))
}

// Write writes object encoding using YAML encoding
func Write(v any, writer io.Writer) error {
	return grows.Write(v, writer, grows.NewEncoderFunc(yaml.NewEncoder))
}

// WriteBytes writes object, returning bytes of the encoding,
// using YAML encoding
func WriteBytes(v any) ([]byte, error) {
	return grows.WriteBytes(v, grows.NewEncoderFunc(yaml.NewEncoder))
}
