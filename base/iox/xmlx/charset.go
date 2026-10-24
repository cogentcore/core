// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build !js

package xmlx

import (
	"io"

	"golang.org/x/net/html/charset"
)

// CharsetReader returns a reader that converts the text of input from the
// character set with the given label to UTF-8. It is meant for
// [xml.Decoder.CharsetReader], which calls it for documents that declare an
// encoding other than UTF-8. On the web, it uses the browser's TextDecoder,
// which keeps the encoding tables of golang.org/x/net/html/charset out of
// web builds.
func CharsetReader(label string, input io.Reader) (io.Reader, error) {
	return charset.NewReaderLabel(label, input)
}
