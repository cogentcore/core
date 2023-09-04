// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package grows

import (
	"bufio"
	"bytes"
	"io"
	"io/fs"
	"os"
)

// Decoder is an interface for standard decoder types
type Decoder interface {
	// Decode decodes from io.Reader specified at creation
	Decode(v any) error
}

// DecoderFunc is a function that creates a new Decoder for given reader
type DecoderFunc func(r io.Reader) Decoder

// NewDecoderFunc returns a DecoderFunc for a specific Decoder type
func NewDecoderFunc[T Decoder](f func(r io.Reader) T) DecoderFunc {
	return func(r io.Reader) Decoder { return f(r) }
}

/*
// OpenFromPaths reads object from given TOML file,
// looking on paths for the file.
func OpenFromPaths(obj any, file string, paths []string) error {
	filename, err := dirs.FindFileOnPaths(paths, file)
	if err != nil {
		log.Println(err)
		return err
	}
	// _, err = toml.DecodeFile(fp, obj)
	fp, err := os.Open(filename)
	defer fp.Close()
	if err != nil {
		log.Println(err)
		return err
	}
	return Read(obj, bufio.NewReader(fp))
}
*/

// Open reads object from the given filename using the given [DecoderFunc]
func Open(v any, filename string, f DecoderFunc) error {
	fp, err := os.Open(filename)
	defer fp.Close()
	if err != nil {
		return err
	}
	return Read(v, bufio.NewReader(fp), f)
}

// OpenFS reads object from the given filename using the given [DecoderFunc],
// using the fs.FS filesystem (e.g., for embed files)
func OpenFS(v any, fsys fs.FS, filename string, f DecoderFunc) error {
	fp, err := fsys.Open(filename)
	defer fp.Close()
	if err != nil {
		return err
	}
	return Read(v, bufio.NewReader(fp), f)
}

// Read reads object encoding from the given reader,
// using the given [DecoderFunc]
func Read(v any, reader io.Reader, f DecoderFunc) error {
	d := f(reader)
	return d.Decode(v)
}

// ReadBytes reads object encoding from the given bytes,
// using the given [DecoderFunc]
func ReadBytes(v any, data []byte, f DecoderFunc) error {
	b := bytes.NewBuffer(data)
	return Read(v, b, f)
}
