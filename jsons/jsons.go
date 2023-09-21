// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package jsons

import (
	"encoding/json"
	"io"
	"io/fs"

	"goki.dev/grows"
)

// Open reads object from the given filename using JSON encoding
func Open(v any, filename string) error {
	return grows.Open(v, filename, grows.NewDecoderFunc(json.NewDecoder))
}

// OpenFiles reads object from the given filenames using JSON encoding
func OpenFiles(v any, filenames []string) error {
	return grows.OpenFiles(v, filenames, grows.NewDecoderFunc(json.NewDecoder))
}

// OpenFS reads object from the given filename using JSON encoding,
// using the fs.FS filesystem (e.g., for embed files)
func OpenFS(v any, fsys fs.FS, filename string) error {
	return grows.OpenFS(v, fsys, filename, grows.NewDecoderFunc(json.NewDecoder))
}

// OpenFilesFS reads object from the given filenames using JSON encoding,
// using the fs.FS filesystem (e.g., for embed files)
func OpenFilesFS(v any, fsys fs.FS, filenames []string) error {
	return grows.OpenFilesFS(v, fsys, filenames, grows.NewDecoderFunc(json.NewDecoder))
}

// Read reads object encoding from the given reader,
// using JSON encoding
func Read(v any, reader io.Reader) error {
	return grows.Read(v, reader, grows.NewDecoderFunc(json.NewDecoder))
}

// ReadBytes reads object encoding from the given bytes,
// using JSON encoding
func ReadBytes(v any, data []byte) error {
	return grows.ReadBytes(v, data, grows.NewDecoderFunc(json.NewDecoder))
}

// Save writes object to the given filename using JSON encoding
func Save(v any, filename string) error {
	return grows.Save(v, filename, grows.NewEncoderFunc(json.NewEncoder))
}

// Write writes object encoding using JSON encoding
func Write(v any, writer io.Writer) error {
	return grows.Write(v, writer, grows.NewEncoderFunc(json.NewEncoder))
}

// WriteBytes writes object, returning bytes of the encoding,
// using JSON encoding
func WriteBytes(v any) ([]byte, error) {
	return grows.WriteBytes(v, grows.NewEncoderFunc(json.NewEncoder))
}

// IndentEncoderFunc is a [grows.EncoderFunc] that sets indentation
var IndentEncoderFunc = func(w io.Writer) grows.Encoder {
	e := json.NewEncoder(w)
	e.SetIndent("", "\t")
	return e
}

// SaveIndent writes object to the given filename using JSON encoding, with indentation
func SaveIndent(v any, filename string) error {
	return grows.Save(v, filename, IndentEncoderFunc)
}

// WriteIndent writes object encoding using JSON encoding, with indentation
func WriteIndent(v any, writer io.Writer) error {
	return grows.Write(v, writer, IndentEncoderFunc)
}

// WriteBytesIndent writes object, returning bytes of the encoding,
// using JSON encoding, with indentation
func WriteBytesIndent(v any) ([]byte, error) {
	return grows.WriteBytes(v, IndentEncoderFunc)
}
