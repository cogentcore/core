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

// TestImageCacheReads tests that the images of the current read are kept
// over those of the reads before it, and that when they do not all fit
// the rest are not cached, so that a page read again finds most of its
// images (a plain LRU, read in the same order each time, found none, as
// the photos of nytimes.com, decoded again in each read of the page), and
// that the URLs of images that failed are kept.
func TestImageCacheReads(t *testing.T) {
	newImage := func() image.Image { return image.NewRGBA(image.Rect(0, 0, 10, 10)) } // 400 bytes
	ic := NewImageCache(1000)
	read := func() (hits int) {
		ic.NewRead()
		for _, u := range []string{"a", "b", "c", "d"} { // 1600 bytes
			if ic.Get(u) != nil {
				hits++
				continue
			}
			ic.Add(u, newImage())
		}
		return hits
	}
	assert.Equal(t, 0, read())
	assert.Equal(t, 2, read(), "the first two of the page are kept")
	assert.Equal(t, 2, read())

	// the images of an earlier read make room for those of a new page
	ic.NewRead()
	ic.Add("e", newImage())
	assert.NotNil(t, ic.Get("e"))

	assert.False(t, ic.Failed("x"))
	ic.AddFailed("x")
	assert.True(t, ic.Failed("x"))
	ic.Reset()
	assert.False(t, ic.Failed("x"))
}

// TestImageCacheFailed tests that an image that can not be decoded is not
// gotten again when the page is read again.
func TestImageCacheFailed(t *testing.T) {
	fsys := fstest.MapFS{"bad.png": {Data: []byte("not an image")}}
	gets := map[string]int{}
	ctx := NewContext()
	ctx.ImageCache = NewImageCache(1 << 20)
	ctx.GetURL = func(url string) (*http.Response, error) {
		gets[url]++
		return GetURLFromFS(fsys, url)
	}
	for range 2 {
		ctx.PageURL = "/page"
		assert.NoError(t, ReadHTMLString(ctx, core.NewBody(), `<img src="bad.png">`))
	}
	assert.Equal(t, map[string]int{"/bad.png": 1}, gets)
}
