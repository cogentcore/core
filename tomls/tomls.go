// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package tomls

import (
	"io"
	"io/fs"

	"github.com/BurntSushi/toml"
	"goki.dev/grows"
)

// Decoder is needed to return a standard Decode function signature for toml.
// Just wraps [toml.Decoder] to satisfy our [grows.Decoder] interface.
// Should not need to use in your code.
type Decoder struct {
	*toml.Decoder
}

// Decode implements the standard [grows.Decoder] signature
func (d *Decoder) Decode(v any) error {
	_, err := d.Decoder.Decode(v)
	return err
}

// NewDecoder returns a new [Decoder]
func NewDecoder(r io.Reader) grows.Decoder { return &Decoder{toml.NewDecoder(r)} }

// Open reads object from the given filename using TOML encoding
func Open(v any, filename string) error {
	return grows.Open(v, filename, NewDecoder)
}

// OpenFS reads object from the given filename using TOML encoding,
// using the fs.FS filesystem (e.g., for embed files)
func OpenFS(v any, fsys fs.FS, filename string) error {
	return grows.OpenFS(v, fsys, filename, NewDecoder)
}

// Read reads object encoding from the given reader,
// using TOML encoding
func Read(v any, reader io.Reader) error {
	return grows.Read(v, reader, NewDecoder)
}

// ReadBytes reads object encoding from the given bytes,
// using TOML encoding
func ReadBytes(v any, data []byte) error {
	return grows.ReadBytes(v, data, NewDecoder)
}

// Save writes object to the given filename using TOML encoding
func Save(v any, filename string) error {
	return grows.Save(v, filename, grows.NewEncoderFunc(toml.NewEncoder))
}

// Write writes object encoding using TOML encoding
func Write(v any, writer io.Writer) error {
	return grows.Write(v, writer, grows.NewEncoderFunc(toml.NewEncoder))
}

// WriteBytes writes object, returning bytes of the encoding,
// using TOML encoding
func WriteBytes(v any) ([]byte, error) {
	return grows.WriteBytes(v, grows.NewEncoderFunc(toml.NewEncoder))
}
