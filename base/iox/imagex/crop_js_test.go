// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build js

package imagex

import (
	"bytes"
	"image"
	"image/color"
	"syscall/js"
	"testing"

	"github.com/stretchr/testify/assert"
)

// browserSources returns the test image as each kind of source that Crop
// and Resize handle: local pixels, wrapped local pixels, and a bitmap
// decoded by the browser, which has no local pixels.
func browserSources(t *testing.T, src *image.RGBA) map[string]image.Image {
	var buf bytes.Buffer
	assert.NoError(t, Write(src, &buf, PNG))
	dec, _, err := Read(&buf)
	assert.NoError(t, err)
	assert.Nil(t, dec.(*JSRGBA).RGBA)
	return map[string]image.Image{"RGBA": src, "WrapJS": WrapJS(src), "decoded": dec}
}

// assertConsistent checks that the bounds, the local pixels and the
// ImageData of the given result of Crop or Resize all have the given size,
// and that its pixels read back from its bitmap match its local pixels.
func assertConsistent(t *testing.T, name string, im image.Image, size image.Point) {
	jim, ok := im.(*JSRGBA)
	if !assert.True(t, ok, name) {
		return
	}
	assert.Equal(t, image.Rectangle{Max: size}, jim.Bounds(), name)
	if !jim.JS.Data.IsUndefined() {
		assert.Equal(t, size.X, jim.JS.Data.Get("width").Int(), name)
		assert.Equal(t, size.Y, jim.JS.Data.Get("height").Int(), name)
	}
	assert.Equal(t, image.Rectangle{Max: size}, jim.Underlying().Bounds(), name)
	back := readBitmapPixels(jim.JS.Bitmap, jim.JS.Bounds)
	assert.Equal(t, back.Pix, jim.RGBA.Pix, "%s: bitmap and local pixels", name)
}

func TestCropResizeBrowser(t *testing.T) {
	if js.Global().Get("createImageBitmap").IsUndefined() {
		t.Skip("createImageBitmap is not available")
	}
	src := testImage()
	rect := image.Rect(3, 5, 13, 9)
	for name, im := range browserSources(t, src) {
		name = "Crop " + name
		cim := Crop(im, rect)
		assertConsistent(t, name, cim, rect.Size())
		for y := range rect.Dy() {
			for x := range rect.Dx() {
				assert.Equal(t, src.RGBAAt(rect.Min.X+x, rect.Min.Y+y), cim.(*JSRGBA).RGBAAt(x, y), "%s %d,%d", name, x, y)
			}
		}
	}

	// Resizing an image of four solid quadrants keeps their colors.
	quads := image.NewRGBA(image.Rect(0, 0, 16, 16))
	clrs := []color.RGBA{{255, 0, 0, 255}, {0, 255, 0, 255}, {0, 0, 255, 255}, {255, 255, 255, 255}}
	for y := range 16 {
		for x := range 16 {
			quads.SetRGBA(x, y, clrs[(y/8)*2+x/8])
		}
	}
	size := image.Pt(8, 8)
	for name, im := range browserSources(t, quads) {
		name = "Resize " + name
		rim := Resize(im, size)
		assertConsistent(t, name, rim, size)
		for i, c := range clrs {
			x, y := 1+(i%2)*5, 1+(i/2)*5
			assert.Equal(t, c, rim.(*JSRGBA).RGBAAt(x, y), "%s %d,%d", name, x, y)
		}
		// Cropping the resized image, as styles.Style.ResizeImage does.
		name = "Crop " + name
		cim := Crop(rim, image.Rect(4, 0, 8, 6))
		assertConsistent(t, name, cim, image.Pt(4, 6))
		assert.Equal(t, clrs[1], cim.(*JSRGBA).RGBAAt(1, 1), name)
		assert.Equal(t, clrs[3], cim.(*JSRGBA).RGBAAt(2, 5), name)
	}
}
