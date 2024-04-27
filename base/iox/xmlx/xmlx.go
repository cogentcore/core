// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package xmlx

import (
	"encoding/xml"
	"io"
	"io/fs"

	"cogentcore.org/core/base/iox"
)

// Open reads the given object from the given filename using XML encoding
func Open(v any, filename string) error {
	return iox.Open(v, filename, iox.NewDecoderFunc(xml.NewDecoder))
}

// OpenFiles reads the given object from the given filenames using XML encoding
func OpenFiles(v any, filenames []string) error {
	return iox.OpenFiles(v, filenames, iox.NewDecoderFunc(xml.NewDecoder))
}

// OpenFS reads the given object from the given filename using XML encoding,
// using the given [fs.FS] filesystem (e.g., for embed files)
func OpenFS(v any, fsys fs.FS, filename string) error {
	return iox.OpenFS(v, fsys, filename, iox.NewDecoderFunc(xml.NewDecoder))
}

// OpenFilesFS reads the given object from the given filenames using XML encoding,
// using the given [fs.FS] filesystem (e.g., for embed files)
func OpenFilesFS(v any, fsys fs.FS, filenames []string) error {
	return iox.OpenFilesFS(v, fsys, filenames, iox.NewDecoderFunc(xml.NewDecoder))
}

// Read reads the given object from the given reader,
// using XML encoding
func Read(v any, reader io.Reader) error {
	return iox.Read(v, reader, iox.NewDecoderFunc(xml.NewDecoder))
}

// ReadBytes reads the given object from the given bytes,
// using XML encoding
func ReadBytes(v any, data []byte) error {
	return iox.ReadBytes(v, data, iox.NewDecoderFunc(xml.NewDecoder))
}

// Save writes the given object to the given filename using XML encoding
func Save(v any, filename string) error {
	return iox.Save(v, filename, iox.NewEncoderFunc(xml.NewEncoder))
}

// Write writes the given object using XML encoding
func Write(v any, writer io.Writer) error {
	return iox.Write(v, writer, iox.NewEncoderFunc(xml.NewEncoder))
}

// WriteBytes writes the given object, returning bytes of the encoding,
// using XML encoding
func WriteBytes(v any) ([]byte, error) {
	return iox.WriteBytes(v, iox.NewEncoderFunc(xml.NewEncoder))
}

// IndentEncoderFunc is a [iox.EncoderFunc] that sets indentation
var IndentEncoderFunc = func(w io.Writer) iox.Encoder {
	e := xml.NewEncoder(w)
	e.Indent("", "\t")
	return e
}

// SaveIndent writes the given object to the given filename using XML encoding, with indentation
func SaveIndent(v any, filename string) error {
	return iox.Save(v, filename, IndentEncoderFunc)
}

// WriteIndent writes the given object using XML encoding, with indentation
func WriteIndent(v any, writer io.Writer) error {
	return iox.Write(v, writer, IndentEncoderFunc)
}

// WriteBytesIndent writes the given object, returning bytes of the encoding,
// using XML encoding, with indentation
func WriteBytesIndent(v any) ([]byte, error) {
	return iox.WriteBytes(v, IndentEncoderFunc)
}
