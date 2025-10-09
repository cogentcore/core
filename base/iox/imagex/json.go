// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package imagex

import (
	"bytes"
	"encoding/json"
	"image"
)

// JSON is a wrapper around an [image.Image] that defines JSON
// Marshal and Unmarshal methods, so that the image will automatically
// be properly saved / loaded when used as a struct field, for example.
// Must be a pointer type to support custom unmarshal function.
// The original image is not anonymously embedded so that you have to
// extract it, otherwise it will be processed inefficiently.
type JSON struct {
	Image image.Image
}

// JSONEncoded is a representation of an image encoded into a byte stream,
// using the PNG encoder. This can be Marshal and Unmarshal'd directly.
type JSONEncoded struct {
	Width  int
	Height int

	// Image is the encoded byte stream, which will be encoded in JSON
	// using Base64
	Image []byte
}

// NewJSON returns a new JSON wrapper around given image,
// to support automatic wrapping and unwrapping.
func NewJSON(im image.Image) *JSON {
	return &JSON{Image: im}
}

func (js *JSON) MarshalJSON() ([]byte, error) {
	id := &JSONEncoded{}
	if js.Image != nil {
		sz := js.Image.Bounds().Size()
		id.Width = sz.X
		id.Height = sz.Y
		ibuf := &bytes.Buffer{}
		Write(js.Image, ibuf, PNG)
		id.Image = ibuf.Bytes()
	}
	return json.Marshal(id)
}

func (js *JSON) UnmarshalJSON(b []byte) error {
	id := &JSONEncoded{}
	err := json.Unmarshal(b, id)
	if err != nil || (id.Width == 0 && id.Height == 0) {
		js.Image = nil
		return err
	}
	im, _, err := image.Decode(bytes.NewReader(id.Image))
	if err != nil {
		js.Image = nil
		return err
	}
	js.Image = im
	return nil
}
