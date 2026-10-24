// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build !js

package imagex

import (
	"image"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestCrop tests Crop with Go images; on web it needs a browser, as
// TestCropResizeBrowser.
func TestCrop(t *testing.T) {
	src := testImage()
	rect := image.Rect(3, 5, 13, 9)
	im := Crop(src, rect)
	assert.Equal(t, image.Rectangle{Max: rect.Size()}, im.Bounds())
	rgba := Unwrap(im).(*image.RGBA)
	for y := range rect.Dy() {
		for x := range rect.Dx() {
			assert.Equal(t, src.RGBAAt(rect.Min.X+x, rect.Min.Y+y), rgba.RGBAAt(x, y), "%d,%d", x, y)
		}
	}
}
