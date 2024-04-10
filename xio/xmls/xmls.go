// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package xmls

import (
	"encoding/xml"
	"io"
	"io/fs"

	"cogentcore.org/core/xio"
)

// Open reads the given object from the given filename using XML encoding
func Open(v any, filename string) error {
	return xio.Open(v, filename, xio.NewDecoderFunc(xml.NewDecoder))
}

// OpenFiles reads the given object from the given filenames using XML encoding
func OpenFiles(v any, filenames []string) error {
	return xio.OpenFiles(v, filenames, xio.NewDecoderFunc(xml.NewDecoder))
}

// OpenFS reads the given object from the given filename using XML encoding,
// using the given [fs.FS] filesystem (e.g., for embed files)
func OpenFS(v any, fsys fs.FS, filename string) error {
	return xio.OpenFS(v, fsys, filename, xio.NewDecoderFunc(xml.NewDecoder))
}

// OpenFilesFS reads the given object from the given filenames using XML encoding,
// using the given [fs.FS] filesystem (e.g., for embed files)
func OpenFilesFS(v any, fsys fs.FS, filenames []string) error {
	return xio.OpenFilesFS(v, fsys, filenames, xio.NewDecoderFunc(xml.NewDecoder))
}

// Read reads the given object from the given reader,
// using XML encoding
func Read(v any, reader io.Reader) error {
	return xio.Read(v, reader, xio.NewDecoderFunc(xml.NewDecoder))
}

// ReadBytes reads the given object from the given bytes,
// using XML encoding
func ReadBytes(v any, data []byte) error {
	return xio.ReadBytes(v, data, xio.NewDecoderFunc(xml.NewDecoder))
}

// Save writes the given object to the given filename using XML encoding
func Save(v any, filename string) error {
	return xio.Save(v, filename, xio.NewEncoderFunc(xml.NewEncoder))
}

// Write writes the given object using XML encoding
func Write(v any, writer io.Writer) error {
	return xio.Write(v, writer, xio.NewEncoderFunc(xml.NewEncoder))
}

// WriteBytes writes the given object, returning bytes of the encoding,
// using XML encoding
func WriteBytes(v any) ([]byte, error) {
	return xio.WriteBytes(v, xio.NewEncoderFunc(xml.NewEncoder))
}

// IndentEncoderFunc is a [xio.EncoderFunc] that sets indentation
var IndentEncoderFunc = func(w io.Writer) xio.Encoder {
	e := xml.NewEncoder(w)
	e.Indent("", "\t")
	return e
}

// SaveIndent writes the given object to the given filename using XML encoding, with indentation
func SaveIndent(v any, filename string) error {
	return xio.Save(v, filename, IndentEncoderFunc)
}

// WriteIndent writes the given object using XML encoding, with indentation
func WriteIndent(v any, writer io.Writer) error {
	return xio.Write(v, writer, IndentEncoderFunc)
}

// WriteBytesIndent writes the given object, returning bytes of the encoding,
// using XML encoding, with indentation
func WriteBytesIndent(v any) ([]byte, error) {
	return xio.WriteBytes(v, IndentEncoderFunc)
}
