// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package imagex

import (
	"bytes"
	"encoding/base64"
	"image"
	"strings"

	"cogentcore.org/core/base/errors"
)

// todo: pass format

// ToBase64 returns bytes of image encoded in given format,
// in Base64 encoding with "image/format" mimetype returned
func ToBase64(img image.Image, f Formats) ([]byte, string) {
	ibuf := &bytes.Buffer{}
	Write(img, ibuf, f)
	ib := ibuf.Bytes()
	eb := make([]byte, base64.StdEncoding.EncodedLen(len(ib)))
	base64.StdEncoding.Encode(eb, ib)
	return eb, "image/" + strings.ToLower(f.String())
}

// Base64SplitLines splits the encoded Base64 bytes into standard lines of 76
// chars each.  The last line also ends in a newline
func Base64SplitLines(b []byte) []byte {
	ll := 76
	sz := len(b)
	nl := (sz / ll)
	rb := make([]byte, sz+nl+1)
	for i := 0; i < nl; i++ {
		st := ll * i
		rst := ll*i + i
		copy(rb[rst:rst+ll], b[st:st+ll])
		rb[rst+ll] = '\n'
	}
	st := ll * nl
	rst := ll*nl + nl
	ln := sz - st
	copy(rb[rst:rst+ln], b[st:st+ln])
	rb[rst+ln] = '\n'
	return rb
}

// FromBase64 returns image from Base64-encoded bytes
func FromBase64(eb []byte) (image.Image, Formats, error) {
	if eb[76] == ' ' {
		eb = bytes.ReplaceAll(eb, []byte(" "), []byte("\n"))
	}
	db := make([]byte, base64.StdEncoding.DecodedLen(len(eb)))
	_, err := base64.StdEncoding.Decode(db, eb)
	if err != nil {
		return nil, None, errors.Log(err)
	}
	return Read(bytes.NewReader(db))
}
