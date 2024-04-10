// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package jsons

import (
	"encoding/json"
	"io"
	"io/fs"

	"cogentcore.org/core/xio"
)

// Open reads the given object from the given filename using JSON encoding
func Open(v any, filename string) error {
	return xio.Open(v, filename, xio.NewDecoderFunc(json.NewDecoder))
}

// OpenFiles reads the given object from the given filenames using JSON encoding
func OpenFiles(v any, filenames []string) error {
	return xio.OpenFiles(v, filenames, xio.NewDecoderFunc(json.NewDecoder))
}

// OpenFS reads the given object from the given filename using JSON encoding,
// using the given [fs.FS] filesystem (e.g., for embed files)
func OpenFS(v any, fsys fs.FS, filename string) error {
	return xio.OpenFS(v, fsys, filename, xio.NewDecoderFunc(json.NewDecoder))
}

// OpenFilesFS reads the given object from the given filenames using JSON encoding,
// using the given [fs.FS] filesystem (e.g., for embed files)
func OpenFilesFS(v any, fsys fs.FS, filenames []string) error {
	return xio.OpenFilesFS(v, fsys, filenames, xio.NewDecoderFunc(json.NewDecoder))
}

// Read reads the given object from the given reader,
// using JSON encoding
func Read(v any, reader io.Reader) error {
	return xio.Read(v, reader, xio.NewDecoderFunc(json.NewDecoder))
}

// ReadBytes reads the given object from the given bytes,
// using JSON encoding
func ReadBytes(v any, data []byte) error {
	return xio.ReadBytes(v, data, xio.NewDecoderFunc(json.NewDecoder))
}

// Save writes the given object to the given filename using JSON encoding
func Save(v any, filename string) error {
	return xio.Save(v, filename, xio.NewEncoderFunc(json.NewEncoder))
}

// Write writes the given object using JSON encoding
func Write(v any, writer io.Writer) error {
	return xio.Write(v, writer, xio.NewEncoderFunc(json.NewEncoder))
}

// WriteBytes writes the given object, returning bytes of the encoding,
// using JSON encoding
func WriteBytes(v any) ([]byte, error) {
	return xio.WriteBytes(v, xio.NewEncoderFunc(json.NewEncoder))
}

// IndentEncoderFunc is a [xio.EncoderFunc] that sets indentation
var IndentEncoderFunc = func(w io.Writer) xio.Encoder {
	e := json.NewEncoder(w)
	e.SetIndent("", "\t")
	return e
}

// SaveIndent writes the given object to the given filename using JSON encoding, with indentation
func SaveIndent(v any, filename string) error {
	return xio.Save(v, filename, IndentEncoderFunc)
}

// WriteIndent writes the given object using JSON encoding, with indentation
func WriteIndent(v any, writer io.Writer) error {
	return xio.Write(v, writer, IndentEncoderFunc)
}

// WriteBytesIndent writes the given object, returning bytes of the encoding,
// using JSON encoding, with indentation
func WriteBytesIndent(v any) ([]byte, error) {
	return xio.WriteBytes(v, IndentEncoderFunc)
}
