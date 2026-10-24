// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package htmlcore

import (
	"container/list"
	"image"
	"sync"
)

// ImageCache is a cache of decoded images, keyed by their resolved URL,
// so that rebuilding a page does not decode the same images again.
// It keeps the most recently used images up to a total size in bytes,
// and it is safe for concurrent use.
//
// The cached images are shared by all of the widgets that show them,
// so they must not be modified. Images are only read when they are
// rendered: [core.Image] and the resizing and cropping functions of
// imagex make new images.
type ImageCache struct {

	// MaxBytes is the maximum total size of the cached images, in bytes
	// of decoded pixels. The least recently used images are removed
	// to keep the total size within it, and an image larger than it
	// is not cached.
	MaxBytes int

	// mu protects the fields below.
	mu sync.Mutex

	// images has the element of lru for each cached URL.
	images map[string]*list.Element

	// lru has the cached images, most recently used first.
	lru list.List

	// bytes is the total size of the cached images.
	bytes int
}

// imageCacheEntry is an entry of [ImageCache.lru].
type imageCacheEntry struct {
	url   string
	image image.Image
	bytes int
}

// NewImageCache returns a new [ImageCache] that keeps images up to
// the given total size in bytes.
func NewImageCache(maxBytes int) *ImageCache {
	return &ImageCache{MaxBytes: maxBytes, images: map[string]*list.Element{}}
}

// Get returns the cached image for the given resolved URL, or nil.
func (ic *ImageCache) Get(url string) image.Image {
	ic.mu.Lock()
	defer ic.mu.Unlock()
	e := ic.images[url]
	if e == nil {
		return nil
	}
	ic.lru.MoveToFront(e)
	return e.Value.(*imageCacheEntry).image
}

// Add adds the given image for the given resolved URL to the cache,
// removing the least recently used images as needed to stay within
// [ImageCache.MaxBytes].
func (ic *ImageCache) Add(url string, img image.Image) {
	n := imageBytes(img)
	ic.mu.Lock()
	defer ic.mu.Unlock()
	if e := ic.images[url]; e != nil {
		ic.remove(e)
	}
	if n > ic.MaxBytes {
		return
	}
	for ic.bytes+n > ic.MaxBytes {
		ic.remove(ic.lru.Back())
	}
	ic.images[url] = ic.lru.PushFront(&imageCacheEntry{url: url, image: img, bytes: n})
	ic.bytes += n
}

// Reset removes all of the images from the cache.
func (ic *ImageCache) Reset() {
	ic.mu.Lock()
	defer ic.mu.Unlock()
	clear(ic.images)
	ic.lru.Init()
	ic.bytes = 0
}

// remove removes the given element of lru from the cache.
func (ic *ImageCache) remove(e *list.Element) {
	ent := ic.lru.Remove(e).(*imageCacheEntry)
	delete(ic.images, ent.url)
	ic.bytes -= ent.bytes
}

// imageBytes returns the size of the decoded pixels of the given image.
func imageBytes(img image.Image) int {
	sz := img.Bounds().Size()
	bpp := 4
	switch img.(type) {
	case *image.Gray, *image.Alpha, *image.Paletted:
		bpp = 1
	case *image.Gray16, *image.Alpha16:
		bpp = 2
	case *image.RGBA64, *image.NRGBA64:
		bpp = 8
	}
	return sz.X * sz.Y * bpp
}
