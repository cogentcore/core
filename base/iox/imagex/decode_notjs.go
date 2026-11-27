// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build !js

package imagex

import (
	"image"
	"io"
)

// decode decodes an image from the given reader, turning JPEG images
// upright as their EXIF orientation says (see [decodeOriented]).
func decode(r io.Reader) (image.Image, string, error) {
	b, err := io.ReadAll(r)
	if err != nil {
		return nil, "", err
	}
	return decodeOriented(b)
}
