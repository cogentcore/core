// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build js

package xmlx

import (
	"fmt"
	"io"
	"strings"
	"syscall/js"
)

// CharsetReader returns a reader that converts the text of input from the
// character set with the given label to UTF-8. It is meant for
// [xml.Decoder.CharsetReader], which calls it for documents that declare an
// encoding other than UTF-8. On the web, it uses the browser's TextDecoder,
// which keeps the encoding tables of golang.org/x/net/html/charset out of
// web builds.
func CharsetReader(label string, input io.Reader) (r io.Reader, err error) {
	if strings.EqualFold(label, "utf-8") || strings.EqualFold(label, "utf8") {
		return input, nil
	}
	data, err := io.ReadAll(input)
	if err != nil {
		return nil, err
	}
	// TextDecoder throws for a label it does not know.
	defer func() {
		if recover() != nil {
			r, err = nil, fmt.Errorf("xmlx: unsupported charset: %q", label)
		}
	}()
	src := js.Global().Get("Uint8Array").New(len(data))
	js.CopyBytesToJS(src, data)
	text := js.Global().Get("TextDecoder").New(label).Call("decode", src).String()
	return strings.NewReader(text), nil
}
