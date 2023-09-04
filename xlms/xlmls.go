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

// Open reads object from the given filename using XML encoding
func Open(v any, filename string) error {
	return grows.Open(v, filename, grows.NewDecoderFunc(xml.NewDecoder))
}

// OpenFS reads object from the given filename using XML encoding,
// using the fs.FS filesystem (e.g., for embed files)
func OpenFS(v any, fsys fs.FS, filename string) error {
	return grows.OpenFS(v, fsys, filename, grows.NewDecoderFunc(xml.NewDecoder))
}

// Read reads object encoding from the given reader,
// using XML encoding
func Read(v any, reader io.Reader) error {
	return grows.Read(v, reader, grows.NewDecoderFunc(xml.NewDecoder))
}

// ReadBytes reads object encoding from the given bytes,
// using XML encoding
func ReadBytes(v any, data []byte) error {
	return grows.ReadBytes(v, data, grows.NewDecoderFunc(xml.NewDecoder))
}

// Save writes object to the given filename using XML encoding
func Save(v any, filename string) error {
	return grows.Save(v, filename, grows.NewEncoderFunc(xml.NewEncoder))
}

// Write writes object encoding using XML encoding
func Write(v any, writer io.Writer) error {
	return grows.Write(v, writer, grows.NewEncoderFunc(xml.NewEncoder))
}

// WriteBytes writes object, returning bytes of the encoding,
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

// SaveIndent writes object to the given filename using XML encoding, with indentation
func SaveIndent(v any, filename string) error {
	return grows.Save(v, filename, IndentEncoderFunc)
}

// WriteIndent writes object encoding using XML encoding, with indentation
func WriteIndent(v any, writer io.Writer) error {
	return grows.Write(v, writer, IndentEncoderFunc)
}

// WriteBytesIndent writes object, returning bytes of the encoding,
// using XML encoding, with indentation
func WriteBytesIndent(v any) ([]byte, error) {
	return grows.WriteBytes(v, IndentEncoderFunc)
}
