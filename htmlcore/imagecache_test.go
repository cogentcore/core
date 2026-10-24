// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package htmlcore

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"net/http"
	"testing"
	"testing/fstest"

	"cogentcore.org/core/core"
	"cogentcore.org/core/tree"
	"github.com/stretchr/testify/assert"
)

// testPNG returns a PNG of a 4x3 image filled with the given color.
func testPNG(t *testing.T, c color.RGBA) []byte {
	img := image.NewRGBA(image.Rect(0, 0, 4, 3))
	for i := 0; i < len(img.Pix); i += 4 {
		img.Pix[i], img.Pix[i+1], img.Pix[i+2], img.Pix[i+3] = c.R, c.G, c.B, c.A
	}
	var b bytes.Buffer
	if err := png.Encode(&b, img); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}

// pageImages returns the images of the Image widgets under w.
func pageImages(w core.Widget) []image.Image {
	var ims []image.Image
	w.AsWidget().WalkDown(func(n tree.Node) bool {
		if im, ok := n.(*core.Image); ok {
			ims = append(ims, im.Image)
		}
		return tree.Continue
	})
	return ims
}

// TestImageCache tests that rebuilding a page gets its images from the
// cache, which is keyed by the URL resolved against the page URL.
func TestImageCache(t *testing.T) {
	red := color.RGBA{255, 0, 0, 255}
	blue := color.RGBA{0, 0, 255, 255}
	fsys := fstest.MapFS{
		"a/img.png": {Data: testPNG(t, red)},
		"b/img.png": {Data: testPNG(t, blue)},
	}
	gets := map[string]int{}
	ctx := NewContext()
	ctx.ImageCache = NewImageCache(1 << 20)
	ctx.GetURL = func(url string) (*http.Response, error) {
		gets[url]++
		return GetURLFromFS(fsys, url)
	}
	read := func(page string) []image.Image {
		ctx.PageURL = page
		b := core.NewBody()
		assert.NoError(t, ReadMDString(ctx, b, "![first](img.png)\n\n![second](img.png)\n"))
		return pageImages(b)
	}

	a1 := read("/a/page")
	a2 := read("/a/page")
	bi := read("/b/page")
	assert.Equal(t, map[string]int{"/a/img.png": 1, "/b/img.png": 1}, gets)
	for _, ims := range [][]image.Image{a1, a2, bi} {
		assert.Len(t, ims, 2)
	}
	for _, im := range append(a1[1:], a2...) {
		assert.Same(t, a1[0], im)
	}
	assert.Equal(t, red, color.RGBAModel.Convert(a1[0].At(0, 0)))
	assert.Equal(t, blue, color.RGBAModel.Convert(bi[0].At(0, 0)))

	ctx.ImageCache.Reset()
	read("/a/page")
	assert.Equal(t, 2, gets["/a/img.png"])
}

// TestImageCacheLRU tests that the cache removes the least recently
// used images to stay within its size.
func TestImageCacheLRU(t *testing.T) {
	newImage := func() image.Image { return image.NewRGBA(image.Rect(0, 0, 10, 10)) } // 400 bytes
	ic := NewImageCache(1000)
	a, b, c := newImage(), newImage(), newImage()
	ic.Add("a", a)
	ic.Add("b", b)
	assert.Same(t, a, ic.Get("a")) // a is now more recently used than b
	ic.Add("c", c)
	assert.Nil(t, ic.Get("b"))
	assert.Same(t, a, ic.Get("a"))
	assert.Same(t, c, ic.Get("c"))
	assert.Equal(t, 800, ic.bytes)

	ic.Add("big", image.NewRGBA(image.Rect(0, 0, 20, 20))) // larger than the cache
	assert.Nil(t, ic.Get("big"))
	assert.Equal(t, 800, ic.bytes)

	ic.Add("a", c) // replacing an image keeps one entry for its URL
	assert.Same(t, c, ic.Get("a"))
	assert.Equal(t, 800, ic.bytes)
	assert.Len(t, ic.images, 2)
}
