// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package xmls

import (
	"encoding/xml"
	"io"
	"io/fs"

	"goki.dev/grows"
)

// Open reads the given object from the given filename using XML encoding
func Open(v any, filename string) error {
	return grows.Open(v, filename, grows.NewDecoderFunc(xml.NewDecoder))
}

// OpenFiles reads the given object from the given filenames using XML encoding
func OpenFiles(v any, filenames []string) error {
	return grows.OpenFiles(v, filenames, grows.NewDecoderFunc(xml.NewDecoder))
}

// OpenFS reads the given object from the given filename using XML encoding,
// using the given [fs.FS] filesystem (e.g., for embed files)
func OpenFS(v any, fsys fs.FS, filename string) error {
	return grows.OpenFS(v, fsys, filename, grows.NewDecoderFunc(xml.NewDecoder))
}

// OpenFilesFS reads the given object from the given filenames using XML encoding,
// using the given [fs.FS] filesystem (e.g., for embed files)
func OpenFilesFS(v any, fsys fs.FS, filenames []string) error {
	return grows.OpenFilesFS(v, fsys, filenames, grows.NewDecoderFunc(xml.NewDecoder))
}

// Read reads the given object from the given reader,
// using XML encoding
func Read(v any, reader io.Reader) error {
	return grows.Read(v, reader, grows.NewDecoderFunc(xml.NewDecoder))
}

// ReadBytes reads the given object from the given bytes,
// using XML encoding
func ReadBytes(v any, data []byte) error {
	return grows.ReadBytes(v, data, grows.NewDecoderFunc(xml.NewDecoder))
}

// Save writes the given object to the given filename using XML encoding
func Save(v any, filename string) error {
	return grows.Save(v, filename, grows.NewEncoderFunc(xml.NewEncoder))
}

// Write writes the given object using XML encoding
func Write(v any, writer io.Writer) error {
	return grows.Write(v, writer, grows.NewEncoderFunc(xml.NewEncoder))
}

// WriteBytes writes the given object, returning bytes of the encoding,
// using XML encoding
func WriteBytes(v any) ([]byte, error) {
	return grows.WriteBytes(v, grows.NewEncoderFunc(xml.NewEncoder))
}

// IndentEncoderFunc is a [grows.EncoderFunc] that sets indentation
var IndentEncoderFunc = func(w io.Writer) grows.Encoder {
	e := xml.NewEncoder(w)
	e.Indent("", "\t")
	return e
}

// SaveIndent writes the given object to the given filename using XML encoding, with indentation
func SaveIndent(v any, filename string) error {
	return grows.Save(v, filename, IndentEncoderFunc)
}

// WriteIndent writes the given object using XML encoding, with indentation
func WriteIndent(v any, writer io.Writer) error {
	return grows.Write(v, writer, IndentEncoderFunc)
}

// WriteBytesIndent writes the given object, returning bytes of the encoding,
// using XML encoding, with indentation
func WriteBytesIndent(v any) ([]byte, error) {
	return grows.WriteBytes(v, IndentEncoderFunc)
}
