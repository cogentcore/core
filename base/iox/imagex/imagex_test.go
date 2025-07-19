// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package imagex

import (
	"encoding/json"
	"image"
	"image/color"
	"testing"

	"github.com/stretchr/testify/assert"
)

type testObj struct {
	Name    string
	Image   *JSON
	Another string
}

func testImage() *image.RGBA {
	im := image.NewRGBA(image.Rect(0, 0, 16, 16))
	for y := range 16 {
		for x := range 16 {
			im.Set(x, y, color.RGBA{uint8(x * 16), uint8(y * 16), 128, 255})
		}
	}
	return im
}

func TestSave(t *testing.T) {
	im := testImage()
	// this tests Save and Open etc for all formats
	Assert(t, im, "test.png", 1)  // should be exact
	Assert(t, im, "test.jpg", 20) // quite bad
	Assert(t, im, "test.gif", 50) // even worse
	Assert(t, im, "test.tif", 1)
	Assert(t, im, "test.bmp", 1)
	// Assert(t, im, "test.webp") // only for reading, not writing
}

func TestBase64(t *testing.T) {
	im := testImage()
	b, mime := ToBase64PNG(im)
	assert.Equal(t, "image/png", mime)
	bim, err := FromBase64PNG(b)
	assert.NoError(t, err)
	bounds, content, _, _, _, _ := ImagesEqual(im, bim, 1)
	assert.Equal(t, true, bounds)
	assert.Equal(t, true, content)

	b, mime = ToBase64JPG(im)
	assert.Equal(t, "image/jpeg", mime)
	bim, err = FromBase64JPG(b)
	assert.NoError(t, err)
	bounds, content, _, _, _, _ = ImagesEqual(im, bim, 20)
	assert.Equal(t, true, bounds)
	assert.Equal(t, true, content)
}

func TestJSON(t *testing.T) {
	im := testImage()
	jsi := &JSON{Image: im}

	b, err := json.Marshal(jsi)
	assert.NoError(t, err)

	nsi := &JSON{}
	err = json.Unmarshal(b, nsi)
	assert.NoError(t, err)

	ri := nsi.Image.(*image.RGBA)

	assert.Equal(t, im, ri)

	bounds, content, _, _, _, _ := ImagesEqual(im, ri, 1)
	assert.Equal(t, true, bounds)
	assert.Equal(t, true, content)

	jo := &testObj{Name: "testy", Another: "guy"}
	jo.Image = NewJSON(im)

	b, err = json.Marshal(jo)
	assert.NoError(t, err)

	no := &testObj{}
	err = json.Unmarshal(b, no)
	assert.NoError(t, err)

	assert.Equal(t, jo, no)
	bounds, content, _, _, _, _ = ImagesEqual(jo.Image.Image, no.Image.Image, 1)
	assert.Equal(t, true, bounds)
	assert.Equal(t, true, content)
}
