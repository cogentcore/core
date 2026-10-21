// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build !js

package imagex

import (
	"image"
	"io"
)

// decode decodes an image from the given reader.
func decode(r io.Reader) (image.Image, string, error) {
	return image.Decode(r)
}
