// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package grows

import (
	"bufio"
	"bytes"
	"io"
	"os"
)

// Encoder is an interface for standard encoder types
type Encoder interface {
	// Encode encodes to io.Writer specified at creation
	Encode(v any) error
}

// EncoderFunc is a function that creates a new Encoder for given writer
type EncoderFunc func(w io.Writer) Encoder

// NewEncoderFunc returns a EncoderFunc for a specific Encoder type
func NewEncoderFunc[T Encoder](f func(w io.Writer) T) EncoderFunc {
	return func(w io.Writer) Encoder { return f(w) }
}

// Save writes the given object to the given filename using the given [EncoderFunc]
func Save(v any, filename string, f EncoderFunc) error {
	fp, err := os.Create(filename)
	defer fp.Close()
	if err != nil {
		return err
	}
	bw := bufio.NewWriter(fp)
	err = Write(v, bw, f)
	if err != nil {
		return err
	}
	return bw.Flush()
}

// Write writes the given object using the given [EncoderFunc]
func Write(v any, writer io.Writer, f EncoderFunc) error {
	e := f(writer)
	return e.Encode(v)
}

// WriteBytes writes the given object, returning bytes of the encoding,
// using the given [EncoderFunc]
func WriteBytes(v any, f EncoderFunc) ([]byte, error) {
	var b bytes.Buffer
	e := f(&b)
	err := e.Encode(v)
	if err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}
