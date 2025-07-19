// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package imagex

import (
	"encoding/json"
	"image"
)

// JSON is a wrapper around an [image.Image] that defines JSON
// Marshal and Unmarshal methods, so that the image will automatically
// be properly saved / loaded when used as a struct field, for example.
type JSON struct {
	image.Image
}

// JSONEncoded is a representation of an image encoded into a byte stream,
// using the PNG encoder. This can be Marshal and Unmarshal'd directly.
type JSONEncoded struct {
	Width  int
	Height int
	Image  []byte
}

func (js *JSON) MarshalJSON() ([]byte, error) {
	id := &JSONEncoded{}
	if js.Image != nil {
		sz := js.Image.Bounds().Size()
		id.Width = sz.X
		id.Height = sz.Y
		id.Image, _ = ToBase64PNG(js.Image)
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
	im, err := FromBase64PNG(id.Image)
	if err != nil {
		js.Image = nil
		return err
	}
	js.Image = im
	return nil
}
