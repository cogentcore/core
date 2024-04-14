// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package imagex

import (
	"bytes"
	"encoding/base64"
	"errors"
	"image"
	"image/jpeg"
	"image/png"
	"log"
	"strings"
)

// ToBase64PNG returns bytes of image encoded as a PNG in Base64 format
// with "image/png" mimetype returned
func ToBase64PNG(img image.Image) ([]byte, string) {
	ibuf := &bytes.Buffer{}
	png.Encode(ibuf, img)
	ib := ibuf.Bytes()
	eb := make([]byte, base64.StdEncoding.EncodedLen(len(ib)))
	base64.StdEncoding.Encode(eb, ib)
	return eb, "image/png"
}

// ToBase64JPG returns bytes image encoded as a JPG in Base64 format
// with "image/jpeg" mimetype returned
func ToBase64JPG(img image.Image) ([]byte, string) {
	ibuf := &bytes.Buffer{}
	jpeg.Encode(ibuf, img, &jpeg.Options{Quality: 90})
	ib := ibuf.Bytes()
	eb := make([]byte, base64.StdEncoding.EncodedLen(len(ib)))
	base64.StdEncoding.Encode(eb, ib)
	return eb, "image/jpeg"
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

// FromBase64PNG returns image from Base64-encoded bytes in PNG format
func FromBase64PNG(eb []byte) (image.Image, error) {
	if eb[76] == ' ' {
		eb = bytes.ReplaceAll(eb, []byte(" "), []byte("\n"))
	}
	db := make([]byte, base64.StdEncoding.DecodedLen(len(eb)))
	_, err := base64.StdEncoding.Decode(db, eb)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	rb := bytes.NewReader(db)
	return png.Decode(rb)
}

// FromBase64JPG returns image from Base64-encoded bytes in PNG format
func FromBase64JPG(eb []byte) (image.Image, error) {
	if eb[76] == ' ' {
		eb = bytes.ReplaceAll(eb, []byte(" "), []byte("\n"))
	}
	db := make([]byte, base64.StdEncoding.DecodedLen(len(eb)))
	_, err := base64.StdEncoding.Decode(db, eb)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	rb := bytes.NewReader(db)
	return jpeg.Decode(rb)
}

// FromBase64 returns image from Base64-encoded bytes in either PNG or JPEG format
// based on fmt which must end in either png, jpg, or jpeg
func FromBase64(fmt string, eb []byte) (image.Image, error) {
	if strings.HasSuffix(fmt, "png") {
		return FromBase64PNG(eb)
	}
	if strings.HasSuffix(fmt, "jpg") || strings.HasSuffix(fmt, "jpeg") {
		return FromBase64JPG(eb)
	}
	return nil, errors.New("image format must be either png or jpeg")
}
